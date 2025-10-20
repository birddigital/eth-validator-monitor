package middleware

import (
	"context"
	"net/http"
	"strings"
)

// contextKey is a private type for context keys to prevent collisions
type htmxContextKey int

const (
	htmxRequestKey htmxContextKey = iota
	htmxTriggerKey
	htmxTargetKey
	htmxPromptKey
)

// HTMXMiddleware adds HTMX-specific context information to requests
type HTMXMiddleware struct {
	// Future config options can be added here if needed
}

// NewHTMXMiddleware creates a new HTMX middleware instance
func NewHTMXMiddleware() *HTMXMiddleware {
	return &HTMXMiddleware{}
}

// Middleware detects HTMX requests and adds context information
func (m *HTMXMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Detect HTMX request by checking HX-Request header
		isHTMXRequest := r.Header.Get("HX-Request") == "true"
		ctx = context.WithValue(ctx, htmxRequestKey, isHTMXRequest)

		// Store additional HTMX headers in context if present
		if isHTMXRequest {
			if trigger := r.Header.Get("HX-Trigger"); trigger != "" {
				ctx = context.WithValue(ctx, htmxTriggerKey, trigger)
			}
			if target := r.Header.Get("HX-Target"); target != "" {
				ctx = context.WithValue(ctx, htmxTargetKey, target)
			}
			if prompt := r.Header.Get("HX-Prompt"); prompt != "" {
				ctx = context.WithValue(ctx, htmxPromptKey, prompt)
			}
		}

		// Continue with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// IsHTMXRequest checks if the current request is an HTMX request
func IsHTMXRequest(ctx context.Context) bool {
	val, ok := ctx.Value(htmxRequestKey).(bool)
	if !ok {
		return false
	}
	return val
}

// HTMXTrigger returns the HX-Trigger header value if present
func HTMXTrigger(ctx context.Context) (string, bool) {
	val, ok := ctx.Value(htmxTriggerKey).(string)
	return val, ok
}

// HTMXTarget returns the HX-Target header value if present
func HTMXTarget(ctx context.Context) (string, bool) {
	val, ok := ctx.Value(htmxTargetKey).(string)
	return val, ok
}

// HTMXPrompt returns the HX-Prompt header value if present
func HTMXPrompt(ctx context.Context) (string, bool) {
	val, ok := ctx.Value(htmxPromptKey).(string)
	return val, ok
}

// WantsJSON checks if the request prefers JSON responses based on Accept header
func WantsJSON(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	// Check if Accept header contains application/json
	// and doesn't prefer text/html
	return strings.Contains(accept, "application/json") &&
		!strings.Contains(accept, "text/html")
}

// SetHTMXResponse sets HTMX-specific response headers
func SetHTMXResponse(w http.ResponseWriter, trigger string, retarget string, reswap string) {
	if trigger != "" {
		w.Header().Set("HX-Trigger", trigger)
	}
	if retarget != "" {
		w.Header().Set("HX-Retarget", retarget)
	}
	if reswap != "" {
		w.Header().Set("HX-Reswap", reswap)
	}
}

// SetHTMXRedirect tells HTMX to perform a client-side redirect
func SetHTMXRedirect(w http.ResponseWriter, url string) {
	w.Header().Set("HX-Redirect", url)
}

// SetHTMXRefresh tells HTMX to refresh the page
func SetHTMXRefresh(w http.ResponseWriter) {
	w.Header().Set("HX-Refresh", "true")
}
