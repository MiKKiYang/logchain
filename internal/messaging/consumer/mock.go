package consumer

import (
	"context"
	"errors"
	"log"
	"strconv"
	"time"
	"tlng/internal/models"
)

// MockConsumer uses fixed predefined messages for testing.
type MockConsumer struct {
	logger   *log.Logger
	messages chan *models.LogMessage
}

// PredefinedMessages stores the messages to be simulated.
var PredefinedMessages []*models.LogMessage

// init generates fixed test data when the package is loaded.
func init() {
	PredefinedMessages = make([]*models.LogMessage, 0, 3)

	msg1 := &models.LogMessage{
		RequestID:         "a1b1c1d1-e1f1-1111-2222-1234567890ab",
		LogContent:        "Fixed mock log content 1",
		LogHash:           "fixedhash001",
		SourceOrgID:       "mock-org-1",
		ReceivedTimestamp: strconv.FormatInt(time.Now().Unix()-60, 10),
	}
	PredefinedMessages = append(PredefinedMessages, msg1)

	msg2 := &models.LogMessage{
		RequestID:         "a2b2c2d2-e2f2-3333-4444-abcdef123456",
		LogContent:        "Fixed mock log content 2 with more detail",
		LogHash:           "fixedhash002",
		SourceOrgID:       "mock-org-2",
		ReceivedTimestamp: strconv.FormatInt(time.Now().Unix()-30, 10),
	}
	PredefinedMessages = append(PredefinedMessages, msg2)

	// Message 3 has same hash as message 1 (simulates duplicate submission)
	msg3 := &models.LogMessage{
		RequestID:         "a3b3c3d3-e3f3-5555-6666-fedcba654321",
		LogContent:        "Fixed mock log content 1",
		LogHash:           "fixedhash001",
		SourceOrgID:       "mock-org-1",
		ReceivedTimestamp: strconv.FormatInt(time.Now().Unix(), 10),
	}
	PredefinedMessages = append(PredefinedMessages, msg3)
}

// NewMockConsumer creates a MockConsumer and loads predefined messages.
func NewMockConsumer(logger *log.Logger) *MockConsumer {
	mc := &MockConsumer{
		logger:   logger,
		messages: make(chan *models.LogMessage, len(PredefinedMessages)+5),
	}
	logger.Println("[MockConsumer] Initializing with predefined messages...")
	for _, msg := range PredefinedMessages {
		mc.messages <- msg
		logger.Printf("[MockConsumer] Added predefined message: request_id=%s", msg.RequestID)
	}
	logger.Println("[MockConsumer] Predefined messages loaded")
	return mc
}

// Consume reads predefined messages from the channel.
func (m *MockConsumer) Consume(ctx context.Context) (msg *models.LogMessage, ack func(success bool), err error) {
	m.logger.Println("[MockConsumer] Waiting for message...")
	select {
	case <-ctx.Done():
		m.logger.Println("[MockConsumer] Context cancelled, stopping consumption")
		return nil, nil, ctx.Err()
	case msg := <-m.messages:
		if msg == nil {
			m.logger.Println("[MockConsumer] Message channel closed")
			return nil, nil, errors.New("message channel closed")
		}
		m.logger.Printf("[MockConsumer] Consumed message: request_id=%s", msg.RequestID)

		ackCallback := func(success bool) {
			if success {
				m.logger.Printf("[MockConsumer] ACK received for message: request_id=%s", msg.RequestID)
			} else {
				m.logger.Printf("[MockConsumer] NACK received for message: request_id=%s. Re-queueing (mock)", msg.RequestID)
				select {
				case m.messages <- msg:
					m.logger.Printf("[MockConsumer] Message re-queued: request_id=%s", msg.RequestID)
				default:
					m.logger.Printf("[MockConsumer] Warning: Failed to re-queue message (channel full?): request_id=%s", msg.RequestID)
				}
			}
		}
		return msg, ackCallback, nil
	}
}

// Close closes the message channel.
func (m *MockConsumer) Close() error {
	m.logger.Println("[MockConsumer] Closing...")
	close(m.messages)
	return nil
}

var _ Consumer = (*MockConsumer)(nil)
