# Database Migrations

## Overview

This directory contains database migration scripts for the State Database (PostgreSQL).

## Current Schema

Based on the design document, the main table is:

### Tbl_Log_Status
Tracks the complete lifecycle status of log notarization tasks.

**Columns:**
- `request_id` (PK) - Internal tracking ID
- `log_hash` (Indexed) - Content fingerprint for reverse queries
- `status` (Enum) - RECEIVED, PROCESSING, COMPLETED, FAILED
- `tx_hash` - Blockchain transaction hash
- `on_chain_log_id` - Contract-returned on-chain ID
- `block_height` - Block number
- `error_message` - Failure details

## Migration Strategy

🚧 **TODO**: Migration framework to be implemented

Consider using:
- **golang-migrate** for Go-based migrations
- **Flyway** or **Liquibase** for Java-based alternatives
- Custom migration scripts with versioning

## Database Setup

Current setup uses Docker Compose with PostgreSQL. See:
- `docker-compose.yml` for database service
- `scripts/db/init-db.sql` for initial schema
- `scripts/db/test-data.sql` for test data

## Notes

- All timestamps should use UTC
- Index on `log_hash` is critical for reverse lookup performance
- Consider partitioning by date for large-scale deployments
- Backup strategies should be implemented for production