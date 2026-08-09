# Event-Sourced CQRS Migration Plan for usermgmt

> Transform usermgmt from CRUD with fire-and-forget events to fully event-sourced CQRS using go-cqrs-lite's Decider pattern, with Casbin as an event-driven projection.

**Date**: 2026-06-15\
**Status**: PROPOSED\
**Impact**: BREAKING — new module major version (v3)

---

## Table of Contents

1. [Current State Analysis](#1-current-state-analysis)
2. [Target Architecture](#2-target-architecture)
3. [Key Design Decisions](#3-key-design-decisions)
4. [Pareto Breakdown](#4-pareto-breakdown)
5. [Comprehensive Plan (Medium Granularity)](#5-comprehensive-plan-medium-granularity)
6. [Detailed Breakdown (Fine Granularity)](#6-detailed-breakdown-fine-granularity)
7. [Risk Analysis](#7-risk-analysis)
8. [Migration Strategy](#8-migration-strategy)

---

## 1. Current State Analysis

### What Exists Today

| Aspect                 | Current State                                                   | Problem                                                                                            |
| ---------------------- | --------------------------------------------------------------- | -------------------------------------------------------------------------------------------------- |
| **User storage**       | `InMemoryUserStore` — `map[UserID]*User`                        | State store, not event log. No audit trail. No replay.                                             |
| **Role storage**       | Duplicated: `User.Roles` field + Casbin group policies          | Must manually keep in sync. Partial failure risk in `UpdateRoles`.                                 |
| **Session storage**    | `InMemorySessionStore` — `map[string]*Session`                  | Fine — sessions are ephemeral, not domain state.                                                   |
| **Events**             | `EventHandler func(userID, event any)` callback                 | Fire-and-forget. Not persisted. Not ordered. Not versioned.                                        |
| **Transaction safety** | Manual rollback in `Register` (delete user on failure)          | Fragile. `UpdateRoles` mutates user before authz — inconsistency if authz succeeds but save fails. |
| **Domain logic**       | Scattered across `Service` methods + `User` methods             | No separation of command (decide) from query (read).                                               |
| **Read/write models**  | Same `User` struct for persistence, serialization, domain logic | No CQRS separation.                                                                                |

### What's Already Good (Keep As-Is)

- **Authz (Casbin wrapper)** — excellent API, well-tested. Only the _data source_ changes (events instead of direct mutation).
- **SessionStore** — sessions are ephemeral auth artifacts, not domain state. Keep as-is.
- **AccountLockout** — ephemeral security state. Keep as-is.
- **HTTP handlers** — thin layer over Service. Interface stays; internals change.
- **SessionMiddleware** — reads from SessionStore. Unaffected.
- **Errors** — sentinel errors and error→HTTP mapping stay identical.

---

## 2. Target Architecture

```
                    ┌─────────────────────┐
                    │   HTTP Handlers     │
                    │ (register/login/    │
                    │  logout/me)         │
                    └────────┬────────────┘
                             │
                    ┌────────▼────────────┐
                    │      Service        │
                    │ (app orchestration) │
                    └──┬─────┬──────┬─────┘
                       │     │      │
          ┌────────────▼┐  ┌─▼──┐  ┌▼──────────────┐
          │  Command    │  │Query│  │  Session      │
          │ Dispatcher  │  │Disp.│  │  Store        │
          └──────┬──────┘  └──┬──┘  │ (unchanged)   │
                 │            │      └───────────────┘
          ┌──────▼──────┐     │
          │ Decider     │     │
          │ Repository  │     │
          │ [UserState] │     │
          └──┬───────┬──┘     │
   Load     │       │ Save    │
  /Fold     │       │+Publish │
             │       │         │
    ┌────────▼──┐ ┌──▼───────┐ │
    │ Event     │ │ Event    │ │
    │ Store     │ │ Bus      │ │
    │ (source   │ │ (pub/sub)│ │
    │  of truth)│ │         │ │
    └───────────┘ └──┬──┬───┘ │
                     │  │     │
         ┌───────────┘  │     │
         ▼              ▼     │
┌─────────────┐ ┌─────────────▼─┐
│ UserRead    │ │ Casbin        │
│ Model       │ │ Projection    │
│ (projection)│ │ (projection)  │
└──────┬──────┘ └───────┬───────┘
       │                │
       ▼                ▼
┌─────────────┐ ┌───────────────┐
│ FindByID    │ │ Casbin        │
│ FindByEmail │ │ Enforcer      │
│ (query side)│ │ (authz checks)│
└─────────────┘ └───────────────┘
```

### Write Path (Command)

```
HTTP → Service → Command.Dispatch → DeciderRepo.Execute
  → Load events for User aggregate from Event Store
  → Fold events through pure foldUser() → UserState
  → Call pure decide() function with state + command intent
  → decide() returns new events (or error)
  → Save events to Event Store (optimistic concurrency)
  → Publish events to Event Bus
    → UserReadModel projection updates (replaces UserStore)
    → CasbinProjection updates Casbin policies
```

### Read Path (Query)

```
HTTP → Service → UserReadModel.FindByID/FindByEmail → *User → Response
```

### Login Path (Not Event-Sourced)

```
HTTP → Service.Login
  → UserReadModel.FindByEmail → get aggregate ID + password hash
  → Verify password (bcrypt compare)
  → Check lockout
  → SessionStore.Create → session token
  → Publish UserLoggedInEvent on bus (audit/notification only)
  → Return session
```

---

## 3. Key Design Decisions

### 3.1 What Becomes Event-Sourced

| Concern                                                | Event-Sourced? | Rationale                                                                |
| ------------------------------------------------------ | -------------- | ------------------------------------------------------------------------ |
| **User aggregate** (email, name, password hash, roles) | YES            | Core domain entity. All state changes are events.                        |
| **Casbin policies**                                    | PROJECTION     | Policies derived from UserRegistered + RolesUpdated events.              |
| **Sessions**                                           | NO             | Ephemeral auth artifacts. Lost on restart is acceptable.                 |
| **Lockout**                                            | NO             | Ephemeral security state. Not domain state.                              |
| **Login events**                                       | NO (bus-only)  | Published on bus for audit but NOT on User stream. Keeps aggregate lean. |

### 3.2 Aggregate Boundary

**One aggregate: `User`.** One event stream per user, keyed by `id.AggregateID`.

Sessions, Casbin, and lockout are separate concerns outside the aggregate boundary.

### 3.3 Events (6 total)

| Event                | Payload                                   | When             |
| -------------------- | ----------------------------------------- | ---------------- |
| `UserRegistered`     | email, display_name, password_hash, roles | User creation    |
| `PasswordChanged`    | password_hash                             | Password change  |
| `RolesUpdated`       | roles[], domain                           | Role replacement |
| `EmailChanged`       | email                                     | Email update     |
| `DisplayNameChanged` | display_name                              | Name update      |
| `UserDeleted`        | reason (tombstone)                        | User deletion    |

### 3.4 Commands (6 total)

| Command             | Decide Function Guards                                 | Events Emitted            |
| ------------------- | ------------------------------------------------------ | ------------------------- |
| `RegisterUser`      | User must not exist (state.Email == "")                | `UserRegistered`          |
| `ChangePassword`    | User must exist + not deleted                          | `PasswordChanged`         |
| `UpdateRoles`       | User must exist + not deleted                          | `RolesUpdated`            |
| `ChangeEmail`       | User must exist + not deleted + email actually changed | `EmailChanged`            |
| `ChangeDisplayName` | User must exist + not deleted + name changed           | `DisplayNameChanged`      |
| `DeleteUser`        | User must exist + not deleted                          | `UserDeleted` (tombstone) |

### 3.5 Password Handling

- **New password hashing**: Done in the Service layer BEFORE dispatching the command. Commands carry bcrypt hashes, not plaintext passwords. Keeps command log safe.
- **Old password verification**: Done in the Service layer against the loaded aggregate state (or read model). This is an authentication concern, not a domain invariant. The decide function only checks "user exists + not deleted".
- **Rationale**: Keeping bcrypt (~250ms per op) out of the decide function keeps the pure domain logic fast and testable. Password verification is an application-layer concern.

### 3.6 Casbin as Projection

The Casbin projection subscribes to:

- `UserRegistered` → `AddGroupPolicy(subject, role, domain)` for each role
- `RolesUpdated` → Remove old group policies, add new ones for the domain
- `UserDeleted` → Remove all group policies for the user

Casbin policy state is **fully derived from events**. On replay, the projection rebuilds the entire policy set from scratch. The event store is the single source of truth for authorization.

### 3.7 Read-Your-Writes Consistency

With `memory.MemoryBus`, `bus.Publish()` blocks until all handlers complete (projection updates finish). By the time `deciderRepo.Execute()` returns, the read model and Casbin are updated. This gives us **read-your-writes consistency** in single-process deployments.

For production with async message buses, consumers accept eventual consistency — that's inherent to CQRS.

### 3.8 UserID Bridge

```
usermgmt.UserID (brandid.ID[userBrand, string])
    ↕ .Get() / NewUserID()
go-cqrs-lite id.AggregateID (string)
```

Conversion happens at the Service layer boundary. Domain code uses `id.AggregateID`.

### 3.9 Module Version

`usermgmt/v2.0.0` is tagged. This is a breaking change → **bump to v3**:

- Module path: `github.com/larsartmann/cqrs-htmx/usermgmt/v3`
- Directory: `usermgmt/v3/` (Go major version directory convention)

Actually — since we're transforming in place and v2 hasn't been widely adopted (pre-1.0 energy), we could also just evolve the v2 module. **Recommendation: evolve in-place, document as breaking in CHANGELOG, let Lars decide on versioning.**

### 3.10 Dependencies to Add

| New Dependency               | Purpose                                         |
| ---------------------------- | ----------------------------------------------- |
| `go-cqrs-lite/decider/v2`    | Decider[State], Repository[State]               |
| `go-cqrs-lite/memory/v2`     | In-memory EventStore, EventBus, CheckpointStore |
| `go-cqrs-lite/command/v2`    | Command Dispatcher, RegisterTyped               |
| `go-cqrs-lite/query/v2`      | Query Dispatcher, RegisterTyped                 |
| `go-cqrs-lite/projection/v2` | Projection Runner                               |
| `go-cqrs-lite/id/v2`         | AggregateID, EventID                            |

Currently only `event/v2` is a direct dependency.

---

## 4. Pareto Breakdown

### The 1% That Delivers 51% of the Result

**The pure domain layer**: events, commands, state, fold function, decide functions. These 6 files (~400 lines total) ARE the event-sourced domain. Everything else is wiring.

Without these, nothing works. With these, the rest is mechanical plumbing.

### The 4% That Delivers 64% of the Result

**Add the Repository wiring**: connect the domain to go-cqrs-lite infrastructure (memory store + bus + decider repo). Now you can `Execute` commands and persist events.

### The 20% That Delivers 80% of the Result

**Add projections + Service rewrite**: UserReadModel projection (query side), Casbin projection (authz), and rewrite Service to dispatch commands. HTTP layer stays almost the same.

### The Remaining 80%

Tests, edge cases, documentation, cleanup, lockout integration, session integration, error mapping, examples, benchmarks.

---

## 5. Comprehensive Plan (Medium Granularity)

> Tasks sorted by impact. Each is 30–100 minutes.

| #                                          | Task                                                                                  | Impact   | Effort | Depends On |
| ------------------------------------------ | ------------------------------------------------------------------------------------- | -------- | ------ | ---------- |
| **Phase 1: Domain Core (Pure Functions)**  |                                                                                       |          |        |            |
| 1                                          | Write ADR: Event-Sourced User Aggregate                                               | HIGH     | 30min  | —          |
| 2                                          | Define event type constants + aggregate type                                          | HIGH     | 15min  | —          |
| 3                                          | Define event payload structs (6 events)                                               | HIGH     | 30min  | 2          |
| 4                                          | Define command structs (6 commands)                                                   | HIGH     | 30min  | 2          |
| 5                                          | Define `UserState` struct + `foldUser()` pure function                                | CRITICAL | 45min  | 3          |
| 6                                          | Write decide functions (6 commands) — pure domain logic                               | CRITICAL | 60min  | 4,5        |
| 7                                          | Write unit tests for `foldUser()` — table-driven, every event type                    | CRITICAL | 45min  | 5          |
| 8                                          | Write unit tests for decide functions — guards, validation, event creation            | CRITICAL | 60min  | 6          |
| **Phase 2: Infrastructure Wiring**         |                                                                                       |          |        |            |
| 9                                          | Add new dependencies to `go.mod` (decider, memory, command, query, projection, id)    | HIGH     | 15min  | —          |
| 10                                         | Create `Setup()` function — wires memory store + bus + decider repo                   | HIGH     | 45min  | 9,5        |
| 11                                         | Create command dispatcher registration (`RegisterTyped` for all 6 commands)           | HIGH     | 45min  | 10,6       |
| 12                                         | Test infrastructure wiring — dispatch a RegisterUser command, verify events saved     | HIGH     | 30min  | 11         |
| **Phase 3: Projections**                   |                                                                                       |          |        |            |
| 13                                         | Create `UserReadModel` projection (replaces UserStore) — Name, EventTypes, Handle     | CRITICAL | 60min  | 3          |
| 14                                         | Add email index to UserReadModel (FindByEmail support)                                | HIGH     | 30min  | 13         |
| 15                                         | Create `CasbinProjection` — updates Casbin policies from events                       | HIGH     | 60min  | 3          |
| 16                                         | Wire projection Runner — replay from journal + live subscription                      | HIGH     | 30min  | 13,15      |
| 17                                         | Test projections — replay events, verify read model + Casbin state                    | HIGH     | 45min  | 16         |
| **Phase 4: Service Layer Rewrite**         |                                                                                       |          |        |            |
| 18                                         | Rewrite `ServiceConfig` — accept EventStore, EventBus, decider.Repository             | CRITICAL | 45min  | 10         |
| 19                                         | Rewrite `Service.Register()` → hash password, dispatch RegisterUser, create session   | CRITICAL | 60min  | 11         |
| 20                                         | Rewrite `Service.ChangePassword()` → verify old, hash new, dispatch ChangePassword    | CRITICAL | 45min  | 11         |
| 21                                         | Rewrite `Service.UpdateRoles()` → dispatch UpdateRoles (Casbin updated by projection) | CRITICAL | 45min  | 11         |
| 22                                         | Rewrite `Service.Login()` → query read model, verify password, create session         | CRITICAL | 45min  | 13         |
| 23                                         | Rewrite `Service.GetUser()` → query UserReadModel                                     | HIGH     | 15min  | 13         |
| 24                                         | Add new methods: `ChangeEmail()`, `ChangeDisplayName()`, `DeleteUser()`               | MEDIUM   | 45min  | 11         |
| 25                                         | Rewrite `Service.Authenticate()` → session check + read model lookup                  | HIGH     | 30min  | 13         |
| **Phase 5: Session & Lockout Integration** |                                                                                       |          |        |            |
| 26                                         | Wire SessionStore into new Service (unchanged interface)                              | MEDIUM   | 30min  | 19         |
| 27                                         | Wire AccountLockout into new Service.Login                                            | MEDIUM   | 30min  | 22         |
| 28                                         | Publish UserLoggedInEvent on bus (not on aggregate stream)                            | LOW      | 15min  | 22         |
| **Phase 6: HTTP Layer**                    |                                                                                       |          |        |            |
| 29                                         | Update HTTP handlers to match new Service API (minimal changes)                       | MEDIUM   | 30min  | 19-25      |
| 30                                         | Update error→HTTP status mapping for new error types                                  | LOW      | 15min  | 29         |
| **Phase 7: Test Rewrite**                  |                                                                                       |          |        |            |
| 31                                         | Rewrite Service tests — all methods, happy + error paths                              | CRITICAL | 120min | 19-25      |
| 32                                         | Rewrite handler tests — all routes, cookies, error codes                              | HIGH     | 90min  | 29         |
| 33                                         | Add integration test: full event-sourced flow (register→login→change→delete)          | HIGH     | 60min  | 31         |
| 34                                         | Add projection consistency test: write event, verify read model + Casbin              | HIGH     | 45min  | 17         |
| 35                                         | Update fuzz tests for new validation paths                                            | LOW      | 30min  | 31         |
| 36                                         | Update benchmarks for new architecture                                                | LOW      | 30min  | 31         |
| **Phase 8: Cleanup & Documentation**       |                                                                                       |          |        |            |
| 37                                         | Remove old CRUD code (InMemoryUserStore, old Service methods)                         | HIGH     | 30min  | 31         |
| 38                                         | Update `events.go` — old EventHandler callback replaced by bus subscriptions          | MEDIUM   | 30min  | 28         |
| 39                                         | Update AGENTS.md — new architecture, dependencies, patterns                           | HIGH     | 45min  | 37         |
| 40                                         | Update CHANGELOG.md — breaking changes, migration notes                               | HIGH     | 30min  | 37         |
| 41                                         | Update examples and example_test.go                                                   | MEDIUM   | 45min  | 37         |
| 42                                         | Verify: full build, all tests pass, lint clean, coverage maintained                   | CRITICAL | 30min  | ALL        |

---

## 6. Detailed Breakdown (Fine Granularity)

> Each task is ≤15 minutes. Sorted by execution order within phases.

### Phase 1: Domain Core

| #    | Task                                                                                                 | Est |
| ---- | ---------------------------------------------------------------------------------------------------- | --- |
| 1.1  | Create `docs/adr/0006-event-sourced-user-aggregate.md` — document decision, alternatives, tradeoffs  | 15m |
| 2.1  | Create `es_constants.go` — `aggregateTypeUser`, 6 event type constants, 6 command type constants     | 10m |
| 3.1  | Add `UserRegisteredPayload` struct with JSON tags                                                    | 5m  |
| 3.2  | Add `PasswordChangedPayload` struct                                                                  | 5m  |
| 3.3  | Add `RolesUpdatedPayload` struct                                                                     | 5m  |
| 3.4  | Add `EmailChangedPayload` struct                                                                     | 5m  |
| 3.5  | Add `DisplayNameChangedPayload` struct                                                               | 5m  |
| 3.6  | Add `UserDeletedPayload` struct                                                                      | 5m  |
| 3.7  | Add `marshalPayload()` helper (encode to JSON bytes for event.NewEvent)                              | 10m |
| 4.1  | Add `RegisterUserCmd` struct — fields, `Type()`, `AggregateID()`, constructor                        | 10m |
| 4.2  | Add `ChangePasswordCmd` struct                                                                       | 10m |
| 4.3  | Add `UpdateRolesCmd` struct                                                                          | 10m |
| 4.4  | Add `ChangeEmailCmd` struct                                                                          | 10m |
| 4.5  | Add `ChangeDisplayNameCmd` struct                                                                    | 10m |
| 4.6  | Add `DeleteUserCmd` struct                                                                           | 10m |
| 5.1  | Create `es_state.go` — `UserState` struct (Email, DisplayName, PasswordHash, Roles, Deleted, Exists) | 10m |
| 5.2  | Write `foldUser()` — case for UserRegistered (set all fields)                                        | 10m |
| 5.3  | Write `foldUser()` — case for PasswordChanged (update hash)                                          | 5m  |
| 5.4  | Write `foldUser()` — case for RolesUpdated (replace roles)                                           | 10m |
| 5.5  | Write `foldUser()` — case for EmailChanged (update email)                                            | 5m  |
| 5.6  | Write `foldUser()` — case for DisplayNameChanged (update name)                                       | 5m  |
| 5.7  | Write `foldUser()` — case for UserDeleted (set Deleted=true, tombstone)                              | 10m |
| 5.8  | Write `foldUser()` — default case (ignore unknown, return state)                                     | 5m  |
| 6.1  | Write `decideRegisterUser()` — guard: user must not exist. Create UserRegistered event               | 15m |
| 6.2  | Write `decideChangePassword()` — guard: user exists + not deleted. Create PasswordChanged event      | 10m |
| 6.3  | Write `decideUpdateRoles()` — guard: user exists + not deleted. Create RolesUpdated event            | 10m |
| 6.4  | Write `decideChangeEmail()` — guard: exists + email changed. Create EmailChanged event               | 10m |
| 6.5  | Write `decideChangeDisplayName()` — guard: exists + name changed. Create DisplayNameChanged event    | 10m |
| 6.6  | Write `decideDeleteUser()` — guard: exists + not deleted. Create UserDeleted (mark tombstone)        | 10m |
| 7.1  | Test foldUser: empty events → zero state                                                             | 5m  |
| 7.2  | Test foldUser: UserRegistered → correct state                                                        | 10m |
| 7.3  | Test foldUser: PasswordChanged → hash updated                                                        | 5m  |
| 7.4  | Test foldUser: RolesUpdated → roles replaced                                                         | 10m |
| 7.5  | Test foldUser: EmailChanged → email updated                                                          | 5m  |
| 7.6  | Test foldUser: DisplayNameChanged → name updated                                                     | 5m  |
| 7.7  | Test foldUser: UserDeleted → Deleted=true                                                            | 5m  |
| 7.8  | Test foldUser: multiple events in sequence → correct final state                                     | 10m |
| 7.9  | Test foldUser: unknown event type → state unchanged                                                  | 5m  |
| 8.1  | Test decideRegisterUser: success → 1 event with correct payload                                      | 10m |
| 8.2  | Test decideRegisterUser: user already exists → Conflict error                                        | 10m |
| 8.3  | Test decideRegisterUser: empty email → Rejection error                                               | 10m |
| 8.4  | Test decideChangePassword: success → 1 event                                                         | 10m |
| 8.5  | Test decideChangePassword: user not found → error                                                    | 10m |
| 8.6  | Test decideChangePassword: user deleted → error                                                      | 10m |
| 8.7  | Test decideUpdateRoles: success → 1 event                                                            | 10m |
| 8.8  | Test decideUpdateRoles: user deleted → error                                                         | 10m |
| 8.9  | Test decideDeleteUser: success → tombstone event                                                     | 10m |
| 8.10 | Test decideDeleteUser: already deleted → error                                                       | 10m |

### Phase 2: Infrastructure Wiring

| #    | Task                                                                                                     | Est |
| ---- | -------------------------------------------------------------------------------------------------------- | --- |
| 9.1  | Add decider/v2, memory/v2, command/v2, query/v2, projection/v2, id/v2 to go.mod                          | 10m |
| 9.2  | Run `go mod tidy` — verify all transitive deps resolve                                                   | 5m  |
| 10.1 | Create `es_setup.go` — `EventSourcedConfig` struct (EventStore, EventBus, optional SnapshotStore)        | 10m |
| 10.2 | Create `DefaultSetup()` — returns memory store + bus + decider repo with sensible defaults               | 15m |
| 10.3 | Create `Decider()` helper — returns `decider.Decider[UserState]{Initial: UserState{}, Fold: foldUser}`   | 5m  |
| 10.4 | Create `NewRepository()` — wraps store + bus + decider into `*decider.Repository[UserState]`             | 10m |
| 11.1 | Create `es_commands.go` — `RegisterCommands(d *command.Dispatcher, repo *decider.Repository[UserState])` | 15m |
| 11.2 | Wire RegisterUser → `repo.Execute(ctx, cmd.AggregateID(), aggType, decideRegisterUser(...))`             | 10m |
| 11.3 | Wire ChangePassword → repo.Execute                                                                       | 10m |
| 11.4 | Wire UpdateRoles → repo.Execute                                                                          | 10m |
| 11.5 | Wire ChangeEmail → repo.Execute                                                                          | 10m |
| 11.6 | Wire ChangeDisplayName → repo.Execute                                                                    | 10m |
| 11.7 | Wire DeleteUser → repo.Execute                                                                           | 10m |
| 12.1 | Integration test: create store+bus+repo+dispatcher, dispatch RegisterUser, verify event in store         | 15m |
| 12.2 | Integration test: dispatch ChangePassword after Register, verify 2 events in stream                      | 15m |

### Phase 3: Projections

| #     | Task                                                                                                                            | Est |
| ----- | ------------------------------------------------------------------------------------------------------------------------------- | --- |
| 13.1  | Create `es_readmodel.go` — `UserReadModel` struct with `map[id.AggregateID]*ReadUser` + `map[string]id.AggregateID` email index | 10m |
| 13.2  | Implement `Name()` → `"user-read-model"`                                                                                        | 2m  |
| 13.3  | Implement `EventTypes()` → all 6 event types                                                                                    | 5m  |
| 13.4  | Implement `Handle()` — case UserRegistered: insert user + email index                                                           | 15m |
| 13.5  | Implement `Handle()` — case PasswordChanged: update hash                                                                        | 5m  |
| 13.6  | Implement `Handle()` — case RolesUpdated: replace roles                                                                         | 10m |
| 13.7  | Implement `Handle()` — case EmailChanged: update email + index                                                                  | 10m |
| 13.8  | Implement `Handle()` — case DisplayNameChanged: update name                                                                     | 5m  |
| 13.9  | Implement `Handle()` — case UserDeleted: delete from map + index                                                                | 10m |
| 13.10 | Implement `FindByID(id.AggregateID) (*ReadUser, bool)`                                                                          | 10m |
| 13.11 | Implement `FindByEmail(string) (*ReadUser, bool)`                                                                               | 10m |
| 13.12 | Define `ReadUser` struct — projection-side user representation (same fields as current User but read-only)                      | 10m |
| 14.1  | Add `Count()` method to UserReadModel                                                                                           | 5m  |
| 14.2  | Test email index consistency on email change (old index removed, new added)                                                     | 10m |
| 15.1  | Create `es_casbin_projection.go` — `CasbinProjection` struct wrapping `*Authz`                                                  | 10m |
| 15.2  | Implement `Name()` → `"casbin-projection"`                                                                                      | 2m  |
| 15.3  | Implement `EventTypes()` → UserRegistered, RolesUpdated, UserDeleted                                                            | 5m  |
| 15.4  | Implement `Handle()` — UserRegistered: AddGroupPolicy for each role in user's domain                                            | 15m |
| 15.5  | Implement `Handle()` — RolesUpdated: remove old roles, add new ones for domain                                                  | 15m |
| 15.6  | Implement `Handle()` — UserDeleted: remove all group policies for user                                                          | 15m |
| 15.7  | Test CasbinProjection: register user → verify group policies exist                                                              | 15m |
| 15.8  | Test CasbinProjection: update roles → verify old removed, new added                                                             | 15m |
| 15.9  | Test CasbinProjection: delete user → verify all policies removed                                                                | 10m |
| 16.1  | Create `es_projection_setup.go` — `StartProjections(journal, bus, checkpoint, readModel, casbinProj)`                           | 15m |
| 16.2  | Wire projection.Runner: register UserReadModel + CasbinProjection, start in goroutine                                           | 10m |
| 17.1  | Test: dispatch RegisterUser → verify UserReadModel has the user                                                                 | 10m |
| 17.2  | Test: dispatch RegisterUser → verify CasbinProjection has group policies                                                        | 10m |
| 17.3  | Test: dispatch UpdateRoles → verify read model roles + Casbin policies updated                                                  | 15m |
| 17.4  | Test: dispatch DeleteUser → verify read model entry removed + Casbin cleaned                                                    | 10m |
| 17.5  | Test: replay from journal → verify all projections rebuilt correctly                                                            | 15m |

### Phase 4: Service Layer Rewrite

| #    | Task                                                                                                                                | Est |
| ---- | ----------------------------------------------------------------------------------------------------------------------------------- | --- |
| 18.1 | Rewrite `ServiceConfig` — replace UserStore with EventStore + EventBus; keep SessionStore, Authz, Lockout                           | 15m |
| 18.2 | Rewrite `Service` struct — hold `*decider.Repository[UserState]`, `*command.Dispatcher`, `*UserReadModel`, `*Authz`, `SessionStore` | 15m |
| 18.3 | Rewrite `NewService()` — create default store+bus+repo if not provided, start projections, register commands                        | 15m |
| 19.1 | Rewrite `Register()` — validate request, hash password, build RegisterUserCmd, dispatch, wait for read model, create session        | 15m |
| 19.2 | Handle Register dispatch errors (Conflict → ErrEmailExists, Rejection → ErrValidation)                                              | 10m |
| 19.3 | Session creation after successful register (unchanged logic)                                                                        | 10m |
| 19.4 | UserID bridge: convert usermgmt.UserID ↔ id.AggregateID                                                                             | 10m |
| 20.1 | Rewrite `ChangePassword()` — load state from read model, verify old password, hash new, dispatch ChangePasswordCmd                  | 15m |
| 21.1 | Rewrite `UpdateRoles()` — dispatch UpdateRolesCmd (Casbin updated by projection automatically)                                      | 10m |
| 21.2 | Verify UpdateRoles no longer needs manual Casbin sync                                                                               | 10m |
| 22.1 | Rewrite `Login()` — FindByEmail on read model, verify password, lockout check, create session                                       | 15m |
| 22.2 | Publish UserLoggedInEvent on bus after successful login                                                                             | 10m |
| 23.1 | Rewrite `GetUser()` — query UserReadModel.FindByID                                                                                  | 5m  |
| 23.2 | Convert ReadUser → User for API compatibility                                                                                       | 10m |
| 24.1 | Add `ChangeEmail(ctx, userID, newEmail)` — validate, dispatch ChangeEmailCmd                                                        | 10m |
| 24.2 | Add `ChangeDisplayName(ctx, userID, newName)` — dispatch ChangeDisplayNameCmd                                                       | 10m |
| 24.3 | Add `DeleteUser(ctx, userID, reason)` — dispatch DeleteUserCmd, delete sessions                                                     | 15m |
| 25.1 | Rewrite `Authenticate()` — session store lookup + read model lookup for user                                                        | 10m |
| 25.2 | Handle deleted user in Authenticate (read model returns not-found)                                                                  | 5m  |

### Phase 5: Session & Lockout Integration

| #    | Task                                                                                               | Est |
| ---- | -------------------------------------------------------------------------------------------------- | --- |
| 26.1 | Verify SessionStore interface unchanged — no code changes needed                                   | 5m  |
| 26.2 | Verify session creation in Register/Login uses same SessionStore.Create                            | 5m  |
| 26.3 | Test: register creates session, session valid, logout deletes session                              | 10m |
| 27.1 | Wire AccountLockout into Login — check before password verify, record on failure, reset on success | 10m |
| 27.2 | Test: lockout triggers after max attempts                                                          | 10m |
| 27.3 | Test: lockout resets on successful login                                                           | 10m |
| 28.1 | Define `UserLoggedInEvent` bus event (not aggregate event — just for notification)                 | 5m  |
| 28.2 | Publish via `bus.Publish()` after successful login                                                 | 5m  |
| 28.3 | Backward-compat: bridge to old EventHandler callback pattern if configured                         | 10m |

### Phase 6: HTTP Layer

| #    | Task                                                                                   | Est |
| ---- | -------------------------------------------------------------------------------------- | --- |
| 29.1 | Update `AuthHandler` to use new Service (minimal — same method signatures)             | 15m |
| 29.2 | Add route for ChangeEmail if desired (optional — may be future work)                   | 10m |
| 29.3 | Add route for DeleteUser if desired (optional)                                         | 10m |
| 29.4 | Verify cookie management unchanged                                                     | 5m  |
| 30.1 | Map event.Error families to HTTP statuses (Conflict→409, Rejection→400, Transient→500) | 10m |

### Phase 7: Test Rewrite

| #     | Task                                                                                  | Est |
| ----- | ------------------------------------------------------------------------------------- | --- |
| 31.1  | Rewrite `main_test.go` helpers — new newTestService that creates event-sourced setup  | 15m |
| 31.2  | Rewrite Register tests — success, duplicate email, validation, display name           | 15m |
| 31.3  | Rewrite Login tests — success, wrong password, not found, locked                      | 15m |
| 31.4  | Rewrite ChangePassword tests — success, wrong old, too short, user not found          | 15m |
| 31.5  | Rewrite UpdateRoles tests — success, user not found, verify Casbin updated            | 15m |
| 31.6  | Rewrite GetUser tests — found, not found                                              | 10m |
| 31.7  | Rewrite Authenticate tests — valid, expired, deleted user                             | 15m |
| 31.8  | Rewrite Logout tests — success, no session                                            | 10m |
| 31.9  | Add ChangeEmail tests                                                                 | 10m |
| 31.10 | Add ChangeDisplayName tests                                                           | 10m |
| 31.11 | Add DeleteUser tests — success, already deleted, sessions revoked                     | 15m |
| 31.12 | Rewrite mock_test.go — mock event store, bus, read model                              | 15m |
| 32.1  | Rewrite handler_register_test.go                                                      | 10m |
| 32.2  | Rewrite handler_login_test.go                                                         | 10m |
| 32.3  | Rewrite handler_logout_test.go                                                        | 10m |
| 32.4  | Rewrite handler_me_test.go                                                            | 10m |
| 32.5  | Rewrite handler_session_test.go                                                       | 10m |
| 32.6  | Rewrite handler_misc_test.go                                                          | 15m |
| 32.7  | Rewrite coverage_handlers_test.go                                                     | 15m |
| 33.1  | Integration test: register → login → me → change password → re-login → logout         | 15m |
| 33.2  | Integration test: register → update roles → authorize with new roles                  | 15m |
| 33.3  | Integration test: register → delete → verify gone from read model + Casbin            | 15m |
| 34.1  | Projection consistency: dispatch command, immediately query read model                | 10m |
| 34.2  | Projection consistency: verify Casbin enforce matches event-derived state             | 15m |
| 34.3  | Projection replay: write 10 events, restart projection, verify full rebuild           | 15m |
| 35.1  | Update fuzz tests for RegisterRequest.Validate                                        | 10m |
| 35.2  | Update fuzz tests for LoginRequest.Validate                                           | 10m |
| 35.3  | Add fuzz test for foldUser (property: folding same events always produces same state) | 15m |
| 36.1  | Benchmark: Register (command dispatch + projection)                                   | 10m |
| 36.2  | Benchmark: Login (read model query + password verify)                                 | 10m |
| 36.3  | Benchmark: ChangePassword (command dispatch)                                          | 10m |

### Phase 8: Cleanup & Documentation

| #    | Task                                                                                                      | Est |
| ---- | --------------------------------------------------------------------------------------------------------- | --- |
| 37.1 | Remove `store.go` (InMemoryUserStore — replaced by UserReadModel projection)                              | 5m  |
| 37.2 | Remove `UserStore` interface (replaced by UserReadModel)                                                  | 5m  |
| 37.3 | Remove old `user.go` mutation methods (SetRoles, ChangePassword, SetEmail, etc.) — state is immutable now | 10m |
| 37.4 | Remove `events.go` old EventHandler callback (replaced by bus)                                            | 5m  |
| 37.5 | Remove old `service_register.go`, `service_login.go`, `service_misc.go` (replaced by new Service)         | 5m  |
| 37.6 | Clean up unused imports                                                                                   | 5m  |
| 38.1 | Update `events.go` to contain only event payload structs (if not already in es_events.go)                 | 10m |
| 38.2 | Provide bus.SubscribeAll adapter for consumers who used the old EventHandler                              | 10m |
| 39.1 | Update AGENTS.md — new architecture diagram, new dependencies, new patterns                               | 15m |
| 39.2 | Update AGENTS.md — new key decisions (event sourcing, Casbin as projection, password handling)            | 15m |
| 39.3 | Update AGENTS.md — new gotchas (read-your-writes, UserID bridge, projection startup)                      | 15m |
| 40.1 | Update CHANGELOG.md — BREAKING section, migration guide                                                   | 15m |
| 40.2 | Update go.mod version comment if bumping to v3                                                            | 5m  |
| 41.1 | Update example_test.go — new NewService API                                                               | 15m |
| 41.2 | Update example_test.go — new Service.Register usage                                                       | 15m |
| 42.1 | Run `nix run .#test` — all modules pass                                                                   | 5m  |
| 42.2 | Run `nix run .#lint` — zero issues                                                                        | 5m  |
| 42.3 | Run `nix run .#coverage` — verify coverage ≥ 90%                                                          | 5m  |
| 42.4 | Run `nix run .#build` — all modules build                                                                 | 5m  |
| 42.5 | Run `nix flake check` — formatting + apps verified                                                        | 5m  |

---

## 7. Risk Analysis

| Risk                                                                                      | Probability                         | Impact | Mitigation                                                                                                                                            |
| ----------------------------------------------------------------------------------------- | ----------------------------------- | ------ | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Projection startup race**: Projection runner may miss events fired before it subscribes | MEDIUM                              | HIGH   | MemoryBus blocks publishers → synchronous delivery. Add small startup delay like example/user/. For async buses, consumer handles via journal replay. |
| **Read-your-writes breaks with async bus**                                                | LOW (in-memory) / HIGH (production) | HIGH   | Document clearly: in-memory bus = strong consistency; async bus = eventual. Consumers choose their consistency model.                                 |
| **Casbin projection out of sync**                                                         | MEDIUM                              | HIGH   | Projection is idempotent for AddGroupPolicy/RemoveGroupPolicy. Journal replay rebuilds from scratch. Add reconciliation test.                         |
| **Password hash in events**                                                               | LOW                                 | MEDIUM | Hash is bcrypt — already irreversible. Event payloads are internal, not exposed via API. Same security as current approach.                           |
| **UserID type bridge bugs**                                                               | MEDIUM                              | MEDIUM | Comprehensive bridge tests. Conversion at Service boundary only — domain code uses id.AggregateID exclusively.                                        |
| **Test coverage drop during rewrite**                                                     | HIGH                                | MEDIUM | Phase 7 is comprehensive. Write new tests BEFORE removing old ones.                                                                                   |
| **Breaking consumer API**                                                                 | CERTAIN                             | HIGH   | Document in CHANGELOG. This is v2→v3 breaking change. Provide migration guide.                                                                        |
| **Event schema evolution**                                                                | LOW (first version)                 | LOW    | Use SchemaVersion field on events. Add upcasters when schema changes. Not needed for v1 of event-sourced system.                                      |

---

## 8. Migration Strategy

### For Consumers (Library Users)

**Before (CRUD):**

```go
svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{
    UserStore: usermgmt.NewInMemoryUserStore(),
    // ...
})
resp, err := svc.Register(ctx, usermgmt.RegisterRequest{...})
```

**After (Event-Sourced):**

```go
svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{
    EventStore: memory.NewMemoryStore(),   // NEW
    EventBus:   memory.NewMemoryBus(),     // NEW
    // UserStore removed — replaced by projection
    SessionStore: usermgmt.NewInMemorySessionStore(),  // unchanged
    Authz:       usermgmt.NewAuthz(),      // unchanged
    // ...
})
resp, err := svc.Register(ctx, usermgmt.RegisterRequest{...})  // same API!
```

The Service method signatures stay the same. The config changes. Internals are completely different.

### File Renaming Map

| Old File                                     | New File                                                           | Status          |
| -------------------------------------------- | ------------------------------------------------------------------ | --------------- |
| `store.go` (UserStore + InMemoryUserStore)   | REMOVED — replaced by `es_readmodel.go` (UserReadModel projection) | Delete          |
| `user.go` (User entity + mutation methods)   | `es_state.go` (UserState — immutable, folded from events)          | Replace         |
| `events.go` (EventHandler + 4 event structs) | `es_events.go` (6 event payloads for event store)                  | Replace         |
| `service_core.go`                            | `es_service.go` (rewritten)                                        | Replace         |
| `service_register.go`                        | merged into `es_service.go`                                        | Merge + Replace |
| `service_login.go`                           | merged into `es_service.go`                                        | Merge + Replace |
| `service_misc.go`                            | merged into `es_service.go`                                        | Merge + Replace |
| —                                            | `es_constants.go` (NEW)                                            | Create          |
| —                                            | `es_commands.go` (NEW)                                             | Create          |
| —                                            | `es_decide.go` (NEW)                                               | Create          |
| —                                            | `es_readmodel.go` (NEW)                                            | Create          |
| —                                            | `es_casbin_projection.go` (NEW)                                    | Create          |
| —                                            | `es_setup.go` (NEW)                                                | Create          |
| `authz_*.go`                                 | unchanged                                                          | Keep            |
| `lockout.go`                                 | unchanged                                                          | Keep            |
| `http.go`                                    | minimal changes                                                    | Update          |
| `middleware.go`                              | unchanged                                                          | Keep            |
| `id.go`                                      | unchanged                                                          | Keep            |
| `errors.go`                                  | minimal additions                                                  | Update          |

---

## Summary Statistics

| Metric                 | Value                                  |
| ---------------------- | -------------------------------------- |
| Total medium tasks     | 42                                     |
| Total fine tasks       | ~130                                   |
| Estimated total effort | ~40-50 hours                           |
| New files              | 7                                      |
| Removed files          | 5 (merged/replaced)                    |
| New events             | 6                                      |
| New commands           | 6                                      |
| New dependencies       | 6 (go-cqrs-lite sub-modules)           |
| Breaking changes       | YES (ServiceConfig, UserStore removed) |
