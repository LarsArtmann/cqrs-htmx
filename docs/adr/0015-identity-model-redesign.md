# ADR 0015: Identity Model Redesign — Actor, Tenant, Membership

**Date:** 2026-06-21
**Status:** Accepted
**Supersedes:** Part of the event-sourced aggregate design (ADR 0006)

## Context

The original `usermgmt` module had 7 foundational identity model gaps:

1. **No Actor abstraction** — `UserID` is the only identity type. No union of user | bot.
2. **No TenantID** — Casbin `domain` is a raw string with no lifecycle.
3. **No role hierarchy** — Single flat `admin` role, no super_admin inheritance.
4. **No impersonation** — Sessions lack origin tracking.
5. **Roles on User** — Roles are part of UserState, not a Membership entity.
6. **No Bot authentication** — No API tokens, no machine identity.
7. **No actor chain in events** — Event metadata has no impersonator tracking.

These gaps block multi-tenancy, machine-to-machine auth, and compliance audit trails.

## Decision

### 1. Actor as a kind-discriminated struct (not sealed interface)

```go
type ActorID struct {
    kind ActorKind  // ActorUser | ActorBot
    raw  string
}
```

**Rationale:** Go doesn't have sealed types. A struct with a kind discriminator
is the Go-idiomatic approach. `ActorIDFromUser()` and `ActorIDFromBot()` are
the only constructors. `PrefixedString()` produces audit-friendly `"user:01JX..."`
format for event metadata.

### 2. Membership as an INDEPENDENT aggregate

```go
type MembershipState struct {
    ActorID  ActorID
    TenantID TenantID
    Roles    []Role
    Removed  bool
}
```

**Rationale:** The user explicitly chose independent aggregate over Tenant-owned
because they want to support **1 member in multiple tenants/orgs**. A separate
aggregate with a derived ID (`DeriveAggregateID("membership", actorID, tenantID)`)
allows each membership to have its own event stream and lifecycle.

This follows the same CQRS read/write split as UserState/User:
- `MembershipState` = aggregate state (write model, used by decider/repository)
- `Membership` = read model (projection, queryable via MembershipReadModel)

### 3. Casbin g2 role hierarchy

```
g2: super_admin > admin
g2: admin > user
g2: user > viewer
```

**Rationale:** Casbin's `g2` role manager provides global role inheritance.
A super_admin inherits all permissions from admin > user > viewer. This is
additive — existing policies are unaffected.

### 4. SessionOrigin sealed interface

```go
type SessionOrigin interface { isSessionOrigin() }
type DirectLogin struct { AuthenticatedAs ActorID }
type Impersonation struct { By ActorID; Reason string; At time.Time }
```

**Rationale:** Makes impossible states unrepresentable. A boolean `IsImpersonated`
field could have `true` with `nil` ImpersonatorID. The sealed interface prevents this.

### 5. Event store IS the audit trail

No separate audit events. `EventOptionsFromContext` propagates `actor_id` and
`impersonator_id` into every event's custom metadata. Audit = SQL query on the
events table's JSONB metadata column.

### 6. HMAC-SHA256 + pepper for bot tokens (planned)

API tokens are 256-bit random — brute force is infeasible. The threat is DB-leak
defense. Pepper (server-side secret outside DB) achieves this at ~0 cost per
request. Argon2id's 10-100ms per bot API call is unjustified.

## Consequences

### Positive

- Multi-tenant RBAC works: same actor can have different roles in different tenants
- Role hierarchy: super_admin inherits everything, no policy duplication
- Audit trail: every event carries the full actor chain for compliance queries
- Type safety: ActorID prevents mixing user/bot IDs at the type level

### Negative

- **Schema v2**: Events now carry `schema_version: 2`. Old v0/v1 events need
  upcasters (infrastructure exists, production upcasters not yet registered).
- **Public API change**: `RolesForActor` takes `ActorID`, not `string`. This is
  a breaking change for any consumer using the old signature.
- **Membership aggregate adds complexity**: A second decider repository,
  read model, and projection must be maintained alongside the User aggregate.

### Neutral

- `RolesUpdatedPayload.Domain` remains `string` for backward compatibility.
  Future migration to `TenantID` requires an upcaster.

## Implementation Status

| Component | Status | Commit |
|-----------|--------|--------|
| ActorID, TenantID, BotID types | Done | 0db9379 |
| SessionOrigin sealed interface | Done | 0db9379 |
| Casbin g2 role hierarchy | Done | 0db9379 |
| Membership events/commands/state/fold | Done | 573e2e4 |
| Context actor chain propagation | Done | b99ca78 |
| MembershipDecider + decide functions | Done | 0f06f34 |
| RegisterMembershipCommands | Done | 0f06f34 |
| MembershipReadModel projection | Done | 0f06f34 |
| CasbinProjection membership events | Done | 0f06f34 |
| Service/Setup wiring | Done | 0f06f34 |
| Session struct update (ActorID + Origin) | Planned | — |
| BeginImpersonation/EndImpersonation | Planned | — |
| Tenant aggregate | Planned | — |
| Bot aggregate + HMAC tokens | Planned | — |
| Schema v1→v2 upcasters | Planned | — |
| Remove Roles from UserState | Planned | — |

## Related

- [Brainstorming Report](../brainstorming/2026-06-21_actor-tenant-impersonation-redesign.html)
- [Pareto Plan](../planning/2026-06-21_07-31_identity-model-redesign-pareto.md)
- [Wiring Plan](../planning/2026-06-21_08-23_membership-wiring-execution.md)
- [Self-Review](../status/2026-06-21_07-58_identity-redesign-brutal-self-review.md)
- ADR 0006 (event-sourced user aggregate)
