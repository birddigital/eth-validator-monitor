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

// ListAlerts retrieves alerts based on filter criteria
func (r *AlertRepository) ListAlerts(ctx context.Context, filter *models.AlertFilter) ([]*models.Alert, error) {
	query := `
		SELECT id, validator_index, alert_type, severity, title, message,
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

	query += " ORDER BY created_at DESC"

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

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list alerts: %w", err)
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

	return alerts, rows.Err()
}

// GetAlert retrieves a single alert by ID
func (r *AlertRepository) GetAlert(ctx context.Context, id int32) (*models.Alert, error) {
	query := `
		SELECT id, validator_index, alert_type, severity, title, message,
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
