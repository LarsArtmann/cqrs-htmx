# Comprehensive Execution Plan — All Remaining TODOs

**Date:** 2026-05-24 | **Source:** 20 status reports (2026-05-20 → 2026-05-24) consolidated  
**Root Coverage:** 96.7% | **Usermgmt Coverage:** 91.1% | **Lint:** 0/0 | **Tests:** ✅ all pass

## Current State Summary

| Metric   | Root     | usermgmt | integration_test | datastar-demo |
| -------- | -------- | -------- | ---------------- | ------------- |
| Tests    | ✅ pass  | ✅ pass  | ✅ pass          | build only    |
| Coverage | 96.7%    | 91.1%    | N/A              | N/A           |
| Lint     | 0 issues | 0 issues | 0 issues         | N/A           |
| Race     | ✅ clean | ✅ clean | ✅ clean         | N/A           |

---

## Legend

| Symbol | Meaning                              |
| ------ | ------------------------------------ |
| 🔴     | Security/Correctness — must fix      |
| 🟡     | Coverage gap — improves reliability  |
| 🟢     | API/Feature — adds customer value    |
| 🔵     | Infrastructure/Tooling — improves DX |
| ⬛     | Documentation — improves adoption    |
| 🚫     | BLOCKED — external dependency        |
| ❌     | REJECTED — not recommended           |
| ✅     | ALREADY DONE                         |

---

## Plan — Sorted by Impact × Customer Value / Effort

> Each task is ≤12 min. Grouped by priority tier.

### Tier 1: Coverage Gaps — Test Quality (High Impact, Low Effort)

These directly improve reliability and catch regressions. Most are single-test additions.

| #   | ID    | Task                                                                          | Area                      | Impact | Effort | Status |
| --- | ----- | ----------------------------------------------------------------------------- | ------------------------- | ------ | ------ | ------ |
| 1   | T1-01 | Test `authMode.String()` — 0% coverage, 3 branches (none/required/authorized) | root/options.go           | 🟡     | 5min   | OPEN   |
| 2   | T1-02 | Test `logging.Hijack` non-Hijacker error path — 60% → 100%                    | root/logging.go:223       | 🟡     | 8min   | OPEN   |
| 3   | T1-03 | Test `csrfTokenFromRequest` context fallback path — 66.7% → 100%              | root/csrf.go:221          | 🟡     | 8min   | OPEN   |
| 4   | T1-04 | Test `ratelimit.Push` error path on heap — 75% → 100%                         | root/ratelimit.go:321     | 🟡     | 8min   | OPEN   |
| 5   | T1-05 | Test `sameSite` default case — 83.3% → 100%                                   | root/csrf.go:113          | 🟡     | 5min   | OPEN   |
| 6   | T1-06 | Test `buildGorillaOptions` custom path/domain — 88.9% → 100%                  | root/csrf.go:174          | 🟡     | 8min   | OPEN   |
| 7   | T1-07 | Create error-injecting mock enforcer for usermgmt authz tests                 | usermgmt/test helpers     | 🟡     | 10min  | OPEN   |
| 8   | T1-08 | Test `Authz.Enforce` error path — 75% → 100%                                  | usermgmt/authz.go:158     | 🟡     | 5min   | OPEN   |
| 9   | T1-09 | Test `Authz.EnforceAny` error path — 75% → 100%                               | usermgmt/authz.go:167     | 🟡     | 5min   | OPEN   |
| 10  | T1-10 | Test `Authz.EnforceEx` error path — 75% → 100%                                | usermgmt/authz.go:192     | 🟡     | 5min   | OPEN   |
| 11  | T1-11 | Test `Authz.Apply` remove+add error paths — 69.2% → ~90%                      | usermgmt/authz.go:241     | 🟡     | 10min  | OPEN   |
| 12  | T1-12 | Test remaining authz methods error paths (8 methods × 75%)                    | usermgmt/authz.go:270-369 | 🟡     | 10min  | OPEN   |
| 13  | T1-13 | Test `generateToken` rand.Read error path — 75% → 100%                        | usermgmt/user.go:147      | 🟡     | 8min   | OPEN   |
| 14  | T1-14 | Test `handleAuthEndpoint` timeout/error paths — 80% → ~95%                    | usermgmt/http.go:115      | 🟡     | 10min  | OPEN   |
| 15  | T1-15 | Test `handleLogin` error paths — 80% → ~95%                                   | usermgmt/http.go:98       | 🟡     | 10min  | OPEN   |
| 16  | T1-16 | Test `NewSession` + `SetPasswordWithCost` + `MarshalJSON` error paths         | usermgmt/user.go          | 🟡     | 10min  | OPEN   |
| 17  | T1-17 | Test `RecordFailure` lockout concurrent eviction — 80% → ~95%                 | usermgmt/lockout.go:73    | 🟡     | 10min  | OPEN   |
| 18  | T1-18 | Test `Register` rollback paths (role fail, session fail) — 83.3% → ~95%       | usermgmt/service.go:147   | 🟡     | 10min  | OPEN   |

### Tier 2: Security Hardening (High Impact, Medium Effort)

| #   | ID    | Task                                                                          | Area               | Impact | Effort | Status |
| --- | ----- | ----------------------------------------------------------------------------- | ------------------ | ------ | ------ | ------ |
| 19  | T2-01 | Add CSP default value in SecurityHeadersConfig (currently empty string)       | root/security.go   | 🔴     | 8min   | OPEN   |
| 20  | T2-02 | Set HSTS recommended default value (max-age=31536000)                         | root/security.go   | 🔴     | 5min   | OPEN   |
| 21  | T2-03 | Add `X-Request-ID` response header propagation when generated                 | root/middleware.go | 🟢     | 10min  | OPEN   |
| 22  | T2-04 | Add request method validation (reject GET for command endpoints)              | root/handler.go    | 🔴     | 10min  | OPEN   |
| 23  | T2-05 | Sanitize `Response.Redirect` URL for HTMX (HX-Redirect currently unsanitized) | root/response.go   | 🔴     | 8min   | OPEN   |
| 24  | T2-06 | Log warning when `enrichUserID` extractor returns error (currently silent)    | root/app.go        | 🟡     | 5min   | OPEN   |
| 25  | T2-07 | Evaluate Go 1.26 `CrossOriginProtection` to supplement gorilla/csrf           | root/csrf.go       | 🔴     | 10min  | OPEN   |

### Tier 3: API Polish & Type Safety (Medium Impact, Low Effort)

| #   | ID    | Task                                                                                                    | Area                | Impact | Effort | Status   |
| --- | ----- | ------------------------------------------------------------------------------------------------------- | ------------------- | ------ | ------ | -------- |
| 26  | T3-01 | Export AuthMode constructors: `AuthModeNone()`, `AuthModeRequired()`, `AuthModeAuthorized()`            | root/options.go     | 🟢     | 8min   | OPEN     |
| 27  | T3-02 | Add `NotificationLevel.String()` (fmt.Stringer for logging)                                             | root/notify.go      | 🟢     | 5min   | OPEN     |
| 28  | T3-03 | Add `StatusRecorder` method godoc (4 exported methods undocumented)                                     | root/logging.go     | ⬛     | 5min   | OPEN     |
| 29  | T3-04 | Fix `StatusRecorder.Push` wrapping — breaks `errors.Is()` matching                                      | root/logging.go     | 🟡     | 8min   | OPEN     |
| 30  | T3-05 | Fix `decodeFormValues` — consider `gorilla/schema` for proper form decoding (currently JSON round-trip) | root/decoder.go     | 🟢     | 10min  | RESEARCH |
| 31  | T3-06 | Fix `NewUser` defaults to `RoleViewer` but `Register` adds `RoleUser` — confusing dual role             | usermgmt/service.go | 🟡     | 8min   | OPEN     |
| 32  | T3-07 | Fix `UserStore.Save` O(n) email index scan — add `oldEmail` lookup                                      | usermgmt/store.go   | 🟡     | 10min  | OPEN     |
| 33  | T3-08 | Add `Session.Valid` deprecation note (use `!IsExpired() && TokenMatches()`)                             | usermgmt/user.go    | ⬛     | 3min   | OPEN     |

### Tier 4: Benchmarks & Examples (Medium Impact, Low Effort)

| #   | ID    | Task                                                                                 | Area                       | Impact | Effort | Status |
| --- | ----- | ------------------------------------------------------------------------------------ | -------------------------- | ------ | ------ | ------ |
| 34  | T4-01 | Benchmark new Response methods: `JSON`, `Body`, `WriteString`, `Status`              | root/benchmark_test.go     | 🔵     | 10min  | OPEN   |
| 35  | T4-02 | Add `ExampleWithSuccessStatus`, `ExampleWithMaxBodySize`, `ExampleOnError`           | root/example_test.go       | ⬛     | 10min  | OPEN   |
| 36  | T4-03 | Add `ExampleResponse_JSON`, `ExampleResponse_Body`                                   | root/example_test.go       | ⬛     | 10min  | OPEN   |
| 37  | T4-04 | Add `BenchmarkService_Login`, `BenchmarkService_Register` for usermgmt               | usermgmt/benchmark_test.go | 🔵     | 10min  | OPEN   |
| 38  | T4-05 | Add `BenchmarkTokenMatches` for constant-time comparison perf                        | usermgmt/benchmark_test.go | 🔵     | 5min   | OPEN   |
| 39  | T4-06 | Add usermgmt fuzz tests: `FuzzRegisterRequest_Validate`, `FuzzLoginRequest_Validate` | usermgmt/fuzz_test.go      | 🟡     | 10min  | OPEN   |

### Tier 5: Feature Additions (Medium Impact, Medium Effort)

| #   | ID    | Task                                                                                      | Area              | Impact | Effort | Status |
| --- | ----- | ----------------------------------------------------------------------------------------- | ----------------- | ------ | ------ | ------ |
| 40  | T5-01 | Add `HealthHandler(app)` — checks dispatcher availability, returns 200/503                | root/             | 🟢     | 10min  | OPEN   |
| 41  | T5-02 | Add `GracefulShutdowner` interface with timeout context                                   | root/             | 🟢     | 10min  | OPEN   |
| 42  | T5-03 | Add typed error response: `ErrorResponse{Error, Code, Details}` for machine-readable APIs | root/             | 🟢     | 10min  | OPEN   |
| 43  | T5-04 | Add `NotificationLevel.MarshalJSON()` for structured logging                              | root/notify.go    | 🟢     | 5min   | OPEN   |
| 44  | T5-05 | Add SameSite enforcement tests (Lax/Strict/None matrix)                                   | root/csrf_test.go | 🟡     | 8min   | OPEN   |

### Tier 6: Infrastructure & Tooling (Low-Medium Impact, Medium Effort)

| #   | ID    | Task                                                              | Area                     | Impact | Effort | Status |
| --- | ----- | ----------------------------------------------------------------- | ------------------------ | ------ | ------ | ------ |
| 45  | T6-01 | Add CI coverage threshold: fail below 90%                         | .github/workflows/ci.yml | 🔵     | 5min   | OPEN   |
| 46  | T6-02 | Add `golangci-lint` `modernize` linter for Go 1.22+ patterns      | root/.golangci.yml       | 🔵     | 8min   | OPEN   |
| 47  | T6-03 | Add `datastar-demo` basic test file (at least smoke test)         | examples/datastar-demo/  | 🟡     | 10min  | OPEN   |
| 48  | T6-04 | Clarify datastar-demo ownership (move to go-cqrs-lite repo?)      | examples/                | ⬛     | 3min   | OPEN   |
| 49  | T6-05 | Update TODO_LIST.md — mark 50-improvements items, update coverage | TODO_LIST.md             | ⬛     | 10min  | OPEN   |
| 50  | T6-06 | Update FEATURES.md — coverage, benchmarks, new features           | FEATURES.md              | ⬛     | 10min  | OPEN   |
| 51  | T6-07 | Update CONTRIBUTING.md — reflect current patterns                 | CONTRIBUTING.md          | ⬛     | 10min  | OPEN   |
| 52  | T6-08 | Update AGENTS.md — add new gotchas from 50-improvements session   | AGENTS.md                | ⬛     | 8min   | OPEN   |

### Tier 7: Large / Future (High Impact, High Effort — Deferred)

| #   | ID    | Task                                                          | Area          | Impact | Effort | Status   |
| --- | ----- | ------------------------------------------------------------- | ------------- | ------ | ------ | -------- |
| 53  | T7-01 | OpenTelemetry tracing: Before/AfterDispatch span creation     | root/         | 🟢     | 2h+    | DEFERRED |
| 54  | T7-02 | Nix flake migration for CI                                    | project root  | 🔵     | 2h+    | DEFERRED |
| 55  | T7-03 | Usermgmt SQL store implementation (per ADR 0003)              | usermgmt/     | 🟢     | 4h+    | DEFERRED |
| 56  | T7-04 | Persistent rate limiter (Redis-backed)                        | root/         | 🟢     | 2h+    | DEFERRED |
| 57  | T7-05 | WebSocket/SSE notification bridge                             | root/         | 🟢     | 3h+    | DEFERRED |
| 58  | T7-06 | Request coalescing/deduplication middleware                   | root/         | 🟢     | 2h+    | DEFERRED |
| 59  | T7-07 | Prometheus metrics middleware                                 | root/         | 🟢     | 2h+    | DEFERRED |
| 60  | T7-08 | Session token rotation on role/password change                | usermgmt/     | 🔴     | 1h+    | DEFERRED |
| 61  | T7-09 | Per-session CSRF token binding                                | root/         | 🔴     | 2h+    | DEFERRED |
| 62  | T7-10 | Replace cockroachdb/errors with stdlib (low value, high risk) | root+usermgmt | 🔵     | 4h+    | DEFERRED |

### BLOCKED (External Dependencies)

| #   | ID    | Task                                                 | Blocker                                | Status     |
| --- | ----- | ---------------------------------------------------- | -------------------------------------- | ---------- |
| 63  | TB-01 | BrandNamer for root module marker types              | Upstream go-cqrs-lite unexported types | 🚫 BLOCKED |
| 64  | TB-02 | Dependabot CVE alerts (2 moderate)                   | `gh auth login` token expired          | 🚫 BLOCKED |
| 65  | TB-03 | gorilla/csrf CVE remediation (TrustedOrigins bypass) | No upstream fix available              | 🚫 BLOCKED |

### RESOLVED / REJECTED (For Reference Only)

| #   | ID  | Task                          | Resolution                                            |
| --- | --- | ----------------------------- | ----------------------------------------------------- |
| —   | —   | TypedHandler[T] on App        | ✅ RESOLVED: RenderTemplResult[T] already covers this |
| —   | —   | UserID type split             | ✅ RESOLVED: ADR 0002 — keep separate intentionally   |
| —   | —   | writeJSON dedup root↔usermgmt | ❌ REJECTED: would couple independent modules         |
| —   | —   | ValidateID adoption           | ✅ RESOLVED: ParseUserID already validates            |
| —   | —   | Publisher/Subscriber ISP      | ❌ NOT APPLICABLE: cqrs-htmx doesn't publish events   |
| —   | —   | CatalogMeta → zero-cost API   | DEFERRED: upstream deprecated but functional          |
| —   | —   | LSP stale cache investigation | DEFERRED: cosmetic only, not a real issue             |

---

## Summary Statistics

| Category                      | Count        | Est. Total Effort |
| ----------------------------- | ------------ | ----------------- |
| Tier 1: Coverage Gaps         | 18 tasks     | ~2.5h             |
| Tier 2: Security              | 7 tasks      | ~1h               |
| Tier 3: API Polish            | 8 tasks      | ~1h               |
| Tier 4: Benchmarks & Examples | 6 tasks      | ~1h               |
| Tier 5: Features              | 5 tasks      | ~45min            |
| Tier 6: Infrastructure        | 8 tasks      | ~1.5h             |
| Tier 7: Deferred (Large)      | 10 tasks     | 22h+              |
| Blocked                       | 3 tasks      | —                 |
| **Total Actionable**          | **52 tasks** | **~8h**           |

## Recommended Execution Order

1. **T1-01 → T1-06** — Root coverage gaps (45 min, immediate test quality)
2. **T2-01 → T2-04** — Security hardening (33 min, critical safety)
3. **T1-07 → T1-18** — Usermgmt coverage gaps (2h, mock enforcer + error paths)
4. **T3-01 → T3-03, T3-06 → T3-07** — API polish (40 min, consumer-facing)
5. **T4-01 → T4-06** — Benchmarks & examples (55 min, documentation quality)
6. **T5-01 → T5-05** — Feature additions (45 min, new capabilities)
7. **T6-01 → T6-08** — Infrastructure & docs (1.5h, project health)
8. **T3-04 → T3-05, T3-08** — Minor fixes (20 min, last polish)
9. **T2-05 → T2-07** — Advanced security (25 min, deeper hardening)

---

_Generated from 20 status reports spanning 2026-05-20 to 2026-05-24._
