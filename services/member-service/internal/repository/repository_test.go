package repository

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"reciprocal-clubs-backend/pkg/shared/config"
	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/services/member-service/internal/models"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Auto-migrate the schema
	err = db.AutoMigrate(
		&models.Member{},
		&models.MemberProfile{},
		&models.Address{},
		&models.EmergencyContact{},
		&models.MemberPreferences{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return db
}

func TestMemberRepository_CreateMember(t *testing.T) {
	db := setupTestDB(t)
	logger := logging.NewLogger(&config.LoggingConfig{Level: "debug"}, "test")

	repo := &memberRepository{
		db:     db,
		logger: logger,
	}

	ctx := context.Background()

	// Create test member
	member := &models.Member{
		ClubID:         1,
		UserID:         1,
		MembershipType: models.MembershipTypeRegular,
		Status:         models.MemberStatusActive,
	}

	err := repo.CreateMember(ctx, member)
	if err != nil {
		t.Errorf("CreateMember failed: %v", err)
	}

	// Verify member was created
	if member.ID == 0 {
		t.Error("Member ID should be set after creation")
	}

	if member.MemberNumber == "" {
		t.Error("Member number should be generated")
	}
}

func TestMemberRepository_GetMemberByID(t *testing.T) {
	db := setupTestDB(t)
	logger := logging.NewLogger(&config.LoggingConfig{Level: "debug"}, "test")

	repo := &memberRepository{
		db:     db,
		logger: logger,
	}

	ctx := context.Background()

	// Create test member
	member := &models.Member{
		ClubID:         1,
		UserID:         1,
		MembershipType: models.MembershipTypeRegular,
		Status:         models.MemberStatusActive,
	}

	err := repo.CreateMember(ctx, member)
	if err != nil {
		t.Fatalf("Failed to create test member: %v", err)
	}

	// Get member by ID
	retrieved, err := repo.GetMemberByID(ctx, member.ID)
	if err != nil {
		t.Errorf("GetMemberByID failed: %v", err)
	}

	if retrieved.ID != member.ID {
		t.Errorf("Expected member ID %d, got %d", member.ID, retrieved.ID)
	}

	if retrieved.UserID != member.UserID {
		t.Errorf("Expected user ID %d, got %d", member.UserID, retrieved.UserID)
	}
}

func TestMemberRepository_UpdateMember(t *testing.T) {
	db := setupTestDB(t)
	logger := logging.NewLogger(&config.LoggingConfig{Level: "debug"}, "test")

	repo := &memberRepository{
		db:     db,
		logger: logger,
	}

	ctx := context.Background()

	// Create test member
	member := &models.Member{
		ClubID:         1,
		UserID:         1,
		MembershipType: models.MembershipTypeRegular,
		Status:         models.MemberStatusActive,
	}

	err := repo.CreateMember(ctx, member)
	if err != nil {
		t.Fatalf("Failed to create test member: %v", err)
	}

	// Update member status
	member.Status = models.MemberStatusSuspended
	err = repo.UpdateMember(ctx, member)
	if err != nil {
		t.Errorf("UpdateMember failed: %v", err)
	}

	// Verify update
	retrieved, err := repo.GetMemberByID(ctx, member.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated member: %v", err)
	}

	if retrieved.Status != models.MemberStatusSuspended {
		t.Errorf("Expected status %s, got %s", models.MemberStatusSuspended, retrieved.Status)
	}
}