package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/services/member-service/internal/models"
	"reciprocal-clubs-backend/services/member-service/internal/service"
	"reciprocal-clubs-backend/services/member-service/proto/memberpb"
)

// Handler implements the gRPC MemberService interface
type Handler struct {
	memberpb.UnimplementedMemberServiceServer
	service service.Service
	logger  logging.Logger
}

// NewHandler creates a new gRPC handler
func NewHandler(service service.Service, logger logging.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// CreateMember creates a new member
func (h *Handler) CreateMember(ctx context.Context, req *memberpb.CreateMemberRequest) (*memberpb.CreateMemberResponse, error) {
	h.logger.Info("gRPC CreateMember called", map[string]interface{}{
		"user_id": req.GetUserId(),
		"club_id": req.GetClubId(),
	})

	// Convert proto request to service request
	serviceReq := &service.CreateMemberRequest{
		ClubID:         uint(req.GetClubId()),
		UserID:         uint(req.GetUserId()),
		MembershipType: protoToModelMembershipType(req.GetMembershipType()),
		Profile:        convertCreateProfileRequest(req.GetProfile()),
	}

	member, err := h.service.CreateMember(ctx, serviceReq)
	if err != nil {
		h.logger.Error("Failed to create member", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, status.Errorf(codes.Internal, "failed to create member: %v", err)
	}

	return &memberpb.CreateMemberResponse{
		Member: convertMemberToProto(member),
	}, nil
}

// GetMember retrieves a member by ID
func (h *Handler) GetMember(ctx context.Context, req *memberpb.GetMemberRequest) (*memberpb.GetMemberResponse, error) {
	member, err := h.service.GetMember(ctx, uint(req.GetMemberId()))
	if err != nil {
		h.logger.Error("Failed to get member", map[string]interface{}{
			"error":     err.Error(),
			"member_id": req.GetMemberId(),
		})
		return nil, status.Errorf(codes.NotFound, "member not found: %v", err)
	}

	return &memberpb.GetMemberResponse{
		Member: convertMemberToProto(member),
	}, nil
}

// GetMemberByUserID retrieves a member by user ID
func (h *Handler) GetMemberByUserID(ctx context.Context, req *memberpb.GetMemberByUserIDRequest) (*memberpb.GetMemberResponse, error) {
	member, err := h.service.GetMemberByUserID(ctx, uint(req.GetUserId()))
	if err != nil {
		h.logger.Error("Failed to get member by user ID", map[string]interface{}{
			"error":   err.Error(),
			"user_id": req.GetUserId(),
		})
		return nil, status.Errorf(codes.NotFound, "member not found: %v", err)
	}

	return &memberpb.GetMemberResponse{
		Member: convertMemberToProto(member),
	}, nil
}

// GetMemberByMemberNumber retrieves a member by member number
func (h *Handler) GetMemberByMemberNumber(ctx context.Context, req *memberpb.GetMemberByMemberNumberRequest) (*memberpb.GetMemberResponse, error) {
	member, err := h.service.GetMemberByMemberNumber(ctx, req.GetMemberNumber())
	if err != nil {
		h.logger.Error("Failed to get member by member number", map[string]interface{}{
			"error":         err.Error(),
			"member_number": req.GetMemberNumber(),
		})
		return nil, status.Errorf(codes.NotFound, "member not found: %v", err)
	}

	return &memberpb.GetMemberResponse{
		Member: convertMemberToProto(member),
	}, nil
}

// GetMembersByClub retrieves members for a specific club
func (h *Handler) GetMembersByClub(ctx context.Context, req *memberpb.GetMembersByClubRequest) (*memberpb.GetMembersByClubResponse, error) {
	members, err := h.service.GetMembersByClub(ctx, uint(req.GetClubId()), int(req.GetLimit()), int(req.GetOffset()))
	if err != nil {
		h.logger.Error("Failed to get members by club", map[string]interface{}{
			"error":   err.Error(),
			"club_id": req.GetClubId(),
		})
		return nil, status.Errorf(codes.Internal, "failed to get members: %v", err)
	}

	protoMembers := make([]*memberpb.Member, len(members))
	for i, member := range members {
		protoMembers[i] = convertMemberToProto(member)
	}

	return &memberpb.GetMembersByClubResponse{
		Members:    protoMembers,
		TotalCount: int32(len(members)), // TODO: Get actual total count
	}, nil
}

// UpdateMemberProfile updates a member's profile
func (h *Handler) UpdateMemberProfile(ctx context.Context, req *memberpb.UpdateMemberProfileRequest) (*memberpb.UpdateMemberProfileResponse, error) {
	// Convert proto update request to service request
	serviceReq := convertUpdateProfileRequest(req.GetProfile())

	member, err := h.service.UpdateMemberProfile(ctx, uint(req.GetMemberId()), serviceReq)
	if err != nil {
		h.logger.Error("Failed to update member profile", map[string]interface{}{
			"error":     err.Error(),
			"member_id": req.GetMemberId(),
		})
		return nil, status.Errorf(codes.Internal, "failed to update member profile: %v", err)
	}

	return &memberpb.UpdateMemberProfileResponse{
		Member: convertMemberToProto(member),
	}, nil
}

// SuspendMember suspends a member
func (h *Handler) SuspendMember(ctx context.Context, req *memberpb.SuspendMemberRequest) (*memberpb.SuspendMemberResponse, error) {
	member, err := h.service.SuspendMember(ctx, uint(req.GetMemberId()), req.GetReason())
	if err != nil {
		h.logger.Error("Failed to suspend member", map[string]interface{}{
			"error":     err.Error(),
			"member_id": req.GetMemberId(),
		})
		return nil, status.Errorf(codes.Internal, "failed to suspend member: %v", err)
	}

	return &memberpb.SuspendMemberResponse{
		Member: convertMemberToProto(member),
	}, nil
}

// ReactivateMember reactivates a member
func (h *Handler) ReactivateMember(ctx context.Context, req *memberpb.ReactivateMemberRequest) (*memberpb.ReactivateMemberResponse, error) {
	member, err := h.service.ReactivateMember(ctx, uint(req.GetMemberId()))
	if err != nil {
		h.logger.Error("Failed to reactivate member", map[string]interface{}{
			"error":     err.Error(),
			"member_id": req.GetMemberId(),
		})
		return nil, status.Errorf(codes.Internal, "failed to reactivate member: %v", err)
	}

	return &memberpb.ReactivateMemberResponse{
		Member: convertMemberToProto(member),
	}, nil
}

// DeleteMember deletes a member
func (h *Handler) DeleteMember(ctx context.Context, req *memberpb.DeleteMemberRequest) (*emptypb.Empty, error) {
	err := h.service.DeleteMember(ctx, uint(req.GetMemberId()))
	if err != nil {
		h.logger.Error("Failed to delete member", map[string]interface{}{
			"error":     err.Error(),
			"member_id": req.GetMemberId(),
		})
		return nil, status.Errorf(codes.Internal, "failed to delete member: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// ValidateMemberAccess validates if a member can access facilities
func (h *Handler) ValidateMemberAccess(ctx context.Context, req *memberpb.ValidateMemberAccessRequest) (*memberpb.ValidateMemberAccessResponse, error) {
	canAccess, err := h.service.ValidateMemberAccess(ctx, uint(req.GetMemberId()))
	if err != nil {
		h.logger.Error("Failed to validate member access", map[string]interface{}{
			"error":     err.Error(),
			"member_id": req.GetMemberId(),
		})
		return nil, status.Errorf(codes.Internal, "failed to validate member access: %v", err)
	}

	return &memberpb.ValidateMemberAccessResponse{
		CanAccess: canAccess,
	}, nil
}

// CheckMembershipStatus checks membership status
func (h *Handler) CheckMembershipStatus(ctx context.Context, req *memberpb.CheckMembershipStatusRequest) (*memberpb.CheckMembershipStatusResponse, error) {
	memberStatus, err := h.service.CheckMembershipStatus(ctx, uint(req.GetMemberId()))
	if err != nil {
		h.logger.Error("Failed to check membership status", map[string]interface{}{
			"error":     err.Error(),
			"member_id": req.GetMemberId(),
		})
		return nil, status.Errorf(codes.Internal, "failed to check membership status: %v", err)
	}

	return &memberpb.CheckMembershipStatusResponse{
		Status: convertMembershipStatusToProto(memberStatus),
	}, nil
}

// GetMemberAnalytics gets member analytics
func (h *Handler) GetMemberAnalytics(ctx context.Context, req *memberpb.GetMemberAnalyticsRequest) (*memberpb.GetMemberAnalyticsResponse, error) {
	analytics, err := h.service.GetMemberAnalytics(ctx, uint(req.GetClubId()))
	if err != nil {
		h.logger.Error("Failed to get member analytics", map[string]interface{}{
			"error":   err.Error(),
			"club_id": req.GetClubId(),
		})
		return nil, status.Errorf(codes.Internal, "failed to get member analytics: %v", err)
	}

	return &memberpb.GetMemberAnalyticsResponse{
		Analytics: convertMemberAnalyticsToProto(analytics),
	}, nil
}

// HealthCheck performs health check
func (h *Handler) HealthCheck(ctx context.Context, req *emptypb.Empty) (*memberpb.HealthCheckResponse, error) {
	err := h.service.HealthCheck(ctx)
	if err != nil {
		return &memberpb.HealthCheckResponse{
			Status: "unhealthy",
		}, nil
	}

	return &memberpb.HealthCheckResponse{
		Status: "healthy",
	}, nil
}

// Conversion functions

func convertMemberToProto(member *models.Member) *memberpb.Member {
	proto := &memberpb.Member{
		Id:                uint32(member.ID),
		ClubId:            uint32(member.ClubID),
		UserId:            uint32(member.UserID),
		MemberNumber:      member.MemberNumber,
		MembershipType:    modelToProtoMembershipType(member.MembershipType),
		Status:            modelToProtoMemberStatus(member.Status),
		BlockchainIdentity: member.BlockchainIdentity,
		JoinedAt:          timestamppb.New(member.JoinedAt),
		CreatedAt:         timestamppb.New(member.CreatedAt),
		UpdatedAt:         timestamppb.New(member.UpdatedAt),
	}

	if member.Profile != nil {
		proto.Profile = convertMemberProfileToProto(member.Profile)
	}

	return proto
}

func convertMemberProfileToProto(profile *models.MemberProfile) *memberpb.MemberProfile {
	proto := &memberpb.MemberProfile{
		Id:          uint32(profile.ID),
		FirstName:   profile.FirstName,
		LastName:    profile.LastName,
		PhoneNumber: profile.PhoneNumber,
		CreatedAt:   timestamppb.New(profile.CreatedAt),
		UpdatedAt:   timestamppb.New(profile.UpdatedAt),
	}

	if profile.DateOfBirth != nil {
		proto.DateOfBirth = timestamppb.New(*profile.DateOfBirth)
	}

	if profile.Address != nil {
		proto.Address = &memberpb.Address{
			Id:         uint32(profile.Address.ID),
			Street:     profile.Address.Street,
			City:       profile.Address.City,
			State:      profile.Address.State,
			PostalCode: profile.Address.PostalCode,
			Country:    profile.Address.Country,
		}
	}

	if profile.EmergencyContact != nil {
		proto.EmergencyContact = &memberpb.EmergencyContact{
			Id:           uint32(profile.EmergencyContact.ID),
			Name:         profile.EmergencyContact.Name,
			Relationship: profile.EmergencyContact.Relationship,
			PhoneNumber:  profile.EmergencyContact.PhoneNumber,
			Email:        profile.EmergencyContact.Email,
		}
	}

	if profile.Preferences != nil {
		proto.Preferences = &memberpb.MemberPreferences{
			Id:                 uint32(profile.Preferences.ID),
			EmailNotifications: profile.Preferences.EmailNotifications,
			SmsNotifications:   profile.Preferences.SMSNotifications,
			PushNotifications:  profile.Preferences.PushNotifications,
			MarketingEmails:    profile.Preferences.MarketingEmails,
		}
	}

	return proto
}

func convertCreateProfileRequest(req *memberpb.CreateMemberProfileRequest) service.CreateProfileRequest {
	profile := service.CreateProfileRequest{
		FirstName:   req.GetFirstName(),
		LastName:    req.GetLastName(),
		PhoneNumber: req.GetPhoneNumber(),
	}

	if req.GetDateOfBirth() != nil {
		dateStr := req.GetDateOfBirth().AsTime().Format("2006-01-02")
		profile.DateOfBirth = &dateStr
	}

	if req.GetAddress() != nil {
		profile.Address = &service.CreateAddressRequest{
			Street:     req.GetAddress().GetStreet(),
			City:       req.GetAddress().GetCity(),
			State:      req.GetAddress().GetState(),
			PostalCode: req.GetAddress().GetPostalCode(),
			Country:    req.GetAddress().GetCountry(),
		}
	}

	if req.GetEmergencyContact() != nil {
		profile.EmergencyContact = &service.CreateEmergencyContactRequest{
			Name:         req.GetEmergencyContact().GetName(),
			Relationship: req.GetEmergencyContact().GetRelationship(),
			PhoneNumber:  req.GetEmergencyContact().GetPhoneNumber(),
			Email:        req.GetEmergencyContact().GetEmail(),
		}
	}

	if req.GetPreferences() != nil {
		profile.Preferences = &service.CreatePreferencesRequest{
			EmailNotifications: req.GetPreferences().GetEmailNotifications(),
			SMSNotifications:   req.GetPreferences().GetSmsNotifications(),
			PushNotifications:  req.GetPreferences().GetPushNotifications(),
			MarketingEmails:    req.GetPreferences().GetMarketingEmails(),
		}
	}

	return profile
}

func convertUpdateProfileRequest(req *memberpb.UpdateProfileRequest) *service.UpdateProfileRequest {
	update := &service.UpdateProfileRequest{}

	if req.FirstName != nil {
		update.FirstName = req.FirstName
	}
	if req.LastName != nil {
		update.LastName = req.LastName
	}
	if req.PhoneNumber != nil {
		update.PhoneNumber = req.PhoneNumber
	}

	// TODO: Handle other fields

	return update
}

func convertMembershipStatusToProto(status *service.MembershipStatus) *memberpb.MembershipStatusInfo {
	proto := &memberpb.MembershipStatusInfo{
		MemberId:       uint32(status.MemberID),
		Status:         modelToProtoMemberStatus(status.Status),
		MembershipType: modelToProtoMembershipType(status.MembershipType),
		CanAccess:      status.CanAccess,
	}

	// Parse joined_at timestamp
	if parsedTime, err := time.Parse("2006-01-02T15:04:05Z", status.JoinedAt); err == nil {
		proto.JoinedAt = timestamppb.New(parsedTime)
	}

	return proto
}

func convertMemberAnalyticsToProto(analytics *service.MemberAnalytics) *memberpb.MemberAnalytics {
	proto := &memberpb.MemberAnalytics{
		TotalMembers:        analytics.TotalMembers,
		ActiveMembers:       analytics.ActiveMembers,
		NewMembersThisMonth: analytics.NewMembersThisMonth,
	}

	for _, dist := range analytics.MembershipDistribution {
		proto.MembershipDistribution = append(proto.MembershipDistribution, &memberpb.MembershipTypeCount{
			Type:  modelToProtoMembershipType(dist.Type),
			Count: dist.Count,
		})
	}

	for _, dist := range analytics.StatusDistribution {
		proto.StatusDistribution = append(proto.StatusDistribution, &memberpb.MemberStatusCount{
			Status: modelToProtoMemberStatus(dist.Status),
			Count:  dist.Count,
		})
	}

	return proto
}

// Enum conversion functions
func protoToModelMembershipType(pt memberpb.MembershipType) models.MembershipType {
	switch pt {
	case memberpb.MembershipType_MEMBERSHIP_TYPE_REGULAR:
		return models.MembershipTypeRegular
	case memberpb.MembershipType_MEMBERSHIP_TYPE_VIP:
		return models.MembershipTypeVIP
	case memberpb.MembershipType_MEMBERSHIP_TYPE_CORPORATE:
		return models.MembershipTypeCorporate
	case memberpb.MembershipType_MEMBERSHIP_TYPE_STUDENT:
		return models.MembershipTypeStudent
	case memberpb.MembershipType_MEMBERSHIP_TYPE_SENIOR:
		return models.MembershipTypeSenior
	default:
		return models.MembershipTypeRegular
	}
}

func modelToProtoMembershipType(mt models.MembershipType) memberpb.MembershipType {
	switch mt {
	case models.MembershipTypeRegular:
		return memberpb.MembershipType_MEMBERSHIP_TYPE_REGULAR
	case models.MembershipTypeVIP:
		return memberpb.MembershipType_MEMBERSHIP_TYPE_VIP
	case models.MembershipTypeCorporate:
		return memberpb.MembershipType_MEMBERSHIP_TYPE_CORPORATE
	case models.MembershipTypeStudent:
		return memberpb.MembershipType_MEMBERSHIP_TYPE_STUDENT
	case models.MembershipTypeSenior:
		return memberpb.MembershipType_MEMBERSHIP_TYPE_SENIOR
	default:
		return memberpb.MembershipType_MEMBERSHIP_TYPE_REGULAR
	}
}

func modelToProtoMemberStatus(ms models.MemberStatus) memberpb.MemberStatus {
	switch ms {
	case models.MemberStatusActive:
		return memberpb.MemberStatus_MEMBER_STATUS_ACTIVE
	case models.MemberStatusSuspended:
		return memberpb.MemberStatus_MEMBER_STATUS_SUSPENDED
	case models.MemberStatusExpired:
		return memberpb.MemberStatus_MEMBER_STATUS_EXPIRED
	case models.MemberStatusPending:
		return memberpb.MemberStatus_MEMBER_STATUS_PENDING
	default:
		return memberpb.MemberStatus_MEMBER_STATUS_ACTIVE
	}
}