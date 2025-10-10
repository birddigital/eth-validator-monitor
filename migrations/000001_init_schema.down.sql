-- Drop tables in reverse order to handle foreign key dependencies
DROP TABLE IF EXISTS network_stats;
DROP TABLE IF EXISTS validator_performance;
DROP TABLE IF EXISTS alerts;
DROP TABLE IF EXISTS validator_snapshots;
DROP TABLE IF EXISTS validators;
