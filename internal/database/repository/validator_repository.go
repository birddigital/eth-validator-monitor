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

// ValidatorRepository handles validator database operations
type ValidatorRepository struct {
	pool *pgxpool.Pool
}

// NewValidatorRepository creates a new validator repository
func NewValidatorRepository(pool *pgxpool.Pool) *ValidatorRepository {
	return &ValidatorRepository{
		pool: pool,
	}
}

// CreateValidator inserts a new validator
func (r *ValidatorRepository) CreateValidator(ctx context.Context, validator *models.Validator) error {
	query := `
		INSERT INTO validators (
			validator_index, pubkey, withdrawal_credentials, effective_balance,
			slashed, activation_epoch, activation_eligibility_epoch, exit_epoch,
			withdrawable_epoch, name, tags, monitored
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at`

	err := r.pool.QueryRow(ctx, query,
		validator.ValidatorIndex,
		validator.Pubkey,
		validator.WithdrawalCredentials,
		validator.EffectiveBalance,
		validator.Slashed,
		validator.ActivationEpoch,
		validator.ActivationEligibilityEpoch,
		validator.ExitEpoch,
		validator.WithdrawableEpoch,
		validator.Name,
		validator.Tags,
		validator.Monitored,
	).Scan(&validator.ID, &validator.CreatedAt, &validator.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create validator: %w", err)
	}

	return nil
}

// BatchCreateValidators inserts multiple validators efficiently
func (r *ValidatorRepository) BatchCreateValidators(ctx context.Context, validators []*models.Validator) error {
	if len(validators) == 0 {
		return nil
	}

	// Use COPY for best performance with large batches
	copyFrom := pgx.CopyFromSlice(len(validators), func(i int) ([]interface{}, error) {
		v := validators[i]
		return []interface{}{
			v.ValidatorIndex,
			v.Pubkey,
			v.WithdrawalCredentials,
			v.EffectiveBalance,
			v.Slashed,
			v.ActivationEpoch,
			v.ActivationEligibilityEpoch,
			v.ExitEpoch,
			v.WithdrawableEpoch,
			v.Name,
			v.Tags,
			v.Monitored,
			time.Now(),
			time.Now(),
		}, nil
	})

	_, err := r.pool.CopyFrom(
		ctx,
		[]string{"validators"},
		[]string{
			"validator_index", "pubkey", "withdrawal_credentials", "effective_balance",
			"slashed", "activation_epoch", "activation_eligibility_epoch", "exit_epoch",
			"withdrawable_epoch", "name", "tags", "monitored", "created_at", "updated_at",
		},
		copyFrom,
	)

	if err != nil {
		return fmt.Errorf("failed to batch create validators: %w", err)
	}

	return nil
}

// GetValidatorByIndex retrieves a validator by index
func (r *ValidatorRepository) GetValidatorByIndex(ctx context.Context, index int64) (*models.Validator, error) {
	validator := &models.Validator{}
	query := `
		SELECT id, validator_index, pubkey, withdrawal_credentials, effective_balance,
			   slashed, activation_epoch, activation_eligibility_epoch, exit_epoch,
			   withdrawable_epoch, name, tags, monitored, created_at, updated_at
		FROM validators
		WHERE validator_index = $1`

	err := r.pool.QueryRow(ctx, query, index).Scan(
		&validator.ID,
		&validator.ValidatorIndex,
		&validator.Pubkey,
		&validator.WithdrawalCredentials,
		&validator.EffectiveBalance,
		&validator.Slashed,
		&validator.ActivationEpoch,
		&validator.ActivationEligibilityEpoch,
		&validator.ExitEpoch,
		&validator.WithdrawableEpoch,
		&validator.Name,
		&validator.Tags,
		&validator.Monitored,
		&validator.CreatedAt,
		&validator.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get validator: %w", err)
	}

	return validator, nil
}

// ListValidators retrieves validators with filtering
func (r *ValidatorRepository) ListValidators(ctx context.Context, filter *models.ValidatorFilter) ([]*models.Validator, error) {
	query := strings.Builder{}
	query.WriteString(`
		SELECT id, validator_index, pubkey, withdrawal_credentials, effective_balance,
			   slashed, activation_epoch, activation_eligibility_epoch, exit_epoch,
			   withdrawable_epoch, name, tags, monitored, created_at, updated_at
		FROM validators
		WHERE 1=1`)

	args := []interface{}{}
	argCount := 0

	// Apply filters
	if len(filter.ValidatorIndices) > 0 {
		argCount++
		query.WriteString(fmt.Sprintf(" AND validator_index = ANY($%d)", argCount))
		args = append(args, filter.ValidatorIndices)
	}

	if len(filter.Pubkeys) > 0 {
		argCount++
		query.WriteString(fmt.Sprintf(" AND pubkey = ANY($%d)", argCount))
		args = append(args, filter.Pubkeys)
	}

	if len(filter.Tags) > 0 {
		argCount++
		query.WriteString(fmt.Sprintf(" AND tags && $%d", argCount))
		args = append(args, filter.Tags)
	}

	if filter.Monitored != nil {
		argCount++
		query.WriteString(fmt.Sprintf(" AND monitored = $%d", argCount))
		args = append(args, *filter.Monitored)
	}

	if filter.Slashed != nil {
		argCount++
		query.WriteString(fmt.Sprintf(" AND slashed = $%d", argCount))
		args = append(args, *filter.Slashed)
	}

	query.WriteString(" ORDER BY validator_index")

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
		return nil, fmt.Errorf("failed to list validators: %w", err)
	}
	defer rows.Close()

	validators := []*models.Validator{}
	for rows.Next() {
		validator := &models.Validator{}
		err := rows.Scan(
			&validator.ID,
			&validator.ValidatorIndex,
			&validator.Pubkey,
			&validator.WithdrawalCredentials,
			&validator.EffectiveBalance,
			&validator.Slashed,
			&validator.ActivationEpoch,
			&validator.ActivationEligibilityEpoch,
			&validator.ExitEpoch,
			&validator.WithdrawableEpoch,
			&validator.Name,
			&validator.Tags,
			&validator.Monitored,
			&validator.CreatedAt,
			&validator.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan validator: %w", err)
		}
		validators = append(validators, validator)
	}

	return validators, nil
}

// UpdateValidator updates a validator's information
func (r *ValidatorRepository) UpdateValidator(ctx context.Context, validator *models.Validator) error {
	query := `
		UPDATE validators
		SET effective_balance = $2, slashed = $3, name = $4, tags = $5, monitored = $6
		WHERE validator_index = $1
		RETURNING updated_at`

	err := r.pool.QueryRow(ctx, query,
		validator.ValidatorIndex,
		validator.EffectiveBalance,
		validator.Slashed,
		validator.Name,
		validator.Tags,
		validator.Monitored,
	).Scan(&validator.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to update validator: %w", err)
	}

	return nil
}

// DeleteValidator deletes a validator
func (r *ValidatorRepository) DeleteValidator(ctx context.Context, validatorIndex int64) error {
	query := `DELETE FROM validators WHERE validator_index = $1`

	_, err := r.pool.Exec(ctx, query, validatorIndex)
	if err != nil {
		return fmt.Errorf("failed to delete validator: %w", err)
	}

	return nil
}

// CountValidators counts validators matching the filter
func (r *ValidatorRepository) CountValidators(ctx context.Context, filter *models.ValidatorFilter) (int, error) {
	query := strings.Builder{}
	query.WriteString("SELECT COUNT(*) FROM validators WHERE 1=1")

	args := []interface{}{}
	argCount := 0

	// Apply same filters as ListValidators
	if len(filter.ValidatorIndices) > 0 {
		argCount++
		query.WriteString(fmt.Sprintf(" AND validator_index = ANY($%d)", argCount))
		args = append(args, filter.ValidatorIndices)
	}

	if len(filter.Pubkeys) > 0 {
		argCount++
		query.WriteString(fmt.Sprintf(" AND pubkey = ANY($%d)", argCount))
		args = append(args, filter.Pubkeys)
	}

	if len(filter.Tags) > 0 {
		argCount++
		query.WriteString(fmt.Sprintf(" AND tags && $%d", argCount))
		args = append(args, filter.Tags)
	}

	if filter.Monitored != nil {
		argCount++
		query.WriteString(fmt.Sprintf(" AND monitored = $%d", argCount))
		args = append(args, *filter.Monitored)
	}

	if filter.Slashed != nil {
		argCount++
		query.WriteString(fmt.Sprintf(" AND slashed = $%d", argCount))
		args = append(args, *filter.Slashed)
	}

	var count int
	err := r.pool.QueryRow(ctx, query.String(), args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count validators: %w", err)
	}

	return count, nil
}