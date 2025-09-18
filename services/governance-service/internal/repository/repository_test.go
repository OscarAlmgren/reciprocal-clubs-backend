package repository

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/services/governance-service/internal/models"
)

// Mock logger for testing
type mockLogger struct{}

func (m *mockLogger) Info(msg string, fields map[string]interface{})  {}
func (m *mockLogger) Error(msg string, fields map[string]interface{}) {}
func (m *mockLogger) Debug(msg string, fields map[string]interface{}) {}
func (m *mockLogger) Warn(msg string, fields map[string]interface{})  {}
func (m *mockLogger) Fatal(msg string, fields map[string]interface{}) {}
func (m *mockLogger) With(fields map[string]interface{}) logging.Logger { return m }
func (m *mockLogger) WithContext(ctx context.Context) logging.Logger { return m }

func setupTestDB(t *testing.T) (*gorm.DB, *Repository) {
	// Create in-memory SQLite database for testing
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Auto-migrate the schema
	err = db.AutoMigrate(
		&models.Proposal{},
		&models.Vote{},
		&models.VotingPeriod{},
		&models.VotingRights{},
		&models.GovernancePolicy{},
		&models.VoteResult{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Create repository
	repo := NewRepository(db, &mockLogger{})

	return db, repo
}

func TestRepository_CreateProposal(t *testing.T) {
	_, repo := setupTestDB(t)
	ctx := context.Background()

	proposal := &models.Proposal{
		ClubID:           1,
		Title:            "Test Proposal",
		Description:      "This is a test proposal",
		Type:             models.ProposalTypePolicyChange,
		Status:           models.ProposalStatusDraft,
		ProposerID:       1,
		VotingMethod:     models.VotingMethodSimpleMajority,
		QuorumRequired:   50,
		MajorityRequired: 50,
		VotingStartTime:  time.Now().Add(time.Hour),
		VotingEndTime:    time.Now().Add(24 * time.Hour),
	}

	err := repo.CreateProposal(ctx, proposal)
	if err != nil {
		t.Errorf("CreateProposal() error = %v", err)
	}

	if proposal.ID == 0 {
		t.Error("CreateProposal() did not set ID")
	}
}

func TestRepository_GetProposal(t *testing.T) {
	_, repo := setupTestDB(t)
	ctx := context.Background()

	// Create a proposal first
	proposal := &models.Proposal{
		ClubID:      1,
		Title:       "Test Proposal",
		Description: "This is a test proposal",
		Status:      models.ProposalStatusDraft,
		ProposerID:  1,
	}
	repo.CreateProposal(ctx, proposal)

	// Test getting the proposal
	retrieved, err := repo.GetProposal(ctx, proposal.ID)
	if err != nil {
		t.Errorf("GetProposal() error = %v", err)
	}

	if retrieved.Title != proposal.Title {
		t.Errorf("GetProposal() title = %v, want %v", retrieved.Title, proposal.Title)
	}

	// Test getting non-existent proposal
	_, err = repo.GetProposal(ctx, 999)
	if err == nil {
		t.Error("GetProposal() should return error for non-existent proposal")
	}
}

func TestRepository_GetProposalsByClub(t *testing.T) {
	_, repo := setupTestDB(t)
	ctx := context.Background()

	// Create proposals for different clubs
	proposals := []*models.Proposal{
		{
			ClubID:      1,
			Title:       "Club 1 Proposal 1",
			Description: "Description 1",
			Status:      models.ProposalStatusDraft,
			ProposerID:  1,
		},
		{
			ClubID:      1,
			Title:       "Club 1 Proposal 2",
			Description: "Description 2",
			Status:      models.ProposalStatusActive,
			ProposerID:  1,
		},
		{
			ClubID:      2,
			Title:       "Club 2 Proposal 1",
			Description: "Description 3",
			Status:      models.ProposalStatusDraft,
			ProposerID:  2,
		},
	}

	for _, proposal := range proposals {
		repo.CreateProposal(ctx, proposal)
	}

	// Test getting proposals for club 1
	club1Proposals, err := repo.GetProposalsByClub(ctx, 1)
	if err != nil {
		t.Errorf("GetProposalsByClub() error = %v", err)
	}

	if len(club1Proposals) != 2 {
		t.Errorf("GetProposalsByClub() count = %d, want %d", len(club1Proposals), 2)
	}

	// Test getting proposals for club 2
	club2Proposals, err := repo.GetProposalsByClub(ctx, 2)
	if err != nil {
		t.Errorf("GetProposalsByClub() error = %v", err)
	}

	if len(club2Proposals) != 1 {
		t.Errorf("GetProposalsByClub() count = %d, want %d", len(club2Proposals), 1)
	}
}

func TestRepository_GetProposalsByStatus(t *testing.T) {
	_, repo := setupTestDB(t)
	ctx := context.Background()

	// Create proposals with different statuses
	proposals := []*models.Proposal{
		{
			ClubID:      1,
			Title:       "Draft Proposal",
			Description: "Description",
			Status:      models.ProposalStatusDraft,
			ProposerID:  1,
		},
		{
			ClubID:      1,
			Title:       "Active Proposal",
			Description: "Description",
			Status:      models.ProposalStatusActive,
			ProposerID:  1,
		},
		{
			ClubID:      2,
			Title:       "Another Draft",
			Description: "Description",
			Status:      models.ProposalStatusDraft,
			ProposerID:  2,
		},
	}

	for _, proposal := range proposals {
		repo.CreateProposal(ctx, proposal)
	}

	// Test getting draft proposals for club 1
	draftProposals, err := repo.GetProposalsByStatus(ctx, 1, models.ProposalStatusDraft)
	if err != nil {
		t.Errorf("GetProposalsByStatus() error = %v", err)
	}

	if len(draftProposals) != 1 {
		t.Errorf("GetProposalsByStatus() count = %d, want %d", len(draftProposals), 1)
	}

	// Test getting all draft proposals (clubID = 0)
	allDraftProposals, err := repo.GetProposalsByStatus(ctx, 0, models.ProposalStatusDraft)
	if err != nil {
		t.Errorf("GetProposalsByStatus() error = %v", err)
	}

	if len(allDraftProposals) != 2 {
		t.Errorf("GetProposalsByStatus() count = %d, want %d", len(allDraftProposals), 2)
	}
}

func TestRepository_CreateVote(t *testing.T) {
	_, repo := setupTestDB(t)
	ctx := context.Background()

	// Create a proposal first
	proposal := &models.Proposal{
		ClubID:      1,
		Title:       "Test Proposal",
		Description: "Description",
		Status:      models.ProposalStatusActive,
		ProposerID:  1,
	}
	repo.CreateProposal(ctx, proposal)

	vote := &models.Vote{
		ProposalID: proposal.ID,
		MemberID:   1,
		ClubID:     1,
		Choice:     models.VoteChoiceYes,
		Weight:     1.0,
		Reason:     "I support this proposal",
	}

	err := repo.CreateVote(ctx, vote)
	if err != nil {
		t.Errorf("CreateVote() error = %v", err)
	}

	if vote.ID == 0 {
		t.Error("CreateVote() did not set ID")
	}

	// Test duplicate vote (should fail)
	duplicateVote := &models.Vote{
		ProposalID: proposal.ID,
		MemberID:   1,
		ClubID:     1,
		Choice:     models.VoteChoiceNo,
		Weight:     1.0,
	}

	err = repo.CreateVote(ctx, duplicateVote)
	if err == nil {
		t.Error("CreateVote() should fail for duplicate vote")
	}
}

func TestRepository_GetVoteByMemberAndProposal(t *testing.T) {
	_, repo := setupTestDB(t)
	ctx := context.Background()

	// Create a proposal and vote
	proposal := &models.Proposal{
		ClubID:      1,
		Title:       "Test Proposal",
		Description: "Description",
		Status:      models.ProposalStatusActive,
		ProposerID:  1,
	}
	repo.CreateProposal(ctx, proposal)

	vote := &models.Vote{
		ProposalID: proposal.ID,
		MemberID:   1,
		ClubID:     1,
		Choice:     models.VoteChoiceYes,
		Weight:     1.0,
	}
	repo.CreateVote(ctx, vote)

	// Test getting the vote
	retrieved, err := repo.GetVoteByMemberAndProposal(ctx, 1, proposal.ID)
	if err != nil {
		t.Errorf("GetVoteByMemberAndProposal() error = %v", err)
	}

	if retrieved.Choice != models.VoteChoiceYes {
		t.Errorf("GetVoteByMemberAndProposal() choice = %v, want %v", retrieved.Choice, models.VoteChoiceYes)
	}

	// Test getting non-existent vote
	_, err = repo.GetVoteByMemberAndProposal(ctx, 999, proposal.ID)
	if err == nil {
		t.Error("GetVoteByMemberAndProposal() should return error for non-existent vote")
	}
}

func TestRepository_GetVotesByProposal(t *testing.T) {
	_, repo := setupTestDB(t)
	ctx := context.Background()

	// Create a proposal
	proposal := &models.Proposal{
		ClubID:      1,
		Title:       "Test Proposal",
		Description: "Description",
		Status:      models.ProposalStatusActive,
		ProposerID:  1,
	}
	repo.CreateProposal(ctx, proposal)

	// Create multiple votes
	votes := []*models.Vote{
		{
			ProposalID: proposal.ID,
			MemberID:   1,
			ClubID:     1,
			Choice:     models.VoteChoiceYes,
			Weight:     1.0,
		},
		{
			ProposalID: proposal.ID,
			MemberID:   2,
			ClubID:     1,
			Choice:     models.VoteChoiceNo,
			Weight:     1.0,
		},
		{
			ProposalID: proposal.ID,
			MemberID:   3,
			ClubID:     1,
			Choice:     models.VoteChoiceAbstain,
			Weight:     1.0,
		},
	}

	for _, vote := range votes {
		repo.CreateVote(ctx, vote)
	}

	// Test getting all votes for the proposal
	proposalVotes, err := repo.GetVotesByProposal(ctx, proposal.ID)
	if err != nil {
		t.Errorf("GetVotesByProposal() error = %v", err)
	}

	if len(proposalVotes) != 3 {
		t.Errorf("GetVotesByProposal() count = %d, want %d", len(proposalVotes), 3)
	}
}

func TestRepository_CreateVotingPeriod(t *testing.T) {
	_, repo := setupTestDB(t)
	ctx := context.Background()

	// Create a proposal first
	proposal := &models.Proposal{
		ClubID:      1,
		Title:       "Test Proposal",
		Description: "Description",
		Status:      models.ProposalStatusDraft,
		ProposerID:  1,
	}
	repo.CreateProposal(ctx, proposal)

	votingPeriod := &models.VotingPeriod{
		ProposalID: proposal.ID,
		ClubID:     1,
		StartTime:  time.Now(),
		EndTime:    time.Now().Add(24 * time.Hour),
		IsActive:   true,
	}

	err := repo.CreateVotingPeriod(ctx, votingPeriod)
	if err != nil {
		t.Errorf("CreateVotingPeriod() error = %v", err)
	}

	if votingPeriod.ID == 0 {
		t.Error("CreateVotingPeriod() did not set ID")
	}
}

func TestRepository_GetActiveVotingPeriods(t *testing.T) {
	_, repo := setupTestDB(t)
	ctx := context.Background()

	now := time.Now()

	// Create proposals and voting periods
	proposal1 := &models.Proposal{ClubID: 1, Title: "P1", Description: "D1", ProposerID: 1}
	proposal2 := &models.Proposal{ClubID: 1, Title: "P2", Description: "D2", ProposerID: 1}
	proposal3 := &models.Proposal{ClubID: 2, Title: "P3", Description: "D3", ProposerID: 2}

	repo.CreateProposal(ctx, proposal1)
	repo.CreateProposal(ctx, proposal2)
	repo.CreateProposal(ctx, proposal3)

	periods := []*models.VotingPeriod{
		{
			ProposalID: proposal1.ID,
			ClubID:     1,
			StartTime:  now.Add(-time.Hour),
			EndTime:    now.Add(time.Hour),
			IsActive:   true,
		},
		{
			ProposalID: proposal2.ID,
			ClubID:     1,
			StartTime:  now.Add(-time.Hour),
			EndTime:    now.Add(-30 * time.Minute), // Expired
			IsActive:   true,
		},
		{
			ProposalID: proposal3.ID,
			ClubID:     2,
			StartTime:  now.Add(-time.Hour),
			EndTime:    now.Add(time.Hour),
			IsActive:   true,
		},
	}

	for _, period := range periods {
		repo.CreateVotingPeriod(ctx, period)
	}

	// Test getting active periods for club 1
	activePeriods, err := repo.GetActiveVotingPeriods(ctx, 1)
	if err != nil {
		t.Errorf("GetActiveVotingPeriods() error = %v", err)
	}

	if len(activePeriods) != 1 {
		t.Errorf("GetActiveVotingPeriods() count = %d, want %d", len(activePeriods), 1)
	}

	// Test getting all active periods
	allActivePeriods, err := repo.GetActiveVotingPeriods(ctx, 0)
	if err != nil {
		t.Errorf("GetActiveVotingPeriods() error = %v", err)
	}

	if len(allActivePeriods) != 2 {
		t.Errorf("GetActiveVotingPeriods() count = %d, want %d", len(allActivePeriods), 2)
	}
}

func TestRepository_CreateVotingRights(t *testing.T) {
	_, repo := setupTestDB(t)
	ctx := context.Background()

	votingRights := &models.VotingRights{
		MemberID:      1,
		ClubID:        1,
		CanVote:       true,
		CanPropose:    false,
		VotingWeight:  1.0,
		Role:          "member",
		EffectiveFrom: time.Now(),
	}

	err := repo.CreateVotingRights(ctx, votingRights)
	if err != nil {
		t.Errorf("CreateVotingRights() error = %v", err)
	}

	if votingRights.ID == 0 {
		t.Error("CreateVotingRights() did not set ID")
	}
}

func TestRepository_GetVotingRights(t *testing.T) {
	_, repo := setupTestDB(t)
	ctx := context.Background()

	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	// Create voting rights with different effective periods
	rights := []*models.VotingRights{
		{
			MemberID:      1,
			ClubID:        1,
			CanVote:       true,
			VotingWeight:  1.0,
			EffectiveFrom: past,
			// No end date - current and valid
		},
		{
			MemberID:       2,
			ClubID:         1,
			CanVote:        true,
			VotingWeight:   1.0,
			EffectiveFrom:  past,
			EffectiveUntil: &past, // Expired
		},
		{
			MemberID:      3,
			ClubID:        1,
			CanVote:       true,
			VotingWeight:  1.0,
			EffectiveFrom: future, // Not yet effective
		},
	}

	for _, right := range rights {
		repo.CreateVotingRights(ctx, right)
	}

	// Test getting current valid rights
	validRights, err := repo.GetVotingRights(ctx, 1, 1)
	if err != nil {
		t.Errorf("GetVotingRights() error = %v", err)
	}

	if !validRights.CanVote {
		t.Error("GetVotingRights() should return valid voting rights")
	}

	// Test getting expired rights
	_, err = repo.GetVotingRights(ctx, 2, 1)
	if err == nil {
		t.Error("GetVotingRights() should not return expired rights")
	}

	// Test getting future rights
	_, err = repo.GetVotingRights(ctx, 3, 1)
	if err == nil {
		t.Error("GetVotingRights() should not return future rights")
	}
}

func TestRepository_CreateGovernancePolicy(t *testing.T) {
	_, repo := setupTestDB(t)
	ctx := context.Background()

	policy := &models.GovernancePolicy{
		ClubID:        1,
		Name:          "Test Policy",
		Description:   "Test policy description",
		PolicyType:    "voting",
		Rules:         map[string]interface{}{"quorum": 50},
		IsActive:      true,
		Version:       1,
		EffectiveFrom: time.Now(),
		CreatedBy:     1,
	}

	err := repo.CreateGovernancePolicy(ctx, policy)
	if err != nil {
		t.Errorf("CreateGovernancePolicy() error = %v", err)
	}

	if policy.ID == 0 {
		t.Error("CreateGovernancePolicy() did not set ID")
	}
}

func TestRepository_GetActiveGovernancePolicies(t *testing.T) {
	_, repo := setupTestDB(t)
	ctx := context.Background()

	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	// Create policies with different active states and effective periods
	policies := []*models.GovernancePolicy{
		{
			ClubID:        1,
			Name:          "Active Policy 1",
			PolicyType:    "voting",
			Rules:         map[string]interface{}{"rule": "value"},
			IsActive:      true,
			EffectiveFrom: past,
			CreatedBy:     1,
		},
		{
			ClubID:        1,
			Name:          "Active Policy 2",
			PolicyType:    "membership",
			Rules:         map[string]interface{}{"rule": "value"},
			IsActive:      true,
			EffectiveFrom: past,
			EffectiveUntil: &future,
			CreatedBy:     1,
		},
		{
			ClubID:        1,
			Name:          "Inactive Policy",
			PolicyType:    "voting",
			Rules:         map[string]interface{}{"rule": "value"},
			IsActive:      false,
			EffectiveFrom: past,
			CreatedBy:     1,
		},
		{
			ClubID:        1,
			Name:          "Expired Policy",
			PolicyType:    "voting",
			Rules:         map[string]interface{}{"rule": "value"},
			IsActive:      true,
			EffectiveFrom: past,
			EffectiveUntil: &past,
			CreatedBy:     1,
		},
		{
			ClubID:        2,
			Name:          "Other Club Policy",
			PolicyType:    "voting",
			Rules:         map[string]interface{}{"rule": "value"},
			IsActive:      true,
			EffectiveFrom: past,
			CreatedBy:     2,
		},
	}

	for _, policy := range policies {
		repo.CreateGovernancePolicy(ctx, policy)
	}

	// Test getting active policies for club 1
	activePolicies, err := repo.GetActiveGovernancePolicies(ctx, 1)
	if err != nil {
		t.Errorf("GetActiveGovernancePolicies() error = %v", err)
	}

	if len(activePolicies) != 2 {
		t.Errorf("GetActiveGovernancePolicies() count = %d, want %d", len(activePolicies), 2)
	}
}

func TestRepository_CreateOrUpdateVoteResult(t *testing.T) {
	_, repo := setupTestDB(t)
	ctx := context.Background()

	// Create a proposal first
	proposal := &models.Proposal{
		ClubID:      1,
		Title:       "Test Proposal",
		Description: "Description",
		ProposerID:  1,
	}
	repo.CreateProposal(ctx, proposal)

	voteResult := &models.VoteResult{
		ProposalID:   proposal.ID,
		ClubID:       1,
		TotalVotes:   10,
		YesVotes:     6,
		NoVotes:      4,
		WeightedYes:  6.0,
		WeightedNo:   4.0,
		TotalWeight:  10.0,
		QuorumMet:    true,
		Passed:       true,
	}

	// Test creating new vote result
	err := repo.CreateOrUpdateVoteResult(ctx, voteResult)
	if err != nil {
		t.Errorf("CreateOrUpdateVoteResult() error = %v", err)
	}

	// Test updating existing vote result
	voteResult.YesVotes = 7
	voteResult.NoVotes = 3
	voteResult.WeightedYes = 7.0
	voteResult.WeightedNo = 3.0

	err = repo.CreateOrUpdateVoteResult(ctx, voteResult)
	if err != nil {
		t.Errorf("CreateOrUpdateVoteResult() update error = %v", err)
	}

	// Verify the update
	retrieved, err := repo.GetVoteResult(ctx, proposal.ID)
	if err != nil {
		t.Errorf("GetVoteResult() error = %v", err)
	}

	if retrieved.YesVotes != 7 {
		t.Errorf("Vote result not updated: yes votes = %d, want %d", retrieved.YesVotes, 7)
	}
}

func TestRepository_HealthCheck(t *testing.T) {
	_, repo := setupTestDB(t)
	ctx := context.Background()

	err := repo.HealthCheck(ctx)
	if err != nil {
		t.Errorf("HealthCheck() error = %v", err)
	}
}