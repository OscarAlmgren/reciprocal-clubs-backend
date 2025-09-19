package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"reciprocal-clubs-backend/pkg/shared/config"
	"reciprocal-clubs-backend/pkg/shared/logging"
)

type RepositoryTestSuite struct {
	suite.Suite
	db   *gorm.DB
	repo Repository
}

func (suite *RepositoryTestSuite) SetupSuite() {
	// Setup in-memory SQLite database for testing
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	suite.Require().NoError(err)

	// Auto-migrate the schema
	err = db.AutoMigrate(
		&AnalyticsEvent{},
		&AnalyticsMetric{},
		&AnalyticsReport{},
		&Dashboard{},
	)
	suite.Require().NoError(err)

	suite.db = db
	loggingConfig := &config.LoggingConfig{Level: "info", Format: "console", Output: "stdout"}
	logger := logging.NewLogger(loggingConfig, "analytics-service-test")
	suite.repo = NewRepository(db, logger)
}

func (suite *RepositoryTestSuite) TearDownTest() {
	// Clean up data after each test
	suite.db.Exec("DELETE FROM analytics_events")
	suite.db.Exec("DELETE FROM analytics_metrics")
	suite.db.Exec("DELETE FROM analytics_reports")
	suite.db.Exec("DELETE FROM analytics_dashboards")
}

func (suite *RepositoryTestSuite) TestIsHealthy() {
	assert.True(suite.T(), suite.repo.IsHealthy())
}

func (suite *RepositoryTestSuite) TestRecordEvent() {
	event := &AnalyticsEvent{
		ClubID:    "test-club-1",
		EventType: "member_visit",
		Data: map[string]interface{}{
			"member_id": "member-123",
			"location":  "gym",
		},
		Timestamp: time.Now(),
	}

	err := suite.repo.RecordEvent(event)
	assert.NoError(suite.T(), err)
	assert.NotZero(suite.T(), event.ID)

	// Verify event was stored
	var storedEvent AnalyticsEvent
	err = suite.db.First(&storedEvent, event.ID).Error
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), event.ClubID, storedEvent.ClubID)
	assert.Equal(suite.T(), event.EventType, storedEvent.EventType)
}

func (suite *RepositoryTestSuite) TestRecordMetric() {
	metric := &AnalyticsMetric{
		ClubID:      "test-club-1",
		MetricName:  "visitor_count",
		MetricValue: 25.0,
		Tags: map[string]interface{}{
			"location": "main_entrance",
		},
		Timestamp: time.Now(),
	}

	err := suite.repo.RecordMetric(metric)
	assert.NoError(suite.T(), err)
	assert.NotZero(suite.T(), metric.ID)

	// Verify metric was stored
	var storedMetric AnalyticsMetric
	err = suite.db.First(&storedMetric, metric.ID).Error
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), metric.ClubID, storedMetric.ClubID)
	assert.Equal(suite.T(), metric.MetricName, storedMetric.MetricName)
	assert.Equal(suite.T(), metric.MetricValue, storedMetric.MetricValue)
}

func (suite *RepositoryTestSuite) TestGetMetricsByClub() {
	clubID := "test-club-1"
	now := time.Now()
	timeRange := TimeRange{
		Start: now.Add(-1 * time.Hour),
		End:   now,
	}

	// Create test metrics
	metrics := []*AnalyticsMetric{
		{
			ClubID:      clubID,
			MetricName:  "visitor_count",
			MetricValue: 10.0,
			Timestamp:   now.Add(-30 * time.Minute),
		},
		{
			ClubID:      clubID,
			MetricName:  "visitor_count",
			MetricValue: 15.0,
			Timestamp:   now.Add(-15 * time.Minute),
		},
		{
			ClubID:      "other-club",
			MetricName:  "visitor_count",
			MetricValue: 20.0,
			Timestamp:   now.Add(-20 * time.Minute),
		},
	}

	for _, metric := range metrics {
		err := suite.repo.RecordMetric(metric)
		assert.NoError(suite.T(), err)
	}

	// Test retrieval
	result, err := suite.repo.GetMetricsByClub(clubID, timeRange)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), result, 2) // Should only get metrics for test-club-1
}

func (suite *RepositoryTestSuite) TestGetEventsByClub() {
	clubID := "test-club-1"
	now := time.Now()
	timeRange := TimeRange{
		Start: now.Add(-1 * time.Hour),
		End:   now,
	}

	// Create test events
	events := []*AnalyticsEvent{
		{
			ClubID:    clubID,
			EventType: "member_visit",
			Data:      map[string]interface{}{"member_id": "123"},
			Timestamp: now.Add(-30 * time.Minute),
		},
		{
			ClubID:    clubID,
			EventType: "facility_usage",
			Data:      map[string]interface{}{"facility": "gym"},
			Timestamp: now.Add(-15 * time.Minute),
		},
		{
			ClubID:    "other-club",
			EventType: "member_visit",
			Data:      map[string]interface{}{"member_id": "456"},
			Timestamp: now.Add(-20 * time.Minute),
		},
	}

	for _, event := range events {
		err := suite.repo.RecordEvent(event)
		assert.NoError(suite.T(), err)
	}

	// Test retrieval
	result, err := suite.repo.GetEventsByClub(clubID, timeRange)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), result, 2) // Should only get events for test-club-1
}

func (suite *RepositoryTestSuite) TestCreateReport() {
	report := &AnalyticsReport{
		ClubID:     "test-club-1",
		ReportType: "usage",
		Title:      "Test Usage Report",
		Data: map[string]interface{}{
			"total_visits": 100,
			"unique_users": 75,
		},
		GeneratedAt: time.Now(),
	}

	err := suite.repo.CreateReport(report)
	assert.NoError(suite.T(), err)
	assert.NotZero(suite.T(), report.ID)

	// Verify report was stored
	var storedReport AnalyticsReport
	err = suite.db.First(&storedReport, report.ID).Error
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), report.ClubID, storedReport.ClubID)
	assert.Equal(suite.T(), report.ReportType, storedReport.ReportType)
	assert.Equal(suite.T(), report.Title, storedReport.Title)
}

func (suite *RepositoryTestSuite) TestGetReportsByClub() {
	clubID := "test-club-1"

	// Create test reports
	reports := []*AnalyticsReport{
		{
			ClubID:      clubID,
			ReportType:  "usage",
			Title:       "Usage Report 1",
			Data:        map[string]interface{}{"total": 100},
			GeneratedAt: time.Now().Add(-2 * time.Hour),
		},
		{
			ClubID:      clubID,
			ReportType:  "engagement",
			Title:       "Engagement Report 1",
			Data:        map[string]interface{}{"score": 85},
			GeneratedAt: time.Now().Add(-1 * time.Hour),
		},
		{
			ClubID:      "other-club",
			ReportType:  "usage",
			Title:       "Other Club Report",
			Data:        map[string]interface{}{"total": 50},
			GeneratedAt: time.Now().Add(-1 * time.Hour),
		},
	}

	for _, report := range reports {
		err := suite.repo.CreateReport(report)
		assert.NoError(suite.T(), err)
	}

	// Test retrieval - all reports for club
	result, err := suite.repo.GetReportsByClub(clubID, "")
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), result, 2)

	// Test retrieval - specific report type
	result, err = suite.repo.GetReportsByClub(clubID, "usage")
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), result, 1)
	assert.Equal(suite.T(), "usage", result[0].ReportType)
}

func (suite *RepositoryTestSuite) TestDashboardOperations() {
	dashboard := &Dashboard{
		ClubID:      "test-club-1",
		Name:        "Test Dashboard",
		Description: "A test dashboard",
		Panels: map[string]interface{}{
			"panel1": map[string]interface{}{
				"title": "Visitor Count",
				"type":  "graph",
			},
		},
		IsPublic:  true,
		CreatedBy: "admin@test.com",
	}

	// Test Create
	err := suite.repo.CreateDashboard(dashboard)
	assert.NoError(suite.T(), err)
	assert.NotZero(suite.T(), dashboard.ID)

	// Test Get
	retrieved, err := suite.repo.GetDashboard(dashboard.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), dashboard.Name, retrieved.Name)
	assert.Equal(suite.T(), dashboard.ClubID, retrieved.ClubID)

	// Test Update
	dashboard.Name = "Updated Dashboard"
	err = suite.repo.UpdateDashboard(dashboard)
	assert.NoError(suite.T(), err)

	updated, err := suite.repo.GetDashboard(dashboard.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Updated Dashboard", updated.Name)

	// Test List
	dashboards, err := suite.repo.ListDashboards("test-club-1", 10, 0)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), dashboards, 1)

	// Test Delete
	err = suite.repo.DeleteDashboard(dashboard.ID)
	assert.NoError(suite.T(), err)

	_, err = suite.repo.GetDashboard(dashboard.ID)
	assert.Error(suite.T(), err)
}

func (suite *RepositoryTestSuite) TestAggregateMetrics() {
	clubID := "test-club-1"
	now := time.Now()
	timeRange := TimeRange{
		Start: now.Add(-1 * time.Hour),
		End:   now,
	}

	// Create test events
	events := []*AnalyticsEvent{
		{
			ClubID:    clubID,
			EventType: "member_visit",
			Data:      map[string]interface{}{"member_id": "123"},
			Timestamp: now.Add(-30 * time.Minute),
		},
		{
			ClubID:    clubID,
			EventType: "facility_usage",
			Data:      map[string]interface{}{"facility": "gym"},
			Timestamp: now.Add(-15 * time.Minute),
		},
		{
			ClubID:    clubID,
			EventType: "member_visit",
			Data:      map[string]interface{}{"member_id": "456"},
			Timestamp: now.Add(-10 * time.Minute),
		},
	}

	for _, event := range events {
		err := suite.repo.RecordEvent(event)
		assert.NoError(suite.T(), err)
	}

	// Test aggregation
	result, err := suite.repo.AggregateMetrics(clubID, timeRange)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), result, "total_events")
	assert.Contains(suite.T(), result, "unique_event_types")
	assert.Equal(suite.T(), int64(3), result["total_events"])
	assert.Equal(suite.T(), int64(2), result["unique_event_types"])
}

func (suite *RepositoryTestSuite) TestGetRealtimeMetrics() {
	clubID := "test-club-1"
	now := time.Now()

	// Create recent events and metrics
	recentEvent := &AnalyticsEvent{
		ClubID:    clubID,
		EventType: "member_visit",
		Data:      map[string]interface{}{"member_id": "123"},
		Timestamp: now.Add(-2 * time.Minute),
	}

	recentMetric := &AnalyticsMetric{
		ClubID:      clubID,
		MetricName:  "visitor_count",
		MetricValue: 10.0,
		Timestamp:   now.Add(-1 * time.Minute),
	}

	err := suite.repo.RecordEvent(recentEvent)
	assert.NoError(suite.T(), err)

	err = suite.repo.RecordMetric(recentMetric)
	assert.NoError(suite.T(), err)

	// Test realtime metrics
	result, err := suite.repo.GetRealtimeMetrics(clubID)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), result, "recent_events")
	assert.Contains(suite.T(), result, "recent_metrics")
	assert.Contains(suite.T(), result, "timestamp")
}

func (suite *RepositoryTestSuite) TestCleanupOldEvents() {
	clubID := "test-club-1"
	now := time.Now()

	// Create old and new events
	oldEvent := &AnalyticsEvent{
		ClubID:    clubID,
		EventType: "member_visit",
		Data:      map[string]interface{}{"member_id": "123"},
		Timestamp: now.Add(-48 * time.Hour), // 2 days old
	}

	newEvent := &AnalyticsEvent{
		ClubID:    clubID,
		EventType: "member_visit",
		Data:      map[string]interface{}{"member_id": "456"},
		Timestamp: now.Add(-12 * time.Hour), // 12 hours old
	}

	err := suite.repo.RecordEvent(oldEvent)
	assert.NoError(suite.T(), err)

	err = suite.repo.RecordEvent(newEvent)
	assert.NoError(suite.T(), err)

	// Cleanup events older than 24 hours
	cutoffTime := now.Add(-24 * time.Hour)
	err = suite.repo.CleanupOldEvents(cutoffTime)
	assert.NoError(suite.T(), err)

	// Verify old event was deleted, new event remains
	var count int64
	suite.db.Model(&AnalyticsEvent{}).Count(&count)
	assert.Equal(suite.T(), int64(1), count)

	var remainingEvent AnalyticsEvent
	err = suite.db.First(&remainingEvent).Error
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), newEvent.ID, remainingEvent.ID)
}

func (suite *RepositoryTestSuite) TestExportOperations() {
	clubID := "test-club-1"
	now := time.Now()
	timeRange := TimeRange{
		Start: now.Add(-1 * time.Hour),
		End:   now,
	}

	// Create test data
	event := &AnalyticsEvent{
		ClubID:    clubID,
		EventType: "member_visit",
		Data:      map[string]interface{}{"member_id": "123"},
		Timestamp: now.Add(-30 * time.Minute),
	}

	metric := &AnalyticsMetric{
		ClubID:      clubID,
		MetricName:  "visitor_count",
		MetricValue: 10.0,
		Timestamp:   now.Add(-30 * time.Minute),
	}

	report := &AnalyticsReport{
		ClubID:      clubID,
		ReportType:  "usage",
		Title:       "Test Report",
		Data:        map[string]interface{}{"total": 100},
		GeneratedAt: now,
	}

	err := suite.repo.RecordEvent(event)
	assert.NoError(suite.T(), err)

	err = suite.repo.RecordMetric(metric)
	assert.NoError(suite.T(), err)

	err = suite.repo.CreateReport(report)
	assert.NoError(suite.T(), err)

	// Test JSON exports
	eventsJSON, err := suite.repo.ExportEvents(clubID, timeRange, "json")
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), eventsJSON)

	metricsJSON, err := suite.repo.ExportMetrics(clubID, timeRange, "json")
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), metricsJSON)

	reportsJSON, err := suite.repo.ExportReports(clubID, "json")
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), reportsJSON)

	// Test CSV exports
	eventsCSV, err := suite.repo.ExportEvents(clubID, timeRange, "csv")
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), string(eventsCSV), "id,club_id,event_type")

	metricsCSV, err := suite.repo.ExportMetrics(clubID, timeRange, "csv")
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), string(metricsCSV), "id,club_id,metric_name")

	reportsCSV, err := suite.repo.ExportReports(clubID, "csv")
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), string(reportsCSV), "id,club_id,report_type")

	// Test unsupported format
	_, err = suite.repo.ExportEvents(clubID, timeRange, "xml")
	assert.Error(suite.T(), err)
}

func TestRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(RepositoryTestSuite))
}

// Benchmark tests
func BenchmarkRecordEvent(b *testing.B) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&AnalyticsEvent{})

	loggingConfig := &config.LoggingConfig{Level: "error", Format: "console", Output: "stdout"}
	logger := logging.NewLogger(loggingConfig, "analytics-service-bench")
	repo := NewRepository(db, logger)

	event := &AnalyticsEvent{
		ClubID:    "test-club-1",
		EventType: "member_visit",
		Data:      map[string]interface{}{"member_id": "123"},
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		event.ID = 0 // Reset ID for each iteration
		repo.RecordEvent(event)
	}
}

func BenchmarkRecordMetric(b *testing.B) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&AnalyticsMetric{})

	loggingConfig := &config.LoggingConfig{Level: "error", Format: "console", Output: "stdout"}
	logger := logging.NewLogger(loggingConfig, "analytics-service-bench")
	repo := NewRepository(db, logger)

	metric := &AnalyticsMetric{
		ClubID:      "test-club-1",
		MetricName:  "visitor_count",
		MetricValue: 10.0,
		Timestamp:   time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metric.ID = 0 // Reset ID for each iteration
		repo.RecordMetric(metric)
	}
}