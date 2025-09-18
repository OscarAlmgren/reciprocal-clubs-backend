package monitoring

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"reciprocal-clubs-backend/pkg/shared/logging"
)

// AnalyticsMetrics holds all Prometheus metrics for the analytics service
type AnalyticsMetrics struct {
	// Business metrics
	EventsProcessed     *prometheus.CounterVec
	EventsRecorded      *prometheus.CounterVec
	EventsExported      *prometheus.CounterVec
	ReportsGenerated    *prometheus.CounterVec
	DashboardsCreated   *prometheus.CounterVec

	// Data processing metrics
	ProcessingDuration  *prometheus.HistogramVec
	QueueSize          prometheus.Gauge
	ProcessingErrors   *prometheus.CounterVec

	// External integration metrics
	IntegrationRequests *prometheus.CounterVec
	IntegrationLatency  *prometheus.HistogramVec
	IntegrationErrors   *prometheus.CounterVec

	// Repository metrics
	DatabaseQueries     *prometheus.CounterVec
	QueryDuration       *prometheus.HistogramVec
	DatabaseConnections prometheus.Gauge

	// HTTP metrics
	HTTPRequests *prometheus.CounterVec
	HTTPDuration *prometheus.HistogramVec

	// gRPC metrics
	GRPCRequests *prometheus.CounterVec
	GRPCDuration *prometheus.HistogramVec

	// System metrics
	GoRoutines      prometheus.Gauge
	MemoryUsage     prometheus.Gauge
	CPUUsage        prometheus.Gauge

	// Data volume metrics
	EventVolume      *prometheus.CounterVec
	MetricVolume     *prometheus.CounterVec
	ReportSize       *prometheus.HistogramVec
	DataExportSize   *prometheus.HistogramVec

	logger logging.Logger
}

// NewAnalyticsMetrics creates a new instance of analytics metrics
func NewAnalyticsMetrics(logger logging.Logger) *AnalyticsMetrics {
	return &AnalyticsMetrics{
		// Business metrics
		EventsProcessed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "analytics_events_processed_total",
				Help: "Total number of analytics events processed",
			},
			[]string{"club_id", "event_type", "status"},
		),

		EventsRecorded: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "analytics_events_recorded_total",
				Help: "Total number of analytics events recorded",
			},
			[]string{"club_id", "event_type", "source"},
		),

		EventsExported: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "analytics_events_exported_total",
				Help: "Total number of events exported to external systems",
			},
			[]string{"export_type", "status"},
		),

		ReportsGenerated: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "analytics_reports_generated_total",
				Help: "Total number of analytics reports generated",
			},
			[]string{"club_id", "report_type"},
		),

		DashboardsCreated: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "analytics_dashboards_created_total",
				Help: "Total number of dashboards created",
			},
			[]string{"club_id", "dashboard_type"},
		),

		// Data processing metrics
		ProcessingDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "analytics_processing_duration_seconds",
				Help:    "Duration of analytics processing operations",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"operation", "status"},
		),

		QueueSize: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "analytics_queue_size",
				Help: "Current size of analytics processing queue",
			},
		),

		ProcessingErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "analytics_processing_errors_total",
				Help: "Total number of processing errors",
			},
			[]string{"operation", "error_type"},
		),

		// External integration metrics
		IntegrationRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "analytics_integration_requests_total",
				Help: "Total number of external integration requests",
			},
			[]string{"integration", "operation", "status"},
		),

		IntegrationLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "analytics_integration_latency_seconds",
				Help:    "Latency of external integration requests",
				Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30},
			},
			[]string{"integration", "operation"},
		),

		IntegrationErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "analytics_integration_errors_total",
				Help: "Total number of external integration errors",
			},
			[]string{"integration", "operation", "error_type"},
		),

		// Repository metrics
		DatabaseQueries: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "analytics_database_queries_total",
				Help: "Total number of database queries executed",
			},
			[]string{"operation", "table", "status"},
		),

		QueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "analytics_database_query_duration_seconds",
				Help:    "Duration of database queries",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
			},
			[]string{"operation", "table"},
		),

		DatabaseConnections: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "analytics_database_connections_active",
				Help: "Number of active database connections",
			},
		),

		// HTTP metrics
		HTTPRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "analytics_http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status_code"},
		),

		HTTPDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "analytics_http_request_duration_seconds",
				Help:    "Duration of HTTP requests",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"method", "path"},
		),

		// gRPC metrics
		GRPCRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "analytics_grpc_requests_total",
				Help: "Total number of gRPC requests",
			},
			[]string{"method", "status"},
		),

		GRPCDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "analytics_grpc_request_duration_seconds",
				Help:    "Duration of gRPC requests",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"method"},
		),

		// System metrics
		GoRoutines: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "analytics_go_routines_count",
				Help: "Number of active goroutines",
			},
		),

		MemoryUsage: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "analytics_memory_usage_bytes",
				Help: "Memory usage in bytes",
			},
		),

		CPUUsage: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "analytics_cpu_usage_percent",
				Help: "CPU usage percentage",
			},
		),

		// Data volume metrics
		EventVolume: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "analytics_event_volume_bytes",
				Help: "Total volume of event data processed in bytes",
			},
			[]string{"club_id", "event_type"},
		),

		MetricVolume: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "analytics_metric_volume_bytes",
				Help: "Total volume of metric data processed in bytes",
			},
			[]string{"club_id", "metric_name"},
		),

		ReportSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "analytics_report_size_bytes",
				Help:    "Size of generated reports in bytes",
				Buckets: []float64{1024, 4096, 16384, 65536, 262144, 1048576, 4194304, 16777216},
			},
			[]string{"club_id", "report_type"},
		),

		DataExportSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "analytics_data_export_size_bytes",
				Help:    "Size of data exports in bytes",
				Buckets: []float64{1024, 4096, 16384, 65536, 262144, 1048576, 4194304, 16777216, 67108864},
			},
			[]string{"export_type"},
		),

		logger: logger,
	}
}

// RecordEventProcessed records a processed analytics event
func (m *AnalyticsMetrics) RecordEventProcessed(clubID, eventType, status string) {
	m.EventsProcessed.WithLabelValues(clubID, eventType, status).Inc()
}

// RecordEventRecorded records a recorded analytics event
func (m *AnalyticsMetrics) RecordEventRecorded(clubID, eventType, source string) {
	m.EventsRecorded.WithLabelValues(clubID, eventType, source).Inc()
}

// RecordEventExported records an exported event
func (m *AnalyticsMetrics) RecordEventExported(exportType, status string) {
	m.EventsExported.WithLabelValues(exportType, status).Inc()
}

// RecordReportGenerated records a generated report
func (m *AnalyticsMetrics) RecordReportGenerated(clubID, reportType string, size int) {
	m.ReportsGenerated.WithLabelValues(clubID, reportType).Inc()
	m.ReportSize.WithLabelValues(clubID, reportType).Observe(float64(size))
}

// RecordDashboardCreated records a created dashboard
func (m *AnalyticsMetrics) RecordDashboardCreated(clubID, dashboardType string) {
	m.DashboardsCreated.WithLabelValues(clubID, dashboardType).Inc()
}

// RecordProcessingDuration records the duration of a processing operation
func (m *AnalyticsMetrics) RecordProcessingDuration(operation, status string, duration time.Duration) {
	m.ProcessingDuration.WithLabelValues(operation, status).Observe(duration.Seconds())
}

// UpdateQueueSize updates the processing queue size
func (m *AnalyticsMetrics) UpdateQueueSize(size float64) {
	m.QueueSize.Set(size)
}

// RecordProcessingError records a processing error
func (m *AnalyticsMetrics) RecordProcessingError(operation, errorType string) {
	m.ProcessingErrors.WithLabelValues(operation, errorType).Inc()
}

// RecordIntegrationRequest records an external integration request
func (m *AnalyticsMetrics) RecordIntegrationRequest(integration, operation, status string, duration time.Duration) {
	m.IntegrationRequests.WithLabelValues(integration, operation, status).Inc()
	m.IntegrationLatency.WithLabelValues(integration, operation).Observe(duration.Seconds())
}

// RecordIntegrationError records an external integration error
func (m *AnalyticsMetrics) RecordIntegrationError(integration, operation, errorType string) {
	m.IntegrationErrors.WithLabelValues(integration, operation, errorType).Inc()
}

// RecordDatabaseQuery records a database query
func (m *AnalyticsMetrics) RecordDatabaseQuery(operation, table, status string, duration time.Duration) {
	m.DatabaseQueries.WithLabelValues(operation, table, status).Inc()
	m.QueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
}

// UpdateDatabaseConnections updates the database connections count
func (m *AnalyticsMetrics) UpdateDatabaseConnections(count float64) {
	m.DatabaseConnections.Set(count)
}

// RecordHTTPRequest records an HTTP request
func (m *AnalyticsMetrics) RecordHTTPRequest(method, path string, statusCode int, duration time.Duration) {
	statusCodeStr := m.statusCodeToString(statusCode)
	m.HTTPRequests.WithLabelValues(method, path, statusCodeStr).Inc()
	m.HTTPDuration.WithLabelValues(method, path).Observe(duration.Seconds())
}

// RecordGRPCRequest records a gRPC request
func (m *AnalyticsMetrics) RecordGRPCRequest(method, status string, duration time.Duration) {
	m.GRPCRequests.WithLabelValues(method, status).Inc()
	m.GRPCDuration.WithLabelValues(method).Observe(duration.Seconds())
}

// UpdateSystemMetrics updates system-level metrics
func (m *AnalyticsMetrics) UpdateSystemMetrics(goRoutines int, memoryUsage uint64, cpuUsage float64) {
	m.GoRoutines.Set(float64(goRoutines))
	m.MemoryUsage.Set(float64(memoryUsage))
	m.CPUUsage.Set(cpuUsage)
}

// RecordEventVolume records the volume of event data
func (m *AnalyticsMetrics) RecordEventVolume(clubID, eventType string, bytes int) {
	m.EventVolume.WithLabelValues(clubID, eventType).Add(float64(bytes))
}

// RecordMetricVolume records the volume of metric data
func (m *AnalyticsMetrics) RecordMetricVolume(clubID, metricName string, bytes int) {
	m.MetricVolume.WithLabelValues(clubID, metricName).Add(float64(bytes))
}

// RecordDataExportSize records the size of a data export
func (m *AnalyticsMetrics) RecordDataExportSize(exportType string, size int) {
	m.DataExportSize.WithLabelValues(exportType).Observe(float64(size))
}

// GetMetricsHandler returns HTTP handler for Prometheus metrics
func (m *AnalyticsMetrics) GetMetricsHandler() *prometheus.Registry {
	return prometheus.DefaultRegisterer.(*prometheus.Registry)
}

// statusCodeToString converts HTTP status code to string
func (m *AnalyticsMetrics) statusCodeToString(code int) string {
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