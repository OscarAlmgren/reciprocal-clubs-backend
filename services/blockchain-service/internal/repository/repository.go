package repository

import (
	"context"
	"time"

	"reciprocal-clubs-backend/pkg/shared/database"
	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/services/blockchain-service/internal/models"

	"gorm.io/gorm"
)

// Repository handles database operations for blockchain service
type Repository struct {
	*database.BaseRepository
	db     *gorm.DB
	logger logging.Logger
}

// NewRepository creates a new blockchain repository
func NewRepository(db *gorm.DB, logger logging.Logger) *Repository {
	return &Repository{
		db:     db,
		logger: logger,
	}
}

// Transaction operations

// CreateTransaction creates a new blockchain transaction record
func (r *Repository) CreateTransaction(ctx context.Context, transaction *models.Transaction) error {
	if err := r.db.WithContext(ctx).Create(transaction).Error; err != nil {
		r.logger.Error("Failed to create transaction", map[string]interface{}{
			"error":        err.Error(),
			"club_id":      transaction.ClubID,
			"user_id":      transaction.UserID,
			"network":      transaction.Network,
			"type":         transaction.Type,
			"from_address": transaction.FromAddress,
			"to_address":   transaction.ToAddress,
		})
		return err
	}

	r.logger.Info("Transaction created successfully", map[string]interface{}{
		"transaction_id": transaction.ID,
		"club_id":        transaction.ClubID,
		"network":        transaction.Network,
		"type":           transaction.Type,
		"status":         transaction.Status,
	})

	return nil
}

// GetTransactionByID retrieves a transaction by ID
func (r *Repository) GetTransactionByID(ctx context.Context, id uint) (*models.Transaction, error) {
	var transaction models.Transaction
	if err := r.db.WithContext(ctx).First(&transaction, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		r.logger.Error("Failed to get transaction", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		return nil, err
	}

	return &transaction, nil
}

// GetTransactionByHash retrieves a transaction by hash
func (r *Repository) GetTransactionByHash(ctx context.Context, hash string) (*models.Transaction, error) {
	var transaction models.Transaction
	if err := r.db.WithContext(ctx).Where("hash = ?", hash).First(&transaction).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		r.logger.Error("Failed to get transaction by hash", map[string]interface{}{
			"error": err.Error(),
			"hash":  hash,
		})
		return nil, err
	}

	return &transaction, nil
}

// GetTransactionsByClub retrieves transactions for a specific club
func (r *Repository) GetTransactionsByClub(ctx context.Context, clubID uint, network models.Network, limit, offset int) ([]models.Transaction, error) {
	var transactions []models.Transaction
	query := r.db.WithContext(ctx).Where("club_id = ?", clubID)

	if network != "" {
		query = query.Where("network = ?", network)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Order("created_at DESC").Find(&transactions).Error; err != nil {
		r.logger.Error("Failed to get transactions by club", map[string]interface{}{
			"error":   err.Error(),
			"club_id": clubID,
			"network": network,
		})
		return nil, err
	}

	return transactions, nil
}

// GetTransactionsByUser retrieves transactions for a specific user
func (r *Repository) GetTransactionsByUser(ctx context.Context, userID string, clubID uint, network models.Network, limit, offset int) ([]models.Transaction, error) {
	var transactions []models.Transaction
	query := r.db.WithContext(ctx).Where("user_id = ? AND club_id = ?", userID, clubID)

	if network != "" {
		query = query.Where("network = ?", network)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Order("created_at DESC").Find(&transactions).Error; err != nil {
		r.logger.Error("Failed to get transactions by user", map[string]interface{}{
			"error":   err.Error(),
			"user_id": userID,
			"club_id": clubID,
			"network": network,
		})
		return nil, err
	}

	return transactions, nil
}

// GetPendingTransactions retrieves pending transactions
func (r *Repository) GetPendingTransactions(ctx context.Context, network models.Network, limit int) ([]models.Transaction, error) {
	var transactions []models.Transaction
	query := r.db.WithContext(ctx).Where("status IN ?", []models.TransactionStatus{
		models.TransactionStatusPending,
		models.TransactionStatusSubmitted,
	})

	if network != "" {
		query = query.Where("network = ?", network)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Order("created_at ASC").Find(&transactions).Error; err != nil {
		r.logger.Error("Failed to get pending transactions", map[string]interface{}{
			"error":   err.Error(),
			"network": network,
		})
		return nil, err
	}

	return transactions, nil
}

// UpdateTransaction updates an existing transaction
func (r *Repository) UpdateTransaction(ctx context.Context, transaction *models.Transaction) error {
	if err := r.db.WithContext(ctx).Save(transaction).Error; err != nil {
		r.logger.Error("Failed to update transaction", map[string]interface{}{
			"error":          err.Error(),
			"transaction_id": transaction.ID,
		})
		return err
	}

	r.logger.Info("Transaction updated successfully", map[string]interface{}{
		"transaction_id": transaction.ID,
		"status":         transaction.Status,
		"hash":           transaction.Hash,
	})

	return nil
}

// Contract operations

// CreateContract creates a new smart contract record
func (r *Repository) CreateContract(ctx context.Context, contract *models.Contract) error {
	if err := r.db.WithContext(ctx).Create(contract).Error; err != nil {
		r.logger.Error("Failed to create contract", map[string]interface{}{
			"error":   err.Error(),
			"club_id": contract.ClubID,
			"network": contract.Network,
			"name":    contract.Name,
		})
		return err
	}

	r.logger.Info("Contract created successfully", map[string]interface{}{
		"contract_id": contract.ID,
		"club_id":     contract.ClubID,
		"network":     contract.Network,
		"name":        contract.Name,
	})

	return nil
}

// GetContractByID retrieves a contract by ID
func (r *Repository) GetContractByID(ctx context.Context, id uint) (*models.Contract, error) {
	var contract models.Contract
	if err := r.db.WithContext(ctx).First(&contract, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		r.logger.Error("Failed to get contract", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		return nil, err
	}

	return &contract, nil
}

// GetContractByAddress retrieves a contract by address
func (r *Repository) GetContractByAddress(ctx context.Context, address string, network models.Network) (*models.Contract, error) {
	var contract models.Contract
	if err := r.db.WithContext(ctx).Where("address = ? AND network = ?", address, network).First(&contract).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		r.logger.Error("Failed to get contract by address", map[string]interface{}{
			"error":   err.Error(),
			"address": address,
			"network": network,
		})
		return nil, err
	}

	return &contract, nil
}

// GetContractsByClub retrieves contracts for a specific club
func (r *Repository) GetContractsByClub(ctx context.Context, clubID uint, network models.Network) ([]models.Contract, error) {
	var contracts []models.Contract
	query := r.db.WithContext(ctx).Where("club_id = ?", clubID)

	if network != "" {
		query = query.Where("network = ?", network)
	}

	if err := query.Order("created_at DESC").Find(&contracts).Error; err != nil {
		r.logger.Error("Failed to get contracts by club", map[string]interface{}{
			"error":   err.Error(),
			"club_id": clubID,
			"network": network,
		})
		return nil, err
	}

	return contracts, nil
}

// UpdateContract updates an existing contract
func (r *Repository) UpdateContract(ctx context.Context, contract *models.Contract) error {
	if err := r.db.WithContext(ctx).Save(contract).Error; err != nil {
		r.logger.Error("Failed to update contract", map[string]interface{}{
			"error":       err.Error(),
			"contract_id": contract.ID,
		})
		return err
	}

	r.logger.Info("Contract updated successfully", map[string]interface{}{
		"contract_id": contract.ID,
		"name":        contract.Name,
		"is_deployed": contract.IsDeployed,
	})

	return nil
}

// Wallet operations

// CreateWallet creates a new wallet record
func (r *Repository) CreateWallet(ctx context.Context, wallet *models.Wallet) error {
	if err := r.db.WithContext(ctx).Create(wallet).Error; err != nil {
		r.logger.Error("Failed to create wallet", map[string]interface{}{
			"error":   err.Error(),
			"club_id": wallet.ClubID,
			"user_id": wallet.UserID,
			"network": wallet.Network,
			"address": wallet.Address,
		})
		return err
	}

	r.logger.Info("Wallet created successfully", map[string]interface{}{
		"wallet_id": wallet.ID,
		"club_id":   wallet.ClubID,
		"user_id":   wallet.UserID,
		"network":   wallet.Network,
		"address":   wallet.Address,
	})

	return nil
}

// GetWalletByID retrieves a wallet by ID
func (r *Repository) GetWalletByID(ctx context.Context, id uint) (*models.Wallet, error) {
	var wallet models.Wallet
	if err := r.db.WithContext(ctx).First(&wallet, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		r.logger.Error("Failed to get wallet", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		return nil, err
	}

	return &wallet, nil
}

// GetWalletByAddress retrieves a wallet by address
func (r *Repository) GetWalletByAddress(ctx context.Context, address string, network models.Network) (*models.Wallet, error) {
	var wallet models.Wallet
	if err := r.db.WithContext(ctx).Where("address = ? AND network = ?", address, network).First(&wallet).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		r.logger.Error("Failed to get wallet by address", map[string]interface{}{
			"error":   err.Error(),
			"address": address,
			"network": network,
		})
		return nil, err
	}

	return &wallet, nil
}

// GetWalletsByUser retrieves wallets for a specific user
func (r *Repository) GetWalletsByUser(ctx context.Context, userID string, clubID uint, network models.Network) ([]models.Wallet, error) {
	var wallets []models.Wallet
	query := r.db.WithContext(ctx).Where("user_id = ? AND club_id = ?", userID, clubID)

	if network != "" {
		query = query.Where("network = ?", network)
	}

	if err := query.Find(&wallets).Error; err != nil {
		r.logger.Error("Failed to get wallets by user", map[string]interface{}{
			"error":   err.Error(),
			"user_id": userID,
			"club_id": clubID,
			"network": network,
		})
		return nil, err
	}

	return wallets, nil
}

// UpdateWallet updates an existing wallet
func (r *Repository) UpdateWallet(ctx context.Context, wallet *models.Wallet) error {
	if err := r.db.WithContext(ctx).Save(wallet).Error; err != nil {
		r.logger.Error("Failed to update wallet", map[string]interface{}{
			"error":     err.Error(),
			"wallet_id": wallet.ID,
		})
		return err
	}

	return nil
}

// Token operations

// CreateToken creates a new token record
func (r *Repository) CreateToken(ctx context.Context, token *models.Token) error {
	if err := r.db.WithContext(ctx).Create(token).Error; err != nil {
		r.logger.Error("Failed to create token", map[string]interface{}{
			"error":   err.Error(),
			"club_id": token.ClubID,
			"network": token.Network,
			"name":    token.Name,
			"symbol":  token.Symbol,
		})
		return err
	}

	r.logger.Info("Token created successfully", map[string]interface{}{
		"token_id": token.ID,
		"club_id":  token.ClubID,
		"network":  token.Network,
		"name":     token.Name,
		"symbol":   token.Symbol,
	})

	return nil
}

// GetTokenByID retrieves a token by ID
func (r *Repository) GetTokenByID(ctx context.Context, id uint) (*models.Token, error) {
	var token models.Token
	if err := r.db.WithContext(ctx).Preload("Contract").First(&token, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		r.logger.Error("Failed to get token", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		return nil, err
	}

	return &token, nil
}

// GetTokensByClub retrieves tokens for a specific club
func (r *Repository) GetTokensByClub(ctx context.Context, clubID uint, network models.Network) ([]models.Token, error) {
	var tokens []models.Token
	query := r.db.WithContext(ctx).Preload("Contract").Where("club_id = ?", clubID)

	if network != "" {
		query = query.Where("network = ?", network)
	}

	if err := query.Find(&tokens).Error; err != nil {
		r.logger.Error("Failed to get tokens by club", map[string]interface{}{
			"error":   err.Error(),
			"club_id": clubID,
			"network": network,
		})
		return nil, err
	}

	return tokens, nil
}

// UpdateToken updates an existing token
func (r *Repository) UpdateToken(ctx context.Context, token *models.Token) error {
	if err := r.db.WithContext(ctx).Save(token).Error; err != nil {
		r.logger.Error("Failed to update token", map[string]interface{}{
			"error":    err.Error(),
			"token_id": token.ID,
		})
		return err
	}

	return nil
}

// Block operations

// CreateBlock creates a new block record
func (r *Repository) CreateBlock(ctx context.Context, block *models.Block) error {
	if err := r.db.WithContext(ctx).Create(block).Error; err != nil {
		r.logger.Error("Failed to create block", map[string]interface{}{
			"error":   err.Error(),
			"network": block.Network,
			"number":  block.Number,
			"hash":    block.Hash,
		})
		return err
	}

	return nil
}

// GetLatestBlock retrieves the latest block for a network
func (r *Repository) GetLatestBlock(ctx context.Context, network models.Network) (*models.Block, error) {
	var block models.Block
	if err := r.db.WithContext(ctx).Where("network = ?", network).Order("number DESC").First(&block).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		r.logger.Error("Failed to get latest block", map[string]interface{}{
			"error":   err.Error(),
			"network": network,
		})
		return nil, err
	}

	return &block, nil
}

// Event operations

// CreateEvent creates a new event record
func (r *Repository) CreateEvent(ctx context.Context, event *models.Event) error {
	if err := r.db.WithContext(ctx).Create(event).Error; err != nil {
		r.logger.Error("Failed to create event", map[string]interface{}{
			"error":            err.Error(),
			"club_id":          event.ClubID,
			"network":          event.Network,
			"contract_address": event.ContractAddress,
			"event_name":       event.EventName,
			"transaction_hash": event.TransactionHash,
		})
		return err
	}

	return nil
}

// GetEventsByContract retrieves events for a specific contract
func (r *Repository) GetEventsByContract(ctx context.Context, contractAddress string, network models.Network, eventName string, limit, offset int) ([]models.Event, error) {
	var events []models.Event
	query := r.db.WithContext(ctx).Where("contract_address = ? AND network = ?", contractAddress, network)

	if eventName != "" {
		query = query.Where("event_name = ?", eventName)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Order("block_number DESC, log_index DESC").Find(&events).Error; err != nil {
		r.logger.Error("Failed to get events by contract", map[string]interface{}{
			"error":            err.Error(),
			"contract_address": contractAddress,
			"network":          network,
			"event_name":       eventName,
		})
		return nil, err
	}

	return events, nil
}

// GetTransactionStats retrieves transaction statistics for a club
func (r *Repository) GetTransactionStats(ctx context.Context, clubID uint, network models.Network, fromDate, toDate time.Time) (map[string]interface{}, error) {
	var stats struct {
		Total     int64 `json:"total"`
		Pending   int64 `json:"pending"`
		Confirmed int64 `json:"confirmed"`
		Failed    int64 `json:"failed"`
	}

	query := r.db.WithContext(ctx).Model(&models.Transaction{}).Where("club_id = ?", clubID)

	if network != "" {
		query = query.Where("network = ?", network)
	}

	if !fromDate.IsZero() && !toDate.IsZero() {
		query = query.Where("created_at BETWEEN ? AND ?", fromDate, toDate)
	}

	// Get total count
	if err := query.Count(&stats.Total).Error; err != nil {
		return nil, err
	}

	// Get status counts
	statusCounts := make(map[string]int64)
	rows, err := query.Select("status, COUNT(*) as count").Group("status").Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			continue
		}
		statusCounts[status] = count
	}

	return map[string]interface{}{
		"total":     stats.Total,
		"pending":   statusCounts["pending"] + statusCounts["submitted"],
		"confirmed": statusCounts["confirmed"],
		"failed":    statusCounts["failed"],
	}, nil
}