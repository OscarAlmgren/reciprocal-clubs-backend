package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/messaging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	"reciprocal-clubs-backend/services/blockchain-service/internal/models"
	"reciprocal-clubs-backend/services/blockchain-service/internal/repository"
)

// BlockchainService handles business logic for blockchain operations
type BlockchainService struct {
	repo       *repository.Repository
	logger     logging.Logger
	messaging  messaging.MessageBus
	monitoring *monitoring.Monitor
}

// NewService creates a new blockchain service
func NewService(repo *repository.Repository, logger logging.Logger, messaging messaging.MessageBus, monitoring *monitoring.Monitor) *BlockchainService {
	return &BlockchainService{
		repo:       repo,
		logger:     logger,
		messaging:  messaging,
		monitoring: monitoring,
	}
}

// Transaction operations

// CreateTransaction creates a new blockchain transaction
func (s *BlockchainService) CreateTransaction(ctx context.Context, req *CreateTransactionRequest) (*models.Transaction, error) {
	transaction := &models.Transaction{
		ClubID:      req.ClubID,
		UserID:      req.UserID,
		Network:     req.Network,
		Type:        req.Type,
		FromAddress: req.FromAddress,
		ToAddress:   req.ToAddress,
		Value:       req.Value,
		GasLimit:    req.GasLimit,
		GasPrice:    req.GasPrice,
		Data:        req.Data,
		Status:      models.TransactionStatusPending,
	}

	// Set metadata if provided
	if req.Metadata != nil {
		if err := transaction.SetMetadata(req.Metadata); err != nil {
			return nil, fmt.Errorf("failed to set metadata: %v", err)
		}
	}

	if err := s.repo.CreateTransaction(ctx, transaction); err != nil {
		s.monitoring.RecordBusinessEvent("transaction_create_error", fmt.Sprintf("%d", req.ClubID))
		return nil, err
	}

	s.monitoring.RecordBusinessEvent("transaction_created", fmt.Sprintf("%d", req.ClubID))

	// Publish transaction created event
	s.publishTransactionEvent(ctx, "transaction.created", transaction)

	s.logger.Info("Transaction created", map[string]interface{}{
		"transaction_id": transaction.ID,
		"club_id":        transaction.ClubID,
		"network":        transaction.Network,
		"type":           transaction.Type,
		"from_address":   transaction.FromAddress,
		"to_address":     transaction.ToAddress,
	})

	return transaction, nil
}

// GetTransactionByID retrieves a transaction by ID
func (s *BlockchainService) GetTransactionByID(ctx context.Context, id uint) (*models.Transaction, error) {
	transaction, err := s.repo.GetTransactionByID(ctx, id)
	if err != nil {
		s.monitoring.RecordBusinessEvent("transaction_get_error", "1")
		return nil, err
	}

	return transaction, nil
}

// GetTransactionByHash retrieves a transaction by hash
func (s *BlockchainService) GetTransactionByHash(ctx context.Context, hash string) (*models.Transaction, error) {
	transaction, err := s.repo.GetTransactionByHash(ctx, hash)
	if err != nil {
		s.monitoring.RecordBusinessEvent("transaction_get_by_hash_error", "1")
		return nil, err
	}

	return transaction, nil
}

// GetTransactionsByClub retrieves transactions for a club
func (s *BlockchainService) GetTransactionsByClub(ctx context.Context, clubID uint, network models.Network, limit, offset int) ([]models.Transaction, error) {
	transactions, err := s.repo.GetTransactionsByClub(ctx, clubID, network, limit, offset)
	if err != nil {
		s.monitoring.RecordBusinessEvent("transactions_get_error", fmt.Sprintf("%d", clubID))
		return nil, err
	}

	return transactions, nil
}

// GetTransactionsByUser retrieves transactions for a user
func (s *BlockchainService) GetTransactionsByUser(ctx context.Context, userID string, clubID uint, network models.Network, limit, offset int) ([]models.Transaction, error) {
	transactions, err := s.repo.GetTransactionsByUser(ctx, userID, clubID, network, limit, offset)
	if err != nil {
		s.monitoring.RecordBusinessEvent("user_transactions_get_error", fmt.Sprintf("%d", clubID))
		return nil, err
	}

	return transactions, nil
}

// SubmitTransaction submits a transaction to the blockchain
func (s *BlockchainService) SubmitTransaction(ctx context.Context, id uint, hash string, nonce uint64) (*models.Transaction, error) {
	transaction, err := s.repo.GetTransactionByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if transaction.Status != models.TransactionStatusPending {
		return nil, fmt.Errorf("transaction is not in pending status")
	}

	transaction.MarkAsSubmitted(hash)
	transaction.Nonce = nonce

	if err := s.repo.UpdateTransaction(ctx, transaction); err != nil {
		s.monitoring.RecordBusinessEvent("transaction_submit_error", fmt.Sprintf("%d", transaction.ClubID))
		return nil, err
	}

	s.monitoring.RecordBusinessEvent("transaction_submitted", fmt.Sprintf("%d", transaction.ClubID))

	// Publish transaction submitted event
	s.publishTransactionEvent(ctx, "transaction.submitted", transaction)

	s.logger.Info("Transaction submitted", map[string]interface{}{
		"transaction_id": transaction.ID,
		"hash":           transaction.Hash,
		"nonce":          transaction.Nonce,
	})

	return transaction, nil
}

// ConfirmTransaction confirms a transaction on the blockchain
func (s *BlockchainService) ConfirmTransaction(ctx context.Context, hash string, blockNumber uint64, blockHash string, gasUsed uint64) (*models.Transaction, error) {
	transaction, err := s.repo.GetTransactionByHash(ctx, hash)
	if err != nil {
		return nil, err
	}

	transaction.MarkAsConfirmed(blockNumber, blockHash, gasUsed)
	transaction.ConfirmationCount++

	if err := s.repo.UpdateTransaction(ctx, transaction); err != nil {
		s.monitoring.RecordBusinessEvent("transaction_confirm_error", fmt.Sprintf("%d", transaction.ClubID))
		return nil, err
	}

	s.monitoring.RecordBusinessEvent("transaction_confirmed", fmt.Sprintf("%d", transaction.ClubID))

	// Publish transaction confirmed event
	s.publishTransactionEvent(ctx, "transaction.confirmed", transaction)

	s.logger.Info("Transaction confirmed", map[string]interface{}{
		"transaction_id": transaction.ID,
		"hash":           transaction.Hash,
		"block_number":   transaction.BlockNumber,
		"confirmations":  transaction.ConfirmationCount,
	})

	return transaction, nil
}

// FailTransaction marks a transaction as failed
func (s *BlockchainService) FailTransaction(ctx context.Context, hash string, errorMsg string) (*models.Transaction, error) {
	transaction, err := s.repo.GetTransactionByHash(ctx, hash)
	if err != nil {
		return nil, err
	}

	transaction.MarkAsFailed(errorMsg)

	if err := s.repo.UpdateTransaction(ctx, transaction); err != nil {
		s.monitoring.RecordBusinessEvent("transaction_fail_error", fmt.Sprintf("%d", transaction.ClubID))
		return nil, err
	}

	s.monitoring.RecordBusinessEvent("transaction_failed", fmt.Sprintf("%d", transaction.ClubID))

	// Publish transaction failed event
	s.publishTransactionEvent(ctx, "transaction.failed", transaction)

	s.logger.Error("Transaction failed", map[string]interface{}{
		"transaction_id": transaction.ID,
		"hash":           transaction.Hash,
		"error":          errorMsg,
	})

	return transaction, nil
}

// Contract operations

// CreateContract creates a new smart contract record
func (s *BlockchainService) CreateContract(ctx context.Context, req *CreateContractRequest) (*models.Contract, error) {
	contract := &models.Contract{
		ClubID:      req.ClubID,
		Network:     req.Network,
		Name:        req.Name,
		Address:     req.Address,
		ABI:         req.ABI,
		Bytecode:    req.Bytecode,
		Version:     req.Version,
		IsDeployed:  req.IsDeployed,
		DeployedBy:  req.DeployedBy,
		Description: req.Description,
	}

	if req.IsDeployed {
		now := time.Now()
		contract.DeployedAt = &now
	}

	if err := s.repo.CreateContract(ctx, contract); err != nil {
		s.monitoring.RecordBusinessEvent("contract_create_error", fmt.Sprintf("%d", req.ClubID))
		return nil, err
	}

	s.monitoring.RecordBusinessEvent("contract_created", fmt.Sprintf("%d", req.ClubID))

	// Publish contract created event
	s.publishContractEvent(ctx, "contract.created", contract)

	return contract, nil
}

// GetContractByID retrieves a contract by ID
func (s *BlockchainService) GetContractByID(ctx context.Context, id uint) (*models.Contract, error) {
	contract, err := s.repo.GetContractByID(ctx, id)
	if err != nil {
		s.monitoring.RecordBusinessEvent("contract_get_error", "1")
		return nil, err
	}

	return contract, nil
}

// GetContractsByClub retrieves contracts for a club
func (s *BlockchainService) GetContractsByClub(ctx context.Context, clubID uint, network models.Network) ([]models.Contract, error) {
	contracts, err := s.repo.GetContractsByClub(ctx, clubID, network)
	if err != nil {
		s.monitoring.RecordBusinessEvent("contracts_get_error", fmt.Sprintf("%d", clubID))
		return nil, err
	}

	return contracts, nil
}

// Wallet operations

// CreateWallet creates a new wallet record
func (s *BlockchainService) CreateWallet(ctx context.Context, req *CreateWalletRequest) (*models.Wallet, error) {
	wallet := &models.Wallet{
		ClubID:   req.ClubID,
		UserID:   req.UserID,
		Network:  req.Network,
		Address:  req.Address,
		Balance:  "0",
		IsActive: true,
	}

	if err := s.repo.CreateWallet(ctx, wallet); err != nil {
		s.monitoring.RecordBusinessEvent("wallet_create_error", fmt.Sprintf("%d", req.ClubID))
		return nil, err
	}

	s.monitoring.RecordBusinessEvent("wallet_created", fmt.Sprintf("%d", req.ClubID))

	// Publish wallet created event
	s.publishWalletEvent(ctx, "wallet.created", wallet)

	return wallet, nil
}

// GetWalletsByUser retrieves wallets for a user
func (s *BlockchainService) GetWalletsByUser(ctx context.Context, userID string, clubID uint, network models.Network) ([]models.Wallet, error) {
	wallets, err := s.repo.GetWalletsByUser(ctx, userID, clubID, network)
	if err != nil {
		s.monitoring.RecordBusinessEvent("wallets_get_error", fmt.Sprintf("%d", clubID))
		return nil, err
	}

	return wallets, nil
}

// UpdateWalletBalance updates a wallet's balance
func (s *BlockchainService) UpdateWalletBalance(ctx context.Context, address string, network models.Network, balance string, tokenBalances map[string]string) (*models.Wallet, error) {
	wallet, err := s.repo.GetWalletByAddress(ctx, address, network)
	if err != nil {
		return nil, err
	}

	wallet.UpdateBalance(balance)

	if tokenBalances != nil {
		if err := wallet.SetTokenBalances(tokenBalances); err != nil {
			return nil, fmt.Errorf("failed to set token balances: %v", err)
		}
	}

	if err := s.repo.UpdateWallet(ctx, wallet); err != nil {
		s.monitoring.RecordBusinessEvent("wallet_balance_update_error", fmt.Sprintf("%d", wallet.ClubID))
		return nil, err
	}

	s.monitoring.RecordBusinessEvent("wallet_balance_updated", fmt.Sprintf("%d", wallet.ClubID))

	// Publish wallet updated event
	s.publishWalletEvent(ctx, "wallet.balance_updated", wallet)

	return wallet, nil
}

// Token operations

// CreateToken creates a new token record
func (s *BlockchainService) CreateToken(ctx context.Context, req *CreateTokenRequest) (*models.Token, error) {
	token := &models.Token{
		ClubID:      req.ClubID,
		Network:     req.Network,
		ContractID:  req.ContractID,
		Name:        req.Name,
		Symbol:      req.Symbol,
		Decimals:    req.Decimals,
		TotalSupply: req.TotalSupply,
		MaxSupply:   req.MaxSupply,
		IsMintable:  req.IsMintable,
		IsBurnable:  req.IsBurnable,
		IsPausable:  req.IsPausable,
		IsActive:    true,
	}

	if err := s.repo.CreateToken(ctx, token); err != nil {
		s.monitoring.RecordBusinessEvent("token_create_error", fmt.Sprintf("%d", req.ClubID))
		return nil, err
	}

	s.monitoring.RecordBusinessEvent("token_created", fmt.Sprintf("%d", req.ClubID))

	// Publish token created event
	s.publishTokenEvent(ctx, "token.created", token)

	return token, nil
}

// GetTokensByClub retrieves tokens for a club
func (s *BlockchainService) GetTokensByClub(ctx context.Context, clubID uint, network models.Network) ([]models.Token, error) {
	tokens, err := s.repo.GetTokensByClub(ctx, clubID, network)
	if err != nil {
		s.monitoring.RecordBusinessEvent("tokens_get_error", fmt.Sprintf("%d", clubID))
		return nil, err
	}

	return tokens, nil
}

// ProcessPendingTransactions processes pending transactions for monitoring
func (s *BlockchainService) ProcessPendingTransactions(ctx context.Context, network models.Network) error {
	transactions, err := s.repo.GetPendingTransactions(ctx, network, 100)
	if err != nil {
		return err
	}

	for _, transaction := range transactions {
		// Here you would implement actual blockchain monitoring logic
		// This is a placeholder for the real implementation
		s.logger.Info("Processing pending transaction", map[string]interface{}{
			"transaction_id": transaction.ID,
			"hash":           transaction.Hash,
			"network":        transaction.Network,
			"status":         transaction.Status,
		})

		// Example: check transaction status on blockchain
		// If confirmed, call s.ConfirmTransaction()
		// If failed, call s.FailTransaction()
	}

	return nil
}

// GetTransactionStats retrieves transaction statistics
func (s *BlockchainService) GetTransactionStats(ctx context.Context, clubID uint, network models.Network, fromDate, toDate time.Time) (map[string]interface{}, error) {
	stats, err := s.repo.GetTransactionStats(ctx, clubID, network, fromDate, toDate)
	if err != nil {
		s.monitoring.RecordBusinessEvent("stats_get_error", fmt.Sprintf("%d", clubID))
		return nil, err
	}

	return stats, nil
}

// Helper methods for event publishing

func (s *BlockchainService) publishTransactionEvent(ctx context.Context, eventType string, transaction *models.Transaction) {
	data := map[string]interface{}{
		"transaction_id": transaction.ID,
		"club_id":        transaction.ClubID,
		"user_id":        transaction.UserID,
		"network":        transaction.Network,
		"type":           transaction.Type,
		"status":         transaction.Status,
		"hash":           transaction.Hash,
		"from_address":   transaction.FromAddress,
		"to_address":     transaction.ToAddress,
		"value":          transaction.Value,
		"timestamp":      time.Now(),
	}

	jsonData, _ := json.Marshal(data)
	if err := s.messaging.Publish(ctx, eventType, jsonData); err != nil {
		s.logger.Error("Failed to publish transaction event", map[string]interface{}{
			"error":          err.Error(),
			"event_type":     eventType,
			"transaction_id": transaction.ID,
		})
	}
}

func (s *BlockchainService) publishContractEvent(ctx context.Context, eventType string, contract *models.Contract) {
	data := map[string]interface{}{
		"contract_id": contract.ID,
		"club_id":     contract.ClubID,
		"network":     contract.Network,
		"name":        contract.Name,
		"address":     contract.Address,
		"is_deployed": contract.IsDeployed,
		"timestamp":   time.Now(),
	}

	jsonData, _ := json.Marshal(data)
	if err := s.messaging.Publish(ctx, eventType, jsonData); err != nil {
		s.logger.Error("Failed to publish contract event", map[string]interface{}{
			"error":       err.Error(),
			"event_type":  eventType,
			"contract_id": contract.ID,
		})
	}
}

func (s *BlockchainService) publishWalletEvent(ctx context.Context, eventType string, wallet *models.Wallet) {
	data := map[string]interface{}{
		"wallet_id": wallet.ID,
		"club_id":   wallet.ClubID,
		"user_id":   wallet.UserID,
		"network":   wallet.Network,
		"address":   wallet.Address,
		"balance":   wallet.Balance,
		"timestamp": time.Now(),
	}

	jsonData, _ := json.Marshal(data)
	if err := s.messaging.Publish(ctx, eventType, jsonData); err != nil {
		s.logger.Error("Failed to publish wallet event", map[string]interface{}{
			"error":     err.Error(),
			"event_type": eventType,
			"wallet_id": wallet.ID,
		})
	}
}

func (s *BlockchainService) publishTokenEvent(ctx context.Context, eventType string, token *models.Token) {
	data := map[string]interface{}{
		"token_id":     token.ID,
		"club_id":      token.ClubID,
		"network":      token.Network,
		"name":         token.Name,
		"symbol":       token.Symbol,
		"total_supply": token.TotalSupply,
		"timestamp":    time.Now(),
	}

	jsonData, _ := json.Marshal(data)
	if err := s.messaging.Publish(ctx, eventType, jsonData); err != nil {
		s.logger.Error("Failed to publish token event", map[string]interface{}{
			"error":      err.Error(),
			"event_type": eventType,
			"token_id":   token.ID,
		})
	}
}

// Request/Response types

type CreateTransactionRequest struct {
	ClubID      uint                   `json:"club_id" validate:"required"`
	UserID      string                 `json:"user_id" validate:"required"`
	Network     models.Network         `json:"network" validate:"required"`
	Type        models.TransactionType `json:"type" validate:"required"`
	FromAddress string                 `json:"from_address" validate:"required"`
	ToAddress   string                 `json:"to_address" validate:"required"`
	Value       string                 `json:"value" validate:"required"`
	GasLimit    uint64                 `json:"gas_limit" validate:"required"`
	GasPrice    string                 `json:"gas_price" validate:"required"`
	Data        string                 `json:"data,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type CreateContractRequest struct {
	ClubID      uint           `json:"club_id" validate:"required"`
	Network     models.Network `json:"network" validate:"required"`
	Name        string         `json:"name" validate:"required"`
	Address     string         `json:"address" validate:"required"`
	ABI         string         `json:"abi"`
	Bytecode    string         `json:"bytecode,omitempty"`
	Version     string         `json:"version"`
	IsDeployed  bool           `json:"is_deployed"`
	DeployedBy  string         `json:"deployed_by,omitempty"`
	Description string         `json:"description,omitempty"`
}

type CreateWalletRequest struct {
	ClubID  uint           `json:"club_id" validate:"required"`
	UserID  string         `json:"user_id" validate:"required"`
	Network models.Network `json:"network" validate:"required"`
	Address string         `json:"address" validate:"required"`
}

type CreateTokenRequest struct {
	ClubID      uint           `json:"club_id" validate:"required"`
	Network     models.Network `json:"network" validate:"required"`
	ContractID  uint           `json:"contract_id" validate:"required"`
	Name        string         `json:"name" validate:"required"`
	Symbol      string         `json:"symbol" validate:"required"`
	Decimals    uint8          `json:"decimals" validate:"required"`
	TotalSupply string         `json:"total_supply" validate:"required"`
	MaxSupply   string         `json:"max_supply,omitempty"`
	IsMintable  bool           `json:"is_mintable"`
	IsBurnable  bool           `json:"is_burnable"`
	IsPausable  bool           `json:"is_pausable"`
}