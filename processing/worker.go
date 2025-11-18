package worker

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	// Import necessary packages
	blockchain "tlng/blockchain/client"
	"tlng/blockchain/types"
	"tlng/config"
	"tlng/internal/messaging/consumer"
	"tlng/internal/models"
	"tlng/storage/store"
)

// Worker processes messages in batches
type Worker struct {
	workerConfig         config.WorkerConfig
	batchTimeout         time.Duration // Parsed from workerConfig.BatchTimeout
	consumerRetryDelay   time.Duration // Parsed from workerConfig.ConsumerRetryDelay
	blockchainTimeout    time.Duration // Parsed from workerConfig.BlockchainTimeout

	maxTaskRetries   int // Business rule for maximum task retries
	logger           *log.Logger
	store            store.Store
	consumer         consumer.Consumer
	blockchainClient blockchain.BlockchainClient // Interface for blockchain client
}

// New creates a new Worker instance
func New(cfg config.WorkerConfig, maxTaskRetries int, logger *log.Logger, s store.Store, c consumer.Consumer, bc blockchain.BlockchainClient) *Worker {
	// Add default safeguards if needed, though config should handle it
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}

	// Parse time duration strings
	batchTimeout, err := time.ParseDuration(cfg.BatchTimeout)
	if err != nil {
		logger.Printf("Warning: Invalid batch_timeout '%s', using default 1s", cfg.BatchTimeout)
		batchTimeout = 1 * time.Second
	}

	consumerRetryDelay, err := time.ParseDuration(cfg.ConsumerRetryDelay)
	if err != nil {
		logger.Printf("Warning: Invalid consumer_retry_delay '%s', using default 5s", cfg.ConsumerRetryDelay)
		consumerRetryDelay = 5 * time.Second
	}

	blockchainTimeout, err := time.ParseDuration(cfg.BlockchainTimeout)
	if err != nil {
		logger.Printf("Warning: Invalid blockchain_timeout '%s', using default 15s", cfg.BlockchainTimeout)
		blockchainTimeout = 15 * time.Second
	}

	return &Worker{
		workerConfig:         cfg,
		batchTimeout:         batchTimeout,
		consumerRetryDelay:   consumerRetryDelay,
		blockchainTimeout:    blockchainTimeout,
		maxTaskRetries:       maxTaskRetries,
		logger:               logger,
		store:                s,
		consumer:             c,
		blockchainClient:     bc,
	}
}

// Run starts the worker pool
func (w *Worker) Run(ctx context.Context) {
	w.logger.Printf("Starting worker pool with concurrency: %d, BatchSize: %d, BatchTimeout: %s",
		w.workerConfig.Concurrency, w.workerConfig.BatchSize, w.batchTimeout)
	var wg sync.WaitGroup
	for i := 0; i < w.workerConfig.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			w.logger.Printf("Worker %d started", workerID)
			w.processMessagesInBatch(ctx, workerID) // Call the batch processing loop
			w.logger.Printf("Worker %d stopped", workerID)
		}(i + 1)
	}
	wg.Wait()
	w.logger.Println("Worker pool stopped.")
}

// processMessagesInBatch is the main loop for a worker goroutine
func (w *Worker) processMessagesInBatch(ctx context.Context, workerID int) {
	batchMessages := make([]*models.LogMessage, 0, w.workerConfig.BatchSize)
	kafkaAcks := make([]func(success bool), 0, w.workerConfig.BatchSize)
	batchTimer := time.NewTimer(0) // Start with stopped timer
	if !batchTimer.Stop() {
		select {
		case <-batchTimer.C:
		default:
		}
	}

	// Helper function to submit batch
	processBatch := func() {
		if len(batchMessages) == 0 {
			return
		}

		// Stop and drain timer
		if !batchTimer.Stop() {
			select {
			case <-batchTimer.C:
			default:
			}
		}

		// Execute batch processing
		w.processAndAckBatch(ctx, workerID, batchMessages, kafkaAcks)

		// Reset for next batch
		batchMessages = make([]*models.LogMessage, 0, w.workerConfig.BatchSize)
		kafkaAcks = make([]func(success bool), 0, w.workerConfig.BatchSize)
	}

	for {
		select {
		case <-ctx.Done():
			w.logger.Printf("Worker %d: Context cancelled, stopping.", workerID)
			if len(kafkaAcks) > 0 {
				for _, ack := range kafkaAcks {
					ack(false)
				}
			}
			return

		case <-batchTimer.C:
			// Batch timeout reached
			processBatch()

		default:
			consumeCtx, consumeCancel := context.WithTimeout(ctx, 100*time.Millisecond)
			msg, ack, err := w.consumer.Consume(consumeCtx)
			consumeCancel()

			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
					continue
				}
				// Only log real consumer errors
				w.logger.Printf("Worker %d: Consumer error: %v", workerID, err)
				time.Sleep(w.consumerRetryDelay)
				continue
			}

			// Successfully got message
			if msg != nil {
				// Start batch timer on first message
				if len(batchMessages) == 0 {
					batchTimer.Reset(w.batchTimeout)
				}

				batchMessages = append(batchMessages, msg)
				kafkaAcks = append(kafkaAcks, ack)

				// Process immediately if batch is full
				if len(batchMessages) >= w.workerConfig.BatchSize {
					processBatch()
				}
			}
		}
	}
}

// processAndAckBatch handles processing and Kafka acknowledgement
func (w *Worker) processAndAckBatch(ctx context.Context, workerID int, batch []*models.LogMessage, acks []func(success bool)) {
	processingErr := w.handleBatch(ctx, batch) // Process the actual batch

	if processingErr != nil {
		// Transaction FAILED -> Nack ALL messages
		w.logger.Printf("Worker %d: Batch failed: %v (nacking %d messages)", workerID, processingErr, len(acks))
		for _, ack := range acks {
			ack(false)
		}
	} else {
		// Transaction SUCCEEDED -> Ack ALL messages
		for _, ack := range acks {
			ack(true)
		}
	}
}

func (w *Worker) handleBatch(ctx context.Context, batch []*models.LogMessage) error {
	if len(batch) == 0 {
		return nil
	}
	batchStart := time.Now()

	requestIDs := make([]string, 0, len(batch))
	msgMap := make(map[string]*models.LogMessage, len(batch)) // request_id -> message
	for _, msg := range batch {
		if msg.RequestID != "" { // Basic validation
			requestIDs = append(requestIDs, msg.RequestID)
			msgMap[msg.RequestID] = msg
		}
	}
	if len(requestIDs) == 0 {
		return nil
	} // No valid messages

	// --- 1. Pre-process database status ---
	validTasks := make(map[string]*store.LogStatus) // request_id -> task

	dbStart := time.Now()
	tasksFromDB, err := w.store.GetAndMarkBatchAsProcessing(ctx, requestIDs, w.maxTaskRetries)
	dbQueryDuration := time.Since(dbStart)

	if err != nil {
		return fmt.Errorf("DB error: GetAndMarkBatchAsProcessing failed: %v", err)
	}

	validEntries := make([]types.LogEntry, 0, len(tasksFromDB))

	for reqID, task := range tasksFromDB {
		switch task.Status {
		case store.StatusProcessing:
			msg := msgMap[reqID]     // Get corresponding original message
			validTasks[reqID] = task // Add to processing list
			validEntries = append(validEntries, types.LogEntry{
				LogHash:     msg.LogHash,
				LogContent:  msg.LogContent,
				SenderOrgID: msg.SourceOrgID,
				Timestamp:   msg.ReceivedTimestamp,
			})
		case store.StatusFailed:
			// Tasks with max retries exceeded are already marked as FAILED by the database
			// No further action needed - they will be acknowledged and dropped from processing
		}
	}

	// If no valid tasks to submit
	if len(validEntries) == 0 {
		return nil // Ack Kafka messages
	}

	// --- 2. Call blockchain client ---
	invokeCtx, cancel := context.WithTimeout(ctx, w.blockchainTimeout)
	defer cancel()
	bcStart := time.Now()
	batchProof, results, err := w.blockchainClient.SubmitLogsBatch(invokeCtx, validEntries)
	bcDuration := time.Since(bcStart)

	// Helper function to extract keys from map
	getValidRequestIDs := func(tasks map[string]*store.LogStatus) []string {
		ids := make([]string, 0, len(tasks))
		for reqID := range tasks {
			ids = append(ids, reqID)
		}
		return ids
	}

	// --- 3. Process results ---
	if err != nil { // Transaction failed
		w.logger.Printf("Blockchain error: %v", err)
		if markErr := w.store.MarkBatchForRetry(ctx, getValidRequestIDs(validTasks), err.Error()); markErr != nil {
			w.logger.Printf("CRITICAL: MarkBatchForRetry failed: %v", markErr)
		}
		return fmt.Errorf("SubmitLogsBatch failed: %w", err) // Trigger Nack
	}
	resultsMap := make(map[string]types.LogStatusInfo, len(results))
	for _, res := range results {
		resultsMap[res.LogHash] = res
	}

	// Collect completion and failure records for batch updates
	var completions []store.CompletionRecord
	var failures []store.FailureRecord

	for reqID, task := range validTasks {
		statusInfo, found := resultsMap[task.LogHash]
		if !found {
			errMsg := fmt.Sprintf("Missing result for log_hash %s (TxID: %s)", task.LogHash, batchProof.TransactionID)
			failures = append(failures, store.FailureRecord{
				RequestID:    reqID,
				ErrorMessage: errMsg,
			})
			continue
		}

		switch statusInfo.Status {
		case types.StatusSuccess:
			completions = append(completions, store.CompletionRecord{
				RequestID:      reqID,
				TxHash:         batchProof.TransactionID,
				LogHashOnChain: statusInfo.LogHash,
				BlockHeight:    batchProof.BlockHeight,
			})
		default:
			errMsg := fmt.Sprintf("Contract failed: %s - %s", statusInfo.Status, statusInfo.Message)
			failures = append(failures, store.FailureRecord{
				RequestID:    reqID,
				ErrorMessage: errMsg,
			})
		}
	}

	// Execute batch updates sequentially (now optimized with true bulk operations)
	dbUpdateStart := time.Now()
	var updateErrors []string

	// Sequential execution since both operations are now true bulk operations
	if len(completions) > 0 {
		if err := w.store.MarkBatchAsCompleted(ctx, completions); err != nil {
			updateErrors = append(updateErrors, fmt.Sprintf("completion update failed: %v", err))
		}
	}

	if len(failures) > 0 {
		if err := w.store.MarkBatchAsFailed(ctx, failures); err != nil {
			updateErrors = append(updateErrors, fmt.Sprintf("failure update failed: %v", err))
		}
	}

	dbUpdateDuration := time.Since(dbUpdateStart)

	// Log key performance metrics only
	totalTime := time.Since(batchStart)
	w.logger.Printf("Batch performance: size=%d, valid=%d, completions=%d, failures=%d, db_query=%v, db_updates=%v, blockchain=%v, total=%v",
		len(batch), len(validTasks), len(completions), len(failures), dbQueryDuration, dbUpdateDuration, bcDuration, totalTime)

	if len(updateErrors) > 0 {
		w.logger.Printf("DB update errors: %s", strings.Join(updateErrors, "; "))
	}

	return nil // Transaction succeeded, Ack Kafka messages
}
