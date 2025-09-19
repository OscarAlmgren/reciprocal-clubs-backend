package clients

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"reciprocal-clubs-backend/pkg/shared/config"
	"reciprocal-clubs-backend/pkg/shared/logging"
)

// ServiceClients holds all service client connections
type ServiceClients struct {
	AuthService         AuthServiceClient
	MemberService       MemberServiceClient
	ReciprocalService   ReciprocalServiceClient
	BlockchainService   BlockchainServiceClient
	NotificationService NotificationServiceClient
	AnalyticsService    AnalyticsServiceClient
	GovernanceService   GovernanceServiceClient
	httpClient          *http.Client
	logger              logging.Logger
	config              *ServiceClientConfig
}

// NewServiceClients creates and initializes all service clients
func NewServiceClients(cfg *config.Config, logger logging.Logger) (*ServiceClients, error) {
	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: time.Duration(cfg.Service.Timeout) * time.Second,
	}

	// Get service client configuration
	clientConfig := DefaultServiceClientConfig()

	clients := &ServiceClients{
		httpClient: httpClient,
		logger:     logger,
		config:     clientConfig,
	}

	// Initialize service clients
	if err := clients.initializeClients(cfg); err != nil {
		return nil, fmt.Errorf("failed to initialize service clients: %w", err)
	}

	logger.Info("Service clients initialized successfully", map[string]interface{}{
		"auth_address":         clientConfig.AuthServiceAddress,
		"member_address":       clientConfig.MemberServiceAddress,
		"reciprocal_address":   clientConfig.ReciprocalServiceAddress,
		"blockchain_address":   clientConfig.BlockchainServiceAddress,
		"notification_address": clientConfig.NotificationServiceAddress,
		"analytics_address":    clientConfig.AnalyticsServiceAddress,
		"governance_address":   clientConfig.GovernanceServiceAddress,
	})
	return clients, nil
}

// initializeClients initializes all service client connections
func (sc *ServiceClients) initializeClients(cfg *config.Config) error {
	var err error

	// Initialize Auth Service client
	sc.AuthService, err = NewAuthServiceClient(cfg, sc.logger)
	if err != nil {
		return fmt.Errorf("failed to create auth service client: %w", err)
	}

	// Initialize Member Service client
	sc.MemberService, err = NewMemberServiceClient(cfg, sc.logger)
	if err != nil {
		return fmt.Errorf("failed to create member service client: %w", err)
	}

	// Initialize Reciprocal Service client
	sc.ReciprocalService, err = NewReciprocalServiceClient(cfg, sc.logger)
	if err != nil {
		return fmt.Errorf("failed to create reciprocal service client: %w", err)
	}

	// Initialize Blockchain Service client
	sc.BlockchainService, err = NewBlockchainServiceClient(cfg, sc.logger)
	if err != nil {
		return fmt.Errorf("failed to create blockchain service client: %w", err)
	}

	// Initialize Notification Service client
	sc.NotificationService, err = NewNotificationServiceClient(cfg, sc.logger)
	if err != nil {
		return fmt.Errorf("failed to create notification service client: %w", err)
	}

	// Initialize Analytics Service client
	sc.AnalyticsService, err = NewAnalyticsServiceClient(cfg, sc.logger)
	if err != nil {
		return fmt.Errorf("failed to create analytics service client: %w", err)
	}

	// Initialize Governance Service client
	sc.GovernanceService, err = NewGovernanceServiceClient(cfg, sc.logger)
	if err != nil {
		return fmt.Errorf("failed to create governance service client: %w", err)
	}

	return nil
}

// Close closes all service client connections
func (sc *ServiceClients) Close() error {
	var lastErr error

	if err := sc.AuthService.Close(); err != nil {
		sc.logger.Error("Error closing auth service client", map[string]interface{}{"error": err.Error()})
		lastErr = err
	}

	if err := sc.MemberService.Close(); err != nil {
		sc.logger.Error("Error closing member service client", map[string]interface{}{"error": err.Error()})
		lastErr = err
	}

	if err := sc.ReciprocalService.Close(); err != nil {
		sc.logger.Error("Error closing reciprocal service client", map[string]interface{}{"error": err.Error()})
		lastErr = err
	}

	if err := sc.BlockchainService.Close(); err != nil {
		sc.logger.Error("Error closing blockchain service client", map[string]interface{}{"error": err.Error()})
		lastErr = err
	}

	if err := sc.NotificationService.Close(); err != nil {
		sc.logger.Error("Error closing notification service client", map[string]interface{}{"error": err.Error()})
		lastErr = err
	}

	if err := sc.AnalyticsService.Close(); err != nil {
		sc.logger.Error("Error closing analytics service client", map[string]interface{}{"error": err.Error()})
		lastErr = err
	}

	if err := sc.GovernanceService.Close(); err != nil {
		sc.logger.Error("Error closing governance service client", map[string]interface{}{"error": err.Error()})
		lastErr = err
	}

	return lastErr
}

// HealthCheck checks the health of all service connections
func (sc *ServiceClients) HealthCheck(ctx context.Context) error {
	// Check each service client
	services := []struct {
		name   string
		health func(context.Context) error
	}{
		{"auth", sc.AuthService.HealthCheck},
		{"member", sc.MemberService.HealthCheck},
		{"reciprocal", sc.ReciprocalService.HealthCheck},
		{"blockchain", sc.BlockchainService.HealthCheck},
		{"notification", sc.NotificationService.HealthCheck},
		{"analytics", sc.AnalyticsService.HealthCheck},
		{"governance", sc.GovernanceService.HealthCheck},
	}

	for _, service := range services {
		if err := service.health(ctx); err != nil {
			sc.logger.Error("Service health check failed", map[string]interface{}{
				"service": service.name,
				"error":   err.Error(),
			})
			// Don't fail immediately - check all services and log issues
		}
	}

	return nil
}

// GetServiceStatus returns the connection status of all services
func (sc *ServiceClients) GetServiceStatus(ctx context.Context) map[string]bool {
	status := make(map[string]bool)

	services := []struct {
		name   string
		health func(context.Context) error
	}{
		{"auth", sc.AuthService.HealthCheck},
		{"member", sc.MemberService.HealthCheck},
		{"reciprocal", sc.ReciprocalService.HealthCheck},
		{"blockchain", sc.BlockchainService.HealthCheck},
		{"notification", sc.NotificationService.HealthCheck},
		{"analytics", sc.AnalyticsService.HealthCheck},
		{"governance", sc.GovernanceService.HealthCheck},
	}

	for _, service := range services {
		status[service.name] = service.health(ctx) == nil
	}

	return status
}

// RefreshConnections attempts to refresh all service connections
func (sc *ServiceClients) RefreshConnections(cfg *config.Config) error {
	sc.logger.Info("Refreshing service client connections", nil)

	// Close existing connections
	if err := sc.Close(); err != nil {
		sc.logger.Error("Error closing existing connections during refresh", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Reinitialize all clients
	if err := sc.initializeClients(cfg); err != nil {
		return fmt.Errorf("failed to refresh service clients: %w", err)
	}

	sc.logger.Info("Service client connections refreshed successfully", nil)
	return nil
}