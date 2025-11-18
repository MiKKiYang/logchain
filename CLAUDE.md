# CLAUDE.md

This file provides development guidance for working with code in this repository.

## 📚 Documentation Context

This is the **development guide** focused on implementation details, coding standards, and practical development guidance.

- 🏗️ **For system architecture and design decisions**, see [design.md](design.md)
- 🚀 **For project overview and quick start**, see [README.md](README.md)
- 💻 **This document**: Development-specific guidance and implementation details

## Project Overview

This is a Trusted Log Attestation System (可信日志存证系统) - a system for transparently attesting logs on a blockchain with multi-dimensional verifiability.

📖 **For complete system architecture, component specifications, and design decisions, see [design.md](design.md)**

This file provides development-specific guidance and practical implementation details.

Currently implemented services:
1. **Log Ingestion Service** - Core service that receives logs via HTTP/gRPC
2. **Blockchain Processing Service** - Processing service that consumes logs and notarizes them on ChainMaker blockchain

## Architecture (Development Focus)

This section focuses on implementation details. For architectural overview, see [design.md](design.md).

### Implementation Status (Development Focus)

📖 **For detailed component specifications, see [design.md](design.md)**

**✅ Implemented Services:**
- **Log Ingestion Service** (`cmd/ingestion/`) - HTTP/gRPC endpoints with SHA256 hashing and Kafka integration
- **Blockchain Processing Service** (`cmd/engine/`) - Multi-worker Kafka consumer with ChainMaker integration
- **Supporting Infrastructure** - PostgreSQL state database, Kafka message queue, ChainMaker blockchain client

**❌ TODO Components:**
- **API Gateway** - Traefik/Nginx for TLS termination and unified authentication
- **Benthos Adapters** - Direct protocol reception (S3, Syslog, Kafka) with security controls
- **Query Layer** - Multi-dimensional query APIs for different user types

### Project Structure

```
tlng/
├── cmd/                     # Service entry points
│   ├── ingestion/          # ✅ Log Ingestion Service
│   └── engine/             # ✅ Blockchain Processing Service
├── ingestion/              # ✅ Ingestion Layer implementation
│   ├── service/            # Core business logic
│   ├── http/               # HTTP handlers
│   └── grpc/               # gRPC servers
├── processing/             # ✅ Processing Layer implementation
│   └── worker/             # Blockchain workers
├── blockchain/             # ✅ Blockchain Layer
│   └── client/             # ChainMaker SDK wrapper
├── storage/                # ✅ Storage Layer
│   ├── store/              # PostgreSQL interfaces
│   └── migrations/         # Database schema
├── internal/               # Shared utilities
│   ├── messaging/          # Kafka interfaces
│   └── models/             # Data models
├── config/                 # Configuration files
├── proto/                  # gRPC definitions
└── ❌ TODO directories:
    ├── query/               # Query Layer APIs
    ├── ingress/             # API Gateway configs
    └── adapters/            # Benthos configs
```

📖 **For detailed component responsibilities, see [design.md](design.md)**

## Development Commands

### Infrastructure Setup
```bash
# Start dependencies (Kafka, PostgreSQL, Zookeeper)
docker-compose up -d

# Wait for services to be ready, then create Kafka topic
docker-compose exec kafka kafka-topics --create --bootstrap-server localhost:9092 --partitions 6 --replication-factor 1 --topic log_submissions
```

### Building and Running
```bash
# Build Log Ingestion Service
go build -o bin/ingestion ./cmd/ingestion

# Build Attestation Engine
go build -o bin/engine ./cmd/engine

# Run Log Ingestion Service
./bin/ingestion

# Run Attestation Engine
./bin/engine
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests for specific package
go test ./internal/engine/worker

# Run with coverage
go test -cover ./...
```

### Configuration
- Log Ingestion Service config: `./config/ingestion.defaults.yml`
- Blockchain Processing Service config: `./config/engine.defaults.yml`
- Blockchain client config: `./config/blockchain.defaults.yml`

## Key Components

### Log Ingestion Service
- HTTP endpoint: `POST /v1/logs`
- gRPC service: `LogIngestion.SubmitLog`
- Computes SHA256 log_hash and generates UUID request_id
- Writes initial status 'RECEIVED' to State DB
- Pushes messages to Kafka for async processing
- Returns request_id to caller immediately

### Blockchain Processing Service
- Multiple consumers/workers for parallel processing
- Consumes messages from Kafka and updates status to 'PROCESSING'
- Calls Blockchain.SubmitLog(logContent) via ChainMaker SDK
- Updates final status 'COMPLETED'/'FAILED' with transaction details
- Retry mechanism with configurable limits

### Key Components (Development Focus)

📖 **For detailed message flow and protocols, see [design.md](design.md)**

**Log Ingestion Service** (`cmd/ingestion/`):
- HTTP endpoint: `POST /v1/logs`
- gRPC service: `LogIngestion.SubmitLog`
- Computes SHA256 log_hash, generates UUID request_id
- Writes 'RECEIVED' status to State DB
- Pushes to Kafka for async processing
- Returns request_id immediately

**Blockchain Processing Service** (`cmd/engine/`):
- Multi-worker Kafka consumer
- Updates status to 'PROCESSING'
- Calls Blockchain.SubmitLog(logContent)
- Updates final status ('COMPLETED'/'FAILED') with transaction details
- Configurable retry mechanism

### Development Guidelines

**Database Schema:**
📖 **For complete schema details, see [design.md](design.md)**

Core table: `Tbl_Log_Status` tracks log lifecycle from submission to blockchain confirmation.

**Configuration Management:**
- All services use YAML configuration files
- Environment-specific overrides supported
- Sensitive data via environment variables

**Development Best Practices:**
- Structured logging with component prefixes
- Graceful shutdown handling
- Connection pooling for database
- Batch processing for efficiency

## Development Standards

### Code Requirements
- **Language**: All code, comments, and documentation in English only
- **Logging**: Structured logging with component prefixes
- **Configuration**: YAML files with environment overrides
- **Error Handling**: Graceful degradation and retry mechanisms

### Technical Standards
- **Database**: Connection pooling, transaction management
- **Messaging**: Kafka consumer groups, batch processing
- **Blockchain**: Efficient contract calls, transaction batching
- **Testing**: Unit tests with coverage, integration tests for critical paths

### Current Implementation Status

**✅ Completed:**
- Log Ingestion Service (HTTP/gRPC, SHA256 hashing, Kafka integration)
- Blockchain Processing Service (multi-worker, ChainMaker integration)
- State Database (PostgreSQL with lifecycle tracking)
- Supporting Infrastructure (Kafka, configuration management)

**📋 TODO Components:**
📖 **For detailed specifications and priorities, see [design.md](design.md)**
- API Gateway with unified authentication
- Benthos adapters for heterogeneous protocols
- Query Layer with multi-dimensional APIs