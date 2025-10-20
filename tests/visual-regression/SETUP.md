# Visual Regression Testing Setup Guide

## Quick Start

### 1. Install Dependencies

```bash
# Install npm packages (Playwright, etc.)
npm install

# Install Playwright Chromium browser
npm run playwright:install
```

Or via Makefile:

```bash
make playwright-install
```

### 2. Start the Server

Visual tests require the server to be running at `http://localhost:8080`.

**Terminal 1 - Start server:**
```bash
make run
```

Or use development mode with hot-reload:
```bash
make dev
```

### 3. Run Visual Tests

**Terminal 2 - Run tests:**

```bash
# Run all visual regression tests
npm run test:visual
# or
make test-visual

# Run in UI mode (recommended for first time)
npm run test:visual:ui
# or
make test-visual-ui
```

## First-Time Setup Checklist

- [ ] Dependencies installed (`npm install`)
- [ ] Playwright browsers installed (`npm run playwright:install`)
- [ ] Server running at http://localhost:8080 (`make run`)
- [ ] CSS compiled (`make css-build`)
- [ ] Templ templates generated (`make templ-generate`)
- [ ] Visual tests run successfully (`make test-visual`)

## Expected Test Results (First Run)

On the **first run**, Playwright will:

1. **Capture baseline screenshots** for all navigation states:
   - Desktop navigation (1440px, 1920px)
   - Tablet navigation (768px) - closed and open menu
   - Mobile navigation (375px) - closed and open menu
   - Focus indicators
   - Accessibility validation

2. **Store baselines** in:
   ```
   tests/visual-regression/navigation.spec.ts-snapshots/
   ├── desktop-nav-default-desktop-chromium.png
   ├── desktop-nav-focus-desktop-chromium.png
   ├── tablet-nav-closed-tablet-chromium.png
   ├── tablet-nav-menu-open-tablet-chromium.png
   ├── mobile-nav-closed-mobile-chromium.png
   ├── mobile-nav-menu-open-mobile-chromium.png
   └── ... (more screenshots)
   ```

3. **All tests should PASS** on first run (baselines are created)

## Subsequent Runs

On subsequent runs, Playwright will:

1. Capture new screenshots
2. Compare against baselines
3. **FAIL** if differences exceed 5% threshold
4. Generate HTML report with visual diffs

## Viewing Test Results

After tests complete:

```bash
# Open HTML report in browser
open playwright-report/index.html
```

The report shows:
- ✅ Passed tests (screenshots match baselines)
- ❌ Failed tests (visual differences detected)
- 🔍 Visual diff viewer (expected vs actual vs diff)

## Troubleshooting

### Server Not Running

**Error:** `ECONNREFUSED` or timeout errors

**Solution:**
```bash
# Start server in separate terminal
make run
```

### Missing Baselines

**Error:** "Screenshot doesn't exist at..."

**Solution:**
```bash
# Generate baselines (first run)
npm run test:visual
```

### Tests Failing After UI Changes

**If you intentionally changed the UI:**

1. Review diffs in `playwright-report/index.html`
2. If changes look correct, update baselines:
   ```bash
   npm run test:visual:update
   ```
3. Commit new baselines:
   ```bash
   git add tests/visual-regression/
   git commit -m "test: update navigation visual baselines"
   ```

**If you didn't change the UI:**

- Investigate what caused the visual difference
- Check for:
  - CSS compilation issues
  - JavaScript errors
  - Network failures
  - Race conditions in rendering

### Browser Not Installed

**Error:** "Executable doesn't exist at..."

**Solution:**
```bash
npm run playwright:install
```

### Port 8080 Already in Use

**Error:** "EADDRINUSE: address already in use"

**Solution:**
```bash
# Find process using port 8080
lsof -ti:8080

# Kill the process
kill -9 $(lsof -ti:8080)
```

Or change the port in `playwright.config.ts`:
```typescript
baseURL: 'http://localhost:3000',
```

## CI/CD Integration

Visual regression tests run automatically in CI:

```bash
# .github/workflows/test.yml
- name: Install dependencies
  run: npm install

- name: Install Playwright browsers
  run: npm run playwright:install

- name: Build server
  run: make build

- name: Run visual regression tests
  run: npm run test:visual
  env:
    CI: true
```

## Next Steps

1. ✅ Setup complete? Run your first test:
   ```bash
   make test-visual
   ```

2. 📸 Review baselines:
   ```bash
   open tests/visual-regression/navigation.spec.ts-snapshots/
   ```

3. 📖 Learn more:
   - [README.md](README.md) - Full documentation
   - [Playwright Docs](https://playwright.dev/)
   - [Project CLAUDE.md](../../CLAUDE.md) - Visual verification standard

## Quick Reference

```bash
# Setup
npm install                    # Install deps
npm run playwright:install     # Install browsers

# Run tests
npm run test:visual           # Run all tests
npm run test:visual:ui        # Interactive UI mode
npm run test:visual:headed    # Watch browser execute tests

# Update baselines
npm run test:visual:update    # After intentional UI changes

# View results
open playwright-report/index.html
```
