package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/services/notification-service/internal/models"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&models.Notification{},
		&models.NotificationTemplate{},
		&models.NotificationPreference{},
	)
	require.NoError(t, err)

	return db
}

func setupTestRepository(t *testing.T) (*Repository, *gorm.DB) {
	db := setupTestDB(t)
	logger := &logging.MockLogger{}
	repo := NewRepository(db, logger)
	return repo, db
}

func TestRepository_CreateNotification(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	notification := &models.Notification{
		ClubID:    1,
		UserID:    "user123",
		Type:      models.NotificationTypeEmail,
		Status:    models.NotificationStatusPending,
		Priority:  models.NotificationPriorityNormal,
		Title:     "Test Notification",
		Message:   "This is a test notification",
		Recipient: "test@example.com",
		Metadata:  map[string]string{"key": "value"},
	}

	err := repo.CreateNotification(ctx, notification)
	assert.NoError(t, err)
	assert.NotZero(t, notification.ID)
	assert.NotZero(t, notification.CreatedAt)
	assert.NotZero(t, notification.UpdatedAt)
}

func TestRepository_GetNotificationByID(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Create test notification
	notification := &models.Notification{
		ClubID:    1,
		UserID:    "user123",
		Type:      models.NotificationTypeEmail,
		Status:    models.NotificationStatusPending,
		Priority:  models.NotificationPriorityNormal,
		Title:     "Test Notification",
		Message:   "This is a test notification",
		Recipient: "test@example.com",
	}

	err := repo.CreateNotification(ctx, notification)
	require.NoError(t, err)

	// Test getting existing notification
	retrieved, err := repo.GetNotificationByID(ctx, notification.ID)
	assert.NoError(t, err)
	assert.Equal(t, notification.ID, retrieved.ID)
	assert.Equal(t, notification.Title, retrieved.Title)
	assert.Equal(t, notification.Message, retrieved.Message)

	// Test getting non-existent notification
	_, err = repo.GetNotificationByID(ctx, 99999)
	assert.Error(t, err)
	assert.Equal(t, gorm.ErrRecordNotFound, err)
}

func TestRepository_GetNotificationsByClub(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	clubID := uint(1)

	// Create test notifications
	notifications := []*models.Notification{
		{
			ClubID:    clubID,
			UserID:    "user1",
			Type:      models.NotificationTypeEmail,
			Status:    models.NotificationStatusPending,
			Title:     "Notification 1",
			Message:   "Message 1",
			Recipient: "user1@example.com",
		},
		{
			ClubID:    clubID,
			UserID:    "user2",
			Type:      models.NotificationTypeSMS,
			Status:    models.NotificationStatusSent,
			Title:     "Notification 2",
			Message:   "Message 2",
			Recipient: "+1234567890",
		},
		{
			ClubID:    2, // Different club
			UserID:    "user3",
			Type:      models.NotificationTypeEmail,
			Status:    models.NotificationStatusPending,
			Title:     "Notification 3",
			Message:   "Message 3",
			Recipient: "user3@example.com",
		},
	}

	for _, n := range notifications {
		err := repo.CreateNotification(ctx, n)
		require.NoError(t, err)
	}

	// Test getting notifications for club 1
	result, err := repo.GetNotificationsByClub(ctx, clubID, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	// Test pagination
	result, err = repo.GetNotificationsByClub(ctx, clubID, 1, 0)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	result, err = repo.GetNotificationsByClub(ctx, clubID, 1, 1)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestRepository_GetNotificationsByUser(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	userID := "user123"
	clubID := uint(1)

	// Create test notifications
	notifications := []*models.Notification{
		{
			ClubID:    clubID,
			UserID:    userID,
			Type:      models.NotificationTypeEmail,
			Status:    models.NotificationStatusPending,
			Title:     "User Notification 1",
			Message:   "Message 1",
			Recipient: "user123@example.com",
		},
		{
			ClubID:    clubID,
			UserID:    userID,
			Type:      models.NotificationTypePush,
			Status:    models.NotificationStatusSent,
			Title:     "User Notification 2",
			Message:   "Message 2",
			Recipient: "user123@example.com",
		},
		{
			ClubID:    clubID,
			UserID:    "other_user",
			Type:      models.NotificationTypeEmail,
			Status:    models.NotificationStatusPending,
			Title:     "Other User Notification",
			Message:   "Message 3",
			Recipient: "other@example.com",
		},
	}

	for _, n := range notifications {
		err := repo.CreateNotification(ctx, n)
		require.NoError(t, err)
	}

	// Test getting notifications for specific user
	result, err := repo.GetNotificationsByUser(ctx, userID, clubID, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	for _, notification := range result {
		assert.Equal(t, userID, notification.UserID)
		assert.Equal(t, clubID, notification.ClubID)
	}
}

func TestRepository_GetPendingNotifications(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	now := time.Now()
	futureTime := now.Add(1 * time.Hour)

	// Create test notifications
	notifications := []*models.Notification{
		{
			ClubID:    1,
			UserID:    "user1",
			Type:      models.NotificationTypeEmail,
			Status:    models.NotificationStatusPending,
			Priority:  models.NotificationPriorityHigh,
			Title:     "High Priority Pending",
			Message:   "Message 1",
			Recipient: "user1@example.com",
		},
		{
			ClubID:    1,
			UserID:    "user2",
			Type:      models.NotificationTypeEmail,
			Status:    models.NotificationStatusPending,
			Priority:  models.NotificationPriorityNormal,
			Title:     "Normal Priority Pending",
			Message:   "Message 2",
			Recipient: "user2@example.com",
		},
		{
			ClubID:      1,
			UserID:      "user3",
			Type:        models.NotificationTypeEmail,
			Status:      models.NotificationStatusPending,
			Priority:    models.NotificationPriorityNormal,
			Title:       "Future Scheduled",
			Message:     "Message 3",
			Recipient:   "user3@example.com",
			ScheduledFor: &futureTime,
		},
		{
			ClubID:    1,
			UserID:    "user4",
			Type:      models.NotificationTypeEmail,
			Status:    models.NotificationStatusSent,
			Priority:  models.NotificationPriorityNormal,
			Title:     "Already Sent",
			Message:   "Message 4",
			Recipient: "user4@example.com",
		},
	}

	for _, n := range notifications {
		err := repo.CreateNotification(ctx, n)
		require.NoError(t, err)
	}

	// Test getting pending notifications
	result, err := repo.GetPendingNotifications(ctx, 10)
	assert.NoError(t, err)
	assert.Len(t, result, 2) // Only pending and not scheduled for future

	// Verify sorting (high priority first)
	assert.Equal(t, models.NotificationPriorityHigh, result[0].Priority)
	assert.Equal(t, models.NotificationPriorityNormal, result[1].Priority)
}

func TestRepository_GetFailedNotifications(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Create test notifications
	notifications := []*models.Notification{
		{
			ClubID:     1,
			UserID:     "user1",
			Type:       models.NotificationTypeEmail,
			Status:     models.NotificationStatusFailed,
			Priority:   models.NotificationPriorityHigh,
			Title:      "Failed High Priority",
			Message:    "Message 1",
			Recipient:  "user1@example.com",
			RetryCount: 1,
		},
		{
			ClubID:     1,
			UserID:     "user2",
			Type:       models.NotificationTypeEmail,
			Status:     models.NotificationStatusFailed,
			Priority:   models.NotificationPriorityNormal,
			Title:      "Failed Normal Priority",
			Message:    "Message 2",
			Recipient:  "user2@example.com",
			RetryCount: 2,
		},
		{
			ClubID:     1,
			UserID:     "user3",
			Type:       models.NotificationTypeEmail,
			Status:     models.NotificationStatusFailed,
			Priority:   models.NotificationPriorityNormal,
			Title:      "Max Retries Reached",
			Message:    "Message 3",
			Recipient:  "user3@example.com",
			RetryCount: 3,
		},
	}

	for _, n := range notifications {
		err := repo.CreateNotification(ctx, n)
		require.NoError(t, err)
	}

	// Test getting failed notifications that can be retried
	result, err := repo.GetFailedNotifications(ctx, 10)
	assert.NoError(t, err)
	assert.Len(t, result, 2) // Only those with retry_count < 3

	for _, notification := range result {
		assert.Equal(t, models.NotificationStatusFailed, notification.Status)
		assert.True(t, notification.RetryCount < 3)
	}
}

func TestRepository_UpdateNotification(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Create test notification
	notification := &models.Notification{
		ClubID:    1,
		UserID:    "user123",
		Type:      models.NotificationTypeEmail,
		Status:    models.NotificationStatusPending,
		Priority:  models.NotificationPriorityNormal,
		Title:     "Test Notification",
		Message:   "This is a test notification",
		Recipient: "test@example.com",
	}

	err := repo.CreateNotification(ctx, notification)
	require.NoError(t, err)

	// Update notification
	notification.Status = models.NotificationStatusSent
	now := time.Now()
	notification.SentAt = &now

	err = repo.UpdateNotification(ctx, notification)
	assert.NoError(t, err)

	// Verify update
	retrieved, err := repo.GetNotificationByID(ctx, notification.ID)
	assert.NoError(t, err)
	assert.Equal(t, models.NotificationStatusSent, retrieved.Status)
	assert.NotNil(t, retrieved.SentAt)
}

func TestRepository_NotificationTemplate_CRUD(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Test Create
	template := &models.NotificationTemplate{
		ClubID:          1,
		Name:            "Welcome Email",
		Description:     "Welcome email template for new members",
		Type:            models.NotificationTypeEmail,
		SubjectTemplate: "Welcome to {{.ClubName}}!",
		BodyTemplate:    "Dear {{.MemberName}}, welcome to our club!",
		DefaultMetadata: map[string]string{"category": "welcome"},
		IsActive:        true,
	}

	err := repo.CreateNotificationTemplate(ctx, template)
	assert.NoError(t, err)
	assert.NotZero(t, template.ID)

	// Test Get by ID
	retrieved, err := repo.GetNotificationTemplateByID(ctx, template.ID)
	assert.NoError(t, err)
	assert.Equal(t, template.Name, retrieved.Name)
	assert.Equal(t, template.Description, retrieved.Description)

	// Test Get by Club
	templates, err := repo.GetNotificationTemplatesByClub(ctx, 1)
	assert.NoError(t, err)
	assert.Len(t, templates, 1)
	assert.Equal(t, template.ID, templates[0].ID)

	// Test Update
	template.Description = "Updated description"
	err = repo.UpdateNotificationTemplate(ctx, template)
	assert.NoError(t, err)

	updated, err := repo.GetNotificationTemplateByID(ctx, template.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated description", updated.Description)
}

func TestRepository_NotificationPreference_CRUD(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	userID := "user123"
	clubID := uint(1)

	// Test Create
	preference := &models.NotificationPreference{
		UserID:           userID,
		ClubID:           clubID,
		NotificationType: models.NotificationTypeEmail,
		IsEnabled:        true,
		Settings:         map[string]interface{}{"frequency": "daily"},
	}

	err := repo.CreateNotificationPreference(ctx, preference)
	assert.NoError(t, err)
	assert.NotZero(t, preference.ID)

	// Test Get
	preferences, err := repo.GetNotificationPreferences(ctx, userID, clubID)
	assert.NoError(t, err)
	assert.Len(t, preferences, 1)
	assert.Equal(t, preference.ID, preferences[0].ID)

	// Test Update
	preference.IsEnabled = false
	err = repo.UpdateNotificationPreference(ctx, preference)
	assert.NoError(t, err)

	updated, _ := repo.GetNotificationPreferences(ctx, userID, clubID)
	assert.False(t, updated[0].IsEnabled)
}

func TestRepository_GetNotificationStats(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	clubID := uint(1)
	fromDate := time.Now().AddDate(0, 0, -1) // Yesterday
	toDate := time.Now()

	// Create test notifications with different statuses
	notifications := []*models.Notification{
		{
			ClubID:    clubID,
			UserID:    "user1",
			Type:      models.NotificationTypeEmail,
			Status:    models.NotificationStatusPending,
			Title:     "Pending Notification",
			Message:   "Message 1",
			Recipient: "user1@example.com",
		},
		{
			ClubID:    clubID,
			UserID:    "user2",
			Type:      models.NotificationTypeEmail,
			Status:    models.NotificationStatusSent,
			Title:     "Sent Notification",
			Message:   "Message 2",
			Recipient: "user2@example.com",
		},
		{
			ClubID:    clubID,
			UserID:    "user3",
			Type:      models.NotificationTypeEmail,
			Status:    models.NotificationStatusDelivered,
			Title:     "Delivered Notification",
			Message:   "Message 3",
			Recipient: "user3@example.com",
		},
		{
			ClubID:    clubID,
			UserID:    "user4",
			Type:      models.NotificationTypeEmail,
			Status:    models.NotificationStatusFailed,
			Title:     "Failed Notification",
			Message:   "Message 4",
			Recipient: "user4@example.com",
		},
		{
			ClubID:    2, // Different club
			UserID:    "user5",
			Type:      models.NotificationTypeEmail,
			Status:    models.NotificationStatusSent,
			Title:     "Other Club Notification",
			Message:   "Message 5",
			Recipient: "user5@example.com",
		},
	}

	for _, n := range notifications {
		err := repo.CreateNotification(ctx, n)
		require.NoError(t, err)
	}

	// Test getting stats
	stats, err := repo.GetNotificationStats(ctx, clubID, fromDate, toDate)
	assert.NoError(t, err)

	assert.Equal(t, int64(4), stats["total"])
	assert.Equal(t, int64(1), stats["pending"])
	assert.Equal(t, int64(1), stats["sent"])
	assert.Equal(t, int64(1), stats["delivered"])
	assert.Equal(t, int64(1), stats["failed"])
	assert.Equal(t, int64(0), stats["read"])
}