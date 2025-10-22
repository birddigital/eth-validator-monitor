package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ValidatorDetailRepository handles queries for the validator detail page
type ValidatorDetailRepository struct {
	pool *pgxpool.Pool
}

// NewValidatorDetailRepository creates a new validator detail repository
func NewValidatorDetailRepository(pool *pgxpool.Pool) *ValidatorDetailRepository {
	return &ValidatorDetailRepository{pool: pool}
}

// ValidatorDetails represents comprehensive validator information
type ValidatorDetails struct {
	Index                        int64     `json:"index"`
	Pubkey                       string    `json:"pubkey"`
	Name                         *string   `json:"name"`
	WithdrawalCredentials        string    `json:"withdrawal_credentials"`
	EffectiveBalance             int64     `json:"effective_balance"`
	CurrentBalance               *int64    `json:"current_balance"`
	Slashed                      bool      `json:"slashed"`
	ActivationEpoch              int64     `json:"activation_epoch"`
	ActivationEligibilityEpoch   int64     `json:"activation_eligibility_epoch"`
	ExitEpoch                    *int64    `json:"exit_epoch"`
	WithdrawableEpoch            *int64    `json:"withdrawable_epoch"`
	Monitored                    bool      `json:"monitored"`
	Tags                         []string  `json:"tags"`

	// Latest snapshot data
	AttestationEffectiveness     *float64  `json:"attestation_effectiveness"`
	AttestationInclusionDelay    *float64  `json:"attestation_inclusion_delay"`
	IsOnline                     *bool     `json:"is_online"`
	ConsecutiveMissedAttestations *int     `json:"consecutive_missed_attestations"`
	DailyIncome                  *string   `json:"daily_income"`
	APR                          *float64  `json:"apr"`
	LastUpdate                   *time.Time `json:"last_update"`
}

// EffectivenessPoint represents a point in the effectiveness timeline
type EffectivenessPoint struct {
	Date     time.Time `json:"date"`
	AvgScore float64   `json:"avg_score"`
	MinScore float64   `json:"min_score"`
	MaxScore float64   `json:"max_score"`
}

// AttestationStats represents monthly attestation statistics
type AttestationStats struct {
	Month              time.Time `json:"month"`
	TotalAttestations  int       `json:"total_attestations"`
	SuccessfulVotes    int       `json:"successful_votes"`
	MissedVotes        int       `json:"missed_votes"`
	AvgInclusionDelay  float64   `json:"avg_inclusion_delay"`
}

// Alert represents an alert record
type Alert struct {
	ID          int32           `json:"id"`
	Type        string          `json:"alert_type"`
	Severity    string          `json:"severity"`
	Title       string          `json:"title"`
	Message     string          `json:"message"`
	Details     json.RawMessage `json:"details"`
	Status      string          `json:"status"`
	CreatedAt   time.Time       `json:"created_at"`
	ResolvedAt  *time.Time      `json:"resolved_at"`
}

// TimelineEvent represents a validator lifecycle event
type TimelineEvent struct {
	Type        string    `json:"type"`
	Epoch       *int64    `json:"epoch"`
	Slot        *int64    `json:"slot"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
}

// GetValidatorDetails retrieves comprehensive validator metadata with latest snapshot
func (r *ValidatorDetailRepository) GetValidatorDetails(ctx context.Context, validatorIndex int64) (*ValidatorDetails, error) {
	query := `
		SELECT
			v.validator_index,
			v.pubkey,
			v.name,
			v.withdrawal_credentials,
			v.effective_balance,
			v.slashed,
			v.activation_epoch,
			v.activation_eligibility_epoch,
			v.exit_epoch,
			v.withdrawable_epoch,
			v.monitored,
			v.tags,
			vs.balance as current_balance,
			vs.attestation_effectiveness,
			vs.attestation_inclusion_delay,
			vs.is_online,
			vs.consecutive_missed_attestations,
			vs.daily_income,
			vs.apr,
			vs.time as last_update
		FROM validators v
		LEFT JOIN LATERAL (
			SELECT *
			FROM validator_snapshots
			WHERE validator_index = v.validator_index
			ORDER BY time DESC
			LIMIT 1
		) vs ON true
		WHERE v.validator_index = $1
	`

	var details ValidatorDetails
	err := r.pool.QueryRow(ctx, query, validatorIndex).Scan(
		&details.Index,
		&details.Pubkey,
		&details.Name,
		&details.WithdrawalCredentials,
		&details.EffectiveBalance,
		&details.Slashed,
		&details.ActivationEpoch,
		&details.ActivationEligibilityEpoch,
		&details.ExitEpoch,
		&details.WithdrawableEpoch,
		&details.Monitored,
		&details.Tags,
		&details.CurrentBalance,
		&details.AttestationEffectiveness,
		&details.AttestationInclusionDelay,
		&details.IsOnline,
		&details.ConsecutiveMissedAttestations,
		&details.DailyIncome,
		&details.APR,
		&details.LastUpdate,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get validator details: %w", err)
	}

	return &details, nil
}

// GetEffectivenessHistory returns N-day effectiveness data for Chart.js
func (r *ValidatorDetailRepository) GetEffectivenessHistory(ctx context.Context, validatorIndex int64, days int) ([]EffectivenessPoint, error) {
	query := `
		SELECT
			DATE(time) as date,
			AVG(COALESCE(attestation_effectiveness, 0)) as avg_score,
			MIN(COALESCE(attestation_effectiveness, 0)) as min_score,
			MAX(COALESCE(attestation_effectiveness, 0)) as max_score
		FROM validator_snapshots
		WHERE validator_index = $1
		  AND time >= NOW() - INTERVAL '1 day' * $2
		  AND attestation_effectiveness IS NOT NULL
		GROUP BY DATE(time)
		ORDER BY date ASC
	`

	rows, err := r.pool.Query(ctx, query, validatorIndex, days)
	if err != nil {
		return nil, fmt.Errorf("failed to query effectiveness history: %w", err)
	}
	defer rows.Close()

	var points []EffectivenessPoint
	for rows.Next() {
		var p EffectivenessPoint
		if err := rows.Scan(&p.Date, &p.AvgScore, &p.MinScore, &p.MaxScore); err != nil {
			return nil, fmt.Errorf("failed to scan effectiveness point: %w", err)
		}
		points = append(points, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating effectiveness rows: %w", err)
	}

	return points, nil
}

// GetAttestationStats returns monthly attestation statistics
// Note: This is a placeholder implementation since the schema doesn't have an attestations table yet
// We'll derive stats from validator_snapshots for now
func (r *ValidatorDetailRepository) GetAttestationStats(ctx context.Context, validatorIndex int64, months int) ([]AttestationStats, error) {
	query := `
		SELECT
			DATE_TRUNC('month', time) as month,
			COUNT(*) as total_snapshots,
			SUM(CASE WHEN attestation_head_vote THEN 1 ELSE 0 END) as successful_votes,
			SUM(CASE WHEN NOT attestation_head_vote THEN 1 ELSE 0 END) as missed_votes,
			AVG(COALESCE(attestation_inclusion_delay, 0)) as avg_inclusion_delay
		FROM validator_snapshots
		WHERE validator_index = $1
		  AND time >= NOW() - INTERVAL '1 month' * $2
		GROUP BY DATE_TRUNC('month', time)
		ORDER BY month DESC
	`

	rows, err := r.pool.Query(ctx, query, validatorIndex, months)
	if err != nil {
		return nil, fmt.Errorf("failed to query attestation stats: %w", err)
	}
	defer rows.Close()

	var stats []AttestationStats
	for rows.Next() {
		var s AttestationStats
		if err := rows.Scan(&s.Month, &s.TotalAttestations, &s.SuccessfulVotes, &s.MissedVotes, &s.AvgInclusionDelay); err != nil {
			return nil, fmt.Errorf("failed to scan attestation stats: %w", err)
		}
		stats = append(stats, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating attestation stats rows: %w", err)
	}

	return stats, nil
}

// GetRecentAlerts returns the last N alerts for the validator
func (r *ValidatorDetailRepository) GetRecentAlerts(ctx context.Context, validatorIndex int64, limit int) ([]Alert, error) {
	query := `
		SELECT
			id,
			alert_type,
			severity,
			title,
			message,
			details,
			status,
			created_at,
			resolved_at
		FROM alerts
		WHERE validator_index = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, validatorIndex, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent alerts: %w", err)
	}
	defer rows.Close()

	var alerts []Alert
	for rows.Next() {
		var a Alert
		if err := rows.Scan(&a.ID, &a.Type, &a.Severity, &a.Title, &a.Message, &a.Details, &a.Status, &a.CreatedAt, &a.ResolvedAt); err != nil {
			return nil, fmt.Errorf("failed to scan alert: %w", err)
		}
		alerts = append(alerts, a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating alert rows: %w", err)
	}

	return alerts, nil
}

// GetValidatorTimeline returns key lifecycle events from snapshots
// Note: Since there's no validator_events table, we'll create a timeline from snapshots
func (r *ValidatorDetailRepository) GetValidatorTimeline(ctx context.Context, validatorIndex int64) ([]TimelineEvent, error) {
	query := `
		SELECT
			CASE
				WHEN is_online AND NOT LAG(is_online, 1, false) OVER (ORDER BY time) THEN 'came_online'
				WHEN NOT is_online AND LAG(is_online, 1, true) OVER (ORDER BY time) THEN 'went_offline'
				WHEN proposals_executed > LAG(proposals_executed, 1, 0) OVER (ORDER BY time) THEN 'proposed_block'
				WHEN proposals_missed > LAG(proposals_missed, 1, 0) OVER (ORDER BY time) THEN 'missed_proposal'
				ELSE 'snapshot'
			END as event_type,
			CASE
				WHEN is_online AND NOT LAG(is_online, 1, false) OVER (ORDER BY time) THEN 'Validator came online'
				WHEN NOT is_online AND LAG(is_online, 1, true) OVER (ORDER BY time) THEN 'Validator went offline'
				WHEN proposals_executed > LAG(proposals_executed, 1, 0) OVER (ORDER BY time) THEN 'Successfully proposed block'
				WHEN proposals_missed > LAG(proposals_missed, 1, 0) OVER (ORDER BY time) THEN 'Missed block proposal'
				ELSE 'Status snapshot'
			END as description,
			time as timestamp
		FROM validator_snapshots
		WHERE validator_index = $1
		  AND time >= NOW() - INTERVAL '30 days'
		ORDER BY time DESC
		LIMIT 50
	`

	rows, err := r.pool.Query(ctx, query, validatorIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to query validator timeline: %w", err)
	}
	defer rows.Close()

	var events []TimelineEvent
	for rows.Next() {
		var e TimelineEvent
		if err := rows.Scan(&e.Type, &e.Description, &e.Timestamp); err != nil {
			return nil, fmt.Errorf("failed to scan timeline event: %w", err)
		}
		// Only include meaningful events, skip routine snapshots
		if e.Type != "snapshot" {
			events = append(events, e)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating timeline rows: %w", err)
	}

	return events, nil
}
