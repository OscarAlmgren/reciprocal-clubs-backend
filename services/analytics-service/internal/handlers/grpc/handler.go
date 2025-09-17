package grpc

import (
	"context"

	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	"reciprocal-clubs-backend/services/analytics-service/internal/service"

	"google.golang.org/grpc"
)

// AnalyticsServiceServer represents the gRPC service interface
// This would normally be generated from protobuf definitions
type AnalyticsServiceServer interface {
	GetMetrics(context.Context, *GetMetricsRequest) (*GetMetricsResponse, error)
	GetReports(context.Context, *GetReportsRequest) (*GetReportsResponse, error)
	RecordEvent(context.Context, *RecordEventRequest) (*RecordEventResponse, error)
}

// Basic protobuf message structs (normally generated)
type GetMetricsRequest struct {
	ClubId    string `json:"club_id"`
	TimeRange string `json:"time_range"`
}

type GetMetricsResponse struct {
	Metrics map[string]interface{} `json:"metrics"`
}

type GetReportsRequest struct {
	ClubId     string `json:"club_id"`
	ReportType string `json:"report_type"`
}

type GetReportsResponse struct {
	Reports []map[string]interface{} `json:"reports"`
}

type RecordEventRequest struct {
	Event map[string]interface{} `json:"event"`
}

type RecordEventResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type GRPCHandler struct {
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
	// This would normally register the generated gRPC service
	// RegisterAnalyticsServiceServer(server, h)
	h.logger.Info("gRPC services registered for analytics-service", map[string]interface{}{})
}

// Implement the AnalyticsServiceServer interface
func (h *GRPCHandler) GetMetrics(ctx context.Context, req *GetMetricsRequest) (*GetMetricsResponse, error) {
	h.logger.Info("gRPC GetMetrics called", map[string]interface{}{"club_id": req.ClubId, "time_range": req.TimeRange})
	h.monitoring.RecordGRPCRequest("GetMetrics", "success", 0)

	metrics, err := h.service.GetMetrics(req.ClubId, req.TimeRange)
	if err != nil {
		h.logger.Error("Failed to get metrics", map[string]interface{}{"error": err.Error(), "club_id": req.ClubId})
		return nil, err
	}

	return &GetMetricsResponse{
		Metrics: metrics,
	}, nil
}

func (h *GRPCHandler) GetReports(ctx context.Context, req *GetReportsRequest) (*GetReportsResponse, error) {
	h.logger.Info("gRPC GetReports called", map[string]interface{}{"club_id": req.ClubId, "report_type": req.ReportType})
	h.monitoring.RecordGRPCRequest("GetReports", "success", 0)

	reports, err := h.service.GetReports(req.ClubId, req.ReportType)
	if err != nil {
		h.logger.Error("Failed to get reports", map[string]interface{}{"error": err.Error(), "club_id": req.ClubId})
		return nil, err
	}

	return &GetReportsResponse{
		Reports: reports,
	}, nil
}

func (h *GRPCHandler) RecordEvent(ctx context.Context, req *RecordEventRequest) (*RecordEventResponse, error) {
	h.logger.Info("gRPC RecordEvent called", map[string]interface{}{"event_data": req.Event})
	h.monitoring.RecordGRPCRequest("RecordEvent", "success", 0)

	err := h.service.RecordEvent(req.Event)
	if err != nil {
		h.logger.Error("Failed to record event", map[string]interface{}{"error": err.Error(), "event": req.Event})
		return &RecordEventResponse{
			Success: false,
			Message: "Failed to record event: " + err.Error(),
		}, nil
	}

	return &RecordEventResponse{
		Success: true,
		Message: "Event recorded successfully",
	}, nil
}
