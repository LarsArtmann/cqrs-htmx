# ADR 0058: Defer Relationship-Based Authorization (ReBAC)

**Status:** Accepted
**Date:** 2026-11-02
**Related:** [ADR 0057](0057-authorization-contract.md) (authorization contract), [ADR 0044](0044-casbin-first-class-dependency.md) (Casbin in identity-model), [2026-11-02 consumer audit](../status/2026-11-02_10-13_consumer-usage-audit-and-authz-model_status.md)

## Context

The 2026-11-02 audit asked which permission model this ecosystem actually needs.
It compared RBAC (shipped), ABAC, PBAC, OrBAC, Bell–LaPadula multilevel, and
ReBAC, then measured demand instead of guessing.

Demand measurement:

- Every service that gates anything asks a **role-in-domain** question: does this
  actor hold role R in tenant T?
- Zero services ask a traversal question ("can A read D because A owns B, B
  belongs to C, and C owns D?").
- Two services wanted a *global* admin (`super_admin` in `*`) and got it from the
  existing model.
- No service asked for attribute predicates over resource state, and none asked
  for information-flow labels.

So the shipped RBAC-in-domain model is not being outgrown; it is under-used. The
gap the audit found was write-path durability, contract shape, and proof — not
expressiveness. A graph engine would have added operational surface (a tuple
store, consistency questions, recursive query cost) to solve a problem nobody
reported, while the real defect (grants that vanish on restart) stayed unfixed.

## Decision

**Do not implement a relationship-based authorization engine now.**

Instead:

1. **Reserve the seam.** `cqrshtmx.Resource` carries `Type` and `ID` as separate
   fields, so a future engine can key its edges on the object's kind without a
   breaking change to any caller. The string form (`type:id`) stays exactly what
   today's Casbin object matcher sees.
2. **Keep authorization in-process.** No network policy decision point. A remote
   PDP adds a failure mode (PDP down means everything is denied or everything is
   allowed) with no compensating benefit at this scale, and every current caller
   already has the policy state in memory.
3. **Record the trigger** that would reopen this decision.

## Trigger (numeric, not aspirational)

Reopen this ADR when **either** holds:

- **Two or more services** need a permission question whose answer requires
  traversing more than one relationship edge (owner → folder → document class,
  team → repo → branch), or
- **one service** issues such traversals on a hot path more than 100 times per
  request, which makes materializing them into flat roles impractical, or
- a service must grant access **authored by a tenant at runtime** (a customer
  shares a document with another customer's user), which no fixed role
  vocabulary can express.

Until then, the answer to "we need ReBAC" is a new role or a new domain, and the
cost of that is one `GrantRoles` call.

## Consequences

- The vocabulary stays small: five roles, one domain axis, one wildcard.
- `Resource.ID` is currently unused by the shipped model. It is carried, and
  documented as the reservation, so that adding it later is additive.
- If the trigger fires, the migration path is: add a tuple projection next to the
  Casbin projection, implement `Check` against tuples first with a role fallback,
  and keep `Request`/`Decision` unchanged. The contract does not change; only the
  engine behind it does.

## Rejected alternatives

- **Build ReBAC now, migrate consumers later.** Consumers migrate to nothing —
  the audit shows they have not adopted even the shipped model.
- **Adopt an external authorization service (SpiceDB/OpenFGA) now.** Same
  argument plus an operational dependency.
- **Attribute-based policies instead.** No measured demand, and ABAC's predicate
  surface is harder to reason about than a role set for the questions actually
  being asked.
