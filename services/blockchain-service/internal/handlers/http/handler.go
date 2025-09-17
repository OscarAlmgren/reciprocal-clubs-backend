package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	"reciprocal-clubs-backend/services/blockchain-service/internal/models"
	"reciprocal-clubs-backend/services/blockchain-service/internal/service"
)

// HTTPHandler handles HTTP requests for blockchain service
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

// SetupRoutes configures the HTTP routes
func (h *HTTPHandler) SetupRoutes() http.Handler {
	router := mux.NewRouter()

	// Health check
	router.HandleFunc("/health", h.healthCheck).Methods("GET")

	// API routes
	api := router.PathPrefix("/api/v1").Subrouter()

	// Transaction routes
	api.HandleFunc("/transactions", h.createTransaction).Methods("POST")
	api.HandleFunc("/transactions/{id}", h.getTransaction).Methods("GET")
	api.HandleFunc("/transactions/hash/{hash}", h.getTransactionByHash).Methods("GET")
	api.HandleFunc("/transactions/{id}/submit", h.submitTransaction).Methods("POST")
	api.HandleFunc("/transactions/{id}/confirm", h.confirmTransaction).Methods("POST")
	api.HandleFunc("/transactions/{id}/fail", h.failTransaction).Methods("POST")
	api.HandleFunc("/clubs/{clubId}/transactions", h.getClubTransactions).Methods("GET")
	api.HandleFunc("/users/{userId}/transactions", h.getUserTransactions).Methods("GET")

	// Contract routes
	api.HandleFunc("/contracts", h.createContract).Methods("POST")
	api.HandleFunc("/contracts/{id}", h.getContract).Methods("GET")
	api.HandleFunc("/clubs/{clubId}/contracts", h.getClubContracts).Methods("GET")

	// Wallet routes
	api.HandleFunc("/wallets", h.createWallet).Methods("POST")
	api.HandleFunc("/wallets/{address}/balance", h.updateWalletBalance).Methods("PUT")
	api.HandleFunc("/users/{userId}/wallets", h.getUserWallets).Methods("GET")

	// Token routes
	api.HandleFunc("/tokens", h.createToken).Methods("POST")
	api.HandleFunc("/clubs/{clubId}/tokens", h.getClubTokens).Methods("GET")

	// Stats routes
	api.HandleFunc("/clubs/{clubId}/stats", h.getTransactionStats).Methods("GET")

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
		"service": "blockchain-service",
	})
}

// Transaction handlers

func (h *HTTPHandler) createTransaction(w http.ResponseWriter, r *http.Request) {
	var req service.CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	transaction, err := h.service.CreateTransaction(r.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create transaction", map[string]interface{}{
			"error": err.Error(),
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to create transaction")
		return
	}

	h.writeJSON(w, http.StatusCreated, transaction)
}

func (h *HTTPHandler) getTransaction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	transaction, err := h.service.GetTransactionByID(r.Context(), uint(id))
	if err != nil {
		h.logger.Error("Failed to get transaction", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		h.writeError(w, http.StatusNotFound, "Transaction not found")
		return
	}

	h.writeJSON(w, http.StatusOK, transaction)
}

func (h *HTTPHandler) getTransactionByHash(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hash := vars["hash"]

	transaction, err := h.service.GetTransactionByHash(r.Context(), hash)
	if err != nil {
		h.logger.Error("Failed to get transaction by hash", map[string]interface{}{
			"error": err.Error(),
			"hash":  hash,
		})
		h.writeError(w, http.StatusNotFound, "Transaction not found")
		return
	}

	h.writeJSON(w, http.StatusOK, transaction)
}

func (h *HTTPHandler) submitTransaction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	var req struct {
		Hash  string `json:"hash"`
		Nonce uint64 `json:"nonce"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	transaction, err := h.service.SubmitTransaction(r.Context(), uint(id), req.Hash, req.Nonce)
	if err != nil {
		h.logger.Error("Failed to submit transaction", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to submit transaction")
		return
	}

	h.writeJSON(w, http.StatusOK, transaction)
}

func (h *HTTPHandler) confirmTransaction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	var req struct {
		Hash        string `json:"hash"`
		BlockNumber uint64 `json:"block_number"`
		BlockHash   string `json:"block_hash"`
		GasUsed     uint64 `json:"gas_used"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	transaction, err := h.service.ConfirmTransaction(r.Context(), req.Hash, req.BlockNumber, req.BlockHash, req.GasUsed)
	if err != nil {
		h.logger.Error("Failed to confirm transaction", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to confirm transaction")
		return
	}

	h.writeJSON(w, http.StatusOK, transaction)
}

func (h *HTTPHandler) failTransaction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	var req struct {
		Hash         string `json:"hash"`
		ErrorMessage string `json:"error_message"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	transaction, err := h.service.FailTransaction(r.Context(), req.Hash, req.ErrorMessage)
	if err != nil {
		h.logger.Error("Failed to fail transaction", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to fail transaction")
		return
	}

	h.writeJSON(w, http.StatusOK, transaction)
}

func (h *HTTPHandler) getClubTransactions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, err := strconv.ParseUint(vars["clubId"], 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid club ID")
		return
	}

	// Parse query parameters
	networkStr := r.URL.Query().Get("network")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	var network models.Network
	if networkStr != "" {
		network = models.Network(networkStr)
	}

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

	transactions, err := h.service.GetTransactionsByClub(r.Context(), uint(clubID), network, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get club transactions", map[string]interface{}{
			"error":   err.Error(),
			"club_id": clubID,
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to get transactions")
		return
	}

	h.writeJSON(w, http.StatusOK, transactions)
}

func (h *HTTPHandler) getUserTransactions(w http.ResponseWriter, r *http.Request) {
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

	// Parse query parameters
	networkStr := r.URL.Query().Get("network")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	var network models.Network
	if networkStr != "" {
		network = models.Network(networkStr)
	}

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

	transactions, err := h.service.GetTransactionsByUser(r.Context(), userID, uint(clubID), network, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get user transactions", map[string]interface{}{
			"error":   err.Error(),
			"user_id": userID,
			"club_id": clubID,
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to get transactions")
		return
	}

	h.writeJSON(w, http.StatusOK, transactions)
}

// Contract handlers

func (h *HTTPHandler) createContract(w http.ResponseWriter, r *http.Request) {
	var req service.CreateContractRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	contract, err := h.service.CreateContract(r.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create contract", map[string]interface{}{
			"error": err.Error(),
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to create contract")
		return
	}

	h.writeJSON(w, http.StatusCreated, contract)
}

func (h *HTTPHandler) getContract(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	contract, err := h.service.GetContractByID(r.Context(), uint(id))
	if err != nil {
		h.logger.Error("Failed to get contract", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		h.writeError(w, http.StatusNotFound, "Contract not found")
		return
	}

	h.writeJSON(w, http.StatusOK, contract)
}

func (h *HTTPHandler) getClubContracts(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, err := strconv.ParseUint(vars["clubId"], 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid club ID")
		return
	}

	networkStr := r.URL.Query().Get("network")
	var network models.Network
	if networkStr != "" {
		network = models.Network(networkStr)
	}

	contracts, err := h.service.GetContractsByClub(r.Context(), uint(clubID), network)
	if err != nil {
		h.logger.Error("Failed to get club contracts", map[string]interface{}{
			"error":   err.Error(),
			"club_id": clubID,
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to get contracts")
		return
	}

	h.writeJSON(w, http.StatusOK, contracts)
}

// Wallet handlers

func (h *HTTPHandler) createWallet(w http.ResponseWriter, r *http.Request) {
	var req service.CreateWalletRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	wallet, err := h.service.CreateWallet(r.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create wallet", map[string]interface{}{
			"error": err.Error(),
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to create wallet")
		return
	}

	h.writeJSON(w, http.StatusCreated, wallet)
}

func (h *HTTPHandler) updateWalletBalance(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["address"]

	var req struct {
		Network       models.Network    `json:"network"`
		Balance       string            `json:"balance"`
		TokenBalances map[string]string `json:"token_balances,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	wallet, err := h.service.UpdateWalletBalance(r.Context(), address, req.Network, req.Balance, req.TokenBalances)
	if err != nil {
		h.logger.Error("Failed to update wallet balance", map[string]interface{}{
			"error":   err.Error(),
			"address": address,
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to update wallet balance")
		return
	}

	h.writeJSON(w, http.StatusOK, wallet)
}

func (h *HTTPHandler) getUserWallets(w http.ResponseWriter, r *http.Request) {
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

	networkStr := r.URL.Query().Get("network")
	var network models.Network
	if networkStr != "" {
		network = models.Network(networkStr)
	}

	wallets, err := h.service.GetWalletsByUser(r.Context(), userID, uint(clubID), network)
	if err != nil {
		h.logger.Error("Failed to get user wallets", map[string]interface{}{
			"error":   err.Error(),
			"user_id": userID,
			"club_id": clubID,
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to get wallets")
		return
	}

	h.writeJSON(w, http.StatusOK, wallets)
}

// Token handlers

func (h *HTTPHandler) createToken(w http.ResponseWriter, r *http.Request) {
	var req service.CreateTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	token, err := h.service.CreateToken(r.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create token", map[string]interface{}{
			"error": err.Error(),
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to create token")
		return
	}

	h.writeJSON(w, http.StatusCreated, token)
}

func (h *HTTPHandler) getClubTokens(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, err := strconv.ParseUint(vars["clubId"], 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid club ID")
		return
	}

	networkStr := r.URL.Query().Get("network")
	var network models.Network
	if networkStr != "" {
		network = models.Network(networkStr)
	}

	tokens, err := h.service.GetTokensByClub(r.Context(), uint(clubID), network)
	if err != nil {
		h.logger.Error("Failed to get club tokens", map[string]interface{}{
			"error":   err.Error(),
			"club_id": clubID,
		})
		h.writeError(w, http.StatusInternalServerError, "Failed to get tokens")
		return
	}

	h.writeJSON(w, http.StatusOK, tokens)
}

// Stats handlers

func (h *HTTPHandler) getTransactionStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, err := strconv.ParseUint(vars["clubId"], 10, 32)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid club ID")
		return
	}

	// Parse query parameters
	networkStr := r.URL.Query().Get("network")
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	var network models.Network
	if networkStr != "" {
		network = models.Network(networkStr)
	}

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

	stats, err := h.service.GetTransactionStats(r.Context(), uint(clubID), network, fromDate, toDate)
	if err != nil {
		h.logger.Error("Failed to get transaction stats", map[string]interface{}{
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
		next.ServeHTTP(w, r)
		duration := time.Since(start)
		h.monitoring.RecordHTTPRequest(r.Method, r.URL.Path, 200, duration)
	})
}