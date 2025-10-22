-- Rollback: Remove source field and indexes from alerts table
-- Created: 2025-10-22

BEGIN;

-- Drop indexes
DROP INDEX IF EXISTS idx_alerts_validator_index_created_at;
DROP INDEX IF EXISTS idx_alerts_status_created_at;
DROP INDEX IF EXISTS idx_alerts_created_at_desc;
DROP INDEX IF EXISTS idx_alerts_status_severity;
DROP INDEX IF EXISTS idx_alerts_severity;
DROP INDEX IF EXISTS idx_alerts_status;

-- Remove source column
ALTER TABLE alerts
DROP COLUMN IF EXISTS source;

COMMIT;
