package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	"reciprocal-clubs-backend/services/notification-service/internal/service"
)

// HTTPHandler handles HTTP requests for notification service
type HTTPHandler struct {
	service    *service.NotificationService
	logger     logging.Logger
	monitoring *monitoring.Monitor
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(service *service.NotificationService, logger logging.Logger, monitoring *monitoring.Monitor) *HTTPHandler {
	return &HTTPHandler{
		service:    service,
		logger:     logger,
		monitoring: monitoring,
	}
}

// SetupRoutes configures the HTTP routes
func (h *HTTPHandler) SetupRoutes() http.Handler {
	router := mux.NewRouter()

	// Health and monitoring endpoints
	router.HandleFunc("/health", h.healthCheck).Methods("GET")
	router.HandleFunc("/health/live", h.livenessCheck).Methods("GET")
	router.HandleFunc("/health/ready", h.readinessCheck).Methods("GET")
	router.Handle("/metrics", promhttp.Handler()).Methods("GET")

	// API routes
	api := router.PathPrefix("/api/v1").Subrouter()

	// Notification routes
	api.HandleFunc("/notifications", h.createNotification).Methods("POST")
	api.HandleFunc("/notifications/{id}", h.getNotification).Methods("GET")
	api.HandleFunc("/notifications/{id}/read", h.markAsRead).Methods("POST")
	api.HandleFunc("/clubs/{clubId}/notifications", h.getClubNotifications).Methods("GET")
	api.HandleFunc("/users/{userId}/notifications", h.getUserNotifications).Methods("GET")

	// Template routes
	api.HandleFunc("/templates", h.createTemplate).Methods("POST")
	api.HandleFunc("/clubs/{clubId}/templates", h.getClubTemplates).Methods("GET")

	// Stats routes
	api.HandleFunc("/clubs/{clubId}/stats", h.getNotificationStats).Methods("GET")

	// Admin routes for monitoring and management
	admin := api.PathPrefix("/admin").Subrouter()
	admin.HandleFunc("/process/pending", h.processPendingNotifications).Methods("POST")
	admin.HandleFunc("/process/failed", h.retryFailedNotifications).Methods("POST")
	admin.HandleFunc("/notifications/bulk", h.markMultipleAsRead).Methods("POST")
	admin.HandleFunc("/templates/{id}", h.updateTemplate).Methods("PUT")
	admin.HandleFunc("/templates/{id}", h.deleteTemplate).Methods("DELETE")

	// User preferences routes
	api.HandleFunc("/users/{userId}/preferences", h.getUserPreferences).Methods("GET")
	api.HandleFunc("/users/{userId}/preferences", h.updateUserPreferences).Methods("PUT")

	// Bulk operations
	api.HandleFunc("/notifications/bulk", h.createBulkNotifications).Methods("POST")
	api.HandleFunc("/notifications/send", h.sendImmediate).Methods("POST")

	// Add middleware
	router.Use(h.loggingMiddleware)
	router.Use(h.monitoringMiddleware)

	return router
}

// Health check endpoint - comprehensive health check
func (h *HTTPHandler) healthCheck(w http.ResponseWriter, r *http.Request) {
	healthChecker := h.service.GetHealthChecker()
	health := healthChecker.HealthCheck(r.Context())

	// Set appropriate HTTP status based on health
	statusCode := http.StatusOK
	switch health.Status {
	case "unhealthy":
		statusCode = http.StatusServiceUnavailable
	case "degraded":
		statusCode = http.StatusOK // Still accepting requests but with issues
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(health); err != nil {
		h.logger.Error("Failed to encode health check response", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

// Liveness check endpoint - simple check if service is alive
func (h *HTTPHandler) livenessCheck(w http.ResponseWriter, r *http.Request) {
	healthChecker := h.service.GetHealthChecker()
	healthChecker.LivenessHandler()(w, r)
}

// Readiness check endpoint - check if service is ready to accept traffic
func (h *HTTPHandler) readinessCheck(w http.ResponseWriter, r *http.Request) {
	healthChecker := h.service.GetHealthChecker()
	healthChecker.ReadinessHandler()(w, r)
}

// Notification handlers

func (h *HTTPHandler) createNotification(w http.ResponseWriter, r *http.Request) {
	var req service.CreateNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	notification, err := h.service.CreateNotification(r.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create notification", map[string]interface{}{
			"error": err.Error(),
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to create notification")
		return
	}

	h.writeJSON(w, http.StatusCreated, notification)
}

func (h *HTTPHandler) getNotification(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	notification, err := h.service.GetNotificationByID(r.Context(), uint(id))
	if err != nil {
		h.logger.Error("Failed to get notification", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		h.writeError(w, http.StatusNotFound, "Notification not found")
		return
	}

	h.writeJSON(w, http.StatusOK, notification)
}

func (h *HTTPHandler) markAsRead(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	notification, err := h.service.MarkNotificationAsRead(r.Context(), uint(id))
	if err != nil {
		h.logger.Error("Failed to mark notification as read", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to mark notification as read")
		return
	}

	h.writeJSON(w, http.StatusOK, notification)
}

func (h *HTTPHandler) getClubNotifications(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, err := strconv.ParseUint(vars["clubId"], 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid club ID")
		return
	}

	// Parse query parameters for pagination
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 50 // default
	offset := 0 // default

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	notifications, err := h.service.GetNotificationsByClub(r.Context(), uint(clubID), limit, offset)
	if err != nil {
		h.logger.Error("Failed to get club notifications", map[string]interface{}{
			"error":   err.Error(),
			"club_id": clubID,
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to get notifications")
		return
	}

	h.writeJSON(w, http.StatusOK, notifications)
}

func (h *HTTPHandler) getUserNotifications(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]

	clubIDStr := r.URL.Query().Get("club_id")
	if clubIDStr == "" {
		h.writeError(w, http.StatusBadRequest, "club_id query parameter is required")
		return
	}

	clubID, err := strconv.ParseUint(clubIDStr, 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid club ID")
		return
	}

	// Parse query parameters for pagination
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 50 // default
	offset := 0 // default

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	notifications, err := h.service.GetNotificationsByUser(r.Context(), userID, uint(clubID), limit, offset)
	if err != nil {
		h.logger.Error("Failed to get user notifications", map[string]interface{}{
			"error":   err.Error(),
			"user_id": userID,
			"club_id": clubID,
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to get notifications")
		return
	}

	h.writeJSON(w, http.StatusOK, notifications)
}

// Template handlers

func (h *HTTPHandler) createTemplate(w http.ResponseWriter, r *http.Request) {
	var req service.CreateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	template, err := h.service.CreateNotificationTemplate(r.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create template", map[string]interface{}{
			"error": err.Error(),
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to create template")
		return
	}

	h.writeJSON(w, http.StatusCreated, template)
}

func (h *HTTPHandler) getClubTemplates(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, err := strconv.ParseUint(vars["clubId"], 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid club ID")
		return
	}

	templates, err := h.service.GetNotificationTemplatesByClub(r.Context(), uint(clubID))
	if err != nil {
		h.logger.Error("Failed to get club templates", map[string]interface{}{
			"error":   err.Error(),
			"club_id": clubID,
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to get templates")
		return
	}

	h.writeJSON(w, http.StatusOK, templates)
}

// Stats handlers

func (h *HTTPHandler) getNotificationStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, err := strconv.ParseUint(vars["clubId"], 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid club ID")
		return
	}

	// Parse query parameters for date range
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	fromDate := time.Now().AddDate(0, -1, 0) // default to 1 month ago
	toDate := time.Now()                     // default to now

	if fromStr != "" {
		if parsed, err := time.Parse("2006-01-02", fromStr); err == nil {
			fromDate = parsed
		}
	}

	if toStr != "" {
		if parsed, err := time.Parse("2006-01-02", toStr); err == nil {
			toDate = parsed
		}
	}

	stats, err := h.service.GetNotificationStats(r.Context(), uint(clubID), fromDate, toDate)
	if err != nil {
		h.logger.Error("Failed to get notification stats", map[string]interface{}{
			"error":   err.Error(),
			"club_id": clubID,
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to get stats")
		return
	}

	h.writeJSON(w, http.StatusOK, stats)
}

// Utility methods

func (h *HTTPHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *HTTPHandler) writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

// Middleware

func (h *HTTPHandler) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.logger.Info("HTTP request", map[string]interface{}{
			"method": r.Method,
			"path":   r.URL.Path,
			"remote": r.RemoteAddr,
		})
		next.ServeHTTP(w, r)
	})
}

func (h *HTTPHandler) monitoringMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)

		// Record metrics using our custom notification metrics
		metrics := h.service.GetMetrics()
		metrics.RecordHTTPRequest(r.Method, r.URL.Path, wrapped.statusCode, duration)

		// Also record with the shared monitoring for compatibility
		h.monitoring.RecordHTTPRequest(r.Method, r.URL.Path, wrapped.statusCode, duration)
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

// Admin handlers

func (h *HTTPHandler) processPendingNotifications(w http.ResponseWriter, r *http.Request) {
	count := h.service.ProcessScheduledNotifications(r.Context())

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"processed_count": count,
		"status":          "success",
	})
}

func (h *HTTPHandler) retryFailedNotifications(w http.ResponseWriter, r *http.Request) {
	count := h.service.RetryFailedNotificationsWithCount(r.Context())

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"retried_count": count,
		"status":        "success",
	})
}

func (h *HTTPHandler) markMultipleAsRead(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NotificationIDs []uint `json:"notification_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.NotificationIDs) == 0 {
		h.writeError(w, http.StatusBadRequest, "notification_ids cannot be empty")
		return
	}

	// For now, mark them one by one (could be optimized with bulk repository method)
	successCount := 0
	var failedIDs []uint

	for _, id := range req.NotificationIDs {
		_, err := h.service.MarkNotificationAsRead(r.Context(), id)
		if err != nil {
			failedIDs = append(failedIDs, id)
			h.logger.Warn("Failed to mark notification as read", map[string]interface{}{
				"id":    id,
				"error": err.Error(),
			})
		} else {
			successCount++
		}
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success_count": successCount,
		"failed_ids":    failedIDs,
	})
}

func (h *HTTPHandler) updateTemplate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid template ID")
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// For simplicity, this is a placeholder - would need proper update logic
	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":      id,
		"message": "Template update not fully implemented yet",
		"updates": updates,
	})
}

func (h *HTTPHandler) deleteTemplate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid template ID")
		return
	}

	// For simplicity, this is a placeholder - would need proper delete logic
	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":      id,
		"message": "Template deletion not fully implemented yet",
	})
}

// User preferences handlers

func (h *HTTPHandler) getUserPreferences(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]

	clubIDStr := r.URL.Query().Get("club_id")
	if clubIDStr == "" {
		h.writeError(w, http.StatusBadRequest, "club_id query parameter is required")
		return
	}

	clubID, err := strconv.ParseUint(clubIDStr, 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid club ID")
		return
	}

	// This would call a service method that doesn't exist yet
	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":         userID,
		"club_id":         clubID,
		"email_enabled":   true,
		"sms_enabled":     true,
		"push_enabled":    true,
		"in_app_enabled":  true,
		"timezone":        "UTC",
		"preferred_lang":  "en",
		"message":         "User preferences endpoint - implementation pending",
	})
}

func (h *HTTPHandler) updateUserPreferences(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]

	clubIDStr := r.URL.Query().Get("club_id")
	if clubIDStr == "" {
		h.writeError(w, http.StatusBadRequest, "club_id query parameter is required")
		return
	}

	clubID, err := strconv.ParseUint(clubIDStr, 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid club ID")
		return
	}

	var prefs map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&prefs); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// This would call a service method that doesn't exist yet
	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":     userID,
		"club_id":     clubID,
		"preferences": prefs,
		"message":     "User preferences update endpoint - implementation pending",
	})
}

// Bulk operations handlers

func (h *HTTPHandler) createBulkNotifications(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Notifications []service.CreateNotificationRequest `json:"notifications"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.Notifications) == 0 {
		h.writeError(w, http.StatusBadRequest, "notifications cannot be empty")
		return
	}

	var results []interface{}
	successCount := 0
	errorCount := 0

	for _, notificationReq := range req.Notifications {
		notification, err := h.service.CreateNotification(r.Context(), &notificationReq)
		if err != nil {
			results = append(results, map[string]interface{}{
				"error": err.Error(),
			})
			errorCount++
		} else {
			results = append(results, notification)
			successCount++
		}
	}

	h.writeJSON(w, http.StatusCreated, map[string]interface{}{
		"results":       results,
		"success_count": successCount,
		"error_count":   errorCount,
	})
}

func (h *HTTPHandler) sendImmediate(w http.ResponseWriter, r *http.Request) {
	var req service.CreateNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Set high priority for immediate sending
	req.Priority = "critical"

	notification, err := h.service.CreateNotification(r.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create immediate notification", map[string]interface{}{
			"error": err.Error(),
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to create notification")
		return
	}

	// Process immediately
	go h.service.ProcessNotification(r.Context(), notification.ID)

	h.writeJSON(w, http.StatusCreated, map[string]interface{}{
		"success":      true,
		"message":      "Notification sent immediately",
		"notification": notification,
	})
}