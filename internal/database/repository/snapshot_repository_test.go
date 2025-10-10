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

func TestSnapshotRepository_InsertSnapshot(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(context.Background(), pool)

	repo := NewSnapshotRepository(pool)
	ctx := context.Background()

	snapshot := testutil.ValidatorSnapshotFixture(123, time.Now())

	err := repo.InsertSnapshot(ctx, snapshot)
	require.NoError(t, err)
}

func TestSnapshotRepository_BatchInsertSnapshots(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(context.Background(), pool)

	repo := NewSnapshotRepository(pool)
	ctx := context.Background()

	// Create batch of snapshots
	snapshots := testutil.MultipleSnapshotFixtures(123, 50)

	err := repo.BatchInsertSnapshots(ctx, snapshots)
	require.NoError(t, err)

	// Verify snapshots were inserted
	recent, err := repo.GetRecentSnapshots(ctx, 123, 100)
	require.NoError(t, err)
	assert.Len(t, recent, 50)
}

func TestSnapshotRepository_GetLatestSnapshot(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(context.Background(), pool)

	repo := NewSnapshotRepository(pool)
	ctx := context.Background()

	// Insert multiple snapshots at different times
	baseTime := time.Now().Add(-24 * time.Hour)
	for i := 0; i < 10; i++ {
		snapshot := testutil.ValidatorSnapshotFixture(456, baseTime.Add(time.Duration(i)*time.Hour))
		err := repo.InsertSnapshot(ctx, snapshot)
		require.NoError(t, err)
	}

	// Get latest snapshot
	latest, err := repo.GetLatestSnapshot(ctx, 456)
	require.NoError(t, err)
	require.NotNil(t, latest)

	assert.Equal(t, int64(456), latest.ValidatorIndex)
	// Latest should be the most recent one (index 9)
	assert.True(t, latest.Time.After(baseTime.Add(8*time.Hour)))
}

func TestSnapshotRepository_GetLatestSnapshot_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(context.Background(), pool)

	repo := NewSnapshotRepository(pool)
	ctx := context.Background()

	snapshot, err := repo.GetLatestSnapshot(ctx, 99999)
	require.NoError(t, err)
	assert.Nil(t, snapshot)
}

func TestSnapshotRepository_GetSnapshots(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(context.Background(), pool)

	repo := NewSnapshotRepository(pool)
	ctx := context.Background()

	// Create snapshots over time range
	baseTime := time.Now().Add(-48 * time.Hour)
	for i := 0; i < 20; i++ {
		snapshot := testutil.ValidatorSnapshotFixture(789, baseTime.Add(time.Duration(i)*time.Hour))
		err := repo.InsertSnapshot(ctx, snapshot)
		require.NoError(t, err)
	}

	tests := []struct {
		name     string
		filter   *models.SnapshotFilter
		expected int
	}{
		{
			name: "all snapshots",
			filter: &models.SnapshotFilter{
				ValidatorIndex: 789,
				Limit:          100,
			},
			expected: 20,
		},
		{
			name: "with limit",
			filter: &models.SnapshotFilter{
				ValidatorIndex: 789,
				Limit:          10,
			},
			expected: 10,
		},
		{
			name: "with time range",
			filter: &models.SnapshotFilter{
				ValidatorIndex: 789,
				StartTime:      &[]time.Time{baseTime.Add(5 * time.Hour)}[0],
				EndTime:        &[]time.Time{baseTime.Add(15 * time.Hour)}[0],
				Limit:          100,
			},
			expected: 11, // Hours 5-15 inclusive
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshots, err := repo.GetSnapshots(ctx, tt.filter)
			require.NoError(t, err)
			assert.Len(t, snapshots, tt.expected)
		})
	}
}

func TestSnapshotRepository_GetRecentSnapshots(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(context.Background(), pool)

	repo := NewSnapshotRepository(pool)
	ctx := context.Background()

	// Create snapshots
	snapshots := testutil.MultipleSnapshotFixtures(111, 30)
	err := repo.BatchInsertSnapshots(ctx, snapshots)
	require.NoError(t, err)

	// Get recent snapshots
	recent, err := repo.GetRecentSnapshots(ctx, 111, 10)
	require.NoError(t, err)
	assert.Len(t, recent, 10)

	// Verify they're ordered by time DESC
	for i := 1; i < len(recent); i++ {
		assert.True(t, recent[i-1].Time.After(recent[i].Time) || recent[i-1].Time.Equal(recent[i].Time))
	}
}

func TestSnapshotRepository_GetAggregatedStats(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(context.Background(), pool)

	repo := NewSnapshotRepository(pool)
	ctx := context.Background()

	// Create snapshots over 24 hours
	baseTime := time.Now().Add(-24 * time.Hour).Truncate(time.Hour)
	for i := 0; i < 24; i++ {
		snapshot := testutil.ValidatorSnapshotFixture(222, baseTime.Add(time.Duration(i)*time.Hour))
		err := repo.InsertSnapshot(ctx, snapshot)
		require.NoError(t, err)
	}

	tests := []struct {
		name     string
		interval string
		wantErr  bool
	}{
		{
			name:     "hourly aggregation",
			interval: "hourly",
			wantErr:  false,
		},
		{
			name:     "daily aggregation",
			interval: "daily",
			wantErr:  false,
		},
		{
			name:     "invalid interval",
			interval: "weekly",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats, err := repo.GetAggregatedStats(ctx, 222, tt.interval, baseTime, baseTime.Add(24*time.Hour))

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, stats)
				assert.Contains(t, stats, "interval")
				assert.Contains(t, stats, "data")
			}
		})
	}
}

func TestCalculateEffectivenessScore(t *testing.T) {
	tests := []struct {
		name           string
		headVote       bool
		sourceVote     bool
		targetVote     bool
		inclusionDelay int32
		expected       float64
	}{
		{
			name:           "perfect attestation",
			headVote:       true,
			sourceVote:     true,
			targetVote:     true,
			inclusionDelay: 1,
			expected:       100.0,
		},
		{
			name:           "all votes correct, delayed inclusion",
			headVote:       true,
			sourceVote:     true,
			targetVote:     true,
			inclusionDelay: 2,
			expected:       93.75, // 75 + 18.75 (delayed by 1 slot)
		},
		{
			name:           "missed head vote",
			headVote:       false,
			sourceVote:     true,
			targetVote:     true,
			inclusionDelay: 1,
			expected:       75.0,
		},
		{
			name:           "all votes missed",
			headVote:       false,
			sourceVote:     false,
			targetVote:     false,
			inclusionDelay: 1,
			expected:       25.0, // Only inclusion delay score
		},
		{
			name:           "extreme delay",
			headVote:       true,
			sourceVote:     true,
			targetVote:     true,
			inclusionDelay: 10,
			expected:       75.0, // 75 from votes, 0 from delay (maxed out)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := CalculateEffectivenessScore(tt.headVote, tt.sourceVote, tt.targetVote, tt.inclusionDelay)
			assert.InDelta(t, tt.expected, score, 0.01)
		})
	}
}
