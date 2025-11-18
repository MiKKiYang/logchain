# Log Ingestion Service

The Log Ingestion Service provides endpoints for log submission to the Trusted Log Attestation System. It supports both HTTP REST and gRPC interfaces for log ingestion as part of the Ingestion Layer.

## Quick Start Guide

### 1. Start Docker Compose Environment

```bash
docker compose up -d
```

This will start:
- PostgreSQL with automatic schema initialization
- Kafka with `log_submissions` topic
- Zookeeper

## 2. Verify Services are Ready

```bash
# Check PostgreSQL connection
docker compose exec postgres psql -U testuser -d testdb -c "SELECT 1;"

# Check Kafka topic creation
docker compose exec kafka kafka-topics --list --bootstrap-server localhost:9092
```

## 3. Start Log Ingestion Service

```bash
# Build and run
go build -o bin/ingestion ./cmd/ingestion
./bin/ingestion

# Or run directly
go run ./cmd/ingestion/main.go
```

The Log Ingestion Service should start on port 8080 (default).

## 4. Submit Log via HTTP POST

```bash
curl -X POST http://localhost:8080/v1/logs \
-H "Content-Type: application/json" \
-d '{
  "log_content": "This is a live test log from curl!",
  "client_source_org_id": "curl-test-org"
}'
```

Expected response:
```json
{
  "request_id": "uuid-string",
  "status": "RECEIVED"
}
```

## 5. Submit Log via gRPC

```bash
# Using grpcurl (if installed)
grpcurl -plaintext -d '{
  "log_content": "Test log via gRPC",
  "client_source_org_id": "grpc-test-org"
}' localhost:9090 logingestion.LogIngestion/SubmitLog
```

## 6. Verify Database State

```bash
# Connect to PostgreSQL
docker compose exec postgres psql -U testuser -d testdb

# Check the submitted log
SELECT request_id, log_hash, source_org_id, status, received_at_db
FROM tbl_log_status
WHERE request_id = 'your-request-id';

# Check all recent logs
SELECT request_id, status, received_at_db
FROM tbl_log_status
ORDER BY received_at_db DESC
LIMIT 5;
```

## 7. Check Kafka Messages

```bash
# Using kcat (modern kafka cat)
kcat -C -b localhost:9092 -t log_submissions -o beginning -e

# Or using kafka console consumer
docker compose exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic log_submissions \
  --from-beginning \
  --max-messages 5
```

## 8. Test Error Cases

### Invalid JSON
```bash
curl -X POST http://localhost:8080/v1/logs \
-H "Content-Type: application/json" \
-d '{"invalid": json}'
```

### Missing Required Fields
```bash
curl -X POST http://localhost:8080/v1/logs \
-H "Content-Type: application/json" \
-d '{"client_source_org_id": "test"}'
```

## 9. Health Check (if implemented)

```bash
curl http://localhost:8080/health
```

## 10. Cleanup Test Data

```bash
# Connect to PostgreSQL and clear test data
docker compose exec postgres psql -U testuser -d testdb -c "
DELETE FROM tbl_log_status
WHERE source_org_id IN ('curl-test-org', 'grpc-test-org');
"
```

## Troubleshooting

### Common Issues

1. **Port conflicts**: Make sure ports 8080, 8090, 5433, and 9092 are available
2. **Database connection errors**: Verify PostgreSQL container is healthy
3. **Kafka connection errors**: Check Kafka topic exists and broker is accessible
4. **gRPC connection**: Make sure port 8090 is open and gRPC server is running

### Useful Commands

```bash
# Check container status
docker compose ps

# View Log Ingestion Service logs
docker compose logs -f ingestion  # if running in container
# Or just check the console output if running locally

# Reset everything
docker compose down -v
docker compose up -d
```

## Next Steps

After testing the Log Ingestion Service:

1. Start the Attestation Engine to process the submitted logs
2. Monitor the status transitions in the database (RECEIVED → PROCESSING → COMPLETED/FAILED)
3. Check blockchain transaction logs for successful notarizations
4. Once implemented, test the separate Query & Audit Layer service

## Notes

- The database schema is automatically created on container startup
- Kafka topic `log_submissions` is created automatically by the init container
- All timestamps are stored in UTC
- Request IDs are generated as UUIDs