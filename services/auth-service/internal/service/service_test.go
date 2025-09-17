package service

import (
	"testing"

	"reciprocal-clubs-backend/services/auth-service/internal/testutil"
)

func TestServiceSetup(t *testing.T) {
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

	// Test that service dependencies can be set up correctly
	t.Logf("Service test setup completed successfully with club ID: %d, user ID: %d",
		testData.Club.ID, testData.AdminUser.ID)
}