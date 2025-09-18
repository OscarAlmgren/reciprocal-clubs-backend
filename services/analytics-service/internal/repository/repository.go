package repository

import (
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
	Data      map[string]interface{} `json:"data" gorm:"type:jsonb"`
	Timestamp time.Time              `json:"timestamp" gorm:"index"`
	CreatedAt time.Time              `json:"created_at"`
}

type AnalyticsMetric struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	ClubID      string    `json:"club_id" gorm:"index;size:255"`
	MetricName  string    `json:"metric_name" gorm:"size:100"`
	MetricValue float64   `json:"metric_value"`
	Tags        map[string]interface{} `json:"tags" gorm:"type:jsonb"`
	Timestamp   time.Time `json:"timestamp" gorm:"index"`
	CreatedAt   time.Time `json:"created_at"`
}

type AnalyticsReport struct {
	ID         uint                   `json:"id" gorm:"primaryKey"`
	ClubID     string                 `json:"club_id" gorm:"index;size:255"`
	ReportType string                 `json:"report_type" gorm:"size:100"`
	Title      string                 `json:"title" gorm:"size:255"`
	Data       map[string]interface{} `json:"data" gorm:"type:jsonb"`
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
