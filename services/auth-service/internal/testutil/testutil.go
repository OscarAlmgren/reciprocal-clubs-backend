package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"testing"
	"time"

	"reciprocal-clubs-backend/services/auth-service/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestDB holds test database connection and utilities
type TestDB struct {
	DB     *gorm.DB
	rawDB  *sql.DB
	dbName string
}

// NewTestDB creates a new test database connection
func NewTestDB(t *testing.T) *TestDB {
	dbName := fmt.Sprintf("test_auth_service_%d", time.Now().UnixNano())
	
	// Connect to postgres to create test database
	masterDSN := "host=localhost port=5432 user=postgres password=postgres sslmode=disable"
	masterDB, err := sql.Open("postgres", masterDSN)
	if err != nil {
		t.Fatalf("Failed to connect to master database: %v", err)
	}
	defer masterDB.Close()

	// Create test database
	_, err = masterDB.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Connect to test database
	testDSN := fmt.Sprintf("host=localhost port=5432 user=postgres password=postgres dbname=%s sslmode=disable", dbName)
	rawDB, err := sql.Open("postgres", testDSN)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Create GORM connection
	db, err := gorm.Open(postgres.New(postgres.Config{
		Conn: rawDB,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // Silent for tests
	})
	if err != nil {
		t.Fatalf("Failed to create GORM connection: %v", err)
	}

	// Run migrations
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
		t.Fatalf("Failed to run migrations: %v", err)
	}

	testDB := &TestDB{
		DB:     db,
		rawDB:  rawDB,
		dbName: dbName,
	}

	// Setup cleanup
	t.Cleanup(func() {
		testDB.Cleanup(t)
	})

	return testDB
}

// Cleanup drops the test database
func (tdb *TestDB) Cleanup(t *testing.T) {
	// Close GORM connection
	if tdb.rawDB != nil {
		tdb.rawDB.Close()
	}

	// Connect to master to drop test database
	masterDSN := "host=localhost port=5432 user=postgres password=postgres sslmode=disable"
	masterDB, err := sql.Open("postgres", masterDSN)
	if err != nil {
		log.Printf("Failed to connect to master database for cleanup: %v", err)
		return
	}
	defer masterDB.Close()

	// Drop test database
	_, err = masterDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", tdb.dbName))
	if err != nil {
		log.Printf("Failed to drop test database: %v", err)
	}
}

// SeedTestData creates test data for use in tests
func (tdb *TestDB) SeedTestData(t *testing.T) *TestData {
	// Create test club
	club := &models.Club{
		Name:        "Test Club",
		Slug:        "test-club",
		Description: "A club for testing",
		Status:      models.ClubStatusActive,
	}
	if err := tdb.DB.Create(club).Error; err != nil {
		t.Fatalf("Failed to create test club: %v", err)
	}

	// Create test permissions
	readPermission := &models.Permission{
		Name:        "read",
		Description: "Read access",
		Resource:    "users",
		Action:      "read",
	}
	writePermission := &models.Permission{
		Name:        "write",
		Description: "Write access",
		Resource:    "users",
		Action:      "write",
	}
	
	if err := tdb.DB.Create([]*models.Permission{readPermission, writePermission}).Error; err != nil {
		t.Fatalf("Failed to create test permissions: %v", err)
	}

	// Create test roles
	adminRole := &models.Role{
		Name:        "admin",
		Description: "Administrator role",
		ClubID:      club.ID,
	}
	userRole := &models.Role{
		Name:        "user",
		Description: "Regular user role",
		ClubID:      club.ID,
	}

	if err := tdb.DB.Create([]*models.Role{adminRole, userRole}).Error; err != nil {
		t.Fatalf("Failed to create test roles: %v", err)
	}

	// Assign permissions to roles
	adminRolePermissions := []*models.RolePermission{
		{RoleID: adminRole.ID, PermissionID: readPermission.ID},
		{RoleID: adminRole.ID, PermissionID: writePermission.ID},
	}
	userRolePermissions := []*models.RolePermission{
		{RoleID: userRole.ID, PermissionID: readPermission.ID},
	}

	if err := tdb.DB.Create(adminRolePermissions).Error; err != nil {
		t.Fatalf("Failed to create admin role permissions: %v", err)
	}
	if err := tdb.DB.Create(userRolePermissions).Error; err != nil {
		t.Fatalf("Failed to create user role permissions: %v", err)
	}

	// Create test users
	adminUser := &models.User{
		Email:       "admin@test.com",
		DisplayName: "Test Admin",
		HankoUserID: "hanko-admin-123",
		Status:      models.UserStatusActive,
	}
	regularUser := &models.User{
		Email:       "user@test.com",
		DisplayName: "Test User",
		HankoUserID: "hanko-user-456",
		Status:      models.UserStatusActive,
	}

	if err := tdb.DB.Create([]*models.User{adminUser, regularUser}).Error; err != nil {
		t.Fatalf("Failed to create test users: %v", err)
	}

	// Assign roles to users
	userRoles := []*models.UserRole{
		{UserID: adminUser.ID, RoleID: adminRole.ID, ClubID: club.ID},
		{UserID: regularUser.ID, RoleID: userRole.ID, ClubID: club.ID},
	}

	if err := tdb.DB.Create(userRoles).Error; err != nil {
		t.Fatalf("Failed to create user roles: %v", err)
	}

	return &TestData{
		Club:           club,
		AdminUser:      adminUser,
		RegularUser:    regularUser,
		AdminRole:      adminRole,
		UserRole:       userRole,
		ReadPermission: readPermission,
		WritePermission: writePermission,
	}
}

// TestData holds common test data
type TestData struct {
	Club            *models.Club
	AdminUser       *models.User
	RegularUser     *models.User
	AdminRole       *models.Role
	UserRole        *models.Role
	ReadPermission  *models.Permission
	WritePermission *models.Permission
}

// CreateTestUser creates a test user with the given parameters
func CreateTestUser(db *gorm.DB, email, displayName, hankoUserID string) *models.User {
	user := &models.User{
		Email:       email,
		DisplayName: displayName,
		HankoUserID: hankoUserID,
		Status:      models.UserStatusActive,
	}
	
	if err := db.Create(user).Error; err != nil {
		panic(fmt.Sprintf("Failed to create test user: %v", err))
	}
	
	return user
}

// CreateTestClub creates a test club with the given parameters
func CreateTestClub(db *gorm.DB, name, slug string) *models.Club {
	club := &models.Club{
		Name:        name,
		Slug:        slug,
		Description: fmt.Sprintf("Test club %s", name),
		Status:      models.ClubStatusActive,
	}
	
	if err := db.Create(club).Error; err != nil {
		panic(fmt.Sprintf("Failed to create test club: %v", err))
	}
	
	return club
}

// CreateTestSession creates a test session for the given user
func CreateTestSession(db *gorm.DB, userID, clubID uint) *models.UserSession {
	session := &models.UserSession{
		UserID:    userID,
		ClubID:    clubID,
		Token:     fmt.Sprintf("test-token-%d-%d", userID, time.Now().UnixNano()),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Status:    models.SessionStatusActive,
	}
	
	if err := db.Create(session).Error; err != nil {
		panic(fmt.Sprintf("Failed to create test session: %v", err))
	}
	
	return session
}

// AssertNoError fails the test if err is not nil
func AssertNoError(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", msg, err)
	}
}

// AssertError fails the test if err is nil
func AssertError(t *testing.T, err error, msg string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: expected error but got nil", msg)
	}
}

// AssertEqual fails the test if expected != actual
func AssertEqual(t *testing.T, expected, actual interface{}, msg string) {
	t.Helper()
	if expected != actual {
		t.Fatalf("%s: expected %v, got %v", msg, expected, actual)
	}
}

// AssertNotEqual fails the test if expected == actual
func AssertNotEqual(t *testing.T, expected, actual interface{}, msg string) {
	t.Helper()
	if expected == actual {
		t.Fatalf("%s: expected %v to not equal %v", msg, expected, actual)
	}
}

// AssertTrue fails the test if condition is false
func AssertTrue(t *testing.T, condition bool, msg string) {
	t.Helper()
	if !condition {
		t.Fatalf("%s: expected true but got false", msg)
	}
}

// AssertFalse fails the test if condition is true
func AssertFalse(t *testing.T, condition bool, msg string) {
	t.Helper()
	if condition {
		t.Fatalf("%s: expected false but got true", msg)
	}
}

// AssertContains fails the test if haystack doesn't contain needle
func AssertContains(t *testing.T, haystack, needle string, msg string) {
	t.Helper()
	if !contains(haystack, needle) {
		t.Fatalf("%s: expected '%s' to contain '%s'", msg, haystack, needle)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && 
		(haystack == needle || 
		 (len(haystack) > len(needle) && 
		  (haystack[:len(needle)] == needle || 
		   haystack[len(haystack)-len(needle):] == needle ||
		   findInString(haystack, needle))))
}

func findInString(haystack, needle string) bool {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

// TestContext returns a context with timeout for tests
func TestContext() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	return ctx
}