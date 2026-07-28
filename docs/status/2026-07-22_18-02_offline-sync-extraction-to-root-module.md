# Status: Offline Sync Extraction to Root Module — 2026-07-22

> Two major pieces of work this session: (1) hardened `sync-worker.js` from
> prototype to production, then (2) extracted the entire offline sync stack
> (SharedWorker + tab-side client) from `adminui/` to the root `cqrs-htmx`
> module so ANY consumer can use it — not just adminui users.

---

## Session Timeline

| Time   | Work                                                                                                                                                                       |
| ------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ~09:00 | Reviewed sync-worker.js — found 11 bugs (double-retry race, port leak, null-envelope poison, no eviction, thundering herd, etc.)                                           |
| ~09:12 | Rewrote sync-worker.js to production quality (IDB single source of truth, hello/bye protocol, retry count + TTL eviction, staggered/targeted delivery, in-memory fallback) |
| ~09:21 | Wrote status report for the hardening session                                                                                                                              |
| ~09:23 | User asked "why is this admin-only?" — realized it's an architecture mistake                                                                                               |
| ~09:23 | Planned extraction, created execution graph                                                                                                                                |
| ~09:30 | Executed extraction: root module assets + handlers + tests                                                                                                                 |
| ~09:55 | Updated adminui to delegate, regenerated templ, updated docs                                                                                                               |
| ~18:00 | Session resumed for status report                                                                                                                                          |

---

## a) FULLY DONE

### 1. sync-worker.js Production Hardening

Complete rewrite from 244-line prototype to 320-line production code.

| Bug Fixed                                             | How                                                                   |
| ----------------------------------------------------- | --------------------------------------------------------------------- |
| Double-retry race (flush + drainPersisted both fired) | IDB is single source of truth, one unified `flush()`                  |
| Port leak (dead ports never removed)                  | `Map<tabId, port>` + hello/bye protocol + postMessage-failure cleanup |
| Duplicate port on reconnect                           | `hello` replaces existing tabId's port                                |
| Null-envelope poison commands                         | Guard on enqueue: reject as `dead` before persisting                  |
| No eviction (infinite retry)                          | `MAX_RETRIES=10` + `RETRY_TTL_MS=24h` → `{type:"dead"}` + delete      |
| Retry count reset on re-enqueue                       | `store.add` (not `put`) — ConstraintError preserves original          |
| Thundering herd (broadcast to ALL tabs)               | Targeted retry: originating tab preferred, round-robin fallback       |
| No stagger on reconnect                               | 100ms per command, capped 2s                                          |
| Full table scan for count                             | `store.count()` instead of `getAll().length`                          |
| Concurrent flush double-increment                     | `flushing` flag + `flushPending` coalescing                           |
| No enqueue-while-online retry                         | `flush()` after persist if `online` is true                           |
| In-memory fallback inconsistent                       | Full `memQueue` Map mirrors IDB API                                   |
| Silent error swallowing                               | `console.warn` in all catch blocks                                    |

### 2. Root Module Extraction

**New root module files:**

| File                  | Purpose                                                                                                         |
| --------------------- | --------------------------------------------------------------------------------------------------------------- |
| `sync/sync-worker.js` | SharedWorker (copied from hardened version)                                                                     |
| `sync/sync-client.js` | Tab-side client (extracted from admin.js lines 63-402)                                                          |
| `sync_embed.go`       | `//go:embed` declarations + `syncVersion` const                                                                 |
| `sync_serve.go`       | `SyncWorkerHandler()`, `SyncClientHandler()`, `SyncWorkerScriptTag()`, `SyncClientScriptTag()`, `SyncVersion()` |

> **Update 2026-07-28:** `SyncWorkerScriptTag()` was **deleted** in the next session
> (`2026-07-22_18-21_post-extraction-cleanup-and-self-review.md`, item #1) — the URL-string helper
> was unnecessary; consumers use the `<script data-sync-worker-url>` attribute instead. The
> `With` variants (`SyncWorkerHandlerWith`/`SyncClientHandlerWith`) shipped later. See FEATURES.md
> "Offline Sync" row for the current API surface.
> | `sync_serve_test.go` | 8 tests: serve JS, 304-on-ETag, reject POST, version, script tags |

**New root module API (follows `HTMXScriptHandler` pattern):**

```go
mux.Handle("GET /sync-worker.js", cqrshtmx.SyncWorkerHandler())
mux.Handle("GET /sync-client.js", cqrshtmx.SyncClientHandler())
```

### 3. adminui Updated to Delegate

| File                             | Change                                                                                                                                             |
| -------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| `adminui/assets.go`              | Removed `sync-worker.js` from `go:embed`. Added `syncWorkerHandler()` / `syncClientHandler()` delegating to root. Bumped ETag to `adminui-v3.4.0`. |
| `adminui/handler.go`             | Routes `GET /-/sync-worker.js` and `GET /-/sync-client.js` via root handlers                                                                       |
| `adminui/assets/admin.js`        | Slimmed from ~400 to ~65 lines (CSRF, sidebar, toast, confirm only)                                                                                |
| `adminui/assets/sync-worker.js`  | **Deleted** (moved to root)                                                                                                                        |
| `adminui/layout.templ`           | Added conditional `<script src="sync-client.js">` when `SSEURL != ""`                                                                              |
| `adminui/layout_templ.go`        | Regenerated via `templ generate`                                                                                                                   |
| `adminui/coverage_gaps2_test.go` | Added `TestPanel_AssetsServeSyncClient`                                                                                                            |

### 4. Documentation Updated

- `docs/recipes/offline-command-sync.md` — new Step 3 showing root handler wiring
- `AGENTS.md` — root module description updated, gotcha updated
- `CHANGELOG.md` — extraction entry added under `[Unreleased] > Changed`
- `docs/planning/2026-07-22_09-23_extract-offline-sync-to-root-module.md` — execution plan with mermaid graph
- `docs/status/2026-07-22_09-21_sync-worker-production-hardening.md` — previous status report

### 5. Verification

- `go build ./...` — passes
- `go test ./... -race` — all root tests pass (4.1s)
- `go test ./adminui/... -race` — all adminui tests pass (4.2s)
- `node` syntax check — all 3 JS files valid (sync-worker.js, sync-client.js, admin.js)

---

## b) PARTIALLY DONE

### FEATURES.md

Updated in the hardening phase (retry count, TTL, staggered delivery) but NOT updated for the extraction. The Offline Sync row doesn't mention root module handlers or that any consumer can now use it. It still reads as an adminui-only feature.

### Recipe doc completeness

Updated Step 3 to show root handler wiring, but the rest of the recipe still has Phase 2a language in places and doesn't fully reflect the "any consumer" architecture.

---

## c) NOT STARTED

- **`examples/admin-demo/main.go`** — not checked for stale references to adminui sync assets
- **`example_htmx_test.go`** — no `ExampleSyncWorkerHandler()` / `ExampleSyncClientHandler()` entries for godoc discoverability (the existing pattern has `ExampleHTMXScriptHandler()`)
- **Playwright/browser E2E tests** — still not started (same as previous session)
- **IDB graceful-degradation test** — code path exists, no test verifies it
- **ADR update** — ADR-0040 doesn't mention the extraction or the new `dead` message protocol

---

## d) TOTALLY FUCKED UP

### 1. `SyncWorkerScriptTag()` returns a `<script>` tag for a SharedWorker — THIS IS WRONG

SharedWorkers cannot be loaded via `<script>` tags. They're instantiated programmatically via `new SharedWorker(url)`. The function returns `<script src="/worker.js"></script>` which is useless and misleading. I even added a doc comment saying "Note: the SharedWorker is loaded programmatically... This helper exists for consumers who need to reference the worker URL" — but the function STILL returns a script tag. It should either:

- Not exist at all, OR
- Return just the URL string (not a script tag)

**Severity: Medium.** The function compiles, tests pass, but it's a misleading API that a consumer might try to use and wonder why nothing works.

### 2. wsl_v5 lint warnings on sync_serve_test.go

14 lint warnings (missing whitespace above if statements). The tests pass and are correct, but they don't match the project's wsl_v5 linting standard. Would be caught by `golangci-lint`.

**Severity: Low.** Cosmetic, but violates project quality standards.

---

## e) WHAT WE SHOULD IMPROVE

### High Priority

1. **Remove or fix `SyncWorkerScriptTag()`** — it returns a `<script>` tag for something that can't be loaded via `<script>`. Either delete it or rename to `SyncWorkerURL()` returning just the path.

2. **`syncVersion` cache-busting risk** — `serveJS` sets `Cache-Control: public, max-age=31536000, immutable` (1-year). The ETag includes `syncVersion` ("1.0.0"). If the JS changes but the version isn't bumped, clients cache stale code for a year. There's no automated check that forces a version bump on JS changes. Should add a pre-commit hook or make `syncVersion` a hash of the embedded content.

3. **Script load order in layout.templ** — `sync-client.js` loads before `admin.js`. This works (both run synchronously, sync-client registers event listeners, admin sets CSRF config). But it's fragile — if admin.js ever needs to run before sync-client.js registers its `htmx:beforeRequest` handler (e.g., to modify requestConfig.headers that sync-client reads), the ordering would need to flip. Currently fine, but should be documented or made order-independent.

4. **FEATURES.md not updated for extraction** — the Offline Sync row still reads as an adminui-only feature. Should mention root module handlers and "any consumer" availability.

5. **No godoc examples** — root module has `ExampleHTMXScriptHandler()` and `ExampleHTMXScriptHandlerWith()` but no equivalent for sync handlers. Reduces discoverability.

### Medium Priority

6. **The sync-client.js worker URL derivation assumes same base path** — `script.src.replace(/\/sync-client\.js$/, "") + "/sync-worker.js"`. If a consumer mounts them at different paths, it breaks silently. Should document this constraint or add a `data-sync-worker-url` attribute override.

7. **admin-demo example not verified** — `examples/admin-demo/main.go` may have stale comments about sync-worker being an adminui asset.

8. **Lint warnings in test file** — 14 wsl_v5 violations in `sync_serve_test.go`. Quick fix but violates project standards.

9. **ADR-0040 not updated** — doesn't mention the extraction from adminui to root, or the new `dead` message protocol, or the `hello`/`bye` protocol.

### Low Priority

10. **No version constant on the JS files themselves** — `sync-worker.js` has no `VERSION = "1.0.0"` variable. Hard to debug which version is loaded in a browser DevTools console.

11. **`SyncVersion()` returns a Go string** — not exposed to the client-side. A health-check endpoint that reports the sync asset version would help debugging.

---

## f) Up to 50 Things We Should Get Done Next

### Critical Fixes (do first)

1. **Remove or fix `SyncWorkerScriptTag()`** — delete it or rename to return URL string
2. **Fix wsl_v5 lint warnings in `sync_serve_test.go`** — add missing whitespace
3. **Update FEATURES.md** Offline Sync row for extraction
4. **Verify `examples/admin-demo/main.go`** compiles and works with new asset paths
5. **Add `ExampleSyncWorkerHandler()` / `ExampleSyncClientHandler()`** to `example_htmx_test.go` for godoc

### Testing

6. **Playwright E2E: offline → queue → online → retry → ACK** — #1 open item since Phase 2a
7. **Playwright E2E: cross-session retry** (close tabs, reopen, verify IDB drain)
8. **Playwright E2E: `rebuildAndRetry`** (navigate away, verify synthetic div + htmx.ajax)
9. **Playwright E2E: dead command** (exceed MAX_RETRIES, verify dead UI)
10. **Playwright E2E: multiple tabs** (verify targeted retry, not thundering herd)
11. **JS unit test harness** for sync-worker.js and sync-client.js
12. **JS unit test: flush dedup** (concurrent flush coalesces)
13. **JS unit test: eviction** (TTL + max retries → dead)
14. **JS unit test: null envelope guard**
15. **JS unit test: store.add preserves retry count**

### Root Module Polish

16. **Auto-version the sync assets** — derive `syncVersion` from content hash instead of hardcoded "1.0.0"
17. **Add `data-sync-worker-url` attribute override** to sync-client.js for consumers who mount worker at a different path
18. **Add `VERSION` constant inside JS files** for browser DevTools debugging
19. **Add `/health/sync-version` endpoint pattern** to recipe (optional debug endpoint reporting syncVersion)
20. **Add `SyncBundledHandler()`** — serves both worker + client concatenated (like `HTMXExtensionsHandler`) to reduce HTTP requests

### adminui Polish

21. **Verify layout.templ script ordering is robust** — add comment explaining why sync-client.js before admin.js is correct
22. **Update admin-demo example** if it references stale asset paths
23. **Update adminui README** to mention sync-client.js is now root-served

### Documentation

24. **Update ADR-0040** — mention extraction to root module, dead/hello/bye protocol
25. **Write ADR-0042** — "Offline Sync Extraction to Root Module" (the architectural decision)
26. **Update ADR INDEX.md** with ADR-0042
27. **Update recipe doc** — add "standalone consumer" section (no adminui, just root handlers)
28. **Add CSP section to recipe** — explicit `worker-src 'self'` note (not just `default-src`)
29. **Update doc.go** if it mentions adminui-only sync
30. **Add sync handler examples to doc.go** package-level comment

### Code Quality

31. **Modernize JS to `const`/`let`** — both sync-worker.js and sync-client.js use `var`
32. **Add JSDoc type annotations** to message protocol in sync-worker.js
33. **Add `queuedAt` index to IDB schema** (v2 migration) for efficient TTL queries
34. **Debounce flush on burst enqueue** — coalesce rapid enqueues into one flush
35. **Make `MAX_RETRIES`/`RETRY_TTL_MS`/`STAGGER_MS` configurable** via a global JS object
36. **Add circuit breaker** — back off after N consecutive retry failures
37. **Add queue size limit** — prevent unbounded IDB growth

### Robustness

38. **Heartbeat/ping mechanism** — don't rely solely on `navigator.onLine`
39. **IDB quota handling** — broadcast warning to tabs if persistCommand fails with quota error
40. **Port health check** — periodic heartbeat to detect zombie ports
41. **Graceful worker shutdown** — cancel pending setTimeout retry timers on offline

### Admin UI Polish

42. **Dead command badge** — "Failed after N retries" instead of generic "rejected"
43. **Queue depth indicator** — show exact count (not `Math.max` of stale counts)
44. **Manual flush button** — "Retry all queued now"
45. **Queue viewer** — admin panel showing all queued commands

### Integration

46. **Server-side idempotency documentation** — complete middleware example in recipe
47. **WebSocket integration** — sync worker is SSE-only; WS ACK path would need same coordination
48. **Service Worker fallback** — for browsers without SharedWorker (Safari < 16)
49. **JS lint setup** — add `.oxlintrc.json` with SharedWorker/sync-client globals
50. **Pre-commit hook for JS syntax** — `node --check` on all embedded JS files

---

## g) Questions I Can NOT Figure Out Myself

### Q1: Should `SyncWorkerScriptTag()` be deleted or fixed?

It returns a `<script>` tag for a SharedWorker, which is technically wrong (SharedWorkers can't be loaded via `<script>`). Options: (a) delete it entirely — consumers don't need it, sync-client.js derives the URL automatically; (b) rename to `SyncWorkerURL(path) string` returning just the path for CSP meta tag use; (c) keep as-is with the existing "this is not how you load a SharedWorker" doc comment. I lean (a) — delete it, it's a misleading API.

### Q2: Should we write an ADR for the extraction (ADR-0042)?

The extraction reverses an implicit architectural decision ("sync is an adminui feature"). ADR-0040 covered Phase 2b (IndexedDB persistence) but didn't address module placement. Writing ADR-0042 ("Offline Sync Extraction to Root Module") would document WHY it moved (general CQRS+HTMX concern, not admin-specific) and WHY it's not a separate module (same pattern as HTMXScriptHandler, tightly coupled to HTMX events + SSE ACK protocol). Is this worth an ADR, or is it just "obvious follow-the-pattern"?

### Q3: Should the sync asset version be content-hashed instead of hardcoded?

Right now `syncVersion = "1.0.0"` is a hardcoded string. If someone changes sync-worker.js but forgets to bump the version, clients cache stale JS for 1 year (`Cache-Control: immutable`). Options: (a) keep manual version + add a pre-commit hook that fails if the JS changed but the version didn't; (b) compute a content hash at init time (like `fmt.Sprintf("%x", sha256(syncWorkerJS)[:8])`); (c) keep manual and trust ourselves. (b) is most robust but means every JS change gets a new cache-busting ETag automatically. Is this worth the complexity?
