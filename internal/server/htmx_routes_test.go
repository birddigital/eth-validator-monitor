package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/birddigital/eth-validator-monitor/graph/middleware"
	"github.com/birddigital/eth-validator-monitor/internal/web/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHTMXRouteGroup_Integration tests the /api/htmx route group with full middleware stack
func TestHTMXRouteGroup_Integration(t *testing.T) {
	logger := zerolog.Nop()

	cfg := RouterConfig{
		Logger:        &logger,
		Environment:   "test",
		EnableCORS:    false,
		CompressLevel: 0,
	}

	router := NewRouter(cfg)

	// Set up HTMX route group (simulating main.go registerRoutes)
	router.Route("/api/htmx", func(r chi.Router) {
		htmxExampleHandler := handlers.NewHTMXExampleHandler()
		r.Get("/dashboard", htmxExampleHandler.ServeHTTP)
	})

	tests := []struct {
		name            string
		path            string
		headers         map[string]string
		expectedStatus  int
		expectFullPage  bool
		expectPartial   bool
		checkResponse   func(t *testing.T, body string, headers http.Header)
	}{
		{
			name:           "HTMX route without HX-Request returns full page",
			path:           "/api/htmx/dashboard",
			headers:        map[string]string{},
			expectedStatus: http.StatusOK,
			expectFullPage: true,
			expectPartial:  false,
			checkResponse: func(t *testing.T, body string, headers http.Header) {
				assert.Contains(t, body, "</html>", "should contain closing HTML tag")
				assert.Equal(t, "text/html; charset=utf-8", headers.Get("Content-Type"))
			},
		},
		{
			name: "HTMX route with HX-Request returns partial",
			path: "/api/htmx/dashboard",
			headers: map[string]string{
				"HX-Request": "true",
			},
			expectedStatus: http.StatusOK,
			expectFullPage: false,
			expectPartial:  true,
			checkResponse: func(t *testing.T, body string, headers http.Header) {
				assert.NotContains(t, body, "</html>", "should NOT contain closing HTML tag")
				assert.NotContains(t, body, "<head>", "should NOT contain head tag")
				assert.Equal(t, "text/html; charset=utf-8", headers.Get("Content-Type"))
			},
		},
		{
			name: "HTMX route with target header",
			path: "/api/htmx/dashboard",
			headers: map[string]string{
				"HX-Request": "true",
				"HX-Target":  "main-content",
			},
			expectedStatus: http.StatusOK,
			expectPartial:  true,
			checkResponse: func(t *testing.T, body string, headers http.Header) {
				assert.NotContains(t, body, "</html>", "should return partial")
			},
		},
		{
			name: "HTMX route with trigger header",
			path: "/api/htmx/dashboard",
			headers: map[string]string{
				"HX-Request": "true",
				"HX-Trigger": "refresh-button",
			},
			expectedStatus: http.StatusOK,
			expectPartial:  true,
			checkResponse: func(t *testing.T, body string, headers http.Header) {
				assert.NotContains(t, body, "</html>", "should return partial")
			},
		},
		{
			name: "HTMX route with false HX-Request",
			path: "/api/htmx/dashboard",
			headers: map[string]string{
				"HX-Request": "false",
			},
			expectedStatus: http.StatusOK,
			expectFullPage: true,
			checkResponse: func(t *testing.T, body string, headers http.Header) {
				assert.Contains(t, body, "</html>", "should return full page (HX-Request must be 'true')")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "Status code should match")

			if tt.checkResponse != nil {
				tt.checkResponse(t, w.Body.String(), w.Header())
			}
		})
	}
}

// TestHTMXRouteGroup_MiddlewareStack verifies HTMX middleware is properly applied
func TestHTMXRouteGroup_MiddlewareStack(t *testing.T) {
	logger := zerolog.Nop()

	cfg := RouterConfig{
		Logger:        &logger,
		Environment:   "test",
		EnableCORS:    false,
		CompressLevel: 0,
	}

	router := NewRouter(cfg)

	// Add a test handler that verifies HTMX context is available
	router.Route("/api/htmx", func(r chi.Router) {
		r.Get("/context-test", func(w http.ResponseWriter, r *http.Request) {
			// Verify HTMX middleware has populated context
			isHTMX := middleware.IsHTMXRequest(r.Context())
			if isHTMX {
				w.Header().Set("X-Test-HTMX-Detected", "true")
			} else {
				w.Header().Set("X-Test-HTMX-Detected", "false")
			}
			w.WriteHeader(http.StatusOK)
		})
	})

	t.Run("HTMX context available for true header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/htmx/context-test", nil)
		req.Header.Set("HX-Request", "true")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "true", w.Header().Get("X-Test-HTMX-Detected"),
			"HTMX middleware should detect HX-Request header")
	})

	t.Run("HTMX context available for missing header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/htmx/context-test", nil)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "false", w.Header().Get("X-Test-HTMX-Detected"),
			"HTMX middleware should return false for missing header")
	})
}

// TestHTMXRouteGroup_SecurityHeaders verifies security headers are applied to HTMX routes
func TestHTMXRouteGroup_SecurityHeaders(t *testing.T) {
	logger := zerolog.Nop()

	cfg := RouterConfig{
		Logger:        &logger,
		Environment:   "test",
		EnableCORS:    false,
		CompressLevel: 0,
	}

	router := NewRouter(cfg)

	router.Route("/api/htmx", func(r chi.Router) {
		htmxExampleHandler := handlers.NewHTMXExampleHandler()
		r.Get("/dashboard", htmxExampleHandler.ServeHTTP)
	})

	t.Run("security headers present on HTMX routes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/htmx/dashboard", nil)
		req.Header.Set("HX-Request", "true")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Verify key security headers
		assert.NotEmpty(t, w.Header().Get("X-Frame-Options"),
			"X-Frame-Options should be set")
		assert.NotEmpty(t, w.Header().Get("X-Content-Type-Options"),
			"X-Content-Type-Options should be set")
		assert.NotEmpty(t, w.Header().Get("X-Request-ID"),
			"X-Request-ID should be set")
	})
}

// TestHTMXRouteGroup_ContentNegotiation tests Accept header handling
func TestHTMXRouteGroup_ContentNegotiation(t *testing.T) {
	logger := zerolog.Nop()

	cfg := RouterConfig{
		Logger:        &logger,
		Environment:   "test",
		EnableCORS:    false,
		CompressLevel: 0,
	}

	router := NewRouter(cfg)

	// Add handler that responds to Accept header
	router.Route("/api/htmx", func(r chi.Router) {
		r.Get("/content-test", func(w http.ResponseWriter, r *http.Request) {
			if middleware.WantsJSON(r) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"type":"json"}`))
			} else {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`<div>html</div>`))
			}
		})
	})

	tests := []struct {
		name               string
		acceptHeader       string
		expectedContentType string
		expectedBody       string
	}{
		{
			name:               "default to HTML",
			acceptHeader:       "",
			expectedContentType: "text/html; charset=utf-8",
			expectedBody:       "<div>html</div>",
		},
		{
			name:               "explicit HTML request",
			acceptHeader:       "text/html",
			expectedContentType: "text/html; charset=utf-8",
			expectedBody:       "<div>html</div>",
		},
		{
			name:               "JSON request",
			acceptHeader:       "application/json",
			expectedContentType: "application/json",
			expectedBody:       `{"type":"json"}`,
		},
		{
			name:               "prefer HTML over JSON",
			acceptHeader:       "text/html,application/json;q=0.9",
			expectedContentType: "text/html; charset=utf-8",
			expectedBody:       "<div>html</div>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/htmx/content-test", nil)
			if tt.acceptHeader != "" {
				req.Header.Set("Accept", tt.acceptHeader)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, tt.expectedContentType, w.Header().Get("Content-Type"))
			assert.Equal(t, tt.expectedBody, w.Body.String())
		})
	}
}

// TestHTMXRouteGroup_404Handling verifies 404 behavior for non-existent HTMX routes
func TestHTMXRouteGroup_404Handling(t *testing.T) {
	logger := zerolog.Nop()

	cfg := RouterConfig{
		Logger:        &logger,
		Environment:   "test",
		EnableCORS:    false,
		CompressLevel: 0,
	}

	router := NewRouter(cfg)

	router.Route("/api/htmx", func(r chi.Router) {
		htmxExampleHandler := handlers.NewHTMXExampleHandler()
		r.Get("/dashboard", htmxExampleHandler.ServeHTTP)
	})

	t.Run("404 for non-existent HTMX route", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/htmx/nonexistent", nil)
		req.Header.Set("HX-Request", "true")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// TestHTMXRouteGroup_RequestIDPropagation verifies request ID is available in HTMX routes
func TestHTMXRouteGroup_RequestIDPropagation(t *testing.T) {
	logger := zerolog.Nop()

	cfg := RouterConfig{
		Logger:        &logger,
		Environment:   "test",
		EnableCORS:    false,
		CompressLevel: 0,
	}

	router := NewRouter(cfg)

	router.Route("/api/htmx", func(r chi.Router) {
		r.Get("/requestid-test", func(w http.ResponseWriter, r *http.Request) {
			requestID := middleware.MustRequestIDFromContext(r.Context())
			require.NotEmpty(t, requestID, "Request ID should be available in context")
			w.Header().Set("X-Request-ID", requestID)
			w.WriteHeader(http.StatusOK)
		})
	})

	t.Run("request ID available in HTMX routes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/htmx/requestid-test", nil)
		req.Header.Set("HX-Request", "true")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		requestID := w.Header().Get("X-Request-ID")
		assert.NotEmpty(t, requestID, "Request ID should be in response headers")
		assert.Greater(t, len(requestID), 10, "Request ID should be a valid UUID-like string")
	})
}

// TestHTMXRouteGroup_MultipleEndpoints tests multiple HTMX endpoints in the route group
func TestHTMXRouteGroup_MultipleEndpoints(t *testing.T) {
	logger := zerolog.Nop()

	cfg := RouterConfig{
		Logger:        &logger,
		Environment:   "test",
		EnableCORS:    false,
		CompressLevel: 0,
	}

	router := NewRouter(cfg)

	// Register multiple HTMX endpoints
	router.Route("/api/htmx", func(r chi.Router) {
		r.Get("/endpoint1", func(w http.ResponseWriter, r *http.Request) {
			if middleware.IsHTMXRequest(r.Context()) {
				w.Write([]byte("endpoint1-partial"))
			} else {
				w.Write([]byte("endpoint1-full"))
			}
		})

		r.Get("/endpoint2", func(w http.ResponseWriter, r *http.Request) {
			if middleware.IsHTMXRequest(r.Context()) {
				w.Write([]byte("endpoint2-partial"))
			} else {
				w.Write([]byte("endpoint2-full"))
			}
		})

		r.Post("/endpoint3", func(w http.ResponseWriter, r *http.Request) {
			if middleware.IsHTMXRequest(r.Context()) {
				w.Write([]byte("endpoint3-partial"))
			} else {
				w.Write([]byte("endpoint3-full"))
			}
		})
	})

	tests := []struct {
		name         string
		method       string
		path         string
		htmxRequest  bool
		expectedBody string
	}{
		{
			name:         "endpoint1 full page",
			method:       http.MethodGet,
			path:         "/api/htmx/endpoint1",
			htmxRequest:  false,
			expectedBody: "endpoint1-full",
		},
		{
			name:         "endpoint1 HTMX partial",
			method:       http.MethodGet,
			path:         "/api/htmx/endpoint1",
			htmxRequest:  true,
			expectedBody: "endpoint1-partial",
		},
		{
			name:         "endpoint2 full page",
			method:       http.MethodGet,
			path:         "/api/htmx/endpoint2",
			htmxRequest:  false,
			expectedBody: "endpoint2-full",
		},
		{
			name:         "endpoint2 HTMX partial",
			method:       http.MethodGet,
			path:         "/api/htmx/endpoint2",
			htmxRequest:  true,
			expectedBody: "endpoint2-partial",
		},
		{
			name:         "endpoint3 POST HTMX",
			method:       http.MethodPost,
			path:         "/api/htmx/endpoint3",
			htmxRequest:  true,
			expectedBody: "endpoint3-partial",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.htmxRequest {
				req.Header.Set("HX-Request", "true")
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, tt.expectedBody, strings.TrimSpace(w.Body.String()))
		})
	}
}
