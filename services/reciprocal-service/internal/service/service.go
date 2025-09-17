package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/messaging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	"reciprocal-clubs-backend/services/reciprocal-service/internal/models"
	"reciprocal-clubs-backend/services/reciprocal-service/internal/repository"
)

// ReciprocalService handles business logic for reciprocal agreements and visits
type ReciprocalService struct {
	repo       *repository.Repository
	logger     logging.Logger
	messaging  messaging.MessageBus
	monitoring *monitoring.Monitor
}

// NewReciprocalService creates a new reciprocal service
func NewReciprocalService(repo *repository.Repository, logger logging.Logger, messaging messaging.MessageBus, monitoring *monitoring.Monitor) *ReciprocalService {
	return &ReciprocalService{
		repo:       repo,
		logger:     logger,
		messaging:  messaging,
		monitoring: monitoring,
	}
}

// Agreement operations

// CreateAgreement creates a new reciprocal agreement
func (s *ReciprocalService) CreateAgreement(ctx context.Context, req *CreateAgreementRequest) (*models.Agreement, error) {
	agreement := &models.Agreement{
		ProposingClubID: req.ProposingClubID,
		TargetClubID:    req.TargetClubID,
		Title:           req.Title,
		Description:     req.Description,
		Terms:           req.Terms,
		Status:          models.AgreementStatusPending,
		ProposedAt:      time.Now(),
		ProposedByID:    req.ProposedByID,
	}

	if err := s.repo.CreateAgreement(ctx, agreement); err != nil {
		s.monitoring.RecordBusinessEvent("reciprocal_agreement_create_error", fmt.Sprintf("%d", req.ProposingClubID))
		return nil, err
	}

	s.monitoring.RecordBusinessEvent("reciprocal_agreement_created", fmt.Sprintf("%d", req.ProposingClubID))

	// Publish agreement created event
	s.publishAgreementEvent(ctx, "agreement.created", agreement)

	s.logger.Info("Agreement created", map[string]interface{}{
		"agreement_id":      agreement.ID,
		"proposing_club_id": agreement.ProposingClubID,
		"target_club_id":    agreement.TargetClubID,
	})

	return agreement, nil
}

// GetAgreementByID retrieves an agreement by ID
func (s *ReciprocalService) GetAgreementByID(ctx context.Context, id uint) (*models.Agreement, error) {
	agreement, err := s.repo.GetAgreementByID(ctx, id)
	if err != nil {
		s.monitoring.RecordBusinessEvent("reciprocal_agreement_get_error", "1")
		return nil, err
	}

	return agreement, nil
}

// GetAgreementsByClub retrieves agreements for a club
func (s *ReciprocalService) GetAgreementsByClub(ctx context.Context, clubID uint) ([]models.Agreement, error) {
	agreements, err := s.repo.GetAgreementsByClub(ctx, clubID)
	if err != nil {
		s.monitoring.RecordBusinessEvent("reciprocal_agreements_get_error", fmt.Sprintf("%d", clubID))
		return nil, err
	}

	return agreements, nil
}

// UpdateAgreementStatus updates the status of an agreement
func (s *ReciprocalService) UpdateAgreementStatus(ctx context.Context, id uint, newStatus string, reviewedByID string) (*models.Agreement, error) {
	agreement, err := s.repo.GetAgreementByID(ctx, id)
	if err != nil {
		return nil, err
	}

	newAgreementStatus := models.AgreementStatus(newStatus)
	if !agreement.CanTransitionTo(newAgreementStatus) {
		return nil, fmt.Errorf("cannot transition from %s to %s", agreement.Status, newStatus)
	}

	now := time.Now()
	agreement.Status = newAgreementStatus
	agreement.ReviewedAt = &now
	agreement.ReviewedByID = &reviewedByID

	if newAgreementStatus == models.AgreementStatusActive {
		agreement.ActivatedAt = &now
	}

	if err := s.repo.UpdateAgreement(ctx, agreement); err != nil {
		s.monitoring.RecordBusinessEvent("reciprocal_agreement_update_error", fmt.Sprintf("%d", agreement.ProposingClubID))
		return nil, err
	}

	s.monitoring.RecordBusinessEvent("reciprocal_agreement_status_updated", fmt.Sprintf("%d", agreement.ProposingClubID))

	// Publish agreement status updated event
	s.publishAgreementEvent(ctx, "agreement.status_updated", agreement)

	s.logger.Info("Agreement status updated", map[string]interface{}{
		"agreement_id": agreement.ID,
		"old_status":   agreement.Status,
		"new_status":   newStatus,
		"reviewed_by":  reviewedByID,
	})

	return agreement, nil
}

// Visit operations

// RequestVisit creates a new visit request
func (s *ReciprocalService) RequestVisit(ctx context.Context, req *RequestVisitRequest) (*models.Visit, error) {
	// Validate agreement is active
	agreement, err := s.repo.GetAgreementByID(ctx, req.AgreementID)
	if err != nil {
		return nil, err
	}

	if !agreement.IsActive() {
		return nil, fmt.Errorf("agreement is not active")
	}

	// Check for restrictions
	restrictions, err := s.repo.GetActiveRestrictionsForMember(ctx, req.MemberID, req.AgreementID)
	if err != nil {
		return nil, err
	}

	for _, restriction := range restrictions {
		if restriction.RestrictionType == models.RestrictionTypeBlacklist {
			return nil, fmt.Errorf("member is blacklisted from visiting")
		}
	}

	// Generate verification code
	verificationCode, err := s.generateVerificationCode()
	if err != nil {
		return nil, err
	}

	visit := &models.Visit{
		AgreementID:      req.AgreementID,
		MemberID:         req.MemberID,
		VisitingClubID:   req.VisitingClubID,
		HomeClubID:       req.HomeClubID,
		VisitDate:        req.VisitDate,
		Purpose:          req.Purpose,
		GuestCount:       req.GuestCount,
		Status:           models.VisitStatusPending,
		VerificationCode: verificationCode,
		EstimatedCost:    req.EstimatedCost,
		Currency:         req.Currency,
	}

	// Generate QR code data
	visit.QRCodeData = s.generateQRCodeData(visit)

	if err := s.repo.CreateVisit(ctx, visit); err != nil {
		s.monitoring.RecordBusinessEvent("reciprocal_visit_create_error", fmt.Sprintf("%d", req.VisitingClubID))
		return nil, err
	}

	s.monitoring.RecordBusinessEvent("reciprocal_visit_requested", fmt.Sprintf("%d", req.VisitingClubID))

	// Publish visit requested event
	s.publishVisitEvent(ctx, "visit.requested", visit)

	s.logger.Info("Visit requested", map[string]interface{}{
		"visit_id":         visit.ID,
		"member_id":        visit.MemberID,
		"visiting_club_id": visit.VisitingClubID,
		"visit_date":       visit.VisitDate,
	})

	return visit, nil
}

// ConfirmVisit confirms a pending visit
func (s *ReciprocalService) ConfirmVisit(ctx context.Context, id uint, confirmedByID string) (*models.Visit, error) {
	visit, err := s.repo.GetVisitByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if !visit.CanTransitionTo(models.VisitStatusConfirmed) {
		return nil, fmt.Errorf("cannot confirm visit in status %s", visit.Status)
	}

	visit.Status = models.VisitStatusConfirmed
	visit.VerifiedBy = &confirmedByID
	now := time.Now()
	visit.VerifiedAt = &now

	if err := s.repo.UpdateVisit(ctx, visit); err != nil {
		s.monitoring.RecordBusinessEvent("reciprocal_visit_confirm_error", fmt.Sprintf("%d", visit.VisitingClubID))
		return nil, err
	}

	s.monitoring.RecordBusinessEvent("reciprocal_visit_confirmed", fmt.Sprintf("%d", visit.VisitingClubID))

	// Publish visit confirmed event
	s.publishVisitEvent(ctx, "visit.confirmed", visit)

	s.logger.Info("Visit confirmed", map[string]interface{}{
		"visit_id":     visit.ID,
		"confirmed_by": confirmedByID,
	})

	return visit, nil
}

// CheckInVisit checks in a member for their visit
func (s *ReciprocalService) CheckInVisit(ctx context.Context, verificationCode string) (*models.Visit, error) {
	visit, err := s.repo.GetVisitByVerificationCode(ctx, verificationCode)
	if err != nil {
		s.monitoring.RecordBusinessEvent("reciprocal_visit_checkin_error", "1")
		return nil, err
	}

	if !visit.CanTransitionTo(models.VisitStatusCheckedIn) {
		return nil, fmt.Errorf("cannot check in visit in status %s", visit.Status)
	}

	now := time.Now()
	visit.Status = models.VisitStatusCheckedIn
	visit.CheckInTime = &now

	if err := s.repo.UpdateVisit(ctx, visit); err != nil {
		s.monitoring.RecordBusinessEvent("reciprocal_visit_checkin_error", fmt.Sprintf("%d", visit.VisitingClubID))
		return nil, err
	}

	s.monitoring.RecordBusinessEvent("reciprocal_visit_checkedin", fmt.Sprintf("%d", visit.VisitingClubID))

	// Publish visit checked in event
	s.publishVisitEvent(ctx, "visit.checked_in", visit)

	s.logger.Info("Visit checked in", map[string]interface{}{
		"visit_id":          visit.ID,
		"verification_code": verificationCode,
		"check_in_time":     now,
	})

	return visit, nil
}

// CheckOutVisit checks out a member from their visit
func (s *ReciprocalService) CheckOutVisit(ctx context.Context, verificationCode string, actualCost *float64) (*models.Visit, error) {
	visit, err := s.repo.GetVisitByVerificationCode(ctx, verificationCode)
	if err != nil {
		return nil, err
	}

	if !visit.CanTransitionTo(models.VisitStatusCompleted) {
		return nil, fmt.Errorf("cannot check out visit in status %s", visit.Status)
	}

	now := time.Now()
	visit.Status = models.VisitStatusCompleted
	visit.CheckOutTime = &now

	if actualCost != nil {
		visit.ActualCost = actualCost
	}

	// Calculate duration
	if duration := visit.CalculateDuration(); duration != nil {
		visit.Duration = duration
	}

	if err := s.repo.UpdateVisit(ctx, visit); err != nil {
		s.monitoring.RecordBusinessEvent("reciprocal_visit_checkout_error", fmt.Sprintf("%d", visit.VisitingClubID))
		return nil, err
	}

	s.monitoring.RecordBusinessEvent("reciprocal_visit_completed", fmt.Sprintf("%d", visit.VisitingClubID))

	// Publish visit completed event
	s.publishVisitEvent(ctx, "visit.completed", visit)

	s.logger.Info("Visit checked out", map[string]interface{}{
		"visit_id":           visit.ID,
		"verification_code":  verificationCode,
		"check_out_time":     now,
		"duration_minutes":   visit.Duration,
		"actual_cost":        visit.ActualCost,
	})

	return visit, nil
}

// GetVisitByID retrieves a visit by ID
func (s *ReciprocalService) GetVisitByID(ctx context.Context, id uint) (*models.Visit, error) {
	visit, err := s.repo.GetVisitByID(ctx, id)
	if err != nil {
		s.monitoring.RecordBusinessEvent("reciprocal_visit_get_error", "1")
		return nil, err
	}

	return visit, nil
}

// GetMemberVisits retrieves visits for a member
func (s *ReciprocalService) GetMemberVisits(ctx context.Context, memberID uint, limit, offset int) ([]models.Visit, error) {
	visits, err := s.repo.GetVisitsByMember(ctx, memberID, limit, offset)
	if err != nil {
		s.monitoring.RecordBusinessEvent("reciprocal_member_visits_error", fmt.Sprintf("%d", memberID))
		return nil, err
	}

	return visits, nil
}

// GetClubVisits retrieves visits for a club
func (s *ReciprocalService) GetClubVisits(ctx context.Context, clubID uint, limit, offset int) ([]models.Visit, error) {
	visits, err := s.repo.GetVisitsByClub(ctx, clubID, limit, offset)
	if err != nil {
		s.monitoring.RecordBusinessEvent("reciprocal_club_visits_error", fmt.Sprintf("%d", clubID))
		return nil, err
	}

	return visits, nil
}

// GetMemberVisitStats retrieves visit statistics for a member
func (s *ReciprocalService) GetMemberVisitStats(ctx context.Context, memberID uint, clubID uint, year int, month int) (*models.VisitStats, error) {
	stats, err := s.repo.GetMemberVisitStats(ctx, memberID, clubID, year, month)
	if err != nil {
		s.monitoring.RecordBusinessEvent("reciprocal_visit_stats_error", fmt.Sprintf("%d", clubID))
		return nil, err
	}

	return stats, nil
}

// Helper methods

func (s *ReciprocalService) generateVerificationCode() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (s *ReciprocalService) generateQRCodeData(visit *models.Visit) string {
	return fmt.Sprintf("reciprocal-visit:%s:%d:%s", visit.VerificationCode, visit.ID, visit.VisitDate.Format("2006-01-02"))
}

func (s *ReciprocalService) publishAgreementEvent(ctx context.Context, eventType string, agreement *models.Agreement) {
	data := map[string]interface{}{
		"agreement_id":      agreement.ID,
		"proposing_club_id": agreement.ProposingClubID,
		"target_club_id":    agreement.TargetClubID,
		"status":            agreement.Status,
		"timestamp":         time.Now(),
	}

	jsonData, _ := json.Marshal(data)
	if err := s.messaging.Publish(ctx, eventType, jsonData); err != nil {
		s.logger.Error("Failed to publish agreement event", map[string]interface{}{
			"error":        err.Error(),
			"event_type":   eventType,
			"agreement_id": agreement.ID,
		})
	}
}

func (s *ReciprocalService) publishVisitEvent(ctx context.Context, eventType string, visit *models.Visit) {
	data := map[string]interface{}{
		"visit_id":         visit.ID,
		"agreement_id":     visit.AgreementID,
		"member_id":        visit.MemberID,
		"visiting_club_id": visit.VisitingClubID,
		"home_club_id":     visit.HomeClubID,
		"status":           visit.Status,
		"visit_date":       visit.VisitDate,
		"timestamp":        time.Now(),
	}

	jsonData, _ := json.Marshal(data)
	if err := s.messaging.Publish(ctx, eventType, jsonData); err != nil {
		s.logger.Error("Failed to publish visit event", map[string]interface{}{
			"error":      err.Error(),
			"event_type": eventType,
			"visit_id":   visit.ID,
		})
	}
}

// Request/Response types

type CreateAgreementRequest struct {
	ProposingClubID uint                   `json:"proposing_club_id" validate:"required"`
	TargetClubID    uint                   `json:"target_club_id" validate:"required"`
	Title           string                 `json:"title" validate:"required"`
	Description     string                 `json:"description"`
	Terms           models.AgreementTerms  `json:"terms" validate:"required"`
	ProposedByID    string                 `json:"proposed_by_id" validate:"required"`
}

type RequestVisitRequest struct {
	AgreementID    uint      `json:"agreement_id" validate:"required"`
	MemberID       uint      `json:"member_id" validate:"required"`
	VisitingClubID uint      `json:"visiting_club_id" validate:"required"`
	HomeClubID     uint      `json:"home_club_id" validate:"required"`
	VisitDate      time.Time `json:"visit_date" validate:"required"`
	Purpose        string    `json:"purpose"`
	GuestCount     int       `json:"guest_count"`
	EstimatedCost  float64   `json:"estimated_cost"`
	Currency       string    `json:"currency"`
}