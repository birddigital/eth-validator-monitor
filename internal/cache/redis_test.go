package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testData struct {
	ID    int
	Name  string
	Value float64
}

// testRedisCache creates a test Redis cache
func testRedisCache(t *testing.T) *RedisCache {
	t.Helper()

	config := Config{
		Host:      "localhost",
		Port:      6379,
		Password:  "",
		DB:        1, // Use test database
		Strategy:  DefaultTTLStrategy(),
		KeyPrefix: "test",
	}

	redisCache, err := NewRedisCache(config)
	require.NoError(t, err, "Failed to create test Redis cache")

	// Clean test database
	ctx := context.Background()
	_ = redisCache.Flush(ctx)

	t.Cleanup(func() {
		_ = redisCache.Flush(ctx)
		_ = redisCache.Close()
	})

	return redisCache
}

func TestRedisCache_SetAndGet(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cache := testRedisCache(t)
	ctx := context.Background()

	data := &testData{
		ID:    123,
		Name:  "test",
		Value: 45.67,
	}

	// Set data
	err := cache.Set(ctx, "test:key", data, 1*time.Minute)
	require.NoError(t, err)

	// Get data
	var retrieved testData
	err = cache.Get(ctx, "test:key", &retrieved)
	require.NoError(t, err)

	assert.Equal(t, data.ID, retrieved.ID)
	assert.Equal(t, data.Name, retrieved.Name)
	assert.Equal(t, data.Value, retrieved.Value)
}

func TestRedisCache_Get_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cache := testRedisCache(t)
	ctx := context.Background()

	var data testData
	err := cache.Get(ctx, "nonexistent:key", &data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in cache")
}

func TestRedisCache_SetValidatorMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cache := testRedisCache(t)
	ctx := context.Background()

	metadata := map[string]interface{}{
		"index":  123,
		"pubkey": "0x1234...",
		"status": "active",
	}

	err := cache.SetValidatorMetadata(ctx, 123, metadata)
	require.NoError(t, err)

	// Verify it was cached with correct key
	var retrieved map[string]interface{}
	err = cache.Get(ctx, cache.ValidatorMetadataKey(123), &retrieved)
	require.NoError(t, err)
	assert.Equal(t, metadata, retrieved)
}

func TestRedisCache_GetValidatorMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cache := testRedisCache(t)
	ctx := context.Background()

	metadata := map[string]interface{}{
		"index": float64(456), // JSON numbers become float64
		"name":  "validator-456",
	}

	// Set metadata
	err := cache.SetValidatorMetadata(ctx, 456, metadata)
	require.NoError(t, err)

	// Get metadata
	retrieved, err := cache.GetValidatorMetadata(ctx, 456)
	require.NoError(t, err)
	assert.Equal(t, metadata, retrieved)
}

func TestRedisCache_GetValidatorMetadata_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cache := testRedisCache(t)
	ctx := context.Background()

	metadata, err := cache.GetValidatorMetadata(ctx, 99999)
	assert.Error(t, err)
	assert.Nil(t, metadata)
}

func TestRedisCache_TTLStrategies(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cache := testRedisCache(t)
	ctx := context.Background()

	data := &testData{ID: 1, Name: "test"}

	// Test each TTL helper
	tests := []struct {
		name string
		ttl  time.Duration
	}{
		{"validator metadata", GetValidatorMetadataTTL()},
		{"validator snapshot", GetValidatorSnapshotTTL()},
		{"performance metrics", GetPerformanceMetricsTTL()},
		{"alert cache", GetAlertCacheTTL()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "test:" + tt.name
			err := cache.Set(ctx, key, data, tt.ttl)
			require.NoError(t, err)

			var retrieved testData
			err = cache.Get(ctx, key, &retrieved)
			require.NoError(t, err)
			assert.Equal(t, data.ID, retrieved.ID)
		})
	}
}

func TestRedisCache_Expiration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cache := testRedisCache(t)
	ctx := context.Background()

	data := &testData{ID: 1, Name: "expires"}

	// Set with very short TTL
	err := cache.Set(ctx, "test:expires", data, 100*time.Millisecond)
	require.NoError(t, err)

	// Should be available immediately
	var retrieved testData
	err = cache.Get(ctx, "test:expires", &retrieved)
	require.NoError(t, err)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	err = cache.Get(ctx, "test:expires", &retrieved)
	assert.Error(t, err)
}

func TestRedisCache_KeyGenerators(t *testing.T) {
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

func TestRedisCache_ConcurrentAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cache := testRedisCache(t)
	ctx := context.Background()

	// Concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			data := &testData{ID: id, Name: "concurrent"}
			err := cache.Set(ctx, cache.ValidatorMetadataKey(int64(id)), data, 1*time.Minute)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all writes
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all writes succeeded
	for i := 0; i < 10; i++ {
		var data testData
		err := cache.Get(ctx, cache.ValidatorMetadataKey(int64(i)), &data)
		require.NoError(t, err)
		assert.Equal(t, i, data.ID)
	}
}
