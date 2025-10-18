package benchmarks

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/birddigital/eth-validator-monitor/benchmarks/fixtures"
	"github.com/birddigital/eth-validator-monitor/benchmarks/helpers"
)

// BenchmarkBatchInserts tests PostgreSQL batch insert performance
func BenchmarkBatchInserts(b *testing.B) {
	// Setup test database using testcontainers
	ctx := context.Background()
	pool := helpers.SetupTestDB(b)
	defer pool.Close()

	snapshotCounts := []int{100, 1000, 5000, 10000}

	for _, count := range snapshotCounts {
		b.Run(fmt.Sprintf("snapshots_%d", count), func(b *testing.B) {
			snapshots := fixtures.GenerateSnapshots(count)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				b.StopTimer()
				// Clean table between iterations
				helpers.CleanupSnapshots(b, pool)
				b.StartTimer()

				// Use pgx CopyFrom for maximum performance
				err := batchInsertSnapshots(ctx, pool, snapshots)
				if err != nil {
					b.Fatalf("batch insert failed: %v", err)
				}
			}

			// Custom metrics
			b.ReportMetric(float64(count)/b.Elapsed().Seconds(), "inserts/sec")
		})
	}
}

// batchInsertSnapshots performs efficient batch insert using pgx CopyFrom
func batchInsertSnapshots(ctx context.Context, pool *pgxpool.Pool, snapshots []*fixtures.ValidatorSnapshot) error {
	copyCount, err := pool.CopyFrom(
		ctx,
		pgx.Identifier{"validator_snapshots"},
		[]string{"validator_index", "timestamp", "balance", "effective_balance", "effectiveness", "missed_attestations", "proposal_success", "epoch", "slot"},
		pgx.CopyFromSlice(len(snapshots), func(i int) ([]interface{}, error) {
			s := snapshots[i]
			return []interface{}{
				s.ValidatorIndex,
				s.Timestamp,
				s.Balance,
				s.EffectiveBalance,
				s.Effectiveness,
				s.MissedAttestations,
				s.ProposalSuccess,
				s.Epoch,
				s.Slot,
			}, nil
		}),
	)

	if err != nil {
		return fmt.Errorf("copy from failed: %w", err)
	}

	if copyCount != int64(len(snapshots)) {
		return fmt.Errorf("expected to copy %d rows, but copied %d", len(snapshots), copyCount)
	}

	return nil
}

// BenchmarkComplexQueries tests query performance with realistic filters
func BenchmarkComplexQueries(b *testing.B) {
	ctx := context.Background()
	pool := helpers.SetupTestDB(b)
	defer pool.Close()

	// Seed database with realistic data
	seedTestData(b, pool, 10000) // 10k validators, 24 hours of data

	queryTypes := []struct {
		name string
		fn   func(context.Context, *pgxpool.Pool) error
	}{
		{
			name: "time_range_single_validator",
			fn: func(ctx context.Context, pool *pgxpool.Pool) error {
				var count int
				err := pool.QueryRow(ctx, `
					SELECT COUNT(*)
					FROM validator_snapshots
					WHERE validator_index = $1
					AND timestamp >= $2
					AND timestamp <= $3
				`, 1, time.Now().Add(-24*time.Hour), time.Now()).Scan(&count)
				return err
			},
		},
		{
			name: "effectiveness_percentile",
			fn: func(ctx context.Context, pool *pgxpool.Pool) error {
				rows, err := pool.Query(ctx, `
					SELECT validator_index, effectiveness
					FROM validator_snapshots
					WHERE timestamp = (SELECT MAX(timestamp) FROM validator_snapshots)
					AND effectiveness >= $1
					ORDER BY effectiveness DESC
					LIMIT $2
				`, 0.95, 100)
				if err != nil {
					return err
				}
				rows.Close()
				return nil
			},
		},
		{
			name: "aggregated_metrics",
			fn: func(ctx context.Context, pool *pgxpool.Pool) error {
				var avgEffectiveness float64
				var totalMissed int64
				err := pool.QueryRow(ctx, `
					SELECT
						AVG(effectiveness) as avg_effectiveness,
						SUM(missed_attestations) as total_missed
					FROM validator_snapshots
					WHERE timestamp >= $1
					AND timestamp <= $2
				`, time.Now().Add(-1*time.Hour), time.Now()).Scan(&avgEffectiveness, &totalMissed)
				return err
			},
		},
		{
			name: "validator_performance_history",
			fn: func(ctx context.Context, pool *pgxpool.Pool) error {
				rows, err := pool.Query(ctx, `
					SELECT
						timestamp,
						balance,
						effectiveness,
						missed_attestations
					FROM validator_snapshots
					WHERE validator_index = $1
					AND timestamp >= $2
					ORDER BY timestamp DESC
					LIMIT 100
				`, 1, time.Now().Add(-24*time.Hour))
				if err != nil {
					return err
				}
				rows.Close()
				return nil
			},
		},
	}

	for _, qt := range queryTypes {
		b.Run(qt.name, func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				if err := qt.fn(ctx, pool); err != nil {
					b.Fatalf("query failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkConnectionPoolEfficiency tests pgx pool under concurrent load
func BenchmarkConnectionPoolEfficiency(b *testing.B) {
	ctx := context.Background()
	pool := helpers.SetupTestDB(b)
	defer pool.Close()

	seedTestData(b, pool, 1000)

	concurrencyLevels := []int{1, 10, 50, 100}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("concurrent_%d", concurrency), func(b *testing.B) {
			b.ReportAllocs()
			b.SetParallelism(concurrency)

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					var count int
					err := pool.QueryRow(ctx, `
						SELECT COUNT(*)
						FROM validator_snapshots
						WHERE validator_index = $1
						AND timestamp >= $2
					`, 1, time.Now().Add(-1*time.Hour)).Scan(&count)

					if err != nil {
						b.Errorf("query failed: %v", err)
					}
				}
			})
		})
	}
}

// seedTestData populates the database with realistic test data
func seedTestData(b *testing.B, pool *pgxpool.Pool, validatorCount int) {
	ctx := context.Background()

	// Generate 24 hours of snapshots at 12-second intervals
	startTime := time.Now().Add(-24 * time.Hour)
	endTime := time.Now()
	interval := 12 * time.Second

	snapshots := fixtures.GenerateSnapshotsForTimeRange(validatorCount, startTime, endTime, interval)

	b.Logf("Seeding database with %d snapshots for %d validators", len(snapshots), validatorCount)

	err := batchInsertSnapshots(ctx, pool, snapshots)
	if err != nil {
		b.Fatalf("failed to seed test data: %v", err)
	}

	b.Logf("Seeding complete")
}
