package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCacheControl(t *testing.T) {
	tests := []struct {
		name          string
		maxAge        int
		expectedValue string
	}{
		{
			name:          "one year cache",
			maxAge:        31536000,
			expectedValue: "public, max-age=31536000",
		},
		{
			name:          "one day cache",
			maxAge:        86400,
			expectedValue: "public, max-age=86400",
		},
		{
			name:          "one hour cache",
			maxAge:        3600,
			expectedValue: "public, max-age=3600",
		},
		{
			name:          "no cache",
			maxAge:        0,
			expectedValue: "public, max-age=0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test handler that returns success
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("test content"))
			})

			// Wrap with cache control middleware
			wrappedHandler := CacheControl(tt.maxAge)(handler)

			// Create test request and response recorder
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			// Execute the handler
			wrappedHandler.ServeHTTP(rec, req)

			// Assert the Cache-Control header is set correctly
			assert.Equal(t, tt.expectedValue, rec.Header().Get("Cache-Control"),
				"Cache-Control header should match expected value")

			// Assert the response is still successful
			assert.Equal(t, http.StatusOK, rec.Code,
				"Response status should be OK")

			// Assert the response body is unchanged
			assert.Equal(t, "test content", rec.Body.String(),
				"Response body should be unchanged by middleware")
		})
	}
}

func TestCacheControlHeaderPersistence(t *testing.T) {
	// Test that cache control header persists even if handler sets other headers
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Custom-Header", "custom-value")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	wrappedHandler := CacheControl(31536000)(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	// Verify all headers are present
	assert.Equal(t, "public, max-age=31536000", rec.Header().Get("Cache-Control"))
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	assert.Equal(t, "custom-value", rec.Header().Get("X-Custom-Header"))
}

func TestCacheControlWithDifferentHTTPMethods(t *testing.T) {
	methods := []string{http.MethodGet, http.MethodHead, http.MethodOptions}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			wrappedHandler := CacheControl(31536000)(handler)

			req := httptest.NewRequest(method, "/test", nil)
			rec := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rec, req)

			// Cache-Control should be set regardless of HTTP method
			assert.Equal(t, "public, max-age=31536000", rec.Header().Get("Cache-Control"),
				"Cache-Control should be set for %s requests", method)
		})
	}
}
