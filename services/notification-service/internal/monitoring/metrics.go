package monitoring

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"reciprocal-clubs-backend/pkg/shared/logging"
)

// NotificationMetrics holds all Prometheus metrics for the notification service
type NotificationMetrics struct {
	// Business metrics
	NotificationsCreated *prometheus.CounterVec
	NotificationsSent    *prometheus.CounterVec
	NotificationsFailed  *prometheus.CounterVec
	NotificationsRead    *prometheus.CounterVec

	// Delivery metrics by provider
	DeliveryDuration *prometheus.HistogramVec
	DeliveryAttempts *prometheus.CounterVec

	// Queue metrics
	PendingNotifications prometheus.Gauge
	FailedNotifications  prometheus.Gauge

	// Template metrics
	TemplatesUsed *prometheus.CounterVec

	// HTTP metrics
	HTTPRequests *prometheus.CounterVec
	HTTPDuration *prometheus.HistogramVec

	// gRPC metrics
	GRPCRequests *prometheus.CounterVec
	GRPCDuration *prometheus.HistogramVec

	// System metrics
	GoRoutines      prometheus.Gauge
	DatabaseConnections prometheus.Gauge

	logger logging.Logger
}

// NewNotificationMetrics creates a new instance of notification metrics
func NewNotificationMetrics(logger logging.Logger) *NotificationMetrics {
	return &NotificationMetrics{
		// Business metrics
		NotificationsCreated: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "notifications_created_total",
				Help: "Total number of notifications created",
			},
			[]string{"club_id", "type", "priority"},
		),

		NotificationsSent: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "notifications_sent_total",
				Help: "Total number of notifications sent successfully",
			},
			[]string{"club_id", "type", "provider"},
		),

		NotificationsFailed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "notifications_failed_total",
				Help: "Total number of failed notification deliveries",
			},
			[]string{"club_id", "type", "provider", "error_type"},
		),

		NotificationsRead: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "notifications_read_total",
				Help: "Total number of notifications marked as read",
			},
			[]string{"club_id", "type"},
		),

		// Delivery metrics
		DeliveryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "notification_delivery_duration_seconds",
				Help:    "Duration of notification delivery attempts",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"type", "provider", "status"},
		),

		DeliveryAttempts: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "notification_delivery_attempts_total",
				Help: "Total number of delivery attempts",
			},
			[]string{"type", "provider", "attempt"},
		),

		// Queue metrics
		PendingNotifications: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "notifications_pending_count",
				Help: "Number of notifications waiting to be sent",
			},
		),

		FailedNotifications: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "notifications_failed_count",
				Help: "Number of failed notifications available for retry",
			},
		),

		// Template metrics
		TemplatesUsed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "notification_templates_used_total",
				Help: "Total number of times templates were used",
			},
			[]string{"club_id", "template_name", "type"},
		),

		// HTTP metrics
		HTTPRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status_code"},
		),

		HTTPDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "Duration of HTTP requests",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"method", "path"},
		),

		// gRPC metrics
		GRPCRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "grpc_requests_total",
				Help: "Total number of gRPC requests",
			},
			[]string{"method", "status"},
		),

		GRPCDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "grpc_request_duration_seconds",
				Help:    "Duration of gRPC requests",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"method"},
		),

		// System metrics
		GoRoutines: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "go_routines_count",
				Help: "Number of active goroutines",
			},
		),

		DatabaseConnections: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "database_connections_active",
				Help: "Number of active database connections",
			},
		),

		logger: logger,
	}
}

// RecordNotificationCreated records a notification creation
func (m *NotificationMetrics) RecordNotificationCreated(clubID, notificationType, priority string) {
	m.NotificationsCreated.WithLabelValues(clubID, notificationType, priority).Inc()
}

// RecordNotificationSent records a successful notification delivery
func (m *NotificationMetrics) RecordNotificationSent(clubID, notificationType, provider string) {
	m.NotificationsSent.WithLabelValues(clubID, notificationType, provider).Inc()
}

// RecordNotificationFailed records a failed notification delivery
func (m *NotificationMetrics) RecordNotificationFailed(clubID, notificationType, provider, errorType string) {
	m.NotificationsFailed.WithLabelValues(clubID, notificationType, provider, errorType).Inc()
}

// RecordNotificationRead records a notification being read
func (m *NotificationMetrics) RecordNotificationRead(clubID, notificationType string) {
	m.NotificationsRead.WithLabelValues(clubID, notificationType).Inc()
}

// RecordDeliveryDuration records the duration of a delivery attempt
func (m *NotificationMetrics) RecordDeliveryDuration(notificationType, provider, status string, duration time.Duration) {
	m.DeliveryDuration.WithLabelValues(notificationType, provider, status).Observe(duration.Seconds())
}

// RecordDeliveryAttempt records a delivery attempt
func (m *NotificationMetrics) RecordDeliveryAttempt(notificationType, provider string, attemptNumber int) {
	attempt := "1"
	if attemptNumber == 2 {
		attempt = "2"
	} else if attemptNumber == 3 {
		attempt = "3"
	} else if attemptNumber > 3 {
		attempt = "4+"
	}
	m.DeliveryAttempts.WithLabelValues(notificationType, provider, attempt).Inc()
}

// UpdatePendingNotifications updates the pending notifications gauge
func (m *NotificationMetrics) UpdatePendingNotifications(count float64) {
	m.PendingNotifications.Set(count)
}

// UpdateFailedNotifications updates the failed notifications gauge
func (m *NotificationMetrics) UpdateFailedNotifications(count float64) {
	m.FailedNotifications.Set(count)
}

// RecordTemplateUsage records template usage
func (m *NotificationMetrics) RecordTemplateUsage(clubID, templateName, notificationType string) {
	m.TemplatesUsed.WithLabelValues(clubID, templateName, notificationType).Inc()
}

// RecordHTTPRequest records an HTTP request
func (m *NotificationMetrics) RecordHTTPRequest(method, path string, statusCode int, duration time.Duration) {
	statusCodeStr := m.statusCodeToString(statusCode)
	m.HTTPRequests.WithLabelValues(method, path, statusCodeStr).Inc()
	m.HTTPDuration.WithLabelValues(method, path).Observe(duration.Seconds())
}

// RecordGRPCRequest records a gRPC request
func (m *NotificationMetrics) RecordGRPCRequest(method, status string, duration time.Duration) {
	m.GRPCRequests.WithLabelValues(method, status).Inc()
	m.GRPCDuration.WithLabelValues(method).Observe(duration.Seconds())
}

// UpdateSystemMetrics updates system-level metrics
func (m *NotificationMetrics) UpdateSystemMetrics(goRoutines int, dbConnections int) {
	m.GoRoutines.Set(float64(goRoutines))
	m.DatabaseConnections.Set(float64(dbConnections))
}

// GetMetricsHandler returns HTTP handler for Prometheus metrics
func (m *NotificationMetrics) GetMetricsHandler() *prometheus.Registry {
	return prometheus.DefaultRegisterer.(*prometheus.Registry)
}

// statusCodeToString converts HTTP status code to string
func (m *NotificationMetrics) statusCodeToString(code int) string {
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