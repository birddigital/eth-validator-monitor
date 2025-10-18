package benchmarks

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/your-org/eth-validator-monitor/benchmarks/fixtures"
	"github.com/your-org/eth-validator-monitor/benchmarks/helpers"
)

// BenchmarkRedisCacheOperations tests get/set performance
func BenchmarkRedisCacheOperations(b *testing.B) {
	ctx := context.Background()
	rdb := helpers.SetupTestRedis(b)
	defer rdb.Close()

	b.Run("set_validator_snapshot", func(b *testing.B) {
		snapshot := fixtures.GenerateSnapshot()
		data, err := json.Marshal(snapshot)
		if err != nil {
			b.Fatalf("failed to marshal snapshot: %v", err)
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("validator:%d:snapshot", snapshot.ValidatorIndex)
			err := rdb.Set(ctx, key, data, 15*time.Minute).Err()
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("get_validator_snapshot", func(b *testing.B) {
		// Seed cache
		snapshot := fixtures.GenerateSnapshot()
		data, _ := json.Marshal(snapshot)
		key := fmt.Sprintf("validator:%d:snapshot", snapshot.ValidatorIndex)
		rdb.Set(ctx, key, data, 15*time.Minute)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			val, err := rdb.Get(ctx, key).Result()
			if err != nil {
				b.Fatal(err)
			}
			if len(val) == 0 {
				b.Fatal("empty value returned")
			}
		}
	})

	b.Run("get_and_unmarshal", func(b *testing.B) {
		// Seed cache
		snapshot := fixtures.GenerateSnapshot()
		data, _ := json.Marshal(snapshot)
		key := fmt.Sprintf("validator:%d:snapshot", snapshot.ValidatorIndex)
		rdb.Set(ctx, key, data, 15*time.Minute)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			val, err := rdb.Get(ctx, key).Bytes()
			if err != nil {
				b.Fatal(err)
			}

			var result fixtures.ValidatorSnapshot
			err = json.Unmarshal(val, &result)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	batchSizes := []int{10, 100, 1000}

	for _, size := range batchSizes {
		b.Run(fmt.Sprintf("batch_set_%d", size), func(b *testing.B) {
			snapshots := fixtures.GenerateSnapshots(size)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				pipe := rdb.Pipeline()

				for _, snap := range snapshots {
					data, _ := json.Marshal(snap)
					key := fmt.Sprintf("validator:%d:snapshot", snap.ValidatorIndex)
					pipe.Set(ctx, key, data, 15*time.Minute)
				}

				_, err := pipe.Exec(ctx)
				if err != nil {
					b.Fatal(err)
				}
			}

			b.ReportMetric(float64(size)/b.Elapsed().Seconds(), "sets/sec")
		})
	}

	for _, size := range batchSizes {
		b.Run(fmt.Sprintf("batch_get_%d", size), func(b *testing.B) {
			// Seed cache
			snapshots := fixtures.GenerateSnapshots(size)
			pipe := rdb.Pipeline()
			for _, snap := range snapshots {
				data, _ := json.Marshal(snap)
				key := fmt.Sprintf("validator:%d:snapshot", snap.ValidatorIndex)
				pipe.Set(ctx, key, data, 15*time.Minute)
			}
			pipe.Exec(ctx)

			// Build keys list
			keys := make([]string, size)
			for i := 0; i < size; i++ {
				keys[i] = fmt.Sprintf("validator:%d:snapshot", i)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				pipe := rdb.Pipeline()
				for _, key := range keys {
					pipe.Get(ctx, key)
				}
				_, err := pipe.Exec(ctx)
				if err != nil && err != redis.Nil {
					b.Fatal(err)
				}
			}

			b.ReportMetric(float64(size)/b.Elapsed().Seconds(), "gets/sec")
		})
	}
}

// BenchmarkCacheInvalidationPatterns tests invalidation performance
func BenchmarkCacheInvalidationPatterns(b *testing.B) {
	ctx := context.Background()
	rdb := helpers.SetupTestRedis(b)
	defer rdb.Close()

	// Seed cache with 10k validators
	seedRedisCache(b, rdb, 10000)

	b.Run("invalidate_single", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("validator:%d:snapshot", i%10000)
			err := rdb.Del(ctx, key).Err()
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("invalidate_pattern", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			b.StopTimer()
			// Re-seed a subset for pattern matching
			seedRedisCache(b, rdb, 100)
			b.StartTimer()

			// Find and delete keys matching pattern
			var cursor uint64
			var keys []string
			for {
				var foundKeys []string
				var err error
				foundKeys, cursor, err = rdb.Scan(ctx, cursor, "validator:*:snapshot", 100).Result()
				if err != nil {
					b.Fatal(err)
				}

				keys = append(keys, foundKeys...)

				if cursor == 0 {
					break
				}
			}

			if len(keys) > 0 {
				err := rdb.Del(ctx, keys...).Err()
				if err != nil {
					b.Fatal(err)
				}
			}
		}
	})

	b.Run("invalidate_batch", func(b *testing.B) {
		validatorIndices := []uint64{1, 2, 3, 4, 5, 100, 200, 300, 400, 500}

		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			keys := make([]string, len(validatorIndices))
			for j, idx := range validatorIndices {
				keys[j] = fmt.Sprintf("validator:%d:snapshot", idx)
			}

			err := rdb.Del(ctx, keys...).Err()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkCacheTTLManagement tests TTL-related operations
func BenchmarkCacheTTLManagement(b *testing.B) {
	ctx := context.Background()
	rdb := helpers.SetupTestRedis(b)
	defer rdb.Close()

	b.Run("set_with_ttl", func(b *testing.B) {
		snapshot := fixtures.GenerateSnapshot()
		data, _ := json.Marshal(snapshot)

		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("validator:%d:snapshot", i)
			err := rdb.Set(ctx, key, data, 15*time.Minute).Err()
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("extend_ttl", func(b *testing.B) {
		// Seed cache
		snapshot := fixtures.GenerateSnapshot()
		data, _ := json.Marshal(snapshot)
		key := "validator:1:snapshot"
		rdb.Set(ctx, key, data, 15*time.Minute)

		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			err := rdb.Expire(ctx, key, 30*time.Minute).Err()
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("check_ttl", func(b *testing.B) {
		// Seed cache
		snapshot := fixtures.GenerateSnapshot()
		data, _ := json.Marshal(snapshot)
		key := "validator:1:snapshot"
		rdb.Set(ctx, key, data, 15*time.Minute)

		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := rdb.TTL(ctx, key).Result()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// seedRedisCache populates Redis with test data
func seedRedisCache(b *testing.B, rdb *redis.Client, count int) {
	ctx := context.Background()
	snapshots := fixtures.GenerateSnapshots(count)

	pipe := rdb.Pipeline()
	for _, snap := range snapshots {
		data, err := json.Marshal(snap)
		if err != nil {
			b.Fatalf("failed to marshal snapshot: %v", err)
		}

		key := fmt.Sprintf("validator:%d:snapshot", snap.ValidatorIndex)
		pipe.Set(ctx, key, data, 15*time.Minute)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		b.Fatalf("failed to seed Redis cache: %v", err)
	}
}
