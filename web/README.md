# Web Assets & TailwindCSS Setup

This directory contains static assets and TailwindCSS configuration for the Ethereum Validator Monitor web interface.

## Directory Structure

```
web/
├── static/
│   ├── css/
│   │   └── output.css        # Generated TailwindCSS + DaisyUI (gitignored)
│   └── js/
│       └── app.js             # HTMX configuration
├── styles/
│   └── input.css              # TailwindCSS source file
└── README.md                  # This file
```

## TailwindCSS + DaisyUI Configuration

### Setup Files

- `package.json` - Node.js dependencies (tailwindcss, daisyui)
- `tailwind.config.js` - Tailwind configuration with custom Ethereum theme colors
- `web/styles/input.css` - Tailwind directives and custom component classes

### Custom Theme

The light theme is customized with Ethereum brand colors:

```javascript
{
  primary: "#627eea",    // Ethereum blue
  secondary: "#454a75",  // Deep purple
  // ... other DaisyUI theme colors
}
```

### Available DaisyUI Components

- **Layout**: navbar, footer, drawer, menu
- **Navigation**: breadcrumbs, tabs, pagination
- **Forms**: input, textarea, select, checkbox, radio, toggle
- **Data Display**: card, badge, stat, table, avatar
- **Feedback**: alert, loading, modal, toast, progress
- **Actions**: button, dropdown, swap

See [DaisyUI Components](https://daisyui.com/components/) for full documentation.

## Development Workflow

### Initial Setup

```bash
# Install Node.js dependencies
npm install

# Install Go development tools (templ, air)
make install-dev

# Generate initial CSS
npm run css:build
```

### Development Mode (Recommended)

Run these commands in **separate terminals** for hot-reload:

```bash
# Terminal 1: Watch and rebuild CSS on changes
make css-dev

# Terminal 2: Watch templ files for changes
make templ-watch

# Terminal 3: Run server with hot-reload
air
```

**OR** use the single command (runs air + generates CSS once):

```bash
make dev
```

For full parallel watchers, see:

```bash
make dev-all  # Shows instructions for multi-terminal setup
```

### Production Build

```bash
# Build everything (CSS + templ + Go binaries)
make build
```

This will:
1. Generate minified CSS with PurgeCSS
2. Generate templ templates
3. Build server and CLI binaries

## CSS Build Commands

### Direct npm Commands

```bash
# Development: watch mode (unminified, fast rebuilds)
npm run css:dev

# Production: minified, optimized (removes unused classes)
npm run css:build
```

### Makefile Targets

```bash
make css-dev      # Start TailwindCSS watcher
make css-build    # Build production CSS
make css-clean    # Remove generated output.css
```

## VS Code Configuration

The `.vscode/` directory contains:

- **settings.json**: TailwindCSS IntelliSense for `.templ` files
- **extensions.json**: Recommended extensions (templ, tailwindcss, go)

### Features Enabled

- **TailwindCSS IntelliSense**: Autocomplete for Tailwind classes in templ files
- **Class Hover**: See CSS properties on hover
- **Color Previews**: Visual color swatches for color classes
- **Emmet Support**: HTML shortcuts in templ files

## Using Tailwind in templ Templates

### Basic Example

```go
package pages

templ HomePage() {
    <div class="container mx-auto px-4 py-8">
        <h1 class="text-3xl font-bold text-eth-primary">
            Ethereum Validator Monitor
        </h1>

        <div class="grid grid-cols-1 md:grid-cols-3 gap-4 mt-8">
            <div class="stat-card bg-base-100">
                <h2 class="text-lg font-semibold">Total Validators</h2>
                <p class="text-4xl font-bold text-primary">1,234</p>
            </div>
        </div>
    </div>
}
```

### DaisyUI Components

```go
templ DashboardCard(title string, value string) {
    <div class="card bg-base-100 shadow-xl">
        <div class="card-body">
            <h2 class="card-title">{title}</h2>
            <p class="text-3xl font-bold">{value}</p>
            <div class="card-actions justify-end">
                <button class="btn btn-primary btn-sm">View Details</button>
            </div>
        </div>
    </div>
}
```

### Custom Classes

Defined in `web/styles/input.css`:

```css
@layer components {
  .stat-card {
    @apply rounded-lg shadow-md p-6 hover:shadow-lg transition-shadow;
  }

  .nav-link {
    @apply transition-colors duration-200;
  }

  .page-container {
    @apply container mx-auto px-4 py-8;
  }
}
```

## Responsive Design

### Breakpoints (Tailwind defaults)

| Prefix | Min Width | Device          |
|--------|-----------|-----------------|
| sm:    | 640px     | Small tablets   |
| md:    | 768px     | Tablets         |
| lg:    | 1024px    | Laptops         |
| xl:    | 1280px    | Desktops        |
| 2xl:   | 1536px    | Large desktops  |

### Example

```go
<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
    <!-- 1 column on mobile, 2 on tablet, 4 on desktop -->
</div>
```

## Dark Mode

DaisyUI dark mode is configured but not yet implemented. To enable:

1. Add dark mode toggle button
2. Use JavaScript to set `data-theme="dark"` on `<html>` element
3. All colors will automatically switch to dark theme

Example toggle:

```javascript
// Toggle dark mode
document.documentElement.setAttribute('data-theme',
  isDark ? 'dark' : 'light'
);
```

## Performance Optimization

### Production CSS Size

- **Development**: ~50KB unminified
- **Production**: ~6.6KB minified with PurgeCSS

PurgeCSS automatically removes unused classes by scanning:
- `internal/web/**/*.templ`
- `internal/web/**/*.go`
- `web/**/*.html`

### Tips

1. **Use fewer custom classes**: Prefer Tailwind utility classes
2. **Avoid unused components**: Only use DaisyUI components you need
3. **Test builds**: Run `make build` to verify final CSS size

## Troubleshooting

### CSS not updating?

```bash
# Rebuild CSS manually
npm run css:build

# Check if CSS watcher is running
ps aux | grep tailwindcss
```

### Classes not found in CSS?

TailwindCSS JIT only includes classes **actually used** in templates. Check:

1. Class is spelled correctly in templ file
2. Template file is in content paths (see `tailwind.config.js`)
3. CSS was rebuilt after adding the class

### IntelliSense not working?

1. Install recommended VS Code extensions
2. Reload VS Code window: Cmd+Shift+P → "Reload Window"
3. Check `.vscode/settings.json` is configured

## Resources

- [TailwindCSS Docs](https://tailwindcss.com/docs)
- [DaisyUI Components](https://daisyui.com/components/)
- [templ Documentation](https://templ.guide/)
- [HTMX Documentation](https://htmx.org/)
