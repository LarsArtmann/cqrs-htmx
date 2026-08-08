# Full Code Review — 2026-06-14

**Reviewer:** Senior Software Architect (Crush) | **Scope:** All 4 modules (root, usermgmt, integration_test, datastar-demo)

---

## Verdict

**Production-grade library with one critical concurrency bug (now fixed) and minor documentation debt.**

The codebase is mature, well-structured, and shows exceptional attention to type safety, error handling, and API ergonomics. 570+ tests, 96%+ coverage, 0 lint issues across all modules. The review found and fixed **1 critical production bug** (data race), **6 documentation split-brains**, and **1 incomplete test**.

---

## Fixes Applied This Session

### 1. CRITICAL: Data Race in Rate Limiter (Fixed)

**File:** `ratelimit_middleware.go:76`

**Problem:** `perKeyLimiter.limiter()` read `entry.lastUsed` **after** releasing the RLock (line 74→76), while a concurrent goroutine could write `entry.lastUsed` at line 86 under the write lock. The race was non-deterministic (~1 in 5 runs with `-race`) and affected **all concurrent users of `RateLimiterMiddleware`** — a core production code path.

```
WARNING: DATA RACE
Write at 0x... by goroutine A:
  perKeyLimiter.limiter() ratelimit_middleware.go:86  // entry.lastUsed = time.Now()
Previous read at 0x... by goroutine B:
  perKeyLimiter.limiter() ratelimit_middleware.go:76  // time.Since(entry.lastUsed)
```

**Fix:** Moved the freshness check (`entry.lastUsed`) inside the RLock-held region. All writes to `entry.lastUsed` are already under the write lock, so reading it under RLock is now safe. Verified with **10/10 clean race-detector runs** (was ~20% failure rate before).

**Impact:** This was a real production bug. Any deployment using the rate limiter with concurrent requests had a data race. Go's race detector is non-deterministic, so this could silently corrupt rate-limit state in production.

### 2. Orphaned/Truncated `CSRFMiddleware` Doc Comment (Fixed)

**Files:** `csrf_middleware.go`, `csrf_context.go`

**Problem:** The doc comment for `CSRFMiddleware` was split across two files. The actual function in `csrf_middleware.go` had only a fragment `// )(mux)` above it. The full doc comment lived as an orphaned, **truncated** block at the end of `csrf_context.go` (cut off mid-code-block at `cqrshtmx.HTMXMiddleware,`). This meant `go doc CSRFMiddleware` showed nothing useful.

**Fix:** Wrote a complete, proper doc comment on the function in `csrf_middleware.go`. Removed the orphaned truncated comment from `csrf_context.go`.

### 3. Misplaced `RateLimiter` Struct Doc (Fixed)

**File:** `ratelimit_config.go`

**Problem:** The `RateLimiter` struct's doc comment started with copy-pasted text about `RateLimiterMiddleware` (7 lines of wrong subject before the actual description). Also had an orphaned `// limiterEntry holds...` comment at the end of the file (the struct is in `ratelimit_middleware.go`).

**Fix:** Replaced with a correct, concise doc comment. Removed the orphaned `limiterEntry` comment.

### 4. `Apply` Method Doc Comment Split Brain (Fixed)

**Files:** `usermgmt/authz_types.go`, `usermgmt/authz_policies.go`

**Problem:** The `Apply` method's doc comment was split: 5 lines orphaned at the end of `authz_types.go`, 1 fragment line (`// state is partially updated...`) above the actual method in `authz_policies.go`. Neither file had a complete, properly-attached doc.

**Fix:** Consolidated the full doc comment onto the `Apply` method in `authz_policies.go`. Removed the orphan from `authz_types.go`.

### 5. Orphaned Duplicate `RolesForUser` Doc (Fixed)

**File:** `usermgmt/authz_policies.go`

**Problem:** `// RolesForUser returns...` appeared as an orphaned comment at the end of `authz_policies.go`, duplicating the real doc on the method in `authz_roles.go`.

**Fix:** Removed the orphaned duplicate.

### 6. Orphaned `Broadcaster` Doc + Misplaced `splitSSELines` (Fixed)

**Files:** `sse_store.go`, `sse_broadcaster.go`, `sse_event.go`

**Problem:** (a) The `Broadcaster` type had an orphaned, **truncated** doc comment at the end of `sse_store.go` (cut off at `Data: renderTemplate(),`). The actual type in `sse_broadcaster.go` had no doc. (b) `splitSSELines()` was defined in `sse_broadcaster.go` but only used by `WriteSSEEvent()` in `sse_event.go` — misplaced.

**Fix:** (a) Moved a complete `Broadcaster` doc comment to the type declaration in `sse_broadcaster.go`. Removed the orphan from `sse_store.go`. (b) Moved `splitSSELines()` to `sse_event.go` next to its sole caller.

### 7. Incomplete `TestUserRegisteredEvent_JSON` Test (Fixed)

**File:** `usermgmt/events_test.go`

**Problem:** The test was named `_JSON` but never marshaled to JSON — it only checked `evt.Email != "a@b.com"`. Fields `DisplayName`, `Roles`, `OccurredAt` were set but never read (gopls `unusedwrite` warnings). The test gave false confidence about JSON serialization correctness.

**Fix:** Now actually marshals to JSON and asserts the full output string, covering all fields including `omitempty` behavior.

---

## Architecture Assessment

### Strengths

| Area                  | Assessment                                                                                                                                                                                     |
| --------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Type safety**       | Excellent. `authMode` enum (not bool), branded ID types (`UserID`, `CorrelationID`, `RequestID`), context-key sentinels (empty-struct types). Impossible states are genuinely unrepresentable. |
| **Composition**       | Clean. `Enforcer` interface enables mock/fake enforcers. `TemplComponent` duck-types templ without importing it. `HandlerOption` functional options compose naturally.                         |
| **Error handling**    | Centralized and consistent. go-error-family classification → HTTP status mapping. HTMX-aware auth redirects. Per-handler `OnError` callbacks.                                                  |
| **Module boundaries** | Clean. Root ↔ usermgmt have zero mutual imports. Cross-module bridging tested in `integration_test/` only.                                                                                     |
| **Security**          | Strong. CSRF (nosurf with BREACH mitigation), redirect sanitization (path traversal prevention), body size limits, account lockout, `text/plain` error responses (XSS prevention).             |
| **Test coverage**     | 96%+ root, 90%+ usermgmt, 570+ tests, BDD (Ginkgo) + unit + benchmark + fuzz + examples.                                                                                                       |
| **Documentation**     | ADRs for key decisions, FEATURES.md with honest status, extensive godoc comments on all exported APIs.                                                                                         |

### Observations (Not Actioned)

These are non-blocking observations from the review. No changes were made to avoid scope creep.

1. **`DOMAIN_LANGUAGE.md` is an unfilled template.** The file has placeholder text ("Example Term", "A placeholder definition") despite this project having a rich domain vocabulary (Command, Query, Enforcer, Dispatch, Event, etc.). Filling it in would improve onboarding. Low priority — the godoc and ADRs already document these concepts well.

2. **`ParseWSMessageInto` redundant map copy.** In `ws.go:168-169`, after deleting `HEADERS` from `raw`, the code copies `raw` into a new `bodyMap` via `maps.Copy` then marshals `bodyMap`. The copy is redundant — `raw` already has HEADERS removed. Minor allocation waste, not a correctness issue.

3. **`isTrustedProxy` logs `path=""`.** In `csrf_middleware.go:114`, the warning log includes `slog.String("path", "")` — an empty string literal. The request path isn't available at that point in the function signature, but including an empty path field adds noise to logs.

4. **Root package is intentionally flat (19 files).** This is documented and justified in `docs/modularization/PROPOSAL.md`. The errors↔response↔csrf cycle prevents splitting. The flat structure is the right call for a consumer-facing library package.

5. **`Config.Timeout` applies to dispatch only.** This is intentional and documented — decode/auth are not wrapped. Good separation of concerns.

6. **LSP shows stale warnings.** CLI reports 0 lint issues; LSP sometimes shows ~31 stale warnings. Known cache issue, documented in AGENTS.md.

---

## Quality Gates

| Gate              | Root         | usermgmt | integration_test | datastar-demo |
| ----------------- | ------------ | -------- | ---------------- | ------------- |
| **Build**         | ✅           | ✅       | ✅               | ✅            |
| **Tests (-race)** | ✅ 498 specs | ✅       | ✅               | N/A (main)    |
| **Lint**          | 0 issues     | 0 issues | 0 issues         | N/A           |
| **Vet**           | ✅           | ✅       | ✅               | ✅            |
| **Race detector** | 10/10 clean  | ✅       | ✅               | N/A           |

---

## Files Changed

| File                         | Change                                                              |
| ---------------------------- | ------------------------------------------------------------------- |
| `ratelimit_middleware.go`    | Fixed data race: moved `entry.lastUsed` read inside RLock           |
| `csrf_middleware.go`         | Added complete `CSRFMiddleware` doc comment                         |
| `csrf_context.go`            | Removed orphaned truncated `CSRFMiddleware` doc                     |
| `ratelimit_config.go`        | Fixed `RateLimiter` doc; removed orphaned `limiterEntry` comment    |
| `usermgmt/authz_types.go`    | Removed orphaned `Apply` doc comment                                |
| `usermgmt/authz_policies.go` | Consolidated `Apply` doc; removed orphaned `RolesForUser` duplicate |
| `sse_broadcaster.go`         | Added `Broadcaster` doc; removed misplaced `splitSSELines`          |
| `sse_store.go`               | Removed orphaned truncated `Broadcaster` doc                        |
| `sse_event.go`               | Added `splitSSELines` (moved from sse_broadcaster.go)               |
| `usermgmt/events_test.go`    | Fixed `TestUserRegisteredEvent_JSON` to actually test JSON          |

---

## Summary

The most important finding was the **data race in the rate limiter** — a real production bug that could silently corrupt rate-limit state under concurrent load. All other fixes were documentation quality improvements (split-brain doc comments across files) and one incomplete test. The codebase is in excellent shape and ready for production use.

---

## Round 2: Deeper Analysis (Post-Review Self-Critique)

After the initial review, a self-critique revealed several items missed:

### Additional Fixes Applied

| #  | File                                  | Change                                                                                                                                                                                         |
| -- | ------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 8  | `options_htmx.go` → `options_json.go` | `RenderJSON` doc was orphaned at bottom of wrong file. Moved to function declaration.                                                                                                          |
| 9  | `sse_event.go` → `sse_stream.go`      | `SSEStream` doc was orphaned at bottom of `sse_event.go`. Moved to struct declaration.                                                                                                         |
| 10 | `sse_stream.go` → `sse_store.go`      | `SSEEventStore` doc was orphaned at bottom of `sse_stream.go`. Moved to interface declaration.                                                                                                 |
| 11 | `sse_event.go`                        | Added missing `SSEEvent` type-level doc comment.                                                                                                                                               |
| 12 | `ws.go`                               | Removed redundant `maps.Copy` + intermediate `bodyMap` allocation in `ParseWSMessageInto`. After `delete(raw, "HEADERS")`, marshaling `raw` directly is sufficient.                            |
| 13 | `sse_stream.go`                       | Changed `SSEStream.ctx` field and `Context()` return type from `interface{ Done() <-chan struct{} }` to `context.Context`. Consumers expect `context.Context` from a method named `Context()`. |
| 14 | `options_render.go`                   | Added missing `Render` function doc comment.                                                                                                                                                   |
| 15 | `notify.go`                           | Added missing `NotificationLevel.String()` doc comment.                                                                                                                                        |
| 16 | `usermgmt/service_login.go`           | Added missing `LoginRequest` type doc.                                                                                                                                                         |
| 17 | `usermgmt/service_register.go`        | Added missing `RegisterRequest` type doc.                                                                                                                                                      |
| 18 | `usermgmt/service_misc.go`            | Added missing `GetUser` method doc.                                                                                                                                                            |

### What I Should Have Caught Earlier

1. **More orphaned doc comments** — I found 6 in round 1 but there were 3+ more (RenderJSON, SSEStream, SSEEventStore docs all orphaned in wrong files).
2. **The ParseWSMessageInto redundancy** — I noted it as "minor" in the review report but didn't fix it. It was a 3-line fix.
3. **SSEStream.ctx anonymous interface** — Surprising API that I should have flagged as an architecture issue.
4. **Missing exported doc comments** — Should have checked with a tool from the start.

### Items Not Actioned (Existing, Low Priority)

- `flake.nix` missing `meta` attribute block (pre-existing, causes BuildFlow pre-commit warning)
- `go.work` committed for a library (pre-existing BuildFlow warning)
- `csrf_middleware_test.go` at 370 lines (5.7% over 350-line limit)
- `DOMAIN_LANGUAGE.md` still an unfilled template
