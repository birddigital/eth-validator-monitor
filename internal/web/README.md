# Web UI with Templ Templates

This directory contains the web UI components built with [Templ](https://templ.guide), a type-safe templating language for Go.

## Directory Structure

```
internal/web/
├── README.md              # This file
├── generate.go            # go:generate directive for templ
├── handlers/              # HTTP request handlers
│   └── example.go         # Example handler using templ
└── templates/             # Templ template files
    ├── components/        # Reusable UI components
    │   ├── hello_world.templ
    │   └── hello_world_test.go
    ├── pages/             # Full page templates
    └── layouts/           # Layout wrappers
```

## Quick Start

### Install Dependencies

```bash
# Install all development tools including templ and air
make install-dev
```

### Development Workflow

```bash
# Generate templ templates (converts .templ files to Go code)
make templ-generate

# Start development server with hot-reload
make dev
```

The `make dev` command:
1. Generates templ templates
2. Starts `air` file watcher
3. Monitors `.templ` and `.go` files for changes
4. Automatically regenerates and rebuilds on file save

### Manual Commands

```bash
# Generate templates manually
go run github.com/a-h/templ/cmd/templ@latest generate

# Watch for changes (alternative to air)
make templ-watch

# Run tests
go test ./internal/web/...
```

## Writing Templ Components

### Basic Component

Create a new file `internal/web/templates/components/button.templ`:

```templ
package components

templ Button(text string, disabled bool) {
    <button class="btn" disabled?={ disabled }>
        { text }
    </button>
}
```

### Using Components in Handlers

```go
package handlers

import (
    "net/http"
    "github.com/birddigital/eth-validator-monitor/internal/web/templates/components"
)

func ButtonHandler(w http.ResponseWriter, r *http.Request) {
    component := components.Button("Click Me", false)

    w.Header().Set("Content-Type", "text/html; charset=utf-8")

    err := component.Render(r.Context(), w)
    if err != nil {
        http.Error(w, "Failed to render", http.StatusInternalServerError)
        return
    }
}
```

### Testing Components

```go
func TestButton(t *testing.T) {
    component := components.Button("Test", false)
    buf := new(bytes.Buffer)

    err := component.Render(context.Background(), buf)
    if err != nil {
        t.Fatal(err)
    }

    output := buf.String()
    if !strings.Contains(output, "Test") {
        t.Errorf("expected output to contain 'Test'")
    }
}
```

## HTMX Integration

Templ works seamlessly with HTMX for dynamic updates:

```templ
package components

templ ValidatorRow(validator Validator) {
    <tr
        hx-get={ fmt.Sprintf("/api/validators/%s", validator.ID) }
        hx-trigger="every 30s"
        hx-swap="outerHTML"
    >
        <td>{ validator.PublicKey }</td>
        <td>{ fmt.Sprintf("%.4f", validator.Effectiveness) }</td>
    </tr>
}
```

## Hot-Reload Configuration

The project uses Air for hot-reload during development. Configuration in `.air.toml`:

- Watches: `.go`, `.templ`, `.html` files
- Excludes: `*_templ.go` (generated files)
- Build command: `templ generate && go build`

## Best Practices

1. **Component Organization**
   - `components/`: Small, reusable UI pieces (buttons, cards, headers)
   - `pages/`: Complete page templates
   - `layouts/`: Base HTML structure and wrappers

2. **Type Safety**
   - Pass structs instead of primitives for complex data
   - Use Go types for validation at compile time

3. **Generated Files**
   - `*_templ.go` files are auto-generated
   - Added to `.gitignore` (can be committed for reproducible builds)
   - Regenerate with `make templ-generate`

4. **Error Handling**
   - Always check `component.Render()` errors
   - Set appropriate HTTP status codes
   - Log rendering failures for debugging

5. **Performance**
   - Templ generates zero-allocation code
   - Writes directly to `io.Writer`
   - No runtime template parsing

## CI/CD

Ensure CI pipeline generates templ files before building:

```yaml
- name: Generate templates
  run: make templ-generate

- name: Run tests
  run: go test -v ./...

- name: Build
  run: make build
```

## Resources

- [Templ Documentation](https://templ.guide)
- [HTMX Documentation](https://htmx.org)
- [Air Documentation](https://github.com/cosmtrek/air)
