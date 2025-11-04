package storage

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrAPIKeyNotFound      = errors.New("API key not found")
	ErrAPIKeyAlreadyExists = errors.New("API key already exists")
	ErrAPIKeyRevoked       = errors.New("API key has been revoked")
	ErrAPIKeyExpired       = errors.New("API key has expired")
)

// APIKey represents an API key for programmatic access
type APIKey struct {
	ID         int        `db:"id"`
	UserID     uuid.UUID  `db:"user_id"`
	KeyHash    string     `db:"key_hash"`
	KeyPrefix  string     `db:"key_prefix"`
	Name       string     `db:"name"`
	CreatedAt  time.Time  `db:"created_at"`
	LastUsedAt *time.Time `db:"last_used_at"`
	RevokedAt  *time.Time `db:"revoked_at"`
	ExpiresAt  *time.Time `db:"expires_at"`
}

// APIKeyRepository handles API key data persistence
type APIKeyRepository struct {
	pool *pgxpool.Pool
}

// NewAPIKeyRepository creates a new API key repository
func NewAPIKeyRepository(pool *pgxpool.Pool) *APIKeyRepository {
	return &APIKeyRepository{pool: pool}
}

// GenerateAPIKey generates a cryptographically secure API key
// Returns the plain-text key (to show user once) and the hashed version (to store)
func GenerateAPIKey() (plainKey, keyHash, keyPrefix string, err error) {
	// Generate 32 random bytes (256 bits)
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return "", "", "", fmt.Errorf("failed to generate random key: %w", err)
	}

	// Encode to base64 for human-readable format
	plainKey = base64.URLEncoding.EncodeToString(keyBytes)

	// Hash the key using SHA-256 for storage
	hash := sha256.Sum256([]byte(plainKey))
	keyHash = base64.StdEncoding.EncodeToString(hash[:])

	// Store first 8 characters as prefix for display
	if len(plainKey) >= 8 {
		keyPrefix = plainKey[:8]
	} else {
		keyPrefix = plainKey
	}

	return plainKey, keyHash, keyPrefix, nil
}

// CreateAPIKey creates a new API key for a user
// Returns the API key record and the plain-text key (only shown once)
func (r *APIKeyRepository) CreateAPIKey(ctx context.Context, userID uuid.UUID, name string, expiresAt *time.Time) (*APIKey, string, error) {
	// Generate the API key
	plainKey, keyHash, keyPrefix, err := GenerateAPIKey()
	if err != nil {
		return nil, "", err
	}

	query := `
		INSERT INTO api_keys (user_id, key_hash, key_prefix, name, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, key_hash, key_prefix, name, created_at, last_used_at, revoked_at, expires_at
	`

	var apiKey APIKey
	err = r.pool.QueryRow(ctx, query, userID, keyHash, keyPrefix, name, expiresAt).Scan(
		&apiKey.ID,
		&apiKey.UserID,
		&apiKey.KeyHash,
		&apiKey.KeyPrefix,
		&apiKey.Name,
		&apiKey.CreatedAt,
		&apiKey.LastUsedAt,
		&apiKey.RevokedAt,
		&apiKey.ExpiresAt,
	)

	if err != nil {
		return nil, "", fmt.Errorf("failed to create API key: %w", err)
	}

	return &apiKey, plainKey, nil
}

// GetAPIKeyByHash retrieves an API key by its hash (for authentication)
func (r *APIKeyRepository) GetAPIKeyByHash(ctx context.Context, keyHash string) (*APIKey, error) {
	query := `
		SELECT id, user_id, key_hash, key_prefix, name, created_at, last_used_at, revoked_at, expires_at
		FROM api_keys
		WHERE key_hash = $1
	`

	var apiKey APIKey
	err := r.pool.QueryRow(ctx, query, keyHash).Scan(
		&apiKey.ID,
		&apiKey.UserID,
		&apiKey.KeyHash,
		&apiKey.KeyPrefix,
		&apiKey.Name,
		&apiKey.CreatedAt,
		&apiKey.LastUsedAt,
		&apiKey.RevokedAt,
		&apiKey.ExpiresAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAPIKeyNotFound
		}
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	return &apiKey, nil
}

// ValidateAPIKey validates an API key and returns the associated user ID
// Checks if key is revoked or expired
func (r *APIKeyRepository) ValidateAPIKey(ctx context.Context, plainKey string) (uuid.UUID, error) {
	// Hash the provided key
	hash := sha256.Sum256([]byte(plainKey))
	keyHash := base64.StdEncoding.EncodeToString(hash[:])

	// Get the API key
	apiKey, err := r.GetAPIKeyByHash(ctx, keyHash)
	if err != nil {
		return uuid.Nil, err
	}

	// Check if revoked
	if apiKey.RevokedAt != nil {
		return uuid.Nil, ErrAPIKeyRevoked
	}

	// Check if expired
	if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
		return uuid.Nil, ErrAPIKeyExpired
	}

	// Update last used timestamp
	if err := r.UpdateLastUsed(ctx, apiKey.ID); err != nil {
		// Log error but don't fail authentication
		fmt.Printf("Warning: failed to update last_used_at for API key %d: %v\n", apiKey.ID, err)
	}

	return apiKey.UserID, nil
}

// ListAPIKeysByUser retrieves all API keys for a user (excluding hashes)
func (r *APIKeyRepository) ListAPIKeysByUser(ctx context.Context, userID uuid.UUID) ([]APIKey, error) {
	query := `
		SELECT id, user_id, key_hash, key_prefix, name, created_at, last_used_at, revoked_at, expires_at
		FROM api_keys
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}
	defer rows.Close()

	var apiKeys []APIKey
	for rows.Next() {
		var apiKey APIKey
		err := rows.Scan(
			&apiKey.ID,
			&apiKey.UserID,
			&apiKey.KeyHash,
			&apiKey.KeyPrefix,
			&apiKey.Name,
			&apiKey.CreatedAt,
			&apiKey.LastUsedAt,
			&apiKey.RevokedAt,
			&apiKey.ExpiresAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}
		apiKeys = append(apiKeys, apiKey)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating API keys: %w", err)
	}

	return apiKeys, nil
}

// RevokeAPIKey revokes an API key by ID
func (r *APIKeyRepository) RevokeAPIKey(ctx context.Context, keyID int, userID uuid.UUID) error {
	query := `
		UPDATE api_keys
		SET revoked_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL
	`

	result, err := r.pool.Exec(ctx, query, keyID, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke API key: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrAPIKeyNotFound
	}

	return nil
}

// UpdateLastUsed updates the last_used_at timestamp for an API key
func (r *APIKeyRepository) UpdateLastUsed(ctx context.Context, keyID int) error {
	query := `
		UPDATE api_keys
		SET last_used_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query, keyID)
	if err != nil {
		return fmt.Errorf("failed to update last used timestamp: %w", err)
	}

	return nil
}

// DeleteExpiredKeys deletes all expired API keys (cleanup operation)
func (r *APIKeyRepository) DeleteExpiredKeys(ctx context.Context) (int64, error) {
	query := `
		DELETE FROM api_keys
		WHERE expires_at IS NOT NULL AND expires_at < CURRENT_TIMESTAMP
	`

	result, err := r.pool.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired keys: %w", err)
	}

	return result.RowsAffected(), nil
}
