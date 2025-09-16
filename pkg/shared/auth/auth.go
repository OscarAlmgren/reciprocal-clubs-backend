package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"reciprocal-clubs-backend/pkg/shared/config"
	"reciprocal-clubs-backend/pkg/shared/errors"
	"reciprocal-clubs-backend/pkg/shared/logging"

	"github.com/golang-jwt/jwt/v5"
)

// Claims represents JWT claims with multi-tenant support
type Claims struct {
	UserID   uint     `json:"user_id"`
	ClubID   uint     `json:"club_id"`
	Email    string   `json:"email"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

// User represents an authenticated user
type User struct {
	ID          uint     `json:"id"`
	ClubID      uint     `json:"club_id"`
	Email       string   `json:"email"`
	Username    string   `json:"username"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

// AuthProvider defines the authentication interface
type AuthProvider interface {
	GenerateToken(user *User, expiration time.Duration) (string, error)
	ValidateToken(tokenString string) (*Claims, error)
	RefreshToken(tokenString string) (string, error)
	RevokeToken(tokenString string) error
}

// JWTProvider implements AuthProvider using JWT
type JWTProvider struct {
	config *config.AuthConfig
	logger logging.Logger
}

// ContextKey type for context keys
type ContextKey string

const (
	// UserContextKey is the context key for authenticated user
	UserContextKey ContextKey = "auth_user"
	// ClaimsContextKey is the context key for JWT claims
	ClaimsContextKey ContextKey = "auth_claims"
)

// NewJWTProvider creates a new JWT auth provider
func NewJWTProvider(cfg *config.AuthConfig, logger logging.Logger) *JWTProvider {
	return &JWTProvider{
		config: cfg,
		logger: logger,
	}
}

// GenerateToken generates a JWT token for the user
func (p *JWTProvider) GenerateToken(user *User, expiration time.Duration) (string, error) {
	if expiration == 0 {
		expiration = time.Duration(p.config.JWTExpiration) * time.Second
	}

	claims := &Claims{
		UserID:      user.ID,
		ClubID:      user.ClubID,
		Email:       user.Email,
		Username:    user.Username,
		Roles:       user.Roles,
		Permissions: user.Permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    p.config.Issuer,
			Audience:  []string{p.config.Audience},
			Subject:   fmt.Sprintf("%d", user.ID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(p.config.JWTSecret))
	if err != nil {
		p.logger.Error("Failed to generate JWT token", map[string]interface{}{
			"error":   err.Error(),
			"user_id": user.ID,
			"club_id": user.ClubID,
		})
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	p.logger.Debug("JWT token generated", map[string]interface{}{
		"user_id": user.ID,
		"club_id": user.ClubID,
		"expires_at": claims.ExpiresAt.Time,
	})

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns claims
func (p *JWTProvider) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(p.config.JWTSecret), nil
	})

	if err != nil {
		p.logger.Warn("JWT token validation failed", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Validate issuer and audience
	if claims.Issuer != p.config.Issuer {
		return nil, fmt.Errorf("invalid token issuer")
	}

	// Validate audience
	validAudience := false
	for _, aud := range claims.Audience {
		if aud == p.config.Audience {
			validAudience = true
			break
		}
	}
	if !validAudience {
		return nil, fmt.Errorf("invalid token audience")
	}

	return claims, nil
}

// RefreshToken creates a new token from an existing valid token
func (p *JWTProvider) RefreshToken(tokenString string) (string, error) {
	claims, err := p.ValidateToken(tokenString)
	if err != nil {
		return "", fmt.Errorf("cannot refresh invalid token: %w", err)
	}

	user := &User{
		ID:          claims.UserID,
		ClubID:      claims.ClubID,
		Email:       claims.Email,
		Username:    claims.Username,
		Roles:       claims.Roles,
		Permissions: claims.Permissions,
	}

	return p.GenerateToken(user, 0)
}

// RevokeToken revokes a JWT token (in a real implementation, you'd maintain a blacklist)
func (p *JWTProvider) RevokeToken(tokenString string) error {
	// In a production environment, you would:
	// 1. Add the token to a blacklist/revocation list
	// 2. Store it in Redis with expiration
	// 3. Check against this list in ValidateToken

	p.logger.Info("Token revoked", map[string]interface{}{
		"token_prefix": tokenString[:min(len(tokenString), 20)] + "...",
	})

	return nil
}

// Middleware creates an HTTP middleware for JWT authentication
func (p *JWTProvider) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			bearerToken := strings.Split(authHeader, " ")
			if len(bearerToken) != 2 || strings.ToLower(bearerToken[0]) != "bearer" {
				http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
				return
			}

			tokenString := bearerToken[1]
			claims, err := p.ValidateToken(tokenString)
			if err != nil {
				p.logger.Warn("Authentication failed", map[string]interface{}{
					"error": err.Error(),
					"path":  r.URL.Path,
				})
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			// Create user from claims
			user := &User{
				ID:          claims.UserID,
				ClubID:      claims.ClubID,
				Email:       claims.Email,
				Username:    claims.Username,
				Roles:       claims.Roles,
				Permissions: claims.Permissions,
			}

			// Add user and claims to request context
			ctx := context.WithValue(r.Context(), UserContextKey, user)
			ctx = context.WithValue(ctx, ClaimsContextKey, claims)
			ctx = logging.ContextWithUserID(ctx, user.ID)
			ctx = logging.ContextWithClubID(ctx, user.ClubID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRoles creates middleware that requires specific roles
func (p *JWTProvider) RequireRoles(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUserFromContext(r.Context())
			if user == nil {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			if !p.hasAnyRole(user, roles) {
				p.logger.Warn("Access denied - insufficient roles", map[string]interface{}{
					"user_id":       user.ID,
					"required_roles": roles,
					"user_roles":    user.Roles,
					"path":          r.URL.Path,
				})
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequirePermissions creates middleware that requires specific permissions
func (p *JWTProvider) RequirePermissions(permissions ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUserFromContext(r.Context())
			if user == nil {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			if !p.hasAllPermissions(user, permissions) {
				p.logger.Warn("Access denied - insufficient permissions", map[string]interface{}{
					"user_id":            user.ID,
					"required_permissions": permissions,
					"user_permissions":   user.Permissions,
					"path":               r.URL.Path,
				})
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireClub creates middleware that ensures user belongs to specific club
func (p *JWTProvider) RequireClub(clubID uint) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUserFromContext(r.Context())
			if user == nil {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			if user.ClubID != clubID {
				p.logger.Warn("Access denied - wrong club", map[string]interface{}{
					"user_id":     user.ID,
					"user_club":   user.ClubID,
					"required_club": clubID,
					"path":        r.URL.Path,
				})
				http.Error(w, "Access denied", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetUserFromContext extracts the authenticated user from context
func GetUserFromContext(ctx context.Context) *User {
	if user, ok := ctx.Value(UserContextKey).(*User); ok {
		return user
	}
	return nil
}

// GetClaimsFromContext extracts JWT claims from context
func GetClaimsFromContext(ctx context.Context) *Claims {
	if claims, ok := ctx.Value(ClaimsContextKey).(*Claims); ok {
		return claims
	}
	return nil
}

// MustGetUserFromContext extracts user from context or panics
func MustGetUserFromContext(ctx context.Context) *User {
	user := GetUserFromContext(ctx)
	if user == nil {
		panic("no authenticated user in context")
	}
	return user
}

// Helper functions

func (p *JWTProvider) hasAnyRole(user *User, requiredRoles []string) bool {
	userRoleMap := make(map[string]bool)
	for _, role := range user.Roles {
		userRoleMap[role] = true
	}

	for _, role := range requiredRoles {
		if userRoleMap[role] {
			return true
		}
	}
	return false
}

func (p *JWTProvider) hasAllPermissions(user *User, requiredPermissions []string) bool {
	userPermissionMap := make(map[string]bool)
	for _, permission := range user.Permissions {
		userPermissionMap[permission] = true
	}

	for _, permission := range requiredPermissions {
		if !userPermissionMap[permission] {
			return false
		}
	}
	return true
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ValidateUserAccess validates if a user can access a resource for a specific club
func ValidateUserAccess(ctx context.Context, clubID uint) *errors.AppError {
	user := GetUserFromContext(ctx)
	if user == nil {
		return errors.Unauthorized("Authentication required", nil)
	}

	if user.ClubID != clubID {
		return errors.Forbidden("Access denied - wrong club", map[string]interface{}{
			"user_club_id":     user.ClubID,
			"requested_club_id": clubID,
		})
	}

	return nil
}