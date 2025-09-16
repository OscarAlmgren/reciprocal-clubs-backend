package models

import (
	"time"

	"gorm.io/gorm"
)

// Agreement represents a reciprocal agreement between clubs
type Agreement struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	ProposingClubID uint         `json:"proposing_club_id" gorm:"not null;index"`
	TargetClubID  uint           `json:"target_club_id" gorm:"not null;index"`
	Title         string         `json:"title" gorm:"size:255;not null"`
	Description   string         `json:"description" gorm:"type:text"`
	Terms         AgreementTerms `json:"terms" gorm:"type:jsonb"`
	Status        AgreementStatus `json:"status" gorm:"type:varchar(20);not null;default:'pending'"`
	ProposedAt    time.Time      `json:"proposed_at" gorm:"not null"`
	ReviewedAt    *time.Time     `json:"reviewed_at,omitempty"`
	ActivatedAt   *time.Time     `json:"activated_at,omitempty"`
	ExpiresAt     *time.Time     `json:"expires_at,omitempty"`
	
	// Metadata
	ProposedByID  string    `json:"proposed_by_id" gorm:"size:255;not null"`
	ReviewedByID  *string   `json:"reviewed_by_id,omitempty" gorm:"size:255"`
	
	// Blockchain
	BlockchainTxID *string `json:"blockchain_tx_id,omitempty" gorm:"size:255;unique"`
	
	// GORM fields
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	
	// Relationships
	Visits []Visit `json:"visits,omitempty" gorm:"foreignKey:AgreementID"`
}

// AgreementTerms holds the terms and conditions of an agreement
type AgreementTerms struct {
	MaxVisitsPerMonth     int                    `json:"max_visits_per_month"`
	MaxVisitsPerYear      int                    `json:"max_visits_per_year"`
	AllowedVisitDays      []string              `json:"allowed_visit_days"` // monday, tuesday, etc.
	AllowedVisitTimes     *TimeRange            `json:"allowed_visit_times,omitempty"`
	RequireAdvanceBooking bool                   `json:"require_advance_booking"`
	AdvanceBookingDays    int                    `json:"advance_booking_days"`
	AllowedFacilities     []string              `json:"allowed_facilities"`
	ExcludedDates         []time.Time           `json:"excluded_dates"`
	SpecialConditions     map[string]interface{} `json:"special_conditions,omitempty"`
	DiscountPercentage    float64               `json:"discount_percentage"`
	Currency              string                `json:"currency"`
}

// TimeRange represents a time range
type TimeRange struct {
	Start string `json:"start"` // HH:MM format
	End   string `json:"end"`   // HH:MM format
}

// AgreementStatus represents the status of an agreement
type AgreementStatus string

const (
	AgreementStatusPending   AgreementStatus = "pending"
	AgreementStatusApproved  AgreementStatus = "approved"
	AgreementStatusRejected  AgreementStatus = "rejected"
	AgreementStatusActive    AgreementStatus = "active"
	AgreementStatusSuspended AgreementStatus = "suspended"
	AgreementStatusExpired   AgreementStatus = "expired"
	AgreementStatusCancelled AgreementStatus = "cancelled"
)

// Visit represents a member's visit to a reciprocal club
type Visit struct {
	ID           uint        `json:"id" gorm:"primaryKey"`
	AgreementID  uint        `json:"agreement_id" gorm:"not null;index"`
	MemberID     uint        `json:"member_id" gorm:"not null;index"`
	VisitingClubID uint      `json:"visiting_club_id" gorm:"not null;index"`
	HomeClubID   uint        `json:"home_club_id" gorm:"not null;index"`
	
	// Visit details
	VisitDate     time.Time   `json:"visit_date" gorm:"not null;index"`
	CheckInTime   *time.Time  `json:"check_in_time,omitempty"`
	CheckOutTime  *time.Time  `json:"check_out_time,omitempty"`
	Duration      *int        `json:"duration,omitempty"` // in minutes
	
	// Visit purpose and details
	Purpose       string      `json:"purpose" gorm:"size:500"`
	FacilitiesUsed []string   `json:"facilities_used" gorm:"type:jsonb"`
	GuestCount    int         `json:"guest_count" gorm:"default:0"`
	
	// Status and verification
	Status        VisitStatus `json:"status" gorm:"type:varchar(20);not null;default:'pending'"`
	VerificationCode string   `json:"verification_code" gorm:"size:50;unique;not null"`
	QRCodeData    string      `json:"qr_code_data" gorm:"type:text"`
	
	// Staff verification
	VerifiedBy    *string     `json:"verified_by,omitempty" gorm:"size:255"`
	VerifiedAt    *time.Time  `json:"verified_at,omitempty"`
	
	// Cost and billing
	EstimatedCost float64     `json:"estimated_cost" gorm:"type:decimal(10,2)"`
	ActualCost    *float64    `json:"actual_cost,omitempty" gorm:"type:decimal(10,2)"`
	DiscountApplied float64   `json:"discount_applied" gorm:"type:decimal(10,2);default:0"`
	Currency      string      `json:"currency" gorm:"size:3;default:'USD'"`
	
	// Feedback and rating
	MemberRating   *int    `json:"member_rating,omitempty" gorm:"check:member_rating >= 1 AND member_rating <= 5"`
	MemberFeedback *string `json:"member_feedback,omitempty" gorm:"type:text"`
	ClubRating     *int    `json:"club_rating,omitempty" gorm:"check:club_rating >= 1 AND club_rating <= 5"`
	ClubFeedback   *string `json:"club_feedback,omitempty" gorm:"type:text"`
	
	// Blockchain
	BlockchainTxID *string `json:"blockchain_tx_id,omitempty" gorm:"size:255;unique"`
	
	// GORM fields
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	
	// Relationships
	Agreement Agreement `json:"agreement,omitempty" gorm:"foreignKey:AgreementID"`
}

// VisitStatus represents the status of a visit
type VisitStatus string

const (
	VisitStatusPending    VisitStatus = "pending"
	VisitStatusConfirmed  VisitStatus = "confirmed"
	VisitStatusCheckedIn  VisitStatus = "checked_in"
	VisitStatusCompleted  VisitStatus = "completed"
	VisitStatusCancelled  VisitStatus = "cancelled"
	VisitStatusNoShow     VisitStatus = "no_show"
)

// VisitRestriction represents visit restrictions for a member/club combination
type VisitRestriction struct {
	ID           uint   `json:"id" gorm:"primaryKey"`
	AgreementID  uint   `json:"agreement_id" gorm:"not null;index"`
	MemberID     *uint  `json:"member_id,omitempty" gorm:"index"` // null means applies to all members
	ClubID       *uint  `json:"club_id,omitempty" gorm:"index"`   // null means applies to all clubs
	
	// Restriction details
	RestrictionType RestrictionType `json:"restriction_type" gorm:"type:varchar(20);not null"`
	Description     string          `json:"description" gorm:"size:500"`
	StartDate       *time.Time      `json:"start_date,omitempty"`
	EndDate         *time.Time      `json:"end_date,omitempty"`
	IsActive        bool            `json:"is_active" gorm:"default:true"`
	
	// Who applied the restriction
	AppliedByID  string     `json:"applied_by_id" gorm:"size:255;not null"`
	AppliedAt    time.Time  `json:"applied_at" gorm:"not null"`
	Reason       string     `json:"reason" gorm:"size:1000"`
	
	// GORM fields
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// RestrictionType represents types of visit restrictions
type RestrictionType string

const (
	RestrictionTypeSuspension  RestrictionType = "suspension"
	RestrictionTypeLimitation  RestrictionType = "limitation"
	RestrictionTypeBlacklist   RestrictionType = "blacklist"
	RestrictionTypeTemporary   RestrictionType = "temporary"
)

// VisitStats holds visit statistics
type VisitStats struct {
	MemberID        uint    `json:"member_id"`
	ClubID          uint    `json:"club_id"`
	AgreementID     uint    `json:"agreement_id"`
	Month           int     `json:"month"`
	Year            int     `json:"year"`
	VisitCount      int     `json:"visit_count"`
	TotalDuration   int     `json:"total_duration"` // in minutes
	TotalCost       float64 `json:"total_cost"`
	AverageRating   float64 `json:"average_rating"`
	LastVisitDate   *time.Time `json:"last_visit_date,omitempty"`
}

// Validation methods

// IsValidStatus checks if the agreement status is valid
func (s AgreementStatus) IsValid() bool {
	switch s {
	case AgreementStatusPending, AgreementStatusApproved, AgreementStatusRejected,
		 AgreementStatusActive, AgreementStatusSuspended, AgreementStatusExpired,
		 AgreementStatusCancelled:
		return true
	default:
		return false
	}
}

// CanTransitionTo checks if the agreement can transition to the given status
func (a *Agreement) CanTransitionTo(newStatus AgreementStatus) bool {
	switch a.Status {
	case AgreementStatusPending:
		return newStatus == AgreementStatusApproved || newStatus == AgreementStatusRejected
	case AgreementStatusApproved:
		return newStatus == AgreementStatusActive || newStatus == AgreementStatusCancelled
	case AgreementStatusActive:
		return newStatus == AgreementStatusSuspended || newStatus == AgreementStatusExpired || newStatus == AgreementStatusCancelled
	case AgreementStatusSuspended:
		return newStatus == AgreementStatusActive || newStatus == AgreementStatusCancelled
	default:
		return false
	}
}

// IsActive checks if the agreement is currently active
func (a *Agreement) IsActive() bool {
	return a.Status == AgreementStatusActive
}

// IsExpired checks if the agreement has expired
func (a *Agreement) IsExpired() bool {
	if a.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*a.ExpiresAt)
}

// IsValidVisitStatus checks if the visit status is valid
func (s VisitStatus) IsValid() bool {
	switch s {
	case VisitStatusPending, VisitStatusConfirmed, VisitStatusCheckedIn,
		 VisitStatusCompleted, VisitStatusCancelled, VisitStatusNoShow:
		return true
	default:
		return false
	}
}

// CanTransitionTo checks if the visit can transition to the given status
func (v *Visit) CanTransitionTo(newStatus VisitStatus) bool {
	switch v.Status {
	case VisitStatusPending:
		return newStatus == VisitStatusConfirmed || newStatus == VisitStatusCancelled
	case VisitStatusConfirmed:
		return newStatus == VisitStatusCheckedIn || newStatus == VisitStatusCancelled || newStatus == VisitStatusNoShow
	case VisitStatusCheckedIn:
		return newStatus == VisitStatusCompleted
	default:
		return false
	}
}

// IsCompleted checks if the visit is completed
func (v *Visit) IsCompleted() bool {
	return v.Status == VisitStatusCompleted
}

// CalculateDuration calculates the visit duration if both check-in and check-out times are set
func (v *Visit) CalculateDuration() *int {
	if v.CheckInTime != nil && v.CheckOutTime != nil {
		duration := int(v.CheckOutTime.Sub(*v.CheckInTime).Minutes())
		return &duration
	}
	return nil
}

// TableName returns the table name for Agreement
func (Agreement) TableName() string {
	return "reciprocal_agreements"
}

// TableName returns the table name for Visit
func (Visit) TableName() string {
	return "reciprocal_visits"
}

// TableName returns the table name for VisitRestriction
func (VisitRestriction) TableName() string {
	return "reciprocal_visit_restrictions"
}