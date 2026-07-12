# ADR 0006: Event-Sourced User Aggregate

**Date:** 2026-06-15
**Status:** Accepted

> **Update 2026-06-16 (Passwordless migration):** All password code was removed.
> The `ChangePassword` command, `PasswordChanged` event, and bcrypt hashing are gone.
> Authentication is exclusively via WebAuthn (passkeys). The aggregate now has
> **12 events** and **11 commands** (see `es_constants.go`). Credential-related
> events (`CredentialAdded`, `CredentialRemoved`, `EmailVerified`, `TOTPEnabled`,
> `TOTPDisabled`, `ExternalAccountLinked`, `ExternalAccountUnlinked`) were added.
> The original 6-event/6-command design below is preserved for historical context.
>
> **Update 2026-06-22 (v3 migration):** `memory.MemoryBus` was replaced by
> `watermill.EventBus` (GoChannel backend) — see ADR-0016. Module paths moved
> from `event/v2` to `event/v4`.

## Context

The `usermgmt` module stored users via `InMemoryUserStore` — a CRUD `map[UserID]*User` with
fire-and-forget domain events. Problems:

1. **No audit trail** — user state changes are overwritten, not recorded.
2. **Dual-write inconsistency** — roles stored in both `User.Roles` and Casbin policies. The
   `UpdateRoles` method mutates the user before applying Casbin changes; a failure mid-way leaves
   them out of sync.
3. **Manual rollback** — `Register` compensates failures by deleting the user on rollback,
   which is fragile under partial failures.
4. **No replay** — the system cannot be rebuilt from its event history.
5. **No temporal queries** — "what was the user's state at time T?" is impossible.

`go-cqrs-lite` provides full first-class event sourcing support: `Decider[State]`,
`Repository[State]`, `event.Store` with optimistic concurrency, projections with checkpointing,
and snapshots. The `usermgmt` module already depends on `event/v4` for error types.

## Decision

Transform the `User` aggregate into a fully event-sourced aggregate using the Decider pattern.

### Architecture

```
HTTP → Service → CommandDispatcher → DeciderRepository.Execute
  → Load events from Event Store
  → Fold events through pure foldUser() → UserState
  → Call pure decide() function
  → Save new events (optimistic concurrency)
  → Publish events to Event Bus
    → UserReadModel projection (query side)
    → CasbinProjection (authorization)
```

### Aggregate Design

**One aggregate: `User`.** One event stream per user, keyed by `id.AggregateID`.

6 events: `UserRegistered`, `PasswordChanged`, `RolesUpdated`, `EmailChanged`,
`DisplayNameChanged`, `UserDeleted` (tombstone).

6 commands: `RegisterUser`, `ChangePassword`, `UpdateRoles`, `ChangeEmail`,
`ChangeDisplayName`, `DeleteUser`.

### Key Design Choices

1. **Password hashing in the Service layer** — commands carry bcrypt hashes, not plaintext.
   Keeps decide functions fast and testable. Password verification (old password check) is an
   application concern, not a domain invariant.

2. **Casbin as a projection** — all Casbin policies are derived from events. The
   `CasbinProjection` subscribes to `UserRegistered`/`RolesUpdated`/`UserDeleted` events and
   maintains group policies. Single source of truth = event store.

3. **Sessions are NOT event-sourced** — sessions are ephemeral auth artifacts. The
   `SessionStore` interface and `InMemorySessionStore` implementation remain unchanged.

4. **Login is NOT a command** — login is a query (read model lookup) + session side-effect.
   `UserLoggedInEvent` is published on the bus for audit/notification but NOT on the User
   aggregate stream.

5. **Read-your-writes consistency** — `memory.MemoryBus` blocks publishers until all handlers
   complete. Projections update synchronously before `Execute()` returns.

6. **UserID bridge** — `usermgmt.UserID` (branded type) ↔ `id.AggregateID` (string) conversion
   at the Service layer boundary. Domain code uses `id.AggregateID`.

## Consequences

- **Breaking change**: `ServiceConfig` changes, `UserStore` interface removed, `User` mutation
  methods removed. Consumers must update config and stop mutating `User` directly.
- The event store becomes the single source of truth. All read models are projections.
- Authorization is fully derived from events — replay rebuilds the entire policy set.
- Future schema evolution requires upcasters (not needed for v1).
