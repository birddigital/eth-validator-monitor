package repository

import (
	"context"
	"testing"
	"time"

	"github.com/birddigital/eth-validator-monitor/internal/database"
	"github.com/birddigital/eth-validator-monitor/internal/database/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupDashboardTest(t *testing.T) (*DashboardRepository, func()) {
	t.Helper()

	// Create test database config
	cfg := &database.Config{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "postgres",
		Database: "eth_validator_monitor_test",
		SSLMode:  "disable",
	}

	pool, err := database.NewPool(context.Background(), cfg)
	if err != nil {
		t.Skip("Database not available for testing:", err)
	}

	repo := NewDashboardRepository(pool)

	cleanup := func() {
		// Clean up test data
		pool.Exec(context.Background(), "TRUNCATE validators, validator_snapshots, alerts CASCADE")
		pool.Close()
	}

	return repo, cleanup
}

func TestDashboardRepository_GetAggregateMetrics(t *testing.T) {
	repo, cleanup := setupDashboardTest(t)
	defer cleanup()

	ctx := context.Background()

	// Insert test validators
	validatorRepo := NewValidatorRepository(repo.pool)
	now := time.Now()

	validators := []*models.Validator{
		{
			ValidatorIndex:   1,
			Pubkey:           "0x" + string(make([]byte, 96)),
			EffectiveBalance: 32000000000,
			Slashed:          false,
			Monitored:        true,
		},
		{
			ValidatorIndex:   2,
			Pubkey:           "0x" + string(make([]byte, 96)),
			EffectiveBalance: 32000000000,
			Slashed:          false,
			Monitored:        true,
		},
		{
			ValidatorIndex:   3,
			Pubkey:           "0x" + string(make([]byte, 96)),
			EffectiveBalance: 32000000000,
			Slashed:          true,
			Monitored:        true,
		},
	}

	for _, v := range validators {
		err := validatorRepo.CreateValidator(ctx, v)
		require.NoError(t, err)
	}

	// Insert test snapshots
	_, err := repo.pool.Exec(ctx, `
		INSERT INTO validator_snapshots (time, validator_index, balance, attestation_effectiveness)
		VALUES
			($1, 1, 32100000000, 98.5),
			($1, 2, 32050000000, 97.2),
			($1, 3, 31900000000, 85.0)
	`, now)
	require.NoError(t, err)

	// Test GetAggregateMetrics
	metrics, err := repo.GetAggregateMetrics(ctx)
	require.NoError(t, err)
	assert.NotNil(t, metrics)
	assert.Equal(t, 3, metrics.TotalValidators)
	assert.Equal(t, 3, metrics.ActiveValidators)
	assert.Greater(t, metrics.AvgEffectiveness, 90.0)
	assert.Greater(t, metrics.TotalBalanceGwei, int64(96000000000))
	assert.Equal(t, 1, metrics.SlashedValidators)
}

func TestDashboardRepository_GetRecentAlerts(t *testing.T) {
	repo, cleanup := setupDashboardTest(t)
	defer cleanup()

	ctx := context.Background()

	// Insert test alerts
	alertRepo := NewAlertRepository(repo.pool)

	testAlerts := []*models.Alert{
		{
			ValidatorIndex: ptrInt64(1),
			AlertType:      "offline",
			Severity:       models.SeverityCritical,
			Title:          "Validator Offline",
			Message:        "Validator has been offline for 5 minutes",
			Status:         models.AlertStatusActive,
		},
		{
			ValidatorIndex: ptrInt64(2),
			AlertType:      "low_balance",
			Severity:       models.SeverityWarning,
			Title:          "Low Balance",
			Message:        "Validator balance is below threshold",
			Status:         models.AlertStatusActive,
		},
		{
			ValidatorIndex: ptrInt64(3),
			AlertType:      "slashed",
			Severity:       models.SeverityCritical,
			Title:          "Validator Slashed",
			Message:        "Validator has been slashed",
			Status:         models.AlertStatusResolved,
		},
	}

	for _, alert := range testAlerts {
		err := alertRepo.CreateAlert(ctx, alert)
		require.NoError(t, err)
	}

	// Test GetRecentAlerts - should only return active alerts
	alerts, err := repo.GetRecentAlerts(ctx, 5)
	require.NoError(t, err)
	assert.Len(t, alerts, 2) // Only 2 active alerts
	assert.Equal(t, models.AlertStatusActive, alerts[0].Status)
	assert.Equal(t, models.AlertStatusActive, alerts[1].Status)
}

func TestDashboardRepository_GetTopValidators(t *testing.T) {
	repo, cleanup := setupDashboardTest(t)
	defer cleanup()

	ctx := context.Background()

	// Insert test validators
	validatorRepo := NewValidatorRepository(repo.pool)
	now := time.Now()

	validators := []*models.Validator{
		{ValidatorIndex: 1, Pubkey: "0xaaa", EffectiveBalance: 32000000000, Monitored: true, Slashed: false},
		{ValidatorIndex: 2, Pubkey: "0xbbb", EffectiveBalance: 32000000000, Monitored: true, Slashed: false},
		{ValidatorIndex: 3, Pubkey: "0xccc", EffectiveBalance: 32000000000, Monitored: true, Slashed: false},
		{ValidatorIndex: 4, Pubkey: "0xddd", EffectiveBalance: 32000000000, Monitored: true, Slashed: true}, // Should be excluded
	}

	for _, v := range validators {
		err := validatorRepo.CreateValidator(ctx, v)
		require.NoError(t, err)
	}

	// Insert snapshots with different effectiveness scores
	_, err := repo.pool.Exec(ctx, `
		INSERT INTO validator_snapshots (time, validator_index, balance, attestation_effectiveness, daily_income, apr)
		VALUES
			($1, 1, 32100000000, 99.5, 100000, 5.2),
			($1, 2, 32050000000, 98.2, 95000, 5.0),
			($1, 3, 31900000000, 96.0, 90000, 4.8),
			($1, 4, 31800000000, 95.0, 85000, 4.5)
	`, now)
	require.NoError(t, err)

	// Test GetTopValidators
	topValidators, err := repo.GetTopValidators(ctx, 3)
	require.NoError(t, err)
	assert.Len(t, topValidators, 3)

	// Verify validators are sorted by effectiveness descending
	assert.Equal(t, int64(1), topValidators[0].ValidatorIndex)
	assert.Equal(t, 99.5, topValidators[0].Effectiveness)

	assert.Equal(t, int64(2), topValidators[1].ValidatorIndex)
	assert.Equal(t, 98.2, topValidators[1].Effectiveness)

	assert.Equal(t, int64(3), topValidators[2].ValidatorIndex)
	assert.Equal(t, 96.0, topValidators[2].Effectiveness)

	// Verify slashed validator is excluded
	for _, v := range topValidators {
		assert.NotEqual(t, int64(4), v.ValidatorIndex)
	}
}

func TestDashboardRepository_GetSystemHealth(t *testing.T) {
	repo, cleanup := setupDashboardTest(t)
	defer cleanup()

	ctx := context.Background()

	// Insert a validator to ensure monitored count > 0
	validatorRepo := NewValidatorRepository(repo.pool)
	validator := &models.Validator{
		ValidatorIndex:   1,
		Pubkey:           "0xtest",
		EffectiveBalance: 32000000000,
		Monitored:        true,
	}
	err := validatorRepo.CreateValidator(ctx, validator)
	require.NoError(t, err)

	// Insert fresh snapshot data
	_, err = repo.pool.Exec(ctx, `
		INSERT INTO validator_snapshots (time, validator_index, balance, attestation_effectiveness)
		VALUES ($1, 1, 32100000000, 98.5)
	`, time.Now())
	require.NoError(t, err)

	// Test GetSystemHealth
	health, err := repo.GetSystemHealth(ctx)
	require.NoError(t, err)
	assert.NotNil(t, health)
	assert.Equal(t, "healthy", health.DatabaseStatus)
	assert.Equal(t, "fresh", health.DataFreshness)
	assert.Greater(t, health.MonitoredCount, 0)
	assert.False(t, health.LastSnapshotTime.IsZero())
}

func TestDashboardRepository_GetSystemHealth_StaleData(t *testing.T) {
	repo, cleanup := setupDashboardTest(t)
	defer cleanup()

	ctx := context.Background()

	// Insert old snapshot data (20 minutes ago)
	oldTime := time.Now().Add(-20 * time.Minute)
	_, err := repo.pool.Exec(ctx, `
		INSERT INTO validator_snapshots (time, validator_index, balance, attestation_effectiveness)
		VALUES ($1, 1, 32100000000, 98.5)
	`, oldTime)
	require.NoError(t, err)

	// Test GetSystemHealth with stale data
	health, err := repo.GetSystemHealth(ctx)
	require.NoError(t, err)
	assert.NotNil(t, health)
	assert.Equal(t, "healthy", health.DatabaseStatus)
	assert.Equal(t, "stale", health.DataFreshness) // Should detect stale data
}

// Helper function
func ptrInt64(i int64) *int64 {
	return &i
}
