package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"reciprocal-clubs-backend/pkg/shared/config"
	"reciprocal-clubs-backend/pkg/shared/logging"

	"github.com/nats-io/nats.go"
)

// MessageBus defines the message bus interface
type MessageBus interface {
	Publish(ctx context.Context, subject string, data interface{}) error
	PublishSync(ctx context.Context, subject string, data interface{}) error
	Subscribe(subject string, handler MessageHandler) error
	SubscribeQueue(subject, queue string, handler MessageHandler) error
	Request(ctx context.Context, subject string, data interface{}, response interface{}) error
	Close() error
	HealthCheck(ctx context.Context) error
}

// MessageHandler defines the message handler function signature
type MessageHandler func(ctx context.Context, msg *Message) error

// Message represents a message in the system
type Message struct {
	ID          string                 `json:"id"`
	Subject     string                 `json:"subject"`
	Data        json.RawMessage        `json:"data"`
	Headers     map[string]string      `json:"headers"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata"`
	Reply       string                 `json:"reply,omitempty"`
	Retries     int                    `json:"retries"`
	MaxRetries  int                    `json:"max_retries"`
}

// NATSMessageBus implements MessageBus using NATS
type NATSMessageBus struct {
	conn   *nats.Conn
	config *config.NATSConfig
	logger logging.Logger
}

// Event types for the system
const (
	// Member events
	MemberCreatedEvent   = "member.created"
	MemberUpdatedEvent   = "member.updated"
	MemberDeletedEvent   = "member.deleted"
	MemberVerifiedEvent  = "member.verified"

	// Reciprocal events
	AgreementCreatedEvent  = "agreement.created"
	AgreementUpdatedEvent  = "agreement.updated"
	AgreementApprovedEvent = "agreement.approved"
	VisitRecordedEvent     = "visit.recorded"
	VisitVerifiedEvent     = "visit.verified"

	// Blockchain events
	TransactionSubmittedEvent = "transaction.submitted"
	TransactionConfirmedEvent = "transaction.confirmed"
	TransactionFailedEvent    = "transaction.failed"
	ChannelCreatedEvent       = "channel.created"
	ChannelJoinedEvent        = "channel.joined"

	// Notification events
	NotificationSentEvent   = "notification.sent"
	NotificationFailedEvent = "notification.failed"

	// System events
	HealthCheckEvent = "system.health_check"
	ServiceStartedEvent = "service.started"
	ServiceStoppedEvent = "service.stopped"
)

// NewNATSMessageBus creates a new NATS message bus
func NewNATSMessageBus(cfg *config.NATSConfig, logger logging.Logger) (*NATSMessageBus, error) {
	// Configure NATS options
	opts := []nats.Option{
		nats.Name(cfg.ClientID),
		nats.MaxReconnects(cfg.MaxReconnect),
		nats.ReconnectWait(time.Duration(cfg.ReconnectWait) * time.Second),
		nats.Timeout(time.Duration(cfg.ConnectTimeout) * time.Second),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			logger.Error("NATS disconnected", map[string]interface{}{
				"error": err.Error(),
			})
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("NATS reconnected", map[string]interface{}{
				"url": nc.ConnectedUrl(),
			})
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			logger.Info("NATS connection closed", nil)
		}),
	}

	// Connect to NATS
	conn, err := nats.Connect(cfg.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	logger.Info("Connected to NATS", map[string]interface{}{
		"url":       cfg.URL,
		"client_id": cfg.ClientID,
	})

	return &NATSMessageBus{
		conn:   conn,
		config: cfg,
		logger: logger,
	}, nil
}

// Publish publishes a message asynchronously
func (mb *NATSMessageBus) Publish(ctx context.Context, subject string, data interface{}) error {
	message := &Message{
		ID:        generateMessageID(),
		Subject:   subject,
		Headers:   extractHeaders(ctx),
		Timestamp: time.Now().UTC(),
		Metadata:  extractMetadata(ctx),
		Retries:   0,
		MaxRetries: 3,
	}

	// Marshal data
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal message data: %w", err)
	}
	message.Data = dataBytes

	// Marshal complete message
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Publish to NATS
	if err := mb.conn.Publish(subject, messageBytes); err != nil {
		mb.logger.Error("Failed to publish message", map[string]interface{}{
			"error":   err.Error(),
			"subject": subject,
			"message_id": message.ID,
		})
		return fmt.Errorf("failed to publish message: %w", err)
	}

	mb.logger.Debug("Message published", map[string]interface{}{
		"subject":    subject,
		"message_id": message.ID,
	})

	return nil
}

// PublishSync publishes a message synchronously
func (mb *NATSMessageBus) PublishSync(ctx context.Context, subject string, data interface{}) error {
	if err := mb.Publish(ctx, subject, data); err != nil {
		return err
	}

	return mb.conn.Flush()
}

// Subscribe subscribes to a subject
func (mb *NATSMessageBus) Subscribe(subject string, handler MessageHandler) error {
	_, err := mb.conn.Subscribe(subject, mb.createNATSHandler(handler))
	if err != nil {
		mb.logger.Error("Failed to subscribe to subject", map[string]interface{}{
			"error":   err.Error(),
			"subject": subject,
		})
		return fmt.Errorf("failed to subscribe to subject %s: %w", subject, err)
	}

	mb.logger.Info("Subscribed to subject", map[string]interface{}{
		"subject": subject,
	})

	return nil
}

// SubscribeQueue subscribes to a subject with queue group
func (mb *NATSMessageBus) SubscribeQueue(subject, queue string, handler MessageHandler) error {
	_, err := mb.conn.QueueSubscribe(subject, queue, mb.createNATSHandler(handler))
	if err != nil {
		mb.logger.Error("Failed to subscribe to queue", map[string]interface{}{
			"error":   err.Error(),
			"subject": subject,
			"queue":   queue,
		})
		return fmt.Errorf("failed to subscribe to queue %s for subject %s: %w", queue, subject, err)
	}

	mb.logger.Info("Subscribed to queue", map[string]interface{}{
		"subject": subject,
		"queue":   queue,
	})

	return nil
}

// Request makes a request-reply call
func (mb *NATSMessageBus) Request(ctx context.Context, subject string, data interface{}, response interface{}) error {
	message := &Message{
		ID:        generateMessageID(),
		Subject:   subject,
		Headers:   extractHeaders(ctx),
		Timestamp: time.Now().UTC(),
		Metadata:  extractMetadata(ctx),
	}

	// Marshal data
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal request data: %w", err)
	}
	message.Data = dataBytes

	// Marshal complete message
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal request message: %w", err)
	}

	// Make request with timeout
	timeout := time.Duration(mb.config.RequestTimeout) * time.Second
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	resp, err := mb.conn.RequestWithContext(ctxWithTimeout, subject, messageBytes)
	if err != nil {
		mb.logger.Error("Request failed", map[string]interface{}{
			"error":      err.Error(),
			"subject":    subject,
			"message_id": message.ID,
		})
		return fmt.Errorf("request failed: %w", err)
	}

	// Parse response
	var responseMsg Message
	if err := json.Unmarshal(resp.Data, &responseMsg); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Unmarshal response data
	if err := json.Unmarshal(responseMsg.Data, response); err != nil {
		return fmt.Errorf("failed to unmarshal response data: %w", err)
	}

	mb.logger.Debug("Request completed", map[string]interface{}{
		"subject":       subject,
		"message_id":    message.ID,
		"response_time": time.Since(message.Timestamp).Milliseconds(),
	})

	return nil
}

// Close closes the NATS connection
func (mb *NATSMessageBus) Close() error {
	if mb.conn != nil && !mb.conn.IsClosed() {
		mb.conn.Close()
		mb.logger.Info("NATS connection closed", nil)
	}
	return nil
}

// HealthCheck performs a health check on the message bus
func (mb *NATSMessageBus) HealthCheck(ctx context.Context) error {
	if mb.conn == nil || mb.conn.IsClosed() {
		return fmt.Errorf("NATS connection is closed")
	}

	if !mb.conn.IsConnected() {
		return fmt.Errorf("NATS is not connected")
	}

	// Test with a ping
	if err := mb.conn.FlushTimeout(5 * time.Second); err != nil {
		return fmt.Errorf("NATS health check failed: %w", err)
	}

	return nil
}

// createNATSHandler creates a NATS message handler
func (mb *NATSMessageBus) createNATSHandler(handler MessageHandler) func(*nats.Msg) {
	return func(natsMsg *nats.Msg) {
		ctx := context.Background()

		// Parse message
		var message Message
		if err := json.Unmarshal(natsMsg.Data, &message); err != nil {
			mb.logger.Error("Failed to unmarshal message", map[string]interface{}{
				"error":   err.Error(),
				"subject": natsMsg.Subject,
			})
			return
		}

		// Add context from message headers
		ctx = contextFromHeaders(ctx, message.Headers)
		ctx = contextFromMetadata(ctx, message.Metadata)

		// Set reply subject
		message.Reply = natsMsg.Reply

		// Handle message with retry logic
		maxRetries := message.MaxRetries
		if maxRetries == 0 {
			maxRetries = 3
		}

		for attempt := 0; attempt <= maxRetries; attempt++ {
			message.Retries = attempt

			err := handler(ctx, &message)
			if err == nil {
				mb.logger.Debug("Message handled successfully", map[string]interface{}{
					"subject":    message.Subject,
					"message_id": message.ID,
					"attempt":    attempt + 1,
				})
				return
			}

			mb.logger.Warn("Message handling failed", map[string]interface{}{
				"error":      err.Error(),
				"subject":    message.Subject,
				"message_id": message.ID,
				"attempt":    attempt + 1,
				"max_retries": maxRetries,
			})

			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt+1) * time.Second)
			}
		}

		mb.logger.Error("Message handling failed after retries", map[string]interface{}{
			"subject":    message.Subject,
			"message_id": message.ID,
			"max_retries": maxRetries,
		})
	}
}

// Utility functions

func generateMessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}

func extractHeaders(ctx context.Context) map[string]string {
	headers := make(map[string]string)

	if correlationID := logging.GetCorrelationID(ctx); correlationID != "" {
		headers["correlation_id"] = correlationID
	}

	if service := logging.GetService(ctx); service != "" {
		headers["service"] = service
	}

	return headers
}

func extractMetadata(ctx context.Context) map[string]interface{} {
	metadata := make(map[string]interface{})

	if userID := logging.GetUserID(ctx); userID != nil {
		metadata["user_id"] = userID
	}

	if clubID := logging.GetClubID(ctx); clubID != nil {
		metadata["club_id"] = clubID
	}

	return metadata
}

func contextFromHeaders(ctx context.Context, headers map[string]string) context.Context {
	if correlationID, exists := headers["correlation_id"]; exists {
		ctx = logging.ContextWithCorrelationID(ctx, correlationID)
	}

	if service, exists := headers["service"]; exists {
		ctx = logging.ContextWithService(ctx, service)
	}

	return ctx
}

func contextFromMetadata(ctx context.Context, metadata map[string]interface{}) context.Context {
	if userID, exists := metadata["user_id"]; exists {
		ctx = logging.ContextWithUserID(ctx, userID)
	}

	if clubID, exists := metadata["club_id"]; exists {
		ctx = logging.ContextWithClubID(ctx, clubID)
	}

	return ctx
}