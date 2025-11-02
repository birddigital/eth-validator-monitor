# Settings Page Documentation - Task 27.1

This directory contains comprehensive documentation for the Settings Page feature (Task 27.1) of the Ethereum Validator Monitor.

## Quick Start

**Want to understand the settings page quickly?**
→ Read **SETTINGS_PAGE_SUMMARY.md** (5 min read)

**Need detailed technical information?**
→ Read **SETTINGS_PAGE_ANALYSIS.md** (15 min read)

**Looking for a specific file or method?**
→ Use **SETTINGS_QUICK_REFERENCE.md** (lookup guide)

**Need absolute file paths?**
→ Check **SETTINGS_FILES_REFERENCE.txt** (complete reference)

---

## Documentation Overview

### 1. SETTINGS_PAGE_ANALYSIS.md
**Comprehensive Technical Documentation** (4,500+ words)

Contains:
- Complete file-by-file breakdown with code snippets
- Handler architecture and implementation details
- Template structure and component hierarchy
- User model schema and database design
- Authentication and session management explained
- Data flow diagrams
- Integration points for each future subtask
- Testing considerations
- Troubleshooting guide

**Best for:** Deep understanding of how everything works together

### 2. SETTINGS_QUICK_REFERENCE.md
**Lookup & Reference Guide** (2,000+ words)

Contains:
- File locations in table format
- HTTP routes at a glance
- Tab components quick reference
- User model field reference
- Context keys and middleware
- Testing examples with curl commands
- Template regeneration guide
- Common errors and solutions
- Data structure examples

**Best for:** Quick lookups while coding

### 3. SETTINGS_PAGE_SUMMARY.md
**Architecture & Implementation Overview** (3,000+ words)

Contains:
- File dependency graph
- Code statistics and metrics
- Complete architecture diagram (ASCII art)
- Step-by-step request flow walkthrough
- Feature completeness checklist
- Step-by-step instructions for adding new features
- Testing checklist
- Next steps for each subtask

**Best for:** Understanding the overall design and architecture

### 4. SETTINGS_FILES_REFERENCE.txt
**Absolute File Paths Reference**

Contains:
- Complete paths for all 30+ relevant files
- File type and size information
- Directory structure tree
- Edit vs. Don't Edit guidance
- Quick command reference
- Regeneration instructions

**Best for:** Copy-paste file paths and directory navigation

---

## Files Created in Task 27.1

### Handler Files
- **internal/web/handlers/settings.go** (51 lines)
  - Renders main settings page with tab navigation
  - Accepts GET /settings requests
  - Protects with session authentication

- **internal/web/handlers/settings_content.go** (60 lines)
  - Serves HTML fragments for HTMX
  - Accepts GET /api/settings/content?tab=X requests
  - Routes to appropriate tab component

### Template Files (Source - Edit These)
- **internal/web/templates/pages/settings.templ** (~200 lines)
  - Main page structure with tab navigation
  - Contains SettingsPageData struct
  - HTMX integration for dynamic loading

- **internal/web/templates/components/settings_tabs.templ** (~180 lines)
  - 7 tab component functions
  - Currently placeholder alerts
  - Structure ready for real implementations

### Template Files (Generated - DO NOT EDIT)
- **internal/web/templates/pages/settings_templ.go**
  - Auto-generated from settings.templ
  - Regenerate with: `templ generate ./...`

- **internal/web/templates/components/settings_tabs_templ.go**
  - Auto-generated from settings_tabs.templ
  - Regenerate with: `templ generate ./...`

### Modified Files
- **cmd/server/main.go** (~20 lines added)
  - Settings routes registered in registerRoutes()
  - Handlers instantiated and configured
  - Middleware applied (SessionMiddleware + RequireSessionAuth)

---

## Architecture Summary

### Routes
```
GET /settings                   → Full page with layout
GET /api/settings/content       → HTML fragment for HTMX
```

### Authentication
- Protected by `SessionMiddleware` + `RequireSessionAuth`
- Returns 401 Unauthorized if not authenticated
- Context keys available: `SessionUserIDKey`, `SessionUsernameKey`

### Tab Navigation
7 tabs with placeholder content, implementing actual forms in future subtasks:
1. Profile (27.2)
2. Notifications (27.5)
3. API Keys (27.4)
4. UI Preferences (27.6)
5. 2FA (27.3)
6. Sessions (27.7)
7. Account (27.8)

### Frontend
- **Framework:** DaisyUI + Tailwind CSS
- **Interactivity:** HTMX for dynamic tab loading
- **State Management:** Query parameters (?tab=profile)
- **Accessibility:** ARIA labels, semantic HTML

### Database
- **Table:** PostgreSQL users
- **Fields:** id, username, email, password_hash, roles, is_active, created_at, updated_at, last_login
- **Repository:** 9 methods, missing: UpdateProfile, UpdatePassword

---

## Common Tasks

### Editing Templates
1. Edit `.templ` file (not `.templ.go`)
2. Run `templ generate ./...`
3. This regenerates `.templ.go` files
4. Commit both source and generated files

### Adding a New Settings Feature
1. Add form to appropriate tab in `settings_tabs.templ`
2. Create POST handler in `settings_content.go` (or new file)
3. Add POST endpoint in `main.go` routing
4. Add database method to UserRepository (if needed)
5. Run `templ generate ./...` after template changes

### Testing the Settings Page
```bash
# With valid session cookie
curl -b cookies.txt http://localhost:8080/settings

# Test specific tab
curl -b cookies.txt "http://localhost:8080/settings?tab=notifications"

# Without auth (should return 401)
curl http://localhost:8080/settings
```

### Regenerating Templates
```bash
cd /Users/bird/sources/standalone-projects/eth-validator-monitor
templ generate ./...
```

---

## User Model Reference

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

### Available Repository Methods
- `GetUserByID(ctx, userID)` - Get user by UUID
- `GetUserByUsername(ctx, username)` - Get user by username
- `GetUserByEmail(ctx, email)` - Get user by email
- `UpdateLastLogin(ctx, userID)` - Update last login time
- `UpdateUserRoles(ctx, userID, roles)` - Update user roles
- `DeactivateUser(ctx, userID)` - Soft delete user
- `ListUsers(ctx, limit, offset)` - Paginated user list
- `CountUsers(ctx)` - Count active users

### Missing Methods (Need to Implement)
- `UpdateProfile(ctx, userID, email, ...)` - Update profile fields
- `UpdatePassword(ctx, userID, newHash)` - Update password
- Other settings-specific update methods

---

## Context Keys Available in Handlers

After SessionMiddleware + RequireSessionAuth:

```go
// Get user ID from context
userID, ok := r.Context().Value(auth.SessionUserIDKey).(uuid.UUID)

// Get username from context
username, ok := r.Context().Value(auth.SessionUsernameKey).(string)
```

---

## Next Steps (Future Subtasks)

### Task 27.2 - Profile Management
- Update username, email, phone
- Change password
- Validation and error handling

### Task 27.3 - 2FA Setup
- TOTP QR code generation
- Backup codes display
- Setup wizard

### Task 27.4 - API Key Management
- Generate API keys
- List, revoke, rotate keys
- Expiration management

### Task 27.5 - Notification Preferences
- Email alert toggles
- Digest frequency
- Alert type filtering

### Task 27.6 - UI Preferences
- Theme selection (light/dark/auto)
- Language selection
- Layout customization

### Task 27.7 - Session Management
- List active sessions
- Show device/IP info
- Logout from specific sessions

### Task 27.8 - Account Deletion
- Password confirmation
- Cascade delete all data
- Confirmation email

---

## Project Structure

```
/Users/bird/sources/standalone-projects/eth-validator-monitor/

internal/
├── web/
│   ├── handlers/
│   │   ├── settings.go          [NEW]
│   │   └── settings_content.go  [NEW]
│   └── templates/
│       ├── pages/
│       │   ├── settings.templ      [NEW - Edit this]
│       │   └── settings_templ.go   [AUTO - Don't edit]
│       └── components/
│           ├── settings_tabs.templ [NEW - Edit this]
│           └── settings_tabs_templ.go [AUTO - Don't edit]
├── auth/
│   ├── session.go                (Session management)
│   ├── middleware_session.go      (Auth middleware)
│   └── service.go                 (Auth business logic)
├── storage/
│   └── user_repository.go         (User data access)
└── database/
    └── models/                    (Data models)

cmd/server/
└── main.go                        [MODIFIED - routes added]
```

---

## Documentation Files

All documentation is in: `.taskmaster/docs/`

- **README.md** (this file) - Overview and navigation
- **SETTINGS_PAGE_ANALYSIS.md** - Comprehensive technical guide
- **SETTINGS_QUICK_REFERENCE.md** - Quick lookup reference
- **SETTINGS_PAGE_SUMMARY.md** - Architecture and implementation
- **SETTINGS_FILES_REFERENCE.txt** - Absolute file paths

---

## Key Commands

### Run the server
```bash
cd /Users/bird/sources/standalone-projects/eth-validator-monitor
go run cmd/server/main.go
```

### Regenerate templates
```bash
cd /Users/bird/sources/standalone-projects/eth-validator-monitor
templ generate ./...
```

### Run tests
```bash
go test ./...
```

### Navigate to settings page
```
http://localhost:8080/settings          # Default (profile tab)
http://localhost:8080/settings?tab=2fa  # 2FA tab
```

---

## For More Information

- **Go Code Details:** See SETTINGS_PAGE_ANALYSIS.md
- **Quick Lookups:** See SETTINGS_QUICK_REFERENCE.md
- **Architecture:** See SETTINGS_PAGE_SUMMARY.md
- **File Paths:** See SETTINGS_FILES_REFERENCE.txt

---

## Questions?

Refer to the documentation sections above or check the appropriate reference document for:
- How something works → SETTINGS_PAGE_ANALYSIS.md
- Where a file is → SETTINGS_FILES_REFERENCE.txt
- How to test something → SETTINGS_QUICK_REFERENCE.md
- How to add a feature → SETTINGS_PAGE_SUMMARY.md
