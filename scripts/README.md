# Scripts Directory

This directory contains various scripts for the TLNG project.

## Directory Structure

- `db/` - Database initialization and migration scripts
- `migration/` - Database migration scripts for version upgrades
- `data/` - Sample data or data transformation scripts

## Database Scripts

### `db/init-db.sql`
Initial database schema creation script. This script:
- Creates the `tbl_log_status` table if it doesn't exist
- Creates necessary indexes for optimal query performance
- Uses `IF NOT EXISTS` clauses to allow safe re-execution

### Usage
This script is automatically mounted and executed by PostgreSQL container startup via docker-compose.