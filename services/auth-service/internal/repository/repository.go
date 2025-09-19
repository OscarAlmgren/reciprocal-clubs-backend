package repository

import (
	"context"
	"time"

	"reciprocal-clubs-backend/pkg/shared/database"
	"reciprocal-clubs-backend/pkg/shared/errors"
	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/services/auth-service/internal/models"

	"gorm.io/gorm"
)

// AuthRepository handles database operations for authentication
type AuthRepository struct {
	*database.BaseRepository
	db     *database.Database
	logger logging.Logger
}

// NewAuthRepository creates a new auth repository
func NewAuthRepository(db *database.Database, logger logging.Logger) *AuthRepository {
	return &AuthRepository{
		BaseRepository: database.NewBaseRepository(db, logger),
		db:             db,
		logger:         logger,
	}
}

// User operations

// CreateUser creates a new user
func (r *AuthRepository) CreateUser(ctx context.Context, user *models.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		r.logger.Error("Failed to create user", map[string]interface{}{
			"error":         err.Error(),
			"email":         user.Email,
			"hanko_user_id": user.HankoUserID,
		})
		return errors.Internal("Failed to create user", map[string]interface{}{
			"email": user.Email,
		}, err)
	}

	r.logger.Info("User created successfully", map[string]interface{}{
		"user_id":       user.ID,
		"email":         user.Email,
		"hanko_user_id": user.HankoUserID,
		"club_id":       user.ClubID,
	})

	return nil
}

// GetUserByID retrieves a user by ID
func (r *AuthRepository) GetUserByID(ctx context.Context, clubID, userID uint) (*models.User, error) {
	var user models.User
	if err := r.db.WithTenant(clubID).WithContext(ctx).
		Preload("Roles.Role").
		Preload("Sessions").
		First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NotFound("User not found", map[string]interface{}{
				"user_id": userID,
				"club_id": clubID,
			})
		}
		return nil, errors.Internal("Failed to get user", map[string]interface{}{
			"user_id": userID,
			"club_id": clubID,
		}, err)
	}

	return &user, nil
}

// GetUserByEmail retrieves a user by email
func (r *AuthRepository) GetUserByEmail(ctx context.Context, clubID uint, email string) (*models.User, error) {
	var user models.User
	if err := r.db.WithTenant(clubID).WithContext(ctx).
		Preload("Roles.Role").
		Preload("Sessions").
		Where("email = ?", email).
		First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NotFound("User not found", map[string]interface{}{
				"email":   email,
				"club_id": clubID,
			})
		}
		return nil, errors.Internal("Failed to get user by email", map[string]interface{}{
			"email":   email,
			"club_id": clubID,
		}, err)
	}

	return &user, nil
}

// GetUserByPasswordResetToken retrieves a user by password reset token
func (r *AuthRepository) GetUserByPasswordResetToken(ctx context.Context, clubID uint, token string) (*models.User, error) {
	var user models.User
	if err := r.db.WithTenant(clubID).WithContext(ctx).
		Where("password_reset_token = ? AND password_reset_expires_at > ?", token, time.Now()).
		First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NotFound("Invalid or expired reset token", map[string]interface{}{
				"club_id": clubID,
			})
		}
		return nil, errors.Internal("Failed to get user by reset token", map[string]interface{}{
			"club_id": clubID,
		}, err)
	}

	return &user, nil
}

// GetUserByEmailVerificationToken retrieves a user by email verification token
func (r *AuthRepository) GetUserByEmailVerificationToken(ctx context.Context, clubID uint, token string) (*models.User, error) {
	var user models.User
	if err := r.db.WithTenant(clubID).WithContext(ctx).
		Where("email_verification_token = ? AND email_verification_expires_at > ?", token, time.Now()).
		First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NotFound("Invalid or expired verification token", map[string]interface{}{
				"club_id": clubID,
			})
		}
		return nil, errors.Internal("Failed to get user by verification token", map[string]interface{}{
			"club_id": clubID,
		}, err)
	}

	return &user, nil
}

// GetUserByHankoID retrieves a user by Hanko user ID
func (r *AuthRepository) GetUserByHankoID(ctx context.Context, clubID uint, hankoUserID string) (*models.User, error) {
	var user models.User
	if err := r.db.WithTenant(clubID).WithContext(ctx).
		Preload("Roles.Role").
		Preload("Sessions").
		Where("hanko_user_id = ?", hankoUserID).
		First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NotFound("User not found", map[string]interface{}{
				"hanko_user_id": hankoUserID,
				"club_id":       clubID,
			})
		}
		return nil, errors.Internal("Failed to get user by Hanko ID", map[string]interface{}{
			"hanko_user_id": hankoUserID,
			"club_id":       clubID,
		}, err)
	}

	return &user, nil
}

// UpdateUser updates a user
func (r *AuthRepository) UpdateUser(ctx context.Context, user *models.User) error {
	if err := r.db.WithTenant(user.ClubID).WithContext(ctx).Save(user).Error; err != nil {
		r.logger.Error("Failed to update user", map[string]interface{}{
			"error":   err.Error(),
			"user_id": user.ID,
			"club_id": user.ClubID,
		})
		return errors.Internal("Failed to update user", map[string]interface{}{
			"user_id": user.ID,
		}, err)
	}

	return nil
}

// DeleteUser deletes a user (soft delete)
func (r *AuthRepository) DeleteUser(ctx context.Context, clubID, userID uint) error {
	if err := r.db.WithTenant(clubID).WithContext(ctx).Delete(&models.User{}, userID).Error; err != nil {
		return errors.Internal("Failed to delete user", map[string]interface{}{
			"user_id": userID,
		}, err)
	}

	return nil
}

// ListUsers lists users with pagination
func (r *AuthRepository) ListUsers(ctx context.Context, clubID uint, offset, limit int) ([]*models.User, int64, error) {
	var users []*models.User
	var total int64

	query := r.db.WithTenant(clubID).WithContext(ctx).Model(&models.User{})

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, errors.Internal("Failed to count users", nil, err)
	}

	// Get users with pagination
	if err := query.
		Preload("Roles.Role").
		Offset(offset).
		Limit(limit).
		Find(&users).Error; err != nil {
		return nil, 0, errors.Internal("Failed to list users", nil, err)
	}

	return users, total, nil
}

// Club operations

// CreateClub creates a new club
func (r *AuthRepository) CreateClub(ctx context.Context, club *models.Club) error {
	if err := r.db.WithContext(ctx).Create(club).Error; err != nil {
		r.logger.Error("Failed to create club", map[string]interface{}{
			"error": err.Error(),
			"name":  club.Name,
			"slug":  club.Slug,
		})
		return errors.Internal("Failed to create club", map[string]interface{}{
			"name": club.Name,
		}, err)
	}

	r.logger.Info("Club created successfully", map[string]interface{}{
		"club_id": club.ID,
		"name":    club.Name,
		"slug":    club.Slug,
	})

	return nil
}

// GetClubByID retrieves a club by ID
func (r *AuthRepository) GetClubByID(ctx context.Context, clubID uint) (*models.Club, error) {
	var club models.Club
	if err := r.db.WithContext(ctx).First(&club, clubID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NotFound("Club not found", map[string]interface{}{
				"club_id": clubID,
			})
		}
		return nil, errors.Internal("Failed to get club", map[string]interface{}{
			"club_id": clubID,
		}, err)
	}

	return &club, nil
}

// GetClubBySlug retrieves a club by slug
func (r *AuthRepository) GetClubBySlug(ctx context.Context, slug string) (*models.Club, error) {
	var club models.Club
	if err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&club).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NotFound("Club not found", map[string]interface{}{
				"slug": slug,
			})
		}
		return nil, errors.Internal("Failed to get club by slug", map[string]interface{}{
			"slug": slug,
		}, err)
	}

	return &club, nil
}

// UpdateClub updates a club
func (r *AuthRepository) UpdateClub(ctx context.Context, club *models.Club) error {
	if err := r.db.WithContext(ctx).Save(club).Error; err != nil {
		return errors.Internal("Failed to update club", map[string]interface{}{
			"club_id": club.ID,
		}, err)
	}

	return nil
}

// Role operations

// CreateRole creates a new role
func (r *AuthRepository) CreateRole(ctx context.Context, role *models.Role) error {
	if err := r.db.WithTenant(role.ClubID).WithContext(ctx).Create(role).Error; err != nil {
		return errors.Internal("Failed to create role", map[string]interface{}{
			"name":    role.Name,
			"club_id": role.ClubID,
		}, err)
	}

	return nil
}

// GetRoleByName retrieves a role by name
func (r *AuthRepository) GetRoleByName(ctx context.Context, clubID uint, name string) (*models.Role, error) {
	var role models.Role
	if err := r.db.WithTenant(clubID).WithContext(ctx).
		Preload("RolePermissions.Permission").
		Where("name = ?", name).
		First(&role).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NotFound("Role not found", map[string]interface{}{
				"name":    name,
				"club_id": clubID,
			})
		}
		return nil, errors.Internal("Failed to get role", map[string]interface{}{
			"name":    name,
			"club_id": clubID,
		}, err)
	}

	return &role, nil
}

// ListRoles lists all roles for a club
func (r *AuthRepository) ListRoles(ctx context.Context, clubID uint) ([]*models.Role, error) {
	var roles []*models.Role
	if err := r.db.WithTenant(clubID).WithContext(ctx).
		Preload("RolePermissions.Permission").
		Find(&roles).Error; err != nil {
		return nil, errors.Internal("Failed to list roles", map[string]interface{}{
			"club_id": clubID,
		}, err)
	}

	return roles, nil
}

// AssignRole assigns a role to a user
func (r *AuthRepository) AssignRole(ctx context.Context, userRole *models.UserRole) error {
	// Check if role assignment already exists
	var existing models.UserRole
	err := r.db.WithTenant(userRole.ClubID).WithContext(ctx).
		Where("user_id = ? AND role_id = ? AND is_active = ?", userRole.UserID, userRole.RoleID, true).
		First(&existing).Error

	if err == nil {
		return errors.Conflict("Role already assigned to user", map[string]interface{}{
			"user_id": userRole.UserID,
			"role_id": userRole.RoleID,
		})
	}

	if err != gorm.ErrRecordNotFound {
		return errors.Internal("Failed to check existing role assignment", nil, err)
	}

	// Create new role assignment
	if err := r.db.WithTenant(userRole.ClubID).WithContext(ctx).Create(userRole).Error; err != nil {
		return errors.Internal("Failed to assign role", map[string]interface{}{
			"user_id": userRole.UserID,
			"role_id": userRole.RoleID,
		}, err)
	}

	return nil
}

// RevokeRole revokes a role from a user
func (r *AuthRepository) RevokeRole(ctx context.Context, clubID, userID, roleID uint) error {
	if err := r.db.WithTenant(clubID).WithContext(ctx).
		Model(&models.UserRole{}).
		Where("user_id = ? AND role_id = ?", userID, roleID).
		Update("is_active", false).Error; err != nil {
		return errors.Internal("Failed to revoke role", map[string]interface{}{
			"user_id": userID,
			"role_id": roleID,
		}, err)
	}

	return nil
}

// GetUserRoles retrieves all active roles for a user
func (r *AuthRepository) GetUserRoles(ctx context.Context, clubID, userID uint) ([]*models.Role, error) {
	var roles []*models.Role
	if err := r.db.WithTenant(clubID).WithContext(ctx).
		Table("roles").
		Select("roles.*").
		Joins("JOIN user_roles ON roles.id = user_roles.role_id").
		Where("user_roles.user_id = ? AND user_roles.is_active = ? AND (user_roles.expires_at IS NULL OR user_roles.expires_at > ?)",
			userID, true, time.Now()).
		Preload("RolePermissions.Permission").
		Find(&roles).Error; err != nil {
		return nil, errors.Internal("Failed to get user roles", map[string]interface{}{
			"user_id": userID,
		}, err)
	}

	return roles, nil
}

// GetUserPermissions retrieves all permissions for a user
func (r *AuthRepository) GetUserPermissions(ctx context.Context, clubID, userID uint) ([]*models.Permission, error) {
	var permissions []*models.Permission
	if err := r.db.WithTenant(clubID).WithContext(ctx).Raw(`
		SELECT DISTINCT p.*
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		JOIN user_roles ur ON rp.role_id = ur.role_id
		WHERE ur.user_id = ?
		  AND ur.is_active = true
		  AND (ur.expires_at IS NULL OR ur.expires_at > ?)
		  AND p.club_id = ?
	`, userID, time.Now(), clubID).Scan(&permissions).Error; err != nil {
		return nil, errors.Internal("Failed to get user permissions", map[string]interface{}{
			"user_id": userID,
		}, err)
	}

	return permissions, nil
}

// Session operations

// CreateSession creates a new user session
func (r *AuthRepository) CreateSession(ctx context.Context, session *models.UserSession) error {
	if err := r.db.WithTenant(session.ClubID).WithContext(ctx).Create(session).Error; err != nil {
		return errors.Internal("Failed to create session", map[string]interface{}{
			"user_id":          session.UserID,
			"hanko_session_id": session.HankoSessionID,
		}, err)
	}

	return nil
}

// GetSessionByHankoID retrieves a session by Hanko session ID
func (r *AuthRepository) GetSessionByHankoID(ctx context.Context, clubID uint, hankoSessionID string) (*models.UserSession, error) {
	var session models.UserSession
	if err := r.db.WithTenant(clubID).WithContext(ctx).
		Preload("User").
		Where("hanko_session_id = ?", hankoSessionID).
		First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NotFound("Session not found", map[string]interface{}{
				"hanko_session_id": hankoSessionID,
			})
		}
		return nil, errors.Internal("Failed to get session", map[string]interface{}{
			"hanko_session_id": hankoSessionID,
		}, err)
	}

	return &session, nil
}

// UpdateSession updates a session
func (r *AuthRepository) UpdateSession(ctx context.Context, session *models.UserSession) error {
	if err := r.db.WithTenant(session.ClubID).WithContext(ctx).Save(session).Error; err != nil {
		return errors.Internal("Failed to update session", map[string]interface{}{
			"session_id": session.ID,
		}, err)
	}

	return nil
}

// InvalidateSession invalidates a session
func (r *AuthRepository) InvalidateSession(ctx context.Context, clubID uint, hankoSessionID string) error {
	if err := r.db.WithTenant(clubID).WithContext(ctx).
		Model(&models.UserSession{}).
		Where("hanko_session_id = ?", hankoSessionID).
		Updates(map[string]interface{}{
			"is_active": false,
			"logout_at": time.Now(),
		}).Error; err != nil {
		return errors.Internal("Failed to invalidate session", map[string]interface{}{
			"hanko_session_id": hankoSessionID,
		}, err)
	}

	return nil
}

// CleanupExpiredSessions removes expired sessions
func (r *AuthRepository) CleanupExpiredSessions(ctx context.Context, clubID uint) error {
	if err := r.db.WithTenant(clubID).WithContext(ctx).
		Where("expires_at < ? OR (is_active = false AND logout_at < ?)", time.Now(), time.Now().AddDate(0, 0, -30)).
		Delete(&models.UserSession{}).Error; err != nil {
		return errors.Internal("Failed to cleanup expired sessions", map[string]interface{}{
			"club_id": clubID,
		}, err)
	}

	return nil
}

// Audit log operations

// CreateAuditLog creates a new audit log entry
func (r *AuthRepository) CreateAuditLog(ctx context.Context, auditLog *models.AuditLog) error {
	if err := r.db.WithTenant(auditLog.ClubID).WithContext(ctx).Create(auditLog).Error; err != nil {
		r.logger.Error("Failed to create audit log", map[string]interface{}{
			"error":        err.Error(),
			"action":       string(auditLog.Action),
			"user_id":      auditLog.UserID,
			"hanko_user_id": auditLog.HankoUserID,
		})
		return errors.Internal("Failed to create audit log", nil, err)
	}

	return nil
}

// GetAuditLogs retrieves audit logs with pagination
func (r *AuthRepository) GetAuditLogs(ctx context.Context, clubID uint, offset, limit int) ([]*models.AuditLog, int64, error) {
	var logs []*models.AuditLog
	var total int64

	query := r.db.WithTenant(clubID).WithContext(ctx).Model(&models.AuditLog{})

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, errors.Internal("Failed to count audit logs", nil, err)
	}

	// Get logs with pagination
	if err := query.
		Preload("User").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, 0, errors.Internal("Failed to get audit logs", nil, err)
	}

	return logs, total, nil
}

// Additional Role operations

// GetRoleByID retrieves a role by ID
func (r *AuthRepository) GetRoleByID(ctx context.Context, clubID, roleID uint) (*models.Role, error) {
	var role models.Role
	if err := r.db.WithTenant(clubID).WithContext(ctx).
		Preload("RolePermissions.Permission").
		First(&role, roleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NotFound("Role not found", map[string]interface{}{
				"role_id": roleID,
				"club_id": clubID,
			})
		}
		return nil, errors.Internal("Failed to get role", map[string]interface{}{
			"role_id": roleID,
			"club_id": clubID,
		}, err)
	}

	return &role, nil
}

// UpdateRole updates a role
func (r *AuthRepository) UpdateRole(ctx context.Context, role *models.Role) error {
	if err := r.db.WithTenant(role.ClubID).WithContext(ctx).Save(role).Error; err != nil {
		return errors.Internal("Failed to update role", map[string]interface{}{
			"role_id": role.ID,
			"club_id": role.ClubID,
		}, err)
	}

	return nil
}

// DeleteRole deletes a role (soft delete)
func (r *AuthRepository) DeleteRole(ctx context.Context, clubID, roleID uint) error {
	if err := r.db.WithTenant(clubID).WithContext(ctx).Delete(&models.Role{}, roleID).Error; err != nil {
		return errors.Internal("Failed to delete role", map[string]interface{}{
			"role_id": roleID,
			"club_id": clubID,
		}, err)
	}

	return nil
}

// GetRolesWithPagination lists roles with pagination
func (r *AuthRepository) GetRolesWithPagination(ctx context.Context, clubID uint, offset, limit int) ([]*models.Role, int64, error) {
	var roles []*models.Role
	var total int64

	query := r.db.WithTenant(clubID).WithContext(ctx).Model(&models.Role{})

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, errors.Internal("Failed to count roles", map[string]interface{}{
			"club_id": clubID,
		}, err)
	}

	// Get roles with pagination
	if err := query.
		Preload("RolePermissions.Permission").
		Offset(offset).
		Limit(limit).
		Find(&roles).Error; err != nil {
		return nil, 0, errors.Internal("Failed to list roles", map[string]interface{}{
			"club_id": clubID,
		}, err)
	}

	return roles, total, nil
}

// Permission operations

// CreatePermission creates a new permission
func (r *AuthRepository) CreatePermission(ctx context.Context, permission *models.Permission) error {
	if err := r.db.WithTenant(permission.ClubID).WithContext(ctx).Create(permission).Error; err != nil {
		return errors.Internal("Failed to create permission", map[string]interface{}{
			"name":     permission.Name,
			"club_id":  permission.ClubID,
			"resource": permission.Resource,
			"action":   permission.Action,
		}, err)
	}

	return nil
}

// GetPermissionByID retrieves a permission by ID
func (r *AuthRepository) GetPermissionByID(ctx context.Context, clubID, permissionID uint) (*models.Permission, error) {
	var permission models.Permission
	if err := r.db.WithTenant(clubID).WithContext(ctx).First(&permission, permissionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NotFound("Permission not found", map[string]interface{}{
				"permission_id": permissionID,
				"club_id":       clubID,
			})
		}
		return nil, errors.Internal("Failed to get permission", map[string]interface{}{
			"permission_id": permissionID,
			"club_id":       clubID,
		}, err)
	}

	return &permission, nil
}

// ListPermissions lists all permissions for a club
func (r *AuthRepository) ListPermissions(ctx context.Context, clubID uint) ([]*models.Permission, error) {
	var permissions []*models.Permission
	if err := r.db.WithTenant(clubID).WithContext(ctx).Find(&permissions).Error; err != nil {
		return nil, errors.Internal("Failed to list permissions", map[string]interface{}{
			"club_id": clubID,
		}, err)
	}

	return permissions, nil
}

// RolePermission operations

// AssignPermissionToRole assigns a permission to a role
func (r *AuthRepository) AssignPermissionToRole(ctx context.Context, rolePermission *models.RolePermission) error {
	// Check if permission is already assigned to role
	var existing models.RolePermission
	err := r.db.WithTenant(rolePermission.ClubID).WithContext(ctx).
		Where("role_id = ? AND permission_id = ?", rolePermission.RoleID, rolePermission.PermissionID).
		First(&existing).Error

	if err == nil {
		return errors.Conflict("Permission already assigned to role", map[string]interface{}{
			"role_id":       rolePermission.RoleID,
			"permission_id": rolePermission.PermissionID,
		})
	}

	if err != gorm.ErrRecordNotFound {
		return errors.Internal("Failed to check existing permission assignment", nil, err)
	}

	// Create new permission assignment
	if err := r.db.WithTenant(rolePermission.ClubID).WithContext(ctx).Create(rolePermission).Error; err != nil {
		return errors.Internal("Failed to assign permission to role", map[string]interface{}{
			"role_id":       rolePermission.RoleID,
			"permission_id": rolePermission.PermissionID,
		}, err)
	}

	return nil
}

// RemovePermissionFromRole removes a permission from a role
func (r *AuthRepository) RemovePermissionFromRole(ctx context.Context, clubID, roleID, permissionID uint) error {
	if err := r.db.WithTenant(clubID).WithContext(ctx).
		Where("role_id = ? AND permission_id = ?", roleID, permissionID).
		Delete(&models.RolePermission{}).Error; err != nil {
		return errors.Internal("Failed to remove permission from role", map[string]interface{}{
			"role_id":       roleID,
			"permission_id": permissionID,
		}, err)
	}

	return nil
}

// Additional Club operations

// ListClubs lists clubs with pagination
func (r *AuthRepository) ListClubs(ctx context.Context, offset, limit int) ([]*models.Club, int64, error) {
	var clubs []*models.Club
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Club{})

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, errors.Internal("Failed to count clubs", nil, err)
	}

	// Get clubs with pagination
	if err := query.
		Offset(offset).
		Limit(limit).
		Find(&clubs).Error; err != nil {
		return nil, 0, errors.Internal("Failed to list clubs", nil, err)
	}

	return clubs, total, nil
}

// GetClubsByStatus lists clubs by status with pagination
func (r *AuthRepository) GetClubsByStatus(ctx context.Context, status models.ClubStatus, offset, limit int) ([]*models.Club, int64, error) {
	var clubs []*models.Club
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Club{}).Where("status = ?", status)

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, errors.Internal("Failed to count clubs by status", map[string]interface{}{
			"status": string(status),
		}, err)
	}

	// Get clubs with pagination
	if err := query.
		Offset(offset).
		Limit(limit).
		Find(&clubs).Error; err != nil {
		return nil, 0, errors.Internal("Failed to list clubs by status", map[string]interface{}{
			"status": string(status),
		}, err)
	}

	return clubs, total, nil
}

// Advanced Audit Log operations

// GetAuditLogsByUser retrieves audit logs for a specific user
func (r *AuthRepository) GetAuditLogsByUser(ctx context.Context, clubID, userID uint, offset, limit int) ([]*models.AuditLog, int64, error) {
	var logs []*models.AuditLog
	var total int64

	query := r.db.WithTenant(clubID).WithContext(ctx).Model(&models.AuditLog{}).Where("user_id = ?", userID)

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, errors.Internal("Failed to count user audit logs", map[string]interface{}{
			"user_id": userID,
		}, err)
	}

	// Get logs with pagination
	if err := query.
		Preload("User").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, 0, errors.Internal("Failed to get user audit logs", map[string]interface{}{
			"user_id": userID,
		}, err)
	}

	return logs, total, nil
}

// GetAuditLogsByAction retrieves audit logs by action type
func (r *AuthRepository) GetAuditLogsByAction(ctx context.Context, clubID uint, action models.AuditAction, offset, limit int) ([]*models.AuditLog, int64, error) {
	var logs []*models.AuditLog
	var total int64

	query := r.db.WithTenant(clubID).WithContext(ctx).Model(&models.AuditLog{}).Where("action = ?", action)

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, errors.Internal("Failed to count audit logs by action", map[string]interface{}{
			"action": string(action),
		}, err)
	}

	// Get logs with pagination
	if err := query.
		Preload("User").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, 0, errors.Internal("Failed to get audit logs by action", map[string]interface{}{
			"action": string(action),
		}, err)
	}

	return logs, total, nil
}

// GetAuditLogsByTimeRange retrieves audit logs within a time range
func (r *AuthRepository) GetAuditLogsByTimeRange(ctx context.Context, clubID uint, startTime, endTime time.Time, offset, limit int) ([]*models.AuditLog, int64, error) {
	var logs []*models.AuditLog
	var total int64

	query := r.db.WithTenant(clubID).WithContext(ctx).Model(&models.AuditLog{}).
		Where("created_at BETWEEN ? AND ?", startTime, endTime)

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, errors.Internal("Failed to count audit logs by time range", map[string]interface{}{
			"start_time": startTime,
			"end_time":   endTime,
		}, err)
	}

	// Get logs with pagination
	if err := query.
		Preload("User").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, 0, errors.Internal("Failed to get audit logs by time range", map[string]interface{}{
			"start_time": startTime,
			"end_time":   endTime,
		}, err)
	}

	return logs, total, nil
}

// User search and filtering

// SearchUsers searches users by various criteria
func (r *AuthRepository) SearchUsers(ctx context.Context, clubID uint, searchTerm string, status models.UserStatus, offset, limit int) ([]*models.User, int64, error) {
	var users []*models.User
	var total int64

	query := r.db.WithTenant(clubID).WithContext(ctx).Model(&models.User{})

	// Add search conditions
	if searchTerm != "" {
		searchPattern := "%" + searchTerm + "%"
		query = query.Where("email ILIKE ? OR username ILIKE ? OR first_name ILIKE ? OR last_name ILIKE ?",
			searchPattern, searchPattern, searchPattern, searchPattern)
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, errors.Internal("Failed to count search results", map[string]interface{}{
			"search_term": searchTerm,
			"status":      string(status),
		}, err)
	}

	// Get users with pagination
	if err := query.
		Preload("Roles.Role").
		Offset(offset).
		Limit(limit).
		Find(&users).Error; err != nil {
		return nil, 0, errors.Internal("Failed to search users", map[string]interface{}{
			"search_term": searchTerm,
			"status":      string(status),
		}, err)
	}

	return users, total, nil
}

// GetUsersByStatus retrieves users by status
func (r *AuthRepository) GetUsersByStatus(ctx context.Context, clubID uint, status models.UserStatus, offset, limit int) ([]*models.User, int64, error) {
	var users []*models.User
	var total int64

	query := r.db.WithTenant(clubID).WithContext(ctx).Model(&models.User{}).Where("status = ?", status)

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, errors.Internal("Failed to count users by status", map[string]interface{}{
			"status": string(status),
		}, err)
	}

	// Get users with pagination
	if err := query.
		Preload("Roles.Role").
		Offset(offset).
		Limit(limit).
		Find(&users).Error; err != nil {
		return nil, 0, errors.Internal("Failed to get users by status", map[string]interface{}{
			"status": string(status),
		}, err)
	}

	return users, total, nil
}

// Health Check operations

// HealthCheck performs a health check on the database
func (r *AuthRepository) HealthCheck(ctx context.Context) error {
	// Simple ping to check database connectivity
	if err := r.db.WithContext(ctx).Exec("SELECT 1").Error; err != nil {
		return errors.Internal("Database health check failed", nil, err)
	}

	return nil
}

// Transaction support

// MFA Token operations

// CreateMFAToken creates a new MFA token
func (r *AuthRepository) CreateMFAToken(ctx context.Context, token *models.MFAToken) error {
	if err := r.db.WithTenant(token.ClubID).WithContext(ctx).Create(token).Error; err != nil {
		r.logger.Error("Failed to create MFA token", map[string]interface{}{
			"error":   err.Error(),
			"user_id": token.UserID,
			"type":    token.TokenType,
		})
		return errors.Internal("Failed to create MFA token", map[string]interface{}{
			"user_id": token.UserID,
		}, err)
	}

	return nil
}

// GetLatestMFAToken retrieves the latest MFA token for a user and type
func (r *AuthRepository) GetLatestMFAToken(ctx context.Context, userID uint, tokenType models.MFATokenType) (*models.MFAToken, error) {
	var token models.MFAToken
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND token_type = ?", userID, tokenType).
		Order("created_at DESC").
		First(&token).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NotFound("MFA token not found", map[string]interface{}{
				"user_id":    userID,
				"token_type": tokenType,
			})
		}
		return nil, errors.Internal("Failed to get MFA token", map[string]interface{}{
			"user_id":    userID,
			"token_type": tokenType,
		}, err)
	}

	return &token, nil
}

// UpdateMFAToken updates an MFA token
func (r *AuthRepository) UpdateMFAToken(ctx context.Context, token *models.MFAToken) error {
	if err := r.db.WithTenant(token.ClubID).WithContext(ctx).Save(token).Error; err != nil {
		r.logger.Error("Failed to update MFA token", map[string]interface{}{
			"error":    err.Error(),
			"token_id": token.ID,
		})
		return errors.Internal("Failed to update MFA token", map[string]interface{}{
			"token_id": token.ID,
		}, err)
	}

	return nil
}

// InvalidateAllUserSessions invalidates all sessions for a user
func (r *AuthRepository) InvalidateAllUserSessions(ctx context.Context, clubID, userID uint) error {
	if err := r.db.WithTenant(clubID).WithContext(ctx).
		Model(&models.UserSession{}).
		Where("user_id = ?", userID).
		Update("is_active", false).Error; err != nil {
		r.logger.Error("Failed to invalidate user sessions", map[string]interface{}{
			"error":   err.Error(),
			"user_id": userID,
			"club_id": clubID,
		})
		return errors.Internal("Failed to invalidate user sessions", map[string]interface{}{
			"user_id": userID,
		}, err)
	}

	return nil
}

// WithTransaction executes a function within a database transaction
func (r *AuthRepository) WithTransaction(ctx context.Context, fn func(*AuthRepository) error) error {
	return r.db.Transaction(ctx, func(tx *gorm.DB) error {
		txRepo := &AuthRepository{
			BaseRepository: database.NewBaseRepository(&database.Database{DB: tx}, r.logger),
			db:             &database.Database{DB: tx},
			logger:         r.logger,
		}
		return fn(txRepo)
	})
}
