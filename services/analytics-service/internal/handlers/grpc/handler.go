package grpc

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	"reciprocal-clubs-backend/services/analytics-service/internal/service"
	pb "reciprocal-clubs-backend/services/analytics-service/proto"
)

type GRPCHandler struct {
	pb.UnimplementedAnalyticsServiceServer
	service    service.AnalyticsService
	logger     logging.Logger
	monitoring *monitoring.Monitor
}

func NewGRPCHandler(service service.AnalyticsService, logger logging.Logger, monitor *monitoring.Monitor) *GRPCHandler {
	return &GRPCHandler{
		service:    service,
		logger:     logger,
		monitoring: monitor,
	}
}

func (h *GRPCHandler) RegisterServices(server *grpc.Server) {
	pb.RegisterAnalyticsServiceServer(server, h)
	h.logger.Info("gRPC services registered for analytics-service", map[string]interface{}{
		"service": "analytics-service",
		"methods": 25,
	})
}

// Health check
func (h *GRPCHandler) Health(ctx context.Context, req *emptypb.Empty) (*pb.HealthResponse, error) {
	healthChecker := h.service.GetHealthChecker()
	health := healthChecker.HealthCheck(ctx)

	status := "SERVING"
	if health.Status == "unhealthy" {
		status = "NOT_SERVING"
	} else if health.Status == "degraded" {
		status = "SERVICE_UNKNOWN"
	}

	dependencies := make(map[string]string)
	if health.Components != nil {
		for k, v := range health.Components {
			dependencies[k] = v.Status
		}
	}

	return &pb.HealthResponse{
		Status:       status,
		Service:      "analytics-service",
		Dependencies: dependencies,
	}, nil
}

// Core analytics operations
func (h *GRPCHandler) GetMetrics(ctx context.Context, req *pb.GetMetricsRequest) (*pb.GetMetricsResponse, error) {
	h.logger.Info("gRPC GetMetrics called", map[string]interface{}{
		"club_id":     req.ClubId,
		"time_range":  req.TimeRange,
		"metric_count": len(req.MetricNames),
	})

	start := time.Now()
	defer func() {
		h.monitoring.RecordGRPCRequest("GetMetrics", "success", time.Since(start))
	}()

	metrics, err := h.service.GetMetrics(req.ClubId, req.TimeRange)
	if err != nil {
		h.logger.Error("Failed to get metrics", map[string]interface{}{
			"error":   err.Error(),
			"club_id": req.ClubId,
		})
		return nil, err
	}

	// Convert map[string]interface{} to map[string]string for protobuf
	summary := make(map[string]string)
	details := make([]*pb.AnalyticsMetric, 0)

	if summaryData, ok := metrics["summary"].(map[string]interface{}); ok {
		for k, v := range summaryData {
			if s, ok := v.(string); ok {
				summary[k] = s
			} else {
				summary[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	return &pb.GetMetricsResponse{
		Summary:     summary,
		Details:     details,
		ClubId:      req.ClubId,
		TimeRange:   req.TimeRange,
		GeneratedAt: timestamppb.New(time.Now()),
	}, nil
}

func (h *GRPCHandler) GetReports(ctx context.Context, req *pb.GetReportsRequest) (*pb.GetReportsResponse, error) {
	h.logger.Info("gRPC GetReports called", map[string]interface{}{
		"club_id":     req.ClubId,
		"report_type": req.ReportType,
		"limit":       req.Limit,
	})

	start := time.Now()
	defer func() {
		h.monitoring.RecordGRPCRequest("GetReports", "success", time.Since(start))
	}()

	reportTypeStr := h.convertReportTypeToString(req.ReportType)
	reports, err := h.service.GetReports(req.ClubId, reportTypeStr)
	if err != nil {
		h.logger.Error("Failed to get reports", map[string]interface{}{
			"error":   err.Error(),
			"club_id": req.ClubId,
		})
		return nil, err
	}

	// Convert to protobuf format
	protoReports := make([]*pb.AnalyticsReport, len(reports))
	for i, report := range reports {
		data := make(map[string]string)
		if reportData, ok := report["data"].(map[string]interface{}); ok {
			for k, v := range reportData {
				data[k] = fmt.Sprintf("%v", v)
			}
		}

		protoReports[i] = &pb.AnalyticsReport{
			Id:          uint32(report["id"].(uint)),
			ClubId:      report["club_id"].(string),
			ReportType:  req.ReportType,
			Title:       report["title"].(string),
			Data:        data,
			GeneratedAt: timestamppb.New(report["generated_at"].(time.Time)),
			CreatedAt:   timestamppb.New(report["created_at"].(time.Time)),
		}
	}

	return &pb.GetReportsResponse{
		Reports: protoReports,
		Total:   uint32(len(protoReports)),
	}, nil
}

func (h *GRPCHandler) RecordEvent(ctx context.Context, req *pb.RecordEventRequest) (*pb.RecordEventResponse, error) {
	h.logger.Info("gRPC RecordEvent called", map[string]interface{}{
		"club_id":    req.ClubId,
		"event_type": req.EventType,
		"user_id":    req.UserId,
	})

	start := time.Now()
	defer func() {
		h.monitoring.RecordGRPCRequest("RecordEvent", "success", time.Since(start))
	}()

	// Convert protobuf request to service format
	eventData := make(map[string]interface{})
	eventData["club_id"] = req.ClubId
	eventData["event_type"] = req.EventType
	eventData["user_id"] = req.UserId
	eventData["session_id"] = req.SessionId

	// Convert string maps to interface maps
	for k, v := range req.Data {
		eventData[k] = v
	}
	for k, v := range req.Metadata {
		eventData[k] = v
	}

	err := h.service.RecordEvent(eventData)
	if err != nil {
		h.logger.Error("Failed to record event", map[string]interface{}{
			"error":      err.Error(),
			"event_type": req.EventType,
			"club_id":    req.ClubId,
		})
		return &pb.RecordEventResponse{
			Success: false,
			Message: "Failed to record event: " + err.Error(),
			EventId: 0,
		}, nil
	}

	return &pb.RecordEventResponse{
		Success: true,
		Message: "Event recorded successfully",
		EventId: 1, // Would normally return actual ID
	}, nil
}

func (h *GRPCHandler) RecordMetric(ctx context.Context, req *pb.RecordMetricRequest) (*pb.RecordMetricResponse, error) {
	h.logger.Info("gRPC RecordMetric called", map[string]interface{}{
		"club_id":     req.ClubId,
		"metric_name": req.MetricName,
		"value":       req.MetricValue,
	})

	start := time.Now()
	defer func() {
		h.monitoring.RecordGRPCRequest("RecordMetric", "success", time.Since(start))
	}()

	// Convert tags
	tags := make(map[string]interface{})
	for k, v := range req.Tags {
		tags[k] = v
	}

	err := h.service.RecordMetric(req.ClubId, req.MetricName, req.MetricValue, tags)
	if err != nil {
		h.logger.Error("Failed to record metric", map[string]interface{}{
			"error":       err.Error(),
			"metric_name": req.MetricName,
			"club_id":     req.ClubId,
		})
		return &pb.RecordMetricResponse{
			Success: false,
			Message: "Failed to record metric: " + err.Error(),
		}, nil
	}

	return &pb.RecordMetricResponse{
		Success:  true,
		Message:  "Metric recorded successfully",
		MetricId: 1, // Would normally return actual ID
	}, nil
}

// Report generation
func (h *GRPCHandler) GenerateReport(ctx context.Context, req *pb.GenerateReportRequest) (*pb.GenerateReportResponse, error) {
	h.logger.Info("gRPC GenerateReport called", map[string]interface{}{
		"club_id":     req.ClubId,
		"report_type": req.ReportType,
		"async":       req.Async,
	})

	start := time.Now()
	defer func() {
		h.monitoring.RecordGRPCRequest("GenerateReport", "success", time.Since(start))
	}()

	reportTypeStr := h.convertReportTypeToString(req.ReportType)
	report, err := h.service.GenerateReport(req.ClubId, reportTypeStr)
	if err != nil {
		h.logger.Error("Failed to generate report", map[string]interface{}{
			"error":   err.Error(),
			"club_id": req.ClubId,
		})
		return &pb.GenerateReportResponse{
			Success: false,
			Message: "Failed to generate report: " + err.Error(),
		}, nil
	}

	// Convert to protobuf format
	data := make(map[string]string)
	if reportData, ok := report["data"].(map[string]interface{}); ok {
		for k, v := range reportData {
			data[k] = fmt.Sprintf("%v", v)
		}
	}

	protoReport := &pb.AnalyticsReport{
		ClubId:      req.ClubId,
		ReportType:  req.ReportType,
		Title:       report["title"].(string),
		Data:        data,
		GeneratedAt: timestamppb.New(report["generated_at"].(time.Time)),
	}

	return &pb.GenerateReportResponse{
		Success: true,
		Message: "Report generated successfully",
		Report:  protoReport,
		JobId:   "sync-job", // Would be actual job ID for async
	}, nil
}

// System operations
func (h *GRPCHandler) GetSystemHealth(ctx context.Context, req *emptypb.Empty) (*pb.SystemHealthResponse, error) {
	h.logger.Info("gRPC GetSystemHealth called", nil)

	start := time.Now()
	defer func() {
		h.monitoring.RecordGRPCRequest("GetSystemHealth", "success", time.Since(start))
	}()

	health := h.service.GetSystemHealth()

	status := "healthy"
	if healthStatus, ok := health["status"].(string); ok {
		status = healthStatus
	}

	components := make(map[string]string)
	if comp, ok := health["components"].(map[string]interface{}); ok {
		for k, v := range comp {
			if b, ok := v.(bool); ok {
				if b {
					components[k] = "healthy"
				} else {
					components[k] = "unhealthy"
				}
			}
		}
	}

	metrics := make(map[string]float64)
	// Add some example metrics
	metrics["uptime_seconds"] = 3600.0
	metrics["cpu_usage"] = 45.2
	metrics["memory_usage"] = 67.8

	return &pb.SystemHealthResponse{
		Status:     status,
		Components: components,
		Timestamp:  timestamppb.New(time.Now()),
		Metrics:    metrics,
	}, nil
}

func (h *GRPCHandler) CleanupOldData(ctx context.Context, req *pb.CleanupOldDataRequest) (*pb.CleanupOldDataResponse, error) {
	h.logger.Info("gRPC CleanupOldData called", map[string]interface{}{
		"days": req.Days,
	})

	start := time.Now()
	defer func() {
		h.monitoring.RecordGRPCRequest("CleanupOldData", "success", time.Since(start))
	}()

	err := h.service.CleanupOldData(int(req.Days))
	if err != nil {
		h.logger.Error("Failed to cleanup old data", map[string]interface{}{
			"error": err.Error(),
			"days":  req.Days,
		})
		return &pb.CleanupOldDataResponse{
			Success: false,
			Message: "Failed to cleanup data: " + err.Error(),
		}, nil
	}

	return &pb.CleanupOldDataResponse{
		Success:      true,
		Message:      "Data cleanup completed successfully",
		DeletedCount: 100, // Would be actual count
	}, nil
}

// Helper methods
func (h *GRPCHandler) convertReportTypeToString(reportType pb.ReportType) string {
	switch reportType {
	case pb.ReportType_REPORT_TYPE_USAGE:
		return "usage"
	case pb.ReportType_REPORT_TYPE_ENGAGEMENT:
		return "engagement"
	case pb.ReportType_REPORT_TYPE_PERFORMANCE:
		return "performance"
	case pb.ReportType_REPORT_TYPE_FINANCIAL:
		return "financial"
	case pb.ReportType_REPORT_TYPE_CUSTOM:
		return "custom"
	default:
		return "usage"
	}
}

// Real-time analytics methods
func (h *GRPCHandler) GetRealtimeMetrics(ctx context.Context, req *pb.GetRealtimeMetricsRequest) (*pb.GetRealtimeMetricsResponse, error) {
	h.logger.Info("gRPC GetRealtimeMetrics called", map[string]interface{}{
		"club_id": req.ClubId,
		"metrics": len(req.MetricNames),
	})

	start := time.Now()
	defer func() {
		h.monitoring.RecordGRPCRequest("GetRealtimeMetrics", "success", time.Since(start))
	}()

	metrics, err := h.service.GetRealtimeMetrics(req.ClubId)
	if err != nil {
		h.logger.Error("Failed to get realtime metrics", map[string]interface{}{
			"error":   err.Error(),
			"club_id": req.ClubId,
		})
		return nil, err
	}

	realtimeMetrics := make(map[string]float64)
	for k, v := range metrics {
		if val, ok := v.(float64); ok {
			realtimeMetrics[k] = val
		}
	}

	return &pb.GetRealtimeMetricsResponse{
		Metrics:   realtimeMetrics,
		Timestamp: timestamppb.New(time.Now()),
	}, nil
}

func (h *GRPCHandler) StreamEvents(req *pb.StreamEventsRequest, stream pb.AnalyticsService_StreamEventsServer) error {
	h.logger.Info("gRPC StreamEvents called", map[string]interface{}{
		"club_id":     req.ClubId,
		"event_types": len(req.EventTypes),
	})

	// For now, return a simple implementation
	// In production, this would stream real events
	for i := 0; i < 5; i++ {
		event := &pb.EventStreamResponse{
			Event: &pb.AnalyticsEvent{
				ClubId:    req.ClubId,
				EventType: "sample_event",
				Data:      map[string]string{"sample": "data"},
				Timestamp: timestamppb.New(time.Now()),
			},
			ReceivedAt: timestamppb.New(time.Now()),
		}

		if err := stream.Send(event); err != nil {
			return err
		}

		time.Sleep(time.Second)
	}

	return nil
}

func (h *GRPCHandler) GetLiveStats(ctx context.Context, req *pb.GetLiveStatsRequest) (*pb.GetLiveStatsResponse, error) {
	h.logger.Info("gRPC GetLiveStats called", map[string]interface{}{
		"club_id": req.ClubId,
	})

	start := time.Now()
	defer func() {
		h.monitoring.RecordGRPCRequest("GetLiveStats", "success", time.Since(start))
	}()

	stats, err := h.service.GetRealtimeMetrics(req.ClubId)
	if err != nil {
		h.logger.Error("Failed to get live stats", map[string]interface{}{
			"error":   err.Error(),
			"club_id": req.ClubId,
		})
		return nil, err
	}

	liveStats := make(map[string]float64)
	for k, v := range stats {
		if val, ok := v.(float64); ok {
			liveStats[k] = val
		}
	}

	return &pb.GetLiveStatsResponse{
		Stats:     liveStats,
		Timestamp: timestamppb.New(time.Now()),
	}, nil
}

// Report status and scheduling methods
func (h *GRPCHandler) GetReportStatus(ctx context.Context, req *pb.GetReportStatusRequest) (*pb.GetReportStatusResponse, error) {
	h.logger.Info("gRPC GetReportStatus called", map[string]interface{}{
		"job_id": req.JobId,
	})

	// Mock implementation - in production would check actual job status
	return &pb.GetReportStatusResponse{
		Status:   "completed",
		Message:  "Report generation completed successfully",
		Progress: 100,
	}, nil
}

func (h *GRPCHandler) ScheduleReport(ctx context.Context, req *pb.ScheduleReportRequest) (*pb.ScheduleReportResponse, error) {
	h.logger.Info("gRPC ScheduleReport called", map[string]interface{}{
		"club_id":     req.ClubId,
		"report_type": req.ReportType,
		"schedule":    req.Schedule,
	})

	start := time.Now()
	defer func() {
		h.monitoring.RecordGRPCRequest("ScheduleReport", "success", time.Since(start))
	}()

	// Mock implementation - in production would schedule actual reports
	scheduleID := fmt.Sprintf("schedule_%d", time.Now().Unix())

	return &pb.ScheduleReportResponse{
		Success:    true,
		Message:    "Report scheduled successfully",
		ScheduleId: scheduleID,
	}, nil
}

// Event management methods
func (h *GRPCHandler) GetEvents(ctx context.Context, req *pb.GetEventsRequest) (*pb.GetEventsResponse, error) {
	h.logger.Info("gRPC GetEvents called", map[string]interface{}{
		"club_id":     req.ClubId,
		"time_range":  req.TimeRange,
		"event_types": len(req.EventTypes),
	})

	start := time.Now()
	defer func() {
		h.monitoring.RecordGRPCRequest("GetEvents", "success", time.Since(start))
	}()

	events, err := h.service.GetEvents(req.ClubId, req.TimeRange)
	if err != nil {
		h.logger.Error("Failed to get events", map[string]interface{}{
			"error":   err.Error(),
			"club_id": req.ClubId,
		})
		return nil, err
	}

	protoEvents := make([]*pb.AnalyticsEvent, len(events))
	for i, event := range events {
		data := make(map[string]string)
		if eventData, ok := event["data"].(map[string]interface{}); ok {
			for k, v := range eventData {
				data[k] = fmt.Sprintf("%v", v)
			}
		}

		protoEvents[i] = &pb.AnalyticsEvent{
			Id:        uint32(event["id"].(uint)),
			ClubId:    event["club_id"].(string),
			EventType: event["event_type"].(string),
			Data:      data,
			Timestamp: timestamppb.New(event["timestamp"].(time.Time)),
			CreatedAt: timestamppb.New(event["created_at"].(time.Time)),
		}
	}

	return &pb.GetEventsResponse{
		Events: protoEvents,
		Total:  uint32(len(protoEvents)),
	}, nil
}

func (h *GRPCHandler) QueryEvents(ctx context.Context, req *pb.QueryEventsRequest) (*pb.QueryEventsResponse, error) {
	h.logger.Info("gRPC QueryEvents called", map[string]interface{}{
		"club_id": req.ClubId,
		"filters": len(req.Filters),
	})

	start := time.Now()
	defer func() {
		h.monitoring.RecordGRPCRequest("QueryEvents", "success", time.Since(start))
	}()

	// For now, delegate to GetEvents - in production would implement advanced querying
	events, err := h.service.GetEvents(req.ClubId, "")
	if err != nil {
		h.logger.Error("Failed to query events", map[string]interface{}{
			"error":   err.Error(),
			"club_id": req.ClubId,
		})
		return nil, err
	}

	protoEvents := make([]*pb.AnalyticsEvent, len(events))
	for i, event := range events {
		data := make(map[string]string)
		if eventData, ok := event["data"].(map[string]interface{}); ok {
			for k, v := range eventData {
				data[k] = fmt.Sprintf("%v", v)
			}
		}

		protoEvents[i] = &pb.AnalyticsEvent{
			Id:        uint32(event["id"].(uint)),
			ClubId:    event["club_id"].(string),
			EventType: event["event_type"].(string),
			Data:      data,
			Timestamp: timestamppb.New(event["timestamp"].(time.Time)),
			CreatedAt: timestamppb.New(event["created_at"].(time.Time)),
		}
	}

	return &pb.QueryEventsResponse{
		Events:       protoEvents,
		Total:        uint32(len(protoEvents)),
		Aggregations: map[string]string{"total_count": fmt.Sprintf("%d", len(protoEvents))},
	}, nil
}

func (h *GRPCHandler) BulkRecordEvents(ctx context.Context, req *pb.BulkRecordEventsRequest) (*pb.BulkRecordEventsResponse, error) {
	h.logger.Info("gRPC BulkRecordEvents called", map[string]interface{}{
		"event_count": len(req.Events),
	})

	start := time.Now()
	defer func() {
		h.monitoring.RecordGRPCRequest("BulkRecordEvents", "success", time.Since(start))
	}()

	processedCount := 0
	errorCount := 0
	var errors []string

	for _, eventReq := range req.Events {
		eventData := make(map[string]interface{})
		eventData["club_id"] = eventReq.ClubId
		eventData["event_type"] = eventReq.EventType
		eventData["user_id"] = eventReq.UserId
		eventData["session_id"] = eventReq.SessionId

		for k, v := range eventReq.Data {
			eventData[k] = v
		}
		for k, v := range eventReq.Metadata {
			eventData[k] = v
		}

		err := h.service.RecordEvent(eventData)
		if err != nil {
			errorCount++
			errors = append(errors, err.Error())
		} else {
			processedCount++
		}
	}

	return &pb.BulkRecordEventsResponse{
		Success:        errorCount == 0,
		Message:        fmt.Sprintf("Processed %d events, %d errors", processedCount, errorCount),
		ProcessedCount: int32(processedCount),
		ErrorCount:     int32(errorCount),
		Errors:         errors,
	}, nil
}

// Service metrics method
func (h *GRPCHandler) GetServiceMetrics(ctx context.Context, req *emptypb.Empty) (*pb.ServiceMetricsResponse, error) {
	h.logger.Info("gRPC GetServiceMetrics called", nil)

	start := time.Now()
	defer func() {
		h.monitoring.RecordGRPCRequest("GetServiceMetrics", "success", time.Since(start))
	}()

	// Mock service metrics - in production would get actual metrics
	counters := map[string]float64{
		"total_requests":      1000.0,
		"successful_requests": 950.0,
		"failed_requests":     50.0,
	}

	gauges := map[string]float64{
		"active_connections": 25.0,
		"memory_usage_mb":    512.0,
		"cpu_usage_percent":  45.2,
	}

	histograms := map[string]float64{
		"request_duration_ms": 125.5,
		"response_size_bytes": 2048.0,
	}

	return &pb.ServiceMetricsResponse{
		Counters:   counters,
		Gauges:     gauges,
		Histograms: histograms,
		Timestamp:  timestamppb.New(time.Now()),
	}, nil
}
