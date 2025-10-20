import { test, expect } from '@playwright/test';

/**
 * Visual Regression Tests for Responsive Navigation Bar
 *
 * Tests the navigation component across multiple viewports:
 * - Mobile: 375px (iPhone SE)
 * - Tablet: 768px (iPad)
 * - Desktop: 1440px (standard laptop)
 * - Wide: 1920px (desktop monitor)
 *
 * Validates:
 * - Desktop navigation bar with inline links
 * - Mobile hamburger menu appearance and functionality
 * - Responsive logo text switching
 * - Dropdown menu interaction on mobile
 * - Visual consistency and no layout shift
 * - Accessibility features (focus indicators, ARIA labels)
 */

test.describe('Navigation Bar - Visual Regression', () => {
  test.describe('Desktop Navigation (1440px+)', () => {
    test.use({ viewport: { width: 1440, height: 900 } });

    test('should render desktop navigation with inline links', async ({ page }) => {
      await page.goto('/');
      await page.waitForLoadState('networkidle');

      // Screenshot: Default state
      await expect(page.locator('nav.navbar')).toHaveScreenshot('desktop-nav-default.png', {
        threshold: 0.05, // 5% diff threshold
      });
    });

    test('should show focus indicators on keyboard navigation', async ({ page }) => {
      await page.goto('/');
      await page.waitForLoadState('networkidle');

      // Tab through navigation links
      await page.keyboard.press('Tab');
      await page.keyboard.press('Tab'); // Focus on first nav link

      // Screenshot: Focus indicator visible
      await expect(page.locator('nav.navbar')).toHaveScreenshot('desktop-nav-focus.png', {
        threshold: 0.05,
      });
    });

    test('should hide hamburger menu on desktop', async ({ page }) => {
      await page.goto('/');
      await page.waitForLoadState('networkidle');

      // Verify hamburger menu is not visible
      const hamburger = page.locator('.navbar-end.lg\\:hidden');
      await expect(hamburger).not.toBeInViewport();
    });

    test('should show full logo text', async ({ page }) => {
      await page.goto('/');
      await page.waitForLoadState('networkidle');

      // Verify full logo text visible
      const fullLogo = page.locator('text=Ethereum Validator Monitor');
      await expect(fullLogo).toBeVisible();

      // Verify short logo is hidden
      const shortLogo = page.locator('text=ETH Monitor').first();
      await expect(shortLogo).toBeHidden();
    });
  });

  test.describe('Wide Viewport Navigation (1920px)', () => {
    test.use({ viewport: { width: 1920, height: 1080 } });

    test('should render navigation on wide screens', async ({ page }) => {
      await page.goto('/');
      await page.waitForLoadState('networkidle');

      await expect(page.locator('nav.navbar')).toHaveScreenshot('wide-nav-default.png', {
        threshold: 0.05,
      });
    });
  });

  test.describe('Tablet Navigation (768px)', () => {
    test.use({ viewport: { width: 768, height: 1024 } });

    test('should show hamburger menu on tablet', async ({ page }) => {
      await page.goto('/');
      await page.waitForLoadState('networkidle');

      // Screenshot: Tablet navigation (closed menu)
      await expect(page.locator('nav.navbar')).toHaveScreenshot('tablet-nav-closed.png', {
        threshold: 0.05,
      });

      // Verify hamburger is visible
      const hamburger = page.locator('button[aria-label="Open menu"]');
      await expect(hamburger).toBeVisible();
    });

    test('should open dropdown menu on hamburger click', async ({ page }) => {
      await page.goto('/');
      await page.waitForLoadState('networkidle');

      // Click hamburger menu
      const hamburger = page.locator('button[aria-label="Open menu"]');
      await hamburger.click();

      // Wait for dropdown to appear
      await page.waitForSelector('.dropdown-content', { state: 'visible' });

      // Screenshot: Dropdown menu open
      await expect(page).toHaveScreenshot('tablet-nav-menu-open.png', {
        threshold: 0.05,
        fullPage: false,
      });

      // Verify all menu items visible
      await expect(page.locator('.dropdown-content a:has-text("Home")')).toBeVisible();
      await expect(page.locator('.dropdown-content a:has-text("Validators")')).toBeVisible();
      await expect(page.locator('.dropdown-content a:has-text("Metrics")')).toBeVisible();
      await expect(page.locator('.dropdown-content a:has-text("GraphQL")')).toBeVisible();
      await expect(page.locator('.dropdown-content a:has-text("Login")')).toBeVisible();
    });

    test('should hide desktop navigation links on tablet', async ({ page }) => {
      await page.goto('/');
      await page.waitForLoadState('networkidle');

      // Desktop menu should not be in viewport
      const desktopMenu = page.locator('.navbar-center.hidden');
      await expect(desktopMenu).not.toBeInViewport();
    });
  });

  test.describe('Mobile Navigation (375px)', () => {
    test.use({ viewport: { width: 375, height: 667 } });

    test('should render mobile navigation with hamburger', async ({ page }) => {
      await page.goto('/');
      await page.waitForLoadState('networkidle');

      // Screenshot: Mobile navigation (closed)
      await expect(page.locator('nav.navbar')).toHaveScreenshot('mobile-nav-closed.png', {
        threshold: 0.05,
      });
    });

    test('should show short logo text on mobile', async ({ page }) => {
      await page.goto('/');
      await page.waitForLoadState('networkidle');

      // Verify short logo visible on small screens
      const shortLogo = page.locator('span.inline.sm\\:hidden:has-text("ETH Monitor")');
      await expect(shortLogo).toBeVisible();
    });

    test('should open dropdown menu on mobile', async ({ page }) => {
      await page.goto('/');
      await page.waitForLoadState('networkidle');

      // Click hamburger
      const hamburger = page.locator('button[aria-label="Open menu"]');
      await hamburger.click();

      // Wait for dropdown
      await page.waitForSelector('.dropdown-content', { state: 'visible' });

      // Screenshot: Mobile dropdown menu
      await expect(page).toHaveScreenshot('mobile-nav-menu-open.png', {
        threshold: 0.05,
        fullPage: false,
      });
    });

    test('should close dropdown when clicking outside', async ({ page }) => {
      await page.goto('/');
      await page.waitForLoadState('networkidle');

      // Open menu
      const hamburger = page.locator('button[aria-label="Open menu"]');
      await hamburger.click();
      await page.waitForSelector('.dropdown-content', { state: 'visible' });

      // Click outside (on main content)
      await page.locator('main').click();

      // Wait for dropdown to hide
      await page.waitForSelector('.dropdown-content', { state: 'hidden' });

      // Screenshot: Menu closed after outside click
      await expect(page.locator('nav.navbar')).toHaveScreenshot('mobile-nav-closed-after-outside-click.png', {
        threshold: 0.05,
      });
    });

    test('should navigate to link from mobile dropdown', async ({ page }) => {
      await page.goto('/');
      await page.waitForLoadState('networkidle');

      // Open menu
      const hamburger = page.locator('button[aria-label="Open menu"]');
      await hamburger.click();
      await page.waitForSelector('.dropdown-content', { state: 'visible' });

      // Click Validators link
      const validatorsLink = page.locator('.dropdown-content a:has-text("Validators")');
      await validatorsLink.click();

      // Verify navigation occurred
      await expect(page).toHaveURL('/validators');
    });
  });

  test.describe('Accessibility Validation', () => {
    test('should have proper ARIA labels on hamburger button', async ({ page }) => {
      page.setViewportSize({ width: 375, height: 667 });
      await page.goto('/');
      await page.waitForLoadState('networkidle');

      const hamburger = page.locator('button[aria-label="Open menu"]');

      // Verify ARIA attributes
      await expect(hamburger).toHaveAttribute('aria-label', 'Open menu');
      await expect(hamburger).toHaveAttribute('aria-haspopup', 'true');
      await expect(hamburger).toHaveAttribute('tabindex', '0');
    });

    test('should have sufficient touch target size on mobile', async ({ page }) => {
      page.setViewportSize({ width: 375, height: 667 });
      await page.goto('/');
      await page.waitForLoadState('networkidle');

      // Measure hamburger button size
      const hamburger = page.locator('button[aria-label="Open menu"]');
      const box = await hamburger.boundingBox();

      // Verify >= 44px (WCAG 2.1 AA minimum touch target)
      expect(box?.width).toBeGreaterThanOrEqual(44);
      expect(box?.height).toBeGreaterThanOrEqual(44);
    });

    test('should have visible focus indicators', async ({ page }) => {
      await page.goto('/');
      await page.waitForLoadState('networkidle');

      // Focus on logo link
      const logoLink = page.locator('a.btn-ghost:has-text("Ethereum")');
      await logoLink.focus();

      // Screenshot: Focus indicator visible
      await expect(page.locator('nav.navbar')).toHaveScreenshot('nav-focus-indicator.png', {
        threshold: 0.05,
      });

      // Verify element is focused
      await expect(logoLink).toBeFocused();
    });
  });

  test.describe('Layout Stability - No Shift', () => {
    test('should not cause layout shift when loading', async ({ page }) => {
      await page.goto('/');

      // Take screenshot immediately after load
      await page.waitForLoadState('domcontentloaded');
      const screenshotBefore = await page.locator('nav.navbar').screenshot();

      // Wait for network idle
      await page.waitForLoadState('networkidle');
      const screenshotAfter = await page.locator('nav.navbar').screenshot();

      // Screenshots should be identical (no layout shift)
      expect(screenshotBefore).toEqual(screenshotAfter);
    });

    test('should not shift layout when toggling mobile menu', async ({ page }) => {
      page.setViewportSize({ width: 375, height: 667 });
      await page.goto('/');
      await page.waitForLoadState('networkidle');

      // Capture navbar position before opening menu
      const navbarBefore = await page.locator('nav.navbar').boundingBox();

      // Open menu
      const hamburger = page.locator('button[aria-label="Open menu"]');
      await hamburger.click();
      await page.waitForSelector('.dropdown-content', { state: 'visible' });

      // Capture navbar position after opening menu
      const navbarAfter = await page.locator('nav.navbar').boundingBox();

      // Navbar position should not change
      expect(navbarBefore?.y).toEqual(navbarAfter?.y);
      expect(navbarBefore?.height).toEqual(navbarAfter?.height);
    });
  });

  test.describe('Error States', () => {
    test('should handle missing CSS gracefully', async ({ page }) => {
      // Block CSS file
      await page.route('**/output.css', route => route.abort());

      await page.goto('/');
      await page.waitForLoadState('networkidle');

      // Screenshot: Navigation without CSS (fallback)
      await expect(page.locator('nav')).toHaveScreenshot('nav-no-css-fallback.png', {
        threshold: 0.1, // Higher threshold for degraded state
      });

      // Navigation structure should still exist
      await expect(page.locator('nav a:has-text("Home")')).toBeVisible();
    });
  });
});
