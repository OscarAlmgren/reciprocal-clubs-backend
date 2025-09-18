package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/messaging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	"reciprocal-clubs-backend/services/analytics-service/internal/integrations"
	analyticsmonitoring "reciprocal-clubs-backend/services/analytics-service/internal/monitoring"
	"reciprocal-clubs-backend/services/analytics-service/internal/repository"
)

type AnalyticsService interface {
	// Health check
	IsReady() bool

	// Analytics operations
	GetMetrics(clubID string, timeRange string) (map[string]interface{}, error)
	GetReports(clubID string, reportType string) ([]map[string]interface{}, error)
	RecordEvent(eventData map[string]interface{}) error
	GenerateReport(clubID string, reportType string) (map[string]interface{}, error)
	GetRealtimeMetrics(clubID string) (map[string]interface{}, error)
	RecordMetric(clubID string, metricName string, value float64, tags map[string]interface{}) error
	GetEvents(clubID string, timeRange string) ([]map[string]interface{}, error)

	// Maintenance operations
	CleanupOldData(days int) error
	GetSystemHealth() map[string]interface{}

	// External integrations
	ExportData(exportType string, data interface{}) error
	CreateDashboard(clubID string) error
	SendMetricsToExternal(metrics map[string]interface{}) error

	// Monitoring access
	GetHealthChecker() *analyticsmonitoring.HealthChecker
	GetMonitoringMetrics() *analyticsmonitoring.AnalyticsMetrics

	// Event processing
	ProcessAnalyticsEvent(eventType string, data map[string]interface{}) error
	StartEventProcessor() error
	StopEventProcessor() error
}

type service struct {
	repo         repository.Repository
	logger       logging.Logger
	natsClient   messaging.MessageBus
	monitoring   *monitoring.Monitor
	integrations *integrations.AnalyticsIntegrations
	metrics      *analyticsmonitoring.AnalyticsMetrics
	health       *analyticsmonitoring.HealthChecker
	stopChannel  chan bool
}

func NewService(repo repository.Repository, logger logging.Logger, natsClient messaging.MessageBus, monitor *monitoring.Monitor, integrations *integrations.AnalyticsIntegrations) AnalyticsService {
	metrics := analyticsmonitoring.NewAnalyticsMetrics(logger)

	// Get the underlying GORM DB from repository
	// Type assertion to get GetDB method
	repoWithDB, ok := repo.(interface{ GetDB() *gorm.DB })
	var db *gorm.DB
	if ok {
		db = repoWithDB.GetDB()
	}

	health := analyticsmonitoring.NewHealthChecker(db, integrations, logger)

	return &service{
		repo:         repo,
		logger:       logger,
		natsClient:   natsClient,
		monitoring:   monitor,
		integrations: integrations,
		metrics:      metrics,
		health:       health,
		stopChannel:  make(chan bool, 1),
	}
}

func (s *service) IsReady() bool {
	// Check if all dependencies are healthy
	if !s.repo.IsHealthy() {
		s.logger.Error("Repository is not healthy", map[string]interface{}{})
		return false
	}

	if err := s.natsClient.HealthCheck(context.Background()); err != nil {
		s.logger.Error("NATS client is not connected", map[string]interface{}{"error": err.Error()})
		return false
	}

	return true
}

func (s *service) GetMetrics(clubID string, timeRange string) (map[string]interface{}, error) {
	start := time.Now()
	s.monitoring.RecordBusinessEvent("analytics_metrics_requests", clubID)

	// Parse time range
	timeRangeObj, err := s.parseTimeRange(timeRange)
	if err != nil {
		s.metrics.RecordProcessingError("get_metrics", "parse_error")
		s.logger.Error("Invalid time range", map[string]interface{}{"error": err.Error(), "time_range": timeRange})
		return nil, fmt.Errorf("invalid time range: %w", err)
	}

	// Get aggregated metrics
	metrics, err := s.repo.AggregateMetrics(clubID, *timeRangeObj)
	if err != nil {
		s.metrics.RecordProcessingError("get_metrics", "aggregation_error")
		s.logger.Error("Failed to get aggregated metrics", map[string]interface{}{"error": err.Error(), "club_id": clubID})
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	// Get detailed metrics
	detailedMetrics, err := s.repo.GetMetricsByClub(clubID, *timeRangeObj)
	if err != nil {
		s.metrics.RecordProcessingError("get_metrics", "query_error")
		s.logger.Error("Failed to get detailed metrics", map[string]interface{}{"error": err.Error(), "club_id": clubID})
		return nil, fmt.Errorf("failed to get detailed metrics: %w", err)
	}

	// Combine results
	result := map[string]interface{}{
		"summary": metrics,
		"details": detailedMetrics,
		"club_id": clubID,
		"time_range": timeRange,
		"generated_at": time.Now(),
	}

	// Record success metrics
	s.metrics.RecordProcessingDuration("get_metrics", "success", time.Since(start))
	s.logger.Info("Retrieved metrics for club", map[string]interface{}{"club_id": clubID})
	return result, nil
}

func (s *service) GetReports(clubID string, reportType string) ([]map[string]interface{}, error) {
	s.monitoring.RecordBusinessEvent("analytics_reports_requests", clubID)

	reports, err := s.repo.GetReportsByClub(clubID, reportType)
	if err != nil {
		s.logger.Error("Failed to get reports", map[string]interface{}{"error": err.Error(), "club_id": clubID, "report_type": reportType})
		return nil, fmt.Errorf("failed to get reports: %w", err)
	}

	// Convert to generic map format
	result := make([]map[string]interface{}, len(reports))
	for i, report := range reports {
		result[i] = map[string]interface{}{
			"id":           report.ID,
			"club_id":      report.ClubID,
			"report_type":  report.ReportType,
			"title":        report.Title,
			"data":         report.Data,
			"generated_at": report.GeneratedAt,
			"created_at":   report.CreatedAt,
		}
	}

	s.logger.Info("Retrieved reports for club", map[string]interface{}{"club_id": clubID, "count": len(result), "report_type": reportType})
	return result, nil
}

func (s *service) RecordEvent(eventData map[string]interface{}) error {
	start := time.Now()
	eventType := "unknown"
	if et, ok := eventData["event_type"]; ok {
		eventType = fmt.Sprintf("%v", et)
	}

	s.monitoring.RecordBusinessEvent("analytics_events_recorded", fmt.Sprintf("%v", eventData["club_id"]))

	// Validate required fields
	if eventData["club_id"] == nil || eventData["event_type"] == nil {
		s.metrics.RecordProcessingError("record_event", "validation_error")
		return fmt.Errorf("club_id and event_type are required")
	}

	// Create analytics event
	event := &repository.AnalyticsEvent{
		ClubID:    fmt.Sprintf("%v", eventData["club_id"]),
		EventType: eventType,
		Data:      eventData,
		Timestamp: time.Now(),
	}

	// Store in database
	if err := s.repo.RecordEvent(event); err != nil {
		s.metrics.RecordProcessingError("record_event", "database_error")
		s.logger.Error("Failed to record event", map[string]interface{}{"error": err.Error(), "event_type": event.EventType, "club_id": event.ClubID})
		return fmt.Errorf("failed to record event: %w", err)
	}

	// Publish event to NATS for real-time processing
	if err := s.publishEvent(event); err != nil {
		s.metrics.RecordProcessingError("record_event", "publish_error")
		s.logger.Error("Failed to publish event", map[string]interface{}{"error": err.Error(), "event_type": event.EventType, "club_id": event.ClubID})
		// Don't fail the request if publishing fails
	}

	// Record success metrics
	s.metrics.RecordEventRecorded(event.ClubID, event.EventType, "api")
	s.metrics.RecordProcessingDuration("record_event", "success", time.Since(start))
	s.logger.Info("Recorded event for club", map[string]interface{}{"event_type": event.EventType, "club_id": event.ClubID})
	return nil
}

func (s *service) GenerateReport(clubID string, reportType string) (map[string]interface{}, error) {
	s.monitoring.RecordBusinessEvent("analytics_reports_generated", clubID)

	// Generate report based on type
	var reportData map[string]interface{}
	var title string

	switch reportType {
	case "usage":
		reportData = s.generateUsageReport(clubID)
		title = "Usage Report"
	case "engagement":
		reportData = s.generateEngagementReport(clubID)
		title = "Engagement Report"
	case "performance":
		reportData = s.generatePerformanceReport(clubID)
		title = "Performance Report"
	default:
		return nil, fmt.Errorf("unsupported report type: %s", reportType)
	}

	// Store report in database
	report := &repository.AnalyticsReport{
		ClubID:      clubID,
		ReportType:  reportType,
		Title:       title,
		Data:        reportData,
		GeneratedAt: time.Now(),
	}

	if err := s.repo.CreateReport(report); err != nil {
		s.logger.Error("Failed to store report", map[string]interface{}{"error": err.Error(), "club_id": clubID, "report_type": reportType})
		// Continue without failing - return the data even if storage fails
	}

	result := map[string]interface{}{
		"club_id":      clubID,
		"report_type":  reportType,
		"title":        title,
		"data":         reportData,
		"generated_at": time.Now(),
	}

	s.logger.Info("Generated report for club", map[string]interface{}{"report_type": reportType, "club_id": clubID})
	return result, nil
}

func (s *service) ProcessAnalyticsEvent(eventType string, data map[string]interface{}) error {
	// Process different types of analytics events
	switch eventType {
	case "member_visit":
		return s.processMemberVisitEvent(data)
	case "reciprocal_usage":
		return s.processReciprocalUsageEvent(data)
	case "system_metric":
		return s.processSystemMetricEvent(data)
	default:
		s.logger.Warn("Unknown event type", map[string]interface{}{"event_type": eventType})
		return nil
	}
}

func (s *service) StartEventProcessor() error {
	// Subscribe to analytics events from NATS
	err := s.natsClient.Subscribe("analytics.events.*", func(ctx context.Context, msg *messaging.Message) error {
		var event map[string]interface{}
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			s.logger.Error("Failed to unmarshal event", map[string]interface{}{"error": err.Error()})
			return err
		}

		eventType := fmt.Sprintf("%v", event["event_type"])
		return s.ProcessAnalyticsEvent(eventType, event)
	})

	if err != nil {
		return fmt.Errorf("failed to start event processor: %w", err)
	}

	go func() {
		<-s.stopChannel
		// Note: NATS subscriptions are managed by the connection lifecycle
		s.logger.Info("Event processor stop signal received", map[string]interface{}{})
	}()

	s.logger.Info("Analytics event processor started", map[string]interface{}{})
	return nil
}

func (s *service) StopEventProcessor() error {
	select {
	case s.stopChannel <- true:
	default:
		// Channel already has a value or is closed
	}
	s.logger.Info("Analytics event processor stopped", map[string]interface{}{})
	return nil
}

func (s *service) GetRealtimeMetrics(clubID string) (map[string]interface{}, error) {
	s.monitoring.RecordBusinessEvent("analytics_realtime_metrics_requests", clubID)

	metrics, err := s.repo.GetRealtimeMetrics(clubID)
	if err != nil {
		s.logger.Error("Failed to get realtime metrics", map[string]interface{}{"error": err.Error(), "club_id": clubID})
		return nil, fmt.Errorf("failed to get realtime metrics: %w", err)
	}

	s.logger.Info("Retrieved realtime metrics for club", map[string]interface{}{"club_id": clubID})
	return metrics, nil
}

func (s *service) RecordMetric(clubID string, metricName string, value float64, tags map[string]interface{}) error {
	s.monitoring.RecordBusinessEvent("analytics_metrics_recorded", clubID)

	// Validate inputs
	if clubID == "" || metricName == "" {
		return fmt.Errorf("club_id and metric_name are required")
	}

	metric := &repository.AnalyticsMetric{
		ClubID:      clubID,
		MetricName:  metricName,
		MetricValue: value,
		Tags:        tags,
		Timestamp:   time.Now(),
	}

	if err := s.repo.RecordMetric(metric); err != nil {
		s.logger.Error("Failed to record metric", map[string]interface{}{"error": err.Error(), "club_id": clubID, "metric_name": metricName})
		return fmt.Errorf("failed to record metric: %w", err)
	}

	s.logger.Info("Recorded metric for club", map[string]interface{}{"club_id": clubID, "metric_name": metricName, "value": value})
	return nil
}

func (s *service) GetEvents(clubID string, timeRange string) ([]map[string]interface{}, error) {
	s.monitoring.RecordBusinessEvent("analytics_events_requests", clubID)

	// Parse time range
	timeRangeObj, err := s.parseTimeRange(timeRange)
	if err != nil {
		s.logger.Error("Invalid time range", map[string]interface{}{"error": err.Error(), "time_range": timeRange})
		return nil, fmt.Errorf("invalid time range: %w", err)
	}

	// Get events from repository
	events, err := s.repo.GetEventsByClub(clubID, *timeRangeObj)
	if err != nil {
		s.logger.Error("Failed to get events", map[string]interface{}{"error": err.Error(), "club_id": clubID})
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	// Convert to generic map format
	result := make([]map[string]interface{}, len(events))
	for i, event := range events {
		result[i] = map[string]interface{}{
			"id":         event.ID,
			"club_id":    event.ClubID,
			"event_type": event.EventType,
			"data":       event.Data,
			"timestamp":  event.Timestamp,
			"created_at": event.CreatedAt,
		}
	}

	s.logger.Info("Retrieved events for club", map[string]interface{}{"club_id": clubID, "count": len(result), "time_range": timeRange})
	return result, nil
}

func (s *service) CleanupOldData(days int) error {
	s.monitoring.RecordBusinessEvent("analytics_cleanup_operations", "system")

	if days <= 0 {
		return fmt.Errorf("days must be greater than 0")
	}

	// Calculate cutoff time
	cutoffTime := time.Now().AddDate(0, 0, -days)

	if err := s.repo.CleanupOldEvents(cutoffTime); err != nil {
		s.logger.Error("Failed to cleanup old data", map[string]interface{}{"error": err.Error(), "days": days})
		return fmt.Errorf("failed to cleanup old data: %w", err)
	}

	s.logger.Info("Cleaned up old data", map[string]interface{}{"days": days, "cutoff_time": cutoffTime})
	return nil
}

func (s *service) GetSystemHealth() map[string]interface{} {
	health := map[string]interface{}{
		"timestamp": time.Now(),
		"status":    "healthy",
		"components": map[string]interface{}{
			"database":    s.repo.IsHealthy(),
			"nats":        s.natsClient.HealthCheck(context.Background()) == nil,
			"event_processor": len(s.stopChannel) == 0, // Running if stop channel is empty
		},
	}

	// Determine overall status
	components := health["components"].(map[string]interface{})
	allHealthy := true
	for _, status := range components {
		if !status.(bool) {
			allHealthy = false
			break
		}
	}

	if allHealthy {
		health["status"] = "healthy"
	} else {
		health["status"] = "degraded"
	}

	return health
}

func (s *service) ExportData(exportType string, data interface{}) error {
	s.monitoring.RecordBusinessEvent("analytics_data_exports", "system")

	if s.integrations == nil {
		return fmt.Errorf("integrations not configured")
	}

	if err := s.integrations.ExportData(context.Background(), data, exportType); err != nil {
		s.logger.Error("Failed to export data", map[string]interface{}{"error": err.Error(), "export_type": exportType})
		return fmt.Errorf("failed to export data: %w", err)
	}

	s.logger.Info("Data exported successfully", map[string]interface{}{"export_type": exportType})
	return nil
}

func (s *service) CreateDashboard(clubID string) error {
	s.monitoring.RecordBusinessEvent("analytics_dashboard_creations", clubID)

	if s.integrations == nil {
		return fmt.Errorf("integrations not configured")
	}

	// Create dashboard configuration
	dashboardConfig := map[string]interface{}{
		"title":   fmt.Sprintf("Analytics Dashboard - Club %s", clubID),
		"club_id": clubID,
		"panels": []map[string]interface{}{
			{
				"title": "Event Count",
				"type":  "stat",
				"query": fmt.Sprintf("analytics_events_total{club_id=\"%s\"}", clubID),
			},
			{
				"title": "Event Rate",
				"type":  "graph",
				"query": fmt.Sprintf("rate(analytics_events_total{club_id=\"%s\"}[5m])", clubID),
			},
		},
	}

	if err := s.integrations.CreateDashboard(context.Background(), dashboardConfig); err != nil {
		s.logger.Error("Failed to create dashboard", map[string]interface{}{"error": err.Error(), "club_id": clubID})
		return fmt.Errorf("failed to create dashboard: %w", err)
	}

	s.logger.Info("Dashboard created successfully", map[string]interface{}{"club_id": clubID})
	return nil
}

func (s *service) SendMetricsToExternal(metrics map[string]interface{}) error {
	s.monitoring.RecordBusinessEvent("analytics_external_metrics", "system")

	if s.integrations == nil {
		return fmt.Errorf("integrations not configured")
	}

	if err := s.integrations.SendMetrics(context.Background(), metrics); err != nil {
		s.logger.Error("Failed to send metrics to external systems", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("failed to send metrics: %w", err)
	}

	s.logger.Info("Metrics sent to external systems", map[string]interface{}{"metric_count": len(metrics)})
	return nil
}

func (s *service) GetHealthChecker() *analyticsmonitoring.HealthChecker {
	return s.health
}

func (s *service) GetMonitoringMetrics() *analyticsmonitoring.AnalyticsMetrics {
	return s.metrics
}

// Private helper methods

func (s *service) parseTimeRange(timeRange string) (*repository.TimeRange, error) {
	now := time.Now()
	var start, end time.Time

	switch timeRange {
	case "1h":
		start = now.Add(-time.Hour)
		end = now
	case "24h":
		start = now.Add(-24 * time.Hour)
		end = now
	case "7d":
		start = now.Add(-7 * 24 * time.Hour)
		end = now
	case "30d":
		start = now.Add(-30 * 24 * time.Hour)
		end = now
	default:
		return nil, fmt.Errorf("unsupported time range: %s", timeRange)
	}

	return &repository.TimeRange{Start: start, End: end}, nil
}

func (s *service) publishEvent(event *repository.AnalyticsEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	subject := fmt.Sprintf("analytics.events.%s", event.EventType)
	return s.natsClient.Publish(context.Background(), subject, data)
}

func (s *service) generateUsageReport(clubID string) map[string]interface{} {
	// Mock usage report data
	return map[string]interface{}{
		"total_visits":    150,
		"unique_visitors": 85,
		"peak_hours":      []string{"12:00-13:00", "18:00-19:00"},
		"popular_areas":   []string{"gym", "restaurant", "pool"},
	}
}

func (s *service) generateEngagementReport(clubID string) map[string]interface{} {
	// Mock engagement report data
	return map[string]interface{}{
		"average_session_duration": "45 minutes",
		"return_visit_rate":        0.65,
		"popular_activities":       []string{"fitness", "dining", "events"},
		"member_satisfaction":      4.2,
	}
}

func (s *service) generatePerformanceReport(clubID string) map[string]interface{} {
	// Mock performance report data
	return map[string]interface{}{
		"response_time_avg": "150ms",
		"uptime":            0.995,
		"error_rate":        0.002,
		"throughput":        "1200 req/min",
	}
}

func (s *service) processMemberVisitEvent(data map[string]interface{}) error {
	// Process member visit event
	s.logger.Info("Processing member visit event", map[string]interface{}{"data": data})
	return nil
}

func (s *service) processReciprocalUsageEvent(data map[string]interface{}) error {
	// Process reciprocal usage event
	s.logger.Info("Processing reciprocal usage event", map[string]interface{}{"data": data})
	return nil
}

func (s *service) processSystemMetricEvent(data map[string]interface{}) error {
	// Process system metric event
	s.logger.Info("Processing system metric event", map[string]interface{}{"data": data})
	return nil
}
