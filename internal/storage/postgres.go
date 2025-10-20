package storage

import (
	"context"
	"database/sql"
	"fmt"
	"math/big"
	"time"

	_ "github.com/lib/pq"
	"github.com/birddigital/eth-validator-monitor/pkg/types"
)

// PostgresStorage implements the Storage interface using PostgreSQL
type PostgresStorage struct {
	db               *sql.DB
	batchSize        int
	retentionDays    int
	snapshotBuffer   chan *types.ValidatorSnapshot
	performanceBuffer chan *types.PerformanceMetrics
	alertBuffer      chan *types.Alert
	ctx              context.Context
	cancel           context.CancelFunc
}

// Config holds the PostgreSQL storage configuration
type Config struct {
	Host          string
	Port          int
	User          string
	Password      string
	Database      string
	MaxConns      int
	MaxIdleConns  int
	BatchSize     int
	RetentionDays int
	FlushInterval time.Duration
}

// NewPostgresStorage creates a new PostgreSQL storage instance
func NewPostgresStorage(cfg Config) (*PostgresStorage, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(time.Hour)

	// Verify connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	storage := &PostgresStorage{
		db:               db,
		batchSize:        cfg.BatchSize,
		retentionDays:    cfg.RetentionDays,
		snapshotBuffer:   make(chan *types.ValidatorSnapshot, cfg.BatchSize*2),
		performanceBuffer: make(chan *types.PerformanceMetrics, cfg.BatchSize*2),
		alertBuffer:      make(chan *types.Alert, cfg.BatchSize*2),
		ctx:              ctx,
		cancel:           cancel,
	}

	// Start background flush workers
	go storage.flushSnapshots(cfg.FlushInterval)
	go storage.flushPerformance(cfg.FlushInterval)
	go storage.flushAlerts(cfg.FlushInterval)

	return storage, nil
}

// Close closes the database connection and stops all workers
func (s *PostgresStorage) Close() error {
	s.cancel()

	// Wait a bit for buffers to flush
	time.Sleep(2 * time.Second)

	close(s.snapshotBuffer)
	close(s.performanceBuffer)
	close(s.alertBuffer)

	return s.db.Close()
}

// SaveValidator stores or updates a validator record
func (s *PostgresStorage) SaveValidator(ctx context.Context, validator *types.Validator) error {
	query := `
		INSERT INTO validators (index, pubkey, name, status, activation_epoch, exit_epoch, slashed, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (index) DO UPDATE SET
			pubkey = EXCLUDED.pubkey,
			name = EXCLUDED.name,
			status = EXCLUDED.status,
			activation_epoch = EXCLUDED.activation_epoch,
			exit_epoch = EXCLUDED.exit_epoch,
			slashed = EXCLUDED.slashed,
			updated_at = EXCLUDED.updated_at
	`

	_, err := s.db.ExecContext(
		ctx, query,
		validator.Index,
		validator.Pubkey,
		validator.Name,
		validator.Status,
		validator.ActivationEpoch,
		validator.ExitEpoch,
		validator.Slashed,
		validator.CreatedAt,
		time.Now(),
	)

	return err
}

// GetValidator retrieves a validator by index
func (s *PostgresStorage) GetValidator(ctx context.Context, index int) (*types.Validator, error) {
	query := `
		SELECT index, pubkey, name, status, activation_epoch, exit_epoch, slashed, created_at, updated_at
		FROM validators
		WHERE index = $1
	`

	var v types.Validator
	err := s.db.QueryRowContext(ctx, query, index).Scan(
		&v.Index,
		&v.Pubkey,
		&v.Name,
		&v.Status,
		&v.ActivationEpoch,
		&v.ExitEpoch,
		&v.Slashed,
		&v.CreatedAt,
		&v.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("validator %d not found: %w", index, err)
	}

	return &v, err
}

// SaveSnapshot buffers a validator snapshot for batch insert
func (s *PostgresStorage) SaveSnapshot(ctx context.Context, snapshot *types.ValidatorSnapshot) error {
	select {
	case s.snapshotBuffer <- snapshot:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Buffer full, force immediate flush
		return s.insertSnapshot(ctx, snapshot)
	}
}

// insertSnapshot performs the actual database insert
func (s *PostgresStorage) insertSnapshot(ctx context.Context, snapshot *types.ValidatorSnapshot) error {
	query := `
		INSERT INTO validator_snapshots (
			validator_index, epoch, slot, balance, effective_balance, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := s.db.ExecContext(
		ctx, query,
		snapshot.ValidatorIndex,
		snapshot.Epoch,
		snapshot.Slot,
		snapshot.Balance.String(),
		snapshot.EffectiveBalance.String(),
		snapshot.Timestamp,
	)

	return err
}

// flushSnapshots periodically flushes buffered snapshots
func (s *PostgresStorage) flushSnapshots(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	batch := make([]*types.ValidatorSnapshot, 0, s.batchSize)

	for {
		select {
		case <-s.ctx.Done():
			// Final flush
			if len(batch) > 0 {
				s.batchInsertSnapshots(context.Background(), batch)
			}
			return

		case snapshot := <-s.snapshotBuffer:
			batch = append(batch, snapshot)
			if len(batch) >= s.batchSize {
				s.batchInsertSnapshots(s.ctx, batch)
				batch = batch[:0]
			}

		case <-ticker.C:
			if len(batch) > 0 {
				s.batchInsertSnapshots(s.ctx, batch)
				batch = batch[:0]
			}
		}
	}
}

// batchInsertSnapshots performs batch insert of snapshots
func (s *PostgresStorage) batchInsertSnapshots(ctx context.Context, snapshots []*types.ValidatorSnapshot) error {
	if len(snapshots) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO validator_snapshots (
			validator_index, epoch, slot, balance, effective_balance, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, snapshot := range snapshots {
		_, err := stmt.ExecContext(
			ctx,
			snapshot.ValidatorIndex,
			snapshot.Epoch,
			snapshot.Slot,
			snapshot.Balance.String(),
			snapshot.EffectiveBalance.String(),
			snapshot.Timestamp,
		)
		if err != nil {
			return fmt.Errorf("failed to insert snapshot: %w", err)
		}
	}

	return tx.Commit()
}

// SavePerformance buffers performance metrics for batch insert
func (s *PostgresStorage) SavePerformance(ctx context.Context, perf *types.PerformanceMetrics) error {
	select {
	case s.performanceBuffer <- perf:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return s.insertPerformance(ctx, perf)
	}
}

// insertPerformance performs the actual database insert
func (s *PostgresStorage) insertPerformance(ctx context.Context, perf *types.PerformanceMetrics) error {
	query := `
		INSERT INTO validator_performance (
			validator_index, epoch, attestation_score, proposal_score,
			sync_committee_score, overall_score, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (validator_index, epoch) DO UPDATE SET
			attestation_score = EXCLUDED.attestation_score,
			proposal_score = EXCLUDED.proposal_score,
			sync_committee_score = EXCLUDED.sync_committee_score,
			overall_score = EXCLUDED.overall_score,
			timestamp = EXCLUDED.timestamp
	`

	_, err := s.db.ExecContext(
		ctx, query,
		perf.ValidatorIndex,
		perf.Epoch,
		perf.AttestationScore,
		perf.ProposalScore,
		perf.SyncCommitteeScore,
		perf.OverallScore,
		perf.Timestamp,
	)

	return err
}

// flushPerformance periodically flushes buffered performance metrics
func (s *PostgresStorage) flushPerformance(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	batch := make([]*types.PerformanceMetrics, 0, s.batchSize)

	for {
		select {
		case <-s.ctx.Done():
			if len(batch) > 0 {
				s.batchInsertPerformance(context.Background(), batch)
			}
			return

		case perf := <-s.performanceBuffer:
			batch = append(batch, perf)
			if len(batch) >= s.batchSize {
				s.batchInsertPerformance(s.ctx, batch)
				batch = batch[:0]
			}

		case <-ticker.C:
			if len(batch) > 0 {
				s.batchInsertPerformance(s.ctx, batch)
				batch = batch[:0]
			}
		}
	}
}

// batchInsertPerformance performs batch insert of performance metrics
func (s *PostgresStorage) batchInsertPerformance(ctx context.Context, metrics []*types.PerformanceMetrics) error {
	if len(metrics) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO validator_performance (
			validator_index, epoch, attestation_score, proposal_score,
			sync_committee_score, overall_score, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (validator_index, epoch) DO UPDATE SET
			attestation_score = EXCLUDED.attestation_score,
			proposal_score = EXCLUDED.proposal_score,
			sync_committee_score = EXCLUDED.sync_committee_score,
			overall_score = EXCLUDED.overall_score,
			timestamp = EXCLUDED.timestamp
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, perf := range metrics {
		_, err := stmt.ExecContext(
			ctx,
			perf.ValidatorIndex,
			perf.Epoch,
			perf.AttestationScore,
			perf.ProposalScore,
			perf.SyncCommitteeScore,
			perf.OverallScore,
			perf.Timestamp,
		)
		if err != nil {
			return fmt.Errorf("failed to insert performance: %w", err)
		}
	}

	return tx.Commit()
}

// SaveAlert buffers an alert for batch insert
func (s *PostgresStorage) SaveAlert(ctx context.Context, alert *types.Alert) error {
	select {
	case s.alertBuffer <- alert:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return s.insertAlert(ctx, alert)
	}
}

// insertAlert performs the actual database insert
func (s *PostgresStorage) insertAlert(ctx context.Context, alert *types.Alert) error {
	query := `
		INSERT INTO alerts (
			validator_index, severity, message, triggered_at, acknowledged
		) VALUES ($1, $2, $3, $4, $5)
	`

	_, err := s.db.ExecContext(
		ctx, query,
		alert.ValidatorIndex,
		alert.Severity,
		alert.Message,
		alert.CreatedAt,
		alert.Acknowledged,
	)

	return err
}

// flushAlerts periodically flushes buffered alerts
func (s *PostgresStorage) flushAlerts(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	batch := make([]*types.Alert, 0, s.batchSize)

	for {
		select {
		case <-s.ctx.Done():
			if len(batch) > 0 {
				s.batchInsertAlerts(context.Background(), batch)
			}
			return

		case alert := <-s.alertBuffer:
			batch = append(batch, alert)
			if len(batch) >= s.batchSize {
				s.batchInsertAlerts(s.ctx, batch)
				batch = batch[:0]
			}

		case <-ticker.C:
			if len(batch) > 0 {
				s.batchInsertAlerts(s.ctx, batch)
				batch = batch[:0]
			}
		}
	}
}

// batchInsertAlerts performs batch insert of alerts
func (s *PostgresStorage) batchInsertAlerts(ctx context.Context, alerts []*types.Alert) error {
	if len(alerts) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO alerts (
			validator_index, severity, message, triggered_at, acknowledged
		) VALUES ($1, $2, $3, $4, $5)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, alert := range alerts {
		_, err := stmt.ExecContext(
			ctx,
			alert.ValidatorIndex,
			alert.Severity,
			alert.Message,
			alert.CreatedAt,
			alert.Acknowledged,
		)
		if err != nil {
			return fmt.Errorf("failed to insert alert: %w", err)
		}
	}

	return tx.Commit()
}

// CleanupOldSnapshots removes snapshots older than the retention period
func (s *PostgresStorage) CleanupOldSnapshots(ctx context.Context) error {
	query := `
		DELETE FROM validator_snapshots
		WHERE timestamp < NOW() - INTERVAL '%d days'
	`

	result, err := s.db.ExecContext(ctx, fmt.Sprintf(query, s.retentionDays))
	if err != nil {
		return fmt.Errorf("failed to cleanup old snapshots: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows > 0 {
		fmt.Printf("Cleaned up %d old snapshot records\n", rows)
	}

	return nil
}

// GetValidatorHistory retrieves historical snapshots for a validator
func (s *PostgresStorage) GetValidatorHistory(ctx context.Context, index int, from, to time.Time) ([]*types.ValidatorSnapshot, error) {
	query := `
		SELECT validator_index, epoch, slot, balance, effective_balance, timestamp
		FROM validator_snapshots
		WHERE validator_index = $1
		  AND timestamp >= $2
		  AND timestamp <= $3
		ORDER BY timestamp DESC
	`

	rows, err := s.db.QueryContext(ctx, query, index, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snapshots []*types.ValidatorSnapshot
	for rows.Next() {
		var s types.ValidatorSnapshot
		var balance, effectiveBalance string

		err := rows.Scan(
			&s.ValidatorIndex,
			&s.Epoch,
			&s.Slot,
			&balance,
			&effectiveBalance,
			&s.Timestamp,
		)
		if err != nil {
			return nil, err
		}

		// Parse big.Int values
		s.Balance = new(big.Int)
		s.Balance.SetString(balance, 10)
		s.EffectiveBalance = new(big.Int)
		s.EffectiveBalance.SetString(effectiveBalance, 10)

		snapshots = append(snapshots, &s)
	}

	return snapshots, rows.Err()
}
