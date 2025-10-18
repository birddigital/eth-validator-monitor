package helpers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// SetupTestDB creates a test PostgreSQL database using testcontainers
func SetupTestDB(b *testing.B) *pgxpool.Pool {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "timescale/timescaledb:latest-pg14",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_PASSWORD": "postgres",
			"POSTGRES_DB":       "benchtest",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		b.Fatalf("failed to start postgres container: %v", err)
	}

	b.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			b.Logf("failed to terminate container: %v", err)
		}
	})

	host, err := container.Host(ctx)
	if err != nil {
		b.Fatalf("failed to get container host: %v", err)
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		b.Fatalf("failed to get container port: %v", err)
	}

	connStr := fmt.Sprintf("postgres://postgres:postgres@%s:%s/benchtest?sslmode=disable", host, port.Port())

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		b.Fatalf("failed to create connection pool: %v", err)
	}

	// Create tables
	initSchema(b, pool)

	return pool
}

// SetupTestRedis creates a test Redis instance using testcontainers
func SetupTestRedis(b *testing.B) *redis.Client {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForListeningPort("6379/tcp").WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		b.Fatalf("failed to start redis container: %v", err)
	}

	b.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			b.Logf("failed to terminate container: %v", err)
		}
	})

	host, err := container.Host(ctx)
	if err != nil {
		b.Fatalf("failed to get container host: %v", err)
	}

	port, err := container.MappedPort(ctx, "6379")
	if err != nil {
		b.Fatalf("failed to get container port: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", host, port.Port()),
	})

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		b.Fatalf("failed to connect to redis: %v", err)
	}

	return client
}

// initSchema creates the necessary database tables for benchmarking
func initSchema(b *testing.B, pool *pgxpool.Pool) {
	ctx := context.Background()

	schema := `
	CREATE EXTENSION IF NOT EXISTS timescaledb;

	CREATE TABLE IF NOT EXISTS validators (
		index BIGINT PRIMARY KEY,
		pubkey BYTEA NOT NULL,
		status VARCHAR(50) NOT NULL,
		balance BIGINT NOT NULL,
		activation_epoch BIGINT
	);

	CREATE TABLE IF NOT EXISTS validator_snapshots (
		validator_index BIGINT NOT NULL,
		timestamp TIMESTAMPTZ NOT NULL,
		balance BIGINT NOT NULL,
		effective_balance BIGINT NOT NULL,
		effectiveness DOUBLE PRECISION NOT NULL,
		missed_attestations BIGINT NOT NULL,
		proposal_success BOOLEAN NOT NULL,
		epoch BIGINT NOT NULL,
		slot BIGINT NOT NULL,
		PRIMARY KEY (validator_index, timestamp)
	);

	SELECT create_hypertable('validator_snapshots', 'timestamp', if_not_exists => TRUE);

	CREATE INDEX IF NOT EXISTS idx_snapshots_validator_time
	ON validator_snapshots (validator_index, timestamp DESC);

	CREATE INDEX IF NOT EXISTS idx_snapshots_effectiveness
	ON validator_snapshots (effectiveness DESC);
	`

	_, err := pool.Exec(ctx, schema)
	if err != nil {
		b.Fatalf("failed to create schema: %v", err)
	}
}

// CleanupSnapshots removes all snapshot data from the database
func CleanupSnapshots(b *testing.B, pool *pgxpool.Pool) {
	ctx := context.Background()
	_, err := pool.Exec(ctx, "TRUNCATE validator_snapshots")
	if err != nil {
		b.Fatalf("failed to cleanup snapshots: %v", err)
	}
}

// CleanupValidators removes all validator data from the database
func CleanupValidators(b *testing.B, pool *pgxpool.Pool) {
	ctx := context.Background()
	_, err := pool.Exec(ctx, "TRUNCATE validators CASCADE")
	if err != nil {
		b.Fatalf("failed to cleanup validators: %v", err)
	}
}

// IntPtr returns a pointer to an int (helper for GraphQL tests)
func IntPtr(i int) *int {
	return &i
}

// StringPtr returns a pointer to a string (helper for GraphQL tests)
func StringPtr(s string) *string {
	return &s
}
