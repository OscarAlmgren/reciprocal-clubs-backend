package service

import (
	"context"
	"fmt"
	"testing"

	"reciprocal-clubs-backend/pkg/shared/config"
	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/services/member-service/internal/models"
)

// Mock repository for testing
type mockRepository struct {
	members map[uint]*models.Member
	nextID  uint
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		members: make(map[uint]*models.Member),
		nextID:  1,
	}
}

func (m *mockRepository) CreateMember(ctx context.Context, member *models.Member) error {
	member.ID = m.nextID
	member.MemberNumber = fmt.Sprintf("M%d%d", member.ClubID, m.nextID)
	m.members[member.ID] = member
	m.nextID++
	return nil
}

func (m *mockRepository) GetMemberByID(ctx context.Context, id uint) (*models.Member, error) {
	member, exists := m.members[id]
	if !exists {
		return nil, fmt.Errorf("member not found")
	}
	return member, nil
}

func (m *mockRepository) GetMemberByUserID(ctx context.Context, userID uint) (*models.Member, error) {
	for _, member := range m.members {
		if member.UserID == userID {
			return member, nil
		}
	}
	return nil, fmt.Errorf("member not found")
}

func (m *mockRepository) GetMemberByMemberNumber(ctx context.Context, memberNumber string) (*models.Member, error) {
	for _, member := range m.members {
		if member.MemberNumber == memberNumber {
			return member, nil
		}
	}
	return nil, fmt.Errorf("member not found")
}

func (m *mockRepository) GetMembersByClubID(ctx context.Context, clubID uint, limit, offset int) ([]*models.Member, error) {
	var result []*models.Member
	for _, member := range m.members {
		if member.ClubID == clubID {
			result = append(result, member)
		}
	}
	return result, nil
}

func (m *mockRepository) UpdateMember(ctx context.Context, member *models.Member) error {
	m.members[member.ID] = member
	return nil
}

func (m *mockRepository) DeleteMember(ctx context.Context, id uint) error {
	delete(m.members, id)
	return nil
}

func (m *mockRepository) CreateProfile(ctx context.Context, profile *models.MemberProfile) error {
	profile.ID = m.nextID
	m.nextID++
	return nil
}

func (m *mockRepository) GetProfileByID(ctx context.Context, id uint) (*models.MemberProfile, error) {
	return &models.MemberProfile{ID: id}, nil
}

func (m *mockRepository) UpdateProfile(ctx context.Context, profile *models.MemberProfile) error {
	return nil
}

func (m *mockRepository) CreateAddress(ctx context.Context, address *models.Address) error {
	address.ID = m.nextID
	m.nextID++
	return nil
}

func (m *mockRepository) UpdateAddress(ctx context.Context, address *models.Address) error {
	return nil
}

func (m *mockRepository) CreateEmergencyContact(ctx context.Context, contact *models.EmergencyContact) error {
	contact.ID = m.nextID
	m.nextID++
	return nil
}

func (m *mockRepository) UpdateEmergencyContact(ctx context.Context, contact *models.EmergencyContact) error {
	return nil
}

func (m *mockRepository) CreatePreferences(ctx context.Context, prefs *models.MemberPreferences) error {
	prefs.ID = m.nextID
	m.nextID++
	return nil
}

func (m *mockRepository) UpdatePreferences(ctx context.Context, prefs *models.MemberPreferences) error {
	return nil
}

func (m *mockRepository) GetMemberCountByClub(ctx context.Context, clubID uint) (int64, error) {
	count := int64(0)
	for _, member := range m.members {
		if member.ClubID == clubID {
			count++
		}
	}
	return count, nil
}

func (m *mockRepository) GetActiveMemberCountByClub(ctx context.Context, clubID uint) (int64, error) {
	count := int64(0)
	for _, member := range m.members {
		if member.ClubID == clubID && member.Status == models.MemberStatusActive {
			count++
		}
	}
	return count, nil
}

func (m *mockRepository) GetMembersByStatus(ctx context.Context, status models.MemberStatus, limit, offset int) ([]*models.Member, error) {
	var result []*models.Member
	for _, member := range m.members {
		if member.Status == status {
			result = append(result, member)
		}
	}
	return result, nil
}

func (m *mockRepository) GetMembersByMembershipType(ctx context.Context, membershipType models.MembershipType, limit, offset int) ([]*models.Member, error) {
	var result []*models.Member
	for _, member := range m.members {
		if member.MembershipType == membershipType {
			result = append(result, member)
		}
	}
	return result, nil
}

func (m *mockRepository) HealthCheck(ctx context.Context) error {
	return nil
}

func TestMemberService_CreateMember(t *testing.T) {
	repo := newMockRepository()
	logger := logging.NewLogger(&config.LoggingConfig{Level: "debug"}, "test")
	service := NewService(repo, logger, nil)

	ctx := context.Background()

	req := &CreateMemberRequest{
		ClubID:         1,
		UserID:         1,
		MembershipType: models.MembershipTypeRegular,
		Profile: CreateProfileRequest{
			FirstName: "John",
			LastName:  "Doe",
		},
	}

	member, err := service.CreateMember(ctx, req)
	if err != nil {
		t.Errorf("CreateMember failed: %v", err)
	}

	if member.ID == 0 {
		t.Error("Member ID should be set")
	}

	if member.MemberNumber == "" {
		t.Error("Member number should be generated")
	}

	if member.Status != models.MemberStatusActive {
		t.Errorf("Expected status %s, got %s", models.MemberStatusActive, member.Status)
	}
}

func TestMemberService_GetMember(t *testing.T) {
	repo := newMockRepository()
	logger := logging.NewLogger(&config.LoggingConfig{Level: "debug"}, "test")
	service := NewService(repo, logger, nil)

	ctx := context.Background()

	// Create test member
	req := &CreateMemberRequest{
		ClubID:         1,
		UserID:         1,
		MembershipType: models.MembershipTypeRegular,
		Profile: CreateProfileRequest{
			FirstName: "John",
			LastName:  "Doe",
		},
	}

	createdMember, err := service.CreateMember(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create test member: %v", err)
	}

	// Get member
	retrievedMember, err := service.GetMember(ctx, createdMember.ID)
	if err != nil {
		t.Errorf("GetMember failed: %v", err)
	}

	if retrievedMember.ID != createdMember.ID {
		t.Errorf("Expected member ID %d, got %d", createdMember.ID, retrievedMember.ID)
	}
}

func TestMemberService_SuspendMember(t *testing.T) {
	repo := newMockRepository()
	logger := logging.NewLogger(&config.LoggingConfig{Level: "debug"}, "test")
	service := NewService(repo, logger, nil)

	ctx := context.Background()

	// Create test member
	req := &CreateMemberRequest{
		ClubID:         1,
		UserID:         1,
		MembershipType: models.MembershipTypeRegular,
		Profile: CreateProfileRequest{
			FirstName: "John",
			LastName:  "Doe",
		},
	}

	member, err := service.CreateMember(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create test member: %v", err)
	}

	// Suspend member
	suspendedMember, err := service.SuspendMember(ctx, member.ID, "Test suspension")
	if err != nil {
		t.Errorf("SuspendMember failed: %v", err)
	}

	if suspendedMember.Status != models.MemberStatusSuspended {
		t.Errorf("Expected status %s, got %s", models.MemberStatusSuspended, suspendedMember.Status)
	}
}

func TestMemberService_ValidateMemberAccess(t *testing.T) {
	repo := newMockRepository()
	logger := logging.NewLogger(&config.LoggingConfig{Level: "debug"}, "test")
	service := NewService(repo, logger, nil)

	ctx := context.Background()

	// Create test member
	req := &CreateMemberRequest{
		ClubID:         1,
		UserID:         1,
		MembershipType: models.MembershipTypeRegular,
		Profile: CreateProfileRequest{
			FirstName: "John",
			LastName:  "Doe",
		},
	}

	member, err := service.CreateMember(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create test member: %v", err)
	}

	// Validate access for active member
	canAccess, err := service.ValidateMemberAccess(ctx, member.ID)
	if err != nil {
		t.Errorf("ValidateMemberAccess failed: %v", err)
	}

	if !canAccess {
		t.Error("Active member should have access")
	}

	// Suspend member and test again
	_, err = service.SuspendMember(ctx, member.ID, "Test")
	if err != nil {
		t.Fatalf("Failed to suspend member: %v", err)
	}

	canAccess, err = service.ValidateMemberAccess(ctx, member.ID)
	if err != nil {
		t.Errorf("ValidateMemberAccess failed: %v", err)
	}

	if canAccess {
		t.Error("Suspended member should not have access")
	}
}