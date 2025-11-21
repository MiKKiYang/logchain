# Query Layer

## Overview

The Query Layer provides multi-dimensional query services for different user types as specified in the design document.

## Planned APIs

Based on `../docs/design.md`, this layer will implement three distinct APIs:

### API 1: Task Status Query (for API Callers)
- **Endpoint**: `GET /status/{request_id}`
- **Authentication**: API Key
- **Purpose**: Allows "active push" clients to check on-chain status using the returned `request_id`
- **Scope**: Can only query logs submitted by themselves

### API 2: Content Reverse Lookup (for Non-API Users)
- **Endpoint**: `POST /query_by_content`
- **Authentication**: API Key
- **Purpose**: Allows "passive access" (Syslog/Kafka) users to find on-chain credentials using original log content
- **Request Body**: `{"log_content": "your raw log string"}`
- **Scope**: Can only query logs from their own systems

### API 3: On-chain Public Audit (for Alliance Members)
- **Endpoints**:
  - `GET /log/by_tx/{tx_hash}`
  - `GET /log/{on_chain_log_id}`
- **Authentication**: mTLS + IP whitelist
- **Purpose**: Satisfies "transparent notarization" business requirements for alliance member auditing
- **Scope**: Can audit all on-chain log data

## Implementation Status

🚧 **TODO**: Not yet implemented

This directory is prepared for future implementation of the Query Layer services.

## Architecture Notes

- Will query both State DB (for status) and blockchain (for content)
- Implements multi-level authentication and authorization
- Provides audit logging for all query operations
- Follows the principle of least privilege for each user type