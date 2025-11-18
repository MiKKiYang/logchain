package service

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"time"

	"tlng/internal/messaging/producer"
	"tlng/storage/store"

	"github.com/google/uuid"
)

// LogInput defines the core information required for log submission
type LogInput struct {
	LogContent        string
	ClientLogHash     string     // Optional
	ClientSourceOrgID string     // Optional
	ClientTimestamp   *time.Time // Optional
}

// LogResult defines the return information after successful submission
type LogResult struct {
	RequestID               string
	ServerLogHash           string
	ServerReceivedTimestamp time.Time
}

// Service encapsulates the core business logic of the API gateway
type Service struct {
	store          store.Store
	producer       producer.Producer
	logger         *log.Logger
	batchProcessor *BatchProcessor
}

// NewService creates a new Service instance with configuration
func NewService(s store.Store, p producer.Producer, l *log.Logger, batchSize int, batchTimeout time.Duration, flushChannelBuffer int) *Service {
	return &Service{
		store:          s,
		producer:       p,
		logger:         l,
		batchProcessor: NewBatchProcessor(batchSize, batchTimeout, flushChannelBuffer, s, p, l),
	}
}

// SubmitLog handles the core logic of log submission
func (s *Service) SubmitLog(ctx context.Context, input *LogInput) (*LogResult, error) {
	// Log function start time
	// totalStart := time.Now()
	// s.logger.Println("Service: Starting to process SubmitLog request...")

	// 1. Validate input
	if input.LogContent == "" {
		return nil, fmt.Errorf("log_content cannot be empty")
	}

	// 2. Get received timestamp
	receivedTimestamp := time.Now()

	// 3. Calculate/validate hash
	serverLogHashBytes := sha256.Sum256([]byte(input.LogContent))
	serverLogHash := fmt.Sprintf("%x", serverLogHashBytes)
	if input.ClientLogHash != "" && input.ClientLogHash != serverLogHash {
		return nil, fmt.Errorf("client provided hash '%s' does not match server calculated hash '%s'", input.ClientLogHash, serverLogHash)
	}
	input.ClientLogHash = serverLogHash

	// 4. Generate Request ID
	requestID := uuid.NewString()

	// 5. Construct and return result immediately
	result := &LogResult{
		RequestID:               requestID,
		ServerLogHash:           serverLogHash,
		ServerReceivedTimestamp: receivedTimestamp,
	}

	// 6. Submit to batch processor (asynchronous)
	go s.batchProcessor.SubmitLog(input, requestID)

	// Log total function duration
	// totalDuration := time.Since(totalStart)
	// s.logger.Printf("Service: RequestID: %s, SubmitLog request processing completed (total duration: %s)", requestID, totalDuration)

	return result, nil
}

// Close gracefully shuts down the service
func (s *Service) Close() {
	s.batchProcessor.Close()
}
