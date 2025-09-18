package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	"reciprocal-clubs-backend/services/notification-service/internal/models"
	"reciprocal-clubs-backend/services/notification-service/internal/service"
)

// Mock Service
type MockNotificationService struct {
	mock.Mock
}

func (m *MockNotificationService) CreateNotification(ctx context.Context, req *service.CreateNotificationRequest) (*models.Notification, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*models.Notification), args.Error(1)
}

func (m *MockNotificationService) GetNotificationByID(ctx context.Context, id uint) (*models.Notification, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.Notification), args.Error(1)
}

func (m *MockNotificationService) GetNotificationsByClub(ctx context.Context, clubID uint, limit, offset int) ([]models.Notification, error) {
	args := m.Called(ctx, clubID, limit, offset)
	return args.Get(0).([]models.Notification), args.Error(1)
}

func (m *MockNotificationService) GetNotificationsByUser(ctx context.Context, userID string, clubID uint, limit, offset int) ([]models.Notification, error) {
	args := m.Called(ctx, userID, clubID, limit, offset)
	return args.Get(0).([]models.Notification), args.Error(1)
}

func (m *MockNotificationService) MarkNotificationAsRead(ctx context.Context, id uint) (*models.Notification, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.Notification), args.Error(1)
}

func (m *MockNotificationService) ProcessNotification(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockNotificationService) ProcessScheduledNotifications(ctx context.Context) int {
	args := m.Called(ctx)
	return args.Int(0)
}

func (m *MockNotificationService) RetryFailedNotifications(ctx context.Context) int {
	args := m.Called(ctx)
	return args.Int(0)
}

func (m *MockNotificationService) CreateNotificationTemplate(ctx context.Context, req *service.CreateTemplateRequest) (*models.NotificationTemplate, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*models.NotificationTemplate), args.Error(1)
}

func (m *MockNotificationService) GetNotificationTemplateByID(ctx context.Context, id uint) (*models.NotificationTemplate, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.NotificationTemplate), args.Error(1)
}

func (m *MockNotificationService) GetNotificationTemplatesByClub(ctx context.Context, clubID uint) ([]models.NotificationTemplate, error) {
	args := m.Called(ctx, clubID)
	return args.Get(0).([]models.NotificationTemplate), args.Error(1)
}

func (m *MockNotificationService) UpdateNotificationTemplate(ctx context.Context, template *models.NotificationTemplate) error {
	args := m.Called(ctx, template)
	return args.Error(0)
}

func (m *MockNotificationService) CreateNotificationPreference(ctx context.Context, preference *models.NotificationPreference) error {
	args := m.Called(ctx, preference)
	return args.Error(0)
}

func (m *MockNotificationService) GetNotificationPreferences(ctx context.Context, userID string, clubID uint) ([]models.NotificationPreference, error) {
	args := m.Called(ctx, userID, clubID)
	return args.Get(0).([]models.NotificationPreference), args.Error(1)
}

func (m *MockNotificationService) UpdateNotificationPreference(ctx context.Context, preference *models.NotificationPreference) error {
	args := m.Called(ctx, preference)
	return args.Error(0)
}

func (m *MockNotificationService) GetNotificationStats(ctx context.Context, clubID uint, fromDate, toDate time.Time) (map[string]interface{}, error) {
	args := m.Called(ctx, clubID, fromDate, toDate)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func setupTestHandler(t *testing.T) (*HTTPHandler, *MockNotificationService) {
	mockService := &MockNotificationService{}
	logger := &logging.MockLogger{}
	monitor := &monitoring.MockMonitor{}

	handler := NewHTTPHandler(mockService, logger, monitor)
	return handler, mockService
}

func TestHTTPHandler_HealthCheck(t *testing.T) {
	handler, _ := setupTestHandler(t)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler.healthCheck(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "healthy", response["status"])
	assert.Equal(t, "notification-service", response["service"])
}

func TestHTTPHandler_CreateNotification(t *testing.T) {
	handler, mockService := setupTestHandler(t)

	reqBody := service.CreateNotificationRequest{
		ClubID:    1,
		UserID:    "user123",
		Type:      models.NotificationTypeEmail,
		Priority:  models.NotificationPriorityNormal,
		Title:     "Test Notification",
		Message:   "This is a test notification",
		Recipient: "test@example.com",
		Metadata:  map[string]string{"key": "value"},
	}

	expectedNotification := &models.Notification{
		ID:        123,
		ClubID:    1,
		UserID:    "user123",
		Type:      models.NotificationTypeEmail,
		Status:    models.NotificationStatusPending,
		Priority:  models.NotificationPriorityNormal,
		Title:     "Test Notification",
		Message:   "This is a test notification",
		Recipient: "test@example.com",
		Metadata:  map[string]string{"key": "value"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Setup mock expectations
	mockService.On("CreateNotification", mock.Anything, &reqBody).Return(expectedNotification, nil)

	// Create request
	reqJSON, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/notifications", bytes.NewBuffer(reqJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute
	handler.createNotification(w, req)

	// Verify
	assert.Equal(t, http.StatusCreated, w.Code)

	var response models.Notification
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, expectedNotification.ID, response.ID)
	assert.Equal(t, expectedNotification.Title, response.Title)
	assert.Equal(t, expectedNotification.Message, response.Message)

	mockService.AssertExpectations(t)
}

func TestHTTPHandler_CreateNotification_InvalidJSON(t *testing.T) {
	handler, _ := setupTestHandler(t)

	req := httptest.NewRequest("POST", "/api/v1/notifications", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.createNotification(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Invalid request body", response["error"])
}

func TestHTTPHandler_GetNotification(t *testing.T) {
	handler, mockService := setupTestHandler(t)

	expectedNotification := &models.Notification{
		ID:        123,
		ClubID:    1,
		UserID:    "user123",
		Type:      models.NotificationTypeEmail,
		Status:    models.NotificationStatusSent,
		Priority:  models.NotificationPriorityNormal,
		Title:     "Test Notification",
		Message:   "This is a test notification",
		Recipient: "test@example.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Setup mock expectations
	mockService.On("GetNotificationByID", mock.Anything, uint(123)).Return(expectedNotification, nil)

	// Create request with mux vars
	req := httptest.NewRequest("GET", "/api/v1/notifications/123", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "123"})
	w := httptest.NewRecorder()

	// Execute
	handler.getNotification(w, req)

	// Verify
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.Notification
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, expectedNotification.ID, response.ID)
	assert.Equal(t, expectedNotification.Title, response.Title)

	mockService.AssertExpectations(t)
}

func TestHTTPHandler_GetNotification_InvalidID(t *testing.T) {
	handler, _ := setupTestHandler(t)

	req := httptest.NewRequest("GET", "/api/v1/notifications/invalid", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "invalid"})
	w := httptest.NewRecorder()

	handler.getNotification(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Invalid ID", response["error"])
}

func TestHTTPHandler_MarkAsRead(t *testing.T) {
	handler, mockService := setupTestHandler(t)

	expectedNotification := &models.Notification{
		ID:        123,
		ClubID:    1,
		UserID:    "user123",
		Type:      models.NotificationTypeEmail,
		Status:    models.NotificationStatusRead,
		Priority:  models.NotificationPriorityNormal,
		Title:     "Test Notification",
		Message:   "This is a test notification",
		Recipient: "test@example.com",
		ReadAt:    &time.Time{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Setup mock expectations
	mockService.On("MarkNotificationAsRead", mock.Anything, uint(123)).Return(expectedNotification, nil)

	// Create request
	req := httptest.NewRequest("POST", "/api/v1/notifications/123/read", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "123"})
	w := httptest.NewRecorder()

	// Execute
	handler.markAsRead(w, req)

	// Verify
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.Notification
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, expectedNotification.ID, response.ID)
	assert.Equal(t, models.NotificationStatusRead, response.Status)

	mockService.AssertExpectations(t)
}

func TestHTTPHandler_GetClubNotifications(t *testing.T) {
	handler, mockService := setupTestHandler(t)

	expectedNotifications := []models.Notification{
		{
			ID:        1,
			ClubID:    1,
			UserID:    "user1",
			Type:      models.NotificationTypeEmail,
			Status:    models.NotificationStatusSent,
			Title:     "Notification 1",
			Message:   "Message 1",
			Recipient: "user1@example.com",
		},
		{
			ID:        2,
			ClubID:    1,
			UserID:    "user2",
			Type:      models.NotificationTypeSMS,
			Status:    models.NotificationStatusPending,
			Title:     "Notification 2",
			Message:   "Message 2",
			Recipient: "+1234567890",
		},
	}

	// Setup mock expectations
	mockService.On("GetNotificationsByClub", mock.Anything, uint(1), 50, 0).Return(expectedNotifications, nil)

	// Create request
	req := httptest.NewRequest("GET", "/api/v1/clubs/1/notifications", nil)
	req = mux.SetURLVars(req, map[string]string{"clubId": "1"})
	w := httptest.NewRecorder()

	// Execute
	handler.getClubNotifications(w, req)

	// Verify
	assert.Equal(t, http.StatusOK, w.Code)

	var response []models.Notification
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Len(t, response, 2)
	assert.Equal(t, expectedNotifications[0].ID, response[0].ID)
	assert.Equal(t, expectedNotifications[1].ID, response[1].ID)

	mockService.AssertExpectations(t)
}

func TestHTTPHandler_GetClubNotifications_WithPagination(t *testing.T) {
	handler, mockService := setupTestHandler(t)

	expectedNotifications := []models.Notification{
		{
			ID:        1,
			ClubID:    1,
			UserID:    "user1",
			Type:      models.NotificationTypeEmail,
			Status:    models.NotificationStatusSent,
			Title:     "Notification 1",
			Message:   "Message 1",
			Recipient: "user1@example.com",
		},
	}

	// Setup mock expectations with custom limit and offset
	mockService.On("GetNotificationsByClub", mock.Anything, uint(1), 10, 20).Return(expectedNotifications, nil)

	// Create request with query parameters
	req := httptest.NewRequest("GET", "/api/v1/clubs/1/notifications?limit=10&offset=20", nil)
	req = mux.SetURLVars(req, map[string]string{"clubId": "1"})
	w := httptest.NewRecorder()

	// Execute
	handler.getClubNotifications(w, req)

	// Verify
	assert.Equal(t, http.StatusOK, w.Code)

	var response []models.Notification
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Len(t, response, 1)

	mockService.AssertExpectations(t)
}

func TestHTTPHandler_GetUserNotifications(t *testing.T) {
	handler, mockService := setupTestHandler(t)

	expectedNotifications := []models.Notification{
		{
			ID:        1,
			ClubID:    1,
			UserID:    "user123",
			Type:      models.NotificationTypeEmail,
			Status:    models.NotificationStatusSent,
			Title:     "User Notification 1",
			Message:   "Message 1",
			Recipient: "user123@example.com",
		},
		{
			ID:        2,
			ClubID:    1,
			UserID:    "user123",
			Type:      models.NotificationTypePush,
			Status:    models.NotificationStatusDelivered,
			Title:     "User Notification 2",
			Message:   "Message 2",
			Recipient: "user123@example.com",
		},
	}

	// Setup mock expectations
	mockService.On("GetNotificationsByUser", mock.Anything, "user123", uint(1), 50, 0).Return(expectedNotifications, nil)

	// Create request
	req := httptest.NewRequest("GET", "/api/v1/users/user123/notifications?club_id=1", nil)
	req = mux.SetURLVars(req, map[string]string{"userId": "user123"})
	w := httptest.NewRecorder()

	// Execute
	handler.getUserNotifications(w, req)

	// Verify
	assert.Equal(t, http.StatusOK, w.Code)

	var response []models.Notification
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Len(t, response, 2)
	assert.Equal(t, "user123", response[0].UserID)
	assert.Equal(t, "user123", response[1].UserID)

	mockService.AssertExpectations(t)
}

func TestHTTPHandler_GetUserNotifications_MissingClubID(t *testing.T) {
	handler, _ := setupTestHandler(t)

	req := httptest.NewRequest("GET", "/api/v1/users/user123/notifications", nil)
	req = mux.SetURLVars(req, map[string]string{"userId": "user123"})
	w := httptest.NewRecorder()

	handler.getUserNotifications(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "club_id query parameter is required", response["error"])
}

func TestHTTPHandler_CreateTemplate(t *testing.T) {
	handler, mockService := setupTestHandler(t)

	reqBody := service.CreateTemplateRequest{
		ClubID:          1,
		Name:            "Welcome Email",
		Description:     "Welcome email template for new members",
		Type:            models.NotificationTypeEmail,
		SubjectTemplate: "Welcome to {{.ClubName}}!",
		BodyTemplate:    "Dear {{.MemberName}}, welcome to our club!",
		DefaultMetadata: map[string]string{"category": "welcome"},
	}

	expectedTemplate := &models.NotificationTemplate{
		ID:              456,
		ClubID:          1,
		Name:            "Welcome Email",
		Description:     "Welcome email template for new members",
		Type:            models.NotificationTypeEmail,
		SubjectTemplate: "Welcome to {{.ClubName}}!",
		BodyTemplate:    "Dear {{.MemberName}}, welcome to our club!",
		DefaultMetadata: map[string]string{"category": "welcome"},
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Setup mock expectations
	mockService.On("CreateNotificationTemplate", mock.Anything, &reqBody).Return(expectedTemplate, nil)

	// Create request
	reqJSON, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/templates", bytes.NewBuffer(reqJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute
	handler.createTemplate(w, req)

	// Verify
	assert.Equal(t, http.StatusCreated, w.Code)

	var response models.NotificationTemplate
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, expectedTemplate.ID, response.ID)
	assert.Equal(t, expectedTemplate.Name, response.Name)
	assert.Equal(t, expectedTemplate.Description, response.Description)

	mockService.AssertExpectations(t)
}

func TestHTTPHandler_GetClubTemplates(t *testing.T) {
	handler, mockService := setupTestHandler(t)

	expectedTemplates := []models.NotificationTemplate{
		{
			ID:              1,
			ClubID:          1,
			Name:            "Welcome Email",
			Description:     "Welcome email template",
			Type:            models.NotificationTypeEmail,
			SubjectTemplate: "Welcome!",
			BodyTemplate:    "Welcome to our club!",
			IsActive:        true,
		},
		{
			ID:              2,
			ClubID:          1,
			Name:            "SMS Alert",
			Description:     "SMS alert template",
			Type:            models.NotificationTypeSMS,
			SubjectTemplate: "",
			BodyTemplate:    "Alert: {{.Message}}",
			IsActive:        true,
		},
	}

	// Setup mock expectations
	mockService.On("GetNotificationTemplatesByClub", mock.Anything, uint(1)).Return(expectedTemplates, nil)

	// Create request
	req := httptest.NewRequest("GET", "/api/v1/clubs/1/templates", nil)
	req = mux.SetURLVars(req, map[string]string{"clubId": "1"})
	w := httptest.NewRecorder()

	// Execute
	handler.getClubTemplates(w, req)

	// Verify
	assert.Equal(t, http.StatusOK, w.Code)

	var response []models.NotificationTemplate
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Len(t, response, 2)
	assert.Equal(t, expectedTemplates[0].ID, response[0].ID)
	assert.Equal(t, expectedTemplates[1].ID, response[1].ID)

	mockService.AssertExpectations(t)
}

func TestHTTPHandler_GetNotificationStats(t *testing.T) {
	handler, mockService := setupTestHandler(t)

	expectedStats := map[string]interface{}{
		"total":     int64(100),
		"pending":   int64(5),
		"sent":      int64(80),
		"delivered": int64(75),
		"failed":    int64(5),
		"read":      int64(60),
	}

	// Setup mock expectations
	mockService.On("GetNotificationStats", mock.Anything, uint(1), mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(expectedStats, nil)

	// Create request
	req := httptest.NewRequest("GET", "/api/v1/clubs/1/stats?from=2023-01-01&to=2023-12-31", nil)
	req = mux.SetURLVars(req, map[string]string{"clubId": "1"})
	w := httptest.NewRecorder()

	// Execute
	handler.getNotificationStats(w, req)

	// Verify
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(100), response["total"])
	assert.Equal(t, float64(5), response["pending"])
	assert.Equal(t, float64(80), response["sent"])
	assert.Equal(t, float64(75), response["delivered"])
	assert.Equal(t, float64(5), response["failed"])
	assert.Equal(t, float64(60), response["read"])

	mockService.AssertExpectations(t)
}

func TestHTTPHandler_SetupRoutes(t *testing.T) {
	handler, _ := setupTestHandler(t)

	router := handler.SetupRoutes()

	// Test that routes are properly configured
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHTTPHandler_Middleware(t *testing.T) {
	handler, _ := setupTestHandler(t)

	// Test logging middleware
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := handler.loggingMiddleware(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test monitoring middleware
	wrappedHandler = handler.monitoringMiddleware(testHandler)

	req = httptest.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}