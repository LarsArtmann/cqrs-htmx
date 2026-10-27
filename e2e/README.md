# Offline Sync E2E Tests

Playwright-based browser tests for the cqrs-htmx offline sync SharedWorker (`sync/sync-worker.js` + `sync-client.js`), the adminui/dashboardui surfaces, and the loginpage WebAuthn ceremonies.

## What these tests cover

The E2E suite (`tests/sync.spec.ts`) exercises the offline command retry pipeline end-to-end in a real browser:

1. **Offline command queuing** — commands submitted while offline are persisted to IndexedDB
2. **Retry on reconnect** — queued commands are retried when the SharedWorker detects reconnection
3. **Deduplication** — retried commands don't double-execute
4. **Dead port resilience** — commands survive tab close/reopen

`tests/loginpage-webauthn.spec.ts` drives the rendered login page through
the full register and login passkey ceremonies against a REAL go-webauthn
server (`e2e/server/login.go`, in-memory usermgmt) using Playwright's CDP
virtual authenticator — genuine attestation/assertion bytes, session
cookie, CSRF round-trip, and the ADR-0055 register-issues-cookie
dependency. `tests/loginpage-golden.spec.ts` pins the full-page render DOM
(nonce- and CSRF-insensitive; regenerate intentional render changes with
`WRITE_GOLDEN=1 npx playwright test loginpage-golden`).

## Prerequisites

- Node.js 18+ (or [Bun](https://bun.sh))
- Playwright browsers, installed with the LOCAL runner copy (never `pnpm dlx`):
  `pnpm exec playwright install chromium`
- NixOS only: Playwright's downloaded Chromium cannot run (no FHS linker).
  Point `E2E_BROWSER_PATH` at a Nix-packaged Chromium instead (the
  `nix run .#e2e` app below auto-detects one), e.g.
  `export E2E_BROWSER_PATH="$(nix build nixpkgs#chromium --print-out-paths)/bin/chromium"`

## How to run

```bash
cd e2e

# Install dependencies (first time only)
pnpm install   # or: bun install

# Run the test suite (headless) — the LOCAL @playwright/test copy
pnpm test   # or: pnpm exec playwright test

# Run with visible browser
pnpm run test:headed

# Debug mode (step through each test)
pnpm run test:debug

# View the HTML report after a run
pnpm run report
```

> **Never `pnpm dlx playwright test`.** The dlx'd standalone runner instance
> skews against the locally installed `@playwright/test` (same version,
> different copies) and fails with `Playwright Test did not expect test() to
be called here` (hit live 2026-10-13). Every script in `package.json`
> resolves the local copy.

### One-command path (Nix)

```bash
nix run .#e2e
```

Builds the Go test server, provisions a writable
`PLAYWRIGHT_BROWSERS_PATH`, and auto-detects a Nix Chromium for
`E2E_BROWSER_PATH` — the environment the full battery (73 specs) runs under.

## Test server

The E2E tests spin up a Go test server (`e2e/server/`) that simulates the cqrs-htmx app with sync endpoints. The server is started automatically by Playwright's `webServer` config (see `playwright.config.ts`).

To rebuild the test server:

```bash
cd e2e/server
GOEXPERIMENT=jsonv2 go build -o test-server .
```

## Configuration

- `playwright.config.ts` — browser config, timeouts, webServer setup
- `tsconfig.json` — TypeScript config for test files

## CI integration

The suite is a local battery (`nix run .#e2e`); this repo's CI does not run
browsers. To run it in your own pipeline:

1. Install Playwright browsers: `pnpm exec playwright install --with-deps chromium`
2. Run `cd e2e && pnpm install && pnpm test`
3. Upload `playwright-report/` as a CI artifact

## See also

- `sync/sync-worker.js` / `sync/sync-client.js` — the SharedWorker + client code under test
- ADR-0029 — Offline sync architecture
- ADR-0040 — Retry pipeline fixes
