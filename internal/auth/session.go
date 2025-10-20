package auth

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/rbcervilla/redisstore/v9"
	"github.com/redis/go-redis/v9"
)

const (
	// SessionName is the cookie name for user sessions
	SessionName = "eth-validator-session"
)

// SessionStore wraps Gorilla Sessions with Redis backend
type SessionStore struct {
	store sessions.Store
}

// NewSessionStore creates a Redis-backed session store with secure cookie settings
func NewSessionStore(redisClient *redis.Client, sessionSecret string, maxAge int, secure, httpOnly bool, sameSite string) (*SessionStore, error) {
	// Map SameSite string to http.SameSite constant
	var sameSiteMode http.SameSite
	switch sameSite {
	case "Strict":
		sameSiteMode = http.SameSiteStrictMode
	case "Lax":
		sameSiteMode = http.SameSiteLaxMode
	case "None":
		sameSiteMode = http.SameSiteNoneMode
	default:
		sameSiteMode = http.SameSiteLaxMode
	}

	// Create Redis session store
	store, err := redisstore.NewRedisStore(context.Background(), redisClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create redis session store: %w", err)
	}

	// Set session key prefix for Redis keys
	store.KeyPrefix("session:")

	// Configure cookie options for security
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: httpOnly,
		Secure:   secure,
		SameSite: sameSiteMode,
	})

	return &SessionStore{store: store}, nil
}

// Get retrieves a session for the given request
func (s *SessionStore) Get(r *http.Request) (*sessions.Session, error) {
	return s.store.Get(r, SessionName)
}

// Save persists the session
func (s *SessionStore) Save(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	return session.Save(r, w)
}

// SetUserSession stores user info in the session
func (s *SessionStore) SetUserSession(session *sessions.Session, userID uuid.UUID, username string) {
	session.Values["user_id"] = userID.String()
	session.Values["username"] = username
}

// GetUserID retrieves the user ID from session
func (s *SessionStore) GetUserID(session *sessions.Session) (uuid.UUID, bool) {
	val, ok := session.Values["user_id"]
	if !ok {
		return uuid.Nil, false
	}

	userIDStr, ok := val.(string)
	if !ok {
		return uuid.Nil, false
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, false
	}

	return userID, true
}

// GetUsername retrieves the username from session
func (s *SessionStore) GetUsername(session *sessions.Session) (string, bool) {
	val, ok := session.Values["username"]
	if !ok {
		return "", false
	}

	username, ok := val.(string)
	return username, ok
}

// Destroy clears the session (logout)
func (s *SessionStore) Destroy(session *sessions.Session) {
	session.Options.MaxAge = -1
	session.Values = make(map[interface{}]interface{})
}
