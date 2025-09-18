package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	"reciprocal-clubs-backend/services/member-service/internal/models"
	"reciprocal-clubs-backend/services/member-service/internal/service"
)

// Handler handles HTTP requests for the member service
type Handler struct {
	service service.Service
	logger  logging.Logger
	monitor *monitoring.Monitor
}

// NewHandler creates a new HTTP handler
func NewHandler(service service.Service, logger logging.Logger, monitor *monitoring.Monitor) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
		monitor: monitor,
	}
}

// RegisterRoutes registers all HTTP routes
func (h *Handler) RegisterRoutes(router *mux.Router) {
	// Member endpoints
	api := router.PathPrefix("/api/v1").Subrouter()

	// Member CRUD operations
	api.HandleFunc("/members", h.CreateMember).Methods("POST")
	api.HandleFunc("/members/{id:[0-9]+}", h.GetMember).Methods("GET")
	api.HandleFunc("/members/{id:[0-9]+}", h.UpdateMemberProfile).Methods("PUT")
	api.HandleFunc("/members/{id:[0-9]+}", h.DeleteMember).Methods("DELETE")
	api.HandleFunc("/members/{id:[0-9]+}/suspend", h.SuspendMember).Methods("POST")
	api.HandleFunc("/members/{id:[0-9]+}/reactivate", h.ReactivateMember).Methods("POST")

	// Member lookup endpoints
	api.HandleFunc("/members/by-user/{userId:[0-9]+}", h.GetMemberByUserID).Methods("GET")
	api.HandleFunc("/members/by-number/{memberNumber}", h.GetMemberByMemberNumber).Methods("GET")
	api.HandleFunc("/clubs/{clubId:[0-9]+}/members", h.GetMembersByClub).Methods("GET")

	// Member validation endpoints
	api.HandleFunc("/members/{id:[0-9]+}/validate-access", h.ValidateMemberAccess).Methods("GET")
	api.HandleFunc("/members/{id:[0-9]+}/status", h.CheckMembershipStatus).Methods("GET")

	// Analytics endpoints
	api.HandleFunc("/clubs/{clubId:[0-9]+}/analytics/members", h.GetMemberAnalytics).Methods("GET")

	h.logger.Info("HTTP routes registered", map[string]interface{}{
		"base_path": "/api/v1",
	})
}

// CreateMember creates a new member
func (h *Handler) CreateMember(w http.ResponseWriter, r *http.Request) {
	var req CreateMemberHTTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Convert HTTP request to service request
	serviceReq := &service.CreateMemberRequest{
		ClubID:         req.ClubID,
		UserID:         req.UserID,
		MembershipType: req.MembershipType,
		Profile:        req.Profile,
	}

	member, err := h.service.CreateMember(r.Context(), serviceReq)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create member", err)
		return
	}

	h.writeJSONResponse(w, http.StatusCreated, member)
}

// GetMember retrieves a member by ID
func (h *Handler) GetMember(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid member ID", err)
		return
	}

	member, err := h.service.GetMember(r.Context(), uint(id))
	if err != nil {
		h.writeErrorResponse(w, http.StatusNotFound, "Member not found", err)
		return
	}

	h.writeJSONResponse(w, http.StatusOK, member)
}

// GetMemberByUserID retrieves a member by user ID
func (h *Handler) GetMemberByUserID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconv.ParseUint(vars["userId"], 10, 32)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid user ID", err)
		return
	}

	member, err := h.service.GetMemberByUserID(r.Context(), uint(userID))
	if err != nil {
		h.writeErrorResponse(w, http.StatusNotFound, "Member not found", err)
		return
	}

	h.writeJSONResponse(w, http.StatusOK, member)
}

// GetMemberByMemberNumber retrieves a member by member number
func (h *Handler) GetMemberByMemberNumber(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	memberNumber := vars["memberNumber"]

	member, err := h.service.GetMemberByMemberNumber(r.Context(), memberNumber)
	if err != nil {
		h.writeErrorResponse(w, http.StatusNotFound, "Member not found", err)
		return
	}

	h.writeJSONResponse(w, http.StatusOK, member)
}

// GetMembersByClub retrieves members for a specific club
func (h *Handler) GetMembersByClub(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, err := strconv.ParseUint(vars["clubId"], 10, 32)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid club ID", err)
		return
	}

	// Parse pagination parameters
	limit := 50 // default
	offset := 0 // default

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	members, err := h.service.GetMembersByClub(r.Context(), uint(clubID), limit, offset)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get members", err)
		return
	}

	response := map[string]interface{}{
		"members": members,
		"limit":   limit,
		"offset":  offset,
		"count":   len(members),
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// UpdateMemberProfile updates a member's profile
func (h *Handler) UpdateMemberProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid member ID", err)
		return
	}

	var req service.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	member, err := h.service.UpdateMemberProfile(r.Context(), uint(id), &req)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to update member profile", err)
		return
	}

	h.writeJSONResponse(w, http.StatusOK, member)
}

// SuspendMember suspends a member
func (h *Handler) SuspendMember(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid member ID", err)
		return
	}

	var req SuspendMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	member, err := h.service.SuspendMember(r.Context(), uint(id), req.Reason)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to suspend member", err)
		return
	}

	h.writeJSONResponse(w, http.StatusOK, member)
}

// ReactivateMember reactivates a member
func (h *Handler) ReactivateMember(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid member ID", err)
		return
	}

	member, err := h.service.ReactivateMember(r.Context(), uint(id))
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to reactivate member", err)
		return
	}

	h.writeJSONResponse(w, http.StatusOK, member)
}

// DeleteMember deletes a member
func (h *Handler) DeleteMember(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid member ID", err)
		return
	}

	err = h.service.DeleteMember(r.Context(), uint(id))
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete member", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ValidateMemberAccess validates if a member can access facilities
func (h *Handler) ValidateMemberAccess(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid member ID", err)
		return
	}

	canAccess, err := h.service.ValidateMemberAccess(r.Context(), uint(id))
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to validate member access", err)
		return
	}

	response := map[string]interface{}{
		"member_id":  id,
		"can_access": canAccess,
		"timestamp":  time.Now().UTC(),
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// CheckMembershipStatus checks membership status
func (h *Handler) CheckMembershipStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid member ID", err)
		return
	}

	status, err := h.service.CheckMembershipStatus(r.Context(), uint(id))
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to check membership status", err)
		return
	}

	h.writeJSONResponse(w, http.StatusOK, status)
}

// GetMemberAnalytics gets member analytics for a club
func (h *Handler) GetMemberAnalytics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, err := strconv.ParseUint(vars["clubId"], 10, 32)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid club ID", err)
		return
	}

	analytics, err := h.service.GetMemberAnalytics(r.Context(), uint(clubID))
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get member analytics", err)
		return
	}

	h.writeJSONResponse(w, http.StatusOK, analytics)
}

// Helper methods

func (h *Handler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Record metrics
	if h.monitor != nil {
		h.monitor.RecordHTTPRequest("POST", "/api/v1/members", statusCode, time.Since(time.Now()))
	}
}

func (h *Handler) writeErrorResponse(w http.ResponseWriter, statusCode int, message string, err error) {
	h.logger.Error(message, map[string]interface{}{
		"error":       err.Error(),
		"status_code": statusCode,
	})

	errorResponse := map[string]interface{}{
		"error":   message,
		"status":  statusCode,
		"timestamp": time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(errorResponse)
}

// HTTP request/response types

type CreateMemberHTTPRequest struct {
	ClubID         uint                        `json:"club_id" validate:"required"`
	UserID         uint                        `json:"user_id" validate:"required"`
	MembershipType models.MembershipType       `json:"membership_type" validate:"required"`
	Profile        service.CreateProfileRequest `json:"profile" validate:"required"`
}

type SuspendMemberRequest struct {
	Reason string `json:"reason" validate:"required"`
}

type ErrorResponse struct {
	Error     string    `json:"error"`
	Status    int       `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}