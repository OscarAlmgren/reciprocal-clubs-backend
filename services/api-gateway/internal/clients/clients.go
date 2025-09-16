package clients

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"reciprocal-clubs-backend/pkg/shared/config"
	"reciprocal-clubs-backend/pkg/shared/logging"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
}

// NewServiceClients creates and initializes all service clients
func NewServiceClients(cfg *config.Config, logger logging.Logger) (*ServiceClients, error) {
	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: time.Duration(cfg.Service.Timeout) * time.Second,
	}

	clients := &ServiceClients{
		httpClient: httpClient,
		logger:     logger,
	}

	// Initialize service clients
	if err := clients.initializeClients(cfg); err != nil {
		return nil, fmt.Errorf("failed to initialize service clients: %w", err)
	}

	logger.Info("Service clients initialized successfully", nil)
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
			return fmt.Errorf("%s service health check failed: %w", service.name, err)
		}
	}

	return nil
}

// Service client interfaces and implementations

// AuthServiceClient interface
type AuthServiceClient interface {
	Close() error
	HealthCheck(ctx context.Context) error
	// Add specific auth service methods here
}

// authServiceClient implementation
type authServiceClient struct {
	conn   *grpc.ClientConn
	logger logging.Logger
}

func NewAuthServiceClient(cfg *config.Config, logger logging.Logger) (AuthServiceClient, error) {
	// In a real implementation, get the service address from config
	address := "localhost:9091" // auth-service gRPC port

	conn, err := grpc.Dial(address, 
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to auth service: %w", err)
	}

	return &authServiceClient{
		conn:   conn,
		logger: logger,
	}, nil
}

func (c *authServiceClient) Close() error {
	return c.conn.Close()
}

func (c *authServiceClient) HealthCheck(ctx context.Context) error {
	// In a real implementation, call a health check gRPC method
	// For now, just check connection state
	state := c.conn.GetState()
	if state.String() != "READY" && state.String() != "IDLE" {
		return fmt.Errorf("auth service connection not ready: %s", state)
	}
	return nil
}

// MemberServiceClient interface
type MemberServiceClient interface {
	Close() error
	HealthCheck(ctx context.Context) error
	// Add specific member service methods here
}

type memberServiceClient struct {
	conn   *grpc.ClientConn
	logger logging.Logger
}

func NewMemberServiceClient(cfg *config.Config, logger logging.Logger) (MemberServiceClient, error) {
	address := "localhost:9092" // member-service gRPC port

	conn, err := grpc.Dial(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to member service: %w", err)
	}

	return &memberServiceClient{
		conn:   conn,
		logger: logger,
	}, nil
}

func (c *memberServiceClient) Close() error {
	return c.conn.Close()
}

func (c *memberServiceClient) HealthCheck(ctx context.Context) error {
	state := c.conn.GetState()
	if state.String() != "READY" && state.String() != "IDLE" {
		return fmt.Errorf("member service connection not ready: %s", state)
	}
	return nil
}

// ReciprocalServiceClient interface
type ReciprocalServiceClient interface {
	Close() error
	HealthCheck(ctx context.Context) error
	// Add specific reciprocal service methods here
}

type reciprocalServiceClient struct {
	conn   *grpc.ClientConn
	logger logging.Logger
}

func NewReciprocalServiceClient(cfg *config.Config, logger logging.Logger) (ReciprocalServiceClient, error) {
	address := "localhost:9093" // reciprocal-service gRPC port

	conn, err := grpc.Dial(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to reciprocal service: %w", err)
	}

	return &reciprocalServiceClient{
		conn:   conn,
		logger: logger,
	}, nil
}

func (c *reciprocalServiceClient) Close() error {
	return c.conn.Close()
}

func (c *reciprocalServiceClient) HealthCheck(ctx context.Context) error {
	state := c.conn.GetState()
	if state.String() != "READY" && state.String() != "IDLE" {
		return fmt.Errorf("reciprocal service connection not ready: %s", state)
	}
	return nil
}

// BlockchainServiceClient interface
type BlockchainServiceClient interface {
	Close() error
	HealthCheck(ctx context.Context) error
	// Add specific blockchain service methods here
}

type blockchainServiceClient struct {
	conn   *grpc.ClientConn
	logger logging.Logger
}

func NewBlockchainServiceClient(cfg *config.Config, logger logging.Logger) (BlockchainServiceClient, error) {
	address := "localhost:9094" // blockchain-service gRPC port

	conn, err := grpc.Dial(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to blockchain service: %w", err)
	}

	return &blockchainServiceClient{
		conn:   conn,
		logger: logger,
	}, nil
}

func (c *blockchainServiceClient) Close() error {
	return c.conn.Close()
}

func (c *blockchainServiceClient) HealthCheck(ctx context.Context) error {
	state := c.conn.GetState()
	if state.String() != "READY" && state.String() != "IDLE" {
		return fmt.Errorf("blockchain service connection not ready: %s", state)
	}
	return nil
}

// NotificationServiceClient interface
type NotificationServiceClient interface {
	Close() error
	HealthCheck(ctx context.Context) error
	// Add specific notification service methods here
}

type notificationServiceClient struct {
	conn   *grpc.ClientConn
	logger logging.Logger
}

func NewNotificationServiceClient(cfg *config.Config, logger logging.Logger) (NotificationServiceClient, error) {
	address := "localhost:9095" // notification-service gRPC port

	conn, err := grpc.Dial(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to notification service: %w", err)
	}

	return &notificationServiceClient{
		conn:   conn,
		logger: logger,
	}, nil
}

func (c *notificationServiceClient) Close() error {
	return c.conn.Close()
}

func (c *notificationServiceClient) HealthCheck(ctx context.Context) error {
	state := c.conn.GetState()
	if state.String() != "READY" && state.String() != "IDLE" {
		return fmt.Errorf("notification service connection not ready: %s", state)
	}
	return nil
}

// AnalyticsServiceClient interface
type AnalyticsServiceClient interface {
	Close() error
	HealthCheck(ctx context.Context) error
	// Add specific analytics service methods here
}

type analyticsServiceClient struct {
	conn   *grpc.ClientConn
	logger logging.Logger
}

func NewAnalyticsServiceClient(cfg *config.Config, logger logging.Logger) (AnalyticsServiceClient, error) {
	address := "localhost:9096" // analytics-service gRPC port

	conn, err := grpc.Dial(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to analytics service: %w", err)
	}

	return &analyticsServiceClient{
		conn:   conn,
		logger: logger,
	}, nil
}

func (c *analyticsServiceClient) Close() error {
	return c.conn.Close()
}

func (c *analyticsServiceClient) HealthCheck(ctx context.Context) error {
	state := c.conn.GetState()
	if state.String() != "READY" && state.String() != "IDLE" {
		return fmt.Errorf("analytics service connection not ready: %s", state)
	}
	return nil
}

// GovernanceServiceClient interface
type GovernanceServiceClient interface {
	Close() error
	HealthCheck(ctx context.Context) error
	// Add specific governance service methods here
}

type governanceServiceClient struct {
	conn   *grpc.ClientConn
	logger logging.Logger
}

func NewGovernanceServiceClient(cfg *config.Config, logger logging.Logger) (GovernanceServiceClient, error) {
	address := "localhost:9097" // governance-service gRPC port

	conn, err := grpc.Dial(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to governance service: %w", err)
	}

	return &governanceServiceClient{
		conn:   conn,
		logger: logger,
	}, nil
}

func (c *governanceServiceClient) Close() error {
	return c.conn.Close()
}

func (c *governanceServiceClient) HealthCheck(ctx context.Context) error {
	state := c.conn.GetState()
	if state.String() != "READY" && state.String() != "IDLE" {
		return fmt.Errorf("governance service connection not ready: %s", state)
	}
	return nil
}