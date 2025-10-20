package middleware

import (
	"net/http"
	"strings"
)

// CORSConfig holds CORS middleware configuration
type CORSConfig struct {
	Enabled        bool
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
	MaxAge         int
}

// CORSMiddleware handles Cross-Origin Resource Sharing
type CORSMiddleware struct {
	config CORSConfig
}

// NewCORSMiddleware creates a new CORS middleware with configuration
func NewCORSMiddleware(config CORSConfig) *CORSMiddleware {
	// Set defaults if not provided
	if len(config.AllowedMethods) == 0 {
		config.AllowedMethods = []string{"GET", "POST", "OPTIONS"}
	}
	if len(config.AllowedHeaders) == 0 {
		config.AllowedHeaders = []string{
			"Content-Type",
			"Authorization",
			"X-API-Key",
			"X-Requested-With",
		}
	}
	if config.MaxAge == 0 {
		config.MaxAge = 300 // 5 minutes default
	}

	return &CORSMiddleware{
		config: config,
	}
}

// Middleware returns the HTTP middleware function
func (c *CORSMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If CORS is disabled, pass through
		if !c.config.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		origin := r.Header.Get("Origin")

		// Check if origin is allowed
		allowed := false
		allowedOrigin := ""

		for _, ao := range c.config.AllowedOrigins {
			if ao == "*" {
				allowed = true
				allowedOrigin = "*"
				break
			}
			if ao == origin {
				allowed = true
				allowedOrigin = origin
				break
			}
		}

		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Methods", strings.Join(c.config.AllowedMethods, ", "))
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(c.config.AllowedHeaders, ", "))
			w.Header().Set("Access-Control-Max-Age", intToString(c.config.MaxAge))
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			if allowed {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusForbidden)
			}
			return
		}

		next.ServeHTTP(w, r)
	})
}

// intToString converts int to string
func intToString(n int) string {
	if n == 0 {
		return "0"
	}
	negative := n < 0
	if negative {
		n = -n
	}
	result := ""
	for n > 0 {
		digit := n % 10
		result = string(rune('0'+digit)) + result
		n /= 10
	}
	if negative {
		result = "-" + result
	}
	return result
}
