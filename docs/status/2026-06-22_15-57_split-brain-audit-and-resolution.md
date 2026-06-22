# Split-Brain Audit & Resolution — Comprehensive Status

**Date:** 2026-06-22 15:57
**Branch:** master
**Session scope:** Split-brain data model audit → execution plan → 7 of 10 fixed
**Commits this session:** 13
**Lines changed:** +2,261 / −830 across 49 files
**Test status:** All 4 modules pass with race detector

---

## a) FULLY DONE

### Audit

| Deliverable | Status | Artifact |
|---|---|---|
| Split-brain audit report | Done | `docs/research/SPLIT-BRAIN.html` (53KB, 10 findings, file:line evidence) |
| Comprehensive execution plan | Done | 32 tasks, each ≤12 min, sorted by impact/effort |
| Self-review (M1–M5 findings) | Done | Amended into report — 5 additional gaps found and folded into plan |

### Split-brain fixes shipped

| # | Split Brain | What Was Done | Commit |
|---|---|---|---|
| **1** | Roles in 3 places | Removed `User.Roles`, `User.HasRole()`, `UpdateRolesCmd`, `decideUpdateRoles`, `RolesUpdatedEvent`. Membership aggregate is now the sole source of roles. `RequireAdminRole` now a factory that checks Authz. | `7a8e87f` |
| **6a** | Ghost `UserLoggedInEvent` | Deleted (0 usages). | `6019748` |
| **6b** | Double event emission | Removed direct `s.emit()` calls; enriched bridge to carry full payloads. | `73d7e3e` |
| **6c** | Legacy EventHandler system | Removed `EventHandler` type, `ServiceConfig.EventHandler` field, `bridgeEventHandler`, `emit()` method, `UserRegisteredEvent` struct, and all EventHandler tests. Consumers use the bus directly. | `3f95cf6` |
| **5** | Twin setup constructors | `NewService` now delegates to `NewEventSourcedSetup` for all shared infra (store, bus, repos, projections, authz). -81 lines of duplicated wiring. | `2894292` |
| **10** | Credential/ExternalAccount field clone | Extracted `credentialCore` (8 fields) and `externalAccountCore` (4 fields) embedded structs. Domain types and event payloads now share fields via embedding. | `3541f8a` |
| **7** | SSE/WS broadcaster hook duplication | All 4 inline hook builders (`BroadcastOnSuccess`, `BroadcastOnError`, + WS variants) now delegate to shared `fanOut.broadcastOnSuccessHook`/`broadcastOnErrorHook`. | `1dbbc0b` |
| **8** | Email validation split | `RegisterRequest.Validate` now routes through `ParseEmail` (was using raw `mail.ParseAddress` with different normalization). | `b921dca` |

### Test suite verification

```
==> Root module:          ok  (95.2% coverage, 2.3s)
==> catalog submodule:    ok  (1.0s)
==> usermgmt submodule:   ok  (82.8% coverage, 2.5s with -race)
==> integration_test:     ok  (1.0s)
```

---

## b) PARTIALLY DONE

| Item | What's Done | What Remains |
|---|---|---|
| **Email type enforcement (#8)** | Validation unified through `ParseEmail` at the register boundary | `User.Email`, `UserState.Email`, and event payload `Email` fields are still `string`, not the `Email` branded type. Full enforcement requires changing the `User` struct field type — a breaking API change that touches serialization, the read model, fold functions, and every command. Not done due to cascading scope. |
| **Roles legacy event decoding** | Write path fully removed (no new `RolesUpdated` events can be created). `eventRolesUpdated` constant and `RolesUpdatedPayload` retained and marked legacy for backward-compatible decoding of existing events in stores. | The `MigrateRolesToMemberships` tool exists but hasn't been documented as mandatory upgrade step in CHANGELOG. |

---

## c) NOT STARTED (Skipped — Module Boundary Constraint)

These 3 split brains were identified but **cannot be fixed without breaking the module architecture**:

| # | Split Brain | Why Skipped |
|---|---|---|
| **2** | Two `UserID` types (root `id.UserID` ULID-backed vs usermgmt `brandid.ID` string-backed) | Root and usermgmt are **independent Go modules with zero mutual imports**. Unifying requires either a new shared `identity` sub-module both depend on, or one module adopting the other's ID library. Both are major architectural decisions beyond a split-brain fix. |
| **3** | Two `RateLimitConfig` types (root token-bucket vs usermgmt hand-rolled fixed-window) | Same module boundary constraint. usermgmt cannot import root's `RateLimiterConfig`. The usermgmt limiter is simpler (no eviction, no burst) but serves a different need (per-endpoint HTTP rate limiting within the auth handler). |
| **4** | Actor identity: root `string` vs usermgmt typed `ActorID` | Same root cause as #2 — context API is in root, typed `ActorID` is in usermgmt. |

These are **architectural constraints, not oversights**. The "zero mutual imports" rule is documented in `AGENTS.md` and is intentional.

---

## d) TOTALLY FUCKED UP

### Test struct literal mass-fix: cascading brace errors

When extracting `credentialCore` and `externalAccountCore`, I attempted to batch-fix all test file struct literals using Python regex scripts. This went badly:

1. The regex scripts introduced **double-nesting** (`credentialCore: credentialCore{credentialCore: ...`), **extra closing braces** (`}}}`), and **misindented slice closings**.
2. Each fix attempt created new errors that required more fixes — 8+ iterations of Python scripts to clean up.
3. The root cause: Go struct embedding means promoted fields can't be used directly in struct literals — you must use the embedded field name. The regex scripts couldn't handle the multi-line, nested-brace complexity.
4. **Lesson learned:** When making a struct embedding change that affects 20+ test files, do it manually one file at a time, or use `gofmt` after each edit to catch syntax errors immediately. Automated regex on Go source code is fragile.

**Impact:** ~45 minutes of debugging brace errors that should have been a 10-minute mechanical task. All errors were eventually resolved — the final state compiles and all tests pass.

### Stale LSP diagnostics (false alarm)

After all fixes, the LSP showed 23 `typecheck` warnings in test files. `go vet` and `go test` reported zero errors. The LSP was stale from the intermediate broken states. Not a real issue, but caused confusion about whether the code was actually clean.

---

## e) WHAT WE SHOULD IMPROVE

1. **Roles migration documentation** — The `MigrateRolesToMemberships` tool exists but there's no CHANGELOG entry or upgrade guide for consumers who have existing `RolesUpdated` events in their stores. This is a **breaking change** that needs migration documentation.

2. **Coverage dropped from 88.7% to 82.8%** in usermgmt — Removing the roles tests and EventHandler tests reduced coverage. New tests are needed for the membership-based roles flow (`grantTestRole` → `Authz.Enforce`).

3. **`ExportUser` lost the Roles field** — The export format no longer includes roles. Consumers who depend on exported role data need an alternative (querying `Authz.RolesForUser` during export, or a separate membership export).

4. **`RequireAdminRole` is now a factory** — Breaking API change. Was `func(user *User) error`, now `func(authz *Authz) AuthorizerFunc`. Consumers passing `RequireAdminRole` as a value need to call `RequireAdminRole(authz)` instead.

5. **AGENTS.md is stale** — The architecture section still documents `User.Roles`, `EventHandler`, `UpdateRoles`, and the legacy event structs. Needs updating to reflect the new membership-only model.

6. **No integration test for the roles migration** — `MigrateRolesToMemberships` is critical for existing consumers but has no integration test verifying it works end-to-end against a realistic event store.

---

## f) Top 25 Things to Get Done Next

| Priority | Task | Impact | Effort |
|---|---|---|---|
| **P0** | Update `AGENTS.md` to remove references to `User.Roles`, `EventHandler`, `UpdateRoles`, legacy event structs | High (prevents confusion) | S |
| **P0** | Write CHANGELOG / BREAKING CHANGES entry for v2.7.0 (roles removal, EventHandler removal, constructor change, credential core extraction) | High (consumer communication) | M |
| **P0** | Add integration test for `MigrateRolesToMemberships` against a realistic event store with legacy roles events | High (migration safety) | M |
| **P1** | Restore usermgmt coverage to 88%+ — add tests for membership-based roles flow, credential core, external account core | High (regression safety) | M |
| **P1** | Add roles export to `ExportUser` via `Authz.RolesForUser` query during export | Medium (feature parity) | S |
| **P1** | Run `nix run .#lint` and fix any lint issues from the session's changes | High (quality gate) | S |
| **P1** | Document the `RequireAdminRole` API change (factory pattern) in handler docs | Medium (consumer UX) | XS |
| **P2** | Enforce `Email` branded type on `User.Email` and event payloads (finish split-brain #8) | Medium (type safety) | M |
| **P2** | Extract a shared `identity` sub-module to unify `UserID` across root + usermgmt (split-brain #2) | High (architecture) | L |
| **P2** | Document the module boundary constraint for rate limiters and actor identity in a new ADR | Medium (architecture clarity) | S |
| **P2** | Run `nix fmt` to normalize formatting after all the test file edits | Medium (hygiene) | XS |
| **P3** | Consider deprecating `eventRolesUpdated` and `RolesUpdatedPayload` entirely after 2 release cycles | Low (tech debt) | XS |
| **P3** | Add `gofmt -l` check to pre-commit hook to catch formatting issues before commit | Low (process) | XS |
| **P3** | Review whether `MigrateRolesToMemberships` should be called automatically in `NewService` when legacy events are detected | Medium (consumer UX) | M |
| **P3** | Add `casbinProjection` as exported field or accessor on `EventSourcedSetup` for consumers who need it | Low (API completeness) | XS |
| **P3** | Consider adding `Roles()` method to `Service` that queries memberships and returns roles for a user (replaces the removed `User.HasRole`) | Medium (consumer convenience) | S |
| **P4** | Audit `integration_test` module for any references to removed APIs | Medium (compatibility) | S |
| **P4** | Review whether `NewEventSourcedSetup` should accept a custom `Authz` (currently always creates its own) | Low (flexibility) | S |
| **P4** | Add benchmarks for the new membership-based roles lookup vs old `User.HasRole` (perf regression check) | Low (performance) | S |
| **P4** | Consider whether `UserRegisteredPayload.Roles` should be removed (currently retained for backward compat but no longer meaningful) | Low (tech debt) | XS |
| **P4** | Document the `credentialCore` / `externalAccountCore` embedding pattern in a brief ADR | Low (knowledge sharing) | S |
| **P5** | Review all `//nolint:gocognit` comments — the constructor simplification may have reduced complexity enough to remove some | Low (hygiene) | XS |
| **P5** | Consider unifying the two `Membership.HasRole` / `Membership.HasAnyRole` methods with `slices.Contains` (LSP hint) | Low (code quality) | XS |
| **P5** | Review `events.go` — now just a package comment; consider whether the file should be renamed or merged into `doc.go` | Low (hygiene) | XS |
| **P5** | Consider adding `Service.RolesForUser(userID UserID, tenant TenantID) ([]Role, error)` as a convenience method | Low (consumer UX) | XS |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should `MigrateRolesToMemberships` be called automatically inside `NewService` when legacy roles events are detected in the store?**

Arguments for automatic:
- Zero-friction upgrade for existing consumers — their stores are migrated on first boot
- Prevents the split-brain from manifesting if someone has old events

Arguments against:
- `NewService` doesn't read the event store during construction (it starts projections but doesn't scan historical events)
- Automatic mutation of an event store on startup is a dangerous side effect for a library
- The migration dispatches commands that produce NEW events — doing this implicitly could double-migrate or conflict with consumer-controlled migration timing

The core tension: **a library should not mutate its consumer's event store implicitly**, but requiring consumers to know about and call `MigrateRolesToMemberships` manually creates a footgun. The right answer likely depends on whether this library considers itself "infrastructure" (never mutate) or "framework" (opinionated defaults), which is a product-level decision I can't make.
