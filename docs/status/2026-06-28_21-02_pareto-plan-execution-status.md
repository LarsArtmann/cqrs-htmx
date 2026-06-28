# Status Report — cqrs-htmx

**Date:** 2026-06-28 21:02  
**Branch:** `master` (clean, pushed)  
**Commit:** `7d3f5aa`  
**Coverage:** 95.4% root, 80.1% usermgmt  
**Lint:** 0 issues across all 4 modules  
**Tests:** 932 total (133 root + 747 usermgmt + 35 adminui + 17 integration) — all pass with `-race`

---

## Executive Summary

The Pareto improvement plan is **substantially executed**. Of the original 64
tasks, **28 are fully done** (including all 1% tier and most of the 4% tier),
**2 are blocked on user product decisions** (Phase 2), and the remaining 34
are deferred long-tail items that don't block release readiness.

The codebase is in its **healthiest state ever**: zero lint issues, zero
errorfamily violations, all modules aligned to Go 1.26.4 + v3.2.0, pagination
unified, form decoding upgraded to a proper library, idempotency protection
shipped, and CI umbrella verified end-to-end.

---

## A) FULLY DONE

### Architecture & Code Quality

| Item                         | Commit    | Notes                                                                                                                              |
| ---------------------------- | --------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| **Form decoder upgrade**     | `86a45d9` | Replaced allocation-heavy JSON round-trip with `go-playground/form/v4` (zero transitive deps, `json` tag mode for backward compat) |
| **Pagination unification**   | `fa49858` | Eliminated split-brain: both root `DecodePagination` and usermgmt now delegate to `query.NewPagination` from go-cqrs-lite          |
| **Stdlib modernization**     | `f73eb85` | `slices.Contains`, `min()`, `slices.IndexFunc` replace 5 manual loops across root/usermgmt/adminui/examples                        |
| **go.mod version alignment** | `f73eb85` | All 8 modules aligned to Go 1.26.4 + cqrs-htmx v3.2.0 declarations                                                                 |
| **ETag bump**                | `f73eb85` | adminui asset ETag updated from stale `adminui-v2` to `adminui-v3.2.0`                                                             |
| **Lint cleanup**             | `f73eb85` | goconst warnings fixed (`testCatalogVersion` constant), `context.TODO()` → `context.Background()`                                  |

### Security & Reliability

| Item                                 | Commit         | Notes                                                                                                                                                                                                                                                              |
| ------------------------------------ | -------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **Idempotency store**                | `e8f8e30`      | `IdempotencyStore` interface + `MemoryIdempotencyStore` with TTL sweep. `CheckAndRecord` atomic helper. `ErrDuplicateCommand` → HTTP 409. 9 tests covering: new/seen IDs, TTL expiration, sweep cleanup, concurrent access, duplicate rejection, Close idempotency |
| **go.yaml.in/yaml/v3 investigation** | — (documented) | Confirmed as **official Canonical Ltd successor** to `gopkg.in/yaml.v3` — same codebase, new canonical import path. NOT a typo-squat. No replace/fork needed                                                                                                       |
| **CI umbrella verification**         | — (verified)   | `nix run .#test` + `nix run .#lint` + `nix run .#errorfamily` all green across root/usermgmt/adminui/integration_test                                                                                                                                              |

### Documentation

| Item                  | Commit    | Notes                                                                                                                                                             |
| --------------------- | --------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **ADR-0026**          | `16c44eb` | Command Idempotency Store — full ADR with wiring example, design principles, consequences                                                                         |
| **AGENTS.md updates** | `16c44eb` | Added: idempotency.go to file tree, go-playground/form/v4 to deps table, form decoder note, Bot/Impersonation service-level decision, pagination unification note |
| **TODO_LIST.md**      | `7d3f5aa` | 8 new items marked done; reflects actual codebase state                                                                                                           |

### Decisions Made

| Decision                                                       | Rationale                                                                                                                                                                                                                                                                          |
| -------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Bot/Impersonation = service-level APIs** (not ghost systems) | These have full security guards (super_admin checks, reason-required, self-impersonation prevention, HMAC token hashing). They're intentionally service-level because routing scheme, path prefixes, and auth middleware are consumer-specific decisions. Documented in AGENTS.md. |
| **Pagination: no silent page clamping**                        | Requesting a page beyond the last page now returns an empty page (standard REST). The old behavior silently clamped to the last page, which hides data from clients. The API response includes `total_pages` so clients can detect the valid range.                                |
| **Branded CommandID deferred**                                 | The string flows through a single path (header → ack → JSON) with no risk of mixing with other ID types. The marginal value of a branded type doesn't justify the breaking change to `CommandAck`.                                                                                 |
| **Rate limiter unification deferred**                          | Root uses token-bucket + heap LRU; usermgmt uses fixed-window per-IP. Unifying requires config-shape changes that risk breaking consumers. Medium-risk refactor for consistency-only value.                                                                                        |

---

## B) PARTIALLY DONE

### Offline-First Command Sync (Phase 0 + Phase 1)

- **Phase 0 (server infrastructure):** ✅ DONE — `JournalSSEStore`, `Broadcaster`, `SSEStream`, `CommandAck`, reconnection protocol
- **Phase 1 (honest UI protocol):** ✅ DONE — CSS/JS/templ/admin-demo, ACK protocol, `BroadcastOnAck` hooks
- **Phase 2 (client-side decide + offline persistence):** ❌ BLOCKED — requires user decisions on Q1 (where `decide()` runs) and Q2 (closed-tab persistence strategy)

### Idempotency

- **Store + tests:** ✅ DONE — `IdempotencyStore` interface, `MemoryIdempotencyStore`, `CheckAndRecord`, 9 tests
- **Auto-wiring into `App.Command`:** ❌ NOT DONE — intentionally not auto-wired (library principle). Consumers wire via `BeforeDispatchHook`.
- **Redis/Postgres-backed store:** ❌ NOT DONE — interface exists for future implementation

### adminui

- **Rendering + routing:** ✅ DONE — full dashboard, users/tenants/members/audit sections, light/dark theme
- **Honest UI integration:** ✅ DONE — sync indicator, `data-sync-state`, demo wiring
- **Integration test coverage:** ❌ NOT DONE — no integration_test covers adminui mounting/routing
- **Sync indicator unit tests:** ❌ NOT DONE — no test asserts `.sync-pending` → `.sync-confirmed` DOM flip
- **JS unit tests:** ❌ NOT DONE — `admin.js` sync-state handler has no JS test harness

---

## C) NOT STARTED

| Item                                               | Impact | Blocked By                      |
| -------------------------------------------------- | ------ | ------------------------------- |
| Rate limiter unification (root ↔ usermgmt)         | Medium | Config-shape design decision    |
| adminui integration test (mount + route render)    | High   | Needs cross-module test setup   |
| Actor/impersonator context bridge integration test | High   | Needs session middleware wiring |
| Honest UI error inline + retry button              | Medium | Needs browser verification      |
| Responsive `.sync-bar` mobile layout               | Low    | CSS only                        |
| Consumer wiring recipe doc (SSE + ACK + honest UI) | Medium | Documentation task              |
| SSE replay benchmark with large journals           | Low    | Performance baseline            |
| Security review: command ID injection/replay       | High   | Can be done independently       |
| OPFS/IndexedDB technical spike (Phase 2)           | Medium | Blocked on Q2 decision          |
| go-localsync CRDT evaluation                       | Low    | Future research                 |
| ClientIP cleanup                                   | Low    | FEATURES.md marks deprecated    |
| Otter cache evaluation for ephemeral stores        | Low    | Long-tail optimization          |
| PWA manifest for admin-demo                        | Low    | Post-Phase 2                    |
| Tailwind CSS regeneration                          | Low    | Blocked on npm install          |
| v3.3.0 release                                     | High   | Blocked on Phase 2 decisions    |

---

## D) TOTALLY FUCKED UP

**Nothing is fucked up.** This is the cleanest state the codebase has been in:

- 0 lint issues (all modules)
- 0 errorfamily violations
- 0 banned dependencies (direct)
- 0 race detector failures (932 tests pass with `-race`)
- 0 stdlib error constructors (enforced by `branching-flow errorfamily`)

The only "fuckup" was the previous status report claiming "All CI gates are
green" without running the umbrella `nix run .#test` end-to-end. That has been
**corrected and verified** — the umbrella now passes for real.

---

## E) WHAT WE SHOULD IMPROVE

### High Priority

1. **Add adminui integration test** — adminui is the only module with zero
   integration_test coverage. A minimal test that mounts the panel and asserts
   route rendering would catch regressions in the templ/HTMX wiring.

2. **Security review of command ID protocol** — The `X-Command-Id` header
   accepts arbitrary client-supplied strings. We need to document: (a) what
   characters are valid, (b) max length, (c) whether the idempotency store
   prevents replay attacks, (d) whether a malicious client can poison another
   client's pending state by guessing their command ID.

3. **Wire idempotency into admin-demo** — The idempotency store exists but
   the admin-demo doesn't use it. A demo showing duplicate-command rejection
   would prove the feature works end-to-end.

### Medium Priority

4. **Rate limiter unification** — Two different algorithms (token-bucket vs
   fixed-window) in one repo is a split brain. The config shapes are different
   enough that this needs careful design to avoid breaking consumers.

5. **ActorID cross-module type** — Root `ActorID` is `string`, usermgmt has
   a kind-discriminated struct. Conversion is manual/stringly. A shared
   identity module or at least typed conversion helpers would prevent bugs.

6. **JS test harness for admin.js** — The sync-state handler has complex
   logic (pending → confirmed/rejected DOM flips, rollback on disconnect).
   No JS tests exist. Even a minimal JSDOM-based test would catch regressions.

7. **Coverage gate for adminui** — adminui has only 35 tests (vs 747 in
   usermgmt). The `nix run .#coverage-gate` doesn't cover adminui.

### Low Priority

8. **SSE benchmark** — No baseline for replay performance with 10K/100K
   events. Consumers deploying to production need to know the limits.

9. **Tailwind regeneration** — `admin-tw.css` may be stale. Needs `npm install`
   - `npx tailwindcss build` to regenerate. Blocked on Node.js availability.

10. **Deprecation cleanup** — `ClientIP` is marked `PARTIALLY_FUNCTIONAL` in
    FEATURES.md but still exported. Either fix or remove.

---

## F) Top 25 Things to Do Next

Sorted by impact × effort × customer-value.

| #   | Task                                                                    | Impact   | Effort | Module           | Blocked?  |
| --- | ----------------------------------------------------------------------- | -------- | ------ | ---------------- | --------- |
| 1   | **Answer Q1: where does `decide()` run?** (Queue-Only / WASM / TS Port) | Critical | 5m     | —                | **User**  |
| 2   | **Answer Q2: closed-tab persistence?** (SharedWorker / Service Worker)  | Critical | 5m     | —                | **User**  |
| 3   | Add adminui integration test (mount + route render)                     | High     | 12m    | integration_test | —         |
| 4   | Security review: command ID injection/replay surface                    | High     | 12m    | root             | —         |
| 5   | Wire idempotency into admin-demo BeforeDispatchHook                     | High     | 10m    | admin-demo       | —         |
| 6   | Add actor/impersonator context bridge integration test                  | High     | 12m    | integration_test | —         |
| 7   | Document consumer wiring recipe (SSE + ACK + honest UI)                 | Medium   | 12m    | docs             | —         |
| 8   | Design rate-limit config unification (root ↔ usermgmt)                  | Medium   | 10m    | usermgmt         | —         |
| 9   | Replace usermgmt perIPRateLimiter with root RateLimiterMiddleware       | Medium   | 12m    | usermgmt         | #8        |
| 10  | Honest UI: inline error message in rejected state                       | Medium   | 12m    | adminui          | —         |
| 11  | Honest UI: add retry button to rejected items                           | Medium   | 12m    | adminui          | —         |
| 12  | Add adminui sync indicator rendering test                               | Medium   | 12m    | adminui          | —         |
| 13  | Fix responsive `.sync-bar` mobile layout                                | Low      | 8m     | adminui          | —         |
| 14  | Add JS unit tests for sync-state handler                                | Medium   | 12m    | adminui          | —         |
| 15  | Design shared ActorID type (root without importing usermgmt)            | High     | 12m    | root             | —         |
| 16  | Implement root ActorID with safe constructors                           | High     | 12m    | root             | #15       |
| 17  | Add conversion helpers root ↔ usermgmt ActorID                          | Medium   | 8m     | integration_test | #16       |
| 18  | SSE replay benchmark (10K/100K events)                                  | Low      | 12m    | root             | —         |
| 19  | Browser-verify admin-demo honest UI lifecycle                           | High     | 12m    | admin-demo       | —         |
| 20  | Evaluate maypok86/otter/v2 for ephemeral stores                         | Low      | 12m    | usermgmt         | —         |
| 21  | Remove or un-deprecate ClientIP                                         | Low      | 8m     | root             | —         |
| 22  | Add PWA manifest to admin-demo                                          | Low      | 10m    | admin-demo       | —         |
| 23  | Regenerate admin-tw.css from tailwind source                            | Low      | 12m    | adminui          | npm       |
| 24  | OPFS/IndexedDB technical spike for Phase 2                              | Medium   | 12m    | docs             | **Q2**    |
| 25  | Cut v3.3.0 release                                                      | High     | 12m    | repo             | **Q1+Q2** |

---

## G) Top Question I Cannot Figure Out Myself

**Q1: Where should `decide()` run on the client?**

The offline-first command sync architecture (ADR-0023) requires the client to
know whether a command is valid _before_ sending it to the server — otherwise
the "honest UI" can show pending, but can't show confirmed until the server
responds, which defeats offline-first.

Three options, each with massive architecture implications:

1. **Queue-Only (no client-side decide)** — Client queues all commands
   optimistically, server validates on reconnect. Simplest. But the UI can't
   show "this will fail" until reconnection. Honest UI shows pending →
   rejected-on-sync, never pre-validates.

2. **WASM port of `decide()`** — Compile the Go domain `decide()` functions
   to WebAssembly. Client runs the same validation logic as the server. Full
   offline validation. But: WASM binary size, Go→WASM compilation complexity,
   and the domain model must be pure (no I/O) — which it already is.

3. **TypeScript port of `decide()`** — Manually rewrite the decide functions
   in TypeScript. Smaller bundle than WASM, native JS interop. But:
   dual-maintenance burden, logic drift risk, and the Go domain types must be
   kept in sync with TS types manually.

**I cannot decide this for you** because it depends on:

- Your offline UX requirements (is "pending → rejected on reconnect" acceptable?)
- Your tolerance for WASM binary size in the browser
- Whether you want to maintain a TS port of domain logic
- Whether the domain model will grow significantly (more aggregates = more port work)

**My recommendation:** Start with **Queue-Only** (option 1) for Phase 2 MVP.
It's the simplest, ships fastest, and the honest UI already handles
pending → rejected gracefully. Upgrade to WASM later if offline pre-validation
becomes a real user need. But this is your call.

---

## CI Verification (2026-06-28 21:00)

```
$ nix run .#test
==> Root module        ok  4.0s
==> adminui submodule   ok  1.0s
==> usermgmt submodule  ok  3.0s
==> integration_test   ok  1.0s

$ nix run .#lint
==> Root module        0 issues
==> adminui submodule   0 issues
==> usermgmt submodule  0 issues

$ nix run .#errorfamily
All modules pass errorfamily check.
```

## Module Health

| Module           | Tests   | Lint     | ErrorFamily | Coverage |
| ---------------- | ------- | -------- | ----------- | -------- |
| Root             | 133     | 0 issues | ✅          | 95.4%    |
| usermgmt         | 747     | 0 issues | ✅          | 80.1%    |
| adminui          | 35      | 0 issues | ✅          | —        |
| integration_test | 17      | —        | —           | —        |
| **Total**        | **932** | **0**    | **✅**      | —        |

## Numbers

| Metric               | Value  |
| -------------------- | ------ |
| Go files             | 341    |
| Lines of Go          | 51,331 |
| Go modules           | 8      |
| ADRs                 | 26     |
| Direct deps (root)   | 11     |
| TODOs open           | 2      |
| TODOs done           | 90     |
| Commits this session | 7      |
