# Offline Sync E2E Testing — Status Report

**Date:** 2026-07-29 00:17
**Task:** Playwright E2E browser testing of the SharedWorker IndexedDB offline command queue (ADR-0029 + ADR-0040). The #1 deferred item across every sync status report.

---

## Executive Summary

Built the complete E2E test infrastructure (Go test server + Playwright + 4 test scenarios). All 4 tests fail at the enqueue step. **Root cause found:** HTMX's `requestConfig.parameters` is a `FormData` object that `postMessage` cannot clone across the SharedWorker boundary. The sync-client.js has a serialization bug. Fix is ~5 lines.

---

## a) FULLY DONE

### Go E2E Test Server (`e2e/server/`)

- **`e2e/server/main.go`** — Minimal HTTP server serving HTMX + sync-client.js + sync-worker.js + SSE + POST command endpoint + debug endpoint. All endpoints verified working.
- **`e2e/server/go.mod`** — Separate Go module (`github.com/larsartmann/cqrs-htmx/e2e/server`), added to `go.work`.
- Builds clean with `GOEXPERIMENT=jsonv2 go build`. Server starts, serves all routes, handles POST commands, broadcasts SSE ACK events.

### Playwright Infrastructure

- **`e2e/package.json`** — Playwright + TypeScript devDependencies. Installed via bun.
- **`e2e/playwright.config.ts`** — Configured with auto-starting Go server via `webServer`, NixOS Chromium workaround (`E2E_BROWSER_PATH` env var → `launchOptions.executablePath`), single-worker (SharedWorker state is per-origin), HTML reporter.
- **`e2e/tsconfig.json`** — Strict TypeScript config.
- **`e2e/.gitignore`** — Ignores node_modules, test-results, playwright-report.
- **`e2e/tests/sync.spec.ts`** — 4 E2E tests written, all discovered by Playwright.

### Key Discovery: Playwright + Bun + NixOS Gotchas

- Bun is the JS runtime (`bun x playwright test`). npm not available.
- Playwright's downloaded Chromium cannot run on NixOS (no FHS dynamic linker). Must use system Chromium via `E2E_BROWSER_PATH` env var → `launchOptions.executablePath` in config.
- **Critical Playwright transformer limitation:** TypeScript type assertions (`as any[]`, `import('@playwright/test').Page` in function signatures) inside test files cause silent build failures ("No tests found"). Must use string-based `page.evaluate()` or parameterless functions. This cost ~2 hours of debugging.

---

## b) PARTIALLY DONE

### E2E Tests (4 written, all fail at enqueue step)

- **Test 1:** Offline enqueue persists command envelope to IndexedDB — FAILS (queue depth stays 0)
- **Test 2:** Online flush delivers queued command to server — FAILS (depends on test 1)
- **Test 3:** Cross-session rebuildAndRetry delivers and cleans up — FAILS (depends on test 1)
- **Test 4:** Multiple offline commands queued and delivered — FAILS (depends on test 1)

### Root Cause Identified (fix not applied)

**`sync-client.js` line 388-396** captures `cfg.parameters` from the HTMX `sendError` event detail and passes it directly as `envelope.values` to `postMessage`. In HTMX 2.x, `cfg.parameters` is a **`FormData`** object, not a plain `{key: value}` object. `postMessage` throws `Failed to execute 'postMessage' on 'MessagePort': #<FormData> could not be cloned.` — the command never reaches the SharedWorker.

Proven via debug trace:

```
htmx:beforeRequest: elt=FORM closestCmdId=NONE closestSyncState=NONE
htmx:sendError: elt=FORM closestCmdId=<uuid> closestSyncState=pending
PAGEERROR: Failed to execute 'postMessage' on 'MessagePort': #<FormData> could not be cloned.
```

Direct SharedWorker enqueue (bypassing sync-client) works perfectly — IndexedDB receives the entry, the worker sends retry messages, everything functions.

**Fix:** Convert `FormData` to plain object before postMessage:

```javascript
// In sync-client.js, htmx:sendError handler:
var params = cfg.parameters;
if (params instanceof FormData) {
	var plain = {};
	params.forEach(function (val, key) {
		plain[key] = val;
	});
	params = plain;
}
envelope = {
	verb: cfg.verb || "",
	url: cfg.path || "",
	values: params,
	headers: cfg.headers || null,
};
```

---

## c) NOT STARTED

- Flaky.nix integration (Chromium in devShell or `e2e` app)
- `e2e/README.md` documentation
- TODO_LIST.md / CHANGELOG.md updates
- CI integration

---

## d) TOTALLY FUCKED UP

### The Playwright Transformer Debugging Odyssey (~2 hours wasted)

Spent far too long isolating why `sync.spec.ts` produced "No tests found" while individual test patterns worked in isolation. Root cause was Playwright's esbuild-based transformer silently rejecting TypeScript type annotations in specific positions:

- `as any[]` type assertions on `page.evaluate()` results → build fails
- `import('@playwright/test').Page` inline type imports in function params → build fails
- Module-level arrow functions with DOM type casts (`e.target as IDBOpenDBRequest`) → build fails

Should have checked Playwright's known issues / docs for transformer limitations immediately instead of binary-searching through 17 test file variants.

### IndexedDB Inspection Approach

The string-based `page.evaluate()` scripts for reading IndexedDB from the page context are fragile. Each test opens a separate IDB connection, which can race with the SharedWorker's own IDB transactions. A better approach would be a dedicated debug endpoint on the server or a SharedWorker-to-test bridge.

---

## e) WHAT WE SHOULD IMPROVE

1. **Fix the FormData serialization bug in sync-client.js** — This is a production bug, not just a test issue. ANY HTMX 2.x form submission that fails offline silently drops the command. The enqueue postMessage throws and the command is lost forever.
2. **Replace IndexedDB with a better persistence layer.** The user pointed to [LiveStore](https://github.com/livestorejs/livestore) as a reference. IndexedDB is notoriously painful — flaky transactions, opaque error handling, no observability. LiveStore's approach (using SQLite via WASM or OPFS) would eliminate the entire class of IDB-related flakiness. This is the right long-term move.
3. **Add a test-only debug API to the SharedWorker** — Instead of the tests reading IndexedDB directly (which races with the worker), expose a `debug` message type that returns the current queue state via `postMessage`. This would make tests deterministic.
4. **Fix Playwright transformer compatibility** — Document the limitations (no type assertions in evaluate, no inline type imports) in a comment block at the top of the test file so the next person doesn't waste 2 hours.
5. **The `data-sync-target` wrapper requirement** — The sync-client's `closest()` chain for finding the sync element depends on DOM structure. The server HTML needed a `<div data-sync-target>` wrapper around the form for the sendError handler to find the right element. This should be documented or made more robust.

---

## f) NEXT 50 THINGS TO DO

### Immediate (blocking)

1. Fix FormData serialization bug in `sync-client.js` (convert FormData to plain object before postMessage)
2. Re-run E2E tests — should go green after the fix
3. Add `FormData` conversion for the `rebuildAndRetry` path too (envelope.values → FormData reconstruction on retry)
4. Clean up `e2e/test-results/` and verify tests pass from clean state

### Short-term

5. Write `e2e/README.md` with setup instructions and NixOS Chromium workaround
6. Add `nix run .#e2e` flake app (Chromium + Playwright + Go server)
7. Add Chromium to flake.nix devShell or as a separate `e2e` devShell
8. Add a `sync:debug` message type to sync-worker.js for test introspection (return queue state without reading IDB)
9. Test: offline command ACK cleans up IndexedDB entry
10. Test: command with missing envelope is marked dead
11. Test: dead command after MAX_RETRIES

### Sync Stack Hardening

12. Evaluate LiveStore approach for replacing IndexedDB persistence entirely
13. Investigate OPFS (Origin Private File System) as IDB alternative — synchronous, simpler API
14. Add `FormData` round-trip test (envelope.values → rebuildAndRetry → server receives correct form data)
15. Add WebSocket transport E2E test (currently only SSE)
16. Test: SharedWorker survives page reload (same worker instance)
17. Test: multiple tabs share one SharedWorker
18. Test: offline indicator UI state transitions (idle → pending → queued → confirmed)
19. Test: aria-live region announcements for screen readers
20. Add CSP header test (sync-worker.js must be served with correct MIME type)
21. Test: SharedWorker unavailable graceful degradation (Safari < 16, private browsing)

### CI / Infrastructure

22. Add E2E to `buildflow` pre-commit hook (or separate `e2e` mode)
23. Add E2E to CI pipeline (needs Chromium in CI runner)
24. Add coverage gate concept for JS files (currently only Go has coverage gates)
25. Pin Playwright version in flake.nix for reproducibility
26. Add `bun.lock` to version control for dependency reproducibility
27. Add HTML report artifact to CI (playwright-report/)
28. Add Playwright trace files to CI artifacts on failure

### sync-client.js / sync-worker.js Improvements

29. Add `FormData` → plain object conversion utility (shared by enqueue + rebuildAndRetry)
30. Add `structuredClone` polyfill check (older browsers)
31. Fix: `rebuildAndRetry` uses `htmx.ajax()` which may not reconstruct FormData correctly
32. Add retry backoff jitter (currently fixed STAGGER_MS = 100ms)
33. Add queue depth limit (prevent unbounded IndexedDB growth)
34. Add command deduplication (same commandId enqueued twice)
35. Add `sync:flush` message type (let tabs trigger a manual flush)
36. Add `sync:status` message type (return worker state for debugging)
37. Add worker lifecycle logging (debug mode)
38. Add IndexedDB quota exceeded handling (currently silently degrades to in-memory)
39. Add IndexedDB schema migration support (DB_VERSION = 1, no upgrade path)
40. Test: IndexedDB unavailable degradation path (private browsing)
41. Test: concurrent flush coalescing (flushPending flag)
42. Test: port death detection (broadcast() catch block)

### Documentation

43. Document the Playwright + Bun + NixOS setup in `e2e/README.md`
44. Document the FormData serialization bug as a CHANGELOG entry when fixed
45. Update `docs/recipes/offline-command-sync.md` with the `data-sync-target` requirement
46. Update ADR-0040 with the FormData limitation finding
47. Add `e2e` section to root `AGENTS.md` Quick Reference table
48. Add `e2e` to `TODO_LIST.md` as completed (once tests pass)
49. Write a `docs/guides/e2e-testing.md` guide for consumers
50. Evaluate whether the sync stack should be extracted to its own repo (like LiveStore)

---

## g) Questions I Cannot Answer Myself

1. **Should I fix the FormData bug in `sync-client.js` right now, or is there a reason this was working before (e.g., HTMX 1.x vs 2.x)?** The sync stack was built against HTMX 2.0.10 (embedded). It's possible `requestConfig.parameters` was a plain object in earlier HTMX 2.0 builds and changed to FormData in a patch. If this is a regression from an HTMX upgrade, the fix belongs in sync-client.js (convert FormData), not in HTMX.

2. **Should the IndexedDB persistence layer be replaced with a LiveStore-style approach (SQLite WASM / OPFS)?** The user explicitly flagged IndexedDB as problematic and pointed to LiveStore. This is a significant architectural decision (ADR-level) that affects ADR-0040 and potentially ADR-0029. It would be a breaking change to the sync-worker.js contract. Should I write an ADR proposing the migration, or fix the current IDB approach first and get tests green?

3. **Should the `e2e/` directory be a separate Go module, or should it use the workspace's root module?** Currently it's `github.com/larsartmann/cqrs-htmx/e2e/server` (separate module). The examples are also separate modules. But the e2e tests test the root module's embedded JS — should they live IN the root module as `_test.go` integration tests instead (with Playwright triggered from Go)?

---

## File Inventory

### Created this session

| File                       | Purpose                                                           | Status                  |
| -------------------------- | ----------------------------------------------------------------- | ----------------------- |
| `e2e/server/main.go`       | Go test server (HTMX + sync handlers + SSE + POST endpoint)       | Working                 |
| `e2e/server/go.mod`        | Separate Go module for the test server                            | Working                 |
| `e2e/package.json`         | Playwright + TypeScript dependencies                              | Installed               |
| `e2e/playwright.config.ts` | Playwright config with webServer + NixOS Chromium workaround      | Working                 |
| `e2e/tsconfig.json`        | TypeScript strict config                                          | Working                 |
| `e2e/.gitignore`           | Ignores node_modules, test-results                                | Working                 |
| `e2e/tests/sync.spec.ts`   | 4 E2E tests (offline enqueue, online flush, cross-session, multi) | All fail (FormData bug) |
| `go.work` (modified)       | Added `./e2e/server` to workspace                                 | Done                    |

### Root cause artifacts

| Finding                                         | Evidence                                                                                        |
| ----------------------------------------------- | ----------------------------------------------------------------------------------------------- |
| `FormData` cannot be cloned via postMessage     | `PAGEERROR: Failed to execute 'postMessage' on 'MessagePort': #<FormData> could not be cloned.` |
| Direct SharedWorker enqueue works               | Debug test 14 proved IndexedDB receives entries when bypassing sync-client                      |
| Manual sendError dispatch works                 | Debug test 16 proved enqueue works when `values` is a plain object                              |
| HTMX 2.x `requestConfig.parameters` is FormData | Debug test 17 traced the full event lifecycle                                                   |
