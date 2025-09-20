package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/types/known/emptypb"

	"reciprocal-clubs-backend/pkg/shared/config"
	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	pb "reciprocal-clubs-backend/services/analytics-service/proto"
)

// Mock service for testing
type MockAnalyticsService struct {
	mock.Mock
}

func (m *MockAnalyticsService) IsReady() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockAnalyticsService) GetMetrics(clubID string, timeRange string) (map[string]interface{}, error) {
	args := m.Called(clubID, timeRange)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockAnalyticsService) GetReports(clubID string, reportType string) ([]map[string]interface{}, error) {
	args := m.Called(clubID, reportType)
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *MockAnalyticsService) RecordEvent(eventData map[string]interface{}) error {
	args := m.Called(eventData)
	return args.Error(0)
}

func (m *MockAnalyticsService) GenerateReport(clubID string, reportType string) (map[string]interface{}, error) {
	args := m.Called(clubID, reportType)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockAnalyticsService) GetRealtimeMetrics(clubID string) (map[string]interface{}, error) {
	args := m.Called(clubID)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockAnalyticsService) RecordMetric(clubID string, metricName string, value float64, tags map[string]interface{}) error {
	args := m.Called(clubID, metricName, value, tags)
	return args.Error(0)
}

func (m *MockAnalyticsService) GetEvents(clubID string, timeRange string) ([]map[string]interface{}, error) {
	args := m.Called(clubID, timeRange)
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *MockAnalyticsService) CleanupOldData(days int) error {
	args := m.Called(days)
	return args.Error(0)
}

func (m *MockAnalyticsService) GetSystemHealth() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

func (m *MockAnalyticsService) ExportData(exportType string, data interface{}) error {
	args := m.Called(exportType, data)
	return args.Error(0)
}

func (m *MockAnalyticsService) CreateDashboard(clubID string) error {
	args := m.Called(clubID)
	return args.Error(0)
}

func (m *MockAnalyticsService) SendMetricsToExternal(metrics map[string]interface{}) error {
	args := m.Called(metrics)
	return args.Error(0)
}

func (m *MockAnalyticsService) GetHealthChecker() interface{} {
	args := m.Called()
	return args.Get(0)
}

func (m *MockAnalyticsService) GetMonitoringMetrics() interface{} {
	args := m.Called()
	return args.Get(0)
}

func (m *MockAnalyticsService) ProcessAnalyticsEvent(eventType string, data map[string]interface{}) error {
	args := m.Called(eventType, data)
	return args.Error(0)
}

func (m *MockAnalyticsService) StartEventProcessor() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockAnalyticsService) StopEventProcessor() error {
	args := m.Called()
	return args.Error(0)
}

// Mock health checker
type MockHealthChecker struct {
	mock.Mock
}

func (m *MockHealthChecker) HealthCheck(ctx context.Context) map[string]interface{} {
	args := m.Called(ctx)
	return args.Get(0).(map[string]interface{})
}

type GRPCHandlerTestSuite struct {
	suite.Suite
	mockService *MockAnalyticsService
	handler     *GRPCHandler
	ctx         context.Context
}

func (suite *GRPCHandlerTestSuite) SetupTest() {
	suite.mockService = new(MockAnalyticsService)
	loggingConfig := &config.LoggingConfig{Level: "info", Format: "console", Output: "stdout"}
	logger := logging.NewLogger(loggingConfig, "analytics-service-test")
	monitor := monitoring.NewMonitor(monitoring.Config{ServiceName: "analytics-service-test"})

	suite.handler = NewGRPCHandler(suite.mockService, logger, monitor)
	suite.ctx = context.Background()
}

func (suite *GRPCHandlerTestSuite) TearDownTest() {
	suite.mockService.AssertExpectations(suite.T())
}

func (suite *GRPCHandlerTestSuite) TestHealth() {
	mockHealthChecker := new(MockHealthChecker)
	healthResponse := map[string]interface{}{
		"status": "healthy",
		"dependencies": map[string]interface{}{
			"database": "healthy",
			"nats":     "healthy",
		},
	}

	suite.mockService.On("GetHealthChecker").Return(mockHealthChecker)
	mockHealthChecker.On("HealthCheck", suite.ctx).Return(healthResponse)

	req := &emptypb.Empty{}
	resp, err := suite.handler.Health(suite.ctx, req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.Equal(suite.T(), "SERVING", resp.Status)
	assert.Equal(suite.T(), "analytics-service", resp.Service)
}

func (suite *GRPCHandlerTestSuite) TestGetMetrics() {
	clubID := "test-club-1"
	timeRange := "24h"

	mockMetrics := map[string]interface{}{
		"summary": map[string]interface{}{
			"total_events": "100",
			"total_users":  "25",
		},
		"details": []interface{}{},
	}

	suite.mockService.On("GetMetrics", clubID, timeRange).Return(mockMetrics, nil)

	req := &pb.GetMetricsRequest{
		ClubId:    clubID,
		TimeRange: timeRange,
	}

	resp, err := suite.handler.GetMetrics(suite.ctx, req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.Equal(suite.T(), clubID, resp.ClubId)
	assert.Equal(suite.T(), timeRange, resp.TimeRange)
	assert.Contains(suite.T(), resp.Summary, "total_events")
}

func (suite *GRPCHandlerTestSuite) TestGetReports() {
	clubID := "test-club-1"
	reportType := pb.ReportType_REPORT_TYPE_USAGE

	mockReports := []map[string]interface{}{
		{
			"id":           uint(1),
			"club_id":      clubID,
			"title":        "Usage Report",
			"data":         map[string]interface{}{"total_visits": 100},
			"generated_at": time.Now(),
			"created_at":   time.Now(),
		},
	}

	suite.mockService.On("GetReports", clubID, "usage").Return(mockReports, nil)

	req := &pb.GetReportsRequest{
		ClubId:     clubID,
		ReportType: reportType,
	}

	resp, err := suite.handler.GetReports(suite.ctx, req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.Len(suite.T(), resp.Reports, 1)
	assert.Equal(suite.T(), uint32(1), resp.Reports[0].Id)
	assert.Equal(suite.T(), clubID, resp.Reports[0].ClubId)
}

func (suite *GRPCHandlerTestSuite) TestRecordEvent() {
	clubID := "test-club-1"
	eventType := "member_visit"
	userID := "user-123"

	suite.mockService.On("RecordEvent", mock.AnythingOfType("map[string]interface {}")).Return(nil)

	req := &pb.RecordEventRequest{
		ClubId:    clubID,
		EventType: eventType,
		UserId:    userID,
		Data:      map[string]string{"location": "gym"},
		Metadata:  map[string]string{"device": "mobile"},
	}

	resp, err := suite.handler.RecordEvent(suite.ctx, req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.True(suite.T(), resp.Success)
	assert.Equal(suite.T(), "Event recorded successfully", resp.Message)
}

func (suite *GRPCHandlerTestSuite) TestRecordMetric() {
	clubID := "test-club-1"
	metricName := "visitor_count"
	metricValue := 25.0

	suite.mockService.On("RecordMetric", clubID, metricName, metricValue, mock.AnythingOfType("map[string]interface {}")).Return(nil)

	req := &pb.RecordMetricRequest{
		ClubId:      clubID,
		MetricName:  metricName,
		MetricValue: metricValue,
		Tags:        map[string]string{"location": "entrance"},
	}

	resp, err := suite.handler.RecordMetric(suite.ctx, req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.True(suite.T(), resp.Success)
	assert.Equal(suite.T(), "Metric recorded successfully", resp.Message)
}

func (suite *GRPCHandlerTestSuite) TestGenerateReport() {
	clubID := "test-club-1"
	reportType := pb.ReportType_REPORT_TYPE_USAGE

	mockReport := map[string]interface{}{
		"title":        "Usage Report",
		"data":         map[string]interface{}{"total_visits": 100},
		"generated_at": time.Now(),
	}

	suite.mockService.On("GenerateReport", clubID, "usage").Return(mockReport, nil)

	req := &pb.GenerateReportRequest{
		ClubId:     clubID,
		ReportType: reportType,
	}

	resp, err := suite.handler.GenerateReport(suite.ctx, req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.True(suite.T(), resp.Success)
	assert.Equal(suite.T(), "Report generated successfully", resp.Message)
	assert.NotNil(suite.T(), resp.Report)
	assert.Equal(suite.T(), clubID, resp.Report.ClubId)
}

func (suite *GRPCHandlerTestSuite) TestGetRealtimeMetrics() {
	clubID := "test-club-1"

	mockMetrics := map[string]interface{}{
		"recent_events":  5.0,
		"recent_metrics": 3.0,
		"active_users":   12.0,
	}

	suite.mockService.On("GetRealtimeMetrics", clubID).Return(mockMetrics, nil)

	req := &pb.GetRealtimeMetricsRequest{
		ClubId: clubID,
	}

	resp, err := suite.handler.GetRealtimeMetrics(suite.ctx, req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.Equal(suite.T(), 5.0, resp.Metrics["recent_events"])
	assert.Equal(suite.T(), 3.0, resp.Metrics["recent_metrics"])
	assert.NotNil(suite.T(), resp.Timestamp)
}

func (suite *GRPCHandlerTestSuite) TestGetLiveStats() {
	clubID := "test-club-1"

	mockStats := map[string]interface{}{
		"active_sessions": 15.0,
		"current_load":    0.75,
	}

	suite.mockService.On("GetRealtimeMetrics", clubID).Return(mockStats, nil)

	req := &pb.GetLiveStatsRequest{
		ClubId: clubID,
	}

	resp, err := suite.handler.GetLiveStats(suite.ctx, req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.Equal(suite.T(), 15.0, resp.Stats["active_sessions"])
	assert.NotNil(suite.T(), resp.Timestamp)
}

func (suite *GRPCHandlerTestSuite) TestGetEvents() {
	clubID := "test-club-1"
	timeRange := "24h"

	mockEvents := []map[string]interface{}{
		{
			"id":         uint(1),
			"club_id":    clubID,
			"event_type": "member_visit",
			"data":       map[string]interface{}{"member_id": "123"},
			"timestamp":  time.Now(),
			"created_at": time.Now(),
		},
	}

	suite.mockService.On("GetEvents", clubID, timeRange).Return(mockEvents, nil)

	req := &pb.GetEventsRequest{
		ClubId:    clubID,
		TimeRange: timeRange,
	}

	resp, err := suite.handler.GetEvents(suite.ctx, req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.Len(suite.T(), resp.Events, 1)
	assert.Equal(suite.T(), uint32(1), resp.Events[0].Id)
	assert.Equal(suite.T(), clubID, resp.Events[0].ClubId)
}

func (suite *GRPCHandlerTestSuite) TestBulkRecordEvents() {
	events := []*pb.RecordEventRequest{
		{
			ClubId:    "test-club-1",
			EventType: "member_visit",
			UserId:    "user-123",
		},
		{
			ClubId:    "test-club-1",
			EventType: "facility_usage",
			UserId:    "user-456",
		},
	}

	// Setup expectations for successful event recording
	suite.mockService.On("RecordEvent", mock.AnythingOfType("map[string]interface {}")).Return(nil).Times(2)

	req := &pb.BulkRecordEventsRequest{
		Events: events,
	}

	resp, err := suite.handler.BulkRecordEvents(suite.ctx, req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.True(suite.T(), resp.Success)
	assert.Equal(suite.T(), int32(2), resp.ProcessedCount)
	assert.Equal(suite.T(), int32(0), resp.ErrorCount)
}

func (suite *GRPCHandlerTestSuite) TestGetSystemHealth() {
	mockHealth := map[string]interface{}{
		"status": "healthy",
		"components": map[string]interface{}{
			"database": true,
			"nats":     true,
		},
	}

	suite.mockService.On("GetSystemHealth").Return(mockHealth)

	req := &emptypb.Empty{}
	resp, err := suite.handler.GetSystemHealth(suite.ctx, req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.Equal(suite.T(), "healthy", resp.Status)
	assert.Contains(suite.T(), resp.Components, "database")
	assert.Equal(suite.T(), "healthy", resp.Components["database"])
}

func (suite *GRPCHandlerTestSuite) TestCleanupOldData() {
	days := int32(30)

	suite.mockService.On("CleanupOldData", int(days)).Return(nil)

	req := &pb.CleanupOldDataRequest{
		Days: days,
	}

	resp, err := suite.handler.CleanupOldData(suite.ctx, req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.True(suite.T(), resp.Success)
	assert.Equal(suite.T(), "Data cleanup completed successfully", resp.Message)
}

func (suite *GRPCHandlerTestSuite) TestGetReportStatus() {
	jobID := "job-123"

	req := &pb.GetReportStatusRequest{
		JobId: jobID,
	}

	resp, err := suite.handler.GetReportStatus(suite.ctx, req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.Equal(suite.T(), "completed", resp.Status)
	assert.Equal(suite.T(), int32(100), resp.Progress)
}

func (suite *GRPCHandlerTestSuite) TestScheduleReport() {
	clubID := "test-club-1"
	reportType := pb.ReportType_REPORT_TYPE_USAGE
	schedule := "0 9 * * *" // Daily at 9 AM

	req := &pb.ScheduleReportRequest{
		ClubId:     clubID,
		ReportType: reportType,
		Schedule:   schedule,
	}

	resp, err := suite.handler.ScheduleReport(suite.ctx, req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.True(suite.T(), resp.Success)
	assert.Equal(suite.T(), "Report scheduled successfully", resp.Message)
	assert.NotEmpty(suite.T(), resp.ScheduleId)
}

func (suite *GRPCHandlerTestSuite) TestGetServiceMetrics() {
	req := &emptypb.Empty{}

	resp, err := suite.handler.GetServiceMetrics(suite.ctx, req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.Contains(suite.T(), resp.Counters, "total_requests")
	assert.Contains(suite.T(), resp.Gauges, "active_connections")
	assert.Contains(suite.T(), resp.Histograms, "request_duration_ms")
	assert.NotNil(suite.T(), resp.Timestamp)
}

func (suite *GRPCHandlerTestSuite) TestConvertReportTypeToString() {
	testCases := []struct {
		reportType pb.ReportType
		expected   string
	}{
		{pb.ReportType_REPORT_TYPE_USAGE, "usage"},
		{pb.ReportType_REPORT_TYPE_ENGAGEMENT, "engagement"},
		{pb.ReportType_REPORT_TYPE_PERFORMANCE, "performance"},
		{pb.ReportType_REPORT_TYPE_FINANCIAL, "financial"},
		{pb.ReportType_REPORT_TYPE_CUSTOM, "custom"},
		{pb.ReportType_REPORT_TYPE_UNSPECIFIED, "usage"},
	}

	for _, tc := range testCases {
		result := suite.handler.convertReportTypeToString(tc.reportType)
		assert.Equal(suite.T(), tc.expected, result)
	}
}

func TestGRPCHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(GRPCHandlerTestSuite))
}

// Error handling tests
func TestGRPCHandlerErrorHandling(t *testing.T) {
	mockService := new(MockAnalyticsService)
	loggingConfig := &config.LoggingConfig{Level: "info", Format: "console", Output: "stdout"}
	logger := logging.NewLogger(loggingConfig, "analytics-service-test")
	monitor := monitoring.NewMonitor(monitoring.Config{ServiceName: "test"})
	handler := NewGRPCHandler(mockService, logger, monitor)
	ctx := context.Background()

	t.Run("GetMetrics_ServiceError", func(t *testing.T) {
		clubID := "test-club-1"
		timeRange := "24h"

		mockService.On("GetMetrics", clubID, timeRange).Return(map[string]interface{}{}, assert.AnError)

		req := &pb.GetMetricsRequest{
			ClubId:    clubID,
			TimeRange: timeRange,
		}

		resp, err := handler.GetMetrics(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("RecordEvent_ServiceError", func(t *testing.T) {
		mockService.On("RecordEvent", mock.AnythingOfType("map[string]interface {}")).Return(assert.AnError)

		req := &pb.RecordEventRequest{
			ClubId:    "test-club-1",
			EventType: "member_visit",
		}

		resp, err := handler.RecordEvent(ctx, req)
		assert.NoError(t, err) // Handler doesn't return error, but sets success=false
		assert.NotNil(t, resp)
		assert.False(t, resp.Success)
		assert.Contains(t, resp.Message, "Failed to record event")
	})

	mockService.AssertExpectations(t)
}

// Performance benchmark tests
func BenchmarkGRPCHandler_GetMetrics(b *testing.B) {
	mockService := new(MockAnalyticsService)
	loggingConfig := &config.LoggingConfig{Level: "error", Format: "console", Output: "stdout"}
	logger := logging.NewLogger(loggingConfig, "analytics-service-bench")
	monitor := monitoring.NewMonitor(monitoring.Config{ServiceName: "test"})
	handler := NewGRPCHandler(mockService, logger, monitor)
	ctx := context.Background()

	mockMetrics := map[string]interface{}{
		"summary": map[string]interface{}{"total": "100"},
		"details": []interface{}{},
	}

	mockService.On("GetMetrics", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(mockMetrics, nil)

	req := &pb.GetMetricsRequest{
		ClubId:    "test-club-1",
		TimeRange: "24h",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.GetMetrics(ctx, req)
	}
}

func BenchmarkGRPCHandler_RecordEvent(b *testing.B) {
	mockService := new(MockAnalyticsService)
	loggingConfig := &config.LoggingConfig{Level: "error", Format: "console", Output: "stdout"}
	logger := logging.NewLogger(loggingConfig, "analytics-service-bench")
	monitor := monitoring.NewMonitor(monitoring.Config{ServiceName: "test"})
	handler := NewGRPCHandler(mockService, logger, monitor)
	ctx := context.Background()

	mockService.On("RecordEvent", mock.AnythingOfType("map[string]interface {}")).Return(nil)

	req := &pb.RecordEventRequest{
		ClubId:    "test-club-1",
		EventType: "member_visit",
		UserId:    "user-123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.RecordEvent(ctx, req)
	}
}
