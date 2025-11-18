# Ingestion Layer

This layer contains all components responsible for log ingestion and protocol adaptation in the Trusted Log Attestation System.

## Architecture

According to the design document, this layer consists of two main components:

### 1. Benthos Adapters (`adapters/`)

🚧 **TODO**: Future implementation

These adapters handle heterogeneous protocol conversion:
- **Syslog** - UDP/TCP syslog protocol (port 514)
- **Kafka Topics** - Direct Kafka topic consumption
- **S3** - AWS S3 bucket file processing
- **Other protocols** - Any future heterogeneous data sources

**Workflow:**
1. Receive heterogeneous protocol traffic from API Gateway
2. Parse and standardize data formats
3. Forward processed logs to the Log Ingestion Service

### 2. Log Ingestion Service (`service/`)

✅ **Implemented**

This is the unified processing entry point for all log submissions.

**Components:**
- `http/` - HTTP REST API handlers (`POST /v1/logs`)
- `grpc/` - gRPC service implementations (`LogIngestion.SubmitLog`)
- `core/` - Core business logic and batch processing

**Key Workflows:**
1. **Log Reception**: Receive logs from HTTP/gRPC clients or Benthos adapters
2. **Hash Generation**: Compute SHA256 hash and generate UUID request_id
3. **Immediate Response**: Return request_id to caller immediately
4. **Async Processing**: Batch write to State DB and push to Kafka queue

## Import Paths

```go
// For Log Ingestion Service components
import "tlng/ingestion/service/[component]"

// For core business logic
import core "tlng/ingestion/service/core"

// For future Benthos adapters
import "tlng/ingestion/adapters/[adapter]"
```

## Configuration

Configuration is managed through:
- `../config/ingestion.defaults.yml` - Main ingestion service configuration
- Adapter configurations will be added here when implemented

## API Endpoints

### HTTP Endpoints
- `POST /v1/logs` - Log submission
- `GET /health` - Health check
- `GET /metrics` - Basic metrics

### gRPC Services
- `LogIngestion.SubmitLog` - Log submission

## Message Flow

1. **Direct Submission**: Client → HTTP/gRPC → Log Ingestion Service
2. **Adapter Flow**: Client → API Gateway → Benthos Adapter → Log Ingestion Service
3. **Processing**: Log Ingestion Service → State DB + Kafka → Processing Layer

## Development Status

- ✅ Log Ingestion Service (HTTP/gRPC APIs)
- ✅ Core service logic and batch processing
- ✅ Integration with Kafka and State DB
- 🚧 Benthos Adapters (TODO)

## Testing

```bash
# Build the ingestion service
go build -o bin/ingestion-service ./cmd/ingestion

# Run tests
go test ./ingestion/...

# Run specific component tests
go test ./ingestion/service/...
```