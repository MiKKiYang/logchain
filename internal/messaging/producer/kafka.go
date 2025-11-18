package producer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
	"tlng/config"
	"tlng/internal/models"
)

// KafkaProducer implements the Producer interface
type KafkaProducer struct {
	writer *kafka.Writer
	logger *log.Logger
	topic  string
}

// NewKafkaProducer creates a new KafkaProducer
func NewKafkaProducer(cfg config.KafkaProducerConfig, logger *log.Logger) (*KafkaProducer, error) {
	if len(cfg.Brokers) == 0 || cfg.Topic == "" {
		return nil, errors.New("kafka producer configuration incomplete: both brokers and topic are required")
	}

	// Set defaults for batch settings if not configured
	batchSize := cfg.BatchSize
	if batchSize == 0 {
		batchSize = 100 // Default batch size
	}

	batchTimeout := cfg.BatchTimeout
	if batchTimeout == 0 {
		batchTimeout = 100 * time.Millisecond // Default batch timeout
	}

	batchBytes := cfg.BatchBytes
	if batchBytes == 0 {
		batchBytes = 5 * 1024 * 1024 // Default 5MB
	}

	// Parse required_acks setting
	var requiredAcks kafka.RequiredAcks
	switch cfg.RequiredAcks {
	case "none":
		requiredAcks = kafka.RequireNone
	case "one":
		requiredAcks = kafka.RequireOne
	case "all":
		requiredAcks = kafka.RequireAll
	default:
		requiredAcks = kafka.RequireOne // Default to wait for leader
	}

	// Set async default if not configured
	asyncMode := cfg.Async
	if !cfg.Async && cfg.RequiredAcks == "" {
		asyncMode = true // Default to async mode
	}

	// Set timeouts if not configured
	writeTimeout := cfg.WriteTimeout
	if writeTimeout == 0 {
		writeTimeout = 5 * time.Second
	}

	readTimeout := cfg.ReadTimeout
	if readTimeout == 0 {
		readTimeout = 5 * time.Second
	}

	// Configure Kafka Writer
	w := &kafka.Writer{
		Addr:     kafka.TCP(cfg.Brokers...),
		Topic:    cfg.Topic,
		Balancer: &kafka.LeastBytes{},

		BatchSize:    batchSize,
		BatchTimeout: batchTimeout,
		BatchBytes:   int64(batchBytes),

		// Reliability settings
		RequiredAcks: requiredAcks,
		Async:        asyncMode,

		// Performance settings
		WriteTimeout: writeTimeout,
		ReadTimeout:  readTimeout,

		// Error handling
		ErrorLogger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			logger.Printf("Kafka Writer Error: "+msg, args...)
		}),
	}

	logger.Printf("Kafka producer created, connected to Brokers: %v, Topic: %s", cfg.Brokers, cfg.Topic)

	return &KafkaProducer{
		writer: w,
		logger: logger,
		topic:  cfg.Topic,
	}, nil
}

// Publish sends a message
func (p *KafkaProducer) Publish(ctx context.Context, msg *models.LogMessage) error {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to serialize log message: %w", err)
	}

	kafkaMsg := kafka.Message{
		// Key can be used for partitioning strategy, using RequestID here
		Key:   []byte(msg.RequestID),
		Value: msgBytes,
	}

	// Send message
	err = p.writer.WriteMessages(ctx, kafkaMsg)
	if err != nil {
		// This error is usually local errors like buffer full or context cancellation
		p.logger.Printf("Failed to send Kafka message to buffer (RequestID: %s): %v", msg.RequestID, err)
		return fmt.Errorf("failed to write to Kafka buffer: %w", err)
	}

	// p.logger.Printf("Successfully added Kafka message (RequestID: %s) to send queue (Topic: %s)", msg.RequestID, p.topic)
	return nil
}

// PublishBatch sends log messages in batch to the specified topic
func (p *KafkaProducer) PublishBatch(ctx context.Context, msgs []*models.LogMessage) error {
	if len(msgs) == 0 {
		return nil
	}

	kafkaMsgs := make([]kafka.Message, len(msgs))
	for i, msg := range msgs {
		msgBytes, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("failed to serialize log message (RequestID: %s): %w", msg.RequestID, err)
		}

		kafkaMsgs[i] = kafka.Message{
			Key:   []byte(msg.RequestID),
			Value: msgBytes,
		}
	}

	// Send messages in batch
	err := p.writer.WriteMessages(ctx, kafkaMsgs...)
	if err != nil {
		p.logger.Printf("Failed to send Kafka messages in batch (count: %d): %v", len(msgs), err)
		return fmt.Errorf("failed to batch write to Kafka buffer: %w", err)
	}

	p.logger.Printf("Successfully added %d Kafka messages to send queue (Topic: %s)", len(msgs), p.topic)
	return nil
}

// Close closes the producer
func (p *KafkaProducer) Close() error {
	p.logger.Println("Closing Kafka producer (and flushing buffer)...")
	return p.writer.Close() // Close will attempt to send remaining messages in buffer
}

var _ Producer = (*KafkaProducer)(nil) // Compile-time interface check
