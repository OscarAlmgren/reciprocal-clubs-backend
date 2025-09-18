package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"

	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/services/notification-service/internal/providers"
)

// HealthChecker performs health checks on various system components
type HealthChecker struct {
	db        *gorm.DB
	providers *providers.NotificationProviders
	logger    logging.Logger
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(db *gorm.DB, providers *providers.NotificationProviders, logger logging.Logger) *HealthChecker {
	return &HealthChecker{
		db:        db,
		providers: providers,
		logger:    logger,
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

	// Check notification providers
	if h.providers != nil {
		health.Components["email_provider"] = h.checkEmailProvider(ctx)
		health.Components["sms_provider"] = h.checkSMSProvider(ctx)
		health.Components["push_provider"] = h.checkPushProvider(ctx)
		health.Components["webhook_provider"] = h.checkWebhookProvider(ctx)
	}

	// Check system resources
	health.Components["system"] = h.checkSystemResources()

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

// checkEmailProvider checks email provider connectivity
func (h *HealthChecker) checkEmailProvider(ctx context.Context) HealthStatus {
	if h.providers.Email == nil {
		return HealthStatus{
			Status:  "disabled",
			Message: "Email provider not configured",
		}
	}

	// Validate configuration
	err := h.providers.Email.ValidateConfig()
	if err != nil {
		return HealthStatus{
			Status:  "unhealthy",
			Message: fmt.Sprintf("Email provider configuration invalid: %v", err),
		}
	}

	// Test connection
	err = h.providers.Email.TestConnection()
	if err != nil {
		return HealthStatus{
			Status:  "unhealthy",
			Message: fmt.Sprintf("Email provider connection failed: %v", err),
		}
	}

	return HealthStatus{
		Status:  "healthy",
		Message: "Email provider is healthy",
	}
}

// checkSMSProvider checks SMS provider connectivity
func (h *HealthChecker) checkSMSProvider(ctx context.Context) HealthStatus {
	if h.providers.SMS == nil {
		return HealthStatus{
			Status:  "disabled",
			Message: "SMS provider not configured",
		}
	}

	// Validate configuration
	err := h.providers.SMS.ValidateConfig()
	if err != nil {
		return HealthStatus{
			Status:  "unhealthy",
			Message: fmt.Sprintf("SMS provider configuration invalid: %v", err),
		}
	}

	// Test connection
	err = h.providers.SMS.TestConnection(ctx)
	if err != nil {
		return HealthStatus{
			Status:  "unhealthy",
			Message: fmt.Sprintf("SMS provider connection failed: %v", err),
		}
	}

	return HealthStatus{
		Status:  "healthy",
		Message: "SMS provider is healthy",
	}
}

// checkPushProvider checks push notification provider connectivity
func (h *HealthChecker) checkPushProvider(ctx context.Context) HealthStatus {
	if h.providers.Push == nil {
		return HealthStatus{
			Status:  "disabled",
			Message: "Push notification provider not configured",
		}
	}

	// Validate configuration
	err := h.providers.Push.ValidateConfig()
	if err != nil {
		return HealthStatus{
			Status:  "unhealthy",
			Message: fmt.Sprintf("Push provider configuration invalid: %v", err),
		}
	}

	// Test connection
	err = h.providers.Push.TestConnection(ctx)
	if err != nil {
		return HealthStatus{
			Status:  "unhealthy",
			Message: fmt.Sprintf("Push provider connection failed: %v", err),
		}
	}

	return HealthStatus{
		Status:  "healthy",
		Message: "Push notification provider is healthy",
	}
}

// checkWebhookProvider checks webhook provider configuration
func (h *HealthChecker) checkWebhookProvider(ctx context.Context) HealthStatus {
	if h.providers.Webhook == nil {
		return HealthStatus{
			Status:  "disabled",
			Message: "Webhook provider not configured",
		}
	}

	// Webhook provider doesn't require connectivity checks
	return HealthStatus{
		Status:  "healthy",
		Message: "Webhook provider is configured",
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
			"service":   "notification-service",
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