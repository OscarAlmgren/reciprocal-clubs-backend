package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"reciprocal-clubs-backend/pkg/shared/errors"
	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	"reciprocal-clubs-backend/services/auth-service/internal/service"

	"github.com/gorilla/mux"
)

// HTTPHandler handles HTTP requests for the auth service
type HTTPHandler struct {
	service *service.AuthService
	logger  logging.Logger
	monitor *monitoring.Monitor
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(service *service.AuthService, logger logging.Logger, monitor *monitoring.Monitor) *HTTPHandler {
	return &HTTPHandler{
		service: service,
		logger:  logger,
		monitor: monitor,
	}
}

// RegisterRoutes registers HTTP routes
func (h *HTTPHandler) RegisterRoutes(router *mux.Router) {
	// Health check endpoints
	router.HandleFunc("/health", h.healthCheck).Methods("GET")
	router.HandleFunc("/ready", h.readinessCheck).Methods("GET")

	// Authentication endpoints
	auth := router.PathPrefix("/auth").Subrouter()
	auth.HandleFunc("/register", h.register).Methods("POST")
	auth.HandleFunc("/login/initiate", h.initiatePasskeyLogin).Methods("POST")
	auth.HandleFunc("/login/complete", h.completePasskeyLogin).Methods("POST")
	auth.HandleFunc("/logout", h.logout).Methods("POST")
	auth.HandleFunc("/passkey/register/initiate", h.initiatePasskeyRegistration).Methods("POST")
	auth.HandleFunc("/session/validate", h.validateSession).Methods("POST")

	// User endpoints
	users := router.PathPrefix("/users").Subrouter()
	users.HandleFunc("/{clubId:[0-9]+}/{userId:[0-9]+}", h.getUserWithRoles).Methods("GET")

	// Webhook endpoint for Hanko
	router.HandleFunc("/webhook/hanko", h.handleHankoWebhook).Methods("POST")
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
	// For now, same as health check
	h.healthCheck(w, r)
}

// Authentication handlers

func (h *HTTPHandler) register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req service.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.handleError(w, errors.InvalidInput("Invalid request body", nil, err))
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
		h.handleError(w, errors.InvalidInput("Invalid request body", nil, err))
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
		h.handleError(w, errors.InvalidInput("Invalid request body", nil, err))
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
		h.handleError(w, errors.InvalidInput("Invalid request body", nil, err))
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
		h.handleError(w, errors.InvalidInput("Invalid request body", nil, err))
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
		h.handleError(w, errors.InvalidInput("Invalid request body", nil, err))
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
		h.handleError(w, errors.InvalidInput("Invalid club ID", nil, err))
		return
	}

	userID, err := strconv.ParseUint(vars["userId"], 10, 32)
	if err != nil {
		h.handleError(w, errors.InvalidInput("Invalid user ID", nil, err))
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
	ctx := r.Context()

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
	var appErr *errors.AppError
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

func (h *HTTPHandler) getHTTPStatusCode(errorCode errors.ErrorCode) int {
	switch errorCode {
	case errors.ErrNotFound:
		return http.StatusNotFound
	case errors.ErrInvalidInput:
		return http.StatusBadRequest
	case errors.ErrUnauthorized:
		return http.StatusUnauthorized
	case errors.ErrForbidden:
		return http.StatusForbidden
	case errors.ErrConflict:
		return http.StatusConflict
	case errors.ErrTimeout:
		return http.StatusRequestTimeout
	case errors.ErrUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}