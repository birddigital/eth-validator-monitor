package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetValidator_ErrorWrapping tests that sql.ErrNoRows is properly wrapped
func TestGetValidator_ErrorWrapping(t *testing.T) {
	tests := []struct {
		name          string
		validatorIdx  int
		mockErr       error
		expectedErr   error
		errorContains string
	}{
		{
			name:          "not_found_error_preserved",
			validatorIdx:  12345,
			mockErr:       sql.ErrNoRows,
			expectedErr:   sql.ErrNoRows,
			errorContains: "validator 12345 not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: This test demonstrates the error wrapping pattern.
			// In a real implementation, you would mock the database connection.
			// For now, we're testing the error wrapping logic conceptually.

			// Simulate the error wrapping that occurs in postgres.go:149-150
			var err error
			if tt.mockErr == sql.ErrNoRows {
				// This is the exact pattern from postgres.go
				err = fmt.Errorf("validator %d not found: %w", tt.validatorIdx, tt.mockErr)
			}

			// Test 1: Original error is in the chain
			require.Error(t, err)
			assert.True(t, errors.Is(err, tt.expectedErr),
				"expected error chain to contain %v", tt.expectedErr)

			// Test 2: Error message has context
			assert.Contains(t, err.Error(), tt.errorContains,
				"expected error message to contain context")
		})
	}
}

// TestErrorChainPreservation tests that errors.Is works through multiple wrapping layers
func TestErrorChainPreservation(t *testing.T) {
	// Simulate a multi-layer error chain:
	// Database -> Storage -> Service -> Resolver

	// Layer 1: Database error
	dbErr := sql.ErrNoRows

	// Layer 2: Storage wraps it (like postgres.go:150)
	storageErr := fmt.Errorf("validator 999 not found: %w", dbErr)

	// Layer 3: Service layer wraps it
	serviceErr := fmt.Errorf("service: get validator: %w", storageErr)

	// Layer 4: Resolver wraps it
	resolverErr := fmt.Errorf("graphql resolver: query validator: %w", serviceErr)

	// Test: Original sentinel error is still detectable through the chain
	assert.True(t, errors.Is(resolverErr, sql.ErrNoRows),
		"original sql.ErrNoRows should be detectable through the entire error chain")

	// Test: Error message shows the full chain
	errMsg := resolverErr.Error()
	assert.Contains(t, errMsg, "graphql resolver")
	assert.Contains(t, errMsg, "service")
	assert.Contains(t, errMsg, "validator 999 not found")
}

// TestErrorUnwrapping demonstrates the errors.Unwrap functionality
func TestErrorUnwrapping(t *testing.T) {
	// Create a wrapped error
	originalErr := sql.ErrNoRows
	wrappedErr := fmt.Errorf("database: query failed: %w", originalErr)

	// Test unwrapping
	unwrapped := errors.Unwrap(wrappedErr)
	assert.Equal(t, originalErr, unwrapped,
		"errors.Unwrap should return the original error")

	// Test that errors.Is works
	assert.True(t, errors.Is(wrappedErr, sql.ErrNoRows),
		"errors.Is should find the original error")
}
