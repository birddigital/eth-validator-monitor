package database

import (
	"context"
	"embed"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// MigrationRunner handles database migrations
type MigrationRunner struct {
	pool *pgxpool.Pool
	dsn  string
}

// NewMigrationRunner creates a new migration runner
func NewMigrationRunner(pool *pgxpool.Pool, dsn string) *MigrationRunner {
	return &MigrationRunner{
		pool: pool,
		dsn:  dsn,
	}
}

// RunMigrations executes all pending migrations
func (r *MigrationRunner) RunMigrations(ctx context.Context) error {
	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create migration source driver: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", sourceDriver, r.dsn)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}
	defer m.Close()

	// Run migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	if dirty {
		log.Printf("Warning: Migration version %d is dirty", version)
	} else if version > 0 {
		log.Printf("Successfully migrated to version %d", version)
	}

	return nil
}

// RollbackMigration rolls back the last migration
func (r *MigrationRunner) RollbackMigration(ctx context.Context) error {
	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create migration source driver: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", sourceDriver, r.dsn)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}
	defer m.Close()

	if err := m.Steps(-1); err != nil {
		return fmt.Errorf("failed to rollback migration: %w", err)
	}

	version, _, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	log.Printf("Rolled back to version %d", version)
	return nil
}

// GetVersion returns the current migration version
func (r *MigrationRunner) GetVersion() (uint, bool, error) {
	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return 0, false, fmt.Errorf("failed to create migration source driver: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", sourceDriver, r.dsn)
	if err != nil {
		return 0, false, fmt.Errorf("failed to create migration instance: %w", err)
	}
	defer m.Close()

	version, dirty, err := m.Version()
	if err == migrate.ErrNilVersion {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("failed to get migration version: %w", err)
	}

	return version, dirty, nil
}

// InitializeDatabase creates the database and extensions if they don't exist
func InitializeDatabase(ctx context.Context, cfg *Config) error {
	// Connect to postgres database to create our database
	adminConfig := *cfg
	adminConfig.Database = "postgres"
	adminDSN := adminConfig.BuildDSN()

	conn, err := pgx.Connect(ctx, adminDSN)
	if err != nil {
		return fmt.Errorf("failed to connect to postgres database: %w", err)
	}
	defer conn.Close(ctx)

	// Check if database exists
	var exists bool
	err = conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", cfg.Database).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check database existence: %w", err)
	}

	// Create database if it doesn't exist
	if !exists {
		_, err = conn.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", cfg.Database))
		if err != nil {
			return fmt.Errorf("failed to create database: %w", err)
		}
		log.Printf("Created database: %s", cfg.Database)
	}

	// Connect to our database
	pool, err := NewPool(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer pool.Close()

	// Create extensions if they don't exist
	extensions := []string{"timescaledb", "pg_stat_statements"}
	for _, ext := range extensions {
		_, err = pool.Exec(ctx, fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s CASCADE", ext))
		if err != nil {
			// Log warning but don't fail - TimescaleDB might not be available in dev
			log.Printf("Warning: Failed to create extension %s: %v", ext, err)
		}
	}

	// Run migrations
	runner := NewMigrationRunner(pool, cfg.BuildDSN())
	if err := runner.RunMigrations(ctx); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}