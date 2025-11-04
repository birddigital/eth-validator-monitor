-- Create API keys table for programmatic access
CREATE TABLE IF NOT EXISTS api_keys (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_hash VARCHAR(255) NOT NULL UNIQUE,
    key_prefix VARCHAR(16) NOT NULL,
    name VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP WITH TIME ZONE,
    revoked_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT fk_user
        FOREIGN KEY(user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

-- Index for fast lookups by user
CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);

-- Index for fast lookups by hash (used for authentication)
CREATE INDEX idx_api_keys_hash ON api_keys(key_hash);

-- Index for finding active (non-revoked) keys
CREATE INDEX idx_api_keys_revoked ON api_keys(revoked_at) WHERE revoked_at IS NULL;

-- Index for cleanup of expired keys
CREATE INDEX idx_api_keys_expired ON api_keys(expires_at) WHERE expires_at IS NOT NULL;

-- Add comment
COMMENT ON TABLE api_keys IS 'Stores API keys for programmatic access to the validator monitoring service';
COMMENT ON COLUMN api_keys.key_hash IS 'SHA-256 hash of the API key for secure storage';
COMMENT ON COLUMN api_keys.key_prefix IS 'First 8 characters of the key for display purposes';
COMMENT ON COLUMN api_keys.name IS 'Optional user-friendly name for the API key';
COMMENT ON COLUMN api_keys.last_used_at IS 'Timestamp of last successful authentication';
COMMENT ON COLUMN api_keys.revoked_at IS 'Timestamp when the key was revoked (NULL if active)';
COMMENT ON COLUMN api_keys.expires_at IS 'Optional expiration timestamp';
