# Plan: Convert All 6 usermgmt Projections to Metaengine Declarations

**Date:** 2026-08-09
**Goal:** Replace imperative `projection.Projection` implementations with declarative `system.Evolve[R]()` + `system.Lookup[R]()` / `system.QuerySet[R]()` declarations so the system's internal projection host auto-wires everything — eliminating the separate `ProjectionLayer` host.

---

## Current State

### Architecture (two-host)

```
system.New() ──→ internal projection host (nil — no metaengine projections declared)
                  ↓
ProjectionLayer ──→ separate projectionhost.Host
                    ├── UserReadModel       (projection.Projection)
                    ├── MembershipReadModel (projection.Projection)
                    ├── TenantReadModel     (projection.Projection)
                    ├── BotReadModel        (projection.Projection)
                    ├── CasbinProjection    (projection.Projection)
                    └── AuditLog            (projection.Projection)
```

### Target Architecture (one host)

```
system.New(DomainConfig{
    Evolutions: usermgmt.Evolutions(),
    Projections: usermgmt.Projections(),
}) ──→ internal projection host
        └── projectionadapter.Adapter
              └── metaengine.Store
                    ├── "users"          → Lookup[UserView]
                    ├── "memberships"    → Lookup[MembershipView]
                    ├── "tenants"        → Lookup[TenantView]
                    ├── "bots"           → Lookup[BotView]
                    ├── "authz_policies" → QuerySet[PolicyEntry]
                    └── "audit_log"      → QuerySet[AuditEntry] (LogBackend append)
```

---

## Fold Handler Signature Reference

The metaengine `On[E]()` function accepts these handler shapes (verified from `fold.go`):

| Handler shape | Fold kind | Meaning |
|---|---|---|
| `func(E) (key, val)` | `FoldInsert` | Insert/upsert by key |
| `func(E, prev) val` | `FoldUpdate` | Update existing by key (prev is the current value or zero) |
| `func(E) Remove[V]()` | `FoldRemove` | Remove by key |
| `func(E) Append{Value}` | `FoldAppend` | Append to a log (requires LogBackend engine) |

The `Evolve[R]()` builder's `OnEvolution[R, E]()` wraps these into explicit folds with `(event E, result *R)` mutation handlers for the `func(E, prev) val` shape.

### What the fold handler receives

The `TypeDecoder` wraps each payload in `EventWithID[E]`:
```go
type EventWithID[E any] struct {
    ID      string   // evt.StreamID().String()
    Payload E        // decoded payload struct
}
```

**Gap:** No timestamp (`evt.OccurredAt()`) or metadata (`evt.Metadata().UserID`). The AuditLog needs both. See §5.2 for the solution.

---

## Per-Projection Design

### 1. UserReadModel → `Evolve[UserView]` + `Lookup[UserView]`

#### View struct

```go
type UserView struct {
    ID               string               `json:"id"`
    Email            string               `json:"email"`
    DisplayName      string               `json:"display_name,omitempty"`
    Credentials      []CredentialView     `json:"credentials,omitempty"`
    ExternalAccounts []ExternalAccountView `json:"external_accounts,omitempty"`
    EmailVerified    bool                 `json:"email_verified"`
    TOTPEnabled      bool                 `json:"totp_enabled"`
    CreatedAt        time.Time            `json:"created_at"`
    UpdatedAt        time.Time            `json:"updated_at"`
}
```

**Note:** `TOTPSecret []byte` (tagged `json:"-"` in domain `User`) is dropped — it should not be in a read model projection (security concern). The `ID` field is `string` (from `EventWithID.ID`), not `UserID`, for engine portability.

#### Fold declarations (12 events, all explicit)

| Event | Fold kind | Handler logic |
|---|---|---|
| `UserRegistered` | Insert | `func(e EventWithID[UserRegisteredPayload]) (string, UserView)` — key=e.ID, val=UserView{Email, DisplayName, CreatedAt} |
| `EmailChanged` | Update | `func(e EventWithID[EmailChangedPayload], prev UserView) UserView` — prev.Email=e.Payload.Email, prev.EmailVerified=false |
| `DisplayNameChanged` | Update | Set prev.DisplayName from payload |
| `CredentialAdded` | Update | Append to prev.Credentials |
| `CredentialRemoved` | Update | Filter out by credential ID |
| `UserDeleted` | Remove | `metaengine.Remove[UserView]()` keyed by e.ID |
| `EmailVerified` | Update | Set prev.EmailVerified=true |
| `TOTPEnabled` | Update | Set prev.TOTPEnabled=true |
| `TOTPDisabled` | Update | Set prev.TOTPEnabled=false |
| `ExternalAccountLinked` | Update | Append to prev.ExternalAccounts |
| `ExternalAccountUnlinked` | Update | Filter out by provider+subject |
| `RolesUpdated` | Skip | No-op (roles live in authz). Use `metaengine.Skip` |

#### Query declarations

```go
system.Lookup[UserView]("user_by_id").Key("ID").Done()
system.QuerySet[UserView]("users").Filterable("email", "email_verified").Done()
```

- `FindByID(streamID)` → `system.Get[UserView](ctx, sys, "user_by_id", streamID.String())`
- `FindByEmail(email)` → `system.Find[UserView](ctx, sys, "users", system.Where("email", email))`
- `FindByExternalAccount(...)` → requires a separate edge projection OR a QuerySet filter on a JSON-encoded external_accounts field (see §6.1)

#### Challenges

- **`RolesUpdated` is a no-op** — the current projection validates the payload but does nothing. Use `metaengine.Skip` return type, or just don't register this event for the UserView evolution.
- **`CreatedBy` timestamp** — `EventWithID` doesn't include `evt.OccurredAt()`. See §5.2.
- **`FindByExternalAccount`** — the current read model maintains a `provider+subject → aggID` index. In metaengine, this becomes either a QuerySet with a filter on a serialized field (not SQL-pushdown-friendly), or a separate edge projection. See §6.1.

---

### 2. MembershipReadModel → `Evolve[MembershipView]` + `Lookup[MembershipView]`

#### View struct

```go
type MembershipView struct {
    ID       string   `json:"id"`        // stream ID (actor:tenant pair)
    ActorID  string   `json:"actor_id"`
    TenantID string   `json:"tenant_id"`
    Roles    []string `json:"roles"`
}
```

**Note:** `Roles` is `[]string` instead of `[]identitymodel.Role` for serialization portability. `AddedAt` timestamp lost (see §5.2).

#### Fold declarations (3 events, all explicit)

| Event | Fold kind | Handler logic |
|---|---|---|
| `MemberAdded` | Insert | `func(e EventWithID[MemberAddedPayload]) (string, MembershipView)` — key=e.ID |
| `MemberRolesChanged` | Update | Replace prev.Roles from payload |
| `MemberRemoved` | Remove | Keyed by e.ID |

#### Query declarations

```go
system.Lookup[MembershipView]("membership_by_id").Key("ID").Done()
system.QuerySet[MembershipView]("memberships").Filterable("actor_id", "tenant_id").Done()
```

- `FindByAggregateID(streamID)` → `system.Get[MembershipView](ctx, sys, "membership_by_id", streamID.String())`
- `FindByActor(actorID)` → `system.Find[MembershipView](ctx, sys, "memberships", system.Where("actor_id", actorID))`
- `FindByTenant(tenantID)` → `system.Find[MembershipView](ctx, sys, "memberships", system.Where("tenant_id", tenantID))`

---

### 3. TenantReadModel → `Evolve[TenantView]` + `Lookup[TenantView]`

#### View struct

```go
type TenantView struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    DisplayName string `json:"display_name,omitempty"`
    Suspended   bool   `json:"suspended"`
}
```

**Note:** `Deleted` is dropped — deletion becomes a `FoldRemove`, not a soft-delete flag.

#### Fold declarations (4 events)

| Event | Fold kind | Handler logic |
|---|---|---|
| `TenantCreated` | Insert | key=e.ID, val=TenantView{Name, DisplayName} |
| `TenantSuspended` | Update | prev.Suspended=true |
| `TenantReactivated` | Update | prev.Suspended=false |
| `TenantDeleted` | Remove | Keyed by e.ID |

#### Query declarations

```go
system.Lookup[TenantView]("tenant_by_id").Key("ID").Done()
system.QuerySet[TenantView]("tenants").Filterable("name", "suspended").Done()
```

---

### 4. BotReadModel → `Evolve[BotView]` + `Lookup[BotView]`

#### View struct

```go
type BotView struct {
    ID        string   `json:"id"`
    Name      string   `json:"name"`
    OwnerID   string   `json:"owner_id"`
    TokenHash []byte   `json:"token_hash"`
    Scopes    []string `json:"scopes"`
}
```

**Note:** `Deleted` dropped (becomes Remove fold). `TokenHash` stays as `[]byte` — the memory engine stores it directly, and JSON-serializing engines can base64-encode it.

#### Fold declarations (2 events)

| Event | Fold kind | Handler logic |
|---|---|---|
| `BotRegistered` | Insert | key=e.ID, val=BotView from payload |
| `BotDeleted` | Remove | Keyed by e.ID |

#### Query declarations

```go
system.Lookup[BotView]("bot_by_id").Key("ID").Done()
system.QuerySet[BotView]("bots").Filterable("owner_id").Done()
```

- `FindByTokenHash(hash)` — requires either a separate Lookup keyed by token hash, or a QuerySet scan with a filter. The hash is `[]byte`, which may not filter well as a string. See §6.2.

---

### 5. CasbinProjection → `Evolve[PolicyEntry]` + `QuerySet[PolicyEntry]`

#### The conceptual shift

Currently CasbinProjection delegates to the Casbin library's enforcer (`*authz.Authz`), which manages policy state internally. The metaengine approach models the policy **as data** — a collection of `(subject, role, domain)` entries — and answers enforcement queries as collection lookups.

#### View struct

```go
type PolicyEntry struct {
    Key     string `json:"key"`      // "{subject}:{domain}:{role}" — unique composite
    Subject string `json:"subject"`
    Role    string `json:"role"`
    Domain  string `json:"domain"`
}
```

#### Fold declarations (cross-aggregate — 8 event types produce changes)

| Event | Fold kind | Handler logic |
|---|---|---|
| `UserRegistered` | MultiInsert | For each role in payload → insert PolicyEntry{subject=e.ID, role, domain=e.ID} |
| `RolesUpdated` | MultiInsert + Remove | Remove all for subject+domain, then insert new roles |
| `UserDeleted` | Remove | All entries where subject=e.ID (requires multi-key remove — see §5.3) |
| `MemberAdded` | MultiInsert | For each role → insert PolicyEntry{subject=actorID, role, domain=tenantID} |
| `MemberRolesChanged` | MultiInsert + Remove | Remove all for actor+tenant, then insert new roles |
| `MemberRemoved` | Remove | All entries for actor in tenant domain |
| `TenantDeleted` | Remove | All entries where domain=e.ID |
| `BotDeleted` | Remove | All entries where subject=e.ID |

**No-op events (subscribed for ordering only in current impl):** `CredentialAdded`, `CredentialRemoved`, `ExternalAccountLinked`, `ExternalAccountUnlinked` — these don't affect authz policies. Drop them entirely.

#### Query declarations

```go
system.QuerySet[PolicyEntry]("authz_policies").Filterable("subject", "domain", "role").Done()
```

#### Enforcement

```go
func Enforce(sys *system.System, sub, dom, obj string, act Action) (bool, error) {
    entries, err := system.Find[PolicyEntry](ctx, sys, "authz_policies",
        system.Where("subject", sub),
        system.Where("domain", dom),
    )
    // Check if any entry grants the action
    for _, e := range entries {
        if roleGrantsAction(e.Role, act) { return true, nil }
    }
    return false, nil
}
```

**Challenge:** The current Casbin enforcer supports role inheritance (admin inherits from user). In the fold model, this becomes either:
- A query-time check (traverse role hierarchy)
- Pre-expanded policy entries at insert time (flatten hierarchy)

For the current 5 roles (super_admin, admin, user, viewer, owner), pre-expansion is trivial: when inserting "admin", also insert implicit "user" and "viewer" entries. This keeps enforcement as a simple lookup.

#### Challenges

- **Multi-key remove** — `UserDeleted` needs to remove ALL policy entries for a subject across ALL domains. The metaengine `Remove[V]()` takes a single key. See §5.3.
- **Role inheritance** — see above.
- **Casbin pattern matching** — the current Casbin model supports glob patterns in policies. The usermgmt model doesn't use patterns (all subjects/domains are concrete IDs), so this is not a concern.

---

### 6. AuditLog → `Evolve[AuditEntryView]` + LogBackend Append

#### View struct

```go
type AuditEntryView struct {
    EventType   string    `json:"event_type"`
    AggregateID string    `json:"aggregate_id"`
    OccurredAt  time.Time `json:"occurred_at"`
    UserID      string    `json:"user_id,omitempty"`
    Action      string    `json:"action"`
}
```

#### Fold declarations (12 user event types, all append)

All user events produce an append fold with the same shape:

```go
func(e EventWithID[UserRegisteredPayload]) metaengine.Append {
    return metaengine.Append{Value: AuditEntryView{
        EventType:   "UserRegistered",
        AggregateID: e.ID,
        Action:      "register",
        // OccurredAt and UserID gaps — see §5.2
    }}
}
```

One fold per event type (12 total), each producing a different `Action` string.

**Alternative:** A single fold on a generic `event.Event` that switches on type. But the metaengine fold model is per-event-type, so 12 separate folds is the idiomatic approach.

#### Query declarations

The append fold requires a `LogBackend` engine. Queries are scans with filtering:

```go
// No Lookup — audit log is append-only, no point lookups
// QuerySet for filtered scans:
system.QuerySet[AuditEntryView]("audit_log").Filterable("aggregate_id").Done()
```

**Challenge:** The `appendFold` requires `LogBackend` engine support. The memory engine implements `LogBackend.LogAppend()`. SQLite engine may or may not support it. See §5.4.

---

## 5. Cross-Cutting Challenges

### 5.1 Event Naming Convention Mismatch

`AutoCRUDByNamedEvents` expects `*Created`/`*Updated`/`*Deleted` suffixes. Most identity-model events don't follow this:

- `UserRegistered` (not `UserCreated`)
- `MemberAdded` (not `MemberCreated`)
- `EmailChanged` (not `UserUpdated`)

**Solution:** Use explicit folds for ALL events via `OnEvolution[R, E](builder, eventType, sample, fold)`. Don't rely on convention. This is more verbose but correct and self-documenting.

**Alternative:** Rename events to follow the convention. Rejected — event names are part of the published language and already persisted in event stores.

### 5.2 Missing Event Metadata in Fold Handlers

**Problem:** `EventWithID[E]` provides `ID` (stream ID) and `Payload`, but NOT:
- `evt.OccurredAt()` — needed by UserView (CreatedAt/UpdatedAt) and AuditEntryView (OccurredAt)
- `evt.Metadata().UserID` — needed by AuditEntryView

**Solution A (preferred): Extend `EventWithID`**

Add `OccurredAt time.Time` and `Metadata` fields to the `EventWithID` struct:

```go
type EventWithID[E any] struct {
    ID        string
    Payload   E
    OccurredAt time.Time
    Metadata  event.Metadata
}
```

This is a one-line change to the `Register[E]()` handler in `projectionadapter/typed_decoder.go`. All fold handlers that need timestamp/metadata get it. Those that don't, ignore the extra fields.

**Solution B: Custom `EventDecoder`**

Write a custom `EventDecoder` that wraps the full event:

```go
func fullEventDecoder(evt event.Event) (any, error) {
    // decode payload + attach full event context
}
```

This gives complete access but requires more boilerplate.

**Solution C: Store timestamps in payloads**

Add `OccurredAt` to every payload struct at event creation time. Rejected — pollutes domain events with projection concerns.

**Recommendation:** Solution A — extend `EventWithID`. It's the smallest change with the widest benefit.

### 5.3 Multi-Key Remove

**Problem:** `UserDeleted` needs to remove ALL policy entries for a subject across all domains. `Remove[V]()` takes a single key.

**Current workaround:** The `removeFold` uses a `keyExtractor` function. If the key extractor returns the subject string, and the engine supports prefix/pattern removal, this could work. But the memory engine's `MapBackend.MapDelete()` takes an exact key.

**Solution A: Composite key with prefix scan**

Model PolicyEntry key as `"{subject}:{domain}:{role}"`. For "remove all for subject", use a prefix scan + batch remove. This requires engine support for prefix operations.

**Solution B: Separate "tombstone" collection**

Maintain a separate collection of "deleted subjects". Enforcement queries check against this collection before returning positive.

**Solution C: Accept the limitation — cascade at the application layer**

When a user is deleted, the application dispatches individual `RemoveMember` events for each membership. The CasbinProjection handles each individually. This is the current behavior (the `DeleteUser` command cascades membership deletion).

**Recommendation:** Solution C for now. The `DeleteUser` command already cascades membership deletion in the domain layer (`usermgmt` v4.x — see AGENTS.md). So `UserDeleted` → individual `MemberRemoved` events → individual policy removes. The cross-aggregate `UserDeleted` handler in CasbinProjection becomes redundant.

### 5.4 LogBackend Engine Support

**Problem:** The `appendFold` (for AuditLog) requires `LogBackend` support. The memory engine provides it. SQLite engine may not.

**Check needed:** Verify which engines implement `LogBackend`. If SQLite doesn't, AuditLog must use a `QuerySet` with insert folds (not append), sacrificing the log semantics.

**Fallback:** Model AuditLog as a `QuerySet[AuditEntryView]` with insert folds keyed by `{eventVersion}` or `{aggregateID}:{eventVersion}`. This loses the "append-only log" semantic but works on any engine.

### 5.5 Projection Drain / Read-Your-Writes

The current `ProjectionLayer.WaitForDrain()` polls `Host.Status()` until all workers reach `WorkerLive`. With the system's internal host, `sys.Start(ctx)` starts projections, but there's no built-in "wait for drain" API.

**Solution:** The system's projection host IS a `projectionhost.Host` — the same `WaitForDrain` polling logic applies. Expose a helper or use the system's existing drain mechanism.

---

## 6. Secondary Index Challenges

### 6.1 FindByExternalAccount

**Current:** `map[externalAccountKey]id.StreamID` — O(1) lookup by `(provider, subject)`.

**Metaengine options:**

1. **QuerySet filter** — Add `ExternalAccounts` as a JSON field on UserView, filter with `Where("external_accounts", ...)`. But JSON field filtering is not SQL-pushdown-friendly.

2. **Separate edge projection** — `Edge{From: provider+subject, To: userID}`. Lookup via `system.Get[EdgeView](ctx, sys, "external_accounts", provider+subject)`. This is O(1) and engine-friendly.

3. **Separate Lookup** — `Lookup[ExternalAccountLink]("external_account_links").Key("ProviderSubject").Done()` with insert/remove folds on link/unlink events.

**Recommendation:** Option 3 — a separate lightweight Lookup projection. It's the cleanest and most query-efficient.

### 6.2 FindByTokenHash

**Current:** `map[string]*Bot` keyed by `string(tokenHash)`.

**Metaengine options:**

1. **Separate Lookup** — `Lookup[BotTokenView]("bot_tokens").Key("TokenHash").Done()`. Insert on BotRegistered, remove on BotDeleted.

2. **QuerySet filter** — `QuerySet[BotView]("bots").Filterable("token_hash").Done()`. But `[]byte` → `string` conversion needed for filtering.

**Recommendation:** Option 1 — separate Lookup. Token hash lookups are on the hot path (every API request), so O(1) is important.

---

## 7. Implementation Phases

### Phase 1: Foundation (go-cqrs-lite changes)

**Goal:** Extend the metaengine to support the fold patterns we need.

| Step | Task | Effort |
|---|---|---|
| 1.1 | Extend `EventWithID[E]` with `OccurredAt time.Time` field | Trivial |
| 1.2 | Add `MultiInsert` fold support for Casbin (multiple entries from one event) | Small |
| 1.3 | Verify `LogBackend` support in SQLite engine (for AuditLog) | Research |
| 1.4 | Add `system.Get[R]` / `system.Find[R]` convenience methods (already exist) | Verify |

### Phase 2: View Structs & Evolutions (identity-model or systemadapter)

**Goal:** Define the result types and fold declarations.

| Step | Task | Effort |
|---|---|---|
| 2.1 | Define `UserView`, `MembershipView`, `TenantView`, `BotView`, `PolicyEntry`, `AuditEntryView` structs | Small |
| 2.2 | Define `CredentialView`, `ExternalAccountView` sub-structs | Small |
| 2.3 | Define `ExternalAccountLink`, `BotTokenView` secondary index structs | Small |
| 2.4 | Write `usermgmt.Evolutions()` returning `[]system.EvolutionSpec` for all 6 result types | Medium |
| 2.5 | Write `usermgmt.Projections()` returning `[]system.ProjectionDeclaration` for all lookups/queries | Medium |

### Phase 3: Wire into systemadapter.DomainConfig

**Goal:** Update DomainConfig to declare projections instead of using ProjectionLayer.

| Step | Task | Effort |
|---|---|---|
| 3.1 | Add `Evolutions` and `Projections` fields to the DomainConfig returned by `systemadapter.DomainConfig()` | Small |
| 3.2 | Verify `system.New()` creates the internal projection host with these declarations | Small |
| 3.3 | Verify `sys.MetaEngine()` returns non-nil store | Small |
| 3.4 | Verify `system.Get[R]()` / `system.Find[R]()` work for each view type | Medium |

### Phase 4: Query Helpers

**Goal:** Provide Go-friendly query methods that replace the current read model API.

| Step | Task | Effort |
|---|---|---|
| 4.1 | Write `systemadapter.FindUserByID(ctx, sys, id)` → `system.Get[UserView]` | Small |
| 4.2 | Write `systemadapter.FindUserByEmail(ctx, sys, email)` → `system.Find[UserView]` | Small |
| 4.3 | Write remaining user queries (FindByExternalAccount, AllUsers, Count) | Small |
| 4.4 | Write membership queries (FindByAggregateID, FindByActor, FindByTenant) | Small |
| 4.5 | Write tenant queries (FindByID, FindByName, All) | Small |
| 4.6 | Write bot queries (FindByID, FindByTokenHash, FindByOwner) | Small |
| 4.7 | Write authz enforcement helper (Enforce) | Medium |
| 4.8 | Write audit log queries (Entries, EntriesFor, Recent, Count) | Small |

### Phase 5: Retire ProjectionLayer

**Goal:** Remove the separate host and the old read model implementations.

| Step | Task | Effort |
|---|---|---|
| 5.1 | Mark `ProjectionLayer` as deprecated | Trivial |
| 5.2 | Add migration guide (ProjectionLayer → systemadapter queries) | Medium |
| 5.3 | Update all tests to use `system.Get[R]` / `system.Find[R]` instead of `pl.User.FindByID()` | Large |
| 5.4 | Update `examples/system-demo/main.go` | Small |
| 5.5 | Update adminui/dashboardui if they reference ProjectionLayer | Medium |
| 5.6 | Remove ProjectionLayer (or keep as backward-compat shim) | Medium |

### Phase 6: Testing & Verification

| Step | Task | Effort |
|---|---|---|
| 6.1 | Equivalence test: same events → same query results as old read models | Large |
| 6.2 | SQLite engine test for all 6 projections | Medium |
| 6.3 | Authz enforcement test (admin allowed, plain denied, role inheritance) | Medium |
| 6.4 | Performance comparison (old read model vs metaengine queries) | Optional |
| 6.5 | Drain/read-your-writes test | Small |

---

## 8. File Layout

### New files in systemadapter

```
systemadapter/
├── domain_config.go         (modified — add Evolutions + Projections)
├── projections.go           (deprecated — ProjectionLayer stays for backward compat)
├── type_decoder.go          (unchanged)
├── views.go                 (NEW — UserView, MembershipView, TenantView, etc.)
├── evolutions.go            (NEW — Evolve[R]() declarations for all 6 result types)
├── projections_declarative.go (NEW — Lookup/QuerySet declarations)
├── queries.go               (NEW — FindUserByID, FindByEmail, Enforce, etc.)
├── systemadapter_test.go    (modified — use declarative queries)
├── equivalence_test.go      (NEW — verify parity with old read models)
```

### Files in go-cqrs-lite (upstream changes)

```
go-cqrs-lite/metaengine/projectionadapter/
├── typed_decoder.go         (modified — add OccurredAt to EventWithID)
```

---

## 9. Migration Path for Consumers

### Before (current)

```go
sys, _ := system.New(ctx, systemadapter.DomainConfig(), deployment)
pl, _ := systemadapter.NewProjectionLayer(sys)
pl.Start(ctx)
sys.Start(ctx)

// Query
user, _ := pl.User.FindByID(streamID)
memberships := pl.Membership.FindByTenant(tenantID)
allowed, _ := pl.Authz.Enforce(sub, dom, obj, act)
```

### After (declarative)

```go
sys, _ := system.New(ctx, systemadapter.DomainConfig(), deployment)
// No ProjectionLayer needed — projections are auto-wired by system.New()
sys.Start(ctx)

// Query via metaengine
user, _ := systemadapter.FindUserByID(ctx, sys, streamID)
memberships, _ := systemadapter.FindMembershipsByTenant(ctx, sys, tenantID)
allowed, _ := systemadapter.Enforce(ctx, sys, sub, dom, obj, act)
```

---

## 10. Risk Assessment

| Risk | Probability | Impact | Mitigation |
|---|---|---|---|
| EventWithID extension breaks go-cqrs-lite consumers | Low | Medium | Additive change — existing code ignores extra fields |
| LogBackend not supported by SQLite engine | Medium | Low | Fall back to insert-fold QuerySet for AuditLog |
| Multi-key remove for Casbin not supported | Medium | Medium | Rely on domain-layer cascade (DeleteUser → individual MemberRemoved events) |
| Role inheritance not expressible in fold model | Low | Medium | Pre-expand role hierarchy at insert time |
| Performance regression vs hand-tuned maps | Low | Low | Memory engine uses `MapBackend` — same O(1) lookups |
| Secondary index projections (token hash, external account) add complexity | Medium | Low | Keep them as separate lightweight Lookup projections |
| adminui/dashboardui break when ProjectionLayer is removed | High | High | Keep ProjectionLayer as backward-compat shim; deprecate later |

---

## 11. What NOT to Do

- **Do NOT rename events** to match Created/Updated/Deleted convention — events are published language
- **Do NOT remove Casbin as a dependency** — the Authz engine has other uses (model loading, policy validation). Only the CasbinProjection's fold model changes
- **Do NOT try to do all 6 projections in one PR** — phase them, starting with TenantReadModel (simplest: 4 events, no sub-collections, no secondary indexes)
- **Do NOT block on LogBackend SQLite support** — if it's not there, use insert folds for AuditLog
- **Do NOT remove ProjectionLayer immediately** — keep it as a backward-compat shim that wraps the new declarative queries

---

## 12. Recommended Execution Order

1. **TenantReadModel** — simplest (4 events, no sub-collections, no secondary indexes, convention partially matches)
2. **BotReadModel** — simple (2 events, one secondary index via separate Lookup)
3. **MembershipReadModel** — simple (3 events, query-based secondary indexes)
4. **UserReadModel** — complex (12 events, sub-collection mutations, multiple secondary indexes)
5. **AuditLog** — requires EventWithID extension + LogBackend verification
6. **CasbinProjection** — most complex (cross-aggregate, role inheritance, multi-key remove)
