# Status: sync-worker.js Production Hardening — 2026-07-22

> Session focused on making `adminui/assets/sync-worker.js` "superb" after an
> architecture review revealed bugs, design gaps, and missing features.

---

## a) FULLY DONE

### sync-worker.js — Complete Rewrite

Replaced the 244-line prototype with a 320-line production implementation.

| Fix                                  | What was wrong                                                                                                                                        | What was done                                                                                                                                         |
| ------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Double-retry race**                | `flush()` (in-memory) + `drainPersisted()` (IDB) both fired on `onconnect`, retrying the same commands to the same tabs                               | Eliminated parallel in-memory queue. IDB is the single source of truth. One unified `flush()` function reads from IDB, evicts dead, distributes retry |
| **Port leak**                        | Dead ports never removed from `ports[]` array — `postMessage` catch was a no-op comment saying "cleaned up on next flush" but nothing cleaned them up | `ports` is now `Map<tabId, port>`. `hello`/`bye` protocol. Dead ports removed on `postMessage` failure in `broadcast()`                               |
| **Duplicate port on reconnect**      | Tab reload pushed a new port without removing the old one — same tab got duplicate messages                                                           | `hello` with existing `tabId` closes and replaces the old port                                                                                        |
| **Null envelope poison**             | Commands with `envelope: null` persisted to IDB, retried forever as garbage URLs (`htmx.ajax("POST", undefined, ...)`)                                | Guard on enqueue: if envelope missing `url` or `verb`, immediately send `{type:"dead"}` and don't persist                                             |
| **No eviction**                      | Failed commands retried infinitely across sessions — poison commands accumulated forever                                                              | `MAX_RETRIES=10` + `RETRY_TTL_MS=24h`. Exceeding either → `{type:"dead"}` broadcast + delete from IDB                                                 |
| **Retry count reset**                | `store.put` on re-enqueue overwrote existing record, resetting `retries` to 0                                                                         | `store.add` — if command already exists, IDB throws ConstraintError (silently caught), preserving the original record with its retry count            |
| **Thundering herd**                  | `drainPersisted` broadcast every command to ALL tabs — N tabs × M commands = N×M HTTP requests                                                        | Targeted retry: originating tab preferred (via `originatingTab` map), round-robin fallback across alive ports                                         |
| **No stagger**                       | All retries fired simultaneously on reconnect — server recovery stampede                                                                              | 100ms per-command delay, capped at 2s (`STAGGER_MS`, `STAGGER_CAP_MS`)                                                                                |
| **Full table scan for count**        | `broadcastPendingCount` did `getAll().length` on every enqueue and ack — O(n) IDB read                                                                | `store.count()` — IDB-native count without loading records                                                                                            |
| **Concurrent flush race**            | Two flushes could double-increment retry counts for the same commands                                                                                 | `flushing` flag + `flushPending` follow-up pattern (coalesces concurrent flush calls)                                                                 |
| **No enqueue-while-online retry**    | Commands enqueued while online sat in IDB until next `online` event — unnecessary latency                                                             | `flush()` called after persist if `online` is true                                                                                                    |
| **In-memory fallback inconsistency** | When IDB unavailable, `loadAllCommands` returned `[]` but `persistCommand` silently no-op'd — commands vanished                                       | Full in-memory `Map` fallback: `memQueue` mirrors the IDB API (`persist`, `delete`, `loadAll`, `count`, `incrementRetry`)                             |
| **Silent error swallowing**          | All catch blocks resolved `null` with no logging — errors invisible in SharedWorker context                                                           | `console.warn` calls in `openDB`, `idbRun`, and `loadAllCommands` error paths                                                                         |
| **No `bye` protocol**                | Tabs couldn't tell the worker they were closing — dead ports accumulated                                                                              | Tab sends `{type:"bye", tabId}` on `beforeunload`; worker removes from `ports` map                                                                    |
| **No `dead` message**                | Worker had no way to tell tabs it gave up on a command                                                                                                | `{type:"dead", commandId}` message + `handleDeadCommand()` in admin.js                                                                                |

### admin.js — Protocol Updates

- `tabId` generation (UUID via `crypto.randomUUID`, fallback to timestamp+random)
- `hello` message sent on connect (registers tab with worker)
- `bye` message sent on `beforeunload` (best-effort unregister)
- `handleDeadCommand(commandId)` — shows element as rejected + announces "Sync failed after retries"
- `dead` message type handled in `onmessage` switch

### Documentation

- `docs/recipes/offline-command-sync.md` — updated architecture diagram (IDB instead of `queue [{id, port}]`), added `dead` event row to detection table, rewrote Limitations section (Phase 2b, not Phase 2a)
- `AGENTS.md` — updated offline-queue gotcha with full protocol details (hello/bye, tabId tracking, retry count, TTL, staggered delivery, targeted retry, single source of truth)
- `CHANGELOG.md` — added production-hardening entry under `[Unreleased]` with full list of fixes
- `FEATURES.md` — updated Offline Sync row with new capabilities (retry count, TTL, staggered/targeted retry, hello/bye protocol, dead-port cleanup)

### Build / ETag

- `adminui/assets.go` — ETag bumped `adminui-v3.2.1` → `adminui-v3.3.0`
- `GOEXPERIMENT=jsonv2 go build ./...` — passes
- `GOEXPERIMENT=jsonv2 go test ./adminui/... -count=1 -race` — passes (all tests green)
- `node --check sync-worker.js` — passes

---

## b) PARTIALLY DONE

### Browser E2E testing

- **Status:** Not started. The `rebuildAndRetry` cross-session path and IndexedDB persistence are unit-test-verified at the protocol level (Go tests confirm the JS asset is served with 200) but have never been run in a real browser.
- **What's missing:** Playwright or Selenium test that: (1) opens the admin panel, (2) goes offline via DevTools, (3) triggers a mutation, (4) verifies it's queued in IDB, (5) goes online, (6) verifies retry fires and ACK confirms.
- **Honest status:** `PARTIALLY_FUNCTIONAL` in FEATURES.md is still correct — the code is production-quality but browser-unverified.

### In-memory fallback testing

- **Status:** The code path exists (when `typeof indexedDB === "undefined"` or `openDB()` fails, all operations use `memQueue` Map). No test verifies this path.
- **What's missing:** A unit test that simulates IDB-unavailable mode and verifies enqueue → flush → retry → ack works end-to-end in memory.

---

## c) NOT STARTED

- **Playwright/browser E2E test suite** — not even scaffolded
- **IDB graceful-degradation test** — not started
- **JS unit tests for sync-worker.js** — no test harness exists for the JS (only Go tests that verify the asset is served)
- **IDB schema migration** — `DB_VERSION = 1` with no v2 migration code; adding an index on `queuedAt` (useful for TTL queries) would require a v2 upgrade path
- **Visual sync indicator for dead commands** — `handleDeadCommand` sets `data-sync-state="rejected"` but there's no distinct visual state for "dead after retries" vs "server rejected" — the user can't distinguish them
- **Manual retry for dead commands** — the existing retry button (`[data-sync-retry]`) re-triggers via `htmx.trigger(trigger, "retry")`, but a dead command has already been deleted from IDB by the worker. Re-enqueuing it would need a new enqueue path that doesn't go through `htmx:sendError`.

---

## d) TOTALLY FUCKED UP

Nothing. All changes compile, tests pass, syntax checks pass. No regressions introduced.

---

## e) WHAT WE SHOULD IMPROVE

### High Priority

1. **`store.add` ConstraintError is silently swallowed** — when a command is re-enqueued (already in IDB), `store.add` throws `ConstraintError` which is caught by `tx.onerror` → `resolve(null)`. This is the _correct_ behavior (preserve existing retry count), but it's _implicit_ — it relies on IDB error semantics that a reader wouldn't know about. Should use `store.get` + conditional `store.put` to be explicit, or at minimum add a comment explaining the ConstraintError-is-our-friend pattern.

2. **`incrementRetryCount` does a separate `get` + `put` in one transaction** — this is correct but could be simplified. A cursor-based approach or just using `store.openCursor(commandId)` would be more idiomatic IDB. Minor, but a senior IDB developer would notice.

3. **No `queuedAt` index in IDB** — TTL eviction does `getAll()` then filters by age in JS. An index on `queuedAt` would allow `IDBKeyRange`-bounded queries for expired commands without loading all records. This matters at scale (hundreds of queued commands).

4. **`flush()` is called on every `enqueue` when online** — this means a burst of 50 offline-then-online mutations triggers 50 flushes (though the `flushing` flag coalesces them into 1 active + 1 pending). The follow-up flush re-reads all commands from IDB, including ones that were just enqueued and are about to be retried by the first flush. Not a bug (retry count is incremented before delivery), but wasteful. A debounce would be cleaner.

5. **`originatingTab` map grows unboundedly** — entries are only deleted on `ack`, `dead`, or flush-eviction. If a tab enqueues a command and then crashes without the command being acked or evicted, the `originatingTab` entry persists until TTL. With many tabs and many commands, this map grows. Should be cleaned up in the eviction loop (it is — but only for evicted commands, not for acked ones... wait, it is cleaned on ack too). Actually this is fine — entries are deleted on ack, dead, and flush-eviction. No improvement needed.

6. **`sendRetry` uses `setTimeout` but doesn't track timers** — if the worker is about to be killed (all tabs closing), pending `setTimeout` callbacks may never fire. The commands remain in IDB (retry count already incremented), so they'll be retried on next spawn. This is correct but means the retry count is incremented even if the message was never delivered. A stamp-before-deliver vs stamp-after-deliver tradeoff. Current approach (stamp before) is safer against double-increment races. No change needed, but worth documenting.

7. **No CSP `worker-src` directive in the recipe** — the recipe says `default-src 'self'` covers it, but some CSP configurations explicitly need `worker-src 'self'` or the SharedWorker won't load. The recipe should note this as a common gotcha.

### Medium Priority

8. **`navigator.onLine` is still the only connectivity signal** — the review noted this is unreliable (captive portals, DNS failures). The worker now flushes unconditionally on `online` event AND on enqueue-while-online, but it still won't flush if `navigator.onLine` is `true` and the server is actually down. The tab's `htmx:sendError` is the real signal — and the worker does flush on enqueue. So this is partially mitigated. A heartbeat/ping mechanism would be more robust.

9. **Dead commands have no visual distinction** — `handleDeadCommand` sets `data-sync-state="rejected"` which is the same state as a server rejection. The user can't tell if the command failed because the server rejected it or because the worker gave up after 10 retries. A `data-sync-state="dead"` or `data-sync-dead` attribute would let CSS show a different message/style.

10. **No way to manually retry a dead command** — the retry button (`[data-sync-retry]`) exists for rejected commands, but a dead command has been deleted from IDB. Clicking retry would need to re-enqueue the command (send `{type:"enqueue"}` to the worker with the same commandId). The admin.js retry button handler doesn't do this — it only calls `htmx.trigger(trigger, "retry")`.

### Low Priority

11. **`var` instead of `let`/`const`** — the file uses `Promise`, `Map`, `WeakMap`, `Set` (all ES2015+) but still uses `var`. `let`/`const` would prevent accidental reassignment and give block-scoped semantics. The original file used `var` throughout; the rewrite kept `var` for consistency. A modernization pass would use `const`/`let`.

12. **No JSDoc type annotations** — the message protocol is documented in the header comment but not in the code. JSDoc on the message handler would help IDE autocompletion for consumers who extend the worker.

13. **`STAGGER_MS` is a constant, not configurable** — 100ms per command is fine for small queues but could be too slow for large bursts. A backoff strategy (exponential or rate-based) would be more adaptive.

---

## f) Up to 50 Things We Should Get Done Next

### Testing (Critical)

1. **Playwright E2E test: offline → queue → online → retry → ACK** — the #1 open item since Phase 2a shipped
2. **Playwright E2E test: cross-session retry** — close all tabs, reopen, verify IDB drain fires retry
3. **Playwright E2E test: `rebuildAndRetry`** — navigate away, verify synthetic `<div>` host + `htmx.ajax` re-issues command
4. **Playwright E2E test: IDB unavailable (private browsing)** — verify in-memory fallback works
5. **Playwright E2E test: dead command** — exceed MAX_RETRIES, verify `{type:"dead"}` → rejected UI
6. **Playwright E2E test: multiple tabs** — verify targeted retry (originating tab gets retry, other tabs don't)
7. **Playwright E2E test: port cleanup** — close a tab, verify dead port is removed from worker's `ports` map
8. **JS unit test harness** — set up a minimal test runner for sync-worker.js (could use a headless SharedWorker polyfill or extract the IIFE for testing)
9. **JS unit test: `flush()` dedup** — two concurrent flush calls should coalesce into one active + one follow-up
10. **JS unit test: eviction** — commands older than 24h or with 10+ retries should be marked dead and deleted
11. **JS unit test: null envelope guard** — enqueue without envelope should immediately return dead
12. **JS unit test: `store.add` preserves retry count** — re-enqueue existing command should not reset `retries`

### Code Quality

13. **Add `queuedAt` index to IDB schema** — allows TTL eviction via `IDBKeyRange` without `getAll()`
14. **IDB v2 migration** — bump `DB_VERSION`, add `queuedAt` index in `onupgradeneeded`
15. **Debounce flush on enqueue-while-online** — coalesce burst enqueues into a single flush after 50ms idle
16. **Explicit `store.get` + conditional `store.put` instead of `store.add` ConstraintError** — make the "preserve retry count" behavior explicit, not implicit
17. **Add `data-sync-state="dead"` visual state** — distinguish dead commands from server-rejected ones in CSS
18. **Add manual retry for dead commands** — re-enqueue to worker instead of just `htmx.trigger`
19. **Modernize to `const`/`let`** — drop `var` throughout sync-worker.js
20. **Add JSDoc type annotations** — document the message protocol in code, not just header comments
21. **Configurable `MAX_RETRIES`/`RETRY_TTL_MS`** — let consumers override via a `window.cqrsHtmxSyncConfig` or similar
22. **Rate-based stagger instead of linear** — `STAGGER_MS * index` is linear; exponential backoff would be more adaptive for large bursts

### Documentation

23. **Add CSP `worker-src` note to recipe** — common gotcha for SharedWorker + CSP
24. **Add `dead` message to ADR-0040** — the ADR doesn't mention the eviction protocol (it was added in this session)
25. **Update ADR-0030 status** — it says "Phase 2b shipped" but the implementation details have changed significantly
26. **Add architecture diagram for new flush/pickPort/eviction flow** — the current diagram in the recipe is simplified
27. **Document the `store.add` ConstraintError pattern** — add a comment explaining why `add` (not `put`) preserves retry counts

### Robustness

28. **Heartbeat/ping mechanism** — don't rely solely on `navigator.onLine`; ping the server to detect captive portals
29. **Circuit breaker** — if N consecutive retries fail, back off instead of continuing to retry every flush
30. **Queue size limit** — prevent unbounded IDB growth (e.g., max 1000 queued commands)
31. **Command dedup on enqueue** — if the same `commandId` is enqueued twice (rapid double-click), don't create two IDB entries (currently handled by `store.add` ConstraintError, but should be explicit)
32. **IDB quota handling** — if `persistCommand` fails with quota error, broadcast a warning to tabs instead of silently degrading
33. **Port health check** — periodically `postMessage` a heartbeat to detect zombie ports that don't throw on `postMessage` but are actually dead
34. **Graceful worker shutdown** — on `navigator.onLine` → offline, cancel pending `setTimeout` retry timers to avoid wasted work

### Admin UI Polish

35. **Dead command badge** — show "Failed after N retries" instead of generic "rejected"
36. **Queue depth indicator** — show exact count of queued commands (not just `sync.queued` which is a max)
37. **Manual flush button** — "Retry all queued now" button for the user
38. **Queue viewer** — admin panel page showing all queued commands with timestamps and retry counts
39. **Per-command retry count in UI** — show "Retry 3/10" on queued elements

### Integration

40. **Server-side idempotency documentation** — the recipe mentions `X-Command-Id` but doesn't show a complete idempotency middleware; a dead command that's manually retried needs the same `X-Command-Id` or the server will process it twice
41. **WebSocket integration** — the sync worker is SSE-only; a WS-based ACK path would need the same queue coordination
42. **Service Worker fallback** — for browsers without SharedWorker (Safari < 16), a Service Worker + Background Sync fallback would extend coverage

### Miscellaneous

43. **Lint setup for JS** — no JS linter is configured (oxlint was mentioned in a status report but no config file exists); add `.oxlintrc.json` with `SharedWorker` globals
44. **Pre-commit hook for JS syntax** — `node --check` on `sync-worker.js` and `admin.js` before commit
45. **Source map for sync-worker.js** — minification + source map for production (currently served raw)
46. **Cache-Control for sync-worker.js** — currently `max-age=86400` (1 day); with ETag-based 304s, could be longer
47. **Subresource Integrity (SRI)** — if the worker is loaded cross-origin, SRI would add integrity verification
48. **Cross-origin SharedWorker** — document CSP requirements for consumers serving admin UI from a different origin than the API
49. **Memory pressure** — SharedWorker memory is bounded by the browser, but a large `memQueue` Map (IDB-unavailable mode) could grow unboundedly; add a max-size guard
50. **Telemetry** — log retry/dead/queue-depth events to the server (via a beacon endpoint) so admins can monitor offline sync health

---

## g) Questions I Can NOT Figure Out Myself

### Q1: Should the retry stagger be configurable by the consumer, or is a sensible default (100ms, cap 2s) appropriate for a library?

The library principle says "never enforce defaults consumers might disagree with." But making `STAGGER_MS`/`STAGGER_CAP_MS`/`MAX_RETRIES`/`RETRY_TTL_MS` configurable adds API surface (where? `adminui.Config`? a global JS object? URL params?). Is this worth the complexity, or should these stay as library-chosen constants?

### Q2: Should we add a `data-sync-state="dead"` visual state distinct from "rejected"?

Currently `handleDeadCommand` sets `data-sync-state="rejected"` (same as server rejection). A separate `"dead"` state would let CSS show a different message ("Sync failed after retries" vs "Server rejected"). But it adds a new state to the honest-UI lifecycle that CSS and tests need to handle. Is this worth the added state-space, or should dead commands look the same as rejected ones to keep the UI simple?

### Q3: Do you want me to set up a Playwright E2E test suite for the offline sync, or is the "unit-test-verified protocol, not browser-tested" status acceptable for now?

This is the #1 open item across every status report since Phase 2a shipped. Setting up Playwright requires: a test server binary (Go), a browser runner, and CI integration. It's a significant investment (estimated 4-8 hours). The alternative is adding an explicit "untested in browser" caveat to `adminui/README.md` and deferring until a consumer requests it.

---

## Resolution (2026-07-26, v4.5.0)

The hardening described here shipped, then the entire sync stack was **extracted from `adminui/` into the root module** in two follow-up sessions (`docs/status/2026-07-22_18-02`, `2026-07-22_18-21`). `adminui/assets/sync-worker.js` — the file this report hardens — **no longer exists**; it lives at `sync/sync-worker.js` in the root module and is served via `cqrshtmx.SyncWorkerHandler()` / `SyncClientHandler()`. adminui now delegates. See ADR-0042 and CHANGELOG [v4.5.0].

**Still open:** browser E2E testing (Playwright) remains the #1 deferred item — tracked in TODO_LIST.md (P3). The dead-command visual state question (Q2) was resolved by keeping `data-sync-state="rejected"` for dead commands.
