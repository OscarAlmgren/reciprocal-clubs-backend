package repository

import (
	"testing"

	"reciprocal-clubs-backend/services/auth-service/internal/testutil"
)

func TestAuthRepository(t *testing.T) {
	tdb := testutil.NewTestDB(t)

	// Seed test data which creates clubs, users, roles, etc.
	testData := tdb.SeedTestData(t)

	// Verify test data was created correctly
	if testData.Club == nil {
		t.Fatal("Test club was not created")
	}

	if testData.AdminUser == nil {
		t.Fatal("Admin user was not created")
	}

	if testData.AdminUser.Email != "admin@test.com" {
		t.Fatalf("Expected admin email admin@test.com, got %s", testData.AdminUser.Email)
	}

	if len(testData.AdminUser.Roles) == 0 {
		t.Fatal("Admin user should have roles assigned")
	}

	// Test database seeding worked correctly
	t.Logf("Test completed successfully with club ID: %d, user ID: %d",
		testData.Club.ID, testData.AdminUser.ID)
}