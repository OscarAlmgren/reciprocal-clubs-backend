package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	"reciprocal-clubs-backend/services/governance-service/internal/models"
	"reciprocal-clubs-backend/services/governance-service/internal/service"
)

// GRPCHandler handles gRPC requests for governance service
type GRPCHandler struct {
	service    *service.Service
	logger     logging.Logger
	monitoring *monitoring.Monitor
}

// NewGRPCHandler creates a new gRPC handler
func NewGRPCHandler(service *service.Service, logger logging.Logger, monitoring *monitoring.Monitor) *GRPCHandler {
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
	// pb.RegisterGovernanceServiceServer(server, h)

	h.logger.Info("gRPC services registered", map[string]interface{}{
		"service": "governance-service",
	})
}

// gRPC method implementations for governance service

// Health check method
func (h *GRPCHandler) Check(ctx context.Context, req *HealthCheckRequest) (*HealthCheckResponse, error) {
	h.monitoring.RecordBusinessEvent("grpc_health_check", "governance")

	// Check service health
	if err := h.service.HealthCheck(ctx); err != nil {
		return &HealthCheckResponse{
			Status: "NOT_SERVING",
		}, nil
	}

	return &HealthCheckResponse{
		Status: "SERVING",
	}, nil
}

// Proposal methods

// CreateProposal creates a new governance proposal
func (h *GRPCHandler) CreateProposal(ctx context.Context, req *CreateProposalRequest) (*models.Proposal, error) {
	h.monitoring.RecordBusinessEvent("grpc_create_proposal", "governance")

	h.logger.Info("gRPC CreateProposal called", map[string]interface{}{
		"club_id": req.ClubID,
		"title":   req.Title,
	})

	// Convert gRPC request to service request
	serviceReq := &service.CreateProposalRequest{
		ClubID:           uint(req.ClubID),
		Title:            req.Title,
		Description:      req.Description,
		Type:             models.ProposalType(req.Type),
		ProposerID:       uint(req.ProposerID),
		VotingMethod:     models.VotingMethod(req.VotingMethod),
		QuorumRequired:   int(req.QuorumRequired),
		MajorityRequired: int(req.MajorityRequired),
		Metadata:         req.Metadata,
	}

	// Convert timestamps if provided
	if req.VotingStartTime != nil {
		serviceReq.VotingStartTime = req.VotingStartTime.AsTime()
	}
	if req.VotingEndTime != nil {
		serviceReq.VotingEndTime = req.VotingEndTime.AsTime()
	}

	proposal, err := h.service.CreateProposal(ctx, serviceReq)
	if err != nil {
		h.logger.Error("Failed to create proposal via gRPC", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, status.Errorf(codes.Internal, "failed to create proposal: %v", err)
	}

	return proposal, nil
}

// GetProposal retrieves a proposal by ID
func (h *GRPCHandler) GetProposal(ctx context.Context, req *GetProposalRequest) (*models.Proposal, error) {
	h.monitoring.RecordBusinessEvent("grpc_get_proposal", "governance")

	proposal, err := h.service.GetProposal(ctx, uint(req.ID))
	if err != nil {
		h.logger.Error("Failed to get proposal via gRPC", map[string]interface{}{
			"error": err.Error(),
			"id":    req.ID,
		})
		return nil, status.Errorf(codes.NotFound, "proposal not found: %v", err)
	}

	return proposal, nil
}

// ActivateProposal activates a proposal for voting
func (h *GRPCHandler) ActivateProposal(ctx context.Context, req *ActivateProposalRequest) (*models.Proposal, error) {
	h.monitoring.RecordBusinessEvent("grpc_activate_proposal", "governance")

	proposal, err := h.service.ActivateProposal(ctx, uint(req.ProposalID), uint(req.ActivatorID))
	if err != nil {
		h.logger.Error("Failed to activate proposal via gRPC", map[string]interface{}{
			"error":        err.Error(),
			"proposal_id":  req.ProposalID,
			"activator_id": req.ActivatorID,
		})
		return nil, status.Errorf(codes.FailedPrecondition, "failed to activate proposal: %v", err)
	}

	return proposal, nil
}

// GetProposalsByClub retrieves proposals for a club
func (h *GRPCHandler) GetProposalsByClub(ctx context.Context, req *GetProposalsByClubRequest) (*GetProposalsByClubResponse, error) {
	h.monitoring.RecordBusinessEvent("grpc_get_proposals_by_club", "governance")

	proposals, err := h.service.GetProposalsByClub(ctx, uint(req.ClubID))
	if err != nil {
		h.logger.Error("Failed to get proposals by club via gRPC", map[string]interface{}{
			"error":   err.Error(),
			"club_id": req.ClubID,
		})
		return nil, status.Errorf(codes.Internal, "failed to get proposals: %v", err)
	}

	return &GetProposalsByClubResponse{
		Proposals: proposals,
	}, nil
}

// Vote methods

// CastVote allows a member to vote on a proposal
func (h *GRPCHandler) CastVote(ctx context.Context, req *CastVoteRequest) (*models.Vote, error) {
	h.monitoring.RecordBusinessEvent("grpc_cast_vote", "governance")

	h.logger.Info("gRPC CastVote called", map[string]interface{}{
		"proposal_id": req.ProposalID,
		"member_id":   req.MemberID,
		"choice":      req.Choice,
	})

	// Convert gRPC request to service request
	serviceReq := &service.CastVoteRequest{
		ProposalID: uint(req.ProposalID),
		MemberID:   uint(req.MemberID),
		Choice:     models.VoteChoice(req.Choice),
		Reason:     req.Reason,
		Metadata:   req.Metadata,
	}

	vote, err := h.service.CastVote(ctx, serviceReq)
	if err != nil {
		h.logger.Error("Failed to cast vote via gRPC", map[string]interface{}{
			"error":       err.Error(),
			"proposal_id": req.ProposalID,
			"member_id":   req.MemberID,
		})
		return nil, status.Errorf(codes.FailedPrecondition, "failed to cast vote: %v", err)
	}

	return vote, nil
}

// Voting Rights methods

// CreateVotingRights creates voting rights for a member
func (h *GRPCHandler) CreateVotingRights(ctx context.Context, req *CreateVotingRightsRequest) (*models.VotingRights, error) {
	h.monitoring.RecordBusinessEvent("grpc_create_voting_rights", "governance")

	// Convert gRPC request to service request
	serviceReq := &service.CreateVotingRightsRequest{
		MemberID:     uint(req.MemberID),
		ClubID:       uint(req.ClubID),
		CanVote:      req.CanVote,
		CanPropose:   req.CanPropose,
		VotingWeight: req.VotingWeight,
		Role:         req.Role,
		Restrictions: req.Restrictions,
		Metadata:     req.Metadata,
	}

	// Convert timestamps if provided
	if req.EffectiveFrom != nil {
		serviceReq.EffectiveFrom = req.EffectiveFrom.AsTime()
	}
	if req.EffectiveUntil != nil {
		effectiveUntil := req.EffectiveUntil.AsTime()
		serviceReq.EffectiveUntil = &effectiveUntil
	}

	votingRights, err := h.service.CreateVotingRights(ctx, serviceReq)
	if err != nil {
		h.logger.Error("Failed to create voting rights via gRPC", map[string]interface{}{
			"error":     err.Error(),
			"member_id": req.MemberID,
			"club_id":   req.ClubID,
		})
		return nil, status.Errorf(codes.Internal, "failed to create voting rights: %v", err)
	}

	return votingRights, nil
}

// GetVotingRights retrieves voting rights for a member
func (h *GRPCHandler) GetVotingRights(ctx context.Context, req *GetVotingRightsRequest) (*models.VotingRights, error) {
	h.monitoring.RecordBusinessEvent("grpc_get_voting_rights", "governance")

	votingRights, err := h.service.GetVotingRights(ctx, uint(req.MemberID), uint(req.ClubID))
	if err != nil {
		h.logger.Error("Failed to get voting rights via gRPC", map[string]interface{}{
			"error":     err.Error(),
			"member_id": req.MemberID,
			"club_id":   req.ClubID,
		})
		return nil, status.Errorf(codes.NotFound, "voting rights not found: %v", err)
	}

	return votingRights, nil
}

// Governance Policy methods

// CreateGovernancePolicy creates a new governance policy
func (h *GRPCHandler) CreateGovernancePolicy(ctx context.Context, req *CreateGovernancePolicyRequest) (*models.GovernancePolicy, error) {
	h.monitoring.RecordBusinessEvent("grpc_create_governance_policy", "governance")

	// Convert gRPC request to service request
	serviceReq := &service.CreateGovernancePolicyRequest{
		ClubID:      uint(req.ClubID),
		Name:        req.Name,
		Description: req.Description,
		PolicyType:  req.PolicyType,
		Rules:       req.Rules,
		IsActive:    req.IsActive,
		CreatedBy:   uint(req.CreatedBy),
		Metadata:    req.Metadata,
	}

	// Convert timestamps if provided
	if req.EffectiveFrom != nil {
		serviceReq.EffectiveFrom = req.EffectiveFrom.AsTime()
	}
	if req.EffectiveUntil != nil {
		effectiveUntil := req.EffectiveUntil.AsTime()
		serviceReq.EffectiveUntil = &effectiveUntil
	}

	policy, err := h.service.CreateGovernancePolicy(ctx, serviceReq)
	if err != nil {
		h.logger.Error("Failed to create governance policy via gRPC", map[string]interface{}{
			"error":   err.Error(),
			"name":    req.Name,
			"club_id": req.ClubID,
		})
		return nil, status.Errorf(codes.Internal, "failed to create governance policy: %v", err)
	}

	return policy, nil
}

// GetActiveGovernancePolicies retrieves active policies for a club
func (h *GRPCHandler) GetActiveGovernancePolicies(ctx context.Context, req *GetActiveGovernancePoliciesRequest) (*GetActiveGovernancePoliciesResponse, error) {
	h.monitoring.RecordBusinessEvent("grpc_get_active_governance_policies", "governance")

	policies, err := h.service.GetActiveGovernancePolicies(ctx, uint(req.ClubID))
	if err != nil {
		h.logger.Error("Failed to get active governance policies via gRPC", map[string]interface{}{
			"error":   err.Error(),
			"club_id": req.ClubID,
		})
		return nil, status.Errorf(codes.Internal, "failed to get governance policies: %v", err)
	}

	return &GetActiveGovernancePoliciesResponse{
		Policies: policies,
	}, nil
}

// Request/Response types for gRPC
// Note: In a real implementation, these would be generated from protobuf definitions

type HealthCheckRequest struct{}

type HealthCheckResponse struct {
	Status string `json:"status"`
}

type CreateProposalRequest struct {
	ClubID           uint32                         `json:"club_id"`
	Title            string                         `json:"title"`
	Description      string                         `json:"description"`
	Type             string                         `json:"type"`
	ProposerID       uint32                         `json:"proposer_id"`
	VotingMethod     string                         `json:"voting_method"`
	QuorumRequired   int32                          `json:"quorum_required"`
	MajorityRequired int32                          `json:"majority_required"`
	VotingStartTime  *Timestamp                     `json:"voting_start_time"`
	VotingEndTime    *Timestamp                     `json:"voting_end_time"`
	Metadata         map[string]interface{}         `json:"metadata"`
}

type GetProposalRequest struct {
	ID uint32 `json:"id"`
}

type ActivateProposalRequest struct {
	ProposalID  uint32 `json:"proposal_id"`
	ActivatorID uint32 `json:"activator_id"`
}

type GetProposalsByClubRequest struct {
	ClubID uint32 `json:"club_id"`
}

type GetProposalsByClubResponse struct {
	Proposals []models.Proposal `json:"proposals"`
}

type CastVoteRequest struct {
	ProposalID uint32                     `json:"proposal_id"`
	MemberID   uint32                     `json:"member_id"`
	Choice     string                     `json:"choice"`
	Reason     string                     `json:"reason"`
	Metadata   map[string]interface{}     `json:"metadata"`
}

type CreateVotingRightsRequest struct {
	MemberID       uint32                     `json:"member_id"`
	ClubID         uint32                     `json:"club_id"`
	CanVote        bool                       `json:"can_vote"`
	CanPropose     bool                       `json:"can_propose"`
	VotingWeight   float64                    `json:"voting_weight"`
	Role           string                     `json:"role"`
	Restrictions   []string                   `json:"restrictions"`
	Metadata       map[string]interface{}     `json:"metadata"`
	EffectiveFrom  *Timestamp                 `json:"effective_from"`
	EffectiveUntil *Timestamp                 `json:"effective_until"`
}

type GetVotingRightsRequest struct {
	MemberID uint32 `json:"member_id"`
	ClubID   uint32 `json:"club_id"`
}

type CreateGovernancePolicyRequest struct {
	ClubID         uint32                     `json:"club_id"`
	Name           string                     `json:"name"`
	Description    string                     `json:"description"`
	PolicyType     string                     `json:"policy_type"`
	Rules          map[string]interface{}     `json:"rules"`
	IsActive       bool                       `json:"is_active"`
	EffectiveFrom  *Timestamp                 `json:"effective_from"`
	EffectiveUntil *Timestamp                 `json:"effective_until"`
	CreatedBy      uint32                     `json:"created_by"`
	Metadata       map[string]interface{}     `json:"metadata"`
}

type GetActiveGovernancePoliciesRequest struct {
	ClubID uint32 `json:"club_id"`
}

type GetActiveGovernancePoliciesResponse struct {
	Policies []models.GovernancePolicy `json:"policies"`
}

// Timestamp represents a timestamp (placeholder for protobuf Timestamp)
type Timestamp struct {
	Seconds int64 `json:"seconds"`
	Nanos   int32 `json:"nanos"`
}

// AsTime converts Timestamp to time.Time (placeholder implementation)
func (t *Timestamp) AsTime() time.Time {
	// This would be implemented properly with actual protobuf timestamps
	return time.Unix(t.Seconds, int64(t.Nanos))
}