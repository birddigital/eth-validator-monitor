-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- Validators table
CREATE TABLE validators (
    id SERIAL PRIMARY KEY,
    validator_index BIGINT UNIQUE NOT NULL,
    pubkey VARCHAR(98) UNIQUE NOT NULL,
    withdrawal_credentials VARCHAR(66),
    effective_balance BIGINT,
    slashed BOOLEAN DEFAULT FALSE,
    activation_epoch BIGINT,
    activation_eligibility_epoch BIGINT,
    exit_epoch BIGINT,
    withdrawable_epoch BIGINT,
    name VARCHAR(255),
    tags TEXT[],
    monitored BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Validators indexes
CREATE INDEX idx_validators_pubkey ON validators(pubkey);
CREATE INDEX idx_validators_index ON validators(validator_index);
CREATE INDEX idx_validators_monitored ON validators(monitored) WHERE monitored = TRUE;
CREATE INDEX idx_validators_tags ON validators USING GIN(tags);

-- Validator snapshots table
CREATE TABLE validator_snapshots (
    time TIMESTAMPTZ NOT NULL,
    validator_index BIGINT NOT NULL,
    balance BIGINT NOT NULL,
    effective_balance BIGINT NOT NULL,
    attestation_effectiveness DECIMAL(5,2),
    attestation_inclusion_delay INTEGER,
    attestation_head_vote BOOLEAN,
    attestation_source_vote BOOLEAN,
    attestation_target_vote BOOLEAN,
    proposals_scheduled INTEGER DEFAULT 0,
    proposals_executed INTEGER DEFAULT 0,
    proposals_missed INTEGER DEFAULT 0,
    sync_committee_participation BOOLEAN DEFAULT FALSE,
    slashed BOOLEAN DEFAULT FALSE,
    is_online BOOLEAN DEFAULT TRUE,
    consecutive_missed_attestations INTEGER DEFAULT 0,
    daily_income BIGINT,
    apr DECIMAL(5,2),
    FOREIGN KEY (validator_index) REFERENCES validators(validator_index) ON DELETE CASCADE
);

-- Create hypertable
SELECT create_hypertable('validator_snapshots', 'time',
    chunk_time_interval => INTERVAL '1 day',
    if_not_exists => TRUE
);

-- Validator snapshots indexes
CREATE INDEX idx_validator_snapshots_validator_time ON validator_snapshots (validator_index, time DESC);
CREATE INDEX idx_validator_snapshots_time ON validator_snapshots (time DESC);
CREATE INDEX idx_validator_snapshots_effectiveness ON validator_snapshots (attestation_effectiveness) WHERE attestation_effectiveness IS NOT NULL;
CREATE INDEX idx_validator_snapshots_missed ON validator_snapshots (consecutive_missed_attestations) WHERE consecutive_missed_attestations > 0;

-- Alerts table
CREATE TABLE alerts (
    id SERIAL PRIMARY KEY,
    validator_index BIGINT,
    alert_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('info', 'warning', 'error', 'critical')),
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    details JSONB,
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'acknowledged', 'resolved', 'ignored')),
    acknowledged_at TIMESTAMPTZ,
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    FOREIGN KEY (validator_index) REFERENCES validators(validator_index) ON DELETE CASCADE
);

-- Alerts indexes
CREATE INDEX idx_alerts_validator ON alerts(validator_index);
CREATE INDEX idx_alerts_status ON alerts(status) WHERE status = 'active';
CREATE INDEX idx_alerts_severity ON alerts(severity);
CREATE INDEX idx_alerts_type ON alerts(alert_type);
CREATE INDEX idx_alerts_created_at ON alerts(created_at DESC);
CREATE INDEX idx_alerts_active_validator ON alerts (validator_index, created_at DESC) WHERE status = 'active';

-- Update timestamp trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply trigger to tables
CREATE TRIGGER update_validators_updated_at BEFORE UPDATE ON validators
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_alerts_updated_at BEFORE UPDATE ON alerts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();