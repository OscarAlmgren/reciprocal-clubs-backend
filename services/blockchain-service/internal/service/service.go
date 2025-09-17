package service

import (
	"context"
	"fmt"
	"time"

	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/messaging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	"reciprocal-clubs-backend/services/blockchain-service/internal/models"
	"reciprocal-clubs-backend/services/blockchain-service/internal/repository"
)

// BlockchainService handles business logic for Hyperledger Fabric operations
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

// Fabric Transaction operations

// CreateTransaction creates a new Hyperledger Fabric transaction
func (s *BlockchainService) CreateTransaction(ctx context.Context, req *CreateTransactionRequest) (*models.FabricTransaction, error) {
	transaction := &models.FabricTransaction{
		ClubID:        req.ClubID,
		UserID:        req.UserID,
		Type:          req.Type,
		ChannelID:     req.ChannelID,
		ChaincodeName: req.ChaincodeName,
		Function:      req.Function,
		Status:        models.FabricTransactionStatusPending,
		ClientIdentity: req.ClientIdentity,
	}

	// Set transaction arguments
	transaction.SetArgs(req.Args)

	// Set transient map if provided
	if req.TransientMap != nil {
		if err := transaction.SetTransientMap(req.TransientMap); err != nil {
			return nil, fmt.Errorf("failed to set transient map: %v", err)
		}
	}

	// Set metadata if provided
	if req.Metadata != nil {
		if err := transaction.SetMetadata(req.Metadata); err != nil {
			return nil, fmt.Errorf("failed to set metadata: %v", err)
		}
	}

	if err := s.repo.CreateTransaction(ctx, transaction); err != nil {
		s.monitoring.RecordBusinessEvent("fabric_transaction_create_error", fmt.Sprintf("%d", req.ClubID))
		return nil, err
	}

	s.monitoring.RecordBusinessEvent("fabric_transaction_created", fmt.Sprintf("%d", req.ClubID))

	// Publish transaction created event
	s.publishTransactionEvent(ctx, "fabric.transaction.created", transaction)

	s.logger.Info("Fabric transaction created", map[string]interface{}{
		"transaction_id":   transaction.ID,
		"club_id":          transaction.ClubID,
		"channel_id":       transaction.ChannelID,
		"chaincode_name":   transaction.ChaincodeName,
		"function":         transaction.Function,
		"type":             transaction.Type,
	})

	return transaction, nil
}

// GetTransactionByID retrieves a transaction by ID
func (s *BlockchainService) GetTransactionByID(ctx context.Context, id uint) (*models.FabricTransaction, error) {
	transaction, err := s.repo.GetTransactionByID(ctx, id)
	if err != nil {
		s.monitoring.RecordBusinessEvent("fabric_transaction_get_error", "1")
		return nil, err
	}

	return transaction, nil
}

// GetTransactionByTxID retrieves a transaction by Fabric transaction ID
func (s *BlockchainService) GetTransactionByTxID(ctx context.Context, txID string) (*models.FabricTransaction, error) {
	transaction, err := s.repo.GetTransactionByTxID(ctx, txID)
	if err != nil {
		s.monitoring.RecordBusinessEvent("fabric_transaction_get_by_txid_error", "1")
		return nil, err
	}

	return transaction, nil
}

// GetTransactionsByClubID retrieves transactions for a club
func (s *BlockchainService) GetTransactionsByClubID(ctx context.Context, clubID uint, limit, offset int) ([]*models.FabricTransaction, error) {
	transactions, err := s.repo.GetTransactionsByClubID(ctx, clubID, limit, offset)
	if err != nil {
		s.monitoring.RecordBusinessEvent("fabric_transactions_get_error", fmt.Sprintf("%d", clubID))
		return nil, err
	}

	return transactions, nil
}

// SubmitTransaction submits a transaction to Hyperledger Fabric
func (s *BlockchainService) SubmitTransaction(ctx context.Context, id uint, txID string, endorsingPeers []string) (*models.FabricTransaction, error) {
	transaction, err := s.repo.GetTransactionByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Mark as submitted
	transaction.MarkAsSubmitted(txID, endorsingPeers)

	if err := s.repo.UpdateTransaction(ctx, transaction); err != nil {
		s.monitoring.RecordBusinessEvent("fabric_transaction_submit_error", fmt.Sprintf("%d", transaction.ClubID))
		return nil, err
	}

	s.monitoring.RecordBusinessEvent("fabric_transaction_submitted", fmt.Sprintf("%d", transaction.ClubID))

	// Publish transaction submitted event
	s.publishTransactionEvent(ctx, "fabric.transaction.submitted", transaction)

	s.logger.Info("Fabric transaction submitted", map[string]interface{}{
		"transaction_id":    transaction.ID,
		"club_id":           transaction.ClubID,
		"tx_id":             transaction.TxID,
		"endorsing_peers":   transaction.EndorsingPeers,
		"endorsement_count": transaction.EndorsementCount,
	})

	return transaction, nil
}

// ConfirmTransaction confirms a transaction in a block
func (s *BlockchainService) ConfirmTransaction(ctx context.Context, txID string, blockNumber uint64, blockHash string, txIndex uint) (*models.FabricTransaction, error) {
	transaction, err := s.repo.GetTransactionByTxID(ctx, txID)
	if err != nil {
		return nil, err
	}

	// Mark as confirmed
	transaction.MarkAsConfirmed(blockNumber, blockHash, txIndex)

	if err := s.repo.UpdateTransaction(ctx, transaction); err != nil {
		s.monitoring.RecordBusinessEvent("fabric_transaction_confirm_error", fmt.Sprintf("%d", transaction.ClubID))
		return nil, err
	}

	s.monitoring.RecordBusinessEvent("fabric_transaction_confirmed", fmt.Sprintf("%d", transaction.ClubID))

	// Publish transaction confirmed event
	s.publishTransactionEvent(ctx, "fabric.transaction.confirmed", transaction)

	s.logger.Info("Fabric transaction confirmed", map[string]interface{}{
		"transaction_id": transaction.ID,
		"club_id":        transaction.ClubID,
		"tx_id":          transaction.TxID,
		"block_number":   transaction.BlockNumber,
		"block_hash":     transaction.BlockHash,
		"tx_index":       transaction.TxIndex,
	})

	return transaction, nil
}

// FailTransaction marks a transaction as failed
func (s *BlockchainService) FailTransaction(ctx context.Context, txID string, errorMessage string) (*models.FabricTransaction, error) {
	transaction, err := s.repo.GetTransactionByTxID(ctx, txID)
	if err != nil {
		return nil, err
	}

	// Mark as failed
	transaction.MarkAsFailed(errorMessage)

	if err := s.repo.UpdateTransaction(ctx, transaction); err != nil {
		s.monitoring.RecordBusinessEvent("fabric_transaction_fail_error", fmt.Sprintf("%d", transaction.ClubID))
		return nil, err
	}

	s.monitoring.RecordBusinessEvent("fabric_transaction_failed", fmt.Sprintf("%d", transaction.ClubID))

	// Publish transaction failed event
	s.publishTransactionEvent(ctx, "fabric.transaction.failed", transaction)

	s.logger.Warn("Fabric transaction failed", map[string]interface{}{
		"transaction_id": transaction.ID,
		"club_id":        transaction.ClubID,
		"tx_id":          transaction.TxID,
		"error":          errorMessage,
	})

	return transaction, nil
}

// Channel operations

// CreateChannel creates a new Fabric channel
func (s *BlockchainService) CreateChannel(ctx context.Context, req *CreateChannelRequest) (*models.Channel, error) {
	channel := &models.Channel{
		ClubID:          req.ClubID,
		ChannelID:       req.ChannelID,
		Name:            req.Name,
		Description:     req.Description,
		Type:            req.Type,
		Organizations:   req.Organizations,
		BatchSize:       req.BatchSize,
		BatchTimeout:    req.BatchTimeout,
		MaxMessageCount: req.MaxMessageCount,
		IsActive:        true,
	}

	if err := s.repo.CreateChannel(ctx, channel); err != nil {
		s.monitoring.RecordBusinessEvent("fabric_channel_create_error", fmt.Sprintf("%d", req.ClubID))
		return nil, err
	}

	s.monitoring.RecordBusinessEvent("fabric_channel_created", fmt.Sprintf("%d", req.ClubID))

	// Publish channel created event
	s.publishChannelEvent(ctx, "fabric.channel.created", channel)

	s.logger.Info("Fabric channel created", map[string]interface{}{
		"channel_id":     channel.ChannelID,
		"club_id":        channel.ClubID,
		"name":           channel.Name,
		"type":           channel.Type,
		"organizations":  channel.Organizations,
	})

	return channel, nil
}

// GetChannelByID retrieves a channel by database ID
func (s *BlockchainService) GetChannelByID(ctx context.Context, id uint) (*models.Channel, error) {
	return s.repo.GetChannelByID(ctx, id)
}

// GetChannelByChannelID retrieves a channel by Fabric channel ID
func (s *BlockchainService) GetChannelByChannelID(ctx context.Context, channelID string) (*models.Channel, error) {
	return s.repo.GetChannelByChannelID(ctx, channelID)
}

// GetChannelsByClubID retrieves channels for a club
func (s *BlockchainService) GetChannelsByClubID(ctx context.Context, clubID uint) ([]*models.Channel, error) {
	return s.repo.GetChannelsByClubID(ctx, clubID)
}

// Chaincode operations

// CreateChaincode creates a new chaincode record
func (s *BlockchainService) CreateChaincode(ctx context.Context, req *CreateChaincodeRequest) (*models.Chaincode, error) {
	chaincode := &models.Chaincode{
		ClubID:            req.ClubID,
		ChannelID:         req.ChannelID,
		Name:              req.Name,
		Version:           req.Version,
		Language:          req.Language,
		PackageID:         req.PackageID,
		PackageLabel:      req.PackageLabel,
		PackagePath:       req.PackagePath,
		EndorsementPolicy: req.EndorsementPolicy,
		Description:       req.Description,
		Sequence:          req.Sequence,
		IsInstalled:       false,
		IsCommitted:       false,
	}

	if err := s.repo.CreateChaincode(ctx, chaincode); err != nil {
		s.monitoring.RecordBusinessEvent("fabric_chaincode_create_error", fmt.Sprintf("%d", req.ClubID))
		return nil, err
	}

	s.monitoring.RecordBusinessEvent("fabric_chaincode_created", fmt.Sprintf("%d", req.ClubID))

	s.logger.Info("Fabric chaincode created", map[string]interface{}{
		"chaincode_id":   chaincode.ID,
		"club_id":        chaincode.ClubID,
		"channel_id":     chaincode.ChannelID,
		"name":           chaincode.Name,
		"version":        chaincode.Version,
		"language":       chaincode.Language,
	})

	return chaincode, nil
}

// GetChaincodeByID retrieves a chaincode by ID
func (s *BlockchainService) GetChaincodeByID(ctx context.Context, id uint) (*models.Chaincode, error) {
	return s.repo.GetChaincodeByID(ctx, id)
}

// GetChaincodesByChannelID retrieves chaincodes for a channel
func (s *BlockchainService) GetChaincodesByChannelID(ctx context.Context, channelID string) ([]*models.Chaincode, error) {
	return s.repo.GetChaincodesByChannelID(ctx, channelID)
}

// MarkChaincodeInstalled marks a chaincode as installed
func (s *BlockchainService) MarkChaincodeInstalled(ctx context.Context, id uint) error {
	chaincode, err := s.repo.GetChaincodeByID(ctx, id)
	if err != nil {
		return err
	}

	chaincode.IsInstalled = true
	return s.repo.UpdateChaincode(ctx, chaincode)
}

// MarkChaincodeCommitted marks a chaincode as committed
func (s *BlockchainService) MarkChaincodeCommitted(ctx context.Context, id uint) error {
	chaincode, err := s.repo.GetChaincodeByID(ctx, id)
	if err != nil {
		return err
	}

	chaincode.IsCommitted = true
	return s.repo.UpdateChaincode(ctx, chaincode)
}

// Block operations

// CreateBlock creates a new block record
func (s *BlockchainService) CreateBlock(ctx context.Context, req *CreateBlockRequest) (*models.Block, error) {
	block := &models.Block{
		ClubID:           req.ClubID,
		ChannelID:        req.ChannelID,
		BlockNumber:      req.BlockNumber,
		BlockHash:        req.BlockHash,
		PreviousHash:     req.PreviousHash,
		DataHash:         req.DataHash,
		TransactionCount: req.TransactionCount,
		Timestamp:        req.Timestamp,
		CreatedBy:        req.CreatedBy,
	}

	if err := s.repo.CreateBlock(ctx, block); err != nil {
		s.monitoring.RecordBusinessEvent("fabric_block_create_error", fmt.Sprintf("%d", req.ClubID))
		return nil, err
	}

	s.monitoring.RecordBusinessEvent("fabric_block_created", fmt.Sprintf("%d", req.ClubID))

	s.logger.Info("Fabric block created", map[string]interface{}{
		"block_id":           block.ID,
		"club_id":            block.ClubID,
		"channel_id":         block.ChannelID,
		"block_number":       block.BlockNumber,
		"transaction_count":  block.TransactionCount,
	})

	return block, nil
}

// GetBlockByHash retrieves a block by hash
func (s *BlockchainService) GetBlockByHash(ctx context.Context, blockHash string) (*models.Block, error) {
	return s.repo.GetBlockByHash(ctx, blockHash)
}

// GetBlocksByChannelID retrieves blocks for a channel
func (s *BlockchainService) GetBlocksByChannelID(ctx context.Context, channelID string, limit, offset int) ([]*models.Block, error) {
	return s.repo.GetBlocksByChannelID(ctx, channelID, limit, offset)
}

// Event operations

// CreateEvent creates a new chaincode event record
func (s *BlockchainService) CreateEvent(ctx context.Context, req *CreateEventRequest) (*models.Event, error) {
	event := &models.Event{
		ClubID:        req.ClubID,
		ChannelID:     req.ChannelID,
		ChaincodeName: req.ChaincodeName,
		EventName:     req.EventName,
		TxID:          req.TxID,
		BlockNumber:   req.BlockNumber,
		Payload:       req.Payload,
		EventTime:     req.EventTime,
		IsProcessed:   false,
	}

	if err := s.repo.CreateEvent(ctx, event); err != nil {
		s.monitoring.RecordBusinessEvent("fabric_event_create_error", fmt.Sprintf("%d", req.ClubID))
		return nil, err
	}

	s.monitoring.RecordBusinessEvent("fabric_event_created", fmt.Sprintf("%d", req.ClubID))

	s.logger.Info("Fabric event created", map[string]interface{}{
		"event_id":       event.ID,
		"club_id":        event.ClubID,
		"channel_id":     event.ChannelID,
		"chaincode_name": event.ChaincodeName,
		"event_name":     event.EventName,
		"tx_id":          event.TxID,
	})

	return event, nil
}

// GetUnprocessedEvents retrieves unprocessed events
func (s *BlockchainService) GetUnprocessedEvents(ctx context.Context, limit int) ([]*models.Event, error) {
	return s.repo.GetUnprocessedEvents(ctx, limit)
}

// MarkEventProcessed marks an event as processed
func (s *BlockchainService) MarkEventProcessed(ctx context.Context, id uint) error {
	event, err := s.repo.GetEventByID(ctx, id)
	if err != nil {
		return err
	}

	now := time.Now()
	event.IsProcessed = true
	event.ProcessedAt = &now

	return s.repo.UpdateEvent(ctx, event)
}

// Health check and utility operations

// HealthCheck performs a health check of the blockchain service
func (s *BlockchainService) HealthCheck(ctx context.Context) error {
	// Check database connectivity
	if err := s.repo.HealthCheck(ctx); err != nil {
		return fmt.Errorf("database health check failed: %v", err)
	}

	s.logger.Debug("Blockchain service health check passed", nil)
	return nil
}

// GetServiceStats returns service statistics
func (s *BlockchainService) GetServiceStats(ctx context.Context, clubID uint) (*ServiceStats, error) {
	stats := &ServiceStats{
		ClubID: clubID,
	}

	// Get transaction counts by status
	pendingCount, err := s.repo.GetTransactionCountByStatus(ctx, clubID, models.FabricTransactionStatusPending)
	if err != nil {
		return nil, err
	}
	stats.PendingTransactions = pendingCount

	submittedCount, err := s.repo.GetTransactionCountByStatus(ctx, clubID, models.FabricTransactionStatusSubmitted)
	if err != nil {
		return nil, err
	}
	stats.SubmittedTransactions = submittedCount

	confirmedCount, err := s.repo.GetTransactionCountByStatus(ctx, clubID, models.FabricTransactionStatusConfirmed)
	if err != nil {
		return nil, err
	}
	stats.ConfirmedTransactions = confirmedCount

	failedCount, err := s.repo.GetTransactionCountByStatus(ctx, clubID, models.FabricTransactionStatusFailed)
	if err != nil {
		return nil, err
	}
	stats.FailedTransactions = failedCount

	// Get channel count
	channels, err := s.repo.GetChannelsByClubID(ctx, clubID)
	if err != nil {
		return nil, err
	}
	stats.ActiveChannels = uint(len(channels))

	return stats, nil
}

// Event publishing helpers

func (s *BlockchainService) publishTransactionEvent(ctx context.Context, eventType string, transaction *models.FabricTransaction) {
	eventData := map[string]interface{}{
		"transaction_id":   transaction.ID,
		"club_id":          transaction.ClubID,
		"user_id":          transaction.UserID,
		"type":             transaction.Type,
		"status":           transaction.Status,
		"channel_id":       transaction.ChannelID,
		"chaincode_name":   transaction.ChaincodeName,
		"function":         transaction.Function,
		"tx_id":            transaction.TxID,
		"block_number":     transaction.BlockNumber,
		"timestamp":        time.Now(),
	}

	if err := s.messaging.Publish(ctx, eventType, eventData); err != nil {
		s.logger.Error("Failed to publish transaction event", map[string]interface{}{
			"event_type":     eventType,
			"transaction_id": transaction.ID,
			"error":          err.Error(),
		})
	}
}

func (s *BlockchainService) publishChannelEvent(ctx context.Context, eventType string, channel *models.Channel) {
	eventData := map[string]interface{}{
		"channel_id":     channel.ChannelID,
		"club_id":        channel.ClubID,
		"name":           channel.Name,
		"type":           channel.Type,
		"organizations":  channel.Organizations,
		"is_active":      channel.IsActive,
		"timestamp":      time.Now(),
	}

	if err := s.messaging.Publish(ctx, eventType, eventData); err != nil {
		s.logger.Error("Failed to publish channel event", map[string]interface{}{
			"event_type": eventType,
			"channel_id": channel.ChannelID,
			"error":      err.Error(),
		})
	}
}

// Request and response types for Fabric operations

type CreateTransactionRequest struct {
	ClubID         uint                                   `json:"club_id" validate:"required"`
	UserID         string                                 `json:"user_id" validate:"required"`
	Type           models.FabricTransactionType           `json:"type" validate:"required"`
	ChannelID      string                                 `json:"channel_id" validate:"required"`
	ChaincodeName  string                                 `json:"chaincode_name" validate:"required"`
	Function       string                                 `json:"function" validate:"required"`
	Args           []string                               `json:"args"`
	TransientMap   map[string][]byte                      `json:"transient_map,omitempty"`
	Metadata       map[string]interface{}                 `json:"metadata,omitempty"`
	ClientIdentity string                                 `json:"client_identity,omitempty"`
}

type CreateChannelRequest struct {
	ClubID          uint                   `json:"club_id" validate:"required"`
	ChannelID       string                 `json:"channel_id" validate:"required"`
	Name            string                 `json:"name" validate:"required"`
	Description     string                 `json:"description"`
	Type            models.ChannelType     `json:"type" validate:"required"`
	Organizations   []string               `json:"organizations" validate:"required"`
	BatchSize       uint                   `json:"batch_size"`
	BatchTimeout    string                 `json:"batch_timeout"`
	MaxMessageCount uint                   `json:"max_message_count"`
}

type CreateChaincodeRequest struct {
	ClubID            uint   `json:"club_id" validate:"required"`
	ChannelID         string `json:"channel_id" validate:"required"`
	Name              string `json:"name" validate:"required"`
	Version           string `json:"version" validate:"required"`
	Language          string `json:"language" validate:"required"`
	PackageID         string `json:"package_id"`
	PackageLabel      string `json:"package_label"`
	PackagePath       string `json:"package_path"`
	EndorsementPolicy string `json:"endorsement_policy"`
	Description       string `json:"description"`
	Sequence          uint64 `json:"sequence"`
}

type CreateBlockRequest struct {
	ClubID           uint      `json:"club_id" validate:"required"`
	ChannelID        string    `json:"channel_id" validate:"required"`
	BlockNumber      uint64    `json:"block_number" validate:"required"`
	BlockHash        string    `json:"block_hash" validate:"required"`
	PreviousHash     string    `json:"previous_hash" validate:"required"`
	DataHash         string    `json:"data_hash" validate:"required"`
	TransactionCount uint      `json:"transaction_count"`
	Timestamp        time.Time `json:"timestamp"`
	CreatedBy        string    `json:"created_by"`
}

type CreateEventRequest struct {
	ClubID        uint      `json:"club_id" validate:"required"`
	ChannelID     string    `json:"channel_id" validate:"required"`
	ChaincodeName string    `json:"chaincode_name" validate:"required"`
	EventName     string    `json:"event_name" validate:"required"`
	TxID          string    `json:"tx_id" validate:"required"`
	BlockNumber   uint64    `json:"block_number" validate:"required"`
	Payload       string    `json:"payload"`
	EventTime     time.Time `json:"event_time"`
}

type ServiceStats struct {
	ClubID                 uint `json:"club_id"`
	PendingTransactions    uint `json:"pending_transactions"`
	SubmittedTransactions  uint `json:"submitted_transactions"`
	ConfirmedTransactions  uint `json:"confirmed_transactions"`
	FailedTransactions     uint `json:"failed_transactions"`
	ActiveChannels         uint `json:"active_channels"`
}