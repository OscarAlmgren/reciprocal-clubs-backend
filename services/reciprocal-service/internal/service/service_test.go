package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"reciprocal-clubs-backend/services/reciprocal-service/internal/models"
)

// Mock repository interface
type MockRepository interface {
	CreateAgreement(ctx context.Context, agreement *models.Agreement) error
	GetAgreementByID(ctx context.Context, id uint) (*models.Agreement, error)
	GetAgreementsByClub(ctx context.Context, clubID uint) ([]models.Agreement, error)
	UpdateAgreement(ctx context.Context, agreement *models.Agreement) error
	CreateVisit(ctx context.Context, visit *models.Visit) error
	GetVisitByID(ctx context.Context, id uint) (*models.Visit, error)
	GetVisitByVerificationCode(ctx context.Context, code string) (*models.Visit, error)
	GetVisitsByMember(ctx context.Context, memberID uint, limit, offset int) ([]models.Visit, error)
	GetVisitsByClub(ctx context.Context, clubID uint, limit, offset int) ([]models.Visit, error)
	UpdateVisit(ctx context.Context, visit *models.Visit) error
	GetMemberVisitStats(ctx context.Context, memberID uint, clubID uint, year int, month int) (*models.VisitStats, error)
	GetActiveRestrictionsForMember(ctx context.Context, memberID uint, agreementID uint) ([]models.VisitRestriction, error)
}

// Mock repository for testing
type mockRepository struct {
	agreements    map[uint]*models.Agreement
	visits        map[uint]*models.Visit
	restrictions  []models.VisitRestriction
	nextID        uint
	shouldError   bool
	errorMessage  string
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		agreements:   make(map[uint]*models.Agreement),
		visits:       make(map[uint]*models.Visit),
		restrictions: []models.VisitRestriction{},
		nextID:       1,
	}
}

func (m *mockRepository) setError(message string) {
	m.shouldError = true
	m.errorMessage = message
}

func (m *mockRepository) clearError() {
	m.shouldError = false
	m.errorMessage = ""
}

// Agreement repository methods
func (m *mockRepository) CreateAgreement(ctx context.Context, agreement *models.Agreement) error {
	if m.shouldError {
		return errors.New(m.errorMessage)
	}
	agreement.ID = m.nextID
	m.nextID++
	m.agreements[agreement.ID] = agreement
	return nil
}

func (m *mockRepository) GetAgreementByID(ctx context.Context, id uint) (*models.Agreement, error) {
	if m.shouldError {
		return nil, errors.New(m.errorMessage)
	}
	agreement, exists := m.agreements[id]
	if !exists {
		return nil, errors.New("agreement not found")
	}
	return agreement, nil
}

func (m *mockRepository) GetAgreementsByClub(ctx context.Context, clubID uint) ([]models.Agreement, error) {
	if m.shouldError {
		return nil, errors.New(m.errorMessage)
	}
	var result []models.Agreement
	for _, agreement := range m.agreements {
		if agreement.ProposingClubID == clubID || agreement.TargetClubID == clubID {
			result = append(result, *agreement)
		}
	}
	return result, nil
}

func (m *mockRepository) UpdateAgreement(ctx context.Context, agreement *models.Agreement) error {
	if m.shouldError {
		return errors.New(m.errorMessage)
	}
	m.agreements[agreement.ID] = agreement
	return nil
}

// Visit repository methods
func (m *mockRepository) CreateVisit(ctx context.Context, visit *models.Visit) error {
	if m.shouldError {
		return errors.New(m.errorMessage)
	}
	visit.ID = m.nextID
	m.nextID++
	m.visits[visit.ID] = visit
	return nil
}

func (m *mockRepository) GetVisitByID(ctx context.Context, id uint) (*models.Visit, error) {
	if m.shouldError {
		return nil, errors.New(m.errorMessage)
	}
	visit, exists := m.visits[id]
	if !exists {
		return nil, errors.New("visit not found")
	}
	return visit, nil
}

func (m *mockRepository) GetVisitByVerificationCode(ctx context.Context, code string) (*models.Visit, error) {
	if m.shouldError {
		return nil, errors.New(m.errorMessage)
	}
	for _, visit := range m.visits {
		if visit.VerificationCode == code {
			return visit, nil
		}
	}
	return nil, errors.New("visit not found")
}

func (m *mockRepository) GetVisitsByMember(ctx context.Context, memberID uint, limit, offset int) ([]models.Visit, error) {
	if m.shouldError {
		return nil, errors.New(m.errorMessage)
	}
	var result []models.Visit
	for _, visit := range m.visits {
		if visit.MemberID == memberID {
			result = append(result, *visit)
		}
	}
	return result, nil
}

func (m *mockRepository) GetVisitsByClub(ctx context.Context, clubID uint, limit, offset int) ([]models.Visit, error) {
	if m.shouldError {
		return nil, errors.New(m.errorMessage)
	}
	var result []models.Visit
	for _, visit := range m.visits {
		if visit.VisitingClubID == clubID {
			result = append(result, *visit)
		}
	}
	return result, nil
}

func (m *mockRepository) UpdateVisit(ctx context.Context, visit *models.Visit) error {
	if m.shouldError {
		return errors.New(m.errorMessage)
	}
	m.visits[visit.ID] = visit
	return nil
}

func (m *mockRepository) GetMemberVisitStats(ctx context.Context, memberID uint, clubID uint, year int, month int) (*models.VisitStats, error) {
	if m.shouldError {
		return nil, errors.New(m.errorMessage)
	}
	return &models.VisitStats{
		MemberID:      memberID,
		ClubID:        clubID,
		Year:          year,
		Month:         month,
		VisitCount:    5,
		TotalDuration: 300,
		TotalCost:     150.0,
		AverageRating: 4.5,
	}, nil
}

func (m *mockRepository) GetActiveRestrictionsForMember(ctx context.Context, memberID uint, agreementID uint) ([]models.VisitRestriction, error) {
	if m.shouldError {
		return nil, errors.New(m.errorMessage)
	}
	var result []models.VisitRestriction
	for _, restriction := range m.restrictions {
		if restriction.AgreementID == agreementID &&
		   (restriction.MemberID == nil || *restriction.MemberID == memberID) &&
		   restriction.IsActive {
			result = append(result, restriction)
		}
	}
	return result, nil
}

// Mock logger
type mockLogger struct{}

func (m *mockLogger) Debug(msg string, fields map[string]interface{}) {}
func (m *mockLogger) Info(msg string, fields map[string]interface{})  {}
func (m *mockLogger) Warn(msg string, fields map[string]interface{})  {}
func (m *mockLogger) Error(msg string, fields map[string]interface{}) {}
func (m *mockLogger) Fatal(msg string, fields map[string]interface{}) {}
func (m *mockLogger) With(fields map[string]interface{}) interface{} { return m }

// Mock messaging
type mockMessaging struct{}

func (m *mockMessaging) Publish(ctx context.Context, subject string, data []byte) error {
	return nil
}

func (m *mockMessaging) Subscribe(subject string, handler func([]byte)) error {
	return nil
}

func (m *mockMessaging) Close() error {
	return nil
}

func (m *mockMessaging) HealthCheck() error {
	return nil
}

// Mock monitoring
type mockMonitoring struct{}

func (m *mockMonitoring) RecordBusinessEvent(event, value string) {}
func (m *mockMonitoring) RecordHTTPRequest(method, path string, statusCode int, duration time.Duration) {}

// Test helper to create service with mocks
func createTestService() (*ReciprocalService, *mockRepository) {
	repo := newMockRepository()
	logger := &mockLogger{}
	messaging := &mockMessaging{}
	monitoring := &mockMonitoring{}

	service := NewReciprocalService(repo, logger, messaging, monitoring)
	return service, repo
}

func TestReciprocalService_CreateAgreement(t *testing.T) {
	service, repo := createTestService()
	ctx := context.Background()

	req := &CreateAgreementRequest{
		ProposingClubID: 1,
		TargetClubID:    2,
		Title:           "Test Agreement",
		Description:     "Test Description",
		Terms: models.AgreementTerms{
			MaxVisitsPerMonth: 5,
			MaxVisitsPerYear:  60,
		},
		ProposedByID: "user123",
	}

	t.Run("successful creation", func(t *testing.T) {
		agreement, err := service.CreateAgreement(ctx, req)
		if err != nil {
			t.Errorf("CreateAgreement() error = %v, want nil", err)
			return
		}

		if agreement.ID == 0 {
			t.Error("CreateAgreement() should set ID")
		}
		if agreement.Status != models.AgreementStatusPending {
			t.Errorf("CreateAgreement() status = %v, want %v", agreement.Status, models.AgreementStatusPending)
		}
		if agreement.ProposingClubID != req.ProposingClubID {
			t.Errorf("CreateAgreement() ProposingClubID = %v, want %v", agreement.ProposingClubID, req.ProposingClubID)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		repo.setError("database error")
		defer repo.clearError()

		_, err := service.CreateAgreement(ctx, req)
		if err == nil {
			t.Error("CreateAgreement() error = nil, want error")
		}
	})
}

func TestReciprocalService_UpdateAgreementStatus(t *testing.T) {
	service, repo := createTestService()
	ctx := context.Background()

	// Create test agreement
	agreement := &models.Agreement{
		ID:              1,
		ProposingClubID: 1,
		TargetClubID:    2,
		Status:          models.AgreementStatusPending,
	}
	repo.agreements[1] = agreement

	t.Run("valid status transition", func(t *testing.T) {
		updated, err := service.UpdateAgreementStatus(ctx, 1, "approved", "reviewer123")
		if err != nil {
			t.Errorf("UpdateAgreementStatus() error = %v, want nil", err)
			return
		}

		if updated.Status != models.AgreementStatusApproved {
			t.Errorf("UpdateAgreementStatus() status = %v, want %v", updated.Status, models.AgreementStatusApproved)
		}
		if updated.ReviewedByID == nil || *updated.ReviewedByID != "reviewer123" {
			t.Error("UpdateAgreementStatus() should set ReviewedByID")
		}
	})

	t.Run("invalid status transition", func(t *testing.T) {
		_, err := service.UpdateAgreementStatus(ctx, 1, "expired", "reviewer123")
		if err == nil {
			t.Error("UpdateAgreementStatus() error = nil, want error for invalid transition")
		}
	})

	t.Run("agreement not found", func(t *testing.T) {
		_, err := service.UpdateAgreementStatus(ctx, 999, "approved", "reviewer123")
		if err == nil {
			t.Error("UpdateAgreementStatus() error = nil, want error for not found")
		}
	})
}

func TestReciprocalService_RequestVisit(t *testing.T) {
	service, repo := createTestService()
	ctx := context.Background()

	// Create active agreement
	agreement := &models.Agreement{
		ID:              1,
		ProposingClubID: 1,
		TargetClubID:    2,
		Status:          models.AgreementStatusActive,
	}
	repo.agreements[1] = agreement

	req := &RequestVisitRequest{
		AgreementID:    1,
		MemberID:       123,
		VisitingClubID: 2,
		HomeClubID:     1,
		VisitDate:      time.Now().Add(24 * time.Hour),
		Purpose:        "Business meeting",
		GuestCount:     2,
		EstimatedCost:  100.0,
		Currency:       "USD",
	}

	t.Run("successful visit request", func(t *testing.T) {
		visit, err := service.RequestVisit(ctx, req)
		if err != nil {
			t.Errorf("RequestVisit() error = %v, want nil", err)
			return
		}

		if visit.ID == 0 {
			t.Error("RequestVisit() should set ID")
		}
		if visit.Status != models.VisitStatusPending {
			t.Errorf("RequestVisit() status = %v, want %v", visit.Status, models.VisitStatusPending)
		}
		if visit.VerificationCode == "" {
			t.Error("RequestVisit() should generate verification code")
		}
		if visit.QRCodeData == "" {
			t.Error("RequestVisit() should generate QR code data")
		}
	})

	t.Run("inactive agreement", func(t *testing.T) {
		agreement.Status = models.AgreementStatusPending
		_, err := service.RequestVisit(ctx, req)
		if err == nil {
			t.Error("RequestVisit() error = nil, want error for inactive agreement")
		}
		agreement.Status = models.AgreementStatusActive // reset
	})

	t.Run("blacklisted member", func(t *testing.T) {
		memberID := uint(123)
		restriction := models.VisitRestriction{
			AgreementID:     1,
			MemberID:        &memberID,
			RestrictionType: models.RestrictionTypeBlacklist,
			IsActive:        true,
		}
		repo.restrictions = append(repo.restrictions, restriction)

		_, err := service.RequestVisit(ctx, req)
		if err == nil {
			t.Error("RequestVisit() error = nil, want error for blacklisted member")
		}

		repo.restrictions = []models.VisitRestriction{} // reset
	})
}

func TestReciprocalService_CheckInVisit(t *testing.T) {
	service, repo := createTestService()
	ctx := context.Background()

	// Create confirmed visit
	visit := &models.Visit{
		ID:               1,
		Status:           models.VisitStatusConfirmed,
		VerificationCode: "test-code-123",
	}
	repo.visits[1] = visit

	t.Run("successful check in", func(t *testing.T) {
		updated, err := service.CheckInVisit(ctx, "test-code-123")
		if err != nil {
			t.Errorf("CheckInVisit() error = %v, want nil", err)
			return
		}

		if updated.Status != models.VisitStatusCheckedIn {
			t.Errorf("CheckInVisit() status = %v, want %v", updated.Status, models.VisitStatusCheckedIn)
		}
		if updated.CheckInTime == nil {
			t.Error("CheckInVisit() should set CheckInTime")
		}
	})

	t.Run("invalid verification code", func(t *testing.T) {
		_, err := service.CheckInVisit(ctx, "invalid-code")
		if err == nil {
			t.Error("CheckInVisit() error = nil, want error for invalid code")
		}
	})

	t.Run("invalid status transition", func(t *testing.T) {
		visit.Status = models.VisitStatusCompleted
		_, err := service.CheckInVisit(ctx, "test-code-123")
		if err == nil {
			t.Error("CheckInVisit() error = nil, want error for invalid transition")
		}
	})
}

func TestReciprocalService_CheckOutVisit(t *testing.T) {
	service, repo := createTestService()
	ctx := context.Background()

	checkInTime := time.Now().Add(-2 * time.Hour)
	visit := &models.Visit{
		ID:               1,
		Status:           models.VisitStatusCheckedIn,
		VerificationCode: "test-code-123",
		CheckInTime:      &checkInTime,
	}
	repo.visits[1] = visit

	t.Run("successful check out", func(t *testing.T) {
		actualCost := 150.0
		updated, err := service.CheckOutVisit(ctx, "test-code-123", &actualCost)
		if err != nil {
			t.Errorf("CheckOutVisit() error = %v, want nil", err)
			return
		}

		if updated.Status != models.VisitStatusCompleted {
			t.Errorf("CheckOutVisit() status = %v, want %v", updated.Status, models.VisitStatusCompleted)
		}
		if updated.CheckOutTime == nil {
			t.Error("CheckOutVisit() should set CheckOutTime")
		}
		if updated.ActualCost == nil || *updated.ActualCost != actualCost {
			t.Errorf("CheckOutVisit() ActualCost = %v, want %v", updated.ActualCost, &actualCost)
		}
		if updated.Duration == nil {
			t.Error("CheckOutVisit() should calculate duration")
		}
	})

	t.Run("check out without actual cost", func(t *testing.T) {
		visit.Status = models.VisitStatusCheckedIn // reset
		updated, err := service.CheckOutVisit(ctx, "test-code-123", nil)
		if err != nil {
			t.Errorf("CheckOutVisit() error = %v, want nil", err)
			return
		}

		if updated.ActualCost != nil {
			t.Error("CheckOutVisit() ActualCost should be nil when not provided")
		}
	})
}

func TestReciprocalService_GetMemberVisitStats(t *testing.T) {
	service, repo := createTestService()
	ctx := context.Background()

	t.Run("successful stats retrieval", func(t *testing.T) {
		stats, err := service.GetMemberVisitStats(ctx, 123, 456, 2024, 3)
		if err != nil {
			t.Errorf("GetMemberVisitStats() error = %v, want nil", err)
			return
		}

		if stats.MemberID != 123 {
			t.Errorf("GetMemberVisitStats() MemberID = %v, want %v", stats.MemberID, 123)
		}
		if stats.ClubID != 456 {
			t.Errorf("GetMemberVisitStats() ClubID = %v, want %v", stats.ClubID, 456)
		}
		if stats.VisitCount != 5 {
			t.Errorf("GetMemberVisitStats() VisitCount = %v, want %v", stats.VisitCount, 5)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		repo.setError("database error")
		defer repo.clearError()

		_, err := service.GetMemberVisitStats(ctx, 123, 456, 2024, 3)
		if err == nil {
			t.Error("GetMemberVisitStats() error = nil, want error")
		}
	})
}