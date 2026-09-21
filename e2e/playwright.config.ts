import { defineConfig, devices } from "@playwright/test";

/**
 * Playwright config for cqrs-htmx offline sync E2E tests.
 *
 * The webServer option auto-starts the Go test server (e2e/server/main.go)
 * before tests and stops it after. GOEXPERIMENT=jsonv2 is mandatory for
 * the cqrs-htmx module to build.
 *
 * Run: pnpm dlx playwright test
 * Debug: pnpm dlx playwright test --headed --debug
 */
export default defineConfig({
  testDir: "./tests",
  fullyParallel: false, // SharedWorker + IndexedDB state is shared per origin
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  workers: 1, // single-worker: offline/online toggles affect the whole context
  reporter: [["list"], ["html", { open: "never" }]],
  timeout: 60_000,
  expect: { timeout: 15_000 },

  use: {
    baseURL: "http://localhost:18923",
    // On NixOS, Playwright's downloaded Chromium cannot run (no FHS linker).
    // Use the system/Nix Chromium via E2E_BROWSER_PATH when set.
    launchOptions: {
      executablePath: process.env.E2E_BROWSER_PATH || undefined,
    },
    trace: "on-first-retry",
    screenshot: "only-on-failure",
    video: "retain-on-failure",
  },

  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],

  webServer: [
    {
      command: "go run .",
      cwd: "./server",
      url: "http://localhost:18923/health",
      reuseExistingServer: !process.env.CI,
      timeout: 120_000, // first `go run` compiles; subsequent runs are fast
      env: {
        GOEXPERIMENT: "jsonv2",
        // e2e/server is a workspace member on the go 1.27.1 floor; pin the
        // toolchain so the suite also runs outside the nix devShell.
        GOTOOLCHAIN: "go1.27.1",
        // Pin workspace resolution explicitly (relative to this webServer's
        // cwd): inside `nix develop` the devShell exports GOWORK=off, which
        // would silently compile the demo against PUBLISHED module tags
        // instead of the local workspace members (observed 2026-09-21: the
        // adminui pagination gate failed with 65 rows because the published
        // adminui predates pagination).
        GOWORK: "../go.work",
      },
    },
    {
      // admin-demo for tests/admin-behavior.spec.ts (adminui behavior gate).
      // Needs the workspace toolchain pin: the demo rides the go 1.27.1
      // floor while the ambient go is 1.26.7/GOTOOLCHAIN=local.
      command: "go run .",
      cwd: "../examples/admin-demo",
      url: "http://localhost:18930/",
      reuseExistingServer: !process.env.CI,
      timeout: 120_000,
      env: {
        GOEXPERIMENT: "jsonv2",
        GOTOOLCHAIN: "go1.27.1",
        // See the first webServer: pin the workspace so a devShell-inherited
        // GOWORK=off cannot silently resolve the demo against published tags
        // (this cwd sits two levels below the repo root).
        GOWORK: "../../go.work",
      },
    },
  ],
});
