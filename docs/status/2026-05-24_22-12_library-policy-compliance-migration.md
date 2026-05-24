# Status Report: Library Policy Compliance Migration

**Date:** 2026-05-24 22:12
**Branch:** master
**Status:** PASSED — 0 violations (was 4)

---

## Executive Summary

Resolved all 4 library-policy violations by replacing `gorilla/csrf` with `justinas/nosurf` and `cockroachdb/errors` (plus transitive `pkg/errors`) with `go-error-family` via `go-cqrs-lite/core/event`.

**Net result:** -224 lines, 25 files changed, 0 violations, all 378+ tests pass.

---

## A) FULLY DONE

### 1. gorilla/csrf → justinas/nosurf (CVE migration)

**Why:** gorilla/csrf has unfixed CVE-2025-47909 (TrustedOrigins MITM) plus ongoing origin-validation issues (#185-205).

**Changes (root module):**

- `csrf.go` — Complete rewrite. Replaced `csrf.Protect()` middleware pattern with `nosurf.New()` handler pattern. Added `configureNosurfHandler()` to map `CSRFConfig` to nosurf's `SetBaseCookie/SetFailureHandler/SetIsTLSFunc/SetIsAllowedOriginFunc` API. Added `translateCSRFHeaders()` for custom header/field name support (nosurf hardcodes `X-CSRF-Token` / `csrf_token`). Added `Sec-Fetch-Site: same-origin` fallback for plain HTTP requests (replaces gorilla's `PlaintextHTTPRequest`). Removed `Secret` field from `CSRFConfig` (nosurf uses `crypto/rand` — no HMAC secret needed). Removed `sameSite()` converter (nosurf uses `http.SameSite` directly). Removed `secret()` padding logic. Removed `buildGorillaOptions()`.
- `csrf_handler.go` — Replaced gorilla per-handler pattern with direct `nosurf.New()` instantiation in `executeCSRFValidation()`. Removed `csrfProtect` field from `handlerConfig`.
- `csrf_test.go` — Removed `Secret` from all configs. Added `Sec-Fetch-Site: same-origin` header to POST requests. Removed `csrfConfigWithSecret` helper. Removed HMAC-signed token tests (nosurf uses random tokens). Updated `CSRFConfig.Validate` tests (empty config now valid — no secret required).
- `internal_test.go` — Replaced gorilla/csrf import with nosurf. Rewrote `TestBuildGorillaOptions*` tests as `TestConfigureNosurfHandler*` tests.
- `benchmark_test.go` — Updated `BenchmarkCSRFMiddleware` to use context-based token capture instead of raw cookie value.
- `fuzz_test.go` — Removed `Secret` field, `sameSite()`, `secret()` from fuzz targets.
- `coverage_test.go` — Removed `Secret` from `CSRFConfig` literal.
- `integration_test.go` — Removed `Secret` from `CSRFConfig`.

**API changes:**

- `CSRFConfig.Secret` removed (BREAKING — nosurf uses crypto/rand, no secret needed)
- `CSRFConfig.Validate()` no longer errors on empty secret
- `ErrCSRFInvalid` comment updated to reference nosurf

### 2. cockroachdb/errors → go-error-family (via go-cqrs-lite/core/event)

**Why:** library-policy bans `pkg/errors` (cockroachdb/errors is a transitive dep that brings it in). go-error-family provides behavioral classification (Rejection/Conflict/Transient/Corruption/Infrastructure) with BSD exit codes and retry decisions.

**Changes (root module):**

- `errors.go` — All sentinel errors now use `event.NewRejection()`, `event.NewInfrastructure()`, `event.NewTransient()` instead of `errors.New()`. `ErrDispatchFailed` uses `fmt.Errorf()` (plain sentinel) registered as Transient. Added `stderrors` import for `errors.Is`. Removed cockroachdb/errors import. Removed `errorfamily.RegisterClassification()` calls (sentinels carry their own family via `*event.Error` type).
- `authz.go` — All `errors.WithMessagef()` → `event.NewRejection(...).WithCause()`. All `errors.Wrapf()` → `event.NewTransient(...).WithCause()`. Removed cockroachdb/errors import.
- `app.go` — `errors.New()` → `event.NewInfrastructure()`. Removed cockroachdb/errors import.
- `options.go` — `errors.WithMessagef()` → `event.NewRejection(...).WithCause()`. Removed cockroachdb/errors import.
- `csrf.go` — `errors.WithMessage()` → `event.NewRejection(...).WithCause()`. Removed cockroachdb/errors import.

**Changes (usermgmt submodule):**

- `errors.go` — All 10 sentinels use `event.NewRejection()`. Removed cockroachdb/errors import.
- `authz.go` — All 19 `errors.Wrapf()` → `event.NewTransient("casbin_error", ...).WithCause()`. `errors.WithMessagef(ErrForbidden, ...)` → `event.NewRejection("forbidden", ...).WithCause(ErrForbidden)`. Removed cockroachdb/errors import.
- `service.go` — All `errors.Wrapf()` → `event.NewTransient("internal", ...).WithCause()`. All `errors.WithMessagef(ErrValidation, ...)` → `event.NewRejection("validation", ...).WithCause(ErrValidation)`. Removed cockroachdb/errors import.
- `store.go` — `errors.Wrapf()` → `event.NewTransient("session_create_failed", ...).WithCause()`. Removed cockroachdb/errors import.
- `user.go` — `errors.Wrapf()` → `event.NewTransient("token_gen_failed", ...).WithCause()`. Removed cockroachdb/errors import.
- `http.go` — Switched from cockroachdb/errors to stdlib `errors` (only uses `errors.Is`). Removed cockroachdb/errors import.

### 3. Dependency cleanup

**Root go.mod:**

- Removed: `gorilla/csrf`, `cockroachdb/errors`, `pkg/errors` (indirect), `gorilla/securecookie` (indirect), `cockroachdb/logtags` (indirect), `cockroachdb/redact` (indirect), `getsentry/sentry-go` (indirect)
- Added: `justinas/nosurf v1.2.0`

**Usermgmt go.mod:**

- Removed: `cockroachdb/errors`, all cockroachdb indirect deps, `getsentry/sentry-go`
- Added: `go-cqrs-lite/core v1.5.0` (brings go-error-family transitively)

**Integration_test go.mod:**

- Updated dependency on root module

### 4. Verification

- library-policy: **0 violations** (was 4)
- Root tests: **378 passed**, 0 failed, 96.7% coverage
- Usermgmt tests: **All passed**, 0 failed, 91.1% coverage
- Integration tests: **All passed**, 0 failed
- Race detector: Clean (all modules tested with `-race`)
- Build: All 4 modules compile cleanly

---

## B) PARTIALLY DONE

None — all violations resolved.

---

## C) NOT STARTED

- AGENTS.md update with new dependency info and API changes
- golangci-lint run (only LSP-checked; full lint not run post-migration)
- Benchmark comparison (gorilla/csrf vs nosurf performance)
- golines formatting on app.go (LSP warning, not verified via CLI)

---

## D) TOTALLY FUCKED UP

Nothing — all clean.

---

## E) WHAT WE SHOULD IMPROVE

1. **CSRFConfig.Secret removal is a BREAKING CHANGE** — consumers setting `Secret` will get compile errors. Should document migration path.
2. **CSRFConfig no longer validates empty secret** — nosurf generates random tokens per-instance, so no secret is needed. But consumers who relied on stable tokens across restarts need to know this.
3. **Token format changed** — gorilla/csrf used HMAC-SHA256 masked tokens; nosurf uses random 32-byte tokens with one-time-pad masking. Tokens are not interchangeable. Existing sessions will need new CSRF cookies.
4. **httputil.go had dead code removed** — 29 lines deleted, needs review.
5. **usermgmt authz.go uses generic "casbin_error" code** for all transient errors — could be more specific for better debugging.
6. **Coverage dropped slightly** — root: 97.3% → 96.7% (new nosurf integration paths), usermgmt: 91.1% (stable).

---

## F) Top 25 Things We Should Do Next

1. Update AGENTS.md with nosurf migration notes, removed Secret field, new error family pattern
2. Run full `golangci-lint run` on root + usermgmt
3. Fix golines formatting warning on app.go
4. Run `go mod tidy` on all 4 modules to verify cleanliness
5. Add test for nosurf token masking behavior (BREACH mitigation)
6. Add test for CSRFConfig.Validate() with empty secret (new behavior)
7. Add benchmark comparison gorilla/csrf vs nosurf (was in benchmark_test.go)
8. Update CSRF documentation in comments (remove gorilla references)
9. Test nosurf with HTTPS (TLS) requests
10. Test nosurf TrustedOrigins with actual cross-origin requests
11. Add migration guide in docs/ for consumers upgrading
12. Verify integration_test module works with `GOWORK=off` CI
13. Add test for translateCSRFHeaders with custom field name
14. Add test for Sec-Fetch-Site fallback in CSRFMiddleware
15. Verify datastar-demo still builds
16. Run `go vet` on all modules
17. Check if `httputil.go` dead code removal affects anything
18. Add CHANGELOG entry for breaking changes
19. Verify CSRF double-submit pattern still works end-to-end with HTMX
20. Test nosurf SameSite=None mode with actual browser
21. Verify nosurf token persistence across cookie regeneration
22. Add error family assertions in existing tests (verify classification)
23. Run `library-policy` in CI configuration
24. Consider adding nosurf exemption tests for API routes
25. Review if usermgmt service.go error codes should be more specific

---

## G) Top #1 Question I Cannot Figure Out Myself

**Should `CSRFConfig.Secret` be kept as a no-op field (deprecated, ignored) for backward compatibility, or is a clean break acceptable?**

nosurf doesn't use HMAC secrets — it uses `crypto/rand` for token generation. The `Secret` field existed purely for gorilla/csrf's signing mechanism. Removing it is cleaner but breaks consumers at compile time. Keeping it as deprecated is confusing. I opted for clean removal but this should be confirmed by the project owner.

---

## Test Results Summary

| Module         | Tests    | Status           | Coverage |
| -------------- | -------- | ---------------- | -------- |
| Root           | 378      | PASS             | 96.7%    |
| Usermgmt       | All      | PASS             | 91.1%    |
| Integration    | All      | PASS             | N/A      |
| Library Policy | 84 rules | **0 violations** | —        |

## Files Changed

```
 app.go                  |   4 +-
 authz.go                |  30 ++++-----
 benchmark_test.go       |  21 ++++--
 coverage_test.go        |   1 -
 csrf.go                 | 171 ++++++++++++++++++++++++------------------------
 csrf_handler.go         |  36 ++++++----
 csrf_test.go            |  63 +++++-------------
 errors.go               |  66 +++++++++----------
 fuzz_test.go            |  12 ++--
 go.mod                  |  14 +---
 go.sum                  |  60 ++---------------
 httputil.go             |  29 ++------
 integration_test.go     |   1 -
 integration_test/go.mod |  17 +----
 integration_test/go.sum |  74 ++-------------------
 internal_test.go        |  93 +++++++++++++++-----------
 options.go              |   6 +-
 usermgmt/authz.go       |  52 +++++++--------
 usermgmt/errors.go      |  22 +++----
 usermgmt/go.mod         |  16 +----
 usermgmt/go.sum         |  81 ++++-------------------
 usermgmt/http.go        |   3 +-
 usermgmt/service.go     |  41 ++++++------
 usermgmt/store.go       |   6 +-
 usermgmt/user.go        |   5 +-
 25 files changed, 350 insertions(+), 574 deletions(-)
```
