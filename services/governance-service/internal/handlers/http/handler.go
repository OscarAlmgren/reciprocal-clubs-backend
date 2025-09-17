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

	// Example routes - replace with actual governance routes
	api.HandleFunc("/examples", h.createExample).Methods("POST")
	api.HandleFunc("/examples", h.listExamples).Methods("GET")
	api.HandleFunc("/examples/{id}", h.getExample).Methods("GET")
	api.HandleFunc("/examples/{id}", h.updateExample).Methods("PUT")
	api.HandleFunc("/examples/{id}", h.deleteExample).Methods("DELETE")

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

// Example handlers - replace with actual governance handlers

func (h *HTTPHandler) createExample(w http.ResponseWriter, r *http.Request) {
	var req service.CreateExampleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	example, err := h.service.CreateExample(r.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create example", map[string]interface{}{
			"error": err.Error(),
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to create example")
		return
	}

	h.writeJSON(w, http.StatusCreated, example)
}

func (h *HTTPHandler) getExample(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	example, err := h.service.GetExample(r.Context(), uint(id))
	if err != nil {
		h.logger.Error("Failed to get example", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		h.writeError(w, http.StatusNotFound, "Example not found")
		return
	}

	h.writeJSON(w, http.StatusOK, example)
}

func (h *HTTPHandler) updateExample(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	var req service.UpdateExampleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	example, err := h.service.UpdateExample(r.Context(), uint(id), &req)
	if err != nil {
		h.logger.Error("Failed to update example", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to update example")
		return
	}

	h.writeJSON(w, http.StatusOK, example)
}

func (h *HTTPHandler) deleteExample(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	if err := h.service.DeleteExample(r.Context(), uint(id)); err != nil {
		h.logger.Error("Failed to delete example", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to delete example")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *HTTPHandler) listExamples(w http.ResponseWriter, r *http.Request) {
	examples, err := h.service.ListExamples(r.Context())
	if err != nil {
		h.logger.Error("Failed to list examples", map[string]interface{}{
			"error": err.Error(),
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to list examples")
		return
	}

	h.writeJSON(w, http.StatusOK, examples)
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