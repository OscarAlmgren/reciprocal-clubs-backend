package models

import (
	"time"
	"fmt"
	"strings"
	"gorm.io/gorm"
)

// ProposalStatus represents the current status of a proposal
type ProposalStatus string

const (
	ProposalStatusDraft     ProposalStatus = "draft"
	ProposalStatusActive    ProposalStatus = "active"
	ProposalStatusPassed    ProposalStatus = "passed"
	ProposalStatusRejected  ProposalStatus = "rejected"
	ProposalStatusExpired   ProposalStatus = "expired"
	ProposalStatusCancelled ProposalStatus = "cancelled"
)

// ProposalType represents different types of governance proposals
type ProposalType string

const (
	ProposalTypePolicyChange ProposalType = "policy_change"
	ProposalTypeBudget       ProposalType = "budget"
	ProposalTypeStrategic    ProposalType = "strategic"
	ProposalTypeMembership   ProposalType = "membership"
	ProposalTypeAmendment    ProposalType = "amendment"
	ProposalTypeOther        ProposalType = "other"
)

// VoteChoice represents a member's vote on a proposal
type VoteChoice string

const (
	VoteChoiceYes     VoteChoice = "yes"
	VoteChoiceNo      VoteChoice = "no"
	VoteChoiceAbstain VoteChoice = "abstain"
)

// VotingMethod represents how votes are counted
type VotingMethod string

const (
	VotingMethodSimpleMajority VotingMethod = "simple_majority"
	VotingMethodSupermajority  VotingMethod = "supermajority"
	VotingMethodUnanimous      VotingMethod = "unanimous"
	VotingMethodWeighted       VotingMethod = "weighted"
)

// Proposal represents a governance proposal that members can vote on
type Proposal struct {
	ID              uint                   `json:"id" gorm:"primaryKey"`
	ClubID          uint                   `json:"club_id" gorm:"not null;index"`
	Title           string                 `json:"title" gorm:"size:255;not null"`
	Description     string                 `json:"description" gorm:"type:text;not null"`
	Type            ProposalType           `json:"type" gorm:"type:varchar(50);not null"`
	Status          ProposalStatus         `json:"status" gorm:"type:varchar(20);not null;default:'draft'"`
	ProposerID      uint                   `json:"proposer_id" gorm:"not null;index"`
	VotingMethod    VotingMethod           `json:"voting_method" gorm:"type:varchar(30);not null;default:'simple_majority'"`
	QuorumRequired  int                    `json:"quorum_required" gorm:"not null;default:50"`
	MajorityRequired int                   `json:"majority_required" gorm:"not null;default:50"`
	VotingStartTime time.Time              `json:"voting_start_time"`
	VotingEndTime   time.Time              `json:"voting_end_time"`
	Metadata        map[string]interface{} `json:"metadata" gorm:"type:jsonb"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	DeletedAt       gorm.DeletedAt         `json:"-" gorm:"index"`

	// Relationships
	Votes           []Vote                 `json:"votes,omitempty" gorm:"foreignKey:ProposalID"`
	VotingPeriod    *VotingPeriod          `json:"voting_period,omitempty" gorm:"foreignKey:ProposalID"`
}

// Vote represents a member's vote on a proposal
type Vote struct {
	ID         uint                   `json:"id" gorm:"primaryKey"`
	ProposalID uint                   `json:"proposal_id" gorm:"not null;index"`
	MemberID   uint                   `json:"member_id" gorm:"not null;index"`
	ClubID     uint                   `json:"club_id" gorm:"not null;index"`
	Choice     VoteChoice             `json:"choice" gorm:"type:varchar(10);not null"`
	Weight     float64                `json:"weight" gorm:"not null;default:1.0"`
	Reason     string                 `json:"reason" gorm:"type:text"`
	Metadata   map[string]interface{} `json:"metadata" gorm:"type:jsonb"`
	VotedAt    time.Time              `json:"voted_at" gorm:"not null"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`

	// Relationships
	Proposal   *Proposal              `json:"proposal,omitempty" gorm:"foreignKey:ProposalID"`
}

// VotingPeriod represents the active voting period for a proposal
type VotingPeriod struct {
	ID               uint                   `json:"id" gorm:"primaryKey"`
	ProposalID       uint                   `json:"proposal_id" gorm:"not null;uniqueIndex"`
	ClubID           uint                   `json:"club_id" gorm:"not null;index"`
	StartTime        time.Time              `json:"start_time" gorm:"not null"`
	EndTime          time.Time              `json:"end_time" gorm:"not null"`
	ExtendedUntil    *time.Time             `json:"extended_until,omitempty"`
	IsActive         bool                   `json:"is_active" gorm:"not null;default:false"`
	NotificationsEnabled bool               `json:"notifications_enabled" gorm:"not null;default:true"`
	Metadata         map[string]interface{} `json:"metadata" gorm:"type:jsonb"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`

	// Relationships
	Proposal         *Proposal              `json:"proposal,omitempty" gorm:"foreignKey:ProposalID"`
}

// VotingRights represents a member's voting rights within a club
type VotingRights struct {
	ID              uint                   `json:"id" gorm:"primaryKey"`
	MemberID        uint                   `json:"member_id" gorm:"not null;index"`
	ClubID          uint                   `json:"club_id" gorm:"not null;index"`
	CanVote         bool                   `json:"can_vote" gorm:"not null;default:true"`
	CanPropose      bool                   `json:"can_propose" gorm:"not null;default:false"`
	VotingWeight    float64                `json:"voting_weight" gorm:"not null;default:1.0"`
	Role            string                 `json:"role" gorm:"size:50"`
	Restrictions    []string               `json:"restrictions" gorm:"type:jsonb"`
	Metadata        map[string]interface{} `json:"metadata" gorm:"type:jsonb"`
	EffectiveFrom   time.Time              `json:"effective_from" gorm:"not null"`
	EffectiveUntil  *time.Time             `json:"effective_until,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	DeletedAt       gorm.DeletedAt         `json:"-" gorm:"index"`
}

// GovernancePolicy represents club governance rules and policies
type GovernancePolicy struct {
	ID                 uint                   `json:"id" gorm:"primaryKey"`
	ClubID             uint                   `json:"club_id" gorm:"not null;index"`
	Name               string                 `json:"name" gorm:"size:255;not null"`
	Description        string                 `json:"description" gorm:"type:text"`
	PolicyType         string                 `json:"policy_type" gorm:"size:50;not null"`
	Rules              map[string]interface{} `json:"rules" gorm:"type:jsonb;not null"`
	IsActive           bool                   `json:"is_active" gorm:"not null;default:true"`
	Version            int                    `json:"version" gorm:"not null;default:1"`
	PreviousVersionID  *uint                  `json:"previous_version_id,omitempty"`
	EffectiveFrom      time.Time              `json:"effective_from" gorm:"not null"`
	EffectiveUntil     *time.Time             `json:"effective_until,omitempty"`
	CreatedBy          uint                   `json:"created_by" gorm:"not null"`
	Metadata           map[string]interface{} `json:"metadata" gorm:"type:jsonb"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
	DeletedAt          gorm.DeletedAt         `json:"-" gorm:"index"`
}

// VoteResult represents aggregated voting results for a proposal
type VoteResult struct {
	ID                uint                   `json:"id" gorm:"primaryKey"`
	ProposalID        uint                   `json:"proposal_id" gorm:"not null;uniqueIndex"`
	ClubID            uint                   `json:"club_id" gorm:"not null;index"`
	TotalVotes        int                    `json:"total_votes" gorm:"not null;default:0"`
	YesVotes          int                    `json:"yes_votes" gorm:"not null;default:0"`
	NoVotes           int                    `json:"no_votes" gorm:"not null;default:0"`
	AbstainVotes      int                    `json:"abstain_votes" gorm:"not null;default:0"`
	WeightedYes       float64                `json:"weighted_yes" gorm:"not null;default:0"`
	WeightedNo        float64                `json:"weighted_no" gorm:"not null;default:0"`
	WeightedAbstain   float64                `json:"weighted_abstain" gorm:"not null;default:0"`
	TotalWeight       float64                `json:"total_weight" gorm:"not null;default:0"`
	QuorumMet         bool                   `json:"quorum_met" gorm:"not null;default:false"`
	Passed            bool                   `json:"passed" gorm:"not null;default:false"`
	CalculatedAt      time.Time              `json:"calculated_at" gorm:"not null"`
	Metadata          map[string]interface{} `json:"metadata" gorm:"type:jsonb"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`

	// Relationships
	Proposal          *Proposal              `json:"proposal,omitempty" gorm:"foreignKey:ProposalID"`
}

// Table name methods
func (Proposal) TableName() string {
	return "governance_proposals"
}

func (Vote) TableName() string {
	return "governance_votes"
}

func (VotingPeriod) TableName() string {
	return "governance_voting_periods"
}

func (VotingRights) TableName() string {
	return "governance_voting_rights"
}

func (GovernancePolicy) TableName() string {
	return "governance_policies"
}

func (VoteResult) TableName() string {
	return "governance_vote_results"
}

// Business logic methods

// CanTransitionTo checks if a proposal can transition to a new status
func (p *Proposal) CanTransitionTo(newStatus ProposalStatus) bool {
	validTransitions := map[ProposalStatus][]ProposalStatus{
		ProposalStatusDraft:     {ProposalStatusActive, ProposalStatusCancelled},
		ProposalStatusActive:    {ProposalStatusPassed, ProposalStatusRejected, ProposalStatusExpired, ProposalStatusCancelled},
		ProposalStatusPassed:    {},
		ProposalStatusRejected:  {},
		ProposalStatusExpired:   {},
		ProposalStatusCancelled: {},
	}

	validTargets, exists := validTransitions[p.Status]
	if !exists {
		return false
	}

	for _, validTarget := range validTargets {
		if validTarget == newStatus {
			return true
		}
	}
	return false
}

// IsVotingActive checks if voting is currently active for the proposal
func (p *Proposal) IsVotingActive() bool {
	if p.Status != ProposalStatusActive {
		return false
	}

	now := time.Now()
	return now.After(p.VotingStartTime) && now.Before(p.VotingEndTime)
}

// HasExpired checks if the proposal voting period has expired
func (p *Proposal) HasExpired() bool {
	return time.Now().After(p.VotingEndTime)
}

// GetVotingDuration returns the duration of the voting period
func (p *Proposal) GetVotingDuration() time.Duration {
	return p.VotingEndTime.Sub(p.VotingStartTime)
}

// Validate validates the proposal data
func (p *Proposal) Validate() error {
	if strings.TrimSpace(p.Title) == "" {
		return fmt.Errorf("proposal title cannot be empty")
	}

	if strings.TrimSpace(p.Description) == "" {
		return fmt.Errorf("proposal description cannot be empty")
	}

	if p.ClubID == 0 {
		return fmt.Errorf("club ID is required")
	}

	if p.ProposerID == 0 {
		return fmt.Errorf("proposer ID is required")
	}

	if p.QuorumRequired < 0 || p.QuorumRequired > 100 {
		return fmt.Errorf("quorum required must be between 0 and 100")
	}

	if p.MajorityRequired < 0 || p.MajorityRequired > 100 {
		return fmt.Errorf("majority required must be between 0 and 100")
	}

	if !p.VotingStartTime.IsZero() && !p.VotingEndTime.IsZero() {
		if p.VotingEndTime.Before(p.VotingStartTime) {
			return fmt.Errorf("voting end time must be after start time")
		}
	}

	return nil
}

// CanMemberVote checks if a member can vote on the proposal
func (vr *VotingRights) CanMemberVote() bool {
	if !vr.CanVote {
		return false
	}

	now := time.Now()
	if now.Before(vr.EffectiveFrom) {
		return false
	}

	if vr.EffectiveUntil != nil && now.After(*vr.EffectiveUntil) {
		return false
	}

	return true
}

// CalculateQuorum calculates if quorum is met based on total eligible voters
func (vr *VoteResult) CalculateQuorum(totalEligibleVoters int, quorumRequired int) bool {
	if totalEligibleVoters == 0 {
		return false
	}

	participationRate := float64(vr.TotalVotes) / float64(totalEligibleVoters) * 100
	return participationRate >= float64(quorumRequired)
}

// CalculateMajority calculates if majority is achieved based on voting method
func (vr *VoteResult) CalculateMajority(votingMethod VotingMethod, majorityRequired int) bool {
	switch votingMethod {
	case VotingMethodSimpleMajority:
		return vr.YesVotes > vr.NoVotes
	case VotingMethodSupermajority:
		totalDecisiveVotes := vr.YesVotes + vr.NoVotes
		if totalDecisiveVotes == 0 {
			return false
		}
		yesPercentage := float64(vr.YesVotes) / float64(totalDecisiveVotes) * 100
		return yesPercentage >= float64(majorityRequired)
	case VotingMethodUnanimous:
		return vr.NoVotes == 0 && vr.YesVotes > 0
	case VotingMethodWeighted:
		totalDecisiveWeight := vr.WeightedYes + vr.WeightedNo
		if totalDecisiveWeight == 0 {
			return false
		}
		yesPercentage := vr.WeightedYes / totalDecisiveWeight * 100
		return yesPercentage >= float64(majorityRequired)
	default:
		return false
	}
}