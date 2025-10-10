-- Create validators table
CREATE TABLE validators (
    index INTEGER PRIMARY KEY,
    pubkey VARCHAR(98) NOT NULL UNIQUE,
    name VARCHAR(255),
    status VARCHAR(20) NOT NULL,
    activation_epoch INTEGER,
    exit_epoch INTEGER,
    slashed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_validators_status ON validators(status);
CREATE INDEX idx_validators_pubkey ON validators(pubkey);

-- Create validator snapshots table (epoch-level data)
CREATE TABLE validator_snapshots (
    id BIGSERIAL PRIMARY KEY,
    validator_index INTEGER NOT NULL REFERENCES validators(index) ON DELETE CASCADE,
    epoch INTEGER NOT NULL,
    slot INTEGER NOT NULL,
    timestamp TIMESTAMP NOT NULL,

    -- Balance data
    balance BIGINT NOT NULL,
    effective_balance BIGINT NOT NULL,

    -- Performance data
    attestation_success BOOLEAN,
    attestation_inclusion_delay INTEGER,
    proposal_success BOOLEAN,

    -- Scoring
    performance_score NUMERIC(5,2),
    network_percentile NUMERIC(5,2),

    created_at TIMESTAMP DEFAULT NOW(),

    UNIQUE(validator_index, epoch)
);

CREATE INDEX idx_snapshots_validator_epoch ON validator_snapshots(validator_index, epoch DESC);
CREATE INDEX idx_snapshots_timestamp ON validator_snapshots(timestamp DESC);

-- Create alerts table
CREATE TABLE alerts (
    id BIGSERIAL PRIMARY KEY,
    validator_index INTEGER NOT NULL REFERENCES validators(index) ON DELETE CASCADE,
    severity VARCHAR(20) NOT NULL, -- critical, warning, info
    type VARCHAR(50) NOT NULL, -- offline, slashed, performance_degraded, etc.
    message TEXT NOT NULL,
    metadata JSONB,
    acknowledged BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_alerts_validator ON alerts(validator_index, created_at DESC);
CREATE INDEX idx_alerts_severity ON alerts(severity, created_at DESC);
CREATE INDEX idx_alerts_acknowledged ON alerts(acknowledged, created_at DESC);

-- Create validator performance table
CREATE TABLE validator_performance (
    id BIGSERIAL PRIMARY KEY,
    validator_index INTEGER NOT NULL REFERENCES validators(index) ON DELETE CASCADE,
    timestamp TIMESTAMP NOT NULL,

    -- Uptime metrics
    uptime_percentage NUMERIC(5,2) NOT NULL,
    consecutive_misses INTEGER DEFAULT 0,
    total_missed INTEGER DEFAULT 0,

    -- Effectiveness metrics
    attestation_score NUMERIC(5,2) NOT NULL,
    proposal_success INTEGER DEFAULT 0,
    proposal_missed INTEGER DEFAULT 0,

    -- Rewards metrics (stored as strings for big integers)
    expected_rewards VARCHAR(255),
    actual_rewards VARCHAR(255),
    effectiveness NUMERIC(5,2),

    -- Comparative metrics
    network_average NUMERIC(5,2),
    percentile NUMERIC(5,2),

    -- Risk indicators
    slashing_risk VARCHAR(20), -- none, low, medium, high
    inactivity_score INTEGER DEFAULT 0,

    created_at TIMESTAMP DEFAULT NOW(),

    UNIQUE(validator_index, timestamp)
);

CREATE INDEX idx_performance_validator ON validator_performance(validator_index, timestamp DESC);

-- Create network stats table for caching
CREATE TABLE network_stats (
    id SERIAL PRIMARY KEY,
    current_epoch INTEGER NOT NULL,
    current_slot INTEGER NOT NULL,
    total_validators INTEGER NOT NULL,
    active_validators INTEGER NOT NULL,
    pending_validators INTEGER DEFAULT 0,
    exiting_validators INTEGER DEFAULT 0,
    slashed_validators INTEGER DEFAULT 0,
    average_balance VARCHAR(255), -- stored as string for big integer
    total_staked VARCHAR(255), -- stored as string for big integer
    participation_rate NUMERIC(5,4),
    timestamp TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_network_stats_timestamp ON network_stats(timestamp DESC);
