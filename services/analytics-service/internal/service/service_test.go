package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"reciprocal-clubs-backend/pkg/shared/config"
	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/messaging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	"reciprocal-clubs-backend/services/analytics-service/internal/integrations"
	"reciprocal-clubs-backend/services/analytics-service/internal/repository"
)

// Mock implementations
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) IsHealthy() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockRepository) RecordEvent(event *repository.AnalyticsEvent) error {
	args := m.Called(event)
	return args.Error(0)
}

func (m *MockRepository) GetMetricsByClub(clubID string, timeRange repository.TimeRange) ([]*repository.AnalyticsMetric, error) {
	args := m.Called(clubID, timeRange)
	return args.Get(0).([]*repository.AnalyticsMetric), args.Error(1)
}

func (m *MockRepository) GetReportsByClub(clubID string, reportType string) ([]*repository.AnalyticsReport, error) {
	args := m.Called(clubID, reportType)
	return args.Get(0).([]*repository.AnalyticsReport), args.Error(1)
}

func (m *MockRepository) AggregateMetrics(clubID string, timeRange repository.TimeRange) (map[string]interface{}, error) {
	args := m.Called(clubID, timeRange)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockRepository) CreateReport(report *repository.AnalyticsReport) error {
	args := m.Called(report)
	return args.Error(0)
}

func (m *MockRepository) RecordMetric(metric *repository.AnalyticsMetric) error {
	args := m.Called(metric)
	return args.Error(0)
}

func (m *MockRepository) GetEventsByClub(clubID string, timeRange repository.TimeRange) ([]*repository.AnalyticsEvent, error) {
	args := m.Called(clubID, timeRange)
	return args.Get(0).([]*repository.AnalyticsEvent), args.Error(1)
}

func (m *MockRepository) GetRealtimeMetrics(clubID string) (map[string]interface{}, error) {
	args := m.Called(clubID)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockRepository) CleanupOldEvents(olderThan time.Time) error {
	args := m.Called(olderThan)
	return args.Error(0)
}

// Add other mock methods for advanced analytics, dashboard, and export operations
func (m *MockRepository) GetTrendAnalysis(clubID string, metricName string, timeRange repository.TimeRange) (map[string]interface{}, error) {
	args := m.Called(clubID, metricName, timeRange)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockRepository) GetCorrelationAnalysis(clubID string, metricNames []string, timeRange repository.TimeRange) (map[string]interface{}, error) {
	args := m.Called(clubID, metricNames, timeRange)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockRepository) GetPredictiveAnalytics(clubID string, metricName string, forecastDays int) (map[string]interface{}, error) {
	args := m.Called(clubID, metricName, forecastDays)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockRepository) GetAnomalyDetection(clubID string, metricName string, timeRange repository.TimeRange) (map[string]interface{}, error) {
	args := m.Called(clubID, metricName, timeRange)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockRepository) CreateDashboard(dashboard *repository.Dashboard) error {
	args := m.Called(dashboard)
	return args.Error(0)
}

func (m *MockRepository) GetDashboard(dashboardID uint) (*repository.Dashboard, error) {
	args := m.Called(dashboardID)
	return args.Get(0).(*repository.Dashboard), args.Error(1)
}

func (m *MockRepository) UpdateDashboard(dashboard *repository.Dashboard) error {
	args := m.Called(dashboard)
	return args.Error(0)
}

func (m *MockRepository) DeleteDashboard(dashboardID uint) error {
	args := m.Called(dashboardID)
	return args.Error(0)
}

func (m *MockRepository) ListDashboards(clubID string, limit, offset int) ([]*repository.Dashboard, error) {
	args := m.Called(clubID, limit, offset)
	return args.Get(0).([]*repository.Dashboard), args.Error(1)
}

func (m *MockRepository) ExportEvents(clubID string, timeRange repository.TimeRange, format string) ([]byte, error) {
	args := m.Called(clubID, timeRange, format)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockRepository) ExportMetrics(clubID string, timeRange repository.TimeRange, format string) ([]byte, error) {
	args := m.Called(clubID, timeRange, format)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockRepository) ExportReports(clubID string, format string) ([]byte, error) {
	args := m.Called(clubID, format)
	return args.Get(0).([]byte), args.Error(1)
}

// Mock example methods (can be removed in production)
func (m *MockRepository) CreateExample(example interface{}) error {
	args := m.Called(example)
	return args.Error(0)
}

func (m *MockRepository) GetExampleByID(id uint) (interface{}, error) {
	args := m.Called(id)
	return args.Get(0), args.Error(1)
}

func (m *MockRepository) UpdateExample(example interface{}) error {
	args := m.Called(example)
	return args.Error(0)
}

func (m *MockRepository) DeleteExample(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockRepository) ListExamples(limit, offset int) ([]interface{}, error) {
	args := m.Called(limit, offset)
	return args.Get(0).([]interface{}), args.Error(1)
}

type MockMessageBus struct {
	mock.Mock
}

func (m *MockMessageBus) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockMessageBus) Publish(ctx context.Context, subject string, data []byte) error {
	args := m.Called(ctx, subject, data)
	return args.Error(0)
}

func (m *MockMessageBus) Subscribe(subject string, handler messaging.MessageHandler) error {
	args := m.Called(subject, handler)
	return args.Error(0)
}

func (m *MockMessageBus) Close() error {
	args := m.Called()
	return args.Error(0)
}

type ServiceTestSuite struct {
	suite.Suite
	mockRepo        *MockRepository
	mockNATS        *MockMessageBus
	logger          logging.Logger
	monitor         *monitoring.Monitor
	integrations    *integrations.AnalyticsIntegrations
	service         AnalyticsService
}

func (suite *ServiceTestSuite) SetupTest() {
	suite.mockRepo = new(MockRepository)
	suite.mockNATS = new(MockMessageBus)
	loggingConfig := &config.LoggingConfig{Level: "info", Format: "console", Output: "stdout"}
	suite.logger = logging.NewLogger(loggingConfig, "analytics-service-test")
	suite.monitor = monitoring.NewMonitor(monitoring.Config{ServiceName: "analytics-service-test"})
	suite.integrations = integrations.NewAnalyticsIntegrations(integrations.Config{}, suite.logger)

	suite.service = NewService(
		suite.mockRepo,
		suite.logger,
		suite.mockNATS,
		suite.monitor,
		suite.integrations,
	)
}

func (suite *ServiceTestSuite) TearDownTest() {
	suite.mockRepo.AssertExpectations(suite.T())
	suite.mockNATS.AssertExpectations(suite.T())
}

func (suite *ServiceTestSuite) TestIsReady() {
	// Test case: all dependencies healthy
	suite.mockRepo.On("IsHealthy").Return(true)
	suite.mockNATS.On("HealthCheck", mock.AnythingOfType("*context.emptyCtx")).Return(nil)

	ready := suite.service.IsReady()
	assert.True(suite.T(), ready)

	// Test case: repository unhealthy
	suite.mockRepo.On("IsHealthy").Return(false)

	ready = suite.service.IsReady()
	assert.False(suite.T(), ready)
}

func (suite *ServiceTestSuite) TestRecordEvent() {
	eventData := map[string]interface{}{
		"club_id":    "test-club-1",
		"event_type": "member_visit",
		"user_id":    "user-123",
	}

	// Setup expectations
	suite.mockRepo.On("RecordEvent", mock.AnythingOfType("*repository.AnalyticsEvent")).Return(nil)
	suite.mockNATS.On("Publish", mock.AnythingOfType("*context.emptyCtx"), "analytics.events.member_visit", mock.AnythingOfType("[]uint8")).Return(nil)

	err := suite.service.RecordEvent(eventData)
	assert.NoError(suite.T(), err)
}

func (suite *ServiceTestSuite) TestRecordEventValidationError() {
	eventData := map[string]interface{}{
		"event_type": "member_visit", // Missing club_id
	}

	err := suite.service.RecordEvent(eventData)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "club_id and event_type are required")
}

func (suite *ServiceTestSuite) TestGetMetrics() {
	clubID := "test-club-1"
	timeRange := "24h"

	mockAggregation := map[string]interface{}{
		"total_events":       100,
		"unique_event_types": 5,
	}

	mockMetrics := []*repository.AnalyticsMetric{
		{
			ID:          1,
			ClubID:      clubID,
			MetricName:  "visitor_count",
			MetricValue: 25.0,
		},
	}

	// Setup expectations
	suite.mockRepo.On("AggregateMetrics", clubID, mock.AnythingOfType("repository.TimeRange")).Return(mockAggregation, nil)
	suite.mockRepo.On("GetMetricsByClub", clubID, mock.AnythingOfType("repository.TimeRange")).Return(mockMetrics, nil)

	result, err := suite.service.GetMetrics(clubID, timeRange)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), result, "summary")
	assert.Contains(suite.T(), result, "details")
	assert.Equal(suite.T(), clubID, result["club_id"])
}

func (suite *ServiceTestSuite) TestGetMetricsInvalidTimeRange() {
	clubID := "test-club-1"
	timeRange := "invalid"

	_, err := suite.service.GetMetrics(clubID, timeRange)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "invalid time range")
}

func (suite *ServiceTestSuite) TestGetReports() {
	clubID := "test-club-1"
	reportType := "usage"

	mockReports := []*repository.AnalyticsReport{
		{
			ID:          1,
			ClubID:      clubID,
			ReportType:  reportType,
			Title:       "Usage Report",
			Data:        map[string]interface{}{"total_visits": 100},
			GeneratedAt: time.Now(),
		},
	}

	// Setup expectations
	suite.mockRepo.On("GetReportsByClub", clubID, reportType).Return(mockReports, nil)

	result, err := suite.service.GetReports(clubID, reportType)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), result, 1)
	assert.Equal(suite.T(), clubID, result[0]["club_id"])
	assert.Equal(suite.T(), reportType, result[0]["report_type"])
}

func (suite *ServiceTestSuite) TestGenerateReport() {
	clubID := "test-club-1"
	reportType := "usage"

	// Setup expectations
	suite.mockRepo.On("CreateReport", mock.AnythingOfType("*repository.AnalyticsReport")).Return(nil)

	result, err := suite.service.GenerateReport(clubID, reportType)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), clubID, result["club_id"])
	assert.Equal(suite.T(), reportType, result["report_type"])
	assert.Contains(suite.T(), result, "data")
	assert.Contains(suite.T(), result, "generated_at")
}

func (suite *ServiceTestSuite) TestGenerateReportUnsupportedType() {
	clubID := "test-club-1"
	reportType := "unsupported"

	_, err := suite.service.GenerateReport(clubID, reportType)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "unsupported report type")
}

func (suite *ServiceTestSuite) TestRecordMetric() {
	clubID := "test-club-1"
	metricName := "visitor_count"
	value := 25.0
	tags := map[string]interface{}{"location": "entrance"}

	// Setup expectations
	suite.mockRepo.On("RecordMetric", mock.AnythingOfType("*repository.AnalyticsMetric")).Return(nil)

	err := suite.service.RecordMetric(clubID, metricName, value, tags)
	assert.NoError(suite.T(), err)
}

func (suite *ServiceTestSuite) TestRecordMetricValidationError() {
	err := suite.service.RecordMetric("", "metric", 1.0, nil)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "club_id and metric_name are required")
}

func (suite *ServiceTestSuite) TestGetEvents() {
	clubID := "test-club-1"
	timeRange := "24h"

	mockEvents := []*repository.AnalyticsEvent{
		{
			ID:        1,
			ClubID:    clubID,
			EventType: "member_visit",
			Data:      map[string]interface{}{"member_id": "123"},
			Timestamp: time.Now(),
			CreatedAt: time.Now(),
		},
	}

	// Setup expectations
	suite.mockRepo.On("GetEventsByClub", clubID, mock.AnythingOfType("repository.TimeRange")).Return(mockEvents, nil)

	result, err := suite.service.GetEvents(clubID, timeRange)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), result, 1)
	assert.Equal(suite.T(), clubID, result[0]["club_id"])
	assert.Equal(suite.T(), "member_visit", result[0]["event_type"])
}

func (suite *ServiceTestSuite) TestGetRealtimeMetrics() {
	clubID := "test-club-1"

	mockMetrics := map[string]interface{}{
		"recent_events":  5,
		"recent_metrics": 3,
		"timestamp":      time.Now(),
	}

	// Setup expectations
	suite.mockRepo.On("GetRealtimeMetrics", clubID).Return(mockMetrics, nil)

	result, err := suite.service.GetRealtimeMetrics(clubID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), mockMetrics, result)
}

func (suite *ServiceTestSuite) TestCleanupOldData() {
	days := 30

	// Setup expectations
	suite.mockRepo.On("CleanupOldEvents", mock.AnythingOfType("time.Time")).Return(nil)

	err := suite.service.CleanupOldData(days)
	assert.NoError(suite.T(), err)
}

func (suite *ServiceTestSuite) TestCleanupOldDataInvalidDays() {
	days := 0

	err := suite.service.CleanupOldData(days)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "days must be greater than 0")
}

func (suite *ServiceTestSuite) TestGetSystemHealth() {
	// Setup expectations
	suite.mockRepo.On("IsHealthy").Return(true)
	suite.mockNATS.On("HealthCheck", mock.AnythingOfType("*context.emptyCtx")).Return(nil)

	result := suite.service.GetSystemHealth()
	assert.Contains(suite.T(), result, "status")
	assert.Contains(suite.T(), result, "components")
	assert.Contains(suite.T(), result, "timestamp")

	components := result["components"].(map[string]interface{})
	assert.True(suite.T(), components["database"].(bool))
	assert.True(suite.T(), components["nats"].(bool))
}

func (suite *ServiceTestSuite) TestProcessAnalyticsEvent() {
	eventType := "member_visit"
	data := map[string]interface{}{
		"member_id": "123",
		"club_id":   "test-club-1",
	}

	err := suite.service.ProcessAnalyticsEvent(eventType, data)
	assert.NoError(suite.T(), err)
}

func (suite *ServiceTestSuite) TestProcessAnalyticsEventUnknownType() {
	eventType := "unknown_event"
	data := map[string]interface{}{}

	err := suite.service.ProcessAnalyticsEvent(eventType, data)
	assert.NoError(suite.T(), err) // Should not error for unknown types, just log
}

func (suite *ServiceTestSuite) TestParseTimeRange() {
	s := &service{} // Access private method through type assertion

	testCases := []struct {
		input    string
		expected bool
	}{
		{"1h", true},
		{"24h", true},
		{"7d", true},
		{"30d", true},
		{"invalid", false},
	}

	for _, tc := range testCases {
		result, err := s.parseTimeRange(tc.input)
		if tc.expected {
			assert.NoError(suite.T(), err, "Input: %s", tc.input)
			assert.NotNil(suite.T(), result, "Input: %s", tc.input)
		} else {
			assert.Error(suite.T(), err, "Input: %s", tc.input)
		}
	}
}

func TestServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ServiceTestSuite))
}

// Integration tests (using real dependencies but with test configuration)
func TestServiceIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	// These would use real database and messaging connections
	// but with test configurations
	t.Run("RealDatabaseIntegration", func(t *testing.T) {
		// Test with actual database connection
		t.Skip("Integration test - implement with real database")
	})

	t.Run("RealMessagingIntegration", func(t *testing.T) {
		// Test with actual NATS connection
		t.Skip("Integration test - implement with real messaging")
	})
}

// Benchmark tests
func BenchmarkRecordEvent(b *testing.B) {
	mockRepo := new(MockRepository)
	mockNATS := new(MockMessageBus)
	loggingConfig := &config.LoggingConfig{Level: "error", Format: "console", Output: "stdout"}
	logger := logging.NewLogger(loggingConfig, "analytics-service-bench")
	monitor := monitoring.NewMonitor(monitoring.Config{ServiceName: "test"})
	integrations := integrations.NewAnalyticsIntegrations(integrations.Config{}, logger)

	service := NewService(mockRepo, logger, mockNATS, monitor, integrations)

	mockRepo.On("RecordEvent", mock.AnythingOfType("*repository.AnalyticsEvent")).Return(nil)
	mockNATS.On("Publish", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8")).Return(nil)

	eventData := map[string]interface{}{
		"club_id":    "test-club-1",
		"event_type": "member_visit",
		"user_id":    "user-123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.RecordEvent(eventData)
	}
}

func BenchmarkGetMetrics(b *testing.B) {
	mockRepo := new(MockRepository)
	mockNATS := new(MockMessageBus)
	loggingConfig := &config.LoggingConfig{Level: "error", Format: "console", Output: "stdout"}
	logger := logging.NewLogger(loggingConfig, "analytics-service-bench")
	monitor := monitoring.NewMonitor(monitoring.Config{ServiceName: "test"})
	integrations := integrations.NewAnalyticsIntegrations(integrations.Config{}, logger)

	service := NewService(mockRepo, logger, mockNATS, monitor, integrations)

	mockAggregation := map[string]interface{}{"total": 100}
	mockMetrics := []*repository.AnalyticsMetric{}

	mockRepo.On("AggregateMetrics", mock.AnythingOfType("string"), mock.AnythingOfType("repository.TimeRange")).Return(mockAggregation, nil)
	mockRepo.On("GetMetricsByClub", mock.AnythingOfType("string"), mock.AnythingOfType("repository.TimeRange")).Return(mockMetrics, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.GetMetrics("test-club-1", "24h")
	}
}