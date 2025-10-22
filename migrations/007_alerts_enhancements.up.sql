-- Migration: Add source field and indexes to alerts table for alerts management page
-- Created: 2025-10-22

BEGIN;

-- Add source column to alerts table
ALTER TABLE alerts
ADD COLUMN IF NOT EXISTS source VARCHAR(255) NOT NULL DEFAULT 'system';

-- Add comment explaining the source field
COMMENT ON COLUMN alerts.source IS 'Source of the alert (e.g., validator_collector, manual, external_api)';

-- Create indexes for efficient filtering on alerts management page
-- Index on status for filtering by alert status (new, read, dismissed, etc.)
CREATE INDEX IF NOT EXISTS idx_alerts_status
ON alerts(status);

-- Index on severity for filtering by severity level (info, warning, error, critical)
CREATE INDEX IF NOT EXISTS idx_alerts_severity
ON alerts(severity);

-- Composite index for combined status + severity filtering (common query pattern)
CREATE INDEX IF NOT EXISTS idx_alerts_status_severity
ON alerts(status, severity);

-- Composite index for pagination with created_at DESC (most recent first)
CREATE INDEX IF NOT EXISTS idx_alerts_created_at_desc
ON alerts(created_at DESC);

-- Composite index for filtering by status with pagination
CREATE INDEX IF NOT EXISTS idx_alerts_status_created_at
ON alerts(status, created_at DESC);

-- Composite index for filtering by validator_index (for validator-specific alerts)
CREATE INDEX IF NOT EXISTS idx_alerts_validator_index_created_at
ON alerts(validator_index, created_at DESC)
WHERE validator_index IS NOT NULL;

COMMIT;
