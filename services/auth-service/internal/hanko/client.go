package hanko

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"reciprocal-clubs-backend/pkg/shared/logging"
)

// HankoClient provides integration with Hanko passkey service
type HankoClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	logger     logging.Logger
}

// Config represents Hanko client configuration
type Config struct {
	BaseURL    string        `json:"base_url"`
	APIKey     string        `json:"api_key"`
	Timeout    time.Duration `json:"timeout"`
	RetryCount int          `json:"retry_count"`
}

// User represents a Hanko user
type HankoUser struct {
	ID             string     `json:"id"`
	Email          string     `json:"email"`
	EmailVerified  bool       `json:"email_verified"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	WebauthnID     string     `json:"webauthn_id,omitempty"`
	HasPasskey     bool       `json:"has_passkey"`
}

// Session represents a Hanko session
type HankoSession struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateUserRequest represents the request to create a user
type CreateUserRequest struct {
	Email string `json:"email"`
}

// CreateUserResponse represents the response from creating a user
type CreateUserResponse struct {
	User   HankoUser `json:"user"`
	Status string    `json:"status"`
}

// ValidateSessionRequest represents session validation request
type ValidateSessionRequest struct {
	SessionToken string `json:"session_token"`
}

// ValidateSessionResponse represents session validation response
type ValidateSessionResponse struct {
	Valid   bool         `json:"valid"`
	Session HankoSession `json:"session,omitempty"`
	User    HankoUser    `json:"user,omitempty"`
}

// PasskeyRegistrationRequest represents passkey registration request
type PasskeyRegistrationRequest struct {
	UserID string `json:"user_id"`
}

// PasskeyRegistrationResponse represents passkey registration response
type PasskeyRegistrationResponse struct {
	RegistrationOptions map[string]interface{} `json:"registration_options"`
}

// PasskeyAuthenticationRequest represents passkey authentication request
type PasskeyAuthenticationRequest struct {
	UserEmail string `json:"user_email,omitempty"`
	UserID    string `json:"user_id,omitempty"`
}

// PasskeyAuthenticationResponse represents passkey authentication response
type PasskeyAuthenticationResponse struct {
	AuthenticationOptions map[string]interface{} `json:"authentication_options"`
}

// VerifyPasskeyRequest represents passkey verification request
type VerifyPasskeyRequest struct {
	UserID           string                 `json:"user_id"`
	CredentialResult map[string]interface{} `json:"credential_result"`
}

// VerifyPasskeyResponse represents passkey verification response
type VerifyPasskeyResponse struct {
	Success     bool         `json:"success"`
	Session     HankoSession `json:"session,omitempty"`
	ErrorCode   string       `json:"error_code,omitempty"`
	ErrorDetail string       `json:"error_detail,omitempty"`
}

// WebhookEvent represents a Hanko webhook event
type WebhookEvent struct {
	Type      string                 `json:"type"`
	UserID    string                 `json:"user_id"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// NewHankoClient creates a new Hanko client
func NewHankoClient(config Config, logger logging.Logger) *HankoClient {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &HankoClient{
		baseURL: config.BaseURL,
		apiKey:  config.APIKey,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		logger: logger,
	}
}

// CreateUser creates a new user in Hanko
func (c *HankoClient) CreateUser(ctx context.Context, email string) (*HankoUser, error) {
	req := CreateUserRequest{
		Email: email,
	}

	var resp CreateUserResponse
	if err := c.makeRequest(ctx, "POST", "/users", req, &resp); err != nil {
		return nil, fmt.Errorf("failed to create user in Hanko: %w", err)
	}

	c.logger.Info("User created in Hanko", map[string]interface{}{
		"hanko_user_id": resp.User.ID,
		"email":         resp.User.Email,
	})

	return &resp.User, nil
}

// GetUser retrieves a user from Hanko
func (c *HankoClient) GetUser(ctx context.Context, userID string) (*HankoUser, error) {
	var user HankoUser
	if err := c.makeRequest(ctx, "GET", fmt.Sprintf("/users/%s", userID), nil, &user); err != nil {
		return nil, fmt.Errorf("failed to get user from Hanko: %w", err)
	}

	return &user, nil
}

// GetUserByEmail retrieves a user from Hanko by email
func (c *HankoClient) GetUserByEmail(ctx context.Context, email string) (*HankoUser, error) {
	var user HankoUser
	if err := c.makeRequest(ctx, "GET", fmt.Sprintf("/users?email=%s", email), nil, &user); err != nil {
		return nil, fmt.Errorf("failed to get user by email from Hanko: %w", err)
	}

	return &user, nil
}

// ValidateSession validates a Hanko session token
func (c *HankoClient) ValidateSession(ctx context.Context, sessionToken string) (*ValidateSessionResponse, error) {
	req := ValidateSessionRequest{
		SessionToken: sessionToken,
	}

	var resp ValidateSessionResponse
	if err := c.makeRequest(ctx, "POST", "/sessions/validate", req, &resp); err != nil {
		return nil, fmt.Errorf("failed to validate session: %w", err)
	}

	return &resp, nil
}

// InitiatePasskeyRegistration starts passkey registration process
func (c *HankoClient) InitiatePasskeyRegistration(ctx context.Context, userID string) (*PasskeyRegistrationResponse, error) {
	req := PasskeyRegistrationRequest{
		UserID: userID,
	}

	var resp PasskeyRegistrationResponse
	if err := c.makeRequest(ctx, "POST", "/webauthn/registration/initialize", req, &resp); err != nil {
		return nil, fmt.Errorf("failed to initiate passkey registration: %w", err)
	}

	c.logger.Info("Passkey registration initiated", map[string]interface{}{
		"user_id": userID,
	})

	return &resp, nil
}

// InitiatePasskeyAuthentication starts passkey authentication process
func (c *HankoClient) InitiatePasskeyAuthentication(ctx context.Context, userEmail string) (*PasskeyAuthenticationResponse, error) {
	req := PasskeyAuthenticationRequest{
		UserEmail: userEmail,
	}

	var resp PasskeyAuthenticationResponse
	if err := c.makeRequest(ctx, "POST", "/webauthn/authentication/initialize", req, &resp); err != nil {
		return nil, fmt.Errorf("failed to initiate passkey authentication: %w", err)
	}

	c.logger.Info("Passkey authentication initiated", map[string]interface{}{
		"user_email": userEmail,
	})

	return &resp, nil
}

// VerifyPasskey verifies a passkey authentication result
func (c *HankoClient) VerifyPasskey(ctx context.Context, userID string, credentialResult map[string]interface{}) (*VerifyPasskeyResponse, error) {
	req := VerifyPasskeyRequest{
		UserID:           userID,
		CredentialResult: credentialResult,
	}

	var resp VerifyPasskeyResponse
	if err := c.makeRequest(ctx, "POST", "/webauthn/authentication/finalize", req, &resp); err != nil {
		return nil, fmt.Errorf("failed to verify passkey: %w", err)
	}

	if resp.Success {
		c.logger.Info("Passkey authentication successful", map[string]interface{}{
			"user_id":    userID,
			"session_id": resp.Session.ID,
		})
	} else {
		c.logger.Warn("Passkey authentication failed", map[string]interface{}{
			"user_id":      userID,
			"error_code":   resp.ErrorCode,
			"error_detail": resp.ErrorDetail,
		})
	}

	return &resp, nil
}

// InvalidateSession invalidates a Hanko session
func (c *HankoClient) InvalidateSession(ctx context.Context, sessionID string) error {
	if err := c.makeRequest(ctx, "DELETE", fmt.Sprintf("/sessions/%s", sessionID), nil, nil); err != nil {
		return fmt.Errorf("failed to invalidate session: %w", err)
	}

	c.logger.Info("Session invalidated", map[string]interface{}{
		"session_id": sessionID,
	})

	return nil
}

// DeleteUser deletes a user from Hanko
func (c *HankoClient) DeleteUser(ctx context.Context, userID string) error {
	if err := c.makeRequest(ctx, "DELETE", fmt.Sprintf("/users/%s", userID), nil, nil); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	c.logger.Info("User deleted from Hanko", map[string]interface{}{
		"user_id": userID,
	})

	return nil
}

// HealthCheck performs a health check on the Hanko service
func (c *HankoClient) HealthCheck(ctx context.Context) error {
	if err := c.makeRequest(ctx, "GET", "/health", nil, nil); err != nil {
		return fmt.Errorf("Hanko health check failed: %w", err)
	}

	return nil
}

// makeRequest makes an HTTP request to the Hanko API
func (c *HankoClient) makeRequest(ctx context.Context, method, path string, reqBody, respBody interface{}) error {
	url := c.baseURL + path

	var bodyReader *bytes.Reader
	if reqBody != nil {
		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	var req *http.Request
	var err error
	
	if bodyReader != nil {
		req, err = http.NewRequestWithContext(ctx, method, url, bodyReader)
	} else {
		req, err = http.NewRequestWithContext(ctx, method, url, nil)
	}
	
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	// Add correlation ID if available
	if correlationID := logging.GetCorrelationID(ctx); correlationID != "" {
		req.Header.Set("X-Correlation-ID", correlationID)
	}

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Hanko API request failed", map[string]interface{}{
			"error":  err.Error(),
			"method": method,
			"url":    url,
		})
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.logger.Error("Hanko API returned error", map[string]interface{}{
			"status_code": resp.StatusCode,
			"method":      method,
			"url":         url,
		})
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Decode response if expected
	if respBody != nil {
		if err := json.NewDecoder(resp.Body).Decode(respBody); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	c.logger.Debug("Hanko API request successful", map[string]interface{}{
		"method":      method,
		"url":         path,
		"status_code": resp.StatusCode,
	})

	return nil
}

// MockHankoClient provides a mock implementation for testing
type MockHankoClient struct {
	logger logging.Logger
	users  map[string]*HankoUser
	sessions map[string]*HankoSession
}

// NewMockHankoClient creates a new mock Hanko client
func NewMockHankoClient(logger logging.Logger) *MockHankoClient {
	return &MockHankoClient{
		logger:   logger,
		users:    make(map[string]*HankoUser),
		sessions: make(map[string]*HankoSession),
	}
}

// CreateUser creates a mock user
func (m *MockHankoClient) CreateUser(ctx context.Context, email string) (*HankoUser, error) {
	userID := fmt.Sprintf("mock_user_%d", time.Now().UnixNano())
	user := &HankoUser{
		ID:            userID,
		Email:         email,
		EmailVerified: true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		HasPasskey:    false,
	}

	m.users[userID] = user
	m.logger.Info("Mock user created", map[string]interface{}{
		"user_id": userID,
		"email":   email,
	})

	return user, nil
}

// GetUser retrieves a mock user
func (m *MockHankoClient) GetUser(ctx context.Context, userID string) (*HankoUser, error) {
	user, exists := m.users[userID]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

// GetUserByEmail retrieves a mock user by email
func (m *MockHankoClient) GetUserByEmail(ctx context.Context, email string) (*HankoUser, error) {
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

// ValidateSession validates a mock session
func (m *MockHankoClient) ValidateSession(ctx context.Context, sessionToken string) (*ValidateSessionResponse, error) {
	session, exists := m.sessions[sessionToken]
	if !exists || session.ExpiresAt.Before(time.Now()) {
		return &ValidateSessionResponse{Valid: false}, nil
	}

	user := m.users[session.UserID]
	return &ValidateSessionResponse{
		Valid:   true,
		Session: *session,
		User:    *user,
	}, nil
}

// Other mock methods...
func (m *MockHankoClient) InitiatePasskeyRegistration(ctx context.Context, userID string) (*PasskeyRegistrationResponse, error) {
	return &PasskeyRegistrationResponse{
		RegistrationOptions: map[string]interface{}{
			"challenge": "mock_challenge",
			"rp":        map[string]string{"name": "Reciprocal Clubs"},
		},
	}, nil
}

func (m *MockHankoClient) InitiatePasskeyAuthentication(ctx context.Context, userEmail string) (*PasskeyAuthenticationResponse, error) {
	return &PasskeyAuthenticationResponse{
		AuthenticationOptions: map[string]interface{}{
			"challenge": "mock_auth_challenge",
		},
	}, nil
}

func (m *MockHankoClient) VerifyPasskey(ctx context.Context, userID string, credentialResult map[string]interface{}) (*VerifyPasskeyResponse, error) {
	sessionID := fmt.Sprintf("mock_session_%d", time.Now().UnixNano())
	session := &HankoSession{
		ID:        sessionID,
		UserID:    userID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}

	m.sessions[sessionID] = session

	return &VerifyPasskeyResponse{
		Success: true,
		Session: *session,
	}, nil
}

func (m *MockHankoClient) InvalidateSession(ctx context.Context, sessionID string) error {
	delete(m.sessions, sessionID)
	return nil
}

func (m *MockHankoClient) DeleteUser(ctx context.Context, userID string) error {
	delete(m.users, userID)
	return nil
}

func (m *MockHankoClient) HealthCheck(ctx context.Context) error {
	return nil
}