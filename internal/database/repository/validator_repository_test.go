package repository

import (
	"context"
	"testing"

	"github.com/birddigital/eth-validator-monitor/internal/database/models"
	"github.com/birddigital/eth-validator-monitor/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatorRepository_CreateValidator(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(context.Background(), pool)

	repo := NewValidatorRepository(pool)
	ctx := context.Background()

	validator := testutil.ValidatorFixture(123)

	err := repo.CreateValidator(ctx, validator)
	require.NoError(t, err)
	assert.NotZero(t, validator.ID)
	assert.NotZero(t, validator.CreatedAt)
	assert.NotZero(t, validator.UpdatedAt)
}

func TestValidatorRepository_GetValidatorByIndex(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(context.Background(), pool)

	repo := NewValidatorRepository(pool)
	ctx := context.Background()

	// Create test validator
	created := testutil.ValidatorFixture(456)
	err := repo.CreateValidator(ctx, created)
	require.NoError(t, err)

	// Retrieve validator
	retrieved, err := repo.GetValidatorByIndex(ctx, 456)
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	assert.Equal(t, created.ValidatorIndex, retrieved.ValidatorIndex)
	assert.Equal(t, created.Pubkey, retrieved.Pubkey)
	assert.Equal(t, created.EffectiveBalance, retrieved.EffectiveBalance)
}

func TestValidatorRepository_GetValidatorByIndex_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(context.Background(), pool)

	repo := NewValidatorRepository(pool)
	ctx := context.Background()

	validator, err := repo.GetValidatorByIndex(ctx, 99999)
	require.NoError(t, err)
	assert.Nil(t, validator)
}

func TestValidatorRepository_ListValidators(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(context.Background(), pool)

	repo := NewValidatorRepository(pool)
	ctx := context.Background()

	// Create multiple validators
	validators := testutil.MultipleValidatorFixtures(5)
	for _, v := range validators {
		err := repo.CreateValidator(ctx, v)
		require.NoError(t, err)
	}

	tests := []struct {
		name     string
		filter   *models.ValidatorFilter
		expected int
	}{
		{
			name: "no filter",
			filter: &models.ValidatorFilter{
				Limit: 10,
			},
			expected: 5,
		},
		{
			name: "with limit",
			filter: &models.ValidatorFilter{
				Limit: 3,
			},
			expected: 3,
		},
		{
			name: "with monitored filter",
			filter: &models.ValidatorFilter{
				Monitored: func() *bool { b := true; return &b }(),
				Limit:     10,
			},
			expected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := repo.ListValidators(ctx, tt.filter)
			require.NoError(t, err)
			assert.Len(t, result, tt.expected)
		})
	}
}

func TestValidatorRepository_UpdateValidator(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(context.Background(), pool)

	repo := NewValidatorRepository(pool)
	ctx := context.Background()

	// Create validator
	validator := testutil.ValidatorFixture(789)
	err := repo.CreateValidator(ctx, validator)
	require.NoError(t, err)

	// Update validator
	validator.EffectiveBalance = 31000000000
	validator.Monitored = false

	err = repo.UpdateValidator(ctx, validator)
	require.NoError(t, err)

	// Verify update
	updated, err := repo.GetValidatorByIndex(ctx, 789)
	require.NoError(t, err)
	assert.Equal(t, int64(31000000000), updated.EffectiveBalance)
	assert.False(t, updated.Monitored)
}

func TestValidatorRepository_DeleteValidator(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(context.Background(), pool)

	repo := NewValidatorRepository(pool)
	ctx := context.Background()

	// Create validator
	validator := testutil.ValidatorFixture(999)
	err := repo.CreateValidator(ctx, validator)
	require.NoError(t, err)

	// Delete validator
	err = repo.DeleteValidator(ctx, 999)
	require.NoError(t, err)

	// Verify deletion
	deleted, err := repo.GetValidatorByIndex(ctx, 999)
	require.NoError(t, err)
	assert.Nil(t, deleted)
}

func TestValidatorRepository_BatchCreateValidators(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(context.Background(), pool)

	repo := NewValidatorRepository(pool)
	ctx := context.Background()

	// Create batch of validators
	validators := testutil.MultipleValidatorFixtures(100)

	err := repo.BatchCreateValidators(ctx, validators)
	require.NoError(t, err)

	// Verify all were created
	filter := &models.ValidatorFilter{Limit: 200}
	result, err := repo.ListValidators(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, result, 100)
}

func TestValidatorRepository_CountValidators(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(context.Background(), pool)

	repo := NewValidatorRepository(pool)
	ctx := context.Background()

	// Create validators with different monitoring status
	validators := testutil.MultipleValidatorFixtures(10)
	for i, v := range validators {
		if i < 5 {
			v.Monitored = false
		}
		err := repo.CreateValidator(ctx, v)
		require.NoError(t, err)
	}

	tests := []struct {
		name     string
		filter   *models.ValidatorFilter
		expected int
	}{
		{
			name:     "count all",
			filter:   &models.ValidatorFilter{},
			expected: 10,
		},
		{
			name: "count monitored",
			filter: &models.ValidatorFilter{
				Monitored: func() *bool { b := true; return &b }(),
			},
			expected: 5,
		},
		{
			name: "count not monitored",
			filter: &models.ValidatorFilter{
				Monitored: func() *bool { b := false; return &b }(),
			},
			expected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, err := repo.CountValidators(ctx, tt.filter)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, count)
		})
	}
}
