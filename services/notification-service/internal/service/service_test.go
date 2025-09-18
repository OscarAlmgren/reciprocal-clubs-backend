package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/messaging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	"reciprocal-clubs-backend/services/notification-service/internal/models"
)

// Mock Repository
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateNotification(ctx context.Context, notification *models.Notification) error {
	args := m.Called(ctx, notification)
	return args.Error(0)
}

func (m *MockRepository) GetNotificationByID(ctx context.Context, id uint) (*models.Notification, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.Notification), args.Error(1)
}

func (m *MockRepository) GetNotificationsByClub(ctx context.Context, clubID uint, limit, offset int) ([]models.Notification, error) {
	args := m.Called(ctx, clubID, limit, offset)
	return args.Get(0).([]models.Notification), args.Error(1)
}

func (m *MockRepository) GetNotificationsByUser(ctx context.Context, userID string, clubID uint, limit, offset int) ([]models.Notification, error) {
	args := m.Called(ctx, userID, clubID, limit, offset)
	return args.Get(0).([]models.Notification), args.Error(1)
}

func (m *MockRepository) GetPendingNotifications(ctx context.Context, limit int) ([]models.Notification, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]models.Notification), args.Error(1)
}

func (m *MockRepository) GetFailedNotifications(ctx context.Context, limit int) ([]models.Notification, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]models.Notification), args.Error(1)
}

func (m *MockRepository) UpdateNotification(ctx context.Context, notification *models.Notification) error {
	args := m.Called(ctx, notification)
	return args.Error(0)
}

func (m *MockRepository) CreateNotificationTemplate(ctx context.Context, template *models.NotificationTemplate) error {
	args := m.Called(ctx, template)
	return args.Error(0)
}

func (m *MockRepository) GetNotificationTemplateByID(ctx context.Context, id uint) (*models.NotificationTemplate, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.NotificationTemplate), args.Error(1)
}

func (m *MockRepository) GetNotificationTemplatesByClub(ctx context.Context, clubID uint) ([]models.NotificationTemplate, error) {
	args := m.Called(ctx, clubID)
	return args.Get(0).([]models.NotificationTemplate), args.Error(1)
}

func (m *MockRepository) UpdateNotificationTemplate(ctx context.Context, template *models.NotificationTemplate) error {
	args := m.Called(ctx, template)
	return args.Error(0)
}

func (m *MockRepository) CreateNotificationPreference(ctx context.Context, preference *models.NotificationPreference) error {
	args := m.Called(ctx, preference)
	return args.Error(0)
}

func (m *MockRepository) GetNotificationPreferences(ctx context.Context, userID string, clubID uint) ([]models.NotificationPreference, error) {
	args := m.Called(ctx, userID, clubID)
	return args.Get(0).([]models.NotificationPreference), args.Error(1)
}

func (m *MockRepository) UpdateNotificationPreference(ctx context.Context, preference *models.NotificationPreference) error {
	args := m.Called(ctx, preference)
	return args.Error(0)
}

func (m *MockRepository) GetNotificationStats(ctx context.Context, clubID uint, fromDate, toDate time.Time) (map[string]interface{}, error) {
	args := m.Called(ctx, clubID, fromDate, toDate)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

// Mock Publisher
type MockPublisher struct {
	mock.Mock
}

func (m *MockPublisher) Publish(ctx context.Context, subject string, data interface{}) error {
	args := m.Called(ctx, subject, data)
	return args.Error(0)
}

func (m *MockPublisher) Subscribe(subject string, handler messaging.MessageHandler) error {
	args := m.Called(subject, handler)
	return args.Error(0)
}

func (m *MockPublisher) Close() error {
	args := m.Called()
	return args.Error(0)
}

func setupTestService(t *testing.T) (*NotificationService, *MockRepository, *MockPublisher) {
	mockRepo := &MockRepository{}
	mockPublisher := &MockPublisher{}
	logger := &logging.MockLogger{}
	monitor := &monitoring.MockMonitor{}

	service := &NotificationService{
		repo:      mockRepo,
		publisher: mockPublisher,
		logger:    logger,
		monitor:   monitor,
	}

	return service, mockRepo, mockPublisher
}

func TestNotificationService_CreateNotification(t *testing.T) {
	service, mockRepo, _ := setupTestService(t)
	ctx := context.Background()

	req := &CreateNotificationRequest{
		ClubID:    1,
		UserID:    "user123",
		Type:      models.NotificationTypeEmail,
		Priority:  models.NotificationPriorityNormal,
		Title:     "Test Notification",
		Message:   "This is a test notification",
		Recipient: "test@example.com",
		Metadata:  map[string]string{"key": "value"},
	}

	// Setup mock expectations
	mockRepo.On("CreateNotification", ctx, mock.AnythingOfType("*models.Notification")).Return(nil).Run(func(args mock.Arguments) {
		notification := args.Get(1).(*models.Notification)
		notification.ID = 123 // Simulate DB assigning ID
		notification.CreatedAt = time.Now()
		notification.UpdatedAt = time.Now()
	})

	// Execute
	result, err := service.CreateNotification(ctx, req)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, uint(123), result.ID)
	assert.Equal(t, req.ClubID, result.ClubID)
	assert.Equal(t, req.UserID, result.UserID)
	assert.Equal(t, req.Type, result.Type)
	assert.Equal(t, req.Priority, result.Priority)
	assert.Equal(t, req.Title, result.Title)
	assert.Equal(t, req.Message, result.Message)
	assert.Equal(t, req.Recipient, result.Recipient)
	assert.Equal(t, models.NotificationStatusPending, result.Status)

	mockRepo.AssertExpectations(t)
}

func TestNotificationService_CreateNotificationWithScheduling(t *testing.T) {
	service, mockRepo, _ := setupTestService(t)
	ctx := context.Background()

	futureTime := time.Now().Add(1 * time.Hour)
	req := &CreateNotificationRequest{
		ClubID:       1,
		UserID:       "user123",
		Type:         models.NotificationTypeEmail,
		Priority:     models.NotificationPriorityNormal,
		Title:        "Scheduled Notification",
		Message:      "This is a scheduled notification",
		Recipient:    "test@example.com",
		ScheduledFor: &futureTime,
	}

	// Setup mock expectations
	mockRepo.On("CreateNotification", ctx, mock.AnythingOfType("*models.Notification")).Return(nil).Run(func(args mock.Arguments) {
		notification := args.Get(1).(*models.Notification)
		notification.ID = 124
		notification.CreatedAt = time.Now()
		notification.UpdatedAt = time.Now()
	})

	// Execute
	result, err := service.CreateNotification(ctx, req)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.ScheduledFor)
	assert.Equal(t, futureTime.Unix(), result.ScheduledFor.Unix())

	mockRepo.AssertExpectations(t)
}

func TestNotificationService_GetNotificationByID(t *testing.T) {
	service, mockRepo, _ := setupTestService(t)
	ctx := context.Background()

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
	mockRepo.On("GetNotificationByID", ctx, uint(123)).Return(expectedNotification, nil)

	// Execute
	result, err := service.GetNotificationByID(ctx, 123)

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, expectedNotification, result)

	mockRepo.AssertExpectations(t)
}

func TestNotificationService_GetNotificationsByClub(t *testing.T) {
	service, mockRepo, _ := setupTestService(t)
	ctx := context.Background()

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
	mockRepo.On("GetNotificationsByClub", ctx, uint(1), 50, 0).Return(expectedNotifications, nil)

	// Execute
	result, err := service.GetNotificationsByClub(ctx, 1, 50, 0)

	// Verify
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, expectedNotifications, result)

	mockRepo.AssertExpectations(t)
}

func TestNotificationService_GetNotificationsByUser(t *testing.T) {
	service, mockRepo, _ := setupTestService(t)
	ctx := context.Background()

	userID := "user123"
	clubID := uint(1)

	expectedNotifications := []models.Notification{
		{
			ID:        1,
			ClubID:    clubID,
			UserID:    userID,
			Type:      models.NotificationTypeEmail,
			Status:    models.NotificationStatusSent,
			Title:     "User Notification 1",
			Message:   "Message 1",
			Recipient: "user123@example.com",
		},
		{
			ID:        2,
			ClubID:    clubID,
			UserID:    userID,
			Type:      models.NotificationTypePush,
			Status:    models.NotificationStatusDelivered,
			Title:     "User Notification 2",
			Message:   "Message 2",
			Recipient: "user123@example.com",
		},
	}

	// Setup mock expectations
	mockRepo.On("GetNotificationsByUser", ctx, userID, clubID, 50, 0).Return(expectedNotifications, nil)

	// Execute
	result, err := service.GetNotificationsByUser(ctx, userID, clubID, 50, 0)

	// Verify
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, expectedNotifications, result)

	mockRepo.AssertExpectations(t)
}

func TestNotificationService_MarkNotificationAsRead(t *testing.T) {
	service, mockRepo, _ := setupTestService(t)
	ctx := context.Background()

	notification := &models.Notification{
		ID:        123,
		ClubID:    1,
		UserID:    "user123",
		Type:      models.NotificationTypeEmail,
		Status:    models.NotificationStatusDelivered,
		Priority:  models.NotificationPriorityNormal,
		Title:     "Test Notification",
		Message:   "This is a test notification",
		Recipient: "test@example.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Setup mock expectations
	mockRepo.On("GetNotificationByID", ctx, uint(123)).Return(notification, nil)
	mockRepo.On("UpdateNotification", ctx, mock.AnythingOfType("*models.Notification")).Return(nil).Run(func(args mock.Arguments) {
		updatedNotification := args.Get(1).(*models.Notification)
		assert.Equal(t, models.NotificationStatusRead, updatedNotification.Status)
		assert.NotNil(t, updatedNotification.ReadAt)
	})

	// Execute
	result, err := service.MarkNotificationAsRead(ctx, 123)

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, models.NotificationStatusRead, result.Status)
	assert.NotNil(t, result.ReadAt)

	mockRepo.AssertExpectations(t)
}

func TestNotificationService_ProcessNotification(t *testing.T) {
	service, mockRepo, mockPublisher := setupTestService(t)
	ctx := context.Background()

	notification := &models.Notification{
		ID:        123,
		ClubID:    1,
		UserID:    "user123",
		Type:      models.NotificationTypeEmail,
		Status:    models.NotificationStatusPending,
		Priority:  models.NotificationPriorityNormal,
		Title:     "Test Notification",
		Message:   "This is a test notification",
		Recipient: "test@example.com",
		Metadata:  map[string]string{"template": "welcome"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Setup mock expectations
	mockRepo.On("GetNotificationByID", ctx, uint(123)).Return(notification, nil)
	mockRepo.On("UpdateNotification", ctx, mock.AnythingOfType("*models.Notification")).Return(nil).Times(2) // Once for sent, once for delivered

	// Mock email sending (simulated by publish)
	mockPublisher.On("Publish", ctx, "notification.email.send", mock.Anything).Return(nil)

	// Execute
	err := service.ProcessNotification(ctx, 123)

	// Verify
	assert.NoError(t, err)

	// Verify that UpdateNotification was called to mark as sent
	mockRepo.AssertExpectations(t)
	mockPublisher.AssertExpectations(t)
}

func TestNotificationService_ProcessScheduledNotifications(t *testing.T) {
	service, mockRepo, _ := setupTestService(t)
	ctx := context.Background()

	pendingNotifications := []models.Notification{
		{
			ID:        1,
			ClubID:    1,
			UserID:    "user1",
			Type:      models.NotificationTypeEmail,
			Status:    models.NotificationStatusPending,
			Title:     "Pending 1",
			Message:   "Message 1",
			Recipient: "user1@example.com",
		},
		{
			ID:        2,
			ClubID:    1,
			UserID:    "user2",
			Type:      models.NotificationTypeSMS,
			Status:    models.NotificationStatusPending,
			Title:     "Pending 2",
			Message:   "Message 2",
			Recipient: "+1234567890",
		},
	}

	// Setup mock expectations
	mockRepo.On("GetPendingNotifications", ctx, 100).Return(pendingNotifications, nil)
	mockRepo.On("UpdateNotification", ctx, mock.AnythingOfType("*models.Notification")).Return(nil).Times(4) // 2 notifications * 2 updates each

	// Execute
	processed := service.ProcessScheduledNotifications(ctx)

	// Verify
	assert.Equal(t, 2, processed)

	mockRepo.AssertExpectations(t)
}

func TestNotificationService_RetryFailedNotifications(t *testing.T) {
	service, mockRepo, _ := setupTestService(t)
	ctx := context.Background()

	failedNotifications := []models.Notification{
		{
			ID:           1,
			ClubID:       1,
			UserID:       "user1",
			Type:         models.NotificationTypeEmail,
			Status:       models.NotificationStatusFailed,
			Title:        "Failed 1",
			Message:      "Message 1",
			Recipient:    "user1@example.com",
			RetryCount:   1,
			FailureReason: "SMTP timeout",
		},
		{
			ID:           2,
			ClubID:       1,
			UserID:       "user2",
			Type:         models.NotificationTypeSMS,
			Status:       models.NotificationStatusFailed,
			Title:        "Failed 2",
			Message:      "Message 2",
			Recipient:    "+1234567890",
			RetryCount:   0,
			FailureReason: "Invalid number",
		},
	}

	// Setup mock expectations
	mockRepo.On("GetFailedNotifications", ctx, 50).Return(failedNotifications, nil)
	mockRepo.On("UpdateNotification", ctx, mock.AnythingOfType("*models.Notification")).Return(nil).Times(4) // 2 notifications * 2 updates each

	// Execute
	retried := service.RetryFailedNotifications(ctx)

	// Verify
	assert.Equal(t, 2, retried)

	mockRepo.AssertExpectations(t)
}

func TestNotificationService_CreateNotificationTemplate(t *testing.T) {
	service, mockRepo, _ := setupTestService(t)
	ctx := context.Background()

	req := &CreateTemplateRequest{
		ClubID:          1,
		Name:            "Welcome Email",
		Description:     "Welcome email template for new members",
		Type:            models.NotificationTypeEmail,
		SubjectTemplate: "Welcome to {{.ClubName}}!",
		BodyTemplate:    "Dear {{.MemberName}}, welcome to our club!",
		DefaultMetadata: map[string]string{"category": "welcome"},
	}

	// Setup mock expectations
	mockRepo.On("CreateNotificationTemplate", ctx, mock.AnythingOfType("*models.NotificationTemplate")).Return(nil).Run(func(args mock.Arguments) {
		template := args.Get(1).(*models.NotificationTemplate)
		template.ID = 456
		template.CreatedAt = time.Now()
		template.UpdatedAt = time.Now()
	})

	// Execute
	result, err := service.CreateNotificationTemplate(ctx, req)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, uint(456), result.ID)
	assert.Equal(t, req.ClubID, result.ClubID)
	assert.Equal(t, req.Name, result.Name)
	assert.Equal(t, req.Description, result.Description)
	assert.Equal(t, req.Type, result.Type)
	assert.Equal(t, req.SubjectTemplate, result.SubjectTemplate)
	assert.Equal(t, req.BodyTemplate, result.BodyTemplate)
	assert.True(t, result.IsActive)

	mockRepo.AssertExpectations(t)
}

func TestNotificationService_GetNotificationTemplatesByClub(t *testing.T) {
	service, mockRepo, _ := setupTestService(t)
	ctx := context.Background()

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
	mockRepo.On("GetNotificationTemplatesByClub", ctx, uint(1)).Return(expectedTemplates, nil)

	// Execute
	result, err := service.GetNotificationTemplatesByClub(ctx, 1)

	// Verify
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, expectedTemplates, result)

	mockRepo.AssertExpectations(t)
}

func TestNotificationService_GetNotificationStats(t *testing.T) {
	service, mockRepo, _ := setupTestService(t)
	ctx := context.Background()

	clubID := uint(1)
	fromDate := time.Now().AddDate(0, 0, -7) // 7 days ago
	toDate := time.Now()

	expectedStats := map[string]interface{}{
		"total":     int64(100),
		"pending":   int64(5),
		"sent":      int64(80),
		"delivered": int64(75),
		"failed":    int64(5),
		"read":      int64(60),
	}

	// Setup mock expectations
	mockRepo.On("GetNotificationStats", ctx, clubID, fromDate, toDate).Return(expectedStats, nil)

	// Execute
	result, err := service.GetNotificationStats(ctx, clubID, fromDate, toDate)

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, expectedStats, result)

	mockRepo.AssertExpectations(t)
}

func TestNotificationService_ValidateCreateNotificationRequest(t *testing.T) {
	service, _, _ := setupTestService(t)

	tests := []struct {
		name    string
		req     *CreateNotificationRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid request",
			req: &CreateNotificationRequest{
				ClubID:    1,
				UserID:    "user123",
				Type:      models.NotificationTypeEmail,
				Priority:  models.NotificationPriorityNormal,
				Title:     "Test",
				Message:   "Test message",
				Recipient: "test@example.com",
			},
			wantErr: false,
		},
		{
			name: "Missing ClubID",
			req: &CreateNotificationRequest{
				UserID:    "user123",
				Type:      models.NotificationTypeEmail,
				Title:     "Test",
				Message:   "Test message",
				Recipient: "test@example.com",
			},
			wantErr: true,
			errMsg:  "club_id is required",
		},
		{
			name: "Missing UserID",
			req: &CreateNotificationRequest{
				ClubID:    1,
				Type:      models.NotificationTypeEmail,
				Title:     "Test",
				Message:   "Test message",
				Recipient: "test@example.com",
			},
			wantErr: true,
			errMsg:  "user_id is required",
		},
		{
			name: "Empty Title",
			req: &CreateNotificationRequest{
				ClubID:    1,
				UserID:    "user123",
				Type:      models.NotificationTypeEmail,
				Message:   "Test message",
				Recipient: "test@example.com",
			},
			wantErr: true,
			errMsg:  "title is required",
		},
		{
			name: "Empty Message",
			req: &CreateNotificationRequest{
				ClubID:    1,
				UserID:    "user123",
				Type:      models.NotificationTypeEmail,
				Title:     "Test",
				Recipient: "test@example.com",
			},
			wantErr: true,
			errMsg:  "message is required",
		},
		{
			name: "Empty Recipient",
			req: &CreateNotificationRequest{
				ClubID:  1,
				UserID:  "user123",
				Type:    models.NotificationTypeEmail,
				Title:   "Test",
				Message: "Test message",
			},
			wantErr: true,
			errMsg:  "recipient is required",
		},
		{
			name: "Invalid email recipient",
			req: &CreateNotificationRequest{
				ClubID:    1,
				UserID:    "user123",
				Type:      models.NotificationTypeEmail,
				Title:     "Test",
				Message:   "Test message",
				Recipient: "invalid-email",
			},
			wantErr: true,
			errMsg:  "invalid email format",
		},
		{
			name: "Invalid SMS recipient",
			req: &CreateNotificationRequest{
				ClubID:    1,
				UserID:    "user123",
				Type:      models.NotificationTypeSMS,
				Title:     "Test",
				Message:   "Test message",
				Recipient: "invalid-phone",
			},
			wantErr: true,
			errMsg:  "invalid phone number format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateCreateNotificationRequest(tt.req)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}