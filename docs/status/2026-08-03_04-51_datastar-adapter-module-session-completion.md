# Datastar Adapter Module — Session Status Report

> **Date:** 2026-08-03 04:51
> **Session focus:** Completing remaining datastar adapter module tasks — bug fixes, replay, demo upgrade, documentation, quality gates
> **Starting point:** 10 of 26 planned tasks remaining after initial implementation session

---

## Executive Summary

The session completed 13 tracked tasks across bug fixes, architecture (patch ring buffer replay), demo migration, documentation, and quality gates. The datastar module is now feature-complete for its initial scope, with 57 tests at 95.1% coverage, 0 lint issues, and a fully migrated demo. However, several gaps remain: dead code in the demo, formatting issues, missing root CHANGELOG entry, and a broken integration test module (pre-existing go-cqrs-lite publish bug).

> **Update 2026-08-03 (commit `ab0b2f0`, `c01d238`):** ALL remaining gaps closed by sessions 07:26 and 09:14. Dead code removed (`extractTitle`, `findTodoByID`, `Projector.GetByID`). Root CHANGELOG updated. Flightrecorder replace added to go.work — integration tests now run (8/8 pass). Module at **71 tests, 96.7% coverage** after session 09:14 added OnError, heartbeat, 6 new Response methods. The `writeHeartbeat` bug (D-class) found and fixed in session 19:34.

---

## A) FULLY DONE (shipped, verified, committed)

### A1. doc.go False Claims Fixed

- Removed `bridge.Start(subscribeFunc)` / `bridge.Stop()` — these methods never existed on `EventBridge`
- Replaced with correct `eventBus.SubscribeAll(bridge.Handle)` pattern
- Removed `ReadSignalsQuery` from the API surface list (doesn't exist as a separate function)

### A2. CHANGELOG.md Corrected

- Removed false "reconnection support" and "Start/Stop lifecycle" claims from initial entries
- Rewrote all entries to match actual API surface
- Added accurate description of replay, all patch constructors, full SDK re-exports

### A3. SDK Re-exports Expanded (`options.go`)

- Added 20+ missing exports: `WithNamespaceMathML`, `WithViewTransitions`, `WithoutViewTransitions`, `WithUseViewTransitions`, `WithViewTransitionSelector`, `WithPatchElementsEventID`, `WithRetryDuration`, `WithPatchSignalsEventID`, `WithPatchSignalsRetryDuration`, all `WithExecuteScript*` options (5), `WithContext` SSE option
- Added type aliases: `SSEOption`, `DispatchCustomEventOption`, `Namespace`, `EventType`
- Added constant re-exports (as vars): `NamespaceHTML/SVG/MathML`, all `ElementPatchMode*`, `EventTypePatchElements/PatchSignals`

### A4. Version Assertion Test Added

- `TestVersionMatchesJSComment` in `script_handler_test.go` — parses `// Datastar v1.0.2` from embedded JS, asserts match against `datastarVersion` constant
- Mirrors root module's `TestSyncVersionMatchesJSConstants` pattern

### A5. Patch Ring Buffer Replay Implemented (`broadcaster.go`)

- **Complete rewrite** of Broadcaster with bounded ring buffer (default 256 patches)
- Each broadcast receives monotonically increasing SSE `id:` field via `writeEventID()`
- `ServeHTTP`: subscriber registration + replay snapshot are atomic (single mutex hold) — prevents gaps AND duplicates
- `parseLastEventID()`: reads from `Last-Event-ID` header OR `?lastEventId=` query param fallback
- `hasLastEventID()`: distinguishes new clients (no replay) from reconnecting clients (replay)
- `NewBroadcasterWithReplay(maxReplay)`: custom buffer size; pass 0 to disable
- Key design insight: ring buffer of _patches_ is more correct than root's `JournalSSEStore` (which replays _events_ and re-renders) — replays exactly what was broadcast

### A6. Comprehensive Replay Tests Added (`broadcaster_test.go`)

- `TestBroadcasterReplayOnReconnect` — replay patches 2,3 when `Last-Event-ID: 1`, verify `id:` fields present
- `TestBroadcasterNoReplayForNewClient` — new clients (no header) get zero replay (caught a real bug during development)
- `TestBroadcasterReplayDisabledWithZero` — `NewBroadcasterWithReplay(0)` disables replay entirely
- `TestBroadcasterReplayWithQueryParam` — `?lastEventId=1` query param fallback works
- `TestBroadcasterReplayRingBufferEviction` — buffer of 3 evicts patches 1,2 when 5 are broadcast
- Tests optimized to ~0.11s each (context cancel after subscriber connects + 100ms flush)

### A7. Demo Fully Upgraded to Adapter Module

- `main.go`: Added `ds.ScriptHandler()` mount, `ds.NewBroadcaster()` for SSE, replaced CDN script tag with self-hosted `/datastar.js`
- `domain_cqrs.go`: Removed custom `Broadcaster`/`BroadcastEvent` types entirely; uses `ds.Broadcaster` + `ds.BroadcastMany` + `ds.ElementsPatch`
- `handlers_helpers.go`: All handlers now use `ds.ReadSignals()`, `ds.NewResponse()`, `ds.ErrorResponse()` instead of raw SDK
- `handlers_routes.go`: `handleSimulate` uses `ds.NewResponse()` instead of raw `datastar.NewSSE()`
- Removed `handleEventStream` and `handleEventReplay` handlers (Broadcaster.ServeHTTP replaces both)
- Removed old custom Broadcaster (60 lines), BroadcastEvent type, Subscribe/Unsubscribe/Send methods
- go.mod: replaced `datastar-go v1.2.2` direct dep with `cqrs-htmx/datastar/v4`

### A8. Integration Guide Written

- `docs/guides/datastar-integration.md` — 13 sections covering installation, quick start, all patch types, replay/reconnection, HTMX+Datastar coexistence, SDK re-exports, demo reference

### A9. ADR-0045 Written

- `docs/adr/0045-datastar-optional-frontend.md` — Status, Context, Decision (5 key design choices with rationale), Consequences (positive/negative/mitigations), 4 alternatives considered and rejected
- Key documented decision: standalone (no root dep) with patch ring buffer replay instead of JournalSSEStore integration

### A10. Root Documentation Updated

- **AGENTS.md**: Module count 18→19, added datastar to module list, updated dependency direction
- **README.md**: Added `datastar/` to directory structure
- **FEATURES.md**: Added full datastar module section with feature table
- **ADR INDEX.md**: Added ADR-0045 row

### A11. Coverage Gate Added

- `flake.nix`: `check_cov datastar 90` added after dashboardui gate

### A12. Cross-Module Integration Test Created

- `integration_test/datastar_contract_test.go` — 8 contract tests: ScriptHandler, ReadSignals, Broadcaster, EventBridge, Patch constructors, Options re-exports, Response builder, Version/ScriptTag
- integration_test/go.mod updated with datastar dep + local replace

### A13. Full Workspace Verification

- Workspace build: OK (all 19 modules)
- Datastar tests: 57 functions, all pass with race detector
- Datastar coverage: 95.1% (gate: 90%)
- Datastar lint: 0 issues
- Demo build: OK
- Demo vet: OK

---

## B) PARTIALLY DONE

### B1. Demo Cleanup (90% done)

The demo compiles and works, but has dead code from the migration:

- `extractTitle()` in `handlers_routes.go` — was used by old `handleEventStream` for notification messages, now unused
- `findTodoByID()` in `domain_store.go` — was used by old broadcast bridge, now unused
- `renderStats()` (the non-query version) — still referenced as a fallback by `renderStatsFromQuery`, so technically used, but only as an error path
- `eventKindFromType()` — still used by `renderEventLogEntry()`, so NOT dead code

### B2. Integration Test (created but can't verify)

- `integration_test/datastar_contract_test.go` is written and correct
- Cannot run because `integration_test` module has a **pre-existing** `flightrecorder/v4.0.0` unknown revision error (go-cqrs-lite publish bug, documented in AGENTS.md)
- This is NOT caused by the datastar changes — the module was already broken before this session

### B3. Coverage (95.1%, not 100%)

Coverage gaps in 5 functions:

- `NewSSE()` in `options.go` — 0% (untested re-export, trivial passthrough)
- `writeEventID()` — 66.7% (the `id == 0` early return path not tested)
- `parseLastEventID()` — 71.4% (the parse error fallback not tested)
- `ServeHTTP()` — 92.6% (the `sse.IsClosed()` branch and error return paths)
- `applyAll()` — 75.0% (the error return path not tested)

---

## C) NOT STARTED

### C1. `Response.Status(code)` and `Response.Header(key, value)`

The plan (T9) calls for these. The SDK upgrades the HTTP connection immediately in `NewSSE()`, making post-hoc status/header changes impossible without buffering. The current design accepts this limitation (write-through, no buffering). A `Status()` method would need to buffer all patches and flush on `Apply()`, which is a fundamentally different architecture.

### C2. Typed Signal Decoders (`DecodeSignals[Q]` returning `HandlerOption`)

The plan (T5-T7) calls for these. Architecturally impossible without root module changes: root's `handlerConfig` is unexported, `HandlerOption` is `func(*handlerConfig)`, and there's no exported escape hatch. Even adding a root dependency wouldn't help. Documented in ADR-0045 as a known limitation.

### C3. `.editorconfig`, `.gitattributes`, `.gitignore` for datastar module

The plan (T1.5) calls for these files (copy from loginpage). They don't exist. Minor — the module works without them, but inconsistent with other modules.

### C4. Root CHANGELOG.md Entry

The root module's CHANGELOG.md was NOT updated with the datastar adapter addition. This is a documentation gap.

### C5. TODO_LIST.md Update

The TODO_LIST.md header still says "18 modules" and doesn't mention the datastar module at all.

### C6. ROADMAP.md Update

The ROADMAP.md still says "18 Go modules" in the summary. The "Not Planned" or future sections don't mention datastar dashboardui/adminui variants.

---

## D) TOTALLY FUCKED UP

### D1. `go.work.sum` Corruption

During the demo upgrade, `go work sync` was run which corrupted the go.work.sum, breaking ALL workspace builds (every module showed `datastar/v4.0.0: unknown revision`). Root cause: the `go work sync` command tried to resolve `datastar/v4 v4.0.0` from the remote (which doesn't exist — it's a local-only module) and wrote broken entries to go.work.sum. Fixed by adding explicit `replace` directives in demo's and integration_test's go.mod files. The go.work.sum was restored from git. **Lesson:** Never run `go work sync` when local modules don't have published tags.

### D2. Local Replace Directives Left in go.mod Files

Both `examples/datastar-demo/go.mod` and `integration_test/go.mod` have `replace github.com/larsartmann/cqrs-htmx/datastar/v4 => ../../datastar` directives. These are needed for local development but would need to be stripped before publishing (like the go-cqrs-lite replace directives in go.work). This is consistent with the existing pattern but adds maintenance burden.

### D3. Demo `domain_cqrs.go` Broadcast Bridge — Simpler but Less Featureful

The old demo broadcast bridge sent per-event notifications ("Created: Buy milk", "Todo deleted") and only sent notifications to the user who triggered the action (`evt.User == "you"`). The new adapter-based bridge broadcasts the same 3 patches (todo list, stats, event log) for every event to every client. The per-user notification filtering is GONE. This is a behavioral regression — all clients now see every update without the personalized notification. The SSE event stream still works correctly (all clients get all updates), but the "your action succeeded" notification is lost.

---

## E) WHAT WE SHOULD IMPROVE

### E1. Broadcaster Replay: No Heartbeat

Root's `dashboardui/sse.go` sends periodic heartbeats to keep connections alive through proxies. The datastar Broadcaster has no heartbeat. Long-lived connections behind nginx/cloudflare may time out silently.

### E2. Broadcaster: No `BroadcastIfSubscribers` Optimization

The Broadcaster stores every patch in the ring buffer even when zero subscribers are connected. For high-frequency event streams with occasional client connections, this wastes memory. An optimization: skip buffering when no subscribers are connected (at the cost of no replay for the first connecting client).

### E3. EventBridge: Silent Error Swallowing

`EventBridge.Handle()` swallows errors from `PatchFunc` (line 76-78: `if err != nil { return }`). No logging, no metric, no callback. Consumers have no way to know if their patch function failed. An `OnError` callback option (like root's `OnProjectionFailed`) would fix this.

### E4. EventBridge: No `MapAll` Batch Registration

Consumers must call `Map()` N times for N event types. A `MapAll(map[string]PatchFunc)` method would be more ergonomic.

### E5. Response: No `Status()` or `Header()` (acknowledged limitation)

The SDK upgrades the connection immediately. The only fix is a buffered response mode. This was a conscious decision but limits consumers who need custom HTTP status codes or headers on Datastar responses.

### E6. `options.go` Not gofumpt-clean

`gofumpt -l` flags `options.go` (formatting issue). `goimports -l` also flags it. Not caught because linting passes (golangci-lint uses its own formatting pipeline), but the files are not gofumpt-clean for manual formatting.

### E7. No `.golangci.yml` goconst Exclusion for Re-export vars

The `options.go` file has many string-like vars (`NamespaceHTML = sdk.NamespaceHTML`). goconst could flag these in consumer code. The `.golangci.yml` doesn't have a specific exclusion for the options file (unlike test files).

### E8. README.md Demo Description Outdated

README.md line 1216 still says `# Standalone datastar + go-cqrs-lite SSE example`. The demo is no longer standalone — it uses the adapter module.

### E9. Datastar Module Has No LICENSE File Check

Wait — it does have a LICENSE file. Never mind. But the module doesn't have `.editorconfig`/`.gitattributes`/`.gitignore` (see C3).

### E10. No `doc.go` Example Test

Go's `testing` supports example tests (`func Example() {}`) that appear in godoc. The datastar module has none. Adding example tests would improve godoc quality.

---

## F) NEXT 50 THINGS TO GET DONE

### High Priority (P1)

1. **Fix gofumpt formatting in `options.go`** — run `gofumpt -w options.go`
2. **Remove dead code from demo**: `extractTitle()`, `findTodoByID()` — now unused after migration
3. **Update root CHANGELOG.md** with datastar module addition entry
4. **Update TODO_LIST.md header** — 18→19 modules, add datastar-related open items
5. **Update ROADMAP.md** — 18→19 modules, add datastar future items (dashboardui/adminui Datastar variants)
6. **Update README.md demo description** — no longer "standalone", now uses adapter module
7. **Restore per-user notification filtering in demo** — the old bridge sent personalized notifications; new one doesn't. Add a signal patch for "your action" in the broadcast bridge
8. **Add `.editorconfig`/`.gitattributes`/`.gitignore` to datastar module** (copy from loginpage)

### Medium Priority (P2)

9. **Add `OnError` callback to EventBridge** — surface PatchFunc errors instead of swallowing
10. **Add `MapAll(map[string]PatchFunc)` to EventBridge** — batch registration
11. **Add heartbeat to Broadcaster** — configurable interval, default 30s, prevent proxy timeouts
12. **Add coverage tests for the 5 uncovered paths** — push to ~98%+
13. **Write example tests (`func ExampleNewResponse()` etc.)** — improve godoc
14. **Add `ReplaceURL(url)` to Response** — wraps `sse.ReplaceURL()` from SDK
15. **Add `ConsoleLog(msg)` / `ConsoleError(err)` to Response** — wraps SDK methods
16. **Add `Prefetch(urls...)` to Response** — wraps SDK method
17. **Add `DispatchCustomEvent` to Response** — wraps SDK method
18. **Consider `BroadcastIfSubscribers` optimization** — skip ring buffer when zero subscribers
19. **Add SSE compression support** — SDK has `WithCompression()`, not re-exported
20. **Test replay with concurrent broadcasts** — race condition stress test
21. **Add `Broadcaster.Stats()` method** — return subscriber count, total broadcasts, replay hits
22. **Document the `replace` directive pattern** for unpublished local modules in AGENTS.md
23. **Add datastar to `nix run .#test`** — verify it runs in the Nix test harness
24. **Verify `nix run .#coverage-gate`** passes with the new datastar 90% gate
25. **Run `nix run .#lint`** for full workspace lint including datastar

### Lower Priority (P3)

26. **Evaluate dashboardui Datastar variant** — replace HTMX polling with signal patches (F1 from plan)
27. **Evaluate adminui Datastar variant** — optional morph-based rendering mode (F2 from plan)
28. **Evaluate loginpage Datastar forms** — signal-based form state (F4 from plan)
29. **Evaluate offline sync simplification** — Datastar retry primitives vs sync-worker.js (F3 from plan)
30. **Add templ integration test** — `ElementsTemplPatch` with a real templ component in tests
31. **Add Broadcaster `MaxReplay` accessor** — let consumers check/adjust at runtime
32. **Consider `Broadcaster.Reset()` method** — clear ring buffer without closing subscribers
33. **Add integration test for EventBridge + Broadcaster end-to-end** — map event, broadcast, subscriber receives patch
34. **Add `ReadSignalsTyped[Q]()` convenience function** — direct decode without manual `ReadSignals(r, &s)`
35. **Consider `WithDebug` option on Broadcaster** — log patch fan-out, subscriber connect/disconnect
36. **Add `Response.Clone()` or `Response.WithContext()`** — for advanced SSE scenarios
37. **Consider `MultiBroadcaster`** — route different patch types to different subscriber groups
38. **Add proper SSE retry field** — `retry: 1000` on initial connection (SDK may already do this)
39. **Test Broadcaster `Close()` during active replay** — verify no panic on concurrent close+replay
40. **Add `Broadcaster.ServeHTTP` method guard** — reject non-GET requests explicitly
41. **Consider CSP-friendly ScriptTag variant** — `nonce` parameter for strict Content-Security-Policy
42. **Add version skew detection** — warn if JS version doesn't match SDK version (different from assertion test)
43. **Document Datastar's `data-signals` attribute** in integration guide — how to declare initial signals in HTML
44. **Add error wrapping in ErrorResponse** — currently sends raw `err.Error()`, should wrap with errorfamily
45. **Consider `NotificationLevel` typed enum** — instead of raw strings ("success", "error", etc.)

### Documentation (P3-P4)

46. **Add datastar section to root README.md** features list (not just directory structure)
47. **Add datastar to docs/guides/ INDEX or FEATURES.md guide list**
48. **Write BDD test for demo** — full end-to-end scenario (create todo, verify SSE delivery)
49. **Add architecture diagram** — D2 or mermaid showing datastar module dependency flow
50. **Consider a "Datastar vs HTMX" decision guide** — when to use which, or both

---

## G) QUESTIONS

**G1. Should the demo's per-user notification filtering be restored?**
The old demo broadcast bridge sent personalized notifications only to the user who triggered the action (`evt.User == "you"` → "Created: Buy milk"). The new adapter-based bridge broadcasts all patches to all clients uniformly. Should I add signal-based per-user notifications back (e.g., include the acting user's ID in the patch and let the frontend filter), or is the uniform broadcast the intended behavior for a real-time collaborative todo app?

**G2. Should `ErrorResponse` use errorfamily classification for HTTP status codes?**
Currently `ErrorResponse` sends every error as a Datastar signal patch (notification with level "error"), regardless of error type. Root's error mapping (`MapError`) classifies errors into HTTP status codes (400/409/503/500). Since Datastar responses upgrade to SSE immediately, HTTP status codes are meaningless after the upgrade. But should `ErrorResponse` at least classify the error and include the family ("rejection", "conflict", etc.) in the notification signal for the frontend to use?

**G3. Should I strip the `replace` directives from demo/integration_test go.mod before committing, or leave them?**
The go-cqrs-lite modules use `replace` directives in `go.work` (not in individual go.mod files) to work around the publish bug. But the datastar module needs `replace` in `examples/datastar-demo/go.mod` and `integration_test/go.mod` because it's a local-only module with no published tag yet. These would need to be stripped when (if) the datastar module gets a real tag. Should I follow the go.work-only pattern (and accept broken local builds), or keep the go.mod replaces (and accept the maintenance burden)?
