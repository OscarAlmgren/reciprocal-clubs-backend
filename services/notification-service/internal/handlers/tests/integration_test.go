package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"net"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/messaging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	grpcHandlers "reciprocal-clubs-backend/services/notification-service/internal/handlers/grpc"
	httpHandlers "reciprocal-clubs-backend/services/notification-service/internal/handlers/http"
	"reciprocal-clubs-backend/services/notification-service/internal/models"
	"reciprocal-clubs-backend/services/notification-service/internal/providers"
	"reciprocal-clubs-backend/services/notification-service/internal/repository"
	"reciprocal-clubs-backend/services/notification-service/internal/service"
	pb "reciprocal-clubs-backend/services/notification-service/proto"
)

// Integration test suite
type NotificationIntegrationTestSuite struct {
	suite.Suite
	db              *gorm.DB
	service         *service.NotificationService
	httpHandler     *httpHandlers.HTTPHandler
	grpcHandler     *grpcHandlers.GRPCHandler
	grpcServer      *grpc.Server
	grpcClient      pb.NotificationServiceClient
	grpcConn        *grpc.ClientConn
	listener        *bufconn.Listener
	httpServer      *httptest.Server
}

func (suite *NotificationIntegrationTestSuite) SetupSuite() {
	// Setup in-memory database
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

	// Initialize dependencies
	logger := &TestLogger{}
	repo := repository.NewRepository(db, logger)
	mockProviders := &MockNotificationProviders{}
	mockMessaging := &MockMessageBus{}
	mockMonitor := &MockMonitor{}

	// Initialize service
	suite.service = service.NewService(repo, mockProviders, logger, mockMessaging, mockMonitor)

	// Initialize handlers
	suite.httpHandler = httpHandlers.NewHTTPHandler(suite.service, logger, mockMonitor)
	suite.grpcHandler = grpcHandlers.NewGRPCHandler(suite.service, logger, mockMonitor)

	// Setup gRPC server with in-memory connection
	suite.listener = bufconn.Listen(1024 * 1024)
	suite.grpcServer = grpc.NewServer()
	suite.grpcHandler.RegisterServices(suite.grpcServer)

	go func() {
		if err := suite.grpcServer.Serve(suite.listener); err != nil {
			logger.Error("gRPC server failed", map[string]interface{}{"error": err.Error()})
		}
	}()

	// Setup gRPC client
	conn, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return suite.listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	suite.Require().NoError(err)

	suite.grpcConn = conn
	suite.grpcClient = pb.NewNotificationServiceClient(conn)

	// Setup HTTP server
	suite.httpServer = httptest.NewServer(suite.httpHandler.SetupRoutes())
}

func (suite *NotificationIntegrationTestSuite) SetupTest() {
	// Clean up before each test
	suite.db.Exec("DELETE FROM notifications")
	suite.db.Exec("DELETE FROM notification_templates")
	suite.db.Exec("DELETE FROM notification_preferences")
}

func (suite *NotificationIntegrationTestSuite) TearDownSuite() {
	if suite.grpcConn != nil {
		suite.grpcConn.Close()
	}
	if suite.grpcServer != nil {
		suite.grpcServer.Stop()
	}
	if suite.httpServer != nil {
		suite.httpServer.Close()
	}
	if suite.db != nil {
		sqlDB, _ := suite.db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}
}

// Test gRPC CreateNotification
func (suite *NotificationIntegrationTestSuite) TestGRPC_CreateNotification_Success() {
	ctx := context.Background()
	req := &pb.CreateNotificationRequest{
		ClubId:    1,
		UserId:    "user123",
		Type:      pb.NotificationType_NOTIFICATION_TYPE_EMAIL,
		Priority:  pb.NotificationPriority_NOTIFICATION_PRIORITY_NORMAL,
		Title:     "Test Subject",
		Message:   "Test Message",
		Recipient: "test@example.com",
	}

	resp, err := suite.grpcClient.CreateNotification(ctx, req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.NotNil(suite.T(), resp.Notification)
	assert.Equal(suite.T(), req.ClubId, resp.Notification.ClubId)
	assert.Equal(suite.T(), req.Title, resp.Notification.Title)
	assert.Equal(suite.T(), pb.NotificationStatus_NOTIFICATION_STATUS_PENDING, resp.Notification.Status)
}

// Test gRPC GetNotification
func (suite *NotificationIntegrationTestSuite) TestGRPC_GetNotification_Success() {
	ctx := context.Background()

	// First create a notification
	createReq := &pb.CreateNotificationRequest{
		ClubId:    1,
		Type:      pb.NotificationType_NOTIFICATION_TYPE_EMAIL,
		Title:     "Test Subject",
		Message:   "Test Message",
		Recipient: "test@example.com",
	}

	createResp, err := suite.grpcClient.CreateNotification(ctx, createReq)
	suite.Require().NoError(err)

	// Then retrieve it
	getReq := &pb.GetNotificationRequest{
		Id: createResp.Notification.Id,
	}

	getResp, err := suite.grpcClient.GetNotification(ctx, getReq)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), getResp)
	assert.Equal(suite.T(), createResp.Notification.Id, getResp.Notification.Id)
	assert.Equal(suite.T(), createReq.Title, getResp.Notification.Title)
}

// Test gRPC Health check
func (suite *NotificationIntegrationTestSuite) TestGRPC_Health_Success() {
	ctx := context.Background()

	resp, err := suite.grpcClient.Health(ctx, &emptypb.Empty{})

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.Equal(suite.T(), "SERVING", resp.Status)
	assert.Equal(suite.T(), "notification-service", resp.Service)
}

// Test HTTP API endpoints
func (suite *NotificationIntegrationTestSuite) TestHTTP_CreateNotification_Success() {
	reqBody := map[string]interface{}{
		"club_id":   1,
		"user_id":   "user123",
		"type":      "email",
		"priority":  "normal",
		"subject":   "Test Subject",
		"message":   "Test Message",
		"recipient": "test@example.com",
	}

	jsonBody, err := json.Marshal(reqBody)
	suite.Require().NoError(err)

	resp, err := http.Post(
		suite.httpServer.URL+"/api/v1/notifications",
		"application/json",
		bytes.NewReader(jsonBody),
	)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response["notification"])
}

// Test HTTP Health endpoint
func (suite *NotificationIntegrationTestSuite) TestHTTP_Health_Success() {
	resp, err := http.Get(suite.httpServer.URL + "/health")
	suite.Require().NoError(err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "healthy", response["status"])
}

// Test end-to-end notification flow
func (suite *NotificationIntegrationTestSuite) TestE2E_NotificationFlow() {
	ctx := context.Background()

	// 1. Create a notification template
	templateReq := &pb.CreateTemplateRequest{
		ClubId:          1,
		Name:            "Welcome Template",
		Type:            pb.NotificationType_NOTIFICATION_TYPE_EMAIL,
		SubjectTemplate: "Welcome {{.Name}}",
		BodyTemplate:    "Welcome to our club, {{.Name}}!",
	}

	templateResp, err := suite.grpcClient.CreateTemplate(ctx, templateReq)
	suite.Require().NoError(err)
	assert.NotNil(suite.T(), templateResp.Template)

	// 2. Create a notification
	notificationReq := &pb.CreateNotificationRequest{
		ClubId:    1,
		UserId:    "user123",
		Type:      pb.NotificationType_NOTIFICATION_TYPE_EMAIL,
		Priority:  pb.NotificationPriority_NOTIFICATION_PRIORITY_NORMAL,
		Title:     "Welcome John",
		Message:   "Welcome to our club, John!",
		Recipient: "john@example.com",
	}

	notificationResp, err := suite.grpcClient.CreateNotification(ctx, notificationReq)
	suite.Require().NoError(err)
	assert.Equal(suite.T(), pb.NotificationStatus_NOTIFICATION_STATUS_PENDING, notificationResp.Notification.Status)

	// 3. Mark as read
	markReadReq := &pb.MarkAsReadRequest{
		Id: notificationResp.Notification.Id,
	}

	markReadResp, err := suite.grpcClient.MarkAsRead(ctx, markReadReq)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), pb.NotificationStatus_NOTIFICATION_STATUS_READ, markReadResp.Notification.Status)

	// 4. Get club notifications
	clubNotificationsReq := &pb.GetClubNotificationsRequest{
		ClubId: 1,
		Limit:  10,
		Offset: 0,
	}

	clubNotificationsResp, err := suite.grpcClient.GetClubNotifications(ctx, clubNotificationsReq)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), clubNotificationsResp.Notifications, 1)

	// 5. Get templates
	templatesReq := &pb.GetClubTemplatesRequest{
		ClubId: 1,
	}

	templatesResp, err := suite.grpcClient.GetClubTemplates(ctx, templatesReq)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), templatesResp.Templates, 1)
	assert.Equal(suite.T(), "Welcome Template", templatesResp.Templates[0].Name)
}

func TestNotificationIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(NotificationIntegrationTestSuite))
}

// Mock implementations for testing

type MockNotificationProviders struct{}

func (m *MockNotificationProviders) ValidateConfig() error {
	return nil
}

func (m *MockNotificationProviders) TestConnections(ctx context.Context) error {
	return nil
}

type TestLogger struct{}

func (l *TestLogger) Debug(msg string, fields map[string]interface{}) {}
func (l *TestLogger) Info(msg string, fields map[string]interface{})  {}
func (l *TestLogger) Warn(msg string, fields map[string]interface{})  {}
func (l *TestLogger) Error(msg string, fields map[string]interface{}) {}
func (l *TestLogger) Fatal(msg string, fields map[string]interface{}) {}
func (l *TestLogger) With(fields map[string]interface{}) logging.Logger { return l }
func (l *TestLogger) WithContext(ctx context.Context) logging.Logger { return l }

type MockMessageBus struct{}

func (m *MockMessageBus) Publish(ctx context.Context, subject string, data []byte) error {
	return nil
}

func (m *MockMessageBus) Subscribe(subject string, handler messaging.MessageHandler) error {
	return nil
}

func (m *MockMessageBus) Close() error {
	return nil
}

func (m *MockMessageBus) HealthCheck(ctx context.Context) error {
	return nil
}

type MockMonitor struct{}

func (m *MockMonitor) RecordBusinessEvent(eventType, category string) {}

func (m *MockMonitor) RecordHTTPRequest(method, path string, statusCode int, duration time.Duration) {}

func (m *MockMonitor) HealthCheckHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	}
}

func (m *MockMonitor) ReadinessCheckHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
	}
}

func (m *MockMonitor) StartMetricsServer() {}

func (m *MockMonitor) RegisterHealthCheck(checker monitoring.HealthChecker) {}