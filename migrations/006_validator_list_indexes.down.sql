-- Drop indexes created in 006_validator_list_indexes.up.sql
DROP INDEX IF EXISTS idx_validators_status_index;
DROP INDEX IF EXISTS idx_validators_pubkey_prefix;
DROP INDEX IF EXISTS idx_validators_index_search;
DROP INDEX IF EXISTS idx_snapshots_validator_created;
DROP INDEX IF EXISTS idx_snapshots_effectiveness;
DROP INDEX IF EXISTS idx_snapshots_balance;
DROP INDEX IF EXISTS idx_validators_status_effectiveness;
