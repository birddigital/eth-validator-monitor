package auth

import (
	"context"

	"github.com/google/uuid"
)

// GetJWTUserIDFromContext retrieves user ID from JWT claims in context
func GetJWTUserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	claims, ok := GetUserClaims(ctx)
	if !ok || claims == nil {
		return uuid.Nil, false
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return uuid.Nil, false
	}
	return userID, true
}

// GetJWTUsernameFromContext retrieves username from JWT claims in context
func GetJWTUsernameFromContext(ctx context.Context) (string, bool) {
	claims, ok := GetUserClaims(ctx)
	if !ok || claims == nil {
		return "", false
	}
	return claims.Username, true
}

// GetJWTRolesFromContext retrieves roles from JWT claims in context
func GetJWTRolesFromContext(ctx context.Context) ([]string, bool) {
	claims, ok := GetUserClaims(ctx)
	if !ok || claims == nil {
		return nil, false
	}
	return claims.Roles, len(claims.Roles) > 0
}

// GetUserIDFromContext retrieves the authenticated user ID from context
// This works with session-based, JWT, and API key authentication
func GetUserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	// Try session-based authentication first
	if userID, ok := GetSessionUserIDFromContext(ctx); ok && userID != uuid.Nil {
		return userID, true
	}

	// Try JWT-based authentication
	if userID, ok := GetJWTUserIDFromContext(ctx); ok && userID != uuid.Nil {
		return userID, true
	}

	// Try API key-based authentication
	if userID, ok := GetAPIKeyUserIDFromContext(ctx); ok && userID != uuid.Nil {
		return userID, true
	}

	return uuid.Nil, false
}

// GetUsernameFromContext retrieves the authenticated username from context
// This works with both session-based and JWT authentication
func GetUsernameFromContext(ctx context.Context) (string, bool) {
	// Try session-based authentication first
	if username, ok := GetSessionUsernameFromContext(ctx); ok && username != "" {
		return username, true
	}

	// Try JWT-based authentication
	if username, ok := GetJWTUsernameFromContext(ctx); ok && username != "" {
		return username, true
	}

	return "", false
}

// GetRolesFromContext retrieves the authenticated user's roles from context
// This works with both session-based and JWT authentication
func GetRolesFromContext(ctx context.Context) ([]string, bool) {
	// Try JWT-based authentication first (has role info)
	if roles, ok := GetJWTRolesFromContext(ctx); ok && len(roles) > 0 {
		return roles, true
	}

	// Session-based auth doesn't store roles in context
	// Would need to query database if needed
	return nil, false
}
