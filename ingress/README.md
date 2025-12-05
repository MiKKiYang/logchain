# Ingress Layer - API Gateway

## Overview

The Ingress Layer serves as the traffic entry point and routing layer for the entire trusted log attestation system. It implements a comprehensive API Gateway using **Nginx with OpenResty** (for Lua support), providing TLS termination, unified authentication, protocol routing, and load balancing.

## Responsibilities

Based on `../docs/design.md`, this layer handles:

### 1. TLS Termination
- Processes all HTTPS requests
- Decrypts traffic so internal services don't need to handle SSL/TLS certificates
- Ensures secure communication channels
- Supports HTTP/2

### 2. Unified Authentication
- **API Key Authentication**: 
  - Validates API client identity for log submission and query operations
  - Supports file-based, Redis-based, or external service-based validation
  - Passes client identity to backend services via HTTP headers
- **mTLS + IP Whitelist Authentication**: Dual authentication for consortium members
  - Validates client certificates signed by consortium CA
  - Verifies IP addresses against whitelist
  - Enables on-chain data audit access

### 3. Protocol Routing
- **HTTP/gRPC Routes**: 
  - `POST /v1/logs` → Log Ingestion Service (API Key auth)
  - `gRPC SubmitLog` → Log Ingestion Service (API Key auth)
- **Query Routes**:
  - `GET /status/{request_id}` → Query Service (API Key auth)
  - `POST /query_by_content` → Query Service (API Key auth)
  - `GET /log/by_tx/{tx_hash}` → Query Service (mTLS + IP whitelist)
  - `GET /log/{on_chain_log_id}` → Query Service (mTLS + IP whitelist)

### 4. Load Balancing
- Distributes incoming traffic across available service instances
- Uses least-connection algorithm for optimal distribution
- Supports health checks and automatic failover
- Ensures high availability and performance
- Supports horizontal scaling of backend services
- Supports horizontal scaling of the nginx instance itself 

### 5. Rate Limiting
- API submission: 100 requests/second (burst: 20)
- Query operations: 50 requests/second (burst: 10)
- Audit operations: 20 requests/second (burst: 5)

### 6. Audit Logging
- Records all authentication events (success/failure)
- Logs critical operations with detailed context
- Provides audit trail for security compliance

## Quick Start

### Prerequisites

- Docker and Docker Compose
- OpenSSL (for certificate generation)

### 1. Generate SSL Certificates

```bash
cd ingress
./scripts/generate-ssl-certs.sh
```

This will create:
- Server certificate and key (`ssl/cert.pem`, `ssl/key.pem`)
- CA certificate for mTLS (`ssl/ca-cert.pem`)

### 2. Setup Configuration Files

```bash
./scripts/setup-config.sh
```

This will create configuration files from examples:
- `nginx/conf.d/api-keys.json`
- `nginx/conf.d/consortium-ip-whitelist.json`

**Important**: Update these files with your actual API keys and IP whitelists.

### 3. Validate Configuration

```bash
./scripts/validate-config.sh
```

This validates:
- Nginx configuration syntax
- JSON configuration files
- SSL certificates

### 4. Start the API Gateway

```bash
docker-compose up -d
```

The API Gateway will be available at:
- HTTP: `http://localhost:80` (redirects to HTTPS)
- HTTPS: `https://localhost:443`
- gRPC: `localhost:50052`

### 5. Verify Health

```bash
curl http://localhost/health
# Should return: healthy
```

## Configuration

### API Key Authentication

API keys are stored in `nginx/conf.d/api-keys.json`:

```json
{
  "your-api-key-here": {
    "client_id": "client-001",
    "org_id": "org-abc",
    "status": "active",
    "permissions": ["submit_log", "query_status", "query_by_content"],
    "created_at": "2024-01-01T00:00:00Z",
    "expires_at": "2026-01-01T00:00:00Z"
  }
}
```

**Authentication Methods** (configured via `API_KEY_AUTH_METHOD` environment variable):
- `file`: File-based (default, for development)
- `redis`: Redis-based (for production)
- `service`: External auth service (for centralized management)

### mTLS + IP Whitelist Authentication

For consortium members, configure:

**IP Whitelist** (`nginx/conf.d/consortium-ip-whitelist.json`):
```json
{
  "members": {
    "member-001": {
      "name": "Regulatory Authority A",
      "ip_whitelist": [
        "192.168.1.0",
        "10.0.0.100"
      ]
    }
  }
}
```

**Generate Client Certificates**:
```bash
./scripts/generate-client-cert.sh member-001 "Regulatory Authority A"
```
- Client certificate for mTLS (`ssl/clients/member-001/client-cert.pem`, `ssl/clients/member-001/client-key.pem`, `ssl/clients/member-001/client.p12`)

### Backend Services

Configure backend service addresses in `nginx/nginx.conf`:

```nginx
upstream ingestion_http {
    least_conn;
    server ingestion-service:8091 max_fails=3 fail_timeout=30s;
    # Add more instances for load balancing:
    # server ingestion-service-2:8091 max_fails=3 fail_timeout=30s;
    keepalive 32;
}
```

## API Endpoints

### Log Submission (API Key Authentication)

**HTTP:**
```bash
curl -X POST https://api-gateway/v1/logs \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"log_content": "your log content here"}'
```

**gRPC:**
```bash
grpcurl -plaintext \
  -H "X-API-Key: your-api-key" \
  -d '{"log_content": "your log content here"}' \
  api-gateway:50052 \
  logingestion.LogIngestion/SubmitLog
```

### Query Operations (API Key Authentication)

**Status Query:**
```bash
curl https://api-gateway/status/{request_id} \
  -H "X-API-Key: your-api-key"
```

**Content-based Query:**
```bash
curl -X POST https://api-gateway/query_by_content \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"log_content": "your log content here"}'
```

### Audit Operations (mTLS + IP Whitelist)

**Query by Transaction Hash:**
```bash
curl https://api-gateway/log/by_tx/{tx_hash} \
  --cert ssl/clients/member-001/client-cert.pem \
  --key ssl/clients/member-001/client-key.pem \
  --cacert ssl/ca-cert.pem
```

**Query by On-Chain Log ID:**
```bash
curl https://api-gateway/log/{on_chain_log_id} \
  --cert ssl/clients/member-001/client-cert.pem \
  --key ssl/clients/member-001/client-key.pem \
  --cacert ssl/ca-cert.pem
```
## Production Deployment

### 1. Replace Self-Signed Certificates

Replace the generated self-signed certificates with certificates from a trusted CA:

```bash
# Copy your certificates
cp /path/to/your/cert.pem ssl/cert.pem
cp /path/to/your/key.pem ssl/key.pem
cp /path/to/your/ca-cert.pem ssl/ca-cert.pem

# Set proper permissions
chmod 600 ssl/key.pem
chmod 644 ssl/cert.pem ssl/ca-cert.pem
```
### 2. Configure API Keys

Update `nginx/conf.d/api-keys.json` with actual API keys. Consider:
- Using Redis or external auth service for centralized management

### 3. Configure IP Whitelists

Update `nginx/conf.d/consortium-ip-whitelist.json` with actual consortium member IPs.

### 4. Enable Monitoring

Monitor:
- Access logs: `logs/access.log`
- Error logs: `logs/error.log`
- Audit logs: `logs/audit.log`

### 5. Set Environment Variables

```bash
# In docker-compose.yml or .env file
API_KEY_AUTH_METHOD=redis  # or "service" for external auth
REDIS_HOST=redis
REDIS_PORT=6379
AUTH_SERVICE_URL=http://auth-service:8080/validate
```

## Troubleshooting

### Check Nginx Configuration

```bash
docker exec nginx-api-gateway nginx -t
```

### View Logs

```bash
# Access logs
docker exec nginx-api-gateway tail -f /var/log/nginx/access.log

# Error logs
docker exec nginx-api-gateway tail -f /var/log/nginx/error.log

# Audit logs
docker exec nginx-api-gateway tail -f /var/log/nginx/audit.log
```

### Reload Configuration

```bash
docker exec nginx-api-gateway nginx -s reload
```

## Integration with Other Services

The API Gateway integrates with:

- **Log Ingestion Service**: Receives log submissions via HTTP/gRPC
- **Query Service**: Routes query requests for status and content-based queries
- **Redis** (optional): For API key storage if using Redis auth method
- **Auth Service** (optional): External authentication service

Ensure these services are on the same Docker network if you prepare use docker-compose (`logchain-network`).
