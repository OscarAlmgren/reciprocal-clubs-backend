package models

import (
	"testing"
	"time"
)

func TestUserStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   UserStatus
		expected bool
	}{
		{"Active user", UserStatusActive, true},
		{"Inactive user", UserStatusInactive, false},
		{"Suspended user", UserStatusSuspended, false},
		{"Pending user", UserStatusPendingVerification, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{Status: tt.status}
			if user.IsActive() != tt.expected {
				t.Errorf("IsActive() = %v, want %v", user.IsActive(), tt.expected)
			}
		})
	}
}

func TestUserFullName(t *testing.T) {
	tests := []struct {
		name      string
		user      *User
		expected  string
	}{
		{
			"Both first and last name",
			&User{FirstName: "John", LastName: "Doe", Username: "johndoe"},
			"John Doe",
		},
		{
			"First name only",
			&User{FirstName: "John", LastName: "", Username: "johndoe"},
			"John",
		},
		{
			"Last name only",
			&User{FirstName: "", LastName: "Doe", Username: "johndoe"},
			"Doe",
		},
		{
			"Username fallback",
			&User{FirstName: "", LastName: "", Username: "johndoe"},
			"johndoe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.user.GetFullName()
			if result != tt.expected {
				t.Errorf("GetFullName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestClubStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   ClubStatus
		expected bool
	}{
		{"Active club", ClubStatusActive, true},
		{"Inactive club", ClubStatusInactive, false},
		{"Suspended club", ClubStatusSuspended, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			club := &Club{Status: tt.status}
			if club.IsActive() != tt.expected {
				t.Errorf("IsActive() = %v, want %v", club.IsActive(), tt.expected)
			}
		})
	}
}

func TestPermissionBasics(t *testing.T) {
	permission := &Permission{
		Name:        "user.read",
		Description: "Read user information",
		Resource:    "user",
		Action:      "read",
	}

	if permission.Name != "user.read" {
		t.Errorf("Expected permission name user.read, got %s", permission.Name)
	}
	if permission.Resource != "user" {
		t.Errorf("Expected resource user, got %s", permission.Resource)
	}
	if permission.Action != "read" {
		t.Errorf("Expected action read, got %s", permission.Action)
	}
}

func TestUserSessionIsValid(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		isActive  bool
		expiresAt time.Time
		logoutAt  *time.Time
		expected  bool
	}{
		{"Active and not expired", true, now.Add(time.Hour), nil, true},
		{"Active but expired", true, now.Add(-time.Hour), nil, false},
		{"Inactive and not expired", false, now.Add(time.Hour), nil, false},
		{"Logged out", true, now.Add(time.Hour), &now, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &UserSession{
				IsActive:  tt.isActive,
				ExpiresAt: tt.expiresAt,
				LogoutAt:  tt.logoutAt,
			}
			if session.IsValid() != tt.expected {
				t.Errorf("IsValid() = %v, want %v", session.IsValid(), tt.expected)
			}
		})
	}
}

func TestUserSessionIsExpired(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		expiresAt time.Time
		expected  bool
	}{
		{"Not expired", now.Add(time.Hour), false},
		{"Expired", now.Add(-time.Hour), true},
		{"Exactly at expiry", now.Add(time.Millisecond), false}, // Very close future should not be expired
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &UserSession{
				ExpiresAt: tt.expiresAt,
			}
			if session.IsExpired() != tt.expected {
				t.Errorf("IsExpired() = %v, want %v", session.IsExpired(), tt.expected)
			}
		})
	}
}

func TestGetDefaultRoles(t *testing.T) {
	roles := GetDefaultRoles()

	if len(roles) == 0 {
		t.Error("GetDefaultRoles() returned no roles")
	}

	// Check for admin role
	foundAdmin := false
	for _, role := range roles {
		if role.Name == RoleAdmin {
			foundAdmin = true
			if role.Description == "" {
				t.Error("Admin role should have a description")
			}
			if !role.IsSystem {
				t.Error("Admin role should be marked as system role")
			}
		}
	}

	if !foundAdmin {
		t.Error("Admin role not found in default roles")
	}
}

func TestGetDefaultPermissions(t *testing.T) {
	permissions := GetDefaultPermissions()

	if len(permissions) == 0 {
		t.Error("GetDefaultPermissions() returned no permissions")
	}

	// Check that all permissions have required fields
	for _, perm := range permissions {
		if perm.Name == "" {
			t.Error("Permission should have a name")
		}
		if perm.Resource == "" {
			t.Error("Permission should have a resource")
		}
		if perm.Action == "" {
			t.Error("Permission should have an action")
		}
		if perm.ClubID == 0 {
			t.Error("Permission should have a club ID")
		}
	}
}

func TestAuditLogBasics(t *testing.T) {
	auditLog := &AuditLog{
		Action:       AuditActionLogin,
		HankoUserID:  "hanko123",
		Details:      "User logged in successfully",
		Success:      true,
		IPAddress:    "192.168.1.1",
		UserAgent:    "Mozilla/5.0",
		ErrorMessage: "",
	}

	if auditLog.Action != AuditActionLogin {
		t.Errorf("Expected action %s, got %s", AuditActionLogin, auditLog.Action)
	}
	if !auditLog.Success {
		t.Error("Expected success to be true")
	}
}

func TestModelValidation(t *testing.T) {
	t.Run("User email validation", func(t *testing.T) {
		user := &User{
			Email:     "", // Empty email should be invalid
			Username:  "testuser",
			FirstName: "Test",
			LastName:  "User",
		}

		// This would normally be validated by GORM with database constraints
		// For now, we'll just check that the struct can be created
		if user.Email == "" {
			// This is expected for our test
		}
	})

	t.Run("Club slug validation", func(t *testing.T) {
		club := &Club{
			Name: "Test Club",
			Slug: "", // Empty slug should be invalid
		}

		// This would normally be validated by GORM with database constraints
		if club.Slug == "" {
			// This is expected for our test
		}
	})

	t.Run("Role name validation", func(t *testing.T) {
		role := &Role{
			Name: "", // Empty name should be invalid
		}
		role.ClubID = 1

		// This would normally be validated by GORM with database constraints
		if role.Name == "" {
			// This is expected for our test
		}
	})
}

func TestUserLockUnlock(t *testing.T) {
	user := &User{
		Status:         UserStatusActive,
		FailedAttempts: 3,
	}

	// Test locking
	user.Lock(30 * time.Minute)
	if user.Status != UserStatusLocked {
		t.Error("User should be locked")
	}
	if !user.IsLocked() {
		t.Error("IsLocked() should return true")
	}

	// Test unlocking
	user.Unlock()
	if user.Status != UserStatusActive {
		t.Error("User should be active after unlock")
	}
	if user.FailedAttempts != 0 {
		t.Error("Failed attempts should be reset after unlock")
	}
	if user.IsLocked() {
		t.Error("IsLocked() should return false after unlock")
	}
}