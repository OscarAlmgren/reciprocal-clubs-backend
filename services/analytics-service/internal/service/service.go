package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/messaging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
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

	// Event processing
	ProcessAnalyticsEvent(eventType string, data map[string]interface{}) error
	StartEventProcessor() error
	StopEventProcessor() error
}

type service struct {
	repo        repository.Repository
	logger      logging.Logger
	natsClient  messaging.MessageBus
	monitoring  *monitoring.Monitor
	stopChannel chan bool
}

func NewService(repo repository.Repository, logger logging.Logger, natsClient messaging.MessageBus, monitor *monitoring.Monitor) AnalyticsService {
	return &service{
		repo:        repo,
		logger:      logger,
		natsClient:  natsClient,
		monitoring:  monitor,
		stopChannel: make(chan bool, 1),
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
	s.monitoring.RecordBusinessEvent("analytics_metrics_requests", clubID)

	// Parse time range
	timeRangeObj, err := s.parseTimeRange(timeRange)
	if err != nil {
		s.logger.Error("Invalid time range", map[string]interface{}{"error": err.Error(), "time_range": timeRange})
		return nil, fmt.Errorf("invalid time range: %w", err)
	}

	// Get aggregated metrics
	metrics, err := s.repo.AggregateMetrics(clubID, *timeRangeObj)
	if err != nil {
		s.logger.Error("Failed to get aggregated metrics", map[string]interface{}{"error": err.Error(), "club_id": clubID})
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	// Get detailed metrics
	detailedMetrics, err := s.repo.GetMetricsByClub(clubID, *timeRangeObj)
	if err != nil {
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
	_ = "unknown" // placeholder for eventType default
	if et, ok := eventData["event_type"]; ok {
		_ = fmt.Sprintf("%v", et) // eventType extracted but not used directly
	}

	s.monitoring.RecordBusinessEvent("analytics_events_recorded", fmt.Sprintf("%v", eventData["club_id"]))

	// Validate required fields
	if eventData["club_id"] == nil || eventData["event_type"] == nil {
		return fmt.Errorf("club_id and event_type are required")
	}

	// Create analytics event
	event := &repository.AnalyticsEvent{
		ClubID:    fmt.Sprintf("%v", eventData["club_id"]),
		EventType: fmt.Sprintf("%v", eventData["event_type"]),
		Data:      eventData,
		Timestamp: time.Now(),
	}

	// Store in database
	if err := s.repo.RecordEvent(event); err != nil {
		s.logger.Error("Failed to record event", map[string]interface{}{"error": err.Error(), "event_type": event.EventType, "club_id": event.ClubID})
		return fmt.Errorf("failed to record event: %w", err)
	}

	// Publish event to NATS for real-time processing
	if err := s.publishEvent(event); err != nil {
		s.logger.Error("Failed to publish event", map[string]interface{}{"error": err.Error(), "event_type": event.EventType, "club_id": event.ClubID})
		// Don't fail the request if publishing fails
	}

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
	_ = &repository.AnalyticsReport{
		ClubID:      clubID,
		ReportType:  reportType,
		Title:       title,
		Data:        reportData,
		GeneratedAt: time.Now(),
	}

	// Note: You'd need to add a CreateReport method to the repository
	// For now, returning the generated data

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
