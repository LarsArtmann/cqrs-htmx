# ADR-0027: decide() stays on the server (Queue-Only client)

**Date:** 2026-06-28
**Status:** Accepted
**Supersedes:** Q1 in [ADR-0025](0025-phase2-research.md) (which framed this as an open question)

## Context

ADR-0025 laid out three options for where domain validation (`decide()`)
should run in the offline-first command sync architecture:

- **Queue-Only** — client queues commands blindly; server validates on sync.
- **WASM port** — compile Go `decide()` to WASM; client validates locally.
- **TypeScript port** — rewrite `decide()` in TS for client-side validation.

ADR-0025 recommended Queue-Only as the "simplest MVP." This ADR upgrades
that recommendation to a **definitive architectural decision** with a
stronger justification that ADR-0025 did not articulate.

## Decision

**Queue-Only is not just the MVP — it is the only architecturally correct
choice for cqrs-htmx as a library.**

The client never runs `decide()`. The library provides the queue, the sync
transport (SSE/ACK), and the honest UI states (pending → confirmed/rejected).
Pre-validation is explicitly out of scope.

### Why Queue-Only is the library-correct choice

1. **cqrs-htmx is a library, not an application.** Consumers write their
   *own* `decide()` functions for *their* domain. The library cannot ship a
   WASM or TS port of `decide()` because the library does not own the
   consumer's domain logic. Only `usermgmt`'s decide functions exist in this
   repo, and those are a consumer-facing feature, not a library mechanism.

2. **WASM/TS port is a consumer concern, not a library concern.** If a
   consumer wants offline pre-validation, *they* compile *their* decide
   functions to WASM. The library's job is to provide the transport protocol
   (command ID → ACK → honest UI state), not to dictate validation strategy.

3. **The honest UI already handles the failure case gracefully.** The
   pending → rejected transition (ADR-0024) is a first-class UI state. A
   command that fails server-side validation shows "rejected" with the
   server's error message. This is not a degraded UX — it is the correct UX
   for a system where the server is the source of truth.

4. **The decide functions are already pure** (event-sourcing decider
   pattern: `decide(state, cmd) → ([]event, error)`, no I/O). This means
   WASM is *always available as a future option* for any consumer who wants
   it, without restructuring the library. The door is open; we just don't
   force consumers through it.

5. **No client/server validation divergence.** Queue-Only has one source of
   truth for validation: the server. WASM/TS ports create a second source
   that must be kept in sync, with logic drift as an ever-present risk.

## Consequences

### Positive

- **Unblocks all Phase 2 work immediately.** No WASM compilation pipeline,
  no TS port, no dual-maintenance. The client just needs a command queue +
  SSE listener + honest UI.
- **Zero client-side domain code.** Smaller bundle, no reverse-engineering
  of business rules from client WASM.
- **Single validation source.** The server is authoritative. No drift.
- **Consumer extensibility preserved.** A consumer who wants WASM
  pre-validation can add it themselves; the library doesn't block it.

### Negative

- **No instant offline validation feedback.** A command queued offline
  shows "pending" until reconnect, then may flip to "rejected" if the
  server disagrees. This is acceptable for most UX requirements and is the
  standard behavior of optimistic-offline systems (e.g., mobile apps).

### Neutral

- **Future WASM cookbook.** If consumers ask for it, we can write a
  `docs/cookbook/offline-validation-wasm.md` recipe showing how to compile
  usermgmt's (or their own) decide functions to WASM. This is
  documentation, not library code.

## What this means for Phase 2 implementation

Phase 2 client work is now scoped to:

1. **Command queue** — buffer commands while offline, replay on reconnect.
2. **SSE listener** — consume ACKs from the server, flip honest UI states.
3. **Idempotency** — client generates `X-Command-Id` per command; the
   server's `IdempotencyStore` (ADR-0026) prevents double-execution on
   replay.
4. **Persistence** (Q2, separate decision) — whether the queue survives a
   closed tab.

No WASM, no TypeScript domain port, no client-side validation logic.

## References

- [ADR-0023: Command sync](0023-command-sync.md) — the sync protocol
- [ADR-0024: Honest UI](0024-honest-ui.md) — pending/confirmed/rejected states
- [ADR-0025: Phase 2 research](0025-phase2-research.md) — original options analysis
- [ADR-0026: Idempotency store](0026-command-idempotency-store.md) — duplicate prevention
