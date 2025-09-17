package handlers

import (
	"context"
	"errors"

	apperrors "reciprocal-clubs-backend/pkg/shared/errors"
	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	"reciprocal-clubs-backend/services/auth-service/internal/service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuthServiceServer is the gRPC server implementation
type AuthServiceServer struct {
	service *service.AuthService
	logger  logging.Logger
	monitor *monitoring.Monitor
}

// NewGRPCHandler creates a new gRPC handler
func NewGRPCHandler(service *service.AuthService, logger logging.Logger, monitor *monitoring.Monitor) *AuthServiceServer {
	return &AuthServiceServer{
		service: service,
		logger:  logger,
		monitor: monitor,
	}
}

// Register the server with the gRPC server
func (s *AuthServiceServer) RegisterServer(server *grpc.Server) {
	// This would register with the generated protobuf service registration
	// e.g., pb.RegisterAuthServiceServer(server, s)
	// For now, we'll use a placeholder since we haven't generated the protobuf files
}

// Authentication gRPC methods

// RegisterUser creates a new user account with passkey setup
func (s *AuthServiceServer) RegisterUser(ctx context.Context, req *RegisterUserRequest) (*RegisterUserResponse, error) {
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

	return &RegisterUserResponse{
		User:    s.convertUserToProto(response.User),
		Success: true,
	}, nil
}

// InitiatePasskeyLogin starts the passkey authentication process
func (s *AuthServiceServer) InitiatePasskeyLogin(ctx context.Context, req *InitiatePasskeyLoginRequest) (*InitiatePasskeyLoginResponse, error) {
	serviceReq := &service.LoginRequest{
		Email:    req.Email,
		ClubSlug: req.ClubSlug,
	}

	response, err := s.service.InitiatePasskeyAuthentication(ctx, serviceReq)
	if err != nil {
		return nil, s.handleError(err)
	}

	s.logger.Debug("Passkey login initiated via gRPC", map[string]interface{}{
		"email": req.Email,
	})

	return &InitiatePasskeyLoginResponse{
		Options: response.Options,
		UserID:  response.UserID,
	}, nil
}

// CompletePasskeyLogin completes the passkey authentication process
func (s *AuthServiceServer) CompletePasskeyLogin(ctx context.Context, req *CompletePasskeyLoginRequest) (*CompletePasskeyLoginResponse, error) {
	response, err := s.service.CompletePasskeyAuthentication(ctx, req.ClubSlug, req.UserID, req.CredentialResult)
	if err != nil {
		return nil, s.handleError(err)
	}

	s.logger.Info("Passkey login completed via gRPC", map[string]interface{}{
		"user_id": response.User.ID,
		"email":   response.User.Email,
	})

	return &CompletePasskeyLoginResponse{
		User:         s.convertUserToProto(response.User),
		Token:        response.Token,
		RefreshToken: response.RefreshToken,
		ExpiresAt:    &response.ExpiresAt,
	}, nil
}

// Logout invalidates a user session
func (s *AuthServiceServer) Logout(ctx context.Context, req *LogoutRequest) (*LogoutResponse, error) {
	err := s.service.Logout(ctx, uint(req.UserId), uint(req.ClubId), req.SessionToken)
	if err != nil {
		return nil, s.handleError(err)
	}

	s.logger.Info("User logged out via gRPC", map[string]interface{}{
		"user_id": req.UserId,
		"club_id": req.ClubId,
	})

	return &LogoutResponse{
		Success: true,
	}, nil
}

// ValidateSession validates a session token
func (s *AuthServiceServer) ValidateSession(ctx context.Context, req *ValidateSessionRequest) (*ValidateSessionResponse, error) {
	user, err := s.service.ValidateSession(ctx, req.SessionToken)
	if err != nil {
		return nil, s.handleError(err)
	}

	return &ValidateSessionResponse{
		Valid: true,
		User:  s.convertUserToProto(user),
	}, nil
}

// InitiatePasskeyRegistration starts the passkey registration process for an existing user
func (s *AuthServiceServer) InitiatePasskeyRegistration(ctx context.Context, req *InitiatePasskeyRegistrationRequest) (*InitiatePasskeyRegistrationResponse, error) {
	response, err := s.service.InitiatePasskeyRegistration(ctx, uint(req.UserId), uint(req.ClubId))
	if err != nil {
		return nil, s.handleError(err)
	}

	s.logger.Info("Passkey registration initiated via gRPC", map[string]interface{}{
		"user_id": req.UserId,
		"club_id": req.ClubId,
	})

	return &InitiatePasskeyRegistrationResponse{
		Options: response.Options,
	}, nil
}

// User management gRPC methods

// GetUser retrieves a user by ID
func (s *AuthServiceServer) GetUser(ctx context.Context, req *GetUserRequest) (*GetUserResponse, error) {
	user, err := s.service.GetUser(ctx, uint(req.ClubId), uint(req.UserId))
	if err != nil {
		return nil, s.handleError(err)
	}

	return &GetUserResponse{
		User: s.convertUserToProto(user),
	}, nil
}

// GetUserWithRoles retrieves a user with their roles and permissions
func (s *AuthServiceServer) GetUserWithRoles(ctx context.Context, req *GetUserWithRolesRequest) (*GetUserWithRolesResponse, error) {
	userWithRoles, err := s.service.GetUserWithRoles(ctx, uint(req.ClubId), uint(req.UserId))
	if err != nil {
		return nil, s.handleError(err)
	}

	// Convert role names to ProtoRole objects
	protoRoles := make([]*ProtoRole, len(userWithRoles.RoleNames))
	for i, roleName := range userWithRoles.RoleNames {
		protoRoles[i] = &ProtoRole{
			Name: roleName,
		}
	}

	// Convert permission names to ProtoPermission objects
	protoPermissions := make([]*ProtoPermission, len(userWithRoles.Permissions))
	for i, permissionName := range userWithRoles.Permissions {
		protoPermissions[i] = &ProtoPermission{
			Name: permissionName,
		}
	}

	return &GetUserWithRolesResponse{
		User:        s.convertUserToProto(&userWithRoles.User),
		Roles:       protoRoles,
		Permissions: protoPermissions,
	}, nil
}

// Health check gRPC methods

// HealthCheck performs a health check
func (s *AuthServiceServer) HealthCheck(ctx context.Context, req *HealthCheckRequest) (*HealthCheckResponse, error) {
	err := s.service.HealthCheck(ctx)
	if err != nil {
		return &HealthCheckResponse{
			Status:  "UNHEALTHY",
			Message: err.Error(),
		}, nil
	}

	return &HealthCheckResponse{
		Status:  "HEALTHY",
		Message: "Auth service is healthy",
	}, nil
}

// Helper methods for converting between service models and protobuf messages


// Error handling

func (s *AuthServiceServer) handleError(err error) error {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		grpcCode := s.getGRPCStatusCode(appErr.Code)

		s.logger.Error("gRPC request failed", map[string]interface{}{
			"error":       appErr.Error(),
			"error_code":  string(appErr.Code),
			"grpc_code":   grpcCode,
			"fields":      appErr.Fields,
		})

		return status.Error(grpcCode, appErr.Message)
	}

	// Generic error
	s.logger.Error("Unexpected gRPC error", map[string]interface{}{
		"error": err.Error(),
	})

	return status.Error(codes.Internal, "Internal server error")
}

func (s *AuthServiceServer) getGRPCStatusCode(errorCode apperrors.ErrorCode) codes.Code {
	switch errorCode {
	case apperrors.ErrNotFound:
		return codes.NotFound
	case apperrors.ErrInvalidInput:
		return codes.InvalidArgument
	case apperrors.ErrUnauthorized:
		return codes.Unauthenticated
	case apperrors.ErrForbidden:
		return codes.PermissionDenied
	case apperrors.ErrConflict:
		return codes.AlreadyExists
	case apperrors.ErrTimeout:
		return codes.DeadlineExceeded
	case apperrors.ErrUnavailable:
		return codes.Unavailable
	default:
		return codes.Internal
	}
}

