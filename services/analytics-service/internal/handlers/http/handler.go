package http

import (
	"encoding/json"
	"net/http"
	"time"

	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	"reciprocal-clubs-backend/services/analytics-service/internal/service"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type HTTPHandler struct {
	service    service.AnalyticsService
	logger     logging.Logger
	monitoring *monitoring.Monitor
}

func NewHTTPHandler(service service.AnalyticsService, logger logging.Logger, monitor *monitoring.Monitor) *HTTPHandler {
	return &HTTPHandler{
		service:    service,
		logger:     logger,
		monitoring: monitor,
	}
}

func (h *HTTPHandler) SetupRoutes() http.Handler {
	router := mux.NewRouter()

	// Health endpoints
	router.HandleFunc("/health", h.HealthCheck).Methods("GET")
	router.HandleFunc("/ready", h.ReadinessCheck).Methods("GET")
	router.HandleFunc("/live", h.LivenessCheck).Methods("GET")

	// Monitoring endpoints
	router.Handle("/metrics", promhttp.Handler()).Methods("GET")

	// Analytics endpoints
	api := router.PathPrefix("/api/v1").Subrouter()

	// Core analytics
	api.HandleFunc("/analytics/metrics", h.GetMetrics).Methods("GET")
	api.HandleFunc("/analytics/metrics", h.RecordMetric).Methods("POST")
	api.HandleFunc("/analytics/reports", h.GetReports).Methods("GET")
	api.HandleFunc("/analytics/reports/generate", h.GenerateReport).Methods("POST")
	api.HandleFunc("/analytics/events", h.GetEvents).Methods("GET")
	api.HandleFunc("/analytics/events", h.RecordEvent).Methods("POST")
	api.HandleFunc("/analytics/events/bulk", h.BulkRecordEvents).Methods("POST")

	// Real-time analytics
	api.HandleFunc("/analytics/realtime/metrics", h.GetRealtimeMetrics).Methods("GET")
	api.HandleFunc("/analytics/live/stats", h.GetLiveStats).Methods("GET")

	// Dashboard operations
	api.HandleFunc("/analytics/dashboards", h.ListDashboards).Methods("GET")
	api.HandleFunc("/analytics/dashboards", h.CreateDashboard).Methods("POST")
	api.HandleFunc("/analytics/dashboards/{id}", h.GetDashboard).Methods("GET")
	api.HandleFunc("/analytics/dashboards/{id}", h.UpdateDashboard).Methods("PUT")
	api.HandleFunc("/analytics/dashboards/{id}", h.DeleteDashboard).Methods("DELETE")

	// Data export
	api.HandleFunc("/analytics/export/events", h.ExportEvents).Methods("GET")
	api.HandleFunc("/analytics/export/metrics", h.ExportMetrics).Methods("GET")
	api.HandleFunc("/analytics/export/reports", h.ExportReports).Methods("GET")

	// System operations
	api.HandleFunc("/analytics/system/health", h.GetSystemHealth).Methods("GET")
	api.HandleFunc("/analytics/system/cleanup", h.CleanupOldData).Methods("POST")

	// Add middleware
	router.Use(h.LoggingMiddleware)
	router.Use(h.MonitoringMiddleware)

	return router
}

func (h *HTTPHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	// Use the comprehensive health checker
	healthChecker := h.service.GetHealthChecker()
	if healthChecker != nil {
		handler := healthChecker.HTTPHealthHandler()
		handler.ServeHTTP(w, r)
	} else {
		// Fallback to simple health check
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "healthy",
			"service": "analytics-service",
			"timestamp": time.Now(),
		})
	}
}

func (h *HTTPHandler) ReadinessCheck(w http.ResponseWriter, r *http.Request) {
	// Use the comprehensive readiness checker
	healthChecker := h.service.GetHealthChecker()
	if healthChecker != nil {
		handler := healthChecker.ReadinessHandler()
		handler.ServeHTTP(w, r)
	} else {
		// Fallback to simple readiness check
		ready := h.service.IsReady()
		w.Header().Set("Content-Type", "application/json")
		if ready {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "ready",
				"service": "analytics-service",
			})
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "not ready",
				"service": "analytics-service",
			})
		}
	}
}

func (h *HTTPHandler) LivenessCheck(w http.ResponseWriter, r *http.Request) {
	// Use the comprehensive liveness checker
	healthChecker := h.service.GetHealthChecker()
	if healthChecker != nil {
		handler := healthChecker.LivenessHandler()
		handler.ServeHTTP(w, r)
	} else {
		// Fallback to simple liveness check
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"alive": true,
			"timestamp": time.Now(),
			"service": "analytics-service",
		})
	}
}

func (h *HTTPHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	clubID := r.URL.Query().Get("club_id")
	timeRange := r.URL.Query().Get("time_range")

	metrics, err := h.service.GetMetrics(clubID, timeRange)
	if err != nil {
		h.logger.Error("Failed to get metrics", map[string]interface{}{"error": err.Error(), "club_id": clubID})
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func (h *HTTPHandler) GetReports(w http.ResponseWriter, r *http.Request) {
	clubID := r.URL.Query().Get("club_id")
	reportType := r.URL.Query().Get("type")

	reports, err := h.service.GetReports(clubID, reportType)
	if err != nil {
		h.logger.Error("Failed to get reports", map[string]interface{}{"error": err.Error(), "club_id": clubID})
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reports)
}

func (h *HTTPHandler) RecordEvent(w http.ResponseWriter, r *http.Request) {
	var event map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := h.service.RecordEvent(event); err != nil {
		h.logger.Error("Failed to record event", map[string]interface{}{"error": err.Error(), "event": event})
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "event recorded"})
}

func (h *HTTPHandler) LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.logger.Info("HTTP Request", map[string]interface{}{"method": r.Method, "path": r.URL.Path})
		next.ServeHTTP(w, r)
	})
}

func (h *HTTPHandler) MonitoringMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer that captures the status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Process the request
		next.ServeHTTP(rw, r)

		// Record metrics
		duration := time.Since(start)
		h.monitoring.RecordHTTPRequest(r.Method, r.URL.Path, rw.statusCode, duration)

		// Record detailed metrics if analytics metrics are available
		if analyticsMetrics := h.service.GetMonitoringMetrics(); analyticsMetrics != nil {
			analyticsMetrics.RecordHTTPRequest(r.Method, r.URL.Path, rw.statusCode, duration)
		}
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Additional HTTP handler methods

func (h *HTTPHandler) RecordMetric(w http.ResponseWriter, r *http.Request) {
	var metricRequest struct {
		ClubID      string                 `json:"club_id"`
		MetricName  string                 `json:"metric_name"`
		MetricValue float64                `json:"metric_value"`
		Tags        map[string]interface{} `json:"tags"`
	}

	if err := json.NewDecoder(r.Body).Decode(&metricRequest); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	err := h.service.RecordMetric(metricRequest.ClubID, metricRequest.MetricName, metricRequest.MetricValue, metricRequest.Tags)
	if err != nil {
		h.logger.Error("Failed to record metric", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "metric recorded"})
}

func (h *HTTPHandler) GenerateReport(w http.ResponseWriter, r *http.Request) {
	var reportRequest struct {
		ClubID     string `json:"club_id"`
		ReportType string `json:"report_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reportRequest); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	report, err := h.service.GenerateReport(reportRequest.ClubID, reportRequest.ReportType)
	if err != nil {
		h.logger.Error("Failed to generate report", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

func (h *HTTPHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
	clubID := r.URL.Query().Get("club_id")
	timeRange := r.URL.Query().Get("time_range")

	events, err := h.service.GetEvents(clubID, timeRange)
	if err != nil {
		h.logger.Error("Failed to get events", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

func (h *HTTPHandler) BulkRecordEvents(w http.ResponseWriter, r *http.Request) {
	var bulkRequest struct {
		Events []map[string]interface{} `json:"events"`
	}

	if err := json.NewDecoder(r.Body).Decode(&bulkRequest); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	processedCount := 0
	errorCount := 0
	var errors []string

	for _, event := range bulkRequest.Events {
		if err := h.service.RecordEvent(event); err != nil {
			errorCount++
			errors = append(errors, err.Error())
		} else {
			processedCount++
		}
	}

	response := map[string]interface{}{
		"processed_count": processedCount,
		"error_count":     errorCount,
		"errors":          errors,
		"success":         errorCount == 0,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *HTTPHandler) GetRealtimeMetrics(w http.ResponseWriter, r *http.Request) {
	clubID := r.URL.Query().Get("club_id")

	metrics, err := h.service.GetRealtimeMetrics(clubID)
	if err != nil {
		h.logger.Error("Failed to get realtime metrics", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func (h *HTTPHandler) GetLiveStats(w http.ResponseWriter, r *http.Request) {
	clubID := r.URL.Query().Get("club_id")

	stats, err := h.service.GetRealtimeMetrics(clubID)
	if err != nil {
		h.logger.Error("Failed to get live stats", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *HTTPHandler) ListDashboards(w http.ResponseWriter, r *http.Request) {
	clubID := r.URL.Query().Get("club_id")

	// Mock implementation - would integrate with service layer
	dashboards := []map[string]interface{}{
		{
			"id":          1,
			"name":        "Analytics Dashboard",
			"description": "Main analytics dashboard",
			"club_id":     clubID,
			"is_public":   true,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"dashboards": dashboards,
		"total":      len(dashboards),
	})
}

func (h *HTTPHandler) CreateDashboard(w http.ResponseWriter, r *http.Request) {
	var dashboardRequest struct {
		ClubID      string                 `json:"club_id"`
		Name        string                 `json:"name"`
		Description string                 `json:"description"`
		Panels      map[string]interface{} `json:"panels"`
		IsPublic    bool                   `json:"is_public"`
	}

	if err := json.NewDecoder(r.Body).Decode(&dashboardRequest); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	err := h.service.CreateDashboard(dashboardRequest.ClubID)
	if err != nil {
		h.logger.Error("Failed to create dashboard", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success":    true,
		"message":    "Dashboard created successfully",
		"dashboard": dashboardRequest,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (h *HTTPHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dashboardID := vars["id"]

	// Mock implementation
	dashboard := map[string]interface{}{
		"id":          dashboardID,
		"name":        "Analytics Dashboard",
		"description": "Main analytics dashboard",
		"panels":      map[string]interface{}{},
		"is_public":   true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dashboard)
}

func (h *HTTPHandler) UpdateDashboard(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dashboardID := vars["id"]

	var updateRequest map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updateRequest); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Mock implementation
	updateRequest["id"] = dashboardID
	updateRequest["updated_at"] = time.Now()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"message":   "Dashboard updated successfully",
		"dashboard": updateRequest,
	})
}

func (h *HTTPHandler) DeleteDashboard(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dashboardID := vars["id"]

	// Mock implementation
	h.logger.Info("Dashboard deleted", map[string]interface{}{"dashboard_id": dashboardID})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Dashboard deleted successfully",
	})
}

func (h *HTTPHandler) ExportEvents(w http.ResponseWriter, r *http.Request) {
	clubID := r.URL.Query().Get("club_id")
	format := r.URL.Query().Get("format")
	timeRange := r.URL.Query().Get("time_range")

	if format == "" {
		format = "json"
	}

	events, err := h.service.GetEvents(clubID, timeRange)
	if err != nil {
		h.logger.Error("Failed to get events for export", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	switch format {
	case "json":
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=events.json")
		json.NewEncoder(w).Encode(events)
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=events.csv")
		// Mock CSV implementation
		w.Write([]byte("id,club_id,event_type,timestamp\n"))
		for range events {
			w.Write([]byte("1," + clubID + ",sample_event," + time.Now().Format(time.RFC3339) + "\n"))
		}
	default:
		http.Error(w, "Unsupported format", http.StatusBadRequest)
	}
}

func (h *HTTPHandler) ExportMetrics(w http.ResponseWriter, r *http.Request) {
	clubID := r.URL.Query().Get("club_id")
	format := r.URL.Query().Get("format")
	timeRange := r.URL.Query().Get("time_range")

	if format == "" {
		format = "json"
	}

	metrics, err := h.service.GetMetrics(clubID, timeRange)
	if err != nil {
		h.logger.Error("Failed to get metrics for export", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	switch format {
	case "json":
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=metrics.json")
		json.NewEncoder(w).Encode(metrics)
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=metrics.csv")
		w.Write([]byte("metric_name,value,timestamp\n"))
		w.Write([]byte("sample_metric,100," + time.Now().Format(time.RFC3339) + "\n"))
	default:
		http.Error(w, "Unsupported format", http.StatusBadRequest)
	}
}

func (h *HTTPHandler) ExportReports(w http.ResponseWriter, r *http.Request) {
	clubID := r.URL.Query().Get("club_id")
	format := r.URL.Query().Get("format")

	if format == "" {
		format = "json"
	}

	reports, err := h.service.GetReports(clubID, "")
	if err != nil {
		h.logger.Error("Failed to get reports for export", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	switch format {
	case "json":
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=reports.json")
		json.NewEncoder(w).Encode(reports)
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=reports.csv")
		w.Write([]byte("id,report_type,title,generated_at\n"))
		for range reports {
			w.Write([]byte("1,usage,Sample Report," + time.Now().Format(time.RFC3339) + "\n"))
		}
	default:
		http.Error(w, "Unsupported format", http.StatusBadRequest)
	}
}

func (h *HTTPHandler) GetSystemHealth(w http.ResponseWriter, r *http.Request) {
	health := h.service.GetSystemHealth()

	statusCode := http.StatusOK
	if status, ok := health["status"].(string); ok && status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(health)
}

func (h *HTTPHandler) CleanupOldData(w http.ResponseWriter, r *http.Request) {
	var cleanupRequest struct {
		Days int `json:"days"`
	}

	if err := json.NewDecoder(r.Body).Decode(&cleanupRequest); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	err := h.service.CleanupOldData(cleanupRequest.Days)
	if err != nil {
		h.logger.Error("Failed to cleanup old data", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Old data cleanup completed successfully",
	})
}
