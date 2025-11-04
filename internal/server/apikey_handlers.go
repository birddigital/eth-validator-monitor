package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/birddigital/eth-validator-monitor/internal/auth"
	"github.com/birddigital/eth-validator-monitor/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// APIKeyHandlers handles HTTP API key management endpoints
type APIKeyHandlers struct {
	apiKeyRepo *storage.APIKeyRepository
}

// NewAPIKeyHandlers creates new API key handlers
func NewAPIKeyHandlers(apiKeyRepo *storage.APIKeyRepository) *APIKeyHandlers {
	return &APIKeyHandlers{
		apiKeyRepo: apiKeyRepo,
	}
}

// CreateAPIKeyRequest is the request body for creating an API key
type CreateAPIKeyRequest struct {
	Name      string  `json:"name"`
	ExpiresIn *int    `json:"expiresIn,omitempty"` // Days until expiration (optional)
}

// CreateAPIKeyResponse is the response for API key creation
// WARNING: The plainKey is only returned once and cannot be retrieved again
type CreateAPIKeyResponse struct {
	ID        int       `json:"id"`
	PlainKey  string    `json:"plainKey"` // Only shown once!
	KeyPrefix string    `json:"keyPrefix"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

// APIKeyListResponse is the response for listing API keys
type APIKeyListResponse struct {
	ID         int        `json:"id"`
	KeyPrefix  string     `json:"keyPrefix"` // First 8 chars for identification
	Name       string     `json:"name"`
	CreatedAt  time.Time  `json:"createdAt"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
	RevokedAt  *time.Time `json:"revokedAt,omitempty"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
	IsActive   bool       `json:"isActive"` // Computed: not revoked and not expired
}

// CreateAPIKey handles POST /api/keys
// Creates a new API key for the authenticated user
func (h *APIKeyHandlers) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user ID from context (set by auth middleware)
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		respondError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondValidationError(w, "Invalid request body", map[string]string{"body": "invalid JSON"}, http.StatusBadRequest)
		return
	}

	// Validate name
	if req.Name == "" {
		respondValidationError(w, "Validation failed", map[string]string{
			"name": "API key name is required",
		}, http.StatusBadRequest)
		return
	}

	// Calculate expiration time if provided
	var expiresAt *time.Time
	if req.ExpiresIn != nil {
		if *req.ExpiresIn <= 0 {
			respondValidationError(w, "Validation failed", map[string]string{
				"expiresIn": "must be a positive number of days",
			}, http.StatusBadRequest)
			return
		}
		expiry := time.Now().AddDate(0, 0, *req.ExpiresIn)
		expiresAt = &expiry
	}

	// Create API key
	apiKey, plainKey, err := h.apiKeyRepo.CreateAPIKey(r.Context(), userID, req.Name, expiresAt)
	if err != nil {
		respondError(w, "Failed to create API key", http.StatusInternalServerError)
		return
	}

	// Return response with plain-text key (only time it's shown!)
	response := CreateAPIKeyResponse{
		ID:        apiKey.ID,
		PlainKey:  plainKey,
		KeyPrefix: apiKey.KeyPrefix,
		Name:      apiKey.Name,
		CreatedAt: apiKey.CreatedAt,
		ExpiresAt: apiKey.ExpiresAt,
	}

	respondJSON(w, response, http.StatusCreated)
}

// ListAPIKeys handles GET /api/keys
// Lists all API keys for the authenticated user
func (h *APIKeyHandlers) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user ID from context
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		respondError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Retrieve API keys
	apiKeys, err := h.apiKeyRepo.ListAPIKeysByUser(r.Context(), userID)
	if err != nil {
		respondError(w, "Failed to retrieve API keys", http.StatusInternalServerError)
		return
	}

	// Transform to response format (excluding hashes)
	response := make([]APIKeyListResponse, 0, len(apiKeys))
	now := time.Now()
	for _, key := range apiKeys {
		isActive := key.RevokedAt == nil && (key.ExpiresAt == nil || now.Before(*key.ExpiresAt))
		response = append(response, APIKeyListResponse{
			ID:         key.ID,
			KeyPrefix:  key.KeyPrefix,
			Name:       key.Name,
			CreatedAt:  key.CreatedAt,
			LastUsedAt: key.LastUsedAt,
			RevokedAt:  key.RevokedAt,
			ExpiresAt:  key.ExpiresAt,
			IsActive:   isActive,
		})
	}

	respondJSON(w, response, http.StatusOK)
}

// RevokeAPIKey handles DELETE /api/keys/{id}
// Revokes an API key by ID
func (h *APIKeyHandlers) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user ID from context
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		respondError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse key ID from URL
	keyIDStr := chi.URLParam(r, "id")
	keyID, err := strconv.Atoi(keyIDStr)
	if err != nil {
		respondError(w, "Invalid API key ID", http.StatusBadRequest)
		return
	}

	// Revoke the API key
	err = h.apiKeyRepo.RevokeAPIKey(r.Context(), keyID, userID)
	if err != nil {
		if err == storage.ErrAPIKeyNotFound {
			respondError(w, "API key not found or already revoked", http.StatusNotFound)
			return
		}
		respondError(w, "Failed to revoke API key", http.StatusInternalServerError)
		return
	}

	// Return success with no content
	w.WriteHeader(http.StatusNoContent)
}

// respondJSON sends a JSON response with the given status code
func respondJSON(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Log error but can't change headers at this point
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// respondError sends a standard error response
func respondError(w http.ResponseWriter, message string, statusCode int) {
	respondJSON(w, ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
	}, statusCode)
}

// respondValidationError sends a validation error with field-level details
func respondValidationError(w http.ResponseWriter, message string, fields map[string]string, statusCode int) {
	respondJSON(w, ErrorResponse{
		Error:   "Validation Error",
		Message: message,
		Fields:  fields,
	}, statusCode)
}
