package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Member represents a club member in the system
type Member struct {
	ID                 uint           `json:"id" gorm:"primaryKey"`
	ClubID             uint           `json:"club_id" gorm:"not null;index"`
	UserID             uint           `json:"user_id" gorm:"not null;index;uniqueIndex:idx_club_user"`
	MemberNumber       string         `json:"member_number" gorm:"uniqueIndex;size:50;not null"`
	MembershipType     MembershipType `json:"membership_type" gorm:"not null;default:'REGULAR'"`
	Status             MemberStatus   `json:"status" gorm:"not null;default:'ACTIVE'"`
	BlockchainIdentity string         `json:"blockchain_identity" gorm:"size:255"`

	// Profile relationship
	ProfileID uint           `json:"profile_id" gorm:"index"`
	Profile   *MemberProfile `json:"profile,omitempty" gorm:"foreignKey:ProfileID"`

	// Timestamps
	JoinedAt  time.Time      `json:"joined_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// MemberProfile contains detailed member information
type MemberProfile struct {
	ID              uint                 `json:"id" gorm:"primaryKey"`
	FirstName       string               `json:"first_name" gorm:"size:100;not null"`
	LastName        string               `json:"last_name" gorm:"size:100;not null"`
	DateOfBirth     *time.Time           `json:"date_of_birth"`
	PhoneNumber     string               `json:"phone_number" gorm:"size:20"`

	// Address relationship
	AddressID       *uint                `json:"address_id" gorm:"index"`
	Address         *Address             `json:"address,omitempty" gorm:"foreignKey:AddressID"`

	// Emergency contact relationship
	EmergencyContactID *uint             `json:"emergency_contact_id" gorm:"index"`
	EmergencyContact   *EmergencyContact `json:"emergency_contact,omitempty" gorm:"foreignKey:EmergencyContactID"`

	// Preferences relationship
	PreferencesID   *uint              `json:"preferences_id" gorm:"index"`
	Preferences     *MemberPreferences `json:"preferences,omitempty" gorm:"foreignKey:PreferencesID"`

	// Timestamps
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// Address represents a physical address
type Address struct {
	ID         uint   `json:"id" gorm:"primaryKey"`
	Street     string `json:"street" gorm:"size:255;not null"`
	City       string `json:"city" gorm:"size:100;not null"`
	State      string `json:"state" gorm:"size:100;not null"`
	PostalCode string `json:"postal_code" gorm:"size:20;not null"`
	Country    string `json:"country" gorm:"size:100;not null"`

	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// EmergencyContact represents emergency contact information
type EmergencyContact struct {
	ID           uint   `json:"id" gorm:"primaryKey"`
	Name         string `json:"name" gorm:"size:200;not null"`
	Relationship string `json:"relationship" gorm:"size:100"`
	PhoneNumber  string `json:"phone_number" gorm:"size:20;not null"`
	Email        string `json:"email" gorm:"size:255"`

	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// MemberPreferences represents member notification and communication preferences
type MemberPreferences struct {
	ID                 uint `json:"id" gorm:"primaryKey"`
	EmailNotifications bool `json:"email_notifications" gorm:"default:true"`
	SMSNotifications   bool `json:"sms_notifications" gorm:"default:false"`
	PushNotifications  bool `json:"push_notifications" gorm:"default:true"`
	MarketingEmails    bool `json:"marketing_emails" gorm:"default:false"`

	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// MembershipType enum for different membership levels
type MembershipType string

const (
	MembershipTypeRegular   MembershipType = "REGULAR"
	MembershipTypeVIP       MembershipType = "VIP"
	MembershipTypeCorporate MembershipType = "CORPORATE"
	MembershipTypeStudent   MembershipType = "STUDENT"
	MembershipTypeSenior    MembershipType = "SENIOR"
)

// MemberStatus enum for member account status
type MemberStatus string

const (
	MemberStatusActive    MemberStatus = "ACTIVE"
	MemberStatusSuspended MemberStatus = "SUSPENDED"
	MemberStatusExpired   MemberStatus = "EXPIRED"
	MemberStatusPending   MemberStatus = "PENDING"
)

// TableName overrides the table name for GORM
func (Member) TableName() string {
	return "members"
}

func (MemberProfile) TableName() string {
	return "member_profiles"
}

func (Address) TableName() string {
	return "addresses"
}

func (EmergencyContact) TableName() string {
	return "emergency_contacts"
}

func (MemberPreferences) TableName() string {
	return "member_preferences"
}

// BeforeCreate generates member number if not provided
func (m *Member) BeforeCreate(tx *gorm.DB) error {
	if m.MemberNumber == "" {
		// Generate member number based on club ID and timestamp
		m.MemberNumber = generateMemberNumber(m.ClubID)
	}
	return nil
}

// generateMemberNumber creates a unique member number
func generateMemberNumber(clubID uint) string {
	timestamp := time.Now().Unix()
	return fmt.Sprintf("M%d%d", clubID, timestamp%1000000)
}

// IsActive checks if the member is in active status
func (m *Member) IsActive() bool {
	return m.Status == MemberStatusActive
}

// CanAccess checks if member can access club facilities
func (m *Member) CanAccess() bool {
	return m.Status == MemberStatusActive || m.Status == MemberStatusPending
}

// GetFullName returns the member's full name
func (mp *MemberProfile) GetFullName() string {
	return fmt.Sprintf("%s %s", mp.FirstName, mp.LastName)
}

// GetFormattedAddress returns formatted address string
func (a *Address) GetFormattedAddress() string {
	return fmt.Sprintf("%s, %s, %s %s, %s",
		a.Street, a.City, a.State, a.PostalCode, a.Country)
}