package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// TestDBPool creates a test database connection pool
func TestDBPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	connString := GetTestDBConnString()
	pool, err := pgxpool.New(context.Background(), connString)
	require.NoError(t, err, "Failed to create test database pool")

	t.Cleanup(func() {
		pool.Close()
	})

	return pool
}

// TestRedisClient creates a test Redis client
func TestRedisClient(t *testing.T) *redis.Client {
	t.Helper()

	client := redis.NewClient(&redis.Options{
		Addr:     GetTestRedisAddr(),
		Password: "",
		DB:       1, // Use test database
	})

	// Verify connection
	ctx := context.Background()
	err := client.Ping(ctx).Err()
	require.NoError(t, err, "Failed to connect to test Redis")

	t.Cleanup(func() {
		// Clear test database
		client.FlushDB(ctx)
		client.Close()
	})

	return client
}

// GetTestDBConnString returns the test database connection string
func GetTestDBConnString() string {
	// Use environment variable or default to local test database
	return "postgresql://postgres:postgres@localhost:5432/eth_validator_test?sslmode=disable"
}

// GetTestRedisAddr returns the test Redis address
func GetTestRedisAddr() string {
	return "localhost:6379"
}

// FixedTime returns a fixed time for testing
func FixedTime() time.Time {
	return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
}

// WaitForCondition waits for a condition to be true with timeout
func WaitForCondition(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("Timeout waiting for condition: %s", message)
}

// SetupTestDB creates and migrates a test database
func SetupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	pool := TestDBPool(t)

	// Run migrations
	ctx := context.Background()
	err := RunMigrations(ctx, pool)
	require.NoError(t, err, "Failed to run test database migrations")

	return pool
}

// RunMigrations runs database migrations for tests
func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	// TODO: Implement actual migration logic
	// For now, just create basic tables
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS validators (
			id SERIAL PRIMARY KEY,
			validator_index BIGINT UNIQUE NOT NULL,
			pubkey TEXT UNIQUE NOT NULL,
			withdrawal_credentials TEXT,
			effective_balance BIGINT,
			slashed BOOLEAN DEFAULT FALSE,
			activation_epoch BIGINT,
			activation_eligibility_epoch BIGINT,
			exit_epoch BIGINT,
			withdrawable_epoch BIGINT,
			name TEXT,
			tags TEXT[],
			monitored BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS validator_snapshots (
			time TIMESTAMP NOT NULL,
			validator_index BIGINT NOT NULL,
			balance BIGINT,
			effective_balance BIGINT,
			attestation_effectiveness DOUBLE PRECISION,
			attestation_inclusion_delay INT,
			attestation_head_vote BOOLEAN,
			attestation_source_vote BOOLEAN,
			attestation_target_vote BOOLEAN,
			proposals_scheduled INT DEFAULT 0,
			proposals_executed INT DEFAULT 0,
			proposals_missed INT DEFAULT 0,
			sync_committee_participation DOUBLE PRECISION DEFAULT 0,
			slashed BOOLEAN DEFAULT FALSE,
			is_online BOOLEAN DEFAULT TRUE,
			consecutive_missed_attestations INT DEFAULT 0,
			daily_income BIGINT DEFAULT 0,
			apr DOUBLE PRECISION DEFAULT 0,
			PRIMARY KEY (time, validator_index)
		)`,
	}

	for _, migration := range migrations {
		_, err := pool.Exec(ctx, migration)
		if err != nil {
			return err
		}
	}

	return nil
}

// CleanupTestDB removes all test data
func CleanupTestDB(ctx context.Context, pool *pgxpool.Pool) error {
	tables := []string{
		"validator_snapshots",
		"validators",
	}

	for _, table := range tables {
		_, err := pool.Exec(ctx, "TRUNCATE TABLE "+table+" CASCADE")
		if err != nil {
			return err
		}
	}

	return nil
}
