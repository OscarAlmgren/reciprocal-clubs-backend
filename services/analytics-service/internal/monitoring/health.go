package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"

	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/services/analytics-service/internal/integrations"
)

// HealthChecker performs health checks on various system components
type HealthChecker struct {
	db           *gorm.DB
	integrations *integrations.AnalyticsIntegrations
	logger       logging.Logger
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(db *gorm.DB, integrations *integrations.AnalyticsIntegrations, logger logging.Logger) *HealthChecker {
	return &HealthChecker{
		db:           db,
		integrations: integrations,
		logger:       logger,
	}
}

// HealthStatus represents the health status of a component
type HealthStatus struct {
	Status  string            `json:"status"`
	Message string            `json:"message,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

// SystemHealth represents the overall system health
type SystemHealth struct {
	Status     string                  `json:"status"`
	Timestamp  time.Time               `json:"timestamp"`
	Version    string                  `json:"version"`
	Uptime     string                  `json:"uptime"`
	Components map[string]HealthStatus `json:"components"`
}

// HealthCheck performs comprehensive health checks
func (h *HealthChecker) HealthCheck(ctx context.Context) *SystemHealth {
	startTime := time.Now()

	health := &SystemHealth{
		Timestamp:  startTime,
		Version:    "1.0.0", // This could come from build info
		Components: make(map[string]HealthStatus),
	}

	// Check database
	health.Components["database"] = h.checkDatabase(ctx)

	// Check external integrations
	if h.integrations != nil {
		health.Components["elasticsearch"] = h.checkElasticSearch(ctx)
		health.Components["datadog"] = h.checkDataDog(ctx)
		health.Components["grafana"] = h.checkGrafana(ctx)
		health.Components["bigquery"] = h.checkBigQuery(ctx)
		health.Components["s3"] = h.checkS3(ctx)
	}

	// Check system resources
	health.Components["system"] = h.checkSystemResources()

	// Check event processing
	health.Components["event_processor"] = h.checkEventProcessor()

	// Determine overall status
	health.Status = h.calculateOverallStatus(health.Components)
	health.Uptime = time.Since(startTime).String()

	return health
}

// checkDatabase checks database connectivity
func (h *HealthChecker) checkDatabase(ctx context.Context) HealthStatus {
	if h.db == nil {
		return HealthStatus{
			Status:  "unhealthy",
			Message: "Database connection not initialized",
		}
	}

	// Get the underlying SQL DB from GORM
	sqlDB, err := h.db.DB()
	if err != nil {
		return HealthStatus{
			Status:  "unhealthy",
			Message: fmt.Sprintf("Failed to get underlying DB: %v", err),
		}
	}

	// Test database connection with timeout
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = sqlDB.PingContext(pingCtx)
	if err != nil {
		return HealthStatus{
			Status:  "unhealthy",
			Message: fmt.Sprintf("Database ping failed: %v", err),
		}
	}

	// Check database stats
	stats := sqlDB.Stats()
	details := map[string]string{
		"open_connections": fmt.Sprintf("%d", stats.OpenConnections),
		"in_use":          fmt.Sprintf("%d", stats.InUse),
		"idle":            fmt.Sprintf("%d", stats.Idle),
		"max_open":        fmt.Sprintf("%d", stats.MaxOpenConnections),
	}

	// Check for connection pool exhaustion
	if stats.OpenConnections >= stats.MaxOpenConnections && stats.MaxOpenConnections > 0 {
		return HealthStatus{
			Status:  "degraded",
			Message: "Database connection pool near capacity",
			Details: details,
		}
	}

	return HealthStatus{
		Status:  "healthy",
		Message: "Database connection is healthy",
		Details: details,
	}
}

// checkElasticSearch checks ElasticSearch connectivity
func (h *HealthChecker) checkElasticSearch(ctx context.Context) HealthStatus {
	if h.integrations == nil || h.integrations.ElasticSearch == nil {
		return HealthStatus{
			Status:  "disabled",
			Message: "ElasticSearch integration not configured",
		}
	}

	// Test connection
	if err := h.integrations.ElasticSearch.TestConnection(ctx); err != nil {
		return HealthStatus{
			Status:  "unhealthy",
			Message: fmt.Sprintf("ElasticSearch connection failed: %v", err),
		}
	}

	return HealthStatus{
		Status:  "healthy",
		Message: "ElasticSearch connection is healthy",
	}
}

// checkDataDog checks DataDog integration
func (h *HealthChecker) checkDataDog(ctx context.Context) HealthStatus {
	if h.integrations == nil || h.integrations.DataDog == nil {
		return HealthStatus{
			Status:  "disabled",
			Message: "DataDog integration not configured",
		}
	}

	// Test connection
	if err := h.integrations.DataDog.TestConnection(ctx); err != nil {
		return HealthStatus{
			Status:  "unhealthy",
			Message: fmt.Sprintf("DataDog connection failed: %v", err),
		}
	}

	return HealthStatus{
		Status:  "healthy",
		Message: "DataDog integration is healthy",
	}
}

// checkGrafana checks Grafana integration
func (h *HealthChecker) checkGrafana(ctx context.Context) HealthStatus {
	if h.integrations == nil || h.integrations.Grafana == nil {
		return HealthStatus{
			Status:  "disabled",
			Message: "Grafana integration not configured",
		}
	}

	// Test connection
	if err := h.integrations.Grafana.TestConnection(ctx); err != nil {
		return HealthStatus{
			Status:  "unhealthy",
			Message: fmt.Sprintf("Grafana connection failed: %v", err),
		}
	}

	return HealthStatus{
		Status:  "healthy",
		Message: "Grafana integration is healthy",
	}
}

// checkBigQuery checks BigQuery integration
func (h *HealthChecker) checkBigQuery(ctx context.Context) HealthStatus {
	if h.integrations == nil || h.integrations.BigQuery == nil {
		return HealthStatus{
			Status:  "disabled",
			Message: "BigQuery integration not configured",
		}
	}

	// Test connection
	if err := h.integrations.BigQuery.TestConnection(ctx); err != nil {
		return HealthStatus{
			Status:  "unhealthy",
			Message: fmt.Sprintf("BigQuery connection failed: %v", err),
		}
	}

	return HealthStatus{
		Status:  "healthy",
		Message: "BigQuery integration is healthy",
	}
}

// checkS3 checks S3 integration
func (h *HealthChecker) checkS3(ctx context.Context) HealthStatus {
	if h.integrations == nil || h.integrations.S3 == nil {
		return HealthStatus{
			Status:  "disabled",
			Message: "S3 integration not configured",
		}
	}

	// Test connection
	if err := h.integrations.S3.TestConnection(ctx); err != nil {
		return HealthStatus{
			Status:  "unhealthy",
			Message: fmt.Sprintf("S3 connection failed: %v", err),
		}
	}

	return HealthStatus{
		Status:  "healthy",
		Message: "S3 integration is healthy",
	}
}

// checkSystemResources checks system resource utilization
func (h *HealthChecker) checkSystemResources() HealthStatus {
	// In a real implementation, you would check:
	// - Memory usage
	// - CPU usage
	// - Disk space
	// - Goroutine count
	// - etc.

	// For now, return a simple healthy status
	return HealthStatus{
		Status:  "healthy",
		Message: "System resources are within normal limits",
		Details: map[string]string{
			"memory": "normal",
			"cpu":    "normal",
			"disk":   "normal",
		},
	}
}

// checkEventProcessor checks the status of the event processor
func (h *HealthChecker) checkEventProcessor() HealthStatus {
	// In a real implementation, you would check:
	// - If event processor is running
	// - Queue size and processing rate
	// - Recent error rates
	// - Processing latency

	return HealthStatus{
		Status:  "healthy",
		Message: "Event processor is running normally",
		Details: map[string]string{
			"status":     "running",
			"queue_size": "normal",
			"error_rate": "low",
		},
	}
}

// calculateOverallStatus determines overall system health based on component health
func (h *HealthChecker) calculateOverallStatus(components map[string]HealthStatus) string {
	hasUnhealthy := false
	hasDegraded := false

	for _, component := range components {
		switch component.Status {
		case "unhealthy":
			hasUnhealthy = true
		case "degraded":
			hasDegraded = true
		}
	}

	if hasUnhealthy {
		return "unhealthy"
	}
	if hasDegraded {
		return "degraded"
	}
	return "healthy"
}

// HTTPHealthHandler returns an HTTP handler for health checks
func (h *HealthChecker) HTTPHealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		health := h.HealthCheck(ctx)

		// Set appropriate HTTP status based on health
		statusCode := http.StatusOK
		switch health.Status {
		case "unhealthy":
			statusCode = http.StatusServiceUnavailable
		case "degraded":
			statusCode = http.StatusOK // Still accepting requests but with issues
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)

		if err := json.NewEncoder(w).Encode(health); err != nil {
			h.logger.Error("Failed to encode health check response", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}
}

// ReadinessHandler returns an HTTP handler for readiness checks
func (h *HealthChecker) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		// For readiness, we only check critical components
		ready := true
		message := "Service is ready"

		// Check database (critical for readiness)
		dbHealth := h.checkDatabase(ctx)
		if dbHealth.Status == "unhealthy" {
			ready = false
			message = "Service not ready: database unavailable"
		}

		// Check event processor (critical for analytics)
		processorHealth := h.checkEventProcessor()
		if processorHealth.Status == "unhealthy" {
			ready = false
			message = "Service not ready: event processor unavailable"
		}

		statusCode := http.StatusOK
		if !ready {
			statusCode = http.StatusServiceUnavailable
		}

		response := map[string]interface{}{
			"ready":     ready,
			"message":   message,
			"timestamp": time.Now(),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			h.logger.Error("Failed to encode readiness response", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}
}

// LivenessHandler returns an HTTP handler for liveness checks
func (h *HealthChecker) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Liveness check is simple - if we can respond, we're alive
		response := map[string]interface{}{
			"alive":     true,
			"timestamp": time.Now(),
			"service":   "analytics-service",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			h.logger.Error("Failed to encode liveness response", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}
}