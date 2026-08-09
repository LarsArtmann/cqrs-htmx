# Status Report: TODO Follow-Up Cleanup — Execution Session

**Date:** 2026-07-10 19:31\
**Session:** Single execution session, following the prior 2-session TODO+cleanup work\
**Branch:** master (uncommitted changes)\
**Files changed:** 15 files (13 modified + 2 go.sum auto-updated)\
**Diff:** +285 / -273 lines

---

## Verification Snapshot (end of this session)

| Check               | Command                                   | Result                                                       |
| ------------------- | ----------------------------------------- | ------------------------------------------------------------ |
| Root tests          | `go test ./... -count=1 -race`            | PASS (4.0s)                                                  |
| Usermgmt tests      | `GOWORK=off go test ./... -count=1 -race` | PASS (2.6s)                                                  |
| Integration tests   | `GOWORK=off go test ./... -count=1 -race` | PASS (1.0s)                                                  |
| Adminui tests       | `GOWORK=off go test ./... -count=1 -race` | PASS (1.0s)                                                  |
| Strategy modules    | `GOWORK=off go test ./... -count=1 -race` | PASS (totp 1.0s, webauthn 1.0s, oauth2 1.1s)                 |
| Root lint           | `golangci-lint run`                       | **0 issues**                                                 |
| Usermgmt lint       | `GOWORK=off golangci-lint run`            | **0 issues** (was 1 — maintidx on es_readmodel.go:Handle)    |
| Errorfamily         | `nix run .#errorfamily`                   | All modules pass (0 stdlib error constructors)               |
| Module architecture | `nix run .#check-modules`                 | All modules within budget, no version drift, no abs replaces |
| Coverage gate       | `nix run .#coverage-gate`                 | PASS (root 94.2%, usermgmt 75.1%, all above thresholds)      |
| Formatting          | `nix fmt`                                 | 150 files formatted, 0 changed                               |
| Flake check         | `nix flake check`                         | All checks passed                                            |

**Milestone:** All modules now report **0 lint issues**. The last remaining lint finding (`maintidx` complexity 38 on `es_readmodel.go:Handle`) is eliminated.

---

## A) FULLY DONE (verified this session)

### 1. `UserReadModel.Handle` refactored — maintidx eliminated (`usermgmt/es_readmodel.go`)

**Problem:** 160-line, 12-case switch statement with cyclomatic complexity 38, triggering the `maintidx` linter. Only remaining lint issue in the entire codebase.

**Solution:** Extracted into a dispatch table pattern:

- `UserReadModel.handlers` field: `map[event.Type]userEventHandler` populated in `NewUserReadModel()`
- `userEventHandler` type: `func(m *UserReadModel, aggID id.AggregateID, evt event.Event) error`
- 12 per-event handler methods (`handleUserRegistered`, `handleEmailChanged`, etc.)
- `decodePayload[T]` generic helper: eliminates 10 copies of the `unmarshalPayload` + `WrapCorruption` boilerplate
- `Handle()` reduced from 190 lines to 8 lines

**Bonus:** `decodePayload[T]` is reusable. The membership/tenant/bot read models (if they grow similar switches) can adopt the same pattern.

### 2. OAuth2 context losses fixed (`usermgmt/service_oauth2.go`)

**Problem:** `BeginOAuthLogin` and `FinishOAuthLogin` created Transient errors without attaching the `provider` name, making provider-specific failures indistinguishable in logs.

**Fix:** Added `.WithContext("provider", provider)` to:

- `BeginOAuthLogin`: `"oauth2 begin login"` error
- `FinishOAuthLogin`: `"exchange oauth2 token"` error

### 3. `ErrorCode` exported + `writeDispatchError` consolidated (`errors.go`, `logging.go`, `structured_error.go`, `usermgmt/http.go`)

**Problem:** Two different error-code traversal strategies:

- Root's `errorCode()`: walks full chain via `errors.Unwrap` loop, returns **deepest** (domain-specific) code
- usermgmt's `writeDispatchError`: used `errors.As` which returns **outermost** (infrastructure wrapper) code

**Fix:**

- Exported `errorCode` → `ErrorCode` in root `errors.go`
- Updated all 3 internal callers (`errors.go:309`, `logging.go:236`, `structured_error.go:120`)
- Rewrote `writeDispatchError` to use `cqrshtmx.ErrorCode(err)` — single traversal strategy
- Removed now-unused `"errors"` and `"event/v3"` imports from `usermgmt/http.go`

### 4. `ErrorRecorder` extracted from `StatusRecorder` (`logging.go`, `handler.go`)

**Problem:** `StatusRecorder` had two responsibilities: recording HTTP status AND capturing dispatch errors. SRP violation noted in prior status report.

**Fix:**

- New `ErrorRecorder` struct with `dispatchErr` field
- Exported methods: `SetDispatchError(err error)` and `DispatchError() error`
- `StatusRecorder` embeds `ErrorRecorder` via composition (not inheritance)
- `dispatchErrorRecorder` interface updated to use `SetDispatchError` (exported)
- `handler.go` call site updated
- `RequestLoggingSlog` uses `rw.DispatchError()` instead of `rw.dispatchErr`

### 5. `errors.AsType` adoption (`usermgmt/service_register.go`)

**Problem:** gopls hint: `errors.As(err, &ee)` can be simplified using `errors.AsType[*event.Error]`.

**Fix:** Replaced `var ee *event.Error; errors.As(err, &ee)` with `ee, ok := errors.AsType[*event.Error](err)`.

### 6. Transient/Conflict `error_ctx` tests enhanced (`logging_test.go`)

**Problem:** Prior session added tests for Transient/Conflict families but they only checked `error_family` and `error_code`, not `error_ctx_*`. No test verified that context key-values traverse wrapped errors for non-Rejection families.

**Fix:** Refactored 3 duplicated test cases into a `DescribeTable` with `dispatchErrorLogEntry` helper. Each entry now includes `.WithContext(...)` on the error and asserts the corresponding `error_ctx_*` field in the log output. Also eliminated `dupl` lint warnings from the duplicated test structure.

### 7. OIDC v3.19.0 → v3.20.0 (`usermgmt/oauth2/go.mod`, `AGENTS.md`)

**Problem:** OIDC dependency was one version behind latest.

**Fix:** `go get github.com/coreos/go-oidc/v3@v3.20.0` + `go mod tidy`. Updated version reference in `AGENTS.md` dependency table. All 18 OAuth2 tests pass.

### 8. CHANGELOG + AGENTS.md updated

- Full `[Unreleased]` section updated with 6 new Changed entries and 1 new Fixed entry
- Coverage numbers updated in AGENTS.md (74.5% → 75.1% usermgmt, 94.3% → 94.2% root)
- Lint status updated: "All modules report 0 issues"
- OIDC version reference updated in dependency table

---

## B) PARTIALLY DONE

### Nothing is partially done

All started work items are complete and verified.

---

## C) NOT STARTED (from prior TODO, not addressed this session)

### P0 — Release & Infrastructure (external blockers)

- Create GitHub Releases for 6 tags (requires GitHub CLI auth)
- Verify `go get @v4.2.1` resolves from Go proxy
- Check pkg.go.dev pickup
- Configure `go-auto-upgrade` to skip `encoding/json` → `v2` migration

### P1 — Documentation & Process

- Pre-release verification script (`nix run .#release-checklist`)
- Release process documentation in CONTRIBUTING.md
- `nix run .#check-docs-freshness` app
- Research go-cqrs-lite v3.6.0/v3.7.0 release notes (repo was 404)
- Research httputil v0.5.0 changes
- Research templ-components v0.10.0 changes

### P2 — Architecture

- God-package split (domain layer extraction from usermgmt)
- Root module SSE/WS/ratelimit extraction into sub-packages
- Shared types module for WebAuthnUserData/OAuth2UserInfo
- Raise strategy module dep budgets (all at capacity: totp 2/2, webauthn 2/2, oauth2 4/4)

### P2 — Features

- `OnSubscribe`/`OnUnsubscribe` hooks on fanOut/Broadcaster
- `broadcaster.ServeSSE()` high-level helper

### P2-P3 — Testing, Features, Tech Debt

(see section F for full list)

---

## D) TOTALLY FUCKED UP (nothing)

No regressions, no broken changes, no data loss.

One misstep during the session: attempted to remove explicit type arguments from `scenario.Given[*RegisterBotCmd, BotState](...)` calls based on gopls `infertypeargs` diagnostics. The compiler rejected this — gopls was wrong, the `Cmd` type parameter cannot be inferred from the method chain. Immediately reverted. This is a **known gopls false positive** for chained generic builder APIs.

---

## E) WHAT WE SHOULD IMPROVE

### Process observations from this session

1. **gopls `infertypeargs` is unreliable for method chains**: The diagnostic suggests removing type arguments that the Go compiler actually requires. This is a gopls issue, not a code issue. We should NOT blindly apply gopls hints without verifying the compiler agrees. 17 `infertypeargs` hints remain in scenario test files — they are all false positives.

2. **Indirect dependency version drift between modules**: Root has `x/sync v0.21.0` + `x/text v0.39.0`, usermgmt has `x/sync v0.22.0` + `x/text v0.40.0`. Both pass the version-drift checker (it only checks direct deps), but the drift is untidy. A workspace-wide `go get -u` or nix flake update would align them. Not harmful — the workspace resolves to the higher version at build time.

3. **`writeError` still used for non-dispatch errors**: 8 call sites across `oauth2_http.go` and `verification_totp_http.go` still use `writeError(w, status, msg)` instead of `writeDispatchError`. These are arguably correct — they handle non-dispatch errors (rate limits, auth checks, provider failures) where there's no classified error to extract a code from. But the two-pattern situation (`writeError` vs `writeDispatchError`) could confuse future readers. A doc comment on `writeError` explaining "use writeDispatchError for dispatch errors" would help.

4. **`handler_me_test.go:34` uses `writeError` directly in test code**: This is a test-only call, mimicking what a real handler would do. Not wrong, but inconsistent with the production `writeDispatchError` pattern.

5. **`ErrorRecorder` has unexported field accessed via exported methods**: The `dispatchErr` field is unexported (correct), but `exhaustruct` linter required us to write `ErrorRecorder{dispatchErr: nil}` in the `NewStatusRecorder` constructor — accessing an unexported field from outside the struct (but within the same package). This works because they're in the same package, but it's slightly odd. A `NewErrorRecorder()` constructor would be cleaner but adds ceremony for zero behavioral benefit.

### Code improvements (noticed but not fixing)

6. **The `decodePayload[T]` generic helper in `es_readmodel.go` could be shared with the membership/tenant/bot read models**: They have the same `unmarshalPayload` + `WrapCorruption` pattern repeated. Not urgent — each read model has 3-5 events, so the boilerplate is smaller. But a shared `internal` package (or just copying the helper) would be consistent.

7. **`integration_test/go.sum` was modified during `go mod tidy`**: An indirect dependency (`go.uber.org/mock v0.6.0` was downloaded). This is normal — the tidy pulled in a transitive test dependency. Not a problem, but the go.sum churn shows the integration_test module hadn't been tidied recently.

---

## F) Up to 50 things we should get done next

### P0 — Must do before next release

1. **Create GitHub Releases** for all 6 tags — consumers see no release notes today
2. **Verify `go get @v4.2.1`** resolves from the Go proxy
3. **Check pkg.go.dev** picked up v4.2.1
4. **Configure go-auto-upgrade exclusion** for `encoding/json` → `v2` migration
5. **Decide: v4.2.2 patch or v4.3.0?** — CSRFTestToken + ErrorCode export + SetDispatchError export are breaking changes on unreleased API. If no consumer has upgraded, fold into existing tag; otherwise new tag.

### P1 — High impact, well-scoped

6. **Align indirect dep versions across modules** — root `x/sync v0.21→v0.22`, `x/text v0.39→v0.40`, `x/net v0.56→v0.57`. Run workspace-wide `go work sync` or `go get -u` in each module.
7. **Write `nix run .#release-checklist` app** — validates CHANGELOG, version refs, migration docs before tagging.
8. **Add release process docs to CONTRIBUTING.md** — tag naming, CHANGELOG order, GitHub release steps.
9. **Write `nix run .#check-docs-freshness` app** — scan .md files for stale version strings.
10. **Research go-cqrs-lite v3.6.0/v3.7.0 release notes** — may have missed capabilities.
11. **Research httputil v0.5.0 changes** — only use ClientIP, verify no breaking changes.
12. **Document `writeError` vs `writeDispatchError` usage** — add doc comment to `writeError` explaining when to use which.
13. **Add `NewErrorRecorder()` constructor** — cleaner than `ErrorRecorder{dispatchErr: nil}` in `NewStatusRecorder`.

### P1 — Testing gaps

14. **OAuth2 FinishLogin integration test** — only BeginLogin tested cross-module.
15. **usermgmt HTTP handler coverage** — oauth2_http.go, credential_http.go edge cases at ~0%.
16. **Postgres setup test** — `NewPostgresEventSourcedSetup` has no test (needs test DB).
17. **adminui coverage improvement** — 66.8% → target 70%+.
18. **Property-based tests for event fold functions** — `foldUser`, `foldMembership`, `foldTenant`, `foldBot`.
19. **Contract tests** between root and usermgmt (RateLimiter boundary).
20. **Integration test importing published version** (not local replace).
21. **Benchmark dedup.Ring vs old map** for typical journal sizes.
22. **Test `ErrorCode` with deeply wrapped errors** — verify deepest-code traversal works through 3+ levels of wrapping.
23. **Test `ErrorRecorder` standalone** — verify it works independently of `StatusRecorder` (new capability from this session).

### P2 — Architecture

24. **God-package split: domain layer extraction** — 20 pure fold/decide files → `usermgmt/domain/`. #1 architectural debt.
25. **Root module: extract SSE/WS/ratelimit** into optional sub-packages — 16 of 46 files have zero core coupling.
26. **Shared types module** for WebAuthnUserData, OAuth2UserInfo — eliminates JSON serialization boundary smell.
27. **Raise strategy module dep budgets** — totp 2→3, webauthn 2→3, oauth2 4→5 (all at capacity).
28. **Standardize import grouping** — some files separate go-error-family/go-cqrs-lite imports, others group them.
29. **TypedRepository adoption** — eliminate command type assertions across all deciders.
30. **Unify error-writing patterns** — `writeError` vs `writeDispatchError` vs oauth2_http's custom redirect pattern.
31. **Apply dispatch-table pattern to other read models** — membership/tenant/bot read models have similar switches (smaller, but same pattern).

### P2 — Features

32. **`OnSubscribe`/`OnUnsubscribe` hooks** on fanOut/Broadcaster — consumer-requested connection metrics.
33. **`broadcaster.ServeSSE()` helper** — 2 consumers wrote identical boilerplate.
34. **Configurable TTLs** — lockout TTL, OAuth2 state TTL, verification token TTL.
35. **Admin UI: TOTP management views** — enable/disable, show QR code.
36. **Admin UI: OAuth2 link/unlink views**.
37. **Snapshot integration** for high-event-volume aggregates (>10K events).
38. **Redis adapters** for SessionStore/OAuth2StateStore/IdempotencyStore.
39. **MySQL support** for event store (Postgres + SQLite only currently).
40. **Persistent offline queue (Phase 2b)** — OPFS persistence per ADR-0030.

### P3 — Polish & Future

41. **OpenAPI spec generation** for HTTP endpoints.
42. **Consumer-facing v3→v4 codemod** — automated migration tool.
43. **Evaluate encoding/json/v2** when Go stabilizes it.
44. **Document provider implementation guide** — how to write custom TOTPProvider/WebAuthnProvider/OAuth2Provider.
45. **Automate GitHub Release creation via CI** on tag push (`.github/workflows/release.yml`).
46. **Load testing benchmarks** for SSE broadcaster under high fan-out.
47. **Consider upgrading casbin to v3.11.0 snapshot** — currently at v3.10.0 stable; v3.11.0 snapshots exist but are not stable releases.
48. **Remove stale gopls `infertypeargs` diagnostics** — 17 false positives in scenario tests. Consider filing a gopls issue or adding `//nolint:infertypeargs` comments.
49. **Review `handler_me_test.go:34`** — uses `writeError` directly in test (should it use the test helper?).
50. **Wire `nix run .#coverage-gate`** into the pre-commit/verification routine — not currently part of the default verification flow.

---

## G) Top 2 Questions

### Q1: Should the breaking API changes from this session (ErrorCode export, SetDispatchError export, ErrorRecorder type) trigger v4.2.2 or v4.3.0?

Three changes this session are technically breaking on the v4 module path:

- `errorCode` → `ErrorCode` (unexported → exported, but callers in the same package only)
- `setDispatchError` → `SetDispatchError` (was unexported, called via interface)
- New `ErrorRecorder` type (new exported type, but additive)

However: `setDispatchError` was unexported (no external consumer could call it), and `ErrorCode` was also unexported. The only real external impact is if someone was using reflection or type-asserting against the `dispatchErrorRecorder` interface — which was also unexported. So **these are not actually breaking for external consumers**. They could fold into v4.2.1 (unreleased) or ship as v4.2.2.

**I cannot decide this myself** because it depends on whether the v4.2.1 tag has external consumers I don't know about, and whether the maintainer considers the module pre-stable or stable.

### Q2: Should we upgrade the indirect dependencies (x/sync, x/text, x/net) across all modules now, or wait for the next nix-flake-update?

Root has `x/sync v0.21.0`, usermgmt has `x/sync v0.22.0` (latest is v0.22.0). Same pattern for x/text (v0.39 vs v0.40) and x/net (v0.56 vs v0.57). These are all indirect. The version-drift checker passes because it only checks direct dependencies. `go mod tidy` in each module independently keeps them at different versions because the direct deps that pull them transitively resolve differently.

**I cannot decide this myself** because: (a) a workspace-wide `go get -u` could pull in other unexpected changes, (b) the `nix-flake-update` bot may already be scheduled to handle this, and (c) these are indirect deps with no behavioral impact — upgrading them is purely cosmetic tidiness vs. potential CI churn risk.
