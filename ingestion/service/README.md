# Log Ingestion Service

This is the unified processing entry point for all log submissions in the Trusted Log Attestation System.

## Overview

The Log Ingestion Service handles:
- Direct client submissions via HTTP/gRPC APIs
- Standardized logs from Benthos adapters
- SHA256 hash computation and UUID request_id generation
- Asynchronous batch processing to State DB and Kafka

## Architecture

**Components:**
- `http/` - HTTP REST API handlers
- `grpc/` - gRPC service implementations
- `core/` - Core business logic and batch processing

## Key Workflows

1. **Log Reception**: Receive logs from HTTP/gRPC clients or Benthos adapters
2. **Hash Generation**: Compute SHA256 hash and generate UUID request_id
3. **Immediate Response**: Return request_id to caller immediately
4. **Async Processing**: Batch write to State DB and push to Kafka queue

## API Endpoints

- `POST /v1/logs` - HTTP endpoint for log submission
- `LogIngestion.SubmitLog` - gRPC service for log submission

## Import Path

Services in this directory use the import path:
```go
import "tlng/ingestion/service/[component]"

// For core business logic
import core "tlng/ingestion/service/core"
```

## Configuration

Configuration is managed through the main ingestion service config:
- See `../config/ingestion.defaults.yml`