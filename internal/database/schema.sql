-- Ethereum Validator Monitor Database Schema
-- Using TimescaleDB for efficient time-series data storage

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- =====================================================
-- VALIDATORS TABLE
-- =====================================================
-- Stores validator metadata and current state
CREATE TABLE IF NOT EXISTS validators (
    id SERIAL PRIMARY KEY,
    validator_index BIGINT UNIQUE NOT NULL,
    pubkey VARCHAR(98) UNIQUE NOT NULL, -- 0x-prefixed hex string
    withdrawal_credentials VARCHAR(66),
    effective_balance BIGINT,
    slashed BOOLEAN DEFAULT FALSE,
    activation_epoch BIGINT,
    activation_eligibility_epoch BIGINT,
    exit_epoch BIGINT,
    withdrawable_epoch BIGINT,

    -- Custom metadata
    name VARCHAR(255),
    tags TEXT[], -- Array of tags for grouping validators
    monitored BOOLEAN DEFAULT TRUE,

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for validators table
CREATE INDEX idx_validators_pubkey ON validators(pubkey);
CREATE INDEX idx_validators_index ON validators(validator_index);
CREATE INDEX idx_validators_monitored ON validators(monitored) WHERE monitored = TRUE;
CREATE INDEX idx_validators_tags ON validators USING GIN(tags);

-- =====================================================
-- VALIDATOR_SNAPSHOTS TABLE (TimescaleDB Hypertable)
-- =====================================================
-- Time-series data for validator performance metrics
CREATE TABLE IF NOT EXISTS validator_snapshots (
    time TIMESTAMPTZ NOT NULL,
    validator_index BIGINT NOT NULL,

    -- Balance information
    balance BIGINT NOT NULL,
    effective_balance BIGINT NOT NULL,

    -- Performance metrics
    attestation_effectiveness DECIMAL(5,2), -- McDonald's formula result (0-100%)
    attestation_inclusion_delay INTEGER,
    attestation_head_vote BOOLEAN,
    attestation_source_vote BOOLEAN,
    attestation_target_vote BOOLEAN,

    -- Proposal information
    proposals_scheduled INTEGER DEFAULT 0,
    proposals_executed INTEGER DEFAULT 0,
    proposals_missed INTEGER DEFAULT 0,

    -- Sync committee participation
    sync_committee_participation BOOLEAN DEFAULT FALSE,

    -- Slashing information
    slashed BOOLEAN DEFAULT FALSE,

    -- Network participation
    is_online BOOLEAN DEFAULT TRUE,
    consecutive_missed_attestations INTEGER DEFAULT 0,

    -- Calculated metrics
    daily_income BIGINT, -- Income earned in last 24h
    apr DECIMAL(5,2), -- Annual percentage rate

    -- Foreign key constraint
    FOREIGN KEY (validator_index) REFERENCES validators(validator_index) ON DELETE CASCADE
);

-- Create TimescaleDB hypertable with 1-day chunks
SELECT create_hypertable('validator_snapshots', 'time',
    chunk_time_interval => INTERVAL '1 day',
    if_not_exists => TRUE
);

-- Create composite index for efficient queries
CREATE INDEX idx_validator_snapshots_validator_time
    ON validator_snapshots (validator_index, time DESC);

-- Index for time-based queries
CREATE INDEX idx_validator_snapshots_time
    ON validator_snapshots (time DESC);

-- Index for performance queries
CREATE INDEX idx_validator_snapshots_effectiveness
    ON validator_snapshots (attestation_effectiveness)
    WHERE attestation_effectiveness IS NOT NULL;

-- Index for monitoring missed attestations
CREATE INDEX idx_validator_snapshots_missed
    ON validator_snapshots (consecutive_missed_attestations)
    WHERE consecutive_missed_attestations > 0;

-- =====================================================
-- ALERTS TABLE
-- =====================================================
-- Stores alerts and notifications for validators
CREATE TABLE IF NOT EXISTS alerts (
    id SERIAL PRIMARY KEY,
    validator_index BIGINT,

    -- Alert information
    alert_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('info', 'warning', 'error', 'critical')),

    -- Alert content
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    details JSONB,

    -- Status tracking
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'acknowledged', 'resolved', 'ignored')),
    acknowledged_at TIMESTAMPTZ,
    resolved_at TIMESTAMPTZ,

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    -- Foreign key (nullable for system-wide alerts)
    FOREIGN KEY (validator_index) REFERENCES validators(validator_index) ON DELETE CASCADE
);

-- Indexes for alerts table
CREATE INDEX idx_alerts_validator ON alerts(validator_index);
CREATE INDEX idx_alerts_status ON alerts(status) WHERE status = 'active';
CREATE INDEX idx_alerts_severity ON alerts(severity);
CREATE INDEX idx_alerts_type ON alerts(alert_type);
CREATE INDEX idx_alerts_created_at ON alerts(created_at DESC);

-- Composite index for active alerts by validator
CREATE INDEX idx_alerts_active_validator
    ON alerts (validator_index, created_at DESC)
    WHERE status = 'active';

-- =====================================================
-- AGGREGATED_METRICS TABLE (TimescaleDB Hypertable)
-- =====================================================
-- Pre-aggregated metrics for faster queries
CREATE TABLE IF NOT EXISTS aggregated_metrics (
    time TIMESTAMPTZ NOT NULL,
    validator_index BIGINT NOT NULL,
    interval_type VARCHAR(20) NOT NULL CHECK (interval_type IN ('1h', '24h', '7d', '30d')),

    -- Aggregated balance metrics
    avg_balance BIGINT,
    min_balance BIGINT,
    max_balance BIGINT,

    -- Aggregated performance metrics
    avg_effectiveness DECIMAL(5,2),
    min_effectiveness DECIMAL(5,2),
    max_effectiveness DECIMAL(5,2),

    -- Participation metrics
    total_attestations INTEGER,
    missed_attestations INTEGER,
    participation_rate DECIMAL(5,2),

    -- Income metrics
    total_income BIGINT,
    avg_apr DECIMAL(5,2),

    -- Uptime
    uptime_percentage DECIMAL(5,2),

    PRIMARY KEY (time, validator_index, interval_type),
    FOREIGN KEY (validator_index) REFERENCES validators(validator_index) ON DELETE CASCADE
);

-- Create TimescaleDB hypertable for aggregated metrics
SELECT create_hypertable('aggregated_metrics', 'time',
    chunk_time_interval => INTERVAL '7 days',
    if_not_exists => TRUE
);

-- Index for querying by validator and interval
CREATE INDEX idx_aggregated_metrics_lookup
    ON aggregated_metrics (validator_index, interval_type, time DESC);

-- =====================================================
-- CONTINUOUS AGGREGATES (TimescaleDB)
-- =====================================================

-- Hourly aggregates
CREATE MATERIALIZED VIEW validator_hourly_stats
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 hour', time) AS hour,
    validator_index,
    AVG(balance)::BIGINT as avg_balance,
    AVG(attestation_effectiveness) as avg_effectiveness,
    COUNT(*) FILTER (WHERE attestation_effectiveness < 95) as suboptimal_attestations,
    MAX(consecutive_missed_attestations) as max_missed
FROM validator_snapshots
GROUP BY hour, validator_index
WITH NO DATA;

-- Create refresh policy for hourly stats
SELECT add_continuous_aggregate_policy('validator_hourly_stats',
    start_offset => INTERVAL '2 hours',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour');

-- Daily aggregates
CREATE MATERIALIZED VIEW validator_daily_stats
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 day', time) AS day,
    validator_index,
    AVG(balance)::BIGINT as avg_balance,
    MIN(balance)::BIGINT as min_balance,
    MAX(balance)::BIGINT as max_balance,
    AVG(attestation_effectiveness) as avg_effectiveness,
    SUM(CASE WHEN attestation_effectiveness < 95 THEN 1 ELSE 0 END) as suboptimal_count,
    AVG(apr) as avg_apr
FROM validator_snapshots
GROUP BY day, validator_index
WITH NO DATA;

-- Create refresh policy for daily stats
SELECT add_continuous_aggregate_policy('validator_daily_stats',
    start_offset => INTERVAL '2 days',
    end_offset => INTERVAL '1 day',
    schedule_interval => INTERVAL '1 day');

-- =====================================================
-- HELPER FUNCTIONS
-- =====================================================

-- Function to calculate validator effectiveness over a time period
CREATE OR REPLACE FUNCTION calculate_validator_effectiveness(
    p_validator_index BIGINT,
    p_start_time TIMESTAMPTZ,
    p_end_time TIMESTAMPTZ
) RETURNS DECIMAL AS $$
BEGIN
    RETURN (
        SELECT AVG(attestation_effectiveness)
        FROM validator_snapshots
        WHERE validator_index = p_validator_index
          AND time >= p_start_time
          AND time <= p_end_time
    );
END;
$$ LANGUAGE plpgsql;

-- Function to get validator income over a time period
CREATE OR REPLACE FUNCTION calculate_validator_income(
    p_validator_index BIGINT,
    p_start_time TIMESTAMPTZ,
    p_end_time TIMESTAMPTZ
) RETURNS BIGINT AS $$
DECLARE
    start_balance BIGINT;
    end_balance BIGINT;
BEGIN
    SELECT balance INTO start_balance
    FROM validator_snapshots
    WHERE validator_index = p_validator_index
      AND time >= p_start_time
    ORDER BY time ASC
    LIMIT 1;

    SELECT balance INTO end_balance
    FROM validator_snapshots
    WHERE validator_index = p_validator_index
      AND time <= p_end_time
    ORDER BY time DESC
    LIMIT 1;

    RETURN COALESCE(end_balance - start_balance, 0);
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- TRIGGERS
-- =====================================================

-- Update validator updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_validators_updated_at
    BEFORE UPDATE ON validators
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_alerts_updated_at
    BEFORE UPDATE ON alerts
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- =====================================================
-- DATA RETENTION POLICIES
-- =====================================================

-- Keep raw snapshots for 90 days
SELECT add_retention_policy('validator_snapshots',
    drop_after => INTERVAL '90 days',
    if_not_exists => TRUE);

-- Keep aggregated metrics for 1 year
SELECT add_retention_policy('aggregated_metrics',
    drop_after => INTERVAL '365 days',
    if_not_exists => TRUE);

-- Keep alerts for 180 days
-- Note: Alerts table is not a hypertable, so we use a different approach
CREATE OR REPLACE FUNCTION cleanup_old_alerts()
RETURNS void AS $$
BEGIN
    DELETE FROM alerts
    WHERE created_at < NOW() - INTERVAL '180 days'
      AND status IN ('resolved', 'ignored');
END;
$$ LANGUAGE plpgsql;

-- Create a scheduled job for alert cleanup (requires pg_cron extension)
-- SELECT cron.schedule('cleanup-alerts', '0 2 * * *', 'SELECT cleanup_old_alerts();');

COMMENT ON TABLE validators IS 'Stores Ethereum validator metadata and configuration';
COMMENT ON TABLE validator_snapshots IS 'Time-series data for validator performance metrics';
COMMENT ON TABLE alerts IS 'Alert and notification history for validators';
COMMENT ON TABLE aggregated_metrics IS 'Pre-computed metrics for performance optimization';