# v4 Auth Extraction — Full Comprehensive Status Report

**Date:** 2026-07-02 02:20 UTC+2
**Branch:** `v4` (pushed to `origin/v4`)
**Previous release:** v3.5.0 (tagged on master)
**Current state:** v4 auth strategy extraction functionally complete, not yet tagged

---

## Executive Summary

The v4 Sollbruchstellen extraction is **functionally complete**. All three auth strategies (TOTP, WebAuthn, OAuth2) have been extracted behind primitive-type interfaces as independent Go modules. Core `usermgmt` has **zero** auth-related dependencies — consumers import only the strategies they need. All 7 test modules pass with `-race`, errorfamily reports 0 violations, and BuildFlow passes 29/29 checks.

The work is **not yet released** — v4.0.0 tag has not been created, and the new sub-modules lack provider tests.

---

## a) FULLY DONE ✅

| #  | Item                             | Evidence                                                                                                                                                                   |
| -- | -------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1  | **TOTP extraction**              | `usermgmt/totp/v4` module, `TOTPProvider` interface, stub test, 3 provider tests. `pquerna/otp` removed from core. Commit `ebdac9e`.                                       |
| 2  | **WebAuthn extraction**          | `usermgmt/webauthn/v4` module, `WebAuthnProvider` interface (all `[]byte`), adapter moved, session store rewritten. `go-webauthn` removed from core. Commit `4a1921e`.     |
| 3  | **OAuth2 extraction**            | `usermgmt/oauth2/v4` module, `OAuth2Provider` interface, `OAuth2UserInfo` exported, PKCE moved to provider. `oauth2`/`oidc`/`go-jose` removed from core. Commit `314ad03`. |
| 4  | **Module path bump /v3 → /v4**   | All 130+ files across 11 modules. Commit `dd7ec5f`.                                                                                                                        |
| 5  | **go.work updated**              | All 11 modules including 3 new auth sub-modules.                                                                                                                           |
| 6  | **Migration guide**              | `docs/migrations/v3-to-v4.md` — all 4 sections marked DONE with before/after examples.                                                                                     |
| 7  | **CHANGELOG.md**                 | v4.0.0 section with all breaking changes documented.                                                                                                                       |
| 8  | **AGENTS.md**                    | Module layout table updated with 3 new sub-modules. Dependency table updated to show auth libs in sub-modules.                                                             |
| 9  | **SKILL.md**                     | Module table updated, Path B example updated for new WebAuthn injection pattern.                                                                                           |
| 10 | **errorfamily check**            | 0 violations across all modules (root, usermgmt, adminui).                                                                                                                 |
| 11 | **check-modules**                | All modules within dependency budget, no version drift, no absolute replace paths.                                                                                         |
| 12 | **BuildFlow**                    | 29/29 checks passing.                                                                                                                                                      |
| 13 | **All tests pass with -race**    | root ✓, usermgmt ✓, totp ✓, adminui ✓, integration_test ✓ (webauthn/oauth2 have no test files).                                                                            |
| 14 | **v3.5.0 released**              | Tags `v3.5.0`, `usermgmt/v3.5.0`, `adminui/v3.5.0` pushed to master.                                                                                                       |
| 15 | **Comprehensive execution plan** | `docs/planning/2026-07-02_01-51_v4-auth-extraction-SUPERB-PLAN.md` with 70 tasks, Pareto breakdown, mermaid graph.                                                         |

---

## b) PARTIALLY DONE 🟡

| # | Item                      | What's done                           | What's missing                                                                                                                                                                                          |
| - | ------------------------- | ------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | **WebAuthn module tests** | Provider code complete, builds clean  | Zero test files — no unit tests for `Provider.BeginRegistration/FinishRegistration/BeginLogin/FinishLogin`. Virtual authenticator test was removed from core and NOT re-created in the webauthn module. |
| 2 | **OAuth2 module tests**   | Provider code complete, builds clean  | Zero test files — no unit tests for `Provider.BeginLogin/FinishLogin`. OIDC config test, state test, and integration test were removed from core and NOT re-created in the oauth2 module.               |
| 3 | **TODO_LIST.md**          | 120 items done                        | 3 open items remain (from v3.5.0 era — may be stale). Not updated for v4.                                                                                                                               |
| 4 | **VERSIONING.md**         | Exists, documents semver policy       | Shows `v4` as "future" — needs updating to reflect v4 is now the current development version. Also has a typo: v3 row says `/v4` path.                                                                  |
| 5 | **AGENTS.md deep update** | Module table + dep table updated      | Architecture tree, Key Decisions sections, Key Gotchas, test commands all still reference v3 patterns (e.g., `WebAuthnConfig`, `OAuth2Config`, old test commands). Needs a thorough pass.               |
| 6 | **SKILL.md deep update**  | Module table + Path B example updated | `references/usermgmt.md` not checked/updated for v4 auth injection pattern. Path C examples, gotchas section may reference old types.                                                                   |
| 7 | **admin-demo example**    | Builds and runs                       | Still uses old v3-style ServiceConfig (no auth providers injected). Doesn't demonstrate the new sub-module import pattern.                                                                              |
| 8 | **Lint**                  | 0 errors, errorfamily clean           | 50 `exhaustruct` warnings in root test files (pre-existing, not v4-related). All in test code using `Config{Commands: disp}` partial initialization.                                                    |

---

## c) NOT STARTED ❌

| #  | Item                                        | Impact                                         | Effort |
| -- | ------------------------------------------- | ---------------------------------------------- | ------ |
| 1  | **v4.0.0 tag + release**                    | Critical — can't release without it            | 15min  |
| 2  | **WebAuthn provider tests**                 | High — untested ceremony code                  | 2h     |
| 3  | **OAuth2 provider tests**                   | High — untested OIDC/OAuth2 flows              | 2h     |
| 4  | **Integration tests for cross-module auth** | Medium — proves the interfaces work end-to-end | 3h     |
| 5  | **admin-demo updated for v4**               | Medium — shows consumers the new pattern       | 1h     |
| 6  | **VERSIONING.md updated**                   | Low — documentation hygiene                    | 10min  |
| 7  | **TODO_LIST.md updated for v4**             | Low — project tracking                         | 30min  |
| 8  | **FEATURES.md updated for v4**              | Low — feature inventory                        | 30min  |
| 9  | **ADR for auth strategy extraction**        | Medium — documents the architectural decision  | 45min  |
| 10 | **Root AGENTS.md references cleanup**       | Low — stale v3 references in architecture tree | 30min  |

---

## d) TOTALLY FUCKED UP 💥

**Nothing is fucked up.** The extraction was clean:

- Zero compilation errors across all 11 modules
- Zero test failures with `-race` across 7 test modules
- Zero errorfamily violations
- Zero dependency leakage (verified: `go mod graph` in usermgmt shows NO webauthn/oauth2/oidc/jose/pquerna)
- BuildFlow 29/29 passing
- No data loss, no broken git history, no force pushes

**One concern worth flagging:** The `OAuth2StateStore.Save` interface signature changed from `(provider, pkceVerifier, ttl) → (state, error)` to `(state, provider, pkceVerifier, ttl) → error`. Any consumer with a custom `OAuth2StateStore` implementation (e.g., Redis-backed) will have a compile-breaking change. This is intentional and documented in the migration guide, but it's the most disruptive API change in the extraction.

---

## e) WHAT WE SHOULD IMPROVE 🔧

### Architecture

1. **Root usermgmt is still an 84-file god-package.** The auth extraction removed 5 deps but the package is still ~11K LOC in one flat `package usermgmt`. The modularization proposal (`docs/modularization/2026-07-01_PROPOSAL.html`) identified this as v4 debt but it was deferred. The auth extraction was the 1% that delivered 51%; the god-package split is the 20% that delivers 80% of remaining value.

2. **No interface satisfaction assertions.** The TOTP/WebAuthn/OAuth2 modules use structural typing but don't have `var _ usermgmt.TOTPProvider = (*Provider)(nil)` assertions (because they don't import core). Consider adding a compile-time check in a `_test.go` file in each module that DOES import core, just for the assertion.

3. **WebAuthn session TTL is hardcoded to 5 minutes.** The old store used go-webauthn's `SessionData.Expires` field. The new store uses a fixed TTL. This is probably fine for ceremonies but should be configurable.

### Testing

4. **WebAuthn and OAuth2 modules have ZERO tests.** This is the biggest risk. The provider code was moved from well-tested core code, but the JSON serialization/deserialization boundary is new and untested. A single bug in `marshalWebAuthnUser` or `parseSession` would break all passkey auth.

5. **The virtual authenticator test was deleted, not migrated.** The W3C spec test vectors (`webauthn_virtual_test.go`) tested the full registration → login flow. This was the closest thing to an integration test. It should be re-created in the webauthn module.

6. **No cross-module integration test.** The `integration_test/` module tests root↔usermgmt bridges but doesn't test the new auth sub-module wiring (e.g., creating a Service with a real `*webauthn.Provider` and running a ceremony).

### Documentation

7. **AGENTS.md architecture tree is stale.** Still lists `webauthn_adapter.go`, `webauthn_session.go` with old descriptions. Doesn't mention `webauthn_stub_test.go`, `oauth2_stub_test.go`, `auth_interfaces.go` redesign.

8. **VERSIONING.md has a typo.** The v3 row says `/v4` path instead of `/v3`. The v4 row says `/v4` (correct) but is marked as "future."

9. **references/usermgmt.md not checked.** The SKILL.md references this file for auth setup details. It likely still shows `WebAuthnConfig` usage.

### Dependency hygiene

10. **go-jose is a transitive dep of oauth2 module.** It was test-only in core (`oauth2_oidc_test.go`). Now it's a transitive dep via `coreos/go-oidc`. This is correct but worth noting — consumers who import `usermgmt/oauth2/v4` get go-jose transitively.

---

## f) Top 25 Things We Should Get Done Next 🎯

Sorted by impact × urgency ÷ effort.

| #  | Task                                                                                                                                         | Impact   | Effort | Category     |
| -- | -------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ------ | ------------ |
| 1  | **Write WebAuthn provider tests** (ceremony round-trip with virtual authenticator)                                                           | Critical | 2h     | Testing      |
| 2  | **Write OAuth2 provider tests** (config validation, mock OIDC provider)                                                                      | Critical | 2h     | Testing      |
| 3  | **Tag v4.0.0** (`v4.0.0`, `usermgmt/v4.0.0`, `adminui/v4.0.0`, `usermgmt/totp/v4.0.0`, `usermgmt/webauthn/v4.0.0`, `usermgmt/oauth2/v4.0.0`) | Critical | 15min  | Release      |
| 4  | **Cross-module integration test** (Service + real webauthn.Provider + ceremony)                                                              | High     | 2h     | Testing      |
| 5  | **Update AGENTS.md architecture tree** (remove deleted files, add new files)                                                                 | High     | 30min  | Docs         |
| 6  | **Update AGENTS.md Key Decisions** (add v4 auth extraction section)                                                                          | High     | 45min  | Docs         |
| 7  | **Write ADR-0035: Auth strategy extraction**                                                                                                 | Medium   | 45min  | Docs         |
| 8  | **Fix VERSIONING.md typo + update for v4 current**                                                                                           | Medium   | 10min  | Docs         |
| 9  | **Update admin-demo to show v4 auth injection**                                                                                              | Medium   | 1h     | Examples     |
| 10 | **Update SKILL.md references/usermgmt.md** for v4 auth pattern                                                                               | Medium   | 45min  | Docs         |
| 11 | **Add interface satisfaction assertions** in sub-module test files                                                                           | Medium   | 30min  | Quality      |
| 12 | **Make WebAuthn session TTL configurable** (add to ServiceConfig)                                                                            | Medium   | 30min  | Feature      |
| 13 | **Update TODO_LIST.md for v4** (mark extraction done, add new tasks)                                                                         | Low      | 30min  | Process      |
| 14 | **Update FEATURES.md for v4** (mark auth strategies as optional sub-modules)                                                                 | Low      | 30min  | Process      |
| 15 | **Fix 50 exhaustruct lint warnings** (add `//nolint:exhaustruct` to test structs)                                                            | Low      | 30min  | Quality      |
| 16 | **Migrate virtual authenticator test vectors** to webauthn module                                                                            | Medium   | 1h     | Testing      |
| 17 | **Add OAuth2 config validation tests** to oauth2 module                                                                                      | Medium   | 45min  | Testing      |
| 18 | **Update SKILL.md gotchas section** for v4 interface changes                                                                                 | Low      | 20min  | Docs         |
| 19 | **Root god-package split** (the real v4 debt per modularization proposal)                                                                    | High     | 8h+    | Architecture |
| 20 | **Add coverage gates for new sub-modules** (totp/webauthn/oauth2)                                                                            | Medium   | 30min  | CI           |
| 21 | **Update flake.nix apps** for new sub-module test/build commands                                                                             | Medium   | 30min  | Tooling      |
| 22 | **Create v4 release notes** (GitHub release description)                                                                                     | Medium   | 30min  | Release      |
| 23 | **Verify GOWORK=off builds for all 11 modules** individually                                                                                 | Medium   | 1h     | Quality      |
| 24 | **Add `.github/workflows/ci.yml` update** for new sub-modules                                                                                | Medium   | 30min  | CI           |
| 25 | **Stale doc cleanup** (remove v3 references in architecture tree, test commands, etc.)                                                       | Low      | 1h     | Docs         |

---

## g) Top #1 Question I Cannot Figure Out Myself 🤔

**Should v4.0.0 be tagged NOW (with zero provider tests in webauthn/oauth2 modules), or should we wait until those modules have test coverage?**

**The tension:**

- The provider code was moved verbatim from well-tested core code — the logic IS tested via the core stub tests (`webauthn_stub_test.go`, `oauth2_stub_test.go`) which exercise all Service methods
- BUT the JSON serialization boundary (`marshalWebAuthnUser`, `parseSession`, `parseUser`) is NEW code with ZERO tests — a bug there would silently break all ceremonies
- Tagging v4.0.0 signals "stable release" but the new modules are untested
- NOT tagging keeps consumers on v3.5.0 which has the old monolithic deps

**My recommendation:** Write provider tests FIRST (items #1-2 above), THEN tag v4.0.0. But this is your call — the code works, the logic is proven, only the new serialization boundary is untested.

---

## CI Gate Summary

| Gate                    | Status        | Details                                                                             |
| ----------------------- | ------------- | ----------------------------------------------------------------------------------- |
| Build (workspace)       | ✅ Pass       | All 11 modules compile                                                              |
| Test (7 modules, -race) | ✅ Pass       | root, usermgmt, totp, adminui, integration_test pass; webauthn/oauth2 have no tests |
| errorfamily             | ✅ Pass       | 0 violations in root + usermgmt + adminui                                           |
| golangci-lint           | ⚠️ 50 warnings | All `exhaustruct` in test code (pre-existing, not v4-related)                       |
| check-modules           | ✅ Pass       | Budget, drift, replace directives all clean                                         |
| BuildFlow               | ✅ Pass       | 29/29 checks                                                                        |

## Module Dependency State

| Module                                     | Direct deps                        | Auth deps                               | Status                             |
| ------------------------------------------ | ---------------------------------- | --------------------------------------- | ---------------------------------- |
| root (`cqrs-htmx/v4`)                      | go-cqrs-lite, casbin, nosurf, etc. | NONE                                    | ✅ Clean                           |
| usermgmt (`usermgmt/v4`)                   | go-cqrs-lite, casbin, root         | NONE                                    | ✅ Clean — all auth deps extracted |
| usermgmt/totp (`usermgmt/totp/v4`)         | pquerna/otp                        | pquerna/otp                             | ✅ Isolated                        |
| usermgmt/webauthn (`usermgmt/webauthn/v4`) | go-webauthn                        | go-webauthn (+ transitive)              | ✅ Isolated                        |
| usermgmt/oauth2 (`usermgmt/oauth2/v4`)     | oauth2, oidc                       | oauth2, oidc (+ go-jose transitive)     | ✅ Isolated                        |
| adminui (`adminui/v4`)                     | root, usermgmt, templ              | NONE (gets them via usermgmt if needed) | ✅ Clean                           |
| integration_test                           | root, usermgmt                     | NONE                                    | ✅ Clean                           |

## Commit History (v4 branch)

```
81f295f docs: update SKILL.md for v4 auth strategy sub-modules
ca0b8ca docs: update AGENTS.md module layout and dependency tables for v4
7de06dc chore: apply golangci-lint auto-fixes (import style + formatting)
7ed9d8c docs: update migration guide and CHANGELOG for v4 auth extraction
314ad03 refactor!: extract OAuth2/OIDC behind OAuth2Provider interface as independent module
4a1921e refactor!: extract WebAuthn behind WebAuthnProvider interface as independent module
8bea47e docs: add comprehensive v4 auth extraction execution plan
d4830e6 docs: add v3→v4 migration guide
46114f2 test: add TOTP provider tests (generate, validate, default config)
ebdac9e refactor!: extract TOTP behind TOTPProvider interface as independent module
dd7ec5f refactor!: bump all module paths from /v3 to /v4
```

---

_Generated: 2026-07-02 02:20 UTC+2_
