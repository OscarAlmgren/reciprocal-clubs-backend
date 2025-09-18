package repository

import (
	"context"
	"time"

	"reciprocal-clubs-backend/pkg/shared/database"
	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/services/reciprocal-service/internal/models"

	"gorm.io/gorm"
)

// Repository handles database operations for reciprocal service
type Repository struct {
	*database.BaseRepository
	db     *gorm.DB
	logger logging.Logger
}

// NewGORMRepository creates a new reciprocal repository
func NewGORMRepository(db *gorm.DB, logger logging.Logger) *Repository {
	return &Repository{
		db:     db,
		logger: logger,
	}
}

// Agreement operations

// CreateAgreement creates a new reciprocal agreement
func (r *Repository) CreateAgreement(ctx context.Context, agreement *models.Agreement) error {
	if err := r.db.WithContext(ctx).Create(agreement).Error; err != nil {
		r.logger.Error("Failed to create agreement", map[string]interface{}{
			"error":             err.Error(),
			"proposing_club_id": agreement.ProposingClubID,
			"target_club_id":    agreement.TargetClubID,
		})
		return err
	}

	r.logger.Info("Agreement created successfully", map[string]interface{}{
		"agreement_id":      agreement.ID,
		"proposing_club_id": agreement.ProposingClubID,
		"target_club_id":    agreement.TargetClubID,
		"status":            agreement.Status,
	})

	return nil
}

// GetAgreementByID retrieves an agreement by ID
func (r *Repository) GetAgreementByID(ctx context.Context, id uint) (*models.Agreement, error) {
	var agreement models.Agreement
	if err := r.db.WithContext(ctx).Preload("Visits").First(&agreement, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		r.logger.Error("Failed to get agreement", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		return nil, err
	}

	return &agreement, nil
}

// GetAgreementsByClub retrieves agreements for a specific club
func (r *Repository) GetAgreementsByClub(ctx context.Context, clubID uint) ([]models.Agreement, error) {
	var agreements []models.Agreement
	if err := r.db.WithContext(ctx).
		Where("proposing_club_id = ? OR target_club_id = ?", clubID, clubID).
		Find(&agreements).Error; err != nil {
		r.logger.Error("Failed to get agreements by club", map[string]interface{}{
			"error":   err.Error(),
			"club_id": clubID,
		})
		return nil, err
	}

	return agreements, nil
}

// UpdateAgreement updates an existing agreement
func (r *Repository) UpdateAgreement(ctx context.Context, agreement *models.Agreement) error {
	if err := r.db.WithContext(ctx).Save(agreement).Error; err != nil {
		r.logger.Error("Failed to update agreement", map[string]interface{}{
			"error":        err.Error(),
			"agreement_id": agreement.ID,
		})
		return err
	}

	r.logger.Info("Agreement updated successfully", map[string]interface{}{
		"agreement_id": agreement.ID,
		"status":       agreement.Status,
	})

	return nil
}

// GetActiveAgreements retrieves all active agreements
func (r *Repository) GetActiveAgreements(ctx context.Context) ([]models.Agreement, error) {
	var agreements []models.Agreement
	if err := r.db.WithContext(ctx).
		Where("status = ?", models.AgreementStatusActive).
		Find(&agreements).Error; err != nil {
		r.logger.Error("Failed to get active agreements", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	return agreements, nil
}

// Visit operations

// CreateVisit creates a new visit record
func (r *Repository) CreateVisit(ctx context.Context, visit *models.Visit) error {
	if err := r.db.WithContext(ctx).Create(visit).Error; err != nil {
		r.logger.Error("Failed to create visit", map[string]interface{}{
			"error":           err.Error(),
			"agreement_id":    visit.AgreementID,
			"member_id":       visit.MemberID,
			"visiting_club_id": visit.VisitingClubID,
		})
		return err
	}

	r.logger.Info("Visit created successfully", map[string]interface{}{
		"visit_id":         visit.ID,
		"agreement_id":     visit.AgreementID,
		"member_id":        visit.MemberID,
		"visiting_club_id": visit.VisitingClubID,
		"status":           visit.Status,
	})

	return nil
}

// GetVisitByID retrieves a visit by ID
func (r *Repository) GetVisitByID(ctx context.Context, id uint) (*models.Visit, error) {
	var visit models.Visit
	if err := r.db.WithContext(ctx).Preload("Agreement").First(&visit, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		r.logger.Error("Failed to get visit", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		return nil, err
	}

	return &visit, nil
}

// GetVisitByVerificationCode retrieves a visit by verification code
func (r *Repository) GetVisitByVerificationCode(ctx context.Context, code string) (*models.Visit, error) {
	var visit models.Visit
	if err := r.db.WithContext(ctx).
		Preload("Agreement").
		Where("verification_code = ?", code).
		First(&visit).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		r.logger.Error("Failed to get visit by verification code", map[string]interface{}{
			"error": err.Error(),
			"code":  code,
		})
		return nil, err
	}

	return &visit, nil
}

// GetVisitsByMember retrieves visits for a specific member
func (r *Repository) GetVisitsByMember(ctx context.Context, memberID uint, limit, offset int) ([]models.Visit, error) {
	var visits []models.Visit
	query := r.db.WithContext(ctx).Where("member_id = ?", memberID)

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Order("visit_date DESC").Find(&visits).Error; err != nil {
		r.logger.Error("Failed to get visits by member", map[string]interface{}{
			"error":     err.Error(),
			"member_id": memberID,
		})
		return nil, err
	}

	return visits, nil
}

// GetVisitsByClub retrieves visits for a specific club
func (r *Repository) GetVisitsByClub(ctx context.Context, clubID uint, limit, offset int) ([]models.Visit, error) {
	var visits []models.Visit
	query := r.db.WithContext(ctx).Where("visiting_club_id = ?", clubID)

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Order("visit_date DESC").Find(&visits).Error; err != nil {
		r.logger.Error("Failed to get visits by club", map[string]interface{}{
			"error":   err.Error(),
			"club_id": clubID,
		})
		return nil, err
	}

	return visits, nil
}

// UpdateVisit updates an existing visit
func (r *Repository) UpdateVisit(ctx context.Context, visit *models.Visit) error {
	if err := r.db.WithContext(ctx).Save(visit).Error; err != nil {
		r.logger.Error("Failed to update visit", map[string]interface{}{
			"error":    err.Error(),
			"visit_id": visit.ID,
		})
		return err
	}

	r.logger.Info("Visit updated successfully", map[string]interface{}{
		"visit_id": visit.ID,
		"status":   visit.Status,
	})

	return nil
}

// GetMemberVisitStats retrieves visit statistics for a member
func (r *Repository) GetMemberVisitStats(ctx context.Context, memberID uint, clubID uint, year int, month int) (*models.VisitStats, error) {
	var stats models.VisitStats

	query := r.db.WithContext(ctx).
		Model(&models.Visit{}).
		Select(`
			member_id,
			visiting_club_id as club_id,
			agreement_id,
			EXTRACT(month FROM visit_date) as month,
			EXTRACT(year FROM visit_date) as year,
			COUNT(*) as visit_count,
			COALESCE(SUM(duration), 0) as total_duration,
			COALESCE(SUM(actual_cost), 0) as total_cost,
			COALESCE(AVG(member_rating), 0) as average_rating,
			MAX(visit_date) as last_visit_date
		`).
		Where("member_id = ? AND visiting_club_id = ?", memberID, clubID).
		Where("EXTRACT(year FROM visit_date) = ?", year)

	if month > 0 {
		query = query.Where("EXTRACT(month FROM visit_date) = ?", month)
	}

	if err := query.Group("member_id, visiting_club_id, agreement_id, EXTRACT(month FROM visit_date), EXTRACT(year FROM visit_date)").
		First(&stats).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Return empty stats if no visits found
			return &models.VisitStats{
				MemberID: memberID,
				ClubID:   clubID,
				Month:    month,
				Year:     year,
			}, nil
		}
		r.logger.Error("Failed to get member visit stats", map[string]interface{}{
			"error":     err.Error(),
			"member_id": memberID,
			"club_id":   clubID,
		})
		return nil, err
	}

	return &stats, nil
}

// VisitRestriction operations

// CreateVisitRestriction creates a new visit restriction
func (r *Repository) CreateVisitRestriction(ctx context.Context, restriction *models.VisitRestriction) error {
	if err := r.db.WithContext(ctx).Create(restriction).Error; err != nil {
		r.logger.Error("Failed to create visit restriction", map[string]interface{}{
			"error":        err.Error(),
			"agreement_id": restriction.AgreementID,
			"member_id":    restriction.MemberID,
			"type":         restriction.RestrictionType,
		})
		return err
	}

	r.logger.Info("Visit restriction created successfully", map[string]interface{}{
		"restriction_id": restriction.ID,
		"agreement_id":   restriction.AgreementID,
		"member_id":      restriction.MemberID,
		"type":           restriction.RestrictionType,
	})

	return nil
}

// GetActiveRestrictionsForMember retrieves active restrictions for a member
func (r *Repository) GetActiveRestrictionsForMember(ctx context.Context, memberID uint, agreementID uint) ([]models.VisitRestriction, error) {
	var restrictions []models.VisitRestriction
	now := time.Now()

	if err := r.db.WithContext(ctx).
		Where("agreement_id = ? AND (member_id = ? OR member_id IS NULL)", agreementID, memberID).
		Where("is_active = true").
		Where("(start_date IS NULL OR start_date <= ?) AND (end_date IS NULL OR end_date >= ?)", now, now).
		Find(&restrictions).Error; err != nil {
		r.logger.Error("Failed to get active restrictions for member", map[string]interface{}{
			"error":        err.Error(),
			"member_id":    memberID,
			"agreement_id": agreementID,
		})
		return nil, err
	}

	return restrictions, nil
}

// UpdateVisitRestriction updates an existing visit restriction
func (r *Repository) UpdateVisitRestriction(ctx context.Context, restriction *models.VisitRestriction) error {
	if err := r.db.WithContext(ctx).Save(restriction).Error; err != nil {
		r.logger.Error("Failed to update visit restriction", map[string]interface{}{
			"error":          err.Error(),
			"restriction_id": restriction.ID,
		})
		return err
	}

	r.logger.Info("Visit restriction updated successfully", map[string]interface{}{
		"restriction_id": restriction.ID,
		"is_active":      restriction.IsActive,
	})

	return nil
}

// GetAgreementsByStatus retrieves agreements by status
func (r *Repository) GetAgreementsByStatus(ctx context.Context, status models.AgreementStatus, limit, offset int) ([]models.Agreement, error) {
	var agreements []models.Agreement
	query := r.db.WithContext(ctx).Where("status = ?", status)

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Order("created_at DESC").Find(&agreements).Error; err != nil {
		r.logger.Error("Failed to get agreements by status", map[string]interface{}{
			"error":  err.Error(),
			"status": status,
		})
		return nil, err
	}

	return agreements, nil
}

// GetVisitsByStatus retrieves visits by status
func (r *Repository) GetVisitsByStatus(ctx context.Context, status models.VisitStatus, limit, offset int) ([]models.Visit, error) {
	var visits []models.Visit
	query := r.db.WithContext(ctx).Where("status = ?", status)

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Order("visit_date DESC").Find(&visits).Error; err != nil {
		r.logger.Error("Failed to get visits by status", map[string]interface{}{
			"error":  err.Error(),
			"status": status,
		})
		return nil, err
	}

	return visits, nil
}

// GetUpcomingVisits retrieves upcoming visits for a club
func (r *Repository) GetUpcomingVisits(ctx context.Context, clubID uint, days int) ([]models.Visit, error) {
	var visits []models.Visit
	startDate := time.Now()
	endDate := startDate.AddDate(0, 0, days)

	if err := r.db.WithContext(ctx).
		Where("visiting_club_id = ?", clubID).
		Where("visit_date BETWEEN ? AND ?", startDate, endDate).
		Where("status IN ?", []models.VisitStatus{models.VisitStatusConfirmed, models.VisitStatusPending}).
		Order("visit_date ASC").
		Find(&visits).Error; err != nil {
		r.logger.Error("Failed to get upcoming visits", map[string]interface{}{
			"error":   err.Error(),
			"club_id": clubID,
			"days":    days,
		})
		return nil, err
	}

	return visits, nil
}

// HealthCheck performs a health check on the repository
func (r *Repository) HealthCheck(ctx context.Context) error {
	var result int
	if err := r.db.WithContext(ctx).Raw("SELECT 1").Scan(&result).Error; err != nil {
		r.logger.Error("Repository health check failed", map[string]interface{}{
			"error": err.Error(),
		})
		return err
	}

	return nil
}