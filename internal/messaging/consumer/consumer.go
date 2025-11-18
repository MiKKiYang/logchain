package consumer

import (
	"context"
	"tlng/internal/models"
)

// Consumer defines the interface for message queue consumers.
type Consumer interface {
	// Consume blocks until a message is received or the context is cancelled.
	// It returns the message, an acknowledgement callback, and any error that occurred.
	// The ack callback: ack(true) for successful processing (message will be deleted);
	// ack(false) for temporary failure (message will be redelivered).
	Consume(ctx context.Context) (msg *models.LogMessage, ack func(success bool), err error)

	// Close gracefully shuts down the consumer connection.
	Close() error
}
