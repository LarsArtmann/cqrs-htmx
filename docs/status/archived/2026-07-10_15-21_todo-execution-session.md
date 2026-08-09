# Status Report — TODO List Execution Session

**Date:** 2026-07-10 15:21
**Session scope:** Execute actionable items from TODO_LIST.md (P0–P2)
**Result:** 9 of ~40 open items completed. All tests pass (-race). 0 lint issues (root). 1 pre-existing lint issue (usermgmt).

---

## a) FULLY DONE (Verified)

| # | Item                                                                                | Files                                             | Tests Added                                            |
| - | ----------------------------------------------------------------------------------- | ------------------------------------------------- | ------------------------------------------------------ |
| 1 | **Depguard lint rule** (P0) — rejects `encoding/json/v2` + `encoding/json/jsontext` | `.golangci.yml`                                   | Verified catches violation                             |
| 2 | **CBOR round-trip tests** (P1) — exercises `codec.ForEncoding` path                 | `usermgmt/es_state_test.go`                       | 4 tests (CBOR, JSON baseline, CBOR fold, mixed stream) |
| 3 | **Decode\*WithRequest tests** (P1) — form, JSON query, form query                   | `feedback_features_test.go`                       | 3 Ginkgo specs                                         |
| 4 | **CSRFTestToken fix** (P1) — returns `(token, cookie)`                              | `csrf_testing.go`, `feedback_features_test.go`    | 2 specs (extraction + GET→POST round-trip)             |
| 5 | **StructuredError Code field** (P1) — fixes split brain with JSONErrorHandler       | `structured_error.go`, `structured_error_test.go` | 2 specs (field + JSON output)                          |
| 6 | **Exhaustive lint fix** (P1) — explicit Transient/Corruption/Infrastructure cases   | `usermgmt/service_register.go`                    | Existing tests pass                                    |
| 7 | **Empty-body godoc** (P1) — documents zero-value T behavior                         | `options_decode.go`                               | —                                                      |
| 8 | **writeDispatchError helper** (P2) — consolidates 15 call sites                     | `usermgmt/http.go` + 4 handler files              | Existing tests pass                                    |
| 9 | **ErrorContext in RequestLoggingSlog** (P2) — error_code/family/context in logs     | `logging.go`, `handler.go`, `logging_test.go`     | 1 spec (full dispatch error logging)                   |

**Verification gates passed:**

- Root tests: `go test ./... -race` — PASS
- usermgmt tests: `go test ./... -race` — PASS
- integration_test: `go test ./... -race` — PASS
- Root lint: `golangci-lint run` — 0 issues
- usermgmt lint: 1 pre-existing issue (`maintidx` on `es_readmodel.go:Handle` — separate TODO)
- errorfamily: 0 violations across all modules
- `nix flake check`: PASS

---

## b) PARTIALLY DONE

### writeDispatchError — Missing request ID in response body

The helper signature is `writeDispatchError(w, _ *http.Request, err error)` but the `*http.Request` parameter is unused (`_`). The TODO said "context-preserving function" — implying the request should be used to enrich the response (e.g., include `request_id`). The parameter exists for future use but does nothing today. This is a YAGNI smell — either use it or don't include it.

**What's missing:** Extract request ID from context and include it in the JSON error body, matching what `jsonBodyWriter` does in root.

### Depguard rule — Only in root `.golangci.yml`

The depguard rule rejecting `encoding/json/v2` was added to root `.golangci.yml` only. The buildflow trap that broke the build on 2026-07-09 hit **ALL** modules. `usermgmt/.golangci.yml` does NOT have the depguard rule. Neither do adminui, totp, webauthn, or oauth2 (they inherit root's config via lint exclusions, but they have separate go.mod files and could be linted independently).

**What's missing:** Add the same depguard rule to `usermgmt/.golangci.yml` at minimum.

---

## c) NOT STARTED (From TODO_LIST.md — items not addressed this session)

### P0 — Release & Infrastructure

1. Create GitHub Releases for 6 tags (v4.2.1, usermgmt/v4.2.0, etc.)
2. Verify `go get @v4.2.1` resolves from Go proxy
3. Check if pkg.go.dev picked up v4.2.1
4. Configure go-auto-upgrade to skip encoding/json → encoding/json/v2 migration

### P1 — Code Quality

5. Refactor `es_readmodel.go:Handle` (160-line switch, maintidx complexity 38)
6. Write pre-release verification script (`nix run .#release-checklist`)
7. Add release process documentation to CONTRIBUTING.md
8. Write `nix run .#check-docs-freshness` app
9. Research go-cqrs-lite v3.6.0 + v3.7.0 release notes
10. Research httputil v0.5.0 changes
11. Research templ-components v0.9.0→v0.10.0 changes

### P2 — Architecture

12. God-package split: domain layer extraction (20 pure files → `usermgmt/domain/`)
13. Root module: extract SSE/WS/ratelimit into optional sub-packages
14. Consider shared types module (`usermgmt/types/`)
15. Raise strategy module dep budgets
16. `OnSubscribe`/`OnUnsubscribe` hooks on `fanOut`/`Broadcaster`
17. `broadcaster.ServeSSE()` high-level helper
18. Fix 2 remaining high-severity context losses in `service_oauth2.go`

### P2 — Testing

19. OAuth2 FinishLogin integration test
20. usermgmt HTTP handler coverage (oauth2_http.go, credential_http.go edge cases, Postgres setup)
21. adminui coverage improvement (66.8% → 70%+)

### P3 — Technical Debt

22-40. OPFS persistence, snapshot integration, TypedRepository, Redis adapters, MySQL support, property-based tests, load testing, OpenAPI spec, codemod, json/v2 evaluation, provider guide, admin UI views, configurable TTLs, benchmark dedup.Ring, integration test with published version, contract tests, import grouping, CI release automation

---

## d) TOTALLY FUCKED UP

Nothing is catastrophically broken. All tests pass. But there are real issues:

### CSRFTestToken is a BREAKING API CHANGE — not flagged

`CSRFTestToken` changed from `func(middleware) string` to `func(middleware) (string, *http.Cookie)`. This is a **source-incompatible** change for any consumer who was using the return value as a bare string expression. This should have been:

1. Flagged as breaking in the CHANGELOG
2. Either kept backward-compatible with a new `CSRFTestTokenWithCookie` function, OR
3. Explicitly called out as a v4.x breaking change

**Impact:** Any consumer test code doing `token := cqrshtmx.CSRFTestToken(mw)` will fail to compile. This is the correct behavior (the old API was broken), but the change needs documentation.

### FEATURES.md not updated

The CSRF row still says `CSRFTestToken(mw)` — doesn't mention the cookie return.

### CHANGELOG not updated

None of the 9 completed items were added to any module's CHANGELOG.md.

---

## e) WHAT WE SHOULD IMPROVE

### Process Improvements

1. **Always update CHANGELOG when making API changes** — especially breaking ones like CSRFTestToken
2. **Apply lint rule changes to ALL module configs** — depguard was only added to root, not usermgmt
3. **Run coverage-gate after changes** — didn't verify coverage didn't drop below thresholds
4. **Update FEATURES.md when feature behavior changes** — CSRFTestToken signature changed
5. **Consider backward compatibility for test helpers** — CSRFTestToken is exported and used by consumers

### Code Quality Improvements

6. **writeDispatchError should use the request** — the `_ *http.Request` parameter is a YAGNI smell
7. **appendDispatchErrorAttrs traverses the error chain twice** — once for `errorCode`, once for context. Could be optimized to single traversal.
8. **captureDispatchError is O(n) Unwrap traversal per error** — fine for error paths, but worth noting
9. **StatusRecorder now carries business logic (error capture)** — this muddies its single responsibility (status tracking). Consider a separate `ErrorRecorder` wrapper.
10. **The depguard rule text references a specific date** — "Broke the build on 2026-07-09" — this will become stale context over time

### Test Quality Improvements

11. **No test for ProblemDetailsErrorHandler emitting the Code field** — tested via `NewStructuredError` but not the actual HTTP handler path
12. **No test for writeDispatchError including the code field** — tested implicitly via existing handler tests, but no unit test verifies the `code` key appears in the response body
13. **The RequestLoggingSlog error context test only covers Rejection family** — should also test Transient, Conflict, and errors without context

---

## f) Top 50 Things to Get Done Next

#### P0 — Critical

1. Add depguard rule to `usermgmt/.golangci.yml`
2. Update CHANGELOG.md for root module (API changes: CSRFTestToken, StructuredError.Code, writeDispatchError, RequestLoggingSlog error context)
3. Update FEATURES.md (CSRFTestToken signature)
4. Create GitHub Releases for 6 tags
5. Verify `go get @v4.2.1` resolves from proxy
6. Configure go-auto-upgrade to skip json/v2 migration
7. Check pkg.go.dev for v4.2.1

#### P1 — High Impact

8. Refactor `es_readmodel.go:Handle` (extract per-event handler map)
9. Wire request ID into `writeDispatchError` response body
10. Add test for ProblemDetailsErrorHandler emitting Code field
11. Add unit test for writeDispatchError code field inclusion
12. Add RequestLoggingSlog tests for Transient/Conflict families
13. Write pre-release verification script
14. Add release process docs to CONTRIBUTING.md
15. Write docs-freshness check nix app
16. Research go-cqrs-lite v3.6.0/v3.7.0 release notes
17. Research httputil v0.5.0 changes
18. Research templ-components v0.10.0 breaking changes
19. Fix 2 context losses in `service_oauth2.go`
20. OAuth2 FinishLogin integration test

#### P2 — Architecture & Features

21. God-package split: domain layer extraction (`usermgmt/domain/`)
22. Root module: extract SSE/WS/ratelimit sub-packages
23. Shared types module (`usermgmt/types/`)
24. Raise strategy module dep budgets
25. OnSubscribe/OnUnsubscribe hooks on fanOut/Broadcaster
26. broadcaster.ServeSSE() high-level helper
27. usermgmt HTTP handler coverage (oauth2_http.go edge cases)
28. adminui coverage improvement (66.8% → 70%+)
29. Postgres setup test coverage
30. Consider ErrorRecorder as separate wrapper (decouple from StatusRecorder)
31. Optimize appendDispatchErrorAttrs to single error-chain traversal
32. Add depguard to adminui/.golangci.yml (if exists)
33. Contract tests between root and usermgmt (RateLimiter boundary)
34. Integration test importing published version (not local replace)
35. Standardize import grouping across codebase

#### P3 — Future

36. Persistent offline queue (OPFS per ADR-0030)
37. Snapshot integration for high-event-volume aggregates
38. TypedRepository adoption across deciders
39. Redis adapters for SessionStore/OAuth2StateStore/IdempotencyStore
40. MySQL event store support
41. Property-based tests for fold functions
42. Load testing benchmarks for SSE broadcaster
43. OpenAPI spec generation
44. Consumer-facing v3→v4 codemod
45. Evaluate encoding/json/v2 adoption (v4.1+ when stable)
46. Provider implementation guide doc
47. Admin UI: TOTP management views
48. Admin UI: OAuth2 link/unlink views
49. Configurable TTLs (lockout, OAuth2 state, verification token)
50. Benchmark dedup.Ring vs old map
51. Automate GitHub Release creation via CI on tag push
52. Consider backward-compatible CSRFTestToken variant

---

## g) Top 2 Questions

### Q1: Should CSRFTestToken's breaking change trigger a patch release (v4.2.2) or wait for v4.3.0?

The signature change from `func(middleware) string` to `func(middleware) (string, *http.Cookie)` is source-incompatible. It's a test helper, so the blast radius is consumer test code only — but it WILL break compilation. Do we want to:

- **(a)** Release v4.2.2 immediately with just this fix + CHANGELOG entry, or
- **(b)** Keep it uncommitted until v4.3.0 batches more changes?

### Q2: Should the depguard lint rule block ALL experimental stdlib packages or just json/v2?

The current rule specifically denies `encoding/json/v2` and `encoding/json/jsontext`. But Go 1.26 has other experimental packages behind `GOEXPERIMENT` flags (arenas, goroutineleakprofile, etc.). The `.golangci.yml` already has `goexperiment.*` build tags enabled. Should we broaden the depguard rule to deny all `GOEXPERIMENT`-gated stdlib packages, or is json/v2 the only one that poses a real risk of auto-migration breakage?
