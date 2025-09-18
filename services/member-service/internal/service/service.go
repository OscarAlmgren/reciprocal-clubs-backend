package service

import (
	"context"
	"fmt"

	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/messaging"
	"reciprocal-clubs-backend/services/member-service/internal/models"
	"reciprocal-clubs-backend/services/member-service/internal/repository"
)

// Service interface defines member business logic operations
type Service interface {
	// Member management
	CreateMember(ctx context.Context, req *CreateMemberRequest) (*models.Member, error)
	GetMember(ctx context.Context, id uint) (*models.Member, error)
	GetMemberByUserID(ctx context.Context, userID uint) (*models.Member, error)
	GetMemberByMemberNumber(ctx context.Context, memberNumber string) (*models.Member, error)
	GetMembersByClub(ctx context.Context, clubID uint, limit, offset int) ([]*models.Member, error)
	UpdateMemberProfile(ctx context.Context, memberID uint, req *UpdateProfileRequest) (*models.Member, error)
	SuspendMember(ctx context.Context, memberID uint, reason string) (*models.Member, error)
	ReactivateMember(ctx context.Context, memberID uint) (*models.Member, error)
	DeleteMember(ctx context.Context, memberID uint) error

	// Member status and validation
	ValidateMemberAccess(ctx context.Context, memberID uint) (bool, error)
	CheckMembershipStatus(ctx context.Context, memberID uint) (*MembershipStatus, error)

	// Analytics and reporting
	GetMemberAnalytics(ctx context.Context, clubID uint) (*MemberAnalytics, error)
	GetMemberCountByStatus(ctx context.Context, status models.MemberStatus) (int64, error)

	// Health check
	HealthCheck(ctx context.Context) error
}

// DTOs for service operations
type CreateMemberRequest struct {
	ClubID         uint                        `json:"club_id" validate:"required"`
	UserID         uint                        `json:"user_id" validate:"required"`
	MembershipType models.MembershipType       `json:"membership_type" validate:"required"`
	Profile        CreateProfileRequest        `json:"profile" validate:"required"`
}

type CreateProfileRequest struct {
	FirstName       string                      `json:"first_name" validate:"required"`
	LastName        string                      `json:"last_name" validate:"required"`
	DateOfBirth     *string                     `json:"date_of_birth,omitempty"`
	PhoneNumber     string                      `json:"phone_number,omitempty"`
	Address         *CreateAddressRequest       `json:"address,omitempty"`
	EmergencyContact *CreateEmergencyContactRequest `json:"emergency_contact,omitempty"`
	Preferences     *CreatePreferencesRequest   `json:"preferences,omitempty"`
}

type CreateAddressRequest struct {
	Street     string `json:"street" validate:"required"`
	City       string `json:"city" validate:"required"`
	State      string `json:"state" validate:"required"`
	PostalCode string `json:"postal_code" validate:"required"`
	Country    string `json:"country" validate:"required"`
}

type CreateEmergencyContactRequest struct {
	Name         string `json:"name" validate:"required"`
	Relationship string `json:"relationship,omitempty"`
	PhoneNumber  string `json:"phone_number" validate:"required"`
	Email        string `json:"email,omitempty"`
}

type CreatePreferencesRequest struct {
	EmailNotifications bool `json:"email_notifications"`
	SMSNotifications   bool `json:"sms_notifications"`
	PushNotifications  bool `json:"push_notifications"`
	MarketingEmails    bool `json:"marketing_emails"`
}

type UpdateProfileRequest struct {
	FirstName       *string                     `json:"first_name,omitempty"`
	LastName        *string                     `json:"last_name,omitempty"`
	DateOfBirth     *string                     `json:"date_of_birth,omitempty"`
	PhoneNumber     *string                     `json:"phone_number,omitempty"`
	Address         *CreateAddressRequest       `json:"address,omitempty"`
	EmergencyContact *CreateEmergencyContactRequest `json:"emergency_contact,omitempty"`
	Preferences     *CreatePreferencesRequest   `json:"preferences,omitempty"`
}

type MembershipStatus struct {
	MemberID       uint                `json:"member_id"`
	Status         models.MemberStatus `json:"status"`
	MembershipType models.MembershipType `json:"membership_type"`
	CanAccess      bool                `json:"can_access"`
	JoinedAt       string              `json:"joined_at"`
	ExpiresAt      *string             `json:"expires_at,omitempty"`
}

type MemberAnalytics struct {
	TotalMembers        int64                               `json:"total_members"`
	ActiveMembers       int64                               `json:"active_members"`
	NewMembersThisMonth int64                               `json:"new_members_this_month"`
	MembershipDistribution []MembershipTypeCount           `json:"membership_distribution"`
	StatusDistribution  []MemberStatusCount                 `json:"status_distribution"`
}

type MembershipTypeCount struct {
	Type  models.MembershipType `json:"type"`
	Count int64                 `json:"count"`
}

type MemberStatusCount struct {
	Status models.MemberStatus `json:"status"`
	Count  int64               `json:"count"`
}

// memberService implements the Service interface
type memberService struct {
	repo       repository.Repository
	logger     logging.Logger
	messageBus messaging.MessageBus
}

// NewService creates a new member service instance
func NewService(repo repository.Repository, logger logging.Logger, messageBus messaging.MessageBus) Service {
	return &memberService{
		repo:       repo,
		logger:     logger,
		messageBus: messageBus,
	}
}

// CreateMember creates a new member with full profile
func (s *memberService) CreateMember(ctx context.Context, req *CreateMemberRequest) (*models.Member, error) {
	s.logger.Info("Creating new member", map[string]interface{}{
		"user_id":         req.UserID,
		"club_id":         req.ClubID,
		"membership_type": req.MembershipType,
	})

	// Check if member already exists for this user in this club
	existingMember, err := s.repo.GetMemberByUserID(ctx, req.UserID)
	if err == nil && existingMember != nil {
		return nil, fmt.Errorf("member already exists for user %d", req.UserID)
	}

	// Create member profile first
	profile := &models.MemberProfile{
		FirstName:   req.Profile.FirstName,
		LastName:    req.Profile.LastName,
		PhoneNumber: req.Profile.PhoneNumber,
	}

	// Parse date of birth if provided
	if req.Profile.DateOfBirth != nil {
		// Note: In a real implementation, you'd parse the date string
		// For now, we'll leave it nil
	}

	// Create address if provided
	if req.Profile.Address != nil {
		address := &models.Address{
			Street:     req.Profile.Address.Street,
			City:       req.Profile.Address.City,
			State:      req.Profile.Address.State,
			PostalCode: req.Profile.Address.PostalCode,
			Country:    req.Profile.Address.Country,
		}
		if err := s.repo.CreateAddress(ctx, address); err != nil {
			return nil, fmt.Errorf("failed to create address: %w", err)
		}
		profile.AddressID = &address.ID
	}

	// Create emergency contact if provided
	if req.Profile.EmergencyContact != nil {
		contact := &models.EmergencyContact{
			Name:         req.Profile.EmergencyContact.Name,
			Relationship: req.Profile.EmergencyContact.Relationship,
			PhoneNumber:  req.Profile.EmergencyContact.PhoneNumber,
			Email:        req.Profile.EmergencyContact.Email,
		}
		if err := s.repo.CreateEmergencyContact(ctx, contact); err != nil {
			return nil, fmt.Errorf("failed to create emergency contact: %w", err)
		}
		profile.EmergencyContactID = &contact.ID
	}

	// Create preferences with defaults if not provided
	preferences := &models.MemberPreferences{
		EmailNotifications: true,
		SMSNotifications:   false,
		PushNotifications:  true,
		MarketingEmails:    false,
	}
	if req.Profile.Preferences != nil {
		preferences.EmailNotifications = req.Profile.Preferences.EmailNotifications
		preferences.SMSNotifications = req.Profile.Preferences.SMSNotifications
		preferences.PushNotifications = req.Profile.Preferences.PushNotifications
		preferences.MarketingEmails = req.Profile.Preferences.MarketingEmails
	}
	if err := s.repo.CreatePreferences(ctx, preferences); err != nil {
		return nil, fmt.Errorf("failed to create preferences: %w", err)
	}
	profile.PreferencesID = &preferences.ID

	// Create the profile
	if err := s.repo.CreateProfile(ctx, profile); err != nil {
		return nil, fmt.Errorf("failed to create profile: %w", err)
	}

	// Create the member
	member := &models.Member{
		ClubID:         req.ClubID,
		UserID:         req.UserID,
		MembershipType: req.MembershipType,
		Status:         models.MemberStatusActive,
		ProfileID:      profile.ID,
	}

	if err := s.repo.CreateMember(ctx, member); err != nil {
		return nil, fmt.Errorf("failed to create member: %w", err)
	}

	// Reload member with full profile
	createdMember, err := s.repo.GetMemberByID(ctx, member.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to reload created member: %w", err)
	}

	// Publish member created event
	s.publishMemberEvent(ctx, "member.created", createdMember)

	s.logger.Info("Member created successfully", map[string]interface{}{
		"member_id":     createdMember.ID,
		"member_number": createdMember.MemberNumber,
		"user_id":       createdMember.UserID,
		"club_id":       createdMember.ClubID,
	})

	return createdMember, nil
}

// GetMember retrieves a member by ID
func (s *memberService) GetMember(ctx context.Context, id uint) (*models.Member, error) {
	member, err := s.repo.GetMemberByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get member: %w", err)
	}
	return member, nil
}

// GetMemberByUserID retrieves a member by user ID
func (s *memberService) GetMemberByUserID(ctx context.Context, userID uint) (*models.Member, error) {
	member, err := s.repo.GetMemberByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get member by user ID: %w", err)
	}
	return member, nil
}

// GetMemberByMemberNumber retrieves a member by member number
func (s *memberService) GetMemberByMemberNumber(ctx context.Context, memberNumber string) (*models.Member, error) {
	member, err := s.repo.GetMemberByMemberNumber(ctx, memberNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get member by member number: %w", err)
	}
	return member, nil
}

// GetMembersByClub retrieves members for a specific club
func (s *memberService) GetMembersByClub(ctx context.Context, clubID uint, limit, offset int) ([]*models.Member, error) {
	members, err := s.repo.GetMembersByClubID(ctx, clubID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get members by club: %w", err)
	}
	return members, nil
}

// UpdateMemberProfile updates a member's profile information
func (s *memberService) UpdateMemberProfile(ctx context.Context, memberID uint, req *UpdateProfileRequest) (*models.Member, error) {
	// Get existing member
	member, err := s.repo.GetMemberByID(ctx, memberID)
	if err != nil {
		return nil, fmt.Errorf("failed to get member: %w", err)
	}

	if member.Profile == nil {
		return nil, fmt.Errorf("member profile not found")
	}

	// Update profile fields if provided
	if req.FirstName != nil {
		member.Profile.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		member.Profile.LastName = *req.LastName
	}
	if req.PhoneNumber != nil {
		member.Profile.PhoneNumber = *req.PhoneNumber
	}

	// Update profile
	if err := s.repo.UpdateProfile(ctx, member.Profile); err != nil {
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}

	// Publish member updated event
	s.publishMemberEvent(ctx, "member.updated", member)

	s.logger.Info("Member profile updated", map[string]interface{}{
		"member_id": memberID,
	})

	return member, nil
}

// SuspendMember suspends a member account
func (s *memberService) SuspendMember(ctx context.Context, memberID uint, reason string) (*models.Member, error) {
	member, err := s.repo.GetMemberByID(ctx, memberID)
	if err != nil {
		return nil, fmt.Errorf("failed to get member: %w", err)
	}

	member.Status = models.MemberStatusSuspended

	if err := s.repo.UpdateMember(ctx, member); err != nil {
		return nil, fmt.Errorf("failed to suspend member: %w", err)
	}

	// Publish member suspended event
	s.publishMemberEvent(ctx, "member.suspended", member)

	s.logger.Info("Member suspended", map[string]interface{}{
		"member_id": memberID,
		"reason":    reason,
	})

	return member, nil
}

// ReactivateMember reactivates a suspended member
func (s *memberService) ReactivateMember(ctx context.Context, memberID uint) (*models.Member, error) {
	member, err := s.repo.GetMemberByID(ctx, memberID)
	if err != nil {
		return nil, fmt.Errorf("failed to get member: %w", err)
	}

	member.Status = models.MemberStatusActive

	if err := s.repo.UpdateMember(ctx, member); err != nil {
		return nil, fmt.Errorf("failed to reactivate member: %w", err)
	}

	// Publish member reactivated event
	s.publishMemberEvent(ctx, "member.reactivated", member)

	s.logger.Info("Member reactivated", map[string]interface{}{
		"member_id": memberID,
	})

	return member, nil
}

// DeleteMember deletes a member
func (s *memberService) DeleteMember(ctx context.Context, memberID uint) error {
	member, err := s.repo.GetMemberByID(ctx, memberID)
	if err != nil {
		return fmt.Errorf("failed to get member: %w", err)
	}

	if err := s.repo.DeleteMember(ctx, memberID); err != nil {
		return fmt.Errorf("failed to delete member: %w", err)
	}

	// Publish member deleted event
	s.publishMemberEvent(ctx, "member.deleted", member)

	s.logger.Info("Member deleted", map[string]interface{}{
		"member_id": memberID,
	})

	return nil
}

// ValidateMemberAccess checks if a member can access club facilities
func (s *memberService) ValidateMemberAccess(ctx context.Context, memberID uint) (bool, error) {
	member, err := s.repo.GetMemberByID(ctx, memberID)
	if err != nil {
		return false, fmt.Errorf("failed to get member: %w", err)
	}

	return member.CanAccess(), nil
}

// CheckMembershipStatus returns detailed membership status
func (s *memberService) CheckMembershipStatus(ctx context.Context, memberID uint) (*MembershipStatus, error) {
	member, err := s.repo.GetMemberByID(ctx, memberID)
	if err != nil {
		return nil, fmt.Errorf("failed to get member: %w", err)
	}

	status := &MembershipStatus{
		MemberID:       member.ID,
		Status:         member.Status,
		MembershipType: member.MembershipType,
		CanAccess:      member.CanAccess(),
		JoinedAt:       member.JoinedAt.Format("2006-01-02T15:04:05Z"),
	}

	return status, nil
}

// GetMemberAnalytics returns analytics for a club's membership
func (s *memberService) GetMemberAnalytics(ctx context.Context, clubID uint) (*MemberAnalytics, error) {
	totalMembers, err := s.repo.GetMemberCountByClub(ctx, clubID)
	if err != nil {
		return nil, fmt.Errorf("failed to get total member count: %w", err)
	}

	activeMembers, err := s.repo.GetActiveMemberCountByClub(ctx, clubID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active member count: %w", err)
	}

	// TODO: Implement new members this month calculation
	// TODO: Implement membership type distribution
	// TODO: Implement status distribution

	analytics := &MemberAnalytics{
		TotalMembers:           totalMembers,
		ActiveMembers:          activeMembers,
		NewMembersThisMonth:    0, // TODO: Calculate
		MembershipDistribution: []MembershipTypeCount{}, // TODO: Calculate
		StatusDistribution:     []MemberStatusCount{}, // TODO: Calculate
	}

	return analytics, nil
}

// GetMemberCountByStatus returns count of members by status
func (s *memberService) GetMemberCountByStatus(ctx context.Context, status models.MemberStatus) (int64, error) {
	// This would require a new repository method
	// For now, return 0
	return 0, nil
}

// HealthCheck performs service health check
func (s *memberService) HealthCheck(ctx context.Context) error {
	return s.repo.HealthCheck(ctx)
}

// publishMemberEvent publishes member-related events to the message bus
func (s *memberService) publishMemberEvent(ctx context.Context, eventType string, member *models.Member) {
	if s.messageBus == nil {
		return
	}

	event := map[string]interface{}{
		"event_type":      eventType,
		"member_id":       member.ID,
		"member_number":   member.MemberNumber,
		"user_id":         member.UserID,
		"club_id":         member.ClubID,
		"membership_type": member.MembershipType,
		"status":          member.Status,
		"timestamp":       member.UpdatedAt,
	}

	if err := s.messageBus.Publish(ctx, "member.events", event); err != nil {
		s.logger.Error("Failed to publish member event", map[string]interface{}{
			"error":      err.Error(),
			"event_type": eventType,
			"member_id":  member.ID,
		})
	}
}