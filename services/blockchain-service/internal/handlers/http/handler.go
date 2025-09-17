package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	"reciprocal-clubs-backend/services/blockchain-service/internal/service"
)

// HTTPHandler handles HTTP requests for the blockchain service
type HTTPHandler struct {
	service    *service.BlockchainService
	logger     logging.Logger
	monitoring *monitoring.Monitor
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(service *service.BlockchainService, logger logging.Logger, monitoring *monitoring.Monitor) *HTTPHandler {
	return &HTTPHandler{
		service:    service,
		logger:     logger,
		monitoring: monitoring,
	}
}

// SetupRoutes sets up the HTTP routes
func (h *HTTPHandler) SetupRoutes() *mux.Router {
	router := mux.NewRouter()

	// Health check
	router.HandleFunc("/health", h.HealthCheck).Methods("GET")
	router.HandleFunc("/ready", h.ReadinessCheck).Methods("GET")

	// Transaction endpoints
	router.HandleFunc("/transactions", h.CreateTransaction).Methods("POST")
	router.HandleFunc("/transactions/{id:[0-9]+}", h.GetTransaction).Methods("GET")
	router.HandleFunc("/transactions/txid/{txId}", h.GetTransactionByTxID).Methods("GET")
	router.HandleFunc("/transactions/club/{clubId:[0-9]+}", h.GetTransactionsByClub).Methods("GET")
	router.HandleFunc("/transactions/{id:[0-9]+}/submit", h.SubmitTransaction).Methods("PUT")
	router.HandleFunc("/transactions/confirm", h.ConfirmTransaction).Methods("PUT")
	router.HandleFunc("/transactions/fail", h.FailTransaction).Methods("PUT")

	// Channel endpoints
	router.HandleFunc("/channels", h.CreateChannel).Methods("POST")
	router.HandleFunc("/channels/{id:[0-9]+}", h.GetChannel).Methods("GET")
	router.HandleFunc("/channels/channel/{channelId}", h.GetChannelByChannelID).Methods("GET")
	router.HandleFunc("/channels/club/{clubId:[0-9]+}", h.GetChannelsByClub).Methods("GET")

	// Chaincode endpoints
	router.HandleFunc("/chaincodes", h.CreateChaincode).Methods("POST")
	router.HandleFunc("/chaincodes/{id:[0-9]+}", h.GetChaincode).Methods("GET")
	router.HandleFunc("/chaincodes/channel/{channelId}", h.GetChaincodesByChannel).Methods("GET")
	router.HandleFunc("/chaincodes/{id:[0-9]+}/install", h.MarkChaincodeInstalled).Methods("PUT")
	router.HandleFunc("/chaincodes/{id:[0-9]+}/commit", h.MarkChaincodeCommitted).Methods("PUT")

	// Block endpoints
	router.HandleFunc("/blocks", h.CreateBlock).Methods("POST")
	router.HandleFunc("/blocks/hash/{blockHash}", h.GetBlockByHash).Methods("GET")
	router.HandleFunc("/blocks/channel/{channelId}", h.GetBlocksByChannel).Methods("GET")

	// Event endpoints
	router.HandleFunc("/events", h.CreateEvent).Methods("POST")
	router.HandleFunc("/events/unprocessed", h.GetUnprocessedEvents).Methods("GET")
	router.HandleFunc("/events/{id:[0-9]+}/process", h.MarkEventProcessed).Methods("PUT")

	// Statistics
	router.HandleFunc("/stats/club/{clubId:[0-9]+}", h.GetServiceStats).Methods("GET")

	return router
}

// Health check endpoints

func (h *HTTPHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	if err := h.service.HealthCheck(r.Context()); err != nil {
		h.writeErrorResponse(w, http.StatusServiceUnavailable, "Health check failed", err)
		return
	}

	h.writeResponse(w, http.StatusOK, map[string]string{
		"status":    "healthy",
		"service":   "blockchain-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *HTTPHandler) ReadinessCheck(w http.ResponseWriter, r *http.Request) {
	if err := h.service.HealthCheck(r.Context()); err != nil {
		h.writeErrorResponse(w, http.StatusServiceUnavailable, "Service not ready", err)
		return
	}

	h.writeResponse(w, http.StatusOK, map[string]string{
		"status":    "ready",
		"service":   "blockchain-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// Transaction handlers

func (h *HTTPHandler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	var req service.CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	transaction, err := h.service.CreateTransaction(r.Context(), &req)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create transaction", err)
		return
	}

	h.writeResponse(w, http.StatusCreated, transaction)
}

func (h *HTTPHandler) GetTransaction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid transaction ID", err)
		return
	}

	transaction, err := h.service.GetTransactionByID(r.Context(), uint(id))
	if err != nil {
		h.writeErrorResponse(w, http.StatusNotFound, "Transaction not found", err)
		return
	}

	h.writeResponse(w, http.StatusOK, transaction)
}

func (h *HTTPHandler) GetTransactionByTxID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	txID := vars["txId"]

	transaction, err := h.service.GetTransactionByTxID(r.Context(), txID)
	if err != nil {
		h.writeErrorResponse(w, http.StatusNotFound, "Transaction not found", err)
		return
	}

	h.writeResponse(w, http.StatusOK, transaction)
}

func (h *HTTPHandler) GetTransactionsByClub(w http.ResponseWriter, r *http.Request) {
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

	transactions, err := h.service.GetTransactionsByClubID(r.Context(), uint(clubID), limit, offset)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get transactions", err)
		return
	}

	h.writeResponse(w, http.StatusOK, transactions)
}

func (h *HTTPHandler) SubmitTransaction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid transaction ID", err)
		return
	}

	var req struct {
		TxID           string   `json:"tx_id" validate:"required"`
		EndorsingPeers []string `json:"endorsing_peers" validate:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	transaction, err := h.service.SubmitTransaction(r.Context(), uint(id), req.TxID, req.EndorsingPeers)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to submit transaction", err)
		return
	}

	h.writeResponse(w, http.StatusOK, transaction)
}

func (h *HTTPHandler) ConfirmTransaction(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TxID        string `json:"tx_id" validate:"required"`
		BlockNumber uint64 `json:"block_number" validate:"required"`
		BlockHash   string `json:"block_hash" validate:"required"`
		TxIndex     uint   `json:"tx_index" validate:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	transaction, err := h.service.ConfirmTransaction(r.Context(), req.TxID, req.BlockNumber, req.BlockHash, req.TxIndex)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to confirm transaction", err)
		return
	}

	h.writeResponse(w, http.StatusOK, transaction)
}

func (h *HTTPHandler) FailTransaction(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TxID         string `json:"tx_id" validate:"required"`
		ErrorMessage string `json:"error_message" validate:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	transaction, err := h.service.FailTransaction(r.Context(), req.TxID, req.ErrorMessage)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to mark transaction as failed", err)
		return
	}

	h.writeResponse(w, http.StatusOK, transaction)
}

// Channel handlers

func (h *HTTPHandler) CreateChannel(w http.ResponseWriter, r *http.Request) {
	var req service.CreateChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	channel, err := h.service.CreateChannel(r.Context(), &req)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create channel", err)
		return
	}

	h.writeResponse(w, http.StatusCreated, channel)
}

func (h *HTTPHandler) GetChannel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid channel ID", err)
		return
	}

	channel, err := h.service.GetChannelByID(r.Context(), uint(id))
	if err != nil {
		h.writeErrorResponse(w, http.StatusNotFound, "Channel not found", err)
		return
	}

	h.writeResponse(w, http.StatusOK, channel)
}

func (h *HTTPHandler) GetChannelByChannelID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	channelID := vars["channelId"]

	channel, err := h.service.GetChannelByChannelID(r.Context(), channelID)
	if err != nil {
		h.writeErrorResponse(w, http.StatusNotFound, "Channel not found", err)
		return
	}

	h.writeResponse(w, http.StatusOK, channel)
}

func (h *HTTPHandler) GetChannelsByClub(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, err := strconv.ParseUint(vars["clubId"], 10, 32)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid club ID", err)
		return
	}

	channels, err := h.service.GetChannelsByClubID(r.Context(), uint(clubID))
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get channels", err)
		return
	}

	h.writeResponse(w, http.StatusOK, channels)
}

// Chaincode handlers

func (h *HTTPHandler) CreateChaincode(w http.ResponseWriter, r *http.Request) {
	var req service.CreateChaincodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	chaincode, err := h.service.CreateChaincode(r.Context(), &req)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create chaincode", err)
		return
	}

	h.writeResponse(w, http.StatusCreated, chaincode)
}

func (h *HTTPHandler) GetChaincode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid chaincode ID", err)
		return
	}

	chaincode, err := h.service.GetChaincodeByID(r.Context(), uint(id))
	if err != nil {
		h.writeErrorResponse(w, http.StatusNotFound, "Chaincode not found", err)
		return
	}

	h.writeResponse(w, http.StatusOK, chaincode)
}

func (h *HTTPHandler) GetChaincodesByChannel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	channelID := vars["channelId"]

	chaincodes, err := h.service.GetChaincodesByChannelID(r.Context(), channelID)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get chaincodes", err)
		return
	}

	h.writeResponse(w, http.StatusOK, chaincodes)
}

func (h *HTTPHandler) MarkChaincodeInstalled(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid chaincode ID", err)
		return
	}

	if err := h.service.MarkChaincodeInstalled(r.Context(), uint(id)); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to mark chaincode as installed", err)
		return
	}

	h.writeResponse(w, http.StatusOK, map[string]string{"status": "installed"})
}

func (h *HTTPHandler) MarkChaincodeCommitted(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid chaincode ID", err)
		return
	}

	if err := h.service.MarkChaincodeCommitted(r.Context(), uint(id)); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to mark chaincode as committed", err)
		return
	}

	h.writeResponse(w, http.StatusOK, map[string]string{"status": "committed"})
}

// Block handlers

func (h *HTTPHandler) CreateBlock(w http.ResponseWriter, r *http.Request) {
	var req service.CreateBlockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	block, err := h.service.CreateBlock(r.Context(), &req)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create block", err)
		return
	}

	h.writeResponse(w, http.StatusCreated, block)
}

func (h *HTTPHandler) GetBlockByHash(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	blockHash := vars["blockHash"]

	block, err := h.service.GetBlockByHash(r.Context(), blockHash)
	if err != nil {
		h.writeErrorResponse(w, http.StatusNotFound, "Block not found", err)
		return
	}

	h.writeResponse(w, http.StatusOK, block)
}

func (h *HTTPHandler) GetBlocksByChannel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	channelID := vars["channelId"]

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

	blocks, err := h.service.GetBlocksByChannelID(r.Context(), channelID, limit, offset)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get blocks", err)
		return
	}

	h.writeResponse(w, http.StatusOK, blocks)
}

// Event handlers

func (h *HTTPHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var req service.CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	event, err := h.service.CreateEvent(r.Context(), &req)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create event", err)
		return
	}

	h.writeResponse(w, http.StatusCreated, event)
}

func (h *HTTPHandler) GetUnprocessedEvents(w http.ResponseWriter, r *http.Request) {
	limit := 100 // default

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	events, err := h.service.GetUnprocessedEvents(r.Context(), limit)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get unprocessed events", err)
		return
	}

	h.writeResponse(w, http.StatusOK, events)
}

func (h *HTTPHandler) MarkEventProcessed(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid event ID", err)
		return
	}

	if err := h.service.MarkEventProcessed(r.Context(), uint(id)); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to mark event as processed", err)
		return
	}

	h.writeResponse(w, http.StatusOK, map[string]string{"status": "processed"})
}

// Statistics handlers

func (h *HTTPHandler) GetServiceStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, err := strconv.ParseUint(vars["clubId"], 10, 32)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid club ID", err)
		return
	}

	stats, err := h.service.GetServiceStats(r.Context(), uint(clubID))
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get service stats", err)
		return
	}

	h.writeResponse(w, http.StatusOK, stats)
}

// Helper methods

func (h *HTTPHandler) writeResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode response", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

func (h *HTTPHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, message string, err error) {
	h.logger.Error(message, map[string]interface{}{
		"status_code": statusCode,
		"error":       err.Error(),
	})

	h.monitoring.RecordBusinessEvent("http_error", strconv.Itoa(statusCode))

	response := map[string]interface{}{
		"error":     message,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	if err != nil {
		response["details"] = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if encodeErr := json.NewEncoder(w).Encode(response); encodeErr != nil {
		h.logger.Error("Failed to encode error response", map[string]interface{}{
			"error": encodeErr.Error(),
		})
	}
}