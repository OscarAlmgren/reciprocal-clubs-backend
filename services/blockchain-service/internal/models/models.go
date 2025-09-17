package models

import (
	"encoding/json"
	"math/big"
	"time"
	"gorm.io/gorm"
)

// TransactionStatus represents the status of a blockchain transaction
type TransactionStatus string

const (
	TransactionStatusPending   TransactionStatus = "pending"
	TransactionStatusSubmitted TransactionStatus = "submitted"
	TransactionStatusConfirmed TransactionStatus = "confirmed"
	TransactionStatusFailed    TransactionStatus = "failed"
	TransactionStatusExpired   TransactionStatus = "expired"
)

// TransactionType represents the type of blockchain transaction
type TransactionType string

const (
	TransactionTypeTransfer          TransactionType = "transfer"
	TransactionTypeMint              TransactionType = "mint"
	TransactionTypeBurn              TransactionType = "burn"
	TransactionTypeStake             TransactionType = "stake"
	TransactionTypeUnstake           TransactionType = "unstake"
	TransactionTypeClaimRewards      TransactionType = "claim_rewards"
	TransactionTypeCreateAgreement   TransactionType = "create_agreement"
	TransactionTypeActivateAgreement TransactionType = "activate_agreement"
	TransactionTypeRecordVisit       TransactionType = "record_visit"
)

// Network represents different blockchain networks
type Network string

const (
	NetworkEthereum  Network = "ethereum"
	NetworkPolygon   Network = "polygon"
	NetworkBSC       Network = "bsc"
	NetworkArbitrum  Network = "arbitrum"
	NetworkOptimism  Network = "optimism"
	NetworkLocalhost Network = "localhost"
)

// Transaction represents a blockchain transaction
type Transaction struct {
	ID                uint              `json:"id" gorm:"primaryKey"`
	ClubID            uint              `json:"club_id" gorm:"not null;index"`
	UserID            string            `json:"user_id" gorm:"size:255;index"`
	Network           Network           `json:"network" gorm:"size:50;not null"`
	Type              TransactionType   `json:"type" gorm:"size:100;not null"`
	Status            TransactionStatus `json:"status" gorm:"size:50;default:'pending'"`
	Hash              string            `json:"hash,omitempty" gorm:"size:255;uniqueIndex"`
	FromAddress       string            `json:"from_address" gorm:"size:255"`
	ToAddress         string            `json:"to_address" gorm:"size:255"`
	Value             string            `json:"value" gorm:"type:text"` // Store as string to handle big numbers
	GasLimit          uint64            `json:"gas_limit"`
	GasPrice          string            `json:"gas_price" gorm:"type:text"`
	GasUsed           uint64            `json:"gas_used,omitempty"`
	Nonce             uint64            `json:"nonce"`
	Data              string            `json:"data,omitempty" gorm:"type:text"`
	BlockNumber       uint64            `json:"block_number,omitempty"`
	BlockHash         string            `json:"block_hash,omitempty" gorm:"size:255"`
	TransactionIndex  uint              `json:"transaction_index,omitempty"`
	ConfirmationCount uint              `json:"confirmation_count" gorm:"default:0"`
	ErrorMessage      string            `json:"error_message,omitempty" gorm:"type:text"`
	Metadata          string            `json:"metadata,omitempty" gorm:"type:json"`
	SubmittedAt       *time.Time        `json:"submitted_at,omitempty"`
	ConfirmedAt       *time.Time        `json:"confirmed_at,omitempty"`
	FailedAt          *time.Time        `json:"failed_at,omitempty"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
	DeletedAt         gorm.DeletedAt    `json:"-" gorm:"index"`
}

func (Transaction) TableName() string {
	return "blockchain_transactions"
}

// Contract represents a smart contract
type Contract struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	ClubID      uint           `json:"club_id" gorm:"not null;index"`
	Network     Network        `json:"network" gorm:"size:50;not null"`
	Name        string         `json:"name" gorm:"size:255;not null"`
	Address     string         `json:"address" gorm:"size:255;not null"`
	ABI         string         `json:"abi" gorm:"type:text"`
	Bytecode    string         `json:"bytecode,omitempty" gorm:"type:text"`
	Version     string         `json:"version" gorm:"size:50"`
	IsDeployed  bool           `json:"is_deployed" gorm:"default:false"`
	DeployedAt  *time.Time     `json:"deployed_at,omitempty"`
	DeployedBy  string         `json:"deployed_by,omitempty" gorm:"size:255"`
	Description string         `json:"description,omitempty" gorm:"type:text"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Contract) TableName() string {
	return "blockchain_contracts"
}

// Wallet represents a user wallet
type Wallet struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	ClubID          uint           `json:"club_id" gorm:"not null;index"`
	UserID          string         `json:"user_id" gorm:"size:255;not null;index"`
	Network         Network        `json:"network" gorm:"size:50;not null"`
	Address         string         `json:"address" gorm:"size:255;not null;index"`
	IsActive        bool           `json:"is_active" gorm:"default:true"`
	Balance         string         `json:"balance" gorm:"type:text;default:'0'"` // Native token balance
	TokenBalances   string         `json:"token_balances,omitempty" gorm:"type:json"` // ERC20 token balances
	LastSyncedAt    *time.Time     `json:"last_synced_at,omitempty"`
	LastActivity    *time.Time     `json:"last_activity,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Wallet) TableName() string {
	return "blockchain_wallets"
}

// Token represents an ERC20 or similar token
type Token struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	ClubID       uint           `json:"club_id" gorm:"not null;index"`
	Network      Network        `json:"network" gorm:"size:50;not null"`
	ContractID   uint           `json:"contract_id" gorm:"not null"`
	Contract     Contract       `json:"contract" gorm:"foreignKey:ContractID"`
	Name         string         `json:"name" gorm:"size:255;not null"`
	Symbol       string         `json:"symbol" gorm:"size:50;not null"`
	Decimals     uint8          `json:"decimals" gorm:"not null"`
	TotalSupply  string         `json:"total_supply" gorm:"type:text"`
	MaxSupply    string         `json:"max_supply,omitempty" gorm:"type:text"`
	IsMintable   bool           `json:"is_mintable" gorm:"default:false"`
	IsBurnable   bool           `json:"is_burnable" gorm:"default:false"`
	IsPausable   bool           `json:"is_pausable" gorm:"default:false"`
	IsActive     bool           `json:"is_active" gorm:"default:true"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Token) TableName() string {
	return "blockchain_tokens"
}

// Block represents a blockchain block for tracking
type Block struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	Network         Network        `json:"network" gorm:"size:50;not null;index"`
	Number          uint64         `json:"number" gorm:"not null;index"`
	Hash            string         `json:"hash" gorm:"size:255;not null;uniqueIndex"`
	ParentHash      string         `json:"parent_hash" gorm:"size:255"`
	TransactionCount uint           `json:"transaction_count"`
	Timestamp       time.Time      `json:"timestamp"`
	GasLimit        uint64         `json:"gas_limit"`
	GasUsed         uint64         `json:"gas_used"`
	Difficulty      string         `json:"difficulty,omitempty" gorm:"type:text"`
	Miner           string         `json:"miner,omitempty" gorm:"size:255"`
	ProcessedAt     *time.Time     `json:"processed_at,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Block) TableName() string {
	return "blockchain_blocks"
}

// Event represents a blockchain event log
type Event struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	ClubID          uint           `json:"club_id" gorm:"not null;index"`
	Network         Network        `json:"network" gorm:"size:50;not null"`
	ContractAddress string         `json:"contract_address" gorm:"size:255;not null;index"`
	EventName       string         `json:"event_name" gorm:"size:255;not null"`
	TransactionHash string         `json:"transaction_hash" gorm:"size:255;not null;index"`
	BlockNumber     uint64         `json:"block_number" gorm:"not null"`
	LogIndex        uint           `json:"log_index" gorm:"not null"`
	Topics          string         `json:"topics" gorm:"type:json"`
	Data            string         `json:"data" gorm:"type:text"`
	DecodedData     string         `json:"decoded_data,omitempty" gorm:"type:json"`
	ProcessedAt     *time.Time     `json:"processed_at,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Event) TableName() string {
	return "blockchain_events"
}

// Helper methods

// GetValueAsBigInt converts the value string to big.Int
func (t *Transaction) GetValueAsBigInt() *big.Int {
	value := new(big.Int)
	value.SetString(t.Value, 10)
	return value
}

// SetValueFromBigInt sets the value from a big.Int
func (t *Transaction) SetValueFromBigInt(value *big.Int) {
	t.Value = value.String()
}

// GetGasPriceAsBigInt converts the gas price string to big.Int
func (t *Transaction) GetGasPriceAsBigInt() *big.Int {
	gasPrice := new(big.Int)
	gasPrice.SetString(t.GasPrice, 10)
	return gasPrice
}

// SetGasPriceFromBigInt sets the gas price from a big.Int
func (t *Transaction) SetGasPriceFromBigInt(gasPrice *big.Int) {
	t.GasPrice = gasPrice.String()
}

// IsConfirmed checks if the transaction is confirmed
func (t *Transaction) IsConfirmed() bool {
	return t.Status == TransactionStatusConfirmed
}

// IsPending checks if the transaction is pending
func (t *Transaction) IsPending() bool {
	return t.Status == TransactionStatusPending || t.Status == TransactionStatusSubmitted
}

// MarkAsSubmitted updates the transaction status to submitted
func (t *Transaction) MarkAsSubmitted(hash string) {
	t.Status = TransactionStatusSubmitted
	t.Hash = hash
	now := time.Now()
	t.SubmittedAt = &now
}

// MarkAsConfirmed updates the transaction status to confirmed
func (t *Transaction) MarkAsConfirmed(blockNumber uint64, blockHash string, gasUsed uint64) {
	t.Status = TransactionStatusConfirmed
	t.BlockNumber = blockNumber
	t.BlockHash = blockHash
	t.GasUsed = gasUsed
	now := time.Now()
	t.ConfirmedAt = &now
}

// MarkAsFailed updates the transaction status to failed
func (t *Transaction) MarkAsFailed(errorMsg string) {
	t.Status = TransactionStatusFailed
	t.ErrorMessage = errorMsg
	now := time.Now()
	t.FailedAt = &now
}

// GetMetadata parses the metadata JSON
func (t *Transaction) GetMetadata() (map[string]interface{}, error) {
	if t.Metadata == "" {
		return make(map[string]interface{}), nil
	}

	var metadata map[string]interface{}
	err := json.Unmarshal([]byte(t.Metadata), &metadata)
	return metadata, err
}

// SetMetadata sets the metadata as JSON
func (t *Transaction) SetMetadata(metadata map[string]interface{}) error {
	if metadata == nil {
		t.Metadata = ""
		return nil
	}

	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	t.Metadata = string(data)
	return nil
}

// GetTokenBalances parses the token balances JSON
func (w *Wallet) GetTokenBalances() (map[string]string, error) {
	if w.TokenBalances == "" {
		return make(map[string]string), nil
	}

	var balances map[string]string
	err := json.Unmarshal([]byte(w.TokenBalances), &balances)
	return balances, err
}

// SetTokenBalances sets the token balances as JSON
func (w *Wallet) SetTokenBalances(balances map[string]string) error {
	if balances == nil {
		w.TokenBalances = ""
		return nil
	}

	data, err := json.Marshal(balances)
	if err != nil {
		return err
	}
	w.TokenBalances = string(data)
	return nil
}

// UpdateBalance updates the wallet balance and last synced time
func (w *Wallet) UpdateBalance(balance string) {
	w.Balance = balance
	now := time.Now()
	w.LastSyncedAt = &now
}

// GetBalanceAsBigInt converts the balance string to big.Int
func (w *Wallet) GetBalanceAsBigInt() *big.Int {
	balance := new(big.Int)
	balance.SetString(w.Balance, 10)
	return balance
}

// GetTotalSupplyAsBigInt converts the total supply string to big.Int
func (token *Token) GetTotalSupplyAsBigInt() *big.Int {
	supply := new(big.Int)
	supply.SetString(token.TotalSupply, 10)
	return supply
}

// SetTotalSupplyFromBigInt sets the total supply from a big.Int
func (token *Token) SetTotalSupplyFromBigInt(supply *big.Int) {
	token.TotalSupply = supply.String()
}
