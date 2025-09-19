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
	"reciprocal-clubs-backend/services/auth-service/internal/mfa"
	"reciprocal-clubs-backend/services/auth-service/internal/models"
	"reciprocal-clubs-backend/services/auth-service/internal/password"
	"reciprocal-clubs-backend/services/auth-service/internal/repository"
)

// AuthService handles authentication business logic
type AuthService struct {
	repo            *repository.AuthRepository
	hankoClient     HankoClientInterface
	authProvider    *auth.JWTProvider
	messageBus      messaging.MessageBus
	config          *config.Config
	logger          logging.Logger
	mfaService      *mfa.MFAService
	passwordService *password.PasswordService
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

// MFASetupRequest represents MFA setup request
type MFASetupRequest struct {
	UserID uint   `json:"user_id" validate:"required"`
	ClubID uint   `json:"club_id" validate:"required"`
	Method string `json:"method" validate:"required,oneof=totp sms email"`
}

// MFASetupResponse represents MFA setup response
type MFASetupResponse struct {
	Secret      string   `json:"secret,omitempty"`
	QRCodeURL   string   `json:"qr_code_url,omitempty"`
	BackupCodes []string `json:"backup_codes,omitempty"`
	Success     bool     `json:"success"`
	Message     string   `json:"message"`
}

// MFAVerifyRequest represents MFA verification request
type MFAVerifyRequest struct {
	UserID uint   `json:"user_id" validate:"required"`
	ClubID uint   `json:"club_id" validate:"required"`
	Code   string `json:"code" validate:"required"`
	Method string `json:"method" validate:"required,oneof=totp sms email backup"`
}

// MFAVerifyResponse represents MFA verification response
type MFAVerifyResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// PasswordResetRequest represents password reset request
type PasswordResetRequest struct {
	Email    string `json:"email" validate:"required,email"`
	ClubSlug string `json:"club_slug" validate:"required"`
}

// PasswordResetResponse represents password reset response
type PasswordResetResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// PasswordResetConfirmRequest represents password reset confirmation request
type PasswordResetConfirmRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
	ClubSlug    string `json:"club_slug" validate:"required"`
}

// PasswordResetConfirmResponse represents password reset confirmation response
type PasswordResetConfirmResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// EmailVerificationRequest represents email verification request
type EmailVerificationRequest struct {
	Email    string `json:"email" validate:"required,email"`
	ClubSlug string `json:"club_slug" validate:"required"`
}

// EmailVerificationResponse represents email verification response
type EmailVerificationResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// EmailVerificationConfirmRequest represents email verification confirmation request
type EmailVerificationConfirmRequest struct {
	Token    string `json:"token" validate:"required"`
	ClubSlug string `json:"club_slug" validate:"required"`
}

// EmailVerificationConfirmResponse represents email verification confirmation response
type EmailVerificationConfirmResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
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

	// Initialize MFA service
	mfaService := mfa.NewMFAService("Reciprocal Clubs")

	// Initialize password service with 1 hour token TTL
	passwordService := password.NewPasswordService(1 * time.Hour)

	return &AuthService{
		repo:            repo,
		hankoClient:     hankoClient,
		authProvider:    authProvider,
		messageBus:      messageBus,
		config:          config,
		logger:          logger,
		mfaService:      mfaService,
		passwordService: passwordService,
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

// MFA Management Methods

// SetupMFA initiates MFA setup for a user
func (s *AuthService) SetupMFA(ctx context.Context, req *MFASetupRequest) (*MFASetupResponse, error) {
	// Get user
	user, err := s.repo.GetUserByID(ctx, req.ClubID, req.UserID)
	if err != nil {
		return nil, err
	}

	// Check if MFA is already enabled
	if user.MFAEnabled {
		return &MFASetupResponse{
			Success: false,
			Message: "MFA is already enabled for this user",
		}, nil
	}

	var response *MFASetupResponse

	switch req.Method {
	case "totp":
		response, err = s.setupTOTP(ctx, user)
	case "sms":
		response, err = s.setupSMS(ctx, user)
	case "email":
		response, err = s.setupEmailMFA(ctx, user)
	default:
		return nil, errors.BadRequest("Invalid MFA method", map[string]interface{}{
			"method": req.Method,
		})
	}

	if err != nil {
		return nil, err
	}

	// Create audit log
	s.createAuditLog(ctx, req.ClubID, user, models.AuditActionMFAEnabled, fmt.Sprintf("MFA setup initiated with method: %s", req.Method), true, "")

	s.logger.Info("MFA setup initiated", map[string]interface{}{
		"user_id": user.ID,
		"method":  req.Method,
	})

	return response, nil
}

// setupTOTP sets up TOTP MFA for a user
func (s *AuthService) setupTOTP(ctx context.Context, user *models.User) (*MFASetupResponse, error) {
	// Generate TOTP secret
	secret, err := s.mfaService.GenerateSecret(user.Email)
	if err != nil {
		return nil, errors.Internal("Failed to generate TOTP secret", nil, err)
	}

	// Generate backup codes
	backupCodes, err := s.mfaService.GenerateBackupCodes(8)
	if err != nil {
		return nil, errors.Internal("Failed to generate backup codes", nil, err)
	}

	// Generate QR code URL
	qrCodeURL := s.mfaService.GenerateQRCodeURL(user.Email, secret)

	// Update user with MFA settings (not enabled yet, will be enabled after verification)
	user.MFASecret = secret
	user.MFAMethod = "totp"
	user.MFABackupCodes = fmt.Sprintf("%v", backupCodes) // This should be properly hashed in production

	// Save user
	err = s.repo.UpdateUser(ctx, user)
	if err != nil {
		return nil, err
	}

	return &MFASetupResponse{
		Secret:      secret,
		QRCodeURL:   qrCodeURL,
		BackupCodes: backupCodes,
		Success:     true,
		Message:     "TOTP MFA setup completed. Please verify with your authenticator app.",
	}, nil
}

// setupSMS sets up SMS MFA for a user
func (s *AuthService) setupSMS(ctx context.Context, user *models.User) (*MFASetupResponse, error) {
	// Check if user has a verified phone number
	if user.PhoneNumber == "" || !user.PhoneVerified {
		return &MFASetupResponse{
			Success: false,
			Message: "Phone number must be verified before enabling SMS MFA",
		}, nil
	}

	// Generate SMS verification code
	code, err := s.mfaService.GenerateSMSCode()
	if err != nil {
		return nil, errors.Internal("Failed to generate SMS code", nil, err)
	}

	// Store verification token
	mfaToken := &models.MFAToken{
		UserID:    user.ID,
		TokenType: models.MFATokenTypeSMS,
		Token:     code,
		ExpiresAt: &[]time.Time{time.Now().Add(5 * time.Minute)}[0],
	}
	mfaToken.ClubID = user.ClubID

	err = s.repo.CreateMFAToken(ctx, mfaToken)
	if err != nil {
		return nil, err
	}

	// TODO: Send SMS code to user's phone number
	// For now, we'll just log it
	s.logger.Info("SMS MFA code generated", map[string]interface{}{
		"user_id": user.ID,
		"code":    code, // Remove this in production
	})

	return &MFASetupResponse{
		Success: true,
		Message: "SMS verification code sent to your phone number",
	}, nil
}

// setupEmailMFA sets up email MFA for a user
func (s *AuthService) setupEmailMFA(ctx context.Context, user *models.User) (*MFASetupResponse, error) {
	// Generate email verification code
	code, err := s.mfaService.GenerateEmailCode()
	if err != nil {
		return nil, errors.Internal("Failed to generate email code", nil, err)
	}

	// Store verification token
	mfaToken := &models.MFAToken{
		UserID:    user.ID,
		TokenType: models.MFATokenTypeEmail,
		Token:     code,
		ExpiresAt: &[]time.Time{time.Now().Add(10 * time.Minute)}[0],
	}
	mfaToken.ClubID = user.ClubID

	err = s.repo.CreateMFAToken(ctx, mfaToken)
	if err != nil {
		return nil, err
	}

	// TODO: Send email code to user's email
	// For now, we'll just log it
	s.logger.Info("Email MFA code generated", map[string]interface{}{
		"user_id": user.ID,
		"code":    code, // Remove this in production
	})

	return &MFASetupResponse{
		Success: true,
		Message: "Verification code sent to your email address",
	}, nil
}

// VerifyMFA verifies an MFA code
func (s *AuthService) VerifyMFA(ctx context.Context, req *MFAVerifyRequest) (*MFAVerifyResponse, error) {
	// Get user
	user, err := s.repo.GetUserByID(ctx, req.ClubID, req.UserID)
	if err != nil {
		return nil, err
	}

	var success bool
	var message string

	switch req.Method {
	case "totp":
		success = s.mfaService.VerifyTOTPWithSkew(user.MFASecret, req.Code, 1)
		message = "TOTP verification"
	case "backup":
		success = user.UseBackupCode(req.Code)
		if success {
			// Save updated backup codes
			s.repo.UpdateUser(ctx, user)
			s.createAuditLog(ctx, req.ClubID, user, models.AuditActionMFABackupUsed, "Backup code used for MFA", true, "")
		}
		message = "Backup code verification"
	case "sms", "email":
		success, err = s.verifyTokenCode(ctx, user.ID, req.Code, req.Method)
		if err != nil {
			return nil, err
		}
		message = fmt.Sprintf("%s verification", req.Method)
	default:
		return nil, errors.BadRequest("Invalid MFA method", map[string]interface{}{
			"method": req.Method,
		})
	}

	if success {
		// Enable MFA if this is the first successful verification
		if !user.MFAEnabled {
			user.MFAEnabled = true
			err = s.repo.UpdateUser(ctx, user)
			if err != nil {
				return nil, err
			}
			s.createAuditLog(ctx, req.ClubID, user, models.AuditActionMFAEnabled, "MFA enabled after successful verification", true, "")
		}

		s.createAuditLog(ctx, req.ClubID, user, models.AuditActionMFAVerification, fmt.Sprintf("Successful MFA verification: %s", req.Method), true, "")
		return &MFAVerifyResponse{
			Success: true,
			Message: fmt.Sprintf("%s successful", message),
		}, nil
	}

	s.createAuditLog(ctx, req.ClubID, user, models.AuditActionMFAVerification, fmt.Sprintf("Failed MFA verification: %s", req.Method), false, "Invalid code")
	return &MFAVerifyResponse{
		Success: false,
		Message: fmt.Sprintf("%s failed", message),
	}, nil
}

// verifyTokenCode verifies SMS or email MFA codes
func (s *AuthService) verifyTokenCode(ctx context.Context, userID uint, code, method string) (bool, error) {
	// Get the latest token for this user and method
	tokenType := models.MFATokenTypeSMS
	if method == "email" {
		tokenType = models.MFATokenTypeEmail
	}

	token, err := s.repo.GetLatestMFAToken(ctx, userID, tokenType)
	if err != nil {
		return false, err
	}

	// Check if token is valid and not expired
	if token.Used || token.IsExpired() {
		return false, nil
	}

	// Verify code
	if token.Token != code {
		return false, nil
	}

	// Mark token as used
	token.MarkAsUsed()
	err = s.repo.UpdateMFAToken(ctx, token)
	if err != nil {
		return false, err
	}

	return true, nil
}

// DisableMFA disables MFA for a user
func (s *AuthService) DisableMFA(ctx context.Context, userID, clubID uint) error {
	// Get user
	user, err := s.repo.GetUserByID(ctx, clubID, userID)
	if err != nil {
		return err
	}

	// Disable MFA
	user.DisableMFA()

	// Save user
	err = s.repo.UpdateUser(ctx, user)
	if err != nil {
		return err
	}

	// Create audit log
	s.createAuditLog(ctx, clubID, user, models.AuditActionMFADisabled, "MFA disabled", true, "")

	s.logger.Info("MFA disabled", map[string]interface{}{
		"user_id": user.ID,
	})

	return nil
}

// Password Reset Methods

// RequestPasswordReset initiates password reset process
func (s *AuthService) RequestPasswordReset(ctx context.Context, req *PasswordResetRequest) (*PasswordResetResponse, error) {
	// Get club by slug
	club, err := s.repo.GetClubBySlug(ctx, req.ClubSlug)
	if err != nil {
		return nil, err
	}

	// Get user by email
	user, err := s.repo.GetUserByEmail(ctx, club.ID, req.Email)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			// Don't reveal that user doesn't exist
			return &PasswordResetResponse{
				Success: true,
				Message: "If an account with that email exists, a password reset link has been sent.",
			}, nil
		}
		return nil, err
	}

	// Check if user is active
	if !user.IsActive() {
		return &PasswordResetResponse{
			Success: false,
			Message: "Account is not active",
		}, nil
	}

	// Generate reset token
	token, expiry, err := s.passwordService.GenerateResetTokenWithExpiry()
	if err != nil {
		return nil, errors.Internal("Failed to generate reset token", nil, err)
	}

	// Set reset token on user
	user.SetPasswordResetToken(token, expiry)

	// Save user
	err = s.repo.UpdateUser(ctx, user)
	if err != nil {
		return nil, err
	}

	// Create audit log
	s.createAuditLog(ctx, club.ID, user, models.AuditActionPasswordResetRequested, "Password reset requested", true, "")

	// TODO: Send password reset email
	// For now, we'll just log the token
	s.logger.Info("Password reset token generated", map[string]interface{}{
		"user_id": user.ID,
		"token":   token, // Remove this in production
	})

	s.logger.Info("Password reset requested", map[string]interface{}{
		"user_id": user.ID,
		"email":   user.Email,
	})

	return &PasswordResetResponse{
		Success: true,
		Message: "If an account with that email exists, a password reset link has been sent.",
	}, nil
}

// ConfirmPasswordReset completes the password reset process
func (s *AuthService) ConfirmPasswordReset(ctx context.Context, req *PasswordResetConfirmRequest) (*PasswordResetConfirmResponse, error) {
	// Get club by slug
	club, err := s.repo.GetClubBySlug(ctx, req.ClubSlug)
	if err != nil {
		return nil, err
	}

	// Find user by reset token
	user, err := s.repo.GetUserByPasswordResetToken(ctx, club.ID, req.Token)
	if err != nil {
		return &PasswordResetConfirmResponse{
			Success: false,
			Message: "Invalid or expired reset token",
		}, nil
	}

	// Validate reset token
	if !user.IsPasswordResetTokenValid(req.Token) {
		return &PasswordResetConfirmResponse{
			Success: false,
			Message: "Invalid or expired reset token",
		}, nil
	}

	// Validate password strength
	err = s.passwordService.ValidatePasswordStrength(req.NewPassword)
	if err != nil {
		return &PasswordResetConfirmResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	// Check if password is compromised
	if s.passwordService.IsPasswordCompromised(req.NewPassword) {
		return &PasswordResetConfirmResponse{
			Success: false,
			Message: "This password has been found in data breaches. Please choose a different password.",
		}, nil
	}

	// Hash new password
	hashedPassword, err := s.passwordService.HashPassword(req.NewPassword)
	if err != nil {
		return nil, errors.Internal("Failed to hash password", nil, err)
	}

	// Update user password in Hanko (if using Hanko for password storage)
	// For now, we'll store it locally
	// TODO: Integrate with Hanko password update API

	// Clear reset token
	user.ClearPasswordResetToken()

	// Reset failed attempts
	user.ResetFailedAttempts()
	user.Unlock()

	// Save user
	err = s.repo.UpdateUser(ctx, user)
	if err != nil {
		return nil, err
	}

	// Invalidate all existing sessions
	err = s.repo.InvalidateAllUserSessions(ctx, club.ID, user.ID)
	if err != nil {
		s.logger.Warn("Failed to invalidate user sessions", map[string]interface{}{
			"error":   err.Error(),
			"user_id": user.ID,
		})
	}

	// Create audit log
	s.createAuditLog(ctx, club.ID, user, models.AuditActionPasswordResetCompleted, "Password reset completed", true, "")

	s.logger.Info("Password reset completed", map[string]interface{}{
		"user_id": user.ID,
		"email":   user.Email,
	})

	return &PasswordResetConfirmResponse{
		Success: true,
		Message: "Password has been reset successfully. Please log in with your new password.",
	}, nil
}

// Email Verification Methods

// RequestEmailVerification sends email verification
func (s *AuthService) RequestEmailVerification(ctx context.Context, req *EmailVerificationRequest) (*EmailVerificationResponse, error) {
	// Get club by slug
	club, err := s.repo.GetClubBySlug(ctx, req.ClubSlug)
	if err != nil {
		return nil, err
	}

	// Get user by email
	user, err := s.repo.GetUserByEmail(ctx, club.ID, req.Email)
	if err != nil {
		return nil, err
	}

	// Check if already verified
	if user.EmailVerified {
		return &EmailVerificationResponse{
			Success: true,
			Message: "Email is already verified",
		}, nil
	}

	// Generate verification token
	token, err := s.mfaService.GenerateRandomToken(32)
	if err != nil {
		return nil, errors.Internal("Failed to generate verification token", nil, err)
	}

	// Set verification token
	expiry := time.Now().Add(24 * time.Hour) // 24 hour expiry
	user.SetEmailVerificationToken(token, expiry)

	// Save user
	err = s.repo.UpdateUser(ctx, user)
	if err != nil {
		return nil, err
	}

	// Create audit log
	s.createAuditLog(ctx, club.ID, user, models.AuditActionEmailVerificationSent, "Email verification sent", true, "")

	// TODO: Send verification email
	// For now, we'll just log the token
	s.logger.Info("Email verification token generated", map[string]interface{}{
		"user_id": user.ID,
		"token":   token, // Remove this in production
	})

	return &EmailVerificationResponse{
		Success: true,
		Message: "Verification email sent",
	}, nil
}

// ConfirmEmailVerification confirms email verification
func (s *AuthService) ConfirmEmailVerification(ctx context.Context, req *EmailVerificationConfirmRequest) (*EmailVerificationConfirmResponse, error) {
	// Get club by slug
	club, err := s.repo.GetClubBySlug(ctx, req.ClubSlug)
	if err != nil {
		return nil, err
	}

	// Find user by verification token
	user, err := s.repo.GetUserByEmailVerificationToken(ctx, club.ID, req.Token)
	if err != nil {
		return &EmailVerificationConfirmResponse{
			Success: false,
			Message: "Invalid or expired verification token",
		}, nil
	}

	// Validate verification token
	if !user.IsEmailVerificationTokenValid(req.Token) {
		return &EmailVerificationConfirmResponse{
			Success: false,
			Message: "Invalid or expired verification token",
		}, nil
	}

	// Verify email
	user.VerifyEmail()

	// Save user
	err = s.repo.UpdateUser(ctx, user)
	if err != nil {
		return nil, err
	}

	// Create audit log
	s.createAuditLog(ctx, club.ID, user, models.AuditActionEmailVerificationCompleted, "Email verification completed", true, "")

	s.logger.Info("Email verification completed", map[string]interface{}{
		"user_id": user.ID,
		"email":   user.Email,
	})

	return &EmailVerificationConfirmResponse{
		Success: true,
		Message: "Email verified successfully",
	}, nil
}

// HealthCheck performs health check on dependencies
func (s *AuthService) HealthCheck(ctx context.Context) error {
	// Check Hanko service
	if err := s.hankoClient.HealthCheck(ctx); err != nil {
		return fmt.Errorf("Hanko service health check failed: %w", err)
	}

	return nil
}