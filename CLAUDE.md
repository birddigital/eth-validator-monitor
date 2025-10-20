# Claude Code Instructions

## Project: Ethereum Validator Monitor

This is a **Go-based Ethereum validator monitoring system** with Prometheus metrics, GraphQL API, and PostgreSQL storage.

## Specialized Agents

### `/go-crypto` - Golang & Crypto Expert

**CRITICAL**: When working on Go code or Ethereum/validator-related features, **ALWAYS** consult the `/go-crypto` agent first.

Use `/go-crypto` for:
- Implementing Go code (validators, metrics, APIs, data collection)
- Ethereum/Beacon Chain integration and concepts
- Performance optimization and concurrency patterns
- Prometheus metrics design and implementation
- Testing strategies for blockchain data
- Database schema and query optimization
- Security considerations for crypto operations

**Example Usage:**
```
/go-crypto What's the best way to track validator effectiveness scores?
/go-crypto Review this goroutine for race conditions in the collector service
/go-crypto How should I structure Prometheus metrics for attestation rates?
```

**Autonomous Implementation Protocol:**

When working on Go code or Ethereum features:

**CRITICAL FOR NIGHTSHIFT/AUTONOMOUS MODE:**
- When `/nightshift` or autonomous orchestrator is active: **NEVER STOP** after receiving `/go-crypto` response
- Continue implementation **in the same response** without waiting for user confirmation
- Treat slash command responses as synchronous consultations, not conversation breaks

1. **Consultation Phase (Single Turn):**
   - Invoke `/go-crypto` before implementing Go code or crypto features
   - Receive the specialized agent's response with recommendations
   - **DO NOT STOP HERE** - Continue immediately to implementation

2. **Automatic Implementation Phase (Same Turn):**
   - **IMMEDIATELY** implement the `/go-crypto` agent's recommendations
   - **IN THE SAME RESPONSE** - Do not wait for user confirmation
   - Follow the recommended architecture, patterns, and code structure exactly
   - Apply all Go best practices and Ethereum domain knowledge from the response
   - Implement security patterns and performance optimizations as specified

3. **Implementation Requirements:**
   - Use the specialized agent's code examples as templates
   - Follow the file structure and naming conventions recommended
   - Implement all suggested benchmarks, tests, and helpers
   - Apply memory optimization tips and performance targets
   - Add Makefile targets and CI/CD integration as recommended

**Interactive Sessions Only:**
- In non-autonomous (interactive) sessions, you MAY pause after `/go-crypto` to confirm approach
- But in NightShift/autonomous mode: **CONSULTATION → IMPLEMENTATION** must be seamless and immediate

## Visual Verification Standard (Playwright MCP)

**MANDATORY for ALL UI-based tasks and subtasks:**

Every UI task MUST include comprehensive Playwright MCP visual verification requirements in the test strategy. This is non-negotiable for quality assurance.

### Required Visual Verification Components:

1. **Screenshot Capture Points:**
   - Capture screenshots at ALL key interaction points
   - Document before/after states for dynamic interactions
   - Capture all error states, empty states, and loading states
   - Screenshot HTMX partial updates to verify no layout shift

2. **Visual Regression Testing:**
   - Establish baseline screenshots on first implementation
   - Store baselines in `tests/visual-regression/baselines/`
   - Compare subsequent runs against baselines
   - Fail CI if visual diff > 5% (configurable threshold)
   - Require manual approval for baseline updates

3. **Multi-Viewport Testing:**
   - Mobile: 375px width (iPhone SE)
   - Tablet: 768px width (iPad)
   - Desktop: 1440px width (standard laptop)
   - Wide: 1920px width (desktop monitor)
   - Test ALL pages/components at ALL viewports

4. **Theme Testing:**
   - Screenshot light mode
   - Screenshot dark mode (if applicable)
   - Verify theme toggle persistence
   - Test high contrast mode (accessibility)

5. **Accessibility Visual Validation:**
   - Verify focus indicators are visible (keyboard navigation)
   - Check color contrast ratios meet WCAG 2.1 AA (4.5:1 minimum)
   - Screenshot error messages and validation feedback
   - Verify touch targets >= 44px on mobile

6. **HTMX-Specific Verification:**
   - Screenshot before HTMX request
   - Screenshot during loading (if indicator shown)
   - Screenshot after successful swap
   - Verify no layout shift during partial updates
   - Verify no content flashing or jumps

7. **Performance Visual Validation:**
   - Screenshot skeleton loaders/loading states
   - Verify lazy-loaded images render correctly
   - Capture Lighthouse performance report screenshots
   - Screenshot Core Web Vitals dashboard

### Implementation Requirements:

```javascript
// Example Playwright MCP test structure
test('Dashboard renders correctly', async ({ page }) => {
  // Navigate
  await page.goto('/dashboard');

  // Wait for stability
  await page.waitForLoadState('networkidle');

  // Capture baseline
  await page.screenshot({
    path: 'tests/visual-regression/baselines/dashboard-default.png',
    fullPage: true
  });

  // Test interaction
  await page.click('[data-testid="refresh-button"]');
  await page.waitForSelector('[data-loading="true"]');

  // Capture loading state
  await page.screenshot({
    path: 'tests/visual-regression/baselines/dashboard-loading.png'
  });

  // Verify final state
  await page.waitForSelector('[data-loading="false"]');
  const screenshot = await page.screenshot();
  expect(screenshot).toMatchSnapshot('dashboard-refreshed.png', {
    threshold: 0.05 // 5% diff threshold
  });
});
```

### Task Integration:

When creating or updating UI tasks, the test strategy MUST include:

```
Test Strategy:
[existing unit/integration tests...]

Visual Verification (Playwright MCP):
- Capture screenshots at key interaction points
- Establish visual regression baselines for comparison
- Test across viewports: mobile (375px), tablet (768px), desktop (1440px), wide (1920px)
- Verify dark mode rendering (if applicable)
- Test HTMX partial updates (ensure no layout shift or flashing)
- Validate accessibility: focus indicators, color contrast ratios
- Screenshot error states and empty states
- Verify loading states and skeleton screens render correctly
- Use Playwright MCP to navigate, interact, and capture all states
- Store screenshots in tests/visual-regression/baselines/
- Compare against baselines on subsequent runs (fail on >5% diff)

[Component-specific screenshots:]
- Screenshot: [specific state 1]
- Screenshot: [specific state 2]
- Verify: [specific visual behavior]
```

### Enforcement:

- ❌ Tasks without visual verification requirements will be **rejected**
- ✅ All UI tasks must pass visual regression tests before merging
- 📊 Visual regression test results must be included in PR reviews
- 🎯 Failed visual tests require investigation or baseline update approval

## Task Master AI Instructions
**Import Task Master's development workflow commands and guidelines, treat as if import is in the main CLAUDE.md file.**
@./.taskmaster/CLAUDE.md
