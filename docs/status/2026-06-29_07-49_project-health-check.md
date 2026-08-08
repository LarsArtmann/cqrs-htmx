# Status Report — Full Project Health Check

**Date:** 2026-06-29 07:49 CEST
**Branch:** master @ `10fd0b1`
**Latest tags:** root v3.3.0, usermgmt v3.3.0, adminui v3.0.0
**Unreleased work:** Server-Timing, checkpoint replay, delegatingWriter, BasicCommand embedding

---

## Executive Summary

The project is **production-healthy**: all 8 Go modules build and test clean with `-race`, 0 lint issues, 0 errorfamily violations, 94.3% root coverage, `nix flake check` passes. Server-Timing API is fully shipped and documented (ADR-0033, CHANGELOG, FEATURES, benchmarks, end-to-end tests, admin-demo showcase). The ROADMAP has been rewritten to remove dead items (Redis, hand-rolled OTel/Prometheus) and reflect the reality that go-cqrs-lite upstream already provides those capabilities.

**One issue fixed this session:** ADR numbering collision (two 0032s) — resolved by renumbering Server-Timing to 0033.

---

## a) FULLY DONE ✅

### Server-Timing API — Complete Feature Delivery

| Capability                                            | Status | Evidence                                                                                          |
| ----------------------------------------------------- | ------ | ------------------------------------------------------------------------------------------------- |
| Core implementation (`server_timing.go`)              | ✅     | Nil-receiver pattern, 3 entry points (middleware, predicate-gated, Config), thread-safe collector |
| `delegatingWriter` DRY refactor (`responsewriter.go`) | ✅     | Shared Flush/Hijack/Push/Unwrap base for StatusRecorder + serverTimingWriter (~50 lines removed)  |
| Config integration (`Config.ServerTiming`)            | ✅     | 1-line opt-in, wraps every `Command()`/`Query()` handler                                          |
| 33 tests (race-safe)                                  | ✅     | Spec-compliance, Flush delegation, concurrency, end-to-end dispatch, disabled no-op               |
| 6 benchmarks                                          | ✅     | Disabled: 3.6ns/0-allocs, Enabled: 138ns/1-alloc                                                  |
| ADR-0033                                              | ✅     | Header-before-body constraint, nil-receiver, TTFB footgun documented                              |
| CHANGELOG `[Unreleased]`                              | ✅     | Full description with all 3 features                                                              |
| FEATURES.md                                           | ✅     | Server-Timing row under Middleware & Observability                                                |
| ROADMAP.md                                            | ✅     | v3.3.0 rewritten: Server-Timing Done, OTel/Prometheus reframed as upstream-wiring docs            |
| Example                                               | ✅     | `ExampleServerTimingMiddleware` in example_app_test.go                                            |
| Admin-demo showcase                                   | ✅     | `?debug=1` → Server-Timing header in admin-demo/main.go                                           |

### Project-Wide Health (all modules)

| Module                 | Build | Tests (-race) | Lint     | Coverage             |
| ---------------------- | ----- | ------------- | -------- | -------------------- |
| Root (`cqrs-htmx/v3`)  | ✅    | ✅ 4.0s       | 0 issues | 94.3%                |
| usermgmt (`/v3`)       | ✅    | ✅ 3.0s       | 0 issues | 79.3%                |
| adminui (`/v3`)        | ✅    | ✅ 1.0s       | 0 issues | —                    |
| integration_test       | ✅    | ✅ 1.0s       | —        | —                    |
| examples/basic         | ✅    | —             | —        | —                    |
| examples/admin-demo    | ✅    | —             | —        | —                    |
| examples/datastar-demo | ✅    | —             | —        | —                    |
| examples/catalog-demo  | ✅    | —             | —        | —                    |
| **errorfamily gate**   | —     | —             | —        | ✅ 0 violations      |
| **nix flake check**    | —     | —             | —        | ✅ all checks passed |

### Other Completed Work This Session

- **ROADMAP rewrite** — Removed Redis (out of scope per user), PostgreSQL preset (not needed), BadgerDB (pebble better). Replaced with upstream adoption: projectionhost, CatchUpSubscriber, scenario DSL, snapshot
- **`delegatingWriter` refactor** — DRY'd StatusRecorder + serverTimingWriter wrapper duplication into shared base
- **`test-flake` nix app** — `nix run .#test-flake` runs all tests 3x with `-race` for flake detection
- **Admin-demo Server-Timing** — `?debug=1` shows timing header in browser DevTools
- **33 ADRs** — numbering collision fixed (0032-basic-command-embedding + 0033-server-timing)

---

## b) PARTIALLY DONE 🟡

| Item                              | Gap                                                                                                                                                                                                                       | Impact                                                                    |
| --------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------- |
| **OTel/Prometheus wiring guides** | ROADMAP says "Planned" — these are docs tasks (go-cqrs-lite upstream already provides `otel/v3`, `middleware/v3`, `prometheus/v3`). The library doesn't need to implement them, just document how consumers bolt them on. | Medium — closes the v3.3.0 observability promise without writing new code |
| **usermgmt coverage**             | 79.3% (gate is 75%, OK but could be higher). SQL read models and OAuth2 flow are likely gaps.                                                                                                                             | Watch — headroom is shrinking as features outpace tests                   |
| **OPFS persistence (Phase 2b)**   | Offline queue is in-memory only (ADR-0029). Lost on tab close.                                                                                                                                                            | Low — current Phase 2a works for online/offline transitions               |

---

## c) NOT STARTED ⚪

### From ROADMAP v3.3.0 (Observability — docs tasks only)

- [ ] OTel wiring guide — document how to use `go-cqrs-lite/otel/v3` + `middleware/v3` with the App
- [ ] Prometheus wiring guide — document `go-cqrs-lite/prometheus/v3` `Setup()` → `/metrics`

### From ROADMAP v3.4.0 (Upstream Adoption)

- [ ] Adopt `projectionhost/v3` — replace hand-rolled `StartProjections`
- [ ] Adopt `CatchUpSubscriber` — ordered durable projections
- [ ] Adopt `scenario/v3` BDD DSL for usermgmt decider tests
- [ ] Adopt `snapshot/v3` for aggregates with 100+ events
- [ ] Profile and optimize hot paths (dispatch, decode)
- [ ] Benchmark projection replay with large stores (10K+ events)

### From ROADMAP v4.0.0 (Advanced ES)

- [ ] Adopt `kv.Cache` decorator for read-model caching
- [ ] Adopt `schema/v3` upcasters for event evolution

### From ADR-0029 (Offline Queue)

- [ ] OPFS persistence (Phase 2b) — offline queue survives tab close

---

## d) TOTALLY FUCKED UP 🔴

**Nothing is broken.** All modules build, all tests pass, all linters clean.

Minor issues (all fixed this session):

- **ADR numbering collision** — two files numbered 0032 (basic-command-embedding + server-timing). Fixed: server-timing renumbered to 0033.
- **`layout_templ.go` codegen drift** — `:=` → `var =` from templ CLI version difference. Cosmetic, builds fine. Being committed.

---

## e) WHAT WE SHOULD IMPROVE 🚀

### Architecture

1. **OTel wiring guide** — a 1-page markdown doc showing how to wire `go-cqrs-lite/otel/v3` + `middleware/v3` into the App's `BeforeDispatchHook`/`AfterDispatchHook`. Zero new code, closes the v3.3.0 promise.
2. **`projectionhost` adoption** — the biggest code-reduction opportunity. `StartProjections` is ~100 lines of hand-rolled replay+subscribe+checkpoint logic that `projectionhost/v3` handles with crash-restart + DLQ.
3. **`scenario/v3` for decider tests** — usermgmt has 7 pure decide functions that would benefit from Given/When/Then BDD syntax.

### Quality

4. **usermgmt coverage → 82%** — SQL read models + OAuth2 are likely the gaps.
5. **CI: run `test-flake` in CI** — catch intermittent failures.

### Documentation

6. **Prometheus wiring guide** — 1-page doc for `go-cqrs-lite/prometheus/v3` `Setup()`.
7. **Server-Timing in AGENTS.md file tree** — add `responsewriter.go` and `server_timing.go` to the architecture diagram (partially done, responsewriter.go missing).

---

## f) TOP 25 THINGS TO DO NEXT

Sorted by **impact / effort ratio** (highest first):

| #  | Task                                                           | Impact | Effort | Why                                                                        |
| -- | -------------------------------------------------------------- | ------ | ------ | -------------------------------------------------------------------------- |
| 1  | **OTel wiring guide** (docs/observability/otel.md)             | High   | 30 min | Closes v3.3.0 promise; zero new code — upstream provides everything        |
| 2  | **Prometheus wiring guide** (docs/observability/prometheus.md) | Med    | 20 min | Same — upstream `prometheus/v3` does `Setup()` → `/metrics`                |
| 3  | **Add `responsewriter.go` to AGENTS.md file tree**             | Low    | 3 min  | New file not listed in architecture diagram                                |
| 4  | **Adopt `projectionhost/v3`**                                  | High   | 3-4h   | Biggest code-reduction: replaces ~100 LOC hand-rolled projection lifecycle |
| 5  | **Adopt `scenario/v3` for usermgmt decider tests**             | Med    | 2-3h   | BDD Given/When/Then for 7 decide functions; more readable than table tests |
| 6  | **usermgmt coverage → 82%**                                    | High   | 3-4h   | SQL read models + OAuth2 flow gaps                                         |
| 7  | **Adopt `CatchUpSubscriber`**                                  | Med    | 2-3h   | Ordered durable projections; pairs with projectionhost                     |
| 8  | **Adopt `snapshot/v3`**                                        | Med    | 2h     | Aggregates with 100+ events (user with many credentials)                   |
| 9  | **OPFS persistence (Phase 2b)**                                | Med    | 6-8h   | Offline queue survives tab close                                           |
| 10 | **Profile dispatch + decode hot paths**                        | Low    | 4h     | Benchmark-driven optimization                                              |
| 11 | **Benchmark projection replay (10K+ events)**                  | Low    | 3h     | Validate checkpoint replay at scale                                        |
| 12 | **Adopt `kv.Cache` for read-model caching**                    | Low    | 2h     | Otter LRU for hot read models                                              |
| 13 | **Adopt `schema/v3` upcasters**                                | Med    | 3h     | Event evolution without rewriting history                                  |
| 14 | **CI: add `test-flake` to GitHub Actions**                     | Low    | 30 min | Catch intermittent failures in CI                                          |
| 15 | **TTFB guardrail for Server-Timing**                           | Low    | 1-2h   | `MeasureForCommit` variant or lint detector                                |
| 16 | **Remove deprecated `ClientIP`**                               | Low    | 1h     | FEATURES.md says PARTIALLY_FUNCTIONAL; scheduled for removal               |
| 17 | **Admin-demo: add OTel + Prometheus showcase**                 | Low    | 2h     | Runnable demo of both observability axes                                   |
| 18 | **Document `delegatingWriter` pattern in AGENTS.md**           | Low    | 15 min | Architecture decision for future wrapper authors                           |
| 19 | **Integration test: real SQLite WAL + projections**            | Med    | 3h     | Validate production SQLite preset end-to-end                               |
| 20 | **Adopt `deriver/v3` for reactive commands**                   | Low    | 3h     | Event→command derivation; saga alternative                                 |
| 21 | **Admin UI: Server-Timing metrics panel**                      | Low    | 2h     | Show timing data in the dashboard itself                                   |
| 22 | **Fuzz test for Server-Timing header formatting**              | Low    | 1h     | Fuzz `HeaderValue()` with adversarial metric names/descriptions            |
| 23 | **codestyle: document nil-receiver pattern**                   | Low    | 30 min | AGENTS.md convention entry for future opts                                 |
| 24 | **Adopt `transport/http/v3` SSEBroker**                        | Low    | 2h     | Evaluate replacing hand-rolled SSE with upstream broker                    |
| 25 | **Migrate `idempotency.go` aliases to thin wrapper**           | Low    | 1h     | Remove local copy, delegate fully to `idempotency/v3`                      |

---

## g) TOP QUESTION I CANNOT FIGURE OUT MYSELF 🤔

**"Should we adopt `projectionhost/v3` now, or wait for ADR-0031 to resolve?"**

ADR-0031 ("Projection Lifecycle — StartProjections vs projectionhost vs CatchUpSubscriber") is marked **Proposed**, not Accepted. The current `StartProjections` works (manual journal replay + `bus.SubscribeAll` + checkpoint support was just added). Adopting `projectionhost/v3` would:

- **Gain:** crash-restart, backoff, poison-message DLQ, per-projection goroutines — all managed
- **Lose:** the just-added checkpoint logic would need rework, and the watermill bus integration (read-your-writes consistency via `BlockPublishUntilSubscriberAck`) might not map cleanly to projectionhost's pull-based model

The decision depends on whether you value **operational maturity** (projectionhost's DLQ + crash-restart) over the **current simplicity** (bus-driven, read-your-writes, ~100 LOC). I can't decide this because it's an architecture-direction question: is cqrs-htmx moving toward full upstream delegation (projectionhost all the way), or keeping its own projection layer?

---

## Technical Metrics Snapshot

| Metric                          | Value                                                                            |
| ------------------------------- | -------------------------------------------------------------------------------- |
| Root source files               | 45 (including responsewriter.go)                                                 |
| Root test files                 | 83                                                                               |
| ADRs                            | 33 (collision fixed)                                                             |
| Go modules                      | 8 (go.work)                                                                      |
| Go version                      | 1.26.4                                                                           |
| go-cqrs-lite                    | v3.4.0                                                                           |
| Root coverage                   | 94.3% (up from 93.8%)                                                            |
| usermgmt coverage               | 79.3%                                                                            |
| Lint issues                     | 0 (all modules)                                                                  |
| errorfamily violations          | 0                                                                                |
| Server-Timing disabled overhead | 3.6 ns/op, 0 allocs                                                              |
| Nix flake check                 | ✅ all checks passed                                                             |
| Nix apps                        | test, test-race, test-flake (new), test-fuzz, lint, build, coverage, errorfamily |
