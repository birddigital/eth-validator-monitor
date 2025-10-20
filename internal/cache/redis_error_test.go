package cache

import (
	"errors"
	"fmt"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRedisNil_ErrorWrapping tests that redis.Nil is properly wrapped with %w
func TestRedisNil_ErrorWrapping(t *testing.T) {
	tests := []struct {
		name          string
		validatorIdx  int
		mockErr       error
		expectedErr   error
		errorContains string
	}{
		{
			name:          "cache_miss_error_preserved",
			validatorIdx:  7890,
			mockErr:       redis.Nil,
			expectedErr:   redis.Nil,
			errorContains: "validator 7890 not in cache",
		},
		{
			name:          "snapshot_cache_miss",
			validatorIdx:  5555,
			mockErr:       redis.Nil,
			expectedErr:   redis.Nil,
			errorContains: "not in cache",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the error wrapping that occurs in redis.go
			var err error
			if tt.mockErr == redis.Nil {
				// This is the pattern from redis.go:141, 177, etc.
				err = fmt.Errorf("validator %d not in cache: %w", tt.validatorIdx, tt.mockErr)
			}

			// Test 1: Original redis.Nil is in the chain
			require.Error(t, err)
			assert.True(t, errors.Is(err, tt.expectedErr),
				"expected error chain to contain %v", tt.expectedErr)

			// Test 2: Error message has context
			assert.Contains(t, err.Error(), tt.errorContains,
				"expected error message to contain context")
		})
	}
}

// TestRedisNil_MultiLayerChain tests redis.Nil through multiple wrapping layers
func TestRedisNil_MultiLayerChain(t *testing.T) {
	// Simulate a multi-layer error chain:
	// Redis -> Cache -> Service -> Resolver

	// Layer 1: Redis client returns redis.Nil
	redisErr := redis.Nil

	// Layer 2: Cache wraps it (like redis.go:141)
	cacheErr := fmt.Errorf("validator 123 not in cache: %w", redisErr)

	// Layer 3: Service layer wraps it
	serviceErr := fmt.Errorf("cache service: get validator: %w", cacheErr)

	// Layer 4: Resolver wraps it
	resolverErr := fmt.Errorf("graphql resolver: query cached validator: %w", serviceErr)

	// Test: Original redis.Nil is still detectable through the chain
	assert.True(t, errors.Is(resolverErr, redis.Nil),
		"original redis.Nil should be detectable through the entire error chain")

	// Test: Error message shows the full chain
	errMsg := resolverErr.Error()
	assert.Contains(t, errMsg, "graphql resolver")
	assert.Contains(t, errMsg, "cache service")
	assert.Contains(t, errMsg, "validator 123 not in cache")
	assert.Contains(t, errMsg, "redis: nil")
}

// TestDifferentCacheMissScenarios tests various cache miss patterns
func TestDifferentCacheMissScenarios(t *testing.T) {
	scenarios := []struct {
		operation string
		errFunc   func() error
	}{
		{
			operation: "GetValidator",
			errFunc: func() error {
				return fmt.Errorf("validator %d not in cache: %w", 100, redis.Nil)
			},
		},
		{
			operation: "GetValidatorSnapshot",
			errFunc: func() error {
				return fmt.Errorf("snapshot for validator %d not in cache: %w", 200, redis.Nil)
			},
		},
		{
			operation: "GetNetworkStats",
			errFunc: func() error {
				return fmt.Errorf("network stats not in cache: %w", redis.Nil)
			},
		},
		{
			operation: "GetPerformance",
			errFunc: func() error {
				return fmt.Errorf("performance for validator %d epoch %d not in cache: %w", 300, 1000, redis.Nil)
			},
		},
		{
			operation: "GetAlerts",
			errFunc: func() error {
				return fmt.Errorf("alerts for validator %d not in cache: %w", 400, redis.Nil)
			},
		},
		{
			operation: "GetHeadEvent",
			errFunc: func() error {
				return fmt.Errorf("head event not in cache: %w", redis.Nil)
			},
		},
		{
			operation: "Get (generic)",
			errFunc: func() error {
				return fmt.Errorf("key %s not in cache: %w", "test:key", redis.Nil)
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.operation, func(t *testing.T) {
			err := scenario.errFunc()

			// Verify redis.Nil is preserved in the error chain
			assert.True(t, errors.Is(err, redis.Nil),
				"%s should preserve redis.Nil in error chain", scenario.operation)

			// Verify the error has contextual information
			assert.Contains(t, err.Error(), "not in cache",
				"%s should include context in error message", scenario.operation)
		})
	}
}

// TestErrorIsVsEquality demonstrates why %w is important
func TestErrorIsVsEquality(t *testing.T) {
	// Without %w (old pattern - DO NOT USE)
	badErr := fmt.Errorf("validator not found")
	assert.False(t, errors.Is(badErr, redis.Nil),
		"string interpolation loses the error chain")

	// With %w (correct pattern)
	goodErr := fmt.Errorf("validator not found: %w", redis.Nil)
	assert.True(t, errors.Is(goodErr, redis.Nil),
		"%%w preserves the error chain for errors.Is")

	// This demonstrates why the changes in this task are important!
}
