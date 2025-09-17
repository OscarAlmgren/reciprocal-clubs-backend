package handlers

import (
	"time"

	"reciprocal-clubs-backend/services/auth-service/internal/models"
)

// Placeholder gRPC request/response types
// In a real implementation, these would be generated from protobuf files

// RegisterUserRequest represents a user registration request
type RegisterUserRequest struct {
	Email     string `json:"email"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	ClubSlug  string `json:"club_slug"`
}

// RegisterUserResponse represents a user registration response
type RegisterUserResponse struct {
	User    *ProtoUser `json:"user"`
	Success bool       `json:"success"`
}

// InitiatePasskeyLoginRequest represents a passkey login initiation request
type InitiatePasskeyLoginRequest struct {
	Email    string `json:"email"`
	ClubSlug string `json:"club_slug"`
}

// InitiatePasskeyLoginResponse represents a passkey login initiation response
type InitiatePasskeyLoginResponse struct {
	Options map[string]interface{} `json:"options"`
	UserID  string                 `json:"user_id"`
}

// CompletePasskeyLoginRequest represents a passkey login completion request
type CompletePasskeyLoginRequest struct {
	UserID           string                 `json:"user_id"`
	ClubSlug         string                 `json:"club_slug"`
	CredentialResult map[string]interface{} `json:"credential_result"`
}

// CompletePasskeyLoginResponse represents a passkey login completion response
type CompletePasskeyLoginResponse struct {
	User         *ProtoUser `json:"user"`
	Token        string     `json:"token"`
	RefreshToken string     `json:"refresh_token"`
	ExpiresAt    *time.Time `json:"expires_at"`
}

// InitiatePasskeyRegistrationRequest represents a passkey registration initiation request
type InitiatePasskeyRegistrationRequest struct {
	UserId uint32 `json:"user_id"`
	ClubId uint32 `json:"club_id"`
}

// InitiatePasskeyRegistrationResponse represents a passkey registration initiation response
type InitiatePasskeyRegistrationResponse struct {
	Options map[string]interface{} `json:"options"`
}

// GetUserRequest represents a get user request
type GetUserRequest struct {
	UserId uint32 `json:"user_id"`
	ClubId uint32 `json:"club_id"`
}

// GetUserResponse represents a get user response
type GetUserResponse struct {
	User *ProtoUser `json:"user"`
}

// GetUserWithRolesRequest represents a get user with roles request
type GetUserWithRolesRequest struct {
	UserId uint32 `json:"user_id"`
	ClubId uint32 `json:"club_id"`
}

// GetUserWithRolesResponse represents a get user with roles response
type GetUserWithRolesResponse struct {
	User        *ProtoUser        `json:"user"`
	Roles       []*ProtoRole      `json:"roles"`
	Permissions []*ProtoPermission `json:"permissions"`
}

// HealthCheckRequest represents a health check request
type HealthCheckRequest struct{}

// HealthCheckResponse represents a health check response
type HealthCheckResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// LogoutRequest represents a logout request
type LogoutRequest struct {
	UserId       uint32 `json:"user_id"`
	ClubId       uint32 `json:"club_id"`
	SessionToken string `json:"session_token"`
}

// LogoutResponse represents a logout response
type LogoutResponse struct {
	Success bool `json:"success"`
}

// ValidateSessionRequest represents a session validation request
type ValidateSessionRequest struct {
	SessionToken string `json:"session_token"`
}

// ValidateSessionResponse represents a session validation response
type ValidateSessionResponse struct {
	Valid bool       `json:"valid"`
	User  *ProtoUser `json:"user"`
}

// ProtoUser represents a user in protobuf format
type ProtoUser struct {
	Id            uint32 `json:"id"`
	Email         string `json:"email"`
	Username      string `json:"username"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	Status        string `json:"status"`
	EmailVerified bool   `json:"email_verified"`
	CreatedAt     *time.Time `json:"created_at"`
	UpdatedAt     *time.Time `json:"updated_at"`
}

// ProtoRole represents a role in protobuf format
type ProtoRole struct {
	Id          uint32 `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsSystem    bool   `json:"is_system"`
}

// ProtoPermission represents a permission in protobuf format
type ProtoPermission struct {
	Id          uint32 `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
}

// Converter methods
func (s *AuthServiceServer) convertUserToProto(user *models.User) *ProtoUser {
	if user == nil {
		return nil
	}

	return &ProtoUser{
		Id:            uint32(user.ID),
		Email:         user.Email,
		Username:      user.Username,
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		Status:        string(user.Status),
		EmailVerified: user.EmailVerified,
		CreatedAt:     &user.CreatedAt,
		UpdatedAt:     &user.UpdatedAt,
	}
}

func (s *AuthServiceServer) convertRolesToProto(roles []models.UserRole) []*ProtoRole {
	protoRoles := make([]*ProtoRole, len(roles))
	for i, role := range roles {
		protoRoles[i] = &ProtoRole{
			Id:          uint32(role.Role.ID),
			Name:        role.Role.Name,
			Description: role.Role.Description,
			IsSystem:    role.Role.IsSystem,
		}
	}
	return protoRoles
}

func (s *AuthServiceServer) convertPermissionsToProto(permissions []models.Permission) []*ProtoPermission {
	protoPermissions := make([]*ProtoPermission, len(permissions))
	for i, perm := range permissions {
		protoPermissions[i] = &ProtoPermission{
			Id:          uint32(perm.ID),
			Name:        perm.Name,
			Description: perm.Description,
			Resource:    perm.Resource,
			Action:      perm.Action,
		}
	}
	return protoPermissions
}