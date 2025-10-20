package web

import (
	"net/http"
	"strconv"
)

// CacheControl wraps an http.Handler to add Cache-Control headers.
// This middleware adds public cache headers with configurable max-age to improve
// browser caching for static assets.
//
// maxAge is specified in seconds (e.g., 31536000 = 1 year).
func CacheControl(maxAge int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "public, max-age="+strconv.Itoa(maxAge))
			next.ServeHTTP(w, r)
		})
	}
}
