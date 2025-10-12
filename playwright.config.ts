import { defineConfig } from '@playwright/test';

/**
 * Playwright config for CLI E2E tests
 * Tests the Kilometers CLI tool by spawning processes and verifying behavior
 */
export default defineConfig({
  testDir: './tests/e2e-playwright',

  /* Run tests in files in parallel */
  fullyParallel: false, // CLI tests often need to run sequentially

  /* Fail the build on CI if you accidentally left test.only in the source code. */
  forbidOnly: !!process.env.CI,

  /* Retry on CI only */
  retries: process.env.CI ? 2 : 0,

  /* Run tests sequentially for CLI (avoid conflicts) */
  workers: 1,

  /* Reporter to use */
  reporter: [
    ['html', { outputFolder: 'playwright-report-cli' }],
    ['json', { outputFile: 'playwright-report-cli/results.json' }],
    ['list']
  ],

  /* Shared settings */
  use: {
    /* Collect trace when retrying the failed test */
    trace: 'on-first-retry',
  },

  /* Projects for different test suites */
  projects: [
    {
      name: 'cli-tests',
      testMatch: '**/*.spec.ts',
    },
  ],

  /* Global timeout */
  timeout: 60 * 1000, // CLI operations can take longer

  /* Expect timeout */
  expect: {
    timeout: 10 * 1000,
  },

  /* Output directory for test artifacts */
  outputDir: 'test-results-cli',
});
