import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright config for cqrs-htmx offline sync E2E tests.
 *
 * The webServer option auto-starts the Go test server (e2e/server/main.go)
 * before tests and stops it after. GOEXPERIMENT=jsonv2 is mandatory for
 * the cqrs-htmx module to build.
 *
 * Run: npx playwright test
 * Debug: npx playwright test --headed --debug
 */
export default defineConfig({
  testDir: './tests',
  fullyParallel: false, // SharedWorker + IndexedDB state is shared per origin
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  workers: 1, // single-worker: offline/online toggles affect the whole context
  reporter: [['list'], ['html', { open: 'never' }]],
  timeout: 60_000,
  expect: { timeout: 15_000 },

  use: {
    baseURL: 'http://localhost:18923',
    // On NixOS, Playwright's downloaded Chromium cannot run (no FHS linker).
    // Use the system/Nix Chromium via E2E_BROWSER_PATH when set.
    launchOptions: {
      executablePath: process.env.E2E_BROWSER_PATH || undefined,
    },
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],

  webServer: {
    command: 'go run .',
    cwd: './server',
    url: 'http://localhost:18923/health',
    reuseExistingServer: !process.env.CI,
    timeout: 120_000, // first `go run` compiles; subsequent runs are fast
    env: {
      GOEXPERIMENT: 'jsonv2',
    },
  },
});
