package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"reciprocal-clubs-backend/pkg/shared/database"
	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/services/member-service/internal/models"
)

// Repository interface defines member data operations
type Repository interface {
	// Member operations
	CreateMember(ctx context.Context, member *models.Member) error
	GetMemberByID(ctx context.Context, id uint) (*models.Member, error)
	GetMemberByUserID(ctx context.Context, userID uint) (*models.Member, error)
	GetMemberByMemberNumber(ctx context.Context, memberNumber string) (*models.Member, error)
	GetMembersByClubID(ctx context.Context, clubID uint, limit, offset int) ([]*models.Member, error)
	UpdateMember(ctx context.Context, member *models.Member) error
	DeleteMember(ctx context.Context, id uint) error

	// Profile operations
	CreateProfile(ctx context.Context, profile *models.MemberProfile) error
	GetProfileByID(ctx context.Context, id uint) (*models.MemberProfile, error)
	UpdateProfile(ctx context.Context, profile *models.MemberProfile) error

	// Address operations
	CreateAddress(ctx context.Context, address *models.Address) error
	UpdateAddress(ctx context.Context, address *models.Address) error

	// Emergency contact operations
	CreateEmergencyContact(ctx context.Context, contact *models.EmergencyContact) error
	UpdateEmergencyContact(ctx context.Context, contact *models.EmergencyContact) error

	// Preferences operations
	CreatePreferences(ctx context.Context, prefs *models.MemberPreferences) error
	UpdatePreferences(ctx context.Context, prefs *models.MemberPreferences) error

	// Analytics and reporting
	GetMemberCountByClub(ctx context.Context, clubID uint) (int64, error)
	GetActiveMemberCountByClub(ctx context.Context, clubID uint) (int64, error)
	GetMembersByStatus(ctx context.Context, status models.MemberStatus, limit, offset int) ([]*models.Member, error)
	GetMembersByMembershipType(ctx context.Context, membershipType models.MembershipType, limit, offset int) ([]*models.Member, error)

	// Health check
	HealthCheck(ctx context.Context) error
}

// memberRepository implements the Repository interface
type memberRepository struct {
	db     *gorm.DB
	logger logging.Logger
}

// NewRepository creates a new member repository instance
func NewRepository(db *database.Database, logger logging.Logger) Repository {
	return &memberRepository{
		db:     db.DB,
		logger: logger,
	}
}

// CreateMember creates a new member record
func (r *memberRepository) CreateMember(ctx context.Context, member *models.Member) error {
	result := r.db.WithContext(ctx).Create(member)
	if result.Error != nil {
		r.logger.Error("Failed to create member", map[string]interface{}{
			"error":   result.Error.Error(),
			"user_id": member.UserID,
			"club_id": member.ClubID,
		})
		return fmt.Errorf("failed to create member: %w", result.Error)
	}

	r.logger.Info("Member created successfully", map[string]interface{}{
		"member_id":     member.ID,
		"member_number": member.MemberNumber,
		"user_id":       member.UserID,
		"club_id":       member.ClubID,
	})

	return nil
}

// GetMemberByID retrieves a member by ID with full profile
func (r *memberRepository) GetMemberByID(ctx context.Context, id uint) (*models.Member, error) {
	var member models.Member
	result := r.db.WithContext(ctx).
		Preload("Profile").
		Preload("Profile.Address").
		Preload("Profile.EmergencyContact").
		Preload("Profile.Preferences").
		First(&member, id)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("member not found with ID %d", id)
		}
		r.logger.Error("Failed to get member by ID", map[string]interface{}{
			"error":     result.Error.Error(),
			"member_id": id,
		})
		return nil, fmt.Errorf("failed to get member: %w", result.Error)
	}

	return &member, nil
}

// GetMemberByUserID retrieves a member by user ID
func (r *memberRepository) GetMemberByUserID(ctx context.Context, userID uint) (*models.Member, error) {
	var member models.Member
	result := r.db.WithContext(ctx).
		Preload("Profile").
		Preload("Profile.Address").
		Preload("Profile.EmergencyContact").
		Preload("Profile.Preferences").
		Where("user_id = ?", userID).
		First(&member)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("member not found for user ID %d", userID)
		}
		r.logger.Error("Failed to get member by user ID", map[string]interface{}{
			"error":   result.Error.Error(),
			"user_id": userID,
		})
		return nil, fmt.Errorf("failed to get member: %w", result.Error)
	}

	return &member, nil
}

// GetMemberByMemberNumber retrieves a member by member number
func (r *memberRepository) GetMemberByMemberNumber(ctx context.Context, memberNumber string) (*models.Member, error) {
	var member models.Member
	result := r.db.WithContext(ctx).
		Preload("Profile").
		Preload("Profile.Address").
		Preload("Profile.EmergencyContact").
		Preload("Profile.Preferences").
		Where("member_number = ?", memberNumber).
		First(&member)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("member not found with number %s", memberNumber)
		}
		r.logger.Error("Failed to get member by member number", map[string]interface{}{
			"error":         result.Error.Error(),
			"member_number": memberNumber,
		})
		return nil, fmt.Errorf("failed to get member: %w", result.Error)
	}

	return &member, nil
}

// GetMembersByClubID retrieves members for a specific club with pagination
func (r *memberRepository) GetMembersByClubID(ctx context.Context, clubID uint, limit, offset int) ([]*models.Member, error) {
	var members []*models.Member
	result := r.db.WithContext(ctx).
		Preload("Profile").
		Where("club_id = ?", clubID).
		Limit(limit).
		Offset(offset).
		Find(&members)

	if result.Error != nil {
		r.logger.Error("Failed to get members by club ID", map[string]interface{}{
			"error":   result.Error.Error(),
			"club_id": clubID,
		})
		return nil, fmt.Errorf("failed to get members: %w", result.Error)
	}

	return members, nil
}

// UpdateMember updates an existing member record
func (r *memberRepository) UpdateMember(ctx context.Context, member *models.Member) error {
	result := r.db.WithContext(ctx).Save(member)
	if result.Error != nil {
		r.logger.Error("Failed to update member", map[string]interface{}{
			"error":     result.Error.Error(),
			"member_id": member.ID,
		})
		return fmt.Errorf("failed to update member: %w", result.Error)
	}

	r.logger.Info("Member updated successfully", map[string]interface{}{
		"member_id": member.ID,
	})

	return nil
}

// DeleteMember soft deletes a member record
func (r *memberRepository) DeleteMember(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.Member{}, id)
	if result.Error != nil {
		r.logger.Error("Failed to delete member", map[string]interface{}{
			"error":     result.Error.Error(),
			"member_id": id,
		})
		return fmt.Errorf("failed to delete member: %w", result.Error)
	}

	r.logger.Info("Member deleted successfully", map[string]interface{}{
		"member_id": id,
	})

	return nil
}

// CreateProfile creates a new member profile
func (r *memberRepository) CreateProfile(ctx context.Context, profile *models.MemberProfile) error {
	result := r.db.WithContext(ctx).Create(profile)
	if result.Error != nil {
		r.logger.Error("Failed to create member profile", map[string]interface{}{
			"error": result.Error.Error(),
		})
		return fmt.Errorf("failed to create profile: %w", result.Error)
	}

	return nil
}

// GetProfileByID retrieves a member profile by ID
func (r *memberRepository) GetProfileByID(ctx context.Context, id uint) (*models.MemberProfile, error) {
	var profile models.MemberProfile
	result := r.db.WithContext(ctx).
		Preload("Address").
		Preload("EmergencyContact").
		Preload("Preferences").
		First(&profile, id)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("profile not found with ID %d", id)
		}
		return nil, fmt.Errorf("failed to get profile: %w", result.Error)
	}

	return &profile, nil
}

// UpdateProfile updates an existing member profile
func (r *memberRepository) UpdateProfile(ctx context.Context, profile *models.MemberProfile) error {
	result := r.db.WithContext(ctx).Save(profile)
	if result.Error != nil {
		r.logger.Error("Failed to update member profile", map[string]interface{}{
			"error":      result.Error.Error(),
			"profile_id": profile.ID,
		})
		return fmt.Errorf("failed to update profile: %w", result.Error)
	}

	return nil
}

// CreateAddress creates a new address
func (r *memberRepository) CreateAddress(ctx context.Context, address *models.Address) error {
	result := r.db.WithContext(ctx).Create(address)
	if result.Error != nil {
		return fmt.Errorf("failed to create address: %w", result.Error)
	}
	return nil
}

// UpdateAddress updates an existing address
func (r *memberRepository) UpdateAddress(ctx context.Context, address *models.Address) error {
	result := r.db.WithContext(ctx).Save(address)
	if result.Error != nil {
		return fmt.Errorf("failed to update address: %w", result.Error)
	}
	return nil
}

// CreateEmergencyContact creates a new emergency contact
func (r *memberRepository) CreateEmergencyContact(ctx context.Context, contact *models.EmergencyContact) error {
	result := r.db.WithContext(ctx).Create(contact)
	if result.Error != nil {
		return fmt.Errorf("failed to create emergency contact: %w", result.Error)
	}
	return nil
}

// UpdateEmergencyContact updates an existing emergency contact
func (r *memberRepository) UpdateEmergencyContact(ctx context.Context, contact *models.EmergencyContact) error {
	result := r.db.WithContext(ctx).Save(contact)
	if result.Error != nil {
		return fmt.Errorf("failed to update emergency contact: %w", result.Error)
	}
	return nil
}

// CreatePreferences creates new member preferences
func (r *memberRepository) CreatePreferences(ctx context.Context, prefs *models.MemberPreferences) error {
	result := r.db.WithContext(ctx).Create(prefs)
	if result.Error != nil {
		return fmt.Errorf("failed to create preferences: %w", result.Error)
	}
	return nil
}

// UpdatePreferences updates existing member preferences
func (r *memberRepository) UpdatePreferences(ctx context.Context, prefs *models.MemberPreferences) error {
	result := r.db.WithContext(ctx).Save(prefs)
	if result.Error != nil {
		return fmt.Errorf("failed to update preferences: %w", result.Error)
	}
	return nil
}

// GetMemberCountByClub returns total member count for a club
func (r *memberRepository) GetMemberCountByClub(ctx context.Context, clubID uint) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).
		Model(&models.Member{}).
		Where("club_id = ?", clubID).
		Count(&count)

	if result.Error != nil {
		return 0, fmt.Errorf("failed to get member count: %w", result.Error)
	}

	return count, nil
}

// GetActiveMemberCountByClub returns active member count for a club
func (r *memberRepository) GetActiveMemberCountByClub(ctx context.Context, clubID uint) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).
		Model(&models.Member{}).
		Where("club_id = ? AND status = ?", clubID, models.MemberStatusActive).
		Count(&count)

	if result.Error != nil {
		return 0, fmt.Errorf("failed to get active member count: %w", result.Error)
	}

	return count, nil
}

// GetMembersByStatus retrieves members by status with pagination
func (r *memberRepository) GetMembersByStatus(ctx context.Context, status models.MemberStatus, limit, offset int) ([]*models.Member, error) {
	var members []*models.Member
	result := r.db.WithContext(ctx).
		Preload("Profile").
		Where("status = ?", status).
		Limit(limit).
		Offset(offset).
		Find(&members)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get members by status: %w", result.Error)
	}

	return members, nil
}

// GetMembersByMembershipType retrieves members by membership type with pagination
func (r *memberRepository) GetMembersByMembershipType(ctx context.Context, membershipType models.MembershipType, limit, offset int) ([]*models.Member, error) {
	var members []*models.Member
	result := r.db.WithContext(ctx).
		Preload("Profile").
		Where("membership_type = ?", membershipType).
		Limit(limit).
		Offset(offset).
		Find(&members)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get members by membership type: %w", result.Error)
	}

	return members, nil
}

// HealthCheck performs a database health check
func (r *memberRepository) HealthCheck(ctx context.Context) error {
	var result int
	err := r.db.WithContext(ctx).Raw("SELECT 1").Scan(&result).Error
	if err != nil {
		r.logger.Error("Database health check failed", map[string]interface{}{
			"error": err.Error(),
		})
		return fmt.Errorf("database health check failed: %w", err)
	}

	return nil
}