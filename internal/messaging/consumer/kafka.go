package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"tlng/config"
	"tlng/internal/models"

	"github.com/segmentio/kafka-go"
)

// KafkaConsumer implements the Consumer interface to consume log messages from Kafka
type KafkaConsumer struct {
	reader *kafka.Reader
	logger *log.Logger
}

// NewKafkaConsumer creates a new KafkaConsumer instance
func NewKafkaConsumer(cfg config.KafkaConsumerConfig, logger *log.Logger) (*KafkaConsumer, error) {
	if len(cfg.Brokers) == 0 || cfg.Topic == "" || cfg.GroupID == "" {
		return nil, errors.New("incomplete kafka configuration: brokers, topic, group_id are all required")
	}

	// Parse session timeout with default
	sessionTimeout, err := time.ParseDuration(cfg.SessionTimeout)
	if err != nil {
		logger.Printf("Warning: Invalid session_timeout '%s', using default 30s", cfg.SessionTimeout)
		sessionTimeout = 30 * time.Second
	}

	// Parse heartbeat interval with default
	heartbeatInterval, err := time.ParseDuration(cfg.HeartbeatInterval)
	if err != nil {
		logger.Printf("Warning: Invalid heartbeat_interval '%s', using default 3s", cfg.HeartbeatInterval)
		heartbeatInterval = 3 * time.Second
	}

	// Set default auto offset reset
	autoOffsetReset := cfg.AutoOffsetReset
	if autoOffsetReset == "" {
		autoOffsetReset = "earliest"
	}

	// Configure Kafka reader
	readerConfig := kafka.ReaderConfig{
		Brokers:           cfg.Brokers,
		GroupID:           cfg.GroupID,
		Topic:             cfg.Topic,
		MinBytes:          10e3,            // 10KB
		MaxBytes:          10e6,            // 10MB
		MaxWait:           1 * time.Second, // Max wait time for message fetch
		CommitInterval:    time.Second,     // Auto commit interval (used if not manually committing)
		SessionTimeout:    sessionTimeout,
		HeartbeatInterval: heartbeatInterval,
		StartOffset:       kafka.FirstOffset, // Will be overridden by autoOffsetReset
	}

	// Set start offset based on autoOffsetReset
	switch autoOffsetReset {
	case "latest":
		readerConfig.StartOffset = kafka.LastOffset
	case "earliest":
		readerConfig.StartOffset = kafka.FirstOffset
	default:
		logger.Printf("Warning: Unknown auto_offset_reset '%s', using earliest", autoOffsetReset)
		readerConfig.StartOffset = kafka.FirstOffset
	}

	r := kafka.NewReader(readerConfig)

	logger.Printf("Kafka consumer created, connected to Brokers: %v, Topic: %s, GroupID: %s", cfg.Brokers, cfg.Topic, cfg.GroupID)

	return &KafkaConsumer{
		reader: r,
		logger: logger,
	}, nil
}

// Consume implements the Consumer interface by reading messages from Kafka
func (k *KafkaConsumer) Consume(ctx context.Context) (msg *models.LogMessage, ack func(success bool), err error) {
	// Fetch message from Kafka
	kafkaMsg, err := k.reader.FetchMessage(ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			k.logger.Println("Kafka consumer: Context cancelled, stopping consumption.")
			return nil, nil, ctx.Err()
		}
		return nil, nil, err
	}

	// Deserialize message body (assumes JSON format)
	var logMsg models.LogMessage
	if err := json.Unmarshal(kafkaMsg.Value, &logMsg); err != nil {
		k.logger.Printf("Kafka consumer: Failed to deserialize message (Offset: %d): %v. Message will be discarded.", kafkaMsg.Offset, err)
		_ = k.reader.CommitMessages(ctx, kafkaMsg) // Commit offset to avoid blocking
		return nil, nil, fmt.Errorf("message deserialization failed: %w", err)
	}

	// Create ack callback
	ackCallback := func(success bool) {
		commitCtx := context.Background()
		if success {
			if err := k.reader.CommitMessages(commitCtx, kafkaMsg); err != nil {
				k.logger.Printf("Kafka consumer: Failed to commit offset %d: %v", kafkaMsg.Offset, err)
			}
		} else {
			k.logger.Printf("Kafka consumer: NACK received for offset %d (request_id %s). Offset will not be committed.", kafkaMsg.Offset, logMsg.RequestID)
		}
	}

	return &logMsg, ackCallback, nil
}

// Close implements the Consumer interface by closing the Kafka reader
func (k *KafkaConsumer) Close() error {
	k.logger.Println("Closing Kafka consumer...")
	return k.reader.Close()
}

// Ensure KafkaConsumer implements the Consumer interface
var _ Consumer = (*KafkaConsumer)(nil)
