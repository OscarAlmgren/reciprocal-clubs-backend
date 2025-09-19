package repository

import (
	"encoding/json"
	"fmt"
	"time"

	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/services/analytics-service/internal/models"

	"gorm.io/gorm"
)

type Repository interface {
	// Health check
	IsHealthy() bool

	// Analytics operations
	RecordEvent(event *AnalyticsEvent) error
	GetMetricsByClub(clubID string, timeRange TimeRange) ([]*AnalyticsMetric, error)
	GetReportsByClub(clubID string, reportType string) ([]*AnalyticsReport, error)
	AggregateMetrics(clubID string, timeRange TimeRange) (map[string]interface{}, error)
	CreateReport(report *AnalyticsReport) error
	RecordMetric(metric *AnalyticsMetric) error
	GetEventsByClub(clubID string, timeRange TimeRange) ([]*AnalyticsEvent, error)
	GetRealtimeMetrics(clubID string) (map[string]interface{}, error)
	CleanupOldEvents(olderThan time.Time) error

	// Advanced analytics
	GetTrendAnalysis(clubID string, metricName string, timeRange TimeRange) (map[string]interface{}, error)
	GetCorrelationAnalysis(clubID string, metricNames []string, timeRange TimeRange) (map[string]interface{}, error)
	GetPredictiveAnalytics(clubID string, metricName string, forecastDays int) (map[string]interface{}, error)
	GetAnomalyDetection(clubID string, metricName string, timeRange TimeRange) (map[string]interface{}, error)

	// Dashboard operations
	CreateDashboard(dashboard *Dashboard) error
	GetDashboard(dashboardID uint) (*Dashboard, error)
	UpdateDashboard(dashboard *Dashboard) error
	DeleteDashboard(dashboardID uint) error
	ListDashboards(clubID string, limit, offset int) ([]*Dashboard, error)

	// Export operations
	ExportEvents(clubID string, timeRange TimeRange, format string) ([]byte, error)
	ExportMetrics(clubID string, timeRange TimeRange, format string) ([]byte, error)
	ExportReports(clubID string, format string) ([]byte, error)

	// Example operations (replace with actual models)
	CreateExample(example *models.Example) error
	GetExampleByID(id uint) (*models.Example, error)
	UpdateExample(example *models.Example) error
	DeleteExample(id uint) error
	ListExamples(limit, offset int) ([]*models.Example, error)
}

type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type AnalyticsEvent struct {
	ID        uint                   `json:"id" gorm:"primaryKey"`
	ClubID    string                 `json:"club_id" gorm:"index;size:255"`
	EventType string                 `json:"event_type" gorm:"size:100"`
	Data      map[string]interface{} `json:"data" gorm:"serializer:json"`
	Timestamp time.Time              `json:"timestamp" gorm:"index"`
	CreatedAt time.Time              `json:"created_at"`
}

type AnalyticsMetric struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	ClubID      string    `json:"club_id" gorm:"index;size:255"`
	MetricName  string    `json:"metric_name" gorm:"size:100"`
	MetricValue float64   `json:"metric_value"`
	Tags        map[string]interface{} `json:"tags" gorm:"serializer:json"`
	Timestamp   time.Time `json:"timestamp" gorm:"index"`
	CreatedAt   time.Time `json:"created_at"`
}

type AnalyticsReport struct {
	ID         uint                   `json:"id" gorm:"primaryKey"`
	ClubID     string                 `json:"club_id" gorm:"index;size:255"`
	ReportType string                 `json:"report_type" gorm:"size:100"`
	Title      string                 `json:"title" gorm:"size:255"`
	Data       map[string]interface{} `json:"data" gorm:"serializer:json"`
	GeneratedAt time.Time             `json:"generated_at"`
	CreatedAt   time.Time             `json:"created_at"`
}

func (AnalyticsEvent) TableName() string {
	return "analytics_events"
}

func (AnalyticsMetric) TableName() string {
	return "analytics_metrics"
}

func (AnalyticsReport) TableName() string {
	return "analytics_reports"
}

type Dashboard struct {
	ID          uint                   `json:"id" gorm:"primaryKey"`
	ClubID      string                 `json:"club_id" gorm:"index;size:255"`
	Name        string                 `json:"name" gorm:"size:255"`
	Description string                 `json:"description" gorm:"type:text"`
	Panels      map[string]interface{} `json:"panels" gorm:"serializer:json"`
	IsPublic    bool                   `json:"is_public" gorm:"default:false"`
	CreatedBy   string                 `json:"created_by" gorm:"size:255"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

func (Dashboard) TableName() string {
	return "analytics_dashboards"
}

type repository struct {
	db     *gorm.DB
	logger logging.Logger
}

func NewRepository(db *gorm.DB, logger logging.Logger) Repository {
	return &repository{
		db:     db,
		logger: logger,
	}
}

// GetDB returns the underlying GORM database connection
func (r *repository) GetDB() *gorm.DB {
	return r.db
}

func (r *repository) IsHealthy() bool {
	sqlDB, err := r.db.DB()
	if err != nil {
		r.logger.Error("Failed to get database connection", map[string]interface{}{"error": err.Error()})
		return false
	}

	return sqlDB.Ping() == nil
}

func (r *repository) RecordEvent(event *AnalyticsEvent) error {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	if err := r.db.Create(event).Error; err != nil {
		r.logger.Error("Failed to record analytics event", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("failed to record event: %w", err)
	}

	r.logger.Info("Recorded analytics event", map[string]interface{}{"event_type": event.EventType, "club_id": event.ClubID})
	return nil
}

func (r *repository) GetMetricsByClub(clubID string, timeRange TimeRange) ([]*AnalyticsMetric, error) {
	var metrics []*AnalyticsMetric

	query := r.db.Where("club_id = ?", clubID)
	if !timeRange.Start.IsZero() {
		query = query.Where("timestamp >= ?", timeRange.Start)
	}
	if !timeRange.End.IsZero() {
		query = query.Where("timestamp <= ?", timeRange.End)
	}

	if err := query.Find(&metrics).Error; err != nil {
		r.logger.Error("Failed to get metrics", map[string]interface{}{"error": err.Error()})
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	return metrics, nil
}

func (r *repository) GetReportsByClub(clubID string, reportType string) ([]*AnalyticsReport, error) {
	var reports []*AnalyticsReport

	query := r.db.Where("club_id = ?", clubID)
	if reportType != "" {
		query = query.Where("report_type = ?", reportType)
	}

	if err := query.Order("generated_at DESC").Find(&reports).Error; err != nil {
		r.logger.Error("Failed to get reports", map[string]interface{}{"error": err.Error()})
		return nil, fmt.Errorf("failed to get reports: %w", err)
	}

	return reports, nil
}

func (r *repository) AggregateMetrics(clubID string, timeRange TimeRange) (map[string]interface{}, error) {
	// This would contain complex aggregation queries
	// For now, returning a simple aggregation example

	var totalEvents int64
	var uniqueEventTypes int64

	eventQuery := r.db.Model(&AnalyticsEvent{}).Where("club_id = ?", clubID)
	if !timeRange.Start.IsZero() {
		eventQuery = eventQuery.Where("timestamp >= ?", timeRange.Start)
	}
	if !timeRange.End.IsZero() {
		eventQuery = eventQuery.Where("timestamp <= ?", timeRange.End)
	}

	if err := eventQuery.Count(&totalEvents).Error; err != nil {
		return nil, fmt.Errorf("failed to count events: %w", err)
	}

	if err := eventQuery.Distinct("event_type").Count(&uniqueEventTypes).Error; err != nil {
		return nil, fmt.Errorf("failed to count unique event types: %w", err)
	}

	aggregation := map[string]interface{}{
		"total_events":        totalEvents,
		"unique_event_types":  uniqueEventTypes,
		"time_range":          timeRange,
		"generated_at":        time.Now(),
	}

	return aggregation, nil
}

func (r *repository) CreateReport(report *AnalyticsReport) error {
	if err := r.db.Create(report).Error; err != nil {
		r.logger.Error("Failed to create analytics report", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("failed to create report: %w", err)
	}

	r.logger.Info("Created analytics report", map[string]interface{}{"report_type": report.ReportType, "club_id": report.ClubID})
	return nil
}

func (r *repository) RecordMetric(metric *AnalyticsMetric) error {
	if metric.Timestamp.IsZero() {
		metric.Timestamp = time.Now()
	}

	if err := r.db.Create(metric).Error; err != nil {
		r.logger.Error("Failed to record analytics metric", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("failed to record metric: %w", err)
	}

	r.logger.Info("Recorded analytics metric", map[string]interface{}{"metric_name": metric.MetricName, "club_id": metric.ClubID})
	return nil
}

func (r *repository) GetEventsByClub(clubID string, timeRange TimeRange) ([]*AnalyticsEvent, error) {
	var events []*AnalyticsEvent

	query := r.db.Where("club_id = ?", clubID)
	if !timeRange.Start.IsZero() {
		query = query.Where("timestamp >= ?", timeRange.Start)
	}
	if !timeRange.End.IsZero() {
		query = query.Where("timestamp <= ?", timeRange.End)
	}

	if err := query.Order("timestamp DESC").Find(&events).Error; err != nil {
		r.logger.Error("Failed to get events", map[string]interface{}{"error": err.Error()})
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	return events, nil
}

func (r *repository) GetRealtimeMetrics(clubID string) (map[string]interface{}, error) {
	// Get metrics from the last 5 minutes
	fiveMinutesAgo := time.Now().Add(-5 * time.Minute)

	var recentEvents int64
	var recentMetrics int64

	// Count recent events
	if err := r.db.Model(&AnalyticsEvent{}).
		Where("club_id = ? AND timestamp >= ?", clubID, fiveMinutesAgo).
		Count(&recentEvents).Error; err != nil {
		return nil, fmt.Errorf("failed to count recent events: %w", err)
	}

	// Count recent metrics
	if err := r.db.Model(&AnalyticsMetric{}).
		Where("club_id = ? AND timestamp >= ?", clubID, fiveMinutesAgo).
		Count(&recentMetrics).Error; err != nil {
		return nil, fmt.Errorf("failed to count recent metrics: %w", err)
	}

	// Get average metric values for common metrics
	var avgMetrics []struct {
		MetricName string  `json:"metric_name"`
		AvgValue   float64 `json:"avg_value"`
	}

	if err := r.db.Model(&AnalyticsMetric{}).
		Select("metric_name, AVG(metric_value) as avg_value").
		Where("club_id = ? AND timestamp >= ?", clubID, fiveMinutesAgo).
		Group("metric_name").
		Scan(&avgMetrics).Error; err != nil {
		return nil, fmt.Errorf("failed to get average metrics: %w", err)
	}

	metrics := map[string]interface{}{
		"recent_events":     recentEvents,
		"recent_metrics":    recentMetrics,
		"average_metrics":   avgMetrics,
		"timestamp":         time.Now(),
		"time_window":       "5 minutes",
	}

	return metrics, nil
}

func (r *repository) CleanupOldEvents(olderThan time.Time) error {
	// Delete events older than the specified time
	result := r.db.Where("timestamp < ?", olderThan).Delete(&AnalyticsEvent{})
	if result.Error != nil {
		r.logger.Error("Failed to cleanup old events", map[string]interface{}{"error": result.Error.Error()})
		return fmt.Errorf("failed to cleanup old events: %w", result.Error)
	}

	if result.RowsAffected > 0 {
		r.logger.Info("Cleaned up old events", map[string]interface{}{"rows_deleted": result.RowsAffected, "older_than": olderThan})
	}

	// Also cleanup old metrics
	result = r.db.Where("timestamp < ?", olderThan).Delete(&AnalyticsMetric{})
	if result.Error != nil {
		r.logger.Error("Failed to cleanup old metrics", map[string]interface{}{"error": result.Error.Error()})
		return fmt.Errorf("failed to cleanup old metrics: %w", result.Error)
	}

	if result.RowsAffected > 0 {
		r.logger.Info("Cleaned up old metrics", map[string]interface{}{"rows_deleted": result.RowsAffected, "older_than": olderThan})
	}

	return nil
}

// Advanced analytics implementations
func (r *repository) GetTrendAnalysis(clubID string, metricName string, timeRange TimeRange) (map[string]interface{}, error) {
	// Mock implementation - in production would use statistical analysis
	var dataPoints []struct {
		Timestamp time.Time `json:"timestamp"`
		Value     float64   `json:"value"`
	}

	// Get metric values over time
	if err := r.db.Model(&AnalyticsMetric{}).
		Select("timestamp, AVG(metric_value) as value").
		Where("club_id = ? AND metric_name = ? AND timestamp BETWEEN ? AND ?", clubID, metricName, timeRange.Start, timeRange.End).
		Group("DATE_TRUNC('hour', timestamp)").
		Order("timestamp ASC").
		Scan(&dataPoints).Error; err != nil {
		return nil, fmt.Errorf("failed to get trend data: %w", err)
	}

	// Calculate simple trend
	direction := "stable"
	slope := 0.0
	if len(dataPoints) > 1 {
		firstValue := dataPoints[0].Value
		lastValue := dataPoints[len(dataPoints)-1].Value
		if lastValue > firstValue*1.1 {
			direction = "increasing"
			slope = (lastValue - firstValue) / float64(len(dataPoints))
		} else if lastValue < firstValue*0.9 {
			direction = "decreasing"
			slope = (lastValue - firstValue) / float64(len(dataPoints))
		}
	}

	return map[string]interface{}{
		"data_points": dataPoints,
		"summary": map[string]interface{}{
			"direction":      direction,
			"slope":          slope,
			"confidence":     0.85,
			"interpretation": fmt.Sprintf("Metric %s is %s", metricName, direction),
		},
	}, nil
}

func (r *repository) GetCorrelationAnalysis(clubID string, metricNames []string, timeRange TimeRange) (map[string]interface{}, error) {
	// Mock implementation - in production would calculate actual correlations
	correlations := make(map[string]float64)
	significantPairs := []map[string]interface{}{}

	for i, metric1 := range metricNames {
		for j, metric2 := range metricNames {
			if i < j {
				key := fmt.Sprintf("%s-%s", metric1, metric2)
				correlation := 0.5 + (float64(i+j)*0.1) // Mock correlation
				correlations[key] = correlation

				if correlation > 0.7 {
					significantPairs = append(significantPairs, map[string]interface{}{
						"metric1":     metric1,
						"metric2":     metric2,
						"correlation": correlation,
						"significance": 0.95,
					})
				}
			}
		}
	}

	return map[string]interface{}{
		"correlations":     correlations,
		"significant_pairs": significantPairs,
	}, nil
}

func (r *repository) GetPredictiveAnalytics(clubID string, metricName string, forecastDays int) (map[string]interface{}, error) {
	// Mock implementation - in production would use ML models
	predictions := []map[string]interface{}{}
	baseValue := 100.0

	for i := 0; i < forecastDays; i++ {
		timestamp := time.Now().AddDate(0, 0, i+1)
		predictedValue := baseValue + float64(i)*2.5
		upperBound := predictedValue * 1.2
		lowerBound := predictedValue * 0.8

		predictions = append(predictions, map[string]interface{}{
			"timestamp":        timestamp,
			"predicted_value":  predictedValue,
			"confidence_upper": upperBound,
			"confidence_lower": lowerBound,
		})
	}

	return map[string]interface{}{
		"predictions": predictions,
		"summary": map[string]interface{}{
			"model_type":       "linear_trend",
			"accuracy":         0.82,
			"confidence_level": "95%",
		},
	}, nil
}

func (r *repository) GetAnomalyDetection(clubID string, metricName string, timeRange TimeRange) (map[string]interface{}, error) {
	// Mock implementation - in production would use statistical anomaly detection
	anomalies := []map[string]interface{}{
		{
			"timestamp":      time.Now().Add(-2 * time.Hour),
			"value":          150.0,
			"expected_value": 100.0,
			"anomaly_score":  0.85,
			"severity":       "medium",
		},
		{
			"timestamp":      time.Now().Add(-6 * time.Hour),
			"value":          200.0,
			"expected_value": 105.0,
			"anomaly_score":  0.95,
			"severity":       "high",
		},
	}

	summary := map[string]interface{}{
		"total_anomalies":  len(anomalies),
		"high_severity":    1,
		"medium_severity":  1,
		"low_severity":     0,
	}

	return map[string]interface{}{
		"anomalies": anomalies,
		"summary":   summary,
	}, nil
}

// Dashboard operations
func (r *repository) CreateDashboard(dashboard *Dashboard) error {
	if err := r.db.Create(dashboard).Error; err != nil {
		r.logger.Error("Failed to create dashboard", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("failed to create dashboard: %w", err)
	}

	r.logger.Info("Created dashboard", map[string]interface{}{"dashboard_id": dashboard.ID, "club_id": dashboard.ClubID})
	return nil
}

func (r *repository) GetDashboard(dashboardID uint) (*Dashboard, error) {
	var dashboard Dashboard
	if err := r.db.First(&dashboard, dashboardID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("dashboard not found")
		}
		r.logger.Error("Failed to get dashboard", map[string]interface{}{"error": err.Error()})
		return nil, fmt.Errorf("failed to get dashboard: %w", err)
	}
	return &dashboard, nil
}

func (r *repository) UpdateDashboard(dashboard *Dashboard) error {
	if err := r.db.Save(dashboard).Error; err != nil {
		r.logger.Error("Failed to update dashboard", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("failed to update dashboard: %w", err)
	}

	r.logger.Info("Updated dashboard", map[string]interface{}{"dashboard_id": dashboard.ID})
	return nil
}

func (r *repository) DeleteDashboard(dashboardID uint) error {
	if err := r.db.Delete(&Dashboard{}, dashboardID).Error; err != nil {
		r.logger.Error("Failed to delete dashboard", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("failed to delete dashboard: %w", err)
	}

	r.logger.Info("Deleted dashboard", map[string]interface{}{"dashboard_id": dashboardID})
	return nil
}

func (r *repository) ListDashboards(clubID string, limit, offset int) ([]*Dashboard, error) {
	var dashboards []*Dashboard
	query := r.db.Where("club_id = ? OR is_public = ?", clubID, true)

	if err := query.Limit(limit).Offset(offset).Order("created_at DESC").Find(&dashboards).Error; err != nil {
		r.logger.Error("Failed to list dashboards", map[string]interface{}{"error": err.Error()})
		return nil, fmt.Errorf("failed to list dashboards: %w", err)
	}

	return dashboards, nil
}

// Export operations
func (r *repository) ExportEvents(clubID string, timeRange TimeRange, format string) ([]byte, error) {
	events, err := r.GetEventsByClub(clubID, timeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get events for export: %w", err)
	}

	switch format {
	case "json":
		return json.Marshal(events)
	case "csv":
		// Mock CSV export - in production would use proper CSV library
		csv := "id,club_id,event_type,timestamp,created_at\n"
		for _, event := range events {
			csv += fmt.Sprintf("%d,%s,%s,%s,%s\n",
				event.ID, event.ClubID, event.EventType,
				event.Timestamp.Format(time.RFC3339),
				event.CreatedAt.Format(time.RFC3339))
		}
		return []byte(csv), nil
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

func (r *repository) ExportMetrics(clubID string, timeRange TimeRange, format string) ([]byte, error) {
	metrics, err := r.GetMetricsByClub(clubID, timeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics for export: %w", err)
	}

	switch format {
	case "json":
		return json.Marshal(metrics)
	case "csv":
		// Mock CSV export
		csv := "id,club_id,metric_name,metric_value,timestamp,created_at\n"
		for _, metric := range metrics {
			csv += fmt.Sprintf("%d,%s,%s,%.2f,%s,%s\n",
				metric.ID, metric.ClubID, metric.MetricName, metric.MetricValue,
				metric.Timestamp.Format(time.RFC3339),
				metric.CreatedAt.Format(time.RFC3339))
		}
		return []byte(csv), nil
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

func (r *repository) ExportReports(clubID string, format string) ([]byte, error) {
	reports, err := r.GetReportsByClub(clubID, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get reports for export: %w", err)
	}

	switch format {
	case "json":
		return json.Marshal(reports)
	case "csv":
		// Mock CSV export
		csv := "id,club_id,report_type,title,generated_at,created_at\n"
		for _, report := range reports {
			csv += fmt.Sprintf("%d,%s,%s,%s,%s,%s\n",
				report.ID, report.ClubID, report.ReportType, report.Title,
				report.GeneratedAt.Format(time.RFC3339),
				report.CreatedAt.Format(time.RFC3339))
		}
		return []byte(csv), nil
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// Example operations (replace with actual business logic)
func (r *repository) CreateExample(example *models.Example) error {
	if err := r.db.Create(example).Error; err != nil {
		r.logger.Error("Failed to create example", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("failed to create example: %w", err)
	}
	return nil
}

func (r *repository) GetExampleByID(id uint) (*models.Example, error) {
	var example models.Example
	if err := r.db.First(&example, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("example not found")
		}
		r.logger.Error("Failed to get example", map[string]interface{}{"error": err.Error()})
		return nil, fmt.Errorf("failed to get example: %w", err)
	}
	return &example, nil
}

func (r *repository) UpdateExample(example *models.Example) error {
	if err := r.db.Save(example).Error; err != nil {
		r.logger.Error("Failed to update example", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("failed to update example: %w", err)
	}
	return nil
}

func (r *repository) DeleteExample(id uint) error {
	if err := r.db.Delete(&models.Example{}, id).Error; err != nil {
		r.logger.Error("Failed to delete example", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("failed to delete example: %w", err)
	}
	return nil
}

func (r *repository) ListExamples(limit, offset int) ([]*models.Example, error) {
	var examples []*models.Example
	if err := r.db.Limit(limit).Offset(offset).Find(&examples).Error; err != nil {
		r.logger.Error("Failed to list examples", map[string]interface{}{"error": err.Error()})
		return nil, fmt.Errorf("failed to list examples: %w", err)
	}
	return examples, nil
}
