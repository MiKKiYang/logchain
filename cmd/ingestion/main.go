package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"google.golang.org/grpc"

	// Import created packages
	apiconfig "tlng/config"                     // Unified configuration package
	grpchandler "tlng/ingestion/service/grpc"          // gRPC Handler (only includes SubmitLog)
	httphandler "tlng/ingestion/service/http"          // HTTP Handler (only includes SubmitLog)
	"tlng/internal/messaging/producer"         // Kafka producer
	core "tlng/ingestion/service/core"                   // Core Service (only includes SubmitLog logic)
	"tlng/storage/store"                       // Database Store (only needs InsertLogStatus)
	pb "tlng/proto/logingestion"               // Protobuf definitions
)

// API Gateway configuration file path
const apiConfigPath = "./config/ingestion.defaults.yml"

func main() {
	logger := log.New(os.Stdout, "[API-GW] ", log.LstdFlags|log.Lshortfile)
	logger.Println("Starting API Gateway (Ingestion Service)...")

	// 1. Load API Gateway configuration
	cfg, err := apiconfig.LoadApiGatewayConfig(apiConfigPath)
	if err != nil {
		logger.Fatalf("Failed to load API Gateway configuration: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 2. Initialize dependencies (only need DB and Kafka Producer)
	logger.Println("Initializing database connection...")
	dbStore, err := store.NewPostgresStore(ctx, cfg.Database.DSN, cfg.Database.MaxConnections, cfg.Database.MinConnections, logger)
	if err != nil {
		logger.Fatalf("Failed to initialize database store: %v", err)
	}
	defer dbStore.Close()

	logger.Println("Initializing Kafka producer...")
	kafkaProducer, err := producer.NewKafkaProducer(cfg.KafkaProducer, logger)
	if err != nil {
		logger.Fatalf("Failed to initialize Kafka producer: %v", err)
	}
	defer kafkaProducer.Close()

	// 3. Create core Service (using configuration parameters) and Handlers
	coreService := core.NewService(
		dbStore,
		kafkaProducer,
		logger,
		cfg.BatchProcessor.BatchSize,
		cfg.BatchProcessor.BatchTimeout,
		cfg.BatchProcessor.FlushChannelBuffer,
	)
	defer coreService.Close() // Ensure service is closed on exit
	logHttpHandler := httphandler.NewLogHandler(coreService, logger)
	logGrpcService := grpchandler.NewServer(coreService, logger) // gRPC service implementation

	var wg sync.WaitGroup

	// 4. [Conditional startup] HTTP server (only register write routes)
	var httpServer *http.Server
	if cfg.HttpListenAddr != "" {
		mux := http.NewServeMux()
		mux.HandleFunc("/v1/logs", logHttpHandler.SubmitLog) // Only register write Handler

		// Use HTTP server configuration with defaults
		readTimeout := cfg.HttpServer.ReadTimeout
		if readTimeout == 0 {
			readTimeout = 5 * time.Second
		}

		writeTimeout := cfg.HttpServer.WriteTimeout
		if writeTimeout == 0 {
			writeTimeout = 10 * time.Second
		}

		idleTimeout := cfg.HttpServer.IdleTimeout
		if idleTimeout == 0 {
			idleTimeout = 60 * time.Second
		}

		maxHeaderBytes := cfg.HttpServer.MaxHeaderBytes
		if maxHeaderBytes == 0 {
			maxHeaderBytes = 1 << 20 // 1 MB
		}

		// Create HTTP server with optimized settings
		httpServer = &http.Server{
			Addr:           cfg.HttpListenAddr,
			Handler:        mux,
			ReadTimeout:    readTimeout,
			WriteTimeout:   writeTimeout,
			IdleTimeout:    idleTimeout,
			MaxHeaderBytes: maxHeaderBytes,
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			logger.Printf("HTTP server listening on %s", cfg.HttpListenAddr)
			if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Fatalf("HTTP server startup failed: %v", err)
			}
			logger.Println("HTTP server stopped listening.")
		}()
	} else {
		logger.Println("http_listen_addr not configured, skipping HTTP server startup.")
	}

	// 5. [Conditional startup] gRPC server (only register write service)
	var grpcServer *grpc.Server
	if cfg.GrpcListenAddr != "" {
		lis, err := net.Listen("tcp", cfg.GrpcListenAddr)
		if err != nil {
			logger.Fatalf("Unable to listen on gRPC port %s: %v", cfg.GrpcListenAddr, err)
		}
		grpcServer = grpc.NewServer()
		pb.RegisterLogIngestionServer(grpcServer, logGrpcService) // Only register LogIngestion service
		wg.Add(1)
		go func() {
			defer wg.Done()
			logger.Printf("gRPC server listening on %s", cfg.GrpcListenAddr)
			if err := grpcServer.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
				logger.Fatalf("gRPC server startup failed: %v", err)
			}
			logger.Println("gRPC server stopped listening.")
		}()
	} else {
		logger.Println("grpc_listen_addr not configured, skipping gRPC server startup.")
	}

	// 6. Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	logger.Printf("Received shutdown signal: %s, starting graceful shutdown of API Gateway...", sig)
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	if httpServer != nil {
		logger.Println("Shutting down HTTP server...")
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Printf("HTTP server shutdown failed: %v", err)
		} else {
			logger.Println("HTTP server shutdown.")
		}
	}
	if grpcServer != nil {
		logger.Println("Shutting down gRPC server...")
		grpcServer.GracefulStop()
		logger.Println("gRPC server shutdown.")
	}

	// Wait for HTTP server and gRPC server to finish
	wg.Wait()
	logger.Println("All servers stopped. API Gateway shutdown.")
}
