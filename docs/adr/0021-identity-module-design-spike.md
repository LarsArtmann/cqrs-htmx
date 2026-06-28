# ADR 0021: Identity Module Design Spike — Resolving the ActorID Split Brain

**Date:** 2026-06-28
**Status:** Accepted (Design Spike)
**Related:** ADR 0015, ADR 0018, ADR 0019

## Context

The codebase has two incompatible types named `ActorID`:

- **Root module** (`cqrs-htmx/v3`): `type ActorID string` — a simple prefixed
  string (e.g. `"user:01JX..."`, `"bot:ci-deploy"`) stored in request context.
  Defined in `context.go:149`.
- **usermgmt module** (`cqrs-htmx/v3/usermgmt`): `type ActorID struct{ kind
  ActorKind; raw string }` — a kind-discriminated union with domain methods
  (`Kind()`, `String()`, `PrefixedString()`, `IsZero()`, `ParseActorID()`).
  Defined in `usermgmt/id.go:60`.

Root cannot import usermgmt (module boundary). The current bridge:
`string(usermgmtActorID)` → `root.ActorID` and `root.ParseActorID` →
`usermgmt.ParseActorID`. This works but provides no compile-time safety.

### Why They're Different

- **Root** needs ActorID as an opaque context value — store, retrieve, pass to
  event metadata. A string is sufficient and keeps zero dependencies.
- **usermgmt** needs ActorID as a domain value — discriminate between user and
  bot actors, derive Casbin subjects, validate kind at construction.

## Options Considered

### Option A: Extract a shared `identity/` module

Create `github.com/larsartmann/cqrs-htmx/identity/v3` containing `ActorID`,
`ActorKind`, constructors, and `ParseActorID`. Both root and usermgmt import it.

**Pros:** Single source of truth, compile-time safety across modules.
**Cons:** New module to maintain, root gains a dependency, breaking change for
consumers using root's `ActorID string` in context.

### Option B: Use `go-composable-business-types` branded types

Define `ActorID` as `id.ID[ActorBrand, string]` in a shared location.

**Pros:** Consistent with go-branded-id pattern already used for UserID/TenantID.
**Cons:** Loses the kind-discriminated union semantics that usermgmt needs.
ActorID is inherently a sum type (user | bot), not a simple branded string.

### Option C: Move ActorID upstream into `go-cqrs-lite/id/v3`

Both modules already depend on `go-cqrs-lite/id/v3` (for UserID, CorrelationID,
RequestID). Add ActorID there.

**Pros:** Natural home, both modules already depend on it.
**Cons:** ActorID with kind-discrimination is domain-specific to usermgmt, not
a generic identity concern. Pollutes the generic `id` module with domain types.

### Option D: Keep the bridge, document it

Accept the split as intentional. Root's `ActorID string` is the context-layer
representation. usermgmt's `ActorID struct` is the domain representation. The
bridge is explicit and well-tested.

**Pros:** Zero changes, zero breaking changes, no new modules.
**Cons:** No compile-time safety. A future developer might use the wrong
`ActorID` type. The split is a code smell.

## Decision

**Option D (keep the bridge, document it) — for now.**

The split is intentional and working. The bridge (`string()` → `ParseActorID`)
is tested and documented. The real fix (Option A) is a significant
architectural change that:

1. Creates a new module boundary to maintain
2. Introduces a breaking change for consumers
3. Should be batched with ADR 0019 (usermgmt decomposition) — both require
   restructuring module boundaries

### Trigger for Re-evaluation

Adopt Option A when EITHER:
- ADR 0019 unblocks (usermgmt decomposed into sub-packages)
- A consumer reports the ActorID type mismatch as a real bug (not theoretical)
- A third module needs ActorID (currently only root + usermgmt use it)

### Interim Improvements

- **Document the bridge pattern** explicitly in both type's godoc
- **Add a lint rule** that catches accidental string/struct confusion
- **Consider** making root's `ActorID` an opaque type (not raw `string`) with
  `String()` and `ParseActorID()` methods, keeping it string-backed but
  preventing accidental string concatenation

## Rationale

YAGNI. The split works, the bridge is tested, no consumer has reported issues.
Creating a shared identity module is premature until we know what else needs
to live there (tenant context? role context? session origin?). A module
extracted too early will have the wrong boundary and need re-extraction.

The how-to-golang skill recommends `go-composable-business-types/id.ID[Brand,V]`
for branded IDs, but ActorID is not a simple branded ID — it's a sum type
(user | bot). The kind-discrimination is essential to usermgmt's domain logic.
Forcing it into a branded-string shape would lose that safety.

## Consequences

- Root's `ActorID` remains `string` for context storage
- usermgmt's `ActorID` remains a kind-discriminated struct
- The bridge (`string()` / `ParseActorID`) remains the conversion mechanism
- This ADR documents the decision so future readers understand it's intentional
