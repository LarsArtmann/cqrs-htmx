# ADR-0039: Unify ActorID Shape (Proposed)

## Status

PROPOSED — 2026-07-19 (deferred to a coordinated v5 major bump)

This ADR records the decision to unify the two `ActorID` shapes and the
rationale for deferring execution.

## Context

The same concept — "who performed this action" — has two incompatible shapes:

- **Root module** (`cqrshtmx`): `ActorID = brandid.ID[actorBrand, string]` — a
  flat branded string. `NewActorID("...")` constructs it.
- **usermgmt module**: `ActorID` is a kind-discriminated struct
  (`{Kind, Raw}`) so it can carry user/bot/tenant kinds. `NewActorID(kind, raw)`
  constructs it.

Both are exported and both appear in adminui's import graph (which imports
root + usermgmt), so consumers see two same-named types with different shapes
and different constructors. Cross-module conversion is required today and is a
source of confusion (AGENTS.md documents this gotcha).

## Decision (proposed)

**Adopt usermgmt's kind-discriminated struct as the canonical shape and apply it everywhere.**

Rationale: a flat branded string cannot distinguish a user actor from a bot
actor, which matters for audit and authorization. The discriminated struct is
the richer, more honest model. Root's `ActorID` becomes the same struct
(aliased or redefined), and `NewActorID` gains the kind parameter.

## Why deferred

- Root's `NewActorID("...")` is a public constructor; changing its signature is
  source-breaking for every consumer that constructs an ActorID.
- adminui + integration_test both reference the root shape and must be updated
  in lockstep.
- This belongs in the same v5 bump as the other breaking changes (T16, T17,
  T20, T23) so consumers migrate once.

## Consequences (when shipped)

- **Positive:** One ActorID shape across the whole library; no cross-module
  conversion; kinds are first-class.
- **Negative:** Breaking: root `NewActorID` signature changes; adminui and
  integration_test call sites update.
- **Mitigation:** Ship in a single coordinated v5 with a migration note in
  `docs/migrations/`.
