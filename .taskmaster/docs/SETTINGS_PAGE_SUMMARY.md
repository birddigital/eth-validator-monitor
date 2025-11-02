# Settings Page - File Summary & Architecture

## Quick Overview

The settings page from task 27.1 provides a tabbed interface for user account management. It uses:
- **Backend:** Go with Chi router
- **Templates:** Templ framework (HTML templating)
- **Frontend:** HTMX for dynamic tab loading + DaisyUI styling
- **Auth:** Session-based (Redis-backed)
- **Database:** PostgreSQL with user table

---

## Files Created/Modified in Task 27.1

### Handler Files (Go)

#### 1. `internal/web/handlers/settings.go` - 51 lines
**Purpose:** Renders the main settings page with tab navigation

```go
type SettingsHandler struct {
    // Add dependencies as needed
}

func (h *SettingsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request)
```

**What it does:**
- Accepts GET /settings requests
- Extracts active tab from `?tab=` query parameter
- Validates tab name (whitelist: profile, notifications, api-keys, ui-preferences, 2fa, sessions, account)
- Gets username from session context
- Renders full page with layout wrapper

**Dependencies:**
- `auth.SessionUsernameKey` - context key for username
- `pages.SettingsPageWithLayout()` - templ component

---

#### 2. `internal/web/handlers/settings_content.go` - 60 lines
**Purpose:** HTMX endpoint for loading individual tab content

```go
type SettingsContentHandler struct {
    // Add dependencies as needed
}

func (h *SettingsContentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request)
```

**What it does:**
- Accepts GET /api/settings/content?tab=<name> requests
- Routes to appropriate tab component based on query parameter
- Renders HTML fragment (no layout) for HTMX to insert

**Tab Routing:**
```go
switch tab {
case "profile":
    err = components.SettingsProfileTab(username).Render(r.Context(), w)
case "notifications":
    err = components.SettingsNotificationsTab().Render(r.Context(), w)
// ... etc for all 7 tabs
}
```

---

### Template Files (Templ)

#### 3. `internal/web/templates/pages/settings.templ` - ~200 lines
**Purpose:** Main page structure with tab navigation

**Contains:**
- Page header ("Settings" title)
- Tab navigation bar with 7 tabs
- Content container with HTMX integration
- Skeleton loader while content loads

**Key Components:**
```templ
type SettingsPageData struct {
    ActiveTab string  // Which tab to highlight
    Username  string  // Current user's username
    UserEmail string  // User's email (placeholder)
}

templ SettingsPage(data SettingsPageData) { ... }
templ SettingsPageWithLayout(data SettingsPageData) { ... }
```

**HTML Structure:**
```html
<div class="container">
  <!-- Header -->
  <h1>Settings</h1>
  
  <!-- Tab Navigation -->
  <div class="tabs tabs-boxed">
    <a href="/settings?tab=profile" ...>Profile</a>
    <a href="/settings?tab=notifications" ...>Notifications</a>
    ...
  </div>
  
  <!-- Content Container -->
  <div id="settings-content">
    <div hx-get="/api/settings/content?tab=profile"
         hx-trigger="load"
         hx-swap="innerHTML transition:true">
      <!-- Skeleton loader here -->
    </div>
  </div>
</div>
```

---

#### 4. `internal/web/templates/components/settings_tabs.templ` - ~180 lines
**Purpose:** Tab content components (one per tab)

**Components:**
```templ
templ SettingsProfileTab(username string) { ... }
templ SettingsNotificationsTab() { ... }
templ SettingsAPIKeysTab() { ... }
templ SettingsUIPreferencesTab() { ... }
templ Settings2FATab() { ... }
templ SettingsSessionsTab() { ... }
templ SettingsAccountTab() { ... }
```

**Current Status:** All are placeholders with:
- Section header and description
- Info/warning alert with "Coming soon in subtask X.Y" message

**Example Structure:**
```templ
templ SettingsProfileTab(username string) {
  <div class="space-y-6">
    <div>
      <h2>Profile Information</h2>
      <p>Update your account profile information and password</p>
    </div>
    <div class="alert alert-info">
      Welcome, { username }! Profile management will be implemented in subtask 27.2
    </div>
  </div>
}
```

---

### Auto-Generated Template Files (DO NOT EDIT)

#### 5. `internal/web/templates/pages/settings_templ.go`
- Auto-generated from `settings.templ`
- Contains compiled templ runtime code
- Regenerate with: `templ generate ./...`

#### 6. `internal/web/templates/components/settings_tabs_templ.go`
- Auto-generated from `settings_tabs.templ`
- Contains compiled templ runtime code
- Regenerate with: `templ generate ./...`

---

## Existing Files Updated

### 1. `cmd/server/main.go` - ~20 lines added
**Location:** Lines ~500-530 in `registerRoutes()` function

**Changes:**
```go
// Initialize settings handlers
settingsHandler := handlers.NewSettingsHandler()
settingsContentHandler := handlers.NewSettingsContentHandler()

// Register settings routes (protected)
if sessionStore != nil {
    r.Group(func(r chi.Router) {
        r.Use(auth.SessionMiddleware(sessionStore))
        r.Use(auth.RequireSessionAuth)

        r.Get("/settings", settingsHandler.ServeHTTP)
        r.Get("/api/settings/content", settingsContentHandler.ServeHTTP)
    })
}
```

---

## Underlying Infrastructure (Pre-Existing)

### Authentication Layer

#### `internal/auth/session.go`
**SessionStore:**
- Redis-backed session storage
- Gorilla Sessions wrapper
- Methods: Get, Save, SetUserSession, GetUserID, GetUsername, Destroy
- Cookie name: `eth-validator-session`

#### `internal/auth/middleware_session.go`
**Middleware:**
- `SessionMiddleware(sessionStore)` - Optional, extracts session data
- `RequireSessionAuth` - Mandatory, enforces authentication
- Context keys: `SessionUserIDKey`, `SessionUsernameKey`

#### `internal/auth/service.go`
**Service:**
- `Register()` - Create new user with validation
- `Login()` - Authenticate by username
- `LoginByEmail()` - Authenticate by email
- `GetUserByID()` - Retrieve user from DB
- `ChangePassword()` - Not yet implemented

---

### User Model & Database

#### `internal/storage/user_repository.go`
**User Struct:**
```go
type User struct {
    ID           uuid.UUID
    Username     string
    Email        string
    PasswordHash string
    Roles        []string
    IsActive     bool
    CreatedAt    time.Time
    UpdatedAt    time.Time
    LastLogin    *time.Time
}
```

**Repository Methods:**
- CreateUser, GetUserByUsername, GetUserByEmail, GetUserByID
- UpdateLastLogin, UpdateUserRoles, DeactivateUser
- ListUsers, CountUsers

**Missing Methods (for future):**
- UpdateProfile() - update email, name, etc.
- UpdatePassword() - update password hash
- Update() - generic update method

---

## Database Schema

### Users Table
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

---

## File Dependency Graph

```
HTTP Request
    ↓
Chi Router
    ↓
SessionMiddleware (extracts session from Redis)
    ↓
RequireSessionAuth (checks user_id in context)
    ↓
SettingsHandler.ServeHTTP
    ├─ Gets username from SessionUsernameKey context
    ├─ Gets active tab from ?tab= query
    └─ Renders SettingsPageWithLayout(SettingsPageData)
         ↓
      settings_templ.go (compiled from settings.templ)
         ├─ Renders page header
         ├─ Renders tab navigation
         └─ Renders content container with HTMX
              ├─ hx-get="/api/settings/content?tab=profile"
              └─ hx-trigger="load"
                   ↓
            Browser executes HTMX
                   ↓
            GET /api/settings/content?tab=profile
                   ↓
            SettingsContentHandler.ServeHTTP
                ├─ Gets tab from query parameter
                └─ Routes to appropriate tab component
                     ├─ SettingsProfileTab(username)
                     ├─ SettingsNotificationsTab()
                     └─ ... (5 other tabs)
                          ↓
                    settings_tabs_templ.go (compiled)
                     (renders HTML fragment)
                          ↓
                    HTMX receives fragment
                     ↓
                    Swaps into #settings-content div
```

---

## Code Statistics

| File | Lines | Type | Status |
|------|-------|------|--------|
| settings.go | 51 | Go Handler | New |
| settings_content.go | 60 | Go Handler | New |
| settings.templ | 200 | Templ | New |
| settings_tabs.templ | 180 | Templ | New |
| settings_templ.go | 300+ | Go (auto) | Generated |
| settings_tabs_templ.go | 400+ | Go (auto) | Generated |
| main.go changes | ~20 | Go | Modified |
| **Total** | **~1,200+** | Mixed | Complete |

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                     Settings Page (Task 27.1)               │
└─────────────────────────────────────────────────────────────┘

┌─ HANDLERS ──────────────────────────────────────────────────┐
│                                                              │
│  SettingsHandler ────────┐                                  │
│  ├─ GET /settings        │ Renders full page with layout   │
│  └─ Returns: HTML        │ (settings_templ.go)             │
│                          │                                  │
│  SettingsContentHandler──┤                                  │
│  ├─ GET /api/settings/content?tab=X                         │
│  └─ Returns: HTML fragment (settings_tabs_templ.go)         │
│                                                              │
└─────────────────────────────────────────────────────────────┘

┌─ TEMPLATES ─────────────────────────────────────────────────┐
│                                                              │
│  settings.templ ────────────────────────────────────────    │
│  ├─ Page structure                                          │
│  ├─ Tab navigation (7 tabs)                                 │
│  └─ HTMX container (#settings-content)                      │
│                                                              │
│  settings_tabs.templ ───────────────────────────────────    │
│  ├─ SettingsProfileTab()                                    │
│  ├─ SettingsNotificationsTab()                              │
│  ├─ SettingsAPIKeysTab()                                    │
│  ├─ SettingsUIPreferencesTab()                              │
│  ├─ Settings2FATab()                                        │
│  ├─ SettingsSessionsTab()                                   │
│  └─ SettingsAccountTab()                                    │
│                                                              │
└─────────────────────────────────────────────────────────────┘

┌─ AUTHENTICATION ────────────────────────────────────────────┐
│                                                              │
│  SessionMiddleware ──────────────┐                          │
│  ├─ Extract session from Redis   │                          │
│  └─ Populate context             │                          │
│                                  ├─ Middleware stack        │
│  RequireSessionAuth ─────────────┤                          │
│  ├─ Check SessionUserIDKey       │                          │
│  └─ Return 401 if missing        │                          │
│                                  │                          │
│  SessionStore (Redis) ───────────┴─ Session storage         │
│  ├─ Get/Save sessions                                       │
│  └─ Key prefix: "session:"                                  │
│                                                              │
└─────────────────────────────────────────────────────────────┘

┌─ DATABASE ──────────────────────────────────────────────────┐
│                                                              │
│  users table                                                │
│  ├─ id (UUID)                                               │
│  ├─ username (unique)                                       │
│  ├─ email (unique)                                          │
│  ├─ password_hash                                           │
│  ├─ roles (array)                                           │
│  ├─ is_active                                               │
│  ├─ created_at, updated_at                                  │
│  └─ last_login                                              │
│                                                              │
│  UserRepository                                             │
│  ├─ CreateUser()                                            │
│  ├─ GetUserByUsername/Email/ID()                            │
│  ├─ UpdateLastLogin()                                       │
│  ├─ UpdateUserRoles()                                       │
│  ├─ DeactivateUser()                                        │
│  ├─ ListUsers()                                             │
│  └─ CountUsers()                                            │
│                                                              │
└─────────────────────────────────────────────────────────────┘

┌─ STYLING & FRONTEND ────────────────────────────────────────┐
│                                                              │
│  CSS Framework: DaisyUI + Tailwind                          │
│  Components Used:                                           │
│  ├─ tabs (tab navigation)                                   │
│  ├─ alert (placeholder messages)                            │
│  ├─ card (content sections)                                 │
│  └─ glass-card (modern styling)                             │
│                                                              │
│  Interactivity: HTMX                                        │
│  ├─ hx-get="/api/settings/content?tab=X"                    │
│  ├─ hx-trigger="load" (on page load)                        │
│  └─ hx-swap="innerHTML transition:true"                     │
│                                                              │
│  Accessibility:                                             │
│  ├─ ARIA roles (tab, tabpanel)                              │
│  ├─ aria-selected, aria-controls                            │
│  ├─ Semantic HTML                                           │
│  └─ Dark mode support (dark: classes)                       │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

---

## Flow: User Opens Settings Page

1. **User navigates to /settings**
   - Browser sends: `GET /settings`
   - Cookies include: `eth-validator-session=<session-id>`

2. **SessionMiddleware processes request**
   - Retrieves session from Redis using cookie
   - Extracts user_id and username
   - Adds to context: `SessionUserIDKey`, `SessionUsernameKey`

3. **RequireSessionAuth checks authentication**
   - Verifies `SessionUserIDKey` exists in context
   - Returns 401 if user not authenticated
   - Continues if authenticated

4. **SettingsHandler renders page**
   - Extracts `?tab=profile` from query (or defaults to "profile")
   - Validates tab name against whitelist
   - Gets username from context
   - Creates `SettingsPageData{ActiveTab: "profile", Username: "john"}`
   - Calls `SettingsPageWithLayout(data).Render(ctx, w)`

5. **Browser receives HTML**
   - Full page with layout, header, tab navigation
   - Content container has: `hx-get="/api/settings/content?tab=profile"` and `hx-trigger="load"`

6. **HTMX auto-triggers on page load**
   - JavaScript sends: `GET /api/settings/content?tab=profile`
   - Includes same session cookie

7. **SessionMiddleware + RequireSessionAuth again**
   - Authenticates the HTMX request
   - Adds user context

8. **SettingsContentHandler routes to tab component**
   - Gets `tab="profile"` from query
   - Renders `SettingsProfileTab(username)`
   - Returns HTML fragment

9. **HTMX receives fragment**
   - Swaps into `#settings-content` div
   - Applies transition effect
   - User sees "Welcome, john! Profile management coming in 27.2"

---

## Key Features Implemented

✅ **Tab Navigation**
- 7 tabs: Profile, Notifications, API Keys, UI, 2FA, Sessions, Account
- Query-parameter-based (clean URLs: /settings?tab=X)
- Active tab highlighted with CSS

✅ **HTMX Integration**
- Dynamic content loading without page reload
- Loading skeleton while fetching
- Smooth transitions between tabs

✅ **Authentication Protection**
- Both routes require active session
- SessionMiddleware + RequireSessionAuth
- 401 response for unauthenticated requests

✅ **Responsive Design**
- Tab labels hidden on mobile (sm: breakpoint)
- Icons visible on all screen sizes
- Flexbox layout adapts to viewport

✅ **Dark Mode Support**
- `dark:` CSS classes throughout
- Respects system preference
- No hard-coded colors

✅ **Accessibility**
- ARIA roles and labels
- Semantic HTML structure
- Keyboard navigation support
- Screen reader friendly

❌ **Placeholders (Awaiting Subtasks)**
- No actual form implementations
- No backend data processing
- No database persistence for settings
- Hardcoded email placeholder

---

## Next Steps (Future Subtasks)

1. **27.2 - Profile Management**
   - Form for updating username, email
   - Separate password change endpoint
   - Validation and error handling

2. **27.3 - 2FA Setup**
   - TOTP QR code generation
   - Backup codes display
   - Verification flow

3. **27.4 - API Key Management**
   - Generate, list, revoke keys
   - Show key only once
   - Expiration management

4. **27.5 - Notification Preferences**
   - Email alert toggles
   - Digest frequency selection
   - Alert type filtering

5. **27.6 - UI Preferences**
   - Theme selector (light/dark/auto)
   - Language selection
   - Items per page

6. **27.7 - Session Management**
   - List active sessions with device info
   - Logout from specific sessions
   - Last activity tracking

7. **27.8 - Account Deletion**
   - Password confirmation
   - Cascade delete all user data
   - Email notification

---

## Testing Checklist

- [ ] Navigate to /settings while logged in - should load full page
- [ ] Tab navigation works - clicking tabs changes ?tab= parameter
- [ ] HTMX content loads - tab content appears after initial load
- [ ] Unauthenticated access blocked - 401 when not logged in
- [ ] Session persists - can navigate away and return
- [ ] Dark mode works - toggle and verify styling
- [ ] Mobile responsive - test on small screens
- [ ] Keyboard navigation - tab through all interactive elements
- [ ] ARIA labels present - verify with accessibility tools
