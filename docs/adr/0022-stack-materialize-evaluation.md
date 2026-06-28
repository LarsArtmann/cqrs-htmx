# ADR 0022: stack.Materialize Evaluation — Prototype, Findings, and Per-Read-Model Decision

**Date:** 2026-06-28
**Status:** Accepted (evaluation complete — partial adoption recommended)
**Updates:** ADR-0016 §Decision (which rejected Materialize as "too heavy")

## Context

ROADMAP v4.0 lists "stack.Materialize for persistent read models" as a Low-priority
item. It has been evaluated and rejected at least **six times** across planning docs
(2026-06-22 through 2026-06-28), always with the same one-line rationale:

> Our read models have complex event handling (12-event switches, external account
> indexes) that don't fit the declarative OnCreate/OnUpdate/OnTombstone pattern.

ADR-0016 rejected Materialize as "too heavy for a library" because it "requires
consumers to adopt `kv.TypedStore` and rewrites 6 read model internals."

This ADR documents a **from-scratch re-evaluation** that reads the actual v3.1.0
source code, builds a working prototype, and produces a per-read-model decision
matrix instead of a blanket verdict.

## What stack.Materialize Actually Is (v3.1.0)

```go
type Materialize[V any, K fmt.Stringer] struct {
    Store        kv.ViewStore[V, K]
    KeyFromEvent func(evt event.Event) (K, error)
    OnCreate     func(ctx, evt) (*V, error)
    OnUpdate     func(ctx, evt, existing *V) (*V, error)
    OnTombstone  func(ctx, evt, existing *V) (*V, error)
    OnRebirth    func(ctx, evt, existing *V) (*V, error)
}
```

The dispatch logic (replicated from the unexported `handleEvent`):

1. `KeyFromEvent(evt)` → key
2. If `evt.Metadata().Tombstone != nil` → route to `OnTombstone` / `OnRebirth`
3. Otherwise: `Store.Get(key)` → found? `OnUpdate` : `OnCreate`
4. `Store.Set(key, result)`

**Critical v3.1.0 limitations discovered:**

| Limitation                                                                            | Impact                                                                                |
| ------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------- |
| `Materialize` does NOT implement `event.Projection` (no `Name`/`Handle`/`EventTypes`) | Cannot be wired into our `StartProjections` dispatch directly                         |
| Only export is `HandlerFunc()` → Watermill `message.NoPublishHandlerFunc`             | Requires Watermill Router — our setup uses manual journal replay + `bus.SubscribeAll` |
| `handleEvent` is unexported                                                           | Cannot call it from consumer code without replication                                 |

The unreleased development version (local `~/projects/go-cqrs-lite/`) adds
`Name()`, `Handle()`, `EventTypes()`, and `ProjectionName` — making Materialize
a full `event.Projection`. Until that ships, a wrapper is required.

## Correcting the Prior Rejection

The prior "12-event switch doesn't fit" rationale was **partially wrong**:

1. **OnUpdate receives ALL events.** You branch on `evt.Type()` inside the
   callback — the same `switch` statement, just relocated. For read models with
   many event types, Materialize does not eliminate the switch; it moves it.

2. **The real value is the store, not the dispatch.** Materialize pairs with
   `kv.ViewStore`, which provides `ViewQuerier` (WHERE/ORDER/LIMIT), `ViewCounter`
   (COUNT), `ViewResetter` (DELETE FROM for projection rebuild), `ViewBatchSetter`
   (bulk replay), and `TombstoneQuerier` (server-side tombstone filtering). These
   capabilities are free with the store backend, independent of the dispatch model.

3. **The real blocker is `EventTypes()`.** Materialize returns nil (handles all
   types by design). Our dispatch uses `slices.Contains(proj.EventTypes(), evt.Type())`
   which treats nil as "accept none." A wrapper that returns `allTenantEventTypes`
   fixes this trivially.

## Prototype: MaterializedTenantReadModel

A working prototype was built in `es_tenant_materialize.go` (+ tests). It proves
the concept on the **simplest** read model:

| Criterion             | TenantReadModel (current)                              | MaterializedTenantReadModel (prototype)             |
| --------------------- | ------------------------------------------------------ | --------------------------------------------------- |
| Lines of Handle logic | 45 (4-case switch)                                     | 90 (dispatch + 3 callbacks)                         |
| Secondary indexes     | 0                                                      | 0                                                   |
| Tombstone handling    | Hard delete (`delete(map)`)                            | Soft delete (tombstone metadata → `Deleted=true`)   |
| Persistence           | In-memory only                                         | Any `kv.ViewStore` (memory, SQL, Pebble)            |
| Query filtering       | Manual loop                                            | `stack.FilterTombstoned` + `List(policy)`           |
| SQL integration       | Requires `SQLTenantReadModel` wrapper with `syncToSQL` | Native via `storage.SQLViewStore[Tenant, TenantID]` |
| Tests                 | 0 direct (covered via SQL tests)                       | 4 dedicated tests, race-safe                        |

**Test results:** all 4 tests pass (`-race`), 0 lint issues, errorfamily clean.

### What the Prototype Demonstrates

1. **Materialize CAN replace a hand-written projection** — behavioral equivalence
   confirmed for create/suspend/reactivate/delete lifecycle.
2. **Tombstone-aware queries work** — deleted tenants excluded from `FindByID`,
   `FindByName`, `All` via `IsTombstoned()` + `stack.FilterTombstoned`.
3. **The EventTypes gap is bridgeable** — a 1-line wrapper returns
   `allTenantEventTypes`, restoring compatibility with our dispatch routing.
4. **The Handle dispatch is replicable** — the wrapper faithfully copies the
   upstream `handleEvent` logic (documented in the method comment).

### What the Prototype Costs

1. **More code, not less** — 90 LOC vs 45 LOC for TenantReadModel. The dispatch
   replication adds bulk that the upstream `Handle()` method would eliminate.
2. **API signature change** — queries take `context.Context` (Store operations
   require it). Current in-memory `FindByID(aggID)` → `FindByID(ctx, aggID)`.
3. **`IsTombstoned()` method added** to `*Tenant` — non-breaking but a new export.

## Per-Read-Model Decision Matrix

| Read Model              | Events               | Secondary Indexes            | Tombstone?     | Fit           | Decision                                                                                                                                                                                                                |
| ----------------------- | -------------------- | ---------------------------- | -------------- | ------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **TenantReadModel**     | 4                    | 0                            | ✅ Marked      | **Excellent** | **Adopt** — prototype proven, clean 1:1 mapping                                                                                                                                                                         |
| **BotReadModel**        | 2                    | `byTokenHash`                | ✅ Marked      | **Good**      | **Defer** — token-hash lookup → `ViewQuery WHERE token_hash = ?`, low effort once Tenant adopted                                                                                                                        |
| **MembershipReadModel** | 3                    | `byActor` (1→many)           | ❌ Hard delete | **Moderate**  | **Defer** — `FindByActor` is a natural `ViewQuery`, but no tombstone marking means soft-delete semantics would need adding                                                                                              |
| **UserReadModel**       | 12                   | `emails`, `externalAccounts` | ✅ Marked      | **Hard**      | **Reject** — `externalAccounts` is a composite (provider+subject) index that doesn't map to a SQL column or `ViewQuery`; the 12-event switch moves but doesn't shrink; in-memory maps with read-your-writes are simpler |
| **CasbinProjection**    | 12 (cross-aggregate) | External store (Casbin)      | ❌             | **N/A**       | **Never** — side-effecting projection mutates an external authorization store; has no `V` state of its own; fundamentally incompatible with Materialize's value-store model                                             |

## Decision

### 1. Provide a generic MaterializeProjection adapter (not a per-type prototype)

The initial prototype (`MaterializedTenantReadModel`) was **removed** — it was
a ghost system (unreachable from any setup path), redundant with the existing
`SQLTenantReadModel`, and created two split brains (`IsTombstoned()` vs `Deleted`,
incompatible `FindByID` signatures). See the self-review report at
`docs/reviews/2026-06-28_09-30_brutal-self-review-materialize-evaluation.html`
for details.

Instead, a **generic `MaterializeProjection[V, K]` adapter** was built
(`es_materialize_adapter.go`). It wraps ANY `stack.Materialize` as
`event.Projection` using `watermill.EventToMessage` → `Materialize.HandlerFunc`
round-trip — zero dispatch replication, automatically tracks upstream changes.
Any read model can adopt Materialize with a single `NewMaterializeProjection`
call.

### 2. Per-read-model fit unchanged

The per-read-model decision matrix (below) still holds: Tenant/Bot are good fits,
Membership is moderate, UserReadModel and CasbinProjection are rejected. The
adapter makes adoption a one-liner for any read model that fits.

### 3. Reject for UserReadModel and CasbinProjection

UserReadModel's composite external-accounts index and 12-event complexity make
the hand-written projection simpler and more maintainable. CasbinProjection is
architecturally incompatible (side-effecting, no value state).

### 4. Update ADR-0016 assessment

ADR-0016's blanket rejection ("too heavy for a library") is superseded by this
per-read-model analysis. The rejection holds for UserReadModel but not for
TenantReadModel.

## Consequences

### Positive

- **Generic adapter** is reusable — any read model that fits Materialize's
  OnCreate/OnUpdate/OnTombstone model can adopt it with zero boilerplate
- **Zero dispatch replication** — uses upstream's own `HandlerFunc` via
  `EventToMessage` round-trip; automatically benefits from upstream changes
- **Evaluation is evidence-based** — working code + tests, not speculation
- **Future migration path** is clear: when go-cqrs-lite ships Materialize as
  `event.Projection`, the adapter becomes a thin pass-through

### Negative

- **Adapter adds one indirection** per event (event → message → event round-trip).
  Negligible for projection handling; not in a hot path.
- **Adapter is not wired into setup** — consumers who want Materialize must
  create the projection themselves. This is intentional (library principle:
  don't enforce defaults).

### Neutral

- **No breaking changes** — all existing read models are unchanged
- **No new dependencies** — `stack/v3`, `kv/v3`, and `watermill` were already
direct or indirect deps

## When to Revisit

- **go-cqrs-lite v3.2.0+** (or whenever Materialize gains `event.Projection`
  conformance): the adapter's `Handle` can delegate directly to `mat.Handle()`
  instead of the EventToMessage round-trip
- **If UserReadModel's external-accounts index is refactored** to a separate
  view table (enabling `ViewQuery`): reconsider Materialize for User
- **If a second read model adopts the adapter** (Bot/Membership): add
  `IsTombstoned()` to the read-model view type and test with a real SQL store
