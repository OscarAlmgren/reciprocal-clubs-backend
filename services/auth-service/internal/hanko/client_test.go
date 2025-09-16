package hanko

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient("http://localhost:8000", "test-api-key", "test-project")
	
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	
	// Test that we can cast to the concrete type to check internals if needed
	concreteClient, ok := client.(*hankoClient)
	if !ok {
		t.Fatal("Client is not of expected type")
	}
	
	if concreteClient.baseURL != "http://localhost:8000" {
		t.Errorf("Expected baseURL %s, got %s", "http://localhost:8000", concreteClient.baseURL)
	}
}

func TestCreateUser(t *testing.T) {
	tests := []struct {
		name           string
		email          string
		displayName    string
		responseBody   string
		responseStatus int
		expectError    bool
		expectedUserID string
	}{
		{
			name:           "successful user creation",
			email:          "test@example.com",
			displayName:    "Test User",
			responseBody:   `{"id":"user123","email":"test@example.com","display_name":"Test User","created_at":"2023-01-01T00:00:00Z"}`,
			responseStatus: http.StatusCreated,
			expectError:    false,
			expectedUserID: "user123",
		},
		{
			name:           "user already exists",
			email:          "existing@example.com",
			displayName:    "Existing User",
			responseBody:   `{"error":"user already exists"}`,
			responseStatus: http.StatusConflict,
			expectError:    true,
		},
		{
			name:           "invalid email",
			email:          "invalid-email",
			displayName:    "Test User",
			responseBody:   `{"error":"invalid email format"}`,
			responseStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "server error",
			email:          "test@example.com",
			displayName:    "Test User",
			responseBody:   `{"error":"internal server error"}`,
			responseStatus: http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				if r.URL.Path != "/users" {
					t.Errorf("Expected path /users, got %s", r.URL.Path)
				}
				
				// Check headers
				if r.Header.Get("Authorization") != "Bearer test-api-key" {
					t.Errorf("Expected Authorization header with API key")
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type application/json")
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.responseStatus)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			// Create client with test server URL
			client := NewClient(server.URL, "test-api-key", "test-project")
			
			// Test CreateUser
			ctx := context.Background()
			user, err := client.CreateUser(ctx, tt.email, tt.displayName)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if user == nil {
				t.Error("Expected user but got nil")
				return
			}

			if user.ID != tt.expectedUserID {
				t.Errorf("Expected user ID %s, got %s", tt.expectedUserID, user.ID)
			}

			if user.Email != tt.email {
				t.Errorf("Expected email %s, got %s", tt.email, user.Email)
			}

			if user.DisplayName != tt.displayName {
				t.Errorf("Expected display name %s, got %s", tt.displayName, user.DisplayName)
			}
		})
	}
}

func TestGetUser(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		responseBody   string
		responseStatus int
		expectError    bool
		expectedEmail  string
	}{
		{
			name:           "successful user retrieval",
			userID:         "user123",
			responseBody:   `{"id":"user123","email":"test@example.com","display_name":"Test User","created_at":"2023-01-01T00:00:00Z"}`,
			responseStatus: http.StatusOK,
			expectError:    false,
			expectedEmail:  "test@example.com",
		},
		{
			name:           "user not found",
			userID:         "nonexistent",
			responseBody:   `{"error":"user not found"}`,
			responseStatus: http.StatusNotFound,
			expectError:    true,
		},
		{
			name:           "invalid user ID",
			userID:         "",
			responseBody:   `{"error":"invalid user ID"}`,
			responseStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET request, got %s", r.Method)
				}
				
				expectedPath := "/users/" + tt.userID
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.responseStatus)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := NewClient(server.URL, "test-api-key", "test-project")
			
			ctx := context.Background()
			user, err := client.GetUser(ctx, tt.userID)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if user == nil {
				t.Error("Expected user but got nil")
				return
			}

			if user.Email != tt.expectedEmail {
				t.Errorf("Expected email %s, got %s", tt.expectedEmail, user.Email)
			}
		})
	}
}

func TestDeleteUser(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		responseStatus int
		expectError    bool
	}{
		{
			name:           "successful user deletion",
			userID:         "user123",
			responseStatus: http.StatusNoContent,
			expectError:    false,
		},
		{
			name:           "user not found",
			userID:         "nonexistent",
			responseStatus: http.StatusNotFound,
			expectError:    true,
		},
		{
			name:           "server error",
			userID:         "user123",
			responseStatus: http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete {
					t.Errorf("Expected DELETE request, got %s", r.Method)
				}

				w.WriteHeader(tt.responseStatus)
			}))
			defer server.Close()

			client := NewClient(server.URL, "test-api-key", "test-project")
			
			ctx := context.Background()
			err := client.DeleteUser(ctx, tt.userID)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestInitiatePasskeyLogin(t *testing.T) {
	tests := []struct {
		name             string
		email            string
		responseBody     string
		responseStatus   int
		expectError      bool
		expectedUserID   string
		expectedChallenge string
	}{
		{
			name:             "successful login initiation",
			email:            "test@example.com",
			responseBody:     `{"id":"challenge123","user_id":"user123","challenge":"mock-challenge-data","expires_at":"2023-01-01T01:00:00Z"}`,
			responseStatus:   http.StatusOK,
			expectError:      false,
			expectedUserID:   "user123",
			expectedChallenge: "mock-challenge-data",
		},
		{
			name:           "user not found",
			email:          "nonexistent@example.com",
			responseBody:   `{"error":"user not found"}`,
			responseStatus: http.StatusNotFound,
			expectError:    true,
		},
		{
			name:           "invalid email",
			email:          "invalid-email",
			responseBody:   `{"error":"invalid email format"}`,
			responseStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				if r.URL.Path != "/passkey/login/initialize" {
					t.Errorf("Expected path /passkey/login/initialize, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.responseStatus)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := NewClient(server.URL, "test-api-key", "test-project")
			
			ctx := context.Background()
			challenge, err := client.InitiatePasskeyLogin(ctx, tt.email)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if challenge == nil {
				t.Error("Expected challenge but got nil")
				return
			}

			if challenge.UserID != tt.expectedUserID {
				t.Errorf("Expected user ID %s, got %s", tt.expectedUserID, challenge.UserID)
			}

			if challenge.Challenge != tt.expectedChallenge {
				t.Errorf("Expected challenge %s, got %s", tt.expectedChallenge, challenge.Challenge)
			}
		})
	}
}

func TestVerifyPasskey(t *testing.T) {
	tests := []struct {
		name           string
		challengeID    string
		credentialData map[string]interface{}
		responseBody   string
		responseStatus int
		expectError    bool
		expectedUserID string
		expectedVerified bool
	}{
		{
			name:        "successful verification",
			challengeID: "challenge123",
			credentialData: map[string]interface{}{
				"id": "credential123",
				"response": map[string]interface{}{
					"authenticatorData": "mock-auth-data",
					"signature":         "mock-signature",
				},
			},
			responseBody:     `{"user_id":"user123","verified":true,"user":{"id":"user123","email":"test@example.com"}}`,
			responseStatus:   http.StatusOK,
			expectError:      false,
			expectedUserID:   "user123",
			expectedVerified: true,
		},
		{
			name:        "verification failed",
			challengeID: "challenge123",
			credentialData: map[string]interface{}{
				"id": "invalid-credential",
			},
			responseBody:     `{"user_id":"user123","verified":false}`,
			responseStatus:   http.StatusOK,
			expectError:      false,
			expectedUserID:   "user123",
			expectedVerified: false,
		},
		{
			name:           "challenge not found",
			challengeID:    "nonexistent",
			credentialData: map[string]interface{}{},
			responseBody:   `{"error":"challenge not found"}`,
			responseStatus: http.StatusNotFound,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				
				expectedPath := "/passkey/login/finalize/" + tt.challengeID
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.responseStatus)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := NewClient(server.URL, "test-api-key", "test-project")
			
			ctx := context.Background()
			result, err := client.VerifyPasskey(ctx, tt.challengeID, tt.credentialData)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Error("Expected result but got nil")
				return
			}

			if result.UserID != tt.expectedUserID {
				t.Errorf("Expected user ID %s, got %s", tt.expectedUserID, result.UserID)
			}

			if result.Verified != tt.expectedVerified {
				t.Errorf("Expected verified %t, got %t", tt.expectedVerified, result.Verified)
			}
		})
	}
}

func TestInitiatePasskeyRegistration(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		responseBody   string
		responseStatus int
		expectError    bool
		expectedUserID string
	}{
		{
			name:           "successful registration initiation",
			userID:         "user123",
			responseBody:   `{"id":"reg-challenge123","user_id":"user123","challenge":"mock-registration-challenge","expires_at":"2023-01-01T01:00:00Z","user":{"id":"user123","email":"test@example.com"}}`,
			responseStatus: http.StatusOK,
			expectError:    false,
			expectedUserID: "user123",
		},
		{
			name:           "user not found",
			userID:         "nonexistent",
			responseBody:   `{"error":"user not found"}`,
			responseStatus: http.StatusNotFound,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				
				expectedPath := "/passkey/registration/initialize/" + tt.userID
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.responseStatus)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := NewClient(server.URL, "test-api-key", "test-project")
			
			ctx := context.Background()
			challenge, err := client.InitiatePasskeyRegistration(ctx, tt.userID)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if challenge == nil {
				t.Error("Expected challenge but got nil")
				return
			}

			if challenge.UserID != tt.expectedUserID {
				t.Errorf("Expected user ID %s, got %s", tt.expectedUserID, challenge.UserID)
			}
		})
	}
}

func TestValidateSession(t *testing.T) {
	tests := []struct {
		name           string
		sessionToken   string
		responseBody   string
		responseStatus int
		expectError    bool
		expectedEmail  string
	}{
		{
			name:           "valid session",
			sessionToken:   "valid-token-123",
			responseBody:   `{"id":"user123","email":"test@example.com","display_name":"Test User"}`,
			responseStatus: http.StatusOK,
			expectError:    false,
			expectedEmail:  "test@example.com",
		},
		{
			name:           "invalid session",
			sessionToken:   "invalid-token",
			responseBody:   `{"error":"session not found or expired"}`,
			responseStatus: http.StatusUnauthorized,
			expectError:    true,
		},
		{
			name:           "expired session",
			sessionToken:   "expired-token",
			responseBody:   `{"error":"session expired"}`,
			responseStatus: http.StatusUnauthorized,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET request, got %s", r.Method)
				}
				if r.URL.Path != "/me" {
					t.Errorf("Expected path /me, got %s", r.URL.Path)
				}
				
				// Check Authorization header
				expectedAuth := "Bearer " + tt.sessionToken
				if r.Header.Get("Authorization") != expectedAuth {
					t.Errorf("Expected Authorization header %s, got %s", expectedAuth, r.Header.Get("Authorization"))
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.responseStatus)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := NewClient(server.URL, "test-api-key", "test-project")
			
			ctx := context.Background()
			user, err := client.ValidateSession(ctx, tt.sessionToken)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if user == nil {
				t.Error("Expected user but got nil")
				return
			}

			if user.Email != tt.expectedEmail {
				t.Errorf("Expected email %s, got %s", tt.expectedEmail, user.Email)
			}
		})
	}
}

func TestClientTimeout(t *testing.T) {
	// Create a server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"user123"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key", "test-project")
	
	// Set a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	
	_, err := client.GetUser(ctx, "user123")
	if err == nil {
		t.Error("Expected timeout error but got none")
	}
}

func TestNewMockClient(t *testing.T) {
	mockClient := NewMockClient()
	
	if mockClient == nil {
		t.Fatal("NewMockClient returned nil")
	}
	
	ctx := context.Background()
	
	// Test creating a user
	user, err := mockClient.CreateUser(ctx, "test@example.com", "Test User")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if user == nil {
		t.Error("Expected user but got nil")
	}
	
	// Test getting the created user
	retrievedUser, err := mockClient.GetUser(ctx, user.ID)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if retrievedUser == nil {
		t.Error("Expected user but got nil")
	}
	if retrievedUser.Email != user.Email {
		t.Errorf("Expected email %s, got %s", user.Email, retrievedUser.Email)
	}
	
	// Test login initiation
	challenge, err := mockClient.InitiatePasskeyLogin(ctx, user.Email)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if challenge == nil {
		t.Error("Expected challenge but got nil")
	}
	
	// Test passkey verification
	credentialData := map[string]interface{}{
		"id": "test-credential",
	}
	result, err := mockClient.VerifyPasskey(ctx, challenge.ID, credentialData)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result == nil {
		t.Error("Expected result but got nil")
	}
	if !result.Verified {
		t.Error("Expected verification to succeed")
	}
}