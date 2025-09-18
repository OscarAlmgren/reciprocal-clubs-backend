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
	api.HandleFunc("/analytics/metrics", h.GetMetrics).Methods("GET")
	api.HandleFunc("/analytics/reports", h.GetReports).Methods("GET")
	api.HandleFunc("/analytics/events", h.RecordEvent).Methods("POST")

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
