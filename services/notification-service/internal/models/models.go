package models

import (
	"time"
	"gorm.io/gorm"
)

// NotificationStatus represents the status of a notification
type NotificationStatus string

const (
	NotificationStatusPending   NotificationStatus = "pending"
	NotificationStatusSent      NotificationStatus = "sent"
	NotificationStatusDelivered NotificationStatus = "delivered"
	NotificationStatusFailed    NotificationStatus = "failed"
	NotificationStatusRead      NotificationStatus = "read"
)

// NotificationType represents the type of notification
type NotificationType string

const (
	NotificationTypeEmail    NotificationType = "email"
	NotificationTypeSMS      NotificationType = "sms"
	NotificationTypePush     NotificationType = "push"
	NotificationTypeInApp    NotificationType = "in_app"
	NotificationTypeWebhook  NotificationType = "webhook"
)

// NotificationPriority represents the priority level
type NotificationPriority string

const (
	NotificationPriorityLow      NotificationPriority = "low"
	NotificationPriorityNormal   NotificationPriority = "normal"
	NotificationPriorityHigh     NotificationPriority = "high"
	NotificationPritorityCritical NotificationPriority = "critical"
)

// Notification represents a notification to be sent
type Notification struct {
	ID           uint                 `json:"id" gorm:"primaryKey"`
	ClubID       uint                 `json:"club_id" gorm:"not null;index"`
	UserID       *string              `json:"user_id,omitempty" gorm:"index"`
	Type         NotificationType     `json:"type" gorm:"size:50;not null"`
	Priority     NotificationPriority `json:"priority" gorm:"size:50;default:'normal'"`
	Status       NotificationStatus   `json:"status" gorm:"size:50;default:'pending'"`
	Subject      string               `json:"subject" gorm:"size:255"`
	Message      string               `json:"message" gorm:"type:text;not null"`
	Recipient    string               `json:"recipient" gorm:"size:255;not null"`
	Metadata     string               `json:"metadata,omitempty" gorm:"type:json"`
	ScheduledFor *time.Time           `json:"scheduled_for,omitempty"`
	SentAt       *time.Time           `json:"sent_at,omitempty"`
	DeliveredAt  *time.Time           `json:"delivered_at,omitempty"`
	ReadAt       *time.Time           `json:"read_at,omitempty"`
	FailedAt     *time.Time           `json:"failed_at,omitempty"`
	ErrorMessage string               `json:"error_message,omitempty" gorm:"type:text"`
	RetryCount   int                  `json:"retry_count" gorm:"default:0"`
	CreatedAt    time.Time            `json:"created_at"`
	UpdatedAt    time.Time            `json:"updated_at"`
	DeletedAt    gorm.DeletedAt       `json:"-" gorm:"index"`
}

func (Notification) TableName() string {
	return "notifications"
}

// NotificationTemplate represents a reusable notification template
type NotificationTemplate struct {
	ID          uint             `json:"id" gorm:"primaryKey"`
	ClubID      uint             `json:"club_id" gorm:"not null;index"`
	Name        string           `json:"name" gorm:"size:255;not null"`
	Type        NotificationType `json:"type" gorm:"size:50;not null"`
	Subject     string           `json:"subject" gorm:"size:255"`
	Body        string           `json:"body" gorm:"type:text;not null"`
	Variables   string           `json:"variables,omitempty" gorm:"type:json"`
	IsActive    bool             `json:"is_active" gorm:"default:true"`
	CreatedByID string           `json:"created_by_id" gorm:"size:255"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	DeletedAt   gorm.DeletedAt   `json:"-" gorm:"index"`
}

func (NotificationTemplate) TableName() string {
	return "notification_templates"
}

// NotificationPreference represents user notification preferences
type NotificationPreference struct {
	ID              uint             `json:"id" gorm:"primaryKey"`
	ClubID          uint             `json:"club_id" gorm:"not null;index"`
	UserID          string           `json:"user_id" gorm:"size:255;not null;index"`
	NotificationType NotificationType `json:"notification_type" gorm:"size:50;not null"`
	IsEnabled       bool             `json:"is_enabled" gorm:"default:true"`
	DeliveryTime    string           `json:"delivery_time,omitempty" gorm:"size:50"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

// UserPreferences represents comprehensive user notification preferences
type UserPreferences struct {
	ID               uint       `json:"id" gorm:"primaryKey"`
	ClubID           uint       `json:"club_id" gorm:"not null;index"`
	UserID           string     `json:"user_id" gorm:"size:255;not null;index"`
	EmailEnabled     bool       `json:"email_enabled" gorm:"default:true"`
	SMSEnabled       bool       `json:"sms_enabled" gorm:"default:true"`
	PushEnabled      bool       `json:"push_enabled" gorm:"default:true"`
	InAppEnabled     bool       `json:"in_app_enabled" gorm:"default:true"`
	BlockedTypes     string     `json:"blocked_types,omitempty" gorm:"type:json"`
	Timezone         string     `json:"timezone" gorm:"size:50;default:'UTC'"`
	PreferredLang    string     `json:"preferred_lang" gorm:"size:10;default:'en'"`
	QuietHoursStart  *time.Time `json:"quiet_hours_start,omitempty"`
	QuietHoursEnd    *time.Time `json:"quiet_hours_end,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (NotificationPreference) TableName() string {
	return "notification_preferences"
}

func (UserPreferences) TableName() string {
	return "user_preferences"
}

// IsScheduled checks if the notification is scheduled for future delivery
func (n *Notification) IsScheduled() bool {
	return n.ScheduledFor != nil && n.ScheduledFor.After(time.Now())
}

// CanRetry checks if the notification can be retried
func (n *Notification) CanRetry() bool {
	return n.Status == NotificationStatusFailed && n.RetryCount < 3
}

// MarkAsSent updates the notification status to sent
func (n *Notification) MarkAsSent() {
	n.Status = NotificationStatusSent
	now := time.Now()
	n.SentAt = &now
}

// MarkAsDelivered updates the notification status to delivered
func (n *Notification) MarkAsDelivered() {
	n.Status = NotificationStatusDelivered
	now := time.Now()
	n.DeliveredAt = &now
}

// MarkAsRead updates the notification status to read
func (n *Notification) MarkAsRead() {
	n.Status = NotificationStatusRead
	now := time.Now()
	n.ReadAt = &now
}

// MarkAsFailed updates the notification status to failed
func (n *Notification) MarkAsFailed(errorMsg string) {
	n.Status = NotificationStatusFailed
	n.ErrorMessage = errorMsg
	now := time.Now()
	n.FailedAt = &now
	n.RetryCount++
}
