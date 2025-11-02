# Settings Page Quick Reference

## File Locations at a Glance

```
Project Root: /Users/bird/sources/standalone-projects/eth-validator-monitor/

Handlers:
├── internal/web/handlers/settings.go                    (Main page handler)
└── internal/web/handlers/settings_content.go            (HTMX tab content)

Templates (Source):
├── internal/web/templates/pages/settings.templ          (Page layout)
└── internal/web/templates/components/settings_tabs.templ (Tab components)

Templates (Generated - DO NOT EDIT):
├── internal/web/templates/pages/settings_templ.go       (Auto-generated)
└── internal/web/templates/components/settings_tabs_templ.go (Auto-generated)

Authentication:
├── internal/auth/session.go                             (Session store)
├── internal/auth/service.go                             (Auth service)
├── internal/auth/middleware_session.go                  (Session middleware)
└── internal/storage/user_repository.go                  (User database)

Routing:
└── cmd/server/main.go (lines ~500-530)                 (Route registration)
```

## HTTP Routes

| Route | Method | Handler | Protected | Purpose |
|-------|--------|---------|-----------|---------|
| `/settings` | GET | `SettingsHandler` | ✅ Yes | Full settings page with layout |
| `/api/settings/content` | GET | `SettingsContentHandler` | ✅ Yes | HTMX tab content fragment |

## Tab Components

| Tab | Component Function | Status | Subtask |
|-----|-------------------|--------|---------|
| Profile | `SettingsProfileTab(username)` | Placeholder | 27.2 |
| Notifications | `SettingsNotificationsTab()` | Placeholder | 27.5 |
| API Keys | `SettingsAPIKeysTab()` | Placeholder | 27.4 |
| UI Preferences | `SettingsUIPreferencesTab()` | Placeholder | 27.6 |
| 2FA | `Settings2FATab()` | Placeholder | 27.3 |
| Sessions | `SettingsSessionsTab()` | Placeholder | 27.7 |
| Account | `SettingsAccountTab()` | Placeholder | 27.8 |

## User Model Fields

```go
type User struct {
    ID           uuid.UUID   // Primary key
    Username     string      // Unique, required
    Email        string      // Unique, required
    PasswordHash string      // bcrypt hash
    Roles        []string    // e.g., ["user", "admin"]
    IsActive     bool        // Soft delete flag
    CreatedAt    time.Time   // Account creation
    UpdatedAt    time.Time   // Last update
    LastLogin    *time.Time  // Last successful login
}
```

## Session Context Keys

```go
// Available after SessionMiddleware + RequireSessionAuth
ctx.Value(auth.SessionUserIDKey)     // uuid.UUID
ctx.Value(auth.SessionUsernameKey)   // string
```

## Data Flow for Settings Page

```
GET /settings
  ↓
SessionMiddleware → extracts session from Redis
  ↓
RequireSessionAuth → checks user_id in context
  ↓
SettingsHandler.ServeHTTP
  ├─ Gets username from context
  ├─ Gets active tab from ?tab= query param
  └─ Renders SettingsPageWithLayout(data)
  ↓
HTML with HTMX: hx-get="/api/settings/content?tab=profile"
  ↓
Browser loads content via HTMX
  ↓
GET /api/settings/content?tab=profile
  ↓
SettingsContentHandler.ServeHTTP
  ├─ Gets tab from query
  └─ Renders tab component (e.g., SettingsProfileTab)
  ↓
HTML fragment returned, HTMX swaps into #settings-content
```

## Template Structure

### Main Page (`settings.templ`)
- **Type:** Full page component
- **Renders:** Header, tab navigation, content container
- **Data:** `SettingsPageData` struct
- **HTMX:** Loads tab content on page load

### Tab Components (`settings_tabs.templ`)
- **Type:** Content fragments (no layout)
- **Renders:** Title, description, placeholder alert
- **Current Status:** All are placeholder "coming soon" messages
- **Used By:** HTMX /api/settings/content endpoint

## Adding New Settings Features

### Step 1: Update User Model (if needed)
**File:** `internal/storage/user_repository.go`
- Add new fields to `User` struct
- Add new columns to `users` table migration
- Add repository methods for updates

### Step 2: Create Settings Handler
**File:** `internal/web/handlers/settings_content.go`
- Add new case in switch statement for new tab
- Example:
  ```go
  case "my-new-tab":
      err = components.SettingsMyNewTab().Render(r.Context(), w)
  ```

### Step 3: Create Tab Component
**File:** `internal/web/templates/components/settings_tabs.templ`
- Add new templ function: `templ SettingsMyNewTab() { ... }`
- Include form/inputs for collecting settings
- Regenerate: `templ generate ./...`

### Step 4: Register New Tab in Page Navigation
**File:** `internal/web/templates/pages/settings.templ`
- Add new tab link in the tabs container
- Set `data-tab="my-new-tab"`
- Include icon and label

### Step 5: Create POST Handler for Form Submission
**File:** Create new handler file or extend `settings_content.go`
- Handle POST /api/settings/my-new-tab
- Validate inputs
- Call repository to save to database
- Return success/error response

### Step 6: Update Main Routes
**File:** `cmd/server/main.go`
- Register new POST route in settings group
- Ensure middleware protection

## Testing Settings Routes

### Prerequisites
```bash
# Create test user
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "SecurePassword123!",
    "confirmPassword": "SecurePassword123!"
  }'

# Login to get session
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -c cookies.txt \
  -d '{
    "username": "testuser",
    "password": "SecurePassword123!"
  }'
```

### Test Page Load
```bash
curl -b cookies.txt http://localhost:8080/settings
# Should return full HTML page
```

### Test HTMX Endpoint
```bash
curl -b cookies.txt http://localhost:8080/api/settings/content?tab=profile
# Should return HTML fragment for profile tab
```

## Regenerating Templates

After editing any `.templ` files:

```bash
cd /Users/bird/sources/standalone-projects/eth-validator-monitor
templ generate ./...
```

This regenerates:
- `internal/web/templates/pages/settings_templ.go`
- `internal/web/templates/components/settings_tabs_templ.go`

**⚠️ DO NOT manually edit `.templ.go` files** - they will be overwritten.

## Key Integration Points

### For Profile Settings (27.2)
- Endpoint: POST /api/settings/profile
- Fields: username, email, phone (optional)
- Validation: Email uniqueness check
- Repository: `UpdateProfile()` method needed
- Password: Separate endpoint (/api/settings/password)

### For 2FA (27.3)
- Endpoint: POST /api/settings/2fa/setup
- Response: QR code + backup codes
- Storage: New `user_2fa_secrets` table
- Validation: TOTP verification code

### For API Keys (27.4)
- Endpoints: GET, POST, DELETE
- Storage: `api_keys` table
- Fields: key_id, key_hash, expires_at
- Display: Only show once on creation

### For Notifications (27.5)
- Endpoint: POST /api/settings/notifications
- Storage: `user_notification_preferences` table
- Fields: email_alerts, frequency, types

### For UI Preferences (27.6)
- Endpoint: POST /api/settings/preferences
- Fields: theme (light/dark), language, items_per_page
- Storage: users table or separate table

### For Sessions (27.7)
- Endpoint: GET /api/settings/sessions, POST logout/{id}
- Data: IP, user agent, last activity
- Storage: Extend Redis session tracking

### For Account Deletion (27.8)
- Endpoint: POST /api/settings/account/delete
- Validation: Password confirmation
- Cascade: Delete all user data
- Email: Confirmation notification

## Styling

Settings page uses:
- **Framework:** DaisyUI (Tailwind CSS components)
- **Colors:** Dark mode support via `dark:` classes
- **Layout:** Flexbox for tab navigation
- **Components:** Glass-morphism cards (`.glass-card`)
- **Icons:** Inline SVG (Heroicons style)

## Accessibility Features

- **ARIA labels:** `role="tab"`, `aria-selected`, `aria-controls`
- **Semantic HTML:** `<nav>`, proper heading hierarchy
- **Keyboard:** Tab navigation supported
- **Screen readers:** Proper label association
- **Color contrast:** WCAG 2.1 AA compliant

## Database Schema

### Users Table
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY,
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

### Future Tables (for subtasks)
- `api_keys` - for API key management (27.4)
- `user_2fa_secrets` - for 2FA setup (27.3)
- `user_notification_preferences` - for notification settings (27.5)
- `user_ui_preferences` - for UI customization (27.6)
- `user_sessions` - for session tracking (27.7)

## Common Errors & Solutions

### 401 Unauthorized on /settings
- Session not set in cookie
- Session has expired (check Redis TTL)
- SessionMiddleware not applied to route

### No content appears in tab
- HTMX request to /api/settings/content failing
- Check browser console for network errors
- Verify handler is returning 200 status

### Templ compilation errors
- Run `templ fmt ./...` to fix formatting
- Run `templ generate ./...` to compile
- Check for syntax errors in `.templ` files

### User data not loading
- Username hardcoded as placeholder in handler
- Need to implement `GetUserByID()` call in handler
- Requires additional user repository queries

## Performance Considerations

- **Session store:** Redis for fast lookups
- **Page load:** Initial render is full HTML (no SSR)
- **Tab switching:** HTMX fragments are lightweight
- **Caching:** Consider caching static tab templates
- **Database:** Add indexes on username, email for login queries
