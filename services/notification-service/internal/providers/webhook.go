package providers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"reciprocal-clubs-backend/pkg/shared/logging"
)

// WebhookProvider handles webhook delivery
type WebhookProvider struct {
	secretKey  string
	httpClient *http.Client
	logger     logging.Logger
}

// NewWebhookProvider creates a new webhook provider
func NewWebhookProvider(secretKey string, logger logging.Logger) *WebhookProvider {
	return &WebhookProvider{
		secretKey: secretKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// WebhookPayload represents the webhook notification payload
type WebhookPayload struct {
	ID        string                 `json:"id"`
	Event     string                 `json:"event"`
	Timestamp int64                  `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Metadata  map[string]string      `json:"metadata,omitempty"`
}

// WebhookResponse represents webhook delivery response
type WebhookResponse struct {
	StatusCode int    `json:"status_code"`
	Body       string `json:"body"`
	Headers    map[string]string `json:"headers"`
}

// SendWebhook sends a webhook notification
func (w *WebhookProvider) SendWebhook(ctx context.Context, url, notificationID, title, body string, metadata map[string]string) error {
	// Validate inputs
	if url == "" {
		return fmt.Errorf("webhook URL is required")
	}
	if notificationID == "" {
		return fmt.Errorf("notification ID is required")
	}

	// Create webhook payload
	payload := WebhookPayload{
		ID:        notificationID,
		Event:     "notification.delivered",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"title":   title,
			"body":    body,
			"url":     url,
		},
		Metadata: metadata,
	}

	// Convert to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Clubland-Notifications/1.0")
	req.Header.Set("X-Webhook-Timestamp", fmt.Sprintf("%d", payload.Timestamp))

	// Add HMAC signature if secret key is provided
	if w.secretKey != "" {
		signature := w.generateSignature(jsonData)
		req.Header.Set("X-Webhook-Signature", signature)
	}

	// Add custom headers from metadata
	if metadata != nil {
		for key, value := range metadata {
			if key == "webhook_header_" {
				continue // Skip processing this as it's not a valid header
			}
			// Allow custom headers with prefix
			if len(key) > 15 && key[:15] == "webhook_header_" {
				headerName := key[15:]
				req.Header.Set(headerName, value)
			}
		}
	}

	// Send request with retry logic
	var lastErr error
	maxRetries := 3

	for attempt := 1; attempt <= maxRetries; attempt++ {
		resp, err := w.httpClient.Do(req)
		if err != nil {
			lastErr = err
			w.logger.Warn("Webhook delivery attempt failed", map[string]interface{}{
				"attempt": attempt,
				"url":     url,
				"error":   err.Error(),
			})

			if attempt < maxRetries {
				// Exponential backoff: 1s, 2s, 4s
				backoff := time.Duration(1<<(attempt-1)) * time.Second
				time.Sleep(backoff)
				continue
			}
			break
		}
		defer resp.Body.Close()

		// Read response body
		var responseBody bytes.Buffer
		responseBody.ReadFrom(resp.Body)

		// Check status code
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			w.logger.Info("Webhook delivered successfully", map[string]interface{}{
				"url":         url,
				"status_code": resp.StatusCode,
				"attempt":     attempt,
			})
			return nil
		}

		// Log failed attempt
		w.logger.Warn("Webhook delivery failed", map[string]interface{}{
			"attempt":     attempt,
			"url":         url,
			"status_code": resp.StatusCode,
			"response":    responseBody.String(),
		})

		if attempt < maxRetries {
			// Don't retry on client errors (4xx)
			if resp.StatusCode >= 400 && resp.StatusCode < 500 {
				return fmt.Errorf("webhook delivery failed with client error: %d", resp.StatusCode)
			}

			// Exponential backoff for server errors (5xx)
			backoff := time.Duration(1<<(attempt-1)) * time.Second
			time.Sleep(backoff)
		} else {
			lastErr = fmt.Errorf("webhook delivery failed after %d attempts: status %d", maxRetries, resp.StatusCode)
		}
	}

	if lastErr != nil {
		w.logger.Error("Webhook delivery failed permanently", map[string]interface{}{
			"url":      url,
			"attempts": maxRetries,
			"error":    lastErr.Error(),
		})
		return lastErr
	}

	return fmt.Errorf("webhook delivery failed after %d attempts", maxRetries)
}

// generateSignature creates HMAC-SHA256 signature for webhook payload
func (w *WebhookProvider) generateSignature(payload []byte) string {
	h := hmac.New(sha256.New, []byte(w.secretKey))
	h.Write(payload)
	signature := hex.EncodeToString(h.Sum(nil))
	return "sha256=" + signature
}

// VerifySignature verifies webhook signature (useful for webhook receivers)
func (w *WebhookProvider) VerifySignature(payload []byte, signature string) bool {
	if w.secretKey == "" {
		return true // No secret configured, skip verification
	}

	expectedSignature := w.generateSignature(payload)
	return hmac.Equal([]byte(expectedSignature), []byte(signature))
}

// SendBatchWebhooks sends multiple webhooks to different URLs
func (w *WebhookProvider) SendBatchWebhooks(ctx context.Context, webhooks []WebhookDelivery) error {
	if len(webhooks) == 0 {
		return fmt.Errorf("no webhooks to send")
	}

	// Send webhooks concurrently
	type result struct {
		url   string
		error error
	}

	results := make(chan result, len(webhooks))

	for _, webhook := range webhooks {
		go func(wh WebhookDelivery) {
			err := w.SendWebhook(ctx, wh.URL, wh.NotificationID, wh.Title, wh.Body, wh.Metadata)
			results <- result{url: wh.URL, error: err}
		}(webhook)
	}

	// Collect results
	var errors []string
	successCount := 0

	for i := 0; i < len(webhooks); i++ {
		result := <-results
		if result.error != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", result.url, result.error))
		} else {
			successCount++
		}
	}

	w.logger.Info("Batch webhook delivery completed", map[string]interface{}{
		"total":   len(webhooks),
		"success": successCount,
		"failed":  len(errors),
	})

	if len(errors) > 0 {
		return fmt.Errorf("webhook batch delivery had %d failures: %v", len(errors), errors)
	}

	return nil
}

// WebhookDelivery represents a single webhook delivery request
type WebhookDelivery struct {
	URL            string            `json:"url"`
	NotificationID string            `json:"notification_id"`
	Title          string            `json:"title"`
	Body           string            `json:"body"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// ValidateURL validates if the URL is suitable for webhook delivery
func (w *WebhookProvider) ValidateURL(url string) error {
	if url == "" {
		return fmt.Errorf("webhook URL is required")
	}

	// Basic URL validation
	if len(url) < 10 {
		return fmt.Errorf("webhook URL is too short")
	}

	if url[:7] != "http://" && url[:8] != "https://" {
		return fmt.Errorf("webhook URL must start with http:// or https://")
	}

	// For production, you might want to add more validation:
	// - Check if URL is reachable
	// - Validate against allowed domains
	// - Check for localhost/internal IPs in production

	return nil
}

// TestWebhook sends a test webhook to verify connectivity
func (w *WebhookProvider) TestWebhook(ctx context.Context, url string) error {
	if err := w.ValidateURL(url); err != nil {
		return err
	}

	testPayload := WebhookPayload{
		ID:        "test-webhook-" + fmt.Sprintf("%d", time.Now().Unix()),
		Event:     "webhook.test",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"message": "This is a test webhook from Clubland Notifications",
			"test":    true,
		},
	}

	jsonData, err := json.Marshal(testPayload)
	if err != nil {
		return fmt.Errorf("failed to create test payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create test request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Clubland-Notifications/1.0")
	req.Header.Set("X-Webhook-Test", "true")

	if w.secretKey != "" {
		signature := w.generateSignature(jsonData)
		req.Header.Set("X-Webhook-Signature", signature)
	}

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send test webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("test webhook failed with status: %d", resp.StatusCode)
	}

	w.logger.Info("Test webhook sent successfully", map[string]interface{}{
		"url":         url,
		"status_code": resp.StatusCode,
	})

	return nil
}