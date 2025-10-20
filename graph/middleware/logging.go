package middleware

import (
	"fmt"
	"net/http"
	"time"
)

// LoggingMiddleware logs HTTP requests
type LoggingMiddleware struct {
	logger func(format string, args ...interface{})
}

// NewLoggingMiddleware creates a new logging middleware
func NewLoggingMiddleware(logger func(format string, args ...interface{})) *LoggingMiddleware {
	if logger == nil {
		logger = func(format string, args ...interface{}) {
			fmt.Printf(format+"\n", args...)
		}
	}
	return &LoggingMiddleware{
		logger: logger,
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.written += n
	return n, err
}

// Middleware returns the HTTP middleware function
func (l *LoggingMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status
		wrapped := newResponseWriter(w)

		// Get client identifier
		identifier := r.RemoteAddr
		// TODO: Implement GetAPIKey and GetUser context helpers
		// if apiKey, err := GetAPIKey(r.Context()); err == nil {
		// 	identifier = fmt.Sprintf("api_key:%s", apiKey.Name)
		// } else if user, err := GetUser(r.Context()); err == nil {
		// 	identifier = fmt.Sprintf("user:%s", user.Username)
		// }

		// Call next handler
		next.ServeHTTP(wrapped, r)

		// Log request
		duration := time.Since(start)
		l.logger("[GraphQL] %s %s - %d - %s - %d bytes - %v",
			r.Method,
			r.URL.Path,
			wrapped.statusCode,
			identifier,
			wrapped.written,
			duration,
		)
	})
}
