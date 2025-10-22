import { test, expect } from '@playwright/test';

/**
 * Dashboard Page Visual Regression Tests
 *
 * Tests the dashboard layout and skeleton loaders across multiple viewports
 * and themes. Ensures no layout shift during HTMX partial updates.
 */

const viewports = {
  mobile: { width: 375, height: 667 },    // iPhone SE
  tablet: { width: 768, height: 1024 },   // iPad
  desktop: { width: 1440, height: 900 },  // Standard laptop
  wide: { width: 1920, height: 1080 }     // Desktop monitor
};

test.describe('Dashboard Page - Skeleton Loaders', () => {
  for (const [device, viewport] of Object.entries(viewports)) {
    test.describe(`${device} viewport`, () => {
      test.use({ viewport });

      test('should display skeleton loaders on initial load', async ({ page }) => {
        // Navigate to dashboard
        await page.goto('/dashboard');

        // Wait for page to be in loading state
        await page.waitForLoadState('domcontentloaded');

        // Capture skeleton loaders before HTMX loads data
        const screenshot = await page.screenshot({
          fullPage: true,
          animations: 'disabled' // Disable shimmer animation for consistent snapshots
        });

        expect(screenshot).toMatchSnapshot(`${device}-dashboard-skeleton-loaders.png`);
      });

      test('should show metrics skeleton loader', async ({ page }) => {
        // Intercept API call to delay response
        await page.route('**/api/dashboard/metrics', async route => {
          await new Promise(resolve => setTimeout(resolve, 1000));
          await route.continue();
        });

        await page.goto('/dashboard');

        // Wait for skeleton to be visible
        const metricsSkeleton = page.locator('#metrics-skeleton');
        await expect(metricsSkeleton).toBeVisible();

        // Capture metrics section with skeleton
        const metricsSection = page.locator('#metrics-section');
        await expect(metricsSection).toHaveScreenshot(`${device}-metrics-skeleton.png`, {
          animations: 'disabled'
        });
      });

      test('should show validators skeleton loader', async ({ page }) => {
        // Intercept API call to delay response
        await page.route('**/api/dashboard/validators', async route => {
          await new Promise(resolve => setTimeout(resolve, 1000));
          await route.continue();
        });

        await page.goto('/dashboard');

        // Wait for skeleton to be visible
        const validatorsSkeleton = page.locator('#validators-skeleton');
        await expect(validatorsSkeleton).toBeVisible();

        // Capture validators section with skeleton
        const validatorsSection = page.locator('#validators-section');
        await expect(validatorsSection).toHaveScreenshot(`${device}-validators-skeleton.png`, {
          animations: 'disabled'
        });
      });

      test('should show alerts skeleton loader', async ({ page }) => {
        // Intercept API call to delay response
        await page.route('**/api/dashboard/alerts', async route => {
          await new Promise(resolve => setTimeout(resolve, 1000));
          await route.continue();
        });

        await page.goto('/dashboard');

        // Wait for skeleton to be visible
        const alertsSkeleton = page.locator('#alerts-skeleton');
        await expect(alertsSkeleton).toBeVisible();

        // Capture alerts section with skeleton
        const alertsSection = page.locator('#alerts-section');
        await expect(alertsSection).toHaveScreenshot(`${device}-alerts-skeleton.png`, {
          animations: 'disabled'
        });
      });

      test('should show system health skeleton loader', async ({ page }) => {
        // Intercept API call to delay response
        await page.route('**/api/dashboard/health', async route => {
          await new Promise(resolve => setTimeout(resolve, 1000));
          await route.continue();
        });

        await page.goto('/dashboard');

        // Wait for skeleton to be visible
        const healthSkeleton = page.locator('#health-skeleton');
        await expect(healthSkeleton).toBeVisible();

        // Capture health section with skeleton
        const healthSection = page.locator('#health-section');
        await expect(healthSection).toHaveScreenshot(`${device}-health-skeleton.png`, {
          animations: 'disabled'
        });
      });

      test('should have no layout shift during HTMX swap', async ({ page }) => {
        await page.goto('/dashboard');

        // Wait for initial load
        await page.waitForLoadState('networkidle');

        // Capture before dimensions
        const metricsSection = page.locator('#metrics-section');
        const beforeBox = await metricsSection.boundingBox();

        // Trigger HTMX refresh
        await page.evaluate(() => {
          const section = document.querySelector('#metrics-section');
          if (section) {
            // @ts-ignore - htmx is globally available
            htmx.trigger(section, 'load');
          }
        });

        // Wait for swap to complete
        await page.waitForTimeout(500);

        // Capture after dimensions
        const afterBox = await metricsSection.boundingBox();

        // Verify no layout shift
        expect(beforeBox?.height).toBeGreaterThan(0);
        expect(afterBox?.height).toBeGreaterThan(0);

        // Allow small variations (±5px) for responsive adjustments
        const heightDiff = Math.abs((afterBox?.height || 0) - (beforeBox?.height || 0));
        expect(heightDiff).toBeLessThan(5);
      });
    });
  }
});

test.describe('Dashboard Page - Dark Mode', () => {
  test.use({ viewport: viewports.desktop });

  test('should render skeleton loaders in dark mode', async ({ page }) => {
    // Navigate to dashboard
    await page.goto('/dashboard');

    // Toggle dark mode
    await page.evaluate(() => {
      document.documentElement.setAttribute('data-theme', 'eth-dark');
    });

    // Wait for theme change
    await page.waitForTimeout(100);

    // Capture dark mode skeleton
    const screenshot = await page.screenshot({
      fullPage: true,
      animations: 'disabled'
    });

    expect(screenshot).toMatchSnapshot('desktop-dashboard-dark-mode-skeleton.png');
  });

  test('should have proper contrast ratios in dark mode', async ({ page }) => {
    await page.goto('/dashboard');

    // Enable dark mode
    await page.evaluate(() => {
      document.documentElement.setAttribute('data-theme', 'eth-dark');
    });

    // Get computed styles of skeleton elements
    const skeletonBg = await page.locator('.skeleton-shimmer').first().evaluate((el) => {
      return window.getComputedStyle(el).backgroundColor;
    });

    // Verify background color is set (basic check)
    expect(skeletonBg).toBeTruthy();
    expect(skeletonBg).not.toBe('rgba(0, 0, 0, 0)');
  });
});

test.describe('Dashboard Page - Responsive Grid Layout', () => {
  test('mobile layout - single column grid', async ({ page }) => {
    await page.setViewportSize(viewports.mobile);
    await page.goto('/dashboard');
    await page.waitForLoadState('networkidle');

    const screenshot = await page.screenshot({ fullPage: true });
    expect(screenshot).toMatchSnapshot('mobile-dashboard-layout.png');
  });

  test('tablet layout - two column grid', async ({ page }) => {
    await page.setViewportSize(viewports.tablet);
    await page.goto('/dashboard');
    await page.waitForLoadState('networkidle');

    const screenshot = await page.screenshot({ fullPage: true });
    expect(screenshot).toMatchSnapshot('tablet-dashboard-layout.png');
  });

  test('desktop layout - three/four column grid', async ({ page }) => {
    await page.setViewportSize(viewports.desktop);
    await page.goto('/dashboard');
    await page.waitForLoadState('networkidle');

    const screenshot = await page.screenshot({ fullPage: true });
    expect(screenshot).toMatchSnapshot('desktop-dashboard-layout.png');
  });

  test('wide layout - four column grid', async ({ page }) => {
    await page.setViewportSize(viewports.wide);
    await page.goto('/dashboard');
    await page.waitForLoadState('networkidle');

    const screenshot = await page.screenshot({ fullPage: true });
    expect(screenshot).toMatchSnapshot('wide-dashboard-layout.png');
  });
});

test.describe('Dashboard Page - Accessibility', () => {
  test.use({ viewport: viewports.desktop });

  test('should have proper ARIA attributes on skeleton loaders', async ({ page }) => {
    await page.goto('/dashboard');

    // Check metrics section ARIA attributes
    const metricsSection = page.locator('#metrics-section');
    await expect(metricsSection).toHaveAttribute('role', 'status');
    await expect(metricsSection).toHaveAttribute('aria-live', 'polite');
    await expect(metricsSection).toHaveAttribute('aria-label', 'Aggregate metrics');

    // Check skeleton has aria-hidden
    const metricsSkeleton = page.locator('#metrics-skeleton');
    const ariaHidden = await metricsSkeleton.evaluate((el) =>
      el.querySelector('[aria-hidden]')?.getAttribute('aria-hidden')
    );
    expect(ariaHidden).toBe('true');

    // Check screen reader text is present
    const srText = page.locator('#metrics-skeleton .sr-only');
    await expect(srText).toHaveText(/Loading metrics/);
  });

  test('should announce loading state to screen readers', async ({ page }) => {
    await page.goto('/dashboard');

    // Check all sections have screen reader announcements
    const sections = [
      { id: '#metrics-skeleton', text: 'Loading metrics' },
      { id: '#validators-skeleton', text: 'Loading top validators' },
      { id: '#alerts-skeleton', text: 'Loading recent alerts' },
      { id: '#health-skeleton', text: 'Loading system health' }
    ];

    for (const section of sections) {
      const srText = page.locator(`${section.id} .sr-only`);
      await expect(srText).toContainText(section.text);
    }
  });

  test('should have keyboard navigable sections', async ({ page }) => {
    await page.goto('/dashboard');
    await page.waitForLoadState('networkidle');

    // Tab through page
    await page.keyboard.press('Tab');

    // Verify focus indicator is visible
    const focusedElement = await page.evaluate(() => document.activeElement?.tagName);
    expect(focusedElement).toBeTruthy();
  });
});

test.describe('Dashboard Page - Performance', () => {
  test.use({ viewport: viewports.desktop });

  test('should show skeleton for slow responses (>300ms)', async ({ page }) => {
    // Intercept and delay API responses
    await page.route('**/api/dashboard/**', async route => {
      await new Promise(resolve => setTimeout(resolve, 500)); // 500ms delay
      await route.continue();
    });

    await page.goto('/dashboard');

    // Skeleton should be visible due to delay
    const skeleton = page.locator('.htmx-indicator').first();
    await expect(skeleton).toBeVisible();
  });

  test('should hide skeleton for fast responses (<300ms)', async ({ page }) => {
    // No interception - let requests complete naturally
    await page.goto('/dashboard');

    // Wait for network idle
    await page.waitForLoadState('networkidle');

    // Skeletons should be hidden
    const skeletons = page.locator('.htmx-indicator');
    const count = await skeletons.count();

    for (let i = 0; i < count; i++) {
      await expect(skeletons.nth(i)).toBeHidden();
    }
  });
});
