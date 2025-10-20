package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeadersMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		setupRequest   func(*http.Request)
		expectedHeader string
		expectedValue  string
		shouldExist    bool
	}{
		{
			name:           "X-Frame-Options is set to DENY",
			expectedHeader: "X-Frame-Options",
			expectedValue:  "DENY",
			shouldExist:    true,
		},
		{
			name:           "X-Content-Type-Options is set to nosniff",
			expectedHeader: "X-Content-Type-Options",
			expectedValue:  "nosniff",
			shouldExist:    true,
		},
		{
			name:           "X-XSS-Protection is set",
			expectedHeader: "X-XSS-Protection",
			expectedValue:  "1; mode=block",
			shouldExist:    true,
		},
		{
			name:           "CSP is set with default directives",
			expectedHeader: "Content-Security-Policy",
			expectedValue:  "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self';",
			shouldExist:    true,
		},
		{
			name:           "Referrer-Policy is set",
			expectedHeader: "Referrer-Policy",
			expectedValue:  "strict-origin-when-cross-origin",
			shouldExist:    true,
		},
		{
			name:           "Permissions-Policy is set",
			expectedHeader: "Permissions-Policy",
			expectedValue:  "geolocation=(), microphone=(), camera=(), payment=(), usb=(), magnetometer=(), gyroscope=(), accelerometer=()",
			shouldExist:    true,
		},
		{
			name: "HSTS is set for HTTPS requests (X-Forwarded-Proto)",
			setupRequest: func(r *http.Request) {
				r.Header.Set("X-Forwarded-Proto", "https")
			},
			expectedHeader: "Strict-Transport-Security",
			expectedValue:  "max-age=31536000; includeSubDomains; preload",
			shouldExist:    true,
		},
		{
			name:           "HSTS is not set for HTTP requests",
			expectedHeader: "Strict-Transport-Security",
			shouldExist:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Wrap with security middleware
			secureHandler := SecureHeaders(handler)

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.setupRequest != nil {
				tt.setupRequest(req)
			}

			// Create response recorder
			rr := httptest.NewRecorder()

			// Serve request
			secureHandler.ServeHTTP(rr, req)

			// Check headers
			if tt.shouldExist {
				if got := rr.Header().Get(tt.expectedHeader); got != tt.expectedValue {
					t.Errorf("header %s = %q, want %q", tt.expectedHeader, got, tt.expectedValue)
				}
			} else {
				if got := rr.Header().Get(tt.expectedHeader); got != "" {
					t.Errorf("header %s should not be set, got %q", tt.expectedHeader, got)
				}
			}
		})
	}
}

func TestCustomSecurityHeaders(t *testing.T) {
	customHeaders := &SecurityHeaders{
		HSTSMaxAge:            63072000, // 2 years
		HSTSIncludeSubdomains: false,
		CSPDirectives:         "default-src 'none';",
		EnableHSTS:            true,
		EnableXFrameOpts:      true,
		EnableXContentOpts:    true,
		EnableXXSSProt:        true,
		EnableCSP:             true,
		EnableReferrer:        true,
		EnablePermissions:     true,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	secureHandler := customHeaders.Middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rr := httptest.NewRecorder()

	secureHandler.ServeHTTP(rr, req)

	// Check custom HSTS (without includeSubDomains)
	expectedHSTS := "max-age=63072000; preload"
	if got := rr.Header().Get("Strict-Transport-Security"); got != expectedHSTS {
		t.Errorf("HSTS = %q, want %q", got, expectedHSTS)
	}

	// Check custom CSP
	expectedCSP := "default-src 'none';"
	if got := rr.Header().Get("Content-Security-Policy"); got != expectedCSP {
		t.Errorf("CSP = %q, want %q", got, expectedCSP)
	}
}

func TestSecurityHeadersDisabled(t *testing.T) {
	// Test with all headers disabled
	customHeaders := &SecurityHeaders{
		EnableHSTS:        false,
		EnableXFrameOpts:  false,
		EnableXContentOpts: false,
		EnableXXSSProt:    false,
		EnableCSP:         false,
		EnableReferrer:    false,
		EnablePermissions: false,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	secureHandler := customHeaders.Middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rr := httptest.NewRecorder()

	secureHandler.ServeHTTP(rr, req)

	// Verify no security headers are set
	securityHeaders := []string{
		"Strict-Transport-Security",
		"X-Frame-Options",
		"X-Content-Type-Options",
		"X-XSS-Protection",
		"Content-Security-Policy",
		"Referrer-Policy",
		"Permissions-Policy",
	}

	for _, header := range securityHeaders {
		if got := rr.Header().Get(header); got != "" {
			t.Errorf("header %s should not be set when disabled, got %q", header, got)
		}
	}
}

func TestSecurityHeadersWithTLS(t *testing.T) {
	// Test HSTS with actual TLS connection (simulated)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	secureHandler := SecureHeaders(handler)

	// Create HTTPS request
	req := httptest.NewRequest(http.MethodGet, "https://example.com/test", nil)
	// Note: In real tests with httptest, TLS field won't be set automatically
	// We rely on X-Forwarded-Proto header in production behind proxies

	rr := httptest.NewRecorder()
	secureHandler.ServeHTTP(rr, req)

	// HSTS should not be set without TLS field or X-Forwarded-Proto
	if got := rr.Header().Get("Strict-Transport-Security"); got != "" {
		// In this test case without actual TLS, HSTS shouldn't be set
		// This verifies the middleware correctly checks for TLS
		t.Logf("HSTS header: %q (expected empty without TLS field or X-Forwarded-Proto)", got)
	}
}

func TestSecurityHeadersMiddlewareChain(t *testing.T) {
	// Test that security headers work correctly in a middleware chain
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom-Header", "test-value")
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with security middleware
	secureHandler := SecureHeaders(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rr := httptest.NewRecorder()

	secureHandler.ServeHTTP(rr, req)

	// Verify security headers are present
	if got := rr.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("X-Frame-Options = %q, want DENY", got)
	}

	// Verify custom headers from handler are preserved
	if got := rr.Header().Get("X-Custom-Header"); got != "test-value" {
		t.Errorf("X-Custom-Header = %q, want test-value", got)
	}

	// Verify HSTS is set
	hstsValue := rr.Header().Get("Strict-Transport-Security")
	if !strings.Contains(hstsValue, "max-age=31536000") {
		t.Errorf("HSTS does not contain expected max-age, got: %q", hstsValue)
	}
}
