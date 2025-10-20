package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// contextKey is an unexported type for context keys to prevent collisions
type contextKey int

const (
	requestIDKey contextKey = iota
	requestLoggerKey
)

// RequestIDMiddleware generates unique request IDs and embeds them into context
type RequestIDMiddleware struct {
	logger zerolog.Logger
}

// NewRequestIDMiddleware creates a new request ID middleware instance
func NewRequestIDMiddleware(logger zerolog.Logger) *RequestIDMiddleware {
	return &RequestIDMiddleware{
		logger: logger,
	}
}

// Middleware implements the HTTP middleware interface
func (m *RequestIDMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Generate UUID for this request
		requestID := uuid.New().String()

		// Create request-scoped logger with request ID
		requestLogger := m.logger.With().
			Str("request_id", requestID).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Logger()

		// Embed both request ID and logger into context
		ctx := r.Context()
		ctx = WithRequestID(ctx, requestID)
		ctx = WithLogger(ctx, requestLogger)

		// Add request ID to response header for client correlation
		w.Header().Set("X-Request-ID", requestID)

		// Log incoming request
		requestLogger.Debug().Msg("incoming request")

		// Pass modified request to next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// WithRequestID adds a request ID to the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// RequestIDFromContext retrieves the request ID from the context
func RequestIDFromContext(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(requestIDKey).(string)
	return requestID, ok
}

// MustRequestIDFromContext retrieves the request ID or returns empty string
func MustRequestIDFromContext(ctx context.Context) string {
	requestID, _ := RequestIDFromContext(ctx)
	return requestID
}

// WithLogger adds a logger to the context
func WithLogger(ctx context.Context, logger zerolog.Logger) context.Context {
	return context.WithValue(ctx, requestLoggerKey, &logger)
}

// LoggerFromContext retrieves the request-scoped logger from context
func LoggerFromContext(ctx context.Context) (zerolog.Logger, bool) {
	logger, ok := ctx.Value(requestLoggerKey).(*zerolog.Logger)
	if !ok || logger == nil {
		return zerolog.Logger{}, false
	}
	return *logger, true
}

// MustLoggerFromContext retrieves logger or returns a disabled logger
func MustLoggerFromContext(ctx context.Context) zerolog.Logger {
	if logger, ok := LoggerFromContext(ctx); ok {
		return logger
	}
	return zerolog.Nop()
}
