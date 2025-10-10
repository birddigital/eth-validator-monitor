package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/birddigital/eth-validator-monitor/internal/database/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SnapshotRepository handles validator snapshot database operations
type SnapshotRepository struct {
	pool *pgxpool.Pool
}

// NewSnapshotRepository creates a new snapshot repository
func NewSnapshotRepository(pool *pgxpool.Pool) *SnapshotRepository {
	return &SnapshotRepository{
		pool: pool,
	}
}

// InsertSnapshot inserts a single validator snapshot
func (r *SnapshotRepository) InsertSnapshot(ctx context.Context, snapshot *models.ValidatorSnapshot) error {
	query := `
		INSERT INTO validator_snapshots (
			time, validator_index, balance, effective_balance,
			attestation_effectiveness, attestation_inclusion_delay,
			attestation_head_vote, attestation_source_vote, attestation_target_vote,
			proposals_scheduled, proposals_executed, proposals_missed,
			sync_committee_participation, slashed, is_online,
			consecutive_missed_attestations, daily_income, apr
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)`

	_, err := r.pool.Exec(ctx, query,
		snapshot.Time,
		snapshot.ValidatorIndex,
		snapshot.Balance,
		snapshot.EffectiveBalance,
		snapshot.AttestationEffectiveness,
		snapshot.AttestationInclusionDelay,
		snapshot.AttestationHeadVote,
		snapshot.AttestationSourceVote,
		snapshot.AttestationTargetVote,
		snapshot.ProposalsScheduled,
		snapshot.ProposalsExecuted,
		snapshot.ProposalsMissed,
		snapshot.SyncCommitteeParticipation,
		snapshot.Slashed,
		snapshot.IsOnline,
		snapshot.ConsecutiveMissedAttestations,
		snapshot.DailyIncome,
		snapshot.APR,
	)

	if err != nil {
		return fmt.Errorf("failed to insert snapshot: %w", err)
	}

	return nil
}

// BatchInsertSnapshots efficiently inserts multiple snapshots using COPY
func (r *SnapshotRepository) BatchInsertSnapshots(ctx context.Context, snapshots []*models.ValidatorSnapshot) error {
	if len(snapshots) == 0 {
		return nil
	}

	// Use COPY for maximum performance with TimescaleDB
	copyFrom := pgx.CopyFromSlice(len(snapshots), func(i int) ([]interface{}, error) {
		s := snapshots[i]
		return []interface{}{
			s.Time,
			s.ValidatorIndex,
			s.Balance,
			s.EffectiveBalance,
			s.AttestationEffectiveness,
			s.AttestationInclusionDelay,
			s.AttestationHeadVote,
			s.AttestationSourceVote,
			s.AttestationTargetVote,
			s.ProposalsScheduled,
			s.ProposalsExecuted,
			s.ProposalsMissed,
			s.SyncCommitteeParticipation,
			s.Slashed,
			s.IsOnline,
			s.ConsecutiveMissedAttestations,
			s.DailyIncome,
			s.APR,
		}, nil
	})

	_, err := r.pool.CopyFrom(
		ctx,
		[]string{"validator_snapshots"},
		[]string{
			"time", "validator_index", "balance", "effective_balance",
			"attestation_effectiveness", "attestation_inclusion_delay",
			"attestation_head_vote", "attestation_source_vote", "attestation_target_vote",
			"proposals_scheduled", "proposals_executed", "proposals_missed",
			"sync_committee_participation", "slashed", "is_online",
			"consecutive_missed_attestations", "daily_income", "apr",
		},
		copyFrom,
	)

	if err != nil {
		return fmt.Errorf("failed to batch insert snapshots: %w", err)
	}

	return nil
}

// GetLatestSnapshot retrieves the most recent snapshot for a validator
func (r *SnapshotRepository) GetLatestSnapshot(ctx context.Context, validatorIndex int64) (*models.ValidatorSnapshot, error) {
	snapshot := &models.ValidatorSnapshot{}
	query := `
		SELECT time, validator_index, balance, effective_balance,
			   attestation_effectiveness, attestation_inclusion_delay,
			   attestation_head_vote, attestation_source_vote, attestation_target_vote,
			   proposals_scheduled, proposals_executed, proposals_missed,
			   sync_committee_participation, slashed, is_online,
			   consecutive_missed_attestations, daily_income, apr
		FROM validator_snapshots
		WHERE validator_index = $1
		ORDER BY time DESC
		LIMIT 1`

	err := r.pool.QueryRow(ctx, query, validatorIndex).Scan(
		&snapshot.Time,
		&snapshot.ValidatorIndex,
		&snapshot.Balance,
		&snapshot.EffectiveBalance,
		&snapshot.AttestationEffectiveness,
		&snapshot.AttestationInclusionDelay,
		&snapshot.AttestationHeadVote,
		&snapshot.AttestationSourceVote,
		&snapshot.AttestationTargetVote,
		&snapshot.ProposalsScheduled,
		&snapshot.ProposalsExecuted,
		&snapshot.ProposalsMissed,
		&snapshot.SyncCommitteeParticipation,
		&snapshot.Slashed,
		&snapshot.IsOnline,
		&snapshot.ConsecutiveMissedAttestations,
		&snapshot.DailyIncome,
		&snapshot.APR,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest snapshot: %w", err)
	}

	return snapshot, nil
}

// GetSnapshots retrieves snapshots with filtering
func (r *SnapshotRepository) GetSnapshots(ctx context.Context, filter *models.SnapshotFilter) ([]*models.ValidatorSnapshot, error) {
	query := strings.Builder{}
	query.WriteString(`
		SELECT time, validator_index, balance, effective_balance,
			   attestation_effectiveness, attestation_inclusion_delay,
			   attestation_head_vote, attestation_source_vote, attestation_target_vote,
			   proposals_scheduled, proposals_executed, proposals_missed,
			   sync_committee_participation, slashed, is_online,
			   consecutive_missed_attestations, daily_income, apr
		FROM validator_snapshots
		WHERE validator_index = $1`)

	args := []interface{}{filter.ValidatorIndex}
	argCount := 1

	if filter.StartTime != nil {
		argCount++
		query.WriteString(fmt.Sprintf(" AND time >= $%d", argCount))
		args = append(args, *filter.StartTime)
	}

	if filter.EndTime != nil {
		argCount++
		query.WriteString(fmt.Sprintf(" AND time <= $%d", argCount))
		args = append(args, *filter.EndTime)
	}

	query.WriteString(" ORDER BY time DESC")

	if filter.Limit > 0 {
		argCount++
		query.WriteString(fmt.Sprintf(" LIMIT $%d", argCount))
		args = append(args, filter.Limit)
	}

	if filter.Offset > 0 {
		argCount++
		query.WriteString(fmt.Sprintf(" OFFSET $%d", argCount))
		args = append(args, filter.Offset)
	}

	rows, err := r.pool.Query(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshots: %w", err)
	}
	defer rows.Close()

	snapshots := []*models.ValidatorSnapshot{}
	for rows.Next() {
		snapshot := &models.ValidatorSnapshot{}
		err := rows.Scan(
			&snapshot.Time,
			&snapshot.ValidatorIndex,
			&snapshot.Balance,
			&snapshot.EffectiveBalance,
			&snapshot.AttestationEffectiveness,
			&snapshot.AttestationInclusionDelay,
			&snapshot.AttestationHeadVote,
			&snapshot.AttestationSourceVote,
			&snapshot.AttestationTargetVote,
			&snapshot.ProposalsScheduled,
			&snapshot.ProposalsExecuted,
			&snapshot.ProposalsMissed,
			&snapshot.SyncCommitteeParticipation,
			&snapshot.Slashed,
			&snapshot.IsOnline,
			&snapshot.ConsecutiveMissedAttestations,
			&snapshot.DailyIncome,
			&snapshot.APR,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan snapshot: %w", err)
		}
		snapshots = append(snapshots, snapshot)
	}

	return snapshots, nil
}

// GetAggregatedStats retrieves aggregated statistics for a validator
func (r *SnapshotRepository) GetAggregatedStats(ctx context.Context, validatorIndex int64, interval string, startTime, endTime time.Time) (map[string]interface{}, error) {
	var query string

	switch interval {
	case "hourly":
		query = `
			SELECT
				time_bucket('1 hour', time) AS bucket,
				AVG(balance)::BIGINT as avg_balance,
				AVG(attestation_effectiveness) as avg_effectiveness,
				SUM(CASE WHEN attestation_effectiveness < 95 THEN 1 ELSE 0 END) as suboptimal_count
			FROM validator_snapshots
			WHERE validator_index = $1 AND time >= $2 AND time <= $3
			GROUP BY bucket
			ORDER BY bucket DESC`
	case "daily":
		query = `
			SELECT
				time_bucket('1 day', time) AS bucket,
				AVG(balance)::BIGINT as avg_balance,
				MIN(balance)::BIGINT as min_balance,
				MAX(balance)::BIGINT as max_balance,
				AVG(attestation_effectiveness) as avg_effectiveness,
				SUM(CASE WHEN attestation_effectiveness < 95 THEN 1 ELSE 0 END) as suboptimal_count
			FROM validator_snapshots
			WHERE validator_index = $1 AND time >= $2 AND time <= $3
			GROUP BY bucket
			ORDER BY bucket DESC`
	default:
		return nil, fmt.Errorf("unsupported interval: %s", interval)
	}

	rows, err := r.pool.Query(ctx, query, validatorIndex, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get aggregated stats: %w", err)
	}
	defer rows.Close()

	results := []map[string]interface{}{}
	for rows.Next() {
		// Use dynamic scanning based on interval type
		if interval == "hourly" {
			var bucket time.Time
			var avgBalance int64
			var avgEffectiveness float64
			var suboptimalCount int

			err := rows.Scan(&bucket, &avgBalance, &avgEffectiveness, &suboptimalCount)
			if err != nil {
				return nil, fmt.Errorf("failed to scan hourly stats: %w", err)
			}

			results = append(results, map[string]interface{}{
				"bucket":            bucket,
				"avg_balance":       avgBalance,
				"avg_effectiveness": avgEffectiveness,
				"suboptimal_count":  suboptimalCount,
			})
		} else if interval == "daily" {
			var bucket time.Time
			var avgBalance, minBalance, maxBalance int64
			var avgEffectiveness float64
			var suboptimalCount int

			err := rows.Scan(&bucket, &avgBalance, &minBalance, &maxBalance, &avgEffectiveness, &suboptimalCount)
			if err != nil {
				return nil, fmt.Errorf("failed to scan daily stats: %w", err)
			}

			results = append(results, map[string]interface{}{
				"bucket":            bucket,
				"avg_balance":       avgBalance,
				"min_balance":       minBalance,
				"max_balance":       maxBalance,
				"avg_effectiveness": avgEffectiveness,
				"suboptimal_count":  suboptimalCount,
			})
		}
	}

	return map[string]interface{}{
		"interval": interval,
		"data":     results,
	}, nil
}

// CalculateEffectivenessScore calculates validator effectiveness using Jim McDonald's formula
func CalculateEffectivenessScore(headVote, sourceVote, targetVote bool, inclusionDelay int32) float64 {
	score := 0.0

	// Head vote: 25% weight
	if headVote {
		score += 25.0
	}

	// Source vote: 25% weight
	if sourceVote {
		score += 25.0
	}

	// Target vote: 25% weight
	if targetVote {
		score += 25.0
	}

	// Inclusion delay: 25% weight
	// Perfect inclusion delay is 1, maximum penalty at delay >= 5
	if inclusionDelay > 0 {
		inclusionScore := 25.0
		if inclusionDelay > 1 {
			// Reduce score based on delay
			penalty := float64(inclusionDelay-1) * 6.25 // 6.25% per slot delay
			inclusionScore = inclusionScore - penalty
			if inclusionScore < 0 {
				inclusionScore = 0
			}
		}
		score += inclusionScore
	}

	return score
}

// GetRecentSnapshots retrieves recent snapshots for a validator
func (r *SnapshotRepository) GetRecentSnapshots(ctx context.Context, validatorIndex int64, limit int) ([]*models.ValidatorSnapshot, error) {
	filter := &models.SnapshotFilter{
		ValidatorIndex: validatorIndex,
		Limit:          limit,
	}
	return r.GetSnapshots(ctx, filter)
}