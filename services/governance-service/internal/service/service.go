package service

import (
	"context"
	"fmt"
	"time"

	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/messaging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	"reciprocal-clubs-backend/services/governance-service/internal/models"
	"reciprocal-clubs-backend/services/governance-service/internal/repository"
)

// Service handles business logic for governance
type Service struct {
	repo       *repository.Repository
	logger     logging.Logger
	messaging  messaging.MessageBus
	monitoring *monitoring.Monitor
}

// NewService creates a new governance service
func NewService(repo *repository.Repository, logger logging.Logger, messaging messaging.MessageBus, monitoring *monitoring.Monitor) *Service {
	return &Service{
		repo:       repo,
		logger:     logger,
		messaging:  messaging,
		monitoring: monitoring,
	}
}

// Proposal operations

// CreateProposal creates a new governance proposal
func (s *Service) CreateProposal(ctx context.Context, req *CreateProposalRequest) (*models.Proposal, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		s.monitoring.RecordBusinessEvent("governance_proposal_create_validation_error", "1")
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Check if proposer has rights to propose
	votingRights, err := s.repo.GetVotingRights(ctx, req.ProposerID, req.ClubID)
	if err != nil {
		s.monitoring.RecordBusinessEvent("governance_proposal_create_rights_error", "1")
		return nil, fmt.Errorf("failed to check proposer rights: %w", err)
	}

	if !votingRights.CanPropose {
		s.monitoring.RecordBusinessEvent("governance_proposal_create_unauthorized", "1")
		return nil, fmt.Errorf("proposer does not have proposal rights")
	}

	// Set default voting period if not provided
	if req.VotingStartTime.IsZero() {
		req.VotingStartTime = time.Now().Add(time.Hour) // Start voting in 1 hour
	}
	if req.VotingEndTime.IsZero() {
		req.VotingEndTime = req.VotingStartTime.Add(7 * 24 * time.Hour) // 7 days duration
	}

	proposal := &models.Proposal{
		ClubID:           req.ClubID,
		Title:            req.Title,
		Description:      req.Description,
		Type:             req.Type,
		Status:           models.ProposalStatusDraft,
		ProposerID:       req.ProposerID,
		VotingMethod:     req.VotingMethod,
		QuorumRequired:   req.QuorumRequired,
		MajorityRequired: req.MajorityRequired,
		VotingStartTime:  req.VotingStartTime,
		VotingEndTime:    req.VotingEndTime,
		Metadata:         req.Metadata,
	}

	// Validate proposal business rules
	if err := proposal.Validate(); err != nil {
		s.monitoring.RecordBusinessEvent("governance_proposal_create_business_validation_error", "1")
		return nil, fmt.Errorf("proposal validation failed: %w", err)
	}

	if err := s.repo.CreateProposal(ctx, proposal); err != nil {
		s.monitoring.RecordBusinessEvent("governance_proposal_create_error", "1")
		return nil, fmt.Errorf("failed to create proposal: %w", err)
	}

	s.monitoring.RecordBusinessEvent("governance_proposal_created", "1")

	s.logger.Info("Proposal created successfully", map[string]interface{}{
		"proposal_id": proposal.ID,
		"title":       proposal.Title,
		"club_id":     proposal.ClubID,
		"proposer_id": proposal.ProposerID,
	})

	// Send notification event
	s.messaging.Publish(ctx, "governance.proposal.created", map[string]interface{}{
		"proposal_id": proposal.ID,
		"club_id":     proposal.ClubID,
		"title":       proposal.Title,
		"proposer_id": proposal.ProposerID,
	})

	return proposal, nil
}

// ActivateProposal activates a proposal for voting
func (s *Service) ActivateProposal(ctx context.Context, proposalID uint, activatorID uint) (*models.Proposal, error) {
	proposal, err := s.repo.GetProposal(ctx, proposalID)
	if err != nil {
		s.monitoring.RecordBusinessEvent("governance_proposal_activate_not_found", "1")
		return nil, fmt.Errorf("proposal not found: %w", err)
	}

	// Check if activator has rights (proposer or admin)
	if proposal.ProposerID != activatorID {
		votingRights, err := s.repo.GetVotingRights(ctx, activatorID, proposal.ClubID)
		if err != nil || !votingRights.CanPropose {
			s.monitoring.RecordBusinessEvent("governance_proposal_activate_unauthorized", "1")
			return nil, fmt.Errorf("unauthorized to activate proposal")
		}
	}

	// Check if proposal can be activated
	if !proposal.CanTransitionTo(models.ProposalStatusActive) {
		s.monitoring.RecordBusinessEvent("governance_proposal_activate_invalid_status", "1")
		return nil, fmt.Errorf("proposal cannot be activated from current status: %s", proposal.Status)
	}

	// Update proposal status
	proposal.Status = models.ProposalStatusActive
	if err := s.repo.UpdateProposal(ctx, proposal); err != nil {
		s.monitoring.RecordBusinessEvent("governance_proposal_activate_error", "1")
		return nil, fmt.Errorf("failed to activate proposal: %w", err)
	}

	// Create voting period
	votingPeriod := &models.VotingPeriod{
		ProposalID: proposal.ID,
		ClubID:     proposal.ClubID,
		StartTime:  proposal.VotingStartTime,
		EndTime:    proposal.VotingEndTime,
		IsActive:   true,
	}

	if err := s.repo.CreateVotingPeriod(ctx, votingPeriod); err != nil {
		s.monitoring.RecordBusinessEvent("governance_voting_period_create_error", "1")
		return nil, fmt.Errorf("failed to create voting period: %w", err)
	}

	s.monitoring.RecordBusinessEvent("governance_proposal_activated", "1")

	s.logger.Info("Proposal activated for voting", map[string]interface{}{
		"proposal_id": proposal.ID,
		"activator_id": activatorID,
	})

	// Send notification event
	s.messaging.Publish(ctx, "governance.proposal.activated", map[string]interface{}{
		"proposal_id":       proposal.ID,
		"club_id":           proposal.ClubID,
		"title":             proposal.Title,
		"voting_start_time": proposal.VotingStartTime,
		"voting_end_time":   proposal.VotingEndTime,
	})

	return proposal, nil
}

// GetProposal retrieves a proposal by ID
func (s *Service) GetProposal(ctx context.Context, id uint) (*models.Proposal, error) {
	proposal, err := s.repo.GetProposal(ctx, id)
	if err != nil {
		s.monitoring.RecordBusinessEvent("governance_proposal_get_error", "1")
		return nil, err
	}

	return proposal, nil
}

// GetProposalsByClub retrieves all proposals for a club
func (s *Service) GetProposalsByClub(ctx context.Context, clubID uint) ([]models.Proposal, error) {
	proposals, err := s.repo.GetProposalsByClub(ctx, clubID)
	if err != nil {
		s.monitoring.RecordBusinessEvent("governance_proposals_list_error", "1")
		return nil, err
	}

	return proposals, nil
}

// GetActiveProposals retrieves active proposals for voting
func (s *Service) GetActiveProposals(ctx context.Context, clubID uint) ([]models.Proposal, error) {
	proposals, err := s.repo.GetProposalsByStatus(ctx, clubID, models.ProposalStatusActive)
	if err != nil {
		s.monitoring.RecordBusinessEvent("governance_active_proposals_get_error", "1")
		return nil, err
	}

	return proposals, nil
}

// Vote operations

// CastVote allows a member to vote on a proposal
func (s *Service) CastVote(ctx context.Context, req *CastVoteRequest) (*models.Vote, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		s.monitoring.RecordBusinessEvent("governance_vote_cast_validation_error", "1")
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Get proposal
	proposal, err := s.repo.GetProposal(ctx, req.ProposalID)
	if err != nil {
		s.monitoring.RecordBusinessEvent("governance_vote_cast_proposal_not_found", "1")
		return nil, fmt.Errorf("proposal not found: %w", err)
	}

	// Check if voting is active
	if !proposal.IsVotingActive() {
		s.monitoring.RecordBusinessEvent("governance_vote_cast_voting_inactive", "1")
		return nil, fmt.Errorf("voting is not active for this proposal")
	}

	// Check voting rights
	votingRights, err := s.repo.GetVotingRights(ctx, req.MemberID, proposal.ClubID)
	if err != nil {
		s.monitoring.RecordBusinessEvent("governance_vote_cast_rights_error", "1")
		return nil, fmt.Errorf("failed to check voting rights: %w", err)
	}

	if !votingRights.CanMemberVote() {
		s.monitoring.RecordBusinessEvent("governance_vote_cast_unauthorized", "1")
		return nil, fmt.Errorf("member does not have voting rights")
	}

	// Create vote
	vote := &models.Vote{
		ProposalID: req.ProposalID,
		MemberID:   req.MemberID,
		ClubID:     proposal.ClubID,
		Choice:     req.Choice,
		Weight:     votingRights.VotingWeight,
		Reason:     req.Reason,
		Metadata:   req.Metadata,
	}

	if err := s.repo.CreateVote(ctx, vote); err != nil {
		s.monitoring.RecordBusinessEvent("governance_vote_cast_error", "1")
		return nil, fmt.Errorf("failed to cast vote: %w", err)
	}

	s.monitoring.RecordBusinessEvent("governance_vote_cast", "1")

	s.logger.Info("Vote cast successfully", map[string]interface{}{
		"vote_id":     vote.ID,
		"proposal_id": vote.ProposalID,
		"member_id":   vote.MemberID,
		"choice":      vote.Choice,
	})

	// Update vote results
	go s.UpdateVoteResults(context.Background(), req.ProposalID)

	// Send notification event
	s.messaging.Publish(ctx, "governance.vote.cast", map[string]interface{}{
		"vote_id":     vote.ID,
		"proposal_id": vote.ProposalID,
		"member_id":   vote.MemberID,
		"choice":      vote.Choice,
		"club_id":     vote.ClubID,
	})

	return vote, nil
}

// UpdateVoteResults calculates and updates vote results for a proposal
func (s *Service) UpdateVoteResults(ctx context.Context, proposalID uint) error {
	// Get all votes for the proposal
	votes, err := s.repo.GetVotesByProposal(ctx, proposalID)
	if err != nil {
		s.monitoring.RecordBusinessEvent("governance_vote_results_calculation_error", "1")
		return fmt.Errorf("failed to get votes: %w", err)
	}

	// Get proposal
	proposal, err := s.repo.GetProposal(ctx, proposalID)
	if err != nil {
		return fmt.Errorf("failed to get proposal: %w", err)
	}

	// Calculate results
	result := &models.VoteResult{
		ProposalID: proposalID,
		ClubID:     proposal.ClubID,
	}

	for _, vote := range votes {
		result.TotalVotes++
		switch vote.Choice {
		case models.VoteChoiceYes:
			result.YesVotes++
			result.WeightedYes += vote.Weight
		case models.VoteChoiceNo:
			result.NoVotes++
			result.WeightedNo += vote.Weight
		case models.VoteChoiceAbstain:
			result.AbstainVotes++
			result.WeightedAbstain += vote.Weight
		}
		result.TotalWeight += vote.Weight
	}

	// Check quorum and majority (simplified - would need total eligible voters count)
	// For now, assume all votes count towards quorum
	totalEligibleVoters := result.TotalVotes // This should be fetched from membership service
	result.QuorumMet = result.CalculateQuorum(totalEligibleVoters, proposal.QuorumRequired)
	result.Passed = result.QuorumMet && result.CalculateMajority(proposal.VotingMethod, proposal.MajorityRequired)

	if err := s.repo.CreateOrUpdateVoteResult(ctx, result); err != nil {
		s.monitoring.RecordBusinessEvent("governance_vote_results_save_error", "1")
		return fmt.Errorf("failed to save vote results: %w", err)
	}

	s.monitoring.RecordBusinessEvent("governance_vote_results_updated", "1")

	s.logger.Info("Vote results updated", map[string]interface{}{
		"proposal_id":  proposalID,
		"total_votes":  result.TotalVotes,
		"yes_votes":    result.YesVotes,
		"no_votes":     result.NoVotes,
		"quorum_met":   result.QuorumMet,
		"passed":       result.Passed,
	})

	return nil
}

// FinalizeProposal finalizes a proposal when voting period ends
func (s *Service) FinalizeProposal(ctx context.Context, proposalID uint) (*models.Proposal, error) {
	proposal, err := s.repo.GetProposal(ctx, proposalID)
	if err != nil {
		return nil, fmt.Errorf("proposal not found: %w", err)
	}

	// Check if proposal can be finalized
	if proposal.Status != models.ProposalStatusActive {
		return nil, fmt.Errorf("proposal is not active")
	}

	if !proposal.HasExpired() {
		return nil, fmt.Errorf("voting period has not ended")
	}

	// Get vote results
	voteResult, err := s.repo.GetVoteResult(ctx, proposalID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vote results: %w", err)
	}

	// Update proposal status based on results
	if voteResult.Passed {
		proposal.Status = models.ProposalStatusPassed
	} else {
		proposal.Status = models.ProposalStatusRejected
	}

	if err := s.repo.UpdateProposal(ctx, proposal); err != nil {
		s.monitoring.RecordBusinessEvent("governance_proposal_finalize_error", "1")
		return nil, fmt.Errorf("failed to finalize proposal: %w", err)
	}

	// Deactivate voting period
	votingPeriod, err := s.repo.GetVotingPeriodByProposal(ctx, proposalID)
	if err == nil {
		votingPeriod.IsActive = false
		s.repo.UpdateVotingPeriod(ctx, votingPeriod)
	}

	s.monitoring.RecordBusinessEvent("governance_proposal_finalized", "1")

	s.logger.Info("Proposal finalized", map[string]interface{}{
		"proposal_id": proposal.ID,
		"status":      proposal.Status,
		"passed":      voteResult.Passed,
	})

	// Send notification event
	s.messaging.Publish(ctx, "governance.proposal.finalized", map[string]interface{}{
		"proposal_id": proposal.ID,
		"club_id":     proposal.ClubID,
		"title":       proposal.Title,
		"status":      proposal.Status,
		"passed":      voteResult.Passed,
	})

	// Record on blockchain for immutable audit trail
	s.messaging.Publish(ctx, "blockchain.governance.record", map[string]interface{}{
		"proposal_id": proposal.ID,
		"club_id":     proposal.ClubID,
		"title":       proposal.Title,
		"status":      proposal.Status,
		"vote_result": voteResult,
		"finalized_at": time.Now(),
	})

	return proposal, nil
}

// Voting Rights operations

// CreateVotingRights creates voting rights for a member
func (s *Service) CreateVotingRights(ctx context.Context, req *CreateVotingRightsRequest) (*models.VotingRights, error) {
	if err := req.Validate(); err != nil {
		s.monitoring.RecordBusinessEvent("governance_voting_rights_create_validation_error", "1")
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	votingRights := &models.VotingRights{
		MemberID:       req.MemberID,
		ClubID:         req.ClubID,
		CanVote:        req.CanVote,
		CanPropose:     req.CanPropose,
		VotingWeight:   req.VotingWeight,
		Role:           req.Role,
		Restrictions:   req.Restrictions,
		Metadata:       req.Metadata,
		EffectiveFrom:  req.EffectiveFrom,
		EffectiveUntil: req.EffectiveUntil,
	}

	if err := s.repo.CreateVotingRights(ctx, votingRights); err != nil {
		s.monitoring.RecordBusinessEvent("governance_voting_rights_create_error", "1")
		return nil, fmt.Errorf("failed to create voting rights: %w", err)
	}

	s.monitoring.RecordBusinessEvent("governance_voting_rights_created", "1")

	s.logger.Info("Voting rights created", map[string]interface{}{
		"rights_id": votingRights.ID,
		"member_id": votingRights.MemberID,
		"club_id":   votingRights.ClubID,
		"can_vote":  votingRights.CanVote,
	})

	return votingRights, nil
}

// GetVotingRights retrieves voting rights for a member
func (s *Service) GetVotingRights(ctx context.Context, memberID, clubID uint) (*models.VotingRights, error) {
	votingRights, err := s.repo.GetVotingRights(ctx, memberID, clubID)
	if err != nil {
		s.monitoring.RecordBusinessEvent("governance_voting_rights_get_error", "1")
		return nil, err
	}

	return votingRights, nil
}

// Governance Policy operations

// CreateGovernancePolicy creates a new governance policy
func (s *Service) CreateGovernancePolicy(ctx context.Context, req *CreateGovernancePolicyRequest) (*models.GovernancePolicy, error) {
	if err := req.Validate(); err != nil {
		s.monitoring.RecordBusinessEvent("governance_policy_create_validation_error", "1")
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	policy := &models.GovernancePolicy{
		ClubID:        req.ClubID,
		Name:          req.Name,
		Description:   req.Description,
		PolicyType:    req.PolicyType,
		Rules:         req.Rules,
		IsActive:      req.IsActive,
		EffectiveFrom: req.EffectiveFrom,
		EffectiveUntil: req.EffectiveUntil,
		CreatedBy:     req.CreatedBy,
		Metadata:      req.Metadata,
	}

	if err := s.repo.CreateGovernancePolicy(ctx, policy); err != nil {
		s.monitoring.RecordBusinessEvent("governance_policy_create_error", "1")
		return nil, fmt.Errorf("failed to create governance policy: %w", err)
	}

	s.monitoring.RecordBusinessEvent("governance_policy_created", "1")

	s.logger.Info("Governance policy created", map[string]interface{}{
		"policy_id": policy.ID,
		"name":      policy.Name,
		"club_id":   policy.ClubID,
	})

	return policy, nil
}

// GetActiveGovernancePolicies retrieves active policies for a club
func (s *Service) GetActiveGovernancePolicies(ctx context.Context, clubID uint) ([]models.GovernancePolicy, error) {
	policies, err := s.repo.GetActiveGovernancePolicies(ctx, clubID)
	if err != nil {
		s.monitoring.RecordBusinessEvent("governance_policies_get_error", "1")
		return nil, err
	}

	return policies, nil
}

// Health check
func (s *Service) HealthCheck(ctx context.Context) error {
	return s.repo.HealthCheck(ctx)
}

// Request/Response types

type CreateProposalRequest struct {
	ClubID           uint                   `json:"club_id" validate:"required"`
	Title            string                 `json:"title" validate:"required,min=5,max=255"`
	Description      string                 `json:"description" validate:"required,min=10"`
	Type             models.ProposalType    `json:"type" validate:"required"`
	ProposerID       uint                   `json:"proposer_id" validate:"required"`
	VotingMethod     models.VotingMethod    `json:"voting_method"`
	QuorumRequired   int                    `json:"quorum_required"`
	MajorityRequired int                    `json:"majority_required"`
	VotingStartTime  time.Time              `json:"voting_start_time"`
	VotingEndTime    time.Time              `json:"voting_end_time"`
	Metadata         map[string]interface{} `json:"metadata"`
}

func (r *CreateProposalRequest) Validate() error {
	if r.ClubID == 0 {
		return fmt.Errorf("club_id is required")
	}
	if len(r.Title) < 5 || len(r.Title) > 255 {
		return fmt.Errorf("title must be between 5 and 255 characters")
	}
	if len(r.Description) < 10 {
		return fmt.Errorf("description must be at least 10 characters")
	}
	if r.ProposerID == 0 {
		return fmt.Errorf("proposer_id is required")
	}
	if r.QuorumRequired < 0 || r.QuorumRequired > 100 {
		return fmt.Errorf("quorum_required must be between 0 and 100")
	}
	if r.MajorityRequired < 0 || r.MajorityRequired > 100 {
		return fmt.Errorf("majority_required must be between 0 and 100")
	}
	return nil
}

type CastVoteRequest struct {
	ProposalID uint                   `json:"proposal_id" validate:"required"`
	MemberID   uint                   `json:"member_id" validate:"required"`
	Choice     models.VoteChoice      `json:"choice" validate:"required"`
	Reason     string                 `json:"reason"`
	Metadata   map[string]interface{} `json:"metadata"`
}

func (r *CastVoteRequest) Validate() error {
	if r.ProposalID == 0 {
		return fmt.Errorf("proposal_id is required")
	}
	if r.MemberID == 0 {
		return fmt.Errorf("member_id is required")
	}
	if r.Choice != models.VoteChoiceYes && r.Choice != models.VoteChoiceNo && r.Choice != models.VoteChoiceAbstain {
		return fmt.Errorf("invalid vote choice")
	}
	return nil
}

type CreateVotingRightsRequest struct {
	MemberID       uint                   `json:"member_id" validate:"required"`
	ClubID         uint                   `json:"club_id" validate:"required"`
	CanVote        bool                   `json:"can_vote"`
	CanPropose     bool                   `json:"can_propose"`
	VotingWeight   float64                `json:"voting_weight"`
	Role           string                 `json:"role"`
	Restrictions   []string               `json:"restrictions"`
	Metadata       map[string]interface{} `json:"metadata"`
	EffectiveFrom  time.Time              `json:"effective_from"`
	EffectiveUntil *time.Time             `json:"effective_until"`
}

func (r *CreateVotingRightsRequest) Validate() error {
	if r.MemberID == 0 {
		return fmt.Errorf("member_id is required")
	}
	if r.ClubID == 0 {
		return fmt.Errorf("club_id is required")
	}
	if r.VotingWeight < 0 {
		return fmt.Errorf("voting_weight cannot be negative")
	}
	return nil
}

type CreateGovernancePolicyRequest struct {
	ClubID         uint                   `json:"club_id" validate:"required"`
	Name           string                 `json:"name" validate:"required,min=3,max=255"`
	Description    string                 `json:"description"`
	PolicyType     string                 `json:"policy_type" validate:"required"`
	Rules          map[string]interface{} `json:"rules" validate:"required"`
	IsActive       bool                   `json:"is_active"`
	EffectiveFrom  time.Time              `json:"effective_from"`
	EffectiveUntil *time.Time             `json:"effective_until"`
	CreatedBy      uint                   `json:"created_by" validate:"required"`
	Metadata       map[string]interface{} `json:"metadata"`
}

func (r *CreateGovernancePolicyRequest) Validate() error {
	if r.ClubID == 0 {
		return fmt.Errorf("club_id is required")
	}
	if len(r.Name) < 3 || len(r.Name) > 255 {
		return fmt.Errorf("name must be between 3 and 255 characters")
	}
	if r.CreatedBy == 0 {
		return fmt.Errorf("created_by is required")
	}
	if len(r.Rules) == 0 {
		return fmt.Errorf("rules are required")
	}
	return nil
}