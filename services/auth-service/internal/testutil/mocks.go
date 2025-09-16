package testutil

import (
	"context"
	"fmt"
	"sync"
	"time"

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
	users     map[string]*hanko.User
	sessions  map[string]*hanko.Session
	challenges map[string]string
	mu        sync.RWMutex
	shouldFail map[string]bool
}

func NewMockHankoClient() *MockHankoClient {
	return &MockHankoClient{
		users:      make(map[string]*hanko.User),
		sessions:   make(map[string]*hanko.Session),
		challenges: make(map[string]string),
		shouldFail: make(map[string]bool),
	}
}

func (m *MockHankoClient) CreateUser(ctx context.Context, email, displayName string) (*hanko.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFail["CreateUser"] {
		return nil, fmt.Errorf("mock error: create user failed")
	}

	userID := fmt.Sprintf("hanko-user-%d", len(m.users)+1)
	user := &hanko.User{
		ID:          userID,
		Email:       email,
		DisplayName: displayName,
		CreatedAt:   time.Now(),
	}
	m.users[userID] = user
	return user, nil
}

func (m *MockHankoClient) GetUser(ctx context.Context, userID string) (*hanko.User, error) {
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

func (m *MockHankoClient) InitiatePasskeyLogin(ctx context.Context, email string) (*hanko.LoginChallenge, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFail["InitiatePasskeyLogin"] {
		return nil, fmt.Errorf("mock error: initiate login failed")
	}

	// Find user by email
	var user *hanko.User
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
	challenge := &hanko.LoginChallenge{
		ID:          challengeID,
		UserID:      user.ID,
		Challenge:   "mock-challenge-data",
		ExpiresAt:   time.Now().Add(5 * time.Minute),
	}
	
	m.challenges[challengeID] = user.ID
	return challenge, nil
}

func (m *MockHankoClient) VerifyPasskey(ctx context.Context, challengeID string, credentialData map[string]interface{}) (*hanko.VerificationResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFail["VerifyPasskey"] {
		return nil, fmt.Errorf("mock error: verify passkey failed")
	}

	userID, exists := m.challenges[challengeID]
	if !exists {
		return nil, fmt.Errorf("challenge not found")
	}

	user := m.users[userID]
	result := &hanko.VerificationResult{
		UserID:    userID,
		Verified:  true,
		User:      user,
	}
	
	// Clean up challenge
	delete(m.challenges, challengeID)
	return result, nil
}

func (m *MockHankoClient) ValidateSession(ctx context.Context, sessionToken string) (*hanko.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.shouldFail["ValidateSession"] {
		return nil, fmt.Errorf("mock error: validate session failed")
	}

	session, exists := m.sessions[sessionToken]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	user := m.users[session.UserID]
	return user, nil
}

func (m *MockHankoClient) InitiatePasskeyRegistration(ctx context.Context, userID string) (*hanko.RegistrationChallenge, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFail["InitiatePasskeyRegistration"] {
		return nil, fmt.Errorf("mock error: initiate registration failed")
	}

	user, exists := m.users[userID]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}

	challengeID := fmt.Sprintf("reg-challenge-%d", time.Now().UnixNano())
	challenge := &hanko.RegistrationChallenge{
		ID:        challengeID,
		UserID:    userID,
		Challenge: "mock-registration-challenge-data",
		ExpiresAt: time.Now().Add(5 * time.Minute),
		User:      user,
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

func (m *MockHankoClient) AddUser(user *hanko.User) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[user.ID] = user
}

func (m *MockHankoClient) AddSession(token string, session *hanko.Session) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[token] = session
}

func (m *MockHankoClient) GetUserCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.users)
}

func (m *MockHankoClient) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users = make(map[string]*hanko.User)
	m.sessions = make(map[string]*hanko.Session)
	m.challenges = make(map[string]string)
	m.shouldFail = make(map[string]bool)
}