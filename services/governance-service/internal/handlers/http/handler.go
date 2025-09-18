package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	"reciprocal-clubs-backend/services/governance-service/internal/service"
)

// HTTPHandler handles HTTP requests for governance service
type HTTPHandler struct {
	service    *service.Service
	logger     logging.Logger
	monitoring *monitoring.Monitor
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(service *service.Service, logger logging.Logger, monitoring *monitoring.Monitor) *HTTPHandler {
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

	// Proposal routes
	api.HandleFunc("/proposals", h.createProposal).Methods("POST")
	api.HandleFunc("/proposals", h.listProposals).Methods("GET")
	api.HandleFunc("/proposals/{id}", h.getProposal).Methods("GET")
	api.HandleFunc("/proposals/{id}/activate", h.activateProposal).Methods("POST")
	api.HandleFunc("/proposals/{id}/finalize", h.finalizeProposal).Methods("POST")

	// Vote routes
	api.HandleFunc("/proposals/{id}/votes", h.castVote).Methods("POST")
	api.HandleFunc("/proposals/{id}/votes", h.getVotesByProposal).Methods("GET")
	api.HandleFunc("/proposals/{id}/results", h.getVoteResults).Methods("GET")

	// Voting rights routes
	api.HandleFunc("/voting-rights", h.createVotingRights).Methods("POST")
	api.HandleFunc("/members/{member_id}/voting-rights/{club_id}", h.getVotingRights).Methods("GET")

	// Governance policy routes
	api.HandleFunc("/policies", h.createGovernancePolicy).Methods("POST")
	api.HandleFunc("/clubs/{club_id}/policies", h.getActiveGovernancePolicies).Methods("GET")

	// Club-specific routes
	api.HandleFunc("/clubs/{club_id}/proposals", h.getProposalsByClub).Methods("GET")
	api.HandleFunc("/clubs/{club_id}/proposals/active", h.getActiveProposals).Methods("GET")

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
		"service": "governance-service",
	})
}

// Proposal handlers

func (h *HTTPHandler) createProposal(w http.ResponseWriter, r *http.Request) {
	var req service.CreateProposalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	proposal, err := h.service.CreateProposal(r.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create proposal", map[string]interface{}{
			"error": err.Error(),
		})
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusCreated, proposal)
}

func (h *HTTPHandler) getProposal(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	proposal, err := h.service.GetProposal(r.Context(), uint(id))
	if err != nil {
		h.logger.Error("Failed to get proposal", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		h.writeError(w, http.StatusNotFound, "Proposal not found")
		return
	}

	h.writeJSON(w, http.StatusOK, proposal)
}

func (h *HTTPHandler) activateProposal(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	var req struct {
		ActivatorID uint `json:"activator_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	proposal, err := h.service.ActivateProposal(r.Context(), uint(id), req.ActivatorID)
	if err != nil {
		h.logger.Error("Failed to activate proposal", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, proposal)
}

func (h *HTTPHandler) finalizeProposal(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	proposal, err := h.service.FinalizeProposal(r.Context(), uint(id))
	if err != nil {
		h.logger.Error("Failed to finalize proposal", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, proposal)
}

func (h *HTTPHandler) listProposals(w http.ResponseWriter, r *http.Request) {
	clubIDStr := r.URL.Query().Get("club_id")
	if clubIDStr == "" {
		h.writeError(w, http.StatusBadRequest, "club_id parameter is required")
		return
	}

	clubID, err := strconv.ParseUint(clubIDStr, 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid club_id")
		return
	}

	proposals, err := h.service.GetProposalsByClub(r.Context(), uint(clubID))
	if err != nil {
		h.logger.Error("Failed to list proposals", map[string]interface{}{
			"error":   err.Error(),
			"club_id": clubID,
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to list proposals")
		return
	}

	h.writeJSON(w, http.StatusOK, proposals)
}

func (h *HTTPHandler) getProposalsByClub(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, err := strconv.ParseUint(vars["club_id"], 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid club ID")
		return
	}

	proposals, err := h.service.GetProposalsByClub(r.Context(), uint(clubID))
	if err != nil {
		h.logger.Error("Failed to get proposals by club", map[string]interface{}{
			"error":   err.Error(),
			"club_id": clubID,
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to get proposals")
		return
	}

	h.writeJSON(w, http.StatusOK, proposals)
}

func (h *HTTPHandler) getActiveProposals(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, err := strconv.ParseUint(vars["club_id"], 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid club ID")
		return
	}

	proposals, err := h.service.GetActiveProposals(r.Context(), uint(clubID))
	if err != nil {
		h.logger.Error("Failed to get active proposals", map[string]interface{}{
			"error":   err.Error(),
			"club_id": clubID,
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to get active proposals")
		return
	}

	h.writeJSON(w, http.StatusOK, proposals)
}

// Vote handlers

func (h *HTTPHandler) castVote(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	proposalID, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid proposal ID")
		return
	}

	var req service.CastVoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Ensure proposal ID matches URL parameter
	req.ProposalID = uint(proposalID)

	vote, err := h.service.CastVote(r.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to cast vote", map[string]interface{}{
			"error":       err.Error(),
			"proposal_id": proposalID,
		})
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.writeJSON(w, http.StatusCreated, vote)
}

func (h *HTTPHandler) getVotesByProposal(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	proposalID, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid proposal ID")
		return
	}

	// This would need to be implemented in the service layer
	// For now, return placeholder
	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"proposal_id": proposalID,
		"votes":       []interface{}{},
		"message":     "Vote listing not yet implemented",
	})
}

func (h *HTTPHandler) getVoteResults(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	proposalID, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid proposal ID")
		return
	}

	// This would need to be implemented in the service layer
	// For now, return placeholder
	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"proposal_id": proposalID,
		"results":     map[string]interface{}{},
		"message":     "Vote results not yet implemented",
	})
}

// Voting rights handlers

func (h *HTTPHandler) createVotingRights(w http.ResponseWriter, r *http.Request) {
	var req service.CreateVotingRightsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	votingRights, err := h.service.CreateVotingRights(r.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create voting rights", map[string]interface{}{
			"error": err.Error(),
		})
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusCreated, votingRights)
}

func (h *HTTPHandler) getVotingRights(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	memberID, err := strconv.ParseUint(vars["member_id"], 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid member ID")
		return
	}

	clubID, err := strconv.ParseUint(vars["club_id"], 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid club ID")
		return
	}

	votingRights, err := h.service.GetVotingRights(r.Context(), uint(memberID), uint(clubID))
	if err != nil {
		h.logger.Error("Failed to get voting rights", map[string]interface{}{
			"error":     err.Error(),
			"member_id": memberID,
			"club_id":   clubID,
		})
		h.writeError(w, http.StatusNotFound, "Voting rights not found")
		return
	}

	h.writeJSON(w, http.StatusOK, votingRights)
}

// Governance policy handlers

func (h *HTTPHandler) createGovernancePolicy(w http.ResponseWriter, r *http.Request) {
	var req service.CreateGovernancePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	policy, err := h.service.CreateGovernancePolicy(r.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create governance policy", map[string]interface{}{
			"error": err.Error(),
		})
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusCreated, policy)
}

func (h *HTTPHandler) getActiveGovernancePolicies(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, err := strconv.ParseUint(vars["club_id"], 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid club ID")
		return
	}

	policies, err := h.service.GetActiveGovernancePolicies(r.Context(), uint(clubID))
	if err != nil {
		h.logger.Error("Failed to get active governance policies", map[string]interface{}{
			"error":   err.Error(),
			"club_id": clubID,
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to get governance policies")
		return
	}

	h.writeJSON(w, http.StatusOK, policies)
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