package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"reciprocal-clubs-backend/pkg/shared/logging"
)

// SMSProvider handles SMS delivery via Twilio API
type SMSProvider struct {
	accountSID    string
	authToken     string
	fromNumber    string
	baseURL       string
	httpClient    *http.Client
	logger        logging.Logger
}

// NewSMSProvider creates a new SMS provider
func NewSMSProvider(accountSID, authToken, fromNumber string, logger logging.Logger) *SMSProvider {
	return &SMSProvider{
		accountSID: accountSID,
		authToken:  authToken,
		fromNumber: fromNumber,
		baseURL:    "https://api.twilio.com/2010-04-01",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// TwilioMessage represents a Twilio SMS message request
type TwilioMessage struct {
	From string `json:"From"`
	To   string `json:"To"`
	Body string `json:"Body"`
}

// TwilioResponse represents a Twilio API response
type TwilioResponse struct {
	SID         string `json:"sid"`
	Status      string `json:"status"`
	ErrorCode   *int   `json:"error_code,omitempty"`
	ErrorMessage *string `json:"error_message,omitempty"`
}

// SendSMS sends an SMS notification via Twilio
func (s *SMSProvider) SendSMS(ctx context.Context, to, body string, metadata map[string]string) error {
	// Validate inputs
	if to == "" {
		return fmt.Errorf("recipient phone number is required")
	}
	if body == "" {
		return fmt.Errorf("SMS body is required")
	}

	// Ensure phone number is in E.164 format
	to = s.normalizePhoneNumber(to)

	// Truncate message if too long (SMS limit is 160 chars for GSM, 70 for Unicode)
	if len(body) > 160 {
		body = body[:157] + "..."
		s.logger.Warn("SMS body truncated", map[string]interface{}{
			"recipient":     to,
			"original_length": len(body) + 3,
		})
	}

	// Convert to form data (Twilio expects form-encoded data)
	formData := fmt.Sprintf("From=%s&To=%s&Body=%s", s.fromNumber, to, body)

	// Create HTTP request
	url := fmt.Sprintf("%s/Accounts/%s/Messages.json", s.baseURL, s.accountSID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(formData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(s.accountSID, s.authToken)

	// Send request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Error("Failed to send SMS request", map[string]interface{}{
			"error":     err.Error(),
			"recipient": to,
		})
		return fmt.Errorf("failed to send SMS request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var twilioResp TwilioResponse
	if err := json.NewDecoder(resp.Body).Decode(&twilioResp); err != nil {
		return fmt.Errorf("failed to parse Twilio response: %w", err)
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		errorMsg := "Unknown error"
		if twilioResp.ErrorMessage != nil {
			errorMsg = *twilioResp.ErrorMessage
		}
		s.logger.Error("Twilio API error", map[string]interface{}{
			"status_code":   resp.StatusCode,
			"error_code":    twilioResp.ErrorCode,
			"error_message": errorMsg,
			"recipient":     to,
		})
		return fmt.Errorf("Twilio API error (%d): %s", resp.StatusCode, errorMsg)
	}

	s.logger.Info("SMS sent successfully", map[string]interface{}{
		"recipient":   to,
		"message_sid": twilioResp.SID,
		"status":      twilioResp.Status,
	})

	return nil
}

// normalizePhoneNumber ensures phone number is in E.164 format
func (s *SMSProvider) normalizePhoneNumber(phone string) string {
	// Remove all non-digit characters except +
	normalized := ""
	for _, char := range phone {
		if char >= '0' && char <= '9' || char == '+' {
			normalized += string(char)
		}
	}

	// Add + if not present and number doesn't start with it
	if !strings.HasPrefix(normalized, "+") {
		normalized = "+" + normalized
	}

	return normalized
}

// ValidateConfig validates the SMS provider configuration
func (s *SMSProvider) ValidateConfig() error {
	if s.accountSID == "" {
		return fmt.Errorf("Twilio Account SID is required")
	}
	if s.authToken == "" {
		return fmt.Errorf("Twilio Auth Token is required")
	}
	if s.fromNumber == "" {
		return fmt.Errorf("Twilio from number is required")
	}
	return nil
}

// TestConnection tests the Twilio API connection
func (s *SMSProvider) TestConnection(ctx context.Context) error {
	// Test by trying to fetch account details
	url := fmt.Sprintf("%s/Accounts/%s.json", s.baseURL, s.accountSID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create test request: %w", err)
	}

	req.SetBasicAuth(s.accountSID, s.authToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to Twilio API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("Twilio API authentication failed: status %d", resp.StatusCode)
	}

	return nil
}

// GetDeliveryStatus checks the delivery status of an SMS
func (s *SMSProvider) GetDeliveryStatus(ctx context.Context, messageSID string) (string, error) {
	url := fmt.Sprintf("%s/Accounts/%s/Messages/%s.json", s.baseURL, s.accountSID, messageSID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create status request: %w", err)
	}

	req.SetBasicAuth(s.accountSID, s.authToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get message status: %w", err)
	}
	defer resp.Body.Close()

	var twilioResp TwilioResponse
	if err := json.NewDecoder(resp.Body).Decode(&twilioResp); err != nil {
		return "", fmt.Errorf("failed to parse status response: %w", err)
	}

	return twilioResp.Status, nil
}