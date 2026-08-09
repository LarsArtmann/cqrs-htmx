# Status Report — Server-Timing Feature + Full Project Audit

**Date:** 2026-06-29 06:53 CEST
**Session:** Server-Timing API → self-review → improvements → full project audit
**Branch:** master @ `e9ef04b`
**Latest tags:** root v3.3.0, usermgmt v3.3.0, adminui v3.0.0

---

## Executive Summary

This session delivered a **complete, production-grade W3C Server-Timing API** for the library, then self-reviewed it into a leaner, more idiomatic shape. The broader project is **exceptionally healthy**: all 8 Go modules build and test clean with `-race`, 0 lint issues, 0 error-family violations. One deprecated feature (ClientIP) is the only non-fully-functional item in FEATURES.md.

**The honest gap:** the observability roadmap (v3.3.0) promised OpenTelemetry + Prometheus — neither is wired. Server-Timing is a **partial** delivery of that observability promise (per-request timing in the browser), but the programmatic/metrics story is incomplete. The CHANGELOG `[Unreleased]` section is also empty despite 4 new feature commits.

---

## a) FULLY DONE ✅

### Server-Timing API (`server_timing.go` — 387 LOC + 662 LOC tests + 92 LOC bench)

| Capability                                                      | Status  | Evidence                                                             |
| --------------------------------------------------------------- | ------- | -------------------------------------------------------------------- |
| W3C Server-Timing header emission                               | ✅ Done | `serverTimingWriter` injects header at first WriteHeader/Write       |
| `ServerTiming` collector (thread-safe)                          | ✅ Done | Nil-receiver pattern: disabled=nil=natural no-op, no `enabled` field |
| `ServerTimingMiddleware()` (always-on)                          | ✅ Done | Composes via `Chain()`                                               |
| `ServerTimingMiddlewareWhen(pred)` (debug-gated)                | ✅ Done | Predicate-gated, zero-overhead when false                            |
| `Config.ServerTiming` (1-line App integration)                  | ✅ Done | Wraps every `Command()`/`Query()` handler                            |
| `WithServerTiming` / `ServerTimingFromContext`                  | ✅ Done | Mirrors `WithRequestID` pattern                                      |
| `RecordServerTiming` / `MeasureServerTiming` (nil-safe helpers) | ✅ Done | No per-handler `if st != nil` checks                                 |
| Interface preservation (Flusher/Hijacker/Pusher/Unwrap)         | ✅ Done | SSE/WS/HTTP2 still work through wrapper — tested                     |
| RFC 7230 token sanitization + quoted-string escaping            | ✅ Done | Invalid chars → `_`, `"` and `\` escaped                             |
| Sub-millisecond precision (fractional dur)                      | ✅ Done | `strconv.FormatFloat(ms, 'f', -1, 64)`                               |
| 6 benchmarks (overhead proof)                                   | ✅ Done | **Disabled: 3.6 ns/op, 0 allocs**                                    |
| Example (`ExampleServerTimingMiddleware`)                       | ✅ Done | Shows debug-gated pattern                                            |
| 30 tests (race-safe)                                            | ✅ Done | Spec-compliance, Flush delegation, Config integration, concurrency   |

### Project-Wide Health (all modules)

| Module                 | Build | Tests (-race) | Lint     | Coverage        |
| ---------------------- | ----- | ------------- | -------- | --------------- |
| Root (`cqrs-htmx/v3`)  | ✅    | ✅ 4.0s       | 0 issues | 93.8%           |
| usermgmt (`/v3`)       | ✅    | ✅ 2.8s       | 0 issues | 79.3%           |
| adminui (`/v3`)        | ✅    | ✅ 1.0s       | 0 issues | —               |
| integration_test       | ✅    | ✅ 1.0s       | —        | —               |
| examples/basic         | ✅    | —             | —        | —               |
| examples/admin-demo    | ✅    | —             | —        | —               |
| examples/datastar-demo | ✅    | —             | —        | —               |
| examples/catalog-demo  | ✅    | —             | —        | —               |
| **errorfamily gate**   | —     | —             | —        | ✅ 0 violations |

### Other Completed Work (this broader session)

- **v3.3.0 released** — 3 modules tagged (`root`, `usermgmt`, `adminui`)
- **go-cqrs-lite upgraded** to v3.4.0 across all 8 modules
- **Command ID minting bug fixed** — 7 of 20 constructors returned zero ID (critical idempotency bug)
- **Checkpoint-based projection replay** — `StartProjections` gains `CheckpointStore` param
- **Offline command queue (Phase 2a)** — SharedWorker sync-worker.js (ADR-0029)
- **Idempotency store** — `MemoryIdempotencyStore` with atomic `CheckAndRecord` (ADR-0026)
- **ACK protocol** — `CommandAck` + `BroadcastOnAck` (ADR-0024)
- **JournalSSEStore** — production SSE replay backed by `event.SeekableJournal` (ADR-0023)
- **32 ADRs** — latest: 0028 (brand IDs), 0029 (SharedWorker), 0030 (persistence), 0031 (projection lifecycle)

---

## b) PARTIALLY DONE 🟡

| Item                               | Gap                                                                                                                                                               | Impact                                                                                                        |
| ---------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------- |
| **CHANGELOG `[Unreleased]`**       | Empty — Server-Timing, checkpoint replay, Config.ServerTiming not documented                                                                                      | Release-ready? No. Next tag will miss 4 feature commits.                                                      |
| **Observability roadmap (v3.3.0)** | Server-Timing delivers browser-visible timing, but **OpenTelemetry** (wired via go-cqrs-lite otel module) and **Prometheus metrics middleware** are still Planned | The roadmap promise is half-met. Server-Timing is a different axis (developer debug, not production metrics). |
| **Server-Timing ADR**              | No ADR exists for the Server-Timing decision (header-before-body constraint, nil-receiver pattern, TTFB semantics)                                                | 32 ADRs exist for every other significant decision. This gap breaks the pattern.                              |
| **ROADMAP.md**                     | v3.3.0 section says "Done" for CI coverage gate but "Planned" for OTel/Prometheus — doesn't mention Server-Timing at all                                          | Roadmap is stale relative to shipped code.                                                                    |
| **Coverage drift**                 | Root dropped 94.1% → 93.8% (new Server-Timing code: ~88% covered by line-count, bench file isn't covered). usermgmt at 79.3% (gate is 75%, OK but trending down). | Not failing, but the root coverage-gate threshold is 90% — headroom is shrinking.                             |
| **Server-Timing TTFB footgun**     | `defer MeasureServerTiming(ctx,"x")()` silently does nothing for non-streaming handlers (documented in tests, not guarded in API).                                | UX trap — a developer will use the natural defer idiom and wonder why their metric is missing.                |

---

## c) NOT STARTED ⚪

### From ROADMAP v3.3.0 (Observability)

- [ ] **OpenTelemetry wiring** — go-cqrs-lite v3 has an `otel` module; not yet integrated
- [ ] **Prometheus metrics middleware** — dispatch latency histograms, error rate counters

### From ROADMAP v3.4.0 (Persistence & Scale)

- [ ] Redis session store (distributed deployments)
- [ ] Redis OAuth2 state store (multi-instance)
- [ ] PostgreSQL session store preset
- [ ] BadgerDB embedded store alternative
- [ ] Streaming replay profiling (10K+ events)
- [ ] Hot-path optimization (dispatch, decode)

### From ROADMAP v4.0.0 (Advanced ES)

- [ ] CatchUpSubscriber adoption (alternative to manual replay in `StartProjections`)
- [ ] Schema/v3 validator for event payloads at registration
- [ ] Integration tests against real PostgreSQL
- [ ] Database migration tooling (goose / golang-migrate / gnorm)

### From ADR-0029 (Offline Queue)

- [ ] **OPFS persistence (Phase 2b)** — offline queue currently in-memory only (lost on tab close)
- [ ] WASM/TS client-side pre-validation (consumer concern per ADR-0027, but library could provide helpers)

### From Session Work

- [ ] **Server-Timing ADR** (document the header-before-body constraint + nil-receiver decision)
- [ ] **Server-Timing in CHANGELOG** (feature is shipped, not documented)
- [ ] **catalog-demo `go mod tidy`** — 15+ unused deps in go.mod (gopls warnings, pre-existing)

---

## d) TOTALLY FUCKED UP 🔴

**Nothing is critically broken.** All modules build, all tests pass, all linters clean.

The closest thing to "fucked up":

| Issue                         | Severity | Detail                                                                                                                                                     |
| ----------------------------- | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **catalog-demo go.mod bloat** | Low      | 15+ unused dependencies in `examples/catalog-demo/go.mod` (gopls warnings on every load). Pre-existing — not introduced this session. Cosmetic but sloppy. |
| **CHANGELOG gap**             | Medium   | 4 shipped feature commits since v3.3.0 tag, all undocumented in `[Unreleased]`. If someone tags now, the changelog won't reflect reality.                  |
| **Coverage headroom**         | Watch    | Root 93.8% (gate 90%), usermgmt 79.3% (gate 75%). Both pass, but usermgmt has been slowly trending down as new features outpace tests.                     |

---

## e) WHAT WE SHOULD IMPROVE 🚀

### Architecture & Type Model

1. **Server-Timing ADR** — document the core constraint (header-before-body), the nil-receiver pattern decision, and the TTFB semantics. Every other significant decision has an ADR.
2. **TTFB footgun guardrail** — consider a `MeasureForCommit(ctx, name)` variant that auto-calls stop at `flushHeader` time, or at minimum a lint/test helper that detects `defer Measure()()` before a `w.Write` in the same function.
3. **OpenTelemetry seam** — the library already has `BeforeDispatchHook`/`AfterDispatchHook`. An `OTelMiddleware` that creates a span around dispatch would complete the observability story alongside Server-Timing. go-cqrs-lite's `otel` module exists but isn't wired.
4. **`StatusRecorder` → `serverTimingWriter` duplication** — both wrap ResponseWriter and delegate Flush/Hijack/Push. A shared `delegatingWriter` base could DRY this up. Low priority — the pattern is only 2 occurrences.

### Testing & Quality

5. **Server-Timing integration test through the real App** — the `applyServerTiming` helper is tested in isolation, but there's no end-to-end test showing a `Config.ServerTiming`-enabled App producing the header in a real dispatch flow.
6. **usermgmt coverage recovery** — 79.3% → target 82%+ before v3.4.0. The SQL read models and OAuth2 flow are the likely gaps.
7. **Flaky test CI run** — add `-count=3` or `-count=5` to a CI step to catch race conditions that single-run misses.

### Documentation & Release

8. **CHANGELOG `[Unreleased]`** — document Server-Timing, checkpoint replay, Config.ServerTiming, and the nil-receiver refactor. Required before next tag.
9. **ROADMAP alignment** — update v3.3.0 section: mark "Prometheus" as superseded by Server-Timing for the developer-debug axis, or explicitly track both as complementary.
10. **Server-Timing in FEATURES.md** — add a row under "Observability" or "Debugging".

### Operational

11. **catalog-demo `go mod tidy`** — clean up the 15 unused deps. 2-minute fix, removes 30 gopls warnings.
12. **Pre-push hook coverage** — the BuildFlow pre-commit hook runs lint/build but not the coverage gate. Consider adding `nix run .#coverage-gate` to the hook.

---

## f) TOP 25 THINGS TO DO NEXT

Sorted by **impact / effort ratio** (highest first):

| #  | Task                                            | Impact | Effort | Why                                                                           |
| -- | ----------------------------------------------- | ------ | ------ | ----------------------------------------------------------------------------- |
| 1  | **Fill CHANGELOG `[Unreleased]`**               | High   | 10 min | 4 shipped features undocumented; blocks clean release                         |
| 2  | **catalog-demo `go mod tidy`**                  | Medium | 5 min  | Removes 30 gopsl warnings; pure hygiene                                       |
| 3  | **Write Server-Timing ADR (0032)**              | Medium | 20 min | 32 ADRs exist for everything else; this gap breaks the pattern                |
| 4  | **Add Server-Timing to FEATURES.md**            | Low    | 5 min  | Feature inventory must reflect shipped code                                   |
| 5  | **End-to-end App test for Config.ServerTiming** | High   | 30 min | Proves the integration works through real dispatch, not just the helper       |
| 6  | **Update ROADMAP v3.3.0**                       | Medium | 10 min | Mark Server-Timing as delivered; clarify OTel/Prometheus status               |
| 7  | **Wire OpenTelemetry (`OTelMiddleware`)**       | High   | 2-4h   | Completes the v3.3.0 observability promise; go-cqrs-lite otel module exists   |
| 8  | **usermgmt coverage → 82%**                     | High   | 3-4h   | SQL read models + OAuth2 flow likely gaps; gate is 75%, headroom shrinking    |
| 9  | **Redis SessionStore**                          | Medium | 4-6h   | Unblocks multi-instance deployments (ROADMAP v3.4.0)                          |
| 10 | **Redis OAuth2StateStore**                      | Medium | 3-4h   | Pairs with Redis SessionStore for distributed OAuth2                          |
| 11 | **CatchUpSubscriber adoption**                  | Medium | 3-4h   | Replace manual replay in `StartProjections`; cleaner than checkpoint approach |
| 12 | **Prometheus metrics middleware**               | Medium | 3-4h   | Dispatch latency histograms; complements Server-Timing for production         |
| 13 | **OPFS persistence (Phase 2b)**                 | Medium | 6-8h   | Offline queue survives tab close (ADR-0029)                                   |
| 14 | **Integration tests against real PostgreSQL**   | High   | 4-6h   | Currently only SQLite tested; Postgres path is unproven in CI                 |
| 15 | **Schema/v3 event payload validator**           | Medium | 4-6h   | Catch malformed events at registration time                                   |
| 16 | **Database migration tooling**                  | Medium | 4-6h   | `usermgmt/migrations/` exists but no runner integrated                        |
| 17 | **Streaming replay profiling (10K+ events)**    | Low    | 3-4h   | Validate checkpoint replay at scale                                           |
| 18 | **PostgreSQL session store preset**             | Low    | 2-3h   | Reduced boilerplate for PG users                                              |
| 19 | **Hot-path profiling (dispatch, decode)**       | Low    | 4-6h   | Benchmark-driven optimization of dispatch + JSON decode                       |
| 20 | **Shared `delegatingWriter` base**              | Low    | 1-2h   | DRY StatusRecorder + serverTimingWriter wrapper duplication                   |
| 21 | **Server-Timing `MeasureForCommit` variant**    | Low    | 1-2h   | Guardrail against the TTFB defer footgun                                      |
| 22 | **CI: add `-count=3` flake detection**          | Low    | 30 min | Catch race conditions that single-run misses                                  |
| 23 | **CI: add coverage-gate to pre-push hook**      | Low    | 30 min | Catch coverage regressions before push, not after                             |
| 24 | **BadgerDB embedded store**                     | Low    | 4-6h   | Alternative to SQLite for embedded deployments                                |
| 25 | **Admin-demo Server-Timing showcase**           | Low    | 1h     | Show `?debug=1` → Server-Timing header in the runnable demo                   |

---

## g) TOP QUESTION I CANNOT FIGURE OUT MYSELF 🤔

**"Should Server-Timing be the observability story for v3.3.0, or is OpenTelemetry still required?"**

The ROADMAP v3.3.0 section lists two observability items as "Planned":

1. Wire OpenTelemetry via go-cqrs-lite v3 otel module
2. Prometheus metrics middleware

Server-Timing is a **different axis** — it's developer-facing (visible in browser DevTools), not production-facing (metrics pipelines, tracing backends). It doesn't replace either of those roadmap items.

**The question:** Should we:

- **(A)** Declare v3.3.0 "shipped" with Server-Timing as the observability deliverable, and defer OTel/Prometheus to v3.4.0? (Reframe the roadmap.)
- **(B)** Wire OpenTelemetry now to fully deliver the v3.3.0 promise, then tag? (Delays the release.)
- **(C)** Tag v3.3.0 now (Server-Timing only), and tag v3.3.1 or v3.4.0 when OTel is ready? (Incremental.)

I cannot decide this because it depends on your release philosophy: do you ship the roadmap as-written (B), or adapt the roadmap to what was actually built (A/C)? The library is production-quality either way — this is a framing/communication decision, not a technical one.

---

## Technical Metrics Snapshot

| Metric                                  | Value                               |
| --------------------------------------- | ----------------------------------- |
| Root source files                       | 44 (19,668 LOC)                     |
| Root test files                         | 83                                  |
| usermgmt source files                   | ~60 (173 total .go including tests) |
| adminui files                           | 27                                  |
| ADRs                                    | 32                                  |
| Go modules                              | 8 (go.work)                         |
| Go version                              | 1.26.4                              |
| go-cqrs-lite                            | v3.4.0                              |
| Server-Timing disabled overhead         | 3.6 ns/op, 0 allocs                 |
| Server-Timing enabled overhead          | 138 ns/op, 1 alloc (Measure)        |
| Server-Timing header render (5 metrics) | 271 ns/op, 7 allocs                 |
| Root coverage                           | 93.8%                               |
| usermgmt coverage                       | 79.3%                               |
| Lint issues                             | 0 (all modules)                     |
| errorfamily violations                  | 0                                   |
