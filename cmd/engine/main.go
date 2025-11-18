package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	blockchain "tlng/blockchain/client"
	"tlng/config"
	"tlng/internal/messaging/consumer"
	worker "tlng/processing"
	"tlng/storage/store"
)

const engineConfigPath = "./config/engine.defaults.yml"

func main() {
	logger := log.New(os.Stdout, "[ENGINE] ", log.LstdFlags|log.Lshortfile)
	logger.Println("Starting Attestation Engine...")

	// 1. Load Engine Config
	engineCfg, err := config.LoadEngineConfig(engineConfigPath)
	if err != nil {
		logger.Fatalf("FATAL: Failed to load engine configuration: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 2. Initialize Dependencies
	logger.Println("Initializing database connection...")
	dbStore, err := store.NewPostgresStore(ctx, engineCfg.Database.DSN, engineCfg.Database.MinConnections, engineCfg.Database.MaxConnections, logger)
	if err != nil {
		logger.Fatalf("FATAL: Failed to initialize database store: %v", err)
	}
	defer dbStore.Close()

	logger.Println("Initializing blockchain client using configuration files...")
	// Load blockchain client
	bcClientImpl, err := blockchain.NewBlockchainClientFromFile(engineCfg.BlockchainClientConfigPath, logger)
	if err != nil {
		logger.Fatalf("FATAL: Failed to initialize ChainMaker client: %v", err)
	}
	defer bcClientImpl.Close()

	// 3. Initialize Multiple Consumers
	var mqConsumers []consumer.Consumer
	if len(engineCfg.KafkaConsumer.Brokers) > 0 && engineCfg.KafkaConsumer.Brokers[0] != "mock://local" {
		logger.Printf("Initializing %d Kafka message queue consumers...", engineCfg.KafkaConsumer.Count)
		for i := 0; i < engineCfg.KafkaConsumer.Count; i++ {
			kafkaConsumer, err := consumer.NewKafkaConsumer(engineCfg.KafkaConsumer, logger)
			if err != nil {
				logger.Fatalf("FATAL: Failed to initialize Kafka consumer %d: %v", i, err)
			}
			mqConsumers = append(mqConsumers, kafkaConsumer)
		}
	} else {
		logger.Println("Initializing Mock message queue consumer...")
		mqConsumers = append(mqConsumers, consumer.NewMockConsumer(logger))
	}

	// Ensure all consumers are closed on exit
	defer func() {
		for _, c := range mqConsumers {
			c.Close()
		}
	}()

	// 4. Create and Start Multiple Workers
	var workers []*worker.Worker
	var wg sync.WaitGroup

	for i, consumer := range mqConsumers {
		workerInstance := worker.New(engineCfg.Worker, engineCfg.MaxTaskRetries, logger, dbStore, consumer, bcClientImpl)
		workers = append(workers, workerInstance)

		wg.Add(1)
		go func(workerID int, w *worker.Worker) {
			defer wg.Done()
			logger.Printf("Starting worker %d with its dedicated consumer...", workerID)
			w.Run(ctx)
			logger.Printf("Worker %d stopped.", workerID)
		}(i+1, workerInstance)
	}

	logger.Printf("Attestation Engine started with %d workers. Press Ctrl+C to stop.", len(workers))

	// 6. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Println("Received shutdown signal, initiating graceful shutdown...")
	cancel()

	// Wait for all workers to finish
	logger.Println("Waiting for all workers to finish...")
	wg.Wait()

	logger.Println("Attestation Engine shut down gracefully.")
}
