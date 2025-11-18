package producer

import (
	"context"
	"tlng/internal/models"
)

// Producer defines the interface for message queue producer
type Producer interface {
	// Publish sends a single log message to the specified topic
	Publish(ctx context.Context, msg *models.LogMessage) error

	// PublishBatch sends log messages in batch to the specified topic
	PublishBatch(ctx context.Context, msgs []*models.LogMessage) error

	// Close closes the producer connection
	Close() error
}
