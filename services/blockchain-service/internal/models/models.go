package models

import (
	"encoding/json"
	"time"
	"gorm.io/gorm"
)

// FabricTransactionStatus represents the status of a Hyperledger Fabric transaction
type FabricTransactionStatus string

const (
	FabricTransactionStatusPending   FabricTransactionStatus = "pending"
	FabricTransactionStatusSubmitted FabricTransactionStatus = "submitted"
	FabricTransactionStatusConfirmed FabricTransactionStatus = "confirmed"
	FabricTransactionStatusFailed    FabricTransactionStatus = "failed"
	FabricTransactionStatusExpired   FabricTransactionStatus = "expired"
)

// FabricTransactionType represents the type of blockchain transaction
type FabricTransactionType string

const (
	FabricTransactionTypeMemberRegistration FabricTransactionType = "member_registration"
	FabricTransactionTypeVisitRecord        FabricTransactionType = "visit_record"
	FabricTransactionTypeAgreementCreation  FabricTransactionType = "agreement_creation"
	FabricTransactionTypePaymentRecord      FabricTransactionType = "payment_record"
	FabricTransactionTypeQuery              FabricTransactionType = "query"
	FabricTransactionTypeInvoke             FabricTransactionType = "invoke"
)

// ChannelType represents different Fabric channel types
type ChannelType string

const (
	ChannelTypeReciprocal ChannelType = "reciprocal"
	ChannelTypeGovernance ChannelType = "governance"
	ChannelTypeAudit      ChannelType = "audit"
	ChannelTypePayments   ChannelType = "payments"
)

// FabricTransaction represents a Hyperledger Fabric transaction
type FabricTransaction struct {
	ID       uint                    `json:"id" gorm:"primaryKey"`
	ClubID   uint                    `json:"club_id" gorm:"not null;index"`
	UserID   string                  `json:"user_id" gorm:"size:255;index"`
	Type     FabricTransactionType   `json:"type" gorm:"size:100;not null"`
	Status   FabricTransactionStatus `json:"status" gorm:"size:50;default:'pending'"`

	// Fabric-specific identifiers
	TxID       string `json:"tx_id" gorm:"size:255;uniqueIndex"`
	ChannelID  string `json:"channel_id" gorm:"size:255;not null;index"`
	ChaincodeName string `json:"chaincode_name" gorm:"size:255;not null"`
	Function   string `json:"function" gorm:"size:255;not null"`

	// Transaction parameters
	Args         []string `json:"args" gorm:"type:json"`
	TransientMap string   `json:"transient_map,omitempty" gorm:"type:text"` // JSON-encoded map

	// Execution results
	Response     string `json:"response,omitempty" gorm:"type:text"`
	ErrorMessage string `json:"error_message,omitempty" gorm:"type:text"`

	// Block information
	BlockNumber uint64 `json:"block_number,omitempty"`
	BlockHash   string `json:"block_hash,omitempty" gorm:"size:255"`
	TxIndex     uint   `json:"tx_index,omitempty"`

	// Endorsement information
	EndorsingPeers   []string `json:"endorsing_peers" gorm:"type:json"`
	EndorsementCount uint     `json:"endorsement_count" gorm:"default:0"`

	// Metadata and audit
	ClientIdentity string `json:"client_identity,omitempty" gorm:"size:255"`
	Metadata       string `json:"metadata,omitempty" gorm:"type:json"`

	// Timestamps
	SubmittedAt *time.Time `json:"submitted_at,omitempty"`
	ConfirmedAt *time.Time `json:"confirmed_at,omitempty"`
	FailedAt    *time.Time `json:"failed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (FabricTransaction) TableName() string {
	return "fabric_transactions"
}

// Channel represents a Hyperledger Fabric channel
type Channel struct {
	ID            uint        `json:"id" gorm:"primaryKey"`
	ClubID        uint        `json:"club_id" gorm:"not null;index"`
	ChannelID     string      `json:"channel_id" gorm:"size:255;not null;uniqueIndex"`
	Name          string      `json:"name" gorm:"size:255;not null"`
	Description   string      `json:"description,omitempty" gorm:"type:text"`
	Type          ChannelType `json:"type" gorm:"size:50;not null"`
	Organizations []string    `json:"organizations" gorm:"type:json"`

	// Channel configuration
	BatchSize     uint   `json:"batch_size" gorm:"default:10"`
	BatchTimeout  string `json:"batch_timeout" gorm:"size:50;default:'2s'"`
	MaxMessageCount uint `json:"max_message_count" gorm:"default:500"`

	// Status
	IsActive    bool       `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Channel) TableName() string {
	return "fabric_channels"
}

// Chaincode represents a deployed chaincode on Fabric
type Chaincode struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	ClubID      uint   `json:"club_id" gorm:"not null;index"`
	ChannelID   string `json:"channel_id" gorm:"size:255;not null;index"`
	Name        string `json:"name" gorm:"size:255;not null"`
	Version     string `json:"version" gorm:"size:50;not null"`
	Language    string `json:"language" gorm:"size:50;not null"` // go, javascript, java

	// Package information
	PackageID    string `json:"package_id" gorm:"size:255;uniqueIndex"`
	PackageLabel string `json:"package_label" gorm:"size:255"`
	PackagePath  string `json:"package_path,omitempty" gorm:"type:text"`

	// Deployment status
	IsInstalled bool   `json:"is_installed" gorm:"default:false"`
	IsCommitted bool   `json:"is_committed" gorm:"default:false"`
	Sequence    uint64 `json:"sequence" gorm:"default:1"`

	// Endorsement policy
	EndorsementPolicy string `json:"endorsement_policy,omitempty" gorm:"type:text"`

	// Metadata
	Description string    `json:"description,omitempty" gorm:"type:text"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Chaincode) TableName() string {
	return "fabric_chaincodes"
}

// Peer represents a Fabric peer node
type Peer struct {
	ID           uint   `json:"id" gorm:"primaryKey"`
	ClubID       uint   `json:"club_id" gorm:"not null;index"`
	Name         string `json:"name" gorm:"size:255;not null"`
	Organization string `json:"organization" gorm:"size:255;not null"`
	Endpoint     string `json:"endpoint" gorm:"size:255;not null"`

	// TLS Configuration
	TLSEnabled   bool   `json:"tls_enabled" gorm:"default:true"`
	TLSCertPath  string `json:"tls_cert_path,omitempty" gorm:"type:text"`
	TLSKeyPath   string `json:"tls_key_path,omitempty" gorm:"type:text"`
	TLSRootCert  string `json:"tls_root_cert,omitempty" gorm:"type:text"`

	// Status
	IsActive    bool      `json:"is_active" gorm:"default:true"`
	LastSeen    *time.Time `json:"last_seen,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Peer) TableName() string {
	return "fabric_peers"
}

// Block represents a Fabric block
type Block struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	ClubID      uint   `json:"club_id" gorm:"not null;index"`
	ChannelID   string `json:"channel_id" gorm:"size:255;not null;index"`
	BlockNumber uint64 `json:"block_number" gorm:"not null;index"`
	BlockHash   string `json:"block_hash" gorm:"size:255;not null;uniqueIndex"`

	// Block content
	PreviousHash    string `json:"previous_hash" gorm:"size:255;not null"`
	DataHash        string `json:"data_hash" gorm:"size:255;not null"`
	TransactionCount uint  `json:"transaction_count" gorm:"default:0"`

	// Block metadata
	Timestamp   time.Time `json:"timestamp"`
	CreatedBy   string    `json:"created_by" gorm:"size:255"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Block) TableName() string {
	return "fabric_blocks"
}

// Event represents a Fabric chaincode event
type Event struct {
	ID            uint   `json:"id" gorm:"primaryKey"`
	ClubID        uint   `json:"club_id" gorm:"not null;index"`
	ChannelID     string `json:"channel_id" gorm:"size:255;not null;index"`
	ChaincodeName string `json:"chaincode_name" gorm:"size:255;not null"`
	EventName     string `json:"event_name" gorm:"size:255;not null"`

	// Event details
	TxID        string `json:"tx_id" gorm:"size:255;not null;index"`
	BlockNumber uint64 `json:"block_number" gorm:"not null;index"`
	Payload     string `json:"payload,omitempty" gorm:"type:text"`

	// Processing status
	IsProcessed bool       `json:"is_processed" gorm:"default:false"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`

	// Timestamps
	EventTime time.Time `json:"event_time"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Event) TableName() string {
	return "fabric_events"
}

// Helper methods for FabricTransaction

// SetArgs sets the transaction arguments from a slice
func (t *FabricTransaction) SetArgs(args []string) {
	t.Args = args
}

// GetArgs returns the transaction arguments
func (t *FabricTransaction) GetArgs() []string {
	if t.Args == nil {
		return []string{}
	}
	return t.Args
}

// SetTransientMap sets the transient map from a map
func (t *FabricTransaction) SetTransientMap(transientMap map[string][]byte) error {
	if transientMap == nil {
		t.TransientMap = ""
		return nil
	}

	// Convert []byte values to base64 for JSON serialization
	jsonMap := make(map[string]string)
	for k, v := range transientMap {
		jsonMap[k] = string(v) // Store as string for simplicity
	}

	jsonData, err := json.Marshal(jsonMap)
	if err != nil {
		return err
	}

	t.TransientMap = string(jsonData)
	return nil
}

// GetTransientMap returns the transient map
func (t *FabricTransaction) GetTransientMap() (map[string][]byte, error) {
	if t.TransientMap == "" {
		return nil, nil
	}

	var jsonMap map[string]string
	err := json.Unmarshal([]byte(t.TransientMap), &jsonMap)
	if err != nil {
		return nil, err
	}

	transientMap := make(map[string][]byte)
	for k, v := range jsonMap {
		transientMap[k] = []byte(v)
	}

	return transientMap, nil
}

// SetMetadata sets the metadata from a map
func (t *FabricTransaction) SetMetadata(metadata map[string]interface{}) error {
	if metadata == nil {
		t.Metadata = ""
		return nil
	}

	jsonData, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	t.Metadata = string(jsonData)
	return nil
}

// GetMetadata returns the metadata as a map
func (t *FabricTransaction) GetMetadata() (map[string]interface{}, error) {
	if t.Metadata == "" {
		return nil, nil
	}

	var metadata map[string]interface{}
	err := json.Unmarshal([]byte(t.Metadata), &metadata)
	if err != nil {
		return nil, err
	}

	return metadata, nil
}

// MarkAsSubmitted marks the transaction as submitted to Fabric
func (t *FabricTransaction) MarkAsSubmitted(txID string, endorsingPeers []string) {
	now := time.Now()
	t.Status = FabricTransactionStatusSubmitted
	t.TxID = txID
	t.EndorsingPeers = endorsingPeers
	t.EndorsementCount = uint(len(endorsingPeers))
	t.SubmittedAt = &now
}

// MarkAsConfirmed marks the transaction as confirmed in a block
func (t *FabricTransaction) MarkAsConfirmed(blockNumber uint64, blockHash string, txIndex uint) {
	now := time.Now()
	t.Status = FabricTransactionStatusConfirmed
	t.BlockNumber = blockNumber
	t.BlockHash = blockHash
	t.TxIndex = txIndex
	t.ConfirmedAt = &now
}

// MarkAsFailed marks the transaction as failed
func (t *FabricTransaction) MarkAsFailed(errorMessage string) {
	now := time.Now()
	t.Status = FabricTransactionStatusFailed
	t.ErrorMessage = errorMessage
	t.FailedAt = &now
}

// IsSuccessful returns true if the transaction is confirmed
func (t *FabricTransaction) IsSuccessful() bool {
	return t.Status == FabricTransactionStatusConfirmed
}

// IsPending returns true if the transaction is pending or submitted
func (t *FabricTransaction) IsPending() bool {
	return t.Status == FabricTransactionStatusPending || t.Status == FabricTransactionStatusSubmitted
}

// HasError returns true if the transaction has failed
func (t *FabricTransaction) HasError() bool {
	return t.Status == FabricTransactionStatusFailed || t.ErrorMessage != ""
}