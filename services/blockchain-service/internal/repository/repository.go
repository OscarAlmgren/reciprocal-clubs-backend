package repository

import (
	"context"

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

// Fabric Transaction operations

// CreateTransaction creates a new Fabric transaction record
func (r *Repository) CreateTransaction(ctx context.Context, transaction *models.FabricTransaction) error {
	if err := r.db.WithContext(ctx).Create(transaction).Error; err != nil {
		r.logger.Error("Failed to create Fabric transaction", map[string]interface{}{
			"error":          err.Error(),
			"club_id":        transaction.ClubID,
			"user_id":        transaction.UserID,
			"channel_id":     transaction.ChannelID,
			"chaincode_name": transaction.ChaincodeName,
			"function":       transaction.Function,
			"type":           transaction.Type,
		})
		return err
	}

	r.logger.Info("Fabric transaction created successfully", map[string]interface{}{
		"transaction_id":   transaction.ID,
		"club_id":          transaction.ClubID,
		"channel_id":       transaction.ChannelID,
		"chaincode_name":   transaction.ChaincodeName,
		"function":         transaction.Function,
		"type":             transaction.Type,
		"status":           transaction.Status,
	})

	return nil
}

// UpdateTransaction updates a Fabric transaction
func (r *Repository) UpdateTransaction(ctx context.Context, transaction *models.FabricTransaction) error {
	if err := r.db.WithContext(ctx).Save(transaction).Error; err != nil {
		r.logger.Error("Failed to update Fabric transaction", map[string]interface{}{
			"error":          err.Error(),
			"transaction_id": transaction.ID,
		})
		return err
	}

	return nil
}

// GetTransactionByID retrieves a Fabric transaction by ID
func (r *Repository) GetTransactionByID(ctx context.Context, id uint) (*models.FabricTransaction, error) {
	var transaction models.FabricTransaction
	if err := r.db.WithContext(ctx).First(&transaction, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		r.logger.Error("Failed to get Fabric transaction", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		return nil, err
	}

	return &transaction, nil
}

// GetTransactionByTxID retrieves a Fabric transaction by transaction ID
func (r *Repository) GetTransactionByTxID(ctx context.Context, txID string) (*models.FabricTransaction, error) {
	var transaction models.FabricTransaction
	if err := r.db.WithContext(ctx).Where("tx_id = ?", txID).First(&transaction).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		r.logger.Error("Failed to get Fabric transaction by tx_id", map[string]interface{}{
			"error": err.Error(),
			"tx_id": txID,
		})
		return nil, err
	}

	return &transaction, nil
}

// GetTransactionsByClubID retrieves Fabric transactions for a specific club
func (r *Repository) GetTransactionsByClubID(ctx context.Context, clubID uint, limit, offset int) ([]*models.FabricTransaction, error) {
	var transactions []*models.FabricTransaction
	query := r.db.WithContext(ctx).Where("club_id = ?", clubID)

	if limit > 0 {
		query = query.Limit(limit)
	}

	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Order("created_at DESC").Find(&transactions).Error; err != nil {
		r.logger.Error("Failed to get Fabric transactions by club", map[string]interface{}{
			"error":   err.Error(),
			"club_id": clubID,
		})
		return nil, err
	}

	return transactions, nil
}

// GetTransactionCountByStatus retrieves transaction count by status for a club
func (r *Repository) GetTransactionCountByStatus(ctx context.Context, clubID uint, status models.FabricTransactionStatus) (uint, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.FabricTransaction{}).
		Where("club_id = ? AND status = ?", clubID, status).
		Count(&count).Error; err != nil {
		r.logger.Error("Failed to get Fabric transaction count by status", map[string]interface{}{
			"error":   err.Error(),
			"club_id": clubID,
			"status":  status,
		})
		return 0, err
	}

	return uint(count), nil
}

// Channel operations

// CreateChannel creates a new Fabric channel record
func (r *Repository) CreateChannel(ctx context.Context, channel *models.Channel) error {
	if err := r.db.WithContext(ctx).Create(channel).Error; err != nil {
		r.logger.Error("Failed to create Fabric channel", map[string]interface{}{
			"error":      err.Error(),
			"club_id":    channel.ClubID,
			"channel_id": channel.ChannelID,
			"name":       channel.Name,
			"type":       channel.Type,
		})
		return err
	}

	r.logger.Info("Fabric channel created successfully", map[string]interface{}{
		"channel_id":     channel.ChannelID,
		"club_id":        channel.ClubID,
		"name":           channel.Name,
		"type":           channel.Type,
		"organizations":  channel.Organizations,
	})

	return nil
}

// GetChannelByID retrieves a channel by database ID
func (r *Repository) GetChannelByID(ctx context.Context, id uint) (*models.Channel, error) {
	var channel models.Channel
	if err := r.db.WithContext(ctx).First(&channel, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		r.logger.Error("Failed to get Fabric channel", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		return nil, err
	}

	return &channel, nil
}

// GetChannelByChannelID retrieves a channel by Fabric channel ID
func (r *Repository) GetChannelByChannelID(ctx context.Context, channelID string) (*models.Channel, error) {
	var channel models.Channel
	if err := r.db.WithContext(ctx).Where("channel_id = ?", channelID).First(&channel).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		r.logger.Error("Failed to get Fabric channel by channel_id", map[string]interface{}{
			"error":      err.Error(),
			"channel_id": channelID,
		})
		return nil, err
	}

	return &channel, nil
}

// GetChannelsByClubID retrieves channels for a specific club
func (r *Repository) GetChannelsByClubID(ctx context.Context, clubID uint) ([]*models.Channel, error) {
	var channels []*models.Channel
	if err := r.db.WithContext(ctx).Where("club_id = ?", clubID).Find(&channels).Error; err != nil {
		r.logger.Error("Failed to get Fabric channels by club", map[string]interface{}{
			"error":   err.Error(),
			"club_id": clubID,
		})
		return nil, err
	}

	return channels, nil
}

// Chaincode operations

// CreateChaincode creates a new chaincode record
func (r *Repository) CreateChaincode(ctx context.Context, chaincode *models.Chaincode) error {
	if err := r.db.WithContext(ctx).Create(chaincode).Error; err != nil {
		r.logger.Error("Failed to create chaincode", map[string]interface{}{
			"error":      err.Error(),
			"club_id":    chaincode.ClubID,
			"channel_id": chaincode.ChannelID,
			"name":       chaincode.Name,
			"version":    chaincode.Version,
		})
		return err
	}

	r.logger.Info("Chaincode created successfully", map[string]interface{}{
		"chaincode_id": chaincode.ID,
		"club_id":      chaincode.ClubID,
		"channel_id":   chaincode.ChannelID,
		"name":         chaincode.Name,
		"version":      chaincode.Version,
		"language":     chaincode.Language,
	})

	return nil
}

// UpdateChaincode updates a chaincode
func (r *Repository) UpdateChaincode(ctx context.Context, chaincode *models.Chaincode) error {
	if err := r.db.WithContext(ctx).Save(chaincode).Error; err != nil {
		r.logger.Error("Failed to update chaincode", map[string]interface{}{
			"error":        err.Error(),
			"chaincode_id": chaincode.ID,
		})
		return err
	}

	return nil
}

// GetChaincodeByID retrieves a chaincode by ID
func (r *Repository) GetChaincodeByID(ctx context.Context, id uint) (*models.Chaincode, error) {
	var chaincode models.Chaincode
	if err := r.db.WithContext(ctx).First(&chaincode, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		r.logger.Error("Failed to get chaincode", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		return nil, err
	}

	return &chaincode, nil
}

// GetChaincodesByChannelID retrieves chaincodes for a specific channel
func (r *Repository) GetChaincodesByChannelID(ctx context.Context, channelID string) ([]*models.Chaincode, error) {
	var chaincodes []*models.Chaincode
	if err := r.db.WithContext(ctx).Where("channel_id = ?", channelID).Find(&chaincodes).Error; err != nil {
		r.logger.Error("Failed to get chaincodes by channel", map[string]interface{}{
			"error":      err.Error(),
			"channel_id": channelID,
		})
		return nil, err
	}

	return chaincodes, nil
}

// Block operations

// CreateBlock creates a new block record
func (r *Repository) CreateBlock(ctx context.Context, block *models.Block) error {
	if err := r.db.WithContext(ctx).Create(block).Error; err != nil {
		r.logger.Error("Failed to create Fabric block", map[string]interface{}{
			"error":        err.Error(),
			"club_id":      block.ClubID,
			"channel_id":   block.ChannelID,
			"block_number": block.BlockNumber,
			"block_hash":   block.BlockHash,
		})
		return err
	}

	r.logger.Info("Fabric block created successfully", map[string]interface{}{
		"block_id":           block.ID,
		"club_id":            block.ClubID,
		"channel_id":         block.ChannelID,
		"block_number":       block.BlockNumber,
		"transaction_count":  block.TransactionCount,
	})

	return nil
}

// GetBlockByHash retrieves a block by hash
func (r *Repository) GetBlockByHash(ctx context.Context, blockHash string) (*models.Block, error) {
	var block models.Block
	if err := r.db.WithContext(ctx).Where("block_hash = ?", blockHash).First(&block).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		r.logger.Error("Failed to get Fabric block by hash", map[string]interface{}{
			"error":      err.Error(),
			"block_hash": blockHash,
		})
		return nil, err
	}

	return &block, nil
}

// GetBlocksByChannelID retrieves blocks for a specific channel
func (r *Repository) GetBlocksByChannelID(ctx context.Context, channelID string, limit, offset int) ([]*models.Block, error) {
	var blocks []*models.Block
	query := r.db.WithContext(ctx).Where("channel_id = ?", channelID)

	if limit > 0 {
		query = query.Limit(limit)
	}

	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Order("block_number DESC").Find(&blocks).Error; err != nil {
		r.logger.Error("Failed to get Fabric blocks by channel", map[string]interface{}{
			"error":      err.Error(),
			"channel_id": channelID,
		})
		return nil, err
	}

	return blocks, nil
}

// Event operations

// CreateEvent creates a new chaincode event record
func (r *Repository) CreateEvent(ctx context.Context, event *models.Event) error {
	if err := r.db.WithContext(ctx).Create(event).Error; err != nil {
		r.logger.Error("Failed to create Fabric event", map[string]interface{}{
			"error":          err.Error(),
			"club_id":        event.ClubID,
			"channel_id":     event.ChannelID,
			"chaincode_name": event.ChaincodeName,
			"event_name":     event.EventName,
			"tx_id":          event.TxID,
		})
		return err
	}

	r.logger.Info("Fabric event created successfully", map[string]interface{}{
		"event_id":       event.ID,
		"club_id":        event.ClubID,
		"channel_id":     event.ChannelID,
		"chaincode_name": event.ChaincodeName,
		"event_name":     event.EventName,
		"tx_id":          event.TxID,
	})

	return nil
}

// UpdateEvent updates an event
func (r *Repository) UpdateEvent(ctx context.Context, event *models.Event) error {
	if err := r.db.WithContext(ctx).Save(event).Error; err != nil {
		r.logger.Error("Failed to update Fabric event", map[string]interface{}{
			"error":    err.Error(),
			"event_id": event.ID,
		})
		return err
	}

	return nil
}

// GetEventByID retrieves an event by ID
func (r *Repository) GetEventByID(ctx context.Context, id uint) (*models.Event, error) {
	var event models.Event
	if err := r.db.WithContext(ctx).First(&event, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		r.logger.Error("Failed to get Fabric event", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		return nil, err
	}

	return &event, nil
}

// GetUnprocessedEvents retrieves unprocessed events
func (r *Repository) GetUnprocessedEvents(ctx context.Context, limit int) ([]*models.Event, error) {
	var events []*models.Event
	query := r.db.WithContext(ctx).Where("is_processed = ?", false)

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Order("event_time ASC").Find(&events).Error; err != nil {
		r.logger.Error("Failed to get unprocessed Fabric events", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	return events, nil
}

// Health check

// HealthCheck performs a health check on the repository
func (r *Repository) HealthCheck(ctx context.Context) error {
	var result int
	if err := r.db.WithContext(ctx).Raw("SELECT 1").Scan(&result).Error; err != nil {
		r.logger.Error("Repository health check failed", map[string]interface{}{
			"error": err.Error(),
		})
		return err
	}

	return nil
}