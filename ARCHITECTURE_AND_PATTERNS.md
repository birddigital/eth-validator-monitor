# Architecture and Design Patterns

## 1. Layered Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    HTTP Client / Browser                     │
└────────────────────────────┬────────────────────────────────┘
                             │
                    POST /api/auth/login
                             │
┌─────────────────────────────▼────────────────────────────────┐
│                  HTTP Server (Chi Router)                     │
│              Port 8080 (configurable via env)               │
└─────────────────────────────┬────────────────────────────────┘
                             │
┌─────────────────────────────▼────────────────────────────────┐
│                    Middleware Chain                           │
│  1. Request ID → 2. Real IP → 3. Logging → 4. Panic        │
│  5. Compression → 6. Timeout → 7. CORS → 8. Rate Limit    │
│  9. Security Headers → 10. HTMX Detection                  │
└─────────────────────────────┬────────────────────────────────┘
                             │
┌─────────────────────────────▼────────────────────────────────┐
│                Route Handler / Controller                     │
│            (AuthHandlers.Login in auth_handlers.go)         │
│  • Extract request body (JSON decode)                       │
│  • Call service layer for business logic                    │
│  • Handle errors from service layer                         │
│  • Build response (JSON encode)                             │
└─────────────────────────────┬────────────────────────────────┘
                             │
┌─────────────────────────────▼────────────────────────────────┐
│                    Service Layer                             │
│            (AuthService in auth/service.go)                 │
│  • Validate input (ValidateLogin)                           │
│  • Fetch user from repository                               │
│  • Verify password (VerifyPassword)                         │
│  • Update last login                                        │
│  • Return user or error                                     │
└─────────────────────────────┬────────────────────────────────┘
                             │
┌─────────────────────────────▼────────────────────────────────┐
│                 Repository Layer                             │
│         (UserRepository in storage/user_repository.go)      │
│  • SQL query construction                                   │
│  • Database operations (CRUD)                               │
│  • Error handling and conversion                            │
│  • Return domain objects                                    │
└─────────────────────────────┬────────────────────────────────┘
                             │
┌─────────────────────────────▼────────────────────────────────┐
│            Database Layer (PostgreSQL)                        │
│  • Persistent storage via pgxpool                           │
│  • SQL execution                                            │
│  • Connection pooling (25 max)                              │
└─────────────────────────────┬────────────────────────────────┘
                             │
                         users table
```

---

## 2. Request/Response Flow (Example: POST /api/auth/login)

```
┌──────────────────────────────────────────────────────────────┐
│ Client sends POST /api/auth/login                            │
│ Body: {"username": "john", "password": "secret123"}         │
└──────────────────────┬───────────────────────────────────────┘
                       │
                       ▼
┌──────────────────────────────────────────────────────────────┐
│ 1. Router receives request (Chi)                             │
│    Match route to handler: AuthHandlers.Login                │
└──────────────────────┬───────────────────────────────────────┘
                       │
                       ▼
┌──────────────────────────────────────────────────────────────┐
│ 2. Middleware Chain Executes                                 │
│    - Request ID: Add X-Request-ID header                    │
│    - Logging: Log request method/path                       │
│    - Rate Limit: Check requests per IP                      │
│    [If auth required: SessionMiddleware]                     │
│    [If auth required: RequireSessionAuth]                    │
└──────────────────────┬───────────────────────────────────────┘
                       │
                       ▼
┌──────────────────────────────────────────────────────────────┐
│ 3. AuthHandlers.Login() executes                             │
│    │                                                         │
│    ├─► json.Decoder.Decode(&req)                           │
│    │   └─► LoginRequest{Username, Password}                │
│    │                                                         │
│    ├─► authService.Login(ctx, req.Username, req.Password)  │
│    │                                                         │
│    └─► Handle response                                      │
└──────────────────────┬───────────────────────────────────────┘
                       │
                       ▼
┌──────────────────────────────────────────────────────────────┐
│ 4. AuthService.Login() executes                              │
│    │                                                         │
│    ├─► validator.ValidateLogin()                            │
│    │   ├─► Check username not empty                         │
│    │   └─► Check password not empty                         │
│    │       └─► Return ValidationError if invalid            │
│    │                                                         │
│    ├─► userRepo.GetUserByUsername(ctx, username)           │
│    │   └─► Return user or ErrUserNotFound                  │
│    │                                                         │
│    ├─► auth.VerifyPassword(user.PasswordHash, password)    │
│    │   ├─► bcrypt.CompareHashAndPassword()                 │
│    │   └─► Return ErrInvalidCredentials if mismatch        │
│    │                                                         │
│    ├─► userRepo.UpdateLastLogin(ctx, user.ID)             │
│    │                                                         │
│    └─► Return *storage.User                                 │
└──────────────────────┬───────────────────────────────────────┘
                       │
                       ▼
┌──────────────────────────────────────────────────────────────┐
│ 5. Back in AuthHandlers.Login()                              │
│    │                                                         │
│    ├─► If error:                                            │
│    │   ├─► ValidationError? respondValidationError()       │
│    │   ├─► ErrInvalidCredentials? respondError(401)        │
│    │   └─► Other? respondError(500)                        │
│    │                                                         │
│    ├─► If success:                                          │
│    │   ├─► sessionStore.Get(r) → session                   │
│    │   ├─► sessionStore.SetUserSession(session, ...)       │
│    │   ├─► sessionStore.Save(r, w, session)                │
│    │   │   └─► Set-Cookie header in response              │
│    │   └─► respondJSON(UserResponse, 200)                  │
│    │                                                         │
│    └─► Write response to ResponseWriter                     │
└──────────────────────┬───────────────────────────────────────┘
                       │
                       ▼
┌──────────────────────────────────────────────────────────────┐
│ 6. Response sent to client                                   │
│    Status: 200 OK                                            │
│    Headers:                                                  │
│      Content-Type: application/json                         │
│      Set-Cookie: session_id=...; Path=/; HttpOnly; Secure   │
│    Body:                                                     │
│    {                                                         │
│      "id": "uuid-here",                                     │
│      "username": "john",                                    │
│      "email": "john@example.com",                           │
│      "roles": ["user"]                                      │
│    }                                                         │
└──────────────────────────────────────────────────────────────┘
                       │
                       ▼
┌──────────────────────────────────────────────────────────────┐
│ 7. Client stores session cookie                              │
│    Next requests include: Cookie: session_id=...            │
└──────────────────────────────────────────────────────────────┘
```

---

## 3. Dependency Injection Pattern

```
main.go
  │
  ├─► Create database pool
  │   └─► *pgxpool.Pool
  │
  ├─► Create Redis client
  │   └─► *redis.Client
  │
  ├─► Create repositories
  │   ├─► UserRepository(pool)
  │   ├─► DashboardRepository(pool)
  │   └─► ValidatorListRepository(pool)
  │
  ├─► Create session store
  │   └─► SessionStore(redisClient, secretKey, ...)
  │
  ├─► Create services
  │   └─► AuthService(userRepository)
  │
  ├─► Create handlers
  │   ├─► AuthHandlers(authService, sessionStore)
  │   ├─► DashboardHandler(dashboardService, healthMonitor)
  │   └─► SettingsHandler(userRepository, validator)
  │
  ├─► Create router with middleware
  │   └─► NewRouter(routerCfg)
  │
  ├─► Register routes (handlers depend on services/repos)
  │   └─► r.Post("/api/auth/login", authHandlers.Login)
  │
  └─► Start HTTP server
      └─► server.Start(ctx)
```

**Benefits**:
- Easy to test (mock dependencies)
- Loose coupling between layers
- Single responsibility
- Clear dependency graph

---

## 4. Error Handling Strategy

```
Service Layer
  │
  └─► Custom errors
       ├─► ErrUserAlreadyExists
       ├─► ErrInvalidCredentials
       ├─► ErrUserNotFound
       ├─► ValidationError (with field-level details)
       └─► fmt.Errorf (wrapped with context)

           │
           ▼
Handler Layer
  │
  ├─► Check error type
  │   ├─► ValidationError? 
  │   │   └─► respondValidationError(fields, 400)
  │   ├─► ErrInvalidCredentials?
  │   │   └─► respondError("Invalid credentials", 401)
  │   ├─► ErrUserAlreadyExists?
  │   │   └─► respondError("User already exists", 409)
  │   └─► Other error?
  │       └─► respondError("Internal server error", 500)
  │
  └─► Client receives HTTP response with status code + JSON error

HTTP Status Codes:
  200 OK              ✓ Request succeeded
  201 Created         ✓ Resource created
  400 Bad Request     ✗ Client validation error
  401 Unauthorized    ✗ Authentication failed
  403 Forbidden       ✗ Insufficient permissions
  404 Not Found       ✗ Resource not found
  409 Conflict        ✗ Duplicate resource
  500 Internal Error  ✗ Server error
```

---

## 5. Service Layer Pattern

```
AuthService
  │
  ├─► Register(ctx, username, password, email, roles)
  │   ├─ Input validation (via Validator)
  │   ├─ Check uniqueness (via UserRepository)
  │   ├─ Hash password (via auth.HashPassword)
  │   ├─ Create user (via UserRepository)
  │   └─ Return User or error
  │
  ├─► Login(ctx, username, password)
  │   ├─ Input validation
  │   ├─ Fetch user by username
  │   ├─ Verify password
  │   ├─ Update last login
  │   └─ Return User or error
  │
  ├─► GetUserByID(ctx, userID)
  │   └─ Delegate to UserRepository
  │
  └─► ChangePassword(ctx, userID, oldPassword, newPassword)
      ├─ Verify old password
      ├─ Hash new password
      ├─ Update in database
      └─ Return error or nil

Business Logic Separation:
  ✓ Service layer handles business rules
  ✓ Repository layer handles data access
  ✓ Handler layer handles HTTP concerns
  ✓ Each layer has single responsibility
```

---

## 6. Repository Pattern

```
UserRepository
  │
  ├─► CreateUser(ctx, username, email, passwordHash, roles)
  │   ├─ INSERT query
  │   ├─ RETURNING clause to get created user
  │   ├─ Error handling:
  │   │  └─ Unique constraint → ErrUserAlreadyExists
  │   └─ Return *User
  │
  ├─► GetUserByID(ctx, userID)
  │   ├─ SELECT query by ID
  │   ├─ WHERE is_active = true (soft delete)
  │   ├─ Error handling:
  │   │  └─ No rows → ErrUserNotFound
  │   └─ Return *User
  │
  ├─► GetUserByUsername(ctx, username)
  │   ├─ SELECT query by username
  │   ├─ WHERE is_active = true
  │   └─ Return *User
  │
  ├─► GetUserByEmail(ctx, email)
  │   ├─ SELECT query by email
  │   ├─ WHERE is_active = true
  │   └─ Return *User
  │
  ├─► UpdatePassword(ctx, userID, passwordHash)
  │   ├─ UPDATE query
  │   ├─ Check rows affected (must be > 0)
  │   └─ Return error or nil
  │
  ├─► UpdateUserRoles(ctx, userID, roles)
  │   ├─ UPDATE query (PostgreSQL array column)
  │   └─ Return error or nil
  │
  ├─► UpdateLastLogin(ctx, userID)
  │   ├─ UPDATE with NOW()
  │   └─ Return error or nil
  │
  ├─► DeactivateUser(ctx, userID)
  │   ├─ UPDATE is_active = false (soft delete)
  │   └─ Return error or nil
  │
  ├─► ListUsers(ctx, limit, offset)
  │   ├─ SELECT with pagination
  │   ├─ ORDER BY created_at DESC
  │   └─ Return []*User, error
  │
  └─► CountUsers(ctx)
      └─ SELECT COUNT(*), return int, error

Key Principles:
  ✓ All database access goes through repository
  ✓ Prepared statements with parameterized queries ($1, $2, ...)
  ✓ Transaction support via context
  ✓ Error type conversion (pgconn.PgError → domain errors)
  ✓ Soft delete support (is_active flag)
```

---

## 7. Middleware Chain Execution Order

```
Request arrives
  │
  ▼
1. Request ID Middleware (generates unique ID for tracing)
  │ ctx = context.WithValue(ctx, requestIDKey, "abc123")
  │ w.Header().Set("X-Request-ID", "abc123")
  │
  ▼
2. Real IP Middleware (extract true IP behind proxy)
  │ r.RemoteAddr = getClientIP(r)
  │
  ▼
3. Logging Middleware (log request start)
  │ logger.Info().Str("method", r.Method).Str("path", r.URL.Path).Msg("request start")
  │
  ▼
4. Panic Recovery Middleware (defer/recover)
  │ defer func() {
  │   if err := recover(); err != nil {
  │     logger.Error().Interface("panic", err).Msg("panic recovered")
  │     http.Error(w, "Internal Server Error", 500)
  │   }
  │ }()
  │
  ▼
5. Compression Middleware (gzip if Accept-Encoding header)
  │ Compress response body
  │
  ▼
6. Timeout Middleware (cancel context after 60s)
  │ ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
  │ defer cancel()
  │
  ▼
7. CORS Middleware (if enabled)
  │ Add Access-Control-Allow-* headers
  │ Handle OPTIONS preflight
  │
  ▼
8. Rate Limit Middleware (if enabled)
  │ Check requests per IP per second
  │ Return 429 Too Many Requests if exceeded
  │
  ▼
9. Security Headers Middleware
  │ X-Frame-Options: DENY
  │ X-Content-Type-Options: nosniff
  │ X-XSS-Protection: 1; mode=block
  │ Content-Security-Policy: ...
  │
  ▼
10. HTMX Detection Middleware
  │ Check HX-Request header
  │ Add to context for handlers
  │
  ▼
11. Optional: Session Middleware (for protected routes)
  │ r.Use(auth.SessionMiddleware(sessionStore))
  │ Extract session from cookie/Redis
  │ Add user ID to context
  │
  ▼
12. Optional: Require Auth (for protected routes)
  │ r.Use(auth.RequireSessionAuth)
  │ Check if user is in context
  │ Return 401 if not authenticated
  │
  ▼
13. Route Handler executes
  │ (e.g., AuthHandlers.Login)
  │
  ▼
Response sent back through middleware chain (in reverse order)
  │
  ▼
Client receives response
```

---

## 8. Context Flow in Handlers

```
func (h *AuthHandlers) Login(w http.ResponseWriter, r *http.Request) http.Handler {
    //
    // r.Context() contains:
    //
    // └─► requestIDKey: "abc123-def456"
    //     └─ Set by: Request ID Middleware
    //     └─ Access: appmiddleware.MustRequestIDFromContext(ctx)
    //
    // └─► remoteAddrKey: "192.168.1.1"
    //     └─ Set by: Real IP Middleware
    //
    // └─► userIDKey: uuid.UUID (if authenticated)
    //     └─ Set by: Session Middleware
    //     └─ Access: auth.GetSessionUserIDFromContext(ctx)
    //
    // └─► userClaimsKey: *Claims (if JWT authenticated)
    //     └─ Set by: JWT Middleware
    //     └─ Access: auth.GetUserClaims(ctx)
    //
    // └─► htmxRequestKey: true (if HTMX request)
    //     └─ Set by: HTMX Detection Middleware
    //
    
    // Service layer receives context
    user, err := h.authService.Login(r.Context(), username, password)
    
    // Service layer passes context to repository
    // Repository uses context for:
    //   - Query timeout
    //   - Cancellation support
    //   - Request tracing
    
    if err != nil {
        // Handle error...
    }
}
```

---

## 9. Test Structure

```
test/
├─ integration_test.go
│  ├─ TestAuthFlow (full request/response cycle)
│  ├─ TestDatabaseConnection
│  ├─ TestRedisConnection
│  └─ Uses testcontainers for PostgreSQL/Redis
│
└─ unit_test.go
   ├─ TestValidateLogin
   ├─ TestHashPassword
   ├─ TestVerifyPassword
   ├─ TestJWTToken
   └─ No external dependencies

Testing Pattern:
  ✓ Table-driven tests (test multiple inputs)
  ✓ Mocking dependencies via interfaces
  ✓ Fixtures for common test data
  ✓ Subtests for related test cases
  ✓ Cleanup via t.Cleanup()
  ✓ Parallel execution with t.Parallel()
```

---

## 10. Configuration Layers

```
Application Startup
  │
  ▼
1. Load .env file (via godotenv)
   ├─ development: .env (gitignored)
   └─ production: environment variables
  │
  ▼
2. Parse environment variables
   ├─ string: HTTP_PORT, DB_HOST, etc.
   ├─ int: DB_PORT, RATE_LIMIT_RPS, etc.
   ├─ bool: CORS_ENABLED, RATE_LIMIT_ENABLED, etc.
   ├─ duration: SESSION_MAX_AGE, JWT_ACCESS_TOKEN_TTL, etc.
   └─ slice: CORS_ALLOWED_ORIGINS (comma-separated)
  │
  ▼
3. Validate configuration
   ├─ Required fields present?
   ├─ Values in valid ranges?
   ├─ Consistency checks (e.g., min < max connections)
   └─ Environment-specific validation
  │
  ▼
4. Return Config struct
   ├─ Used throughout application
   └─ Immutable after startup

Config structure:
  Config
    ├─ Server
    │   ├─ HTTPPort
    │   ├─ GinMode (debug/release)
    │   ├─ CORSEnabled
    │   ├─ RateLimitRPS
    │   └─ ...
    ├─ Database
    │   ├─ Host, Port, User, Password
    │   ├─ SSLMode
    │   ├─ MaxConnections, MinConnections
    │   └─ ...
    ├─ Redis
    │   ├─ Addr, Password, DB
    │   └─ ...
    ├─ JWT
    │   ├─ SecretKey, Issuer
    │   ├─ AccessTokenDuration
    │   └─ RefreshTokenDuration
    ├─ Session
    │   ├─ SecretKey, MaxAge
    │   ├─ Secure, HttpOnly, SameSite
    │   └─ ...
    └─ Logging
        ├─ Level (debug/info/warn/error)
        ├─ Format (json/console)
        └─ ...
```

---

## 11. API Key Implementation Architecture (Recommended)

```
┌─ Database Layer
│  └─ api_keys table
│     ├─ id (UUID primary key)
│     ├─ user_id (UUID, FK to users)
│     ├─ key_hash (VARCHAR, bcrypt hash)
│     ├─ key_prefix (VARCHAR, "sk_ab12...")
│     ├─ name (VARCHAR, user-friendly)
│     ├─ scopes (TEXT[], ["read:validators", "write:alerts"])
│     ├─ is_active (BOOLEAN)
│     ├─ expires_at (TIMESTAMP, nullable)
│     ├─ last_used_at (TIMESTAMP, nullable)
│     ├─ created_at, updated_at
│     └─ indexes: user_id, key_hash, expires_at
│
├─ Repository Layer
│  └─ APIKeyRepository
│     ├─ CreateAPIKey(ctx, userID, name, scopes, expiresAt) → APIKey
│     ├─ GetAPIKeyByHash(ctx, keyHash) → APIKey (for validation)
│     ├─ GetAPIKeysByUserID(ctx, userID) → []*APIKey
│     ├─ DeleteAPIKey(ctx, keyID) → error
│     ├─ UpdateLastUsed(ctx, keyID) → error
│     └─ ListAPIKeys(ctx, userID, limit, offset) → []*APIKey, count
│
├─ Service Layer
│  └─ APIKeyService
│     ├─ GenerateAPIKey(ctx, userID, name, scopes, expiresAt)
│     │  └─ Returns: (fullKeyString, *APIKey, error)
│     │     └─ Only visible once!
│     ├─ ValidateAPIKey(ctx, keyString) → (*APIKey, error)
│     │  ├─ Hash provided key
│     │  ├─ Look up in database
│     │  ├─ Check is_active and not expired
│     │  ├─ Update last_used_at
│     │  └─ Return APIKey or error
│     ├─ RevokeAPIKey(ctx, userID, keyID) → error
│     ├─ ListUserAPIKeys(ctx, userID) → []*APIKey, error
│     └─ CheckKeyScopes(key *APIKey, required []string) → bool
│
├─ Middleware Layer
│  └─ APIKeyAuthMiddleware
│     ├─ Extract from X-API-Key or Authorization header
│     ├─ Call apiKeyService.ValidateAPIKey()
│     ├─ Add *APIKey to request context
│     ├─ Handle errors:
│     │  ├─ 400: Invalid format
│     │  ├─ 401: Not found / expired
│     │  ├─ 410: Revoked
│     │  └─ 403: Insufficient scopes
│     └─ Continue to next handler
│
├─ Handler Layer
│  └─ APIKeysHandler
│     ├─ POST /api/keys
│     │  ├─ Extract from authenticated user context
│     │  ├─ Call service.GenerateAPIKey()
│     │  ├─ Return {key: "...", apiKey: {id, name, ...}}
│     │  └─ Key only shown once
│     ├─ GET /api/keys
│     │  ├─ List user's API keys
│     │  ├─ Show prefix only, not full key
│     │  └─ Include: name, scopes, last_used_at, expires_at
│     ├─ GET /api/keys/{id}
│     │  └─ Get details of specific key
│     └─ DELETE /api/keys/{id}
│        └─ Revoke key (soft delete)
│
└─ Router Integration
   └─ r.Route("/api/keys", func(r chi.Router) {
        r.Use(auth.SessionMiddleware(sessionStore))
        r.Use(auth.RequireSessionAuth)
        r.Post("/", apiKeysHandler.GenerateAPIKey)
        r.Get("/", apiKeysHandler.ListAPIKeys)
        r.Get("/{id}", apiKeysHandler.GetAPIKeyDetails)
        r.Delete("/{id}", apiKeysHandler.RevokeAPIKey)
      })

Usage Flow:
  1. User: POST /api/keys → Create new key
  2. Server: Generate random 32-byte key
  3. Server: Hash with bcrypt
  4. Server: Store hash in database
  5. Server: Return full key (only time shown)
  6. User: Copy key, use in X-API-Key header
  7. Middleware: Hash provided key
  8. Middleware: Look up hash in database
  9. Middleware: Validate (active, not expired)
  10. Middleware: Add to context, update last_used_at
  11. Handler: Access via context
```

---

## 12. Database Connection Lifecycle

```
Application Start
  │
  ▼
1. Create database.Config
   ├─ Host, Port, User, Password
   ├─ SSL Mode
   ├─ Pool size (max, min)
   └─ Timeouts
  │
  ▼
2. Build connection string
   └─ postgres://user:pass@host:port/db?sslmode=...
  │
  ▼
3. Build pgxpool.Config
   ├─ BeforeConnect hook
   │  └─ Set application_name, statement_timeout, lock_timeout
   └─ AfterConnect hook
      └─ Prepare frequently used statements
  │
  ▼
4. Create connection pool
   ├─ NewWithConfig(ctx, poolConfig)
   ├─ Spawn minConnections immediately
   └─ Lazy-create up to maxConnections
  │
  ▼
5. Ping database to verify
   └─ Check SSL is in use (if configured)
  │
  ▼
6. Return *pgxpool.Pool to application
   ├─ Pass to repositories
   ├─ Pass to services (if needed)
   └─ Pass to handlers (if needed)
  │
  ▼
Application runs
  │
  ├─► All queries use pool.QueryRow() / pool.Query() / pool.Exec()
  │   └─ Automatically get/return connections from pool
  │   └─ Connection reused if available
  │   └─ New connection created if needed
  │
  └─► Connection management is automatic
      ├─ Idle timeout: Connection closed after 30min idle
      ├─ Max lifetime: Connection closed after 1 hour
      ├─ Health check: Periodic ping every 1 minute
      └─ All transparent to application code

Shutdown
  │
  ▼
1. Graceful shutdown signal
  │
  ▼
2. Stop accepting new requests
  │
  ▼
3. Wait for in-flight requests (30s timeout)
  │
  ▼
4. Close database pool
   ├─ pool.Close()
   ├─ Return all connections
   ├─ Close underlying TCP connections
   └─ No new queries allowed
  │
  ▼
5. Exit application
```

---

## 13. Session Management Flow

```
User Login Request
  │
  ▼
1. Handler receives request
   ├─ Validate credentials
   └─ Service.Login() succeeds → User returned
  │
  ▼
2. Create/Get session
   ├─ sessionStore.Get(r)
   │  └─ Check for existing session cookie
   │  └─ If no cookie: create empty session
   │  └─ If cookie: look up in Redis
   │
  ▼
3. Set user data in session
   ├─ sessionStore.SetUserSession(session, userID, username)
   │  └─ session.Values["user_id"] = userID
   │  └─ session.Values["username"] = username
   │
  ▼
4. Save session
   ├─ sessionStore.Save(r, w, session)
   │  ├─ Encode session.Values (JSON)
   │  ├─ Sign with SESSION_SECRET_KEY
   │  ├─ Store in Redis with TTL (e.g., 24 hours)
   │  └─ Set Set-Cookie header in response
   │      └─ Cookie name: "session_id"
   │      └─ Cookie value: signed session token
   │      └─ Secure: true/false (config)
   │      └─ HttpOnly: true (can't access from JS)
   │      └─ SameSite: Strict/Lax/None (config)
   │      └─ Max-Age: 86400 (24 hours, matches TTL)
   │
  ▼
5. Client receives response
   ├─ Stores session cookie automatically
   └─ Browser sends cookie in all requests to domain
  │
  ▼
Subsequent Requests with Session
  │
  ├─► Cookie included: Cookie: session_id=...
  │
  ├─► SessionMiddleware executes
  │   ├─ sessionStore.Get(r)
  │   │  ├─ Extract session_id from Cookie header
  │   │  ├─ Look up in Redis
  │   │  ├─ Verify signature
  │   │  ├─ Check not expired
  │   │  └─ Decode session.Values
  │   │
  │   └─ Add to request context
  │      └─ ctx.WithValue("user_id", userID)
  │
  ├─► Handler can access user:
  │   └─ userID, ok := auth.GetSessionUserIDFromContext(r.Context())
  │
  └─► Continue with authenticated request
  │
  ▼
Logout
  │
  ├─► sessionStore.Destroy(session)
  │   └─ Clear session.Values
  │
  ├─► sessionStore.Save(r, w, session)
  │   ├─ Delete from Redis
  │   └─ Set Set-Cookie with Max-Age: -1 (delete cookie)
  │
  └─► Client cookie is deleted

Session Expiry (automatic)
  │
  ├─► Configured in SESSION_MAX_AGE (default: 24h)
  │
  ├─► Redis key expires after TTL
  │
  ├─► Client cookie expires after Max-Age
  │
  └─► Next request fails with "no valid session" → 401 Unauthorized
```

---

## Summary: Clean Architecture Benefits

```
Benefits of this architecture:
  │
  ├─► Testability
  │   ├─ Mock repositories in service tests
  │   ├─ Mock services in handler tests
  │   └─ Integration tests with testcontainers
  │
  ├─► Maintainability
  │   ├─ Clear separation of concerns
  │   ├─ Easy to understand code flow
  │   └─ Easy to locate where logic happens
  │
  ├─► Scalability
  │   ├─ Add new features by creating new services/handlers
  │   ├─ Share services across multiple handlers
  │   └─ Share repositories across multiple services
  │
  ├─► Reusability
  │   ├─ Same service can serve REST + GraphQL
  │   ├─ Same middleware can protect REST + GraphQL
  │   └─ Services can be used in background jobs
  │
  └─► Error Handling
      ├─ Centralized custom errors
      ├─ Service errors → HTTP status codes in handler
      ├─ Validation errors with field-level details
      └─ Proper error context preservation
```

---

This architecture supports API key implementation seamlessly:
- New `api_keys` table in database
- New `APIKeyRepository` following same pattern
- New `APIKeyService` following same pattern
- New `APIKeyAuthMiddleware` following existing middleware pattern
- New `APIKeysHandler` following same handler pattern
- Register in router with same dependency injection pattern

All the infrastructure is already in place!
