package service

import (
	"context"
	"log"
	"sync"
	"time"

	"tlng/internal/messaging/producer"
	"tlng/internal/models"
	"tlng/storage/store"
)

// BatchProcessor handles batching of log requests for improved throughput
type BatchProcessor struct {
	batchSize    int
	batchTimeout time.Duration
	logger       *log.Logger
	store        store.Store
	producer     producer.Producer

	// Buffers
	buffer      []*batchEntry
	bufferMutex sync.Mutex
	ticker      *time.Ticker
	flushChan   chan []*batchEntry

	// Context for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

type batchEntry struct {
	input     *LogInput
	requestID string
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(batchSize int, batchTimeout time.Duration, flushChannelBuffer int,
	store store.Store, producer producer.Producer, logger *log.Logger) *BatchProcessor {

	ctx, cancel := context.WithCancel(context.Background())

	bp := &BatchProcessor{
		batchSize:    batchSize,
		batchTimeout: batchTimeout,
		logger:       logger,
		store:        store,
		producer:     producer,
		buffer:       make([]*batchEntry, 0, batchSize),
		flushChan:    make(chan []*batchEntry, flushChannelBuffer), // Configurable buffer for flush requests
		ctx:          ctx,
		cancel:       cancel,
	}

	// Start background goroutines
	bp.wg.Add(2)
	go bp.batchTimer()
	go bp.batchProcessor()

	return bp
}

// SubmitLog adds a log to the batch with pre-generated request ID
func (bp *BatchProcessor) SubmitLog(input *LogInput, requestID string) {
	entry := &batchEntry{
		input:     input,
		requestID: requestID,
	}

	// Add to buffer
	bp.bufferMutex.Lock()
	bp.buffer = append(bp.buffer, entry)
	shouldFlush := len(bp.buffer) >= bp.batchSize
	bp.bufferMutex.Unlock()

	// Trigger flush if buffer is full
	if shouldFlush {
		select {
		case bp.flushChan <- bp.getAndResetBuffer():
		default:
			bp.logger.Printf("Flush channel full, will flush on next timer")
		}
	}
}

// batchTimer handles periodic flushing
func (bp *BatchProcessor) batchTimer() {
	defer bp.wg.Done()

	bp.ticker = time.NewTicker(bp.batchTimeout)
	defer bp.ticker.Stop()

	for {
		select {
		case <-bp.ticker.C:
			bp.flushIfNeeded()
		case <-bp.ctx.Done():
			return
		}
	}
}

// batchProcessor handles actual batch processing
func (bp *BatchProcessor) batchProcessor() {
	defer bp.wg.Done()

	for {
		select {
		case batch := <-bp.flushChan:
			if len(batch) > 0 {
				bp.processBatch(batch)
			}
		case <-bp.ctx.Done():
			// Process remaining buffer before shutdown
			bp.bufferMutex.Lock()
			remaining := bp.buffer
			bp.buffer = nil
			bp.bufferMutex.Unlock()

			if len(remaining) > 0 {
				bp.processBatch(remaining)
			}
			return
		}
	}
}

// flushIfNeeded flushes the buffer if it has entries
func (bp *BatchProcessor) flushIfNeeded() {
	bp.bufferMutex.Lock()
	if len(bp.buffer) == 0 {
		bp.bufferMutex.Unlock()
		return
	}

	batch := make([]*batchEntry, len(bp.buffer))
	copy(batch, bp.buffer)
	bp.buffer = bp.buffer[:0] // Reset buffer
	bp.bufferMutex.Unlock()

	select {
	case bp.flushChan <- batch:
	default:
		// If flush channel is full, put it back in buffer
		bp.bufferMutex.Lock()
		bp.buffer = append(batch, bp.buffer...)
		bp.bufferMutex.Unlock()
	}
}

// getAndResetBuffer safely gets the current buffer and resets it
func (bp *BatchProcessor) getAndResetBuffer() []*batchEntry {
	bp.bufferMutex.Lock()
	defer bp.bufferMutex.Unlock()

	batch := make([]*batchEntry, len(bp.buffer))
	copy(batch, bp.buffer)
	bp.buffer = bp.buffer[:0]
	return batch
}

// processBatch handles the actual batch processing
func (bp *BatchProcessor) processBatch(batch []*batchEntry) {
	if len(batch) == 0 {
		return
	}

	start := time.Now()
	// bp.logger.Printf("Processing batch of %d logs", len(batch))

	// Prepare batch data
	logStatuses := make([]*store.LogStatus, len(batch))
	kafkaMessages := make([]*models.LogMessage, len(batch))

	for i := range batch {

		logHash := batch[i].input.ClientLogHash
		sourceOrgID := batch[i].input.ClientSourceOrgID

		logStatuses[i] = &store.LogStatus{
			RequestID:         batch[i].requestID,
			LogHash:           logHash,
			SourceOrgID:       sourceOrgID,
			ReceivedTimestamp: time.Now(),
			Status:            store.StatusReceived,
		}

		kafkaMessages[i] = &models.LogMessage{
			RequestID:         batch[i].requestID,
			LogContent:        batch[i].input.LogContent,
			LogHash:           logHash,
			SourceOrgID:       sourceOrgID,
			ReceivedTimestamp: time.Now().Format(time.RFC3339Nano),
		}
	}

	// Batch database insert
	dbStart := time.Now()
	dbErr := bp.store.InsertLogStatusBatch(context.Background(), logStatuses)
	dbDuration := time.Since(dbStart)

	if dbErr != nil {
		bp.logger.Printf("Batch database insert failed: %v", dbErr)
		// Notify all entries of failure
		for range batch {
			// In production, you might want to retry or use a dead letter queue
		}
		return
	}

	// Batch Kafka publish
	kafkaStart := time.Now()
	kafkaErr := bp.producer.PublishBatch(context.Background(), kafkaMessages)
	kafkaDuration := time.Since(kafkaStart)

	if kafkaErr != nil {
		bp.logger.Printf("Batch Kafka publish failed: %v", kafkaErr)
		// Handle failure - might need to retry or use dead letter queue
		return
	}

	totalDuration := time.Since(start)
	bp.logger.Printf("Batch processed: %d logs, DB: %v, Kafka: %v, Total: %v",
		len(batch), dbDuration, kafkaDuration, totalDuration)
}

// Close gracefully shuts down the batch processor
func (bp *BatchProcessor) Close() {
	bp.cancel()
	bp.wg.Wait()
	close(bp.flushChan)
}
