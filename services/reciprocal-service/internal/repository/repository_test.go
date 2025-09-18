package repository

import (
	"context"
	"testing"
	"time"

	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/services/reciprocal-service/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Mock logger for testing
type mockLogger struct{}

func (m *mockLogger) Debug(msg string, fields map[string]interface{}) {}
func (m *mockLogger) Info(msg string, fields map[string]interface{})  {}
func (m *mockLogger) Warn(msg string, fields map[string]interface{})  {}
func (m *mockLogger) Error(msg string, fields map[string]interface{}) {}
func (m *mockLogger) Fatal(msg string, fields map[string]interface{}) {}
func (m *mockLogger) With(fields map[string]interface{}) logging.Logger { return m }

// Setup test database
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // Disable logging for tests
	})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Auto-migrate tables
	err = db.AutoMigrate(
		&models.Agreement{},
		&models.Visit{},
		&models.VisitRestriction{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return db
}

// Test helper to create repository with test database
func createTestRepository(t *testing.T) *Repository {
	db := setupTestDB(t)
	logger := &mockLogger{}
	return NewGORMRepository(db, logger)
}

func TestRepository_CreateAgreement(t *testing.T) {
	repo := createTestRepository(t)
	ctx := context.Background()

	agreement := &models.Agreement{
		ProposingClubID: 1,
		TargetClubID:    2,
		Title:           "Test Agreement",
		Description:     "Test Description",
		Terms: models.AgreementTerms{
			MaxVisitsPerMonth: 5,
			MaxVisitsPerYear:  60,
		},
		Status:       models.AgreementStatusPending,
		ProposedAt:   time.Now(),
		ProposedByID: "user123",
	}

	err := repo.CreateAgreement(ctx, agreement)
	if err != nil {
		t.Errorf("CreateAgreement() error = %v, want nil", err)
	}

	if agreement.ID == 0 {
		t.Error("CreateAgreement() should set ID")
	}
}

func TestRepository_GetAgreementByID(t *testing.T) {
	repo := createTestRepository(t)
	ctx := context.Background()

	// Create test agreement
	agreement := &models.Agreement{
		ProposingClubID: 1,
		TargetClubID:    2,
		Title:           "Test Agreement",
		Status:          models.AgreementStatusPending,
		ProposedAt:      time.Now(),
		ProposedByID:    "user123",
	}
	err := repo.CreateAgreement(ctx, agreement)
	if err != nil {
		t.Fatalf("Failed to create test agreement: %v", err)
	}

	t.Run("existing agreement", func(t *testing.T) {
		found, err := repo.GetAgreementByID(ctx, agreement.ID)
		if err != nil {
			t.Errorf("GetAgreementByID() error = %v, want nil", err)
			return
		}

		if found.ID != agreement.ID {
			t.Errorf("GetAgreementByID() ID = %v, want %v", found.ID, agreement.ID)
		}
		if found.Title != agreement.Title {
			t.Errorf("GetAgreementByID() Title = %v, want %v", found.Title, agreement.Title)
		}
	})

	t.Run("non-existing agreement", func(t *testing.T) {
		_, err := repo.GetAgreementByID(ctx, 999)
		if err == nil {
			t.Error("GetAgreementByID() error = nil, want error for non-existing agreement")
		}
	})
}

func TestRepository_GetAgreementsByClub(t *testing.T) {
	repo := createTestRepository(t)
	ctx := context.Background()

	// Create test agreements
	agreement1 := &models.Agreement{
		ProposingClubID: 1,
		TargetClubID:    2,
		Title:           "Agreement 1",
		Status:          models.AgreementStatusPending,
		ProposedAt:      time.Now(),
		ProposedByID:    "user123",
	}
	agreement2 := &models.Agreement{
		ProposingClubID: 2,
		TargetClubID:    3,
		Title:           "Agreement 2",
		Status:          models.AgreementStatusActive,
		ProposedAt:      time.Now(),
		ProposedByID:    "user456",
	}
	agreement3 := &models.Agreement{
		ProposingClubID: 3,
		TargetClubID:    1,
		Title:           "Agreement 3",
		Status:          models.AgreementStatusApproved,
		ProposedAt:      time.Now(),
		ProposedByID:    "user789",
	}

	for _, agreement := range []*models.Agreement{agreement1, agreement2, agreement3} {
		err := repo.CreateAgreement(ctx, agreement)
		if err != nil {
			t.Fatalf("Failed to create test agreement: %v", err)
		}
	}

	agreements, err := repo.GetAgreementsByClub(ctx, 1)
	if err != nil {
		t.Errorf("GetAgreementsByClub() error = %v, want nil", err)
		return
	}

	// Club 1 should have 2 agreements (as proposing club in agreement1, as target club in agreement3)
	if len(agreements) != 2 {
		t.Errorf("GetAgreementsByClub() returned %d agreements, want 2", len(agreements))
	}
}

func TestRepository_UpdateAgreement(t *testing.T) {
	repo := createTestRepository(t)
	ctx := context.Background()

	// Create test agreement
	agreement := &models.Agreement{
		ProposingClubID: 1,
		TargetClubID:    2,
		Title:           "Test Agreement",
		Status:          models.AgreementStatusPending,
		ProposedAt:      time.Now(),
		ProposedByID:    "user123",
	}
	err := repo.CreateAgreement(ctx, agreement)
	if err != nil {
		t.Fatalf("Failed to create test agreement: %v", err)
	}

	// Update the agreement
	agreement.Status = models.AgreementStatusApproved
	reviewedBy := "reviewer456"
	agreement.ReviewedByID = &reviewedBy
	now := time.Now()
	agreement.ReviewedAt = &now

	err = repo.UpdateAgreement(ctx, agreement)
	if err != nil {
		t.Errorf("UpdateAgreement() error = %v, want nil", err)
	}

	// Verify the update
	updated, err := repo.GetAgreementByID(ctx, agreement.ID)
	if err != nil {
		t.Fatalf("Failed to get updated agreement: %v", err)
	}

	if updated.Status != models.AgreementStatusApproved {
		t.Errorf("UpdateAgreement() status = %v, want %v", updated.Status, models.AgreementStatusApproved)
	}
}

func TestRepository_CreateVisit(t *testing.T) {
	repo := createTestRepository(t)
	ctx := context.Background()

	visit := &models.Visit{
		AgreementID:      1,
		MemberID:         123,
		VisitingClubID:   2,
		HomeClubID:       1,
		VisitDate:        time.Now().Add(24 * time.Hour),
		Purpose:          "Business meeting",
		GuestCount:       2,
		Status:           models.VisitStatusPending,
		VerificationCode: "test-code-123",
		EstimatedCost:    100.0,
		Currency:         "USD",
	}

	err := repo.CreateVisit(ctx, visit)
	if err != nil {
		t.Errorf("CreateVisit() error = %v, want nil", err)
	}

	if visit.ID == 0 {
		t.Error("CreateVisit() should set ID")
	}
}

func TestRepository_GetVisitByVerificationCode(t *testing.T) {
	repo := createTestRepository(t)
	ctx := context.Background()

	// Create test visit
	visit := &models.Visit{
		AgreementID:      1,
		MemberID:         123,
		VisitingClubID:   2,
		HomeClubID:       1,
		VisitDate:        time.Now().Add(24 * time.Hour),
		Status:           models.VisitStatusPending,
		VerificationCode: "unique-code-456",
		EstimatedCost:    100.0,
		Currency:         "USD",
	}
	err := repo.CreateVisit(ctx, visit)
	if err != nil {
		t.Fatalf("Failed to create test visit: %v", err)
	}

	t.Run("existing verification code", func(t *testing.T) {
		found, err := repo.GetVisitByVerificationCode(ctx, "unique-code-456")
		if err != nil {
			t.Errorf("GetVisitByVerificationCode() error = %v, want nil", err)
			return
		}

		if found.ID != visit.ID {
			t.Errorf("GetVisitByVerificationCode() ID = %v, want %v", found.ID, visit.ID)
		}
		if found.VerificationCode != visit.VerificationCode {
			t.Errorf("GetVisitByVerificationCode() VerificationCode = %v, want %v", found.VerificationCode, visit.VerificationCode)
		}
	})

	t.Run("non-existing verification code", func(t *testing.T) {
		_, err := repo.GetVisitByVerificationCode(ctx, "non-existing-code")
		if err == nil {
			t.Error("GetVisitByVerificationCode() error = nil, want error for non-existing code")
		}
	})
}

func TestRepository_GetVisitsByMember(t *testing.T) {
	repo := createTestRepository(t)
	ctx := context.Background()

	memberID := uint(123)

	// Create test visits
	visit1 := &models.Visit{
		AgreementID:      1,
		MemberID:         memberID,
		VisitingClubID:   2,
		HomeClubID:       1,
		VisitDate:        time.Now().Add(24 * time.Hour),
		Status:           models.VisitStatusPending,
		VerificationCode: "code-1",
	}
	visit2 := &models.Visit{
		AgreementID:      1,
		MemberID:         memberID,
		VisitingClubID:   3,
		HomeClubID:       1,
		VisitDate:        time.Now().Add(48 * time.Hour),
		Status:           models.VisitStatusConfirmed,
		VerificationCode: "code-2",
	}
	visit3 := &models.Visit{
		AgreementID:      1,
		MemberID:         456, // Different member
		VisitingClubID:   2,
		HomeClubID:       1,
		VisitDate:        time.Now().Add(72 * time.Hour),
		Status:           models.VisitStatusPending,
		VerificationCode: "code-3",
	}

	for _, visit := range []*models.Visit{visit1, visit2, visit3} {
		err := repo.CreateVisit(ctx, visit)
		if err != nil {
			t.Fatalf("Failed to create test visit: %v", err)
		}
	}

	visits, err := repo.GetVisitsByMember(ctx, memberID, 10, 0)
	if err != nil {
		t.Errorf("GetVisitsByMember() error = %v, want nil", err)
		return
	}

	// Should return 2 visits for memberID 123
	if len(visits) != 2 {
		t.Errorf("GetVisitsByMember() returned %d visits, want 2", len(visits))
	}
}

func TestRepository_GetVisitsByClub(t *testing.T) {
	repo := createTestRepository(t)
	ctx := context.Background()

	clubID := uint(2)

	// Create test visits
	visit1 := &models.Visit{
		AgreementID:      1,
		MemberID:         123,
		VisitingClubID:   clubID,
		HomeClubID:       1,
		VisitDate:        time.Now().Add(24 * time.Hour),
		Status:           models.VisitStatusPending,
		VerificationCode: "code-1",
	}
	visit2 := &models.Visit{
		AgreementID:      1,
		MemberID:         456,
		VisitingClubID:   clubID,
		HomeClubID:       1,
		VisitDate:        time.Now().Add(48 * time.Hour),
		Status:           models.VisitStatusConfirmed,
		VerificationCode: "code-2",
	}
	visit3 := &models.Visit{
		AgreementID:      1,
		MemberID:         789,
		VisitingClubID:   3, // Different club
		HomeClubID:       1,
		VisitDate:        time.Now().Add(72 * time.Hour),
		Status:           models.VisitStatusPending,
		VerificationCode: "code-3",
	}

	for _, visit := range []*models.Visit{visit1, visit2, visit3} {
		err := repo.CreateVisit(ctx, visit)
		if err != nil {
			t.Fatalf("Failed to create test visit: %v", err)
		}
	}

	visits, err := repo.GetVisitsByClub(ctx, clubID, 10, 0)
	if err != nil {
		t.Errorf("GetVisitsByClub() error = %v, want nil", err)
		return
	}

	// Should return 2 visits for clubID 2
	if len(visits) != 2 {
		t.Errorf("GetVisitsByClub() returned %d visits, want 2", len(visits))
	}
}

func TestRepository_UpdateVisit(t *testing.T) {
	repo := createTestRepository(t)
	ctx := context.Background()

	// Create test visit
	visit := &models.Visit{
		AgreementID:      1,
		MemberID:         123,
		VisitingClubID:   2,
		HomeClubID:       1,
		VisitDate:        time.Now().Add(24 * time.Hour),
		Status:           models.VisitStatusPending,
		VerificationCode: "test-code",
	}
	err := repo.CreateVisit(ctx, visit)
	if err != nil {
		t.Fatalf("Failed to create test visit: %v", err)
	}

	// Update the visit
	visit.Status = models.VisitStatusConfirmed
	confirmedBy := "staff123"
	visit.VerifiedBy = &confirmedBy
	now := time.Now()
	visit.VerifiedAt = &now

	err = repo.UpdateVisit(ctx, visit)
	if err != nil {
		t.Errorf("UpdateVisit() error = %v, want nil", err)
	}

	// Verify the update
	updated, err := repo.GetVisitByID(ctx, visit.ID)
	if err != nil {
		t.Fatalf("Failed to get updated visit: %v", err)
	}

	if updated.Status != models.VisitStatusConfirmed {
		t.Errorf("UpdateVisit() status = %v, want %v", updated.Status, models.VisitStatusConfirmed)
	}
}

func TestRepository_GetActiveRestrictionsForMember(t *testing.T) {
	repo := createTestRepository(t)
	ctx := context.Background()

	memberID := uint(123)
	agreementID := uint(1)

	// Create test restrictions
	restriction1 := &models.VisitRestriction{
		AgreementID:     agreementID,
		MemberID:        &memberID,
		RestrictionType: models.RestrictionTypeSuspension,
		IsActive:        true,
		AppliedByID:     "admin123",
		AppliedAt:       time.Now(),
		Reason:          "Test suspension",
	}

	restriction2 := &models.VisitRestriction{
		AgreementID:     agreementID,
		MemberID:        nil, // Applies to all members
		RestrictionType: models.RestrictionTypeLimitation,
		IsActive:        true,
		AppliedByID:     "admin456",
		AppliedAt:       time.Now(),
		Reason:          "General limitation",
	}

	restriction3 := &models.VisitRestriction{
		AgreementID:     agreementID,
		MemberID:        &memberID,
		RestrictionType: models.RestrictionTypeBlacklist,
		IsActive:        false, // Inactive
		AppliedByID:     "admin789",
		AppliedAt:       time.Now(),
		Reason:          "Inactive blacklist",
	}

	for _, restriction := range []*models.VisitRestriction{restriction1, restriction2, restriction3} {
		err := repo.CreateVisitRestriction(ctx, restriction)
		if err != nil {
			t.Fatalf("Failed to create test restriction: %v", err)
		}
	}

	restrictions, err := repo.GetActiveRestrictionsForMember(ctx, memberID, agreementID)
	if err != nil {
		t.Errorf("GetActiveRestrictionsForMember() error = %v, want nil", err)
		return
	}

	// Should return 2 active restrictions (specific member + general)
	if len(restrictions) != 2 {
		t.Errorf("GetActiveRestrictionsForMember() returned %d restrictions, want 2", len(restrictions))
	}
}

func TestRepository_HealthCheck(t *testing.T) {
	repo := createTestRepository(t)
	ctx := context.Background()

	err := repo.HealthCheck(ctx)
	if err != nil {
		t.Errorf("HealthCheck() error = %v, want nil", err)
	}
}