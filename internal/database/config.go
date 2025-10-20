package database

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SSLMode represents PostgreSQL SSL modes
type SSLMode string

const (
	// SSLModeDisable - No SSL
	SSLModeDisable SSLMode = "disable"

	// SSLModeAllow - Try SSL, fallback to non-SSL
	SSLModeAllow SSLMode = "allow"

	// SSLModePrefer - Try SSL first, fallback to non-SSL (default)
	SSLModePrefer SSLMode = "prefer"

	// SSLModeRequire - Require SSL (no certificate verification)
	SSLModeRequire SSLMode = "require"

	// SSLModeVerifyCA - Require SSL and verify CA certificate
	SSLModeVerifyCA SSLMode = "verify-ca"

	// SSLModeVerifyFull - Require SSL, verify CA and hostname
	SSLModeVerifyFull SSLMode = "verify-full"
)

// Config holds database configuration parameters
type Config struct {
	Host                  string
	Port                  int
	User                  string
	Password              string
	Database              string
	SSLMode               SSLMode
	SSLCert               string // Path to client certificate (for verify-ca/verify-full)
	SSLKey                string // Path to client key (for verify-ca/verify-full)
	SSLRootCert           string // Path to root CA certificate (for verify-ca/verify-full)
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
		SSLMode:               SSLModeRequire, // Default to require SSL
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

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("database host cannot be empty")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("database port must be between 1 and 65535")
	}
	if c.User == "" {
		return fmt.Errorf("database user cannot be empty")
	}
	if c.Database == "" {
		return fmt.Errorf("database name cannot be empty")
	}
	if c.MaxConnections < c.MinConnections {
		return fmt.Errorf("max connections (%d) must be >= min connections (%d)", c.MaxConnections, c.MinConnections)
	}
	if c.MaxConnections <= 0 {
		return fmt.Errorf("max connections must be positive")
	}
	if c.ConnectTimeout <= 0 {
		return fmt.Errorf("connect timeout must be positive")
	}

	// Validate SSL mode
	validModes := map[SSLMode]bool{
		SSLModeDisable:    true,
		SSLModeAllow:      true,
		SSLModePrefer:     true,
		SSLModeRequire:    true,
		SSLModeVerifyCA:   true,
		SSLModeVerifyFull: true,
	}

	if !validModes[c.SSLMode] {
		return fmt.Errorf("invalid SSL mode: %s (valid: disable, allow, prefer, require, verify-ca, verify-full)", c.SSLMode)
	}

	// For production, enforce secure SSL modes
	if os.Getenv("ENV") == "production" {
		if c.SSLMode == SSLModeDisable || c.SSLMode == SSLModeAllow || c.SSLMode == SSLModePrefer {
			return fmt.Errorf("production environment requires SSL mode 'require', 'verify-ca', or 'verify-full'")
		}
	}

	return nil
}

// BuildDSN creates a database connection string
func (c *Config) BuildDSN() string {
	// Base connection string
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s&connect_timeout=%d",
		url.QueryEscape(c.User),
		url.QueryEscape(c.Password),
		c.Host,
		c.Port,
		c.Database,
		c.SSLMode,
		int(c.ConnectTimeout.Seconds()),
	)

	// Add SSL certificate paths if provided
	if c.SSLCert != "" {
		connStr += "&sslcert=" + url.QueryEscape(c.SSLCert)
	}
	if c.SSLKey != "" {
		connStr += "&sslkey=" + url.QueryEscape(c.SSLKey)
	}
	if c.SSLRootCert != "" {
		connStr += "&sslrootcert=" + url.QueryEscape(c.SSLRootCert)
	}

	return connStr
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
				epoch, slot, timestamp, validator_index, balance, effective_balance,
				attestation_success, attestation_inclusion_delay, proposal_success,
				performance_score, network_percentile
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
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

	// Verify SSL is in use if required
	if cfg.SSLMode != SSLModeDisable {
		var sslInUse bool
		err = pool.QueryRow(ctx, "SELECT pg_catalog.ssl_is_used()").Scan(&sslInUse)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("failed to check SSL status: %w", err)
		}

		if !sslInUse {
			pool.Close()
			return nil, fmt.Errorf("SSL is not in use despite sslmode=%s", cfg.SSLMode)
		}
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