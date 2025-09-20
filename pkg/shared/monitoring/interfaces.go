package monitoring

import (
	"context"
	"net/http"
	"time"
)

// MonitoringInterface defines the interface for monitoring functionality
type MonitoringInterface interface {
	// HTTP metrics
	RecordHTTPRequest(method, endpoint string, statusCode int, duration time.Duration)

	// gRPC metrics
	RecordGRPCRequest(method, status string, duration time.Duration)

	// Business metrics
	RecordBusinessEvent(eventType, clubID string)

	// Database metrics
	RecordDatabaseConnections(count int)
	RecordActiveConnections(count int)

	// Message bus metrics
	RecordMessageReceived(subject string)
	RecordMessagePublished(subject string)

	// Health checking
	RegisterHealthCheck(checker HealthChecker)
	GetSystemHealth(ctx context.Context) *SystemHealth

	// Lifecycle
	UpdateServiceUptime()

	// Handler for metrics endpoint
	GetMetricsHandler() http.Handler
}

// Ensure Monitor implements MonitoringInterface
var _ MonitoringInterface = (*Monitor)(nil)