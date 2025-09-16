package handlers

import (
	"context"
	"time"

	"reciprocal-clubs-backend/pkg/shared/errors"
	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	"reciprocal-clubs-backend/services/auth-service/internal/models"
	"reciprocal-clubs-backend/services/auth-service/internal/service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
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
		Email:       req.Email,
		DisplayName: req.DisplayName,
		ClubSlug:    req.ClubSlug,
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
		User:             s.convertUserToProto(response.User),
		RegistrationData: response.RegistrationData,
		Success:          true,
	}, nil
}

// InitiatePasskeyLogin starts the passkey authentication process
func (s *AuthServiceServer) InitiatePasskeyLogin(ctx context.Context, req *InitiatePasskeyLoginRequest) (*InitiatePasskeyLoginResponse, error) {
	serviceReq := &service.LoginRequest{
		Email:    req.Email,
		ClubSlug: req.ClubSlug,
	}

	response, err := s.service.InitiatePasskeyLogin(ctx, serviceReq)
	if err != nil {
		return nil, s.handleError(err)
	}

	s.logger.Debug("Passkey login initiated via gRPC", map[string]interface{}{
		"email": req.Email,
	})

	return &InitiatePasskeyLoginResponse{
		Challenge:    response.Challenge,
		HankoUserId:  response.HankoUserID,
		LoginData:    response.LoginData,
	}, nil
}

// CompletePasskeyLogin completes the passkey authentication process
func (s *AuthServiceServer) CompletePasskeyLogin(ctx context.Context, req *CompletePasskeyLoginRequest) (*CompletePasskeyLoginResponse, error) {
	response, err := s.service.CompletePasskeyLogin(ctx, req.ClubSlug, req.HankoUserId, req.CredentialResult)
	if err != nil {
		return nil, s.handleError(err)
	}

	s.logger.Info("Passkey login completed via gRPC", map[string]interface{}{
		"user_id": response.User.ID,
		"email":   response.User.Email,
	})

	return &CompletePasskeyLoginResponse{
		User:         s.convertUserToProto(response.User),
		SessionToken: response.SessionToken,
		ExpiresAt:    timestamppb.New(response.ExpiresAt),
		Success:      true,
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
		Message: "Logged out successfully",
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
		Challenge:        response.Challenge,
		RegistrationData: response.RegistrationData,
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

	return &GetUserWithRolesResponse{
		User:        s.convertUserToProto(&userWithRoles.User),
		Roles:       s.convertRolesToProto(userWithRoles.Roles),
		Permissions: s.convertPermissionsToProto(userWithRoles.Permissions),
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

func (s *AuthServiceServer) convertUserToProto(user *models.User) *User {
	if user == nil {
		return nil
	}

	return &User{
		Id:          uint64(user.ID),
		Email:       user.Email,
		DisplayName: user.DisplayName,
		HankoUserId: user.HankoUserID,
		Status:      string(user.Status),
		CreatedAt:   timestamppb.New(user.CreatedAt),
		UpdatedAt:   timestamppb.New(user.UpdatedAt),
	}
}

func (s *AuthServiceServer) convertRolesToProto(roles []models.Role) []*Role {
	protoRoles := make([]*Role, len(roles))
	for i, role := range roles {
		protoRoles[i] = &Role{
			Id:          uint64(role.ID),
			Name:        role.Name,
			Description: role.Description,
			ClubId:      uint64(role.ClubID),
			CreatedAt:   timestamppb.New(role.CreatedAt),
			UpdatedAt:   timestamppb.New(role.UpdatedAt),
		}
	}
	return protoRoles
}

func (s *AuthServiceServer) convertPermissionsToProto(permissions []models.Permission) []*Permission {
	protoPermissions := make([]*Permission, len(permissions))
	for i, permission := range permissions {
		protoPermissions[i] = &Permission{
			Id:          uint64(permission.ID),
			Name:        permission.Name,
			Description: permission.Description,
			Resource:    permission.Resource,
			Action:      permission.Action,
			CreatedAt:   timestamppb.New(permission.CreatedAt),
			UpdatedAt:   timestamppb.New(permission.UpdatedAt),
		}
	}
	return protoPermissions
}

// Error handling

func (s *AuthServiceServer) handleError(err error) error {
	var appErr *errors.AppError
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

func (s *AuthServiceServer) getGRPCStatusCode(errorCode errors.ErrorCode) codes.Code {
	switch errorCode {
	case errors.ErrNotFound:
		return codes.NotFound
	case errors.ErrInvalidInput:
		return codes.InvalidArgument
	case errors.ErrUnauthorized:
		return codes.Unauthenticated
	case errors.ErrForbidden:
		return codes.PermissionDenied
	case errors.ErrConflict:
		return codes.AlreadyExists
	case errors.ErrTimeout:
		return codes.DeadlineExceeded
	case errors.ErrUnavailable:
		return codes.Unavailable
	default:
		return codes.Internal
	}
}

// Protobuf message structs (these would normally be generated from .proto files)
// Including them here as placeholders for the actual generated code

type RegisterUserRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	ClubSlug    string `json:"club_slug"`
}

type RegisterUserResponse struct {
	User             *User                  `json:"user"`
	RegistrationData map[string]interface{} `json:"registration_data"`
	Success          bool                   `json:"success"`
}

type InitiatePasskeyLoginRequest struct {
	Email    string `json:"email"`
	ClubSlug string `json:"club_slug"`
}

type InitiatePasskeyLoginResponse struct {
	Challenge    string                 `json:"challenge"`
	HankoUserId  string                 `json:"hanko_user_id"`
	LoginData    map[string]interface{} `json:"login_data"`
}

type CompletePasskeyLoginRequest struct {
	ClubSlug         string                 `json:"club_slug"`
	HankoUserId      string                 `json:"hanko_user_id"`
	CredentialResult map[string]interface{} `json:"credential_result"`
}

type CompletePasskeyLoginResponse struct {
	User         *User                  `json:"user"`
	SessionToken string                 `json:"session_token"`
	ExpiresAt    *timestamppb.Timestamp `json:"expires_at"`
	Success      bool                   `json:"success"`
}

type LogoutRequest struct {
	UserId       uint64 `json:"user_id"`
	ClubId       uint64 `json:"club_id"`
	SessionToken string `json:"session_token"`
}

type LogoutResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type ValidateSessionRequest struct {
	SessionToken string `json:"session_token"`
}

type ValidateSessionResponse struct {
	Valid bool  `json:"valid"`
	User  *User `json:"user"`
}

type InitiatePasskeyRegistrationRequest struct {
	UserId uint64 `json:"user_id"`
	ClubId uint64 `json:"club_id"`
}

type InitiatePasskeyRegistrationResponse struct {
	Challenge        string                 `json:"challenge"`
	RegistrationData map[string]interface{} `json:"registration_data"`
}

type GetUserRequest struct {
	ClubId uint64 `json:"club_id"`
	UserId uint64 `json:"user_id"`
}

type GetUserResponse struct {
	User *User `json:"user"`
}

type GetUserWithRolesRequest struct {
	ClubId uint64 `json:"club_id"`
	UserId uint64 `json:"user_id"`
}

type GetUserWithRolesResponse struct {
	User        *User         `json:"user"`
	Roles       []*Role       `json:"roles"`
	Permissions []*Permission `json:"permissions"`
}

type HealthCheckRequest struct{}

type HealthCheckResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type User struct {
	Id          uint64                 `json:"id"`
	Email       string                 `json:"email"`
	DisplayName string                 `json:"display_name"`
	HankoUserId string                 `json:"hanko_user_id"`
	Status      string                 `json:"status"`
	CreatedAt   *timestamppb.Timestamp `json:"created_at"`
	UpdatedAt   *timestamppb.Timestamp `json:"updated_at"`
}

type Role struct {
	Id          uint64                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	ClubId      uint64                 `json:"club_id"`
	CreatedAt   *timestamppb.Timestamp `json:"created_at"`
	UpdatedAt   *timestamppb.Timestamp `json:"updated_at"`
}

type Permission struct {
	Id          uint64                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Resource    string                 `json:"resource"`
	Action      string                 `json:"action"`
	CreatedAt   *timestamppb.Timestamp `json:"created_at"`
	UpdatedAt   *timestamppb.Timestamp `json:"updated_at"`
}