package repository

import (
	"testing"
	"time"

	"reciprocal-clubs-backend/services/auth-service/internal/models"
	"reciprocal-clubs-backend/services/auth-service/internal/testutil"
)

func TestUserCRUD(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	repo := NewRepository(tdb.DB)
	ctx := testutil.TestContext()

	club := testutil.CreateTestClub(tdb.DB, "Test Club", "test-club")

	// Create user
	user := &models.User{
		Email:       "jane@example.com",
		LastName: "Doe",
		FirstName: "Jane",
		HankoUserID: "hanko-123",
		Status:      models.UserStatusActive,
	}

	err := repo.CreateUser(ctx, user)
	testutil.AssertNoError(t, err, "create user")

	// Get user
	got, err := repo.GetUserByID(ctx, user.ID)
	testutil.AssertNoError(t, err, "get user by id")
	if got.Email != "jane@example.com" {
		t.Fatalf("expected email jane@example.com got %s", got.Email)
	}

	// Assign role
	role := &models.Role{Name: "member", Description: "Member", ClubID: club.ID}
	tdb.DB.Create(role)
	err = repo.AssignRoleToUser(ctx, user.ID, role.ID, club.ID)
	testutil.AssertNoError(t, err, "assign role")

	// Get roles
	roles, err := repo.GetUserRoles(ctx, user.ID, club.ID)
	testutil.AssertNoError(t, err, "get user roles")
	if len(roles) != 1 || roles[0].Name != "member" {
		t.Fatalf("expected 1 role=member got %+v", roles)
	}

	// Update user
	user.FirstName = "Jane Updated"
	err = repo.UpdateUser(ctx, user)
	testutil.AssertNoError(t, err, "update user")

	// Delete user
	err = repo.DeleteUser(ctx, user.ID)
	testutil.AssertNoError(t, err, "delete user")

	_, err = repo.GetUserByID(ctx, user.ID)
	if err == nil {
		t.Fatalf("expected error when getting deleted user")
	}
}

func TestSessionLifecycle(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	repo := NewRepository(tdb.DB)
	ctx := testutil.TestContext()

	club := testutil.CreateTestClub(tdb.DB, "Test Club", "test-club")
	user := testutil.CreateTestUser(tdb.DB, "john@example.com", "John", "hanko-xyz")

	// Create session
	session := &models.UserSession{
		UserID:    user.ID,
		ClubID:    club.ID,
		Token:     "token-abc",
		ExpiresAt: time.Now().Add(1 * time.Hour),
		Status:    models.SessionStatusActive,
	}
	err := repo.CreateSession(ctx, session)
	testutil.AssertNoError(t, err, "create session")

	// Get by token
	got, err := repo.GetSessionByToken(ctx, "token-abc")
	testutil.AssertNoError(t, err, "get session by token")
	if got.UserID != user.ID {
		t.Fatalf("expected session userID %d got %d", user.ID, got.UserID)
	}

	// Revoke
	err = repo.RevokeSession(ctx, session.ID)
	testutil.AssertNoError(t, err, "revoke session")

	// Should be revoked
	got, err = repo.GetSessionByToken(ctx, "token-abc")
	testutil.AssertNoError(t, err, "get session after revoke")
	if got.Status != models.SessionStatusRevoked {
		t.Fatalf("expected revoked got %s", got.Status)
	}
}

func TestPermissionsByRole(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	repo := NewRepository(tdb.DB)
	ctx := testutil.TestContext()

	club := testutil.CreateTestClub(tdb.DB, "Test Club", "test-club")

	read := &models.Permission{Name: "users.read", Description: "Read users", Resource: "users", Action: "read"}
	write := &models.Permission{Name: "users.write", Description: "Write users", Resource: "users", Action: "write"}
	tdb.DB.Create(&read)
	tdb.DB.Create(&write)

	role := &models.Role{Name: "admin", ClubID: club.ID}
	tdb.DB.Create(&role)

	tdb.DB.Create(&models.RolePermission{RoleID: role.ID, PermissionID: read.ID})
	tdb.DB.Create(&models.RolePermission{RoleID: role.ID, PermissionID: write.ID})

	perms, err := repo.GetPermissionsByRole(ctx, role.ID)
	testutil.AssertNoError(t, err, "get permissions by role")
	if len(perms) != 2 {
		t.Fatalf("expected 2 permissions got %d", len(perms))
	}
}
