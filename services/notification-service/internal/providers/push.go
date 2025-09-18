package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"reciprocal-clubs-backend/pkg/shared/logging"
)

// PushProvider handles push notifications via Firebase Cloud Messaging (FCM)
type PushProvider struct {
	serverKey   string
	projectID   string
	baseURL     string
	httpClient  *http.Client
	logger      logging.Logger
}

// NewPushProvider creates a new push notification provider
func NewPushProvider(serverKey, projectID string, logger logging.Logger) *PushProvider {
	return &PushProvider{
		serverKey:  serverKey,
		projectID:  projectID,
		baseURL:    "https://fcm.googleapis.com/fcm/send",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// FCMMessage represents a Firebase Cloud Messaging message
type FCMMessage struct {
	To           string                 `json:"to,omitempty"`
	RegistrationIDs []string            `json:"registration_ids,omitempty"`
	Data         map[string]string      `json:"data,omitempty"`
	Notification FCMNotification        `json:"notification"`
	Android      *FCMAndroidConfig      `json:"android,omitempty"`
	APNS         *FCMAPNSConfig         `json:"apns,omitempty"`
	Priority     string                 `json:"priority,omitempty"`
	TimeToLive   int                    `json:"time_to_live,omitempty"`
}

// FCMNotification represents the notification payload
type FCMNotification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Icon  string `json:"icon,omitempty"`
	Sound string `json:"sound,omitempty"`
	Badge string `json:"badge,omitempty"`
	Tag   string `json:"tag,omitempty"`
	Color string `json:"color,omitempty"`
}

// FCMAndroidConfig represents Android-specific configuration
type FCMAndroidConfig struct {
	Priority     string                    `json:"priority,omitempty"`
	TTL          string                    `json:"ttl,omitempty"`
	Notification *FCMAndroidNotification   `json:"notification,omitempty"`
}

// FCMAndroidNotification represents Android notification settings
type FCMAndroidNotification struct {
	Icon        string   `json:"icon,omitempty"`
	Color       string   `json:"color,omitempty"`
	Sound       string   `json:"sound,omitempty"`
	Tag         string   `json:"tag,omitempty"`
	ClickAction string   `json:"click_action,omitempty"`
	BodyLocKey  string   `json:"body_loc_key,omitempty"`
	BodyLocArgs []string `json:"body_loc_args,omitempty"`
	TitleLocKey string   `json:"title_loc_key,omitempty"`
	TitleLocArgs []string `json:"title_loc_args,omitempty"`
}

// FCMAPNSConfig represents iOS-specific configuration
type FCMAPNSConfig struct {
	Headers map[string]string `json:"headers,omitempty"`
	Payload FCMAPNSPayload    `json:"payload"`
}

// FCMAPNSPayload represents iOS notification payload
type FCMAPNSPayload struct {
	APS FCMAPSData `json:"aps"`
}

// FCMAPSData represents iOS APS data
type FCMAPSData struct {
	Alert            interface{} `json:"alert,omitempty"`
	Badge            int         `json:"badge,omitempty"`
	Sound            string      `json:"sound,omitempty"`
	ContentAvailable int         `json:"content-available,omitempty"`
	Category         string      `json:"category,omitempty"`
}

// FCMResponse represents FCM API response
type FCMResponse struct {
	MulticastID  int64       `json:"multicast_id"`
	Success      int         `json:"success"`
	Failure      int         `json:"failure"`
	CanonicalIDs int         `json:"canonical_ids"`
	Results      []FCMResult `json:"results"`
}

// FCMResult represents individual message result
type FCMResult struct {
	MessageID      string `json:"message_id,omitempty"`
	RegistrationID string `json:"registration_id,omitempty"`
	Error          string `json:"error,omitempty"`
}

// SendPush sends a push notification via FCM
func (p *PushProvider) SendPush(ctx context.Context, deviceToken, title, body string, metadata map[string]string) error {
	// Validate inputs
	if deviceToken == "" {
		return fmt.Errorf("device token is required")
	}
	if title == "" {
		return fmt.Errorf("notification title is required")
	}
	if body == "" {
		return fmt.Errorf("notification body is required")
	}

	// Build FCM message
	message := FCMMessage{
		To: deviceToken,
		Notification: FCMNotification{
			Title: title,
			Body:  body,
		},
		Priority:   "high",
		TimeToLive: 3600, // 1 hour
	}

	// Add metadata as data payload
	if metadata != nil {
		message.Data = metadata
	}

	// Add platform-specific configurations from metadata
	if metadata != nil {
		p.addPlatformSpecificConfig(&message, metadata)
	}

	// Convert to JSON
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal FCM message: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "key="+p.serverKey)

	// Send request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		p.logger.Error("Failed to send push notification request", map[string]interface{}{
			"error":        err.Error(),
			"device_token": deviceToken,
		})
		return fmt.Errorf("failed to send push request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var fcmResp FCMResponse
	if err := json.NewDecoder(resp.Body).Decode(&fcmResp); err != nil {
		return fmt.Errorf("failed to parse FCM response: %w", err)
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		p.logger.Error("FCM API error", map[string]interface{}{
			"status_code":  resp.StatusCode,
			"device_token": deviceToken,
			"response":     fcmResp,
		})
		return fmt.Errorf("FCM API error: status %d", resp.StatusCode)
	}

	// Check individual message results
	if fcmResp.Failure > 0 && len(fcmResp.Results) > 0 {
		result := fcmResp.Results[0]
		if result.Error != "" {
			p.logger.Error("FCM message delivery failed", map[string]interface{}{
				"error":        result.Error,
				"device_token": deviceToken,
			})
			return fmt.Errorf("FCM delivery failed: %s", result.Error)
		}
	}

	p.logger.Info("Push notification sent successfully", map[string]interface{}{
		"device_token":  deviceToken,
		"message_id":    fcmResp.Results[0].MessageID,
		"multicast_id":  fcmResp.MulticastID,
	})

	return nil
}

// addPlatformSpecificConfig adds iOS and Android specific configurations
func (p *PushProvider) addPlatformSpecificConfig(message *FCMMessage, metadata map[string]string) {
	// Android configuration
	if icon, ok := metadata["android_icon"]; ok {
		if message.Android == nil {
			message.Android = &FCMAndroidConfig{}
		}
		if message.Android.Notification == nil {
			message.Android.Notification = &FCMAndroidNotification{}
		}
		message.Android.Notification.Icon = icon
	}

	if color, ok := metadata["android_color"]; ok {
		if message.Android == nil {
			message.Android = &FCMAndroidConfig{}
		}
		if message.Android.Notification == nil {
			message.Android.Notification = &FCMAndroidNotification{}
		}
		message.Android.Notification.Color = color
	}

	if sound, ok := metadata["android_sound"]; ok {
		if message.Android == nil {
			message.Android = &FCMAndroidConfig{}
		}
		if message.Android.Notification == nil {
			message.Android.Notification = &FCMAndroidNotification{}
		}
		message.Android.Notification.Sound = sound
	}

	// iOS configuration
	if sound, ok := metadata["ios_sound"]; ok {
		if message.APNS == nil {
			message.APNS = &FCMAPNSConfig{}
		}
		message.APNS.Payload.APS.Sound = sound
	}

	if category, ok := metadata["ios_category"]; ok {
		if message.APNS == nil {
			message.APNS = &FCMAPNSConfig{}
		}
		message.APNS.Payload.APS.Category = category
	}

	// Common configurations
	if icon, ok := metadata["icon"]; ok {
		message.Notification.Icon = icon
	}

	if sound, ok := metadata["sound"]; ok {
		message.Notification.Sound = sound
	}

	if color, ok := metadata["color"]; ok {
		message.Notification.Color = color
	}
}

// SendMulticast sends a push notification to multiple devices
func (p *PushProvider) SendMulticast(ctx context.Context, deviceTokens []string, title, body string, metadata map[string]string) error {
	if len(deviceTokens) == 0 {
		return fmt.Errorf("at least one device token is required")
	}

	message := FCMMessage{
		RegistrationIDs: deviceTokens,
		Notification: FCMNotification{
			Title: title,
			Body:  body,
		},
		Priority:   "high",
		TimeToLive: 3600,
	}

	if metadata != nil {
		message.Data = metadata
		p.addPlatformSpecificConfig(&message, metadata)
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal FCM message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "key="+p.serverKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send multicast push: %w", err)
	}
	defer resp.Body.Close()

	var fcmResp FCMResponse
	if err := json.NewDecoder(resp.Body).Decode(&fcmResp); err != nil {
		return fmt.Errorf("failed to parse FCM response: %w", err)
	}

	p.logger.Info("Multicast push notification sent", map[string]interface{}{
		"device_count":  len(deviceTokens),
		"success_count": fcmResp.Success,
		"failure_count": fcmResp.Failure,
		"multicast_id":  fcmResp.MulticastID,
	})

	return nil
}

// ValidateConfig validates the push provider configuration
func (p *PushProvider) ValidateConfig() error {
	if p.serverKey == "" {
		return fmt.Errorf("FCM server key is required")
	}
	if p.projectID == "" {
		return fmt.Errorf("FCM project ID is required")
	}
	return nil
}

// TestConnection tests the FCM API connection
func (p *PushProvider) TestConnection(ctx context.Context) error {
	// Create a test message (won't be delivered due to invalid token)
	testMessage := FCMMessage{
		To: "test_token_for_validation",
		Notification: FCMNotification{
			Title: "Test",
			Body:  "Test connection",
		},
		// Note: FCM v1 API doesn't have DryRun in the message body
		// We'll just use invalid token which will fail gracefully
	}

	jsonData, err := json.Marshal(testMessage)
	if err != nil {
		return fmt.Errorf("failed to create test message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create test request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "key="+p.serverKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to FCM API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return fmt.Errorf("FCM authentication failed: invalid server key")
	}

	return nil
}