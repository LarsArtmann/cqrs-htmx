# Pareto Comprehensive Plan — cqrs-htmx (Round 3)

**Created:** 2026-06-18 20:39 · **Method:** Pareto (1%→51%, 4%→64%, 20%→80%) · **Granularity:** every task ≤ 12 min · **Sources:** TODO_LIST.md, ROADMAP.md, FEATURES.md, ADRs, brutal-self-review round 1 + round 2, code audit

> **Supersedes:** `2026-06-18_17-12_comprehensive-remaining-work-plan.md`. Adds 4 items discovered in round-2 self-review (signing/encryption), splits all >12 min tasks into ≤12 min chunks, re-sorts by score. Every "open" item was re-verified against the live codebase — false positives excluded.

## Priority Score

`Score = Impact(1–5) × CustomerValue(1–5)` — sorted descending; ties broken by lower effort.

---

## Pareto Tier Summary

| Tier              | Theme                                  | Tasks  | Est. total  | Value unlock                                                     |
| ----------------- | -------------------------------------- | ------ | ----------- | ---------------------------------------------------------------- |
| **T1 — 1%→~51%**  | Postgres `SessionStore` (production)   | 7      | ~68 min     | Unblocks multi-instance production deployments                   |
| **T2 — 4%→~64%**  | Event schema versioning (upcasters)    | 8      | ~80 min     | Defensive, **time-sensitive** — cheap before first schema change |
| **T3 — ~15%**     | Onboarding example (`examples/basic/`) | 5      | ~48 min     | Adoption — first impression for new consumers                    |
| **T4 — ~15%**     | Observability (OTel + Prometheus)      | 8      | ~78 min     | Operability — tracing + metrics first-class                      |
| **T5 — ~10%**     | Crypto confidence + quick wins         | 9      | ~83 min     | Property tests, status hygiene, small polish                     |
| **T6 — tail**     | Arch polish, benchmarks, fuzz          | 13     | ~115 min    | Hardening; low risk                                              |
| **T7 — deferred** | Demand-driven / blocked (ALL included) | 22     | ~multi      | Logged for completeness; not this cycle                          |
|                   | **TOTAL**                              | **72** | ~9h focused |                                                                  |

---

## Master Task Table (sorted by Score ↓, then Effort ↑)

| #  | Tier | Task                                                                                        | Impact | Value | Effort | Score  | Status / Notes                                              |
| -- | ---- | ------------------------------------------------------------------------------------------- | :----: | :---: | :----: | :----: | ----------------------------------------------------------- |
| 1  | T1   | **ADR 0012**: SQL `SessionStore` pattern (no driver dep in core)                            |   4    |   5   |  10m   | **20** | Production unblock; mirror existing SQL event-store pattern |
| 2  | T1   | `sessions` table DDL in `usermgmt/docs/SQL_STORES.md`                                       |   4    |   5   |  10m   | **20** | Postgres schema                                             |
| 3  | T1   | `SQLSessionStore` skeleton + interface conformance                                          |   4    |   5   |  10m   | **20** | Implements `SessionStore`                                   |
| 4  | T1   | Implement Create/Get/Delete session                                                         |   4    |   5   |  10m   | **20** | Core CRUD                                                   |
| 5  | T1   | Implement TTL/cleanup sweeper                                                               |   4    |   5   |  10m   | **20** | Expired-session reaping                                     |
| 6  | T1   | Contract tests vs `SessionStore` interface                                                  |   4    |   5   |  10m   | **20** | In-memory + SQL parity                                      |
| 7  | T1   | Consumer wiring example in doc                                                              |   4    |   5   |   8m   | **20** | No driver import in core                                    |
| 8  | T2   | **ADR 0013**: upcaster registry design                                                      |   4    |   4   |  10m   | **16** | Time-sensitive defensive infra                              |
| 9  | T2   | Define `Upcaster` func type + `UpcasterRegistry` struct                                     |   4    |   4   |  10m   | **16** | Foundation types                                            |
| 10 | T2   | Implement `Register` + version-chain resolution                                             |   4    |   4   |  10m   | **16** | `rawV(n)→rawV(n+1)` chaining                                |
| 11 | T2   | Wire upcast into `unmarshalPayload` decode path                                             |   4    |   4   |  10m   | **16** | Event payload decode                                        |
| 12 | T2   | Wire upcast into projection decode path                                                     |   4    |   4   |  10m   | **16** | Read-model + Casbin projection                              |
| 13 | T2   | Tests: single-step upcast                                                                   |   4    |   4   |  10m   | **16** | Unit                                                        |
| 14 | T2   | Tests: multi-step chain + missing-version error                                             |   4    |   4   |  10m   | **16** | Unit                                                        |
| 15 | T2   | Property test: upcast preserves fold invariants                                             |   4    |   4   |  10m   | **16** | rapid-based                                                 |
| 16 | T3   | Scaffold `examples/basic/` go.mod + main.go                                                 |   3    |   4   |  10m   | **12** | Onboarding                                                  |
| 17 | T3   | Register 1 command + 1 query in basic example                                               |   3    |   4   |  10m   | **12** | Onboarding                                                  |
| 18 | T3   | HTMX handler + HTML fragment in basic example                                               |   3    |   4   |  10m   | **12** | Onboarding                                                  |
| 19 | T3   | SSE live-update wiring in basic example                                                     |   3    |   4   |  10m   | **12** | Showcases transports                                        |
| 20 | T3   | `examples/basic/` README                                                                    |   3    |   4   |   8m   | **12** | Onboarding                                                  |
| 21 | T5   | Verify README SSE/WS API table completeness vs code; fill gaps                              |   3    |   4   |  10m   | **12** | Consumer-facing                                             |
| 22 | T5   | Property-based tests: sign→encrypt→decrypt→verify round-trip (empty/large/binary)           |   3    |   3   |  12m   | **9**  | NEW (round-2 review) — crypto correctness confidence        |
| 23 | T5   | Fix stale `transport-parity-status.md` (says WSDispatchHandler NOT STARTED — actually done) |   3    |   3   |   8m   | **9**  | Status accuracy                                             |
| 24 | T5   | Sync `TODO_LIST.md` open items to reality                                                   |   3    |   3   |  10m   | **9**  | Status accuracy                                             |
| 25 | T4   | Extract OTel middleware into first-class `otel.go`                                          |   3    |   3   |  10m   | **9**  | Pattern lives only in example today                         |
| 26 | T4   | Config option to accept a `trace.Tracer`                                                    |   3    |   3   |  10m   | **9**  | Observability                                               |
| 27 | T4   | OTel tests with fake tracer                                                                 |   3    |   3   |  10m   | **9**  | Observability                                               |
| 28 | T4   | Prometheus metrics middleware (dispatch latency, error rates)                               |   3    |   3   |  10m   | **9**  | Operability                                                 |
| 29 | T4   | Metrics doc + default collectors                                                            |   3    |   3   |   8m   | **9**  | Operability                                                 |
| 30 | T4   | Metrics tests                                                                               |   3    |   3   |  10m   | **9**  | Operability                                                 |
| 31 | T5   | Root `App` security hooks convenience helper (for consumers using root without usermgmt)    |   2    |   3   |  12m   | **6**  | NEW (round-2 review) — product direction                    |
| 32 | T5   | Document WS backpressure drop-policy in `ws_broadcaster.go` godoc                           |   2    |   3   |   8m   | **6**  | Buffer-full behavior                                        |
| 33 | T5   | `errors.Unwrap` support in `StructuredError`                                                |   2    |   3   |  10m   | **6**  | Chain traversal                                             |
| 34 | T5   | Fix `StartProjections` `time.Sleep(50ms)` → synchronous replay signal                       |   3    |   2   |  12m   | **6**  | NEW (round-2 review) — BLOCKED upstream (`projection/v2`)   |
| 35 | T6   | Shared `FanOut[T]` interface (unify SSE/WS broadcasters)                                    |   2    |   3   |  10m   | **6**  | Arch                                                        |
| 36 | T6   | godoc runnable `Example` for `WSBroadcaster`                                                |   2    |   3   |  10m   | **6**  | Docs                                                        |
| 37 | T5   | `fmt.Stringer` for `StructuredError` (`String()`→JSON)                                      |   2    |   2   |   5m   | **4**  | Convenience                                                 |
| 38 | T5   | Delegate `Chain` to httputil v0.3.0 (7 lines identical semantics)                           |   2    |   2   |   8m   | **4**  | NEW (round-2 review) — DRY                                  |
| 39 | T6   | SSE Heartbeat benchmark                                                                     |   2    |   2   |   8m   | **4**  | Perf                                                        |
| 40 | T6   | `WSBroadcaster` benchmark (mirror SSE)                                                      |   2    |   2   |  10m   | **4**  | Perf                                                        |
| 41 | T6   | Fuzz `WriteWSMessage`                                                                       |   2    |   2   |   8m   | **4**  | Robustness                                                  |
| 42 | T6   | Fuzz `StructuredError.JSON()`                                                               |   2    |   2   |   6m   | **4**  | Robustness                                                  |
| 43 | T6   | `TypedWSBroadcaster[T]`                                                                     |   2    |   2   |  10m   | **4**  | Type safety                                                 |
| 44 | T6   | `io.WriterTo` for `WSMessage` streaming encode                                              |   2    |   2   |  10m   | **4**  | Arch                                                        |
| 45 | T6   | Catalog: register SSE/WS exports                                                            |   2    |   2   |  10m   | **4**  | Docs gen                                                    |
| 46 | T6   | `RotateCSRFToken()` helper impl                                                             |   2    |   2   |   8m   | **4**  | Feature                                                     |
| 47 | T6   | `RotateCSRFToken()` tests                                                                   |   2    |   2   |   8m   | **4**  | Feature                                                     |
| 48 | T6   | Document `hx-ext="ws"` interplay                                                            |   2    |   2   |   8m   | **4**  | Docs                                                        |
| 49 | T6   | Real PostgreSQL integration test (testcontainer) — design                                   |   2    |   2   |  10m   | **4**  | Validates SQLSessionStore                                   |
| 50 | T7   | **OAuth2/OIDC** — ADR design spike                                                          |   3    |   4   |  12m   | **12** | DEFERRED — security-heavy, additive                         |
| 51 | T7   | **OAuth2/OIDC** — `ExternalAccountLinked` event + commands design                           |   3    |   4   |  12m   | **12** | DEFERRED — mirrors WebAuthn                                 |
| 52 | T7   | **OAuth2/OIDC** — impl: provider config (Google/GitHub)                                     |   4    |   4   |  12m   | **16** | DEFERRED — post-design                                      |
| 53 | T7   | **OAuth2/OIDC** — impl: callback handler + state/CSRF                                       |   4    |   4   |  12m   | **16** | DEFERRED — post-design                                      |
| 54 | T7   | **OAuth2/OIDC** — impl: integration tests                                                   |   4    |   4   |  12m   | **16** | DEFERRED — post-design                                      |
| 55 | T7   | **Redis `SessionStore`** — design + interface check                                         |   3    |   4   |  12m   | **12** | DEFERRED — distributed deployments                          |
| 56 | T7   | **Redis `SessionStore`** — impl: CRUD + TTL                                                 |   3    |   4   |  12m   | **12** | DEFERRED                                                    |
| 57 | T7   | **Redis `SessionStore`** — contract tests                                                   |   3    |   4   |  12m   | **12** | DEFERRED                                                    |
| 58 | T7   | **BadgerDB** — embedded event store design                                                  |   2    |   2   |  12m   | **4**  | DEFERRED — niche                                            |
| 59 | T7   | **BadgerDB** — impl + tests                                                                 |   2    |   2   |  12m   | **4**  | DEFERRED                                                    |
| 60 | T7   | **Numeric branded IDs** — type design (auto-increment PKs)                                  |   2    |   3   |  12m   | **6**  | DEFERRED — ADR 0003 pattern                                 |
| 61 | T7   | **Numeric branded IDs** — impl + tests                                                      |   2    |   3   |  12m   | **6**  | DEFERRED                                                    |
| 62 | T7   | **DB migration tooling** — evaluate goose/golang-migrate/gnorm                              |   2    |   3   |  12m   | **6**  | DEFERRED                                                    |
| 63 | T7   | **DB migration tooling** — integration doc                                                  |   2    |   3   |  12m   | **6**  | DEFERRED                                                    |
| 64 | T7   | **BrandNamer** for root module markers                                                      |   2    |   2   |   —    | **4**  | BLOCKED upstream (`go-cqrs-lite` unexported markers)        |
| 65 | T7   | **`WSEventStore`** — design WS replay protocol                                              |   2    |   2   |  12m   | **4**  | DEFERRED — no WS Last-Event-ID spec                         |
| 66 | T7   | **`WSEventStore`** — impl skeleton + replay logic                                           |   2    |   2   |  12m   | **4**  | DEFERRED                                                    |
| 67 | T7   | **`WSEventStore`** — tests                                                                  |   2    |   2   |   9m   | **4**  | DEFERRED                                                    |
| 68 | T7   | **WS `OnClose(fn)`** — callback registration impl                                           |   1    |   2   |  12m   | **2**  | DEFERRED                                                    |
| 69 | T7   | **WS `OnClose(fn)`** — tests                                                                |   1    |   2   |   8m   | **2**  | DEFERRED — duplicates WS lib close handler                  |

---

## Execution Order Recommendation

1. **T1 first** (Postgres `SessionStore`) — highest customer value; unblocks production. ~68 min.
2. **T2 next** (schema versioning) — time-sensitive; cheapest now. ~80 min.
3. **T3 + T5** interleaved — onboarding + quick wins (incl. new crypto property tests). ~131 min.
4. **T4** — observability (OTel + Prometheus). ~78 min.
5. **T6** — opportunistic polish (benchmarks, fuzz, arch). ~115 min.
6. **T7** — only on concrete demand. Multi-session.

## Excluded (verified done / false positives)

README SSE/WS update · CI `govulncheck` · ADR 0010 transport parity · `SECURITY.md` · `integration_transport_test.go` (SSE-error + WS-dispatch) · `BroadcastOnError` unused-param fix · `example_otel_test.go` pattern · signing/encryption opt-in (ADR 0011, done round-2) · `SecurityHooks` extraction (done round-2) · `EventSourcedSetup` interface types (done round-2) · `waitForUser` context-based deadline (done round-2) · integration_test signing/encryption v2.5.0 alignment (done round-2).

## New vs Prior Plan (delta from 17:12 plan)

| Delta | Item                                                                               | Source              |
| ----- | ---------------------------------------------------------------------------------- | ------------------- |
| NEW   | #22 Property-based crypto round-trip tests                                         | Round-2 review      |
| NEW   | #31 Root `App` security hooks helper                                               | Round-2 review      |
| NEW   | #34 Fix `StartProjections` `time.Sleep(50ms)` (BLOCKED upstream)                   | Round-2 review      |
| NEW   | #38 Delegate `Chain` to httputil v0.3.0                                            | Round-2 review      |
| SPLIT | #52-54 OAuth impl (was "multi") → 3× 12m                                           | Max-12min rule      |
| SPLIT | #55-57 Redis SessionStore (was "multi") → 3× 12m                                   | Max-12min rule      |
| SPLIT | #58-59 BadgerDB (was "multi") → 2× 12m                                             | Max-12min rule      |
| SPLIT | #60-61 Numeric branded IDs (was "multi") → 2× 12m                                  | Max-12min rule      |
| SPLIT | #62-63 DB migration tooling (was "multi") → 2× 12m                                 | Max-12min rule      |
| SPLIT | #65-67 WSEventStore (was 45m) → 3× ≤12m                                            | Max-12min rule      |
| SPLIT | #68-69 WS OnClose (was 20m) → 2× ≤12m                                              | Max-12min rule      |
| FIXED | ADR numbering: upcaster ADR → 0013 (0011 is signing/encryption; 0012=SessionStore) | Conflict resolution |
