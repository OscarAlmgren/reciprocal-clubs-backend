package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/services/notification-service/internal/models"
	"reciprocal-clubs-backend/services/notification-service/internal/repository"
)

// Test suite for notification repository
type NotificationRepositoryTestSuite struct {
	suite.Suite
	db     *gorm.DB
	repo   *repository.Repository
	logger logging.Logger
}

func (suite *NotificationRepositoryTestSuite) SetupSuite() {
	// Setup in-memory SQLite database for testing
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	suite.Require().NoError(err)

	// Migrate the schema
	err = db.AutoMigrate(
		&models.Notification{},
		&models.NotificationTemplate{},
		&models.NotificationPreference{},
		&models.UserPreferences{},
	)
	suite.Require().NoError(err)

	suite.db = db
	suite.logger = &TestLogger{}
	suite.repo = repository.NewRepository(db, suite.logger)
}

func (suite *NotificationRepositoryTestSuite) SetupTest() {
	// Clean up tables before each test
	suite.db.Exec("DELETE FROM notifications")
	suite.db.Exec("DELETE FROM notification_templates")
	suite.db.Exec("DELETE FROM notification_preferences")
}

func (suite *NotificationRepositoryTestSuite) TearDownSuite() {
	sqlDB, err := suite.db.DB()
	if err == nil {
		sqlDB.Close()
	}
}

// Test CreateNotification
func (suite *NotificationRepositoryTestSuite) TestCreateNotification_Success() {
	ctx := context.Background()
	notification := &models.Notification{
		ClubID:    1,
		Type:      models.NotificationTypeEmail,
		Priority:  models.NotificationPriorityNormal,
		Subject:   "Test Subject",
		Message:   "Test Message",
		Recipient: "test@example.com",
		Status:    models.NotificationStatusPending,
	}

	err := suite.repo.CreateNotification(ctx, notification)

	assert.NoError(suite.T(), err)
	assert.NotZero(suite.T(), notification.ID)
	assert.NotZero(suite.T(), notification.CreatedAt)
}

// Test GetNotificationByID
func (suite *NotificationRepositoryTestSuite) TestGetNotificationByID_Success() {
	ctx := context.Background()

	// Create test notification
	notification := &models.Notification{
		ClubID:    1,
		Type:      models.NotificationTypeEmail,
		Subject:   "Test Subject",
		Message:   "Test Message",
		Recipient: "test@example.com",
		Status:    models.NotificationStatusPending,
	}
	err := suite.repo.CreateNotification(ctx, notification)
	suite.Require().NoError(err)

	// Retrieve notification
	result, err := suite.repo.GetNotificationByID(ctx, notification.ID)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), notification.ID, result.ID)
	assert.Equal(suite.T(), notification.ClubID, result.ClubID)
	assert.Equal(suite.T(), notification.Subject, result.Subject)
}

func (suite *NotificationRepositoryTestSuite) TestGetNotificationByID_NotFound() {
	ctx := context.Background()

	result, err := suite.repo.GetNotificationByID(ctx, 999)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Equal(suite.T(), gorm.ErrRecordNotFound, err)
}

// Test GetNotificationsByClub
func (suite *NotificationRepositoryTestSuite) TestGetNotificationsByClub_Success() {
	ctx := context.Background()
	clubID := uint(1)

	// Create test notifications
	notifications := []*models.Notification{
		{
			ClubID:    clubID,
			Type:      models.NotificationTypeEmail,
			Subject:   "Test 1",
			Message:   "Message 1",
			Recipient: "test1@example.com",
			Status:    models.NotificationStatusPending,
		},
		{
			ClubID:    clubID,
			Type:      models.NotificationTypeSMS,
			Subject:   "Test 2",
			Message:   "Message 2",
			Recipient: "+1234567890",
			Status:    models.NotificationStatusSent,
		},
		{
			ClubID:    2, // Different club
			Type:      models.NotificationTypeEmail,
			Subject:   "Test 3",
			Message:   "Message 3",
			Recipient: "test3@example.com",
			Status:    models.NotificationStatusPending,
		},
	}

	for _, notification := range notifications {
		err := suite.repo.CreateNotification(ctx, notification)
		suite.Require().NoError(err)
	}

	// Retrieve notifications for club 1
	results, err := suite.repo.GetNotificationsByClub(ctx, clubID, 10, 0)

	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), results, 2) // Only notifications for club 1

	for _, result := range results {
		assert.Equal(suite.T(), clubID, result.ClubID)
	}
}

// Test GetPendingNotifications
func (suite *NotificationRepositoryTestSuite) TestGetPendingNotifications_Success() {
	ctx := context.Background()

	// Create test notifications with different statuses
	notifications := []*models.Notification{
		{
			ClubID:    1,
			Type:      models.NotificationTypeEmail,
			Subject:   "Pending 1",
			Message:   "Message 1",
			Recipient: "test1@example.com",
			Status:    models.NotificationStatusPending,
		},
		{
			ClubID:    1,
			Type:      models.NotificationTypeEmail,
			Subject:   "Pending 2",
			Message:   "Message 2",
			Recipient: "test2@example.com",
			Status:    models.NotificationStatusPending,
		},
		{
			ClubID:    1,
			Type:      models.NotificationTypeEmail,
			Subject:   "Sent",
			Message:   "Message 3",
			Recipient: "test3@example.com",
			Status:    models.NotificationStatusSent,
		},
	}

	for _, notification := range notifications {
		err := suite.repo.CreateNotification(ctx, notification)
		suite.Require().NoError(err)
	}

	// Retrieve pending notifications
	results, err := suite.repo.GetPendingNotifications(ctx, 10)

	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), results, 2) // Only pending notifications

	for _, result := range results {
		assert.Equal(suite.T(), models.NotificationStatusPending, result.Status)
	}
}

// Test MarkMultipleAsRead
func (suite *NotificationRepositoryTestSuite) TestMarkMultipleAsRead_Success() {
	ctx := context.Background()

	// Create test notifications
	notifications := []*models.Notification{
		{
			ClubID:    1,
			Type:      models.NotificationTypeEmail,
			Subject:   "Test 1",
			Message:   "Message 1",
			Recipient: "test1@example.com",
			Status:    models.NotificationStatusSent,
		},
		{
			ClubID:    1,
			Type:      models.NotificationTypeEmail,
			Subject:   "Test 2",
			Message:   "Message 2",
			Recipient: "test2@example.com",
			Status:    models.NotificationStatusSent,
		},
	}

	var notificationIDs []uint
	for _, notification := range notifications {
		err := suite.repo.CreateNotification(ctx, notification)
		suite.Require().NoError(err)
		notificationIDs = append(notificationIDs, notification.ID)
	}

	// Mark as read
	count, err := suite.repo.MarkMultipleAsRead(ctx, notificationIDs)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(2), count)

	// Verify notifications are marked as read
	for _, id := range notificationIDs {
		notification, err := suite.repo.GetNotificationByID(ctx, id)
		suite.Require().NoError(err)
		assert.Equal(suite.T(), models.NotificationStatusRead, notification.Status)
		assert.NotNil(suite.T(), notification.ReadAt)
	}
}

// Test CreateNotificationTemplate
func (suite *NotificationRepositoryTestSuite) TestCreateNotificationTemplate_Success() {
	ctx := context.Background()
	template := &models.NotificationTemplate{
		ClubID:      1,
		Name:        "Welcome Template",
		Type:        models.NotificationTypeEmail,
		Subject:     "Welcome {{.Name}}",
		Body:        "Welcome to our club, {{.Name}}!",
		Variables:   `{"name": "string", "club_name": "string"}`,
		IsActive:    true,
		CreatedByID: "user123",
	}

	err := suite.repo.CreateNotificationTemplate(ctx, template)

	assert.NoError(suite.T(), err)
	assert.NotZero(suite.T(), template.ID)
	assert.NotZero(suite.T(), template.CreatedAt)
}

// Test GetUserPreferences
func (suite *NotificationRepositoryTestSuite) TestGetUserPreferences_DefaultWhenNotFound() {
	ctx := context.Background()
	userID := "user123"
	clubID := uint(1)

	// Should return default preferences when none exist
	preferences, err := suite.repo.GetUserPreferences(ctx, userID, clubID)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), preferences)
	assert.Equal(suite.T(), userID, preferences.UserID)
	assert.Equal(suite.T(), clubID, preferences.ClubID)
	assert.True(suite.T(), preferences.EmailEnabled) // Default should be true
	assert.Equal(suite.T(), "en", preferences.PreferredLang)
	assert.Equal(suite.T(), "UTC", preferences.Timezone)
}

// Test UpsertUserPreferences
func (suite *NotificationRepositoryTestSuite) TestUpsertUserPreferences_Create() {
	ctx := context.Background()
	preference := &models.UserPreferences{
		UserID:        "user123",
		ClubID:        1,
		EmailEnabled:  true,
		SMSEnabled:    false,
		PushEnabled:   true,
		InAppEnabled:  true,
		PreferredLang: "es",
		Timezone:      "America/New_York",
	}

	err := suite.repo.UpsertUserPreferences(ctx, preference)

	assert.NoError(suite.T(), err)
	assert.NotZero(suite.T(), preference.ID)

	// Verify it was created
	result, err := suite.repo.GetUserPreferences(ctx, "user123", 1)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "es", result.PreferredLang)
	assert.False(suite.T(), result.SMSEnabled)
}

// Test GetNotificationStats
func (suite *NotificationRepositoryTestSuite) TestGetNotificationStats_Success() {
	ctx := context.Background()
	clubID := uint(1)
	fromDate := time.Now().Add(-24 * time.Hour)
	toDate := time.Now()

	// Create test notifications with different statuses
	notifications := []*models.Notification{
		{
			ClubID:    clubID,
			Type:      models.NotificationTypeEmail,
			Subject:   "Test 1",
			Message:   "Message 1",
			Recipient: "test1@example.com",
			Status:    models.NotificationStatusPending,
		},
		{
			ClubID:    clubID,
			Type:      models.NotificationTypeEmail,
			Subject:   "Test 2",
			Message:   "Message 2",
			Recipient: "test2@example.com",
			Status:    models.NotificationStatusSent,
		},
		{
			ClubID:    clubID,
			Type:      models.NotificationTypeEmail,
			Subject:   "Test 3",
			Message:   "Message 3",
			Recipient: "test3@example.com",
			Status:    models.NotificationStatusFailed,
		},
	}

	for _, notification := range notifications {
		err := suite.repo.CreateNotification(ctx, notification)
		suite.Require().NoError(err)
	}

	// Get stats
	stats, err := suite.repo.GetNotificationStats(ctx, clubID, fromDate, toDate)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), stats)
	assert.Equal(suite.T(), int64(3), stats["total"])
	assert.Equal(suite.T(), int64(1), stats["pending"])
	assert.Equal(suite.T(), int64(1), stats["sent"])
	assert.Equal(suite.T(), int64(1), stats["failed"])
}

func TestNotificationRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(NotificationRepositoryTestSuite))
}

// Test logger implementation
type TestLogger struct{}

func (l *TestLogger) Debug(msg string, fields map[string]interface{}) {}
func (l *TestLogger) Info(msg string, fields map[string]interface{})  {}
func (l *TestLogger) Warn(msg string, fields map[string]interface{})  {}
func (l *TestLogger) Error(msg string, fields map[string]interface{}) {}
func (l *TestLogger) Fatal(msg string, fields map[string]interface{}) {}
func (l *TestLogger) With(fields map[string]interface{}) logging.Logger { return l }
func (l *TestLogger) WithContext(ctx context.Context) logging.Logger { return l }