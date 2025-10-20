package auth

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// Session-specific context keys (reusing the contextKey type from jwt.go)
const (
	// SessionUserIDKey is the context key for session-based user ID
	SessionUserIDKey contextKey = "session_user_id"
	// SessionUsernameKey is the context key for session-based username
	SessionUsernameKey contextKey = "session_username"
)

// SessionMiddleware adds session context to requests
func SessionMiddleware(sessionStore *SessionStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, err := sessionStore.Get(r)
			if err != nil {
				// Session error - continue without auth context
				next.ServeHTTP(w, r)
				return
			}

			// Extract user info from session
			userID, hasUserID := sessionStore.GetUserID(session)
			username, hasUsername := sessionStore.GetUsername(session)

			ctx := r.Context()
			if hasUserID {
				ctx = context.WithValue(ctx, SessionUserIDKey, userID)
			}
			if hasUsername {
				ctx = context.WithValue(ctx, SessionUsernameKey, username)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireSessionAuth middleware ensures user is authenticated via session
func RequireSessionAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := GetSessionUserIDFromContext(r.Context())
		if !ok || userID == uuid.Nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// GetSessionUserIDFromContext retrieves the authenticated user ID from session context
func GetSessionUserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(SessionUserIDKey).(uuid.UUID)
	return userID, ok
}

// GetSessionUsernameFromContext retrieves the authenticated username from session context
func GetSessionUsernameFromContext(ctx context.Context) (string, bool) {
	username, ok := ctx.Value(SessionUsernameKey).(string)
	return username, ok
}
