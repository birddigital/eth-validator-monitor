# Codebase Exploration Summary: eth-validator-monitor

## Quick Reference

### 1. Project Tech Stack

| Layer | Technology | Version |
|-------|-----------|---------|
| **Language** | Go | 1.25.0 |
| **HTTP Router** | Chi | v5.2.3 |
| **GraphQL** | gqlgen | v0.17.81 |
| **Database** | PostgreSQL 15 | pgx/v5 |
| **Cache** | Redis 7 | go-redis/v9 |
| **Auth** | JWT + Sessions | golang-jwt/v5, gorilla/sessions |
| **Migration** | golang-migrate | v4.19.0 |
| **Logging** | zerolog | v1.34.0 |
| **API Testing** | Playwright | Built-in |

---

### 2. Directory Structure (Key Files)

```
cmd/server/main.go
├─ Entry point
├─ Config loading
├─ Database initialization
├─ Redis connection
├─ Router setup
└─ Route registration

internal/
├─ api/
│  ├─ graphql/          # GraphQL schema + resolvers
│  ├─ rest/             # REST endpoints
│  └─ middleware/       # HTTP middleware (auth, CORS, logging, rate limit)
├─ auth/
│  ├─ service.go        # Login/Register business logic
│  ├─ jwt.go            # Token generation/validation
│  ├─ password.go       # bcrypt hashing
│  ├─ session.go        # Redis session store
│  └─ validator.go      # Input validation
├─ database/
│  ├─ config.go         # PostgreSQL connection pool setup
│  ├─ migrate.go        # Migration runner
│  └─ repository/       # Database queries (validators, dashboards)
├─ server/
│  ├─ server.go         # HTTP server with graceful shutdown
│  ├─ router.go         # Chi router with middleware chain
│  └─ auth_handlers.go  # POST /api/auth/* endpoints
├─ storage/
│  ├─ user_repository.go # User CRUD (PostgreSQL)
│  └─ postgres.go       # Error handling utilities
└─ web/handlers/        # Page rendering + SSE handlers

migrations/
├─ 000001_init_schema.up.sql      # Initial database schema
├─ 000002_fix_validator_schema.up.sql
├─ 000003_add_users_table.up.sql  # User table (id, username, email, password_hash, roles)
└─ *.sql                          # Additional migrations

docker-compose.yml                # PostgreSQL, Redis, Prometheus, Grafana setup
.env.example                      # Configuration template
```

---

### 3. Database Schema

#### Users Table (000003_add_users_table.up.sql)
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

#### Connection Pattern
```go
// internal/database/config.go
pool, err := database.NewPool(ctx, dbCfg)  // pgx connection pool
defer pool.Close()

// Use for queries
var user User
err := pool.QueryRow(ctx, query, args...).Scan(&user.ID)
```

**Pool Settings**:
- Max connections: 25
- Min connections: 5
- Max lifetime: 1 hour
- Idle timeout: 30 minutes
- Connection timeout: 5s
- Statement timeout: 30s

---

### 4. API Endpoints Overview

#### Authentication (Session-based)
```
POST   /api/auth/register      # Register user → creates session
POST   /api/auth/login         # Login user → creates session
POST   /api/auth/logout        # Logout → destroys session
GET    /api/auth/me            # Get current user (requires session)
```

#### Dashboard
```
GET    /api/dashboard/metrics
GET    /api/dashboard/alerts
GET    /api/dashboard/validators
GET    /api/dashboard/health
```

#### Validators
```
GET    /api/validators/list
GET    /validators/{index}
GET    /validators/{index}/sse
```

#### Alerts
```
GET    /api/alerts
POST   /alerts/batch
GET    /alerts/count
```

#### Settings (Protected)
```
GET    /api/settings/content
POST   /api/settings/profile
POST   /api/settings/password
```

#### GraphQL
```
POST   /graphql              # GraphQL API
GET    /playground           # Interactive playground (debug mode)
```

---

### 5. Authentication Flow

#### Session-Based (Default)
```
User Input
    ↓
POST /api/auth/login
    ↓
AuthHandlers.Login()
    ↓
AuthService.Login()
    ├─ ValidateLogin() → *ValidationError if invalid
    ├─ FetchUser by username
    ├─ VerifyPassword() with bcrypt
    ├─ UpdateLastLogin()
    └─ Return *storage.User
    ↓
SessionStore.SetUserSession()
    └─ Set session cookie
    ↓
Response: 200 OK + Set-Cookie header
    ↓
Subsequent requests include session cookie
    ↓
Middleware: auth.SessionMiddleware() → extracts user ID from context
```

#### JWT-Based (Optional, if JWT_SECRET_KEY set)
```
POST /graphql with Authorization: Bearer <token>
    ↓
middleware.AuthMiddleware()
    ├─ Extract token from header
    ├─ JWTService.ValidateToken()
    │  └─ Parse and verify signature
    ├─ Check expiry
    └─ Add claims to context
    ↓
Resolver receives authenticated context
    ↓
Use auth.GetUserClaims(ctx) to access claims
```

---

### 6. Handler Pattern (REST)

**File**: `internal/server/auth_handlers.go`

```go
// 1. Define request/response types
type LoginRequest struct {
    Username string `json:"username"`
    Password string `json:"password"`
}

type UserResponse struct {
    ID       string   `json:"id"`
    Username string   `json:"username"`
    Email    string   `json:"email"`
    Roles    []string `json:"roles"`
}

// 2. Create handler struct with dependencies
type AuthHandlers struct {
    authService  *auth.Service
    sessionStore *auth.SessionStore
}

// 3. Implement handler method
func (h *AuthHandlers) Login(w http.ResponseWriter, r *http.Request) {
    // Decode request
    var req LoginRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondValidationError(w, "Invalid request", 
            map[string]string{"body": "invalid JSON"}, 
            http.StatusBadRequest)
        return
    }

    // Call service (handles business logic + validation)
    user, err := h.authService.Login(r.Context(), req.Username, req.Password)
    if err != nil {
        if verr, ok := err.(*auth.ValidationError); ok {
            respondValidationError(w, "Validation failed", verr.Fields, http.StatusBadRequest)
            return
        }
        if err == auth.ErrInvalidCredentials {
            respondValidationError(w, "Invalid credentials", 
                map[string]string{"credentials": "invalid username or password"}, 
                http.StatusUnauthorized)
            return
        }
        respondError(w, "Login failed", http.StatusInternalServerError)
        return
    }

    // Update session
    session, _ := h.sessionStore.Get(r)
    h.sessionStore.SetUserSession(session, user.ID, user.Username)
    h.sessionStore.Save(r, w, session)

    // Return response
    respondJSON(w, UserResponse{
        ID:       user.ID.String(),
        Username: user.Username,
        Email:    user.Email,
        Roles:    user.Roles,
    }, http.StatusOK)
}

// 4. Helper functions for consistent responses
func respondJSON(w http.ResponseWriter, data interface{}, statusCode int) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)
    json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, message string, statusCode int) {
    respondJSON(w, ErrorResponse{Error: message}, statusCode)
}

func respondValidationError(w http.ResponseWriter, message string, fields map[string]string, statusCode int) {
    respondJSON(w, ErrorResponse{
        Error:   http.StatusText(statusCode),
        Message: message,
        Fields:  fields,  // Field-level errors
    }, statusCode)
}
```

---

### 7. Repository Pattern (Data Layer)

**File**: `internal/storage/user_repository.go`

```go
// 1. Define repository struct
type UserRepository struct {
    pool *pgxpool.Pool
}

// 2. Implement CRUD methods
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
    
    // Handle unique constraint violation (duplicate username/email)
    if err != nil {
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) && pgErr.Code == "23505" {
            return nil, ErrUserAlreadyExists
        }
        return nil, fmt.Errorf("failed to create user: %w", err)
    }
    
    return &user, nil
}

func (r *UserRepository) GetUserByID(ctx context.Context, userID uuid.UUID) (*User, error) {
    query := `SELECT id, username, email, password_hash, roles, is_active, created_at, updated_at, last_login 
              FROM users WHERE id = $1 AND is_active = true`
    
    var user User
    err := r.pool.QueryRow(ctx, query, userID).Scan(
        &user.ID, &user.Username, &user.Email, &user.PasswordHash, 
        &user.Roles, &user.IsActive, &user.CreatedAt, &user.UpdatedAt, &user.LastLogin,
    )
    
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, ErrUserNotFound
        }
        return nil, fmt.Errorf("failed to get user: %w", err)
    }
    
    return &user, nil
}

func (r *UserRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
    query := `UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2 AND is_active = true`
    
    result, err := r.pool.Exec(ctx, query, passwordHash, userID)
    if err != nil {
        return fmt.Errorf("failed to update password: %w", err)
    }
    
    if result.RowsAffected() == 0 {
        return ErrUserNotFound
    }
    
    return nil
}
```

---

### 8. Middleware Chain

**File**: `internal/server/router.go`

Middleware applied in order:
1. **Request ID** - Unique ID for tracing (X-Request-ID header)
2. **Real IP** - Extract real IP behind proxies
3. **Logging** - Structure request/response logs with zerolog
4. **Panic Recovery** - Catch panics, return 500, log with full context
5. **Compression** - gzip compression (level 5)
6. **Timeout** - 60s request timeout
7. **CORS** - If enabled (via config)
8. **Rate Limiting** - Per IP rate limiting (if enabled)
9. **Security Headers** - X-Frame-Options, X-Content-Type-Options, etc.
10. **HTMX Detection** - Detect HTMX requests for special handling

**Adding to routes**:
```go
// Public route
r.Get("/health", handler)

// Protected route (requires session auth)
r.Group(func(r chi.Router) {
    r.Use(auth.SessionMiddleware(sessionStore))
    r.Use(auth.RequireSessionAuth)
    r.Get("/api/protected", protectedHandler)
})

// API Key protected (custom middleware)
r.Group(func(r chi.Router) {
    r.Use(apiKeyAuthMiddleware)
    r.Get("/api/v1/data", apiHandler)
})
```

---

### 9. Configuration Loading

**File**: `internal/config/config.go`

```go
// Loads from environment variables via godotenv

// Server
HTTP_PORT=8080
GIN_MODE=debug
PROMETHEUS_PORT=9090
RATE_LIMIT_ENABLED=true
RATE_LIMIT_RPS=10
RATE_LIMIT_BURST=20
CORS_ENABLED=true
CORS_ALLOWED_ORIGINS=http://localhost:3000

// Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=validator_monitor
DB_PASSWORD=postgres
DB_NAME=validator_monitor
DB_SSL_MODE=disable

// Redis
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

// JWT (optional)
JWT_SECRET_KEY=your-secret-key
JWT_ISSUER=validator-monitor
JWT_ACCESS_TOKEN_TTL=15m
JWT_REFRESH_TOKEN_TTL=168h

// Session
SESSION_SECRET_KEY=your-session-secret
SESSION_MAX_AGE=24h
SESSION_SECURE=false
SESSION_HTTPONLY=true
SESSION_SAMESITE=Strict

// Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

---

### 10. Service Layer Pattern

**File**: `internal/auth/service.go`

```go
// 1. Define service with dependencies
type Service struct {
    userRepo  *storage.UserRepository
    validator *Validator
}

// 2. Implement business logic methods
func (s *Service) Register(ctx context.Context, username, password, confirmPassword, email string, roles []string) (*storage.User, error) {
    // Validate input
    if err := s.validator.ValidateRegistration(username, email, password, confirmPassword); err != nil {
        return nil, err  // Returns *ValidationError with field-level errors
    }
    
    // Check uniqueness
    existing, err := s.userRepo.GetUserByUsername(ctx, username)
    if existing != nil {
        return nil, ErrUserAlreadyExists
    }
    
    // Hash password
    hashedPassword, err := auth.HashPassword(password)
    if err != nil {
        return nil, fmt.Errorf("failed to hash password: %w", err)
    }
    
    // Create user via repository
    user, err := s.userRepo.CreateUser(ctx, username, email, hashedPassword, roles)
    if err != nil {
        return nil, fmt.Errorf("failed to create user: %w", err)
    }
    
    return user, nil
}

func (s *Service) Login(ctx context.Context, username, password string) (*storage.User, error) {
    // Validate
    if err := s.validator.ValidateLogin(username, password); err != nil {
        return nil, err
    }
    
    // Fetch user
    user, err := s.userRepo.GetUserByUsername(ctx, username)
    if err != nil {
        if errors.Is(err, storage.ErrUserNotFound) {
            return nil, ErrInvalidCredentials  // Don't leak if user exists
        }
        return nil, err
    }
    
    // Verify password
    if err := auth.VerifyPassword(user.PasswordHash, password); err != nil {
        return nil, ErrInvalidCredentials
    }
    
    // Update last login
    s.userRepo.UpdateLastLogin(ctx, user.ID)
    
    return user, nil
}
```

---

### 11. Error Handling

**Pattern**:
```go
// Define custom errors
var (
    ErrInvalidCredentials = errors.New("invalid username or password")
    ErrUserAlreadyExists  = errors.New("user already exists")
    ErrUserNotFound       = errors.New("user not found")
)

// In service layer: use error checks
if err == auth.ErrInvalidCredentials {
    return nil, err  // Let caller decide HTTP status
}

// In handler: convert to HTTP responses
if err == auth.ErrInvalidCredentials {
    respondValidationError(w, "Invalid credentials",
        map[string]string{"credentials": "invalid username or password"},
        http.StatusUnauthorized)
    return
}

// Validation errors contain field-level details
if verr, ok := err.(*auth.ValidationError); ok {
    respondValidationError(w, "Validation failed", verr.Fields, http.StatusBadRequest)
    return
}
```

---

### 12. Database Migrations

**Location**: `migrations/`

**Tool**: golang-migrate/migrate

**Files**:
- `000001_init_schema.up.sql` / `.down.sql`
- `000002_fix_validator_schema.up.sql` / `.down.sql`
- `000003_add_users_table.up.sql` / `.down.sql`

**Commands**:
```bash
make migrate-up         # Apply all pending migrations
make migrate-down       # Rollback last migration
make migrate-create NAME=name  # Create new migration
```

**Creating new migration**:
```bash
# Creates: migrations/000004_your_migration.up.sql
#          migrations/000004_your_migration.down.sql
make migrate-create NAME=your_migration
```

---

### 13. Context and Values

**User context keys**:
```go
// Session-based
userID, ok := auth.GetSessionUserIDFromContext(r.Context())

// JWT-based
claims, ok := auth.GetUserClaims(r.Context())
userID := claims.UserID
roles := claims.Roles

// Custom values
ctx := context.WithValue(r.Context(), "api_key", apiKey)
apiKey := r.Context().Value("api_key").(*APIKey)
```

---

### 14. Key Files Summary

| File | Lines | Purpose |
|------|-------|---------|
| `cmd/server/main.go` | 300+ | Entry point, route registration |
| `internal/auth/service.go` | 150+ | Business logic (login, register) |
| `internal/auth/jwt.go` | 120+ | JWT generation/validation |
| `internal/server/router.go` | 100+ | Middleware chain setup |
| `internal/server/auth_handlers.go` | 180+ | REST endpoint handlers |
| `internal/storage/user_repository.go` | 200+ | User CRUD operations |
| `internal/database/config.go` | 150+ | Database pool configuration |
| `migrations/000003_add_users_table.up.sql` | 20+ | User table schema |

---

### 15. Testing Infrastructure

**Test Packages Used**:
- `github.com/stretchr/testify` - Assertions and mocks
- `github.com/testcontainers/testcontainers-go` - Docker containers for integration tests

**Test Patterns**:
```bash
make test              # Run all tests
make test-coverage     # Coverage report
make test-coverage && open coverage.html
make lint              # Run linters
make security-scan     # Security scan with gosec
```

---

## Next Steps for API Key Implementation

1. **Create Database Migration** - New table for API keys (hash, prefix, scopes, expiry)
2. **Create Models** - `APIKey` struct and repository methods
3. **Create Service** - `APIKeyService` for generation, validation, revocation
4. **Create Middleware** - HTTP middleware for API key authentication
5. **Create Handlers** - REST endpoints for CRUD operations
6. **Register Routes** - Add to router in main.go
7. **Add Tests** - Unit and integration tests
8. **Update Documentation** - README and API docs

See `/API_KEY_IMPLEMENTATION_GUIDE.md` for detailed checklist and code examples!
