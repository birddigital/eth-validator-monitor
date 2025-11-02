# Authentication & Password Handling Implementation Guide

## Overview

The Ethereum Validator Monitor uses a **dual authentication system**:
1. **Session-based authentication** (HTTP forms/web UI) - Redis-backed with Gorilla Sessions
2. **JWT-based authentication** (GraphQL API) - Stateless token-based auth

## Password Hashing & Validation

### Password Hashing Method: **Bcrypt**

- **Location**: `/Users/bird/sources/standalone-projects/eth-validator-monitor/internal/auth/password.go`
- **Cost**: `12` (bcrypt cost factor - recommended for production)
- **Minimum Length**: `8` characters

#### Hash Function
```go
// HashPassword generates bcrypt hash of the password
func HashPassword(password string) (string, error) {
    if len(password) < MinPasswordLength {
        return "", ErrPasswordTooShort
    }
    hash, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
    if err != nil {
        return "", fmt.Errorf("failed to hash password: %w", err)
    }
    return string(hash), nil
}
```

#### Verify Function
```go
// VerifyPassword compares password with hash
func VerifyPassword(hashedPassword, password string) error {
    err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
    if err != nil {
        if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
            return ErrPasswordMismatch
        }
        return fmt.Errorf("password verification failed: %w", err)
    }
    return nil
}
```

### Password Validation

- **Location**: `/Users/bird/sources/standalone-projects/eth-validator-monitor/internal/auth/validator.go`

#### Strength Requirements
Password must contain:
- Minimum 8 characters
- At least 1 uppercase letter
- At least 1 lowercase letter
- At least 1 number
- At least 1 special character (punctuation or symbol)

#### Validation Method
```go
func (v *Validator) ValidatePassword(password string) error {
    // Checks: length, uppercase, lowercase, number, special char
    // Returns *ValidationError with field-specific messages
}
```

#### Other Validations
- **Email**: RFC-compliant format via regex
- **Username**: 3-255 characters, required
- **Registration**: Validates all fields + password match
- **Login**: Validates username and password presence

---

## Session Management (Web UI)

### Session Store Configuration

- **Location**: `/Users/bird/sources/standalone-projects/eth-validator-monitor/internal/auth/session.go`
- **Backend**: Redis (via `redisstore` package)
- **Cookie Name**: `eth-validator-session`
- **Store Type**: `gorilla/sessions.Store` (Redis-backed)

#### Session Initialization
```go
sessionStore, err := auth.NewSessionStore(
    redisClient,
    sessionSecret,
    maxAge,         // Expiration time in seconds
    secure,         // HTTPS only
    httpOnly,       // JavaScript cannot access
    sameSite,       // Cookie SameSite mode (Strict/Lax/None)
)
```

#### Session Structure
Stores two values:
- `user_id` (UUID as string)
- `username` (string)

#### Session Methods
```go
// Store user info in session
sessionStore.SetUserSession(session, userID, username)

// Retrieve user ID from session
userID, ok := sessionStore.GetUserID(session)

// Retrieve username from session
username, ok := sessionStore.GetUsername(session)

// Logout - invalidate session
sessionStore.Destroy(session)
```

### Session Middleware

- **Location**: `/Users/bird/sources/standalone-projects/eth-validator-monitor/internal/auth/middleware_session.go`

#### Context Keys
- `SessionUserIDKey` - User ID extracted from session
- `SessionUsernameKey` - Username extracted from session

#### Middleware Functions
```go
// Add session context to request
SessionMiddleware(sessionStore)

// Require session authentication (returns 401 if not authenticated)
RequireSessionAuth(next http.Handler)

// Get user ID from context
GetSessionUserIDFromContext(ctx) (uuid.UUID, bool)

// Get username from context
GetSessionUsernameFromContext(ctx) (string, bool)
```

---

## JWT Authentication (GraphQL API)

### JWT Service

- **Location**: `/Users/bird/sources/standalone-projects/eth-validator-monitor/internal/auth/jwt.go`
- **Algorithm**: HS256 (HMAC with SHA-256)
- **Duration Configuration**: Configurable access and refresh token durations

#### Token Claims
```go
type Claims struct {
    UserID   string   `json:"user_id"`
    Username string   `json:"username"`
    Roles    []string `json:"roles,omitempty"`
    jwt.RegisteredClaims // Standard JWT claims (exp, iat, nbf, iss, sub, jti)
}
```

#### JWT Generation
```go
// Access token (short-lived)
accessToken, err := jwtService.GenerateAccessToken(userID, username, roles)

// Refresh token (long-lived)
refreshToken, err := jwtService.GenerateRefreshToken(userID)
```

#### JWT Validation
```go
// Validate and parse token
claims, err := jwtService.ValidateToken(tokenString)
// Returns Claims or error (invalid, expired, or signature error)
```

#### Token Usage in Requests
```
Authorization: Bearer <jwt_token>
```

#### Context Integration
```go
// Add claims to context
ctx = auth.WithUserClaims(ctx, claims)

// Get claims from context
claims, ok := auth.GetUserClaims(ctx)

// Require auth (returns error if not authenticated)
claims, err := auth.RequireAuth(ctx)

// Check user role
hasRole := auth.HasRole(ctx, "admin")
```

### JWT Middleware

- **Location**: `/Users/bird/sources/standalone-projects/eth-validator-monitor/graph/middleware/auth.go`

#### Behavior
- Extracts token from `Authorization: Bearer <token>` header
- Validates token signature and expiration
- Adds claims to request context (unauthenticated requests continue)
- Invalid tokens logged and ignored (GraphQL resolvers handle auth)

---

## Authentication Service

- **Location**: `/Users/bird/sources/standalone-projects/eth-validator-monitor/internal/auth/service.go`

### Core Methods

#### Registration
```go
func (s *Service) Register(
    ctx context.Context,
    username, password, confirmPassword, email string,
    roles []string,
) (*storage.User, error) {
    // Validates all fields via Validator
    // Checks username/email uniqueness
    // Hashes password with HashPassword
    // Creates user in database
    // Default role: ["user"]
}
```

#### Login
```go
func (s *Service) Login(ctx context.Context, username, password string) (*storage.User, error) {
    // Validates inputs
    // Fetches user by username
    // Verifies password with VerifyPassword
    // Updates last_login timestamp
    // Returns user or ErrInvalidCredentials
}
```

#### Login by Email
```go
func (s *Service) LoginByEmail(ctx context.Context, email, password string) (*storage.User, error) {
    // Same as Login but uses email instead of username
}
```

#### Get User
```go
func (s *Service) GetUserByID(ctx context.Context, userID uuid.UUID) (*storage.User, error) {
    // Fetches user details from repository
}
```

#### Change Password
```go
func (s *Service) ChangePassword(
    ctx context.Context,
    userID uuid.UUID,
    oldPassword, newPassword string,
) error {
    // NOTE: Currently returns error - UpdatePassword not implemented in UserRepository
}
```

### Error Types
```go
ErrInvalidCredentials   // Invalid username/password
ErrUserAlreadyExists    // Duplicate email or username
ErrPasswordTooShort     // Password < 8 characters
```

---

## User Repository

- **Location**: `/Users/bird/sources/standalone-projects/eth-validator-monitor/internal/storage/user_repository.go`
- **Database**: PostgreSQL (pgx driver)

### User Model
```go
type User struct {
    ID           uuid.UUID  // Primary key
    Username     string     // Unique, required
    Email        string     // Unique, required
    PasswordHash string     // Bcrypt hash
    Roles        []string   // PostgreSQL array type
    IsActive     bool       // Soft-delete flag
    CreatedAt    time.Time
    UpdatedAt    time.Time
    LastLogin    *time.Time // Nullable, updated after login
}
```

### Key Methods

#### CreateUser
```go
func (r *UserRepository) CreateUser(
    ctx context.Context,
    username, email, passwordHash string,
    roles []string,
) (*User, error) {
    // Inserts new user
    // Returns ErrUserAlreadyExists on constraint violation
}
```

#### GetUserByUsername
```go
func (r *UserRepository) GetUserByUsername(ctx context.Context, username string) (*User, error) {
    // Retrieves active user (is_active = true)
    // Returns ErrUserNotFound if not found
}
```

#### GetUserByEmail
```go
func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
    // Retrieves active user by email
    // Returns ErrUserNotFound if not found
}
```

#### GetUserByID
```go
func (r *UserRepository) GetUserByID(ctx context.Context, userID uuid.UUID) (*User, error) {
    // Retrieves active user by UUID
    // Used by GetSessionUserIDFromContext, RequireAuth, etc.
}
```

#### UpdateLastLogin
```go
func (r *UserRepository) UpdateLastLogin(ctx context.Context, userID uuid.UUID) error {
    // Sets last_login = NOW()
}
```

#### UpdateUserRoles
```go
func (r *UserRepository) UpdateUserRoles(ctx context.Context, userID uuid.UUID, roles []string) error {
    // Updates roles array and updated_at
}
```

#### DeactivateUser
```go
func (r *UserRepository) DeactivateUser(ctx context.Context, userID uuid.UUID) error {
    // Soft-delete: sets is_active = false
}
```

#### Other Methods
- `ListUsers(limit, offset)` - Paginated list of active users
- `CountUsers()` - Total active user count

---

## Login Handlers (Web UI)

### Login POST Handler

- **Location**: `/Users/bird/sources/standalone-projects/eth-validator-monitor/internal/web/handlers/login_post.go`
- **Framework**: Formflow (declarative form handling)
- **Protocol**: HTTP POST with HTML forms + HTMX support

#### Flow
1. Validates email and password fields
2. Calls `authService.LoginByEmail(email, password)`
3. Creates session with `sessionStore.SetUserSession(userID, username)`
4. Saves session via `sessionStore.Save()`
5. Redirects to `/dashboard` or `?redirect=` query param

#### Error Handling
- Validation errors: Renders form with field errors
- Invalid credentials: Generic "Invalid email or password" message
- HTMX requests: Partial form re-render
- Traditional requests: Full page re-render

### Register POST Handler

- **Location**: `/Users/bird/sources/standalone-projects/eth-validator-monitor/internal/web/handlers/register_post.go`
- **Framework**: Formflow
- **Fields**: Email, username, password, password_confirm, terms checkbox

#### Flow
1. Validates all fields via Formflow + auth.Validator
2. Calls `authService.Register(username, password, passwordConfirm, email, ["user"])`
3. Creates session for new user
4. Saves session
5. Redirects to `/dashboard`

#### Password Field Validation
- Custom validator captures password for confirmation check
- Confirms passwords match before registration
- Auth service performs full strength validation

#### Error Handling
- Field-level errors from Validator
- Handles `ErrUserAlreadyExists` and `ErrPasswordTooShort`
- HTMX and traditional form support
- Password fields not repopulated in error responses

---

## Getting Current User in Handlers

### Session-Based (Web UI)

```go
// In handler with middleware
ctx := r.Context()

// Get user ID
userID, ok := auth.GetSessionUserIDFromContext(ctx)
if !ok {
    // Not authenticated
}

// Get username
username, ok := auth.GetSessionUsernameFromContext(ctx)
if !ok {
    // Not authenticated
}

// Fetch full user details
user, err := authService.GetUserByID(ctx, userID)
```

### JWT-Based (GraphQL API)

```go
// In GraphQL resolver
ctx := r.Context()

// Get claims
claims, ok := auth.GetUserClaims(ctx)
if !ok {
    // Not authenticated
}

userID := claims.UserID
username := claims.Username
roles := claims.Roles

// Fetch full user details
user, err := userRepo.GetUserByID(ctx, uuid.Parse(claims.UserID))

// Or require auth (returns error if not authenticated)
claims, err := auth.RequireAuth(ctx)
if err != nil {
    // Not authenticated
}
```

---

## API Endpoints

### HTTP Authentication (Web UI)

- **POST /login** - Session-based login (Formflow)
- **POST /register** - User registration (Formflow)
- **GET/POST /api/auth/login** - JSON API login
- **GET/POST /api/auth/register** - JSON API registration
- **POST /api/auth/logout** - Clear session
- **GET /api/auth/me** - Get current authenticated user (session)

### GraphQL Mutations

- **Register(input: RegisterInput!)** - Create user + JWT tokens
- **Login(input: LoginInput!)** - Authenticate + JWT tokens
- **RefreshToken(refreshToken: String!)** - Exchange refresh token for new access token

### GraphQL Queries

- **Me** - Get current authenticated user (requires JWT in header)

---

## Configuration

### Session Configuration
Set in application config:
- `SESSION_SECRET` - Secret key for cookie signing
- `SESSION_MAX_AGE` - Expiration in seconds (e.g., 86400 = 24 hours)
- `SESSION_SECURE` - HTTPS only (true in production)
- `SESSION_HTTP_ONLY` - Disable JavaScript access (recommended: true)
- `SESSION_SAME_SITE` - "Strict", "Lax", or "None" (default: "Lax")

### JWT Configuration
Set in application config:
- `JWT_SECRET` - Secret key for token signing
- `JWT_ISSUER` - Token issuer identifier
- `JWT_ACCESS_DURATION` - Access token expiration (e.g., 15m)
- `JWT_REFRESH_DURATION` - Refresh token expiration (e.g., 7 days)

---

## File Structure Summary

```
internal/auth/
├── password.go                 # Bcrypt hashing and verification
├── validator.go                # Input validation (password, email, username)
├── service.go                  # Authentication business logic (register, login)
├── session.go                  # Redis-backed session store (Gorilla Sessions)
├── middleware_session.go       # Session context middleware
└── jwt.go                      # JWT token generation and validation

internal/storage/
└── user_repository.go          # PostgreSQL user persistence

internal/web/handlers/
├── login_post.go               # HTTP login form handler (Formflow)
└── register_post.go            # HTTP registration form handler (Formflow)

internal/server/
└── auth_handlers.go            # JSON API handlers (/api/auth/*)

graph/
├── middleware/
│   └── auth.go                 # JWT middleware (GraphQL)
└── resolver/
    └── auth.resolvers.go       # GraphQL auth mutations and queries
```

---

## Security Best Practices Implemented

✅ **Password Hashing**: Bcrypt with cost=12 (slow and resistant to GPU attacks)
✅ **Password Validation**: Strong password requirements enforced
✅ **Session Cookies**: HttpOnly, Secure, SameSite flags
✅ **JWT Signature**: HS256 with configurable secret
✅ **Timing Attacks**: Bcrypt.CompareHashAndPassword is constant-time
✅ **Credential Privacy**: Error messages don't reveal if user exists
✅ **Last Login Tracking**: Timestamp updated on successful login
✅ **Soft Deletes**: Users deactivated via is_active flag
✅ **Token Expiration**: Configurable access and refresh token durations
✅ **Role-Based Access**: Roles stored in JWT claims and user record

---

## TODO / Future Improvements

- **ChangePassword**: Currently unimplemented - needs `UserRepository.UpdatePassword()` method
- **Password Reset**: Not yet implemented - would need email verification
- **Token Blacklist**: No logout tracking for JWT tokens
- **2FA**: Multi-factor authentication not implemented
- **Rate Limiting**: No protection against brute-force attacks
- **Audit Logging**: Login/logout/password change events not logged
