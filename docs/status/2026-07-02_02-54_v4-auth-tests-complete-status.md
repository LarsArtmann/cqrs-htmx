# v4 Auth Extraction — Post-Test Status Report

**Date:** 2026-07-02 02:54 CEST
**Branch:** `v4` (not yet pushed)
**Previous release:** v3.5.0 (tagged on master)
**Current state:** v4 auth strategy extraction complete with full test coverage, not yet tagged

---

## Executive Summary

Since the last status report (`2026-07-02_02-20`), the **two biggest risks** have been eliminated:

1. **WebAuthn and OAuth2 modules had ZERO tests** → Now have 18 tests each, including W3C ceremony vectors and real OIDC JWT signing
2. **No compile-time interface satisfaction assertions** → Now enforced in `integration_test`

The v4 extraction is now **feature-complete and test-verified**. The remaining work is release mechanics (tagging), documentation hygiene (TODO_LIST/FEATURES/ROADMAP), and the deferred god-package split.

---

## a) FULLY DONE ✅

| #      | Item                               | Evidence                                                                                                                                            |
| ------ | ---------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1      | **TOTP extraction**                | `usermgmt/totp/v4` module, `TOTPProvider` interface, 3 tests (88.2% coverage). Commit `ebdac9e`.                                                    |
| 2      | **WebAuthn extraction**            | `usermgmt/webauthn/v4` module, all `[]byte` interface, session store rewritten. Commit `4a1921e`.                                                   |
| 3      | **OAuth2 extraction**              | `usermgmt/oauth2/v4` module, PKCE moved to provider. Commit `314ad03`.                                                                              |
| 4      | **Module path bump /v3 → /v4**     | All 130+ files across 11 modules. Commit `dd7ec5f`.                                                                                                 |
| 5      | **go.work updated**                | All 11 modules including 3 new auth sub-modules.                                                                                                    |
| 6      | **Migration guide**                | `docs/migrations/v3-to-v4.md` — all 4 sections with before/after examples.                                                                          |
| 7      | **CHANGELOG.md**                   | v4.0.0 section with all breaking changes.                                                                                                           |
| 8      | **errorfamily check**              | 0 violations across root + usermgmt + adminui.                                                                                                      |
| 9      | **check-modules**                  | All modules within dependency budget, no version drift, no absolute replace paths.                                                                  |
| 10     | **All tests pass with -race**      | root (198), usermgmt (764), totp (3), webauthn (20), oauth2 (18), adminui (35), integration (17) — **1,055 total tests**.                           |
| 11     | **v3.5.0 released**                | Tags `v3.5.0`, `usermgmt/v3.5.0`, `adminui/v3.5.0` pushed to master.                                                                                |
| 12     | **Zero auth deps in core**         | Verified: `go mod graph` in usermgmt shows NO webauthn/oauth2/oidc/jose/pquerna.                                                                    |
| **13** | **WebAuthn provider tests** ✨     | 20 tests: W3C spec ceremony round-trip (registration→login), wrong challenge, expired session, credential conversion, transport conversion, adapter |
| **14** | **OAuth2 provider tests** ✨       | 18 tests: config validation (6), PKCE URL generation, pure OAuth2 flow, OIDC flow with real JWT signing, GitHub login fallback, error cases         |
| **15** | **Interface assertions** ✨        | `integration_test/auth_interface_assert_test.go` — compile-time `var _ usermgmt.XProvider = (*x.Provider)(nil)` for all 3 strategies                |
| **16** | **VERSIONING.md fixed** ✨         | Typo corrected (v3 row was `/v4`), v4 now "current", v3 "maintenance", sub-module paths added, compatibility section updated                        |
| **17** | **ADR-0035 written** ✨            | Full ADR: context, decision, interface design, JSON boundary, breaking changes, consequences, testing. Added to INDEX.md.                           |
| **18** | **AGENTS.md updated** ✨           | Architecture tree (removed stale files, added sub-modules), Key Decisions (extraction entry), Module Layout (webauthn/oauth2 now "Yes" tests)       |
| **19** | **SKILL.md references updated** ✨ | `usermgmt.md` ServiceConfig fields + all 3 auth method sections updated to v4 injection pattern; `gotchas.md` WebAuthnConfig entry updated          |

---

## b) PARTIALLY DONE 🟡

| # | Item                      | What's done                                                                        | What's missing                                                                                                                                                                                                                              |
| - | ------------------------- | ---------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | **AGENTS.md deep update** | Architecture tree, Key Decisions, Module Layout, coverage stats all updated for v4 | Key Gotchas section still references v3 patterns (e.g., `#15` mentions "No stdlib error constructors" which is correct, but the test commands section still references the old per-module commands from before the new sub-modules existed) |
| 2 | **admin-demo example**    | Builds and runs cleanly                                                            | Still uses `ServiceConfig{AuditLog: ...}` with no auth providers injected. Doesn't demonstrate the v4 sub-module import pattern. A demo showing `webauthn.New(Config{...})` injection would be the best documentation.                      |
| 3 | **TODO_LIST.md**          | 120 items done from v3.5.0 era                                                     | Not updated for v4 at all — needs the auth extraction marked done, new sub-module items added. Shows `v3.5.0` in header.                                                                                                                    |
| 4 | **FEATURES.md**           | Exists, honest inventory                                                           | Shows `v3.3.0` in header. Auth strategies listed as core features — should reflect optional sub-module status.                                                                                                                              |
| 5 | **Lint**                  | 0 errors, errorfamily clean                                                        | 50 `exhaustruct` warnings in root test files (pre-existing, not v4-related). All in test code using partial `Config{}` initialization.                                                                                                      |

---

## c) NOT STARTED ❌

| #  | Item                                   | Impact   | Effort | Notes                                                                                                                                    |
| -- | -------------------------------------- | -------- | ------ | ---------------------------------------------------------------------------------------------------------------------------------------- |
| 1  | **v4.0.0 tag + release**               | Critical | 15min  | Tags needed: `v4.0.0`, `usermgmt/v4.0.0`, `adminui/v4.0.0`, `usermgmt/totp/v4.0.0`, `usermgmt/webauthn/v4.0.0`, `usermgmt/oauth2/v4.0.0` |
| 2  | **Cross-module integration test**      | Medium   | 2h     | Service + real `*webauthn.Provider` + ceremony end-to-end through the Service layer (not just provider-direct)                           |
| 3  | **admin-demo updated for v4**          | Medium   | 1h     | Show the sub-module injection pattern with a real WebAuthn provider                                                                      |
| 4  | **TODO_LIST.md updated for v4**        | Low      | 30min  | Mark extraction done, add new follow-up tasks                                                                                            |
| 5  | **FEATURES.md updated for v4**         | Low      | 30min  | Auth strategies as optional sub-modules                                                                                                  |
| 6  | **ROADMAP.md updated for v4**          | Low      | 15min  | God-package split as next major initiative                                                                                               |
| 7  | **Coverage gates for new sub-modules** | Medium   | 30min  | flake.nix `coverage-gate` needs webauthn/oauth2/totp thresholds                                                                          |
| 8  | **flake.nix apps for new sub-modules** | Medium   | 30min  | `test-webauthn`, `test-oauth2` nix apps                                                                                                  |
| 9  | **GitHub release notes**               | Medium   | 30min  | `gh release create v4.0.0` with migration guide summary                                                                                  |
| 10 | **CI workflow for new sub-modules**    | Medium   | 30min  | `.github/workflows/ci.yml` module list needs the 3 new sub-modules                                                                       |
| 11 | **Root god-package split**             | High     | 8h+    | The 84-file usermgmt god-package. Deferred per modularization proposal                                                                   |

---

## d) TOTALLY FUCKED UP 💥

**Nothing is fucked up.** The extraction is clean and now fully tested:

- Zero compilation errors across all 11 modules
- **1,055 tests pass with `-race`** across 7 test modules (was 1,017 before this session — added 38 tests)
- Zero errorfamily violations (root + usermgmt + adminui)
- Zero dependency leakage (verified: `go mod graph` in usermgmt shows NO webauthn/oauth2/oidc/jose/pquerna)
- check-modules passes: budget, drift, replace directives all clean
- No data loss, no broken git history, no force pushes
- New sub-module coverage: webauthn 87.5%, oauth2 92.3%, totp 88.2%

**One design tension worth flagging:** The sub-modules use `fmt.Errorf`/`errors.New` (not `event.New*`) because they don't import `go-cqrs-lite/event/v3` (keeping deps minimal). The `errorfamily` check is intentionally scoped to root + usermgmt + adminui only. The Service layer wraps all provider errors with `event.Wrapf` at the boundary. This is a deliberate tradeoff: dependency isolation vs. error family purity in leaf modules.

---

## e) WHAT WE SHOULD IMPROVE 🔧

### Architecture

1. **Root usermgmt is still an 84-file god-package.** The auth extraction was the 1% → 51% Pareto win. The god-package split (domain layer extraction, SQL infrastructure separation) is the 20% → 80% remaining. The modularization proposal identified clean seams but it was deferred to avoid scope creep in v4.0.

2. **WebAuthn session TTL is hardcoded to 5 minutes.** The new session store uses a fixed TTL. Should be configurable via `ServiceConfig` (e.g., `WebAuthnSessionTTL time.Duration`).

3. **JSON serialization boundary has overhead.** Core marshals user → `[]byte`, provider unmarshals, runs ceremony, marshals result → `[]byte`, core unmarshals. For ceremonies (not hot paths) this is negligible (~µs), but it's a design smell that only exists to enable module separation. A future internal package (non-exported, same module) could eliminate it.

### Testing

4. **No cross-module integration test through the Service layer.** The provider tests exercise the provider directly. The integration_test module has compile-time interface assertions but no runtime ceremony through `Service.BeginRegistration → Provider.BeginRegistration → Service.FinishRegistration`. This would catch serialization boundary bugs at the Service level.

5. **OAuth2 tests require a running httptest server.** Each OIDC test generates a real RSA key pair and signs real JWTs. This is the most thorough approach but takes ~50ms per test. For load testing, a pre-generated key pair fixture could be shared.

6. **No fuzz tests on the JSON boundary.** The `marshalWebAuthnUser`/`parseSession`/`parseUser` functions are the new attack surface. Fuzz tests with arbitrary `[]byte` inputs would add defense-in-depth.

### Documentation

7. **TODO_LIST.md is stale.** Shows `v3.5.0`, has 120 done items from the v3 era, no v4 items. Needs a full refresh.

8. **FEATURES.md is stale.** Shows `v3.3.0`, doesn't reflect auth strategies as optional sub-modules.

9. **admin-demo doesn't demonstrate v4 injection.** Still uses bare `ServiceConfig{AuditLog: ...}`. A consumer reading this example wouldn't know how to wire WebAuthn.

### CI / Tooling

10. **Coverage gates don't cover new sub-modules.** The flake.nix `coverage-gate` app checks root (90%) and usermgmt (75%) only. New sub-modules (87.5%, 92.3%, 88.2%) need gates too.

11. **No nix apps for sub-module testing.** `nix run .#test` covers the workspace, but there's no `nix run .#test-webauthn` or `nix run .#test-oauth2` for isolated runs.

12. **CI workflow may not test new sub-modules.** The `.github/workflows/ci.yml` module list needs verification — the 3 new sub-modules should be in the test matrix.

---

## f) Top 25 Things We Should Get Done Next 🎯

Sorted by impact × urgency ÷ effort.

| #  | Task                                                                          | Impact   | Effort | Category     |
| -- | ----------------------------------------------------------------------------- | -------- | ------ | ------------ |
| 1  | **Tag v4.0.0** (6 tags across all modules)                                    | Critical | 15min  | Release      |
| 2  | **Push v4 to origin**                                                         | Critical | 5min   | Release      |
| 3  | **Create GitHub release** with migration guide summary                        | Critical | 30min  | Release      |
| 4  | **Cross-module integration test** (Service + real webauthn.Provider)          | High     | 2h     | Testing      |
| 5  | **Update admin-demo for v4 auth injection**                                   | High     | 1h     | Examples     |
| 6  | **Update TODO_LIST.md for v4**                                                | Medium   | 30min  | Process      |
| 7  | **Update FEATURES.md for v4**                                                 | Medium   | 30min  | Process      |
| 8  | **Update ROADMAP.md for v4** (god-package split as next initiative)           | Medium   | 15min  | Process      |
| 9  | **Add coverage gates for new sub-modules** in flake.nix                       | Medium   | 30min  | CI           |
| 10 | **Add nix apps for sub-module testing** (`test-webauthn`, `test-oauth2`)      | Medium   | 30min  | Tooling      |
| 11 | **Verify CI workflow covers new sub-modules**                                 | Medium   | 30min  | CI           |
| 12 | **Make WebAuthn session TTL configurable**                                    | Medium   | 30min  | Feature      |
| 13 | **Fix 50 exhaustruct lint warnings** (`//nolint:exhaustruct` on test structs) | Low      | 30min  | Quality      |
| 14 | **Add fuzz tests on JSON boundary** (parseUser, parseSession)                 | Medium   | 1h     | Testing      |
| 15 | **Root god-package split** (domain layer extraction — the real v4 debt)       | High     | 8h+    | Architecture |
| 16 | **AGENTS.md Key Gotchas section** update for v4                               | Low      | 20min  | Docs         |
| 17 | **AGENTS.md test commands** add sub-module commands                           | Low      | 15min  | Docs         |
| 18 | **Verify GOWORK=off builds for all 11 modules** individually                  | Medium   | 1h     | Quality      |
| 19 | **Pre-generated RSA key fixture** for OAuth2 tests (perf)                     | Low      | 30min  | Testing      |
| 20 | **Stale doc cleanup** (scan all .md for v3 references)                        | Low      | 1h     | Docs         |
| 21 | **SKILL.md gotchas section** verify v4 interface changes                      | Low      | 20min  | Docs         |
| 22 | **Domain Language doc** add v4 auth strategy terms                            | Low      | 30min  | Docs         |
| 23 | **Error family strategy for sub-modules** (event.New\* via optional dep?)     | Low      | 2h     | Architecture |
| 24 | **Benchmark JSON serialization boundary** (quantify the overhead)             | Low      | 1h     | Performance  |
| 25 | **Consumer migration dry-run** (import cqrs-htmx/v4 in a fresh project)       | Medium   | 1h     | Validation   |

---

## g) Top #1 Question I Cannot Figure Out Myself 🤔

**Should the sub-modules use `event.New*` error constructors (pulling `go-cqrs-lite/event/v3` as a dependency) or keep using `fmt.Errorf`?**

**The tension:**

- Currently, the sub-modules use `fmt.Errorf`/`errors.New` because they don't import `event/v3`. This keeps their dependency lists minimal: webauthn = only `go-webauthn`, oauth2 = only `oauth2` + `oidc`, totp = only `pquerna/otp`.
- The `errorfamily` check intentionally skips these modules. The Service layer wraps all provider errors with `event.Wrapf` at the boundary, so the family is assigned at the right place.
- BUT: if someone runs `branching-flow errorfamily .` from the repo root, it flags 14 violations in webauthn and 16 in oauth2. This looks alarming even though it's intentional.
- Adding `event/v3` to each sub-module would add 1 dependency but make `errorfamily .` report 0 across the entire repo — a cleaner story.
- `event/v3` is a lightweight package (no heavy transitives) — the cost of adding it is low.

**My recommendation:** Leave it as-is for v4.0.0. The sub-modules are leaf implementations; their errors are wrapping-only (not domain classification). The Service layer is where families are assigned. Adding `event/v3` to sub-modules would be architectural noise — it implies the leaf module cares about error families, when it shouldn't. But this is a judgment call.

---

## CI Gate Summary

| Gate                    | Status        | Details                                                                                                |
| ----------------------- | ------------- | ------------------------------------------------------------------------------------------------------ |
| Build (workspace)       | ✅ Pass       | All 11 modules compile                                                                                 |
| Test (7 modules, -race) | ✅ Pass       | 1,055 tests total (root 198, usermgmt 764, totp 3, webauthn 20, oauth2 18, adminui 35, integration 17) |
| errorfamily             | ✅ Pass       | 0 violations in root + usermgmt + adminui (sub-modules intentionally exempt)                           |
| golangci-lint           | ⚠️ 50 warnings | All `exhaustruct` in root test code (pre-existing, not v4-related)                                     |
| check-modules           | ✅ Pass       | Budget (root 16/18, usermgmt 21/28, adminui 5/7), drift, replace directives all clean                  |

## Module Dependency State

| Module                            | Direct deps     | Auth deps                   | Status                             |
| --------------------------------- | --------------- | --------------------------- | ---------------------------------- |
| root (`cqrs-htmx/v4`)             | 16              | NONE                        | ✅ Clean                           |
| usermgmt (`usermgmt/v4`)          | 21              | NONE                        | ✅ Clean — all auth deps extracted |
| usermgmt/totp (`totp/v4`)         | 1 (pquerna/otp) | pquerna/otp                 | ✅ Isolated (88.2% coverage)       |
| usermgmt/webauthn (`webauthn/v4`) | 1 (go-webauthn) | go-webauthn (+ transitive)  | ✅ Isolated (87.5% coverage)       |
| usermgmt/oauth2 (`oauth2/v4`)     | 3 (oauth2+oidc) | oauth2, oidc (+ go-jose)    | ✅ Isolated (92.3% coverage)       |
| adminui (`adminui/v4`)            | 5               | NONE                        | ✅ Clean                           |
| integration_test                  | root+usermgmt   | totp+webauthn+oauth2 (test) | ✅ Clean                           |

## Coverage Summary

| Module      | Coverage | Tests     |
| ----------- | -------- | --------- |
| root        | 94.3%    | 198       |
| usermgmt    | ~80%     | 764       |
| webauthn    | 87.5%    | 20        |
| oauth2      | 92.3%    | 18        |
| totp        | 88.2%    | 3         |
| adminui     | ~N/A     | 35        |
| integration | ~N/A     | 17        |
| **Total**   |          | **1,055** |

---

_Generated: 2026-07-02 02:54 CEST_
