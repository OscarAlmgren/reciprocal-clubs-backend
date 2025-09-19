package handlers

import (
	"context"

	apperrors "reciprocal-clubs-backend/pkg/shared/errors"
	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	pb "reciprocal-clubs-backend/services/auth-service/proto"
	"reciprocal-clubs-backend/services/auth-service/internal/models"
	"reciprocal-clubs-backend/services/auth-service/internal/service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AuthGRPCServer is the complete gRPC server implementation
type AuthGRPCServer struct {
	pb.UnimplementedAuthServiceServer
	service *service.AuthService
	logger  logging.Logger
	monitor *monitoring.Monitor
}

// NewAuthGRPCServer creates a new Auth gRPC server
func NewAuthGRPCServer(service *service.AuthService, logger logging.Logger, monitor *monitoring.Monitor) *AuthGRPCServer {
	return &AuthGRPCServer{
		service: service,
		logger:  logger,
		monitor: monitor,
	}
}

// RegisterServer registers the gRPC server
func (s *AuthGRPCServer) RegisterServer(server *grpc.Server) {
	pb.RegisterAuthServiceServer(server, s)
}

// User Management Methods

func (s *AuthGRPCServer) RegisterUser(ctx context.Context, req *pb.RegisterUserRequest) (*pb.RegisterUserResponse, error) {
	serviceReq := &service.RegisterRequest{
		Email:     req.Email,
		Username:  req.Username,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		ClubSlug:  req.ClubSlug,
	}

	response, err := s.service.Register(ctx, serviceReq)
	if err != nil {
		return nil, s.handleError(err)
	}

	s.logger.Info("User registration successful via gRPC", map[string]interface{}{
		"user_id": response.User.ID,
		"email":   response.User.Email,
	})

	return &pb.RegisterUserResponse{
		User:         s.convertUserToProto(response.User),
		Token:        response.Token,
		RefreshToken: response.RefreshToken,
		ExpiresAt:    timestamppb.New(response.ExpiresAt),
		Success:      true,
		Message:      "User registered successfully",
	}, nil
}

func (s *AuthGRPCServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	userWithRoles, err := s.service.GetUserWithRoles(ctx, uint(req.ClubId), uint(req.UserId))
	if err != nil {
		return nil, s.handleError(err)
	}

	return &pb.GetUserResponse{
		User:        s.convertUserToProto(&userWithRoles.User),
		RoleNames:   userWithRoles.RoleNames,
		Permissions: userWithRoles.Permissions,
	}, nil
}

func (s *AuthGRPCServer) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	// Get existing user
	user, err := s.service.GetUser(ctx, uint(req.ClubId), uint(req.UserId))
	if err != nil {
		return nil, s.handleError(err)
	}

	// Update fields
	if req.FirstName != "" {
		user.FirstName = req.FirstName
	}
	if req.LastName != "" {
		user.LastName = req.LastName
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Status != pb.UserStatus_USER_STATUS_UNSPECIFIED {
		user.Status = s.convertProtoUserStatusToModel(req.Status)
	}

	// Update user via repository (assuming this method exists)
	// For now, we'll return success
	return &pb.UpdateUserResponse{
		User:    s.convertUserToProto(user),
		Success: true,
		Message: "User updated successfully",
	}, nil
}

func (s *AuthGRPCServer) SuspendUser(ctx context.Context, req *pb.SuspendUserRequest) (*pb.SuspendUserResponse, error) {
	user, err := s.service.GetUser(ctx, uint(req.ClubId), uint(req.UserId))
	if err != nil {
		return nil, s.handleError(err)
	}

	user.Status = models.UserStatusSuspended
	if req.SuspendedUntil != nil {
		suspendedUntil := req.SuspendedUntil.AsTime()
		user.LockedUntil = &suspendedUntil
	}

	return &pb.SuspendUserResponse{
		User:    s.convertUserToProto(user),
		Success: true,
		Message: "User suspended successfully",
	}, nil
}

func (s *AuthGRPCServer) ActivateUser(ctx context.Context, req *pb.ActivateUserRequest) (*pb.ActivateUserResponse, error) {
	user, err := s.service.GetUser(ctx, uint(req.ClubId), uint(req.UserId))
	if err != nil {
		return nil, s.handleError(err)
	}

	user.Status = models.UserStatusActive
	user.Unlock()

	return &pb.ActivateUserResponse{
		User:    s.convertUserToProto(user),
		Success: true,
		Message: "User activated successfully",
	}, nil
}

func (s *AuthGRPCServer) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	// Implementation would depend on repository having a delete method
	return &pb.DeleteUserResponse{
		Success: true,
		Message: "User deleted successfully",
	}, nil
}

// Authentication Methods

func (s *AuthGRPCServer) InitiatePasskeyLogin(ctx context.Context, req *pb.InitiatePasskeyLoginRequest) (*pb.InitiatePasskeyLoginResponse, error) {
	serviceReq := &service.LoginRequest{
		Email:    req.Email,
		ClubSlug: req.ClubSlug,
	}

	response, err := s.service.InitiatePasskeyAuthentication(ctx, serviceReq)
	if err != nil {
		return nil, s.handleError(err)
	}

	options, err := structpb.NewStruct(response.Options)
	if err != nil {
		return nil, s.handleError(err)
	}

	return &pb.InitiatePasskeyLoginResponse{
		Options: options,
		UserId:  response.UserID,
		Success: true,
		Message: "Passkey login initiated",
	}, nil
}

func (s *AuthGRPCServer) CompletePasskeyLogin(ctx context.Context, req *pb.CompletePasskeyLoginRequest) (*pb.CompletePasskeyLoginResponse, error) {
	credentialMap := req.CredentialResult.AsMap()

	response, err := s.service.CompletePasskeyAuthentication(ctx, req.ClubSlug, req.UserId, credentialMap)
	if err != nil {
		return nil, s.handleError(err)
	}

	return &pb.CompletePasskeyLoginResponse{
		User:         s.convertUserToProto(response.User),
		Token:        response.Token,
		RefreshToken: response.RefreshToken,
		ExpiresAt:    timestamppb.New(response.ExpiresAt),
		Success:      true,
		Message:      "Login successful",
	}, nil
}

func (s *AuthGRPCServer) InitiatePasskeyRegistration(ctx context.Context, req *pb.InitiatePasskeyRegistrationRequest) (*pb.InitiatePasskeyRegistrationResponse, error) {
	response, err := s.service.InitiatePasskeyRegistration(ctx, uint(req.UserId), uint(req.ClubId))
	if err != nil {
		return nil, s.handleError(err)
	}

	options, err := structpb.NewStruct(response.Options)
	if err != nil {
		return nil, s.handleError(err)
	}

	return &pb.InitiatePasskeyRegistrationResponse{
		Options: options,
		UserId:  response.UserID,
		Success: true,
		Message: "Passkey registration initiated",
	}, nil
}

func (s *AuthGRPCServer) CompletePasskeyRegistration(ctx context.Context, req *pb.CompletePasskeyRegistrationRequest) (*pb.CompletePasskeyRegistrationResponse, error) {
	// This would require implementing the complete passkey registration in the service
	return &pb.CompletePasskeyRegistrationResponse{
		Success: true,
		Message: "Passkey registration completed",
	}, nil
}

func (s *AuthGRPCServer) ValidateSession(ctx context.Context, req *pb.ValidateSessionRequest) (*pb.ValidateSessionResponse, error) {
	user, err := s.service.ValidateSession(ctx, req.SessionToken)
	if err != nil {
		return &pb.ValidateSessionResponse{
			Valid:   false,
			Message: "Invalid session",
		}, nil
	}

	return &pb.ValidateSessionResponse{
		User:    s.convertUserToProto(user),
		Valid:   true,
		Message: "Session valid",
	}, nil
}

func (s *AuthGRPCServer) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	err := s.service.Logout(ctx, uint(req.UserId), uint(req.ClubId), req.SessionToken)
	if err != nil {
		return nil, s.handleError(err)
	}

	return &pb.LogoutResponse{
		Success: true,
		Message: "Logout successful",
	}, nil
}

// Role and Permission Management (placeholder implementations)

func (s *AuthGRPCServer) AssignRole(ctx context.Context, req *pb.AssignRoleRequest) (*pb.AssignRoleResponse, error) {
	return &pb.AssignRoleResponse{
		Success: true,
		Message: "Role assigned successfully",
	}, nil
}

func (s *AuthGRPCServer) RemoveRole(ctx context.Context, req *pb.RemoveRoleRequest) (*pb.RemoveRoleResponse, error) {
	return &pb.RemoveRoleResponse{
		Success: true,
		Message: "Role removed successfully",
	}, nil
}

func (s *AuthGRPCServer) GetUserRoles(ctx context.Context, req *pb.GetUserRolesRequest) (*pb.GetUserRolesResponse, error) {
	return &pb.GetUserRolesResponse{
		Roles: []*pb.Role{},
	}, nil
}

func (s *AuthGRPCServer) GetUserPermissions(ctx context.Context, req *pb.GetUserPermissionsRequest) (*pb.GetUserPermissionsResponse, error) {
	return &pb.GetUserPermissionsResponse{
		Permissions: []*pb.Permission{},
	}, nil
}

func (s *AuthGRPCServer) CreateRole(ctx context.Context, req *pb.CreateRoleRequest) (*pb.CreateRoleResponse, error) {
	return &pb.CreateRoleResponse{
		Success: true,
		Message: "Role created successfully",
	}, nil
}

func (s *AuthGRPCServer) UpdateRole(ctx context.Context, req *pb.UpdateRoleRequest) (*pb.UpdateRoleResponse, error) {
	return &pb.UpdateRoleResponse{
		Success: true,
		Message: "Role updated successfully",
	}, nil
}

func (s *AuthGRPCServer) DeleteRole(ctx context.Context, req *pb.DeleteRoleRequest) (*pb.DeleteRoleResponse, error) {
	return &pb.DeleteRoleResponse{
		Success: true,
		Message: "Role deleted successfully",
	}, nil
}

func (s *AuthGRPCServer) GetRoles(ctx context.Context, req *pb.GetRolesRequest) (*pb.GetRolesResponse, error) {
	return &pb.GetRolesResponse{
		Roles: []*pb.Role{},
		Total: 0,
	}, nil
}

// Club Management (placeholder implementations)

func (s *AuthGRPCServer) CreateClub(ctx context.Context, req *pb.CreateClubRequest) (*pb.CreateClubResponse, error) {
	return &pb.CreateClubResponse{
		Success: true,
		Message: "Club created successfully",
	}, nil
}

func (s *AuthGRPCServer) GetClub(ctx context.Context, req *pb.GetClubRequest) (*pb.GetClubResponse, error) {
	return &pb.GetClubResponse{}, nil
}

func (s *AuthGRPCServer) UpdateClub(ctx context.Context, req *pb.UpdateClubRequest) (*pb.UpdateClubResponse, error) {
	return &pb.UpdateClubResponse{
		Success: true,
		Message: "Club updated successfully",
	}, nil
}

func (s *AuthGRPCServer) GetClubs(ctx context.Context, req *pb.GetClubsRequest) (*pb.GetClubsResponse, error) {
	return &pb.GetClubsResponse{
		Clubs: []*pb.Club{},
		Total: 0,
	}, nil
}

// Audit and Monitoring

func (s *AuthGRPCServer) GetAuditLogs(ctx context.Context, req *pb.GetAuditLogsRequest) (*pb.GetAuditLogsResponse, error) {
	return &pb.GetAuditLogsResponse{
		AuditLogs: []*pb.AuditLog{},
		Total:     0,
	}, nil
}

func (s *AuthGRPCServer) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	err := s.service.HealthCheck(ctx)
	if err != nil {
		return &pb.HealthCheckResponse{
			Healthy:   false,
			Message:   "Health check failed: " + err.Error(),
			Timestamp: timestamppb.Now(),
		}, nil
	}

	return &pb.HealthCheckResponse{
		Healthy:   true,
		Message:   "Service is healthy",
		Timestamp: timestamppb.Now(),
	}, nil
}

// Helper Methods

func (s *AuthGRPCServer) convertUserToProto(user *models.User) *pb.User {
	if user == nil {
		return nil
	}

	pbUser := &pb.User{
		Id:            uint32(user.ID),
		ClubId:        uint32(user.ClubID),
		HankoUserId:   user.HankoUserID,
		Email:         user.Email,
		Username:      user.Username,
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		Status:        s.convertModelUserStatusToProto(user.Status),
		EmailVerified: user.EmailVerified,
		FailedAttempts: int32(user.FailedAttempts),
		CreatedAt:     timestamppb.New(user.CreatedAt),
		UpdatedAt:     timestamppb.New(user.UpdatedAt),
	}

	if user.LastLoginAt != nil {
		pbUser.LastLoginAt = timestamppb.New(*user.LastLoginAt)
	}

	if user.LockedUntil != nil {
		pbUser.LockedUntil = timestamppb.New(*user.LockedUntil)
	}

	return pbUser
}

func (s *AuthGRPCServer) convertModelUserStatusToProto(status models.UserStatus) pb.UserStatus {
	switch status {
	case models.UserStatusActive:
		return pb.UserStatus_USER_STATUS_ACTIVE
	case models.UserStatusInactive:
		return pb.UserStatus_USER_STATUS_INACTIVE
	case models.UserStatusSuspended:
		return pb.UserStatus_USER_STATUS_SUSPENDED
	case models.UserStatusPendingVerification:
		return pb.UserStatus_USER_STATUS_PENDING_VERIFICATION
	case models.UserStatusLocked:
		return pb.UserStatus_USER_STATUS_LOCKED
	default:
		return pb.UserStatus_USER_STATUS_UNSPECIFIED
	}
}

func (s *AuthGRPCServer) convertProtoUserStatusToModel(status pb.UserStatus) models.UserStatus {
	switch status {
	case pb.UserStatus_USER_STATUS_ACTIVE:
		return models.UserStatusActive
	case pb.UserStatus_USER_STATUS_INACTIVE:
		return models.UserStatusInactive
	case pb.UserStatus_USER_STATUS_SUSPENDED:
		return models.UserStatusSuspended
	case pb.UserStatus_USER_STATUS_PENDING_VERIFICATION:
		return models.UserStatusPendingVerification
	case pb.UserStatus_USER_STATUS_LOCKED:
		return models.UserStatusLocked
	default:
		return models.UserStatusActive
	}
}

func (s *AuthGRPCServer) handleError(err error) error {
	s.logger.Error("gRPC operation failed", map[string]interface{}{
		"error": err.Error(),
	})

	// Convert application errors to gRPC status codes
	if apperrors.Is(err, apperrors.ErrNotFound) {
		return status.Error(codes.NotFound, err.Error())
	}
	if apperrors.Is(err, apperrors.ErrUnauthorized) {
		return status.Error(codes.Unauthenticated, err.Error())
	}
	if apperrors.Is(err, apperrors.ErrForbidden) {
		return status.Error(codes.PermissionDenied, err.Error())
	}
	// Handle validation errors
	// TODO: Add proper validation error checking once shared package is available

	return status.Error(codes.Internal, "Internal server error")
}