# Visual Regression Testing

This directory contains Playwright-based visual regression tests for the Ethereum Validator Monitor web UI.

## Overview

Visual regression testing ensures UI consistency across different viewports and interactions. All screenshots are captured and compared against baselines to detect unintended visual changes.

## Test Structure

```
tests/visual-regression/
├── README.md                           # This file
├── navigation.spec.ts                  # Navigation component tests
└── baselines/                          # Screenshot baselines
    └── navigation/                     # Navigation-specific baselines
```

## Viewports Tested

All UI components are tested across these viewports:

- **Mobile**: 375px (iPhone SE)
- **Tablet**: 768px (iPad)
- **Desktop**: 1440px (standard laptop)
- **Wide**: 1920px (desktop monitor)

## Running Tests

### Prerequisites

```bash
# Install dependencies
npm install

# Install Playwright browsers (Chromium only for efficiency)
npm run playwright:install
```

### Test Commands

```bash
# Run all visual regression tests
npm run test:visual

# Run tests with UI mode (interactive)
npm run test:visual:ui

# Run tests in headed mode (see browser)
npm run test:visual:headed

# Update baseline screenshots (after intentional UI changes)
npm run test:visual:update
```

### Makefile Integration

```bash
# Run visual regression tests via Makefile
make test-visual

# Update baselines
make test-visual-update
```

## Writing New Visual Tests

### Test Template

```typescript
import { test, expect } from '@playwright/test';

test.describe('Component Name - Visual Regression', () => {
  test.describe('Desktop (1440px)', () => {
    test.use({ viewport: { width: 1440, height: 900 } });

    test('should render component correctly', async ({ page }) => {
      await page.goto('/page-url');
      await page.waitForLoadState('networkidle');

      // Capture baseline
      await expect(page.locator('.component-selector')).toHaveScreenshot('component-desktop.png', {
        threshold: 0.05, // 5% diff threshold
      });
    });
  });

  // Add mobile, tablet, wide viewport tests...
});
```

### Required Test Coverage

Every UI component MUST include:

1. **Screenshots at key interaction points**
   - Default state
   - Hover states
   - Active/pressed states
   - Error states
   - Empty states
   - Loading states

2. **Multi-viewport testing**
   - Test all 4 viewports (mobile, tablet, desktop, wide)

3. **Accessibility validation**
   - Focus indicators visible
   - Touch targets >= 44px on mobile
   - ARIA attributes present

4. **Layout stability**
   - No layout shift during load
   - No content jumps during interactions

5. **Error handling**
   - Graceful degradation without CSS
   - Network error states

## Baseline Management

### Updating Baselines

Baselines should only be updated when:

1. **Intentional UI changes** - You've deliberately modified the design
2. **Browser updates** - Playwright or browser versions changed rendering
3. **New features** - Adding new components or states

**NEVER update baselines to "fix" failing tests without visual inspection!**

### Update Process

```bash
# 1. Review what changed
npm run test:visual

# 2. Visually inspect the diff in playwright-report/index.html
open playwright-report/index.html

# 3. If changes are intentional, update baselines
npm run test:visual:update

# 4. Commit new baselines
git add tests/visual-regression/baselines/
git commit -m "test: update visual regression baselines for [reason]"
```

## Configuration

### Threshold Tuning

The default threshold is **5%** (0.05) for acceptable pixel differences. Adjust in tests:

```typescript
await expect(element).toHaveScreenshot('name.png', {
  threshold: 0.05, // 5% diff allowed
});
```

Use higher thresholds (0.1+) only for:
- Degraded/fallback states
- Dynamic content (animations, timestamps)
- Third-party widgets

### CI/CD Integration

Visual regression tests run automatically in CI:

```yaml
# .github/workflows/test.yml
- name: Run visual regression tests
  run: npm run test:visual
  env:
    CI: true
```

CI configuration differences:
- Tests run in headless mode
- Retries enabled (2 attempts)
- Single worker (no parallel execution)
- Strict baseline matching (no threshold tolerance)

## Debugging Failed Tests

### View Test Report

```bash
# Generate HTML report
npm run test:visual

# Open report in browser
open playwright-report/index.html
```

The report shows:
- Expected vs actual screenshots
- Pixel diff highlighting
- Test execution logs
- Network activity

### Common Failures

**Flaky tests due to timing:**
```typescript
// Add explicit waits for dynamic content
await page.waitForLoadState('networkidle');
await page.waitForSelector('.element', { state: 'visible' });
```

**Font rendering differences:**
```typescript
// Use higher threshold for text-heavy components
threshold: 0.08
```

**Animation-related failures:**
```typescript
// Disable animations in test
await page.addStyleTag({
  content: '* { animation: none !important; transition: none !important; }'
});
```

## Best Practices

### DO

✅ Wait for `networkidle` before capturing screenshots
✅ Use semantic selectors (`.navbar`, `button[aria-label="..."]`)
✅ Test all viewports for responsive components
✅ Document what each screenshot validates
✅ Keep threshold low (5%) for strict validation
✅ Test accessibility (focus, ARIA, touch targets)
✅ Test error states and edge cases
✅ Update baselines with clear commit messages

### DON'T

❌ Update baselines without visual review
❌ Use overly permissive thresholds (>10%)
❌ Skip viewport testing for responsive components
❌ Hardcode viewport sizes in tests (use `test.use()`)
❌ Capture screenshots of dynamic timestamps
❌ Test third-party iframes or external content
❌ Ignore accessibility validation
❌ Commit without running tests first

## Integration with Playwright MCP

This project uses **Playwright MCP** (Model Context Protocol) for AI-powered test generation and maintenance. The MCP server can:

- Generate test scripts from user descriptions
- Capture screenshots during sessions
- Auto-update baselines after approved UI changes
- Provide visual test coverage reports

See `.mcp.json` for Playwright MCP configuration.

## Troubleshooting

### "Browser not installed" error

```bash
npm run playwright:install
```

### Tests fail on CI but pass locally

Check for:
- Font differences (system fonts vs CI environment)
- Timezone rendering differences
- Screen resolution mismatches

Solution: Run tests in Docker locally to match CI environment.

### Baselines stored in Git LFS

If baseline PNGs are large, consider Git LFS:

```bash
git lfs track "tests/visual-regression/baselines/**/*.png"
```

## Related Documentation

- [Playwright Documentation](https://playwright.dev/)
- [Visual Regression Testing Guide](https://playwright.dev/docs/test-snapshots)
- [Project CLAUDE.md](../../CLAUDE.md) - Visual verification standard
- [Task Master AI Guide](../../.taskmaster/CLAUDE.md) - UI task requirements
