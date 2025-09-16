package service

import (
	"testing"

	"reciprocal-clubs-backend/services/auth-service/internal/repository"
	"reciprocal-clubs-backend/services/auth-service/internal/testutil"
)

func TestRegistration(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	repo := repository.NewRepository(tdb.DB)
	hankoClient := testutil.NewMockHankoClient()
	logger := testutil.NewMockLogger()
	msgBus := testutil.NewMockMessageBus()
	
	service := NewAuthService(repo, hankoClient, logger, msgBus)
	ctx := testutil.TestContext()

	// Seed test data
	testData := tdb.SeedTestData(t)

	req := &RegisterRequest{
		Email:       "newuser@example.com",
		DisplayName: "New User",
		ClubSlug:    testData.Club.Slug,
	}

	resp, err := service.Register(ctx, req)
	testutil.AssertNoError(t, err, "register user")
	testutil.AssertEqual(t, "newuser@example.com", resp.User.Email, "user email")

	// Check that Hanko user was created
	testutil.AssertEqual(t, 1, hankoClient.GetUserCount(), "hanko user count")

	// Check that message was published
	messages := msgBus.GetMessages()
	testutil.AssertEqual(t, 1, len(messages), "message count")
}

func TestPasskeyLoginFlow(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	repo := repository.NewRepository(tdb.DB)
	hankoClient := testutil.NewMockHankoClient()
	logger := testutil.NewMockLogger()
	msgBus := testutil.NewMockMessageBus()
	
	service := NewAuthService(repo, hankoClient, logger, msgBus)
	ctx := testutil.TestContext()

	// Seed test data
	testData := tdb.SeedTestData(t)

	// Add user to Hanko mock
	hankoClient.AddUser(&hanko.User{
		ID:          testData.AdminUser.HankoUserID,
		Email:       testData.AdminUser.Email,
		DisplayName: testData.AdminUser.DisplayName,
	})

	// Test login initiation
	loginReq := &LoginRequest{
		Email:    testData.AdminUser.Email,
		ClubSlug: testData.Club.Slug,
	}

	initResp, err := service.InitiatePasskeyLogin(ctx, loginReq)
	testutil.AssertNoError(t, err, "initiate login")
	testutil.AssertNotEqual(t, "", initResp.Challenge, "challenge should not be empty")

	// Test login completion
	credentialData := map[string]interface{}{
		"id": "test-credential",
		"response": map[string]interface{}{
			"authenticatorData": "mock-data",
		},
	}

	completeResp, err := service.CompletePasskeyLogin(ctx, testData.Club.Slug, testData.AdminUser.HankoUserID, credentialData)
	testutil.AssertNoError(t, err, "complete login")
	testutil.AssertEqual(t, testData.AdminUser.Email, completeResp.User.Email, "user email")
	testutil.AssertNotEqual(t, "", completeResp.SessionToken, "session token should not be empty")

	// Test session validation
	user, err := service.ValidateSession(ctx, completeResp.SessionToken)
	testutil.AssertNoError(t, err, "validate session")
	testutil.AssertEqual(t, testData.AdminUser.Email, user.Email, "validated user email")
}

func TestGetUserWithRoles(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	repo := repository.NewRepository(tdb.DB)
	hankoClient := testutil.NewMockHankoClient()
	logger := testutil.NewMockLogger()
	msgBus := testutil.NewMockMessageBus()
	
	service := NewAuthService(repo, hankoClient, logger, msgBus)
	ctx := testutil.TestContext()

	// Seed test data
	testData := tdb.SeedTestData(t)

	userWithRoles, err := service.GetUserWithRoles(ctx, testData.Club.ID, testData.AdminUser.ID)
	testutil.AssertNoError(t, err, "get user with roles")
	
	testutil.AssertEqual(t, testData.AdminUser.Email, userWithRoles.User.Email, "user email")
	testutil.AssertTrue(t, len(userWithRoles.Roles) > 0, "user should have roles")
	testutil.AssertTrue(t, len(userWithRoles.Permissions) > 0, "user should have permissions")
}

func TestLogout(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	repo := repository.NewRepository(tdb.DB)
	hankoClient := testutil.NewMockHankoClient()
	logger := testutil.NewMockLogger()
	msgBus := testutil.NewMockMessageBus()
	
	service := NewAuthService(repo, hankoClient, logger, msgBus)
	ctx := testutil.TestContext()

	// Seed test data
	testData := tdb.SeedTestData(t)

	// Create a session
	session := testutil.CreateTestSession(tdb.DB, testData.AdminUser.ID, testData.Club.ID)

	err := service.Logout(ctx, testData.AdminUser.ID, testData.Club.ID, session.Token)
	testutil.AssertNoError(t, err, "logout")

	// Try to validate the session - should fail
	_, err = service.ValidateSession(ctx, session.Token)
	testutil.AssertError(t, err, "session should be invalid after logout")
}

func TestErrorHandling(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	repo := repository.NewRepository(tdb.DB)
	hankoClient := testutil.NewMockHankoClient()
	logger := testutil.NewMockLogger()
	msgBus := testutil.NewMockMessageBus()
	
	service := NewAuthService(repo, hankoClient, logger, msgBus)
	ctx := testutil.TestContext()

	// Test with Hanko failure
	hankoClient.SetShouldFail("CreateUser", true)

	req := &RegisterRequest{
		Email:       "fail@example.com",
		DisplayName: "Fail User",
		ClubSlug:    "nonexistent-club",
	}

	_, err := service.Register(ctx, req)
	testutil.AssertError(t, err, "registration should fail with hanko error")
}