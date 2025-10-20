-- Rollback validator schema changes
BEGIN;

-- Step 1: Recreate old 'index' column
ALTER TABLE validators ADD COLUMN IF NOT EXISTS index INTEGER;

-- Step 2: Recreate old 'status' column
ALTER TABLE validators ADD COLUMN IF NOT EXISTS status VARCHAR(20);

-- Step 3: Copy data back from validator_index to index
UPDATE validators
SET index = validator_index::INTEGER,
    status = CASE
        WHEN slashed THEN 'slashed'
        WHEN exit_epoch IS NOT NULL THEN 'exited'
        WHEN activation_epoch IS NOT NULL THEN 'active'
        ELSE 'pending'
    END
WHERE index IS NULL;

-- Step 4: Update foreign key references in validator_performance back to 'index'
ALTER TABLE validator_performance ADD COLUMN IF NOT EXISTS old_validator_index INTEGER;

UPDATE validator_performance vp
SET old_validator_index = v.index
FROM validators v
WHERE vp.validator_index = v.validator_index::INTEGER;

ALTER TABLE validator_performance DROP CONSTRAINT IF EXISTS validator_performance_validator_index_fkey;
ALTER TABLE validator_performance DROP COLUMN IF EXISTS validator_index;
ALTER TABLE validator_performance RENAME COLUMN old_validator_index TO validator_index;
ALTER TABLE validator_performance ALTER COLUMN validator_index SET NOT NULL;

-- Step 5: Update foreign key references in alerts back to 'index'
ALTER TABLE alerts ADD COLUMN IF NOT EXISTS old_validator_index INTEGER;

UPDATE alerts a
SET old_validator_index = v.index
FROM validators v
WHERE a.validator_index = v.validator_index::INTEGER;

ALTER TABLE alerts DROP CONSTRAINT IF EXISTS alerts_validator_index_fkey;
ALTER TABLE alerts DROP COLUMN IF EXISTS validator_index;
ALTER TABLE alerts RENAME COLUMN old_validator_index TO validator_index;

-- Step 6: Update foreign key references in validator_snapshots back to 'index'
ALTER TABLE validator_snapshots ADD COLUMN IF NOT EXISTS old_validator_index INTEGER;

UPDATE validator_snapshots vs
SET old_validator_index = v.index
FROM validators v
WHERE vs.validator_index = v.validator_index;

ALTER TABLE validator_snapshots DROP CONSTRAINT IF EXISTS validator_snapshots_validator_index_fkey;
ALTER TABLE validator_snapshots DROP COLUMN IF EXISTS validator_index;
ALTER TABLE validator_snapshots RENAME COLUMN old_validator_index TO validator_index;
ALTER TABLE validator_snapshots ALTER COLUMN validator_index SET NOT NULL;

-- Step 7: Recreate old foreign keys referencing 'index'
ALTER TABLE validator_snapshots
  ADD CONSTRAINT validator_snapshots_validator_index_fkey
  FOREIGN KEY (validator_index) REFERENCES validators(index) ON DELETE CASCADE;

ALTER TABLE alerts
  ADD CONSTRAINT alerts_validator_index_fkey
  FOREIGN KEY (validator_index) REFERENCES validators(index) ON DELETE CASCADE;

ALTER TABLE validator_performance
  ADD CONSTRAINT validator_performance_validator_index_fkey
  FOREIGN KEY (validator_index) REFERENCES validators(index) ON DELETE CASCADE;

-- Step 8: Drop new constraints
ALTER TABLE validators DROP CONSTRAINT IF EXISTS validators_pkey;
ALTER TABLE validators DROP CONSTRAINT IF EXISTS validators_validator_index_unique;

-- Step 9: Recreate old primary key on 'index'
ALTER TABLE validators ADD PRIMARY KEY (index);

-- Step 10: Drop new indexes
DROP INDEX IF EXISTS idx_validators_validator_index;
DROP INDEX IF EXISTS idx_validators_monitored;
DROP INDEX IF EXISTS idx_validators_tags;

-- Step 11: Recreate old index
CREATE INDEX IF NOT EXISTS idx_validators_index ON validators(index);

-- Step 12: Drop triggers
DROP TRIGGER IF EXISTS update_validators_updated_at ON validators;
DROP TRIGGER IF EXISTS update_alerts_updated_at ON alerts;
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Step 13: Revert timestamp columns
ALTER TABLE validators
  ALTER COLUMN created_at TYPE TIMESTAMP,
  ALTER COLUMN updated_at TYPE TIMESTAMP;

-- Step 14: Drop new columns
ALTER TABLE validators
  DROP COLUMN IF EXISTS id,
  DROP COLUMN IF EXISTS validator_index,
  DROP COLUMN IF EXISTS withdrawal_credentials,
  DROP COLUMN IF EXISTS effective_balance,
  DROP COLUMN IF EXISTS activation_eligibility_epoch,
  DROP COLUMN IF EXISTS withdrawable_epoch,
  DROP COLUMN IF EXISTS tags,
  DROP COLUMN IF EXISTS monitored;

COMMIT;
