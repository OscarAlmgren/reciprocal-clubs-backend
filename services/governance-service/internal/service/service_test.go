package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"gorm.io/gorm"

	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/messaging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	"reciprocal-clubs-backend/services/governance-service/internal/models"
)

// Mock implementations for testing
type mockRepository struct {
	proposals     map[uint]*models.Proposal
	votes         map[uint]*models.Vote
	votingRights  map[string]*models.VotingRights // key: "memberID:clubID"
	policies      map[uint]*models.GovernancePolicy
	voteResults   map[uint]*models.VoteResult
	votingPeriods map[uint]*models.VotingPeriod
	nextID        uint
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		proposals:     make(map[uint]*models.Proposal),
		votes:         make(map[uint]*models.Vote),
		votingRights:  make(map[string]*models.VotingRights),
		policies:      make(map[uint]*models.GovernancePolicy),
		voteResults:   make(map[uint]*models.VoteResult),
		votingPeriods: make(map[uint]*models.VotingPeriod),
		nextID:        1,
	}
}

func (r *mockRepository) CreateProposal(ctx context.Context, proposal *models.Proposal) error {
	proposal.ID = r.nextID
	r.nextID++
	r.proposals[proposal.ID] = proposal
	return nil
}

func (r *mockRepository) GetProposal(ctx context.Context, id uint) (*models.Proposal, error) {
	if proposal, exists := r.proposals[id]; exists {
		return proposal, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *mockRepository) GetProposalsByClub(ctx context.Context, clubID uint) ([]models.Proposal, error) {
	var proposals []models.Proposal
	for _, proposal := range r.proposals {
		if proposal.ClubID == clubID {
			proposals = append(proposals, *proposal)
		}
	}
	return proposals, nil
}

func (r *mockRepository) GetProposalsByStatus(ctx context.Context, clubID uint, status models.ProposalStatus) ([]models.Proposal, error) {
	var proposals []models.Proposal
	for _, proposal := range r.proposals {
		if (clubID == 0 || proposal.ClubID == clubID) && proposal.Status == status {
			proposals = append(proposals, *proposal)
		}
	}
	return proposals, nil
}

func (r *mockRepository) UpdateProposal(ctx context.Context, proposal *models.Proposal) error {
	r.proposals[proposal.ID] = proposal
	return nil
}

func (r *mockRepository) CreateVote(ctx context.Context, vote *models.Vote) error {
	vote.ID = r.nextID
	r.nextID++
	r.votes[vote.ID] = vote
	return nil
}

func (r *mockRepository) GetVoteByMemberAndProposal(ctx context.Context, memberID, proposalID uint) (*models.Vote, error) {
	for _, vote := range r.votes {
		if vote.MemberID == memberID && vote.ProposalID == proposalID {
			return vote, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *mockRepository) GetVotesByProposal(ctx context.Context, proposalID uint) ([]models.Vote, error) {
	var votes []models.Vote
	for _, vote := range r.votes {
		if vote.ProposalID == proposalID {
			votes = append(votes, *vote)
		}
	}
	return votes, nil
}

func (r *mockRepository) CreateVotingPeriod(ctx context.Context, period *models.VotingPeriod) error {
	period.ID = r.nextID
	r.nextID++
	r.votingPeriods[period.ID] = period
	return nil
}

func (r *mockRepository) GetVotingPeriodByProposal(ctx context.Context, proposalID uint) (*models.VotingPeriod, error) {
	for _, period := range r.votingPeriods {
		if period.ProposalID == proposalID {
			return period, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *mockRepository) UpdateVotingPeriod(ctx context.Context, period *models.VotingPeriod) error {
	r.votingPeriods[period.ID] = period
	return nil
}

func (r *mockRepository) CreateVotingRights(ctx context.Context, rights *models.VotingRights) error {
	rights.ID = r.nextID
	r.nextID++
	key := fmt.Sprintf("%d:%d", rights.MemberID, rights.ClubID)
	r.votingRights[key] = rights
	return nil
}

func (r *mockRepository) GetVotingRights(ctx context.Context, memberID, clubID uint) (*models.VotingRights, error) {
	key := fmt.Sprintf("%d:%d", memberID, clubID)
	if rights, exists := r.votingRights[key]; exists {
		return rights, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *mockRepository) CreateGovernancePolicy(ctx context.Context, policy *models.GovernancePolicy) error {
	policy.ID = r.nextID
	r.nextID++
	r.policies[policy.ID] = policy
	return nil
}

func (r *mockRepository) GetActiveGovernancePolicies(ctx context.Context, clubID uint) ([]models.GovernancePolicy, error) {
	var policies []models.GovernancePolicy
	for _, policy := range r.policies {
		if policy.ClubID == clubID && policy.IsActive {
			policies = append(policies, *policy)
		}
	}
	return policies, nil
}

func (r *mockRepository) CreateOrUpdateVoteResult(ctx context.Context, result *models.VoteResult) error {
	r.voteResults[result.ProposalID] = result
	return nil
}

func (r *mockRepository) GetVoteResult(ctx context.Context, proposalID uint) (*models.VoteResult, error) {
	if result, exists := r.voteResults[proposalID]; exists {
		return result, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *mockRepository) HealthCheck(ctx context.Context) error {
	return nil
}

type mockLogger struct{}

func (m *mockLogger) Info(msg string, fields map[string]interface{})  {}
func (m *mockLogger) Error(msg string, fields map[string]interface{}) {}
func (m *mockLogger) Debug(msg string, fields map[string]interface{}) {}
func (m *mockLogger) Warn(msg string, fields map[string]interface{})  {}
func (m *mockLogger) Fatal(msg string, fields map[string]interface{}) {}
func (m *mockLogger) With(fields map[string]interface{}) logging.Logger { return m }
func (m *mockLogger) WithContext(ctx context.Context) logging.Logger { return m }

type mockMessaging struct{}

func (m *mockMessaging) Publish(ctx context.Context, topic string, message interface{}) error { return nil }
func (m *mockMessaging) Subscribe(topic string, handler interface{}) error { return nil }
func (m *mockMessaging) Close() error { return nil }
func (m *mockMessaging) HealthCheck() error { return nil }

type mockMonitoring struct{}

func (m *mockMonitoring) RecordBusinessEvent(event, value string) {}
func (m *mockMonitoring) RecordHTTPRequest(method, path string, status int, duration time.Duration) {}
func (m *mockMonitoring) StartMetricsServer() {}

// Import gorm for the error
import "gorm.io/gorm"

// Import fmt for string formatting in mock
import "fmt"

// Import required packages
import (
	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/messaging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
)

func setupTestService() (*Service, *mockRepository) {
	repo := newMockRepository()
	logger := &mockLogger{}
	messaging := &mockMessaging{}
	monitor := &mockMonitoring{}

	service := NewService(repo, logger, messaging, monitor)
	return service, repo
}

func TestService_CreateProposal(t *testing.T) {
	service, repo := setupTestService()
	ctx := context.Background()

	// Setup voting rights for the proposer
	votingRights := &models.VotingRights{
		MemberID:      1,
		ClubID:        1,
		CanVote:       true,
		CanPropose:    true,
		VotingWeight:  1.0,
		EffectiveFrom: time.Now().Add(-time.Hour),
	}
	repo.CreateVotingRights(ctx, votingRights)

	tests := []struct {
		name    string
		request *CreateProposalRequest
		wantErr bool
	}{
		{
			name: "Valid proposal",
			request: &CreateProposalRequest{
				ClubID:           1,
				Title:            "Test Proposal",
				Description:      "This is a test proposal for testing purposes",
				Type:             models.ProposalTypePolicyChange,
				ProposerID:       1,
				VotingMethod:     models.VotingMethodSimpleMajority,
				QuorumRequired:   50,
				MajorityRequired: 50,
			},
			wantErr: false,
		},
		{
			name: "Empty title",
			request: &CreateProposalRequest{
				ClubID:      1,
				Title:       "",
				Description: "This is a test proposal for testing purposes",
				ProposerID:  1,
			},
			wantErr: true,
		},
		{
			name: "Empty description",
			request: &CreateProposalRequest{
				ClubID:      1,
				Title:       "Test Proposal",
				Description: "Short",
				ProposerID:  1,
			},
			wantErr: true,
		},
		{
			name: "Zero club ID",
			request: &CreateProposalRequest{
				ClubID:      0,
				Title:       "Test Proposal",
				Description: "This is a test proposal for testing purposes",
				ProposerID:  1,
			},
			wantErr: true,
		},
		{
			name: "Invalid proposer (no rights)",
			request: &CreateProposalRequest{
				ClubID:      1,
				Title:       "Test Proposal",
				Description: "This is a test proposal for testing purposes",
				ProposerID:  999, // No voting rights exist for this member
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proposal, err := service.CreateProposal(ctx, tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.CreateProposal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && proposal == nil {
				t.Error("Service.CreateProposal() returned nil proposal without error")
			}
			if !tt.wantErr && proposal.Status != models.ProposalStatusDraft {
				t.Errorf("Service.CreateProposal() proposal status = %v, want %v", proposal.Status, models.ProposalStatusDraft)
			}
		})
	}
}

func TestService_ActivateProposal(t *testing.T) {
	service, repo := setupTestService()
	ctx := context.Background()

	// Setup voting rights
	votingRights := &models.VotingRights{
		MemberID:      1,
		ClubID:        1,
		CanVote:       true,
		CanPropose:    true,
		VotingWeight:  1.0,
		EffectiveFrom: time.Now().Add(-time.Hour),
	}
	repo.CreateVotingRights(ctx, votingRights)

	// Create a draft proposal
	proposal := &models.Proposal{
		ID:               1,
		ClubID:           1,
		Title:            "Test Proposal",
		Description:      "Test Description",
		Status:           models.ProposalStatusDraft,
		ProposerID:       1,
		VotingStartTime:  time.Now().Add(time.Hour),
		VotingEndTime:    time.Now().Add(24 * time.Hour),
		QuorumRequired:   50,
		MajorityRequired: 50,
	}
	repo.CreateProposal(ctx, proposal)

	tests := []struct {
		name        string
		proposalID  uint
		activatorID uint
		wantErr     bool
	}{
		{
			name:        "Valid activation by proposer",
			proposalID:  1,
			activatorID: 1,
			wantErr:     false,
		},
		{
			name:        "Proposal not found",
			proposalID:  999,
			activatorID: 1,
			wantErr:     true,
		},
		{
			name:        "Unauthorized activator",
			proposalID:  1,
			activatorID: 999,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset proposal status for each test
			if proposal, exists := repo.proposals[tt.proposalID]; exists {
				proposal.Status = models.ProposalStatusDraft
			}

			activatedProposal, err := service.ActivateProposal(ctx, tt.proposalID, tt.activatorID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.ActivateProposal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && activatedProposal.Status != models.ProposalStatusActive {
				t.Errorf("Service.ActivateProposal() proposal status = %v, want %v", activatedProposal.Status, models.ProposalStatusActive)
			}
		})
	}
}

func TestService_CastVote(t *testing.T) {
	service, repo := setupTestService()
	ctx := context.Background()

	// Setup voting rights
	votingRights := &models.VotingRights{
		MemberID:      1,
		ClubID:        1,
		CanVote:       true,
		CanPropose:    true,
		VotingWeight:  1.0,
		EffectiveFrom: time.Now().Add(-time.Hour),
	}
	repo.CreateVotingRights(ctx, votingRights)

	// Create an active proposal
	now := time.Now()
	proposal := &models.Proposal{
		ID:              1,
		ClubID:          1,
		Title:           "Test Proposal",
		Description:     "Test Description",
		Status:          models.ProposalStatusActive,
		ProposerID:      1,
		VotingStartTime: now.Add(-time.Hour),
		VotingEndTime:   now.Add(time.Hour),
	}
	repo.CreateProposal(ctx, proposal)

	tests := []struct {
		name    string
		request *CastVoteRequest
		wantErr bool
	}{
		{
			name: "Valid vote",
			request: &CastVoteRequest{
				ProposalID: 1,
				MemberID:   1,
				Choice:     models.VoteChoiceYes,
				Reason:     "I support this proposal",
			},
			wantErr: false,
		},
		{
			name: "Invalid proposal ID",
			request: &CastVoteRequest{
				ProposalID: 999,
				MemberID:   1,
				Choice:     models.VoteChoiceYes,
			},
			wantErr: true,
		},
		{
			name: "Member without voting rights",
			request: &CastVoteRequest{
				ProposalID: 1,
				MemberID:   999,
				Choice:     models.VoteChoiceYes,
			},
			wantErr: true,
		},
		{
			name: "Invalid choice",
			request: &CastVoteRequest{
				ProposalID: 1,
				MemberID:   1,
				Choice:     "invalid_choice",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vote, err := service.CastVote(ctx, tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.CastVote() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && vote == nil {
				t.Error("Service.CastVote() returned nil vote without error")
			}
			if !tt.wantErr && vote.Choice != tt.request.Choice {
				t.Errorf("Service.CastVote() vote choice = %v, want %v", vote.Choice, tt.request.Choice)
			}
		})
	}
}

func TestService_CreateVotingRights(t *testing.T) {
	service, _ := setupTestService()
	ctx := context.Background()

	tests := []struct {
		name    string
		request *CreateVotingRightsRequest
		wantErr bool
	}{
		{
			name: "Valid voting rights",
			request: &CreateVotingRightsRequest{
				MemberID:      1,
				ClubID:        1,
				CanVote:       true,
				CanPropose:    false,
				VotingWeight:  1.0,
				Role:          "member",
				EffectiveFrom: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "Zero member ID",
			request: &CreateVotingRightsRequest{
				MemberID:      0,
				ClubID:        1,
				CanVote:       true,
				VotingWeight:  1.0,
				EffectiveFrom: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "Zero club ID",
			request: &CreateVotingRightsRequest{
				MemberID:      1,
				ClubID:        0,
				CanVote:       true,
				VotingWeight:  1.0,
				EffectiveFrom: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "Negative voting weight",
			request: &CreateVotingRightsRequest{
				MemberID:      1,
				ClubID:        1,
				CanVote:       true,
				VotingWeight:  -1.0,
				EffectiveFrom: time.Now(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rights, err := service.CreateVotingRights(ctx, tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.CreateVotingRights() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && rights == nil {
				t.Error("Service.CreateVotingRights() returned nil rights without error")
			}
			if !tt.wantErr && rights.MemberID != tt.request.MemberID {
				t.Errorf("Service.CreateVotingRights() member ID = %v, want %v", rights.MemberID, tt.request.MemberID)
			}
		})
	}
}

func TestService_CreateGovernancePolicy(t *testing.T) {
	service, _ := setupTestService()
	ctx := context.Background()

	tests := []struct {
		name    string
		request *CreateGovernancePolicyRequest
		wantErr bool
	}{
		{
			name: "Valid governance policy",
			request: &CreateGovernancePolicyRequest{
				ClubID:        1,
				Name:          "Test Policy",
				Description:   "Test policy description",
				PolicyType:    "voting",
				Rules:         map[string]interface{}{"quorum": 50},
				IsActive:      true,
				CreatedBy:     1,
				EffectiveFrom: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "Empty name",
			request: &CreateGovernancePolicyRequest{
				ClubID:        1,
				Name:          "",
				PolicyType:    "voting",
				Rules:         map[string]interface{}{"quorum": 50},
				CreatedBy:     1,
				EffectiveFrom: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "Zero club ID",
			request: &CreateGovernancePolicyRequest{
				ClubID:        0,
				Name:          "Test Policy",
				PolicyType:    "voting",
				Rules:         map[string]interface{}{"quorum": 50},
				CreatedBy:     1,
				EffectiveFrom: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "Empty rules",
			request: &CreateGovernancePolicyRequest{
				ClubID:        1,
				Name:          "Test Policy",
				PolicyType:    "voting",
				Rules:         map[string]interface{}{},
				CreatedBy:     1,
				EffectiveFrom: time.Now(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy, err := service.CreateGovernancePolicy(ctx, tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.CreateGovernancePolicy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && policy == nil {
				t.Error("Service.CreateGovernancePolicy() returned nil policy without error")
			}
			if !tt.wantErr && policy.Name != tt.request.Name {
				t.Errorf("Service.CreateGovernancePolicy() policy name = %v, want %v", policy.Name, tt.request.Name)
			}
		})
	}
}

func TestService_UpdateVoteResults(t *testing.T) {
	service, repo := setupTestService()
	ctx := context.Background()

	// Create a proposal
	proposal := &models.Proposal{
		ID:               1,
		ClubID:           1,
		VotingMethod:     models.VotingMethodSimpleMajority,
		QuorumRequired:   50,
		MajorityRequired: 50,
	}
	repo.CreateProposal(ctx, proposal)

	// Create some votes
	votes := []*models.Vote{
		{
			ID:         1,
			ProposalID: 1,
			MemberID:   1,
			Choice:     models.VoteChoiceYes,
			Weight:     1.0,
		},
		{
			ID:         2,
			ProposalID: 1,
			MemberID:   2,
			Choice:     models.VoteChoiceYes,
			Weight:     1.0,
		},
		{
			ID:         3,
			ProposalID: 1,
			MemberID:   3,
			Choice:     models.VoteChoiceNo,
			Weight:     1.0,
		},
	}

	for _, vote := range votes {
		repo.CreateVote(ctx, vote)
	}

	err := service.UpdateVoteResults(ctx, 1)
	if err != nil {
		t.Errorf("Service.UpdateVoteResults() error = %v", err)
		return
	}

	// Check if vote results were created
	result, err := repo.GetVoteResult(ctx, 1)
	if err != nil {
		t.Errorf("Failed to get vote result: %v", err)
		return
	}

	if result.TotalVotes != 3 {
		t.Errorf("UpdateVoteResults() total votes = %v, want %v", result.TotalVotes, 3)
	}
	if result.YesVotes != 2 {
		t.Errorf("UpdateVoteResults() yes votes = %v, want %v", result.YesVotes, 2)
	}
	if result.NoVotes != 1 {
		t.Errorf("UpdateVoteResults() no votes = %v, want %v", result.NoVotes, 1)
	}
}

func TestService_FinalizeProposal(t *testing.T) {
	service, repo := setupTestService()
	ctx := context.Background()

	// Create an expired active proposal
	proposal := &models.Proposal{
		ID:              1,
		ClubID:          1,
		Status:          models.ProposalStatusActive,
		VotingStartTime: time.Now().Add(-2 * time.Hour),
		VotingEndTime:   time.Now().Add(-time.Hour), // Expired
	}
	repo.CreateProposal(ctx, proposal)

	// Create vote result
	voteResult := &models.VoteResult{
		ProposalID: 1,
		ClubID:     1,
		TotalVotes: 10,
		YesVotes:   7,
		NoVotes:    3,
		QuorumMet:  true,
		Passed:     true,
	}
	repo.CreateOrUpdateVoteResult(ctx, voteResult)

	// Create voting period
	votingPeriod := &models.VotingPeriod{
		ProposalID: 1,
		ClubID:     1,
		IsActive:   true,
	}
	repo.CreateVotingPeriod(ctx, votingPeriod)

	finalizedProposal, err := service.FinalizeProposal(ctx, 1)
	if err != nil {
		t.Errorf("Service.FinalizeProposal() error = %v", err)
		return
	}

	if finalizedProposal.Status != models.ProposalStatusPassed {
		t.Errorf("FinalizeProposal() status = %v, want %v", finalizedProposal.Status, models.ProposalStatusPassed)
	}

	// Check that voting period was deactivated
	period, _ := repo.GetVotingPeriodByProposal(ctx, 1)
	if period.IsActive {
		t.Error("FinalizeProposal() should deactivate voting period")
	}
}

func TestService_HealthCheck(t *testing.T) {
	service, _ := setupTestService()
	ctx := context.Background()

	err := service.HealthCheck(ctx)
	if err != nil {
		t.Errorf("Service.HealthCheck() error = %v", err)
	}
}