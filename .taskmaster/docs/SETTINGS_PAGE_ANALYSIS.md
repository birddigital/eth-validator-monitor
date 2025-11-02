# Settings Page Analysis - Task 27.1

## Overview
Task 27.1 created the foundational settings page structure for the Ethereum Validator Monitor. The implementation uses a tab-based interface with HTMX for dynamic content loading.

---

## File Locations & Structure

### 1. Handler Files

#### `/Users/bird/sources/standalone-projects/eth-validator-monitor/internal/web/handlers/settings.go`
**Purpose:** Main settings page handler - renders the full settings page with tab navigation

**Key Components:**
- `SettingsHandler` struct - handles GET /settings requests
- `NewSettingsHandler()` - factory function
- `ServeHTTP(w http.ResponseWriter, r *http.Request)` - main handler
  - Extracts active tab from query parameter (`?tab=profile`)
  - Validates tab name against allowed list to prevent XSS
  - Gets user info from session context
  - Renders full page with layout

**Valid Tab Names:**
- `profile` (default)
- `notifications`
- `api-keys`
- `ui-preferences`
- `2fa`
- `sessions`
- `account`

**Current Limitations:**
- Email is hardcoded placeholder: `username + "@example.com"`
- No actual user data fetching from database
- Username extracted from session context (key: `SessionUsernameKey`)

#### `/Users/bird/sources/standalone-projects/eth-validator-monitor/internal/web/handlers/settings_content.go`
**Purpose:** HTMX endpoint handler - dynamically loads tab content without full page reload

**Key Components:**
- `SettingsContentHandler` struct - handles GET /api/settings/content requests
- `NewSettingsContentHandler()` - factory function
- `ServeHTTP(w http.ResponseWriter, r *http.Request)` - HTMX handler
  - Accepts `?tab=<name>` query parameter
  - Renders only the tab content component (no layout)
  - Returns HTML fragment for HTMX swap

**Routing:**
- Called via HTMX: `hx-get="/api/settings/content?tab=profile"`
- Swaps content with: `hx-swap="innerHTML transition:true"`
- Shows loading skeleton: `hx-indicator="#settings-skeleton"`

---

### 2. Template Files

#### Source Templates (`.templ` files - human-readable)

**`/Users/bird/sources/standalone-projects/eth-validator-monitor/internal/web/templates/pages/settings.templ`**
- Main page component
- Renders page header, tab navigation bar, and content container
- Uses HTMX to load content from `/api/settings/content`
- Contains `SettingsPageData` struct with fields:
  - `ActiveTab` (string) - currently selected tab
  - `Username` (string) - authenticated user's username
  - `UserEmail` (string) - user's email address
- Functions:
  - `SettingsPage(data SettingsPageData)` - page content without layout
  - `SettingsPageWithLayout(data SettingsPageData)` - with base layout wrapper
  - `getActiveTab(activeTab string)` - helper to default to "profile"

**`/Users/bird/sources/standalone-projects/eth-validator-monitor/internal/web/templates/components/settings_tabs.templ`**
- Tab content components for each section
- Functions (one per tab):
  - `SettingsProfileTab(username string)` - profile/password management placeholder
  - `SettingsNotificationsTab()` - notification preferences placeholder
  - `SettingsAPIKeysTab()` - API key management placeholder
  - `SettingsUIPreferencesTab()` - theme/UI customization placeholder
  - `Settings2FATab()` - two-factor authentication setup placeholder
  - `SettingsSessionsTab()` - active sessions management placeholder
  - `SettingsAccountTab()` - account deletion/settings placeholder

**Structure for Each Tab:**
```
<div class="space-y-6">
  <!-- Header section -->
  <div>
    <h2>Tab Title</h2>
    <p>Description</p>
  </div>
  
  <!-- Info/Warning Alert -->
  <div class="alert alert-info/alert-warning">
    <!-- Placeholder content -->
  </div>
</div>
```

#### Generated Templates (`.templ.go` files - auto-generated, don't edit)

**`/Users/bird/sources/standalone-projects/eth-validator-monitor/internal/web/templates/pages/settings_templ.go`**
- Auto-generated from `settings.templ`
- Contains compiled templ components
- File size: ~large (templ runtime overhead)
- **DO NOT EDIT** - regenerate with `templ generate ./...` if source changes

**`/Users/bird/sources/standalone-projects/eth-validator-monitor/internal/web/templates/components/settings_tabs_templ.go`**
- Auto-generated from `settings_tabs.templ`
- Contains compiled tab components
- **DO NOT EDIT** - regenerate with `templ generate ./...` if source changes

---

### 3. Routing Configuration

**File:** `/Users/bird/sources/standalone-projects/eth-validator-monitor/cmd/server/main.go`

**Routes Registered (lines ~500-530):**
```go
// Settings page routes (protected - requires authentication)
if sessionStore != nil {
    r.Group(func(r chi.Router) {
        r.Use(auth.SessionMiddleware(sessionStore))
        r.Use(auth.RequireSessionAuth)

        // Settings page route
        r.Get("/settings", settingsHandler.ServeHTTP)

        // Settings content API route (HTMX)
        r.Get("/api/settings/content", settingsContentHandler.ServeHTTP)
    })
}
```

**Route Details:**
- **GET /settings** - Full page render with layout (SettingsHandler)
- **GET /api/settings/content?tab=<name>** - Tab content fragment (SettingsContentHandler)
- **Middleware Stack:**
  1. `SessionMiddleware(sessionStore)` - extracts session data into context
  2. `RequireSessionAuth` - ensures user is authenticated
- **Authentication:** Both routes require valid session

---

## User Model & Database Schema

### User Model

**File:** `/Users/bird/sources/standalone-projects/eth-validator-monitor/internal/storage/user_repository.go`

**User Struct:**
```go
type User struct {
    ID           uuid.UUID  `db:"id"`
    Username     string     `db:"username"`
    Email        string     `db:"email"`
    PasswordHash string     `db:"password_hash"`
    Roles        []string   `db:"roles"`
    IsActive     bool       `db:"is_active"`
    CreatedAt    time.Time  `db:"created_at"`
    UpdatedAt    time.Time  `db:"updated_at"`
    LastLogin    *time.Time `db:"last_login"`
}
```

**User Fields:**
- **ID:** UUID primary key
- **Username:** String, unique, required
- **Email:** String, unique, required
- **PasswordHash:** bcrypt hash of password
- **Roles:** PostgreSQL array of role strings (e.g., `["user", "admin"]`)
- **IsActive:** Boolean (soft delete flag) - default true
- **CreatedAt:** Timestamp of account creation
- **UpdatedAt:** Timestamp of last update
- **LastLogin:** Nullable timestamp of last successful login

### User Repository Methods

**Location:** `/Users/bird/sources/standalone-projects/eth-validator-monitor/internal/storage/user_repository.go`

**Available Methods:**
1. `CreateUser(ctx, username, email, passwordHash, roles)` - creates new user
2. `GetUserByUsername(ctx, username)` - retrieves active user by username
3. `GetUserByEmail(ctx, email)` - retrieves active user by email
4. `GetUserByID(ctx, userID)` - retrieves active user by UUID
5. `UpdateLastLogin(ctx, userID)` - updates last_login timestamp
6. `UpdateUserRoles(ctx, userID, roles)` - updates roles array
7. `DeactivateUser(ctx, userID)` - soft delete (sets is_active=false)
8. `ListUsers(ctx, limit, offset)` - paginated user list
9. `CountUsers(ctx)` - total active user count

**Missing Methods (for future subtasks):**
- `UpdateUser()` - update profile fields (email, etc.)
- `UpdatePassword()` - update password hash
- `UpdateProfile()` - update email and other profile data

### Database Table: users

**Schema (from migrations):**
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    roles TEXT[] NOT NULL DEFAULT ARRAY['user'],
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    last_login TIMESTAMP NULL
);
```

**Indexes:**
- Primary key on `id`
- Unique constraints on `username` and `email`
- (Likely has index on `is_active` for fast active user queries)

---

## Authentication & Session Management

### Session Management

**File:** `/Users/bird/sources/standalone-projects/eth-validator-monitor/internal/auth/session.go`

**SessionStore Struct:**
- Wraps Gorilla Sessions with Redis backend
- Uses Redis for persistent session storage (key prefix: `session:`)
- Cookie name: `eth-validator-session`

**Session Methods:**
- `NewSessionStore()` - creates Redis-backed session store
- `Get(r *http.Request)` - retrieves session from request
- `Save(r, w, session)` - persists session to Redis
- `SetUserSession(session, userID, username)` - stores user info in session
- `GetUserID(session)` - retrieves user ID from session
- `GetUsername(session)` - retrieves username from session
- `Destroy(session)` - clears session on logout

**Session Cookie Configuration:**
- `MaxAge` - configurable session lifetime
- `HttpOnly` - prevents JavaScript access
- `Secure` - HTTPS only flag (configurable)
- `SameSite` - CSRF protection (Lax/Strict/None)
- `Path` - "/" (all routes)

**Session Data Stored in Redis:**
```
session:<random-id> = {
    user_id: "uuid-string",
    username: "username-string"
}
```

### Authentication Service

**File:** `/Users/bird/sources/standalone-projects/eth-validator-monitor/internal/auth/service.go`

**Service Methods:**
1. `Register(ctx, username, password, confirmPassword, email, roles)` - creates new user account
   - Validates all fields
   - Checks email/username uniqueness
   - Hashes password with bcrypt
   - Returns `*storage.User` or error
   
2. `Login(ctx, username, password)` - authenticates user
   - Validates credentials format
   - Fetches user by username
   - Verifies password against hash
   - Updates last_login timestamp
   - Returns `*storage.User` or `ErrInvalidCredentials`

3. `LoginByEmail(ctx, email, password)` - authenticates by email instead

4. `GetUserByID(ctx, userID)` - retrieves user from database

5. `ChangePassword(ctx, userID, oldPassword, newPassword)` - **NOT YET IMPLEMENTED**

### Session Middleware

**File:** `/Users/bird/sources/standalone-projects/eth-validator-monitor/internal/auth/middleware_session.go`

**Middleware Functions:**

1. **SessionMiddleware(sessionStore)** - Optional middleware
   - Extracts session data if present
   - Populates context with user info
   - Does NOT enforce authentication
   - Allows anonymous requests to continue

2. **RequireSessionAuth** - Mandatory middleware
   - Checks for `SessionUserIDKey` in context
   - Returns HTTP 401 Unauthorized if missing
   - Blocks unauthenticated requests

**Context Keys Used:**
- `SessionUserIDKey` - contains `uuid.UUID` of authenticated user
- `SessionUsernameKey` - contains string username

**Helper Functions:**
- `GetSessionUserIDFromContext(ctx)` - retrieves user ID from context
- `GetSessionUsernameFromContext(ctx)` - retrieves username from context

---

## Current Settings Page Structure

### Tab Layout

The settings page uses a **horizontal tab navigation** with 7 tabs:

```
[Profile] [Notifications] [API Keys] [UI] [2FA] [Sessions] [Account]
```

Each tab shows a specific section:

1. **Profile** - User profile information and password change (27.2)
2. **Notifications** - Email/alert notification preferences (27.5)
3. **API Keys** - API token generation and management (27.4)
4. **UI Preferences** - Theme, language, layout preferences (27.6)
5. **2FA** - Two-factor authentication setup (27.3)
6. **Sessions** - Active login sessions management (27.7)
7. **Account** - Account deletion and danger zone (27.8)

### Current Implementation Status

**COMPLETED (Task 27.1):**
- ✅ Page structure with tab navigation
- ✅ HTMX integration for dynamic tab loading
- ✅ Session-based authentication protection
- ✅ Skeleton loader while content loads
- ✅ Tab component stubs (placeholder alerts)
- ✅ Responsive tab layout (mobile-friendly)
- ✅ Dark mode support (CSS classes ready)
- ✅ Accessibility (ARIA labels, semantic HTML)

**PLACEHOLDERS (Awaiting Implementation):**
- ⚠️ All tab content sections show "Coming soon" alerts
- ⚠️ No actual form implementations
- ⚠️ No backend data persistence
- ⚠️ No CRUD operations for settings

**PENDING SUBTASKS:**
- 27.2 - Profile management (name, email update)
- 27.3 - 2FA setup (TOTP QR code)
- 27.4 - API key generation
- 27.5 - Notification preferences
- 27.6 - UI preferences (theme, etc.)
- 27.7 - Session management
- 27.8 - Account deletion

---

## Data Flow Diagram

```
User Request (GET /settings)
    ↓
SessionMiddleware
    ├─ Extracts session from Redis
    └─ Populates context with user_id, username
    ↓
RequireSessionAuth
    ├─ Checks SessionUserIDKey in context
    └─ Returns 401 if missing
    ↓
SettingsHandler.ServeHTTP
    ├─ Extracts tab from query (?tab=profile)
    ├─ Gets username from context
    ├─ Creates SettingsPageData struct
    └─ Renders SettingsPageWithLayout (templ)
    ↓
HTML Response (with HTMX directives)
    ├─ Tab navigation rendered
    └─ Content container with hx-get="/api/settings/content?tab=profile"
    ↓
Browser executes HTMX on load
    ├─ GET /api/settings/content?tab=profile
    └─ SessionMiddleware + RequireSessionAuth applied
    ↓
SettingsContentHandler.ServeHTTP
    ├─ Gets tab from query parameter
    ├─ Renders SettingsProfileTab component
    └─ Returns HTML fragment
    ↓
HTMX swaps content into #settings-content
    └─ User sees profile tab content
```

---

## Key Integration Points for Future Subtasks

### For Profile Management (27.2):
- **Handler:** Extend `SettingsHandler` with POST /settings/profile
- **Template:** Add form fields to `SettingsProfileTab`
- **Repository:** Add `UpdateProfile()` method to UserRepository
- **Validation:** Use existing `Validator` from auth package

### For 2FA Setup (27.3):
- **Packages:** Use `github.com/pquerna/otp/totp` for TOTP generation
- **Storage:** Add `user_2fa_secrets` table for backup codes
- **Handler:** New endpoints for QR code generation, verification
- **Template:** Add setup wizard component

### For API Keys (27.4):
- **Storage:** Create `api_keys` table with:
  - Key ID (UUID)
  - User ID (FK to users)
  - Key hash (bcrypt)
  - Last used timestamp
  - Expires at
- **Handler:** GET (list), POST (create), DELETE (revoke)
- **Template:** Key display (only once), expiration, usage stats

### For Notifications (27.5):
- **Storage:** Create `user_notification_preferences` table
- **Fields:** Email alerts, digest frequency, alert types
- **Handler:** GET (current preferences), POST (update)
- **Template:** Checkbox grid for notification types

### For UI Preferences (27.6):
- **Storage:** Add to users table or new `user_ui_preferences` table
- **Fields:** Theme (light/dark/auto), language, items per page
- **Handler:** GET, POST with persistence
- **Template:** Dropdown selectors, preview

### For Sessions (27.7):
- **Storage:** Add to session tracking (expand session.go)
- **Fields:** IP address, user agent, last activity, created at
- **Handler:** GET active sessions, POST /logout/{session-id}
- **Template:** Session table with device info, logout buttons

### For Account Deletion (27.8):
- **Handler:** POST /settings/account/delete with password verification
- **Cascade:** Delete all user-related data (sessions, API keys, preferences)
- **Notification:** Send confirmation email
- **Template:** Confirmation dialog with irreversible warning

---

## Code Generation & Build Notes

### Templ Compilation
The template files use **templ** framework with auto-code-generation.

**To regenerate templates after editing `.templ` files:**
```bash
cd /Users/bird/sources/standalone-projects/eth-validator-monitor
templ generate ./...
```

**Generated files:**
- `settings_templ.go` - compiled from `settings.templ`
- `settings_tabs_templ.go` - compiled from `settings_tabs.templ`

**DO NOT MANUALLY EDIT** `.templ.go` files - changes will be lost on regeneration.

### Routes Registration
Settings handlers are registered in `main.go` lines ~500-530 within:
```go
func registerRoutes(..., settingsHandler, settingsContentHandler, ...)
```

These handlers must be:
1. Instantiated in main()
2. Passed to registerRoutes()
3. Registered with appropriate middleware

---

## Testing Considerations

### Authentication Requirements
Both settings routes require active session:
```go
r.Use(auth.SessionMiddleware(sessionStore))
r.Use(auth.RequireSessionAuth)
```

**Test setup needed:**
1. Create test user via `authService.Register()`
2. Generate session via `sessionStore.SetUserSession()`
3. Set session cookie in test requests

### HTMX Testing
Settings content endpoint returns HTML fragments, not full pages.

**Test expectations:**
- GET /api/settings/content returns 200 with HTML
- No <!DOCTYPE>, <html>, or full page structure
- Content matches tab type requested

### Visual Verification (Playwright MCP)
Per project standards, UI testing requires:
- Screenshots at key interaction points
- Visual regression baselines
- Multi-viewport testing (mobile, tablet, desktop)
- Dark mode verification
- Accessibility validation

---

## Summary Table

| Component | File | Type | Status |
|-----------|------|------|--------|
| Page Handler | `settings.go` | Go | ✅ Complete |
| Content Handler | `settings_content.go` | Go | ✅ Complete |
| Page Template | `settings.templ` | Templ | ✅ Complete |
| Tab Components | `settings_tabs.templ` | Templ | ✅ Complete |
| Generated Page | `settings_templ.go` | Go (auto) | ✅ Generated |
| Generated Tabs | `settings_tabs_templ.go` | Go (auto) | ✅ Generated |
| User Model | `user_repository.go` | Go | ✅ Complete |
| Session Store | `session.go` | Go | ✅ Complete |
| Auth Service | `service.go` | Go | ✅ Complete |
| Middleware | `middleware_session.go` | Go | ✅ Complete |
| Routing | `main.go` (lines 500-530) | Go | ✅ Registered |
| Database | migrations (users table) | SQL | ✅ Migrated |

