-- Drop triggers
DROP TRIGGER IF EXISTS update_alerts_updated_at ON alerts;
DROP TRIGGER IF EXISTS update_validators_updated_at ON validators;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables (cascade will handle foreign keys and indexes)
DROP TABLE IF EXISTS alerts CASCADE;
DROP TABLE IF EXISTS validator_snapshots CASCADE;
DROP TABLE IF EXISTS validators CASCADE;

-- Note: We don't drop extensions as they might be used by other schemas