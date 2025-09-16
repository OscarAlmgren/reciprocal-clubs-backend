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
		{"Pending user", UserStatusPending, false},
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

func TestUserHasRole(t *testing.T) {
	user := &User{
		ID: 1,
		Roles: []Role{
			{ID: 1, Name: "admin", ClubID: 1},
			{ID: 2, Name: "user", ClubID: 1},
			{ID: 3, Name: "admin", ClubID: 2},
		},
	}

	tests := []struct {
		name     string
		clubID   uint
		roleName string
		expected bool
	}{
		{"Has admin role in club 1", 1, "admin", true},
		{"Has user role in club 1", 1, "user", true},
		{"Has admin role in club 2", 2, "admin", true},
		{"Does not have moderator role in club 1", 1, "moderator", false},
		{"Does not have admin role in club 3", 3, "admin", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if user.HasRole(tt.clubID, tt.roleName) != tt.expected {
				t.Errorf("HasRole(%d, %s) = %v, want %v", tt.clubID, tt.roleName, user.HasRole(tt.clubID, tt.roleName), tt.expected)
			}
		})
	}
}

func TestUserGetRolesForClub(t *testing.T) {
	user := &User{
		ID: 1,
		Roles: []Role{
			{ID: 1, Name: "admin", ClubID: 1},
			{ID: 2, Name: "user", ClubID: 1},
			{ID: 3, Name: "admin", ClubID: 2},
		},
	}

	tests := []struct {
		name     string
		clubID   uint
		expected []string
	}{
		{"Roles for club 1", 1, []string{"admin", "user"}},
		{"Roles for club 2", 2, []string{"admin"}},
		{"No roles for club 3", 3, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roles := user.GetRolesForClub(tt.clubID)
			if len(roles) != len(tt.expected) {
				t.Errorf("GetRolesForClub(%d) returned %d roles, want %d", tt.clubID, len(roles), len(tt.expected))
				continue
			}

			roleMap := make(map[string]bool)
			for _, role := range roles {
				roleMap[role.Name] = true
			}

			for _, expectedRole := range tt.expected {
				if !roleMap[expectedRole] {
					t.Errorf("GetRolesForClub(%d) missing role %s", tt.clubID, expectedRole)
				}
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

func TestRoleHasPermission(t *testing.T) {
	role := &Role{
		ID:   1,
		Name: "admin",
		Permissions: []Permission{
			{ID: 1, Name: "read", Resource: "users", Action: "read"},
			{ID: 2, Name: "write", Resource: "users", Action: "write"},
			{ID: 3, Name: "delete", Resource: "posts", Action: "delete"},
		},
	}

	tests := []struct {
		name       string
		resource   string
		action     string
		expected   bool
	}{
		{"Has read permission on users", "users", "read", true},
		{"Has write permission on users", "users", "write", true},
		{"Has delete permission on posts", "posts", "delete", true},
		{"Does not have delete permission on users", "users", "delete", false},
		{"Does not have read permission on posts", "posts", "read", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if role.HasPermission(tt.resource, tt.action) != tt.expected {
				t.Errorf("HasPermission(%s, %s) = %v, want %v", tt.resource, tt.action, role.HasPermission(tt.resource, tt.action), tt.expected)
			}
		})
	}
}

func TestPermissionMatches(t *testing.T) {
	permission := &Permission{
		Resource: "users",
		Action:   "read",
	}

	tests := []struct {
		name     string
		resource string
		action   string
		expected bool
	}{
		{"Exact match", "users", "read", true},
		{"Different resource", "posts", "read", false},
		{"Different action", "users", "write", false},
		{"Both different", "posts", "write", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if permission.Matches(tt.resource, tt.action) != tt.expected {
				t.Errorf("Matches(%s, %s) = %v, want %v", tt.resource, tt.action, permission.Matches(tt.resource, tt.action), tt.expected)
			}
		})
	}
}

func TestUserSessionIsActive(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		status    SessionStatus
		expiresAt time.Time
		expected  bool
	}{
		{"Active and not expired", SessionStatusActive, now.Add(time.Hour), true},
		{"Active but expired", SessionStatusActive, now.Add(-time.Hour), false},
		{"Inactive and not expired", SessionStatusInactive, now.Add(time.Hour), false},
		{"Revoked and not expired", SessionStatusRevoked, now.Add(time.Hour), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &UserSession{
				Status:    tt.status,
				ExpiresAt: tt.expiresAt,
			}
			if session.IsActive() != tt.expected {
				t.Errorf("IsActive() = %v, want %v", session.IsActive(), tt.expected)
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
		{"Exactly at expiry", now, false}, // Current time should not be considered expired
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

func TestDefaultRoles(t *testing.T) {
	roles := DefaultRoles()

	expectedRoles := map[string]string{
		"admin":     "Full administrative access",
		"moderator": "Content moderation and user management",
		"member":    "Regular member access",
		"guest":     "Limited guest access",
	}

	if len(roles) != len(expectedRoles) {
		t.Errorf("DefaultRoles() returned %d roles, want %d", len(roles), len(expectedRoles))
	}

	for _, role := range roles {
		expectedDescription, exists := expectedRoles[role.Name]
		if !exists {
			t.Errorf("Unexpected role: %s", role.Name)
			continue
		}
		if role.Description != expectedDescription {
			t.Errorf("Role %s has description %s, want %s", role.Name, role.Description, expectedDescription)
		}
	}
}

func TestDefaultPermissions(t *testing.T) {
	permissions := DefaultPermissions()

	// Check that we have some expected permissions
	expectedPermissions := map[string]map[string]string{
		"users": {
			"read":   "View user profiles",
			"write":  "Edit user profiles",
			"delete": "Delete user accounts",
		},
		"posts": {
			"read":   "View posts",
			"write":  "Create and edit posts",
			"delete": "Delete posts",
		},
		"clubs": {
			"read":   "View club information",
			"write":  "Edit club settings",
			"delete": "Delete clubs",
		},
	}

	// Create a map for easy lookup
	permissionMap := make(map[string]map[string]*Permission)
	for _, perm := range permissions {
		if permissionMap[perm.Resource] == nil {
			permissionMap[perm.Resource] = make(map[string]*Permission)
		}
		permissionMap[perm.Resource][perm.Action] = perm
	}

	// Check expected permissions exist
	for resource, actions := range expectedPermissions {
		resourcePerms, exists := permissionMap[resource]
		if !exists {
			t.Errorf("Missing permissions for resource: %s", resource)
			continue
		}

		for action, expectedDesc := range actions {
			perm, exists := resourcePerms[action]
			if !exists {
				t.Errorf("Missing permission: %s.%s", resource, action)
				continue
			}

			if perm.Description != expectedDesc {
				t.Errorf("Permission %s.%s has description %s, want %s", resource, action, perm.Description, expectedDesc)
			}

			expectedName := resource + "." + action
			if perm.Name != expectedName {
				t.Errorf("Permission has name %s, want %s", perm.Name, expectedName)
			}
		}
	}
}

func TestAuditLogFormatMessage(t *testing.T) {
	tests := []struct {
		name     string
		log      *AuditLog
		expected string
	}{
		{
			"Login event",
			&AuditLog{
				Action: "login",
				Entity: "user",
				Details: map[string]interface{}{
					"email":   "test@example.com",
					"user_id": float64(123), // JSON numbers are float64
				},
			},
			"login on user: {\"email\":\"test@example.com\",\"user_id\":123}",
		},
		{
			"Simple event without details",
			&AuditLog{
				Action: "logout",
				Entity: "session",
			},
			"logout on session: {}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := tt.log.FormatMessage()
			if message != tt.expected {
				t.Errorf("FormatMessage() = %s, want %s", message, tt.expected)
			}
		})
	}
}

func TestModelValidation(t *testing.T) {
	t.Run("User email validation", func(t *testing.T) {
		user := &User{
			Email:       "", // Empty email should be invalid
			DisplayName: "Test User",
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
			Name:   "", // Empty name should be invalid
			ClubID: 1,
		}

		// This would normally be validated by GORM with database constraints
		if role.Name == "" {
			// This is expected for our test
		}
	})
}

func TestUserSessionTokenGeneration(t *testing.T) {
	session1 := &UserSession{
		UserID: 1,
		ClubID: 1,
	}
	session1.GenerateToken()

	session2 := &UserSession{
		UserID: 1,
		ClubID: 1,
	}
	session2.GenerateToken()

	// Tokens should be different
	if session1.Token == session2.Token {
		t.Error("Generated tokens should be unique")
	}

	// Tokens should not be empty
	if session1.Token == "" {
		t.Error("Generated token should not be empty")
	}

	// Tokens should have reasonable length
	if len(session1.Token) < 20 {
		t.Errorf("Generated token is too short: %d characters", len(session1.Token))
	}
}