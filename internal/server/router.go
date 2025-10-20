package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"

	appmiddleware "github.com/birddigital/eth-validator-monitor/graph/middleware"
)

// RouterConfig holds configuration for the HTTP router
type RouterConfig struct {
	Logger         *zerolog.Logger
	Environment    string // "development" or "production"
	EnableCORS     bool
	AllowedOrigins []string
	CompressLevel  int // 0-9, 0 = no compression
	RateLimitRPS   int // Requests per second for rate limiting
	RateLimitBurst int // Burst size for rate limiter
}

// NewRouter creates a new Chi router with core middleware
func NewRouter(cfg RouterConfig) *chi.Mux {
	r := chi.NewRouter()

	// Set middleware stack
	r.Use(setupMiddleware(cfg)...)

	return r
}

// setupMiddleware configures the middleware chain
func setupMiddleware(cfg RouterConfig) []func(http.Handler) http.Handler {
	var middlewares []func(http.Handler) http.Handler

	// 1. Request ID (MUST be first to ensure all logs have request ID)
	if cfg.Logger != nil {
		requestIDMiddleware := appmiddleware.NewRequestIDMiddleware(*cfg.Logger)
		middlewares = append(middlewares, requestIDMiddleware.Middleware)
	}

	// 2. Real IP extraction (for accurate logging behind proxies)
	middlewares = append(middlewares, middleware.RealIP)

	// 3. Structured logging with zerolog (using existing logging middleware)
	if cfg.Logger != nil {
		loggingMiddleware := appmiddleware.NewLoggingMiddleware(func(format string, args ...interface{}) {
			cfg.Logger.Info().Msgf(format, args...)
		})
		middlewares = append(middlewares, loggingMiddleware.Middleware)
	}

	// 4. Recovery from panics (log and return 500)
	middlewares = append(middlewares, panicRecoveryMiddleware(cfg.Logger))

	// 5. Compression (gzip)
	if cfg.CompressLevel > 0 {
		middlewares = append(middlewares, middleware.Compress(cfg.CompressLevel))
	}

	// 6. Request timeout (prevent long-running requests)
	middlewares = append(middlewares, middleware.Timeout(60*time.Second))

	// 7. CORS (if enabled) - use existing CORS middleware
	if cfg.EnableCORS {
		corsMiddleware := appmiddleware.NewCORSMiddleware(appmiddleware.CORSConfig{
			Enabled:        true,
			AllowedOrigins: cfg.AllowedOrigins,
			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-API-Key"},
		})
		middlewares = append(middlewares, corsMiddleware.Middleware)
	}

	// 8. Rate limiting (if configured) - use existing rate limit middleware
	if cfg.RateLimitRPS > 0 {
		rateLimitMiddleware := appmiddleware.NewRateLimiter(appmiddleware.RateLimiterConfig{
			Enabled:        true,
			RequestsPerSec: float64(cfg.RateLimitRPS),
			Burst:          cfg.RateLimitBurst,
		})
		middlewares = append(middlewares, rateLimitMiddleware.Middleware)
	}

	// 9. Security headers - use existing security middleware
	middlewares = append(middlewares, appmiddleware.SecureHeaders)

	// 10. HTMX detection and context (must be after security headers)
	htmxMiddleware := appmiddleware.NewHTMXMiddleware()
	middlewares = append(middlewares, htmxMiddleware.Middleware)

	return middlewares
}

// panicRecoveryMiddleware handles panics and logs them with zerolog
func panicRecoveryMiddleware(logger *zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Log the panic with full context
					if logger != nil {
						reqID := appmiddleware.MustRequestIDFromContext(r.Context())
						logger.Error().
							Interface("panic", err).
							Str("request_id", reqID).
							Str("method", r.Method).
							Str("path", r.URL.Path).
							Str("remote_addr", r.RemoteAddr).
							Msg("panic_recovered")
					}

					// Return 500 error
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
