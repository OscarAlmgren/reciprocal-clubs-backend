package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/messaging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	"reciprocal-clubs-backend/services/notification-service/internal/models"
	notificationmonitoring "reciprocal-clubs-backend/services/notification-service/internal/monitoring"
	"reciprocal-clubs-backend/services/notification-service/internal/providers"
	"reciprocal-clubs-backend/services/notification-service/internal/repository"
)

// NotificationService handles business logic for notifications
type NotificationService struct {
	repo       *repository.Repository
	providers  *providers.NotificationProviders
	logger     logging.Logger
	messaging  messaging.MessageBus
	monitoring monitoring.MonitoringInterface
	metrics    *notificationmonitoring.NotificationMetrics
	health     *notificationmonitoring.HealthChecker
}

// NewService creates a new notification service
func NewService(repo *repository.Repository, providers *providers.NotificationProviders, logger logging.Logger, messaging messaging.MessageBus, monitoring monitoring.MonitoringInterface) *NotificationService {
	metrics := notificationmonitoring.NewNotificationMetrics(logger)
	health := notificationmonitoring.NewHealthChecker(repo.GetDB(), providers, logger)

	return &NotificationService{
		repo:       repo,
		providers:  providers,
		logger:     logger,
		messaging:  messaging,
		monitoring: monitoring,
		metrics:    metrics,
		health:     health,
	}
}

// Notification operations

// CreateNotification creates a new notification
func (s *NotificationService) CreateNotification(ctx context.Context, req *CreateNotificationRequest) (*models.Notification, error) {
	// Validate request
	if err := s.validateCreateNotificationRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

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
		s.metrics.RecordNotificationFailed(fmt.Sprintf("%d", req.ClubID), string(req.Type), "repository", "create_error")
		return nil, err
	}

	s.metrics.RecordNotificationCreated(fmt.Sprintf("%d", req.ClubID), string(req.Type), string(req.Priority))

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
		// Note: Using "unknown" for club_id as we don't have it when get fails
		s.metrics.RecordNotificationFailed("unknown", "unknown", "repository", "get_error")
		return nil, err
	}

	return notification, nil
}

// GetNotificationsByClub retrieves notifications for a club
func (s *NotificationService) GetNotificationsByClub(ctx context.Context, clubID uint, limit, offset int) ([]models.Notification, error) {
	notifications, err := s.repo.GetNotificationsByClub(ctx, clubID, limit, offset)
	if err != nil {
		s.metrics.RecordNotificationFailed(fmt.Sprintf("%d", clubID), "unknown", "repository", "get_club_error")
		return nil, err
	}

	return notifications, nil
}

// GetNotificationsByUser retrieves notifications for a user
func (s *NotificationService) GetNotificationsByUser(ctx context.Context, userID string, clubID uint, limit, offset int) ([]models.Notification, error) {
	notifications, err := s.repo.GetNotificationsByUser(ctx, userID, clubID, limit, offset)
	if err != nil {
		s.metrics.RecordNotificationFailed(fmt.Sprintf("%d", clubID), "unknown", "repository", "get_user_error")
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
		s.metrics.RecordNotificationFailed(fmt.Sprintf("%d", notification.ClubID), string(notification.Type), "repository", "update_error")
		return nil, err
	}

	s.metrics.RecordNotificationRead(fmt.Sprintf("%d", notification.ClubID), string(notification.Type))

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

// ProcessNotification processes a specific notification by ID (public method for gRPC)
func (s *NotificationService) ProcessNotification(ctx context.Context, id uint) error {
	notification, err := s.repo.GetNotificationByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get notification %d: %w", id, err)
	}

	go s.processNotification(context.Background(), notification)
	return nil
}

// ProcessScheduledNotifications processes all pending notifications and returns count
func (s *NotificationService) ProcessScheduledNotifications(ctx context.Context) int {
	notifications, err := s.repo.GetPendingNotifications(ctx, 100)
	if err != nil {
		s.logger.Error("Failed to get pending notifications", map[string]interface{}{
			"error": err.Error(),
		})
		return 0
	}

	for _, notification := range notifications {
		go s.processNotification(context.Background(), &notification)
	}

	s.logger.Info("Processed scheduled notifications", map[string]interface{}{
		"count": len(notifications),
	})

	return len(notifications)
}

// RetryFailedNotifications with count return for monitoring
func (s *NotificationService) RetryFailedNotificationsWithCount(ctx context.Context) int {
	notifications, err := s.repo.GetFailedNotifications(ctx, 50)
	if err != nil {
		s.logger.Error("Failed to get failed notifications", map[string]interface{}{
			"error": err.Error(),
		})
		return 0
	}

	for _, notification := range notifications {
		go s.processNotification(context.Background(), &notification)
	}

	s.logger.Info("Retried failed notifications", map[string]interface{}{
		"count": len(notifications),
	})

	return len(notifications)
}

// processNotification handles the actual delivery of a notification
func (s *NotificationService) processNotification(ctx context.Context, notification *models.Notification) {
	s.logger.Info("Processing notification", map[string]interface{}{
		"notification_id": notification.ID,
		"type":            notification.Type,
		"recipient":       notification.Recipient,
	})

	startTime := time.Now()
	providerName := s.getProviderName(notification.Type)
	s.metrics.RecordDeliveryAttempt(string(notification.Type), providerName, 1)

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

	duration := time.Since(startTime)
	clubID := fmt.Sprintf("%d", notification.ClubID)
	notificationType := string(notification.Type)

	if err != nil {
		notification.MarkAsFailed(err.Error())
		s.metrics.RecordNotificationFailed(clubID, notificationType, providerName, "delivery_error")
		s.metrics.RecordDeliveryDuration(notificationType, providerName, "failed", duration)
		s.logger.Error("Failed to send notification", map[string]interface{}{
			"error":           err.Error(),
			"notification_id": notification.ID,
			"type":            notification.Type,
			"duration_ms":     duration.Milliseconds(),
		})
	} else {
		notification.MarkAsSent()
		s.metrics.RecordNotificationSent(clubID, notificationType, providerName)
		s.metrics.RecordDeliveryDuration(notificationType, providerName, "success", duration)
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
		s.metrics.RecordNotificationFailed(fmt.Sprintf("%d", req.ClubID), string(req.Type), "repository", "template_create_error")
		return nil, err
	}

	s.metrics.RecordTemplateUsage(fmt.Sprintf("%d", req.ClubID), req.Name, string(req.Type))

	return template, nil
}

// GetNotificationTemplatesByClub retrieves templates for a club
func (s *NotificationService) GetNotificationTemplatesByClub(ctx context.Context, clubID uint) ([]models.NotificationTemplate, error) {
	templates, err := s.repo.GetNotificationTemplatesByClub(ctx, clubID)
	if err != nil {
		s.metrics.RecordNotificationFailed(fmt.Sprintf("%d", clubID), "unknown", "repository", "templates_get_error")
		return nil, err
	}

	return templates, nil
}

// GetNotificationStats retrieves notification statistics
func (s *NotificationService) GetNotificationStats(ctx context.Context, clubID uint, fromDate, toDate time.Time) (map[string]interface{}, error) {
	stats, err := s.repo.GetNotificationStats(ctx, clubID, fromDate, toDate)
	if err != nil {
		s.metrics.RecordNotificationFailed(fmt.Sprintf("%d", clubID), "unknown", "repository", "stats_get_error")
		return nil, err
	}

	return stats, nil
}

// Notification delivery methods (stubs - implement with actual providers)

func (s *NotificationService) sendEmail(ctx context.Context, notification *models.Notification) error {
	if s.providers.Email == nil {
		return fmt.Errorf("email provider not configured")
	}

	// Parse metadata for additional email configuration
	metadata := make(map[string]string)
	if notification.Metadata != "" {
		var metaMap map[string]string
		if err := json.Unmarshal([]byte(notification.Metadata), &metaMap); err == nil {
			metadata = metaMap
		}
	}

	// Send email via provider
	err := s.providers.Email.SendEmail(ctx, notification.Recipient, notification.Subject, notification.Message, metadata)
	if err != nil {
		s.logger.Error("Failed to send email", map[string]interface{}{
			"error":           err.Error(),
			"notification_id": notification.ID,
			"recipient":       notification.Recipient,
		})
		return err
	}

	s.logger.Info("Email sent successfully", map[string]interface{}{
		"notification_id": notification.ID,
		"recipient":       notification.Recipient,
	})

	return nil
}

func (s *NotificationService) sendSMS(ctx context.Context, notification *models.Notification) error {
	if s.providers.SMS == nil {
		return fmt.Errorf("SMS provider not configured")
	}

	// Parse metadata
	metadata := make(map[string]string)
	if notification.Metadata != "" {
		var metaMap map[string]string
		if err := json.Unmarshal([]byte(notification.Metadata), &metaMap); err == nil {
			metadata = metaMap
		}
	}

	// For SMS, we use the message content (subject + message combined if needed)
	body := notification.Message
	if notification.Subject != "" && notification.Subject != notification.Message {
		body = notification.Subject + ": " + notification.Message
	}

	// Send SMS via provider
	err := s.providers.SMS.SendSMS(ctx, notification.Recipient, body, metadata)
	if err != nil {
		s.logger.Error("Failed to send SMS", map[string]interface{}{
			"error":           err.Error(),
			"notification_id": notification.ID,
			"recipient":       notification.Recipient,
		})
		return err
	}

	s.logger.Info("SMS sent successfully", map[string]interface{}{
		"notification_id": notification.ID,
		"recipient":       notification.Recipient,
	})

	return nil
}

func (s *NotificationService) sendPush(ctx context.Context, notification *models.Notification) error {
	if s.providers.Push == nil {
		return fmt.Errorf("push notification provider not configured")
	}

	// Parse metadata
	metadata := make(map[string]string)
	if notification.Metadata != "" {
		var metaMap map[string]string
		if err := json.Unmarshal([]byte(notification.Metadata), &metaMap); err == nil {
			metadata = metaMap
		}
	}

	// Send push notification via provider
	err := s.providers.Push.SendPush(ctx, notification.Recipient, notification.Subject, notification.Message, metadata)
	if err != nil {
		s.logger.Error("Failed to send push notification", map[string]interface{}{
			"error":           err.Error(),
			"notification_id": notification.ID,
			"recipient":       notification.Recipient,
		})
		return err
	}

	s.logger.Info("Push notification sent successfully", map[string]interface{}{
		"notification_id": notification.ID,
		"recipient":       notification.Recipient,
	})

	return nil
}

func (s *NotificationService) sendInApp(ctx context.Context, notification *models.Notification) error {
	// In-app notifications are delivered via message bus for real-time delivery
	eventData := map[string]interface{}{
		"notification_id": notification.ID,
		"club_id":         notification.ClubID,
		"user_id":         notification.UserID,
		"type":            "in_app",
		"subject":         notification.Subject,
		"message":         notification.Message,
		"recipient":       notification.Recipient,
		"metadata":        notification.Metadata,
		"timestamp":       time.Now(),
	}

	jsonData, err := json.Marshal(eventData)
	if err != nil {
		return fmt.Errorf("failed to marshal in-app notification data: %w", err)
	}

	// Publish to message bus for real-time delivery to connected clients
	subject := fmt.Sprintf("notification.in_app.%s", notification.Recipient)
	if err := s.messaging.Publish(ctx, subject, jsonData); err != nil {
		s.logger.Error("Failed to publish in-app notification", map[string]interface{}{
			"error":           err.Error(),
			"notification_id": notification.ID,
			"recipient":       notification.Recipient,
		})
		return err
	}

	s.logger.Info("In-app notification published successfully", map[string]interface{}{
		"notification_id": notification.ID,
		"recipient":       notification.Recipient,
		"subject":         subject,
	})

	return nil
}

func (s *NotificationService) sendWebhook(ctx context.Context, notification *models.Notification) error {
	if s.providers.Webhook == nil {
		return fmt.Errorf("webhook provider not configured")
	}

	// Parse metadata
	metadata := make(map[string]string)
	if notification.Metadata != "" {
		var metaMap map[string]string
		if err := json.Unmarshal([]byte(notification.Metadata), &metaMap); err == nil {
			metadata = metaMap
		}
	}

	// Send webhook via provider
	err := s.providers.Webhook.SendWebhook(
		ctx,
		notification.Recipient, // URL for webhook
		fmt.Sprintf("%d", notification.ID),
		notification.Subject,
		notification.Message,
		metadata,
	)
	if err != nil {
		s.logger.Error("Failed to send webhook", map[string]interface{}{
			"error":           err.Error(),
			"notification_id": notification.ID,
			"webhook_url":     notification.Recipient,
		})
		return err
	}

	s.logger.Info("Webhook sent successfully", map[string]interface{}{
		"notification_id": notification.ID,
		"webhook_url":     notification.Recipient,
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

// Validation methods

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	phoneRegex = regexp.MustCompile(`^\+?[1-9]\d{1,14}$`) // E.164 format
)

// validateCreateNotificationRequest validates the create notification request
func (s *NotificationService) validateCreateNotificationRequest(req *CreateNotificationRequest) error {
	if req.ClubID == 0 {
		return fmt.Errorf("club_id is required")
	}

	if req.UserID != nil && strings.TrimSpace(*req.UserID) == "" {
		return fmt.Errorf("user_id cannot be empty when provided")
	}

	if strings.TrimSpace(req.Subject) == "" {
		return fmt.Errorf("subject is required")
	}

	if strings.TrimSpace(req.Message) == "" {
		return fmt.Errorf("message is required")
	}

	if strings.TrimSpace(req.Recipient) == "" {
		return fmt.Errorf("recipient is required")
	}

	// Validate recipient format based on notification type
	switch req.Type {
	case models.NotificationTypeEmail:
		if !emailRegex.MatchString(req.Recipient) {
			return fmt.Errorf("invalid email format")
		}
	case models.NotificationTypeSMS:
		if !phoneRegex.MatchString(req.Recipient) {
			return fmt.Errorf("invalid phone number format")
		}
	case models.NotificationTypePush:
		// For push notifications, recipient could be a device token
		if len(req.Recipient) < 10 {
			return fmt.Errorf("invalid device token format")
		}
	case models.NotificationTypeWebhook:
		// For webhooks, recipient should be a URL
		if !strings.HasPrefix(req.Recipient, "http://") && !strings.HasPrefix(req.Recipient, "https://") {
			return fmt.Errorf("webhook recipient must be a valid URL")
		}
	}

	return nil
}

// validateCreateTemplateRequest validates the create template request
func (s *NotificationService) validateCreateTemplateRequest(req *CreateTemplateRequest) error {
	if req.ClubID == 0 {
		return fmt.Errorf("club_id is required")
	}

	if strings.TrimSpace(req.Name) == "" {
		return fmt.Errorf("name is required")
	}

	if strings.TrimSpace(req.Body) == "" {
		return fmt.Errorf("body is required")
	}

	if strings.TrimSpace(req.CreatedByID) == "" {
		return fmt.Errorf("created_by_id is required")
	}

	return nil
}

// getProviderName returns the provider name for a given notification type
func (s *NotificationService) getProviderName(notificationType models.NotificationType) string {
	switch notificationType {
	case models.NotificationTypeEmail:
		return "smtp"
	case models.NotificationTypeSMS:
		return "twilio"
	case models.NotificationTypePush:
		return "fcm"
	case models.NotificationTypeWebhook:
		return "webhook"
	case models.NotificationTypeInApp:
		return "database"
	default:
		return "unknown"
	}
}

// GetHealthChecker returns the health checker for external use
func (s *NotificationService) GetHealthChecker() *notificationmonitoring.HealthChecker {
	return s.health
}

// GetMetrics returns the metrics instance for external use
func (s *NotificationService) GetMetrics() *notificationmonitoring.NotificationMetrics {
	return s.metrics
}