# Offline Sync E2E Tests

Playwright-based browser tests for the cqrs-htmx offline sync SharedWorker (`sync/sync-worker.js` + `sync/sync-client.js`).

## What these tests cover

The E2E suite (`tests/sync.spec.ts`) exercises the offline command retry pipeline end-to-end in a real browser:

1. **Offline command queuing** — commands submitted while offline are persisted to IndexedDB
2. **Retry on reconnect** — queued commands are retried when the SharedWorker detects reconnection
3. **Deduplication** — retried commands don't double-execute
4. **Dead port resilience** — commands survive tab close/reopen

## Prerequisites

- Node.js 18+ (or [Bun](https://bun.sh))
- Playwright browsers: `pnpm dlx playwright install --with-deps chromium`

## How to run

```bash
cd e2e

# Install dependencies (first time only)
bun install   # or: pnpm install

# Run the test suite (headless)
bun run test  # or: pnpm test

# Run with visible browser
bun run test:headed

# Debug mode (step through each test)
bun run test:debug

# View the HTML report after a run
bun run report
```

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

These tests are **not** wired into `flake.nix` yet (Playwright requires browser binaries that aren't in the Nix shell). To run in CI:

1. Install Playwright browsers in your CI pipeline
2. Run `cd e2e && bun install && bun run test`
3. Upload `playwright-report/` as a CI artifact

## See also

- `sync/sync-worker.js` / `sync/sync-client.js` — the SharedWorker + client code under test
- ADR-0029 — Offline sync architecture
- ADR-0040 — Retry pipeline fixes
