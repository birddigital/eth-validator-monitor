package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSMiddleware_Enabled(t *testing.T) {
	tests := []struct {
		name           string
		config         CORSConfig
		origin         string
		method         string
		expectAllowed  bool
		expectStatus   int
	}{
		{
			name: "allows whitelisted origin",
			config: CORSConfig{
				Enabled:        true,
				AllowedOrigins: []string{"http://localhost:3000"},
				AllowedMethods: []string{"GET", "POST"},
			},
			origin:        "http://localhost:3000",
			method:        "POST",
			expectAllowed: true,
			expectStatus:  http.StatusOK,
		},
		{
			name: "blocks non-whitelisted origin",
			config: CORSConfig{
				Enabled:        true,
				AllowedOrigins: []string{"http://localhost:3000"},
				AllowedMethods: []string{"GET", "POST"},
			},
			origin:        "http://evil.com",
			method:        "POST",
			expectAllowed: false,
			expectStatus:  http.StatusOK, // Passes through but no CORS headers
		},
		{
			name: "allows wildcard origin",
			config: CORSConfig{
				Enabled:        true,
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET", "POST"},
			},
			origin:        "http://anywhere.com",
			method:        "POST",
			expectAllowed: true,
			expectStatus:  http.StatusOK,
		},
		{
			name: "allows multiple whitelisted origins",
			config: CORSConfig{
				Enabled: true,
				AllowedOrigins: []string{
					"http://localhost:3000",
					"http://localhost:5173",
					"https://app.example.com",
				},
				AllowedMethods: []string{"GET", "POST"},
			},
			origin:        "https://app.example.com",
			method:        "POST",
			expectAllowed: true,
			expectStatus:  http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewCORSMiddleware(tt.config).Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(tt.method, "/graphql", nil)
			req.Header.Set("Origin", tt.origin)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectStatus {
				t.Errorf("expected status %d, got %d", tt.expectStatus, w.Code)
			}

			allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
			hasHeader := allowOrigin != ""

			if tt.expectAllowed && !hasHeader {
				t.Errorf("expected CORS headers but none found")
			}
			if !tt.expectAllowed && hasHeader {
				t.Errorf("expected no CORS headers but found: %s", allowOrigin)
			}

			// Verify the correct origin is returned (not always *)
			if tt.expectAllowed && allowOrigin != "*" && allowOrigin != tt.origin {
				t.Errorf("expected origin %q in header, got %q", tt.origin, allowOrigin)
			}
		})
	}
}

func TestCORSMiddleware_Disabled(t *testing.T) {
	config := CORSConfig{
		Enabled:        false,
		AllowedOrigins: []string{"http://localhost:3000"},
	}

	handler := NewCORSMiddleware(config).Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/graphql", nil)
	req.Header.Set("Origin", "http://evil.com")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should pass through without CORS headers
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("expected no CORS headers when disabled")
	}
}

func TestCORSMiddleware_PreflightRequest(t *testing.T) {
	tests := []struct {
		name         string
		config       CORSConfig
		origin       string
		expectStatus int
		expectCORS   bool
	}{
		{
			name: "allows preflight for whitelisted origin",
			config: CORSConfig{
				Enabled:        true,
				AllowedOrigins: []string{"http://localhost:3000"},
				AllowedMethods: []string{"GET", "POST"},
			},
			origin:       "http://localhost:3000",
			expectStatus: http.StatusOK,
			expectCORS:   true,
		},
		{
			name: "blocks preflight for non-whitelisted origin",
			config: CORSConfig{
				Enabled:        true,
				AllowedOrigins: []string{"http://localhost:3000"},
				AllowedMethods: []string{"GET", "POST"},
			},
			origin:       "http://evil.com",
			expectStatus: http.StatusForbidden,
			expectCORS:   false,
		},
		{
			name: "disabled CORS passes through OPTIONS",
			config: CORSConfig{
				Enabled:        false,
				AllowedOrigins: []string{"http://localhost:3000"},
			},
			origin:       "http://evil.com",
			expectStatus: http.StatusOK,
			expectCORS:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewCORSMiddleware(tt.config).Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("OPTIONS", "/graphql", nil)
			req.Header.Set("Origin", tt.origin)
			req.Header.Set("Access-Control-Request-Method", "POST")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectStatus {
				t.Errorf("expected status %d, got %d", tt.expectStatus, w.Code)
			}

			hasCORS := w.Header().Get("Access-Control-Allow-Origin") != ""
			if hasCORS != tt.expectCORS {
				t.Errorf("expected CORS headers: %v, got: %v", tt.expectCORS, hasCORS)
			}
		})
	}
}

func TestCORSMiddleware_Headers(t *testing.T) {
	config := CORSConfig{
		Enabled:        true,
		AllowedOrigins: []string{"http://localhost:3000"},
		AllowedMethods: []string{"GET", "POST", "PUT"},
		AllowedHeaders: []string{"Content-Type", "Authorization", "X-Custom-Header"},
		MaxAge:         600,
	}

	handler := NewCORSMiddleware(config).Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/graphql", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Check all CORS headers are set
	if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Errorf("expected origin header to be set correctly")
	}

	methods := w.Header().Get("Access-Control-Allow-Methods")
	if methods != "GET, POST, PUT" {
		t.Errorf("expected methods 'GET, POST, PUT', got %q", methods)
	}

	headers := w.Header().Get("Access-Control-Allow-Headers")
	if headers != "Content-Type, Authorization, X-Custom-Header" {
		t.Errorf("expected headers to include custom header, got %q", headers)
	}

	maxAge := w.Header().Get("Access-Control-Max-Age")
	if maxAge != "600" {
		t.Errorf("expected max age '600', got %q", maxAge)
	}

	credentials := w.Header().Get("Access-Control-Allow-Credentials")
	if credentials != "true" {
		t.Errorf("expected credentials 'true', got %q", credentials)
	}
}

func TestCORSMiddleware_DefaultValues(t *testing.T) {
	config := CORSConfig{
		Enabled:        true,
		AllowedOrigins: []string{"http://localhost:3000"},
		// No methods, headers, or maxAge specified
	}

	middleware := NewCORSMiddleware(config)

	// Check defaults are set
	if len(middleware.config.AllowedMethods) == 0 {
		t.Error("expected default methods to be set")
	}
	if len(middleware.config.AllowedHeaders) == 0 {
		t.Error("expected default headers to be set")
	}
	if middleware.config.MaxAge == 0 {
		t.Error("expected default max age to be set")
	}
}

func TestIntToString(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{42, "42"},
		{300, "300"},
		{1000, "1000"},
		{-1, "-1"},
		{-42, "-42"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := intToString(tt.input)
			if result != tt.expected {
				t.Errorf("intToString(%d) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func BenchmarkCORSMiddleware_Allowed(b *testing.B) {
	config := CORSConfig{
		Enabled:        true,
		AllowedOrigins: []string{"http://localhost:3000"},
		AllowedMethods: []string{"GET", "POST"},
	}

	handler := NewCORSMiddleware(config).Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/graphql", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

func BenchmarkCORSMiddleware_Disabled(b *testing.B) {
	config := CORSConfig{
		Enabled:        false,
		AllowedOrigins: []string{"http://localhost:3000"},
	}

	handler := NewCORSMiddleware(config).Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/graphql", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}
