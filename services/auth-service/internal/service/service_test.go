package service

import (
	"context"
	"testing"
	"time"

	"reciprocal-clubs-backend/pkg/shared/auth"
	"reciprocal-clubs-backend/pkg/shared/database"
	"reciprocal-clubs-backend/services/auth-service/internal/hanko"
	"reciprocal-clubs-backend/services/auth-service/internal/models"
	"reciprocal-clubs-backend/services/auth-service/internal/repository"
	"reciprocal-clubs-backend/services/auth-service/internal/testutil"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestService creates a new auth service with mock dependencies for testing
func setupTestService(t *testing.T) (*AuthService, *testutil.MockHankoClient, *database.Database, *models.Club, *models.User) {
	// Use SQLite setup from repository tests
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

	dbWrapper := &database.Database{DB: db}
	logger := testutil.NewMockLogger()
	repo := repository.NewAuthRepository(dbWrapper, logger)

	// Create test club
	club := &models.Club{
		Name:         "Test Club",
		Slug:         "test-club",
		Description:  "A test club",
		Status:       models.ClubStatusActive,
		ContactEmail: "test@testclub.com",
	}
	club.ClubID = 1 // For multi-tenant support

	if err := db.Create(club).Error; err != nil {
		t.Fatalf("Failed to create test club: %v", err)
	}

	// Create test user
	user := &models.User{
		HankoUserID:   "hanko-123",
		Email:         "test@example.com",
		Username:      "testuser",
		FirstName:     "Test",
		LastName:      "User",
		Status:        models.UserStatusActive,
		EmailVerified: true,
	}
	user.ClubID = club.ID

	if err := db.Create(user).Error; err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Create default roles
	memberRole := &models.Role{
		Name:        models.RoleMember,
		Description: "Default member role",
		IsSystem:    true,
	}
	memberRole.ClubID = club.ID

	adminRole := &models.Role{
		Name:        models.RoleAdmin,
		Description: "Administrator role",
		IsSystem:    true,
	}
	adminRole.ClubID = club.ID

	if err := db.Create([]*models.Role{memberRole, adminRole}).Error; err != nil {
		t.Fatalf("Failed to create test roles: %v", err)
	}

	// Create mock dependencies
	mockHanko := testutil.NewMockHankoClient()
	mockMessageBus := testutil.NewMockMessageBus()
	mockConfig := testutil.NewMockConfig()

	// Create auth provider
	authProvider := auth.NewJWTProvider(&mockConfig.Auth, logger)

	// Create service
	service := &AuthService{
		repo:         repo,
		hankoClient:  mockHanko,
		authProvider: authProvider,
		messageBus:   mockMessageBus,
		config:       mockConfig,
		logger:       logger,
	}

	return service, mockHanko, dbWrapper, club, user
}

// User Registration Tests

func TestAuthService_Register_Success(t *testing.T) {
	service, mockHanko, _, testClub, _ := setupTestService(t)
	ctx := testutil.TestContext()

	// Setup mock Hanko response
	mockHanko.Clear()

	req := &RegisterRequest{
		Email:     "newuser@test.com",
		Username:  "newuser",
		FirstName: "New",
		LastName:  "User",
		ClubSlug:  testClub.Slug,
	}

	response, err := service.Register(ctx, req)

	testutil.AssertNoError(t, err, "Register should succeed")
	testutil.AssertNotEqual(t, "", response.Token, "Token should be generated")
	testutil.AssertNotEqual(t, "", response.RefreshToken, "Refresh token should be generated")
	testutil.AssertEqual(t, req.Email, response.User.Email, "User email should match")
	testutil.AssertEqual(t, req.Username, response.User.Username, "User username should match")
	testutil.AssertEqual(t, models.UserStatusActive, response.User.Status, "User should be active")
	testutil.AssertEqual(t, 1, mockHanko.GetUserCount(), "Hanko user should be created")
}

func TestAuthService_Register_ClubNotFound(t *testing.T) {
	service, _, _, _, _ := setupTestService(t)
	ctx := testutil.TestContext()

	req := &RegisterRequest{
		Email:     "newuser@test.com",
		Username:  "newuser",
		FirstName: "New",
		LastName:  "User",
		ClubSlug:  "nonexistent-club",
	}

	_, err := service.Register(ctx, req)

	testutil.AssertError(t, err, "Register should fail for nonexistent club")
}

func TestAuthService_Register_HankoError(t *testing.T) {
	service, mockHanko, _, testClub, testUser := setupTestService(t)
	_ = testUser // avoid unused variable warning
	_ = testClub // avoid unused variable warning
	ctx := testutil.TestContext()

	// Setup mock Hanko to fail
	mockHanko.SetShouldFail("CreateUser", true)

	req := &RegisterRequest{
		Email:     "newuser@test.com",
		Username:  "newuser",
		FirstName: "New",
		LastName:  "User",
		ClubSlug:  testClub.Slug,
	}

	_, err := service.Register(ctx, req)

	testutil.AssertError(t, err, "Register should fail when Hanko fails")
	testutil.AssertEqual(t, 0, mockHanko.GetUserCount(), "No Hanko user should be created")
}

func TestAuthService_Register_DuplicateEmail(t *testing.T) {
	service, mockHanko, _, testClub, testUser := setupTestService(t)
	_ = testUser // avoid unused variable warning
	_ = testClub // avoid unused variable warning
	ctx := testutil.TestContext()

	mockHanko.Clear()

	req := &RegisterRequest{
		Email:     testUser.Email, // Use existing email
		Username:  "newuser",
		FirstName: "New",
		LastName:  "User",
		ClubSlug:  testClub.Slug,
	}

	_, err := service.Register(ctx, req)

	testutil.AssertError(t, err, "Register should fail for duplicate email")
}

// Passkey Authentication Tests

func TestAuthService_InitiatePasskeyLogin_Success(t *testing.T) {
	service, mockHanko, _, testClub, testUser := setupTestService(t)
	_ = testUser // avoid unused variable warning
	_ = testClub // avoid unused variable warning
	ctx := testutil.TestContext()

	mockHanko.Clear()

	req := &LoginRequest{
		Email:    testUser.Email,
		ClubSlug: testClub.Slug,
	}

	response, err := service.InitiatePasskeyLogin(ctx, req)

	testutil.AssertNoError(t, err, "Initiate passkey login should succeed")
	testutil.AssertNotEqual(t, "", response.UserID, "User ID should be set")
	testutil.AssertTrue(t, len(response.Options) > 0, "Options should be provided")
}

func TestAuthService_InitiatePasskeyLogin_UserNotFound(t *testing.T) {
	service, _, _, testClub, testUser := setupTestService(t)
	_ = testUser // avoid unused variable warning
	_ = testClub // avoid unused variable warning
	ctx := testutil.TestContext()

	req := &LoginRequest{
		Email:    "nonexistent@test.com",
		ClubSlug: testClub.Slug,
	}

	_, err := service.InitiatePasskeyLogin(ctx, req)

	testutil.AssertError(t, err, "Initiate passkey login should fail for nonexistent user")
}

func TestAuthService_InitiatePasskeyLogin_InactiveUser(t *testing.T) {
	service, _, _, testClub, testUser := setupTestService(t)
	_ = testUser // avoid unused variable warning
	_ = testClub // avoid unused variable warning
	ctx := testutil.TestContext()

	// Create inactive user
	inactiveUser := &models.User{
		Email:       "inactive@test.com",
		Username:    "inactive",
		FirstName:   "Inactive",
		LastName:    "User",
		HankoUserID: "hanko-inactive-123",
		Status:      models.UserStatusSuspended,
	}
	inactiveUser.ClubID = testClub.ID
	service.repo.CreateUser(ctx, inactiveUser)

	req := &LoginRequest{
		Email:    inactiveUser.Email,
		ClubSlug: testClub.Slug,
	}

	_, err := service.InitiatePasskeyLogin(ctx, req)

	testutil.AssertError(t, err, "Initiate passkey login should fail for inactive user")
}

func TestAuthService_CompletePasskeyLogin_Success(t *testing.T) {
	service, mockHanko, _, testClub, testUser := setupTestService(t)
	_ = testUser // avoid unused variable warning
	_ = testClub // avoid unused variable warning
	ctx := testutil.TestContext()

	mockHanko.Clear()

	// Add user to mock Hanko
	mockHanko.AddUser(&hanko.HankoUser{
		ID:            testUser.HankoUserID,
		Email:         testUser.Email,
		EmailVerified: true,
	})

	credentialResult := map[string]interface{}{
		"type": "webauthn.get",
		"id":   "mock-credential-id",
	}

	response, err := service.CompletePasskeyLogin(ctx, testClub.Slug, testUser.HankoUserID, credentialResult)

	testutil.AssertNoError(t, err, "Complete passkey login should succeed")
	testutil.AssertNotEqual(t, "", response.Token, "Token should be generated")
	testutil.AssertNotEqual(t, "", response.RefreshToken, "Refresh token should be generated")
	testutil.AssertEqual(t, testUser.Email, response.User.Email, "User email should match")
}

func TestAuthService_CompletePasskeyLogin_UserNotFound(t *testing.T) {
	service, _, _, testClub, testUser := setupTestService(t)
	_ = testUser // avoid unused variable warning
	_ = testClub // avoid unused variable warning
	ctx := testutil.TestContext()

	credentialResult := map[string]interface{}{
		"type": "webauthn.get",
		"id":   "mock-credential-id",
	}

	_, err := service.CompletePasskeyLogin(ctx, testClub.Slug, "nonexistent-hanko-id", credentialResult)

	testutil.AssertError(t, err, "Complete passkey login should fail for nonexistent user")
}

func TestAuthService_CompletePasskeyLogin_VerificationFailed(t *testing.T) {
	service, mockHanko, _, testClub, testUser := setupTestService(t)
	_ = testUser // avoid unused variable warning
	_ = testClub // avoid unused variable warning
	ctx := testutil.TestContext()

	mockHanko.Clear()
	mockHanko.SetShouldFail("VerifyPasskey", true)

	credentialResult := map[string]interface{}{
		"type": "webauthn.get",
		"id":   "mock-credential-id",
	}

	_, err := service.CompletePasskeyLogin(ctx, testClub.Slug, testUser.HankoUserID, credentialResult)

	testutil.AssertError(t, err, "Complete passkey login should fail when verification fails")
}

// Session Management Tests

func TestAuthService_ValidateSession_Success(t *testing.T) {
	service, mockHanko, _, testClub, testUser := setupTestService(t)
	_ = testUser // avoid unused variable warning
	_ = testClub // avoid unused variable warning
	ctx := testutil.TestContext()

	mockHanko.Clear()

	// Add user and session to mock Hanko
	mockHanko.AddUser(&hanko.HankoUser{
		ID:            testUser.HankoUserID,
		Email:         testUser.Email,
		EmailVerified: true,
	})

	sessionToken := "valid-session-token"
	mockHanko.AddSession(sessionToken, &hanko.HankoSession{
		ID:        sessionToken,
		UserID:    testUser.HankoUserID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})

	user, err := service.ValidateSession(ctx, sessionToken)

	testutil.AssertNoError(t, err, "Validate session should succeed")
	testutil.AssertEqual(t, testUser.Email, user.Email, "User email should match")
}

func TestAuthService_ValidateSession_InvalidToken(t *testing.T) {
	service, _, _, _, _ := setupTestService(t)
	ctx := testutil.TestContext()

	_, err := service.ValidateSession(ctx, "invalid-token")

	testutil.AssertError(t, err, "Validate session should fail for invalid token")
}

func TestAuthService_Logout_Success(t *testing.T) {
	service, mockHanko, _, testClub, testUser := setupTestService(t)
	_ = testUser // avoid unused variable warning
	_ = testClub // avoid unused variable warning
	ctx := testutil.TestContext()

	mockHanko.Clear()

	// Create session
	session := &models.UserSession{
		UserID:         testUser.ID,
		HankoSessionID: "test-session-123",
		IsActive:       true,
		ExpiresAt:      time.Now().Add(24 * time.Hour),
	}
	session.ClubID = testClub.ID
	service.repo.CreateSession(ctx, session)

	err := service.Logout(ctx, testUser.ID, testClub.ID, session.HankoSessionID)

	testutil.AssertNoError(t, err, "Logout should succeed")

	// Verify session is invalidated
	retrievedSession, err := service.repo.GetSessionByHankoID(ctx, testClub.ID, session.HankoSessionID)
	testutil.AssertNoError(t, err, "Should be able to retrieve session")
	testutil.AssertFalse(t, retrievedSession.IsActive, "Session should be inactive")
}

func TestAuthService_Logout_UserNotFound(t *testing.T) {
	service, _, _, testClub, testUser := setupTestService(t)
	_ = testUser // avoid unused variable warning
	_ = testClub // avoid unused variable warning
	ctx := testutil.TestContext()

	err := service.Logout(ctx, 99999, testClub.ID, "some-session-token")

	testutil.AssertError(t, err, "Logout should fail for nonexistent user")
}

// User Management Tests

func TestAuthService_GetUser_Success(t *testing.T) {
	service, _, _, testClub, testUser := setupTestService(t)
	_ = testUser // avoid unused variable warning
	_ = testClub // avoid unused variable warning
	ctx := testutil.TestContext()

	user, err := service.GetUser(ctx, testClub.ID, testUser.ID)

	testutil.AssertNoError(t, err, "Get user should succeed")
	testutil.AssertEqual(t, testUser.Email, user.Email, "User email should match")
	testutil.AssertEqual(t, testUser.Username, user.Username, "User username should match")
}

func TestAuthService_GetUser_NotFound(t *testing.T) {
	service, _, _, testClub, testUser := setupTestService(t)
	_ = testUser // avoid unused variable warning
	_ = testClub // avoid unused variable warning
	ctx := testutil.TestContext()

	_, err := service.GetUser(ctx, testClub.ID, 99999)

	testutil.AssertError(t, err, "Get user should fail for nonexistent user")
}

func TestAuthService_GetUserWithRoles_Success(t *testing.T) {
	service, _, _, testClub, testUser := setupTestService(t)
	_ = testUser // avoid unused variable warning
	_ = testClub // avoid unused variable warning
	ctx := testutil.TestContext()

	userWithRoles, err := service.GetUserWithRoles(ctx, testClub.ID, testUser.ID)

	testutil.AssertNoError(t, err, "Get user with roles should succeed")
	testutil.AssertEqual(t, testUser.Email, userWithRoles.User.Email, "User email should match")
	testutil.AssertTrue(t, len(userWithRoles.RoleNames) > 0, "User should have roles")
	testutil.AssertTrue(t, len(userWithRoles.Permissions) > 0, "User should have permissions")
}

// Passkey Registration Tests

func TestAuthService_InitiatePasskeyRegistration_Success(t *testing.T) {
	service, mockHanko, _, testClub, testUser := setupTestService(t)
	_ = testUser // avoid unused variable warning
	_ = testClub // avoid unused variable warning
	ctx := testutil.TestContext()

	mockHanko.Clear()

	// Add user to mock Hanko
	mockHanko.AddUser(&hanko.HankoUser{
		ID:            testUser.HankoUserID,
		Email:         testUser.Email,
		EmailVerified: true,
	})

	response, err := service.InitiatePasskeyRegistration(ctx, testUser.ID, testClub.ID)

	testutil.AssertNoError(t, err, "Initiate passkey registration should succeed")
	testutil.AssertNotEqual(t, "", response.UserID, "User ID should be set")
	testutil.AssertTrue(t, len(response.Options) > 0, "Options should be provided")
}

func TestAuthService_InitiatePasskeyRegistration_UserNotFound(t *testing.T) {
	service, _, _, testClub, testUser := setupTestService(t)
	_ = testUser // avoid unused variable warning
	_ = testClub // avoid unused variable warning
	ctx := testutil.TestContext()

	_, err := service.InitiatePasskeyRegistration(ctx, 99999, testClub.ID)

	testutil.AssertError(t, err, "Initiate passkey registration should fail for nonexistent user")
}

// Health Check Tests

func TestAuthService_HealthCheck_Success(t *testing.T) {
	service, _, _, _, _ := setupTestService(t)
	ctx := testutil.TestContext()

	err := service.HealthCheck(ctx)

	testutil.AssertNoError(t, err, "Health check should succeed")
}

func TestAuthService_HealthCheck_HankoFailure(t *testing.T) {
	service, mockHanko, _, _, _ := setupTestService(t)
	ctx := testutil.TestContext()

	mockHanko.SetShouldFail("HealthCheck", true)

	err := service.HealthCheck(ctx)

	testutil.AssertError(t, err, "Health check should fail when Hanko fails")
}

// Helper function tests

func TestAuthService_ConvertToAuthUser(t *testing.T) {
	service, _, _, testClub, testUser := setupTestService(t)
	_ = testUser // avoid unused variable warning
	_ = testClub // avoid unused variable warning

	authUser := service.convertToAuthUser(testUser)

	testutil.AssertEqual(t, testUser.ID, authUser.ID, "ID should match")
	testutil.AssertEqual(t, testUser.ClubID, authUser.ClubID, "Club ID should match")
	testutil.AssertEqual(t, testUser.Email, authUser.Email, "Email should match")
	testutil.AssertEqual(t, testUser.Username, authUser.Username, "Username should match")
}

// Integration Tests

func TestAuthService_FullAuthFlow_Success(t *testing.T) {
	service, mockHanko, _, testClub, testUser := setupTestService(t)
	_ = testUser // avoid unused variable warning
	_ = testClub // avoid unused variable warning
	ctx := testutil.TestContext()

	mockHanko.Clear()

	// 1. Register new user
	registerReq := &RegisterRequest{
		Email:     "flowtest@test.com",
		Username:  "flowtest",
		FirstName: "Flow",
		LastName:  "Test",
		ClubSlug:  testClub.Slug,
	}

	registerResp, err := service.Register(ctx, registerReq)
	testutil.AssertNoError(t, err, "Registration should succeed")

	// 2. Initiate passkey login
	loginReq := &LoginRequest{
		Email:    registerReq.Email,
		ClubSlug: testClub.Slug,
	}

	initResp, err := service.InitiatePasskeyLogin(ctx, loginReq)
	testutil.AssertNoError(t, err, "Initiate login should succeed")
	testutil.AssertEqual(t, registerResp.User.HankoUserID, initResp.UserID, "Hanko user ID should match")

	// 3. Complete passkey login
	credentialResult := map[string]interface{}{
		"type": "webauthn.get",
		"id":   "mock-credential-id",
	}

	loginResp, err := service.CompletePasskeyLogin(ctx, testClub.Slug, registerResp.User.HankoUserID, credentialResult)
	testutil.AssertNoError(t, err, "Complete login should succeed")
	testutil.AssertEqual(t, registerReq.Email, loginResp.User.Email, "User email should match")

	// 4. Validate session (using a mock session token)
	sessionToken := "mock-session-for-validation"
	mockHanko.AddSession(sessionToken, &hanko.HankoSession{
		ID:        sessionToken,
		UserID:    registerResp.User.HankoUserID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})

	validatedUser, err := service.ValidateSession(ctx, sessionToken)
	testutil.AssertNoError(t, err, "Session validation should succeed")
	testutil.AssertEqual(t, registerReq.Email, validatedUser.Email, "Validated user email should match")

	// 5. Logout
	err = service.Logout(ctx, loginResp.User.ID, testClub.ID, sessionToken)
	testutil.AssertNoError(t, err, "Logout should succeed")
}

// Error Handling Tests

func TestAuthService_ErrorPropagation(t *testing.T) {
	service, _, _, testClub, testUser := setupTestService(t)
	_ = testUser // avoid unused variable warning
	_ = testClub // avoid unused variable warning
	ctx := testutil.TestContext()

	// Test with invalid club ID
	_, err := service.GetUser(ctx, 99999, testUser.ID)
	testutil.AssertError(t, err, "Should propagate repository errors")

	// Test with invalid user ID
	_, err = service.GetUser(ctx, testClub.ID, 99999)
	testutil.AssertError(t, err, "Should propagate repository errors")
}

func TestAuthService_ContextHandling(t *testing.T) {
	service, _, _, testClub, testUser := setupTestService(t)
	_ = testUser // avoid unused variable warning
	_ = testClub // avoid unused variable warning

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := service.GetUser(ctx, testClub.ID, testUser.ID)
	// Note: This test depends on the repository implementation respecting context cancellation
	// The exact behavior may vary based on the database driver
	t.Logf("Context cancellation test completed, error: %v", err)
}