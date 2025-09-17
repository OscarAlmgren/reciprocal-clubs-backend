package hanko

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewHankoClient(t *testing.T) {
	config := Config{
		BaseURL: "http://localhost:8000",
		APIKey:  "test-api-key",
		Timeout: 30 * time.Second,
	}
	client := NewHankoClient(config, nil)

	if client == nil {
		t.Fatal("NewHankoClient returned nil")
	}

	if client.baseURL != "http://localhost:8000" {
		t.Errorf("Expected baseURL %s, got %s", "http://localhost:8000", client.baseURL)
	}
}

func TestCreateUser(t *testing.T) {
	tests := []struct {
		name           string
		email          string
		responseBody   string
		responseStatus int
		expectError    bool
		expectedUserID string
	}{
		{
			name:           "successful user creation",
			email:          "test@example.com",
			responseBody:   `{"user":{"id":"user123","email":"test@example.com","created_at":"2023-01-01T00:00:00Z"},"status":"created"}`,
			responseStatus: http.StatusCreated,
			expectError:    false,
			expectedUserID: "user123",
		},
		{
			name:           "user already exists",
			email:          "existing@example.com",
			responseBody:   `{"error":"user already exists"}`,
			responseStatus: http.StatusConflict,
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
				if r.URL.Path != "/users" {
					t.Errorf("Expected path /users, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.responseStatus)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			config := Config{
				BaseURL: server.URL,
				APIKey:  "test-api-key",
				Timeout: 30 * time.Second,
			}
			client := NewHankoClient(config, nil)

			ctx := context.Background()
			user, err := client.CreateUser(ctx, tt.email)

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
			responseBody:   `{"id":"user123","email":"test@example.com","created_at":"2023-01-01T00:00:00Z"}`,
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

			config := Config{
				BaseURL: server.URL,
				APIKey:  "test-api-key",
				Timeout: 30 * time.Second,
			}
			client := NewHankoClient(config, nil)

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

			config := Config{
				BaseURL: server.URL,
				APIKey:  "test-api-key",
				Timeout: 30 * time.Second,
			}
			client := NewHankoClient(config, nil)

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

func TestInitiatePasskeyAuthentication(t *testing.T) {
	tests := []struct {
		name           string
		email          string
		responseBody   string
		responseStatus int
		expectError    bool
	}{
		{
			name:           "successful authentication initiation",
			email:          "test@example.com",
			responseBody:   `{"authentication_options":{"challenge":"mock-challenge-data"}}`,
			responseStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "user not found",
			email:          "nonexistent@example.com",
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
				if r.URL.Path != "/webauthn/authentication/initialize" {
					t.Errorf("Expected path /webauthn/authentication/initialize, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.responseStatus)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			config := Config{
				BaseURL: server.URL,
				APIKey:  "test-api-key",
				Timeout: 30 * time.Second,
			}
			client := NewHankoClient(config, nil)

			ctx := context.Background()
			resp, err := client.InitiatePasskeyAuthentication(ctx, tt.email)

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

			if resp == nil {
				t.Error("Expected response but got nil")
				return
			}

			if resp.AuthenticationOptions == nil {
				t.Error("Expected authentication options but got nil")
			}
		})
	}
}

func TestVerifyPasskey(t *testing.T) {
	tests := []struct {
		name               string
		userID             string
		credentialData     map[string]interface{}
		responseBody       string
		responseStatus     int
		expectError        bool
		expectedSuccess    bool
	}{
		{
			name:   "successful verification",
			userID: "user123",
			credentialData: map[string]interface{}{
				"id": "credential123",
			},
			responseBody:    `{"success":true,"session":{"id":"session123","user_id":"user123"}}`,
			responseStatus:  http.StatusOK,
			expectError:     false,
			expectedSuccess: true,
		},
		{
			name:   "verification failed",
			userID: "user123",
			credentialData: map[string]interface{}{
				"id": "invalid-credential",
			},
			responseBody:    `{"success":false}`,
			responseStatus:  http.StatusOK,
			expectError:     false,
			expectedSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST request, got %s", r.Method)
				}

				if r.URL.Path != "/webauthn/authentication/finalize" {
					t.Errorf("Expected path /webauthn/authentication/finalize, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.responseStatus)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			config := Config{
				BaseURL: server.URL,
				APIKey:  "test-api-key",
				Timeout: 30 * time.Second,
			}
			client := NewHankoClient(config, nil)

			ctx := context.Background()
			result, err := client.VerifyPasskey(ctx, tt.userID, tt.credentialData)

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

			if result.Success != tt.expectedSuccess {
				t.Errorf("Expected success %t, got %t", tt.expectedSuccess, result.Success)
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
	}{
		{
			name:           "successful registration initiation",
			userID:         "user123",
			responseBody:   `{"registration_options":{"challenge":"mock-registration-challenge"}}`,
			responseStatus: http.StatusOK,
			expectError:    false,
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

				if r.URL.Path != "/webauthn/registration/initialize" {
					t.Errorf("Expected path /webauthn/registration/initialize, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.responseStatus)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			config := Config{
				BaseURL: server.URL,
				APIKey:  "test-api-key",
				Timeout: 30 * time.Second,
			}
			client := NewHankoClient(config, nil)

			ctx := context.Background()
			resp, err := client.InitiatePasskeyRegistration(ctx, tt.userID)

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

			if resp == nil {
				t.Error("Expected response but got nil")
				return
			}

			if resp.RegistrationOptions == nil {
				t.Error("Expected registration options but got nil")
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
			responseBody:   `{"valid":true,"user":{"id":"user123","email":"test@example.com"},"session":{"id":"session123"}}`,
			responseStatus: http.StatusOK,
			expectError:    false,
			expectedEmail:  "test@example.com",
		},
		{
			name:           "invalid session",
			sessionToken:   "invalid-token",
			responseBody:   `{"valid":false}`,
			responseStatus: http.StatusOK,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				if r.URL.Path != "/sessions/validate" {
					t.Errorf("Expected path /sessions/validate, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.responseStatus)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			config := Config{
				BaseURL: server.URL,
				APIKey:  "test-api-key",
				Timeout: 30 * time.Second,
			}
			client := NewHankoClient(config, nil)

			ctx := context.Background()
			resp, err := client.ValidateSession(ctx, tt.sessionToken)

			if tt.expectError {
				if err == nil && resp.Valid {
					t.Error("Expected error or invalid session but got valid session")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if resp == nil {
				t.Error("Expected response but got nil")
				return
			}

			if !resp.Valid {
				t.Error("Expected valid session")
				return
			}

			if resp.User.Email != tt.expectedEmail {
				t.Errorf("Expected email %s, got %s", tt.expectedEmail, resp.User.Email)
			}
		})
	}
}