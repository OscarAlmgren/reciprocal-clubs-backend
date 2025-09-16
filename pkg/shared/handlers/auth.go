package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"reciprocal-clubs-backend/pkg/shared/auth"
	"reciprocal-clubs-backend/pkg/shared/errors"
	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/utils"
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	*HTTPHandler
	authProvider auth.AuthProvider
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(logger logging.Logger, authProvider auth.AuthProvider) *AuthHandler {
	return &AuthHandler{
		HTTPHandler:  NewHTTPHandler(logger),
		authProvider: authProvider,
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	ClubID   uint   `json:"club_id,omitempty"`
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Username  string `json:"username" validate:"required,min=3,max=50"`
	Password  string `json:"password" validate:"required,min=8"`
	FirstName string `json:"first_name" validate:"required,min=1,max=100"`
	LastName  string `json:"last_name" validate:"required,min=1,max=100"`
	ClubID    uint   `json:"club_id" validate:"required,min=1"`
	Phone     string `json:"phone,omitempty"`
}

// RefreshRequest represents a token refresh request
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// AuthResponse represents an authentication response
type AuthResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
	User         *UserInfo `json:"user"`
}

// UserInfo represents user information in auth response
type UserInfo struct {
	ID          uint     `json:"id"`
	ClubID      uint     `json:"club_id"`
	Email       string   `json:"email"`
	Username    string   `json:"username"`
	FirstName   string   `json:"first_name,omitempty"`
	LastName    string   `json:"last_name,omitempty"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := h.ParseJSONBody(r, &req); err != nil {
		h.WriteError(w, r, err)
		return
	}

	// Validate request
	if err := h.validateLoginRequest(&req); err != nil {
		h.WriteError(w, r, err)
		return
	}

	// Here you would typically:
	// 1. Validate credentials against user service/database
	// 2. Check if user is active
	// 3. Generate tokens
	// 4. Log the login attempt

	// For now, we'll create a mock implementation
	user, err := h.authenticateUser(r.Context(), req.Email, req.Password, req.ClubID)
	if err != nil {
		h.logger.Warn("Login attempt failed", map[string]interface{}{
			"email":   req.Email,
			"club_id": req.ClubID,
			"error":   err.Error(),
		})
		h.WriteError(w, r, err)
		return
	}

	// Generate access token
	accessToken, err := h.authProvider.GenerateToken(user, 24*time.Hour)
	if err != nil {
		h.logger.Error("Failed to generate access token", map[string]interface{}{
			"user_id": user.ID,
			"error":   err.Error(),
		})
		h.WriteError(w, r, errors.Internal("Token generation failed", nil, err))
		return
	}

	// Generate refresh token (optional - you might store this differently)
	refreshToken, err := h.authProvider.GenerateToken(user, 7*24*time.Hour) // 7 days
	if err != nil {
		h.logger.Error("Failed to generate refresh token", map[string]interface{}{
			"user_id": user.ID,
			"error":   err.Error(),
		})
		// Continue without refresh token
		refreshToken = ""
	}

	response := AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		TokenType:    "Bearer",
		User: &UserInfo{
			ID:          user.ID,
			ClubID:      user.ClubID,
			Email:       user.Email,
			Username:    user.Username,
			Roles:       user.Roles,
			Permissions: user.Permissions,
		},
	}

	h.logger.Info("User logged in successfully", map[string]interface{}{
		"user_id": user.ID,
		"club_id": user.ClubID,
		"email":   user.Email,
	})

	h.WriteResponse(w, r, http.StatusOK, response)
}

// Register handles user registration
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := h.ParseJSONBody(r, &req); err != nil {
		h.WriteError(w, r, err)
		return
	}

	// Validate request
	if err := h.validateRegisterRequest(&req); err != nil {
		h.WriteError(w, r, err)
		return
	}

	// Here you would typically:
	// 1. Check if user already exists
	// 2. Validate club exists
	// 3. Hash password
	// 4. Create user record
	// 5. Send verification email (optional)
	// 6. Generate initial tokens

	// For now, create a mock implementation
	user, err := h.createUser(r.Context(), &req)
	if err != nil {
		h.logger.Error("User registration failed", map[string]interface{}{
			"email":   req.Email,
			"club_id": req.ClubID,
			"error":   err.Error(),
		})
		h.WriteError(w, r, err)
		return
	}

	// Generate access token for the new user
	accessToken, err := h.authProvider.GenerateToken(user, 24*time.Hour)
	if err != nil {
		h.logger.Error("Failed to generate token for new user", map[string]interface{}{
			"user_id": user.ID,
			"error":   err.Error(),
		})
		h.WriteError(w, r, errors.Internal("Token generation failed", nil, err))
		return
	}

	response := AuthResponse{
		AccessToken: accessToken,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		TokenType:   "Bearer",
		User: &UserInfo{
			ID:          user.ID,
			ClubID:      user.ClubID,
			Email:       user.Email,
			Username:    user.Username,
			Roles:       user.Roles,
			Permissions: user.Permissions,
		},
	}

	h.logger.Info("User registered successfully", map[string]interface{}{
		"user_id": user.ID,
		"club_id": user.ClubID,
		"email":   user.Email,
	})

	h.WriteResponse(w, r, http.StatusCreated, response)
}

// RefreshToken handles token refresh
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := h.ParseJSONBody(r, &req); err != nil {
		h.WriteError(w, r, err)
		return
	}

	if strings.TrimSpace(req.RefreshToken) == "" {
		err := errors.InvalidInput("Refresh token is required", nil, nil)
		h.WriteError(w, r, err)
		return
	}

	// Refresh the token
	newAccessToken, err := h.authProvider.RefreshToken(req.RefreshToken)
	if err != nil {
		h.logger.Warn("Token refresh failed", map[string]interface{}{
			"error": err.Error(),
		})
		h.WriteError(w, r, errors.Unauthorized("Invalid refresh token", nil))
		return
	}

	// Validate the new token to get user info
	claims, err := h.authProvider.ValidateToken(newAccessToken)
	if err != nil {
		h.logger.Error("Generated token validation failed", map[string]interface{}{
			"error": err.Error(),
		})
		h.WriteError(w, r, errors.Internal("Token validation failed", nil, err))
		return
	}

	response := AuthResponse{
		AccessToken: newAccessToken,
		ExpiresAt:   claims.ExpiresAt.Time,
		TokenType:   "Bearer",
		User: &UserInfo{
			ID:          claims.UserID,
			ClubID:      claims.ClubID,
			Email:       claims.Email,
			Username:    claims.Username,
			Roles:       claims.Roles,
			Permissions: claims.Permissions,
		},
	}

	h.logger.Info("Token refreshed successfully", map[string]interface{}{
		"user_id": claims.UserID,
	})

	h.WriteResponse(w, r, http.StatusOK, response)
}

// Logout handles user logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Extract token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		h.WriteError(w, r, errors.Unauthorized("Authorization header required", nil))
		return
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		h.WriteError(w, r, errors.Unauthorized("Invalid authorization header format", nil))
		return
	}

	token := parts[1]

	// Revoke the token
	if err := h.authProvider.RevokeToken(token); err != nil {
		h.logger.Error("Token revocation failed", map[string]interface{}{
			"error": err.Error(),
		})
		// Don't fail the logout even if revocation fails
	}

	// Get user from context for logging
	user := auth.GetUserFromContext(r.Context())
	if user != nil {
		h.logger.Info("User logged out", map[string]interface{}{
			"user_id": user.ID,
			"club_id": user.ClubID,
		})
	}

	response := map[string]interface{}{
		"message": "Logged out successfully",
	}

	h.WriteResponse(w, r, http.StatusOK, response)
}

// Me returns current user information
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		h.WriteError(w, r, errors.Unauthorized("Authentication required", nil))
		return
	}

	userInfo := UserInfo{
		ID:          user.ID,
		ClubID:      user.ClubID,
		Email:       user.Email,
		Username:    user.Username,
		Roles:       user.Roles,
		Permissions: user.Permissions,
	}

	h.WriteResponse(w, r, http.StatusOK, userInfo)
}

// ChangePassword handles password change requests
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		h.WriteError(w, r, errors.Unauthorized("Authentication required", nil))
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password" validate:"required"`
		NewPassword     string `json:"new_password" validate:"required,min=8"`
	}

	if err := h.ParseJSONBody(r, &req); err != nil {
		h.WriteError(w, r, err)
		return
	}

	// Validate new password
	if !utils.IsValidPassword(req.NewPassword) {
		err := errors.InvalidInput("New password does not meet requirements", map[string]interface{}{
			"requirements": "At least 8 characters with uppercase, lowercase, and numbers",
		}, nil)
		h.WriteError(w, r, err)
		return
	}

	// Here you would:
	// 1. Verify current password
	// 2. Hash new password
	// 3. Update password in database
	// 4. Optionally revoke all existing tokens

	h.logger.Info("Password changed successfully", map[string]interface{}{
		"user_id": user.ID,
	})

	response := map[string]interface{}{
		"message": "Password changed successfully",
	}

	h.WriteResponse(w, r, http.StatusOK, response)
}

// Validation functions

func (h *AuthHandler) validateLoginRequest(req *LoginRequest) error {
	if strings.TrimSpace(req.Email) == "" {
		return errors.InvalidInput("Email is required", nil, nil)
	}

	if !utils.IsValidEmail(req.Email) {
		return errors.InvalidInput("Invalid email format", nil, nil)
	}

	if strings.TrimSpace(req.Password) == "" {
		return errors.InvalidInput("Password is required", nil, nil)
	}

	if len(req.Password) < 8 {
		return errors.InvalidInput("Password must be at least 8 characters", nil, nil)
	}

	return nil
}

func (h *AuthHandler) validateRegisterRequest(req *RegisterRequest) error {
	if strings.TrimSpace(req.Email) == "" {
		return errors.InvalidInput("Email is required", nil, nil)
	}

	if !utils.IsValidEmail(req.Email) {
		return errors.InvalidInput("Invalid email format", nil, nil)
	}

	if strings.TrimSpace(req.Username) == "" {
		return errors.InvalidInput("Username is required", nil, nil)
	}

	if !utils.IsValidUsername(req.Username) {
		return errors.InvalidInput("Invalid username format", map[string]interface{}{
			"requirements": "3-50 characters, alphanumeric, underscores, and hyphens only",
		}, nil)
	}

	if strings.TrimSpace(req.Password) == "" {
		return errors.InvalidInput("Password is required", nil, nil)
	}

	if !utils.IsValidPassword(req.Password) {
		return errors.InvalidInput("Password does not meet requirements", map[string]interface{}{
			"requirements": "At least 8 characters with uppercase, lowercase, and numbers",
		}, nil)
	}

	if strings.TrimSpace(req.FirstName) == "" {
		return errors.InvalidInput("First name is required", nil, nil)
	}

	if strings.TrimSpace(req.LastName) == "" {
		return errors.InvalidInput("Last name is required", nil, nil)
	}

	if req.ClubID == 0 {
		return errors.InvalidInput("Club ID is required", nil, nil)
	}

	if req.Phone != "" && !utils.IsValidPhoneNumber(req.Phone) {
		return errors.InvalidInput("Invalid phone number format", nil, nil)
	}

	return nil
}

// Mock functions (these would be replaced with actual service calls)

func (h *AuthHandler) authenticateUser(ctx context.Context, email, password string, clubID uint) (*auth.User, error) {
	// This is a mock implementation
	// In a real implementation, you would:
	// 1. Query the user service/database
	// 2. Verify password hash
	// 3. Check user status
	// 4. Load user roles and permissions

	// Mock successful authentication
	user := &auth.User{
		ID:          12345,
		ClubID:      clubID,
		Email:       email,
		Username:    strings.Split(email, "@")[0],
		Roles:       []string{"member"},
		Permissions: []string{"read:profile", "update:profile"},
	}

	// If clubID is 0, use default club
	if user.ClubID == 0 {
		user.ClubID = 1
	}

	return user, nil
}

func (h *AuthHandler) createUser(ctx context.Context, req *RegisterRequest) (*auth.User, error) {
	// This is a mock implementation
	// In a real implementation, you would:
	// 1. Check if email/username already exists
	// 2. Validate club exists
	// 3. Hash password
	// 4. Create user record in database
	// 5. Set up default roles/permissions

	// Mock user creation
	user := &auth.User{
		ID:          67890, // Mock generated ID
		ClubID:      req.ClubID,
		Email:       req.Email,
		Username:    req.Username,
		Roles:       []string{"member"}, // Default role
		Permissions: []string{"read:profile", "update:profile"},
	}

	return user, nil
}
