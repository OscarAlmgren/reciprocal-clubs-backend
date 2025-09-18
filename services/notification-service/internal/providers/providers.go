package providers

import (
	"context"

	"reciprocal-clubs-backend/pkg/shared/logging"
)

// NotificationProviders holds all notification delivery providers
type NotificationProviders struct {
	Email   *EmailProvider
	SMS     *SMSProvider
	Push    *PushProvider
	Webhook *WebhookProvider
	logger  logging.Logger
}

// NewNotificationProviders creates a new providers instance
func NewNotificationProviders(config *ProvidersConfig, logger logging.Logger) *NotificationProviders {
	providers := &NotificationProviders{
		logger: logger,
	}

	// Initialize email provider
	if config.Email != nil {
		providers.Email = NewEmailProvider(
			config.Email.SMTPHost,
			config.Email.SMTPPort,
			config.Email.SMTPUsername,
			config.Email.SMTPPassword,
			config.Email.FromEmail,
			logger,
		)
	}

	// Initialize SMS provider
	if config.SMS != nil {
		providers.SMS = NewSMSProvider(
			config.SMS.AccountSID,
			config.SMS.AuthToken,
			config.SMS.FromNumber,
			logger,
		)
	}

	// Initialize push provider
	if config.Push != nil {
		providers.Push = NewPushProvider(
			config.Push.ServerKey,
			config.Push.ProjectID,
			logger,
		)
	}

	// Initialize webhook provider
	if config.Webhook != nil {
		providers.Webhook = NewWebhookProvider(
			config.Webhook.SecretKey,
			logger,
		)
	}

	return providers
}

// ProvidersConfig holds configuration for all providers
type ProvidersConfig struct {
	Email   *EmailConfig   `json:"email,omitempty"`
	SMS     *SMSConfig     `json:"sms,omitempty"`
	Push    *PushConfig    `json:"push,omitempty"`
	Webhook *WebhookConfig `json:"webhook,omitempty"`
}

// EmailConfig holds email provider configuration
type EmailConfig struct {
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     string `json:"smtp_port"`
	SMTPUsername string `json:"smtp_username"`
	SMTPPassword string `json:"smtp_password"`
	FromEmail    string `json:"from_email"`
}

// SMSConfig holds SMS provider configuration
type SMSConfig struct {
	AccountSID string `json:"account_sid"`
	AuthToken  string `json:"auth_token"`
	FromNumber string `json:"from_number"`
}

// PushConfig holds push notification provider configuration
type PushConfig struct {
	ServerKey string `json:"server_key"`
	ProjectID string `json:"project_id"`
}

// WebhookConfig holds webhook provider configuration
type WebhookConfig struct {
	SecretKey string `json:"secret_key"`
}

// ValidateConfig validates all provider configurations
func (np *NotificationProviders) ValidateConfig() error {
	if np.Email != nil {
		if err := np.Email.ValidateConfig(); err != nil {
			return err
		}
	}

	if np.SMS != nil {
		if err := np.SMS.ValidateConfig(); err != nil {
			return err
		}
	}

	if np.Push != nil {
		if err := np.Push.ValidateConfig(); err != nil {
			return err
		}
	}

	// Webhook provider doesn't require validation as secret key is optional

	return nil
}

// TestConnections tests connectivity to all configured providers
func (np *NotificationProviders) TestConnections(ctx context.Context) error {
	if np.Email != nil {
		if err := np.Email.TestConnection(); err != nil {
			np.logger.Error("Email provider connection test failed", map[string]interface{}{
				"error": err.Error(),
			})
			return err
		}
		np.logger.Info("Email provider connection test passed", nil)
	}

	if np.SMS != nil {
		if err := np.SMS.TestConnection(ctx); err != nil {
			np.logger.Error("SMS provider connection test failed", map[string]interface{}{
				"error": err.Error(),
			})
			return err
		}
		np.logger.Info("SMS provider connection test passed", nil)
	}

	if np.Push != nil {
		if err := np.Push.TestConnection(ctx); err != nil {
			np.logger.Error("Push provider connection test failed", map[string]interface{}{
				"error": err.Error(),
			})
			return err
		}
		np.logger.Info("Push provider connection test passed", nil)
	}

	// Webhook provider doesn't have a connection test as it's outbound only

	np.logger.Info("All provider connection tests passed", nil)
	return nil
}

// GetEnabledProviders returns a list of enabled provider types
func (np *NotificationProviders) GetEnabledProviders() []string {
	var enabled []string

	if np.Email != nil {
		enabled = append(enabled, "email")
	}
	if np.SMS != nil {
		enabled = append(enabled, "sms")
	}
	if np.Push != nil {
		enabled = append(enabled, "push")
	}
	if np.Webhook != nil {
		enabled = append(enabled, "webhook")
	}

	return enabled
}

// IsProviderEnabled checks if a specific provider type is enabled
func (np *NotificationProviders) IsProviderEnabled(providerType string) bool {
	switch providerType {
	case "email":
		return np.Email != nil
	case "sms":
		return np.SMS != nil
	case "push":
		return np.Push != nil
	case "webhook":
		return np.Webhook != nil
	default:
		return false
	}
}