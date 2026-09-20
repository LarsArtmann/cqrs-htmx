# Status Report — cqrs-htmx

**Date:** 2026-05-19 23:29 | **Session:** Branded UserID migration + comprehensive review
**Commit Base:** `eddf90d` (docs: comprehensive status report post-9-skill review)

---

## Metrics Snapshot

| Metric     | Root Module | usermgmt    | Total    |
| ---------- | ----------- | ----------- | -------- |
| Coverage   | 94.8%       | 85.0%       | ~92%     |
| Build      | PASS        | PASS        | PASS     |
| Tests      | PASS (race) | PASS (race) | PASS     |
| Lint       | 0 issues    | 0 issues    | 0 issues |
| Go vet     | PASS        | PASS        | PASS     |
| Prod files | 15          | 10          | 25       |
| Test files | 20          | 6           | 26       |
| Prod LOC   | ~3,967      | —           | ~3,967   |
| Test LOC   | ~7,070      | —           | ~7,070   |
| Go version | 1.26.2      | 1.26.2      | —        |

---

## a) FULLY DONE

### This Session — Branded UserID Migration (17/18 violations fixed)

Created `usermgmt/id.go` with `UserID = brandid.ID[userBrand, string]` via `go-branded-id v0.1.0`. All user ID fields and parameters across the usermgmt submodule are now strongly typed:

| File                     | What Changed                                                                                                                                         |
| ------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| `usermgmt/id.go`         | **NEW** — branded `UserID` type alias, `NewUserID()` constructor                                                                                     |
| `usermgmt/user.go`       | `User.ID`, `Session.UserID` → `UserID`; `NewUser()`, `NewSession()` params                                                                           |
| `usermgmt/store.go`      | `UserStore`/`SessionStore` interfaces + `InMemoryUserStore`/`InMemorySessionStore` impls — all `string` ID params → `UserID`; maps keyed by `UserID` |
| `usermgmt/service.go`    | `RegisterRequest.ID` → `UserID`; `GetUser()`, `UpdateRoles()`, `ChangePassword()`, `logAuth()` params                                                |
| `usermgmt/authz.go`      | `RolesForUser()`, `ImplicitRolesForUser()`, `ImplicitPermissionsForUser()`, `DomainsForUser()` — accept `UserID`, `.String()` at Casbin boundary     |
| `usermgmt/middleware.go` | `UserIDFromRequest()` — `.String()` for cqrs-htmx compatibility                                                                                      |
| 6 test files             | All string user IDs → `NewUserID("...")` comparisons                                                                                                 |

**Key design decisions:**

- `GroupPolicy.User` and `Domain` remain `string` — Casbin is an external system boundary, no benefit from branding
- `UserIDFromRequest()` returns `string` — preserves cqrs-htmx `UserIDExtractor` compatibility (bridge pattern)
- `InMemoryUserStore.emails` map changed from `map[string]string` → `map[string]UserID` for type consistency
- `RegisterRequest.Validate()` uses `r.ID.IsZero()` instead of `r.ID == ""`

### Previous Sessions — Complete Feature Set (29/29 features)

All 29 features in FEATURES.md are `FULLY_FUNCTIONAL`:

- App Builder, Command/Query Dispatch, JSON/Form Decoding
- Casbin Authorization + Middleware, User Identity Propagation
- HTMX Request Context + Accessors + Response Builder
- Notifications, Templ Integration, Error Classification
- Correlation ID, Request ID, Request Validation, Timeout Propagation
- Request Logging, Rate Limiting, Security Headers, CSRF Protection
- Middleware Chain, Lifecycle Hooks, Swap Strategies, Handler Options

### Previous Sessions — All TODO Items P0-P4 Complete

All items in `TODO_LIST.md` sections P0-P4 are marked `[x] DONE`:

- 6/6 P0 Security items
- 7/7 P1 Code Quality items
- 5/5 P2 Architecture items
- 12/12 P3 Feature items
- 6/6 P4 Polish items

---

## b) PARTIALLY DONE

### P5 Open Items — 11 items, 0 done

The P5 section in `TODO_LIST.md` contains 11 open items from the 2026-05-19 comprehensive review. None have been started. These are categorized as:

- **Correctness (2):** context mismatch in `applyQueryResponse`, nil context in tests
- **Production Safety (2):** rate limiter max-keys cap, CSRF Protect instance caching
- **Deduplication High (4):** generic HTMX accessor, generic decoder, generic validation, unified notification
- **Deduplication Medium (3):** shared logging context, shared error handler core, generic ID helpers
- **File Organization (1):** split `csrf.go` (445 lines)

---

## c) NOT STARTED

| #  | Item                                                                                          | Priority     | Impact                                       |
| -- | --------------------------------------------------------------------------------------------- | ------------ | -------------------------------------------- |
| 1  | Fix context mismatch in `applyQueryResponse` (handler.go:124)                                 | Correctness  | Low — render rarely fails from timeout       |
| 2  | Fix nil context in usermgmt tests (handler_test.go:314,322)                                   | Correctness  | Low — test-only, SA1012 warning              |
| 3  | Add max-keys cap to rate limiter                                                              | Production   | Medium — memory leak under pathological keys |
| 4  | Cache CSRF Protect instance                                                                   | Production   | Medium — per-request allocation overhead     |
| 5  | Generic HTMX accessor (collapse 8 functions)                                                  | Dedup        | Medium — 8→1 function                        |
| 6  | Generic decoder (collapse 4 functions)                                                        | Dedup        | Medium — 4→1 function                        |
| 7  | Generic validation (collapse 2 functions)                                                     | Dedup        | Low — 2→1 function                           |
| 8  | Unified notification implementation                                                           | Dedup        | Medium — merge 2 parallel impls              |
| 9  | Shared logging context extraction                                                             | Dedup        | Low — 3 blocks → 1 helper                    |
| 10 | Shared error handler core                                                                     | Dedup        | Low — 2 paths → 1 function                   |
| 11 | Generic ID helpers (`parseID[T]`)                                                             | Dedup        | Low — 3 functions → 1 generic                |
| 12 | Split `csrf.go` into `csrf.go` + `csrf_helpers.go`                                            | Organization | Low — readability                            |
| 13 | usermgmt coverage improvement (85% → 95%+)                                                    | Quality      | Medium                                       |
| 14 | Update FEATURES.md metrics (coverage: 94.7 → 94.8, add usermgmt branded UserID)               | Docs         | Low                                          |
| 15 | FEATURES.md still says "29 features" — branded UserID is a feature-level change not reflected | Docs         | Low                                          |

---

## d) TOTALLY FUCKED UP

**Nothing is fucked up.** Both modules build clean, all tests pass with race detector, zero lint issues, zero vet issues.

**Known wart:** LSP shows ~31 stale `golangci_lint_ls` warnings that CLI `golangci-lint run` does not report — this is a known LSP cache issue (AGENTS.md gotcha #8), not a real problem.

---

## e) WHAT WE SHOULD IMPROVE!

### Immediate Concerns

1. **usermgmt coverage gap (85% vs 94.8% root)** — The branded UserID migration didn't add new tests for the `id.go` type itself. The `NewUserID()` constructor, `UserID.IsZero()`, and the JSON round-trip through `RegisterRequest.ID` are tested implicitly but not explicitly.

2. **FEATURES.md and TODO_LIST.md are stale** — Neither reflects the branded UserID migration just completed. Coverage metric says 94.7% but is now 94.8%. The usermgmt `id.go` file isn't in the architecture diagram. TODO_LIST.md doesn't have the branded UserID work tracked.

3. **Rate limiter memory leak is the highest-impact unfixed item** — `perKeyLimiter.limiters` grows unbounded. Under per-IP limiting with many unique IPs, this leaks memory indefinitely. This should be P0 for production deployments.

4. **CSRF Protect creates a new gorilla/csrf instance per request** — `executeCSRFValidation` in `csrf.go` calls `csrf.Protect()` on every request when `CSRFProtect` handler option is used. This should be cached.

5. **The P5 deduplication items would significantly reduce maintenance surface** — 8 HTMX accessor functions, 4 decoder functions, 2 validation functions could each collapse to a single generic. This would make the codebase much easier to reason about.

### Architectural Observations

6. **usermgmt is still `string`-typed at the Casbin boundary** — `GroupPolicy.User`, `GroupPolicy.Domain`, `Policy.Subject`, `Policy.Domain` are all raw strings. This is intentional (Casbin is external), but it means the type safety has a seam. If Casbin ever gets a typed Go API, we could close this gap.

7. **`UserIDFromRequest` returns `string` for cqrs-htmx compatibility** — This bridge function is the only place where `UserID` leaks back to `string`. It works correctly, but it's a reminder that the two modules (root + usermgmt) have independent ID type systems. A future unification (making usermgmt depend on root's `id.UserID`) would close this gap but create a circular dependency — not worth it.

8. **No integration tests between root module and usermgmt** — The modules are tested independently. The `UserIDFromRequest` bridge is the only connection point, and it's tested in `handler_test.go`, but a full integration test (register → get user → cqrs dispatch with user context) would catch seam issues.

---

## f) Top 25 Things We Should Get Done Next

### Tier 1: Production Safety (do first)

| # | Item                                                              | Effort | Impact                            |
| - | ----------------------------------------------------------------- | ------ | --------------------------------- |
| 1 | Add `MaxKeys` cap to rate limiter with LRU eviction               | 2h     | Prevents OOM in production        |
| 2 | Cache CSRF Protect instance at handler creation time              | 1h     | Eliminates per-request allocation |
| 3 | Fix context mismatch in `applyQueryResponse` (use enriched `ctx`) | 15m    | Correctness consistency           |
| 4 | Fix nil context in usermgmt tests → `context.TODO()`              | 10m    | Silences SA1012 warnings          |

### Tier 2: Deduplication (high leverage)

| # | Item                                                                | Effort | Impact                                      |
| - | ------------------------------------------------------------------- | ------ | ------------------------------------------- |
| 5 | Generic HTMX accessor — collapse 8 functions to 1                   | 1h     | 8→1 function, major readability win         |
| 6 | Generic decoder — collapse 4 decoder functions to 1                 | 1h     | 4→1, eliminates the most duplicated pattern |
| 7 | Unified notification — merge `notifyOption` + `triggerNotification` | 30m    | 2→1 implementation                          |
| 8 | Generic validation — collapse `ValidateCommand`/`ValidateQuery`     | 30m    | 2→1                                         |

### Tier 3: Documentation Accuracy

| #  | Item                                                              | Effort | Impact   |
| -- | ----------------------------------------------------------------- | ------ | -------- |
| 9  | Update FEATURES.md — add branded UserID, update coverage to 94.8% | 15m    | Accuracy |
| 10 | Update TODO_LIST.md — mark branded UserID as done, update metrics | 15m    | Accuracy |
| 11 | Update AGENTS.md architecture tree — add `id.go` to usermgmt      | 5m     | Accuracy |

### Tier 4: Test Quality

| #  | Item                                                                                 | Effort | Impact                   |
| -- | ------------------------------------------------------------------------------------ | ------ | ------------------------ |
| 12 | Add explicit tests for `usermgmt/id.go` (NewUserID, IsZero, String, JSON round-trip) | 30m    | Coverage + explicitness  |
| 13 | Improve usermgmt coverage from 85% → 90%+                                            | 2h     | Reliability              |
| 14 | Add integration test: usermgmt register → cqrs-htmx dispatch with user context       | 1h     | Cross-module correctness |

### Tier 5: Code Organization

| #  | Item                                                                      | Effort | Impact      |
| -- | ------------------------------------------------------------------------- | ------ | ----------- |
| 15 | Split `csrf.go` (445 lines) → `csrf.go` + `csrf_helpers.go`               | 30m    | Readability |
| 16 | Generic ID helpers — `parseID[T]` for ParseUserID/CorrelationID/RequestID | 30m    | 3→1         |
| 17 | Shared logging context extraction helper                                  | 30m    | 3→1 blocks  |
| 18 | Shared error handler core extraction                                      | 30m    | 2→1 paths   |

### Tier 6: Future-Proofing

| #  | Item                                                                       | Effort | Impact                                      |
| -- | -------------------------------------------------------------------------- | ------ | ------------------------------------------- |
| 19 | Consider `TriggerID` branded type in `HTMXRequest`                         | 30m    | Consistency (was skipped in this migration) |
| 20 | Add `SessionToken` branded type for session tokens                         | 1h     | Prevents token/ID confusion                 |
| 21 | Evaluate go-branded-id for numeric IDs (future store backends)             | 1h     | Readiness for SQL stores                    |
| 22 | Add fuzz tests for decoder functions                                       | 2h     | Robustness                                  |
| 23 | Benchmark usermgmt operations (auth, session, bcrypt)                      | 1h     | Performance baseline                        |
| 24 | Consider extracting shared branded ID types to a separate internal package | 2h     | Cross-module type sharing                   |
| 25 | Investigate LSP stale diagnostics issue (golangci_lint_ls)                 | 2h     | Developer experience                        |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should `usermgmt.UserID` be the same type as the root module's `cqrshtmx.UserID` (which is `id.UserID` from `go-cqrs-lite`)?**

Currently they are completely independent branded types:

- Root module: `type UserID = id.UserID` (ULID-backed, from `go-cqrs-lite/core/pkg/id`)
- usermgmt: `type UserID = brandid.ID[userBrand, string]` (string-backed, from `go-branded-id`)

These are **incompatible types**. If a consumer uses both modules, they cannot pass a `usermgmt.UserID` where `cqrshtmx.UserID` is expected, or vice versa. The `UserIDFromRequest` bridge explicitly converts via `.String()`.

**Options:**

1. **Keep them separate** (current) — Clean dependency boundaries, no circular deps, but consumers must convert
2. **Make usermgmt depend on `go-cqrs-lite/core/pkg/id`** — Share the same `id.UserID` type, but usermgmt now depends on the CQRS infrastructure (feels wrong for a general user management package)
3. **Extract a shared `go-user-id` package** — Both modules depend on a tiny shared type package. Clean, but adds another package to maintain
4. **Make usermgmt depend on root module** — Circular dependency (root already imports usermgmt indirectly through consumer code, but not in go.mod)

I cannot determine the right tradeoff here because it depends on your product direction: is `usermgmt` meant to be used standalone (with any Go HTTP framework), or is it always paired with `cqrs-htmx`?

---

## Changes in This Session

### Uncommitted Files

| File                       | Status               | Lines Changed |
| -------------------------- | -------------------- | ------------- |
| `usermgmt/id.go`           | NEW (not yet staged) | +7            |
| `usermgmt/go.mod`          | MODIFIED             | +1 dep        |
| `usermgmt/go.sum`          | MODIFIED             | +2 checksums  |
| `usermgmt/user.go`         | MODIFIED             | ~8 changes    |
| `usermgmt/store.go`        | MODIFIED             | ~12 changes   |
| `usermgmt/service.go`      | MODIFIED             | ~10 changes   |
| `usermgmt/authz.go`        | MODIFIED             | ~8 changes    |
| `usermgmt/middleware.go`   | MODIFIED             | 1 change      |
| `usermgmt/user_test.go`    | MODIFIED             | ~19 changes   |
| `usermgmt/service_test.go` | MODIFIED             | ~16 changes   |
| `usermgmt/handler_test.go` | MODIFIED             | ~6 changes    |
| `usermgmt/authz_test.go`   | MODIFIED             | ~3 changes    |
| `usermgmt/lockout_test.go` | MODIFIED             | 1 change      |
| `AGENTS.md`                | MODIFIED             | ~4 additions  |

### Total Diff Stats

```
AGENTS.md                | 16 +++++++----
usermgmt/authz.go        | 16 +++++------
usermgmt/authz_test.go   |  6 ++--
usermgmt/handler_test.go | 16 +++++------
usermgmt/lockout_test.go |  5 +++-
usermgmt/service_test.go | 74 ++++++++++++++++++++++++++++++--------------
usermgmt/user_test.go    | 48 +++++++++++++++----------------
7 files changed, 109 insertions(+), 72 deletions(-)
```

(Plus new files: `usermgmt/id.go`, modified `usermgmt/go.mod`, `usermgmt/go.sum`)
