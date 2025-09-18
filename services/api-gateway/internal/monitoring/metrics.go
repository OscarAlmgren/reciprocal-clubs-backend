package monitoring

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"reciprocal-clubs-backend/pkg/shared/logging"
)

// APIGatewayMetrics holds all Prometheus metrics for the API Gateway
type APIGatewayMetrics struct {
	// HTTP metrics
	HTTPRequests     *prometheus.CounterVec
	HTTPDuration     *prometheus.HistogramVec
	HTTPErrors       *prometheus.CounterVec
	HTTPResponseSize *prometheus.HistogramVec

	// GraphQL metrics
	GraphQLOperations    *prometheus.CounterVec
	GraphQLDuration      *prometheus.HistogramVec
	GraphQLErrors        *prometheus.CounterVec
	GraphQLComplexity    *prometheus.HistogramVec
	GraphQLDepth         *prometheus.HistogramVec
	GraphQLResolvers     *prometheus.CounterVec
	GraphQLSubscriptions prometheus.Gauge

	// Rate limiting metrics
	RateLimitExceeded  *prometheus.CounterVec
	RateLimitRemaining *prometheus.GaugeVec
	RateLimitRequests  *prometheus.CounterVec

	// Authentication metrics
	AuthAttempts     *prometheus.CounterVec
	AuthFailures     *prometheus.CounterVec
	AuthTokens       *prometheus.CounterVec
	AuthLatency      *prometheus.HistogramVec
	ActiveSessions   prometheus.Gauge

	// Service client metrics
	ServiceRequests  *prometheus.CounterVec
	ServiceLatency   *prometheus.HistogramVec
	ServiceErrors    *prometheus.CounterVec
	ServiceTimeouts  *prometheus.CounterVec
	ServiceRetries   *prometheus.CounterVec

	// Cache metrics (if implemented)
	CacheHits   *prometheus.CounterVec
	CacheMisses *prometheus.CounterVec
	CacheSize   *prometheus.GaugeVec

	// Security metrics
	SecurityViolations *prometheus.CounterVec
	IPBlocked          *prometheus.CounterVec
	RequestsBlocked    *prometheus.CounterVec

	// System metrics
	ActiveConnections prometheus.Gauge
	MemoryUsage       prometheus.Gauge
	GoroutineCount    prometheus.Gauge
	GCDuration        prometheus.Histogram

	// Business metrics
	UserRegistrations *prometheus.CounterVec
	UserLogins        *prometheus.CounterVec
	QueryOperations   *prometheus.CounterVec

	logger logging.Logger
}

// NewAPIGatewayMetrics creates a new instance of API Gateway metrics
func NewAPIGatewayMetrics(logger logging.Logger) *APIGatewayMetrics {
	return &APIGatewayMetrics{
		// HTTP metrics
		HTTPRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status_code"},
		),

		HTTPDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "api_gateway_http_request_duration_seconds",
				Help:    "Duration of HTTP requests",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"method", "path"},
		),

		HTTPErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_http_errors_total",
				Help: "Total number of HTTP errors",
			},
			[]string{"method", "path", "error_type"},
		),

		HTTPResponseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "api_gateway_http_response_size_bytes",
				Help:    "Size of HTTP responses in bytes",
				Buckets: []float64{100, 1000, 10000, 100000, 1000000, 10000000},
			},
			[]string{"method", "path"},
		),

		// GraphQL metrics
		GraphQLOperations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_graphql_operations_total",
				Help: "Total number of GraphQL operations",
			},
			[]string{"operation_type", "operation_name", "status"},
		),

		GraphQLDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "api_gateway_graphql_operation_duration_seconds",
				Help:    "Duration of GraphQL operations",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30},
			},
			[]string{"operation_type", "operation_name"},
		),

		GraphQLErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_graphql_errors_total",
				Help: "Total number of GraphQL errors",
			},
			[]string{"operation_type", "operation_name", "error_type"},
		),

		GraphQLComplexity: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "api_gateway_graphql_query_complexity",
				Help:    "Complexity score of GraphQL queries",
				Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
			},
			[]string{"operation_type"},
		),

		GraphQLDepth: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "api_gateway_graphql_query_depth",
				Help:    "Depth of GraphQL queries",
				Buckets: []float64{1, 2, 3, 5, 8, 12, 15, 20},
			},
			[]string{"operation_type"},
		),

		GraphQLResolvers: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_graphql_resolvers_total",
				Help: "Total number of resolver calls",
			},
			[]string{"resolver_name", "status"},
		),

		GraphQLSubscriptions: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "api_gateway_graphql_subscriptions_active",
				Help: "Number of active GraphQL subscriptions",
			},
		),

		// Rate limiting metrics
		RateLimitExceeded: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_rate_limit_exceeded_total",
				Help: "Total number of rate limit violations",
			},
			[]string{"client_type", "endpoint"},
		),

		RateLimitRemaining: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "api_gateway_rate_limit_remaining",
				Help: "Remaining rate limit quota",
			},
			[]string{"client_id", "endpoint"},
		),

		RateLimitRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_rate_limit_requests_total",
				Help: "Total number of rate limit checks",
			},
			[]string{"client_type", "result"},
		),

		// Authentication metrics
		AuthAttempts: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_auth_attempts_total",
				Help: "Total number of authentication attempts",
			},
			[]string{"method", "result"},
		),

		AuthFailures: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_auth_failures_total",
				Help: "Total number of authentication failures",
			},
			[]string{"method", "reason"},
		),

		AuthTokens: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_auth_tokens_total",
				Help: "Total number of token operations",
			},
			[]string{"operation", "status"},
		),

		AuthLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "api_gateway_auth_operation_duration_seconds",
				Help:    "Duration of authentication operations",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5},
			},
			[]string{"operation"},
		),

		ActiveSessions: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "api_gateway_active_sessions",
				Help: "Number of active user sessions",
			},
		),

		// Service client metrics
		ServiceRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_service_requests_total",
				Help: "Total number of backend service requests",
			},
			[]string{"service", "method", "status"},
		),

		ServiceLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "api_gateway_service_request_duration_seconds",
				Help:    "Duration of backend service requests",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"service", "method"},
		),

		ServiceErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_service_errors_total",
				Help: "Total number of backend service errors",
			},
			[]string{"service", "method", "error_type"},
		),

		ServiceTimeouts: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_service_timeouts_total",
				Help: "Total number of backend service timeouts",
			},
			[]string{"service", "method"},
		),

		ServiceRetries: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_service_retries_total",
				Help: "Total number of backend service retries",
			},
			[]string{"service", "method"},
		),

		// Cache metrics
		CacheHits: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_cache_hits_total",
				Help: "Total number of cache hits",
			},
			[]string{"cache_type"},
		),

		CacheMisses: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_cache_misses_total",
				Help: "Total number of cache misses",
			},
			[]string{"cache_type"},
		),

		CacheSize: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "api_gateway_cache_size_bytes",
				Help: "Current cache size in bytes",
			},
			[]string{"cache_type"},
		),

		// Security metrics
		SecurityViolations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_security_violations_total",
				Help: "Total number of security violations",
			},
			[]string{"violation_type", "source"},
		),

		IPBlocked: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_ip_blocked_total",
				Help: "Total number of blocked IP addresses",
			},
			[]string{"reason"},
		),

		RequestsBlocked: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_requests_blocked_total",
				Help: "Total number of blocked requests",
			},
			[]string{"reason", "endpoint"},
		),

		// System metrics
		ActiveConnections: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "api_gateway_active_connections",
				Help: "Number of active connections",
			},
		),

		MemoryUsage: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "api_gateway_memory_usage_bytes",
				Help: "Memory usage in bytes",
			},
		),

		GoroutineCount: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "api_gateway_goroutines_count",
				Help: "Number of active goroutines",
			},
		),

		GCDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "api_gateway_gc_duration_seconds",
				Help:    "Duration of garbage collection cycles",
				Buckets: []float64{0.00001, 0.0001, 0.001, 0.01, 0.1, 1},
			},
		),

		// Business metrics
		UserRegistrations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_user_registrations_total",
				Help: "Total number of user registrations",
			},
			[]string{"club_id", "status"},
		),

		UserLogins: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_user_logins_total",
				Help: "Total number of user logins",
			},
			[]string{"club_id", "status"},
		),

		QueryOperations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_query_operations_total",
				Help: "Total number of query operations by type",
			},
			[]string{"operation_type", "domain"},
		),

		logger: logger,
	}
}

// Record HTTP request metrics
func (m *APIGatewayMetrics) RecordHTTPRequest(method, path string, statusCode int, duration time.Duration, responseSize int) {
	statusStr := statusCodeToString(statusCode)
	m.HTTPRequests.WithLabelValues(method, path, statusStr).Inc()
	m.HTTPDuration.WithLabelValues(method, path).Observe(duration.Seconds())
	m.HTTPResponseSize.WithLabelValues(method, path).Observe(float64(responseSize))

	if statusCode >= 400 {
		errorType := "client_error"
		if statusCode >= 500 {
			errorType = "server_error"
		}
		m.HTTPErrors.WithLabelValues(method, path, errorType).Inc()
	}
}

// Record GraphQL operation metrics
func (m *APIGatewayMetrics) RecordGraphQLOperation(operationType, operationName, status string, duration time.Duration, complexity, depth int) {
	m.GraphQLOperations.WithLabelValues(operationType, operationName, status).Inc()
	m.GraphQLDuration.WithLabelValues(operationType, operationName).Observe(duration.Seconds())
	m.GraphQLComplexity.WithLabelValues(operationType).Observe(float64(complexity))
	m.GraphQLDepth.WithLabelValues(operationType).Observe(float64(depth))
}

// Record rate limit metrics
func (m *APIGatewayMetrics) RecordRateLimit(clientType, endpoint string, exceeded bool, remaining int) {
	result := "allowed"
	if exceeded {
		result = "exceeded"
		m.RateLimitExceeded.WithLabelValues(clientType, endpoint).Inc()
	}
	m.RateLimitRequests.WithLabelValues(clientType, result).Inc()
	// Note: RateLimitRemaining would be updated elsewhere with actual client ID
}

// Record authentication metrics
func (m *APIGatewayMetrics) RecordAuthAttempt(method, result string) {
	m.AuthAttempts.WithLabelValues(method, result).Inc()
}

func (m *APIGatewayMetrics) RecordAuthFailure(method, reason string) {
	m.AuthFailures.WithLabelValues(method, reason).Inc()
}

func (m *APIGatewayMetrics) RecordAuthOperation(operation, status string, duration time.Duration) {
	m.AuthTokens.WithLabelValues(operation, status).Inc()
	m.AuthLatency.WithLabelValues(operation).Observe(duration.Seconds())
}

// Record service client metrics
func (m *APIGatewayMetrics) RecordServiceRequest(service, method, status string, duration time.Duration) {
	m.ServiceRequests.WithLabelValues(service, method, status).Inc()
	m.ServiceLatency.WithLabelValues(service, method).Observe(duration.Seconds())
}

func (m *APIGatewayMetrics) RecordServiceError(service, method, errorType string) {
	m.ServiceErrors.WithLabelValues(service, method, errorType).Inc()
}

// Record business metrics
func (m *APIGatewayMetrics) RecordUserRegistration(clubID, status string) {
	m.UserRegistrations.WithLabelValues(clubID, status).Inc()
}

func (m *APIGatewayMetrics) RecordUserLogin(clubID, status string) {
	m.UserLogins.WithLabelValues(clubID, status).Inc()
}

// Update system metrics
func (m *APIGatewayMetrics) UpdateSystemMetrics(activeConnections int, memoryUsage uint64, goroutineCount int) {
	m.ActiveConnections.Set(float64(activeConnections))
	m.MemoryUsage.Set(float64(memoryUsage))
	m.GoroutineCount.Set(float64(goroutineCount))
}

// Helper function to convert status codes to strings
func statusCodeToString(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500 && code < 600:
		return "5xx"
	default:
		return "unknown"
	}
}