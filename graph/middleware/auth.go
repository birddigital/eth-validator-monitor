package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// AuthContextKey is the context key for auth information
type AuthContextKey string

const (
	// UserContextKey stores authenticated user info
	UserContextKey AuthContextKey = "user"
	// APIKeyContextKey stores API key info
	APIKeyContextKey AuthContextKey = "apiKey"
)

// UserInfo contains authenticated user information
type UserInfo struct {
	ID       string
	Username string
	Roles    []string
}

// APIKeyInfo contains API key information
type APIKeyInfo struct {
	Key    string
	Name   string
	Scopes []string
}

// AuthMiddleware handles authentication
type AuthMiddleware struct {
	apiKeys     map[string]*APIKeyInfo
	requireAuth bool
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(requireAuth bool) *AuthMiddleware {
	return &AuthMiddleware{
		apiKeys:     make(map[string]*APIKeyInfo),
		requireAuth: requireAuth,
	}
}

// RegisterAPIKey adds an API key to the whitelist
func (m *AuthMiddleware) RegisterAPIKey(key, name string, scopes []string) {
	m.apiKeys[key] = &APIKeyInfo{
		Key:    key,
		Name:   name,
		Scopes: scopes,
	}
}

// Middleware returns the HTTP middleware function
func (m *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Check for API key in header
		apiKey := r.Header.Get("X-API-Key")
		if apiKey != "" {
			if keyInfo, ok := m.apiKeys[apiKey]; ok {
				ctx = context.WithValue(ctx, APIKeyContextKey, keyInfo)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			// Invalid API key
			if m.requireAuth {
				http.Error(w, "Invalid API key", http.StatusUnauthorized)
				return
			}
		}

		// Check for Bearer token in Authorization header
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			// TODO: Validate JWT token and extract user info
			// For now, just pass through
			_ = token
		}

		// If auth is required and no valid credentials provided
		if m.requireAuth && ctx.Value(APIKeyContextKey) == nil && ctx.Value(UserContextKey) == nil {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetAPIKey retrieves API key info from context
func GetAPIKey(ctx context.Context) (*APIKeyInfo, error) {
	val := ctx.Value(APIKeyContextKey)
	if val == nil {
		return nil, fmt.Errorf("no API key in context")
	}
	keyInfo, ok := val.(*APIKeyInfo)
	if !ok {
		return nil, fmt.Errorf("invalid API key in context")
	}
	return keyInfo, nil
}

// GetUser retrieves user info from context
func GetUser(ctx context.Context) (*UserInfo, error) {
	val := ctx.Value(UserContextKey)
	if val == nil {
		return nil, fmt.Errorf("no user in context")
	}
	userInfo, ok := val.(*UserInfo)
	if !ok {
		return nil, fmt.Errorf("invalid user in context")
	}
	return userInfo, nil
}

// HasScope checks if API key has required scope
func HasScope(ctx context.Context, scope string) bool {
	apiKey, err := GetAPIKey(ctx)
	if err != nil {
		return false
	}
	for _, s := range apiKey.Scopes {
		if s == scope || s == "*" {
			return true
		}
	}
	return false
}

// RequireScope is a directive function to check scopes
func RequireScope(ctx context.Context, scope string) error {
	if !HasScope(ctx, scope) {
		return fmt.Errorf("insufficient permissions: %s scope required", scope)
	}
	return nil
}
