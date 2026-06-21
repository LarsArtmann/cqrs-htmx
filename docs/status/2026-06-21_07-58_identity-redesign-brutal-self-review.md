# Status Report — Identity Model Redesign

**Date:** 2026-06-21 07:58 | **Author:** Crush | **Branch:** master

**Scope:** Actor/Tenant/Membership/Impersonation redesign (Pareto plan at
`docs/planning/2026-06-21_07-31_identity-model-redesign-pareto.md`)

---

## Executive Summary

**3 of 7 tiers implemented. The keystone types exist but are largely unwired.**
The type definitions are solid, but the redesign stopped at "types compile" and
did not reach "types are used by the system." Several methods are dead code,
the Membership aggregate has no lifecycle wiring, and the CasbinProjection still
ignores membership events entirely.

**Verdict:** The foundation is poured but the house has no plumbing.

---

## A. FULLY DONE (working, tested, verified)

| Item | Commit | Tests | Lint |
|------|--------|-------|------|
| ActorID struct (kind + raw + Kind/String/PrefixedString/IsZero) | 0db9379 | 5 | ✅ |
| TenantID + BotID branded types | 0db9379 | 2 | ✅ |
| SessionOrigin sealed interface (DirectLogin, Impersonation) | 0db9379 | 2 | ✅ |
| Membership read-model struct (HasRole, HasAnyRole) | 0db9379 | 2 | ✅ |
| RoleSuperAdmin + Casbin g2 hierarchy (super_admin > admin > user > viewer) | 0db9379 | 2 | ✅ |
| ActorKind enum with String() | 0db9379 | 1 | ✅ |
| Membership event payloads (MemberAdded, MemberRolesChanged, MemberRemoved) | 573e2e4 | 0 direct | ✅ |
| Membership commands (AddMemberCmd, UpdateMemberRolesCmd, RemoveMemberCmd) | 573e2e4 | 2 | ✅ |
| MembershipState + foldMembership (pure function) | 573e2e4 | 4 | ✅ |
| Context: WithActorID/ActorFromContext, WithImpersonatorID/ImpersonatorFromContext | b99ca78 | 4 | ✅ |
| EventOptionsFromContext propagates actor_id + impersonator_id | b99ca78 | 2 | ✅ |
| DomainsForUser filters empty domains (g2 artifact fix) | 0db9379 | existing | ✅ |

**Total: 26 new tests, all passing. 0 lint issues. Build green across all 4 modules.**

---

## B. PARTIALLY DONE (types exist but wiring is incomplete)

### B1. Membership Aggregate — NO Lifecycle Wiring

The Membership aggregate has events, commands, state, and fold — but **none of it
is connected to the dispatch system**:

- `RegisterCommands()` (`es_dispatch.go`) does NOT register `AddMemberCmd`,
  `UpdateMemberRolesCmd`, or `RemoveMemberCmd`. No decider functions exist for
  membership commands.
- `foldMembership` is never called by any repository or projection.
- No `decider.Repository[MembershipState]` exists.
- No read-model projection for memberships exists.

**Status:** The aggregate is defined but inert. Dispatching a membership command
would fail with "unknown command type."

### B2. CasbinProjection — Ignores Membership Events

`CasbinProjection.EventTypes()` returns only User event types. The membership
events (`eventMemberAdded`, `eventMemberRolesChanged`, `eventMemberRemoved`)
are declared in `allMembershipEventTypes` (with a `//nolint:unused` comment
claiming "wired into CasbinProjection in Tier 2" — **this is false**).

**Status:** Membership events have zero effect on authorization policies.

### B3. Context Actor Chain — Uses Raw Strings

`WithActorID(ctx, actorID string)` takes a raw string, not a typed ActorID.
This is because the root module can't import usermgmt (clean module boundary).
The tradeoff is documented but the type safety gap remains.

**Status:** Works functionally, but any string can be stored as "actor ID."

### B4. Schema Version Bump — No Upcasters

`currentSchemaVersion` was bumped to 2, but:
- No v1→v2 upcaster functions are registered (only test fixtures exist).
- `RolesUpdatedPayload.Domain` is still `string`, not `TenantID` (the comment
  in es_constants.go claims it was changed — it wasn't).
- Core deciders (`decideRegisterUser`, `decideUpdateRoles`) don't stamp
  `SchemaVersion: currentSchemaVersion` on their payloads.

**Status:** Schema version is cosmetic — no migration logic backs it.

---

## C. NOT STARTED

| Task | Description |
|------|-------------|
| Session struct update | `Session.UserID` → `Session.ActorID`, add `Origin SessionOrigin` |
| SessionStore interface update | Accept Session struct, not just UserID |
| BeginImpersonation / EndImpersonation | Service methods + security guards |
| Session middleware update | Extract actor + impersonator from session, inject into context |
| Tenant aggregate | State, events, commands, fold, decider, wiring |
| Bot aggregate | State, events, commands, HMAC-SHA256 pepper token hash |
| API token middleware | Bearer token → Bot resolution |
| Migration projection | Replay old UserState.Roles → Membership events |
| Decide functions for membership commands | `decideAddMember`, `decideUpdateMemberRoles`, `decideRemoveMember` |
| Membership read model | `MembershipReadModel` projection for queries |
| Remove Roles from UserState | Breaking change deferred until Membership is fully wired |
| EventOptionsFromContext in usermgmt | Root has it; usermgmt doesn't use it yet |

---

## D. TOTALLY FUCKED UP (bugs, design errors, honesty)

### D1. `RolesForActor` is type-unsafe AND dead code

```go
// authz_roles.go:80
func (a *Authz) RolesForActor(actorID string, tenantID TenantID) ([]Role, error) {
    return a.RolesForUser(NewUserID(actorID), tenantID.Get())  // BUG: creates UserID from raw string
}
```

**Bug:** `NewUserID(actorID)` wraps any string as a UserID. If a bot ID is passed,
it's silently cast to UserID — the exact type confusion the ActorID redesign was
supposed to prevent. Should take `ActorID` type, not raw `string`.

**Dead code:** Zero callers anywhere in the codebase. Same for `ImplicitRolesForActor`.

### D2. MembershipState / Membership split brain

Two structs model the same domain concept:
- `MembershipState` (es_membership_state.go) — aggregate state, has `Removed bool`
- `Membership` (user.go) — read model, has `AddedAt time.Time`

Both have `HasRole()` — duplicated logic. They share no interface. Consumers must
know which to use.

### D3. `allMembershipEventTypes` nolint comment is a lie

```go
//nolint:unused // wired into CasbinProjection in Tier 2
```

It is NOT wired. The comment is misleading. Should either wire it or remove it.

### D4. es_constants.go comment claims Domain→TenantID change that didn't happen

```go
// v2 adds: TenantID on RolesUpdatedPayload, Membership aggregate events.
```

`RolesUpdatedPayload` still has `Domain string`. The comment is wrong.

### D5. Stale test comment about schema version

```go
// es_upcaster_test.go:70
// Future v1→v2 upcaster (not yet active since currentSchemaVersion=1)
```

`currentSchemaVersion` is now 2. The comment is stale.

---

## E. WHAT WE SHOULD IMPROVE

### E1. Stop adding types without wiring them

Every new type should be wired end-to-end in the same commit: type → decider →
dispatch → projection → test. An unwired type is dead code with extra steps.

### E2. Resolve the MembershipState/Membership split brain

Either:
- Make `Membership` the read-model projection OF `MembershipState` (one is the
  source of truth, the other is the query model), OR
- Have only ONE struct with both `Removed` and `AddedAt` fields.

### E3. Use typed ActorID in Authz methods

`RolesForActor` should take `ActorID`, not `string`. The conversion to Casbin's
string-based subject should happen inside the method, not at the call site.

### E4. Write the ADR BEFORE implementing, not after

ADR 0015 should have been written before any code. It would have caught the
split brain and the wiring gaps before they became commits.

### E5. Consider using go-cqrs-lite's multi-aggregate patterns

go-cqrs-lite v2.6.0 may have patterns for managing multiple aggregate types
(User, Membership, Tenant, Bot) in the same service. Check before hand-rolling.

### E6. Consider well-established libraries

- **HMAC token hashing:** `crypto/hmac` + `crypto/sha256` (stdlib) is correct.
  No external lib needed.
- **Pepper management:** Consider HashiCorp Vault or AWS KMS for pepper storage
  in production. Document the pattern; don't implement the KMS client.
- **Role hierarchy:** Casbin's g2 is the right choice. Already implemented.

### E7. Consider a Tenant-owned Membership model

Instead of Membership as a separate aggregate with a derived ID, consider making
Memberships part of the Tenant aggregate. `AddMember` would be a command on
Tenant, not on Membership. This simplifies the aggregate ID problem and ensures
tenant-scoped consistency.

---

## F. Top 25 Things to Get Done Next

Sorted by **impact / effort ratio** (highest first).

| # | Task | Impact | Effort | Notes |
|---|------|--------|--------|-------|
| 1 | **Fix RolesForActor to take ActorID, not string** | 10 | 5min | Type-safety bug fix |
| 2 | **Remove dead `allMembershipEventTypes` nolint** — wire it or remove it | 9 | 5min | Dead code cleanup |
| 3 | **Fix stale comments** (es_constants.go, es_upcaster_test.go) | 7 | 5min | Truth in documentation |
| 4 | **Update AGENTS.md** with new types, file list, event/command counts | 8 | 15min | Docs freshness |
| 5 | **Write ADR 0015** for identity model redesign | 9 | 30min | Process compliance |
| 6 | **Resolve MembershipState/Membership split brain** | 9 | 30min | Architectural cleanup |
| 7 | **Register membership commands in RegisterCommands** | 10 | 45min | Makes aggregate functional |
| 8 | **Write decide functions for membership commands** | 10 | 45min | Pure domain logic |
| 9 | **Wire membership events into CasbinProjection** | 10 | 60min | Makes authz work |
| 10 | **Create MembershipReadModel projection** | 8 | 45min | Query support |
| 11 | **Wire Membership repository in Service** | 9 | 30min | Service integration |
| 12 | **Stamp SchemaVersion in ALL deciders** | 7 | 15min | Consistency |
| 13 | **Write v1→v2 upcaster for RolesUpdatedPayload** | 8 | 30min | Migration support |
| 14 | **Update Session struct: ActorID + Origin** | 8 | 30min | Impersonation foundation |
| 15 | **Update SessionStore interface** | 7 | 30min | Session lifecycle |
| 16 | **Implement BeginImpersonation/EndImpersonation** | 9 | 60min | Core feature |
| 17 | **Update session middleware for actor+impersonator** | 8 | 45min | HTTP integration |
| 18 | **Create Tenant aggregate** (state+events+commands+fold) | 7 | 90min | Multi-tenancy |
| 19 | **Consider Tenant-owned Membership** (refactor from separate aggregate) | 8 | 60min | Simpler model |
| 20 | **Create Bot aggregate + HMAC pepper hash** | 7 | 60min | Machine identity |
| 21 | **API token authentication middleware** | 6 | 45min | Bot auth |
| 22 | **Migration projection: Roles → Membership** | 8 | 60min | Backward compat |
| 23 | **Integration tests for full membership lifecycle** | 9 | 45min | Verify wiring |
| 24 | **Update existing tests for ActorID types** | 7 | 45min | Test correctness |
| 25 | **Catalog integration** (new events/commands in catalog) | 4 | 30min | API docs |

---

## G. Top Question I Cannot Figure Out Myself

**Should Membership be its own aggregate, or should it be part of the Tenant aggregate?**

This is the single biggest architectural decision remaining, and it affects
everything downstream:

**Option A: Membership as separate aggregate (current design)**
- Pros: Independent lifecycle, can add/remove members without loading Tenant state
- Cons: Derived aggregate ID (`DeriveAggregateID("membership", actorID, tenantID)`),
  no transactional consistency with Tenant, need separate read model
- Risk: Two aggregates can disagree (Tenant says "active", Membership says "removed")

**Option B: Membership as part of Tenant aggregate**
- Pros: Transactional consistency (add member + update tenant in one event),
  natural aggregate boundary, simpler ID (just tenant ID)
- Cons: Loading a Tenant with 10,000 members loads all membership state,
  Tenant aggregate becomes large
- Risk: Performance degradation for large tenants

**Option C: Membership as part of User aggregate**
- Pros: Simple — user carries their memberships
- Cons: Can't query "who are the members of tenant X?" without scanning all users,
  couples user identity to tenant membership (the exact decoupling we wanted)

**My recommendation:** Option B (Tenant-owned), because:
1. Transactional consistency is more valuable than independent lifecycle
2. Large-tenant performance can be solved with snapshotting or separate read models
3. The aggregate ID problem disappears
4. It aligns with how the Casbin domain model works (domain = tenant, policies scoped within)

**BUT** — this contradicts the brainstorming report's design. The user should decide
before I implement Tier 3+.

---

## Commit History

| Commit | Description |
|--------|-------------|
| 0db9379 | Tier 0: Keystone types (ActorID, TenantID, BotID, SessionOrigin, Membership, g2) |
| 573e2e4 | Tier 1: Membership aggregate (events, commands, state, fold) |
| b99ca78 | Tier 2: Context propagation (WithActorID, EventOptionsFromContext actor chain) |

---

## Build/Test/Lint Status

```
Root:          ok   (2.3s, -race)
usermgmt:      ok   (2.6s, -race)
integration:   ok   (1.0s, -race)
Lint:          0 issues (both modules)
Coverage:      not re-measured (new code has tests but no coverage report)
```
