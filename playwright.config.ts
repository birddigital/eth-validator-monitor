import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright configuration for visual regression testing
 * See https://playwright.dev/docs/test-configuration
 */
export default defineConfig({
  testDir: './tests/visual-regression',

  // Run tests in parallel
  fullyParallel: true,

  // Fail the build on CI if you accidentally left test.only in the source code
  forbidOnly: !!process.env.CI,

  // Retry on CI only
  retries: process.env.CI ? 2 : 0,

  // Opt out of parallel tests on CI
  workers: process.env.CI ? 1 : undefined,

  // Reporter to use
  reporter: [
    ['html', { outputFolder: 'playwright-report' }],
    ['list']
  ],

  // Shared settings for all projects
  use: {
    // Base URL for tests
    baseURL: process.env.BASE_URL || 'http://localhost:8080',

    // Collect trace when retrying failed tests
    trace: 'on-first-retry',

    // Screenshot on failure
    screenshot: 'only-on-failure',
  },

  // Configure projects for major browsers and viewports
  projects: [
    // Mobile viewport (375px - iPhone SE)
    {
      name: 'mobile-chromium',
      use: {
        ...devices['iPhone SE'],
        viewport: { width: 375, height: 667 }
      },
    },

    // Tablet viewport (768px - iPad)
    {
      name: 'tablet-chromium',
      use: {
        ...devices['iPad'],
        viewport: { width: 768, height: 1024 }
      },
    },

    // Desktop viewport (1440px - standard laptop)
    {
      name: 'desktop-chromium',
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 1440, height: 900 }
      },
    },

    // Wide viewport (1920px - desktop monitor)
    {
      name: 'wide-chromium',
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 1920, height: 1080 }
      },
    },
  ],

  // Run local dev server before starting tests
  webServer: {
    command: 'make run',
    url: 'http://localhost:8080',
    reuseExistingServer: !process.env.CI,
    timeout: 30000,
    stdout: 'pipe',
    stderr: 'pipe',
  },
});
