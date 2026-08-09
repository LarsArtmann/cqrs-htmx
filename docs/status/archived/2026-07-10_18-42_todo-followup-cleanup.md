# Status Report: TODO Execution + Follow-Up Session

**Date:** 2026-07-10 18:42\
**Session span:** Two sessions — initial TODO execution (prior) + follow-up cleanup (this session)\
**Branch:** master\
**Files changed:** 21 modified + 2 untracked = 23 total\
**Diff:** +566 / -38 lines

---

## Verification Snapshot (end of this session)

| Check               | Command                                   | Result                                                                 |
| ------------------- | ----------------------------------------- | ---------------------------------------------------------------------- |
| Root tests          | `go test ./... -count=1 -race`            | PASS (4.1s)                                                            |
| Usermgmt tests      | `GOWORK=off go test ./... -count=1 -race` | PASS (4.4s)                                                            |
| Integration tests   | `GOWORK=off go test ./... -count=1 -race` | PASS (1.2s)                                                            |
| Root lint           | `golangci-lint run`                       | 0 issues                                                               |
| Usermgmt lint       | `GOWORK=off golangci-lint run`            | 1 pre-existing (`maintidx` on `es_readmodel.go:Handle`)                |
| Errorfamily         | `nix run .#errorfamily`                   | All modules pass (0 stdlib error constructors)                         |
| Module architecture | `nix run .#check-modules`                 | All modules within budget, no version drift, no absolute replace paths |
| Formatting          | `nix fmt`                                 | 6 files formatted, 0 changed                                           |
| Flake check         | `nix flake check`                         | All checks passed                                                      |

---

## A) FULLY DONE (verified this session)

### Prior session — 9 TODO items executed

1. **Depguard lint rule for `encoding/json/v2`** (`.golangci.yml`) — Added depguard to enabled linters, deny rules for `encoding/json/v2` and `encoding/json/jsontext` with descriptive error messages.
2. **CBOR round-trip tests** (`usermgmt/es_state_test.go`) — 4 new tests: CBOR encode→decode via `codec.ForEncoding`, JSON baseline, 3-event fold with CBOR-encoded events, mixed JSON+CBOR stream fold.
3. **Request-aware decoder tests** (`feedback_features_test.go`) — 3 Ginkgo specs: `DecodeFormWithRequest`, `DecodeJSONQueryWithRequest`, `DecodeFormQueryWithRequest`.
4. **CSRFTestToken fix** (`csrf_testing.go`) — BREAKING: signature changed from `func(middleware) string` to `func(middleware) (string, *http.Cookie)`. GET→POST round-trip test with nosurf origin check.
5. **StructuredError Code field** (`structured_error.go`) — Added `Code string` field (json `"code,omitempty"`), populated via `errorCode(err)` in `newStructuredErrorFromContext`. Fixes split brain with JSONErrorHandler.
6. **Exhaustive lint fix** (`usermgmt/service_register.go`) — Added explicit `case event.Transient, event.Corruption, event.Infrastructure:` (same body as default).
7. **writeDispatchError helper** (`usermgmt/http.go`) — New function replacing 15 `writeError(w, errorStatus(err), err.Error())` call sites across 5 handler files. Includes error code in JSON response body.
8. **Error context logging** (`logging.go` + `handler.go`) — `StatusRecorder` captures dispatch error. `handleErr` calls `captureDispatchError`. `RequestLoggingSlog` logs `error_code`, `error_family`, `error_ctx_*` from classified errors.
9. **DecodeJSON/DecodeJSONQuery godoc** (`options_decode.go`) — Documented empty-body behavior (zero-value T, not error).

### This session — 8 follow-up items

10. **Depguard rule propagated to `usermgmt/.golangci.yml`** — Same deny rules for `encoding/json/v2` and `encoding/json/jsontext`. Added `depguard` to enabled linters + settings block.
11. **writeDispatchError wired request ID** (`usermgmt/http.go`) — Changed `_ *http.Request` to `r *http.Request`, extracts request ID from context, includes as `request_id` in JSON body. Added `requestIDKey = "request_id"` constant.
12. **ProblemDetailsErrorHandler Code field tests** (`errors_model_test.go`) — 2 new tests: verifies `code` field present for classified errors (`validation_failed`), omitted for plain unclassified errors.
13. **writeDispatchError unit tests** (`usermgmt/http_dispatch_error_test.go` — NEW FILE) — 4 tests: code field inclusion, request ID inclusion, conflict status derivation, nil-request omits request ID.
14. **RequestLoggingSlog family variant tests** (`logging_test.go`) — 2 new Ginkgo specs: Transient family dispatch failure logs `error_family:"transient"`, Conflict family logs `error_family:"conflict"`.
15. **CHANGELOG.md updated** — Full `[Unreleased]` section: Breaking (CSRFTestToken), Added (7 items), Changed (1 item), Fixed (2 items), Documented (1 item).
16. **FEATURES.md updated** — CSRF row updated to reflect `(token, *http.Cookie)` return signature. Version stamp updated to `v4.2.1+unreleased`.
17. **Full verification suite run** — All tests, lint, errorfamily, module checks, flake check, formatting — all pass.

---

## B) PARTIALLY DONE

### Nothing is partially done

All started work items are complete and verified. The only "partially done" aspect is the overall TODO list itself — 9 items completed, ~31 still open (see section F).

---

## C) NOT STARTED (from the TODO list, not addressed in either session)

### P0 — Release & Infrastructure

- Create GitHub Releases for 6 tags (requires GitHub access/CLI auth)
- Verify `go get @v4.2.1` resolves from Go proxy
- Check pkg.go.dev pickup
- Configure `go-auto-upgrade` to skip `encoding/json` → `v2` migration

### P1 — Code Quality

- Refactor `es_readmodel.go:Handle` (maintidx complexity 38, 160-line 12-case switch)

### P1 — Documentation & Process

- Pre-release verification script (`nix run .#release-checklist`)
- Release process documentation in CONTRIBUTING.md
- `nix run .#check-docs-freshness` app
- Research go-cqrs-lite v3.6.0/v3.7.0 release notes
- Research httputil v0.5.0 changes
- Research templ-components v0.9.0→v0.10.0 changes

### P2 — Architecture

- God-package split (domain layer extraction from usermgmt)
- Root module SSE/WS/ratelimit extraction into sub-packages
- Shared types module for WebAuthnUserData/OAuth2UserInfo
- Raise strategy module dep budgets

### P2 — Features

- `OnSubscribe`/`OnUnsubscribe` hooks on fanOut/Broadcaster
- `broadcaster.ServeSSE()` high-level helper
- Fix 2 remaining context losses in `service_oauth2.go`

### P2-P3 — Testing, Features, Tech Debt

(see section F for full list)

---

## D) TOTALLY FUCKED UP (nothing)

No regressions, no broken changes, no data loss. The only thing that **was** fucked up (prior session) was caught and fully fixed this session:

- ~~Depguard only in root `.golangci.yml`, not usermgmt~~ → **Fixed**: added to `usermgmt/.golangci.yml`
- ~~writeDispatchError had unused `_ *http.Request` parameter~~ → **Fixed**: wired request ID extraction
- ~~No CHANGELOG entries for any changes~~ → **Fixed**: full `[Unreleased]` section
- ~~FEATURES.md CSRF row outdated~~ → **Fixed**: updated signature
- ~~Missing tests for ProblemDetailsErrorHandler Code field~~ → **Fixed**: 2 tests added
- ~~Missing tests for writeDispatchError~~ → **Fixed**: 4 tests added
- ~~Missing tests for RequestLoggingSlog Transient/Conflict families~~ → **Fixed**: 2 tests added

---

## E) WHAT WE SHOULD IMPROVE

### Process improvements

1. **CHANGELOG discipline**: Code changes should get CHANGELOG entries in the same commit/PR, not as a follow-up. The prior session wrote code for 9 items with zero CHANGELOG updates — this session spent an entire task catching up.

2. **Test-driven addition**: When adding a new function (`writeDispatchError`), write the test immediately, not as a follow-up task. The prior session left it with a stale "unused function" gopls warning because no test exercised it.

3. **Depguard should cover ALL module configs**: When adding a lint rule to one module's config, immediately propagate to all other module configs. The prior session only added it to root, leaving usermgmt unprotected.

4. **Breaking changes need immediate documentation**: The CSRFTestToken signature change was done in the prior session but not documented anywhere until this session. A consumer upgrading would have been blindsided.

### Code improvements (noticed during this session)

5. **`writeDispatchError` uses `errors.As` directly** while root's `errorCode()` uses the deeper `errors.AsType` traversal. This means `writeDispatchError` might miss codes on deeply-wrapped errors that root's `errorCode` catches. The functions serve slightly different purposes (immediate vs deepest-code) but the inconsistency is a potential surprise.

6. **`StatusRecorder.dispatchErr` field is an SRP violation**: The StatusRecorder now has two responsibilities — recording status AND recording dispatch errors. A separate `ErrorRecorder` wrapper would be cleaner. This was noted in the prior session's status report but accepted as a pragmatic tradeoff.

7. **`gopls` shows stale "unused function" warning**: After adding the `writeDispatchError` call sites across 5 files, gopls still reports the function as unused (`unusedfunc` diagnostic). The function IS used (grep confirms 12+ call sites). The LSP cache appears stale — a `gopls restart` would resolve it.

8. **usermgmt has `verification_totp_http.go:237` and `:244` still using `writeError`** instead of `writeDispatchError` — these are `http.StatusInternalServerError` responses for non-dispatch errors (TOTP code generation failures), so `writeError` is arguably correct there. But the inconsistency could confuse future readers.

9. **`oauth2_http.go:104` still uses `writeError`** for the OAuth2 callback error path — this one writes a custom status + message derived from error classification. It's a different pattern (redirect-based error handling, not JSON). Not wrong, but a third error-writing pattern alongside `writeError` and `writeDispatchError`.

10. **Test for Transient/Conflict in RequestLoggingSlog does not check error_ctx_**: The Rejection test checks `error_ctx_user_id`, but the Transient and Conflict tests only check `error_family` and `error_code`. They don't add `.WithContext()` to the error so there's nothing to check — but it means we're not testing that Transient/Conflict errors CAN carry context through the logging path.

---

## F) Up to 50 things we should get done next

### P0 — Must do before next release

1. **Create GitHub Releases** for all 6 tags — consumers see no release notes today
2. **Verify `go get @v4.2.1`** resolves from the Go proxy
3. **Check pkg.go.dev** picked up v4.2.1
4. **Configure go-auto-upgrade exclusion** for `encoding/json` → `v2` migration
5. **Decide: v4.2.2 patch or v4.3.0?** — CSRFTestToken is a breaking change on an unreleased API version (v4 module path but no v4.3.0 tag yet). If no consumer has upgraded to the new signature, it could be folded into v4.2.1 retroactively. If anyone has, it needs a new tag.

### P1 — High impact, well-scoped

6. **Refactor `es_readmodel.go:Handle`** — extract 12-case switch into a dispatch table or handler map. Only remaining lint issue in the entire codebase.
7. **Fix 2 context losses in `service_oauth2.go`** — `providerName`/`state` not attached to errors in the OAuth2 flow.
8. **Write `nix run .#release-checklist` app** — validates CHANGELOG, version refs, migration docs before tagging.
9. **Add release process docs to CONTRIBUTING.md** — tag naming, CHANGELOG order, GitHub release steps.
10. **Write `nix run .#check-docs-freshness` app** — scan .md files for stale version strings.
11. **Research go-cqrs-lite v3.6.0/v3.7.0 release notes** — may have missed capabilities (repo was 404).
12. **Research httputil v0.5.0 changes** — only use ClientIP, but should verify no breaking changes.
13. **Research templ-components v0.10.0 changes** — adminui dependency, check for breaks.

### P1 — Testing gaps

14. **OAuth2 FinishLogin integration test** — only BeginLogin tested cross-module.
15. **usermgmt HTTP handler coverage** — oauth2_http.go, credential_http.go edge cases at ~0%.
16. **Postgres setup test** — `NewPostgresEventSourcedSetup` has no test (needs test DB).
17. **adminui coverage improvement** — 66.8% → target 70%+.
18. **Property-based tests for event fold functions** — `foldUser`, `foldMembership`, `foldTenant`, `foldBot`.
19. **Contract tests** between root and usermgmt (RateLimiter boundary).
20. **Integration test importing published version** (not local replace).
21. **Test that error_ctx_\* traverses wrapped errors for Transient/Conflict families** (not just Rejection).
22. **Benchmark dedup.Ring vs old map** for typical journal sizes.

### P2 — Architecture

23. **God-package split: domain layer extraction** — 20 pure fold/decide files → `usermgmt/domain/`. #1 architectural debt.
24. **Root module: extract SSE/WS/ratelimit** into optional sub-packages — 16 of 46 files have zero core coupling.
25. **Shared types module** for WebAuthnUserData, OAuth2UserInfo — eliminates JSON serialization boundary smell.
26. **Raise strategy module dep budgets** — totp 2→3, webauthn 2→3, oauth2 4→5 (all at capacity).
27. **Standardize import grouping** — some files separate go-error-family/go-cqrs-lite imports, others group them.
28. **TypedRepository adoption** — eliminate command type assertions across all deciders.
29. **Unify error-writing patterns** — `writeError` vs `writeDispatchError` vs oauth2_http's custom redirect pattern. Document when to use which.

### P2 — Features

30. **`OnSubscribe`/`OnUnsubscribe` hooks** on fanOut/Broadcaster — consumer-requested connection metrics.
31. **`broadcaster.ServeSSE()` helper** — 2 consumers wrote identical boilerplate.
32. **Configurable TTLs** — lockout TTL, OAuth2 state TTL, verification token TTL.
33. **Admin UI: TOTP management views** — enable/disable, show QR code.
34. **Admin UI: OAuth2 link/unlink views**.
35. **Snapshot integration** for high-event-volume aggregates (>10K events).
36. **Redis adapters** for SessionStore/OAuth2StateStore/IdempotencyStore.
37. **MySQL support** for event store (Postgres + SQLite only currently).
38. **Persistent offline queue (Phase 2b)** — OPFS persistence per ADR-0030.

### P3 — Polish & Future

39. **OpenAPI spec generation** for HTTP endpoints.
40. **Consumer-facing v3→v4 codemod** — automated migration tool.
41. **Evaluate encoding/json/v2** when Go stabilizes it (v4.1+ target).
42. **Document provider implementation guide** — how to write custom TOTPProvider/WebAuthnProvider/OAuth2Provider.
43. **Automate GitHub Release creation via CI** on tag push (`.github/workflows/release.yml`).
44. **Load testing benchmarks** for SSE broadcaster under high fan-out.
45. **Consolidate `errors.As` in writeDispatchError with root's `errorCode` pattern** — currently two different traversal strategies.
46. **Consider extracting `ErrorRecorder` from `StatusRecorder`** — fix the SRP violation.
47. **Add `errors.AsType[*event.Error]` suggestion from gopls** in `service_register.go:120`.
48. **Clean up unnecessary type arguments** flagged by gopls in scenario test files (17 instances).
49. **Review `handler_me_test.go:34`** — uses `writeError` directly in test (should it use the test helper?).
50. **Wire `nix run .#coverage-gate`** into the verification routine — we didn't run it this session.

---

## G) Top 2 Questions

### Q1: Should CSRFTestToken's breaking change trigger v4.2.2 or wait for v4.3.0?

CSRFTestToken changed from `func(CSRFMiddleware) string` to `func(CSRFMiddleware) (string, *http.Cookie)`. This is a source-level breaking change. However:

- The v4 module path (`github.com/larsartmann/cqrs-htmx/v4`) has no v4.3.0 tag yet.
- v4.2.1 is the latest root tag (pushed but no GitHub Release).
- No consumers are known to use the test helper in CI (it's a test utility).
- The old signature was fundamentally broken for its intended purpose (CSRF round-trip testing).

**My recommendation**: Fold it into the existing v4.2.1 tag retroactively (amend the tag if no consumer depends on it), OR release v4.2.2 as a patch if v4.2.1 has consumers. The user needs to decide based on their knowledge of downstream consumers.

### Q2: Should the depguard rule also block other GOEXPERIMENT-gated stdlib packages?

Currently we block `encoding/json/v2` and `encoding/json/jsontext` because go-auto-upgrade migrated 26 files to them and broke the build. But the project uses GOEXPERIMENT build tags (`goexperiment.jsonv2`, `goexperiment.arenas`, etc.). Should depguard also block:

- `arena` (experimental)
- `maps` / `slices` from `exp` (if any experimental versions exist)
- Other future experimental stdlib packages that go-auto-upgrade might adopt?

**My recommendation**: No — only block packages that have actually caused build breaks. `encoding/json/v2` is the only confirmed offender. Over-blocking would prevent legitimate experimentation with features the project already enables via GOEXPERIMENT tags. The rule is defense-in-depth against a known attack vector, not a blanket experimental-package policy.
