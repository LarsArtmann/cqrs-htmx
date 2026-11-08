# Browser E2E Battery

Playwright-based browser tests for every client-facing cqrs-htmx surface: the offline-sync SharedWorker, adminui, dashboardui, loginpage (WebAuthn + posture), the root HTMX helpers, the setup bundle, the DataStar adapter, and the health/auditlog bridge dashboards.

## Coverage matrix

| Spec | Surface under test | Server (port) | Notes |
| --- | --- | --- | --- |
| `sync.spec.ts` | offline-sync SharedWorker + pull/push handlers | e2e/server (:18923) | offline/online toggles, IndexedDB queue, dedup, backendId reset |
| `axe.spec.ts` / `axe-probe.spec.ts` | dashboardui accessibility | e2e/server (:18923) | axe sweeps, light + dark |
| `screenshots.spec.ts` | dashboardui visual capture | e2e/server (:18923) | light/dark/mobile → `docs/screenshots/` |
| `admin-behavior.spec.ts` | adminui | examples/admin-demo (:18930) | |
| `datastar.spec.ts` | DataStar adapter demo | examples/datastar-demo (:18931) | the demo that shipped dead-in-browser through every gate (2026-11-07) |
| `root-htmx.spec.ts` | root HTMX helpers (RenderPartialOrFull, OOB, SSE refresh) | examples/basic (:18932) | |
| `setup-bundle.spec.ts` | setup composition root | examples/setup-demo (:18933) | session gates, /health, dual-transport hub |
| `loginpage-webauthn.spec.ts` | loginpage + usermgmt WebAuthn | e2e/server (:18923) | CDP virtual authenticator — **Chromium-only** |
| `loginpage-golden.spec.ts` | loginpage render DOM | e2e/server (:18923) | regenerate with `WRITE_GOLDEN=1` |
| `loginpage-posture.spec.ts` | /auth/me + logout session contract | e2e/server (:18923) | 401 anonymous / 200 with session / revoked after logout |
| `dashboard.spec.ts` | dashboardui static views | e2e/server (:18923) | |
| `dashboard-live.spec.ts` | dashboardui live SSE views | e2e/server (:18923) | journal replay, multi-tab, time-travel, `/api/debug/emit` driver |
| `bridges.spec.ts` | health dashboard + auditlog live viewer | e2e/server (:18923) | `/api/debug/invoke` driver |

## How to run

```bash
cd e2e
bun install   # or: pnpm install

# Full battery, headless — the LOCAL runner copy
./node_modules/.bin/playwright test

# One spec
./node_modules/.bin/playwright test bridges.spec.ts --reporter=line

# Both engines (chromium + firefox) — builds the nixpkgs lib forest the
# prebuilt firefox needs on NixOS, locates the cached Playwright firefox
nix run .#e2e-firefox
```

> **Never `pnpm dlx playwright test`** (or `bunx playwright test` outside
> `nix run .#e2e-firefox`, which resolves the local copy). A dlx'd standalone
> runner skews against the locally installed `@playwright/test` and fails
> with "Playwright Test did not expect test() to be called here" (2026-10-13).

### Sanctioned runs and the stale-server trap

`reuseExistingServer` is ON locally: a leaked server from an earlier run is
silently reused, and you test **yesterday's binary**. Kill before every spec
run:

```bash
pkill -x server; pkill -f '/tmp/e2e-server'; ss -tlnp | grep 18923
```

(The mvdan/sh shell has no `kill` builtin and `/bin/kill` does not exist on
this host — use pkill patterns.)

### NixOS browser notes

Playwright's downloaded Chromium cannot run on NixOS (no FHS linker). Point
`E2E_BROWSER_PATH` at a Nix-packaged Chromium:

```bash
export E2E_BROWSER_PATH="$(nix build nixpkgs#chromium --print-out-paths --no-link)/bin/chromium"
```

Firefox: the `e2e-firefox` app builds a symlink forest of nixpkgs libs
(`E2E_FIREFOX_LIBS`) that the config injects into firefox launches ONLY — a
runner-wide `LD_LIBRARY_PATH` poisons the chromium children (their bundled
nspr resolution breaks; hit live 2026-11-08).

## Test server

`e2e/server/` mounts the loginpage/usermgmt stack, the dashboard, the bridge
dashboards, and the debug drivers (`POST /api/debug/emit`, `POST
/api/debug/invoke`, `POST /api/debug/rotate-backend`). Playwright's
`webServer` entries start it (plus the four example servers) automatically —
the config pins `GOWORK`/`GOTOOLCHAIN`/`GOEXPERIMENT` per server.

When building it by hand, keep the workspace: `GOWORK=<repo>/go.work go
build` — the ambient shell may otherwise resolve published tags instead of
the local modules (observed 2026-09-21).

## CI integration

The battery runs in CI (`.github/workflows/ci.yml`, `e2e` job): sibling repos
cloned + `go.work` paths rewritten, servers pre-built, browsers installed
via `bunx playwright install --with-deps chromium firefox`, both engines.
Firefox is opt-in via `E2E_FIREFOX=1` / `E2E_FIREFOX_PATH` — without either
env the project is absent and the run is chromium-only.

## See also

- `sync/sync-worker.js` / `sync/sync-client.js` — the SharedWorker + client code under test
- ADR-0029 — Offline sync architecture
- ADR-0040 — Retry pipeline fixes
- ADR-0056 — Frontend sync protocol
