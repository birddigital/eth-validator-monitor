package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/birddigital/eth-validator-monitor/internal/database/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DashboardRepository handles dashboard data aggregation queries
type DashboardRepository struct {
	pool *pgxpool.Pool
}

// NewDashboardRepository creates a new dashboard repository
func NewDashboardRepository(pool *pgxpool.Pool) *DashboardRepository {
	return &DashboardRepository{
		pool: pool,
	}
}

// AggregateMetrics represents dashboard-level aggregate metrics
type AggregateMetrics struct {
	TotalValidators   int     `json:"total_validators"`
	ActiveValidators  int     `json:"active_validators"`
	AvgEffectiveness  float64 `json:"avg_effectiveness"`
	TotalBalanceGwei  int64   `json:"total_balance_gwei"`
	SlashedValidators int     `json:"slashed_validators"`
}

// ValidatorSummary represents a top-performing validator
type ValidatorSummary struct {
	ValidatorIndex   int64   `json:"validator_index"`
	Pubkey           string  `json:"pubkey"`
	Name             *string `json:"name,omitempty"`
	Effectiveness    float64 `json:"effectiveness"`
	Balance          int64   `json:"balance"`
	DailyIncome      int64   `json:"daily_income"`
	APR              float64 `json:"apr"`
}

// SystemHealth represents system health indicators
type SystemHealth struct {
	DatabaseStatus    string    `json:"database_status"`
	DataFreshness     string    `json:"data_freshness"`
	LastSnapshotTime  time.Time `json:"last_snapshot_time"`
	MonitoredCount    int       `json:"monitored_count"`
}

// GetAggregateMetrics fetches dashboard-level aggregate statistics
// Uses efficient aggregation query with index on validator_index
func (r *DashboardRepository) GetAggregateMetrics(ctx context.Context) (*AggregateMetrics, error) {
	query := `
		WITH latest_snapshots AS (
			SELECT DISTINCT ON (validator_index)
				validator_index,
				balance,
				attestation_effectiveness
			FROM validator_snapshots
			WHERE time > NOW() - INTERVAL '1 hour'
			ORDER BY validator_index, time DESC
		)
		SELECT
			COUNT(DISTINCT v.validator_index) as total_validators,
			COUNT(DISTINCT v.validator_index) FILTER (WHERE v.monitored = TRUE) as active_validators,
			COALESCE(AVG(ls.attestation_effectiveness), 0) as avg_effectiveness,
			COALESCE(SUM(ls.balance), 0) as total_balance,
			COUNT(DISTINCT v.validator_index) FILTER (WHERE v.slashed = TRUE) as slashed_validators
		FROM validators v
		LEFT JOIN latest_snapshots ls ON v.validator_index = ls.validator_index
		WHERE v.monitored = TRUE
	`

	var metrics AggregateMetrics
	err := r.pool.QueryRow(ctx, query).Scan(
		&metrics.TotalValidators,
		&metrics.ActiveValidators,
		&metrics.AvgEffectiveness,
		&metrics.TotalBalanceGwei,
		&metrics.SlashedValidators,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch aggregate metrics: %w", err)
	}

	return &metrics, nil
}

// GetRecentAlerts fetches the most recent alerts with indexed timestamp query
func (r *DashboardRepository) GetRecentAlerts(ctx context.Context, limit int) ([]*models.Alert, error) {
	query := `
		SELECT
			id, validator_index, alert_type, severity, title, message,
			details, status, acknowledged_at, resolved_at, created_at, updated_at
		FROM alerts
		WHERE status = 'active'
		ORDER BY created_at DESC
		LIMIT $1
	`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recent alerts: %w", err)
	}
	defer rows.Close()

	var alerts []*models.Alert
	for rows.Next() {
		alert := &models.Alert{}
		if err := rows.Scan(
			&alert.ID,
			&alert.ValidatorIndex,
			&alert.AlertType,
			&alert.Severity,
			&alert.Title,
			&alert.Message,
			&alert.Details,
			&alert.Status,
			&alert.AcknowledgedAt,
			&alert.ResolvedAt,
			&alert.CreatedAt,
			&alert.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan alert: %w", err)
		}
		alerts = append(alerts, alert)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating alerts: %w", err)
	}

	return alerts, nil
}

// GetTopValidators fetches top-performing validators by effectiveness
// Uses composite index on (effectiveness, time) for optimal performance
func (r *DashboardRepository) GetTopValidators(ctx context.Context, limit int) ([]*ValidatorSummary, error) {
	query := `
		WITH latest_snapshots AS (
			SELECT DISTINCT ON (validator_index)
				validator_index,
				balance,
				attestation_effectiveness,
				daily_income,
				apr
			FROM validator_snapshots
			WHERE time > NOW() - INTERVAL '1 hour'
				AND attestation_effectiveness IS NOT NULL
			ORDER BY validator_index, time DESC
		)
		SELECT
			v.validator_index,
			v.pubkey,
			v.name,
			COALESCE(ls.attestation_effectiveness, 0) as effectiveness,
			COALESCE(ls.balance, 0) as balance,
			COALESCE(ls.daily_income, 0) as daily_income,
			COALESCE(ls.apr, 0) as apr
		FROM validators v
		INNER JOIN latest_snapshots ls ON v.validator_index = ls.validator_index
		WHERE v.monitored = TRUE
			AND v.slashed = FALSE
		ORDER BY ls.attestation_effectiveness DESC NULLS LAST
		LIMIT $1
	`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch top validators: %w", err)
	}
	defer rows.Close()

	var validators []*ValidatorSummary
	for rows.Next() {
		v := &ValidatorSummary{}
		if err := rows.Scan(
			&v.ValidatorIndex,
			&v.Pubkey,
			&v.Name,
			&v.Effectiveness,
			&v.Balance,
			&v.DailyIncome,
			&v.APR,
		); err != nil {
			return nil, fmt.Errorf("failed to scan validator summary: %w", err)
		}
		validators = append(validators, v)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating validators: %w", err)
	}

	return validators, nil
}

// GetSystemHealth checks various system health indicators
func (r *DashboardRepository) GetSystemHealth(ctx context.Context) (*SystemHealth, error) {
	health := &SystemHealth{
		DatabaseStatus: "healthy",
		DataFreshness:  "unknown",
	}

	// Check database connectivity
	if err := r.pool.Ping(ctx); err != nil {
		health.DatabaseStatus = "unhealthy"
		return health, nil
	}

	// Check data freshness and get latest snapshot time
	var latestSnapshotTime *time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT MAX(time)
		FROM validator_snapshots
	`).Scan(&latestSnapshotTime)

	if err != nil && err != pgx.ErrNoRows {
		health.DataFreshness = "error"
	} else if latestSnapshotTime == nil {
		health.DataFreshness = "no_data"
	} else {
		health.LastSnapshotTime = *latestSnapshotTime
		timeSinceUpdate := time.Since(*latestSnapshotTime)

		if timeSinceUpdate > 15*time.Minute {
			health.DataFreshness = "stale"
		} else if timeSinceUpdate > 5*time.Minute {
			health.DataFreshness = "degraded"
		} else {
			health.DataFreshness = "fresh"
		}
	}

	// Count monitored validators
	err = r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM validators WHERE monitored = TRUE
	`).Scan(&health.MonitoredCount)

	if err != nil {
		health.MonitoredCount = 0
	}

	return health, nil
}
