package models

import (
	"strings"
	"time"

	"reciprocal-clubs-backend/pkg/shared/database"
)

// User represents a user in the system
type User struct {
	database.BaseModel
	HankoUserID    string     `json:"hanko_user_id" gorm:"uniqueIndex;not null"`
	Email          string     `json:"email" gorm:"uniqueIndex;not null"`
	Username       string     `json:"username" gorm:"uniqueIndex;not null"`
	FirstName      string     `json:"first_name"`
	LastName       string     `json:"last_name"`
	Status         UserStatus `json:"status" gorm:"default:'active'"`
	EmailVerified  bool       `json:"email_verified" gorm:"default:false"`
	LastLoginAt    *time.Time `json:"last_login_at"`
	FailedAttempts int        `json:"failed_attempts" gorm:"default:0"`
	LockedUntil    *time.Time `json:"locked_until"`

	// MFA Settings
	MFAEnabled     bool   `json:"mfa_enabled" gorm:"default:false"`
	MFASecret      string `json:"-" gorm:"column:mfa_secret"` // Hidden from JSON
	MFABackupCodes string `json:"-" gorm:"column:mfa_backup_codes;type:text"` // Hidden from JSON
	MFAMethod      string `json:"mfa_method" gorm:"default:'totp'"`
	PhoneNumber    string `json:"phone_number"`
	PhoneVerified  bool   `json:"phone_verified" gorm:"default:false"`

	// Password Reset
	PasswordResetToken     string     `json:"-" gorm:"column:password_reset_token"` // Hidden from JSON
	PasswordResetExpiresAt *time.Time `json:"-" gorm:"column:password_reset_expires_at"` // Hidden from JSON

	// Email Verification
	EmailVerificationToken     string     `json:"-" gorm:"column:email_verification_token"` // Hidden from JSON
	EmailVerificationExpiresAt *time.Time `json:"-" gorm:"column:email_verification_expires_at"` // Hidden from JSON

	// Relationships
	Roles    []UserRole    `json:"roles" gorm:"foreignKey:UserID"`
	Sessions []UserSession `json:"sessions" gorm:"foreignKey:UserID"`
	MFATokens []MFAToken    `json:"mfa_tokens" gorm:"foreignKey:UserID"`
}

type UserStatus string

const (
	UserStatusActive              UserStatus = "active"
	UserStatusInactive            UserStatus = "inactive"
	UserStatusSuspended           UserStatus = "suspended"
	UserStatusPendingVerification UserStatus = "pending_verification"
	UserStatusLocked              UserStatus = "locked"
)

// Club represents a reciprocal club
type Club struct {
	database.BaseModel
	Name        string     `json:"name" gorm:"not null"`
	Slug        string     `json:"slug" gorm:"uniqueIndex;not null"`
	Description string     `json:"description"`
	Location    string     `json:"location"`
	Website     string     `json:"website"`
	LogoURL     string     `json:"logo_url"`
	Status      ClubStatus `json:"status" gorm:"default:'active'"`
	Settings    ClubSettings `json:"settings" gorm:"embedded"`

	// Contact Information
	ContactEmail string `json:"contact_email"`
	ContactPhone string `json:"contact_phone"`
	Address      string `json:"address"`

	// Hanko Configuration - each club can have its own Hanko tenant
	HankoTenantID     string `json:"hanko_tenant_id"`
	HankoAPIKey       string `json:"hanko_api_key" gorm:"type:text"`
	HankoClientSecret string `json:"hanko_client_secret" gorm:"type:text"`

	// Relationships
	Users []User `json:"users" gorm:"foreignKey:ClubID"`
}

type ClubStatus string

const (
	ClubStatusActive    ClubStatus = "active"
	ClubStatusInactive  ClubStatus = "inactive"
	ClubStatusSuspended ClubStatus = "suspended"
)

type ClubSettings struct {
	AllowReciprocal         bool    `json:"allow_reciprocal" gorm:"default:true"`
	RequireApproval         bool    `json:"require_approval" gorm:"default:true"`
	MaxVisitsPerMonth       int     `json:"max_visits_per_month" gorm:"default:10"`
	ReciprocalFee           float64 `json:"reciprocal_fee" gorm:"default:0"`
	EnablePasskeyAuth       bool    `json:"enable_passkey_auth" gorm:"default:true"`
	RequirePasskeyAuth      bool    `json:"require_passkey_auth" gorm:"default:false"`
	SessionTimeoutMinutes   int     `json:"session_timeout_minutes" gorm:"default:480"` // 8 hours
	MaxFailedAttempts       int     `json:"max_failed_attempts" gorm:"default:5"`
	LockoutDurationMinutes  int     `json:"lockout_duration_minutes" gorm:"default:30"`
	EnableAuditLogging      bool    `json:"enable_audit_logging" gorm:"default:true"`
}

// Role represents a user role within a club
type Role struct {
	database.BaseModel
	Name        string `json:"name" gorm:"not null"`
	Description string `json:"description"`
	IsSystem    bool   `json:"is_system" gorm:"default:false"`

	// Relationships
	UserRoles       []UserRole       `json:"user_roles" gorm:"foreignKey:RoleID"`
	RolePermissions []RolePermission `json:"role_permissions" gorm:"foreignKey:RoleID"`
}

// System roles
const (
	RoleAdmin           = "admin"
	RoleMember          = "member"
	RoleStaff           = "staff"
	RoleManager         = "manager"
	RoleReciprocalAdmin = "reciprocal_admin"
	RoleGuest           = "guest"
)

// Permission represents a system permission
type Permission struct {
	database.BaseModel
	Name        string `json:"name" gorm:"uniqueIndex;not null"`
	Description string `json:"description"`
	Resource    string `json:"resource" gorm:"not null"`
	Action      string `json:"action" gorm:"not null"`

	// Relationships
	RolePermissions []RolePermission `json:"role_permissions" gorm:"foreignKey:PermissionID"`
}

// UserRole represents the many-to-many relationship between users and roles
type UserRole struct {
	database.BaseModel
	UserID      uint       `json:"user_id" gorm:"not null"`
	RoleID      uint       `json:"role_id" gorm:"not null"`
	GrantedBy   uint       `json:"granted_by"`
	GrantedAt   time.Time  `json:"granted_at" gorm:"default:CURRENT_TIMESTAMP"`
	ExpiresAt   *time.Time `json:"expires_at"`
	IsActive    bool       `json:"is_active" gorm:"default:true"`

	// Relationships
	User      User  `json:"user" gorm:"foreignKey:UserID"`
	Role      Role  `json:"role" gorm:"foreignKey:RoleID"`
	GrantedByUser *User `json:"granted_by_user" gorm:"foreignKey:GrantedBy"`
}

// RolePermission represents the many-to-many relationship between roles and permissions
type RolePermission struct {
	database.BaseModel
	RoleID       uint `json:"role_id" gorm:"not null"`
	PermissionID uint `json:"permission_id" gorm:"not null"`

	// Relationships
	Role       Role       `json:"role" gorm:"foreignKey:RoleID"`
	Permission Permission `json:"permission" gorm:"foreignKey:PermissionID"`
}

// UserSession represents user authentication sessions
type UserSession struct {
	database.BaseModel
	UserID          uint      `json:"user_id" gorm:"not null"`
	HankoSessionID  string    `json:"hanko_session_id" gorm:"uniqueIndex;not null"`
	JWTToken        string    `json:"jwt_token" gorm:"type:text"`
	RefreshToken    string    `json:"refresh_token" gorm:"type:text"`
	IPAddress       string    `json:"ip_address"`
	UserAgent       string    `json:"user_agent" gorm:"type:text"`
	ExpiresAt       time.Time `json:"expires_at"`
	LastActivityAt  time.Time `json:"last_activity_at" gorm:"default:CURRENT_TIMESTAMP"`
	IsActive        bool      `json:"is_active" gorm:"default:true"`
	LogoutAt        *time.Time `json:"logout_at"`
	
	// Relationships
	User User `json:"user" gorm:"foreignKey:UserID"`
}

// MFAToken represents MFA verification tokens and backup codes
type MFAToken struct {
	database.BaseModel
	UserID      uint      `json:"user_id" gorm:"not null"`
	TokenType   MFATokenType `json:"token_type" gorm:"not null"`
	Token       string    `json:"-" gorm:"column:token;not null"` // Hidden from JSON
	Used        bool      `json:"used" gorm:"default:false"`
	UsedAt      *time.Time `json:"used_at"`
	ExpiresAt   *time.Time `json:"expires_at"`
	Metadata    map[string]interface{} `json:"metadata" gorm:"serializer:json"`

	// Relationships
	User User `json:"user" gorm:"foreignKey:UserID"`
}

type MFATokenType string

const (
	MFATokenTypeBackup       MFATokenType = "backup"
	MFATokenTypeVerification MFATokenType = "verification"
	MFATokenTypeSMS          MFATokenType = "sms"
	MFATokenTypeEmail        MFATokenType = "email"
)

// AuditLog represents audit trail for authentication events
type AuditLog struct {
	database.BaseModel
	UserID       *uint             `json:"user_id"`
	HankoUserID  string            `json:"hanko_user_id"`
	Action       AuditAction       `json:"action" gorm:"not null"`
	Resource     string            `json:"resource"`
	Details      string            `json:"details" gorm:"type:text"`
	IPAddress    string            `json:"ip_address"`
	UserAgent    string            `json:"user_agent" gorm:"type:text"`
	Success      bool              `json:"success"`
	ErrorMessage string            `json:"error_message" gorm:"type:text"`
	Metadata     map[string]interface{} `json:"metadata" gorm:"serializer:json"`

	// Relationships
	User *User `json:"user" gorm:"foreignKey:UserID"`
}

type AuditAction string

const (
	AuditActionLogin              AuditAction = "login"
	AuditActionLogout             AuditAction = "logout"
	AuditActionRegister           AuditAction = "register"
	AuditActionPasswordReset      AuditAction = "password_reset"
	AuditActionEmailVerification  AuditAction = "email_verification"
	AuditActionPasskeyRegistration AuditAction = "passkey_registration"
	AuditActionPasskeyAuthentication AuditAction = "passkey_authentication"
	AuditActionRoleAssigned       AuditAction = "role_assigned"
	AuditActionRoleRemoved        AuditAction = "role_removed"
	AuditActionUserSuspended      AuditAction = "user_suspended"
	AuditActionUserActivated      AuditAction = "user_activated"
	AuditActionAccountLocked      AuditAction = "account_locked"
	AuditActionAccountUnlocked    AuditAction = "account_unlocked"
	AuditActionPermissionGranted  AuditAction = "permission_granted"
	AuditActionPermissionRevoked  AuditAction = "permission_revoked"
	// MFA Actions
	AuditActionMFAEnabled         AuditAction = "mfa_enabled"
	AuditActionMFADisabled        AuditAction = "mfa_disabled"
	AuditActionMFAVerification    AuditAction = "mfa_verification"
	AuditActionMFABackupUsed      AuditAction = "mfa_backup_used"
	// Password Reset Actions
	AuditActionPasswordResetRequested AuditAction = "password_reset_requested"
	AuditActionPasswordResetCompleted AuditAction = "password_reset_completed"
	// Email Verification Actions
	AuditActionEmailVerificationSent AuditAction = "email_verification_sent"
	AuditActionEmailVerificationCompleted AuditAction = "email_verification_completed"
)

// UserWithRoles represents a user with their roles and permissions
type UserWithRoles struct {
	User
	RoleNames   []string `json:"role_names"`
	Permissions []string `json:"permissions"`
}

// Methods for User model

func (u *User) SetClubID(clubID uint) {
	u.ClubID = clubID
}

func (u *User) GetFullName() string {
	if u.FirstName != "" && u.LastName != "" {
		return u.FirstName + " " + u.LastName
	}
	if u.FirstName != "" {
		return u.FirstName
	}
	if u.LastName != "" {
		return u.LastName
	}
	return u.Username
}

func (u *User) IsActive() bool {
	return u.Status == UserStatusActive && (u.LockedUntil == nil || u.LockedUntil.Before(time.Now()))
}

func (u *User) IsLocked() bool {
	return u.LockedUntil != nil && u.LockedUntil.After(time.Now())
}

func (u *User) Lock(duration time.Duration) {
	lockUntil := time.Now().Add(duration)
	u.LockedUntil = &lockUntil
	u.Status = UserStatusLocked
}

func (u *User) Unlock() {
	u.LockedUntil = nil
	u.FailedAttempts = 0
	if u.Status == UserStatusLocked {
		u.Status = UserStatusActive
	}
}

func (u *User) IncrementFailedAttempts() {
	u.FailedAttempts++
}

func (u *User) ResetFailedAttempts() {
	u.FailedAttempts = 0
}

// Methods for Club model

func (c *Club) SetClubID(clubID uint) {
	c.ClubID = clubID
}

func (c *Club) IsActive() bool {
	return c.Status == ClubStatusActive
}

// Methods for UserSession model

func (s *UserSession) SetClubID(clubID uint) {
	s.ClubID = clubID
}

func (s *UserSession) IsExpired() bool {
	return s.ExpiresAt.Before(time.Now())
}

func (s *UserSession) IsValid() bool {
	return s.IsActive && !s.IsExpired() && s.LogoutAt == nil
}

func (s *UserSession) Invalidate() {
	s.IsActive = false
	now := time.Now()
	s.LogoutAt = &now
}

func (s *UserSession) UpdateActivity() {
	s.LastActivityAt = time.Now()
}

// Methods for User model (MFA related)

func (u *User) EnableMFA(secret string, backupCodes []string) {
	u.MFAEnabled = true
	u.MFASecret = secret
	u.MFABackupCodes = strings.Join(backupCodes, ",")
}

func (u *User) DisableMFA() {
	u.MFAEnabled = false
	u.MFASecret = ""
	u.MFABackupCodes = ""
}

func (u *User) GetMFABackupCodes() []string {
	if u.MFABackupCodes == "" {
		return []string{}
	}
	return strings.Split(u.MFABackupCodes, ",")
}

func (u *User) UseBackupCode(code string) bool {
	backupCodes := u.GetMFABackupCodes()
	for i, backupCode := range backupCodes {
		if backupCode == code {
			// Remove the used code
			backupCodes = append(backupCodes[:i], backupCodes[i+1:]...)
			u.MFABackupCodes = strings.Join(backupCodes, ",")
			return true
		}
	}
	return false
}

func (u *User) SetPasswordResetToken(token string, expiry time.Time) {
	u.PasswordResetToken = token
	u.PasswordResetExpiresAt = &expiry
}

func (u *User) ClearPasswordResetToken() {
	u.PasswordResetToken = ""
	u.PasswordResetExpiresAt = nil
}

func (u *User) IsPasswordResetTokenValid(token string) bool {
	if u.PasswordResetToken == "" || u.PasswordResetExpiresAt == nil {
		return false
	}
	return u.PasswordResetToken == token && u.PasswordResetExpiresAt.After(time.Now())
}

func (u *User) SetEmailVerificationToken(token string, expiry time.Time) {
	u.EmailVerificationToken = token
	u.EmailVerificationExpiresAt = &expiry
}

func (u *User) ClearEmailVerificationToken() {
	u.EmailVerificationToken = ""
	u.EmailVerificationExpiresAt = nil
}

func (u *User) IsEmailVerificationTokenValid(token string) bool {
	if u.EmailVerificationToken == "" || u.EmailVerificationExpiresAt == nil {
		return false
	}
	return u.EmailVerificationToken == token && u.EmailVerificationExpiresAt.After(time.Now())
}

func (u *User) VerifyEmail() {
	u.EmailVerified = true
	u.ClearEmailVerificationToken()
}

// Methods for MFAToken model

func (m *MFAToken) SetClubID(clubID uint) {
	m.ClubID = clubID
}

func (m *MFAToken) IsExpired() bool {
	if m.ExpiresAt == nil {
		return false
	}
	return m.ExpiresAt.Before(time.Now())
}

func (m *MFAToken) MarkAsUsed() {
	m.Used = true
	now := time.Now()
	m.UsedAt = &now
}

// Methods for AuditLog model

func (a *AuditLog) SetClubID(clubID uint) {
	a.ClubID = clubID
}

// Helper functions for seeding default data

func GetDefaultRoles() []Role {
	return []Role{
		{
			BaseModel:   database.BaseModel{ClubID: 1}, // Default club
			Name:        RoleAdmin,
			Description: "Full system administration privileges",
			IsSystem:    true,
		},
		{
			BaseModel:   database.BaseModel{ClubID: 1},
			Name:        RoleManager,
			Description: "Club management privileges",
			IsSystem:    true,
		},
		{
			BaseModel:   database.BaseModel{ClubID: 1},
			Name:        RoleStaff,
			Description: "Club staff privileges",
			IsSystem:    true,
		},
		{
			BaseModel:   database.BaseModel{ClubID: 1},
			Name:        RoleMember,
			Description: "Regular club member privileges",
			IsSystem:    true,
		},
		{
			BaseModel:   database.BaseModel{ClubID: 1},
			Name:        RoleReciprocalAdmin,
			Description: "Reciprocal agreement administration",
			IsSystem:    true,
		},
		{
			BaseModel:   database.BaseModel{ClubID: 1},
			Name:        RoleGuest,
			Description: "Limited guest privileges",
			IsSystem:    true,
		},
	}
}

func GetDefaultPermissions() []Permission {
	permissions := []Permission{
		// User management
		{Name: "user.create", Description: "Create users", Resource: "user", Action: "create"},
		{Name: "user.read", Description: "Read user information", Resource: "user", Action: "read"},
		{Name: "user.update", Description: "Update user information", Resource: "user", Action: "update"},
		{Name: "user.delete", Description: "Delete users", Resource: "user", Action: "delete"},
		{Name: "user.suspend", Description: "Suspend users", Resource: "user", Action: "suspend"},

		// Role management
		{Name: "role.create", Description: "Create roles", Resource: "role", Action: "create"},
		{Name: "role.read", Description: "Read role information", Resource: "role", Action: "read"},
		{Name: "role.update", Description: "Update roles", Resource: "role", Action: "update"},
		{Name: "role.delete", Description: "Delete roles", Resource: "role", Action: "delete"},
		{Name: "role.assign", Description: "Assign roles to users", Resource: "role", Action: "assign"},

		// Member management
		{Name: "member.create", Description: "Create members", Resource: "member", Action: "create"},
		{Name: "member.read", Description: "Read member information", Resource: "member", Action: "read"},
		{Name: "member.update", Description: "Update member information", Resource: "member", Action: "update"},
		{Name: "member.delete", Description: "Delete members", Resource: "member", Action: "delete"},

		// Reciprocal management
		{Name: "reciprocal.create", Description: "Create reciprocal agreements", Resource: "reciprocal", Action: "create"},
		{Name: "reciprocal.read", Description: "Read reciprocal agreements", Resource: "reciprocal", Action: "read"},
		{Name: "reciprocal.update", Description: "Update reciprocal agreements", Resource: "reciprocal", Action: "update"},
		{Name: "reciprocal.approve", Description: "Approve reciprocal agreements", Resource: "reciprocal", Action: "approve"},

		// Visit management
		{Name: "visit.create", Description: "Record visits", Resource: "visit", Action: "create"},
		{Name: "visit.read", Description: "Read visit information", Resource: "visit", Action: "read"},
		{Name: "visit.verify", Description: "Verify visits", Resource: "visit", Action: "verify"},

		// Club management
		{Name: "club.read", Description: "Read club information", Resource: "club", Action: "read"},
		{Name: "club.update", Description: "Update club settings", Resource: "club", Action: "update"},

		// Analytics
		{Name: "analytics.read", Description: "Read analytics data", Resource: "analytics", Action: "read"},

		// Governance
		{Name: "governance.create", Description: "Create proposals", Resource: "governance", Action: "create"},
		{Name: "governance.vote", Description: "Vote on proposals", Resource: "governance", Action: "vote"},
		{Name: "governance.read", Description: "Read governance information", Resource: "governance", Action: "read"},

		// System
		{Name: "system.admin", Description: "System administration", Resource: "system", Action: "admin"},
		{Name: "audit.read", Description: "Read audit logs", Resource: "audit", Action: "read"},
	}

	// Set club ID for each permission
	for i := range permissions {
		permissions[i].ClubID = 1 // Default club
	}

	return permissions
}