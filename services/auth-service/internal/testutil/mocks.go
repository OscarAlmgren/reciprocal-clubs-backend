package testutil

import (
	"context"
	"fmt"
	"sync"
	"time"

	"reciprocal-clubs-backend/pkg/shared/config"
	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/messaging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	"reciprocal-clubs-backend/services/auth-service/internal/hanko"
)

// MockLogger implements the logging.Logger interface for testing
type MockLogger struct {
	logs []LogEntry
	mu   sync.RWMutex
}

type LogEntry struct {
	Level   string
	Message string
	Fields  map[string]interface{}
}

func NewMockLogger() *MockLogger {
	return &MockLogger{
		logs: make([]LogEntry, 0),
	}
}

func (m *MockLogger) Debug(msg string, fields map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, LogEntry{Level: "debug", Message: msg, Fields: fields})
}

func (m *MockLogger) Info(msg string, fields map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, LogEntry{Level: "info", Message: msg, Fields: fields})
}

func (m *MockLogger) Warn(msg string, fields map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, LogEntry{Level: "warn", Message: msg, Fields: fields})
}

func (m *MockLogger) Error(msg string, fields map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, LogEntry{Level: "error", Message: msg, Fields: fields})
}

func (m *MockLogger) Fatal(msg string, fields map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, LogEntry{Level: "fatal", Message: msg, Fields: fields})
	panic("Fatal log called")
}

func (m *MockLogger) GetLogs() []LogEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]LogEntry(nil), m.logs...)
}

func (m *MockLogger) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = m.logs[:0]
}

// MockMonitor implements the monitoring.Monitor interface for testing
type MockMonitor struct {
	metrics map[string]float64
	mu      sync.RWMutex
}

func NewMockMonitor() *MockMonitor {
	return &MockMonitor{
		metrics: make(map[string]float64),
	}
}

func (m *MockMonitor) IncrementCounter(name string, labels map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s_%v", name, labels)
	m.metrics[key]++
}

func (m *MockMonitor) RecordDuration(name string, duration time.Duration, labels map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s_%v", name, labels)
	m.metrics[key] = duration.Seconds()
}

func (m *MockMonitor) SetGauge(name string, value float64, labels map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s_%v", name, labels)
	m.metrics[key] = value
}

func (m *MockMonitor) GetMetric(name string, labels map[string]string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	key := fmt.Sprintf("%s_%v", name, labels)
	return m.metrics[key]
}

func (m *MockMonitor) GetAllMetrics() map[string]float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	metrics := make(map[string]float64)
	for k, v := range m.metrics {
		metrics[k] = v
	}
	return metrics
}

// MockMessageBus implements the messaging.MessageBus interface for testing
type MockMessageBus struct {
	messages []Message
	mu       sync.RWMutex
}

type Message struct {
	Subject string
	Data    []byte
}

func NewMockMessageBus() *MockMessageBus {
	return &MockMessageBus{
		messages: make([]Message, 0),
	}
}

func (m *MockMessageBus) Publish(ctx context.Context, subject string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, Message{Subject: subject, Data: data})
	return nil
}

func (m *MockMessageBus) Subscribe(ctx context.Context, subject string, handler messaging.MessageHandler) error {
	// For tests, we don't need to actually subscribe
	return nil
}

func (m *MockMessageBus) Close() error {
	return nil
}

func (m *MockMessageBus) GetMessages() []Message {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]Message(nil), m.messages...)
}

func (m *MockMessageBus) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = m.messages[:0]
}

// MockHankoClient implements the hanko.Client interface for testing
type MockHankoClient struct {
	users     map[string]*hanko.HankoUser
	sessions  map[string]*hanko.HankoSession
	challenges map[string]string
	mu        sync.RWMutex
	shouldFail map[string]bool
}

func NewMockHankoClient() *MockHankoClient {
	return &MockHankoClient{
		users:      make(map[string]*hanko.HankoUser),
		sessions:   make(map[string]*hanko.HankoSession),
		challenges: make(map[string]string),
		shouldFail: make(map[string]bool),
	}
}

func (m *MockHankoClient) CreateUser(ctx context.Context, email string) (*hanko.HankoUser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFail["CreateUser"] {
		return nil, fmt.Errorf("mock error: create user failed")
	}

	userID := fmt.Sprintf("hanko-user-%d", len(m.users)+1)
	user := &hanko.HankoUser{
		ID:            userID,
		Email:         email,
		EmailVerified: true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		HasPasskey:    false,
	}
	m.users[userID] = user
	return user, nil
}

func (m *MockHankoClient) GetUser(ctx context.Context, userID string) (*hanko.HankoUser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.shouldFail["GetUser"] {
		return nil, fmt.Errorf("mock error: get user failed")
	}

	user, exists := m.users[userID]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

func (m *MockHankoClient) DeleteUser(ctx context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFail["DeleteUser"] {
		return fmt.Errorf("mock error: delete user failed")
	}

	delete(m.users, userID)
	return nil
}

func (m *MockHankoClient) InitiatePasskeyAuthentication(ctx context.Context, email string) (*hanko.PasskeyAuthenticationResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFail["InitiatePasskeyLogin"] {
		return nil, fmt.Errorf("mock error: initiate login failed")
	}

	// Find user by email
	var user *hanko.HankoUser
	for _, u := range m.users {
		if u.Email == email {
			user = u
			break
		}
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	challengeID := fmt.Sprintf("challenge-%d", time.Now().UnixNano())
	challenge := &hanko.PasskeyAuthenticationResponse{
		AuthenticationOptions: map[string]interface{}{
			"challenge": "mock-challenge-data",
			"user_id":   user.ID,
			"expires_at": time.Now().Add(5 * time.Minute),
		},
	}
	
	m.challenges[challengeID] = user.ID
	return challenge, nil
}

func (m *MockHankoClient) VerifyPasskey(ctx context.Context, userID string, credentialResult map[string]interface{}) (*hanko.VerifyPasskeyResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFail["VerifyPasskey"] {
		return nil, fmt.Errorf("mock error: verify passkey failed")
	}

	user := m.users[userID]
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	sessionID := fmt.Sprintf("mock_session_%d", time.Now().UnixNano())
	session := &hanko.HankoSession{
		ID:        sessionID,
		UserID:    userID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}
	m.sessions[sessionID] = session

	result := &hanko.VerifyPasskeyResponse{
		Success: true,
		Session: *session,
	}
	return result, nil
}

func (m *MockHankoClient) ValidateSession(ctx context.Context, sessionToken string) (*hanko.ValidateSessionResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.shouldFail["ValidateSession"] {
		return nil, fmt.Errorf("mock error: validate session failed")
	}

	session, exists := m.sessions[sessionToken]
	if !exists {
		return &hanko.ValidateSessionResponse{Valid: false}, nil
	}

	if time.Now().After(session.ExpiresAt) {
		return &hanko.ValidateSessionResponse{Valid: false}, nil
	}

	user := m.users[session.UserID]
	return &hanko.ValidateSessionResponse{
		Valid:   true,
		Session: *session,
		User:    *user,
	}, nil
}

func (m *MockHankoClient) InitiatePasskeyRegistration(ctx context.Context, userID string) (*hanko.PasskeyRegistrationResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFail["InitiatePasskeyRegistration"] {
		return nil, fmt.Errorf("mock error: initiate registration failed")
	}

	_, exists := m.users[userID]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}

	challengeID := fmt.Sprintf("reg-challenge-%d", time.Now().UnixNano())
	challenge := &hanko.PasskeyRegistrationResponse{
		RegistrationOptions: map[string]interface{}{
			"challenge": "mock-registration-challenge-data",
			"user_id":   userID,
			"expires_at": time.Now().Add(5 * time.Minute),
		},
	}
	
	m.challenges[challengeID] = userID
	return challenge, nil
}

// Test helpers for MockHankoClient
func (m *MockHankoClient) SetShouldFail(method string, shouldFail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail[method] = shouldFail
}

func (m *MockHankoClient) AddUser(user *hanko.HankoUser) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[user.ID] = user
}

func (m *MockHankoClient) AddSession(token string, session *hanko.HankoSession) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[token] = session
}

func (m *MockHankoClient) GetUserCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.users)
}

func (m *MockHankoClient) GetUserByEmail(ctx context.Context, email string) (*hanko.HankoUser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

func (m *MockHankoClient) InvalidateSession(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, sessionID)
	return nil
}

func (m *MockHankoClient) HealthCheck(ctx context.Context) error {
	return nil
}

func (m *MockHankoClient) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users = make(map[string]*hanko.HankoUser)
	m.sessions = make(map[string]*hanko.HankoSession)
	m.challenges = make(map[string]string)
	m.shouldFail = make(map[string]bool)
}

// NewMockConfig creates a mock config for testing
func NewMockConfig() *config.Config {
	return &config.Config{
		Service: config.ServiceConfig{
			Name:        "auth-service",
			Version:     "1.0.0",
			Environment: "test",
			Port:        8080,
		},
		Auth: config.AuthConfig{
			JWTSecret:    "test-jwt-secret",
			JWTExpiration: 3600,
		},
		Database: config.DatabaseConfig{
			Host:            "localhost",
			Port:            5432,
			Database:        "test_db",
			User:            "test_user",
			Password:        "test_pass",
			SSLMode:         "disable",
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 300,
		},
	}
}