import { defineConfig, devices } from "@playwright/test";
import fs from "node:fs";
import path from "node:path";

// Absolute path to the repo's go.work. The webServers must resolve Go code
// against the WORKSPACE (local unreleased adminui/cqrs-htmx), but `go` only
// accepts an absolute GOWORK — and inside `nix develop` the devShell exports
// GOWORK=off, which would silently compile the servers against PUBLISHED
// module tags (observed 2026-09-21: the adminui pagination gate failed with
// 65 rows because the published adminui predates pagination).
const GOWORK = path.resolve(__dirname, "../go.work");

// The workspace toolchain floor (go.work's `go` directive), derived from the
// single source of truth so a floor bump (e.g. a stdlib security release)
// never leaves a stale literal here — the 2026-10-11 1.27.2 bump broke the
// suite exactly this way (pinned go1.27.1 < floor).
const goFloor = /^go (\S+)$/m.exec(fs.readFileSync(GOWORK, "utf8"))?.[1] ?? "";
const GOTOOLCHAIN = goFloor ? `go${goFloor}` : undefined;

/**
 * Playwright config for cqrs-htmx offline sync E2E tests.
 *
 * The webServer option auto-starts the Go test server (e2e/server/main.go)
 * before tests and stops it after. GOEXPERIMENT=jsonv2 is mandatory for
 * the cqrs-htmx module to build.
 *
 * Run: pnpm test — the LOCAL @playwright/test runner (node_modules/.bin).
 * NEVER `pnpm dlx playwright test`: the dlx'd standalone runner instance
 * skews against the locally installed @playwright/test and dies with
 * "Playwright Test did not expect test() to be called here" (2026-10-13).
 * One-command path with browser/toolchain handling: nix run .#e2e
 * Debug session (browser visible): pnpm run test:headed
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
        // e2e/server is a workspace member on the go.work floor; pin the
        // toolchain so the suite also runs outside the nix devShell.
        ...(GOTOOLCHAIN ? { GOTOOLCHAIN } : {}),
        // Pin workspace resolution explicitly; see the GOWORK note at the
        // top of this file.
        GOWORK,
      },
    },
    {
      // admin-demo for tests/admin-behavior.spec.ts (adminui behavior gate).
      // Needs the workspace toolchain pin: the demo rides the go.work floor
      // while the ambient go may be older with GOTOOLCHAIN=local.
      command: "go run .",
      cwd: "../examples/admin-demo",
      url: "http://localhost:18930/",
      reuseExistingServer: !process.env.CI,
      timeout: 120_000,
      env: {
        GOEXPERIMENT: "jsonv2",
        ...(GOTOOLCHAIN ? { GOTOOLCHAIN } : {}),
        // See the GOWORK note at the top of this file.
        GOWORK,
      },
    },
  ],
});
