package repository

import (
	"context"
	"fmt"

	"github.com/birddigital/eth-validator-monitor/internal/database/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AlertRepository handles alert database operations
type AlertRepository struct {
	db *pgxpool.Pool
}

// NewAlertRepository creates a new alert repository
func NewAlertRepository(db *pgxpool.Pool) *AlertRepository {
	return &AlertRepository{db: db}
}

// AlertListFilter extends the basic AlertFilter with pagination and sorting
type AlertListFilter struct {
	models.AlertFilter
	SortBy    string // Sort field (created_at, severity, status)
	SortOrder string // asc or desc
	Page      int    // Current page number (for API response metadata)
}

// AlertListResult contains paginated alert results with metadata
type AlertListResult struct {
	Alerts   []*models.Alert `json:"alerts"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
	HasMore  bool            `json:"has_more"`
}

// ListAlerts retrieves alerts based on filter criteria (legacy, for backward compatibility)
func (r *AlertRepository) ListAlerts(ctx context.Context, filter *models.AlertFilter) ([]*models.Alert, error) {
	query, args := r.buildListQuery(filter, "", "")

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list alerts: %w", err)
	}
	defer rows.Close()

	alerts, err := r.scanAlerts(rows)
	if err != nil {
		return nil, err
	}

	return alerts, rows.Err()
}

// ListAlertsWithPagination retrieves alerts with full pagination support and metadata
func (r *AlertRepository) ListAlertsWithPagination(ctx context.Context, filter AlertListFilter) (*AlertListResult, error) {
	// Set defaults
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	if filter.SortBy == "" {
		filter.SortBy = "created_at"
	}
	if filter.SortOrder == "" {
		filter.SortOrder = "desc"
	}

	// Build queries
	query, args := r.buildListQuery(&filter.AlertFilter, filter.SortBy, filter.SortOrder)
	countQuery, countArgs := r.buildCountQuery(&filter.AlertFilter)

	// Execute count and list queries in parallel (following validator_list pattern)
	type result struct {
		alerts []*models.Alert
		total  int64
		err    error
	}

	alertsCh := make(chan result, 1)
	countCh := make(chan result, 1)

	// Fetch alerts
	go func() {
		rows, err := r.db.Query(ctx, query, args...)
		if err != nil {
			alertsCh <- result{err: fmt.Errorf("query alerts: %w", err)}
			return
		}
		defer rows.Close()

		alerts, err := r.scanAlerts(rows)
		if err != nil {
			alertsCh <- result{err: fmt.Errorf("scan alerts: %w", err)}
			return
		}

		alertsCh <- result{alerts: alerts}
	}()

	// Fetch count
	go func() {
		var total int64
		err := r.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
		if err != nil {
			countCh <- result{err: fmt.Errorf("query count: %w", err)}
			return
		}
		countCh <- result{total: total}
	}()

	// Collect results
	alertsResult := <-alertsCh
	countResult := <-countCh

	if alertsResult.err != nil {
		return nil, alertsResult.err
	}
	if countResult.err != nil {
		return nil, countResult.err
	}

	page := (filter.Offset / filter.Limit) + 1
	hasMore := int64(filter.Offset+filter.Limit) < countResult.total

	return &AlertListResult{
		Alerts:   alertsResult.alerts,
		Total:    countResult.total,
		Page:     page,
		PageSize: filter.Limit,
		HasMore:  hasMore,
	}, nil
}

// buildListQuery constructs the alerts query with filters, sorting, and pagination
func (r *AlertRepository) buildListQuery(filter *models.AlertFilter, sortBy, sortOrder string) (string, []interface{}) {
	query := `
		SELECT id, validator_index, alert_type, severity, title, message, source,
		       details, status, acknowledged_at, resolved_at, created_at, updated_at
		FROM alerts
		WHERE 1=1`

	args := []interface{}{}
	argCount := 0

	if filter.ValidatorIndex != nil {
		argCount++
		query += fmt.Sprintf(" AND validator_index = $%d", argCount)
		args = append(args, *filter.ValidatorIndex)
	}

	if filter.AlertType != nil {
		argCount++
		query += fmt.Sprintf(" AND alert_type = $%d", argCount)
		args = append(args, *filter.AlertType)
	}

	if filter.Severity != nil {
		argCount++
		query += fmt.Sprintf(" AND severity = $%d", argCount)
		args = append(args, *filter.Severity)
	}

	if filter.Status != nil {
		argCount++
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, *filter.Status)
	}

	if filter.StartTime != nil {
		argCount++
		query += fmt.Sprintf(" AND created_at >= $%d", argCount)
		args = append(args, *filter.StartTime)
	}

	if filter.EndTime != nil {
		argCount++
		query += fmt.Sprintf(" AND created_at <= $%d", argCount)
		args = append(args, *filter.EndTime)
	}

	// Add sorting (with SQL injection protection)
	orderClause := r.getSortClause(sortBy, sortOrder)
	query += " ORDER BY " + orderClause

	if filter.Limit > 0 {
		argCount++
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filter.Limit)
	}

	if filter.Offset > 0 {
		argCount++
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, filter.Offset)
	}

	return query, args
}

// buildCountQuery constructs the count query with filters
func (r *AlertRepository) buildCountQuery(filter *models.AlertFilter) (string, []interface{}) {
	query := `SELECT COUNT(*) FROM alerts WHERE 1=1`
	args := []interface{}{}
	argCount := 0

	if filter.ValidatorIndex != nil {
		argCount++
		query += fmt.Sprintf(" AND validator_index = $%d", argCount)
		args = append(args, *filter.ValidatorIndex)
	}

	if filter.AlertType != nil {
		argCount++
		query += fmt.Sprintf(" AND alert_type = $%d", argCount)
		args = append(args, *filter.AlertType)
	}

	if filter.Severity != nil {
		argCount++
		query += fmt.Sprintf(" AND severity = $%d", argCount)
		args = append(args, *filter.Severity)
	}

	if filter.Status != nil {
		argCount++
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, *filter.Status)
	}

	if filter.StartTime != nil {
		argCount++
		query += fmt.Sprintf(" AND created_at >= $%d", argCount)
		args = append(args, *filter.StartTime)
	}

	if filter.EndTime != nil {
		argCount++
		query += fmt.Sprintf(" AND created_at <= $%d", argCount)
		args = append(args, *filter.EndTime)
	}

	return query, args
}

// getSortClause returns a safe ORDER BY clause
func (r *AlertRepository) getSortClause(sortBy, sortOrder string) string {
	// Validate and sanitize sort column
	sortColumn := "created_at"
	switch sortBy {
	case "severity":
		sortColumn = "severity"
	case "status":
		sortColumn = "status"
	case "created_at":
		sortColumn = "created_at"
	case "updated_at":
		sortColumn = "updated_at"
	}

	// Validate sort order
	order := "DESC"
	if sortOrder == "asc" || sortOrder == "ASC" {
		order = "ASC"
	}

	return sortColumn + " " + order
}

// scanAlerts scans alert rows into Alert structs
func (r *AlertRepository) scanAlerts(rows interface {
	Next() bool
	Scan(...interface{}) error
}) ([]*models.Alert, error) {
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
			&alert.Source,
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
	return alerts, nil
}

// GetAlert retrieves a single alert by ID
func (r *AlertRepository) GetAlert(ctx context.Context, id int32) (*models.Alert, error) {
	query := `
		SELECT id, validator_index, alert_type, severity, title, message, source,
		       details, status, acknowledged_at, resolved_at, created_at, updated_at
		FROM alerts
		WHERE id = $1`

	alert := &models.Alert{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&alert.ID,
		&alert.ValidatorIndex,
		&alert.AlertType,
		&alert.Severity,
		&alert.Title,
		&alert.Message,
		&alert.Source,
		&alert.Details,
		&alert.Status,
		&alert.AcknowledgedAt,
		&alert.ResolvedAt,
		&alert.CreatedAt,
		&alert.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get alert: %w", err)
	}

	return alert, nil
}

// CreateAlert creates a new alert
func (r *AlertRepository) CreateAlert(ctx context.Context, alert *models.Alert) error {
	query := `
		INSERT INTO alerts (
			validator_index, alert_type, severity, title, message,
			details, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRow(ctx, query,
		alert.ValidatorIndex,
		alert.AlertType,
		alert.Severity,
		alert.Title,
		alert.Message,
		alert.Details,
		alert.Status,
	).Scan(&alert.ID, &alert.CreatedAt, &alert.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create alert: %w", err)
	}

	return nil
}

// UpdateAlertStatus updates an alert's status
func (r *AlertRepository) UpdateAlertStatus(ctx context.Context, id int32, status models.AlertStatus) error {
	query := `
		UPDATE alerts
		SET status = $2,
		    acknowledged_at = CASE WHEN $2 = 'acknowledged' THEN NOW() ELSE acknowledged_at END,
		    resolved_at = CASE WHEN $2 = 'resolved' THEN NOW() ELSE resolved_at END,
		    updated_at = NOW()
		WHERE id = $1`

	_, err := r.db.Exec(ctx, query, id, status)
	if err != nil {
		return fmt.Errorf("failed to update alert status: %w", err)
	}

	return nil
}

// ResolveAlert marks an alert as resolved
func (r *AlertRepository) ResolveAlert(ctx context.Context, id int32) error {
	return r.UpdateAlertStatus(ctx, id, models.AlertStatusResolved)
}
