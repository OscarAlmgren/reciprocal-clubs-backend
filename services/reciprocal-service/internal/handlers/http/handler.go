package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	"reciprocal-clubs-backend/services/reciprocal-service/internal/service"
)

// HTTPHandler handles HTTP requests for reciprocal service
type HTTPHandler struct {
	service    service.ReciprocalServiceInterface
	logger     logging.Logger
	monitoring monitoring.MonitoringInterface
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(service service.ReciprocalServiceInterface, logger logging.Logger, monitoring monitoring.MonitoringInterface) *HTTPHandler {
	return &HTTPHandler{
		service:    service,
		logger:     logger,
		monitoring: monitoring,
	}
}

// SetupRoutes configures the HTTP routes
func (h *HTTPHandler) SetupRoutes() http.Handler {
	router := mux.NewRouter()

	// Health check
	router.HandleFunc("/health", h.healthCheck).Methods("GET")

	// API routes
	api := router.PathPrefix("/api/v1").Subrouter()

	// Agreement routes
	api.HandleFunc("/agreements", h.createAgreement).Methods("POST")
	api.HandleFunc("/agreements/{id}", h.getAgreement).Methods("GET")
	api.HandleFunc("/agreements/{id}/status", h.updateAgreementStatus).Methods("PUT")
	api.HandleFunc("/clubs/{clubId}/agreements", h.getAgreementsByClub).Methods("GET")

	// Visit routes
	api.HandleFunc("/visits", h.requestVisit).Methods("POST")
	api.HandleFunc("/visits/{id}", h.getVisit).Methods("GET")
	api.HandleFunc("/visits/{id}/confirm", h.confirmVisit).Methods("POST")
	api.HandleFunc("/visits/checkin", h.checkInVisit).Methods("POST")
	api.HandleFunc("/visits/checkout", h.checkOutVisit).Methods("POST")
	api.HandleFunc("/members/{memberId}/visits", h.getMemberVisits).Methods("GET")
	api.HandleFunc("/clubs/{clubId}/visits", h.getClubVisits).Methods("GET")
	api.HandleFunc("/members/{memberId}/stats", h.getMemberStats).Methods("GET")

	// Add middleware
	router.Use(h.loggingMiddleware)
	router.Use(h.monitoringMiddleware)

	return router
}

// Health check endpoint
func (h *HTTPHandler) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "reciprocal-service",
	})
}

// Agreement handlers

func (h *HTTPHandler) createAgreement(w http.ResponseWriter, r *http.Request) {
	var req service.CreateAgreementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	agreement, err := h.service.CreateAgreement(r.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create agreement", map[string]interface{}{
			"error": err.Error(),
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to create agreement")
		return
	}

	h.writeJSON(w, http.StatusCreated, agreement)
}

func (h *HTTPHandler) getAgreement(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	agreement, err := h.service.GetAgreementByID(r.Context(), uint(id))
	if err != nil {
		h.logger.Error("Failed to get agreement", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		h.writeError(w, http.StatusNotFound, "Agreement not found")
		return
	}

	h.writeJSON(w, http.StatusOK, agreement)
}

func (h *HTTPHandler) updateAgreementStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	var req struct {
		Status       string `json:"status"`
		ReviewedByID string `json:"reviewed_by_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	agreement, err := h.service.UpdateAgreementStatus(r.Context(), uint(id), req.Status, req.ReviewedByID)
	if err != nil {
		h.logger.Error("Failed to update agreement status", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to update agreement status")
		return
	}

	h.writeJSON(w, http.StatusOK, agreement)
}

func (h *HTTPHandler) getAgreementsByClub(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, err := strconv.ParseUint(vars["clubId"], 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid club ID")
		return
	}

	agreements, err := h.service.GetAgreementsByClub(r.Context(), uint(clubID))
	if err != nil {
		h.logger.Error("Failed to get agreements by club", map[string]interface{}{
			"error":   err.Error(),
			"club_id": clubID,
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to get agreements")
		return
	}

	h.writeJSON(w, http.StatusOK, agreements)
}

// Visit handlers

func (h *HTTPHandler) requestVisit(w http.ResponseWriter, r *http.Request) {
	var req service.RequestVisitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	visit, err := h.service.RequestVisit(r.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to request visit", map[string]interface{}{
			"error": err.Error(),
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to request visit")
		return
	}

	h.writeJSON(w, http.StatusCreated, visit)
}

func (h *HTTPHandler) getVisit(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	visit, err := h.service.GetVisitByID(r.Context(), uint(id))
	if err != nil {
		h.logger.Error("Failed to get visit", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		h.writeError(w, http.StatusNotFound, "Visit not found")
		return
	}

	h.writeJSON(w, http.StatusOK, visit)
}

func (h *HTTPHandler) confirmVisit(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	var req struct {
		ConfirmedByID string `json:"confirmed_by_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	visit, err := h.service.ConfirmVisit(r.Context(), uint(id), req.ConfirmedByID)
	if err != nil {
		h.logger.Error("Failed to confirm visit", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to confirm visit")
		return
	}

	h.writeJSON(w, http.StatusOK, visit)
}

func (h *HTTPHandler) checkInVisit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		VerificationCode string `json:"verification_code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	visit, err := h.service.CheckInVisit(r.Context(), req.VerificationCode)
	if err != nil {
		h.logger.Error("Failed to check in visit", map[string]interface{}{
			"error":             err.Error(),
			"verification_code": req.VerificationCode,
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to check in visit")
		return
	}

	h.writeJSON(w, http.StatusOK, visit)
}

func (h *HTTPHandler) checkOutVisit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		VerificationCode string   `json:"verification_code"`
		ActualCost       *float64 `json:"actual_cost,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	visit, err := h.service.CheckOutVisit(r.Context(), req.VerificationCode, req.ActualCost)
	if err != nil {
		h.logger.Error("Failed to check out visit", map[string]interface{}{
			"error":             err.Error(),
			"verification_code": req.VerificationCode,
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to check out visit")
		return
	}

	h.writeJSON(w, http.StatusOK, visit)
}

func (h *HTTPHandler) getMemberVisits(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	memberID, err := strconv.ParseUint(vars["memberId"], 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid member ID")
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

	visits, err := h.service.GetMemberVisits(r.Context(), uint(memberID), limit, offset)
	if err != nil {
		h.logger.Error("Failed to get member visits", map[string]interface{}{
			"error":     err.Error(),
			"member_id": memberID,
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to get visits")
		return
	}

	h.writeJSON(w, http.StatusOK, visits)
}

func (h *HTTPHandler) getClubVisits(w http.ResponseWriter, r *http.Request) {
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

	visits, err := h.service.GetClubVisits(r.Context(), uint(clubID), limit, offset)
	if err != nil {
		h.logger.Error("Failed to get club visits", map[string]interface{}{
			"error":   err.Error(),
			"club_id": clubID,
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to get visits")
		return
	}

	h.writeJSON(w, http.StatusOK, visits)
}

func (h *HTTPHandler) getMemberStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	memberID, err := strconv.ParseUint(vars["memberId"], 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid member ID")
		return
	}

	// Parse query parameters
	clubIDStr := r.URL.Query().Get("club_id")
	yearStr := r.URL.Query().Get("year")
	monthStr := r.URL.Query().Get("month")

	clubID, err := strconv.ParseUint(clubIDStr, 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid club ID")
		return
	}

	year := time.Now().Year() // default to current year
	if yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil && y > 2020 && y <= 2030 {
			year = y
		}
	}

	month := 0 // default to all months
	if monthStr != "" {
		if m, err := strconv.Atoi(monthStr); err == nil && m >= 1 && m <= 12 {
			month = m
		}
	}

	stats, err := h.service.GetMemberVisitStats(r.Context(), uint(memberID), uint(clubID), year, month)
	if err != nil {
		h.logger.Error("Failed to get member stats", map[string]interface{}{
			"error":     err.Error(),
			"member_id": memberID,
			"club_id":   clubID,
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
		next.ServeHTTP(w, r)
		duration := time.Since(start)
		h.monitoring.RecordHTTPRequest(r.Method, r.URL.Path, 200, duration)
	})
}
