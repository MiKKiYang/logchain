# Attestation Engine Service

The Attestation Engine service consumes log submissions from Kafka and processes them for blockchain attestation. It maintains multiple workers for parallel processing and handles the complete lifecycle of log attestation.

## Quick Start Guide

### 1. Start Docker Compose Environment

```bash
docker compose up -d
```

This will start PostgreSQL, Kafka, and Zookeeper services.

### 2. Prepare Test Data (Required for Standalone Testing)

Since the engine needs log messages in Kafka to process, we'll create test data:

#### Option A: Insert Test Records and Send to Kafka

```bash
# Insert test records into database
docker compose exec postgres psql -U testuser -d testdb -f /scripts/test-data.sql

# Install kcat if not available
# Ubuntu/Debian: sudo apt-get update && sudo apt-get install kafkacat
# MacOS: brew install kcat

# Send mock log messages to Kafka
echo '{"RequestID":"a1b1c1d1-e1f1-1111-2222-1234567890ab","LogContent":"Fixed mock log content 1","LogHash":"fixedhash001","SourceOrgID":"mock-org-1","ReceivedTimestamp":"1761018328"}' | kcat -P -b localhost:9092 -t log_submissions

echo '{"RequestID":"a2b2c2d2-e2f2-3333-4444-abcdef123456","LogContent":"Fixed mock log content 2 with more detail","LogHash":"fixedhash002","SourceOrgID":"mock-org-2","ReceivedTimestamp":"1761018358"}' | kcat -P -b localhost:9092 -t log_submissions

echo '{"RequestID":"a3b3c3d3-e3f3-5555-6666-fedcba654321","LogContent":"Fixed mock log content 1","LogHash":"fixedhash001","SourceOrgID":"mock-org-1","ReceivedTimestamp":"1761018388"}' | kcat -P -b localhost:9092 -t log_submissions
```

#### Option B: Use API Gateway to Generate Test Data

```bash
# In another terminal, start API Gateway
go run ./cmd/ingestion/main.go

# Send test logs via HTTP
curl -X POST http://localhost:8080/v1/logs \
-H "Content-Type: application/json" \
-d '{
  "log_content": "Test log for engine processing",
  "client_source_org_id": "engine-test-org"
}'
```

### 3. Start the Attestation Engine

```bash
# Build and run
go build -o bin/engine ./cmd/engine
./bin/engine

# Or run directly
go run ./cmd/engine/main.go
```

### 4. Monitor Processing

```bash
# Check database status updates
docker compose exec postgres psql -U testuser -d testdb -c "
SELECT request_id, status, processing_started_at, processing_finished_at, tx_hash, error_message
FROM tbl_log_status
ORDER BY received_at_db DESC
LIMIT 10;
"

# Check Kafka topic messages (consume remaining messages)
kcat -C -b localhost:9092 -t log_submissions -o beginning -e
```

## Testing Scenarios

### Test Normal Processing
1. Send logs to Kafka (or via API Gateway)
2. Start engine
3. Monitor status transitions: RECEIVED → PROCESSING → COMPLETED
4. Check blockchain transaction hash and block height

### Test Error Handling
1. Send malformed JSON messages to Kafka
2. Observe FAILED status with error messages
3. Check retry count for failed messages

### Test Retry Mechanism
1. Configure blockchain to temporarily fail
2. Observe retry count increments
3. Verify eventual success or final failure

## Configuration

The engine configuration is loaded from `./config/engine.defaults.yml` or custom config files:

- **Kafka Settings**: Bootstrap servers, topic name, consumer group
- **Database Settings**: Connection string, pool size
- **Worker Settings**: Number of concurrent workers
- **Blockchain Settings**: ChainMaker connection, contract address
- **Retry Settings**: Max retry attempts, backoff intervals

## Troubleshooting

### Common Issues

1. **No messages processed**: Check Kafka has messages and consumer group is healthy
2. **Database connection errors**: Verify PostgreSQL is running and credentials are correct
3. **Blockchain connection failures**: Check ChainMaker network connectivity and contract deployment
4. **Processing stalls**: Check for error messages in logs and verify worker count

### Useful Commands

```bash
# Check Kafka consumer group status
docker compose exec kafka kafka-consumer-groups \
  --bootstrap-server localhost:9092 \
  --describe --group engine-consumer-group

# Check database connections
docker compose exec postgres psql -U testuser -d testdb -c "SELECT count(*) FROM tbl_log_status;"

# View engine logs (if running in background)
# Logs will show processing status, blockchain transactions, and any errors

# Reset test data
docker compose exec postgres psql -U testuser -d testdb -c "
DELETE FROM tbl_log_status
WHERE source_org_id IN ('mock-org-1', 'mock-org-2', 'engine-test-org');
"
```

## Notes

- The database schema is automatically created on container startup
- Test data insertion is only needed for standalone engine testing
- The engine processes messages in batches for optimal blockchain performance
- All timestamps are stored in UTC
- Failed messages are retried with exponential backoff