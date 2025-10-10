package repository

import (
	"context"
	"testing"
	"time"

	"github.com/birddigital/eth-validator-monitor/internal/database/models"
	"github.com/birddigital/eth-validator-monitor/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_ValidatorLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(context.Background(), pool)

	validatorRepo := NewValidatorRepository(pool)
	snapshotRepo := NewSnapshotRepository(pool)
	ctx := context.Background()

	// Create a validator
	validator := testutil.ValidatorFixture(12345)
	err := validatorRepo.CreateValidator(ctx, validator)
	require.NoError(t, err)
	assert.NotZero(t, validator.ID)

	// Add multiple snapshots over time
	baseTime := time.Now().Add(-24 * time.Hour)
	for i := 0; i < 10; i++ {
		snapshot := testutil.ValidatorSnapshotFixture(12345, baseTime.Add(time.Duration(i)*time.Hour))
		err := snapshotRepo.InsertSnapshot(ctx, snapshot)
		require.NoError(t, err)
	}

	// Verify we can retrieve the validator
	retrieved, err := validatorRepo.GetValidatorByIndex(ctx, 12345)
	require.NoError(t, err)
	assert.Equal(t, validator.Pubkey, retrieved.Pubkey)

	// Verify we can get the latest snapshot
	latest, err := snapshotRepo.GetLatestSnapshot(ctx, 12345)
	require.NoError(t, err)
	assert.NotNil(t, latest)
	assert.Equal(t, int64(12345), latest.ValidatorIndex)

	// Verify we can get recent snapshots
	recent, err := snapshotRepo.GetRecentSnapshots(ctx, 12345, 5)
	require.NoError(t, err)
	assert.Len(t, recent, 5)

	// Verify snapshots are ordered by time DESC
	for i := 1; i < len(recent); i++ {
		assert.True(t, recent[i-1].Time.After(recent[i].Time) || recent[i-1].Time.Equal(recent[i].Time))
	}

	// Update validator
	validator.Monitored = false
	err = validatorRepo.UpdateValidator(ctx, validator)
	require.NoError(t, err)

	// Verify update
	updated, err := validatorRepo.GetValidatorByIndex(ctx, 12345)
	require.NoError(t, err)
	assert.False(t, updated.Monitored)

	// Delete validator
	err = validatorRepo.DeleteValidator(ctx, 12345)
	require.NoError(t, err)

	// Verify deletion
	deleted, err := validatorRepo.GetValidatorByIndex(ctx, 12345)
	require.NoError(t, err)
	assert.Nil(t, deleted)
}

func TestIntegration_BatchOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(context.Background(), pool)

	validatorRepo := NewValidatorRepository(pool)
	snapshotRepo := NewSnapshotRepository(pool)
	ctx := context.Background()

	// Batch create validators
	validators := testutil.MultipleValidatorFixtures(100)
	err := validatorRepo.BatchCreateValidators(ctx, validators)
	require.NoError(t, err)

	// Verify all validators were created
	count, err := validatorRepo.CountValidators(ctx, &models.ValidatorFilter{})
	require.NoError(t, err)
	assert.Equal(t, 100, count)

	// Batch create snapshots for first validator
	snapshots := testutil.MultipleSnapshotFixtures(100, 100)
	err = snapshotRepo.BatchInsertSnapshots(ctx, snapshots)
	require.NoError(t, err)

	// Verify snapshots were inserted
	recent, err := snapshotRepo.GetRecentSnapshots(ctx, 100, 200)
	require.NoError(t, err)
	assert.Len(t, recent, 100)
}

func TestIntegration_ComplexFiltering(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(context.Background(), pool)

	validatorRepo := NewValidatorRepository(pool)
	snapshotRepo := NewSnapshotRepository(pool)
	ctx := context.Background()

	// Create validators with different statuses
	for i := int64(1000); i < 1020; i++ {
		validator := testutil.ValidatorFixture(i)
		if i%2 == 0 {
			validator.Monitored = false
		}
		err := validatorRepo.CreateValidator(ctx, validator)
		require.NoError(t, err)
	}

	// Test monitored filter
	monitoredTrue := true
	monitoredFilter := &models.ValidatorFilter{
		Monitored: &monitoredTrue,
		Limit:     100,
	}
	monitored, err := validatorRepo.ListValidators(ctx, monitoredFilter)
	require.NoError(t, err)
	assert.Equal(t, 10, len(monitored))

	// Test unmonitored filter
	monitoredFalse := false
	unmonitoredFilter := &models.ValidatorFilter{
		Monitored: &monitoredFalse,
		Limit:     100,
	}
	unmonitored, err := validatorRepo.ListValidators(ctx, unmonitoredFilter)
	require.NoError(t, err)
	assert.Equal(t, 10, len(unmonitored))

	// Test specific validator indices
	indices := []int64{1000, 1005, 1010}
	indicesFilter := &models.ValidatorFilter{
		ValidatorIndices: indices,
		Limit:            10,
	}
	filtered, err := validatorRepo.ListValidators(ctx, indicesFilter)
	require.NoError(t, err)
	assert.Equal(t, 3, len(filtered))

	// Add snapshots for time-based filtering
	baseTime := time.Now().Add(-48 * time.Hour).Truncate(time.Hour)
	for i := 0; i < 24; i++ {
		snapshot := testutil.ValidatorSnapshotFixture(1000, baseTime.Add(time.Duration(i)*time.Hour))
		err := snapshotRepo.InsertSnapshot(ctx, snapshot)
		require.NoError(t, err)
	}

	// Test time range filtering
	startTime := baseTime.Add(10 * time.Hour)
	endTime := baseTime.Add(15 * time.Hour)
	timeFilter := &models.SnapshotFilter{
		ValidatorIndex: 1000,
		StartTime:      &startTime,
		EndTime:        &endTime,
		Limit:          100,
	}
	timeFiltered, err := snapshotRepo.GetSnapshots(ctx, timeFilter)
	require.NoError(t, err)
	assert.Equal(t, 6, len(timeFiltered)) // Hours 10-15 inclusive
}

func TestIntegration_ConcurrentOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(context.Background(), pool)

	validatorRepo := NewValidatorRepository(pool)
	ctx := context.Background()

	// Concurrent validator creation
	done := make(chan bool)
	for i := int64(2000); i < 2010; i++ {
		go func(index int64) {
			validator := testutil.ValidatorFixture(index)
			err := validatorRepo.CreateValidator(ctx, validator)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all validators were created
	count, err := validatorRepo.CountValidators(ctx, &models.ValidatorFilter{})
	require.NoError(t, err)
	assert.Equal(t, 10, count)

	// Concurrent reads
	for i := int64(2000); i < 2010; i++ {
		go func(index int64) {
			validator, err := validatorRepo.GetValidatorByIndex(ctx, index)
			assert.NoError(t, err)
			assert.NotNil(t, validator)
			done <- true
		}(i)
	}

	// Wait for all reads
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestIntegration_DataIntegrity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(context.Background(), pool)

	validatorRepo := NewValidatorRepository(pool)
	snapshotRepo := NewSnapshotRepository(pool)
	ctx := context.Background()

	// Create validator
	validator := testutil.ValidatorFixture(3000)
	err := validatorRepo.CreateValidator(ctx, validator)
	require.NoError(t, err)

	// Insert snapshot with specific values
	snapshot := testutil.ValidatorSnapshotFixture(3000, time.Now())
	snapshot.Balance = 32100000000
	snapshot.EffectiveBalance = 32000000000
	err = snapshotRepo.InsertSnapshot(ctx, snapshot)
	require.NoError(t, err)

	// Retrieve and verify exact values
	retrieved, err := snapshotRepo.GetLatestSnapshot(ctx, 3000)
	require.NoError(t, err)
	assert.Equal(t, int64(32100000000), retrieved.Balance)
	assert.Equal(t, int64(32000000000), retrieved.EffectiveBalance)

	// Verify timestamp precision
	timeDiff := snapshot.Time.Sub(retrieved.Time)
	assert.True(t, timeDiff < time.Second, "Timestamp should be preserved within second precision")

	// Verify boolean fields
	if snapshot.AttestationHeadVote != nil {
		assert.Equal(t, *snapshot.AttestationHeadVote, *retrieved.AttestationHeadVote)
	}
}

func TestIntegration_AggregatedStats(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(context.Background(), pool)

	snapshotRepo := NewSnapshotRepository(pool)
	ctx := context.Background()

	// Create snapshots over 24 hours
	baseTime := time.Now().Add(-24 * time.Hour).Truncate(time.Hour)
	for i := 0; i < 24; i++ {
		snapshot := testutil.ValidatorSnapshotFixture(4000, baseTime.Add(time.Duration(i)*time.Hour))
		err := snapshotRepo.InsertSnapshot(ctx, snapshot)
		require.NoError(t, err)
	}

	// Test hourly aggregation
	hourlyStats, err := snapshotRepo.GetAggregatedStats(ctx, 4000, "hourly", baseTime, baseTime.Add(24*time.Hour))
	require.NoError(t, err)
	assert.NotNil(t, hourlyStats)
	assert.Contains(t, hourlyStats, "interval")
	assert.Contains(t, hourlyStats, "data")

	// Test daily aggregation
	dailyStats, err := snapshotRepo.GetAggregatedStats(ctx, 4000, "daily", baseTime, baseTime.Add(24*time.Hour))
	require.NoError(t, err)
	assert.NotNil(t, dailyStats)

	// Test invalid interval
	_, err = snapshotRepo.GetAggregatedStats(ctx, 4000, "weekly", baseTime, baseTime.Add(24*time.Hour))
	assert.Error(t, err)
}

func TestIntegration_EdgeCases(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(context.Background(), pool)

	validatorRepo := NewValidatorRepository(pool)
	snapshotRepo := NewSnapshotRepository(pool)
	ctx := context.Background()

	// Test empty results
	nonExistent, err := validatorRepo.GetValidatorByIndex(ctx, 99999)
	require.NoError(t, err)
	assert.Nil(t, nonExistent)

	// Test empty snapshot results
	noSnapshots, err := snapshotRepo.GetRecentSnapshots(ctx, 99999, 10)
	require.NoError(t, err)
	assert.Len(t, noSnapshots, 0)

	// Test large limit
	largeFilter := &models.ValidatorFilter{
		Limit: 10000,
	}
	largResult, err := validatorRepo.ListValidators(ctx, largeFilter)
	require.NoError(t, err)
	assert.NotNil(t, largResult)

	// Test zero limit (should use default)
	zeroFilter := &models.ValidatorFilter{
		Limit: 0,
	}
	zeroResult, err := validatorRepo.ListValidators(ctx, zeroFilter)
	require.NoError(t, err)
	assert.NotNil(t, zeroResult)
}
