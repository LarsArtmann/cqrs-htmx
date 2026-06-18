# Comprehensive Remaining-Work Plan — cqrs-htmx

**Created:** 2026-06-18 17:12 · **Method:** Pareto (1% → 51%, 4% → 64%, 20% → 80%) · **Granularity:** every task ≤ 12 min · **Verified against:** live codebase (not stale status reports)

> **Methodology note.** Every "open" item from prior status reports was re-verified against the actual code before inclusion. Several previously-listed TODOs were **false positives** and are excluded: README SSE/WS is already updated; CI already runs `govulncheck`; ADR 0010 exists; `SECURITY.md` exists; `integration_transport_test.go` already covers SSE-error + WS-dispatch flows; `BroadcastOnError` has no unused param. The items below are **genuinely open**.

## Priority Score

`Score = Impact(1–5) × CustomerValue(1–5)` — sorted descending; ties broken by lower effort. **DEFERRED** items are listed at the end for completeness ("include ALL TODOs") but are not targeted this cycle.

---

## Pareto Tier Summary

| Tier              | Theme                               | Tasks  | Est. total  | Value unlock                                                     |
| ----------------- | ----------------------------------- | ------ | ----------- | ---------------------------------------------------------------- |
| **T1 — 1%→~20%**  | Status/code hygiene quick wins      | 6      | ~51 min     | Removes stale-info confusion; ships small polish                 |
| **T2 — 4%→~30%**  | Event schema versioning (upcasters) | 8      | ~80 min     | Defensive, **time-sensitive** — cheap before first schema change |
| **T3 — 10%→~20%** | PostgreSQL `SessionStore`           | 7      | ~68 min     | Unblocks multi-instance production deployments                   |
| **T4 — 15%→~15%** | Onboarding example + observability  | 11     | ~106 min    | Adoption (examples) + operability (otel/metrics)                 |
| **T5 — tail**     | Benchmarks, fuzz, arch polish       | 13     | ~110 min    | Hardening; low risk                                              |
| **T6 — deferred** | Demand-driven / blocked             | 10     | —           | Logged for completeness; not this cycle                          |
|                   | **TOTAL**                           | **55** | ~7h focused |                                                                  |

---

## Master Task Table (sorted by Score ↓, then Effort ↑)

| #   | Tier | Task                                                                                        | Impact | Value | Effort | Score  | Status / Notes                                              |
| --- | ---- | ------------------------------------------------------------------------------------------- | :----: | :---: | :----: | :----: | ----------------------------------------------------------- |
| 15  | T3   | **ADR 0012**: SQL `SessionStore` pattern (no driver dep in core)                            |   4    |   5   |  10m   | **20** | Production unblock; mirror existing SQL event-store pattern |
| 16  | T3   | `sessions` table DDL in `usermgmt/docs/SQL_STORES.md`                                       |   4    |   5   |  10m   | **20** | Postgres schema                                             |
| 17  | T3   | `SQLSessionStore` skeleton + interface conformance                                          |   4    |   5   |  10m   | **20** | Implements `SessionStore`                                   |
| 18  | T3   | Implement Create/Get/Delete session                                                         |   4    |   5   |  10m   | **20** | Core CRUD                                                   |
| 19  | T3   | Implement TTL/cleanup sweeper                                                               |   4    |   5   |  10m   | **20** | Expired-session reaping                                     |
| 20  | T3   | Contract tests vs `SessionStore` interface                                                  |   4    |   5   |  10m   | **20** | In-memory + SQL parity                                      |
| 21  | T3   | Consumer wiring example in doc                                                              |   4    |   5   |   8m   | **20** | No driver import in core                                    |
| 7   | T2   | **ADR 0011**: upcaster registry design                                                      |   4    |   4   |  10m   | **16** | Time-sensitive defensive infra                              |
| 8   | T2   | Define `Upcaster` func type + `UpcasterRegistry` struct                                     |   4    |   4   |  10m   | **16** | Foundation types                                            |
| 9   | T2   | Implement `Register` + version-chain resolution                                             |   4    |   4   |  10m   | **16** | `rawV(n)→rawV(n+1)` chaining                                |
| 10  | T2   | Wire upcast into `unmarshalPayload` decode path                                             |   4    |   4   |  10m   | **16** | Event payload decode                                        |
| 11  | T2   | Wire upcast into projection decode path                                                     |   4    |   4   |  10m   | **16** | Read-model + Casbin projection                              |
| 12  | T2   | Tests: single-step upcast                                                                   |   4    |   4   |  10m   | **16** | Unit                                                        |
| 13  | T2   | Tests: multi-step chain + missing-version error                                             |   4    |   4   |  10m   | **16** | Unit                                                        |
| 14  | T2   | Property test: upcast preserves fold invariants                                             |   4    |   4   |  10m   | **16** | rapid-based                                                 |
| 3   | T1   | Verify README SSE/WS API table completeness vs code; fill gaps                              |   3    |   4   |  10m   | **12** | Consumer-facing                                             |
| 22  | T4   | Scaffold `examples/basic/` go.mod + main.go                                                 |   3    |   4   |  10m   | **12** | Onboarding                                                  |
| 23  | T4   | Register 1 command + 1 query in basic example                                               |   3    |   4   |  10m   | **12** | Onboarding                                                  |
| 24  | T4   | HTMX handler + HTML fragment in basic example                                               |   3    |   4   |  10m   | **12** | Onboarding                                                  |
| 25  | T4   | SSE live-update wiring in basic example                                                     |   3    |   4   |  10m   | **12** | Showcases transports                                        |
| 26  | T4   | `examples/basic/` README                                                                    |   3    |   4   |   8m   | **12** | Onboarding                                                  |
| 1   | T1   | Fix stale `transport-parity-status.md` (says WSDispatchHandler NOT STARTED — actually done) |   3    |   3   |   8m   | **9**  | Status accuracy                                             |
| 2   | T1   | Sync `TODO_LIST.md` open items to reality                                                   |   3    |   3   |  10m   | **9**  | Status accuracy                                             |
| 27  | T4   | Extract OTel middleware into first-class `otel.go`                                          |   3    |   3   |  10m   | **9**  | Pattern lives only in example today                         |
| 28  | T4   | Config option to accept a `trace.Tracer`                                                    |   3    |   3   |  10m   | **9**  | Observability                                               |
| 29  | T4   | OTel tests with fake tracer                                                                 |   3    |   3   |  10m   | **9**  | Observability                                               |
| 30  | T4   | Prometheus metrics middleware (dispatch latency, error rates)                               |   3    |   3   |  10m   | **9**  | Operability                                                 |
| 31  | T4   | Metrics doc + default collectors                                                            |   3    |   3   |   8m   | **9**  | Operability                                                 |
| 32  | T4   | Metrics tests                                                                               |   3    |   3   |  10m   | **9**  | Operability                                                 |
| 4   | T1   | Document WS backpressure drop-policy in `ws_broadcaster.go` godoc                           |   2    |   3   |   8m   | **6**  | Buffer-full behavior                                        |
| 6   | T1   | `errors.Unwrap` support in `StructuredError`                                                |   2    |   3   |  10m   | **6**  | Chain traversal                                             |
| 37  | T5   | Shared `FanOut[T]` interface (unify SSE/WS broadcasters)                                    |   2    |   3   |  10m   | **6**  | Arch                                                        |
| 44  | T5   | godoc runnable `Example` for `WSBroadcaster`                                                |   2    |   3   |  10m   | **6**  | Docs                                                        |
| 5   | T1   | `fmt.Stringer` for `StructuredError` (`String()`→JSON)                                      |   2    |   2   |   5m   | **4**  | Convenience                                                 |
| 33  | T5   | SSE Heartbeat benchmark                                                                     |   2    |   2   |   8m   | **4**  | Perf                                                        |
| 34  | T5   | `WSBroadcaster` benchmark (mirror SSE)                                                      |   2    |   2   |  10m   | **4**  | Perf                                                        |
| 35  | T5   | Fuzz `WriteWSMessage`                                                                       |   2    |   2   |   8m   | **4**  | Robustness                                                  |
| 36  | T5   | Fuzz `StructuredError.JSON()`                                                               |   2    |   2   |   6m   | **4**  | Robustness                                                  |
| 38  | T5   | `TypedWSBroadcaster[T]`                                                                     |   2    |   2   |  10m   | **4**  | Type safety                                                 |
| 39  | T5   | `io.WriterTo` for `WSMessage` streaming encode                                              |   2    |   2   |  10m   | **4**  | Arch                                                        |
| 40  | T5   | Catalog: register SSE/WS exports                                                            |   2    |   2   |  10m   | **4**  | Docs gen                                                    |
| 41  | T5   | `RotateCSRFToken()` helper impl                                                             |   2    |   2   |   8m   | **4**  | Feature                                                     |
| 42  | T5   | `RotateCSRFToken()` tests                                                                   |   2    |   2   |   8m   | **4**  | Feature                                                     |
| 43  | T5   | Document `hx-ext="ws"` interplay                                                            |   2    |   2   |   8m   | **4**  | Docs                                                        |
| 45  | T5   | Real PostgreSQL integration test (testcontainer) — design                                   |   2    |   2   |  10m   | **4**  | Validates SQLSessionStore                                   |
| 46  | T6   | **OAuth2/OIDC** — ADR design spike                                                          |   3    |   4   |  12m   | **12** | DEFERRED — security-heavy, additive                         |
| 47  | T6   | **OAuth2/OIDC** — `ExternalAccountLinked` event + commands design                           |   3    |   4   |  12m   | **12** | DEFERRED — mirrors WebAuthn                                 |
| 48  | T6   | **OAuth2/OIDC** — implementation                                                            |   4    |   4   | multi  | **16** | DEFERRED — 2+ sessions                                      |
| 49  | T6   | **Redis `SessionStore`**                                                                    |   3    |   4   | multi  | **12** | DEFERRED — distributed deployments                          |
| 50  | T6   | **BadgerDB** embedded store                                                                 |   2    |   2   | multi  | **4**  | DEFERRED — niche                                            |
| 51  | T6   | **Numeric branded IDs** for auto-increment PKs                                              |   2    |   3   | multi  | **6**  | DEFERRED — ADR 0003 pattern                                 |
| 52  | T6   | **DB migration tooling** (goose / golang-migrate)                                           |   2    |   3   | multi  | **6**  | DEFERRED                                                    |
| 53  | T6   | **BrandNamer** for root module markers                                                      |   2    |   2   |   —    | **4**  | BLOCKED upstream (`go-cqrs-lite` unexported markers)        |
| 54  | T6   | **`WSEventStore`** (WS replay)                                                              |   2    |   2   |  45m   | **4**  | DEFERRED — no WS Last-Event-ID spec                         |
| 55  | T6   | **WS `OnClose(fn)`**                                                                        |   1    |   2   |  20m   | **2**  | DEFERRED — duplicates WS lib close handler                  |

---

## Execution Order Recommendation

1. **T3 first** (Postgres SessionStore) — highest customer value; unblocks production.
2. **T2 next** (schema versioning) — time-sensitive; cheapest now.
3. **T1 + T4 interleaved** — quick wins + onboarding.
4. **T5** as opportunistic polish.
5. **T6** only on concrete demand.

## Excluded (verified done / false positives)

README SSE/WS update · CI `govulncheck` · ADR 0010 transport parity · `SECURITY.md` · `integration_transport_test.go` (SSE-error + WS-dispatch) · `BroadcastOnError` unused-param fix · `govulncheck` in CI · `example_otel_test.go` pattern.
