package models

import (
	"testing"
	"time"
)

func TestProposal_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name          string
		currentStatus ProposalStatus
		newStatus     ProposalStatus
		want          bool
	}{
		{"Draft to Active", ProposalStatusDraft, ProposalStatusActive, true},
		{"Draft to Cancelled", ProposalStatusDraft, ProposalStatusCancelled, true},
		{"Active to Passed", ProposalStatusActive, ProposalStatusPassed, true},
		{"Active to Rejected", ProposalStatusActive, ProposalStatusRejected, true},
		{"Active to Expired", ProposalStatusActive, ProposalStatusExpired, true},
		{"Active to Cancelled", ProposalStatusActive, ProposalStatusCancelled, true},
		{"Passed to Active", ProposalStatusPassed, ProposalStatusActive, false},
		{"Draft to Passed", ProposalStatusDraft, ProposalStatusPassed, false},
		{"Rejected to Active", ProposalStatusRejected, ProposalStatusActive, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Proposal{Status: tt.currentStatus}
			if got := p.CanTransitionTo(tt.newStatus); got != tt.want {
				t.Errorf("Proposal.CanTransitionTo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProposal_IsVotingActive(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name            string
		status          ProposalStatus
		votingStartTime time.Time
		votingEndTime   time.Time
		want            bool
	}{
		{
			name:            "Active proposal within voting period",
			status:          ProposalStatusActive,
			votingStartTime: now.Add(-time.Hour),
			votingEndTime:   now.Add(time.Hour),
			want:            true,
		},
		{
			name:            "Active proposal before voting period",
			status:          ProposalStatusActive,
			votingStartTime: now.Add(time.Hour),
			votingEndTime:   now.Add(2 * time.Hour),
			want:            false,
		},
		{
			name:            "Active proposal after voting period",
			status:          ProposalStatusActive,
			votingStartTime: now.Add(-2 * time.Hour),
			votingEndTime:   now.Add(-time.Hour),
			want:            false,
		},
		{
			name:            "Draft proposal within voting period",
			status:          ProposalStatusDraft,
			votingStartTime: now.Add(-time.Hour),
			votingEndTime:   now.Add(time.Hour),
			want:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Proposal{
				Status:          tt.status,
				VotingStartTime: tt.votingStartTime,
				VotingEndTime:   tt.votingEndTime,
			}
			if got := p.IsVotingActive(); got != tt.want {
				t.Errorf("Proposal.IsVotingActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProposal_HasExpired(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name          string
		votingEndTime time.Time
		want          bool
	}{
		{
			name:          "Future end time",
			votingEndTime: now.Add(time.Hour),
			want:          false,
		},
		{
			name:          "Past end time",
			votingEndTime: now.Add(-time.Hour),
			want:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Proposal{VotingEndTime: tt.votingEndTime}
			if got := p.HasExpired(); got != tt.want {
				t.Errorf("Proposal.HasExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProposal_Validate(t *testing.T) {
	tests := []struct {
		name     string
		proposal Proposal
		wantErr  bool
	}{
		{
			name: "Valid proposal",
			proposal: Proposal{
				Title:            "Test Proposal",
				Description:      "This is a test proposal description",
				ClubID:           1,
				ProposerID:       1,
				QuorumRequired:   50,
				MajorityRequired: 50,
				VotingStartTime:  time.Now().Add(time.Hour),
				VotingEndTime:    time.Now().Add(24 * time.Hour),
			},
			wantErr: false,
		},
		{
			name: "Empty title",
			proposal: Proposal{
				Title:       "",
				Description: "This is a test proposal description",
				ClubID:      1,
				ProposerID:  1,
			},
			wantErr: true,
		},
		{
			name: "Empty description",
			proposal: Proposal{
				Title:       "Test Proposal",
				Description: "",
				ClubID:      1,
				ProposerID:  1,
			},
			wantErr: true,
		},
		{
			name: "Zero club ID",
			proposal: Proposal{
				Title:       "Test Proposal",
				Description: "This is a test proposal description",
				ClubID:      0,
				ProposerID:  1,
			},
			wantErr: true,
		},
		{
			name: "Zero proposer ID",
			proposal: Proposal{
				Title:       "Test Proposal",
				Description: "This is a test proposal description",
				ClubID:      1,
				ProposerID:  0,
			},
			wantErr: true,
		},
		{
			name: "Invalid quorum",
			proposal: Proposal{
				Title:            "Test Proposal",
				Description:      "This is a test proposal description",
				ClubID:           1,
				ProposerID:       1,
				QuorumRequired:   150,
				MajorityRequired: 50,
			},
			wantErr: true,
		},
		{
			name: "Invalid majority",
			proposal: Proposal{
				Title:            "Test Proposal",
				Description:      "This is a test proposal description",
				ClubID:           1,
				ProposerID:       1,
				QuorumRequired:   50,
				MajorityRequired: -10,
			},
			wantErr: true,
		},
		{
			name: "End time before start time",
			proposal: Proposal{
				Title:            "Test Proposal",
				Description:      "This is a test proposal description",
				ClubID:           1,
				ProposerID:       1,
				QuorumRequired:   50,
				MajorityRequired: 50,
				VotingStartTime:  time.Now().Add(24 * time.Hour),
				VotingEndTime:    time.Now().Add(time.Hour),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.proposal.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Proposal.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVotingRights_CanMemberVote(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name           string
		canVote        bool
		effectiveFrom  time.Time
		effectiveUntil *time.Time
		want           bool
	}{
		{
			name:          "Can vote, no restrictions",
			canVote:       true,
			effectiveFrom: past,
			want:          true,
		},
		{
			name:          "Cannot vote",
			canVote:       false,
			effectiveFrom: past,
			want:          false,
		},
		{
			name:          "Before effective date",
			canVote:       true,
			effectiveFrom: future,
			want:          false,
		},
		{
			name:           "After expiration",
			canVote:        true,
			effectiveFrom:  past,
			effectiveUntil: &past,
			want:           false,
		},
		{
			name:           "Within valid period",
			canVote:        true,
			effectiveFrom:  past,
			effectiveUntil: &future,
			want:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vr := &VotingRights{
				CanVote:        tt.canVote,
				EffectiveFrom:  tt.effectiveFrom,
				EffectiveUntil: tt.effectiveUntil,
			}
			if got := vr.CanMemberVote(); got != tt.want {
				t.Errorf("VotingRights.CanMemberVote() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVoteResult_CalculateQuorum(t *testing.T) {
	tests := []struct {
		name                  string
		totalVotes            int
		totalEligibleVoters   int
		quorumRequired        int
		want                  bool
	}{
		{
			name:                "Quorum met exactly",
			totalVotes:          50,
			totalEligibleVoters: 100,
			quorumRequired:      50,
			want:                true,
		},
		{
			name:                "Quorum exceeded",
			totalVotes:          60,
			totalEligibleVoters: 100,
			quorumRequired:      50,
			want:                true,
		},
		{
			name:                "Quorum not met",
			totalVotes:          40,
			totalEligibleVoters: 100,
			quorumRequired:      50,
			want:                false,
		},
		{
			name:                "No eligible voters",
			totalVotes:          10,
			totalEligibleVoters: 0,
			quorumRequired:      50,
			want:                false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vr := &VoteResult{TotalVotes: tt.totalVotes}
			if got := vr.CalculateQuorum(tt.totalEligibleVoters, tt.quorumRequired); got != tt.want {
				t.Errorf("VoteResult.CalculateQuorum() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVoteResult_CalculateMajority(t *testing.T) {
	tests := []struct {
		name             string
		yesVotes         int
		noVotes          int
		weightedYes      float64
		weightedNo       float64
		votingMethod     VotingMethod
		majorityRequired int
		want             bool
	}{
		{
			name:         "Simple majority - yes wins",
			yesVotes:     6,
			noVotes:      4,
			votingMethod: VotingMethodSimpleMajority,
			want:         true,
		},
		{
			name:         "Simple majority - no wins",
			yesVotes:     4,
			noVotes:      6,
			votingMethod: VotingMethodSimpleMajority,
			want:         false,
		},
		{
			name:         "Simple majority - tie",
			yesVotes:     5,
			noVotes:      5,
			votingMethod: VotingMethodSimpleMajority,
			want:         false,
		},
		{
			name:             "Supermajority - met",
			yesVotes:         7,
			noVotes:          3,
			votingMethod:     VotingMethodSupermajority,
			majorityRequired: 65,
			want:             true,
		},
		{
			name:             "Supermajority - not met",
			yesVotes:         6,
			noVotes:          4,
			votingMethod:     VotingMethodSupermajority,
			majorityRequired: 65,
			want:             false,
		},
		{
			name:         "Unanimous - achieved",
			yesVotes:     10,
			noVotes:      0,
			votingMethod: VotingMethodUnanimous,
			want:         true,
		},
		{
			name:         "Unanimous - not achieved",
			yesVotes:     9,
			noVotes:      1,
			votingMethod: VotingMethodUnanimous,
			want:         false,
		},
		{
			name:             "Weighted - met",
			weightedYes:      7.5,
			weightedNo:       2.5,
			votingMethod:     VotingMethodWeighted,
			majorityRequired: 70,
			want:             true,
		},
		{
			name:             "Weighted - not met",
			weightedYes:      6.0,
			weightedNo:       4.0,
			votingMethod:     VotingMethodWeighted,
			majorityRequired: 70,
			want:             false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vr := &VoteResult{
				YesVotes:    tt.yesVotes,
				NoVotes:     tt.noVotes,
				WeightedYes: tt.weightedYes,
				WeightedNo:  tt.weightedNo,
			}
			if got := vr.CalculateMajority(tt.votingMethod, tt.majorityRequired); got != tt.want {
				t.Errorf("VoteResult.CalculateMajority() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProposal_GetVotingDuration(t *testing.T) {
	start := time.Now()
	end := start.Add(24 * time.Hour)

	p := &Proposal{
		VotingStartTime: start,
		VotingEndTime:   end,
	}

	duration := p.GetVotingDuration()
	expected := 24 * time.Hour

	if duration != expected {
		t.Errorf("GetVotingDuration() = %v, want %v", duration, expected)
	}
}

// Test table name methods
func TestTableNames(t *testing.T) {
	tests := []struct {
		model interface{ TableName() string }
		want  string
	}{
		{&Proposal{}, "governance_proposals"},
		{&Vote{}, "governance_votes"},
		{&VotingPeriod{}, "governance_voting_periods"},
		{&VotingRights{}, "governance_voting_rights"},
		{&GovernancePolicy{}, "governance_policies"},
		{&VoteResult{}, "governance_vote_results"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.model.TableName(); got != tt.want {
				t.Errorf("TableName() = %v, want %v", got, tt.want)
			}
		})
	}
}