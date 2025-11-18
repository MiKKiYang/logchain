-- Test data for Attestation Engine testing
-- This script inserts sample log entries with 'RECEIVED' status for engine processing

INSERT INTO tbl_log_status (request_id, log_hash, source_org_id, received_timestamp, status, retry_count) VALUES
('a1b1c1d1-e1f1-1111-2222-1234567890ab', 'fixedhash001', 'mock-org-1', NOW() - interval '60 second', 'RECEIVED', 0),
('a2b2c2d2-e2f2-3333-4444-abcdef123456', 'fixedhash002', 'mock-org-2', NOW() - interval '30 second', 'RECEIVED', 0),
('a3b3c3d3-e3f3-5555-6666-fedcba654321', 'fixedhash001', 'mock-org-1', NOW(), 'RECEIVED', 0);