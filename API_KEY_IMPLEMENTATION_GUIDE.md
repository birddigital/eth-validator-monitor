# API Key Generation and Management Implementation Guide

## Project Overview

**Language**: Go 1.25.0  
**Framework**: Chi v5 HTTP router + gqlgen GraphQL  
**Database**: PostgreSQL 15 with pgx/v5  
**Caching**: Redis 7 with go-redis/v9  
**Authentication**: Session-based + JWT (optional)  
**ORM**: Direct SQL with repository pattern (no ORM)

---

## 1. Project Structure

### Key Directories

```
eth-validator-monitor/
├── cmd/
│   └── server/
│       └── main.go                    # Application entry point
├── internal/
│   ├── api/
│   │   ├── graphql/                   # GraphQL schema and resolvers
│   │   ├── middleware/                # HTTP middleware (auth, CORS, rate limit, etc.)
│   │   └── rest/                      # REST endpoints
│   ├── auth/                          # Authentication logic
│   │   ├── service.go                 # Auth business logic
│   │   ├── jwt.go                     # JWT token handling
│   │   ├── password.go                # Password hashing (bcrypt)
│   │   ├── session.go                 # Session management (Redis)
│   │   ├── middleware_session.go      # Session middleware
│   │   └── validator.go               # Input validation
│   ├── database/
│   │   ├── config.go                  # PostgreSQL connection config + pool
│   │   ├── migrate.go                 # Migration runner
│   │   ├── models/                    # Data models (not used heavily)
│   │   ├── repository/                # Database repositories
│   │   └── migrations/                # SQL migration files
│   ├── server/
│   │   ├── server.go                  # HTTP server setup
│   │   ├── router.go                  # Chi router with middleware chain
│   │   └── auth_handlers.go           # Auth endpoint handlers
│   ├── storage/
│   │   ├── postgres.go                # PostgreSQL utilities
│   │   └── user_repository.go         # User CRUD operations
│   ├── services/                      # Business logic
│   └── web/
│       └── handlers/                  # HTTP request handlers
├── migrations/
│   ├── 000001_init_schema.up.sql
│   ├── 000002_fix_validator_schema.up.sql
│   ├── 000003_add_users_table.up.sql  # User table schema
│   └── *.sql                          # Additional migrations
├── docker-compose.yml                 # Docker setup
├── Makefile                           # Build and test commands
└── go.mod                             # Go module definition
```

---

## 2. Database Setup

### Current Database Schema

#### Users Table
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    roles TEXT[] DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_login TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_is_active ON users(is_active);
```

**Location**: `/Users/bird/sources/standalone-projects/eth-validator-monitor/migrations/000003_add_users_table.up.sql`

### Database Connection Pattern

**File**: `/Users/bird/sources/standalone-projects/eth-validator-monitor/internal/database/config.go`

```go
// Create connection pool with configured settings
pool, err := database.NewPool(ctx, dbCfg)
if err != nil {
    logger.Logger.Fatal().Err(err).Msg("Failed to connect to database")
}
defer pool.Close()

// Use pool for all queries
var user User
err := pool.QueryRow(ctx, query, args...).Scan(&user.ID, &user.Name)
```

**Key Features**:
- pgx connection pooling (max 25 connections by default)
- Before/after connection hooks for prepared statements
- Statement timeout: 30s
- Lock timeout: 10s
- Idle transaction timeout: 60s
- SSL mode support (require/verify-ca/verify-full for production)

### Database Migrations

**Location**: `/Users/bird/sources/standalone-projects/eth-validator-monitor/migrations/`

**Migration Tool**: golang-migrate/migrate/v4

**Commands**:
```bash
make migrate-up         # Apply all pending migrations
make migrate-down       # Rollback last migration
make migrate-create NAME=name_here  # Create new migration
```

---

## 3. Existing API Patterns

### REST API Endpoints (HTTP)

**Base URL**: `http://localhost:8080`

**Authentication Routes**:
```
POST   /api/auth/register    # Register new user
POST   /api/auth/login       # Login user (creates session)
POST   /api/auth/logout      # Logout user
GET    /api/auth/me          # Get current user (authenticated)
```

**Dashboard Routes**:
```
GET    /api/dashboard/metrics
GET    /api/dashboard/alerts
GET    /api/dashboard/validators
GET    /api/dashboard/health
```

**Alerts Routes**:
```
GET    /api/alerts           # List alerts (JSON)
POST   /alerts/batch         # Batch alert actions
GET    /alerts/count         # Get alert count
```

**Validators Routes**:
```
GET    /api/validators/list  # List validators (JSON)
GET    /validators/{index}   # Get validator detail
GET    /validators/{index}/sse  # Validator SSE stream
```

**Settings Routes** (authenticated):
```
GET    /api/settings/content # Get settings page
POST   /api/settings/profile # Update profile
POST   /api/settings/password # Change password
```

### REST Handler Pattern

**File**: `/Users/bird/sources/standalone-projects/eth-validator-monitor/internal/server/auth_handlers.go`

```go
// Handler struct with dependencies
type AuthHandlers struct {
    authService  *auth.Service
    sessionStore *auth.SessionStore
}

// Request/Response types
type RegisterRequest struct {
    Username        string   `json:"username"`
    Password        string   `json:"password"`
    ConfirmPassword string   `json:"confirmPassword"`
    Email           string   `json:"email"`
    Roles           []string `json:"roles,omitempty"`
}

type UserResponse struct {
    ID       string   `json:"id"`
    Username string   `json:"username"`
    Email    string   `json:"email"`
    Roles    []string `json:"roles"`
}

// Handler method (implements http.Handler)
func (h *AuthHandlers) Register(w http.ResponseWriter, r *http.Request) {
    var req RegisterRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondValidationError(w, "Invalid request body", 
            map[string]string{"body": "invalid JSON"}, 
            http.StatusBadRequest)
        return
    }

    user, err := h.authService.Register(r.Context(), 
        req.Username, req.Password, req.ConfirmPassword, 
        req.Email, req.Roles)
    if err != nil {
        // Handle validation errors
        if verr, ok := err.(*auth.ValidationError); ok {
            respondValidationError(w, "Validation failed", 
                verr.Fields, http.StatusBadRequest)
            return
        }
        // ... more error handling
    }

    respondJSON(w, UserResponse{
        ID:       user.ID.String(),
        Username: user.Username,
        Email:    user.Email,
        Roles:    user.Roles,
    }, http.StatusCreated)
}

// Response helpers
func respondJSON(w http.ResponseWriter, data interface{}, statusCode int) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)
    json.NewEncoder(w).Encode(data)
}
```

### Router and Middleware Chain

**File**: `/Users/bird/sources/standalone-projects/eth-validator-monitor/internal/server/router.go`

**Middleware Stack** (in order):
1. Request ID (unique ID for tracing)
2. Real IP extraction (for proxies)
3. Structured logging (zerolog)
4. Panic recovery
5. Gzip compression
6. Request timeout (60s)
7. CORS (if enabled)
8. Rate limiting (if enabled)
9. Security headers
10. HTMX detection

**Registering Routes**:
```go
router := server.NewRouter(routerCfg)

// Public routes
r.Get("/health", healthHandler)

// Session-based auth routes
r.Route("/api/auth", func(r chi.Router) {
    r.Use(auth.SessionMiddleware(sessionStore))
    r.Post("/register", authHandlers.Register)
    r.Post("/login", authHandlers.Login)
    r.With(auth.RequireSessionAuth).Get("/me", authHandlers.Me)
})

// Protected routes (require session auth)
r.Group(func(r chi.Router) {
    r.Use(auth.SessionMiddleware(sessionStore))
    r.Use(auth.RequireSessionAuth)
    r.Get("/settings", settingsHandler.ServeHTTP)
})
```

---

## 4. Authentication and Middleware Patterns

### Session Authentication

**File**: `/Users/bird/sources/standalone-projects/eth-validator-monitor/internal/auth/session.go`

- Uses Redis store via go-redis and redisstore
- Sessions stored as HTTP cookies (secure, httponly by default)
- Session timeout: configurable (default from config)

**Middleware**:
```go
// Add session middleware to routes
r.Use(auth.SessionMiddleware(sessionStore))

// Require authentication
r.Use(auth.RequireSessionAuth)
```

**Context Access**:
```go
// In handlers: get user ID from context
userID, ok := auth.GetSessionUserIDFromContext(r.Context())
```

### JWT Authentication

**File**: `/Users/bird/sources/standalone-projects/eth-validator-monitor/internal/auth/jwt.go`

- HS256 signing algorithm
- Custom claims with UserID, Username, Roles
- Access token TTL: 15 minutes (configurable)
- Refresh token TTL: 7 days (configurable)

**Token Generation**:
```go
jwtService := auth.NewJWTService(
    secretKey,
    issuer,
    15*time.Minute,  // access duration
    7*24*time.Hour,  // refresh duration
)

accessToken, err := jwtService.GenerateAccessToken(userID, username, roles)
```

**GraphQL Middleware** (JWT):
```go
authMiddleware := middleware.NewAuthMiddleware(jwtService, logger)
r.Use(authMiddleware.Middleware)
```

### Password Hashing

**File**: `/Users/bird/sources/standalone-projects/eth-validator-monitor/internal/auth/password.go`

- Uses bcrypt with cost 12
- Hash generation and verification provided

```go
hash, err := auth.HashPassword(password)
err := auth.VerifyPassword(hash, password)
```

---

## 5. Service Layer Pattern

**File**: `/Users/bird/sources/standalone-projects/eth-validator-monitor/internal/auth/service.go`

```go
type Service struct {
    userRepo  *storage.UserRepository
    validator *Validator
}

func NewService(userRepo *storage.UserRepository) *Service {
    return &Service{
        userRepo:  userRepo,
        validator: NewValidator(),
    }
}

// Business logic methods
func (s *Service) Register(ctx context.Context, username, password, confirmPassword, email string, roles []string) (*storage.User, error) {
    // Validate inputs
    if err := s.validator.ValidateRegistration(username, email, password, confirmPassword); err != nil {
        return nil, err // Returns *ValidationError with field-level errors
    }

    // Check uniqueness
    // Hash password
    // Create user via repository
    return user, nil
}

func (s *Service) Login(ctx context.Context, username, password string) (*storage.User, error) {
    // Validate
    // Fetch user
    // Verify password
    // Update last login
    return user, nil
}

func (s *Service) GetUserByID(ctx context.Context, userID uuid.UUID) (*storage.User, error) {
    return s.userRepo.GetUserByID(ctx, userID)
}
```

---

## 6. Repository Pattern (Data Layer)

**File**: `/Users/bird/sources/standalone-projects/eth-validator-monitor/internal/storage/user_repository.go`

```go
type UserRepository struct {
    pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
    return &UserRepository{pool: pool}
}

// CRUD methods
func (r *UserRepository) CreateUser(ctx context.Context, username, email, passwordHash string, roles []string) (*User, error) {
    query := `
        INSERT INTO users (username, email, password_hash, roles)
        VALUES ($1, $2, $3, $4)
        RETURNING id, username, email, password_hash, roles, is_active, created_at, updated_at, last_login
    `
    var user User
    err := r.pool.QueryRow(ctx, query, username, email, passwordHash, roles).Scan(
        &user.ID, &user.Username, &user.Email, &user.PasswordHash, 
        &user.Roles, &user.IsActive, &user.CreatedAt, &user.UpdatedAt, &user.LastLogin,
    )
    // Handle unique constraint violation
    if err != nil {
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) && pgErr.Code == "23505" {
            return nil, ErrUserAlreadyExists
        }
    }
    return &user, err
}

func (r *UserRepository) GetUserByID(ctx context.Context, userID uuid.UUID) (*User, error) {
    query := `
        SELECT id, username, email, password_hash, roles, is_active, created_at, updated_at, last_login
        FROM users
        WHERE id = $1 AND is_active = true
    `
    var user User
    err := r.pool.QueryRow(ctx, query, userID).Scan(
        &user.ID, &user.Username, &user.Email, &user.PasswordHash, 
        &user.Roles, &user.IsActive, &user.CreatedAt, &user.UpdatedAt, &user.LastLogin,
    )
    if err != nil && errors.Is(err, pgx.ErrNoRows) {
        return nil, ErrUserNotFound
    }
    return &user, err
}

func (r *UserRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
    query := `
        UPDATE users
        SET password_hash = $1, updated_at = NOW()
        WHERE id = $2 AND is_active = true
    `
    result, err := r.pool.Exec(ctx, query, passwordHash, userID)
    if result.RowsAffected() == 0 {
        return ErrUserNotFound
    }
    return err
}

func (r *UserRepository) UpdateUserRoles(ctx context.Context, userID uuid.UUID, roles []string) error {
    // Similar pattern
}

// ... more methods
```

---

## 7. Configuration Pattern

**File**: `/Users/bird/sources/standalone-projects/eth-validator-monitor/internal/config/config.go`

```go
type Config struct {
    Server struct {
        HTTPPort              string
        GinMode              string  // "debug" or "release"
        CORSEnabled          bool
        CORSAllowedOrigins   []string
        RateLimitRequestsPerSec uint
        RateLimitBurst       uint
    }
    Database struct {
        Host     string
        Port     string
        User     string
        Password string
        Name     string
        SSLMode  string
    }
    Redis struct {
        Addr     string
        Password string
        DB       int
    }
    JWT struct {
        SecretKey            string
        Issuer              string
        AccessTokenDuration  time.Duration
        RefreshTokenDuration time.Duration
    }
    Session struct {
        SecretKey string
        MaxAge    time.Duration
        Secure    bool
        HttpOnly  bool
        SameSite  string
    }
    Logging struct {
        Level      string
        Format     string
        OutputPath string
        MaxSizeMB  int
        MaxBackups int
        MaxAgeDays int
        Compress   bool
    }
}

func Load() (*Config, error) {
    // Loads from environment variables (via godotenv)
    // Returns validated config
}
```

**Environment Variables**:
```bash
# Server
HTTP_PORT=8080
GIN_MODE=debug
CORS_ENABLED=true
CORS_ALLOWED_ORIGINS=http://localhost:3000

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=validator_monitor
DB_PASSWORD=postgres
DB_NAME=validator_monitor
DB_SSL_MODE=disable

# Redis
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# JWT (optional)
JWT_SECRET_KEY=your-secret-key-here
JWT_ISSUER=validator-monitor
JWT_ACCESS_TOKEN_TTL=15m
JWT_REFRESH_TOKEN_TTL=168h

# Session (optional)
SESSION_SECRET_KEY=your-session-secret-key
SESSION_MAX_AGE=24h
SESSION_SECURE=false
SESSION_HTTPONLY=true
SESSION_SAMESITE=Strict

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

---

## 8. Implementation Checklist for API Key Management

### Phase 1: Database Schema
- [ ] Create migration `000004_add_api_keys_table.up.sql`
- [ ] Create `api_keys` table with:
  - `id` (UUID, primary key)
  - `user_id` (UUID, foreign key to users)
  - `key_hash` (VARCHAR, bcrypt hash of the actual key)
  - `key_prefix` (VARCHAR, first 8 chars for display)
  - `name` (VARCHAR, user-friendly name)
  - `is_active` (BOOLEAN, default true)
  - `last_used_at` (TIMESTAMP, nullable)
  - `expires_at` (TIMESTAMP, nullable for no expiry)
  - `scopes` (TEXT[], default '{}' for granular permissions)
  - `created_at` (TIMESTAMP, default NOW())
  - `updated_at` (TIMESTAMP, default NOW())
- [ ] Create indexes on:
  - `idx_api_keys_user_id` for lookups by user
  - `idx_api_keys_key_hash` for validation
  - `idx_api_keys_expires_at` for cleanup queries

### Phase 2: Models and Repository
- [ ] Create `internal/storage/api_key.go` with:
  - `APIKey` struct (matching schema)
  - Error types (`ErrAPIKeyNotFound`, `ErrAPIKeyExpired`, `ErrInvalidAPIKey`)
- [ ] Create `APIKeyRepository` in `internal/storage/api_key_repository.go`:
  - `CreateAPIKey(ctx, userID, name, scopes, expiresAt)` → APIKey
  - `GetAPIKeyByHash(ctx, keyHash)` → APIKey (for validation)
  - `GetAPIKeysByUserID(ctx, userID)` → []*APIKey (list user's keys)
  - `GetAPIKeyByID(ctx, keyID)` → APIKey
  - `DeleteAPIKey(ctx, keyID)` → error (soft delete with is_active=false)
  - `UpdateLastUsed(ctx, keyID)` → error
  - `RevokeAPIKey(ctx, keyID)` → error
  - `ListAPIKeys(ctx, userID, limit, offset)` → []*APIKey, total count

### Phase 3: Service Layer
- [ ] Create `internal/auth/api_key_service.go`:
  - `GenerateAPIKey(ctx, userID, name, scopes, expiresAt)` → (keyString, *APIKey, error)
    - Generate cryptographically secure random key (32 bytes)
    - Hash with bcrypt
    - Store in database
    - Return full key (only time it's visible!)
  - `ValidateAPIKey(ctx, keyString)` → (*APIKey, error)
    - Hash provided key
    - Look up in database
    - Check if expired, active, and not revoked
    - Update last_used_at
  - `ListUserAPIKeys(ctx, userID)` → []*APIKey, error
  - `RevokeAPIKey(ctx, userID, keyID)` → error
  - `CheckKeyScopes(key *APIKey, requiredScopes []string)` → bool

### Phase 4: HTTP Middleware
- [ ] Create `internal/auth/middleware_api_key.go`:
  - `APIKeyAuthMiddleware(apiKeyService *APIKeyService)` → http.HandlerFunc
    - Check `X-API-Key` or `Authorization: Bearer` header
    - Call `apiKeyService.ValidateAPIKey()`
    - Add to request context
    - Handle errors (401 Unauthorized, 410 Gone for revoked)

### Phase 5: REST Endpoints
- [ ] Create `internal/web/handlers/api_keys.go`:
  - `APIKeysHandler` struct with dependencies
  - `GenerateAPIKey(w, r)` - POST /api/keys (returns full key once)
  - `ListAPIKeys(w, r)` - GET /api/keys (list user's keys)
  - `RevokeAPIKey(w, r)` - DELETE /api/keys/{id} (soft delete)
  - `GetAPIKeyDetails(w, r)` - GET /api/keys/{id} (without secret)
- [ ] Register routes in `cmd/server/main.go`:
  ```go
  r.Route("/api/keys", func(r chi.Router) {
      r.Use(auth.SessionMiddleware(sessionStore))
      r.Use(auth.RequireSessionAuth)
      r.Post("/", apiKeysHandler.GenerateAPIKey)
      r.Get("/", apiKeysHandler.ListAPIKeys)
      r.Get("/{id}", apiKeysHandler.GetAPIKeyDetails)
      r.Delete("/{id}", apiKeysHandler.RevokeAPIKey)
  })
  ```

### Phase 6: GraphQL Integration (Optional)
- [ ] Add GraphQL types in `graph/schema.graphql`:
  ```graphql
  type APIKey {
      id: ID!
      name: String!
      keyPrefix: String!
      scopes: [String!]!
      isActive: Boolean!
      lastUsedAt: Time
      expiresAt: Time
      createdAt: Time!
  }

  extend type Mutation {
      generateAPIKey(name: String!, scopes: [String!]): GenerateAPIKeyResponse!
      revokeAPIKey(id: ID!): Boolean!
  }

  type GenerateAPIKeyResponse {
      key: String!  # Only returned once
      apiKey: APIKey!
  }
  ```
- [ ] Generate GraphQL code: `make generate`
- [ ] Implement resolvers in `graph/schema.resolvers.go`

### Phase 7: Testing
- [ ] Unit tests for API key generation (randomness, format)
- [ ] Unit tests for hashing and validation
- [ ] Integration tests for repository CRUD
- [ ] Integration tests for middleware (valid key, expired, revoked, invalid format)
- [ ] Integration tests for endpoints (create, list, revoke)
- [ ] Edge cases:
  - API key with no expiry
  - API key with past expiry date
  - Revoked key
  - Multiple keys per user
  - Key scope validation

### Phase 8: Documentation
- [ ] Add API key section to README.md
- [ ] Document in docs/API.md:
  - API key generation endpoint
  - Usage in headers
  - Rate limiting per API key
  - Key rotation strategy
  - Scope meanings
- [ ] Add example in examples/ (cURL, JS, Python, Go)

### Phase 9: Security Hardening
- [ ] Add rate limiting per API key (not just per IP)
- [ ] Add key rotation reminders (email alert if key > 90 days old)
- [ ] Add audit logging for API key operations
- [ ] Implement key scope validation for endpoints
- [ ] Add key expiry auto-rotation in background job
- [ ] Monitor for suspicious API key usage patterns

---

## 9. Key Files Reference

| File | Purpose |
|------|---------|
| `/cmd/server/main.go` | Application entry point, route registration |
| `/internal/auth/service.go` | Authentication business logic |
| `/internal/auth/jwt.go` | JWT token generation and validation |
| `/internal/auth/password.go` | Password hashing utilities |
| `/internal/server/router.go` | Chi router with middleware setup |
| `/internal/server/auth_handlers.go` | REST auth endpoints |
| `/internal/storage/user_repository.go` | User CRUD operations |
| `/internal/database/config.go` | PostgreSQL connection and pooling |
| `/internal/config/config.go` | Environment configuration |
| `/migrations/000003_add_users_table.up.sql` | User table schema |
| `/internal/server/server.go` | HTTP server setup with graceful shutdown |
| `/graph/middleware/` | GraphQL authentication middleware |

---

## 10. Dependencies Already Installed

```go
// JWT
github.com/golang-jwt/jwt/v5 v5.3.0

// Database
github.com/jackc/pgx/v5 v5.7.6
github.com/golang-migrate/migrate/v4 v4.19.0

// HTTP Router
github.com/go-chi/chi/v5 v5.2.3

// Redis/Sessions
github.com/redis/go-redis/v9 v9.14.0
github.com/rbcervilla/redisstore/v9 v9.0.0

// Validation
github.com/go-playground/validator/v10 v10.28.0

// Utilities
github.com/google/uuid v1.6.0
golang.org/x/crypto v0.43.0
```

No additional dependencies needed for API key implementation!

---

## 11. Implementation Flow Diagram

```
User Request with API Key
        ↓
API Key Middleware
        ↓
Extract key from header (X-API-Key or Authorization)
        ↓
Hash the key
        ↓
APIKeyService.ValidateAPIKey()
        ↓
Query database for key_hash match
        ↓
Check: is_active=true, expires_at > NOW()
        ↓
Update last_used_at timestamp
        ↓
Add to context: ctx.WithValue(contextKey, apiKey)
        ↓
Next handler receives authenticated context
        ↓
Handler can access: apiKey.UserID, apiKey.Scopes
        ↓
Check scope permissions if needed
        ↓
Return response
```

---

## 12. Error Handling Pattern

```go
var (
    ErrAPIKeyNotFound   = errors.New("api key not found")
    ErrAPIKeyExpired    = errors.New("api key has expired")
    ErrAPIKeyRevoked    = errors.New("api key has been revoked")
    ErrInvalidAPIKey    = errors.New("invalid api key format")
    ErrInsufficientScope = errors.New("api key lacks required scope")
)

// In middleware/handlers, convert to HTTP responses:
type ErrorResponse struct {
    Error   string            `json:"error"`
    Message string            `json:"message,omitempty"`
    Fields  map[string]string `json:"fields,omitempty"`
}

// Usage
if errors.Is(err, ErrAPIKeyNotFound) {
    respondJSON(w, ErrorResponse{
        Error: "Unauthorized",
        Message: "Invalid or expired API key",
    }, http.StatusUnauthorized)
    return
}

if errors.Is(err, ErrAPIKeyRevoked) {
    respondJSON(w, ErrorResponse{
        Error: "Gone",
        Message: "This API key has been revoked",
    }, http.StatusGone)
    return
}
```

---

## 13. Testing Template

```go
package storage_test

import (
    "context"
    "testing"
    "time"
    
    "github.com/birddigital/eth-validator-monitor/internal/storage"
    "github.com/stretchr/testify/assert"
    "github.com/google/uuid"
)

func TestCreateAPIKey(t *testing.T) {
    // Setup: create test pool, repository
    pool := setupTestDB(t)
    defer pool.Close()
    repo := storage.NewAPIKeyRepository(pool)
    
    // Test: create API key
    userID := uuid.New()
    apiKey, err := repo.CreateAPIKey(context.Background(), 
        userID, "my-key", []string{"read:validators"}, nil)
    
    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, apiKey)
    assert.Equal(t, userID, apiKey.UserID)
    assert.Equal(t, "my-key", apiKey.Name)
    assert.True(t, apiKey.IsActive)
}

func TestValidateAPIKey(t *testing.T) {
    // ... similar pattern
}
```

---

## Conclusion

The eth-validator-monitor project follows a clean, layered architecture:

1. **REST Handlers** → JSON request/response
2. **Services** → Business logic and validation
3. **Repositories** → Data access (PostgreSQL)
4. **Models** → Domain objects
5. **Middleware** → Cross-cutting concerns (auth, logging, rate limiting)

For API key implementation, follow these same patterns:
- Create migration for schema
- Create repository for CRUD
- Create service for business logic
- Create middleware for HTTP auth
- Create handlers for REST endpoints
- Add to router in main.go

All infrastructure is already in place!
