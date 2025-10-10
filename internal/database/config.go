package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Config holds database configuration parameters
type Config struct {
	Host                  string
	Port                  int
	User                  string
	Password              string
	Database              string
	SSLMode               string
	MaxConnections        int32
	MinConnections        int32
	MaxConnectionLifetime time.Duration
	MaxConnectionIdleTime time.Duration
	HealthCheckPeriod     time.Duration
	ConnectTimeout        time.Duration

	// Performance tuning
	StatementCacheMode   string // "prepare" or "describe"
	DefaultQueryExecMode pgx.QueryExecMode
}

// DefaultConfig returns default database configuration
func DefaultConfig() *Config {
	return &Config{
		Host:                  "localhost",
		Port:                  5432,
		User:                  "validator_monitor",
		Database:              "validator_monitor",
		SSLMode:               "prefer",
		MaxConnections:        25,
		MinConnections:        5,
		MaxConnectionLifetime: time.Hour,
		MaxConnectionIdleTime: time.Minute * 30,
		HealthCheckPeriod:     time.Minute,
		ConnectTimeout:        time.Second * 5,
		StatementCacheMode:    "prepare",
		DefaultQueryExecMode:  pgx.QueryExecModeDescribeExec,
	}
}

// BuildDSN creates a database connection string
func (c *Config) BuildDSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s&connect_timeout=%d",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
		c.SSLMode,
		int(c.ConnectTimeout.Seconds()),
	)
}

// BuildPoolConfig creates a pgxpool configuration
func (c *Config) BuildPoolConfig() (*pgxpool.Config, error) {
	poolConfig, err := pgxpool.ParseConfig(c.BuildDSN())
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// Connection pool settings
	poolConfig.MaxConns = c.MaxConnections
	poolConfig.MinConns = c.MinConnections
	poolConfig.MaxConnLifetime = c.MaxConnectionLifetime
	poolConfig.MaxConnIdleTime = c.MaxConnectionIdleTime
	poolConfig.HealthCheckPeriod = c.HealthCheckPeriod

	// Connection configuration
	poolConfig.ConnConfig.DefaultQueryExecMode = c.DefaultQueryExecMode

	// Before connect hook for connection initialization
	poolConfig.BeforeConnect = func(ctx context.Context, cfg *pgx.ConnConfig) error {
		// Set application name for monitoring
		cfg.RuntimeParams["application_name"] = "eth-validator-monitor"

		// Set statement timeout to prevent runaway queries
		cfg.RuntimeParams["statement_timeout"] = "30s"

		// Set lock timeout to prevent deadlocks
		cfg.RuntimeParams["lock_timeout"] = "10s"

		// Set idle transaction timeout
		cfg.RuntimeParams["idle_in_transaction_session_timeout"] = "60s"

		return nil
	}

	// After connect hook for connection setup
	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		// Prepare frequently used statements
		_, err := conn.Prepare(ctx, "get_validator", `
			SELECT validator_index, pubkey, effective_balance, slashed, name, monitored
			FROM validators
			WHERE validator_index = $1
		`)
		if err != nil {
			return fmt.Errorf("failed to prepare get_validator statement: %w", err)
		}

		_, err = conn.Prepare(ctx, "insert_snapshot", `
			INSERT INTO validator_snapshots (
				time, validator_index, balance, effective_balance,
				attestation_effectiveness, is_online, consecutive_missed_attestations
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
		`)
		if err != nil {
			return fmt.Errorf("failed to prepare insert_snapshot statement: %w", err)
		}

		return nil
	}

	return poolConfig, nil
}

// NewPool creates a new database connection pool
func NewPool(ctx context.Context, cfg *Config) (*pgxpool.Pool, error) {
	poolConfig, err := cfg.BuildPoolConfig()
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}

// PoolStats contains connection pool statistics
type PoolStats struct {
	AcquireCount         int64
	AcquireDuration      time.Duration
	AcquiredConns        int32
	CanceledAcquireCount int64
	ConstructingConns    int32
	EmptyAcquireCount    int64
	IdleConns            int32
	MaxConns             int32
	TotalConns           int32
	NewConnsCount        int64
	MaxLifetimeDestroyCount int64
	MaxIdleDestroyCount     int64
}

// GetPoolStats returns current connection pool statistics
func GetPoolStats(pool *pgxpool.Pool) *PoolStats {
	stats := pool.Stat()
	return &PoolStats{
		AcquireCount:         stats.AcquireCount(),
		AcquireDuration:      stats.AcquireDuration(),
		AcquiredConns:        stats.AcquiredConns(),
		CanceledAcquireCount: stats.CanceledAcquireCount(),
		ConstructingConns:    stats.ConstructingConns(),
		EmptyAcquireCount:    stats.EmptyAcquireCount(),
		IdleConns:            stats.IdleConns(),
		MaxConns:             stats.MaxConns(),
		TotalConns:           stats.TotalConns(),
		NewConnsCount:        stats.NewConnsCount(),
		MaxLifetimeDestroyCount: stats.MaxLifetimeDestroyCount(),
		MaxIdleDestroyCount:     stats.MaxIdleDestroyCount(),
	}
}