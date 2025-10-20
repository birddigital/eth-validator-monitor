-- Fix validator schema to match Go model in internal/database/models/validator.go
BEGIN;

-- Step 1: Add new id column (will become primary key)
ALTER TABLE validators
  ADD COLUMN IF NOT EXISTS id SERIAL;

-- Step 2: Add new validator_index column (temporarily nullable)
ALTER TABLE validators
  ADD COLUMN IF NOT EXISTS validator_index BIGINT;

-- Step 3: Copy data from 'index' to 'validator_index'
UPDATE validators
SET validator_index = index
WHERE validator_index IS NULL;

-- Step 4: Add missing columns with appropriate defaults
ALTER TABLE validators
  ADD COLUMN IF NOT EXISTS withdrawal_credentials VARCHAR(66),
  ADD COLUMN IF NOT EXISTS effective_balance BIGINT DEFAULT 0,
  ADD COLUMN IF NOT EXISTS activation_eligibility_epoch INTEGER,
  ADD COLUMN IF NOT EXISTS withdrawable_epoch INTEGER,
  ADD COLUMN IF NOT EXISTS tags TEXT[] DEFAULT '{}',
  ADD COLUMN IF NOT EXISTS monitored BOOLEAN DEFAULT TRUE;

-- Step 5: Update timestamps to use TIMESTAMPTZ for timezone awareness
ALTER TABLE validators
  ALTER COLUMN created_at TYPE TIMESTAMPTZ,
  ALTER COLUMN updated_at TYPE TIMESTAMPTZ;

-- Step 6: Make validator_index NOT NULL now that data is migrated
ALTER TABLE validators
  ALTER COLUMN validator_index SET NOT NULL,
  ALTER COLUMN effective_balance SET NOT NULL,
  ALTER COLUMN tags SET NOT NULL,
  ALTER COLUMN monitored SET NOT NULL;

-- Step 7: Drop old primary key constraint on 'index'
ALTER TABLE validators DROP CONSTRAINT IF EXISTS validators_pkey CASCADE;

-- Step 8: Add new primary key on 'id'
ALTER TABLE validators ADD PRIMARY KEY (id);

-- Step 9: Add unique constraint on validator_index
ALTER TABLE validators ADD CONSTRAINT validators_validator_index_unique UNIQUE (validator_index);

-- Step 10: Update foreign key references in validator_snapshots
-- First, add new column for the new validator_index reference
ALTER TABLE validator_snapshots
  ADD COLUMN IF NOT EXISTS new_validator_index BIGINT;

-- Copy the validator_index values
UPDATE validator_snapshots vs
SET new_validator_index = v.validator_index
FROM validators v
WHERE vs.validator_index = v.index;

-- Drop old foreign key constraint
ALTER TABLE validator_snapshots
  DROP CONSTRAINT IF EXISTS validator_snapshots_validator_index_fkey;

-- Drop old validator_index column and rename new one
ALTER TABLE validator_snapshots
  DROP COLUMN IF EXISTS validator_index CASCADE;

ALTER TABLE validator_snapshots
  RENAME COLUMN new_validator_index TO validator_index;

-- Make it NOT NULL
ALTER TABLE validator_snapshots
  ALTER COLUMN validator_index SET NOT NULL;

-- Add new foreign key constraint
ALTER TABLE validator_snapshots
  ADD CONSTRAINT validator_snapshots_validator_index_fkey
  FOREIGN KEY (validator_index) REFERENCES validators(validator_index) ON DELETE CASCADE;

-- Step 11: Update foreign key references in alerts
-- Add new column for the new validator_index reference
ALTER TABLE alerts
  ADD COLUMN IF NOT EXISTS new_validator_index INTEGER;

-- Copy the validator_index values
UPDATE alerts a
SET new_validator_index = v.validator_index::INTEGER
FROM validators v
WHERE a.validator_index = v.index;

-- Drop old foreign key constraint
ALTER TABLE alerts
  DROP CONSTRAINT IF EXISTS alerts_validator_index_fkey;

-- Drop old validator_index column and rename new one
ALTER TABLE alerts
  DROP COLUMN IF EXISTS validator_index CASCADE;

ALTER TABLE alerts
  RENAME COLUMN new_validator_index TO validator_index;

-- Add new foreign key constraint
ALTER TABLE alerts
  ADD CONSTRAINT alerts_validator_index_fkey
  FOREIGN KEY (validator_index) REFERENCES validators(validator_index) ON DELETE CASCADE;

-- Step 12: Update foreign key references in validator_performance
-- Add new column for the new validator_index reference
ALTER TABLE validator_performance
  ADD COLUMN IF NOT EXISTS new_validator_index INTEGER;

-- Copy the validator_index values
UPDATE validator_performance vp
SET new_validator_index = v.validator_index::INTEGER
FROM validators v
WHERE vp.validator_index = v.index;

-- Drop old foreign key constraint
ALTER TABLE validator_performance
  DROP CONSTRAINT IF EXISTS validator_performance_validator_index_fkey;

-- Drop old validator_index column and rename new one
ALTER TABLE validator_performance
  DROP COLUMN IF EXISTS validator_index CASCADE;

ALTER TABLE validator_performance
  RENAME COLUMN new_validator_index TO validator_index;

-- Make it NOT NULL
ALTER TABLE validator_performance
  ALTER COLUMN validator_index SET NOT NULL;

-- Add new foreign key constraint
ALTER TABLE validator_performance
  ADD CONSTRAINT validator_performance_validator_index_fkey
  FOREIGN KEY (validator_index) REFERENCES validators(validator_index) ON DELETE CASCADE;

-- Step 13: Recreate index on validator_index (old idx_validators_pubkey remains)
DROP INDEX IF EXISTS idx_validators_index;
CREATE INDEX IF NOT EXISTS idx_validators_validator_index ON validators(validator_index);

-- Step 14: Add new indexes for new columns
CREATE INDEX IF NOT EXISTS idx_validators_monitored ON validators(monitored) WHERE monitored = TRUE;
CREATE INDEX IF NOT EXISTS idx_validators_tags ON validators USING GIN(tags);

-- Step 15: Create or replace the updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Step 16: Apply trigger to validators table
DROP TRIGGER IF EXISTS update_validators_updated_at ON validators;
CREATE TRIGGER update_validators_updated_at
  BEFORE UPDATE ON validators
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();

-- Step 17: Apply trigger to alerts table
DROP TRIGGER IF EXISTS update_alerts_updated_at ON alerts;
CREATE TRIGGER update_alerts_updated_at
  BEFORE UPDATE ON alerts
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();

-- Step 18: Finally, drop the old 'index' column
ALTER TABLE validators DROP COLUMN IF EXISTS index;

-- Step 19: Drop old 'status' column (not in Go model)
ALTER TABLE validators DROP COLUMN IF EXISTS status;

COMMIT;
