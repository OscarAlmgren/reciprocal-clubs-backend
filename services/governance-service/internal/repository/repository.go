package repository

import (
	"context"
	"fmt"
	"time"

	"reciprocal-clubs-backend/pkg/shared/database"
	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/services/governance-service/internal/models"

	"gorm.io/gorm"
)

// Repository handles database operations for governance
type Repository struct {
	*database.BaseRepository
	db     *gorm.DB
	logger logging.Logger
}

// NewRepository creates a new governance repository
func NewRepository(db *gorm.DB, logger logging.Logger) *Repository {
	// Convert gorm.DB to database.Database if needed
	// For now, we'll work with the gorm.DB directly
	return &Repository{
		db:     db,
		logger: logger,
	}
}

// Proposal operations

// CreateProposal creates a new governance proposal
func (r *Repository) CreateProposal(ctx context.Context, proposal *models.Proposal) error {
	if err := r.db.WithContext(ctx).Create(proposal).Error; err != nil {
		r.logger.Error("Failed to create proposal", map[string]interface{}{
			"error":    err.Error(),
			"title":    proposal.Title,
			"club_id":  proposal.ClubID,
		})
		return err
	}

	r.logger.Info("Proposal created successfully", map[string]interface{}{
		"proposal_id": proposal.ID,
		"title":       proposal.Title,
		"club_id":     proposal.ClubID,
	})

	return nil
}

// GetProposal retrieves a proposal by ID
func (r *Repository) GetProposal(ctx context.Context, id uint) (*models.Proposal, error) {
	var proposal models.Proposal
	if err := r.db.WithContext(ctx).Preload("Votes").Preload("VotingPeriod").First(&proposal, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("proposal not found")
		}
		r.logger.Error("Failed to get proposal", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		return nil, err
	}

	return &proposal, nil
}

// GetProposalsByClub retrieves all proposals for a specific club
func (r *Repository) GetProposalsByClub(ctx context.Context, clubID uint) ([]models.Proposal, error) {
	var proposals []models.Proposal
	if err := r.db.WithContext(ctx).
		Where("club_id = ?", clubID).
		Preload("Votes").Preload("VotingPeriod").
		Order("created_at DESC").
		Find(&proposals).Error; err != nil {
		r.logger.Error("Failed to get proposals by club", map[string]interface{}{
			"error":   err.Error(),
			"club_id": clubID,
		})
		return nil, err
	}

	return proposals, nil
}

// GetProposalsByStatus retrieves proposals by status
func (r *Repository) GetProposalsByStatus(ctx context.Context, clubID uint, status models.ProposalStatus) ([]models.Proposal, error) {
	var proposals []models.Proposal
	query := r.db.WithContext(ctx)

	if clubID > 0 {
		query = query.Where("club_id = ?", clubID)
	}

	if err := query.Where("status = ?", status).
		Preload("Votes").Preload("VotingPeriod").
		Order("created_at DESC").
		Find(&proposals).Error; err != nil {
		r.logger.Error("Failed to get proposals by status", map[string]interface{}{
			"error":   err.Error(),
			"club_id": clubID,
			"status":  status,
		})
		return nil, err
	}

	return proposals, nil
}

// UpdateProposal updates an existing proposal
func (r *Repository) UpdateProposal(ctx context.Context, proposal *models.Proposal) error {
	if err := r.db.WithContext(ctx).Save(proposal).Error; err != nil {
		r.logger.Error("Failed to update proposal", map[string]interface{}{
			"error":       err.Error(),
			"proposal_id": proposal.ID,
		})
		return err
	}

	r.logger.Info("Proposal updated successfully", map[string]interface{}{
		"proposal_id": proposal.ID,
		"status":      proposal.Status,
	})

	return nil
}

// DeleteProposal soft deletes a proposal
func (r *Repository) DeleteProposal(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&models.Proposal{}, id).Error; err != nil {
		r.logger.Error("Failed to delete proposal", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		return err
	}

	r.logger.Info("Proposal deleted successfully", map[string]interface{}{
		"proposal_id": id,
	})

	return nil
}

// Vote operations

// CreateVote records a member's vote on a proposal
func (r *Repository) CreateVote(ctx context.Context, vote *models.Vote) error {
	// Check if member has already voted
	existingVote, err := r.GetVoteByMemberAndProposal(ctx, vote.MemberID, vote.ProposalID)
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if existingVote != nil {
		return fmt.Errorf("member has already voted on this proposal")
	}

	vote.VotedAt = time.Now()
	if err := r.db.WithContext(ctx).Create(vote).Error; err != nil {
		r.logger.Error("Failed to create vote", map[string]interface{}{
			"error":       err.Error(),
			"member_id":   vote.MemberID,
			"proposal_id": vote.ProposalID,
			"choice":      vote.Choice,
		})
		return err
	}

	r.logger.Info("Vote recorded successfully", map[string]interface{}{
		"vote_id":     vote.ID,
		"member_id":   vote.MemberID,
		"proposal_id": vote.ProposalID,
		"choice":      vote.Choice,
	})

	return nil
}

// GetVote retrieves a vote by ID
func (r *Repository) GetVote(ctx context.Context, id uint) (*models.Vote, error) {
	var vote models.Vote
	if err := r.db.WithContext(ctx).Preload("Proposal").First(&vote, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("vote not found")
		}
		r.logger.Error("Failed to get vote", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		return nil, err
	}

	return &vote, nil
}

// GetVoteByMemberAndProposal retrieves a member's vote on a specific proposal
func (r *Repository) GetVoteByMemberAndProposal(ctx context.Context, memberID, proposalID uint) (*models.Vote, error) {
	var vote models.Vote
	if err := r.db.WithContext(ctx).
		Where("member_id = ? AND proposal_id = ?", memberID, proposalID).
		First(&vote).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		r.logger.Error("Failed to get vote by member and proposal", map[string]interface{}{
			"error":       err.Error(),
			"member_id":   memberID,
			"proposal_id": proposalID,
		})
		return nil, err
	}

	return &vote, nil
}

// GetVotesByProposal retrieves all votes for a specific proposal
func (r *Repository) GetVotesByProposal(ctx context.Context, proposalID uint) ([]models.Vote, error) {
	var votes []models.Vote
	if err := r.db.WithContext(ctx).
		Where("proposal_id = ?", proposalID).
		Order("voted_at ASC").
		Find(&votes).Error; err != nil {
		r.logger.Error("Failed to get votes by proposal", map[string]interface{}{
			"error":       err.Error(),
			"proposal_id": proposalID,
		})
		return nil, err
	}

	return votes, nil
}

// UpdateVote updates an existing vote (if allowed)
func (r *Repository) UpdateVote(ctx context.Context, vote *models.Vote) error {
	if err := r.db.WithContext(ctx).Save(vote).Error; err != nil {
		r.logger.Error("Failed to update vote", map[string]interface{}{
			"error":   err.Error(),
			"vote_id": vote.ID,
		})
		return err
	}

	r.logger.Info("Vote updated successfully", map[string]interface{}{
		"vote_id": vote.ID,
		"choice":  vote.Choice,
	})

	return nil
}

// VotingPeriod operations

// CreateVotingPeriod creates a new voting period
func (r *Repository) CreateVotingPeriod(ctx context.Context, period *models.VotingPeriod) error {
	if err := r.db.WithContext(ctx).Create(period).Error; err != nil {
		r.logger.Error("Failed to create voting period", map[string]interface{}{
			"error":       err.Error(),
			"proposal_id": period.ProposalID,
		})
		return err
	}

	r.logger.Info("Voting period created successfully", map[string]interface{}{
		"period_id":   period.ID,
		"proposal_id": period.ProposalID,
		"start_time":  period.StartTime,
		"end_time":    period.EndTime,
	})

	return nil
}

// GetVotingPeriodByProposal retrieves the voting period for a proposal
func (r *Repository) GetVotingPeriodByProposal(ctx context.Context, proposalID uint) (*models.VotingPeriod, error) {
	var period models.VotingPeriod
	if err := r.db.WithContext(ctx).
		Where("proposal_id = ?", proposalID).
		First(&period).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("voting period not found")
		}
		r.logger.Error("Failed to get voting period by proposal", map[string]interface{}{
			"error":       err.Error(),
			"proposal_id": proposalID,
		})
		return nil, err
	}

	return &period, nil
}

// GetActiveVotingPeriods retrieves all currently active voting periods
func (r *Repository) GetActiveVotingPeriods(ctx context.Context, clubID uint) ([]models.VotingPeriod, error) {
	var periods []models.VotingPeriod
	now := time.Now()

	query := r.db.WithContext(ctx).
		Where("is_active = ? AND start_time <= ? AND end_time > ?", true, now, now)

	if clubID > 0 {
		query = query.Where("club_id = ?", clubID)
	}

	if err := query.Preload("Proposal").Find(&periods).Error; err != nil {
		r.logger.Error("Failed to get active voting periods", map[string]interface{}{
			"error":   err.Error(),
			"club_id": clubID,
		})
		return nil, err
	}

	return periods, nil
}

// UpdateVotingPeriod updates a voting period
func (r *Repository) UpdateVotingPeriod(ctx context.Context, period *models.VotingPeriod) error {
	if err := r.db.WithContext(ctx).Save(period).Error; err != nil {
		r.logger.Error("Failed to update voting period", map[string]interface{}{
			"error":     err.Error(),
			"period_id": period.ID,
		})
		return err
	}

	r.logger.Info("Voting period updated successfully", map[string]interface{}{
		"period_id": period.ID,
		"is_active": period.IsActive,
	})

	return nil
}

// VotingRights operations

// CreateVotingRights creates voting rights for a member
func (r *Repository) CreateVotingRights(ctx context.Context, rights *models.VotingRights) error {
	if err := r.db.WithContext(ctx).Create(rights).Error; err != nil {
		r.logger.Error("Failed to create voting rights", map[string]interface{}{
			"error":     err.Error(),
			"member_id": rights.MemberID,
			"club_id":   rights.ClubID,
		})
		return err
	}

	r.logger.Info("Voting rights created successfully", map[string]interface{}{
		"rights_id": rights.ID,
		"member_id": rights.MemberID,
		"club_id":   rights.ClubID,
	})

	return nil
}

// GetVotingRights retrieves voting rights for a member in a club
func (r *Repository) GetVotingRights(ctx context.Context, memberID, clubID uint) (*models.VotingRights, error) {
	var rights models.VotingRights
	now := time.Now()

	if err := r.db.WithContext(ctx).
		Where("member_id = ? AND club_id = ? AND effective_from <= ? AND (effective_until IS NULL OR effective_until > ?)",
			memberID, clubID, now, now).
		First(&rights).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("voting rights not found")
		}
		r.logger.Error("Failed to get voting rights", map[string]interface{}{
			"error":     err.Error(),
			"member_id": memberID,
			"club_id":   clubID,
		})
		return nil, err
	}

	return &rights, nil
}

// UpdateVotingRights updates voting rights
func (r *Repository) UpdateVotingRights(ctx context.Context, rights *models.VotingRights) error {
	if err := r.db.WithContext(ctx).Save(rights).Error; err != nil {
		r.logger.Error("Failed to update voting rights", map[string]interface{}{
			"error":     err.Error(),
			"rights_id": rights.ID,
		})
		return err
	}

	r.logger.Info("Voting rights updated successfully", map[string]interface{}{
		"rights_id": rights.ID,
	})

	return nil
}

// GovernancePolicy operations

// CreateGovernancePolicy creates a new governance policy
func (r *Repository) CreateGovernancePolicy(ctx context.Context, policy *models.GovernancePolicy) error {
	if err := r.db.WithContext(ctx).Create(policy).Error; err != nil {
		r.logger.Error("Failed to create governance policy", map[string]interface{}{
			"error":   err.Error(),
			"name":    policy.Name,
			"club_id": policy.ClubID,
		})
		return err
	}

	r.logger.Info("Governance policy created successfully", map[string]interface{}{
		"policy_id": policy.ID,
		"name":      policy.Name,
		"club_id":   policy.ClubID,
	})

	return nil
}

// GetGovernancePolicy retrieves a governance policy by ID
func (r *Repository) GetGovernancePolicy(ctx context.Context, id uint) (*models.GovernancePolicy, error) {
	var policy models.GovernancePolicy
	if err := r.db.WithContext(ctx).First(&policy, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("governance policy not found")
		}
		r.logger.Error("Failed to get governance policy", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		return nil, err
	}

	return &policy, nil
}

// GetActiveGovernancePolicies retrieves active governance policies for a club
func (r *Repository) GetActiveGovernancePolicies(ctx context.Context, clubID uint) ([]models.GovernancePolicy, error) {
	var policies []models.GovernancePolicy
	now := time.Now()

	if err := r.db.WithContext(ctx).
		Where("club_id = ? AND is_active = ? AND effective_from <= ? AND (effective_until IS NULL OR effective_until > ?)",
			clubID, true, now, now).
		Order("name ASC").
		Find(&policies).Error; err != nil {
		r.logger.Error("Failed to get active governance policies", map[string]interface{}{
			"error":   err.Error(),
			"club_id": clubID,
		})
		return nil, err
	}

	return policies, nil
}

// VoteResult operations

// CreateOrUpdateVoteResult creates or updates vote results for a proposal
func (r *Repository) CreateOrUpdateVoteResult(ctx context.Context, result *models.VoteResult) error {
	result.CalculatedAt = time.Now()

	// Try to find existing result
	existingResult := &models.VoteResult{}
	err := r.db.WithContext(ctx).Where("proposal_id = ?", result.ProposalID).First(existingResult).Error

	if err == gorm.ErrRecordNotFound {
		// Create new result
		if err := r.db.WithContext(ctx).Create(result).Error; err != nil {
			r.logger.Error("Failed to create vote result", map[string]interface{}{
				"error":       err.Error(),
				"proposal_id": result.ProposalID,
			})
			return err
		}
	} else if err != nil {
		return err
	} else {
		// Update existing result
		result.ID = existingResult.ID
		if err := r.db.WithContext(ctx).Save(result).Error; err != nil {
			r.logger.Error("Failed to update vote result", map[string]interface{}{
				"error":       err.Error(),
				"proposal_id": result.ProposalID,
			})
			return err
		}
	}

	r.logger.Info("Vote result saved successfully", map[string]interface{}{
		"result_id":   result.ID,
		"proposal_id": result.ProposalID,
		"total_votes": result.TotalVotes,
		"passed":      result.Passed,
	})

	return nil
}

// GetVoteResult retrieves vote results for a proposal
func (r *Repository) GetVoteResult(ctx context.Context, proposalID uint) (*models.VoteResult, error) {
	var result models.VoteResult
	if err := r.db.WithContext(ctx).
		Where("proposal_id = ?", proposalID).
		Preload("Proposal").
		First(&result).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("vote result not found")
		}
		r.logger.Error("Failed to get vote result", map[string]interface{}{
			"error":       err.Error(),
			"proposal_id": proposalID,
		})
		return nil, err
	}

	return &result, nil
}

// Health check
func (r *Repository) HealthCheck(ctx context.Context) error {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Proposal{}).Count(&count).Error; err != nil {
		r.logger.Error("Repository health check failed", map[string]interface{}{
			"error": err.Error(),
		})
		return err
	}
	return nil
}