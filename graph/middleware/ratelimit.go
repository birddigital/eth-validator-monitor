package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiterConfig holds rate limiter configuration
type RateLimiterConfig struct {
	Enabled        bool
	RequestsPerSec float64
	Burst          int
}

// RateLimiter manages rate limiting for API endpoints
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
	enabled  bool
}

// NewRateLimiter creates a new rate limiter with configuration
func NewRateLimiter(config RateLimiterConfig) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rate.Limit(config.RequestsPerSec),
		burst:    config.Burst,
		enabled:  config.Enabled,
	}
}

// getLimiter returns a rate limiter for the given identifier
func (rl *RateLimiter) getLimiter(identifier string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[identifier]
	if !exists {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters[identifier] = limiter
	}

	return limiter
}

// cleanupLimiters removes inactive limiters
func (rl *RateLimiter) cleanupLimiters() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		for id, limiter := range rl.limiters {
			// Remove limiters that haven't been used in 5 minutes
			if limiter.Tokens() == float64(rl.burst) {
				delete(rl.limiters, id)
			}
		}
		rl.mu.Unlock()
	}
}

// Middleware returns the HTTP middleware function
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	// Start cleanup goroutine if enabled
	if rl.enabled {
		go rl.cleanupLimiters()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If rate limiting is disabled, pass through
		if !rl.enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Get IP address from request
		identifier := getIPFromRequest(r)

		limiter := rl.getLimiter(identifier)
		if !limiter.Allow() {
			http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Allow checks if a request should be allowed
func (rl *RateLimiter) Allow(ctx context.Context) error {
	if !rl.enabled {
		return nil
	}

	// Default identifier
	identifier := "default"

	limiter := rl.getLimiter(identifier)
	if !limiter.Allow() {
		return fmt.Errorf("rate limit exceeded")
	}

	return nil
}

// getIPFromRequest extracts the client IP address from the request
// It checks X-Forwarded-For, X-Real-IP headers first, then falls back to RemoteAddr
func getIPFromRequest(r *http.Request) string {
	// Check X-Forwarded-For header (may contain multiple IPs, first one is client)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr (remove port if present)
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}

	return ip
}
