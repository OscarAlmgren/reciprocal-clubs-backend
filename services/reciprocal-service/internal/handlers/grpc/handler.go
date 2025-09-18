package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	"reciprocal-clubs-backend/services/reciprocal-service/internal/models"
	"reciprocal-clubs-backend/services/reciprocal-service/internal/service"
)

// GRPCHandler handles gRPC requests for reciprocal service
type GRPCHandler struct {
	service    *service.ReciprocalService
	logger     logging.Logger
	monitoring *monitoring.Monitor
}

// NewGRPCHandler creates a new gRPC handler
func NewGRPCHandler(service *service.ReciprocalService, logger logging.Logger, monitoring *monitoring.Monitor) *GRPCHandler {
	return &GRPCHandler{
		service:    service,
		logger:     logger,
		monitoring: monitoring,
	}
}

// RegisterServices registers gRPC services
func (h *GRPCHandler) RegisterServices(server *grpc.Server) {
	// Register your gRPC services here
	// Example:
	// pb.RegisterReciprocalServiceServer(server, h)

	h.logger.Info("gRPC services registered", map[string]interface{}{
		"service": "reciprocal-service",
	})
}

// gRPC Request/Response types
// These would normally be generated from protobuf files

type CreateAgreementRequest struct {
	ProposingClubID uint                  `json:"proposing_club_id"`
	TargetClubID    uint                  `json:"target_club_id"`
	Title           string                `json:"title"`
	Description     string                `json:"description"`
	Terms           models.AgreementTerms `json:"terms"`
	ProposedByID    string                `json:"proposed_by_id"`
}

type GetAgreementRequest struct {
	ID uint `json:"id"`
}

type GetAgreementsByClubRequest struct {
	ClubID uint `json:"club_id"`
}

type UpdateAgreementStatusRequest struct {
	ID           uint   `json:"id"`
	Status       string `json:"status"`
	ReviewedByID string `json:"reviewed_by_id"`
}

type RequestVisitRequest struct {
	AgreementID    uint      `json:"agreement_id"`
	MemberID       uint      `json:"member_id"`
	VisitingClubID uint      `json:"visiting_club_id"`
	HomeClubID     uint      `json:"home_club_id"`
	VisitDate      time.Time `json:"visit_date"`
	Purpose        string    `json:"purpose"`
	GuestCount     int       `json:"guest_count"`
	EstimatedCost  float64   `json:"estimated_cost"`
	Currency       string    `json:"currency"`
}

type GetVisitRequest struct {
	ID uint `json:"id"`
}

type ConfirmVisitRequest struct {
	ID            uint   `json:"id"`
	ConfirmedByID string `json:"confirmed_by_id"`
}

type CheckInVisitRequest struct {
	VerificationCode string `json:"verification_code"`
}

type CheckOutVisitRequest struct {
	VerificationCode string   `json:"verification_code"`
	ActualCost       *float64 `json:"actual_cost,omitempty"`
}

type GetMemberVisitsRequest struct {
	MemberID uint `json:"member_id"`
	Limit    int  `json:"limit"`
	Offset   int  `json:"offset"`
}

type GetClubVisitsRequest struct {
	ClubID uint `json:"club_id"`
	Limit  int  `json:"limit"`
	Offset int  `json:"offset"`
}

type GetMemberStatsRequest struct {
	MemberID uint `json:"member_id"`
	ClubID   uint `json:"club_id"`
	Year     int  `json:"year"`
	Month    int  `json:"month"`
}

type HealthCheckRequest struct{}

type HealthCheckResponse struct {
	Status string `json:"status"`
}

// gRPC method implementations

// Health check method
func (h *GRPCHandler) Check(ctx context.Context, req *HealthCheckRequest) (*HealthCheckResponse, error) {
	h.monitoring.RecordBusinessEvent("grpc_health_check", "reciprocal")

	h.logger.Info("gRPC health check called", nil)

	return &HealthCheckResponse{
		Status: "SERVING",
	}, nil
}

// Agreement methods

func (h *GRPCHandler) CreateAgreement(ctx context.Context, req *CreateAgreementRequest) (*models.Agreement, error) {
	h.monitoring.RecordBusinessEvent("grpc_create_agreement", "reciprocal")

	h.logger.Info("gRPC CreateAgreement called", map[string]interface{}{
		"proposing_club_id": req.ProposingClubID,
		"target_club_id":    req.TargetClubID,
	})

	// Convert to service request
	serviceReq := &service.CreateAgreementRequest{
		ProposingClubID: req.ProposingClubID,
		TargetClubID:    req.TargetClubID,
		Title:           req.Title,
		Description:     req.Description,
		Terms:           req.Terms,
		ProposedByID:    req.ProposedByID,
	}

	agreement, err := h.service.CreateAgreement(ctx, serviceReq)
	if err != nil {
		h.logger.Error("Failed to create agreement via gRPC", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, status.Errorf(codes.Internal, "failed to create agreement: %v", err)
	}

	return agreement, nil
}

func (h *GRPCHandler) GetAgreement(ctx context.Context, req *GetAgreementRequest) (*models.Agreement, error) {
	h.monitoring.RecordBusinessEvent("grpc_get_agreement", "reciprocal")

	h.logger.Info("gRPC GetAgreement called", map[string]interface{}{
		"id": req.ID,
	})

	agreement, err := h.service.GetAgreementByID(ctx, req.ID)
	if err != nil {
		h.logger.Error("Failed to get agreement via gRPC", map[string]interface{}{
			"error": err.Error(),
			"id":    req.ID,
		})
		return nil, status.Errorf(codes.NotFound, "agreement not found: %v", err)
	}

	return agreement, nil
}

func (h *GRPCHandler) GetAgreementsByClub(ctx context.Context, req *GetAgreementsByClubRequest) (*AgreementsResponse, error) {
	h.monitoring.RecordBusinessEvent("grpc_get_agreements_by_club", "reciprocal")

	h.logger.Info("gRPC GetAgreementsByClub called", map[string]interface{}{
		"club_id": req.ClubID,
	})

	agreements, err := h.service.GetAgreementsByClub(ctx, req.ClubID)
	if err != nil {
		h.logger.Error("Failed to get agreements by club via gRPC", map[string]interface{}{
			"error":   err.Error(),
			"club_id": req.ClubID,
		})
		return nil, status.Errorf(codes.Internal, "failed to get agreements: %v", err)
	}

	return &AgreementsResponse{
		Agreements: agreements,
	}, nil
}

func (h *GRPCHandler) UpdateAgreementStatus(ctx context.Context, req *UpdateAgreementStatusRequest) (*models.Agreement, error) {
	h.monitoring.RecordBusinessEvent("grpc_update_agreement_status", "reciprocal")

	h.logger.Info("gRPC UpdateAgreementStatus called", map[string]interface{}{
		"id":     req.ID,
		"status": req.Status,
	})

	agreement, err := h.service.UpdateAgreementStatus(ctx, req.ID, req.Status, req.ReviewedByID)
	if err != nil {
		h.logger.Error("Failed to update agreement status via gRPC", map[string]interface{}{
			"error": err.Error(),
			"id":    req.ID,
		})
		return nil, status.Errorf(codes.Internal, "failed to update agreement status: %v", err)
	}

	return agreement, nil
}

// Visit methods

func (h *GRPCHandler) RequestVisit(ctx context.Context, req *RequestVisitRequest) (*models.Visit, error) {
	h.monitoring.RecordBusinessEvent("grpc_request_visit", "reciprocal")

	h.logger.Info("gRPC RequestVisit called", map[string]interface{}{
		"agreement_id":     req.AgreementID,
		"member_id":        req.MemberID,
		"visiting_club_id": req.VisitingClubID,
	})

	// Convert to service request
	serviceReq := &service.RequestVisitRequest{
		AgreementID:    req.AgreementID,
		MemberID:       req.MemberID,
		VisitingClubID: req.VisitingClubID,
		HomeClubID:     req.HomeClubID,
		VisitDate:      req.VisitDate,
		Purpose:        req.Purpose,
		GuestCount:     req.GuestCount,
		EstimatedCost:  req.EstimatedCost,
		Currency:       req.Currency,
	}

	visit, err := h.service.RequestVisit(ctx, serviceReq)
	if err != nil {
		h.logger.Error("Failed to request visit via gRPC", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, status.Errorf(codes.Internal, "failed to request visit: %v", err)
	}

	return visit, nil
}

func (h *GRPCHandler) GetVisit(ctx context.Context, req *GetVisitRequest) (*models.Visit, error) {
	h.monitoring.RecordBusinessEvent("grpc_get_visit", "reciprocal")

	h.logger.Info("gRPC GetVisit called", map[string]interface{}{
		"id": req.ID,
	})

	visit, err := h.service.GetVisitByID(ctx, req.ID)
	if err != nil {
		h.logger.Error("Failed to get visit via gRPC", map[string]interface{}{
			"error": err.Error(),
			"id":    req.ID,
		})
		return nil, status.Errorf(codes.NotFound, "visit not found: %v", err)
	}

	return visit, nil
}

func (h *GRPCHandler) ConfirmVisit(ctx context.Context, req *ConfirmVisitRequest) (*models.Visit, error) {
	h.monitoring.RecordBusinessEvent("grpc_confirm_visit", "reciprocal")

	h.logger.Info("gRPC ConfirmVisit called", map[string]interface{}{
		"id":             req.ID,
		"confirmed_by":   req.ConfirmedByID,
	})

	visit, err := h.service.ConfirmVisit(ctx, req.ID, req.ConfirmedByID)
	if err != nil {
		h.logger.Error("Failed to confirm visit via gRPC", map[string]interface{}{
			"error": err.Error(),
			"id":    req.ID,
		})
		return nil, status.Errorf(codes.Internal, "failed to confirm visit: %v", err)
	}

	return visit, nil
}

func (h *GRPCHandler) CheckInVisit(ctx context.Context, req *CheckInVisitRequest) (*models.Visit, error) {
	h.monitoring.RecordBusinessEvent("grpc_checkin_visit", "reciprocal")

	h.logger.Info("gRPC CheckInVisit called", map[string]interface{}{
		"verification_code": req.VerificationCode,
	})

	visit, err := h.service.CheckInVisit(ctx, req.VerificationCode)
	if err != nil {
		h.logger.Error("Failed to check in visit via gRPC", map[string]interface{}{
			"error":             err.Error(),
			"verification_code": req.VerificationCode,
		})
		return nil, status.Errorf(codes.Internal, "failed to check in visit: %v", err)
	}

	return visit, nil
}

func (h *GRPCHandler) CheckOutVisit(ctx context.Context, req *CheckOutVisitRequest) (*models.Visit, error) {
	h.monitoring.RecordBusinessEvent("grpc_checkout_visit", "reciprocal")

	h.logger.Info("gRPC CheckOutVisit called", map[string]interface{}{
		"verification_code": req.VerificationCode,
	})

	visit, err := h.service.CheckOutVisit(ctx, req.VerificationCode, req.ActualCost)
	if err != nil {
		h.logger.Error("Failed to check out visit via gRPC", map[string]interface{}{
			"error":             err.Error(),
			"verification_code": req.VerificationCode,
		})
		return nil, status.Errorf(codes.Internal, "failed to check out visit: %v", err)
	}

	return visit, nil
}

func (h *GRPCHandler) GetMemberVisits(ctx context.Context, req *GetMemberVisitsRequest) (*VisitsResponse, error) {
	h.monitoring.RecordBusinessEvent("grpc_get_member_visits", "reciprocal")

	h.logger.Info("gRPC GetMemberVisits called", map[string]interface{}{
		"member_id": req.MemberID,
	})

	visits, err := h.service.GetMemberVisits(ctx, req.MemberID, req.Limit, req.Offset)
	if err != nil {
		h.logger.Error("Failed to get member visits via gRPC", map[string]interface{}{
			"error":     err.Error(),
			"member_id": req.MemberID,
		})
		return nil, status.Errorf(codes.Internal, "failed to get member visits: %v", err)
	}

	return &VisitsResponse{
		Visits: visits,
	}, nil
}

func (h *GRPCHandler) GetClubVisits(ctx context.Context, req *GetClubVisitsRequest) (*VisitsResponse, error) {
	h.monitoring.RecordBusinessEvent("grpc_get_club_visits", "reciprocal")

	h.logger.Info("gRPC GetClubVisits called", map[string]interface{}{
		"club_id": req.ClubID,
	})

	visits, err := h.service.GetClubVisits(ctx, req.ClubID, req.Limit, req.Offset)
	if err != nil {
		h.logger.Error("Failed to get club visits via gRPC", map[string]interface{}{
			"error":   err.Error(),
			"club_id": req.ClubID,
		})
		return nil, status.Errorf(codes.Internal, "failed to get club visits: %v", err)
	}

	return &VisitsResponse{
		Visits: visits,
	}, nil
}

func (h *GRPCHandler) GetMemberVisitStats(ctx context.Context, req *GetMemberStatsRequest) (*models.VisitStats, error) {
	h.monitoring.RecordBusinessEvent("grpc_get_member_stats", "reciprocal")

	h.logger.Info("gRPC GetMemberVisitStats called", map[string]interface{}{
		"member_id": req.MemberID,
		"club_id":   req.ClubID,
	})

	stats, err := h.service.GetMemberVisitStats(ctx, req.MemberID, req.ClubID, req.Year, req.Month)
	if err != nil {
		h.logger.Error("Failed to get member visit stats via gRPC", map[string]interface{}{
			"error":     err.Error(),
			"member_id": req.MemberID,
			"club_id":   req.ClubID,
		})
		return nil, status.Errorf(codes.Internal, "failed to get member visit stats: %v", err)
	}

	return stats, nil
}

// Response types for lists

type AgreementsResponse struct {
	Agreements []models.Agreement `json:"agreements"`
}

type VisitsResponse struct {
	Visits []models.Visit `json:"visits"`
}