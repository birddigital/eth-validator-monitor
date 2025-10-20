package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestIDMiddleware(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "generates request ID and embeds in context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test logger (disabled for clean test output)
			logger := zerolog.Nop()

			// Track what the downstream handler receives
			var capturedCtx context.Context
			var capturedRequestID string
			var capturedLogger zerolog.Logger
			var capturedLoggerExists bool

			// Create test handler that captures context values
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedCtx = r.Context()
				capturedRequestID, _ = RequestIDFromContext(r.Context())
				capturedLogger, capturedLoggerExists = LoggerFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			})

			// Create middleware and wrap handler
			middleware := NewRequestIDMiddleware(logger)
			handler := middleware.Middleware(testHandler)

			// Create test request
			req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
			rec := httptest.NewRecorder()

			// Execute request
			handler.ServeHTTP(rec, req)

			// Assertions
			require.NotNil(t, capturedCtx, "context should not be nil")

			// Assert request ID exists and is valid UUID
			assert.NotEmpty(t, capturedRequestID, "request ID should not be empty")
			_, err := uuid.Parse(capturedRequestID)
			assert.NoError(t, err, "request ID should be valid UUID")

			// Assert logger exists in context
			assert.True(t, capturedLoggerExists, "logger should exist in context")
			assert.NotNil(t, capturedLogger, "logger should not be nil")

			// Assert response header contains request ID
			assert.Equal(t, capturedRequestID, rec.Header().Get("X-Request-ID"))

			// Assert HTTP status
			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}
}

func TestRequestIDFromContext(t *testing.T) {
	tests := []struct {
		name      string
		setupCtx  func() context.Context
		wantID    string
		wantFound bool
	}{
		{
			name: "retrieves request ID from context",
			setupCtx: func() context.Context {
				return WithRequestID(context.Background(), "test-uuid-123")
			},
			wantID:    "test-uuid-123",
			wantFound: true,
		},
		{
			name: "returns empty when no request ID in context",
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantID:    "",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			gotID, gotFound := RequestIDFromContext(ctx)

			assert.Equal(t, tt.wantID, gotID)
			assert.Equal(t, tt.wantFound, gotFound)
		})
	}
}

func TestMustRequestIDFromContext(t *testing.T) {
	t.Run("returns request ID when present", func(t *testing.T) {
		ctx := WithRequestID(context.Background(), "test-uuid-456")
		got := MustRequestIDFromContext(ctx)
		assert.Equal(t, "test-uuid-456", got)
	})

	t.Run("returns empty string when absent", func(t *testing.T) {
		got := MustRequestIDFromContext(context.Background())
		assert.Equal(t, "", got)
	})
}

func TestLoggerFromContext(t *testing.T) {
	t.Run("retrieves logger from context", func(t *testing.T) {
		logger := zerolog.Nop()
		ctx := WithLogger(context.Background(), logger)

		gotLogger, gotFound := LoggerFromContext(ctx)
		assert.True(t, gotFound)
		assert.NotNil(t, gotLogger)
	})

	t.Run("returns false when no logger in context", func(t *testing.T) {
		_, gotFound := LoggerFromContext(context.Background())
		assert.False(t, gotFound)
	})
}

func TestMustLoggerFromContext(t *testing.T) {
	t.Run("returns logger when present", func(t *testing.T) {
		logger := zerolog.Nop()
		ctx := WithLogger(context.Background(), logger)

		got := MustLoggerFromContext(ctx)
		assert.NotNil(t, got)
	})

	t.Run("returns Nop logger when absent", func(t *testing.T) {
		got := MustLoggerFromContext(context.Background())
		assert.NotNil(t, got)
		// Nop logger is still a valid logger, just doesn't output
	})
}

func TestMultipleRequests(t *testing.T) {
	logger := zerolog.Nop()
	middleware := NewRequestIDMiddleware(logger)

	// Track all generated request IDs
	var requestIDs []string

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID, _ := RequestIDFromContext(r.Context())
		requestIDs = append(requestIDs, requestID)
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.Middleware(testHandler)

	// Make multiple requests
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/graphql", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	// Assert all IDs are unique
	assert.Len(t, requestIDs, 10)
	uniqueIDs := make(map[string]bool)
	for _, id := range requestIDs {
		assert.False(t, uniqueIDs[id], "request ID should be unique")
		uniqueIDs[id] = true
	}
}

func BenchmarkRequestIDMiddleware(b *testing.B) {
	logger := zerolog.Nop()
	middleware := NewRequestIDMiddleware(logger)

	handler := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate reading from context
		_, _ = RequestIDFromContext(r.Context())
		_, _ = LoggerFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/graphql", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}
