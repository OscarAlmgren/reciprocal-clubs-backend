package metrics

import (
	"time"

	"reciprocal-clubs-backend/pkg/shared/monitoring"

	"github.com/prometheus/client_golang/prometheus"
)

// AuthMetrics provides Auth Service specific metrics
type AuthMetrics struct {
	monitor *monitoring.Monitor

	// Authentication metrics
	AuthAttempts          *prometheus.CounterVec
	AuthSuccesses         *prometheus.CounterVec
	AuthFailures          *prometheus.CounterVec
	AuthDuration          *prometheus.HistogramVec
	PasskeyOperations     *prometheus.CounterVec
	PasskeyDuration       *prometheus.HistogramVec

	// User management metrics
	UserRegistrations     *prometheus.CounterVec
	UserStatusChanges     *prometheus.CounterVec
	ActiveUsers           *prometheus.GaugeVec
	UserSessions          *prometheus.GaugeVec
	SessionDuration       *prometheus.HistogramVec

	// Role and permission metrics
	RoleAssignments       *prometheus.CounterVec
	PermissionChecks      *prometheus.CounterVec
	PermissionDenials     *prometheus.CounterVec

	// Security metrics
	FailedAttempts        *prometheus.CounterVec
	AccountLockouts       *prometheus.CounterVec
	SecurityEvents        *prometheus.CounterVec
	AuditEvents           *prometheus.CounterVec

	// External service metrics
	HankoOperations       *prometheus.CounterVec
	HankoDuration         *prometheus.HistogramVec
	HankoErrors           *prometheus.CounterVec

	// Business metrics
	ClubActivity          *prometheus.CounterVec
	UserEngagement        *prometheus.GaugeVec
	FeatureUsage          *prometheus.CounterVec

	// Performance metrics
	CacheHitRate          *prometheus.CounterVec
	DatabaseQueries       *prometheus.CounterVec
	DatabaseDuration      *prometheus.HistogramVec
	ConcurrentRequests    prometheus.Gauge
	QueueSize             prometheus.Gauge

	registry *prometheus.Registry
}

// NewAuthMetrics creates a new AuthMetrics instance
func NewAuthMetrics(monitor *monitoring.Monitor) *AuthMetrics {
	registry := prometheus.NewRegistry()

	metrics := &AuthMetrics{
		monitor: monitor,

		// Authentication metrics
		AuthAttempts: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "auth_attempts_total",
				Help: "Total number of authentication attempts",
			},
			[]string{"method", "club_id", "result", "user_status"},
		),
		AuthSuccesses: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "auth_successes_total",
				Help: "Total number of successful authentications",
			},
			[]string{"method", "club_id", "user_type"},
		),
		AuthFailures: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "auth_failures_total",
				Help: "Total number of failed authentications",
			},
			[]string{"method", "club_id", "reason", "user_id"},
		),
		AuthDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "auth_duration_seconds",
				Help: "Authentication operation duration in seconds",
				Buckets: []float64{.01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"method", "club_id", "result"},
		),
		PasskeyOperations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "passkey_operations_total",
				Help: "Total number of passkey operations",
			},
			[]string{"operation", "club_id", "result", "user_id"},
		),
		PasskeyDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "passkey_duration_seconds",
				Help: "Passkey operation duration in seconds",
				Buckets: []float64{.1, .25, .5, 1, 2.5, 5, 10, 15, 30},
			},
			[]string{"operation", "club_id"},
		),

		// User management metrics
		UserRegistrations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "user_registrations_total",
				Help: "Total number of user registrations",
			},
			[]string{"club_id", "source", "result"},
		),
		UserStatusChanges: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "user_status_changes_total",
				Help: "Total number of user status changes",
			},
			[]string{"club_id", "from_status", "to_status", "reason"},
		),
		ActiveUsers: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "active_users",
				Help: "Number of currently active users",
			},
			[]string{"club_id", "time_window"},
		),
		UserSessions: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "user_sessions_active",
				Help: "Number of active user sessions",
			},
			[]string{"club_id", "session_type"},
		),
		SessionDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "session_duration_seconds",
				Help: "User session duration in seconds",
				Buckets: []float64{300, 600, 1800, 3600, 7200, 14400, 28800, 86400},
			},
			[]string{"club_id", "session_type", "termination_reason"},
		),

		// Role and permission metrics
		RoleAssignments: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "role_assignments_total",
				Help: "Total number of role assignments",
			},
			[]string{"club_id", "role_name", "operation", "granted_by"},
		),
		PermissionChecks: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "permission_checks_total",
				Help: "Total number of permission checks",
			},
			[]string{"club_id", "permission", "resource", "result"},
		),
		PermissionDenials: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "permission_denials_total",
				Help: "Total number of permission denials",
			},
			[]string{"club_id", "permission", "resource", "user_role", "reason"},
		),

		// Security metrics
		FailedAttempts: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "failed_attempts_total",
				Help: "Total number of failed authentication attempts",
			},
			[]string{"club_id", "user_id", "reason", "ip_address", "severity"},
		),
		AccountLockouts: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "account_lockouts_total",
				Help: "Total number of account lockouts",
			},
			[]string{"club_id", "user_id", "reason", "duration", "trigger"},
		),
		SecurityEvents: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "security_events_total",
				Help: "Total number of security events",
			},
			[]string{"club_id", "event_type", "severity", "source", "user_id"},
		),
		AuditEvents: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "audit_events_total",
				Help: "Total number of audit events",
			},
			[]string{"club_id", "action", "resource", "user_id", "result"},
		),

		// External service metrics
		HankoOperations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "hanko_operations_total",
				Help: "Total number of Hanko service operations",
			},
			[]string{"operation", "result", "error_type"},
		),
		HankoDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "hanko_duration_seconds",
				Help: "Hanko service operation duration in seconds",
				Buckets: []float64{.1, .25, .5, 1, 2.5, 5, 10, 15, 30, 60},
			},
			[]string{"operation", "result"},
		),
		HankoErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "hanko_errors_total",
				Help: "Total number of Hanko service errors",
			},
			[]string{"operation", "error_type", "error_code", "retry_count"},
		),

		// Business metrics
		ClubActivity: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "club_activity_total",
				Help: "Total club activity events",
			},
			[]string{"club_id", "activity_type", "user_count", "time_period"},
		),
		UserEngagement: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "user_engagement_score",
				Help: "User engagement score metrics",
			},
			[]string{"club_id", "engagement_type", "time_window"},
		),
		FeatureUsage: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "feature_usage_total",
				Help: "Total feature usage events",
			},
			[]string{"club_id", "feature", "user_type", "context"},
		),

		// Performance metrics
		CacheHitRate: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_operations_total",
				Help: "Total cache operations",
			},
			[]string{"cache_type", "operation", "result"},
		),
		DatabaseQueries: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "database_queries_total",
				Help: "Total number of database queries",
			},
			[]string{"operation", "table", "result", "query_type"},
		),
		DatabaseDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "database_query_duration_seconds",
				Help: "Database query duration in seconds",
				Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5},
			},
			[]string{"operation", "table", "query_type"},
		),
		ConcurrentRequests: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "concurrent_requests",
				Help: "Number of concurrent requests being processed",
			},
		),
		QueueSize: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "request_queue_size",
				Help: "Number of requests waiting in queue",
			},
		),

		registry: registry,
	}

	// Register all metrics
	metrics.registerMetrics()

	return metrics
}

// registerMetrics registers all metrics with the registry
func (m *AuthMetrics) registerMetrics() {
	m.registry.MustRegister(m.AuthAttempts)
	m.registry.MustRegister(m.AuthSuccesses)
	m.registry.MustRegister(m.AuthFailures)
	m.registry.MustRegister(m.AuthDuration)
	m.registry.MustRegister(m.PasskeyOperations)
	m.registry.MustRegister(m.PasskeyDuration)
	m.registry.MustRegister(m.UserRegistrations)
	m.registry.MustRegister(m.UserStatusChanges)
	m.registry.MustRegister(m.ActiveUsers)
	m.registry.MustRegister(m.UserSessions)
	m.registry.MustRegister(m.SessionDuration)
	m.registry.MustRegister(m.RoleAssignments)
	m.registry.MustRegister(m.PermissionChecks)
	m.registry.MustRegister(m.PermissionDenials)
	m.registry.MustRegister(m.FailedAttempts)
	m.registry.MustRegister(m.AccountLockouts)
	m.registry.MustRegister(m.SecurityEvents)
	m.registry.MustRegister(m.AuditEvents)
	m.registry.MustRegister(m.HankoOperations)
	m.registry.MustRegister(m.HankoDuration)
	m.registry.MustRegister(m.HankoErrors)
	m.registry.MustRegister(m.ClubActivity)
	m.registry.MustRegister(m.UserEngagement)
	m.registry.MustRegister(m.FeatureUsage)
	m.registry.MustRegister(m.CacheHitRate)
	m.registry.MustRegister(m.DatabaseQueries)
	m.registry.MustRegister(m.DatabaseDuration)
	m.registry.MustRegister(m.ConcurrentRequests)
	m.registry.MustRegister(m.QueueSize)
}

// Authentication Metrics Methods

func (m *AuthMetrics) RecordAuthAttempt(method, clubID, result, userStatus string) {
	m.AuthAttempts.WithLabelValues(method, clubID, result, userStatus).Inc()
}

func (m *AuthMetrics) RecordAuthSuccess(method, clubID, userType string) {
	m.AuthSuccesses.WithLabelValues(method, clubID, userType).Inc()
}

func (m *AuthMetrics) RecordAuthFailure(method, clubID, reason, userID string) {
	m.AuthFailures.WithLabelValues(method, clubID, reason, userID).Inc()
}

func (m *AuthMetrics) RecordAuthDuration(method, clubID, result string, duration time.Duration) {
	m.AuthDuration.WithLabelValues(method, clubID, result).Observe(duration.Seconds())
}

func (m *AuthMetrics) RecordPasskeyOperation(operation, clubID, result, userID string) {
	m.PasskeyOperations.WithLabelValues(operation, clubID, result, userID).Inc()
}

func (m *AuthMetrics) RecordPasskeyDuration(operation, clubID string, duration time.Duration) {
	m.PasskeyDuration.WithLabelValues(operation, clubID).Observe(duration.Seconds())
}

// User Management Metrics Methods

func (m *AuthMetrics) RecordUserRegistration(clubID, source, result string) {
	m.UserRegistrations.WithLabelValues(clubID, source, result).Inc()
}

func (m *AuthMetrics) RecordUserStatusChange(clubID, fromStatus, toStatus, reason string) {
	m.UserStatusChanges.WithLabelValues(clubID, fromStatus, toStatus, reason).Inc()
}

func (m *AuthMetrics) UpdateActiveUsers(clubID, timeWindow string, count int) {
	m.ActiveUsers.WithLabelValues(clubID, timeWindow).Set(float64(count))
}

func (m *AuthMetrics) UpdateUserSessions(clubID, sessionType string, count int) {
	m.UserSessions.WithLabelValues(clubID, sessionType).Set(float64(count))
}

func (m *AuthMetrics) RecordSessionDuration(clubID, sessionType, terminationReason string, duration time.Duration) {
	m.SessionDuration.WithLabelValues(clubID, sessionType, terminationReason).Observe(duration.Seconds())
}

// Role and Permission Metrics Methods

func (m *AuthMetrics) RecordRoleAssignment(clubID, roleName, operation, grantedBy string) {
	m.RoleAssignments.WithLabelValues(clubID, roleName, operation, grantedBy).Inc()
}

func (m *AuthMetrics) RecordPermissionCheck(clubID, permission, resource, result string) {
	m.PermissionChecks.WithLabelValues(clubID, permission, resource, result).Inc()
}

func (m *AuthMetrics) RecordPermissionDenial(clubID, permission, resource, userRole, reason string) {
	m.PermissionDenials.WithLabelValues(clubID, permission, resource, userRole, reason).Inc()
}

// Security Metrics Methods

func (m *AuthMetrics) RecordFailedAttempt(clubID, userID, reason, ipAddress, severity string) {
	m.FailedAttempts.WithLabelValues(clubID, userID, reason, ipAddress, severity).Inc()
}

func (m *AuthMetrics) RecordAccountLockout(clubID, userID, reason, duration, trigger string) {
	m.AccountLockouts.WithLabelValues(clubID, userID, reason, duration, trigger).Inc()
}

func (m *AuthMetrics) RecordSecurityEvent(clubID, eventType, severity, source, userID string) {
	m.SecurityEvents.WithLabelValues(clubID, eventType, severity, source, userID).Inc()
}

func (m *AuthMetrics) RecordAuditEvent(clubID, action, resource, userID, result string) {
	m.AuditEvents.WithLabelValues(clubID, action, resource, userID, result).Inc()
}

// External Service Metrics Methods

func (m *AuthMetrics) RecordHankoOperation(operation, result, errorType string) {
	m.HankoOperations.WithLabelValues(operation, result, errorType).Inc()
}

func (m *AuthMetrics) RecordHankoDuration(operation, result string, duration time.Duration) {
	m.HankoDuration.WithLabelValues(operation, result).Observe(duration.Seconds())
}

func (m *AuthMetrics) RecordHankoError(operation, errorType, errorCode, retryCount string) {
	m.HankoErrors.WithLabelValues(operation, errorType, errorCode, retryCount).Inc()
}

// Business Metrics Methods

func (m *AuthMetrics) RecordClubActivity(clubID, activityType, userCount, timePeriod string) {
	m.ClubActivity.WithLabelValues(clubID, activityType, userCount, timePeriod).Inc()
}

func (m *AuthMetrics) UpdateUserEngagement(clubID, engagementType, timeWindow string, score float64) {
	m.UserEngagement.WithLabelValues(clubID, engagementType, timeWindow).Set(score)
}

func (m *AuthMetrics) RecordFeatureUsage(clubID, feature, userType, context string) {
	m.FeatureUsage.WithLabelValues(clubID, feature, userType, context).Inc()
}

// Performance Metrics Methods

func (m *AuthMetrics) RecordCacheOperation(cacheType, operation, result string) {
	m.CacheHitRate.WithLabelValues(cacheType, operation, result).Inc()
}

func (m *AuthMetrics) RecordDatabaseQuery(operation, table, result, queryType string) {
	m.DatabaseQueries.WithLabelValues(operation, table, result, queryType).Inc()
}

func (m *AuthMetrics) RecordDatabaseDuration(operation, table, queryType string, duration time.Duration) {
	m.DatabaseDuration.WithLabelValues(operation, table, queryType).Observe(duration.Seconds())
}

func (m *AuthMetrics) IncrementConcurrentRequests() {
	m.ConcurrentRequests.Inc()
}

func (m *AuthMetrics) DecrementConcurrentRequests() {
	m.ConcurrentRequests.Dec()
}

func (m *AuthMetrics) UpdateQueueSize(size int) {
	m.QueueSize.Set(float64(size))
}

// GetRegistry returns the Prometheus registry
func (m *AuthMetrics) GetRegistry() *prometheus.Registry {
	return m.registry
}