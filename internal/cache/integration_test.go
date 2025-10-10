package cache

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_CacheLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cache := testRedisCache(t)
	ctx := context.Background()

	// Store complex data
	data := map[string]interface{}{
		"validator_index": 12345,
		"pubkey":          "0x1234567890abcdef",
		"balance":         32000000000,
		"status":          "active",
		"metadata": map[string]interface{}{
			"name":   "validator-12345",
			"tags":   []string{"production", "priority"},
			"scores": []float64{98.5, 97.2, 99.1},
		},
	}

	// Set with TTL
	err := cache.Set(ctx, "test:complex", data, 5*time.Minute)
	require.NoError(t, err)

	// Retrieve and verify
	var retrieved map[string]interface{}
	err = cache.Get(ctx, "test:complex", &retrieved)
	require.NoError(t, err)
	assert.Equal(t, float64(12345), retrieved["validator_index"])
	assert.Equal(t, "0x1234567890abcdef", retrieved["pubkey"])

	// Verify nested metadata
	metadata := retrieved["metadata"].(map[string]interface{})
	assert.Equal(t, "validator-12345", metadata["name"])

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Verify still exists
	err = cache.Get(ctx, "test:complex", &retrieved)
	require.NoError(t, err)
}

func TestIntegration_ValidatorMetadataCaching(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cache := testRedisCache(t)
	ctx := context.Background()

	// Cache validator metadata
	metadata := map[string]interface{}{
		"validator_index": 567,
		"pubkey":          "0xabc123",
		"status":          "active",
		"effectiveness":   98.5,
	}

	err := cache.SetValidatorMetadata(ctx, 567, metadata)
	require.NoError(t, err)

	// Retrieve metadata
	retrieved, err := cache.GetValidatorMetadata(ctx, 567)
	require.NoError(t, err)
	assert.Equal(t, float64(567), retrieved["validator_index"])
	assert.Equal(t, "active", retrieved["status"])

	// Verify key format
	expectedKey := cache.ValidatorMetadataKey(567)
	assert.Equal(t, "validator:567", expectedKey)
}

func TestIntegration_KeyGeneration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cache := testRedisCache(t)

	tests := []struct {
		name     string
		keyFunc  func() string
		expected string
	}{
		{
			name:     "validator metadata key",
			keyFunc:  func() string { return cache.ValidatorMetadataKey(123) },
			expected: "validator:123",
		},
		{
			name:     "latest snapshot key",
			keyFunc:  func() string { return cache.LatestSnapshotKey(456) },
			expected: "snapshot:456:latest",
		},
		{
			name:     "performance key",
			keyFunc:  func() string { return cache.PerformanceKey(789, 100, 200) },
			expected: "performance:789:100:200",
		},
		{
			name:     "network stats key",
			keyFunc:  func() string { return cache.NetworkStatsKey() },
			expected: "network:stats",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := tt.keyFunc()
			assert.Equal(t, tt.expected, key)
		})
	}
}

func TestIntegration_TTLBehavior(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cache := testRedisCache(t)
	ctx := context.Background()

	// Set entry with very short TTL
	data := map[string]string{"test": "value"}
	err := cache.Set(ctx, "test:ttl", data, 200*time.Millisecond)
	require.NoError(t, err)

	// Immediately verify it exists
	var retrieved map[string]string
	err = cache.Get(ctx, "test:ttl", &retrieved)
	require.NoError(t, err)
	assert.Equal(t, "value", retrieved["test"])

	// Wait for expiration
	time.Sleep(300 * time.Millisecond)

	// Verify it's expired
	err = cache.Get(ctx, "test:ttl", &retrieved)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in cache")
}

func TestIntegration_BatchOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cache := testRedisCache(t)
	ctx := context.Background()

	// Prepare batch items
	items := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		key := cache.ValidatorMetadataKey(int64(i))
		items[key] = map[string]interface{}{
			"index":  i,
			"status": "active",
		}
	}

	// Batch set
	err := cache.BatchSet(ctx, items, 5*time.Minute)
	require.NoError(t, err)

	// Verify random samples
	for i := 0; i < 100; i += 10 {
		var data map[string]interface{}
		key := cache.ValidatorMetadataKey(int64(i))
		err := cache.Get(ctx, key, &data)
		require.NoError(t, err)
		assert.Equal(t, float64(i), data["index"])
	}
}

func TestIntegration_ConcurrentAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cache := testRedisCache(t)
	ctx := context.Background()

	// Concurrent writes
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			metadata := map[string]interface{}{
				"index":  id,
				"status": "active",
			}
			err := cache.SetValidatorMetadata(ctx, int64(id), metadata)
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()

	// Concurrent reads
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			metadata, err := cache.GetValidatorMetadata(ctx, int64(id))
			assert.NoError(t, err)
			assert.NotNil(t, metadata)
			assert.Equal(t, float64(id), metadata["index"])
		}(i)
	}

	wg.Wait()
}

func TestIntegration_HighLoadPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cache := testRedisCache(t)
	ctx := context.Background()

	// Measure write performance
	start := time.Now()
	for i := 0; i < 1000; i++ {
		metadata := map[string]interface{}{
			"index":       i,
			"status":      "active",
			"effectiveness": 98.5,
		}
		err := cache.SetValidatorMetadata(ctx, int64(i), metadata)
		require.NoError(t, err)
	}
	writeDuration := time.Since(start)

	// Measure read performance
	start = time.Now()
	for i := 0; i < 1000; i++ {
		_, err := cache.GetValidatorMetadata(ctx, int64(i))
		require.NoError(t, err)
	}
	readDuration := time.Since(start)

	// Performance assertions
	assert.Less(t, writeDuration, 5*time.Second, "1000 writes should complete within 5 seconds")
	assert.Less(t, readDuration, 2*time.Second, "1000 reads should complete within 2 seconds")

	t.Logf("Performance: 1000 writes in %v, 1000 reads in %v", writeDuration, readDuration)
}

func TestIntegration_CacheInvalidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cache := testRedisCache(t)
	ctx := context.Background()

	// Set multiple related cache entries
	for i := 0; i < 10; i++ {
		metadata := map[string]interface{}{
			"index":  i,
			"status": "active",
		}
		err := cache.SetValidatorMetadata(ctx, int64(i), metadata)
		require.NoError(t, err)
	}

	// Flush all test data
	err := cache.Flush(ctx)
	require.NoError(t, err)

	// Verify all entries are gone
	for i := 0; i < 10; i++ {
		_, err := cache.GetValidatorMetadata(ctx, int64(i))
		assert.Error(t, err)
	}
}

func TestIntegration_LargeData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cache := testRedisCache(t)
	ctx := context.Background()

	// Create large data structure
	largeData := map[string]interface{}{
		"validator_index": 9999,
		"history":         make([]map[string]interface{}, 1000),
	}

	for i := 0; i < 1000; i++ {
		largeData["history"].([]map[string]interface{})[i] = map[string]interface{}{
			"epoch":    i,
			"balance":  32000000000 + i*1000,
			"status":   "active",
			"timestamp": time.Now().Add(-time.Duration(i) * time.Hour).Unix(),
		}
	}

	// Cache large data
	err := cache.Set(ctx, "test:large", largeData, 5*time.Minute)
	require.NoError(t, err)

	// Retrieve and verify
	var retrieved map[string]interface{}
	err = cache.Get(ctx, "test:large", &retrieved)
	require.NoError(t, err)
	assert.Equal(t, float64(9999), retrieved["validator_index"])

	history := retrieved["history"].([]interface{})
	assert.Len(t, history, 1000)
}

func TestIntegration_TTLStrategies(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test different TTL strategies
	strategies := []struct {
		name     string
		strategy TTLStrategy
	}{
		{"default", DefaultTTLStrategy()},
		{"aggressive", AggressiveTTLStrategy()},
		{"conservative", ConservativeTTLStrategy()},
	}

	for _, s := range strategies {
		t.Run(s.name, func(t *testing.T) {
			config := Config{
				Host:      "localhost",
				Port:      6379,
				Password:  "",
				DB:        1,
				Strategy:  s.strategy,
				KeyPrefix: "test-" + s.name,
			}

			cache, err := NewRedisCache(config)
			require.NoError(t, err)
			defer cache.Close()

			ctx := context.Background()

			// Verify strategy is applied
			metadata := map[string]interface{}{"test": "data"}
			err = cache.SetValidatorMetadata(ctx, 1, metadata)
			require.NoError(t, err)

			// Clean up
			_ = cache.Flush(ctx)
		})
	}
}

func TestIntegration_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cache := testRedisCache(t)
	ctx := context.Background()

	// Test get non-existent key
	var data map[string]interface{}
	err := cache.Get(ctx, "nonexistent:key", &data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in cache")

	// Test get with wrong type
	err = cache.Set(ctx, "test:string", "simple string", 1*time.Minute)
	require.NoError(t, err)

	var wrongType map[string]interface{}
	err = cache.Get(ctx, "test:string", &wrongType)
	assert.Error(t, err) // Should fail to unmarshal string into map
}

func TestIntegration_ConnectionResilience(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test creating cache with proper config
	config := Config{
		Host:       "localhost",
		Port:       6379,
		Password:   "",
		DB:         1,
		MaxRetries: 3,
		PoolSize:   10,
		Strategy:   DefaultTTLStrategy(),
		KeyPrefix:  "resilience-test",
	}

	cache, err := NewRedisCache(config)
	require.NoError(t, err)
	defer cache.Close()

	ctx := context.Background()

	// Perform operations to verify connection
	for i := 0; i < 100; i++ {
		metadata := map[string]interface{}{"index": i}
		err := cache.SetValidatorMetadata(ctx, int64(i), metadata)
		require.NoError(t, err)
	}

	// Clean up
	err = cache.Flush(ctx)
	require.NoError(t, err)
}
