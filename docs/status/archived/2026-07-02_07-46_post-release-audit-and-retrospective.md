# Post-Release Audit & Retrospective — v4.0.0 / v4.0.1

**Date:** 2026-07-02 07:46 CEST
**Branch:** `v4` (synced with `master`, pushed to origin)
**Tags:** `v4.0.0` + `v4.0.1` (6 tags each) pushed
**GitHub Release:** [v4.0.0](https://github.com/LarsArtmann/cqrs-htmx/releases/tag/v4.0.0)
**Working tree:** clean

---

## a) FULLY DONE

| #  | Item                                  | Evidence                                                                                                           |
| -- | ------------------------------------- | ------------------------------------------------------------------------------------------------------------------ |
| 1  | **v4.0.0 tagged + released**          | 6 tags pushed, GitHub release created, merged to master                                                            |
| 2  | **v4.0.1 tagged + released**          | 6 tags pushed (post-release quality pass)                                                                          |
| 3  | **Auth strategy extraction**          | TOTP/WebAuthn/OAuth2 in independent Go modules. Core usermgmt: 21 deps, ZERO auth deps                             |
| 4  | **Stale doc fixes**                   | AGENTS.md coverage (94.3%/74.5%), README deps (v3.5.0), file tree (totp/webauthn/oauth2), TODO_LIST deferred items |
| 5  | **TOTP integration test**             | `integration_test/totp_integration_test.go` — full Enable→Verify→Validate chain                                    |
| 6  | **OAuth2 integration test**           | `integration_test/oauth2_integration_test.go` — BeginLogin + PKCE + nil guards                                     |
| 7  | **JSON boundary benchmark**           | 400ns–1.2µs per call (negligible for ceremonies)                                                                   |
| 8  | **Coverage parse helper tests**       | ParseUserID, MustParseUserID, ParseActorID, MustParseEmail                                                         |
| 9  | **Configurable WebAuthnSessionTTL**   | `ServiceConfig.WebAuthnSessionTTL`                                                                                 |
| 10 | **Configurable TOTPPendingSecretTTL** | `ServiceConfig.TOTPPendingSecretTTL`                                                                               |
| 11 | **Pre-generated RSA key fixture**     | `sync.Once` cache in OAuth2 tests                                                                                  |
| 12 | **Fuzz tests**                        | marshalWebAuthnUser, parseUser, parseSession                                                                       |
| 13 | **Consumer migration dry-run**        | Fresh project, `go get v4@v4.0.0`, all modules compile                                                             |
| 14 | **CHANGELOG**                         | v4.0.0 + v4.0.1 sections complete                                                                                  |
| 15 | **Domain Language**                   | Added TOTPProvider, WebAuthnProvider, OAuth2Provider, WebAuthnSessionTTL                                           |
| 16 | **CONTRIBUTING.md**                   | 11 modules, structural typing docs, dependency direction                                                           |
| 17 | **ADR-0035**                          | JSON boundary benchmark data + fuzz/integration test references                                                    |
| 18 | **SKILL.md references**               | WebAuthnSessionTTL + TOTPPendingSecretTTL in ServiceConfig docs                                                    |
| 19 | **Migration guide**                   | Stale "pending" references fixed in dependency table                                                               |

### CI Gate Summary (Final Verified)

| Gate                    | Status | Details                                                                             |
| ----------------------- | ------ | ----------------------------------------------------------------------------------- |
| Build (11 modules)      | PASS   | All compile GOWORK=off                                                              |
| Test (7 modules, -race) | PASS   | 1,085 tests (root + usermgmt + 3 sub-modules + adminui + integration)               |
| Lint (6 modules)        | PASS   | 0 issues                                                                            |
| errorfamily             | PASS   | 0 violations (root + usermgmt + adminui; sub-modules exempt)                        |
| check-modules           | PASS   | 7 modules: isolation + budget + drift + replace                                     |
| coverage-gate           | PASS   | root 94.2%, usermgmt 75.0%, totp 88.2%, webauthn 87.5%, oauth2 92.3%, adminui 66.8% |
| nix flake check         | PASS   | Formatting + devShells + apps                                                       |

---

## b) PARTIALLY DONE

| # | Item                               | What's done                                                                             | What's missing                                                                                                                                                    |
| - | ---------------------------------- | --------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | **usermgmt coverage**              | 75.0% (was 74.5%). Parse helpers now tested.                                            | Still below historical 80.1%. HTTP handlers (oauth2_http.go), Postgres setup, migration helpers all 0%. These need running servers/DBs.                           |
| 2 | **adminui coverage**               | 66.8%. seed_render_test.go covers end-to-end.                                           | Generated `_templ.go` files drag percentage down. Pre-existing, not v4-related.                                                                                   |
| 3 | **Cross-module integration tests** | WebAuthn + TOTP + OAuth2 all have integration tests                                     | OAuth2 test only covers BeginLogin (no FinishLogin — needs mock OAuth2 token server)                                                                              |
| 4 | **errorfamily in sub-modules**     | Root + usermgmt + adminui = 0 violations. Sub-module exemption documented in flake.nix. | Sub-modules use `fmt.Errorf`/`errors.New` (32 constructors). Investigated — adding `go-error-family` as dep to sub-modules would defeat the purpose of isolation. |

---

## c) NOT STARTED

| # | Item                                    | Impact | Effort | Notes                                                                                                                                                       |
| - | --------------------------------------- | ------ | ------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | **Root god-package split**              | High   | 8h+    | 87-file usermgmt god-package. Clean seams identified (domain layer, SQL infra). #1 architectural debt. Sollbruchstellen analysis at `docs/modularization/`. |
| 2 | **OAuth2 FinishLogin integration test** | Medium | 30min  | Needs mock OAuth2 token exchange server (not just BeginLogin redirect)                                                                                      |
| 3 | **scenario/v3 BDD for TOTP**            | Low    | —      | NOT APPLICABLE: TOTP has no aggregate lifecycle — it's a stateless provider                                                                                 |
| 4 | **projectionhost adoption**             | Medium | 2h     | EVALUATED in ADR-0031, rejected as "overkill for single-process deployment"                                                                                 |
| 5 | **CatchUpSubscriber adoption**          | Medium | 2h     | EVALUATED + deferred in ADR-0031, needs sync-wait wrapper                                                                                                   |
| 6 | **Snapshot integration**                | Low    | 2h     | EVALUATED, deferred until >10K events/aggregate                                                                                                             |
| 7 | **OPFS persistence (Phase 2b)**         | Low    | 4h     | ADR-0029/0030: IndexedDB banned, OPFS deferred                                                                                                              |

---

## d) TOTALLY FUCKED UP

### Issue 1: v4.0.0 tags created prematurely — **FIXED (v4.0.1)**

**What happened:** Tags were created at commit `9927a81` (docs commit). Then I did more work: integration tests, configurable TOTP TTL, RSA fixture, benchmarks, coverage tests, doc fixes. These should have been part of v4.0.0 but weren't tagged.

**Root cause:** I didn't plan the full scope before tagging. The "fix stale docs" task was supposed to be the last step before tagging, but I found more issues to fix while doing it, then kept going past the tag point.

**Fix applied:** Created v4.0.1 tags for the post-release work. CHANGELOG has a clear v4.0.1 section. Master and v4 branches are synced and tagged. This is the correct SemVer approach — can't move published tags.

**Lesson:** Plan ALL work before tagging. Execute the plan. Verify. THEN tag. Not the other way around.

### Issue 2: Master was missing post-release work — **FIXED**

v4 branch had `f246bb5` (post-release quality pass) that wasn't merged to master when I did the initial merge. The merge commit for v4.0.0 was at `712c2c0`, but `f246bb5` came after. Fixed by merging v4 → master again.

### Issue 3: CHANGELOG was incomplete — **FIXED**

v4.0.0 section didn't mention `WebAuthnSessionTTL`, `TOTPPendingSecretTTL`, integration tests, benchmarks, RSA fixture, or fuzz tests. Added a comprehensive v4.0.1 section.

---

## e) WHAT WE SHOULD IMPROVE

### Architecture

1. **87-file usermgmt god-package is the #1 debt.** The auth extraction was the 1%→51% Pareto win. The god-package split (domain layer extraction, SQL infrastructure separation) is the 20%→80% remaining. Every new feature makes this harder to split. **Priority: v4.1.**

2. **JSON serialization boundary is a design smell** that only exists for module separation. It's negligible for ceremonies (400ns–1.2µs) but conceptually wrong — we marshal a domain User to JSON, the provider unmarshals it, runs a ceremony, marshals the result, core unmarshals it. A shared types module (importable by both core and sub-modules) would eliminate this, but would add a dependency.

3. **Sub-module error family inconsistency.** Sub-modules use `fmt.Errorf`/`errors.New` while root/usermgmt/adminui enforce `go-error-family`. This is by design (sub-modules shouldn't import `go-cqrs-lite/event/v3`), but it means error classification is inconsistent across the module boundary. A standalone `go-error-family` import (without `event/v3`) could fix this — needs investigation.

### Type Model

4. **`OAuth2UserInfo` deserialization is untyped at the boundary.** The Service receives `[]byte` from the provider and unmarshals into `OAuth2UserInfo`. If the provider returns unexpected fields, they're silently dropped. Consider a validation step or a typed provider response.

5. **`WebAuthnSessionStore` interface uses `[]byte`** for opaque session data. This is correct for module isolation, but consumers implementing Redis-backed stores must serialize/deserialize manually. A `SessionData` type in core (with JSON tags) would help.

### Testing

6. **usermgmt coverage at 75% — HTTP handlers are the gap.** `oauth2_http.go`, parts of `verification_totp_http.go`, and `credential_http.go` edge cases are untested. These need `httptest.Server` fixtures.

7. **No OAuth2 FinishLogin integration test.** Only BeginLogin is tested cross-module. FinishLogin needs a mock token exchange endpoint, which is more complex to set up.

### Process

8. **Tag discipline.** Never tag until ALL planned work is committed, verified, and merged. The v4.0.0/v4.0.1 split was caused by premature tagging.

---

## f) Top 25 Things We Should Get Done Next

Sorted by `Impact × Customer-Value ÷ Effort`.

| #  | Task                                                                                                     | Impact | Effort | Category     |
| -- | -------------------------------------------------------------------------------------------------------- | ------ | ------ | ------------ |
| 1  | **God-package split: domain layer extraction** (20 pure fold/decide files → `usermgmt/domain/`)          | High   | 8h     | Architecture |
| 2  | **God-package split: SQL infrastructure separation** (9 files → `usermgmt/sql/`)                         | High   | 4h     | Architecture |
| 3  | **OAuth2 FinishLogin integration test** (mock token exchange server)                                     | Medium | 30min  | Testing      |
| 4  | **usermgmt HTTP handler coverage** (oauth2_http.go, verification_totp_http.go edge cases)                | Medium | 2h     | Testing      |
| 5  | **Error family strategy for sub-modules** (investigate standalone `go-error-family` import)              | Medium | 2h     | Architecture |
| 6  | **Consumer migration: `go get v4@v4.0.1` dry-run** (verify v4.0.1 tags resolve)                          | Medium | 10min  | Validation   |
| 7  | **Typed `SessionData` in core** (eliminate manual serialization for Redis store implementors)            | Medium | 1h     | Type Model   |
| 8  | **OAuth2UserInfo validation** (reject unexpected provider responses)                                     | Low    | 30min  | Type Model   |
| 9  | **Investigate shared types module** (`usermgmt/types/` for WebAuthnUserData, OAuth2UserInfo)             | Medium | 2h     | Architecture |
| 10 | **Benchmark full WebAuthn ceremony** (not just marshal — measure provider round-trip)                    | Low    | 1h     | Performance  |
| 11 | **scenario/v3 BDD for Bot aggregate** (improve event-sourced test coverage)                              | Low    | 1h     | Testing      |
| 12 | **scenario/v3 BDD for Tenant aggregate** (improve event-sourced test coverage)                           | Low    | 1h     | Testing      |
| 13 | **Adminui coverage improvement** (target 70%+ by testing individual handlers)                            | Low    | 2h     | Testing      |
| 14 | **Consider `projectionhost` for multi-process deployments** (re-evaluate when needed)                    | Low    | 2h     | Architecture |
| 15 | **Snapshot integration** (re-evaluate when >10K events/aggregate in production)                          | Low    | 2h     | Performance  |
| 16 | **OPFS persistence Phase 2b** (SharedWorker offline queue survives tab close)                            | Low    | 4h     | Feature      |
| 17 | **Configurable lockout TTL** (same pattern as WebAuthnSessionTTL, TOTPPendingSecretTTL)                  | Low    | 30min  | Feature      |
| 18 | **Configurable OAuth2 state TTL** (same pattern)                                                         | Low    | 30min  | Feature      |
| 19 | **Configurable verification token TTL** (same pattern)                                                   | Low    | 30min  | Feature      |
| 20 | **Admin UI: add TOTP management views** (enable/disable, show QR code)                                   | Medium | 2h     | Feature      |
| 21 | **Admin UI: add OAuth2 link/unlink views**                                                               | Medium | 2h     | Feature      |
| 22 | **Document provider implementation guide** (how to write a custom TOTPProvider/WebAuthnProvider)         | Low    | 1h     | Docs         |
| 23 | **CI: add `go mod tidy` check** (detect dependency drift)                                                | Medium | 30min  | CI           |
| 24 | **Investigate `watermill.EventToMessage` overhead** (projection adapter round-trip)                      | Low    | 1h     | Performance  |
| 25 | **Root module: extract SSE/WS/ratelimit into optional sub-packages** (16 of 46 files have zero coupling) | Medium | 4h     | Architecture |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should the god-package split happen as v4.1 (minor, backward-compatible) or v5.0 (major, breaking)?**

The tension:

- **Domain layer extraction** (moving fold/decide/state files to `usermgmt/domain/`) is **non-breaking** if done within the same module (consumers import `usermgmt` today, would import `usermgmt/domain` tomorrow — but Go's re-export via `type X = domain.X` aliases could maintain backward compat).
- **SQL infrastructure separation** (moving SQL read models to `usermgmt/sql/`) is **non-breaking** if the types are re-exported.
- BUT: moving the `Service` struct out of the god-package, or splitting it into multiple focused services, WOULD be breaking.

The modularization proposal at `docs/modularization/2026-07-01_SOLLBRUCHSTELLEN.html` identifies 3 seams, but the decision of "minor refactor within module" vs "major version with new module structure" depends on how much API surface we're willing to break — and that's a product/architecture decision only the maintainer can make.

**My recommendation:** Start with the non-breaking domain layer extraction as v4.1. Evaluate consumer impact. Then decide whether the SQL infra split needs v5.0.

---

## Commits This Session

| Commit    | Description                                                        |
| --------- | ------------------------------------------------------------------ |
| `9927a81` | docs: fix stale coverage stats, version refs, and file tree for v4 |
| `f246bb5` | test+feat+docs: v4 post-release quality pass                       |
| `cfddfc4` | docs: add v4.0.1 changelog section for post-release quality pass   |
| `6eebc21` | Merge v4: post-release quality pass (v4.0.1)                       |
