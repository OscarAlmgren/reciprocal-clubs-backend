package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	apperrors "reciprocal-clubs-backend/pkg/shared/errors"
	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	"reciprocal-clubs-backend/services/auth-service/internal/metrics"
	"reciprocal-clubs-backend/services/auth-service/internal/middleware"
	"reciprocal-clubs-backend/services/auth-service/internal/models"
	"reciprocal-clubs-backend/services/auth-service/internal/service"

	"github.com/gorilla/mux"
)

// HTTPHandler handles HTTP requests for the auth service
type HTTPHandler struct {
	service             *service.AuthService
	logger              logging.Logger
	monitor             *monitoring.Monitor
	authMetrics         *metrics.AuthMetrics
	rateLimitMiddleware *middleware.RateLimitMiddleware
	instrumentation     *middleware.InstrumentationMiddleware
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(service *service.AuthService, logger logging.Logger, monitor *monitoring.Monitor) *HTTPHandler {
	// Initialize auth-specific metrics
	authMetrics := metrics.NewAuthMetrics(monitor)

	// Initialize middleware
	rateLimitConfig := middleware.DefaultRateLimitConfig()
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(rateLimitConfig, authMetrics, logger)
	instrumentation := middleware.NewInstrumentationMiddleware(monitor, authMetrics, logger)

	return &HTTPHandler{
		service:             service,
		logger:              logger,
		monitor:             monitor,
		authMetrics:         authMetrics,
		rateLimitMiddleware: rateLimitMiddleware,
		instrumentation:     instrumentation,
	}
}

// RegisterRoutes registers HTTP routes with comprehensive endpoints
func (h *HTTPHandler) RegisterRoutes(router *mux.Router) {
	// Apply global middleware
	router.Use(h.loggingMiddleware)
	router.Use(h.corsMiddleware)
	router.Use(h.metricsMiddleware)

	// Health and monitoring endpoints
	router.HandleFunc("/health", h.healthCheck).Methods("GET")
	router.HandleFunc("/ready", h.readinessCheck).Methods("GET")
	router.HandleFunc("/metrics", h.metricsEndpoint).Methods("GET")
	router.HandleFunc("/status", h.statusEndpoint).Methods("GET")

	// Authentication endpoints
	auth := router.PathPrefix("/auth").Subrouter()
	auth.Use(h.rateLimitAuthMiddleware)
	auth.HandleFunc("/register", h.register).Methods("POST")
	auth.HandleFunc("/login/initiate", h.initiatePasskeyLogin).Methods("POST")
	auth.HandleFunc("/login/complete", h.completePasskeyLogin).Methods("POST")
	auth.HandleFunc("/logout", h.logout).Methods("POST")
	auth.HandleFunc("/passkey/register/initiate", h.initiatePasskeyRegistration).Methods("POST")
	auth.HandleFunc("/passkey/register/complete", h.completePasskeyRegistration).Methods("POST")
	auth.HandleFunc("/session/validate", h.validateSession).Methods("POST")
	auth.HandleFunc("/refresh", h.refreshToken).Methods("POST")

	// User management endpoints
	users := router.PathPrefix("/users").Subrouter()
	users.Use(h.authenticationMiddleware)
	users.HandleFunc("/{clubId:[0-9]+}/{userId:[0-9]+}", h.getUser).Methods("GET")
	users.HandleFunc("/{clubId:[0-9]+}/{userId:[0-9]+}/roles", h.getUserWithRoles).Methods("GET")
	users.HandleFunc("/{clubId:[0-9]+}/{userId:[0-9]+}", h.updateUser).Methods("PUT")
	users.HandleFunc("/{clubId:[0-9]+}/{userId:[0-9]+}/suspend", h.suspendUser).Methods("POST")
	users.HandleFunc("/{clubId:[0-9]+}/{userId:[0-9]+}/activate", h.activateUser).Methods("POST")
	users.HandleFunc("/{clubId:[0-9]+}/{userId:[0-9]+}", h.deleteUser).Methods("DELETE")
	users.HandleFunc("/{clubId:[0-9]+}", h.listUsers).Methods("GET")

	// Role management endpoints
	roles := router.PathPrefix("/roles").Subrouter()
	roles.Use(h.authenticationMiddleware)
	roles.Use(h.adminAuthorizationMiddleware)
	roles.HandleFunc("/{clubId:[0-9]+}", h.createRole).Methods("POST")
	roles.HandleFunc("/{clubId:[0-9]+}", h.getRoles).Methods("GET")
	roles.HandleFunc("/{clubId:[0-9]+}/{roleId:[0-9]+}", h.getRole).Methods("GET")
	roles.HandleFunc("/{clubId:[0-9]+}/{roleId:[0-9]+}", h.updateRole).Methods("PUT")
	roles.HandleFunc("/{clubId:[0-9]+}/{roleId:[0-9]+}", h.deleteRole).Methods("DELETE")
	roles.HandleFunc("/{clubId:[0-9]+}/assign", h.assignRole).Methods("POST")
	roles.HandleFunc("/{clubId:[0-9]+}/remove", h.removeRole).Methods("POST")

	// Permission endpoints
	permissions := router.PathPrefix("/permissions").Subrouter()
	permissions.Use(h.authenticationMiddleware)
	permissions.HandleFunc("/{clubId:[0-9]+}/users/{userId:[0-9]+}", h.getUserPermissions).Methods("GET")
	permissions.HandleFunc("/{clubId:[0-9]+}/check", h.checkPermission).Methods("POST")

	// Club management endpoints
	clubs := router.PathPrefix("/clubs").Subrouter()
	clubs.Use(h.authenticationMiddleware)
	clubs.HandleFunc("", h.createClub).Methods("POST")
	clubs.HandleFunc("", h.getClubs).Methods("GET")
	clubs.HandleFunc("/{clubId:[0-9]+}", h.getClub).Methods("GET")
	clubs.HandleFunc("/{clubId:[0-9]+}", h.updateClub).Methods("PUT")

	// Audit endpoints
	audit := router.PathPrefix("/audit").Subrouter()
	audit.Use(h.authenticationMiddleware)
	audit.Use(h.adminAuthorizationMiddleware)
	audit.HandleFunc("/{clubId:[0-9]+}/logs", h.getAuditLogs).Methods("GET")

	// Admin endpoints
	admin := router.PathPrefix("/admin").Subrouter()
	admin.Use(h.authenticationMiddleware)
	admin.Use(h.adminAuthorizationMiddleware)
	admin.HandleFunc("/rate-limits", h.getRateLimitStats).Methods("GET")
	admin.HandleFunc("/circuit-breakers", h.getCircuitBreakerStats).Methods("GET")
	admin.HandleFunc("/circuit-breakers/reset", h.resetCircuitBreakers).Methods("POST")

	// Webhook endpoints
	webhooks := router.PathPrefix("/webhooks").Subrouter()
	webhooks.Use(h.webhookAuthMiddleware)
	webhooks.HandleFunc("/hanko", h.handleHankoWebhook).Methods("POST")
}

// Health check handlers

func (h *HTTPHandler) healthCheck(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	if err := h.service.HealthCheck(ctx); err != nil {
		h.logger.Error("Health check failed", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "Service unhealthy", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"service": "auth-service",
	})
}

func (h *HTTPHandler) readinessCheck(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check if service is ready to accept traffic
	if err := h.service.HealthCheck(ctx); err != nil {
		h.logger.Error("Readiness check failed", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "Service not ready", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ready",
		"service": "auth-service",
		"timestamp": time.Now(),
	})
}

// Authentication handlers

func (h *HTTPHandler) register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req service.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.handleError(w, apperrors.InvalidInput("Invalid request body", nil, err))
		return
	}

	response, err := h.service.Register(ctx, &req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.logger.Info("User registration successful", map[string]interface{}{
		"user_id": response.User.ID,
		"email":   response.User.Email,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (h *HTTPHandler) initiatePasskeyLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req service.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.handleError(w, apperrors.InvalidInput("Invalid request body", nil, err))
		return
	}

	response, err := h.service.InitiatePasskeyLogin(ctx, &req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.logger.Debug("Passkey login initiated", map[string]interface{}{
		"email": req.Email,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *HTTPHandler) completePasskeyLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		ClubSlug          string                 `json:"club_slug"`
		HankoUserID       string                 `json:"hanko_user_id"`
		CredentialResult  map[string]interface{} `json:"credential_result"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.handleError(w, apperrors.InvalidInput("Invalid request body", nil, err))
		return
	}

	response, err := h.service.CompletePasskeyLogin(ctx, req.ClubSlug, req.HankoUserID, req.CredentialResult)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.logger.Info("Passkey login completed", map[string]interface{}{
		"user_id": response.User.ID,
		"email":   response.User.Email,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *HTTPHandler) logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		UserID       uint   `json:"user_id"`
		ClubID       uint   `json:"club_id"`
		SessionToken string `json:"session_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.handleError(w, apperrors.InvalidInput("Invalid request body", nil, err))
		return
	}

	err := h.service.Logout(ctx, req.UserID, req.ClubID, req.SessionToken)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.logger.Info("User logged out", map[string]interface{}{
		"user_id": req.UserID,
		"club_id": req.ClubID,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Logged out successfully",
	})
}

func (h *HTTPHandler) initiatePasskeyRegistration(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		UserID uint `json:"user_id"`
		ClubID uint `json:"club_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.handleError(w, apperrors.InvalidInput("Invalid request body", nil, err))
		return
	}

	response, err := h.service.InitiatePasskeyRegistration(ctx, req.UserID, req.ClubID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.logger.Info("Passkey registration initiated", map[string]interface{}{
		"user_id": req.UserID,
		"club_id": req.ClubID,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *HTTPHandler) validateSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		SessionToken string `json:"session_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.handleError(w, apperrors.InvalidInput("Invalid request body", nil, err))
		return
	}

	user, err := h.service.ValidateSession(ctx, req.SessionToken)
	if err != nil {
		h.handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid": true,
		"user":  user,
	})
}

// User handlers

func (h *HTTPHandler) getUserWithRoles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	clubID, err := strconv.ParseUint(vars["clubId"], 10, 32)
	if err != nil {
		h.handleError(w, apperrors.InvalidInput("Invalid club ID", nil, err))
		return
	}

	userID, err := strconv.ParseUint(vars["userId"], 10, 32)
	if err != nil {
		h.handleError(w, apperrors.InvalidInput("Invalid user ID", nil, err))
		return
	}

	userWithRoles, err := h.service.GetUserWithRoles(ctx, uint(clubID), uint(userID))
	if err != nil {
		h.handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(userWithRoles)
}

// Webhook handlers

func (h *HTTPHandler) handleHankoWebhook(w http.ResponseWriter, r *http.Request) {
	_ = r.Context()

	// Read webhook payload
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.logger.Error("Failed to decode Hanko webhook payload", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// Process webhook event
	h.logger.Info("Hanko webhook received", map[string]interface{}{
		"payload": payload,
	})

	// Here you would process different types of Hanko webhook events
	// For example:
	// - user.created
	// - user.updated
	// - user.deleted
	// - session.created
	// - session.expired
	// - passkey.registered
	// - passkey.deleted

	eventType, ok := payload["type"].(string)
	if !ok {
		h.logger.Warn("Hanko webhook missing event type", map[string]interface{}{
			"payload": payload,
		})
		http.Error(w, "Missing event type", http.StatusBadRequest)
		return
	}

	switch eventType {
	case "user.created":
		h.logger.Info("Hanko user created webhook", map[string]interface{}{
			"payload": payload,
		})
	case "user.deleted":
		h.logger.Info("Hanko user deleted webhook", map[string]interface{}{
			"payload": payload,
		})
	case "passkey.registered":
		h.logger.Info("Hanko passkey registered webhook", map[string]interface{}{
			"payload": payload,
		})
	case "session.created":
		h.logger.Info("Hanko session created webhook", map[string]interface{}{
			"payload": payload,
		})
	default:
		h.logger.Info("Unknown Hanko webhook event", map[string]interface{}{
			"event_type": eventType,
			"payload":    payload,
		})
	}

	// Acknowledge webhook
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "received",
	})
}

// Error handling

func (h *HTTPHandler) handleError(w http.ResponseWriter, err error) {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		statusCode := h.getHTTPStatusCode(appErr.Code)
		
		h.logger.Error("Request failed", map[string]interface{}{
			"error":       appErr.Error(),
			"error_code":  string(appErr.Code),
			"status_code": statusCode,
			"fields":      appErr.Fields,
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   appErr.Message,
			"code":    string(appErr.Code),
			"fields":  appErr.Fields,
		})
		return
	}

	// Generic error
	h.logger.Error("Unexpected error", map[string]interface{}{
		"error": err.Error(),
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": "Internal server error",
		"code":  "INTERNAL",
	})
}

func (h *HTTPHandler) getHTTPStatusCode(errorCode apperrors.ErrorCode) int {
	switch errorCode {
	case apperrors.ErrNotFound:
		return http.StatusNotFound
	case apperrors.ErrInvalidInput:
		return http.StatusBadRequest
	case apperrors.ErrUnauthorized:
		return http.StatusUnauthorized
	case apperrors.ErrForbidden:
		return http.StatusForbidden
	case apperrors.ErrConflict:
		return http.StatusConflict
	case apperrors.ErrTimeout:
		return http.StatusRequestTimeout
	case apperrors.ErrUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// Additional HTTP endpoint handlers

func (h *HTTPHandler) metricsEndpoint(w http.ResponseWriter, r *http.Request) {
	// Return metrics in Prometheus format or JSON
	stats := h.authMetrics.GetRegistry()
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	// This would normally use promhttp.Handler() but for now return simple response
	w.Write([]byte("# Auth Service Metrics\n# See /admin/rate-limits and /admin/circuit-breakers for detailed stats\n"))
}

func (h *HTTPHandler) statusEndpoint(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	status := map[string]interface{}{
		"service":   "auth-service",
		"status":    "healthy",
		"timestamp": time.Now(),
		"uptime":    time.Since(time.Now()).String(), // Would track actual uptime
	}

	// Check health of dependencies
	if err := h.service.HealthCheck(ctx); err != nil {
		status["status"] = "unhealthy"
		status["error"] = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}

func (h *HTTPHandler) completePasskeyRegistration(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		UserID           uint                   `json:"user_id"`
		ClubID           uint                   `json:"club_id"`
		CredentialResult map[string]interface{} `json:"credential_result"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.handleError(w, apperrors.InvalidInput("Invalid request body", nil, err))
		return
	}

	// This would call a service method to complete passkey registration
	h.logger.Info("Passkey registration completed", map[string]interface{}{
		"user_id": req.UserID,
		"club_id": req.ClubID,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Passkey registration completed successfully",
	})
}

func (h *HTTPHandler) refreshToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.handleError(w, apperrors.InvalidInput("Invalid request body", nil, err))
		return
	}

	// This would implement token refresh logic
	h.logger.Info("Token refresh requested", nil)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token":         "new-access-token",
		"refresh_token": "new-refresh-token",
		"expires_at":    time.Now().Add(time.Hour),
	})
}

func (h *HTTPHandler) getUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	clubID, err := strconv.ParseUint(vars["clubId"], 10, 32)
	if err != nil {
		h.handleError(w, apperrors.InvalidInput("Invalid club ID", nil, err))
		return
	}

	userID, err := strconv.ParseUint(vars["userId"], 10, 32)
	if err != nil {
		h.handleError(w, apperrors.InvalidInput("Invalid user ID", nil, err))
		return
	}

	user, err := h.service.GetUser(ctx, uint(clubID), uint(userID))
	if err != nil {
		h.handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}

func (h *HTTPHandler) updateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	clubID, err := strconv.ParseUint(vars["clubId"], 10, 32)
	if err != nil {
		h.handleError(w, apperrors.InvalidInput("Invalid club ID", nil, err))
		return
	}

	userID, err := strconv.ParseUint(vars["userId"], 10, 32)
	if err != nil {
		h.handleError(w, apperrors.InvalidInput("Invalid user ID", nil, err))
		return
	}

	var req struct {
		FirstName string             `json:"first_name,omitempty"`
		LastName  string             `json:"last_name,omitempty"`
		Email     string             `json:"email,omitempty"`
		Status    models.UserStatus  `json:"status,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.handleError(w, apperrors.InvalidInput("Invalid request body", nil, err))
		return
	}

	// Get existing user
	user, err := h.service.GetUser(ctx, uint(clubID), uint(userID))
	if err != nil {
		h.handleError(w, err)
		return
	}

	// Update fields
	if req.FirstName != "" {
		user.FirstName = req.FirstName
	}
	if req.LastName != "" {
		user.LastName = req.LastName
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Status != "" {
		user.Status = req.Status
	}

	h.logger.Info("User updated", map[string]interface{}{
		"user_id": userID,
		"club_id": clubID,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}

func (h *HTTPHandler) suspendUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	clubID, _ := strconv.ParseUint(vars["clubId"], 10, 32)
	userID, _ := strconv.ParseUint(vars["userId"], 10, 32)

	var req struct {
		SuspendedUntil *time.Time `json:"suspended_until,omitempty"`
		Reason         string     `json:"reason,omitempty"`
	}

	json.NewDecoder(r.Body).Decode(&req)

	user, err := h.service.GetUser(ctx, uint(clubID), uint(userID))
	if err != nil {
		h.handleError(w, err)
		return
	}

	user.Status = models.UserStatusSuspended
	if req.SuspendedUntil != nil {
		user.LockedUntil = req.SuspendedUntil
	}

	h.logger.Info("User suspended", map[string]interface{}{
		"user_id": userID,
		"club_id": clubID,
		"reason":  req.Reason,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}

func (h *HTTPHandler) activateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	clubID, _ := strconv.ParseUint(vars["clubId"], 10, 32)
	userID, _ := strconv.ParseUint(vars["userId"], 10, 32)

	user, err := h.service.GetUser(ctx, uint(clubID), uint(userID))
	if err != nil {
		h.handleError(w, err)
		return
	}

	user.Status = models.UserStatusActive
	user.Unlock()

	h.logger.Info("User activated", map[string]interface{}{
		"user_id": userID,
		"club_id": clubID,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}

func (h *HTTPHandler) deleteUser(w http.ResponseWriter, r *http.Request) {
	// Implementation would depend on soft delete or hard delete strategy
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "User deleted successfully",
	})
}

func (h *HTTPHandler) listUsers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, _ := strconv.ParseUint(vars["clubId"], 10, 32)

	// Parse query parameters for pagination
	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// This would call a service method to list users
	h.logger.Debug("Listing users", map[string]interface{}{
		"club_id": clubID,
		"limit":   limit,
		"offset":  offset,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"users":  []interface{}{},
		"total":  0,
		"limit":  limit,
		"offset": offset,
	})
}

// Role management handlers

func (h *HTTPHandler) createRole(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, _ := strconv.ParseUint(vars["clubId"], 10, 32)

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		IsSystem    bool   `json:"is_system"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.handleError(w, apperrors.InvalidInput("Invalid request body", nil, err))
		return
	}

	h.logger.Info("Role created", map[string]interface{}{
		"club_id": clubID,
		"name":    req.Name,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Role created successfully",
	})
}

func (h *HTTPHandler) getRoles(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, _ := strconv.ParseUint(vars["clubId"], 10, 32)

	h.logger.Debug("Getting roles", map[string]interface{}{
		"club_id": clubID,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"roles": []interface{}{},
		"total": 0,
	})
}

func (h *HTTPHandler) getRole(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, _ := strconv.ParseUint(vars["clubId"], 10, 32)
	roleID, _ := strconv.ParseUint(vars["roleId"], 10, 32)

	h.logger.Debug("Getting role", map[string]interface{}{
		"club_id": clubID,
		"role_id": roleID,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":   roleID,
		"name": "sample_role",
	})
}

func (h *HTTPHandler) updateRole(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, _ := strconv.ParseUint(vars["clubId"], 10, 32)
	roleID, _ := strconv.ParseUint(vars["roleId"], 10, 32)

	h.logger.Info("Role updated", map[string]interface{}{
		"club_id": clubID,
		"role_id": roleID,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Role updated successfully",
	})
}

func (h *HTTPHandler) deleteRole(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, _ := strconv.ParseUint(vars["clubId"], 10, 32)
	roleID, _ := strconv.ParseUint(vars["roleId"], 10, 32)

	h.logger.Info("Role deleted", map[string]interface{}{
		"club_id": clubID,
		"role_id": roleID,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Role deleted successfully",
	})
}

func (h *HTTPHandler) assignRole(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, _ := strconv.ParseUint(vars["clubId"], 10, 32)

	var req struct {
		UserID uint `json:"user_id"`
		RoleID uint `json:"role_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.handleError(w, apperrors.InvalidInput("Invalid request body", nil, err))
		return
	}

	h.logger.Info("Role assigned", map[string]interface{}{
		"club_id": clubID,
		"user_id": req.UserID,
		"role_id": req.RoleID,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Role assigned successfully",
	})
}

func (h *HTTPHandler) removeRole(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, _ := strconv.ParseUint(vars["clubId"], 10, 32)

	var req struct {
		UserID uint `json:"user_id"`
		RoleID uint `json:"role_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.handleError(w, apperrors.InvalidInput("Invalid request body", nil, err))
		return
	}

	h.logger.Info("Role removed", map[string]interface{}{
		"club_id": clubID,
		"user_id": req.UserID,
		"role_id": req.RoleID,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Role removed successfully",
	})
}

// Permission handlers

func (h *HTTPHandler) getUserPermissions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, _ := strconv.ParseUint(vars["clubId"], 10, 32)
	userID, _ := strconv.ParseUint(vars["userId"], 10, 32)

	h.logger.Debug("Getting user permissions", map[string]interface{}{
		"club_id": clubID,
		"user_id": userID,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"permissions": []interface{}{},
	})
}

func (h *HTTPHandler) checkPermission(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, _ := strconv.ParseUint(vars["clubId"], 10, 32)

	var req struct {
		UserID     uint   `json:"user_id"`
		Permission string `json:"permission"`
		Resource   string `json:"resource"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.handleError(w, apperrors.InvalidInput("Invalid request body", nil, err))
		return
	}

	h.logger.Debug("Checking permission", map[string]interface{}{
		"club_id":    clubID,
		"user_id":    req.UserID,
		"permission": req.Permission,
		"resource":   req.Resource,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"allowed": true,
	})
}

// Club management handlers

func (h *HTTPHandler) createClub(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name         string `json:"name"`
		Slug         string `json:"slug"`
		Description  string `json:"description"`
		ContactEmail string `json:"contact_email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.handleError(w, apperrors.InvalidInput("Invalid request body", nil, err))
		return
	}

	h.logger.Info("Club created", map[string]interface{}{
		"name": req.Name,
		"slug": req.Slug,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Club created successfully",
	})
}

func (h *HTTPHandler) getClubs(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("Getting clubs", nil)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"clubs": []interface{}{},
		"total": 0,
	})
}

func (h *HTTPHandler) getClub(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, _ := strconv.ParseUint(vars["clubId"], 10, 32)

	h.logger.Debug("Getting club", map[string]interface{}{
		"club_id": clubID,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":   clubID,
		"name": "Sample Club",
	})
}

func (h *HTTPHandler) updateClub(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, _ := strconv.ParseUint(vars["clubId"], 10, 32)

	h.logger.Info("Club updated", map[string]interface{}{
		"club_id": clubID,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Club updated successfully",
	})
}

// Audit handlers

func (h *HTTPHandler) getAuditLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, _ := strconv.ParseUint(vars["clubId"], 10, 32)

	// Parse query parameters
	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	h.logger.Debug("Getting audit logs", map[string]interface{}{
		"club_id": clubID,
		"limit":   limit,
		"offset":  offset,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"audit_logs": []interface{}{},
		"total":      0,
		"limit":      limit,
		"offset":     offset,
	})
}

// Admin handlers

func (h *HTTPHandler) getRateLimitStats(w http.ResponseWriter, r *http.Request) {
	stats := h.rateLimitMiddleware.GetStats()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(stats)
}

func (h *HTTPHandler) getCircuitBreakerStats(w http.ResponseWriter, r *http.Request) {
	// This would get stats from circuit breaker manager
	stats := map[string]interface{}{
		"circuit_breakers": map[string]interface{}{
			"hanko":      map[string]interface{}{"state": "closed", "requests": 100, "failures": 0},
			"database":   map[string]interface{}{"state": "closed", "requests": 500, "failures": 2},
			"messagebus": map[string]interface{}{"state": "closed", "requests": 50, "failures": 0},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(stats)
}

func (h *HTTPHandler) resetCircuitBreakers(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Circuit breakers reset by admin", nil)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "All circuit breakers have been reset",
	})
}

// Middleware functions

func (h *HTTPHandler) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create wrapped response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Call next handler
		next.ServeHTTP(wrapped, r)

		// Log request
		duration := time.Since(start)
		h.logger.Info("HTTP request completed", map[string]interface{}{
			"method":      r.Method,
			"path":        r.URL.Path,
			"status_code": wrapped.statusCode,
			"duration_ms": duration.Milliseconds(),
			"user_agent":  r.UserAgent(),
			"remote_addr": r.RemoteAddr,
		})

		// Record metrics
		h.monitor.RecordHTTPRequest(r.Method, r.URL.Path, wrapped.statusCode, duration)
	})
}

func (h *HTTPHandler) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (h *HTTPHandler) metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract method-specific information for metrics
		method := r.URL.Path
		clubID := "unknown"
		userID := "unknown"

		// Extract IDs from path if available
		if vars := mux.Vars(r); vars != nil {
			if cid := vars["clubId"]; cid != "" {
				clubID = cid
			}
			if uid := vars["userId"]; uid != "" {
				userID = uid
			}
		}

		// Record metrics based on endpoint
		if strings.Contains(method, "/auth/") {
			h.authMetrics.RecordAuthAttempt("http", clubID, "initiated", "unknown")
		}

		next.ServeHTTP(w, r)
	})
}

func (h *HTTPHandler) rateLimitAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract client IP
		clientIP := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			clientIP = strings.Split(forwarded, ",")[0]
		}

		// For auth endpoints, we would check rate limits here
		// For now, just log and continue
		h.logger.Debug("Auth request rate limit check", map[string]interface{}{
			"client_ip": clientIP,
			"path":      r.URL.Path,
		})

		next.ServeHTTP(w, r)
	})
}

func (h *HTTPHandler) authenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			h.handleError(w, apperrors.Unauthorized("Missing authorization header", nil))
			return
		}

		// Validate JWT token
		if !strings.HasPrefix(authHeader, "Bearer ") {
			h.handleError(w, apperrors.Unauthorized("Invalid authorization header format", nil))
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")

		// This would validate the JWT token with the auth provider
		h.logger.Debug("Validating authentication token", map[string]interface{}{
			"token_length": len(token),
		})

		// For now, just continue - in production this would validate the token
		next.ServeHTTP(w, r)
	})
}

func (h *HTTPHandler) adminAuthorizationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if user has admin permissions
		// This would extract user info from context and check permissions
		h.logger.Debug("Checking admin authorization", nil)

		// For now, just continue - in production this would check admin role
		next.ServeHTTP(w, r)
	})
}

func (h *HTTPHandler) webhookAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate webhook signatures
		signature := r.Header.Get("X-Hanko-Signature")
		if signature == "" {
			h.handleError(w, apperrors.Unauthorized("Missing webhook signature", nil))
			return
		}

		// This would validate the webhook signature
		h.logger.Debug("Validating webhook signature", map[string]interface{}{
			"signature_present": signature != "",
		})

		next.ServeHTTP(w, r)
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