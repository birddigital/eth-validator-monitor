package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ValidatorListFilter struct {
	Search    string // Search by validator index or pubkey prefix
	Status    string // Filter by status (active, exited, slashed, etc.)
	SortBy    string // Sort field (effectiveness, balance, index)
	SortOrder string // asc or desc
	Limit     int    // Page size (default 20)
	Offset    int    // For pagination
}

type ValidatorListItem struct {
	Index                    uint64    `json:"index"`
	Pubkey                   string    `json:"pubkey"`
	Status                   string    `json:"status"`
	Balance                  uint64    `json:"balance"`
	EffectiveBalance         uint64    `json:"effective_balance"`
	AttestationEffectiveness float64   `json:"attestation_effectiveness"`
	LastSeenEpoch            uint64    `json:"last_seen_epoch"`
	IsSlashed                bool      `json:"is_slashed"`
	ActivationEpoch          uint64    `json:"activation_epoch"`
	ExitEpoch                *uint64   `json:"exit_epoch,omitempty"`
	UpdatedAt                time.Time `json:"updated_at"`
}

type ValidatorListResult struct {
	Validators []ValidatorListItem `json:"validators"`
	Total      int64               `json:"total"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"page_size"`
	HasMore    bool                `json:"has_more"`
}

type ValidatorListRepository struct {
	pool *pgxpool.Pool
}

func NewValidatorListRepository(pool *pgxpool.Pool) *ValidatorListRepository {
	return &ValidatorListRepository{pool: pool}
}

func (r *ValidatorListRepository) List(ctx context.Context, filter ValidatorListFilter) (*ValidatorListResult, error) {
	// Set defaults
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 20
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	if filter.SortBy == "" {
		filter.SortBy = "index"
	}
	if filter.SortOrder == "" {
		filter.SortOrder = "asc"
	}

	// Build query with proper SQL injection protection
	query, args := r.buildListQuery(filter)
	countQuery, countArgs := r.buildCountQuery(filter)

	// Execute count and list queries in parallel
	type result struct {
		validators []ValidatorListItem
		total      int64
		err        error
	}

	validatorsCh := make(chan result, 1)
	countCh := make(chan result, 1)

	// Fetch validators
	go func() {
		rows, err := r.pool.Query(ctx, query, args...)
		if err != nil {
			validatorsCh <- result{err: fmt.Errorf("query validators: %w", err)}
			return
		}
		defer rows.Close()

		validators, err := pgx.CollectRows(rows, pgx.RowToStructByName[ValidatorListItem])
		if err != nil {
			validatorsCh <- result{err: fmt.Errorf("collect rows: %w", err)}
			return
		}

		validatorsCh <- result{validators: validators}
	}()

	// Fetch count
	go func() {
		var total int64
		err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
		if err != nil {
			countCh <- result{err: fmt.Errorf("query count: %w", err)}
			return
		}
		countCh <- result{total: total}
	}()

	// Collect results
	validatorsResult := <-validatorsCh
	countResult := <-countCh

	if validatorsResult.err != nil {
		return nil, validatorsResult.err
	}
	if countResult.err != nil {
		return nil, countResult.err
	}

	page := (filter.Offset / filter.Limit) + 1
	hasMore := int64(filter.Offset+filter.Limit) < countResult.total

	return &ValidatorListResult{
		Validators: validatorsResult.validators,
		Total:      countResult.total,
		Page:       page,
		PageSize:   filter.Limit,
		HasMore:    hasMore,
	}, nil
}

func (r *ValidatorListRepository) buildListQuery(filter ValidatorListFilter) (string, []interface{}) {
	// Use subquery to get latest snapshot per validator
	query := `
		SELECT DISTINCT ON (v.validator_index)
			v.validator_index AS index,
			v.pubkey,
			v.status,
			s.balance,
			s.effective_balance,
			s.attestation_effectiveness,
			s.epoch AS last_seen_epoch,
			v.is_slashed,
			v.activation_epoch,
			v.exit_epoch,
			s.created_at AS updated_at
		FROM validators v
		INNER JOIN validator_snapshots s ON v.validator_index = s.validator_index
		WHERE 1=1`

	args := []interface{}{}
	argIdx := 1

	// Add filters
	if filter.Search != "" {
		// Search by index or pubkey prefix
		query += fmt.Sprintf(` AND (
			v.validator_index::text LIKE $%d OR
			v.pubkey LIKE $%d
		)`, argIdx, argIdx+1)
		args = append(args, filter.Search+"%", filter.Search+"%")
		argIdx += 2
	}

	if filter.Status != "" {
		query += fmt.Sprintf(` AND v.status = $%d`, argIdx)
		args = append(args, filter.Status)
		argIdx++
	}

	// Order by to get latest snapshot per validator
	query += ` ORDER BY v.validator_index, s.created_at DESC`

	// Wrap in outer query for sorting and pagination
	sortColumn := r.getSortColumn(filter.SortBy)
	sortOrder := strings.ToUpper(filter.SortOrder)
	if sortOrder != "ASC" && sortOrder != "DESC" {
		sortOrder = "ASC"
	}

	outerQuery := fmt.Sprintf(`
		SELECT * FROM (%s) AS latest_snapshots
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, query, sortColumn, sortOrder, argIdx, argIdx+1)

	args = append(args, filter.Limit, filter.Offset)

	return outerQuery, args
}

func (r *ValidatorListRepository) buildCountQuery(filter ValidatorListFilter) (string, []interface{}) {
	query := `SELECT COUNT(DISTINCT v.validator_index) FROM validators v WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if filter.Search != "" {
		query += fmt.Sprintf(` AND (
			v.validator_index::text LIKE $%d OR
			v.pubkey LIKE $%d
		)`, argIdx, argIdx+1)
		args = append(args, filter.Search+"%", filter.Search+"%")
		argIdx += 2
	}

	if filter.Status != "" {
		query += fmt.Sprintf(` AND v.status = $%d`, argIdx)
		args = append(args, filter.Status)
		argIdx++
	}

	return query, args
}

func (r *ValidatorListRepository) getSortColumn(sortBy string) string {
	switch sortBy {
	case "effectiveness":
		return "attestation_effectiveness"
	case "balance":
		return "balance"
	case "status":
		return "status"
	default:
		return "index"
	}
}
