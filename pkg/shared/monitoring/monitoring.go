package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"reciprocal-clubs-backend/pkg/shared/config"
	"reciprocal-clubs-backend/pkg/shared/logging"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all application metrics
type Metrics struct {
	HTTPRequestDuration    *prometheus.HistogramVec
	HTTPRequestsTotal      *prometheus.CounterVec
	GRPCRequestDuration    *prometheus.HistogramVec
	GRPCRequestsTotal      *prometheus.CounterVec
	DatabaseConnections    prometheus.Gauge
	ActiveConnections      prometheus.Gauge
	MessagesBusReceived    *prometheus.CounterVec
	MessageBusPublished    *prometheus.CounterVec
	BusinessMetrics        *prometheus.CounterVec
	HealthStatus           *prometheus.GaugeVec
	ServiceUptime          prometheus.Counter
	registry               *prometheus.Registry
}

// HealthChecker defines the interface for health checking
type HealthChecker interface {
	HealthCheck(ctx context.Context) error
	Name() string
}

// HealthStatus represents the health status of a component
type HealthStatus struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Error     string    `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Duration  string    `json:"duration"`
}

// SystemHealth represents overall system health
type SystemHealth struct {
	Status     string          `json:"status"`
	Service    string          `json:"service"`
	Version    string          `json:"version"`
	Timestamp  time.Time       `json:"timestamp"`
	Uptime     string          `json:"uptime"`
	Components []HealthStatus  `json:"components"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// Monitor provides monitoring and metrics capabilities
type Monitor struct {
	config       *config.MonitoringConfig
	logger       logging.Logger
	metrics      *Metrics
	healthChecks map[string]HealthChecker
	startTime    time.Time
	serviceName  string
	version      string
}

// NewMonitor creates a new monitor instance
func NewMonitor(cfg *config.MonitoringConfig, logger logging.Logger, serviceName, version string) *Monitor {
	registry := prometheus.NewRegistry()

	metrics := &Metrics{
		HTTPRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "http_request_duration_seconds",
				Help: "HTTP request duration in seconds",
				Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"method", "endpoint", "status_code", "service"},
		),
		HTTPRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status_code", "service"},
		),
		GRPCRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "grpc_request_duration_seconds",
				Help: "gRPC request duration in seconds",
				Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"method", "service", "status"},
		),
		GRPCRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "grpc_requests_total",
				Help: "Total number of gRPC requests",
			},
			[]string{"method", "service", "status"},
		),
		DatabaseConnections: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "database_connections_active",
				Help: "Number of active database connections",
			},
		),
		ActiveConnections: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "active_connections",
				Help: "Number of active connections",
			},
		),
		MessagesBusReceived: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "message_bus_messages_received_total",
				Help: "Total number of messages received from message bus",
			},
			[]string{"subject", "service"},
		),
		MessageBusPublished: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "message_bus_messages_published_total",
				Help: "Total number of messages published to message bus",
			},
			[]string{"subject", "service"},
		),
		BusinessMetrics: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "business_events_total",
				Help: "Total number of business events",
			},
			[]string{"event_type", "club_id", "service"},
		),
		HealthStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "health_status",
				Help: "Health status of system components (1 = healthy, 0 = unhealthy)",
			},
			[]string{"component", "service"},
		),
		ServiceUptime: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "service_uptime_seconds_total",
				Help: "Total service uptime in seconds",
			},
		),
		registry: registry,
	}

	// Register metrics with registry
	registry.MustRegister(metrics.HTTPRequestDuration)
	registry.MustRegister(metrics.HTTPRequestsTotal)
	registry.MustRegister(metrics.GRPCRequestDuration)
	registry.MustRegister(metrics.GRPCRequestsTotal)
	registry.MustRegister(metrics.DatabaseConnections)
	registry.MustRegister(metrics.ActiveConnections)
	registry.MustRegister(metrics.MessagesBusReceived)
	registry.MustRegister(metrics.MessageBusPublished)
	registry.MustRegister(metrics.BusinessMetrics)
	registry.MustRegister(metrics.HealthStatus)
	registry.MustRegister(metrics.ServiceUptime)

	return &Monitor{
		config:       cfg,
		logger:       logger,
		metrics:      metrics,
		healthChecks: make(map[string]HealthChecker),
		startTime:    time.Now(),
		serviceName:  serviceName,
		version:      version,
	}
}

// GetMetrics returns the metrics instance
func (m *Monitor) GetMetrics() *Metrics {
	return m.metrics
}

// RecordHTTPRequest records HTTP request metrics
func (m *Monitor) RecordHTTPRequest(method, endpoint string, statusCode int, duration time.Duration) {
	statusStr := fmt.Sprintf("%d", statusCode)
	m.metrics.HTTPRequestDuration.WithLabelValues(method, endpoint, statusStr, m.serviceName).Observe(duration.Seconds())
	m.metrics.HTTPRequestsTotal.WithLabelValues(method, endpoint, statusStr, m.serviceName).Inc()
}

// RecordGRPCRequest records gRPC request metrics
func (m *Monitor) RecordGRPCRequest(method, status string, duration time.Duration) {
	m.metrics.GRPCRequestDuration.WithLabelValues(method, m.serviceName, status).Observe(duration.Seconds())
	m.metrics.GRPCRequestsTotal.WithLabelValues(method, m.serviceName, status).Inc()
}

// RecordDatabaseConnections records database connection metrics
func (m *Monitor) RecordDatabaseConnections(count int) {
	m.metrics.DatabaseConnections.Set(float64(count))
}

// RecordActiveConnections records active connection metrics
func (m *Monitor) RecordActiveConnections(count int) {
	m.metrics.ActiveConnections.Set(float64(count))
}

// RecordMessageReceived records message bus receive metrics
func (m *Monitor) RecordMessageReceived(subject string) {
	m.metrics.MessagesBusReceived.WithLabelValues(subject, m.serviceName).Inc()
}

// RecordMessagePublished records message bus publish metrics
func (m *Monitor) RecordMessagePublished(subject string) {
	m.metrics.MessageBusPublished.WithLabelValues(subject, m.serviceName).Inc()
}

// RecordBusinessEvent records business event metrics
func (m *Monitor) RecordBusinessEvent(eventType, clubID string) {
	m.metrics.BusinessMetrics.WithLabelValues(eventType, clubID, m.serviceName).Inc()
}

// UpdateServiceUptime updates the service uptime counter
func (m *Monitor) UpdateServiceUptime() {
	uptime := time.Since(m.startTime).Seconds()
	m.metrics.ServiceUptime.Add(uptime)
}

// RegisterHealthCheck registers a health checker
func (m *Monitor) RegisterHealthCheck(checker HealthChecker) {
	m.healthChecks[checker.Name()] = checker
	m.logger.Info("Health check registered", map[string]interface{}{
		"component": checker.Name(),
	})
}

// StartMetricsServer starts the metrics HTTP server
func (m *Monitor) StartMetricsServer() {
	if !m.config.EnableMetrics {
		return
	}

	mux := http.NewServeMux()
	mux.Handle(m.config.MetricsPath, promhttp.HandlerFor(m.metrics.registry, promhttp.HandlerOpts{}))

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", m.config.MetricsPort),
		Handler: mux,
	}

	go func() {
		m.logger.Info("Starting metrics server", map[string]interface{}{
			"port": m.config.MetricsPort,
			"path": m.config.MetricsPath,
		})

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			m.logger.Error("Metrics server failed", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()
}

// HealthCheckHandler returns HTTP handler for health checks
func (m *Monitor) HealthCheckHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		health := m.CheckHealth(ctx)

		w.Header().Set("Content-Type", "application/json")
		
		if health.Status == "healthy" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		if err := json.NewEncoder(w).Encode(health); err != nil {
			m.logger.Error("Failed to encode health response", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}
}

// ReadinessCheckHandler returns HTTP handler for readiness checks
func (m *Monitor) ReadinessCheckHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		health := m.CheckHealth(ctx)

		w.Header().Set("Content-Type", "application/json")
		
		// For readiness, all critical components must be healthy
		allHealthy := true
		for _, component := range health.Components {
			if component.Status != "healthy" {
				allHealthy = false
				break
			}
		}

		if allHealthy {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		if err := json.NewEncoder(w).Encode(health); err != nil {
			m.logger.Error("Failed to encode readiness response", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}
}

// CheckHealth performs health checks on all registered components
func (m *Monitor) CheckHealth(ctx context.Context) *SystemHealth {
	start := time.Now()
	uptime := time.Since(m.startTime)
	
	health := &SystemHealth{
		Service:    m.serviceName,
		Version:    m.version,
		Timestamp:  start,
		Uptime:     uptime.String(),
		Components: make([]HealthStatus, 0, len(m.healthChecks)),
		Metadata: map[string]interface{}{
			"start_time": m.startTime,
			"uptime_seconds": uptime.Seconds(),
		},
	}

	overallHealthy := true

	for name, checker := range m.healthChecks {
		componentStart := time.Now()
		var componentHealth HealthStatus

		checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := checker.HealthCheck(checkCtx)
		cancel()

		componentHealth = HealthStatus{
			Name:      name,
			Timestamp: componentStart,
			Duration:  time.Since(componentStart).String(),
		}

		if err != nil {
			componentHealth.Status = "unhealthy"
			componentHealth.Error = err.Error()
			overallHealthy = false
			m.metrics.HealthStatus.WithLabelValues(name, m.serviceName).Set(0)
		} else {
			componentHealth.Status = "healthy"
			m.metrics.HealthStatus.WithLabelValues(name, m.serviceName).Set(1)
		}

		health.Components = append(health.Components, componentHealth)
	}

	if overallHealthy {
		health.Status = "healthy"
	} else {
		health.Status = "unhealthy"
	}

	return health
}