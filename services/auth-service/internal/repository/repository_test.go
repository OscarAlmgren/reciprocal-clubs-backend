package repository

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"reciprocal-clubs-backend/pkg/shared/config"
	"reciprocal-clubs-backend/pkg/shared/database"
	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/services/auth-service/internal/models"
)

func setupTestDB(t *testing.T) *database.Database {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Auto-migrate the schema
	err = db.AutoMigrate(
		&models.User{},
		&models.Club{},
		&models.Role{},
		&models.Permission{},
		&models.UserRole{},
		&models.RolePermission{},
		&models.UserSession{},
		&models.AuditLog{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return &database.Database{DB: db}
}

func setupTestRepository(t *testing.T) (*AuthRepository, *database.Database) {
	db := setupTestDB(t)
	logger := logging.NewLogger(&config.LoggingConfig{Level: "debug"}, "test")

	repo := NewAuthRepository(db, logger)
	return repo, db
}

func createTestClub(t *testing.T, repo *AuthRepository) *models.Club {
	club := &models.Club{
		Name:         "Test Club",
		Slug:         "test-club",
		Description:  "A test club",
		Status:       models.ClubStatusActive,
		ContactEmail: "test@testclub.com",
	}
	club.ClubID = 1 // For multi-tenant support

	ctx := context.Background()
	err := repo.CreateClub(ctx, club)
	if err != nil {
		t.Fatalf("Failed to create test club: %v", err)
	}

	return club
}

func createTestUser(t *testing.T, repo *AuthRepository, clubID uint) *models.User {
	user := &models.User{
		HankoUserID:   "hanko-123",
		Email:         "test@example.com",
		Username:      "testuser",
		FirstName:     "Test",
		LastName:      "User",
		Status:        models.UserStatusActive,
		EmailVerified: true,
	}
	user.ClubID = clubID

	ctx := context.Background()
	err := repo.CreateUser(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	return user
}

func createTestRole(t *testing.T, repo *AuthRepository, clubID uint) *models.Role {
	role := &models.Role{
		Name:        "test_role",
		Description: "A test role",
		IsSystem:    false,
	}
	role.ClubID = clubID

	ctx := context.Background()
	err := repo.CreateRole(ctx, role)
	if err != nil {
		t.Fatalf("Failed to create test role: %v", err)
	}

	return role
}

// User Repository Tests

func TestAuthRepository_CreateUser(t *testing.T) {
	repo, _ := setupTestRepository(t)
	club := createTestClub(t, repo)

	user := &models.User{
		HankoUserID:   "hanko-456",
		Email:         "newuser@example.com",
		Username:      "newuser",
		FirstName:     "New",
		LastName:      "User",
		Status:        models.UserStatusActive,
		EmailVerified: false,
	}
	user.ClubID = club.ID

	ctx := context.Background()
	err := repo.CreateUser(ctx, user)

	if err != nil {
		t.Errorf("CreateUser failed: %v", err)
	}

	if user.ID == 0 {
		t.Error("User ID should be set after creation")
	}
}

func TestAuthRepository_GetUserByID(t *testing.T) {
	repo, _ := setupTestRepository(t)
	club := createTestClub(t, repo)
	user := createTestUser(t, repo, club.ID)

	ctx := context.Background()
	retrieved, err := repo.GetUserByID(ctx, club.ID, user.ID)

	if err != nil {
		t.Errorf("GetUserByID failed: %v", err)
	}

	if retrieved.ID != user.ID {
		t.Errorf("Expected user ID %d, got %d", user.ID, retrieved.ID)
	}

	if retrieved.Email != user.Email {
		t.Errorf("Expected email %s, got %s", user.Email, retrieved.Email)
	}
}

func TestAuthRepository_GetUserByEmail(t *testing.T) {
	repo, _ := setupTestRepository(t)
	club := createTestClub(t, repo)
	user := createTestUser(t, repo, club.ID)

	ctx := context.Background()
	retrieved, err := repo.GetUserByEmail(ctx, club.ID, user.Email)

	if err != nil {
		t.Errorf("GetUserByEmail failed: %v", err)
	}

	if retrieved.Email != user.Email {
		t.Errorf("Expected email %s, got %s", user.Email, retrieved.Email)
	}
}

func TestAuthRepository_UpdateUser(t *testing.T) {
	repo, _ := setupTestRepository(t)
	club := createTestClub(t, repo)
	user := createTestUser(t, repo, club.ID)

	// Update user
	user.FirstName = "Updated"
	user.Status = models.UserStatusSuspended

	ctx := context.Background()
	err := repo.UpdateUser(ctx, user)

	if err != nil {
		t.Errorf("UpdateUser failed: %v", err)
	}

	// Verify update
	retrieved, err := repo.GetUserByID(ctx, club.ID, user.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated user: %v", err)
	}

	if retrieved.FirstName != "Updated" {
		t.Errorf("Expected first name 'Updated', got %s", retrieved.FirstName)
	}

	if retrieved.Status != models.UserStatusSuspended {
		t.Errorf("Expected status %s, got %s", models.UserStatusSuspended, retrieved.Status)
	}
}

// Role Repository Tests

func TestAuthRepository_CreateRole(t *testing.T) {
	repo, _ := setupTestRepository(t)
	club := createTestClub(t, repo)

	role := &models.Role{
		Name:        "new_role",
		Description: "A new role",
		IsSystem:    false,
	}
	role.ClubID = club.ID

	ctx := context.Background()
	err := repo.CreateRole(ctx, role)

	if err != nil {
		t.Errorf("CreateRole failed: %v", err)
	}

	if role.ID == 0 {
		t.Error("Role ID should be set after creation")
	}
}

func TestAuthRepository_AssignRole(t *testing.T) {
	repo, _ := setupTestRepository(t)
	club := createTestClub(t, repo)
	user := createTestUser(t, repo, club.ID)
	role := createTestRole(t, repo, club.ID)

	userRole := &models.UserRole{
		UserID:    user.ID,
		RoleID:    role.ID,
		GrantedAt: time.Now(),
		IsActive:  true,
	}
	userRole.ClubID = club.ID

	ctx := context.Background()
	err := repo.AssignRole(ctx, userRole)

	if err != nil {
		t.Errorf("AssignRole failed: %v", err)
	}

	// Verify role assignment
	roles, err := repo.GetUserRoles(ctx, club.ID, user.ID)
	if err != nil {
		t.Fatalf("Failed to get user roles: %v", err)
	}

	if len(roles) != 1 {
		t.Errorf("Expected 1 role, got %d", len(roles))
	}

	if roles[0].Name != role.Name {
		t.Errorf("Expected role %s, got %s", role.Name, roles[0].Name)
	}
}

// Session Repository Tests

func TestAuthRepository_CreateSession(t *testing.T) {
	repo, _ := setupTestRepository(t)
	club := createTestClub(t, repo)
	user := createTestUser(t, repo, club.ID)

	session := &models.UserSession{
		UserID:         user.ID,
		HankoSessionID: "hanko-session-123",
		IPAddress:      "192.168.1.1",
		UserAgent:      "TestAgent/1.0",
		ExpiresAt:      time.Now().Add(24 * time.Hour),
		IsActive:       true,
	}
	session.ClubID = club.ID

	ctx := context.Background()
	err := repo.CreateSession(ctx, session)

	if err != nil {
		t.Errorf("CreateSession failed: %v", err)
	}

	if session.ID == 0 {
		t.Error("Session ID should be set after creation")
	}
}

func TestAuthRepository_InvalidateSession(t *testing.T) {
	repo, _ := setupTestRepository(t)
	club := createTestClub(t, repo)
	user := createTestUser(t, repo, club.ID)

	session := &models.UserSession{
		UserID:         user.ID,
		HankoSessionID: "hanko-session-789",
		IPAddress:      "192.168.1.1",
		UserAgent:      "TestAgent/1.0",
		ExpiresAt:      time.Now().Add(24 * time.Hour),
		IsActive:       true,
	}
	session.ClubID = club.ID

	ctx := context.Background()
	repo.CreateSession(ctx, session)

	err := repo.InvalidateSession(ctx, club.ID, session.HankoSessionID)
	if err != nil {
		t.Errorf("InvalidateSession failed: %v", err)
	}

	// Verify session is invalidated
	retrieved, err := repo.GetSessionByHankoID(ctx, club.ID, session.HankoSessionID)
	if err != nil {
		t.Fatalf("Failed to retrieve session: %v", err)
	}

	if retrieved.IsActive {
		t.Error("Session should be inactive after invalidation")
	}

	if retrieved.LogoutAt == nil {
		t.Error("LogoutAt should be set after invalidation")
	}
}

// Health Check Test

func TestAuthRepository_HealthCheck(t *testing.T) {
	repo, _ := setupTestRepository(t)

	ctx := context.Background()
	err := repo.HealthCheck(ctx)

	if err != nil {
		t.Errorf("HealthCheck failed: %v", err)
	}
}

// Transaction Test

func TestAuthRepository_WithTransaction(t *testing.T) {
	repo, _ := setupTestRepository(t)
	club := createTestClub(t, repo)

	ctx := context.Background()
	err := repo.WithTransaction(ctx, func(txRepo *AuthRepository) error {
		// Create user within transaction
		user := &models.User{
			HankoUserID:   "tx-user",
			Email:         "tx@example.com",
			Username:      "txuser",
			FirstName:     "Transaction",
			LastName:      "User",
			Status:        models.UserStatusActive,
			EmailVerified: true,
		}
		user.ClubID = club.ID

		return txRepo.CreateUser(ctx, user)
	})

	if err != nil {
		t.Errorf("WithTransaction failed: %v", err)
	}

	// Verify user was created
	user, err := repo.GetUserByEmail(ctx, club.ID, "tx@example.com")
	if err != nil {
		t.Errorf("Failed to retrieve user created in transaction: %v", err)
	}

	if user.Username != "txuser" {
		t.Errorf("Expected username 'txuser', got %s", user.Username)
	}
}