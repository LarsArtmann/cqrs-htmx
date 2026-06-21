# go-error-family Adoption — Comprehensive Status Report

**Date:** 2026-06-21 04:31
**Session Goal:** `branching-flow errorfamily .` → 0 violations
**Starting Count:** 175 violations across 38 files
**Final Count:** **0 violations** ✅
**Tests:** All modules pass with `-race` (root, usermgmt, integration_test, catalog)
**Lint:** 0 issues (golangci-lint, root + usermgmt)

---

## a) FULLY DONE ✅

### Migration completed across all modules

| Module | Violations Before | Violations After | Status |
|--------|------------------|------------------|--------|
| Root (cqrs-htmx) | 33 | 0 | ✅ Done |
| usermgmt | 132 | 0 | ✅ Done |
| catalog | 1 | 0 | ✅ Done |
| examples/datastar-demo | 8 | 0 | ✅ Done |
| examples/basic | 0 | 0 | ✅ Clean (no violations) |
| integration_test | 0 | 0 | ✅ Clean (no violations) |
| **TOTAL** | **175** | **0** | ✅ |

### What was converted

- **175 stdlib error constructors** (`errors.New`, `fmt.Errorf`) → go-error-family constructors (`event.NewRejection`, `event.WrapTransient`, etc.)
- All constructors re-exported via `go-cqrs-lite/event/v2` (`event.New*`, `event.Wrap*`, `event.Wrapf`, `event.Newf`, `event.Classify`)
- **`ErrDispatchFailed`** changed from `errors.New("...")` (plain sentinel requiring runtime `RegisterClassification`) to `event.NewTransient(...)` (natively classified). Eliminated `sync.Once` + `errorfamily.RegisterClassification` hack in `errors.go`.
- **Dispatch error wrapping**: uses `event.Wrapf(err, event.Classify(err), code, msg)` — preserves the inner domain error's family (Rejection/Conflict/Transient) rather than forcing Transient.
- **`go.mod` tidy**: go-error-family correctly `// indirect` in root + usermgmt (transitive via `event/v2`), direct in `catalog` (imports `go-error-family` directly since catalog has no `event/v2` dep).

### Family assignment rules applied

| Family | HTTP Status | Used for | Count |
|--------|------------|----------|-------|
| Rejection | 400 | parse/validation, bad config, invalid IDs, decode failures | ~45 |
| Conflict | 409 | duplicates (user/email/credential/external-account) | (already classified) |
| Transient | 503 | DB I/O, OAuth2 provider calls, SSE/WS stream writes | ~30 |
| Corruption | 422 | projection payload unmarshal, upcaster failures | ~15 |
| Infrastructure | 500 | marshal failures, event construction, nil deps, cmd registration | ~60 |

### Files changed (38 source files)

**Root module (11 files):**
- `errors.go` — sentinel migration + removed sync.Once machinery
- `decoder.go` — 9 decode/read/form wraps
- `handler.go` — 4 dispatch/method/decode wraps
- `context.go`, `httputil.go`, `options_validate.go` — 3 wraps
- `sse_event.go`, `sse_store.go` — 4 SSE wraps
- `ws.go`, `ws_dispatch.go`, `ws_encoder.go` — 13 WS wraps

**usermgmt module (22 files):**
- `es_decide.go` (25), `es_dispatch.go` (12), `es_readmodel.go` (10)
- `es_casbin_projection.go` (7), `es_projection_setup.go` (5)
- `es_events.go` (1), `es_state.go` (1), `es_upcaster.go` (1)
- `sql_session_store.go` (11), `sql_event_store.go` (3)
- `oauth2.go` (14), `totp.go` (6), `webauthn_service.go` (6)
- `service_misc.go` (7), `service_register.go` (1), `service_oauth2.go` (6)
- `email.go` (3), `email_verification.go` (3), `import_export.go` (6)
- `http.go` (1), `user.go` (1), `random.go` (1), `verification_totp_http.go` (1)

**Other modules:**
- `catalog/serve.go` (1)
- `examples/datastar-demo/domain_commands.go` (6), `handlers_helpers.go` (2)

### Documentation
- `AGENTS.md` updated with:
  - ErrorFamily command in Quick Reference table
  - Comprehensive family assignment rules (5 families → HTTP status)
  - Dispatch wrapping convention (`Wrapf(err, Classify(err), ...)`)
  - `ErrValidation` identity preservation note
  - Updated gotcha #15 (banned stdlib constructors)
  - Removed outdated `sync.Once` reference

---

## b) PARTIALLY DONE ⚠️

### 1. Double-wrapping of classified sentinels (12 instances)

When wrapping an already-classified `*event.Error` sentinel (like `ErrUserNotFound`) with `event.WrapRejection(ErrUserNotFound, newCode, msg)`, the resulting `.Error()` string contains nested brackets:

```
Before: fmt.Errorf("enable totp: %w", ErrUserNotFound)
  → "enable totp: [Rejection:usermgmt.user_not_found] user not found"

After:  event.WrapRejection(ErrUserNotFound, "usermgmt.totp.user_not_found", "enable totp")
  → "[Rejection:usermgmt.totp.user_not_found] enable totp: [Rejection:usermgmt.user_not_found] user not found"
```

**Impact**: HTTP response bodies (which use `err.Error()`) now have double-bracket notation. Not breaking (classification + `errors.Is` still work correctly via Unwrap chain), but degrades response readability.

**Affected files**: `webauthn_service.go` (4), `totp.go` (2), `service_oauth2.go` (1), `email.go` (3), `handler.go` (1), `service_misc.go` (1)

**Correctness verified**: `errors.Is` walks `Unwrap()` → finds `ErrUserNotFound` via pointer identity. All `handler_misc_test.go` errorStatus tests pass. ✅

### 2. Lint fixes applied but not yet committed

After the auto-commit (`58e4b9e`), I fixed `wrapcheck` violations by replacing `event.Compose` with `event.Wrapf(err, event.Classify(err), ...)` and added `//nolint:dupl` on mirrored decider functions. These 7 files have uncommitted changes.

### 3. `fmt.Sprintf` exceptions (2 instances)

`http.go:250` and `verification_totp_http.go:79` use `fmt.Sprintf` (not `fmt.Errorf`) to build message strings for `writeError(w, 400, msg)`. These don't create error objects — just format a response string. Functionally correct but stylistically inconsistent (some validation errors are `*event.Error` objects, these two are plain strings).

---

## c) NOT STARTED ❌

### 1. CI/flake.nix integration
`branching-flow errorfamily .` is NOT in `flake.nix` as a lint app. Enforcement is documented only — no automated gate.

### 2. Pre-commit hook integration
The pre-commit hook (`.git/hooks/pre-commit`) runs `buildflow` only. No `branching-flow errorfamily` check.

### 3. Error code constants
All error codes are inline magic strings (e.g., `"usermgmt.totp.user_not_found"`). No typed constants prevent typos.

### 4. HTTP response body cleanup
`DefaultErrorHandler` writes `err.Error()` to HTTP response, which includes `[Family:code]` bracket notation. Should use `*event.Error.Message` (clean message) or `*event.Error.JSON()` (RFC 7807 format) instead.

### 5. Error code naming convention test
No test validates that all codes follow `package.area.detail` format.

---

## d) TOTALLY FUCKED UP 💥

### Nothing is broken or fucked up.

All tests pass with `-race`. All lint clean. `branching-flow errorfamily .` reports 0. Classification + `errors.Is` chains verified correct. The double-bracket issue is cosmetic, not functional.

### Near-miss: Corruption family almost masked

Initially used `WrapInfrastructure` for projection decode errors in `es_readmodel.go`. Caught during self-review that `unmarshalPayload` already classifies its errors as **Corruption** (422), and wrapping with Infrastructure (500) would mask the inner family. Fixed to `WrapCorruption` before any test ran. `Classify` picks the outermost `*event.Error`, so the outer family must match the inner for decode errors.

---

## e) WHAT WE SHOULD IMPROVE

### Architecture & Type Safety

1. **Error code type safety**: Codes are untyped `string` values. A typo in `"usermgmt.totp.user_not_found"` → `"usermgmt.top.user_not_found"` produces no compile error. Consider typed constants or a code registry.

2. **Double-wrap elimination**: Return classified sentinels directly (`return ErrUserNotFound`) instead of `WrapRejection(ErrUserNotFound, newCode, msg)` when no cause error exists. The code in the sentinel is already specific enough.

3. **HTTP error response format**: `err.Error()` produces `[Family:code] message`. For user-facing HTTP responses, use `*event.Error.Message` (clean) or `StructuredError.JSON()` (RFC 7807). The codebase already has `StructuredError` and `JSONErrorHandler` — these should be the default.

4. **`errorStatus` vs `MapError` divergence**: usermgmt's `errorStatus()` (http.go:336) hardcodes sentinel→status mappings via `errors.Is`. Root's `MapError()` uses family classification. Both coexist with different mappings (e.g., `ErrUserNotFound` → 404 in usermgmt, → 400 via family in root). This split-brain should be unified.

### Process

5. **Automated enforcement**: `branching-flow errorfamily . --exit-code` should run in CI (flake.nix app) and pre-commit hook. Without automation, violations will creep back.

6. **Error code registry test**: A test that enumerates all `*event.Error` sentinels and validates their codes follow the `package.area.detail` naming convention.

### Library Awareness

7. **go-error-family `WithContext`**: Instead of wrapping sentinels with message prefixes (which adds brackets), use `ErrX.WithContext("operation", "enable_totp")` — adds structured context without nesting `.Error()` output.

8. **go-error-family `HandleError`**: The library provides a CLI boundary handler with structured What/Why/Fix/WayOut messages. Currently unused — could improve CLI/operational error reporting.

---

## f) Top 25 Things to Get Done Next

| # | Task | Impact | Effort | Priority |
|---|------|--------|--------|----------|
| 1 | Commit uncommitted Wrapf/nolint/AGENTS.md changes | HIGH | 2min | 🔴 NOW |
| 2 | Fix 12 double-wrapped sentinels → return directly | HIGH | 15min | 🔴 NOW |
| 3 | Add `branching-flow errorfamily --exit-code` to flake.nix as `.#errorfamily` app | HIGH | 10min | 🔴 NOW |
| 4 | Add errorfamily check to pre-commit hook | MEDIUM | 5min | 🔴 NOW |
| 5 | Add `branching-flow errorfamily .` to `nix run .#lint` | MEDIUM | 5min | 🟡 NEXT |
| 6 | Verify `examples/catalog-demo` is clean (not yet checked) | LOW | 2min | 🟡 NEXT |
| 7 | Extract error codes to typed constants in each package | MEDIUM | 30min | 🟡 NEXT |
| 8 | Change `DefaultErrorHandler` to use `Message` field for response body | HIGH | 20min | 🟡 NEXT |
| 9 | Add test: all error codes follow naming convention | LOW | 15min | 🟡 NEXT |
| 10 | Unify `errorStatus` (usermgmt) and `MapError` (root) → single family-based mapper | HIGH | 45min | 🟠 LATER |
| 11 | Add `errorfamily --format sarif` output for CI dashboards | LOW | 10min | 🟠 LATER |
| 12 | Replace `fmt.Sprintf` exceptions in http.go with `event.NewRejection` | LOW | 5min | 🟠 LATER |
| 13 | Add `WithContext` to sentinels for structured error context | MEDIUM | 20min | 🟠 LATER |
| 14 | Document error code taxonomy in `docs/DOMAIN_LANGUAGE.md` | LOW | 15min | 🟠 LATER |
| 15 | Audit `StructuredError` usage — make it the default error response format | MEDIUM | 30min | 🟠 LATER |
| 16 | Add errorfamily to `nix flake check` | LOW | 5min | 🟠 LATER |
| 17 | Consider `errorfamily.Registry.Clone()` for test isolation | LOW | 15min | ⚪ SOMEDAY |
| 18 | Add OpenTelemetry integration for classified errors (span attributes from family) | LOW | 30min | ⚪ SOMEDAY |
| 19 | Evaluate `samber/oops` bridge for application-layer enrichment | LOW | 20min | ⚪ SOMEDAY |
| 20 | Add retry middleware using `event.IsRetryable` (Transient family) | MEDIUM | 45min | ⚪ SOMEDAY |
| 21 | Add `errorfamily.ExitCode` to CLI examples in datastar-demo | LOW | 10min | ⚪ SOMEDAY |
| 22 | Add changelog entry for go-error-family v0.4.0 adoption | LOW | 10min | ⚪ SOMEDAY |
| 23 | Consider branded error code type (`type ErrorCode string`) | LOW | 30min | ⚪ SOMEDAY |
| 24 | Add migration guide for consumers updating to latest version | LOW | 20min | ⚪ SOMEDAY |
| 25 | Evaluate `RegisterStdlibDefaults` for third-party errors (sql, context, os) | LOW | 15min | ⚪ SOMEDAY |

---

## g) Top #1 Question I Cannot Figure Out Myself ❓

**Should the double-wrapped sentinels be unwrapped (return sentinel directly), or should the HTTP error handler be changed to use `*event.Error.Message` instead of `err.Error()`?**

Both approaches fix the response body issue, but they have different tradeoffs:

- **Option A (unwrap sentinels)**: Return `ErrUserNotFound` directly. Loses call-site context ("enable totp", "finish login"). Simpler, less code. But you lose per-call-site error codes (all map to `"usermgmt.user_not_found"`).

- **Option B (fix HTTP handler)**: Keep specific codes but change `DefaultErrorHandlerWithRedirect` to extract `.Message` from `*event.Error` (falling back to `.Error()` for non-classified errors). Fixes ALL bracket issues across the codebase, not just the 12 double-wraps. More work but higher payoff.

- **Option C (both)**: Unwrap where no extra context is needed, fix handler for everything else. Best result but most work.

**I recommend Option C** but want confirmation before executing, since changing `DefaultErrorHandler` affects the public API contract for all consumers.
