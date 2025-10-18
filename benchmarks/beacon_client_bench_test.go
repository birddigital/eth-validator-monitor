package benchmarks

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/birddigital/eth-validator-monitor/benchmarks/fixtures"
)

// BenchmarkBeaconClientRetryLogic tests retry performance under different failure scenarios
func BenchmarkBeaconClientRetryLogic(b *testing.B) {
	scenarios := []struct {
		name        string
		failureRate float64 // 0.0 = no failures, 0.5 = 50% failure rate
	}{
		{"no_failures", 0.0},
		{"10pct_failures", 0.1},
		{"30pct_failures", 0.3},
		{"50pct_failures", 0.5},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			mockBeacon := fixtures.NewMockBeaconClientWithFailures(scenario.failureRate)

			b.ReportAllocs()
			b.ResetTimer()

			successCount := 0
			failureCount := 0

			for i := 0; i < b.N; i++ {
				_, err := mockBeacon.GetValidatorStatus(uint64(i % 1000))
				if err != nil {
					failureCount++
					// Expected for failure scenarios
					continue
				}
				successCount++
			}

			b.StopTimer()

			// Report success rate
			if b.N > 0 {
				successRate := float64(successCount) / float64(b.N)
				b.ReportMetric(successRate*100, "success_%")
			}
		})
	}
}

// BenchmarkBeaconClientBatchRequests tests performance of batched validator queries
func BenchmarkBeaconClientBatchRequests(b *testing.B) {
	validators := fixtures.GenerateValidators(10000)
	mockBeacon := fixtures.NewMockBeaconClient(validators)

	batchSizes := []int{1, 10, 50, 100, 500}

	for _, batchSize := range batchSizes {
		b.Run(fmt.Sprintf("batch_%d", batchSize), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// Simulate batched requests
				for j := 0; j < batchSize; j++ {
					validatorIndex := uint64((i*batchSize + j) % len(validators))
					_, err := mockBeacon.GetValidatorStatus(validatorIndex)
					if err != nil {
						b.Fatalf("failed to get validator status: %v", err)
					}
				}
			}

			// Report throughput
			b.ReportMetric(float64(batchSize)/b.Elapsed().Seconds(), "requests/sec")
		})
	}
}

// BenchmarkBeaconClientConcurrentRequests tests concurrent request handling
func BenchmarkBeaconClientConcurrentRequests(b *testing.B) {
	validators := fixtures.GenerateValidators(10000)
	mockBeacon := fixtures.NewMockBeaconClient(validators)

	concurrencyLevels := []int{1, 10, 50, 100}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("concurrent_%d", concurrency), func(b *testing.B) {
			b.ReportAllocs()
			b.SetParallelism(concurrency)
			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					validatorIndex := uint64(i % len(validators))
					_, err := mockBeacon.GetValidatorStatus(validatorIndex)
					if err != nil {
						b.Errorf("failed to get validator status: %v", err)
					}
					i++
				}
			})
		})
	}
}

// BenchmarkBeaconClientLatencySimulation tests performance under different latency conditions
func BenchmarkBeaconClientLatencySimulation(b *testing.B) {
	validators := fixtures.GenerateValidators(1000)

	latencies := []time.Duration{
		10 * time.Millisecond,
		50 * time.Millisecond,
		100 * time.Millisecond,
		200 * time.Millisecond,
	}

	for _, latency := range latencies {
		b.Run(fmt.Sprintf("latency_%dms", latency.Milliseconds()), func(b *testing.B) {
			mockBeacon := fixtures.NewMockBeaconClient(validators)
			mockBeacon.requestLatency = latency

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := mockBeacon.GetValidatorStatus(uint64(i % len(validators)))
				if err != nil {
					b.Fatalf("failed to get validator status: %v", err)
				}
			}
		})
	}
}

// BenchmarkBeaconClientBalanceFetching tests balance query performance
func BenchmarkBeaconClientBalanceFetching(b *testing.B) {
	validators := fixtures.GenerateValidators(10000)
	mockBeacon := fixtures.NewMockBeaconClient(validators)

	b.Run("single_balance_query", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := mockBeacon.GetValidatorBalance(uint64(i % len(validators)))
			if err != nil {
				b.Fatalf("failed to get balance: %v", err)
			}
		}
	})

	b.Run("batch_balance_queries", func(b *testing.B) {
		batchSize := 100

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			for j := 0; j < batchSize; j++ {
				validatorIndex := uint64((i*batchSize + j) % len(validators))
				_, err := mockBeacon.GetValidatorBalance(validatorIndex)
				if err != nil {
					b.Fatalf("failed to get balance: %v", err)
				}
			}
		}

		b.ReportMetric(float64(batchSize)/b.Elapsed().Seconds(), "balances/sec")
	})
}

// BenchmarkBeaconClientMemoryUsage tests memory efficiency of client operations
func BenchmarkBeaconClientMemoryUsage(b *testing.B) {
	validators := fixtures.GenerateValidators(10000)
	mockBeacon := fixtures.NewMockBeaconClient(validators)

	b.Run("sequential_requests", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Fetch multiple validators sequentially
			for j := 0; j < 1000; j++ {
				validatorIndex := uint64((i*1000 + j) % len(validators))
				_, err := mockBeacon.GetValidatorStatus(validatorIndex)
				if err != nil {
					b.Fatalf("failed to get validator: %v", err)
				}
			}
		}
	})
}
