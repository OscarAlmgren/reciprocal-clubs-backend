package monitoring

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"reciprocal-clubs-backend/pkg/shared/logging"
)

// MemberServiceMetrics holds all Prometheus metrics for the Member Service
type MemberServiceMetrics struct {
	// HTTP metrics
	HTTPRequests     *prometheus.CounterVec
	HTTPDuration     *prometheus.HistogramVec
	HTTPErrors       *prometheus.CounterVec

	// gRPC metrics
	GRPCRequests     *prometheus.CounterVec
	GRPCDuration     *prometheus.HistogramVec
	GRPCErrors       *prometheus.CounterVec

	// Member operations
	MemberOperations *prometheus.CounterVec
	MemberCreated    prometheus.Counter
	MemberUpdated    prometheus.Counter
	MemberSuspended  prometheus.Counter
	MemberDeleted    prometheus.Counter

	// Member status distribution
	MembersByStatus      *prometheus.GaugeVec
	MembersByType        *prometheus.GaugeVec
	MembersByClub        *prometheus.GaugeVec

	// Database operations
	DatabaseQueries  *prometheus.CounterVec
	DatabaseDuration *prometheus.HistogramVec
	DatabaseErrors   *prometheus.CounterVec

	// Repository operations
	RepositoryOperations *prometheus.CounterVec
	RepositoryDuration   *prometheus.HistogramVec
	RepositoryErrors     *prometheus.CounterVec

	// Service operations
	ServiceOperations *prometheus.CounterVec
	ServiceDuration   *prometheus.HistogramVec
	ServiceErrors     *prometheus.CounterVec

	// Business metrics
	ActiveMembers      prometheus.Gauge
	TotalMembers       prometheus.Gauge
	MemberAccessChecks *prometheus.CounterVec
	MemberValidations  *prometheus.CounterVec

	// System metrics
	MemoryUsage    prometheus.Gauge
	GoroutineCount prometheus.Gauge
	GCDuration     prometheus.Histogram

	logger logging.Logger
}

// NewMemberServiceMetrics creates a new instance of Member Service metrics
func NewMemberServiceMetrics(logger logging.Logger) *MemberServiceMetrics {
	return &MemberServiceMetrics{
		// HTTP metrics
		HTTPRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "member_service_http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status_code"},
		),

		HTTPDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "member_service_http_request_duration_seconds",
				Help:    "Duration of HTTP requests",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"method", "path"},
		),

		HTTPErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "member_service_http_errors_total",
				Help: "Total number of HTTP errors",
			},
			[]string{"method", "path", "error_type"},
		),

		// gRPC metrics
		GRPCRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "member_service_grpc_requests_total",
				Help: "Total number of gRPC requests",
			},
			[]string{"method", "status"},
		),

		GRPCDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "member_service_grpc_request_duration_seconds",
				Help:    "Duration of gRPC requests",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"method"},
		),

		GRPCErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "member_service_grpc_errors_total",
				Help: "Total number of gRPC errors",
			},
			[]string{"method", "code"},
		),

		// Member operations
		MemberOperations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "member_service_member_operations_total",
				Help: "Total number of member operations",
			},
			[]string{"operation", "status"},
		),

		MemberCreated: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "member_service_members_created_total",
				Help: "Total number of members created",
			},
		),

		MemberUpdated: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "member_service_members_updated_total",
				Help: "Total number of members updated",
			},
		),

		MemberSuspended: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "member_service_members_suspended_total",
				Help: "Total number of members suspended",
			},
		),

		MemberDeleted: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "member_service_members_deleted_total",
				Help: "Total number of members deleted",
			},
		),

		// Member status distribution
		MembersByStatus: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "member_service_members_by_status",
				Help: "Number of members by status",
			},
			[]string{"status"},
		),

		MembersByType: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "member_service_members_by_type",
				Help: "Number of members by membership type",
			},
			[]string{"membership_type"},
		),

		MembersByClub: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "member_service_members_by_club",
				Help: "Number of members by club",
			},
			[]string{"club_id"},
		),

		// Database operations
		DatabaseQueries: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "member_service_database_queries_total",
				Help: "Total number of database queries",
			},
			[]string{"operation", "table", "status"},
		),

		DatabaseDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "member_service_database_query_duration_seconds",
				Help:    "Duration of database queries",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5},
			},
			[]string{"operation", "table"},
		),

		DatabaseErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "member_service_database_errors_total",
				Help: "Total number of database errors",
			},
			[]string{"operation", "table", "error_type"},
		),

		// Repository operations
		RepositoryOperations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "member_service_repository_operations_total",
				Help: "Total number of repository operations",
			},
			[]string{"operation", "status"},
		),

		RepositoryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "member_service_repository_operation_duration_seconds",
				Help:    "Duration of repository operations",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5},
			},
			[]string{"operation"},
		),

		RepositoryErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "member_service_repository_errors_total",
				Help: "Total number of repository errors",
			},
			[]string{"operation", "error_type"},
		),

		// Service operations
		ServiceOperations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "member_service_service_operations_total",
				Help: "Total number of service operations",
			},
			[]string{"operation", "status"},
		),

		ServiceDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "member_service_service_operation_duration_seconds",
				Help:    "Duration of service operations",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5},
			},
			[]string{"operation"},
		),

		ServiceErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "member_service_service_errors_total",
				Help: "Total number of service errors",
			},
			[]string{"operation", "error_type"},
		),

		// Business metrics
		ActiveMembers: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "member_service_active_members_total",
				Help: "Total number of active members",
			},
		),

		TotalMembers: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "member_service_total_members",
				Help: "Total number of members",
			},
		),

		MemberAccessChecks: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "member_service_access_checks_total",
				Help: "Total number of member access checks",
			},
			[]string{"result"},
		),

		MemberValidations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "member_service_validations_total",
				Help: "Total number of member validations",
			},
			[]string{"operation", "result"},
		),

		// System metrics
		MemoryUsage: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "member_service_memory_usage_bytes",
				Help: "Memory usage in bytes",
			},
		),

		GoroutineCount: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "member_service_goroutines_count",
				Help: "Number of active goroutines",
			},
		),

		GCDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "member_service_gc_duration_seconds",
				Help:    "Duration of garbage collection cycles",
				Buckets: []float64{0.00001, 0.0001, 0.001, 0.01, 0.1, 1},
			},
		),

		logger: logger,
	}
}

// RecordHTTPRequest records HTTP request metrics
func (m *MemberServiceMetrics) RecordHTTPRequest(method, path string, statusCode int, duration time.Duration) {
	statusStr := statusCodeToString(statusCode)
	m.HTTPRequests.WithLabelValues(method, path, statusStr).Inc()
	m.HTTPDuration.WithLabelValues(method, path).Observe(duration.Seconds())

	if statusCode >= 400 {
		errorType := "client_error"
		if statusCode >= 500 {
			errorType = "server_error"
		}
		m.HTTPErrors.WithLabelValues(method, path, errorType).Inc()
	}
}

// RecordGRPCRequest records gRPC request metrics
func (m *MemberServiceMetrics) RecordGRPCRequest(method, status string, duration time.Duration) {
	m.GRPCRequests.WithLabelValues(method, status).Inc()
	m.GRPCDuration.WithLabelValues(method).Observe(duration.Seconds())
}

// RecordGRPCError records gRPC error metrics
func (m *MemberServiceMetrics) RecordGRPCError(method, code string) {
	m.GRPCErrors.WithLabelValues(method, code).Inc()
}

// RecordMemberOperation records member operation metrics
func (m *MemberServiceMetrics) RecordMemberOperation(operation, status string) {
	m.MemberOperations.WithLabelValues(operation, status).Inc()

	switch operation {
	case "create":
		if status == "success" {
			m.MemberCreated.Inc()
		}
	case "update":
		if status == "success" {
			m.MemberUpdated.Inc()
		}
	case "suspend":
		if status == "success" {
			m.MemberSuspended.Inc()
		}
	case "delete":
		if status == "success" {
			m.MemberDeleted.Inc()
		}
	}
}

// RecordDatabaseOperation records database operation metrics
func (m *MemberServiceMetrics) RecordDatabaseOperation(operation, table, status string, duration time.Duration) {
	m.DatabaseQueries.WithLabelValues(operation, table, status).Inc()
	m.DatabaseDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
}

// RecordDatabaseError records database error metrics
func (m *MemberServiceMetrics) RecordDatabaseError(operation, table, errorType string) {
	m.DatabaseErrors.WithLabelValues(operation, table, errorType).Inc()
}

// RecordRepositoryOperation records repository operation metrics
func (m *MemberServiceMetrics) RecordRepositoryOperation(operation, status string, duration time.Duration) {
	m.RepositoryOperations.WithLabelValues(operation, status).Inc()
	m.RepositoryDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// RecordRepositoryError records repository error metrics
func (m *MemberServiceMetrics) RecordRepositoryError(operation, errorType string) {
	m.RepositoryErrors.WithLabelValues(operation, errorType).Inc()
}

// RecordServiceOperation records service operation metrics
func (m *MemberServiceMetrics) RecordServiceOperation(operation, status string, duration time.Duration) {
	m.ServiceOperations.WithLabelValues(operation, status).Inc()
	m.ServiceDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// RecordServiceError records service error metrics
func (m *MemberServiceMetrics) RecordServiceError(operation, errorType string) {
	m.ServiceErrors.WithLabelValues(operation, errorType).Inc()
}

// RecordMemberAccessCheck records member access check metrics
func (m *MemberServiceMetrics) RecordMemberAccessCheck(result string) {
	m.MemberAccessChecks.WithLabelValues(result).Inc()
}

// RecordMemberValidation records member validation metrics
func (m *MemberServiceMetrics) RecordMemberValidation(operation, result string) {
	m.MemberValidations.WithLabelValues(operation, result).Inc()
}

// UpdateMemberStats updates member statistics
func (m *MemberServiceMetrics) UpdateMemberStats(activeMembers, totalMembers int64) {
	m.ActiveMembers.Set(float64(activeMembers))
	m.TotalMembers.Set(float64(totalMembers))
}

// UpdateMembersByStatus updates members by status gauge
func (m *MemberServiceMetrics) UpdateMembersByStatus(status string, count int64) {
	m.MembersByStatus.WithLabelValues(status).Set(float64(count))
}

// UpdateMembersByType updates members by type gauge
func (m *MemberServiceMetrics) UpdateMembersByType(membershipType string, count int64) {
	m.MembersByType.WithLabelValues(membershipType).Set(float64(count))
}

// UpdateMembersByClub updates members by club gauge
func (m *MemberServiceMetrics) UpdateMembersByClub(clubID string, count int64) {
	m.MembersByClub.WithLabelValues(clubID).Set(float64(count))
}

// UpdateSystemMetrics updates system-level metrics
func (m *MemberServiceMetrics) UpdateSystemMetrics(memoryUsage uint64, goroutineCount int) {
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