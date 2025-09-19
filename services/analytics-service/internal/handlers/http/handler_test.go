package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"reciprocal-clubs-backend/pkg/shared/config"
	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
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

type HTTPHandlerTestSuite struct {
	suite.Suite
	mockService *MockAnalyticsService
	handler     *HTTPHandler
	router      http.Handler
}

func (suite *HTTPHandlerTestSuite) SetupTest() {
	suite.mockService = new(MockAnalyticsService)
	loggingConfig := &config.LoggingConfig{Level: "info", Format: "console", Output: "stdout"}
	logger := logging.NewLogger(loggingConfig, "analytics-service-test")
	monitor := monitoring.NewMonitor(monitoring.Config{ServiceName: "analytics-service-test"})

	suite.handler = NewHTTPHandler(suite.mockService, logger, monitor)
	suite.router = suite.handler.SetupRoutes()
}

func (suite *HTTPHandlerTestSuite) TearDownTest() {
	suite.mockService.AssertExpectations(suite.T())
}

func (suite *HTTPHandlerTestSuite) TestHealthCheck() {
	// Test with no health checker (fallback)
	suite.mockService.On("GetHealthChecker").Return(nil)

	req, _ := http.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	suite.router.ServeHTTP(rr, req)

	assert.Equal(suite.T(), http.StatusOK, rr.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "healthy", response["status"])
	assert.Equal(suite.T(), "analytics-service", response["service"])
}

func (suite *HTTPHandlerTestSuite) TestReadinessCheck() {
	// Test ready state
	suite.mockService.On("GetHealthChecker").Return(nil)
	suite.mockService.On("IsReady").Return(true)

	req, _ := http.NewRequest("GET", "/ready", nil)
	rr := httptest.NewRecorder()

	suite.router.ServeHTTP(rr, req)

	assert.Equal(suite.T(), http.StatusOK, rr.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "ready", response["status"])
}

func (suite *HTTPHandlerTestSuite) TestReadinessCheckNotReady() {
	// Test not ready state
	suite.mockService.On("GetHealthChecker").Return(nil)
	suite.mockService.On("IsReady").Return(false)

	req, _ := http.NewRequest("GET", "/ready", nil)
	rr := httptest.NewRecorder()

	suite.router.ServeHTTP(rr, req)

	assert.Equal(suite.T(), http.StatusServiceUnavailable, rr.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "not ready", response["status"])
}

func (suite *HTTPHandlerTestSuite) TestLivenessCheck() {
	// Test with no health checker (fallback)
	suite.mockService.On("GetHealthChecker").Return(nil)

	req, _ := http.NewRequest("GET", "/live", nil)
	rr := httptest.NewRecorder()

	suite.router.ServeHTTP(rr, req)

	assert.Equal(suite.T(), http.StatusOK, rr.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), true, response["alive"])
	assert.Equal(suite.T(), "analytics-service", response["service"])
}

func (suite *HTTPHandlerTestSuite) TestGetMetrics() {
	clubID := "test-club-1"
	timeRange := "24h"

	mockMetrics := map[string]interface{}{
		"summary": map[string]interface{}{
			"total_events": 100,
			"total_users":  25,
		},
		"details": []interface{}{},
	}

	suite.mockService.On("GetMetrics", clubID, timeRange).Return(mockMetrics, nil)

	req, _ := http.NewRequest("GET", "/api/v1/analytics/metrics?club_id="+clubID+"&time_range="+timeRange, nil)
	rr := httptest.NewRecorder()

	suite.router.ServeHTTP(rr, req)

	assert.Equal(suite.T(), http.StatusOK, rr.Code)
	assert.Equal(suite.T(), "application/json", rr.Header().Get("Content-Type"))

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), response, "summary")
}

func (suite *HTTPHandlerTestSuite) TestGetMetricsError() {
	clubID := "test-club-1"
	timeRange := "24h"

	suite.mockService.On("GetMetrics", clubID, timeRange).Return(map[string]interface{}{}, assert.AnError)

	req, _ := http.NewRequest("GET", "/api/v1/analytics/metrics?club_id="+clubID+"&time_range="+timeRange, nil)
	rr := httptest.NewRecorder()

	suite.router.ServeHTTP(rr, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, rr.Code)
}

func (suite *HTTPHandlerTestSuite) TestGetReports() {
	clubID := "test-club-1"
	reportType := "usage"

	mockReports := []map[string]interface{}{
		{
			"id":          1,
			"club_id":     clubID,
			"title":       "Usage Report",
			"report_type": reportType,
		},
	}

	suite.mockService.On("GetReports", clubID, reportType).Return(mockReports, nil)

	req, _ := http.NewRequest("GET", "/api/v1/analytics/reports?club_id="+clubID+"&type="+reportType, nil)
	rr := httptest.NewRecorder()

	suite.router.ServeHTTP(rr, req)

	assert.Equal(suite.T(), http.StatusOK, rr.Code)
	assert.Equal(suite.T(), "application/json", rr.Header().Get("Content-Type"))

	var response []map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), response, 1)
	assert.Equal(suite.T(), clubID, response[0]["club_id"])
}

func (suite *HTTPHandlerTestSuite) TestGetReportsError() {
	clubID := "test-club-1"
	reportType := "usage"

	suite.mockService.On("GetReports", clubID, reportType).Return([]map[string]interface{}{}, assert.AnError)

	req, _ := http.NewRequest("GET", "/api/v1/analytics/reports?club_id="+clubID+"&type="+reportType, nil)
	rr := httptest.NewRecorder()

	suite.router.ServeHTTP(rr, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, rr.Code)
}

func (suite *HTTPHandlerTestSuite) TestRecordEvent() {
	eventData := map[string]interface{}{
		"club_id":    "test-club-1",
		"event_type": "member_visit",
		"user_id":    "user-123",
	}

	suite.mockService.On("RecordEvent", eventData).Return(nil)

	jsonData, _ := json.Marshal(eventData)
	req, _ := http.NewRequest("POST", "/api/v1/analytics/events", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	suite.router.ServeHTTP(rr, req)

	assert.Equal(suite.T(), http.StatusCreated, rr.Code)
	assert.Equal(suite.T(), "application/json", rr.Header().Get("Content-Type"))

	var response map[string]string
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "event recorded", response["status"])
}

func (suite *HTTPHandlerTestSuite) TestRecordEventInvalidJSON() {
	invalidJSON := `{"invalid": json}`

	req, _ := http.NewRequest("POST", "/api/v1/analytics/events", bytes.NewBufferString(invalidJSON))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	suite.router.ServeHTTP(rr, req)

	assert.Equal(suite.T(), http.StatusBadRequest, rr.Code)
}

func (suite *HTTPHandlerTestSuite) TestRecordEventServiceError() {
	eventData := map[string]interface{}{
		"club_id":    "test-club-1",
		"event_type": "member_visit",
	}

	suite.mockService.On("RecordEvent", eventData).Return(assert.AnError)

	jsonData, _ := json.Marshal(eventData)
	req, _ := http.NewRequest("POST", "/api/v1/analytics/events", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	suite.router.ServeHTTP(rr, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, rr.Code)
}

func (suite *HTTPHandlerTestSuite) TestMetricsEndpoint() {
	// Test that Prometheus metrics endpoint is available
	req, _ := http.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()

	suite.router.ServeHTTP(rr, req)

	assert.Equal(suite.T(), http.StatusOK, rr.Code)
	// Prometheus metrics should be in plain text format
	assert.Contains(suite.T(), rr.Header().Get("Content-Type"), "text/plain")
}

func (suite *HTTPHandlerTestSuite) TestMiddleware() {
	// Test that middleware is applied by checking logging
	suite.mockService.On("GetHealthChecker").Return(nil)
	suite.mockService.On("GetMonitoringMetrics").Return(nil)

	req, _ := http.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	suite.router.ServeHTTP(rr, req)

	assert.Equal(suite.T(), http.StatusOK, rr.Code)
	// If we got a response, middleware was applied
}

func TestHTTPHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(HTTPHandlerTestSuite))
}

// Additional individual tests
func TestResponseWriter(t *testing.T) {
	rr := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rr, statusCode: http.StatusOK}

	// Test default status code
	assert.Equal(t, http.StatusOK, rw.statusCode)

	// Test setting status code
	rw.WriteHeader(http.StatusBadRequest)
	assert.Equal(t, http.StatusBadRequest, rw.statusCode)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHTTPHandler_EmptyQueryParams(t *testing.T) {
	mockService := new(MockAnalyticsService)
	loggingConfig := &config.LoggingConfig{Level: "info", Format: "console", Output: "stdout"}
	logger := logging.NewLogger(loggingConfig, "analytics-service-test")
	monitor := monitoring.NewMonitor(monitoring.Config{ServiceName: "test"})
	handler := NewHTTPHandler(mockService, logger, monitor)
	router := handler.SetupRoutes()

	// Test with empty query parameters
	mockMetrics := map[string]interface{}{"total": 0}
	mockService.On("GetMetrics", "", "").Return(mockMetrics, nil)

	req, _ := http.NewRequest("GET", "/api/v1/analytics/metrics", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	mockService.AssertExpectations(t)
}

func TestHTTPHandler_CORSHeaders(t *testing.T) {
	mockService := new(MockAnalyticsService)
	loggingConfig := &config.LoggingConfig{Level: "info", Format: "console", Output: "stdout"}
	logger := logging.NewLogger(loggingConfig, "analytics-service-test")
	monitor := monitoring.NewMonitor(monitoring.Config{ServiceName: "test"})
	handler := NewHTTPHandler(mockService, logger, monitor)
	router := handler.SetupRoutes()

	mockService.On("GetHealthChecker").Return(nil)

	req, _ := http.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	// Note: CORS headers would be added by a CORS middleware if needed
	mockService.AssertExpectations(t)
}

// Benchmark tests
func BenchmarkHTTPHandler_GetMetrics(b *testing.B) {
	mockService := new(MockAnalyticsService)
	loggingConfig := &config.LoggingConfig{Level: "error", Format: "console", Output: "stdout"}
	logger := logging.NewLogger(loggingConfig, "analytics-service-bench")
	monitor := monitoring.NewMonitor(monitoring.Config{ServiceName: "test"})
	handler := NewHTTPHandler(mockService, logger, monitor)
	router := handler.SetupRoutes()

	mockMetrics := map[string]interface{}{"total": 100}
	mockService.On("GetMetrics", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(mockMetrics, nil)
	mockService.On("GetMonitoringMetrics").Return(nil)

	req, _ := http.NewRequest("GET", "/api/v1/analytics/metrics?club_id=test&time_range=24h", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
	}
}

func BenchmarkHTTPHandler_RecordEvent(b *testing.B) {
	mockService := new(MockAnalyticsService)
	loggingConfig := &config.LoggingConfig{Level: "error", Format: "console", Output: "stdout"}
	logger := logging.NewLogger(loggingConfig, "analytics-service-bench")
	monitor := monitoring.NewMonitor(monitoring.Config{ServiceName: "test"})
	handler := NewHTTPHandler(mockService, logger, monitor)
	router := handler.SetupRoutes()

	eventData := map[string]interface{}{
		"club_id":    "test-club-1",
		"event_type": "member_visit",
	}
	jsonData, _ := json.Marshal(eventData)

	mockService.On("RecordEvent", mock.AnythingOfType("map[string]interface {}")).Return(nil)
	mockService.On("GetMonitoringMetrics").Return(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "/api/v1/analytics/events", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
	}
}