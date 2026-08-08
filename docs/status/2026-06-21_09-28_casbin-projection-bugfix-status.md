# Status Report — Identity Redesign: Bug Fixes + CasbinProjection Correction

**Date:** 2026-06-21 09-28 | **Author:** Crush | **Branch:** master

---

## Executive Summary

Self-review of the Membership aggregate wiring revealed **5 critical bugs** in the
CasbinProjection. Role changes and member removals were **silently broken** — the
projection used the aggregate ID hash instead of the actor ID as the Casbin subject,
so policy updates wrote to the wrong key and never affected actual authorization checks.
All 5 bugs are now fixed and verified with Casbin-asserting integration tests.

---

## A. FULLY DONE ✅

### Core Identity Types (commit 0db9379)

- ActorID (kind-discriminated struct), TenantID, BotID branded types
- SessionOrigin sealed interface (DirectLogin, Impersonation)
- Membership read-model struct with HasRole/HasAnyRole
- RoleSuperAdmin + Casbin g2 role hierarchy (super_admin > admin > user > viewer)
- 12 tests, all passing

### Membership Aggregate — Types (commit 573e2e4)

- 3 event payloads (MemberAdded, MemberRolesChanged, MemberRemoved)
- 3 commands (AddMemberCmd, UpdateMemberRolesCmd, RemoveMemberCmd)
- MembershipState + foldMembership (pure function)
- 8 tests (fold, commands, aggregate ID determinism)

### Context Propagation (commit b99ca78)

- WithActorID/ActorFromContext, WithImpersonatorID/ImpersonatorFromContext
- EventOptionsFromContext propagates actor_id + impersonator_id into event metadata
- 6 tests (context set/get, event metadata verification)

### Membership Aggregate — Full Wiring (commit 0f06f34)

- MembershipDecider() + 3 decide functions (guards for existence, mandatory fields)
- RegisterMembershipCommands (3 commands via command.RegisterTyped)
- MembershipReadModel projection (FindByActor, FindByAggregateID)
- CasbinProjection subscribes to membership events
- Service + EventSourcedSetup wired with membership repo + read model
- 4 integration tests (lifecycle, multi-tenant, conflict, rejection)

### Bug Fixes (commit 85ac594)

- **MemberRolesChangedPayload** now carries ActorKind, ActorID, TenantID
- **MemberRemovedPayload** now carries ActorID, TenantID
- **decideUpdateMemberRoles** populates ActorID/TenantID from state
- **decideRemoveMember** populates ActorID/TenantID from state
- **CasbinProjection** uses payload fields (not aggregate hash) as Casbin subject+domain
- **handleMembershipEvent** extracted (reduces gocognit 36→<30)
- **removeAllRolesInDomain** helper (shared by RolesChanged + Removed)
- **Integration tests** now call `assertRolesForActor` after each operation, verifying
  Casbin returns exactly the expected roles and no stale ones

---

## B. PARTIALLY DONE ⚠️

### CasbinProjection for Membership Events

- **Status:** Now WORKING (was broken before commit 85ac594)
- **Remaining:** The `eventRolesUpdated` case still uses the domain fallback hack
  (`domain = subject` when empty). This is pre-existing User aggregate behavior, not
  related to Membership. Will be cleaned up when Roles are migrated off UserState.

### Schema Versioning

- `currentSchemaVersion = 2` but no production upcasters registered
- Only 4 of 12 User deciders stamp SchemaVersion (TOTP, ExternalAccount)
- Core deciders (RegisterUser, UpdateRoles, etc.) don't stamp it

---

## C. NOT STARTED ❌

| Task                                  | Description                                                      |
| ------------------------------------- | ---------------------------------------------------------------- |
| Session struct update                 | `Session.UserID` → `Session.ActorID`, add `Origin SessionOrigin` |
| SessionStore interface update         | Accept Session struct                                            |
| BeginImpersonation / EndImpersonation | Service methods + security guards                                |
| Session middleware update             | Extract actor + impersonator, inject into context                |
| Tenant aggregate                      | State, events, commands, fold, decider, wiring                   |
| Bot aggregate                         | State, events, commands, HMAC-SHA256 pepper token hash           |
| API token middleware                  | Bearer token → Bot resolution                                    |
| Migration projection                  | Replay old UserState.Roles → Membership events                   |
| Remove Roles from UserState           | Breaking change, needs migration                                 |
| Schema v1→v2 upcasters                | Production upcaster registration                                 |
| Catalog integration                   | New events/commands in catalog docs                              |

---

## D. TOTALLY FUCKED UP 💥 (Now Fixed)

All 5 bugs were in the CasbinProjection membership wiring (commit 0f06f34), now
fixed in commit 85ac594:

1. **MemberRolesChangedPayload missing ActorID/TenantID** — projection used aggregate
   hash as Casbin subject, never matched real policies. Role changes were lost.
2. **MemberRemovedPayload missing ActorID/TenantID** — RemoveAllRolesForUser called
   with aggregate hash, not actor ID. Removed members kept their policies.
3. **decideUpdateMemberRoles didn't populate ActorID/TenantID** — state had them
   but payload struct didn't have the fields.
4. **decideRemoveMember had same gap.**
5. **Tests only checked read model, never Casbin** — all tests passed while Casbin
   was silently wrong.

---

## E. WHAT WE SHOULD IMPROVE

1. **Always test the projection output, not just the read model.** The read model
   is a separate projection — testing it doesn't verify Casbin. Every membership
   test must now assert `RolesForActor` to catch projection bugs.

2. **Payload structs should be self-contained.** Every event payload that a
   projection needs to process should carry all required fields. Don't rely on
   looking up aggregate state in a projection — that breaks the projection's
   independence and creates the exact bug we just fixed.

3. **Extract complex switch cases early.** The gocognit issue would have been
   caught earlier if we'd extracted `handleMembershipEvent` in the original
   commit. Complexity breeds bugs.

4. **Consider a shared `removeAllRolesInDomain` on Authz.** The pattern of
   "look up current roles, remove each one" is now in 3 places (RolesUpdated,
   MemberRolesChanged, MemberRemoved). It should be a single Authz method.

---

## F. Top 25 Things to Get Done Next

Sorted by **impact / effort ratio** (highest first).

| #  | Task                                                           | Impact | Effort | Notes                            |
| -- | -------------------------------------------------------------- | ------ | ------ | -------------------------------- |
| 1  | **Stamp SchemaVersion in ALL deciders**                        | 8      | 10min  | Consistency fix                  |
| 2  | **Add `RemoveAllRolesInDomain` to Authz**                      | 7      | 15min  | DRY: 3 callers share pattern     |
| 3  | **Update Session struct: ActorID + Origin**                    | 9      | 30min  | Impersonation foundation         |
| 4  | **Update SessionStore interface**                              | 7      | 30min  | Session lifecycle                |
| 5  | **Implement BeginImpersonation/EndImpersonation**              | 9      | 60min  | Core feature                     |
| 6  | **Update session middleware for actor+impersonator**           | 8      | 45min  | HTTP integration                 |
| 7  | **Create Tenant aggregate** (state+events+commands+fold)       | 7      | 90min  | Multi-tenancy                    |
| 8  | **Wire Tenant aggregate in Service**                           | 6      | 30min  | Service integration              |
| 9  | **Create Bot aggregate + HMAC-SHA256 pepper**                  | 7      | 60min  | Machine identity                 |
| 10 | **API token authentication middleware**                        | 6      | 45min  | Bot auth                         |
| 11 | **Write v1→v2 upcaster for RolesUpdatedPayload**               | 8      | 30min  | Migration support                |
| 12 | **Migration projection: Roles → Membership**                   | 8      | 60min  | Backward compat                  |
| 13 | **Remove Roles from UserState**                                | 7      | 45min  | Breaking change, needs migration |
| 14 | **Tenant read model + queries**                                | 5      | 30min  | Tenant management UI             |
| 15 | **Membership Service methods** (AddMember, RemoveMember, etc.) | 8      | 45min  | Public API                       |
| 16 | **Integration test: impersonation → event metadata**           | 9      | 30min  | Audit trail verification         |
| 17 | **Integration test: cross-tenant isolation**                   | 8      | 30min  | Security verification            |
| 18 | **Property-based tests for foldMembership**                    | 6      | 30min  | Invariant verification           |
| 19 | **Catalog integration** (new events/commands)                  | 4      | 30min  | API docs                         |
| 20 | **Update README examples with ActorID/Membership**             | 4      | 20min  | Docs                             |
| 21 | **Fix RolesUpdated domain fallback hack**                      | 5      | 15min  | Pre-existing tech debt           |
| 22 | **Add Service.AddMember/UpdateMemberRoles/RemoveMember**       | 8      | 30min  | Public Service API               |
| 23 | **Add MembershipHTTPHandler** (REST endpoints)                 | 5      | 45min  | HTTP API for membership          |
| 24 | **Bot scope → Casbin policy mapping**                          | 6      | 30min  | Fine-grained bot authz           |
| 25 | **Pepper rotation strategy** (dual-pepper window)              | 3      | 30min  | Ops documentation                |

---

## G. Top Question I Cannot Figure Out Myself

**Should the Service expose membership operations directly, or should there be a separate `MembershipService`?**

The `Service` struct already has 27+ fields. Adding membership methods to it would
make it even larger. But a separate service would need access to the same dispatcher,
authz, and read model — all of which are already wired into `Service`.

Options:

- **A) Methods on Service**: `svc.AddMember(actorID, tenantID, roles)` — simplest,
  but Service grows
- **B) Separate MembershipService**: wraps a `*Service` reference, accesses its
  dispatcher + authz + membershipReadModel — cleaner separation
- **C) Functional package-level API**: `membership.AddMember(svc, actorID, tenantID, roles)`
  — no new type, but less discoverable

My recommendation: **Option A** for now (consistency with existing User methods),
extract to **Option B** when Service exceeds ~40 fields.

---

## Build/Test/Lint Status

```
Root:          ok   (2.3s, -race)
usermgmt:      ok   (2.5s, -race)
integration:   ok   (1.0s, -race)
Lint:          0 issues (both modules)
Tests:         540+ (26 new identity, 4 membership integration with Casbin assertions)
```

## Commit History

| Commit  | Description                                                            |
| ------- | ---------------------------------------------------------------------- |
| 0db9379 | Tier 0: Keystone types (ActorID, TenantID, BotID, g2 hierarchy)        |
| 573e2e4 | Tier 1: Membership aggregate (events, commands, state, fold)           |
| b99ca78 | Tier 2: Context propagation (WithActorID, EventOptionsFromContext)     |
| 0f06f34 | Membership wiring (decider, dispatch, projection, read model, service) |
| 9c254c3 | ADR 0015 + AGENTS.md update                                            |
| 85ac594 | **Bugfix: CasbinProjection membership bugs + Casbin test assertions**  |
