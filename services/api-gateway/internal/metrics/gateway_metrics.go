package metrics

import (
	"time"

	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// APIGatewayMetrics provides comprehensive metrics for the API Gateway service
type APIGatewayMetrics struct {
	monitor *monitoring.Monitor
	logger  logging.Logger

	// HTTP Request Metrics
	HTTPRequestsTotal      *prometheus.CounterVec
	HTTPRequestDuration    *prometheus.HistogramVec
	HTTPRequestSize        *prometheus.HistogramVec
	HTTPResponseSize       *prometheus.HistogramVec
	HTTPActiveConnections  prometheus.Gauge

	// GraphQL Metrics
	GraphQLOperationsTotal     *prometheus.CounterVec
	GraphQLOperationDuration   *prometheus.HistogramVec
	GraphQLOperationComplexity *prometheus.HistogramVec
	GraphQLErrorsTotal         *prometheus.CounterVec
	GraphQLConcurrentQueries   prometheus.Gauge
	GraphQLSubscriptionsActive prometheus.Gauge
	GraphQLCacheHits           *prometheus.CounterVec
	GraphQLQueryDepth          *prometheus.HistogramVec

	// Service Client Metrics
	ServiceCallsTotal        *prometheus.CounterVec
	ServiceCallDuration      *prometheus.HistogramVec
	ServiceConnectionStatus  *prometheus.GaugeVec
	ServiceCircuitBreakerState *prometheus.GaugeVec
	ServiceRetryAttemptsTotal *prometheus.CounterVec
	ServiceTimeoutsTotal     *prometheus.CounterVec

	// Rate Limiting Metrics
	RateLimitRequestsTotal     *prometheus.CounterVec
	RateLimitRejectedTotal     *prometheus.CounterVec
	RateLimitCurrentLoad       *prometheus.GaugeVec
	RateLimitResetTime         *prometheus.GaugeVec

	// Authentication & Authorization Metrics
	AuthenticationAttemptsTotal *prometheus.CounterVec
	AuthenticationSuccessTotal  *prometheus.CounterVec
	AuthenticationFailuresTotal *prometheus.CounterVec
	AuthorizationChecksTotal    *prometheus.CounterVec
	AuthorizationDeniedTotal    *prometheus.CounterVec
	ActiveSessionsTotal         prometheus.Gauge
	TokenValidationDuration     *prometheus.HistogramVec

	// Business Metrics
	UserOperationsTotal     *prometheus.CounterVec
	ClubOperationsTotal     *prometheus.CounterVec
	MemberOperationsTotal   *prometheus.CounterVec
	VisitOperationsTotal    *prometheus.CounterVec
	AgreementOperationsTotal *prometheus.CounterVec

	// Performance Metrics
	MemoryUsage              prometheus.Gauge
	GoroutineCount          prometheus.Gauge
	GCDuration              *prometheus.HistogramVec
	ResponseTimePercentiles *prometheus.SummaryVec
	DatabaseConnectionPool  *prometheus.GaugeVec

	// Error and Alert Metrics
	ErrorsTotal            *prometheus.CounterVec
	CriticalErrorsTotal    *prometheus.CounterVec
	ServiceDowntime        *prometheus.CounterVec
	AlertsTriggeredTotal   *prometheus.CounterVec
}

// NewAPIGatewayMetrics creates a new instance of API Gateway metrics
func NewAPIGatewayMetrics(monitor *monitoring.Monitor, logger logging.Logger) *APIGatewayMetrics {
	metrics := &APIGatewayMetrics{
		monitor: monitor,
		logger:  logger,
	}

	// Initialize all metrics
	metrics.initializeHTTPMetrics()
	metrics.initializeGraphQLMetrics()
	metrics.initializeServiceClientMetrics()
	metrics.initializeRateLimitMetrics()
	metrics.initializeAuthMetrics()
	metrics.initializeBusinessMetrics()
	metrics.initializePerformanceMetrics()
	metrics.initializeErrorMetrics()

	logger.Info("API Gateway metrics initialized successfully", map[string]interface{}{
		"metrics_count": 35,
	})

	return metrics
}

// initializeHTTPMetrics sets up HTTP-related metrics
func (m *APIGatewayMetrics) initializeHTTPMetrics() {
	m.HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "api_gateway",
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total number of HTTP requests processed by the API Gateway",
		},
		[]string{"method", "endpoint", "status_code", "user_agent"},
	)

	m.HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "api_gateway",
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "Time spent processing HTTP requests",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "endpoint", "status_code"},
	)

	m.HTTPRequestSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "api_gateway",
			Subsystem: "http",
			Name:      "request_size_bytes",
			Help:      "Size of HTTP requests in bytes",
			Buckets:   []float64{64, 256, 1024, 4096, 16384, 65536, 262144, 1048576},
		},
		[]string{"method", "endpoint"},
	)

	m.HTTPResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "api_gateway",
			Subsystem: "http",
			Name:      "response_size_bytes",
			Help:      "Size of HTTP responses in bytes",
			Buckets:   []float64{64, 256, 1024, 4096, 16384, 65536, 262144, 1048576},
		},
		[]string{"method", "endpoint", "status_code"},
	)

	m.HTTPActiveConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "api_gateway",
			Subsystem: "http",
			Name:      "active_connections",
			Help:      "Number of active HTTP connections",
		},
	)
}

// initializeGraphQLMetrics sets up GraphQL-specific metrics
func (m *APIGatewayMetrics) initializeGraphQLMetrics() {
	m.GraphQLOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "api_gateway",
			Subsystem: "graphql",
			Name:      "operations_total",
			Help:      "Total number of GraphQL operations executed",
		},
		[]string{"operation_type", "operation_name", "status", "club_id"},
	)

	m.GraphQLOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "api_gateway",
			Subsystem: "graphql",
			Name:      "operation_duration_seconds",
			Help:      "Time spent executing GraphQL operations",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		},
		[]string{"operation_type", "operation_name", "status"},
	)

	m.GraphQLOperationComplexity = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "api_gateway",
			Subsystem: "graphql",
			Name:      "operation_complexity",
			Help:      "Complexity score of GraphQL operations",
			Buckets:   []float64{1, 5, 10, 25, 50, 100, 200, 300, 500, 1000},
		},
		[]string{"operation_type", "operation_name"},
	)

	m.GraphQLErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "api_gateway",
			Subsystem: "graphql",
			Name:      "errors_total",
			Help:      "Total number of GraphQL errors",
		},
		[]string{"error_type", "operation_name", "field_name"},
	)

	m.GraphQLConcurrentQueries = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "api_gateway",
			Subsystem: "graphql",
			Name:      "concurrent_queries",
			Help:      "Number of currently executing GraphQL queries",
		},
	)

	m.GraphQLSubscriptionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "api_gateway",
			Subsystem: "graphql",
			Name:      "subscriptions_active",
			Help:      "Number of active GraphQL subscriptions",
		},
	)

	m.GraphQLCacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "api_gateway",
			Subsystem: "graphql",
			Name:      "cache_hits_total",
			Help:      "Total number of GraphQL cache hits",
		},
		[]string{"cache_type", "operation_name"},
	)

	m.GraphQLQueryDepth = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "api_gateway",
			Subsystem: "graphql",
			Name:      "query_depth",
			Help:      "Depth of GraphQL queries",
			Buckets:   []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 15, 20},
		},
		[]string{"operation_name"},
	)
}

// initializeServiceClientMetrics sets up service client metrics
func (m *APIGatewayMetrics) initializeServiceClientMetrics() {
	m.ServiceCallsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "api_gateway",
			Subsystem: "service_client",
			Name:      "calls_total",
			Help:      "Total number of service calls made",
		},
		[]string{"service", "method", "status"},
	)

	m.ServiceCallDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "api_gateway",
			Subsystem: "service_client",
			Name:      "call_duration_seconds",
			Help:      "Time spent on service calls",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		},
		[]string{"service", "method", "status"},
	)

	m.ServiceConnectionStatus = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "api_gateway",
			Subsystem: "service_client",
			Name:      "connection_status",
			Help:      "Status of service connections (1=healthy, 0=unhealthy)",
		},
		[]string{"service", "address"},
	)

	m.ServiceCircuitBreakerState = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "api_gateway",
			Subsystem: "service_client",
			Name:      "circuit_breaker_state",
			Help:      "Circuit breaker state (0=closed, 1=open, 2=half-open)",
		},
		[]string{"service"},
	)

	m.ServiceRetryAttemptsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "api_gateway",
			Subsystem: "service_client",
			Name:      "retry_attempts_total",
			Help:      "Total number of service call retry attempts",
		},
		[]string{"service", "method"},
	)

	m.ServiceTimeoutsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "api_gateway",
			Subsystem: "service_client",
			Name:      "timeouts_total",
			Help:      "Total number of service call timeouts",
		},
		[]string{"service", "method"},
	)
}

// initializeRateLimitMetrics sets up rate limiting metrics
func (m *APIGatewayMetrics) initializeRateLimitMetrics() {
	m.RateLimitRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "api_gateway",
			Subsystem: "rate_limit",
			Name:      "requests_total",
			Help:      "Total number of requests evaluated by rate limiter",
		},
		[]string{"limiter_type", "identifier"},
	)

	m.RateLimitRejectedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "api_gateway",
			Subsystem: "rate_limit",
			Name:      "rejected_total",
			Help:      "Total number of requests rejected by rate limiter",
		},
		[]string{"limiter_type", "identifier", "reason"},
	)

	m.RateLimitCurrentLoad = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "api_gateway",
			Subsystem: "rate_limit",
			Name:      "current_load",
			Help:      "Current load as percentage of rate limit",
		},
		[]string{"limiter_type", "identifier"},
	)

	m.RateLimitResetTime = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "api_gateway",
			Subsystem: "rate_limit",
			Name:      "reset_time_seconds",
			Help:      "Time until rate limit resets",
		},
		[]string{"limiter_type", "identifier"},
	)
}

// initializeAuthMetrics sets up authentication and authorization metrics
func (m *APIGatewayMetrics) initializeAuthMetrics() {
	m.AuthenticationAttemptsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "api_gateway",
			Subsystem: "auth",
			Name:      "authentication_attempts_total",
			Help:      "Total number of authentication attempts",
		},
		[]string{"method", "club_id", "user_agent"},
	)

	m.AuthenticationSuccessTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "api_gateway",
			Subsystem: "auth",
			Name:      "authentication_success_total",
			Help:      "Total number of successful authentications",
		},
		[]string{"method", "club_id", "user_type"},
	)

	m.AuthenticationFailuresTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "api_gateway",
			Subsystem: "auth",
			Name:      "authentication_failures_total",
			Help:      "Total number of authentication failures",
		},
		[]string{"method", "club_id", "reason"},
	)

	m.AuthorizationChecksTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "api_gateway",
			Subsystem: "auth",
			Name:      "authorization_checks_total",
			Help:      "Total number of authorization checks",
		},
		[]string{"resource", "action", "club_id"},
	)

	m.AuthorizationDeniedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "api_gateway",
			Subsystem: "auth",
			Name:      "authorization_denied_total",
			Help:      "Total number of authorization denials",
		},
		[]string{"resource", "action", "club_id", "reason"},
	)

	m.ActiveSessionsTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "api_gateway",
			Subsystem: "auth",
			Name:      "active_sessions_total",
			Help:      "Number of active user sessions",
		},
	)

	m.TokenValidationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "api_gateway",
			Subsystem: "auth",
			Name:      "token_validation_duration_seconds",
			Help:      "Time spent validating authentication tokens",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5},
		},
		[]string{"token_type", "validation_method"},
	)
}

// initializeBusinessMetrics sets up business-specific metrics
func (m *APIGatewayMetrics) initializeBusinessMetrics() {
	m.UserOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "api_gateway",
			Subsystem: "business",
			Name:      "user_operations_total",
			Help:      "Total number of user operations",
		},
		[]string{"operation", "club_id", "status"},
	)

	m.ClubOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "api_gateway",
			Subsystem: "business",
			Name:      "club_operations_total",
			Help:      "Total number of club operations",
		},
		[]string{"operation", "club_id", "status"},
	)

	m.MemberOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "api_gateway",
			Subsystem: "business",
			Name:      "member_operations_total",
			Help:      "Total number of member operations",
		},
		[]string{"operation", "club_id", "membership_type", "status"},
	)

	m.VisitOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "api_gateway",
			Subsystem: "business",
			Name:      "visit_operations_total",
			Help:      "Total number of visit operations",
		},
		[]string{"operation", "club_id", "target_club_id", "status"},
	)

	m.AgreementOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "api_gateway",
			Subsystem: "business",
			Name:      "agreement_operations_total",
			Help:      "Total number of agreement operations",
		},
		[]string{"operation", "club_id", "partner_club_id", "status"},
	)
}

// initializePerformanceMetrics sets up performance monitoring metrics
func (m *APIGatewayMetrics) initializePerformanceMetrics() {
	m.MemoryUsage = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "api_gateway",
			Subsystem: "performance",
			Name:      "memory_usage_bytes",
			Help:      "Current memory usage in bytes",
		},
	)

	m.GoroutineCount = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "api_gateway",
			Subsystem: "performance",
			Name:      "goroutines_total",
			Help:      "Number of active goroutines",
		},
	)

	m.GCDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "api_gateway",
			Subsystem: "performance",
			Name:      "gc_duration_seconds",
			Help:      "Time spent in garbage collection",
			Buckets:   []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1, 0.5},
		},
		[]string{"gc_type"},
	)

	m.ResponseTimePercentiles = promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "api_gateway",
			Subsystem:  "performance",
			Name:       "response_time_percentiles",
			Help:       "Response time percentiles for different operations",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.95: 0.005, 0.99: 0.001},
		},
		[]string{"operation_type", "endpoint"},
	)

	m.DatabaseConnectionPool = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "api_gateway",
			Subsystem: "performance",
			Name:      "database_connection_pool",
			Help:      "Database connection pool status",
		},
		[]string{"service", "pool_type", "status"},
	)
}

// initializeErrorMetrics sets up error and alerting metrics
func (m *APIGatewayMetrics) initializeErrorMetrics() {
	m.ErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "api_gateway",
			Subsystem: "errors",
			Name:      "total",
			Help:      "Total number of errors by type and severity",
		},
		[]string{"error_type", "severity", "component", "club_id"},
	)

	m.CriticalErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "api_gateway",
			Subsystem: "errors",
			Name:      "critical_total",
			Help:      "Total number of critical errors requiring immediate attention",
		},
		[]string{"error_type", "component", "club_id"},
	)

	m.ServiceDowntime = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "api_gateway",
			Subsystem: "errors",
			Name:      "service_downtime_seconds_total",
			Help:      "Total downtime in seconds for backend services",
		},
		[]string{"service", "reason"},
	)

	m.AlertsTriggeredTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "api_gateway",
			Subsystem: "errors",
			Name:      "alerts_triggered_total",
			Help:      "Total number of alerts triggered",
		},
		[]string{"alert_type", "severity", "service"},
	)
}

// Recording Methods for HTTP Metrics

func (m *APIGatewayMetrics) RecordHTTPRequest(method, endpoint string, statusCode int, duration time.Duration, responseSize int) {
	statusStr := string(rune(statusCode))
	m.HTTPRequestsTotal.WithLabelValues(method, endpoint, statusStr, "unknown").Inc()
	m.HTTPRequestDuration.WithLabelValues(method, endpoint, statusStr).Observe(duration.Seconds())
	m.HTTPResponseSize.WithLabelValues(method, endpoint, statusStr).Observe(float64(responseSize))
}

func (m *APIGatewayMetrics) RecordHTTPRequestSize(method, endpoint string, requestSize int) {
	m.HTTPRequestSize.WithLabelValues(method, endpoint).Observe(float64(requestSize))
}

func (m *APIGatewayMetrics) IncrementActiveConnections() {
	m.HTTPActiveConnections.Inc()
}

func (m *APIGatewayMetrics) DecrementActiveConnections() {
	m.HTTPActiveConnections.Dec()
}

// Recording Methods for GraphQL Metrics

func (m *APIGatewayMetrics) RecordGraphQLOperation(operationType, operationName, status, clubID string, duration time.Duration, complexity int) {
	m.GraphQLOperationsTotal.WithLabelValues(operationType, operationName, status, clubID).Inc()
	m.GraphQLOperationDuration.WithLabelValues(operationType, operationName, status).Observe(duration.Seconds())
	m.GraphQLOperationComplexity.WithLabelValues(operationType, operationName).Observe(float64(complexity))
}

func (m *APIGatewayMetrics) RecordGraphQLError(errorType, operationName, fieldName string) {
	m.GraphQLErrorsTotal.WithLabelValues(errorType, operationName, fieldName).Inc()
}

func (m *APIGatewayMetrics) IncrementConcurrentQueries() {
	m.GraphQLConcurrentQueries.Inc()
}

func (m *APIGatewayMetrics) DecrementConcurrentQueries() {
	m.GraphQLConcurrentQueries.Dec()
}

func (m *APIGatewayMetrics) IncrementActiveSubscriptions() {
	m.GraphQLSubscriptionsActive.Inc()
}

func (m *APIGatewayMetrics) DecrementActiveSubscriptions() {
	m.GraphQLSubscriptionsActive.Dec()
}

func (m *APIGatewayMetrics) RecordGraphQLCacheHit(cacheType, operationName string) {
	m.GraphQLCacheHits.WithLabelValues(cacheType, operationName).Inc()
}

func (m *APIGatewayMetrics) RecordGraphQLQueryDepth(operationName string, depth int) {
	m.GraphQLQueryDepth.WithLabelValues(operationName).Observe(float64(depth))
}

// Recording Methods for Service Client Metrics

func (m *APIGatewayMetrics) RecordServiceCall(service, method, status string, duration time.Duration) {
	m.ServiceCallsTotal.WithLabelValues(service, method, status).Inc()
	m.ServiceCallDuration.WithLabelValues(service, method, status).Observe(duration.Seconds())
}

func (m *APIGatewayMetrics) UpdateServiceConnectionStatus(service, address string, healthy bool) {
	status := float64(0)
	if healthy {
		status = 1
	}
	m.ServiceConnectionStatus.WithLabelValues(service, address).Set(status)
}

func (m *APIGatewayMetrics) UpdateCircuitBreakerState(service string, state int) {
	m.ServiceCircuitBreakerState.WithLabelValues(service).Set(float64(state))
}

func (m *APIGatewayMetrics) RecordServiceRetryAttempt(service, method string) {
	m.ServiceRetryAttemptsTotal.WithLabelValues(service, method).Inc()
}

func (m *APIGatewayMetrics) RecordServiceTimeout(service, method string) {
	m.ServiceTimeoutsTotal.WithLabelValues(service, method).Inc()
}

// Recording Methods for Rate Limit Metrics

func (m *APIGatewayMetrics) RecordRateLimitRequest(limiterType, identifier string) {
	m.RateLimitRequestsTotal.WithLabelValues(limiterType, identifier).Inc()
}

func (m *APIGatewayMetrics) RecordRateLimitRejection(limiterType, identifier, reason string) {
	m.RateLimitRejectedTotal.WithLabelValues(limiterType, identifier, reason).Inc()
}

func (m *APIGatewayMetrics) UpdateRateLimitLoad(limiterType, identifier string, loadPercentage float64) {
	m.RateLimitCurrentLoad.WithLabelValues(limiterType, identifier).Set(loadPercentage)
}

func (m *APIGatewayMetrics) UpdateRateLimitResetTime(limiterType, identifier string, resetTime time.Duration) {
	m.RateLimitResetTime.WithLabelValues(limiterType, identifier).Set(resetTime.Seconds())
}

// Recording Methods for Authentication Metrics

func (m *APIGatewayMetrics) RecordAuthenticationAttempt(method, clubID, userAgent string) {
	m.AuthenticationAttemptsTotal.WithLabelValues(method, clubID, userAgent).Inc()
}

func (m *APIGatewayMetrics) RecordAuthenticationSuccess(method, clubID, userType string) {
	m.AuthenticationSuccessTotal.WithLabelValues(method, clubID, userType).Inc()
}

func (m *APIGatewayMetrics) RecordAuthenticationFailure(method, clubID, reason string) {
	m.AuthenticationFailuresTotal.WithLabelValues(method, clubID, reason).Inc()
}

func (m *APIGatewayMetrics) RecordAuthorizationCheck(resource, action, clubID string) {
	m.AuthorizationChecksTotal.WithLabelValues(resource, action, clubID).Inc()
}

func (m *APIGatewayMetrics) RecordAuthorizationDenial(resource, action, clubID, reason string) {
	m.AuthorizationDeniedTotal.WithLabelValues(resource, action, clubID, reason).Inc()
}

func (m *APIGatewayMetrics) UpdateActiveSessionsCount(count float64) {
	m.ActiveSessionsTotal.Set(count)
}

func (m *APIGatewayMetrics) RecordTokenValidationDuration(tokenType, validationMethod string, duration time.Duration) {
	m.TokenValidationDuration.WithLabelValues(tokenType, validationMethod).Observe(duration.Seconds())
}

// Recording Methods for Business Metrics

func (m *APIGatewayMetrics) RecordUserOperation(operation, clubID, status string) {
	m.UserOperationsTotal.WithLabelValues(operation, clubID, status).Inc()
}

func (m *APIGatewayMetrics) RecordClubOperation(operation, clubID, status string) {
	m.ClubOperationsTotal.WithLabelValues(operation, clubID, status).Inc()
}

func (m *APIGatewayMetrics) RecordMemberOperation(operation, clubID, membershipType, status string) {
	m.MemberOperationsTotal.WithLabelValues(operation, clubID, membershipType, status).Inc()
}

func (m *APIGatewayMetrics) RecordVisitOperation(operation, clubID, targetClubID, status string) {
	m.VisitOperationsTotal.WithLabelValues(operation, clubID, targetClubID, status).Inc()
}

func (m *APIGatewayMetrics) RecordAgreementOperation(operation, clubID, partnerClubID, status string) {
	m.AgreementOperationsTotal.WithLabelValues(operation, clubID, partnerClubID, status).Inc()
}

// Recording Methods for Performance Metrics

func (m *APIGatewayMetrics) UpdateMemoryUsage(bytes float64) {
	m.MemoryUsage.Set(bytes)
}

func (m *APIGatewayMetrics) UpdateGoroutineCount(count float64) {
	m.GoroutineCount.Set(count)
}

func (m *APIGatewayMetrics) RecordGCDuration(gcType string, duration time.Duration) {
	m.GCDuration.WithLabelValues(gcType).Observe(duration.Seconds())
}

func (m *APIGatewayMetrics) RecordResponseTimePercentile(operationType, endpoint string, responseTime time.Duration) {
	m.ResponseTimePercentiles.WithLabelValues(operationType, endpoint).Observe(responseTime.Seconds())
}

func (m *APIGatewayMetrics) UpdateDatabaseConnectionPool(service, poolType, status string, count float64) {
	m.DatabaseConnectionPool.WithLabelValues(service, poolType, status).Set(count)
}

// Recording Methods for Error Metrics

func (m *APIGatewayMetrics) RecordError(errorType, severity, component, clubID string) {
	m.ErrorsTotal.WithLabelValues(errorType, severity, component, clubID).Inc()
}

func (m *APIGatewayMetrics) RecordCriticalError(errorType, component, clubID string) {
	m.CriticalErrorsTotal.WithLabelValues(errorType, component, clubID).Inc()
}

func (m *APIGatewayMetrics) RecordServiceDowntime(service, reason string, duration time.Duration) {
	m.ServiceDowntime.WithLabelValues(service, reason).Add(duration.Seconds())
}

func (m *APIGatewayMetrics) RecordAlertTriggered(alertType, severity, service string) {
	m.AlertsTriggeredTotal.WithLabelValues(alertType, severity, service).Inc()
}