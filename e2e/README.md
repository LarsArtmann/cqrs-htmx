# Browser E2E Battery

Playwright-based browser tests for every client-facing cqrs-htmx surface: the offline-sync SharedWorker, adminui, dashboardui, loginpage (WebAuthn + posture), the root HTMX helpers, the setup bundle, the DataStar adapter, the health/auditlog bridge dashboards, and the client tracker.

## Coverage matrix

| Spec                                | Surface under test                                        | Server (port)                   | Notes                                                                              |
| ----------------------------------- | --------------------------------------------------------- | ------------------------------- | ---------------------------------------------------------------------------------- |
| `sync.spec.ts`                      | offline-sync SharedWorker + pull/push handlers            | e2e/server (:18923)             | offline/online toggles, IndexedDB queue, dedup, backendId reset                    |
| `axe.spec.ts` / `axe-probe.spec.ts` | dashboardui accessibility                                 | e2e/server (:18923)             | axe sweeps, light + dark                                                           |
| `screenshots.spec.ts`               | dashboardui visual capture                                | e2e/server (:18923)             | light/dark/mobile → `docs/screenshots/`                                            |
| `admin-behavior.spec.ts`            | adminui                                                   | examples/admin-demo (:18930)    |                                                                                    |
| `admin-csrf.spec.ts`                | adminui CSRF refusal (browser-level proof)                | examples/admin-demo (:18930)    | token-less POST → 403; paired with-token control creates                           |
| `admin-a11y.spec.ts`                | adminui accessibility (axe + keyboard)                    | examples/admin-demo (:18930)    | also in the webkit smoke subset                                                    |
| `datastar.spec.ts`                  | DataStar adapter demo                                     | examples/datastar-demo (:18931) | the demo that shipped dead-in-browser through every gate (2026-11-07)              |
| `root-htmx.spec.ts`                 | root HTMX helpers (RenderPartialOrFull, OOB, SSE refresh) | examples/basic (:18932)         |                                                                                    |
| `setup-bundle.spec.ts`              | setup composition root                                    | examples/setup-demo (:18933)    | session gates, /health, dual-transport hub                                         |
| `loginpage-webauthn.spec.ts`        | loginpage + usermgmt WebAuthn                             | e2e/server (:18923)             | CDP virtual authenticator — **Chromium-only**                                      |
| `loginpage-golden.spec.ts`          | loginpage render DOM                                      | e2e/server (:18923)             | regenerate with `WRITE_GOLDEN=1`                                                   |
| `loginpage-posture.spec.ts`         | /auth/me + logout session contract                        | e2e/server (:18923)             | 401 anonymous / 200 with session / revoked after logout                            |
| `dashboard.spec.ts`                 | dashboardui static views                                  | e2e/server (:18923)             |                                                                                    |
| `dashboard-live.spec.ts`            | dashboardui live SSE views                                | e2e/server (:18923)             | journal replay, multi-tab, time-travel, `/api/debug/emit` driver                   |
| `bridges.spec.ts`                   | health dashboard + auditlog live viewer                   | e2e/server (:18923)             | `/api/debug/invoke` driver                                                         |
| `tracker.spec.ts`                   | client tracker (`tracker.js` + `TrackHandler`)            | e2e/server (:18923)             | page view, dwell, click, impression attributed; GPC + `data-tracking="off"` silent |

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

### WebKit advisory smoke (T28, 2026-11-24)

A `webkit` project exists but is ABSENT unless you opt in — like firefox,
it appears only when `E2E_WEBKIT_PATH` (a webkit executable) or
`E2E_WEBKIT` (CI-style: let Playwright resolve its registry build) is set.
Per-project `testMatch` scopes it to the admin smoke subset —
`admin-a11y`, `admin-tenant-access`, and `admin-csrf` — never the full
battery. The leg is ADVISORY (kill point K-T28): the CI step runs with
`continue-on-error` and must never block.

Provisioning (gotcha 66 — ONLY the repo's own pinned driver installer, so
the downloaded build is by construction the revision THAT driver expects;
never copy a browser tree from another host/cache/nix store), then a no-op
re-run of the installer before the first spec:

```bash
cd e2e && PLAYWRIGHT_SKIP_VALIDATE_HOST_REQUIREMENTS=1 \
  PLAYWRIGHT_BROWSERS_PATH=/mnt/buildcache/playwright \
  ./node_modules/.bin/playwright install webkit   # driver pins webkit-2388
```

On NixOS the WPE MiniBrowser needs a lib forest the shared chromium/firefox
one doesn't cover (~40 libs: the gstreamer stack, icu74, enchant, ...) plus
three live traps: `libjpeg8`/`libopus` are the attrs with the sonames
webkit-2388 links (`libjpeg` here is the 62-ABI; `opus` is static-only),
and hb-icu comes from an Ubuntu .deb — harfbuzz dropped the API upstream
while webkit-2388 hard-needs it (`WPEWebProcess: undefined symbol:
hb_icu_script_to_script` without it). Build the forest reproducibly with:

```bash
bash e2e/webkit-forest.sh          # → /tmp/webkit-forest/lib
E2E_WEBKIT_PATH=/mnt/buildcache/playwright/webkit-2388/pw_run.sh \
  E2E_WEBKIT_LIBS=/tmp/webkit-forest/lib \
  ./node_modules/.bin/playwright test --project=webkit --reporter=line
```

The config auto-bypasses BOTH webkit launch wrappers (`pw_run.sh` and the
zip bundle's own `MiniBrowser` script — each REBUILDS `LD_LIBRARY_PATH`
for the browser child and would clobber the injection; first launch died
on `libatk-1.0.so.0` with the forest already exported): it execs
`minibrowser-wpe/bin/MiniBrowser` directly with the bundle env derived
from `E2E_WEBKIT_PATH`. CI is unaffected (`E2E_WEBKIT=1` →
registry-resolved executable + apt libs from `--with-deps`). The
libxml2.so.2 → .so.16 shim in the forest is the FIRST suspect on any
future XML-related webkit red.

Receipt 2026-11-24: pinned-driver install + no-op re-run clean; webkit
smoke 10/10 (admin-a11y + admin-tenant-access + admin-csrf), the same
subset 10/10 under chromium, reproduced 10/10 from a fresh
`webkit-forest.sh` build — first webkit-green run of this battery.

> **Never `pnpm dlx playwright test`** (or `bunx playwright test` outside
> `nix run .#e2e-firefox`, which resolves the local copy). A dlx'd standalone
> runner skews against the locally installed `@playwright/test` and fails
> with "Playwright Test did not expect test() to be called here" (2026-10-13).

> **Serial mode does not share browser contexts.**
> `test.describe.configure({ mode: "serial" })` runs tests in order and stops
> after a failure, but every test still gets a FRESH context - cookies and
> pages do not carry over, and the remaining tests after a failure show as
> "did not run". Establish per-test state explicitly (the posture spec's
> self-contained register-then-logout is the pattern).

### Sanctioned runs and the stale-server trap

`reuseExistingServer` is ON locally: a leaked server from an earlier run is
silently reused, and you test **yesterday's binary**. Kill before every spec
run, with the tool that targets exactly what holds a battery port:

```bash
bash scripts/tools/e2e-kill-strays.sh
```

(Never `pkill -x server`: the bare name matches every process of that name
you can signal, and a stray is defined by the battery ports, not by its name.
The tool refuses while another session's battery is live; coordinate first,
then pass `--force`.)

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

### Engine scope and shared-state discipline (2026-11-08)

- One `webServer` set backs EVERY project in the invocation: both engines
  replay the suite against the SAME demo servers. Never assert absolutes on
  server-side state (item stores, global counters, seeded rows) — assert
  deltas against what the page reports, and tests that delete fixtures must
  seed their own victims. Gotcha 59 owns the full story (the four 2026-11-08
  "firefox divergences" were this class, not engine bugs).
- `sync.spec.ts` is scoped OFF the firefox project (`testIgnore` in the
  config, cited): the SharedWorker's backendId cache reset does not wipe
  under firefox and the interception-based offline tests fail per-build —
  one open investigation row in `TODO_LIST.md`.
- Firefox suspends a BACKGROUND tab's stream processing (probed: zero
  deliveries until the tab is fronted) AND cannot hold two CONCURRENT SSE
  streams in headless mode (the second tab's `@get` dies
  `NS_BINDING_ABORTED`, its replacement hangs; sequential tabs — close the
  first, open the second — connect instantly). The multi-tab spec proves
  its feeds SEQUENTIALLY, identical choreography on both engines.
  Headless chromium never throttles.

## Test server

`e2e/server/` mounts the loginpage/usermgmt stack, the dashboard, the bridge
dashboards, the client-tracking surfaces (tracker script, `POST /-/track`
behind a session, the `/tracking` page under test), and the debug drivers
(`POST /api/debug/emit`, `POST /api/debug/invoke`, `POST
/api/debug/rotate-backend`, `POST /api/debug/tracking-login`, `GET
/api/debug/signals`, `POST /api/debug/signals/reset`). Playwright's
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
env the project is absent and the run is chromium-only. In CI BOTH lanes
are blocking since 2026-11-08 (F20 promotion: three clean firefox
observations + the divergence-resolution wave); locally keep using
`nix run .#e2e-firefox` for the both-engine pass.

WebKit is the THIRD lane and advisory-only (T28/F206, 2026-11-24): one
step installs webkit in-step (`bunx playwright install --with-deps
webkit`) and runs `--project=webkit` with `continue-on-error: true` — a
provisioning or spec failure must never redden CI. Promotion to blocking
requires the same evidence class as firefox got: consecutive clean
observations plus resolved divergences.

## See also

- `sync/sync-worker.js` / `sync/sync-client.js` — the SharedWorker + client code under test
- ADR-0029 — Offline sync architecture
- ADR-0040 — Retry pipeline fixes
- ADR-0056 — Frontend sync protocol
