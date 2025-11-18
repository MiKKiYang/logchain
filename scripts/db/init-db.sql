-- PostgreSQL initialization script for TLNG project
-- Creates tbl_log_status table and indexes if they don't exist

-- Create the main table
CREATE TABLE IF NOT EXISTS tbl_log_status (
    request_id TEXT PRIMARY KEY,
    log_hash TEXT NOT NULL,
    source_org_id TEXT,
    received_timestamp TIMESTAMPTZ,
    status VARCHAR(20) NOT NULL DEFAULT 'RECEIVED',
    received_at_db TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processing_started_at TIMESTAMPTZ,
    processing_finished_at TIMESTAMPTZ,
    tx_hash TEXT,
    block_height BIGINT,
    log_hash_on_chain TEXT,
    error_message TEXT,
    retry_count INTEGER NOT NULL DEFAULT 0
);

-- 1. log_hash index - needed for future content reverse lookup queries
CREATE INDEX IF NOT EXISTS idx_log_status_log_hash ON tbl_log_status (log_hash);

-- 2. tx_hash index - needed for future blockchain audit queries
CREATE INDEX IF NOT EXISTS idx_log_status_tx_hash ON tbl_log_status (tx_hash) WHERE tx_hash IS NOT NULL;

-- 3. Optimized composite index for status + time-based operations
-- This replaces the old partial index and is more flexible
CREATE INDEX IF NOT EXISTS idx_log_status_status_time ON tbl_log_status (status, received_at_db);