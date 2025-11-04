-- Drop API keys table
DROP INDEX IF EXISTS idx_api_keys_expired;
DROP INDEX IF EXISTS idx_api_keys_revoked;
DROP INDEX IF EXISTS idx_api_keys_hash;
DROP INDEX IF EXISTS idx_api_keys_user_id;
DROP TABLE IF EXISTS api_keys;
