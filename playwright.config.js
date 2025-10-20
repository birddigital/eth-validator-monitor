/**
 * Playwright Configuration for Visual Regression Testing
 *
 * This configuration is optimized for testing the Ethereum Validator Monitor
 * web interface across multiple browsers and viewports.
 */

const { defineConfig, devices } = require('@playwright/test');

module.exports = defineConfig({
  // Test directory
  testDir: './tests/visual-regression/specs',

  // Maximum time one test can run (30 seconds)
  timeout: 30000,

  // Fail the build on CI if you accidentally left test.only in the source code
  forbidOnly: !!process.env.CI,

  // Retry on CI only
  retries: process.env.CI ? 2 : 0,

  // Parallel workers (CI: 1, local: auto)
  workers: process.env.CI ? 1 : undefined,

  // Reporter configuration
  reporter: [
    ['html', { outputFolder: 'tests/visual-regression/report' }],
    ['list'],
    ['json', { outputFile: 'tests/visual-regression/results.json' }]
  ],

  // Shared settings for all projects
  use: {
    // Base URL for navigation
    baseURL: process.env.BASE_URL || 'http://localhost:8080',

    // Collect trace when retrying the failed test
    trace: 'on-first-retry',

    // Screenshot on failure
    screenshot: 'only-on-failure',

    // Video on failure
    video: 'retain-on-failure',

    // Maximum time for actions like click, fill, etc.
    actionTimeout: 10000,

    // Navigation timeout
    navigationTimeout: 15000,
  },

  // Configure projects for major browsers
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] },
    },
    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] },
    },

    // Mobile viewports
    {
      name: 'Mobile Chrome',
      use: { ...devices['Pixel 5'] },
    },
    {
      name: 'Mobile Safari',
      use: { ...devices['iPhone 12'] },
    },

    // Tablet viewports
    {
      name: 'iPad',
      use: { ...devices['iPad Pro'] },
    },
  ],

  // Run local dev server before starting tests (optional)
  // Uncomment if you want Playwright to automatically start the server
  // webServer: {
  //   command: 'make run',
  //   port: 8080,
  //   timeout: 120000,
  //   reuseExistingServer: !process.env.CI,
  // },

  // Folder for test artifacts
  outputDir: 'tests/visual-regression/artifacts',

  // Visual comparison settings
  expect: {
    // Maximum allowed visual difference (5%)
    toMatchSnapshot: {
      threshold: 0.05,
      maxDiffPixels: 100,
    },
  },
});
