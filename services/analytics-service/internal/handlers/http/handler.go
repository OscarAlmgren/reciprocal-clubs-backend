package http

import (
	"encoding/json"
	"net/http"

	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	"reciprocal-clubs-backend/services/analytics-service/internal/service"

	"github.com/gorilla/mux"
)

type HTTPHandler struct {
	service    service.AnalyticsService
	logger     logging.Logger
	monitoring monitoring.Service
}

func NewHTTPHandler(service service.AnalyticsService, logger logging.Logger, monitoring monitoring.Service) *HTTPHandler {
	return &HTTPHandler{
		service:    service,
		logger:     logger,
		monitoring: monitoring,
	}
}

func (h *HTTPHandler) SetupRoutes() http.Handler {
	router := mux.NewRouter()

	// Health endpoints
	router.HandleFunc("/health", h.HealthCheck).Methods("GET")
	router.HandleFunc("/ready", h.ReadinessCheck).Methods("GET")

	// Analytics endpoints
	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/analytics/metrics", h.GetMetrics).Methods("GET")
	api.HandleFunc("/analytics/reports", h.GetReports).Methods("GET")
	api.HandleFunc("/analytics/events", h.RecordEvent).Methods("POST")

	// Add middleware
	router.Use(h.LoggingMiddleware)
	router.Use(h.MonitoringMiddleware)

	return router
}

func (h *HTTPHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"service": "analytics-service",
		"timestamp": "2023-01-01T00:00:00Z", // Use actual timestamp
	})
}

func (h *HTTPHandler) ReadinessCheck(w http.ResponseWriter, r *http.Request) {
	// Check dependencies (database, NATS, etc.)
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

func (h *HTTPHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	clubID := r.URL.Query().Get("club_id")
	timeRange := r.URL.Query().Get("time_range")

	metrics, err := h.service.GetMetrics(clubID, timeRange)
	if err != nil {
		h.logger.Error("Failed to get metrics: " + err.Error())
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
		h.logger.Error("Failed to get reports: " + err.Error())
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
		h.logger.Error("Failed to record event: " + err.Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "event recorded"})
}

func (h *HTTPHandler) LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.logger.Info("HTTP Request: " + r.Method + " " + r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func (h *HTTPHandler) MonitoringMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Record metrics
		h.monitoring.IncrementCounter("http_requests_total", map[string]string{
			"method": r.Method,
			"path":   r.URL.Path,
		})

		next.ServeHTTP(w, r)
	})
}
