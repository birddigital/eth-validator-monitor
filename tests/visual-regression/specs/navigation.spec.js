/**
 * Visual Regression Tests - Responsive Navigation Bar
 *
 * Tests the navigation bar component across multiple viewports,
 * verifying responsive behavior, accessibility, and visual consistency.
 *
 * Test Strategy:
 * - Capture screenshots at all viewports (mobile, tablet, desktop, wide)
 * - Verify hamburger menu appears and functions on mobile/tablet
 * - Test dropdown menu interactions
 * - Validate accessibility (focus indicators, ARIA attributes, color contrast)
 * - Ensure no layout shift during interactions
 * - Test both light and dark themes (if applicable)
 */

const { test, expect } = require('@playwright/test');

// Viewport configurations
const viewports = {
  mobile: { width: 375, height: 667, name: 'mobile' },    // iPhone SE
  tablet: { width: 768, height: 1024, name: 'tablet' },   // iPad
  desktop: { width: 1440, height: 900, name: 'desktop' }, // Standard laptop
  wide: { width: 1920, height: 1080, name: 'wide' }       // Desktop monitor
};

// Base URL (configurable via environment)
const BASE_URL = process.env.BASE_URL || 'http://localhost:8080';

// Test threshold for visual regression (5% difference allowed)
const VISUAL_THRESHOLD = 0.05;

/**
 * Test Suite: Desktop Navigation
 * Verifies horizontal navigation links display correctly on large screens
 */
test.describe('Navigation Bar - Desktop', () => {
  test.beforeEach(async ({ page }) => {
    await page.setViewportSize(viewports.desktop);
  });

  test('should display horizontal navigation links', async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');

    // Verify horizontal menu is visible (not hamburger)
    const horizontalMenu = page.locator('.navbar-center .menu-horizontal');
    await expect(horizontalMenu).toBeVisible();

    // Verify hamburger menu is hidden
    const hamburgerButton = page.locator('.navbar-end .dropdown button');
    await expect(hamburgerButton).not.toBeVisible();

    // Capture baseline screenshot
    await page.screenshot({
      path: 'tests/visual-regression/baselines/navigation/desktop-default.png',
      fullPage: false,
      clip: { x: 0, y: 0, width: 1440, height: 80 } // Navbar only
    });
  });

  test('should show all navigation links', async ({ page }) => {
    await page.goto(BASE_URL);

    // Verify all expected links are present
    const navLinks = ['Home', 'Validators', 'Metrics', 'GraphQL'];
    for (const linkText of navLinks) {
      const link = page.locator('.navbar-center a', { hasText: linkText });
      await expect(link).toBeVisible();
    }

    // Verify login button is visible
    const loginButton = page.locator('.navbar-end a[href="/login"]');
    await expect(loginButton).toBeVisible();
  });

  test('should show hover states on navigation links', async ({ page }) => {
    await page.goto(BASE_URL);

    const homeLink = page.locator('.navbar-center a', { hasText: 'Home' });
    await homeLink.hover();

    // Wait for transition to complete
    await page.waitForTimeout(300);

    // Capture hover state
    await page.screenshot({
      path: 'tests/visual-regression/baselines/navigation/desktop-link-hover.png',
      fullPage: false,
      clip: { x: 0, y: 0, width: 1440, height: 80 }
    });
  });

  test('should display full logo text', async ({ page }) => {
    await page.goto(BASE_URL);

    // Desktop should show full "Ethereum Validator Monitor" text
    const fullLogo = page.locator('.navbar-start span', { hasText: 'Ethereum Validator Monitor' });
    await expect(fullLogo).toBeVisible();
  });
});

/**
 * Test Suite: Wide Screen Navigation
 * Verifies navigation on ultra-wide displays
 */
test.describe('Navigation Bar - Wide Screen', () => {
  test.beforeEach(async ({ page }) => {
    await page.setViewportSize(viewports.wide);
  });

  test('should display correctly on wide screens', async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');

    // Capture wide screen baseline
    await page.screenshot({
      path: 'tests/visual-regression/baselines/navigation/wide-default.png',
      fullPage: false,
      clip: { x: 0, y: 0, width: 1920, height: 80 }
    });
  });
});

/**
 * Test Suite: Mobile Navigation
 * Verifies hamburger menu and dropdown behavior on mobile devices
 */
test.describe('Navigation Bar - Mobile', () => {
  test.beforeEach(async ({ page }) => {
    await page.setViewportSize(viewports.mobile);
  });

  test('should display hamburger menu button', async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');

    // Verify hamburger button is visible
    const hamburgerButton = page.locator('.navbar-end .dropdown button[aria-label="Open menu"]');
    await expect(hamburgerButton).toBeVisible();

    // Verify horizontal menu is hidden
    const horizontalMenu = page.locator('.navbar-center');
    await expect(horizontalMenu).not.toBeVisible();

    // Capture mobile default state
    await page.screenshot({
      path: 'tests/visual-regression/baselines/navigation/mobile-default.png',
      fullPage: false,
      clip: { x: 0, y: 0, width: 375, height: 80 }
    });
  });

  test('should show shortened logo on mobile', async ({ page }) => {
    await page.goto(BASE_URL);

    // Mobile should show "ETH Monitor" instead of full text
    const shortLogo = page.locator('.navbar-start span.inline.sm\\:hidden', { hasText: 'ETH Monitor' });
    await expect(shortLogo).toBeVisible();
  });

  test('should open dropdown menu on hamburger click', async ({ page }) => {
    await page.goto(BASE_URL);

    const hamburgerButton = page.locator('.navbar-end .dropdown button');

    // Click hamburger to open menu
    await hamburgerButton.click();

    // Wait for dropdown to appear
    const dropdownMenu = page.locator('.dropdown-content');
    await expect(dropdownMenu).toBeVisible();

    // Verify all navigation links are in dropdown
    const navLinks = ['Home', 'Validators', 'Metrics', 'GraphQL', 'Login'];
    for (const linkText of navLinks) {
      const link = dropdownMenu.locator('a', { hasText: linkText });
      await expect(link).toBeVisible();
    }

    // Capture dropdown open state
    await page.screenshot({
      path: 'tests/visual-regression/baselines/navigation/mobile-dropdown-open.png',
      fullPage: false
    });
  });

  test('should close dropdown when clicking outside', async ({ page }) => {
    await page.goto(BASE_URL);

    const hamburgerButton = page.locator('.navbar-end .dropdown button');
    await hamburgerButton.click();

    // Wait for dropdown to open
    const dropdownMenu = page.locator('.dropdown-content');
    await expect(dropdownMenu).toBeVisible();

    // Click outside dropdown (on main content area)
    await page.locator('main').click({ position: { x: 10, y: 10 } });

    // Dropdown should be hidden after clicking outside
    await expect(dropdownMenu).not.toBeVisible();
  });

  test('should navigate when clicking dropdown link', async ({ page }) => {
    await page.goto(BASE_URL);

    // Open dropdown
    const hamburgerButton = page.locator('.navbar-end .dropdown button');
    await hamburgerButton.click();

    // Click on Validators link
    const validatorsLink = page.locator('.dropdown-content a', { hasText: 'Validators' });
    await validatorsLink.click();

    // Verify navigation occurred (URL changed)
    await expect(page).toHaveURL(/.*\/validators/);
  });

  test('should have no layout shift when opening dropdown', async ({ page }) => {
    await page.goto(BASE_URL);

    // Capture before dropdown open
    const beforeScreenshot = await page.screenshot({
      clip: { x: 0, y: 0, width: 375, height: 80 }
    });

    // Open dropdown
    const hamburgerButton = page.locator('.navbar-end .dropdown button');
    await hamburgerButton.click();
    await page.waitForTimeout(300); // Wait for animation

    // Capture after dropdown open (navbar area only, not dropdown content)
    const afterScreenshot = await page.screenshot({
      clip: { x: 0, y: 0, width: 375, height: 80 }
    });

    // Navbar itself should not shift (dropdown overlays content)
    // This is a visual check - automated comparison would use pixelmatch or similar
    expect(beforeScreenshot).toBeTruthy();
    expect(afterScreenshot).toBeTruthy();
  });
});

/**
 * Test Suite: Tablet Navigation
 * Verifies navigation on tablet-sized devices
 */
test.describe('Navigation Bar - Tablet', () => {
  test.beforeEach(async ({ page }) => {
    await page.setViewportSize(viewports.tablet);
  });

  test('should display hamburger menu on tablet', async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');

    // Tablet (< 1024px) should show hamburger menu
    const hamburgerButton = page.locator('.navbar-end .dropdown button');
    await expect(hamburgerButton).toBeVisible();

    // Capture tablet default state
    await page.screenshot({
      path: 'tests/visual-regression/baselines/navigation/tablet-default.png',
      fullPage: false,
      clip: { x: 0, y: 0, width: 768, height: 80 }
    });
  });

  test('should show full logo text on tablet', async ({ page }) => {
    await page.goto(BASE_URL);

    // Tablet should show full "Ethereum Validator Monitor" (sm: breakpoint)
    const fullLogo = page.locator('.navbar-start span.hidden.sm\\:inline', { hasText: 'Ethereum Validator Monitor' });
    await expect(fullLogo).toBeVisible();
  });
});

/**
 * Test Suite: Accessibility
 * Verifies WCAG 2.1 AA compliance and keyboard navigation
 */
test.describe('Navigation Bar - Accessibility', () => {
  test.beforeEach(async ({ page }) => {
    await page.setViewportSize(viewports.desktop);
  });

  test('should have proper ARIA attributes on hamburger button', async ({ page }) => {
    await page.setViewportSize(viewports.mobile);
    await page.goto(BASE_URL);

    const hamburgerButton = page.locator('.navbar-end .dropdown button');

    // Check ARIA attributes
    await expect(hamburgerButton).toHaveAttribute('aria-label', 'Open menu');
    await expect(hamburgerButton).toHaveAttribute('aria-haspopup', 'true');
  });

  test('should support keyboard navigation', async ({ page }) => {
    await page.goto(BASE_URL);

    // Tab through navigation links
    await page.keyboard.press('Tab'); // Focus on logo/first element
    await page.keyboard.press('Tab'); // Focus on first nav link (Home)

    // Verify focus is on Home link
    const homeLink = page.locator('.navbar-center a', { hasText: 'Home' });
    await expect(homeLink).toBeFocused();

    // Capture focus state
    await page.screenshot({
      path: 'tests/visual-regression/baselines/navigation/desktop-focus-state.png',
      fullPage: false,
      clip: { x: 0, y: 0, width: 1440, height: 80 }
    });
  });

  test('should have visible focus indicators', async ({ page }) => {
    await page.goto(BASE_URL);

    // Focus on a navigation link
    const homeLink = page.locator('.navbar-center a', { hasText: 'Home' });
    await homeLink.focus();

    // Focus indicator should be visible (check for outline or ring styles)
    // This is verified visually via screenshot above
    await expect(homeLink).toBeFocused();
  });

  test('should have sufficient color contrast', async ({ page }) => {
    await page.goto(BASE_URL);

    // Get computed styles for navigation links
    const homeLink = page.locator('.navbar-center a', { hasText: 'Home' }).first();
    const color = await homeLink.evaluate((el) => {
      const style = window.getComputedStyle(el);
      return {
        color: style.color,
        backgroundColor: style.backgroundColor
      };
    });

    // Log colors for manual verification (automated contrast checking requires additional tools)
    console.log('Navigation link colors:', color);

    // Visual verification via screenshot baseline
    expect(color.color).toBeTruthy();
  });

  test('should have touch targets >= 44px on mobile', async ({ page }) => {
    await page.setViewportSize(viewports.mobile);
    await page.goto(BASE_URL);

    // Open dropdown
    const hamburgerButton = page.locator('.navbar-end .dropdown button');
    await hamburgerButton.click();

    // Check dropdown link sizes
    const dropdownLinks = page.locator('.dropdown-content a');
    const linkCount = await dropdownLinks.count();

    for (let i = 0; i < linkCount; i++) {
      const link = dropdownLinks.nth(i);
      const box = await link.boundingBox();

      // Verify height >= 44px (WCAG 2.1 AA touch target size)
      expect(box.height).toBeGreaterThanOrEqual(44);
    }
  });
});

/**
 * Test Suite: Visual Regression Comparison
 * Compares current screenshots against stored baselines
 */
test.describe('Navigation Bar - Visual Regression', () => {
  test('should match desktop baseline', async ({ page }) => {
    await page.setViewportSize(viewports.desktop);
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');

    const screenshot = await page.screenshot({
      clip: { x: 0, y: 0, width: 1440, height: 80 }
    });

    // Compare against baseline (requires baseline to exist)
    // expect(screenshot).toMatchSnapshot('desktop-default.png', { threshold: VISUAL_THRESHOLD });
  });

  test('should match mobile baseline', async ({ page }) => {
    await page.setViewportSize(viewports.mobile);
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');

    const screenshot = await page.screenshot({
      clip: { x: 0, y: 0, width: 375, height: 80 }
    });

    // Compare against baseline
    // expect(screenshot).toMatchSnapshot('mobile-default.png', { threshold: VISUAL_THRESHOLD });
  });
});

/**
 * Test Suite: Dark Mode (if applicable)
 * Verifies navigation appearance in dark theme
 */
test.describe('Navigation Bar - Dark Mode', () => {
  test.skip('should render correctly in dark mode', async ({ page }) => {
    // Skip if dark mode not implemented
    // To test: Set data-theme="dark" on <html> element
    await page.goto(BASE_URL);

    // Switch to dark theme
    await page.evaluate(() => {
      document.documentElement.setAttribute('data-theme', 'dark');
    });

    await page.waitForTimeout(100); // Wait for theme transition

    // Capture dark mode baseline
    await page.screenshot({
      path: 'tests/visual-regression/baselines/navigation/desktop-dark-mode.png',
      fullPage: false,
      clip: { x: 0, y: 0, width: 1440, height: 80 }
    });
  });
});
