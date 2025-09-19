package repository

import (
	"context"
	"time"

	"reciprocal-clubs-backend/pkg/shared/database"
	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/services/notification-service/internal/models"

	"gorm.io/gorm"
)

// Repository handles database operations for notification service
type Repository struct {
	*database.BaseRepository
	db     *gorm.DB
	logger logging.Logger
}

// NewRepository creates a new notification repository
func NewRepository(db *gorm.DB, logger logging.Logger) *Repository {
	return &Repository{
		db:     db,
		logger: logger,
	}
}

// GetDB returns the underlying database connection
func (r *Repository) GetDB() *gorm.DB {
	return r.db
}

// Notification operations

// CreateNotification creates a new notification
func (r *Repository) CreateNotification(ctx context.Context, notification *models.Notification) error {
	if err := r.db.WithContext(ctx).Create(notification).Error; err != nil {
		r.logger.Error("Failed to create notification", map[string]interface{}{
			"error":     err.Error(),
			"club_id":   notification.ClubID,
			"type":      notification.Type,
			"recipient": notification.Recipient,
		})
		return err
	}

	r.logger.Info("Notification created successfully", map[string]interface{}{
		"notification_id": notification.ID,
		"club_id":         notification.ClubID,
		"type":            notification.Type,
		"status":          notification.Status,
	})

	return nil
}

// GetNotificationByID retrieves a notification by ID
func (r *Repository) GetNotificationByID(ctx context.Context, id uint) (*models.Notification, error) {
	var notification models.Notification
	if err := r.db.WithContext(ctx).First(&notification, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		r.logger.Error("Failed to get notification", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		return nil, err
	}

	return &notification, nil
}

// GetNotificationsByClub retrieves notifications for a specific club
func (r *Repository) GetNotificationsByClub(ctx context.Context, clubID uint, limit, offset int) ([]models.Notification, error) {
	var notifications []models.Notification
	query := r.db.WithContext(ctx).Where("club_id = ?", clubID)

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Order("created_at DESC").Find(&notifications).Error; err != nil {
		r.logger.Error("Failed to get notifications by club", map[string]interface{}{
			"error":   err.Error(),
			"club_id": clubID,
		})
		return nil, err
	}

	return notifications, nil
}

// GetNotificationsByUser retrieves notifications for a specific user
func (r *Repository) GetNotificationsByUser(ctx context.Context, userID string, clubID uint, limit, offset int) ([]models.Notification, error) {
	var notifications []models.Notification
	query := r.db.WithContext(ctx).Where("user_id = ? AND club_id = ?", userID, clubID)

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Order("created_at DESC").Find(&notifications).Error; err != nil {
		r.logger.Error("Failed to get notifications by user", map[string]interface{}{
			"error":   err.Error(),
			"user_id": userID,
			"club_id": clubID,
		})
		return nil, err
	}

	return notifications, nil
}

// GetPendingNotifications retrieves notifications ready to be sent
func (r *Repository) GetPendingNotifications(ctx context.Context, limit int) ([]models.Notification, error) {
	var notifications []models.Notification
	query := r.db.WithContext(ctx).
		Where("status = ?", models.NotificationStatusPending).
		Where("scheduled_for IS NULL OR scheduled_for <= ?", time.Now())

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Order("priority DESC, created_at ASC").Find(&notifications).Error; err != nil {
		r.logger.Error("Failed to get pending notifications", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	return notifications, nil
}

// GetFailedNotifications retrieves notifications that can be retried
func (r *Repository) GetFailedNotifications(ctx context.Context, limit int) ([]models.Notification, error) {
	var notifications []models.Notification
	query := r.db.WithContext(ctx).
		Where("status = ? AND retry_count < ?", models.NotificationStatusFailed, 3)

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Order("priority DESC, failed_at ASC").Find(&notifications).Error; err != nil {
		r.logger.Error("Failed to get failed notifications", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	return notifications, nil
}

// UpdateNotification updates an existing notification
func (r *Repository) UpdateNotification(ctx context.Context, notification *models.Notification) error {
	if err := r.db.WithContext(ctx).Save(notification).Error; err != nil {
		r.logger.Error("Failed to update notification", map[string]interface{}{
			"error":           err.Error(),
			"notification_id": notification.ID,
		})
		return err
	}

	r.logger.Info("Notification updated successfully", map[string]interface{}{
		"notification_id": notification.ID,
		"status":          notification.Status,
	})

	return nil
}

// NotificationTemplate operations

// CreateNotificationTemplate creates a new notification template
func (r *Repository) CreateNotificationTemplate(ctx context.Context, template *models.NotificationTemplate) error {
	if err := r.db.WithContext(ctx).Create(template).Error; err != nil {
		r.logger.Error("Failed to create notification template", map[string]interface{}{
			"error":   err.Error(),
			"club_id": template.ClubID,
			"name":    template.Name,
		})
		return err
	}

	r.logger.Info("Notification template created successfully", map[string]interface{}{
		"template_id": template.ID,
		"club_id":     template.ClubID,
		"name":        template.Name,
	})

	return nil
}

// GetNotificationTemplateByID retrieves a notification template by ID
func (r *Repository) GetNotificationTemplateByID(ctx context.Context, id uint) (*models.NotificationTemplate, error) {
	var template models.NotificationTemplate
	if err := r.db.WithContext(ctx).First(&template, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		r.logger.Error("Failed to get notification template", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		return nil, err
	}

	return &template, nil
}

// GetNotificationTemplatesByClub retrieves notification templates for a club
func (r *Repository) GetNotificationTemplatesByClub(ctx context.Context, clubID uint) ([]models.NotificationTemplate, error) {
	var templates []models.NotificationTemplate
	if err := r.db.WithContext(ctx).
		Where("club_id = ? AND is_active = ?", clubID, true).
		Find(&templates).Error; err != nil {
		r.logger.Error("Failed to get notification templates by club", map[string]interface{}{
			"error":   err.Error(),
			"club_id": clubID,
		})
		return nil, err
	}

	return templates, nil
}

// UpdateNotificationTemplate updates an existing notification template
func (r *Repository) UpdateNotificationTemplate(ctx context.Context, template *models.NotificationTemplate) error {
	if err := r.db.WithContext(ctx).Save(template).Error; err != nil {
		r.logger.Error("Failed to update notification template", map[string]interface{}{
			"error":       err.Error(),
			"template_id": template.ID,
		})
		return err
	}

	r.logger.Info("Notification template updated successfully", map[string]interface{}{
		"template_id": template.ID,
		"name":        template.Name,
	})

	return nil
}

// NotificationPreference operations

// CreateNotificationPreference creates a new notification preference
func (r *Repository) CreateNotificationPreference(ctx context.Context, preference *models.NotificationPreference) error {
	if err := r.db.WithContext(ctx).Create(preference).Error; err != nil {
		r.logger.Error("Failed to create notification preference", map[string]interface{}{
			"error":   err.Error(),
			"user_id": preference.UserID,
			"club_id": preference.ClubID,
		})
		return err
	}

	return nil
}

// GetNotificationPreferences retrieves notification preferences for a user
func (r *Repository) GetNotificationPreferences(ctx context.Context, userID string, clubID uint) ([]models.NotificationPreference, error) {
	var preferences []models.NotificationPreference
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND club_id = ?", userID, clubID).
		Find(&preferences).Error; err != nil {
		r.logger.Error("Failed to get notification preferences", map[string]interface{}{
			"error":   err.Error(),
			"user_id": userID,
			"club_id": clubID,
		})
		return nil, err
	}

	return preferences, nil
}

// UpdateNotificationPreference updates an existing notification preference
func (r *Repository) UpdateNotificationPreference(ctx context.Context, preference *models.NotificationPreference) error {
	if err := r.db.WithContext(ctx).Save(preference).Error; err != nil {
		r.logger.Error("Failed to update notification preference", map[string]interface{}{
			"error":        err.Error(),
			"preference_id": preference.ID,
		})
		return err
	}

	return nil
}

// GetNotificationStats retrieves notification statistics for a club
func (r *Repository) GetNotificationStats(ctx context.Context, clubID uint, fromDate, toDate time.Time) (map[string]interface{}, error) {
	var stats struct {
		Total     int64 `json:"total"`
		Pending   int64 `json:"pending"`
		Sent      int64 `json:"sent"`
		Delivered int64 `json:"delivered"`
		Failed    int64 `json:"failed"`
		Read      int64 `json:"read"`
	}

	// Get total count
	if err := r.db.WithContext(ctx).Model(&models.Notification{}).
		Where("club_id = ? AND created_at BETWEEN ? AND ?", clubID, fromDate, toDate).
		Count(&stats.Total).Error; err != nil {
		return nil, err
	}

	// Get status counts
	statusCounts := make(map[string]int64)
	rows, err := r.db.WithContext(ctx).Model(&models.Notification{}).
		Select("status, COUNT(*) as count").
		Where("club_id = ? AND created_at BETWEEN ? AND ?", clubID, fromDate, toDate).
		Group("status").Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			continue
		}
		statusCounts[status] = count
	}

	return map[string]interface{}{
		"total":     stats.Total,
		"pending":   statusCounts["pending"],
		"sent":      statusCounts["sent"],
		"delivered": statusCounts["delivered"],
		"failed":    statusCounts["failed"],
		"read":      statusCounts["read"],
	}, nil
}

// Bulk operations

// CreateBulkNotifications creates multiple notifications in a transaction
func (r *Repository) CreateBulkNotifications(ctx context.Context, notifications []models.Notification) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, notification := range notifications {
			if err := tx.Create(&notification).Error; err != nil {
				r.logger.Error("Failed to create notification in bulk", map[string]interface{}{
					"error":     err.Error(),
					"club_id":   notification.ClubID,
					"type":      notification.Type,
					"recipient": notification.Recipient,
				})
				return err
			}
		}
		return nil
	})
}

// MarkMultipleAsRead marks multiple notifications as read
func (r *Repository) MarkMultipleAsRead(ctx context.Context, notificationIDs []uint) (int64, error) {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&models.Notification{}).
		Where("id IN ?", notificationIDs).
		Where("read_at IS NULL").
		Updates(map[string]interface{}{
			"status":  models.NotificationStatusRead,
			"read_at": now,
		})

	if result.Error != nil {
		r.logger.Error("Failed to mark multiple notifications as read", map[string]interface{}{
			"error": result.Error.Error(),
			"ids":   notificationIDs,
		})
		return 0, result.Error
	}

	r.logger.Info("Marked multiple notifications as read", map[string]interface{}{
		"count": result.RowsAffected,
		"ids":   notificationIDs,
	})

	return result.RowsAffected, nil
}

// DeleteNotificationTemplate soft deletes a notification template
func (r *Repository) DeleteNotificationTemplate(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).
		Model(&models.NotificationTemplate{}).
		Where("id = ?", id).
		Update("is_active", false)

	if result.Error != nil {
		r.logger.Error("Failed to delete notification template", map[string]interface{}{
			"error": result.Error.Error(),
			"id":    id,
		})
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	r.logger.Info("Notification template deleted successfully", map[string]interface{}{
		"template_id": id,
	})

	return nil
}

// GetUserPreferences retrieves user preferences with defaults if not found
func (r *Repository) GetUserPreferences(ctx context.Context, userID string, clubID uint) (*models.UserPreferences, error) {
	var preference models.UserPreferences
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND club_id = ?", userID, clubID).
		First(&preference).Error

	if err == gorm.ErrRecordNotFound {
		// Return default preferences
		return &models.UserPreferences{
			UserID:        userID,
			ClubID:        clubID,
			EmailEnabled:  true,
			SMSEnabled:    true,
			PushEnabled:   true,
			InAppEnabled:  true,
			PreferredLang: "en",
			Timezone:      "UTC",
		}, nil
	}

	if err != nil {
		r.logger.Error("Failed to get user preferences", map[string]interface{}{
			"error":   err.Error(),
			"user_id": userID,
			"club_id": clubID,
		})
		return nil, err
	}

	return &preference, nil
}

// UpsertUserPreferences creates or updates user preferences
func (r *Repository) UpsertUserPreferences(ctx context.Context, preference *models.UserPreferences) error {
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND club_id = ?", preference.UserID, preference.ClubID).
		FirstOrCreate(preference).Error

	if err != nil {
		r.logger.Error("Failed to upsert user preferences", map[string]interface{}{
			"error":   err.Error(),
			"user_id": preference.UserID,
			"club_id": preference.ClubID,
		})
		return err
	}

	r.logger.Info("User preferences updated successfully", map[string]interface{}{
		"user_id": preference.UserID,
		"club_id": preference.ClubID,
	})

	return nil
}

// Advanced query methods

// GetNotificationsByStatus retrieves notifications by status with pagination
func (r *Repository) GetNotificationsByStatus(ctx context.Context, clubID uint, status models.NotificationStatus, limit, offset int) ([]models.Notification, error) {
	var notifications []models.Notification
	query := r.db.WithContext(ctx).
		Where("club_id = ? AND status = ?", clubID, status).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&notifications).Error; err != nil {
		r.logger.Error("Failed to get notifications by status", map[string]interface{}{
			"error":   err.Error(),
			"club_id": clubID,
			"status":  status,
		})
		return nil, err
	}

	return notifications, nil
}

// GetNotificationsByType retrieves notifications by type with pagination
func (r *Repository) GetNotificationsByType(ctx context.Context, clubID uint, notificationType models.NotificationType, limit, offset int) ([]models.Notification, error) {
	var notifications []models.Notification
	query := r.db.WithContext(ctx).
		Where("club_id = ? AND type = ?", clubID, notificationType).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&notifications).Error; err != nil {
		r.logger.Error("Failed to get notifications by type", map[string]interface{}{
			"error":   err.Error(),
			"club_id": clubID,
			"type":    notificationType,
		})
		return nil, err
	}

	return notifications, nil
}

// GetUnreadNotificationsCount gets count of unread notifications for a user
func (r *Repository) GetUnreadNotificationsCount(ctx context.Context, userID string, clubID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.Notification{}).
		Where("user_id = ? AND club_id = ? AND read_at IS NULL", userID, clubID).
		Count(&count).Error

	if err != nil {
		r.logger.Error("Failed to get unread notifications count", map[string]interface{}{
			"error":   err.Error(),
			"user_id": userID,
			"club_id": clubID,
		})
		return 0, err
	}

	return count, nil
}

// GetRecentNotifications gets most recent notifications with optional filters
func (r *Repository) GetRecentNotifications(ctx context.Context, clubID uint, hours int, limit int) ([]models.Notification, error) {
	var notifications []models.Notification
	since := time.Now().Add(time.Duration(-hours) * time.Hour)

	query := r.db.WithContext(ctx).
		Where("club_id = ? AND created_at >= ?", clubID, since).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&notifications).Error; err != nil {
		r.logger.Error("Failed to get recent notifications", map[string]interface{}{
			"error":   err.Error(),
			"club_id": clubID,
			"hours":   hours,
		})
		return nil, err
	}

	return notifications, nil
}