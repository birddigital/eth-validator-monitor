package middleware

import (
	"net/http"
	"strings"

	"github.com/birddigital/eth-validator-monitor/internal/auth"
	"github.com/rs/zerolog"
)

// AuthMiddleware extracts and validates JWT tokens from requests
type AuthMiddleware struct {
	jwtService *auth.JWTService
	logger     *zerolog.Logger
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(jwtService *auth.JWTService, log *zerolog.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		jwtService: jwtService,
		logger:     log,
	}
}

// Middleware is the HTTP middleware function
func (m *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// No auth header - continue without authentication
			next.ServeHTTP(w, r)
			return
		}

		// Expected format: "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			m.logger.Warn().Str("header", authHeader).Msg("malformed authorization header")
			next.ServeHTTP(w, r)
			return
		}

		tokenString := parts[1]

		// Validate token
		claims, err := m.jwtService.ValidateToken(tokenString)
		if err != nil {
			m.logger.Debug().Err(err).Msg("invalid token")
			// Token is invalid - continue without authentication
			// Resolvers will handle authorization checks
			next.ServeHTTP(w, r)
			return
		}

		// Add claims to request context
		ctx := auth.WithUserClaims(r.Context(), claims)

		// Continue with authenticated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
