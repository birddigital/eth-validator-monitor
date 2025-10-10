package middleware

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter manages rate limiting for API endpoints
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
}

// NewRateLimiter creates a new rate limiter
// ratePerSecond: number of requests allowed per second
// burst: maximum burst size
func NewRateLimiter(ratePerSecond float64, burst int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rate.Limit(ratePerSecond),
		burst:    burst,
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
	// Start cleanup goroutine
	go rl.cleanupLimiters()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Use API key as identifier if available, otherwise use IP
		identifier := r.RemoteAddr
		if apiKey, err := GetAPIKey(r.Context()); err == nil {
			identifier = apiKey.Name
		}

		limiter := rl.getLimiter(identifier)
		if !limiter.Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Allow checks if a request should be allowed
func (rl *RateLimiter) Allow(ctx context.Context) error {
	// Default identifier
	identifier := "default"

	// Use API key as identifier if available
	if apiKey, err := GetAPIKey(ctx); err == nil {
		identifier = apiKey.Name
	}

	limiter := rl.getLimiter(identifier)
	if !limiter.Allow() {
		return fmt.Errorf("rate limit exceeded")
	}

	return nil
}
