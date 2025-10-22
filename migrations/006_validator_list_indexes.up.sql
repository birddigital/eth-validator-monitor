-- Composite index for common query patterns
-- Supports filtering by status and sorting
CREATE INDEX IF NOT EXISTS idx_validators_status_index
ON validators (status, validator_index)
WHERE status IN ('active_ongoing', 'pending_initialized', 'exited_slashed', 'exited_unslashed', 'withdrawal_possible', 'withdrawal_done');

-- Index for search by pubkey prefix
-- Uses text_pattern_ops for efficient LIKE queries with prefix matching
CREATE INDEX IF NOT EXISTS idx_validators_pubkey_prefix
ON validators (pubkey text_pattern_ops);

-- Index for search by validator index
CREATE INDEX IF NOT EXISTS idx_validators_index_search
ON validators (validator_index);

-- Index for latest snapshots per validator
-- Supports getting the most recent snapshot efficiently
CREATE INDEX IF NOT EXISTS idx_snapshots_validator_created
ON validator_snapshots (validator_index, created_at DESC);

-- Index for sorting by effectiveness
CREATE INDEX IF NOT EXISTS idx_snapshots_effectiveness
ON validator_snapshots (attestation_effectiveness DESC);

-- Index for sorting by balance
CREATE INDEX IF NOT EXISTS idx_snapshots_balance
ON validator_snapshots (balance DESC);

-- Composite index for validator + status filtering
CREATE INDEX IF NOT EXISTS idx_validators_status_effectiveness
ON validators (status, validator_index)
INCLUDE (pubkey, is_slashed, activation_epoch, exit_epoch);

-- Performance analysis note:
-- These indexes support the following query patterns:
-- 1. Filter by status + sort by index/effectiveness/balance
-- 2. Search by pubkey prefix (LIKE '0x123%')
-- 3. Search by exact validator index
-- 4. Get latest snapshot per validator efficiently
-- 5. Pagination with OFFSET/LIMIT
