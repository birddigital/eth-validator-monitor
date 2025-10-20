package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiter_Middleware(t *testing.T) {
	tests := []struct {
		name           string
		config         RateLimiterConfig
		requests       int
		requestDelay   time.Duration
		expectBlocked  int
	}{
		{
			name: "allows requests within limit",
			config: RateLimiterConfig{
				Enabled:        true,
				RequestsPerSec: 10,
				Burst:          5,
			},
			requests:      3,
			requestDelay:  0,
			expectBlocked: 0,
		},
		{
			name: "blocks requests exceeding burst",
			config: RateLimiterConfig{
				Enabled:        true,
				RequestsPerSec: 1,
				Burst:          3,
			},
			requests:      10,
			requestDelay:  0,
			expectBlocked: 7, // 3 burst allowed, 7 blocked
		},
		{
			name: "disabled rate limiting allows all",
			config: RateLimiterConfig{
				Enabled:        false,
				RequestsPerSec: 1,
				Burst:          1,
			},
			requests:      100,
			requestDelay:  0,
			expectBlocked: 0,
		},
		{
			name: "rate limiting resets over time",
			config: RateLimiterConfig{
				Enabled:        true,
				RequestsPerSec: 5,
				Burst:          2,
			},
			requests:      4,
			requestDelay:  250 * time.Millisecond, // 4 req/s
			expectBlocked: 0,                       // Should all succeed due to refill
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewRateLimiter(tt.config).Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			blocked := 0
			for i := 0; i < tt.requests; i++ {
				req := httptest.NewRequest("POST", "/graphql", nil)
				req.RemoteAddr = "192.0.2.1:1234" // Same IP for all requests
				w := httptest.NewRecorder()

				handler.ServeHTTP(w, req)

				if w.Code == http.StatusTooManyRequests {
					blocked++
				}

				if tt.requestDelay > 0 && i < tt.requests-1 {
					time.Sleep(tt.requestDelay)
				}
			}

			if blocked != tt.expectBlocked {
				t.Errorf("expected %d blocked requests, got %d", tt.expectBlocked, blocked)
			}
		})
	}
}

func TestRateLimiter_DifferentIPs(t *testing.T) {
	config := RateLimiterConfig{
		Enabled:        true,
		RequestsPerSec: 2,
		Burst:          2,
	}

	handler := NewRateLimiter(config).Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test that different IPs have separate rate limits
	ips := []string{"192.0.2.1:1234", "192.0.2.2:1234", "192.0.2.3:1234"}

	for _, ip := range ips {
		// Each IP should be able to make burst number of requests
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest("POST", "/graphql", nil)
			req.RemoteAddr = ip
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("IP %s request %d: expected status 200, got %d", ip, i+1, w.Code)
			}
		}
	}
}

func TestRateLimiter_XForwardedFor(t *testing.T) {
	config := RateLimiterConfig{
		Enabled:        true,
		RequestsPerSec: 1,
		Burst:          2,
	}

	handler := NewRateLimiter(config).Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test X-Forwarded-For header is used for rate limiting
	req1 := httptest.NewRequest("POST", "/graphql", nil)
	req1.RemoteAddr = "192.0.2.100:1234"
	req1.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)

	req2 := httptest.NewRequest("POST", "/graphql", nil)
	req2.RemoteAddr = "192.0.2.200:1234" // Different RemoteAddr
	req2.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.3") // Same X-Forwarded-For first IP
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	req3 := httptest.NewRequest("POST", "/graphql", nil)
	req3.RemoteAddr = "192.0.2.300:1234"
	req3.Header.Set("X-Forwarded-For", "10.0.0.1") // Same IP, should be blocked
	w3 := httptest.NewRecorder()
	handler.ServeHTTP(w3, req3)

	// First two should succeed (burst = 2), third should be blocked
	if w1.Code != http.StatusOK {
		t.Errorf("First request: expected status 200, got %d", w1.Code)
	}
	if w2.Code != http.StatusOK {
		t.Errorf("Second request: expected status 200, got %d", w2.Code)
	}
	if w3.Code != http.StatusTooManyRequests {
		t.Errorf("Third request: expected status 429, got %d", w3.Code)
	}
}

func TestGetIPFromRequest(t *testing.T) {
	tests := []struct {
		name           string
		remoteAddr     string
		xForwardedFor  string
		xRealIP        string
		expectedIP     string
	}{
		{
			name:       "uses X-Forwarded-For first IP",
			remoteAddr: "192.0.2.1:1234",
			xForwardedFor: "10.0.0.1, 10.0.0.2, 10.0.0.3",
			expectedIP: "10.0.0.1",
		},
		{
			name:       "uses X-Real-IP if no X-Forwarded-For",
			remoteAddr: "192.0.2.1:1234",
			xRealIP:    "10.0.0.5",
			expectedIP: "10.0.0.5",
		},
		{
			name:       "uses RemoteAddr if no headers",
			remoteAddr: "192.0.2.1:1234",
			expectedIP: "192.0.2.1",
		},
		{
			name:       "strips port from RemoteAddr",
			remoteAddr: "2001:0db8:85a3:0000:0000:8a2e:0370:7334:8080",
			expectedIP: "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}

			ip := getIPFromRequest(req)
			if ip != tt.expectedIP {
				t.Errorf("expected IP %q, got %q", tt.expectedIP, ip)
			}
		})
	}
}

func BenchmarkRateLimiter_Allowed(b *testing.B) {
	config := RateLimiterConfig{
		Enabled:        true,
		RequestsPerSec: 1000,
		Burst:          1000,
	}

	handler := NewRateLimiter(config).Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/graphql", nil)
	req.RemoteAddr = "192.0.2.1:1234"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

func BenchmarkRateLimiter_Disabled(b *testing.B) {
	config := RateLimiterConfig{
		Enabled:        false,
		RequestsPerSec: 10,
		Burst:          10,
	}

	handler := NewRateLimiter(config).Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/graphql", nil)
	req.RemoteAddr = "192.0.2.1:1234"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}
