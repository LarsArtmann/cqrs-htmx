# storage.RelationalProjection Evaluation

> Should we adopt `storage.RelationalProjection` for multi-table usermgmt read models?

**Date:** 2026-06-28 · **Status:** Evaluated — Defer

## What storage.RelationalProjection Provides

`storage.RelationalProjection` from go-cqrs-lite v3.1.0 provides:

- Declarative multi-table projections via `OnCreate`/`OnUpdate`/`OnTombstone`
- Automatic table management from `view:"col"` struct tags
- Built-in dedup and ordering guarantees

## Current Approach

Our read models use `event.Projection` interface (`Name()` + `Handle()` + `EventTypes()`).
Each read model has a custom `Handle()` with a switch statement over event types.

### UserReadModel Example

- 12-event switch statement
- Email index + external account index
- Tombstone handling
- `UserView` DTO for SQL persistence

### MembershipReadModel Example

- 3-event switch statement
- Actor + tenant indexes

## Evaluation

### Why Defer

1. **Complex event handling doesn't fit declarative model.** The UserReadModel
   handles 12 event types with cross-references (email index, external account
   index, credential list updates). Expressing these as `OnCreate`/`OnUpdate`/
   `OnTombstone` would require significant refactoring with no behavior change.

2. **SQL read models already work.** `SQLUserReadModel` and friends wrap the
   in-memory read models with SQL persistence via `storage.SQLViewStore`. The
   dual approach (in-memory for read-your-writes + SQL for persistence) works.

3. **Adoption requires rewriting 4 read models.** User, Membership, Tenant, Bot
   — each has custom event handling logic that would need conversion.

4. **Benefit is marginal.** The main benefit (declarative multi-table) matters
   when you have many tables per projection. Our read models are single-table
   (one table per aggregate type) with a JSON blob.

### When to Revisit

- When ADR 0019 unblocks (usermgmt decomposition into sub-packages)
- When a read model needs truly multi-table projection (e.g., joining users
  and memberships in a single projection)
- When `AutoMapper` proves insufficient for complex tombstone semantics

## Related

- [ADR 0019: usermgmt Decomposition — Blocked](../../docs/adr/0019-usermgmt-decomposition-blocked.md)
- go-cqrs-lite `storage.SQLViewStore` + `AutoMapper` documentation
