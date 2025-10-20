package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestNewRouter(t *testing.T) {
	logger := zerolog.Nop() // No-op logger for tests

	cfg := RouterConfig{
		Logger:        &logger,
		Environment:   "test",
		EnableCORS:    false,
		CompressLevel: 0,
	}

	router := NewRouter(cfg)
	assert.NotNil(t, router, "Router should not be nil")

	// Add test route
	router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	t.Run("successful request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "test response", w.Body.String())
	})

	t.Run("request ID header added", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.NotEmpty(t, w.Header().Get("X-Request-ID"), "X-Request-ID header should be set")
	})

	t.Run("security headers added", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Check for some security headers
		assert.NotEmpty(t, w.Header().Get("X-Frame-Options"), "X-Frame-Options should be set")
		assert.NotEmpty(t, w.Header().Get("X-Content-Type-Options"), "X-Content-Type-Options should be set")
	})
}

func TestPanicRecovery(t *testing.T) {
	logger := zerolog.Nop()

	cfg := RouterConfig{
		Logger:        &logger,
		Environment:   "test",
		CompressLevel: 0,
	}

	router := NewRouter(cfg)

	// Add route that panics
	router.Get("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	req := httptest.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code, "Should return 500 on panic")
	assert.Contains(t, w.Body.String(), "Internal Server Error", "Should return error message")
}

func TestCORSMiddleware(t *testing.T) {
	logger := zerolog.Nop()

	cfg := RouterConfig{
		Logger:         &logger,
		Environment:    "test",
		EnableCORS:     true,
		AllowedOrigins: []string{"http://localhost:3000"},
		CompressLevel:  0,
	}

	router := NewRouter(cfg)

	router.Get("/api/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	t.Run("CORS preflight request", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/api/test", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		req.Header.Set("Access-Control-Request-Method", "GET")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.NotEmpty(t, w.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("CORS actual request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
	})
}

func TestRateLimitMiddleware(t *testing.T) {
	logger := zerolog.Nop()

	cfg := RouterConfig{
		Logger:         &logger,
		Environment:    "test",
		RateLimitRPS:   2,  // 2 requests per second
		RateLimitBurst: 2,  // Burst of 2
		CompressLevel:  0,
	}

	router := NewRouter(cfg)

	router.Get("/limited", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	t.Run("allows requests within limit", func(t *testing.T) {
		// First request should succeed
		req1 := httptest.NewRequest("GET", "/limited", nil)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusOK, w1.Code)

		// Second request should succeed (within burst)
		req2 := httptest.NewRequest("GET", "/limited", nil)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusOK, w2.Code)
	})
}
