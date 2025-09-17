package service

import (
	"context"
	"fmt"
	"time"

	"reciprocal-clubs-backend/pkg/shared/auth"
	"reciprocal-clubs-backend/pkg/shared/config"
	"reciprocal-clubs-backend/pkg/shared/errors"
	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/messaging"
	"reciprocal-clubs-backend/services/auth-service/internal/hanko"
	"reciprocal-clubs-backend/services/auth-service/internal/models"
	"reciprocal-clubs-backend/services/auth-service/internal/repository"
)

// AuthService handles authentication business logic
type AuthService struct {
	repo         *repository.AuthRepository
	hankoClient  HankoClientInterface
	authProvider *auth.JWTProvider
	messageBus   messaging.MessageBus
	config       *config.Config
	logger       logging.Logger
}

// HankoClientInterface defines the interface for Hanko client
type HankoClientInterface interface {
	CreateUser(ctx context.Context, email string) (*hanko.HankoUser, error)
	GetUser(ctx context.Context, userID string) (*hanko.HankoUser, error)
	GetUserByEmail(ctx context.Context, email string) (*hanko.HankoUser, error)
	ValidateSession(ctx context.Context, sessionToken string) (*hanko.ValidateSessionResponse, error)
	InitiatePasskeyRegistration(ctx context.Context, userID string) (*hanko.PasskeyRegistrationResponse, error)
	InitiatePasskeyAuthentication(ctx context.Context, userEmail string) (*hanko.PasskeyAuthenticationResponse, error)
	VerifyPasskey(ctx context.Context, userID string, credentialResult map[string]interface{}) (*hanko.VerifyPasskeyResponse, error)
	InvalidateSession(ctx context.Context, sessionID string) error
	DeleteUser(ctx context.Context, userID string) error
	HealthCheck(ctx context.Context) error
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	ClubSlug string `json:"club_slug" validate:"required"`
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Username  string `json:"username" validate:"required,min=3,max=50"`
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
	ClubSlug  string `json:"club_slug" validate:"required"`
}

// AuthResponse represents authentication response
type AuthResponse struct {
	User         *models.User `json:"user"`
	Token        string       `json:"token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresAt    time.Time    `json:"expires_at"`
}

// PasskeyResponse represents passkey operation response
type PasskeyResponse struct {
	Options map[string]interface{} `json:"options"`
	UserID  string                 `json:"user_id,omitempty"`
}

// NewAuthService creates a new auth service
func NewAuthService(repo *repository.AuthRepository, messageBus messaging.MessageBus, config *config.Config, logger logging.Logger) *AuthService {
	authProvider := auth.NewJWTProvider(&config.Auth, logger)

	// Initialize Hanko client - use mock for development
	var hankoClient HankoClientInterface
	if config.Service.Environment == "production" {
		hankoClient = hanko.NewHankoClient(hanko.Config{
			BaseURL: "http://hanko:8000", // Hanko service URL
			APIKey:  config.Auth.JWTSecret, // Use appropriate API key
			Timeout: 30 * time.Second,
		}, logger)
	} else {
		hankoClient = hanko.NewMockHankoClient(logger)
	}

	return &AuthService{
		repo:         repo,
		hankoClient:  hankoClient,
		authProvider: authProvider,
		messageBus:   messageBus,
		config:       config,
		logger:       logger,
	}
}

// Register registers a new user
func (s *AuthService) Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error) {
	// Get club by slug
	club, err := s.repo.GetClubBySlug(ctx, req.ClubSlug)
	if err != nil {
		return nil, err
	}

	// Create user in Hanko first
	hankoUser, err := s.hankoClient.CreateUser(ctx, req.Email)
	if err != nil {
		s.logger.Error("Failed to create user in Hanko", map[string]interface{}{
			"error": err.Error(),
			"email": req.Email,
		})
		return nil, errors.Internal("Failed to create user account", map[string]interface{}{
			"email": req.Email,
		}, err)
	}

	// Create user in our database
	user := &models.User{
		HankoUserID:   hankoUser.ID,
		Email:         req.Email,
		Username:      req.Username,
		FirstName:     req.FirstName,
		LastName:      req.LastName,
		Status:        models.UserStatusActive,
		EmailVerified: hankoUser.EmailVerified,
	}
	user.ClubID = club.ID

	// Create user in transaction
	err = s.repo.WithTransaction(ctx, func(txRepo *repository.AuthRepository) error {
		// Create user
		if err := txRepo.CreateUser(ctx, user); err != nil {
			return err
		}

		// Assign default member role
		memberRole, err := txRepo.GetRoleByName(ctx, club.ID, models.RoleMember)
		if err != nil {
			return err
		}

		userRole := &models.UserRole{
			UserID:    user.ID,
			RoleID:    memberRole.ID,
			GrantedAt: time.Now(),
			IsActive:  true,
		}
		userRole.ClubID = club.ID

		return txRepo.AssignRole(ctx, userRole)
	})

	if err != nil {
		// Cleanup Hanko user if database operation failed
		s.hankoClient.DeleteUser(ctx, hankoUser.ID)
		return nil, err
	}

	// Create audit log
	s.createAuditLog(ctx, club.ID, user, models.AuditActionRegister, "User registered successfully", true, "")

	// Publish user registered event
	s.publishUserEvent(ctx, "user.registered", user)

	s.logger.Info("User registered successfully", map[string]interface{}{
		"user_id":       user.ID,
		"email":         user.Email,
		"hanko_user_id": user.HankoUserID,
		"club_id":       club.ID,
	})

	// Generate tokens
	authUser := s.convertToAuthUser(user)
	token, err := s.authProvider.GenerateToken(authUser, 0)
	if err != nil {
		return nil, errors.Internal("Failed to generate token", nil, err)
	}

	refreshToken, err := s.authProvider.GenerateToken(authUser, 24*7*time.Hour)
	if err != nil {
		return nil, errors.Internal("Failed to generate refresh token", nil, err)
	}

	return &AuthResponse{
		User:         user,
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(s.config.Auth.JWTExpiration) * time.Second),
	}, nil
}

// InitiatePasskeyLogin initiates passkey authentication
func (s *AuthService) InitiatePasskeyLogin(ctx context.Context, req *LoginRequest) (*PasskeyResponse, error) {
	// Get club by slug
	club, err := s.repo.GetClubBySlug(ctx, req.ClubSlug)
	if err != nil {
		return nil, err
	}

	// Check if user exists in our database
	user, err := s.repo.GetUserByEmail(ctx, club.ID, req.Email)
	if err != nil {
		if !errors.Is(err, errors.ErrNotFound) {
			return nil, err
		}
		// User doesn't exist, return error
		return nil, errors.NotFound("User not found", map[string]interface{}{
			"email": req.Email,
		})
	}

	// Check user status
	if !user.IsActive() {
		s.createAuditLog(ctx, club.ID, user, models.AuditActionLogin, "Login attempt with inactive account", false, "Account is not active")
		return nil, errors.Forbidden("Account is not active", map[string]interface{}{
			"status": string(user.Status),
		})
	}

	// Initiate passkey authentication with Hanko
	response, err := s.hankoClient.InitiatePasskeyAuthentication(ctx, req.Email)
	if err != nil {
		s.logger.Error("Failed to initiate passkey authentication", map[string]interface{}{
			"error":   err.Error(),
			"email":   req.Email,
			"user_id": user.ID,
		})
		s.createAuditLog(ctx, club.ID, user, models.AuditActionLogin, "Failed to initiate passkey authentication", false, err.Error())
		return nil, errors.Internal("Failed to initiate authentication", nil, err)
	}

	s.logger.Info("Passkey authentication initiated", map[string]interface{}{
		"user_id": user.ID,
		"email":   user.Email,
	})

	return &PasskeyResponse{
		Options: response.AuthenticationOptions,
		UserID:  user.HankoUserID,
	}, nil
}

// CompletePasskeyLogin completes passkey authentication
func (s *AuthService) CompletePasskeyLogin(ctx context.Context, clubSlug string, hankoUserID string, credentialResult map[string]interface{}) (*AuthResponse, error) {
	// Get club by slug
	club, err := s.repo.GetClubBySlug(ctx, clubSlug)
	if err != nil {
		return nil, err
	}

	// Get user by Hanko ID
	user, err := s.repo.GetUserByHankoID(ctx, club.ID, hankoUserID)
	if err != nil {
		return nil, err
	}

	// Check user status
	if !user.IsActive() {
		s.createAuditLog(ctx, club.ID, user, models.AuditActionLogin, "Login attempt with inactive account", false, "Account is not active")
		return nil, errors.Forbidden("Account is not active", map[string]interface{}{
			"status": string(user.Status),
		})
	}

	// Verify passkey with Hanko
	response, err := s.hankoClient.VerifyPasskey(ctx, hankoUserID, credentialResult)
	if err != nil {
		s.logger.Error("Failed to verify passkey", map[string]interface{}{
			"error":         err.Error(),
			"hanko_user_id": hankoUserID,
			"user_id":       user.ID,
		})
		user.IncrementFailedAttempts()
		s.repo.UpdateUser(ctx, user)
		s.createAuditLog(ctx, club.ID, user, models.AuditActionLogin, "Passkey verification failed", false, err.Error())
		return nil, errors.Unauthorized("Authentication failed", nil)
	}

	if !response.Success {
		s.logger.Warn("Passkey authentication failed", map[string]interface{}{
			"hanko_user_id": hankoUserID,
			"user_id":       user.ID,
			"error_code":    response.ErrorCode,
		})
		user.IncrementFailedAttempts()
		s.repo.UpdateUser(ctx, user)
		s.createAuditLog(ctx, club.ID, user, models.AuditActionLogin, "Passkey authentication failed", false, response.ErrorDetail)
		return nil, errors.Unauthorized("Authentication failed", nil)
	}

	// Authentication successful - update user
	user.ResetFailedAttempts()
	user.Unlock()
	now := time.Now()
	user.LastLoginAt = &now

	// Create session
	session := &models.UserSession{
		UserID:         user.ID,
		HankoSessionID: response.Session.ID,
		IPAddress:      s.getIPFromContext(ctx),
		UserAgent:      s.getUserAgentFromContext(ctx),
		ExpiresAt:      response.Session.ExpiresAt,
		IsActive:       true,
	}
	session.ClubID = club.ID

	// Update user and create session in transaction
	err = s.repo.WithTransaction(ctx, func(txRepo *repository.AuthRepository) error {
		if err := txRepo.UpdateUser(ctx, user); err != nil {
			return err
		}
		return txRepo.CreateSession(ctx, session)
	})

	if err != nil {
		return nil, err
	}

	// Create audit log
	s.createAuditLog(ctx, club.ID, user, models.AuditActionLogin, "Successful passkey authentication", true, "")

	// Publish login event
	s.publishUserEvent(ctx, "user.login", user)

	s.logger.Info("User logged in successfully", map[string]interface{}{
		"user_id":    user.ID,
		"email":      user.Email,
		"session_id": session.HankoSessionID,
	})

	// Generate tokens
	authUser := s.convertToAuthUser(user)
	token, err := s.authProvider.GenerateToken(authUser, 0)
	if err != nil {
		return nil, errors.Internal("Failed to generate token", nil, err)
	}

	refreshToken, err := s.authProvider.GenerateToken(authUser, 24*7*time.Hour)
	if err != nil {
		return nil, errors.Internal("Failed to generate refresh token", nil, err)
	}

	// Update session with JWT token
	session.JWTToken = token
	session.RefreshToken = refreshToken
	s.repo.UpdateSession(ctx, session)

	return &AuthResponse{
		User:         user,
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(s.config.Auth.JWTExpiration) * time.Second),
	}, nil
}

// InitiatePasskeyRegistration initiates passkey registration for existing user
func (s *AuthService) InitiatePasskeyRegistration(ctx context.Context, userID uint, clubID uint) (*PasskeyResponse, error) {
	// Get user
	user, err := s.repo.GetUserByID(ctx, clubID, userID)
	if err != nil {
		return nil, err
	}

	// Initiate passkey registration with Hanko
	response, err := s.hankoClient.InitiatePasskeyRegistration(ctx, user.HankoUserID)
	if err != nil {
		s.logger.Error("Failed to initiate passkey registration", map[string]interface{}{
			"error":   err.Error(),
			"user_id": user.ID,
		})
		s.createAuditLog(ctx, clubID, user, models.AuditActionPasskeyRegistration, "Failed to initiate passkey registration", false, err.Error())
		return nil, errors.Internal("Failed to initiate passkey registration", nil, err)
	}

	s.logger.Info("Passkey registration initiated", map[string]interface{}{
		"user_id": user.ID,
		"email":   user.Email,
	})

	return &PasskeyResponse{
		Options: response.RegistrationOptions,
		UserID:  user.HankoUserID,
	}, nil
}

// ValidateSession validates a session token
func (s *AuthService) ValidateSession(ctx context.Context, sessionToken string) (*models.User, error) {
	// Validate session with Hanko
	response, err := s.hankoClient.ValidateSession(ctx, sessionToken)
	if err != nil {
		return nil, errors.Unauthorized("Invalid session", nil)
	}

	if !response.Valid {
		return nil, errors.Unauthorized("Session expired", nil)
	}

	// Get user from our database
	user, err := s.repo.GetUserByHankoID(ctx, 1, response.User.ID) // Note: Need to handle club ID properly
	if err != nil {
		return nil, err
	}

	// Update session activity
	session, err := s.repo.GetSessionByHankoID(ctx, user.ClubID, response.Session.ID)
	if err == nil {
		session.UpdateActivity()
		s.repo.UpdateSession(ctx, session)
	}

	return user, nil
}

// Logout logs out a user
func (s *AuthService) Logout(ctx context.Context, userID, clubID uint, sessionToken string) error {
	// Get user
	user, err := s.repo.GetUserByID(ctx, clubID, userID)
	if err != nil {
		return err
	}

	// Get session by token
	_, err = s.repo.GetSessionByHankoID(ctx, clubID, sessionToken)
	if err != nil {
		// Session not found, but still invalidate in Hanko
		s.hankoClient.InvalidateSession(ctx, sessionToken)
		return nil
	}

	// Invalidate session in both Hanko and our database
	err = s.hankoClient.InvalidateSession(ctx, sessionToken)
	if err != nil {
		s.logger.Warn("Failed to invalidate session in Hanko", map[string]interface{}{
			"error":      err.Error(),
			"session_id": sessionToken,
		})
	}

	err = s.repo.InvalidateSession(ctx, clubID, sessionToken)
	if err != nil {
		return err
	}

	// Create audit log
	s.createAuditLog(ctx, clubID, user, models.AuditActionLogout, "User logged out", true, "")

	// Publish logout event
	s.publishUserEvent(ctx, "user.logout", user)

	s.logger.Info("User logged out successfully", map[string]interface{}{
		"user_id":    user.ID,
		"session_id": sessionToken,
	})

	return nil
}

// GetUserWithRoles retrieves a user with their roles and permissions
func (s *AuthService) GetUserWithRoles(ctx context.Context, clubID, userID uint) (*models.UserWithRoles, error) {
	user, err := s.repo.GetUserByID(ctx, clubID, userID)
	if err != nil {
		return nil, err
	}

	roles, err := s.repo.GetUserRoles(ctx, clubID, userID)
	if err != nil {
		return nil, err
	}

	permissions, err := s.repo.GetUserPermissions(ctx, clubID, userID)
	if err != nil {
		return nil, err
	}

	// Extract role names and permissions
	roleNames := make([]string, len(roles))
	for i, role := range roles {
		roleNames[i] = role.Name
	}

	permissionNames := make([]string, len(permissions))
	for i, permission := range permissions {
		permissionNames[i] = permission.Name
	}

	return &models.UserWithRoles{
		User:        *user,
		RoleNames:   roleNames,
		Permissions: permissionNames,
	}, nil
}

// Helper methods

func (s *AuthService) convertToAuthUser(user *models.User) *auth.User {
	return &auth.User{
		ID:       user.ID,
		ClubID:   user.ClubID,
		Email:    user.Email,
		Username: user.Username,
		// Roles and permissions will be loaded separately if needed
	}
}

func (s *AuthService) createAuditLog(ctx context.Context, clubID uint, user *models.User, action models.AuditAction, details string, success bool, errorMessage string) {
	auditLog := &models.AuditLog{
		UserID:       &user.ID,
		HankoUserID:  user.HankoUserID,
		Action:       action,
		Details:      details,
		Success:      success,
		ErrorMessage: errorMessage,
		IPAddress:    s.getIPFromContext(ctx),
		UserAgent:    s.getUserAgentFromContext(ctx),
	}
	auditLog.ClubID = clubID

	// Create audit log asynchronously
	go func() {
		ctx := context.Background()
		if err := s.repo.CreateAuditLog(ctx, auditLog); err != nil {
			s.logger.Error("Failed to create audit log", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()
}

func (s *AuthService) publishUserEvent(ctx context.Context, eventType string, user *models.User) {
	event := map[string]interface{}{
		"user_id":       user.ID,
		"club_id":       user.ClubID,
		"email":         user.Email,
		"hanko_user_id": user.HankoUserID,
		"timestamp":     time.Now().UTC(),
	}

	go func() {
		ctx := context.Background()
		if err := s.messageBus.Publish(ctx, eventType, event); err != nil {
			s.logger.Error("Failed to publish user event", map[string]interface{}{
				"error":      err.Error(),
				"event_type": eventType,
				"user_id":    user.ID,
			})
		}
	}()
}

func (s *AuthService) getIPFromContext(ctx context.Context) string {
	// Extract IP from context (set by middleware)
	if ip, ok := ctx.Value("client_ip").(string); ok {
		return ip
	}
	return "unknown"
}

func (s *AuthService) getUserAgentFromContext(ctx context.Context) string {
	// Extract user agent from context (set by middleware)
	if ua, ok := ctx.Value("user_agent").(string); ok {
		return ua
	}
	return "unknown"
}

// GetUser retrieves a user by ID
func (s *AuthService) GetUser(ctx context.Context, clubID, userID uint) (*models.User, error) {
	return s.repo.GetUserByID(ctx, clubID, userID)
}

// InitiatePasskeyAuthentication is a wrapper for InitiatePasskeyLogin to match gRPC interface
func (s *AuthService) InitiatePasskeyAuthentication(ctx context.Context, req *LoginRequest) (*PasskeyResponse, error) {
	return s.InitiatePasskeyLogin(ctx, req)
}

// CompletePasskeyAuthentication is a wrapper for CompletePasskeyLogin to match gRPC interface
func (s *AuthService) CompletePasskeyAuthentication(ctx context.Context, clubSlug string, hankoUserID string, credentialResult map[string]interface{}) (*AuthResponse, error) {
	return s.CompletePasskeyLogin(ctx, clubSlug, hankoUserID, credentialResult)
}

// HealthCheck performs health check on dependencies
func (s *AuthService) HealthCheck(ctx context.Context) error {
	// Check Hanko service
	if err := s.hankoClient.HealthCheck(ctx); err != nil {
		return fmt.Errorf("Hanko service health check failed: %w", err)
	}

	return nil
}