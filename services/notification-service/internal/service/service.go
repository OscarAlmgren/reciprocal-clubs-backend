package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/messaging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	"reciprocal-clubs-backend/services/notification-service/internal/models"
	"reciprocal-clubs-backend/services/notification-service/internal/repository"
)

// NotificationService handles business logic for notifications
type NotificationService struct {
	repo       *repository.Repository
	logger     logging.Logger
	messaging  messaging.MessageBus
	monitoring *monitoring.Monitor
}

// NewService creates a new notification service
func NewService(repo *repository.Repository, logger logging.Logger, messaging messaging.MessageBus, monitoring *monitoring.Monitor) *NotificationService {
	return &NotificationService{
		repo:       repo,
		logger:     logger,
		messaging:  messaging,
		monitoring: monitoring,
	}
}

// Notification operations

// CreateNotification creates a new notification
func (s *NotificationService) CreateNotification(ctx context.Context, req *CreateNotificationRequest) (*models.Notification, error) {
	notification := &models.Notification{
		ClubID:       req.ClubID,
		UserID:       req.UserID,
		Type:         req.Type,
		Priority:     req.Priority,
		Subject:      req.Subject,
		Message:      req.Message,
		Recipient:    req.Recipient,
		Metadata:     req.Metadata,
		ScheduledFor: req.ScheduledFor,
		Status:       models.NotificationStatusPending,
	}

	if err := s.repo.CreateNotification(ctx, notification); err != nil {
		s.monitoring.RecordBusinessEvent("notification_create_error", fmt.Sprintf("%d", req.ClubID))
		return nil, err
	}

	s.monitoring.RecordBusinessEvent("notification_created", fmt.Sprintf("%d", req.ClubID))

	// Publish notification created event
	s.publishNotificationEvent(ctx, "notification.created", notification)

	s.logger.Info("Notification created", map[string]interface{}{
		"notification_id": notification.ID,
		"club_id":         notification.ClubID,
		"type":            notification.Type,
		"priority":        notification.Priority,
	})

	// If not scheduled, attempt immediate delivery
	if !notification.IsScheduled() {
		go s.processNotification(context.Background(), notification)
	}

	return notification, nil
}

// GetNotificationByID retrieves a notification by ID
func (s *NotificationService) GetNotificationByID(ctx context.Context, id uint) (*models.Notification, error) {
	notification, err := s.repo.GetNotificationByID(ctx, id)
	if err != nil {
		s.monitoring.RecordBusinessEvent("notification_get_error", "1")
		return nil, err
	}

	return notification, nil
}

// GetNotificationsByClub retrieves notifications for a club
func (s *NotificationService) GetNotificationsByClub(ctx context.Context, clubID uint, limit, offset int) ([]models.Notification, error) {
	notifications, err := s.repo.GetNotificationsByClub(ctx, clubID, limit, offset)
	if err != nil {
		s.monitoring.RecordBusinessEvent("notifications_get_error", fmt.Sprintf("%d", clubID))
		return nil, err
	}

	return notifications, nil
}

// GetNotificationsByUser retrieves notifications for a user
func (s *NotificationService) GetNotificationsByUser(ctx context.Context, userID string, clubID uint, limit, offset int) ([]models.Notification, error) {
	notifications, err := s.repo.GetNotificationsByUser(ctx, userID, clubID, limit, offset)
	if err != nil {
		s.monitoring.RecordBusinessEvent("user_notifications_get_error", fmt.Sprintf("%d", clubID))
		return nil, err
	}

	return notifications, nil
}

// MarkNotificationAsRead marks a notification as read
func (s *NotificationService) MarkNotificationAsRead(ctx context.Context, id uint) (*models.Notification, error) {
	notification, err := s.repo.GetNotificationByID(ctx, id)
	if err != nil {
		return nil, err
	}

	notification.MarkAsRead()

	if err := s.repo.UpdateNotification(ctx, notification); err != nil {
		s.monitoring.RecordBusinessEvent("notification_read_error", fmt.Sprintf("%d", notification.ClubID))
		return nil, err
	}

	s.monitoring.RecordBusinessEvent("notification_read", fmt.Sprintf("%d", notification.ClubID))

	// Publish notification read event
	s.publishNotificationEvent(ctx, "notification.read", notification)

	return notification, nil
}

// ProcessPendingNotifications processes notifications ready to be sent
func (s *NotificationService) ProcessPendingNotifications(ctx context.Context) error {
	notifications, err := s.repo.GetPendingNotifications(ctx, 100)
	if err != nil {
		return err
	}

	for _, notification := range notifications {
		go s.processNotification(context.Background(), &notification)
	}

	return nil
}

// RetryFailedNotifications retries failed notifications that can be retried
func (s *NotificationService) RetryFailedNotifications(ctx context.Context) error {
	notifications, err := s.repo.GetFailedNotifications(ctx, 50)
	if err != nil {
		return err
	}

	for _, notification := range notifications {
		go s.processNotification(context.Background(), &notification)
	}

	return nil
}

// processNotification handles the actual delivery of a notification
func (s *NotificationService) processNotification(ctx context.Context, notification *models.Notification) {
	s.logger.Info("Processing notification", map[string]interface{}{
		"notification_id": notification.ID,
		"type":            notification.Type,
		"recipient":       notification.Recipient,
	})

	var err error
	switch notification.Type {
	case models.NotificationTypeEmail:
		err = s.sendEmail(ctx, notification)
	case models.NotificationTypeSMS:
		err = s.sendSMS(ctx, notification)
	case models.NotificationTypePush:
		err = s.sendPush(ctx, notification)
	case models.NotificationTypeInApp:
		err = s.sendInApp(ctx, notification)
	case models.NotificationTypeWebhook:
		err = s.sendWebhook(ctx, notification)
	default:
		err = fmt.Errorf("unsupported notification type: %s", notification.Type)
	}

	if err != nil {
		notification.MarkAsFailed(err.Error())
		s.monitoring.RecordBusinessEvent("notification_send_failed", fmt.Sprintf("%d", notification.ClubID))
		s.logger.Error("Failed to send notification", map[string]interface{}{
			"error":           err.Error(),
			"notification_id": notification.ID,
			"type":            notification.Type,
		})
	} else {
		notification.MarkAsSent()
		s.monitoring.RecordBusinessEvent("notification_sent", fmt.Sprintf("%d", notification.ClubID))
	}

	if err := s.repo.UpdateNotification(ctx, notification); err != nil {
		s.logger.Error("Failed to update notification status", map[string]interface{}{
			"error":           err.Error(),
			"notification_id": notification.ID,
		})
	}

	// Publish notification status update event
	if err != nil {
		s.publishNotificationEvent(ctx, "notification.failed", notification)
	} else {
		s.publishNotificationEvent(ctx, "notification.sent", notification)
	}
}

// Template operations

// CreateNotificationTemplate creates a new notification template
func (s *NotificationService) CreateNotificationTemplate(ctx context.Context, req *CreateTemplateRequest) (*models.NotificationTemplate, error) {
	template := &models.NotificationTemplate{
		ClubID:      req.ClubID,
		Name:        req.Name,
		Type:        req.Type,
		Subject:     req.Subject,
		Body:        req.Body,
		Variables:   req.Variables,
		IsActive:    true,
		CreatedByID: req.CreatedByID,
	}

	if err := s.repo.CreateNotificationTemplate(ctx, template); err != nil {
		s.monitoring.RecordBusinessEvent("template_create_error", fmt.Sprintf("%d", req.ClubID))
		return nil, err
	}

	s.monitoring.RecordBusinessEvent("template_created", fmt.Sprintf("%d", req.ClubID))

	return template, nil
}

// GetNotificationTemplatesByClub retrieves templates for a club
func (s *NotificationService) GetNotificationTemplatesByClub(ctx context.Context, clubID uint) ([]models.NotificationTemplate, error) {
	templates, err := s.repo.GetNotificationTemplatesByClub(ctx, clubID)
	if err != nil {
		s.monitoring.RecordBusinessEvent("templates_get_error", fmt.Sprintf("%d", clubID))
		return nil, err
	}

	return templates, nil
}

// GetNotificationStats retrieves notification statistics
func (s *NotificationService) GetNotificationStats(ctx context.Context, clubID uint, fromDate, toDate time.Time) (map[string]interface{}, error) {
	stats, err := s.repo.GetNotificationStats(ctx, clubID, fromDate, toDate)
	if err != nil {
		s.monitoring.RecordBusinessEvent("stats_get_error", fmt.Sprintf("%d", clubID))
		return nil, err
	}

	return stats, nil
}

// Notification delivery methods (stubs - implement with actual providers)

func (s *NotificationService) sendEmail(ctx context.Context, notification *models.Notification) error {
	// Implement email sending logic here
	// This would integrate with email providers like SendGrid, AWS SES, etc.
	s.logger.Info("Sending email notification", map[string]interface{}{
		"notification_id": notification.ID,
		"recipient":       notification.Recipient,
	})
	return nil
}

func (s *NotificationService) sendSMS(ctx context.Context, notification *models.Notification) error {
	// Implement SMS sending logic here
	// This would integrate with SMS providers like Twilio, AWS SNS, etc.
	s.logger.Info("Sending SMS notification", map[string]interface{}{
		"notification_id": notification.ID,
		"recipient":       notification.Recipient,
	})
	return nil
}

func (s *NotificationService) sendPush(ctx context.Context, notification *models.Notification) error {
	// Implement push notification logic here
	// This would integrate with Firebase Cloud Messaging, etc.
	s.logger.Info("Sending push notification", map[string]interface{}{
		"notification_id": notification.ID,
		"recipient":       notification.Recipient,
	})
	return nil
}

func (s *NotificationService) sendInApp(ctx context.Context, notification *models.Notification) error {
	// Implement in-app notification logic here
	// This would publish to internal message bus for real-time delivery
	s.logger.Info("Sending in-app notification", map[string]interface{}{
		"notification_id": notification.ID,
		"recipient":       notification.Recipient,
	})
	return nil
}

func (s *NotificationService) sendWebhook(ctx context.Context, notification *models.Notification) error {
	// Implement webhook sending logic here
	// This would make HTTP requests to configured webhook URLs
	s.logger.Info("Sending webhook notification", map[string]interface{}{
		"notification_id": notification.ID,
		"recipient":       notification.Recipient,
	})
	return nil
}

// Helper methods

func (s *NotificationService) publishNotificationEvent(ctx context.Context, eventType string, notification *models.Notification) {
	data := map[string]interface{}{
		"notification_id": notification.ID,
		"club_id":         notification.ClubID,
		"user_id":         notification.UserID,
		"type":            notification.Type,
		"status":          notification.Status,
		"recipient":       notification.Recipient,
		"timestamp":       time.Now(),
	}

	jsonData, _ := json.Marshal(data)
	if err := s.messaging.Publish(ctx, eventType, jsonData); err != nil {
		s.logger.Error("Failed to publish notification event", map[string]interface{}{
			"error":           err.Error(),
			"event_type":      eventType,
			"notification_id": notification.ID,
		})
	}
}

// Request/Response types

type CreateNotificationRequest struct {
	ClubID       uint                         `json:"club_id" validate:"required"`
	UserID       *string                      `json:"user_id,omitempty"`
	Type         models.NotificationType      `json:"type" validate:"required"`
	Priority     models.NotificationPriority  `json:"priority"`
	Subject      string                       `json:"subject"`
	Message      string                       `json:"message" validate:"required"`
	Recipient    string                       `json:"recipient" validate:"required"`
	Metadata     string                       `json:"metadata,omitempty"`
	ScheduledFor *time.Time                   `json:"scheduled_for,omitempty"`
}

type CreateTemplateRequest struct {
	ClubID      uint                    `json:"club_id" validate:"required"`
	Name        string                  `json:"name" validate:"required"`
	Type        models.NotificationType `json:"type" validate:"required"`
	Subject     string                  `json:"subject"`
	Body        string                  `json:"body" validate:"required"`
	Variables   string                  `json:"variables,omitempty"`
	CreatedByID string                  `json:"created_by_id" validate:"required"`
}