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
	monitoring monitoring.Service
}

func NewGRPCHandler(service service.AnalyticsService, logger logging.Logger, monitoring monitoring.Service) *GRPCHandler {
	return &GRPCHandler{
		service:    service,
		logger:     logger,
		monitoring: monitoring,
	}
}

func (h *GRPCHandler) RegisterServices(server *grpc.Server) {
	// This would normally register the generated gRPC service
	// RegisterAnalyticsServiceServer(server, h)
	h.logger.Info("gRPC services registered for analytics-service")
}

// Implement the AnalyticsServiceServer interface
func (h *GRPCHandler) GetMetrics(ctx context.Context, req *GetMetricsRequest) (*GetMetricsResponse, error) {
	h.logger.Info("gRPC GetMetrics called")
	h.monitoring.IncrementCounter("grpc_requests_total", map[string]string{
		"method": "GetMetrics",
		"service": "analytics",
	})

	metrics, err := h.service.GetMetrics(req.ClubId, req.TimeRange)
	if err != nil {
		h.logger.Error("Failed to get metrics: " + err.Error())
		return nil, err
	}

	return &GetMetricsResponse{
		Metrics: metrics,
	}, nil
}

func (h *GRPCHandler) GetReports(ctx context.Context, req *GetReportsRequest) (*GetReportsResponse, error) {
	h.logger.Info("gRPC GetReports called")
	h.monitoring.IncrementCounter("grpc_requests_total", map[string]string{
		"method": "GetReports",
		"service": "analytics",
	})

	reports, err := h.service.GetReports(req.ClubId, req.ReportType)
	if err != nil {
		h.logger.Error("Failed to get reports: " + err.Error())
		return nil, err
	}

	return &GetReportsResponse{
		Reports: reports,
	}, nil
}

func (h *GRPCHandler) RecordEvent(ctx context.Context, req *RecordEventRequest) (*RecordEventResponse, error) {
	h.logger.Info("gRPC RecordEvent called")
	h.monitoring.IncrementCounter("grpc_requests_total", map[string]string{
		"method": "RecordEvent",
		"service": "analytics",
	})

	err := h.service.RecordEvent(req.Event)
	if err != nil {
		h.logger.Error("Failed to record event: " + err.Error())
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
