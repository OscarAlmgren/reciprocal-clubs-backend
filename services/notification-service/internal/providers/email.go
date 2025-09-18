package providers

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"

	"reciprocal-clubs-backend/pkg/shared/logging"
)

// EmailProvider handles email delivery
type EmailProvider struct {
	smtpHost     string
	smtpPort     string
	smtpUsername string
	smtpPassword string
	fromEmail    string
	logger       logging.Logger
}

// NewEmailProvider creates a new email provider
func NewEmailProvider(smtpHost, smtpPort, smtpUsername, smtpPassword, fromEmail string, logger logging.Logger) *EmailProvider {
	return &EmailProvider{
		smtpHost:     smtpHost,
		smtpPort:     smtpPort,
		smtpUsername: smtpUsername,
		smtpPassword: smtpPassword,
		fromEmail:    fromEmail,
		logger:       logger,
	}
}

// SendEmail sends an email notification
func (e *EmailProvider) SendEmail(ctx context.Context, to, subject, body string, metadata map[string]string) error {
	// Validate inputs
	if to == "" {
		return fmt.Errorf("recipient email is required")
	}
	if subject == "" {
		return fmt.Errorf("email subject is required")
	}
	if body == "" {
		return fmt.Errorf("email body is required")
	}

	// Set up authentication
	auth := smtp.PlainAuth("", e.smtpUsername, e.smtpPassword, e.smtpHost)

	// Compose email message
	msg := e.composeMessage(to, subject, body, metadata)

	// Send email
	addr := fmt.Sprintf("%s:%s", e.smtpHost, e.smtpPort)
	err := smtp.SendMail(addr, auth, e.fromEmail, []string{to}, []byte(msg))

	if err != nil {
		e.logger.Error("Failed to send email", map[string]interface{}{
			"error":     err.Error(),
			"recipient": to,
			"subject":   subject,
		})
		return fmt.Errorf("failed to send email: %w", err)
	}

	e.logger.Info("Email sent successfully", map[string]interface{}{
		"recipient": to,
		"subject":   subject,
	})

	return nil
}

// composeMessage creates the email message with proper headers
func (e *EmailProvider) composeMessage(to, subject, body string, metadata map[string]string) string {
	var msg strings.Builder

	// Headers
	msg.WriteString(fmt.Sprintf("From: %s\r\n", e.fromEmail))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/html; charset=utf-8\r\n")

	// Add custom headers from metadata
	if metadata != nil {
		for key, value := range metadata {
			if strings.HasPrefix(key, "header_") {
				headerName := strings.TrimPrefix(key, "header_")
				msg.WriteString(fmt.Sprintf("%s: %s\r\n", headerName, value))
			}
		}
	}

	msg.WriteString("\r\n")

	// Body (support both plain text and HTML)
	if strings.Contains(body, "<html>") || strings.Contains(body, "<div>") {
		msg.WriteString(body)
	} else {
		// Convert plain text to basic HTML
		htmlBody := strings.ReplaceAll(body, "\n", "<br>")
		msg.WriteString(fmt.Sprintf(`
<html>
<body>
%s
</body>
</html>`, htmlBody))
	}

	return msg.String()
}

// ValidateConfig validates the email provider configuration
func (e *EmailProvider) ValidateConfig() error {
	if e.smtpHost == "" {
		return fmt.Errorf("SMTP host is required")
	}
	if e.smtpPort == "" {
		return fmt.Errorf("SMTP port is required")
	}
	if e.smtpUsername == "" {
		return fmt.Errorf("SMTP username is required")
	}
	if e.smtpPassword == "" {
		return fmt.Errorf("SMTP password is required")
	}
	if e.fromEmail == "" {
		return fmt.Errorf("from email is required")
	}
	return nil
}

// TestConnection tests the SMTP connection
func (e *EmailProvider) TestConnection() error {
	addr := fmt.Sprintf("%s:%s", e.smtpHost, e.smtpPort)
	auth := smtp.PlainAuth("", e.smtpUsername, e.smtpPassword, e.smtpHost)

	// Try to connect and authenticate
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer client.Quit()

	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP authentication failed: %w", err)
	}

	return nil
}