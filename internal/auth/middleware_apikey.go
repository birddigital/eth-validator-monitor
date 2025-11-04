package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/birddigital/eth-validator-monitor/internal/storage"
	"github.com/google/uuid"
)

// API Key specific context keys
const (
	// APIKeyUserIDKey is the context key for API key-based user ID
	APIKeyUserIDKey contextKey = "apikey_user_id"
)

// APIKeyMiddleware adds API key authentication context to requests
// Checks for API key in X-API-Key header or Authorization: Bearer header
func APIKeyMiddleware(apiKeyRepo *storage.APIKeyRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try to extract API key from headers
			apiKey := extractAPIKey(r)
			if apiKey == "" {
				// No API key provided - continue without API key auth
				next.ServeHTTP(w, r)
				return
			}

			// Validate the API key
			userID, err := apiKeyRepo.ValidateAPIKey(r.Context(), apiKey)
			if err != nil {
				// Invalid/revoked/expired API key - return 401
				http.Error(w, "Invalid or expired API key", http.StatusUnauthorized)
				return
			}

			// Add user ID to context
			ctx := context.WithValue(r.Context(), APIKeyUserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractAPIKey extracts the API key from request headers
// Supports both X-API-Key and Authorization: Bearer headers
func extractAPIKey(r *http.Request) string {
	// Try X-API-Key header first (recommended for API keys)
	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
		return apiKey
	}

	// Try Authorization: Bearer header
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return parts[1]
		}
	}

	return ""
}

// RequireAPIKeyAuth middleware ensures user is authenticated via API key
func RequireAPIKeyAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := GetAPIKeyUserIDFromContext(r.Context())
		if !ok || userID == uuid.Nil {
			http.Error(w, "Unauthorized: valid API key required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// GetAPIKeyUserIDFromContext retrieves the authenticated user ID from API key context
func GetAPIKeyUserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(APIKeyUserIDKey).(uuid.UUID)
	return userID, ok
}

// RequireAnyAuth middleware ensures user is authenticated via either session, JWT, or API key
func RequireAnyAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if authenticated via any method
		userID, ok := GetUserIDFromContext(r.Context())
		if !ok || userID == uuid.Nil {
			http.Error(w, "Unauthorized: authentication required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
