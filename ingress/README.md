# Ingress Layer

## Overview

The Ingress Layer serves as the traffic entry point and routing layer for the entire trusted log notarization system.

## Responsibilities

Based on `../docs/design.md`, this layer handles:

### 1. TLS Termination
- Processes all HTTPS requests
- Decrypts traffic so internal services don't need to handle SSL/TLS certificates
- Ensures secure communication channels

### 2. Protocol Routing
- **HTTP/gRPC Route**: `POST /v1/logs` or `gRPC SubmitLog` → Log Ingestion Service
- **Heterogeneous Protocol Route**: Syslog (UDP 514), Kafka, S3 → Benthos Adapters → Log Ingestion Service
- **Query Route**: `GET /status/...`, `GET /log/by_tx/...`, etc. → Query Layer

### 3. Load Balancing
- Distributes incoming traffic across available service instances
- Ensures high availability and performance
- Supports horizontal scaling

## Planned Implementation

🚧 **TODO**: Not yet implemented

This directory is prepared for future implementation of:

### API Gateway Component
- **Traefik** or **Nginx Ingress** configuration
- TLS certificate management
- Protocol-specific routing rules
- Load balancing policies
- Rate limiting and DDoS protection

### Integration Points
- Routes to `ingestion/` layer for log submissions
- Routes to `query/` layer for audit operations
- Routes to `benthos/` adapters for protocol conversion (future)

## Configuration Notes

- Will need certificates for TLS termination
- Requires routing rules for different protocols
- Should integrate with authentication mechanisms
- Must handle both synchronous and asynchronous traffic patterns

## Security Considerations

- TLS certificate validation and renewal
- Protocol-specific security policies
- IP-based access controls for alliance members
- Request validation and sanitization