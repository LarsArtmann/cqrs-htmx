# Decision: /sse endpoint shape — session-gating vs stream filtering

**Status:** OPEN — awaiting the user's product/security call
**Raised:** 2026-08-30 (status report §g-2) · **Spike:** `transport/filtered_sse_spike_test.go`
**Scope:** `setup.Config.SSEPath` (the shared, session-gated SSE feed) and the `transport.ServeDomainEvents` lifecycle it mounts

## The question

`setup` can mount ONE shared `/sse` endpoint streaming every domain event
(bootstrapped from the journal, heartbeat, replay). Today it is session-gated
(401 without a session) but every authenticated session sees EVERY event. The
open decision: is that the published shape, or must scoping (stream-type
filtering, or per-user streams) land before the endpoint shape is "public"?

This is a product/security decision, NOT a feasibility question — the
mechanism for every option below is proven and (since 2026-08-30) shipped.

## Options

| Option                                            | What it means                                                                                                                                                                                                                                                                                                                       | Cost                                                                      | Risk                                                                                              |
| ------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| **A. Session-gate only (today's shape)**          | Any authenticated session receives all events. Events are observability data (state changes), not secrets-by-default.                                                                                                                                                                                                               | Zero — shipped.                                                           | If any tenant's events are considered sensitive, scope creep later breaks the published contract. |
| **B. Session-gate + stream-type filter (opt-in)** | Same endpoint; consumers pass `transport.WithSSEFilter(pred)` to scope the feed (e.g. one tenant's stream types). **Mechanism shipped 2026-08-30** (`transport.WithSSEFilter`: live via `SubscribeFilter`, replay fail-closed via a store wrapper — a filter that leaked backfill would be a security hole, so replay filters too). | One additive setup option wiring the pred (≈20 LOC) when wanted.          | Predicates are consumer code — a wrong pred over-shares within the chosen scope.                  |
| **C. Per-user / per-tenant streams**              | Separate endpoints or per-connection preds derived from the session (multi-tenant SaaS shape).                                                                                                                                                                                                                                      | Real work: session→scope mapping, per-pred subscriber pools, tests, docs. | Over-engineering for a library whose consumers may not be multi-tenant.                           |
| **D. Don't publish a shared /sse in setup**       | Remove/never-promote `SSEPath`; consumers compose their own SSE from `transport` + `usermgmt` pieces.                                                                                                                                                                                                                               | Negative (delete).                                                        | Breaks the bundled fullstack story; the admin sync indicator loses its feed.                      |

## Recommendation

**A now, B as the ready escape hatch, C only on demand.** Concretely:

1. Keep the endpoint session-gated and document the "authenticated = full
   feed" contract (single-tenant default; matches adminui/dashboardui needs).
2. `WithSSEFilter` is shipped in `transport` (2026-08-30), so a consumer who
   needs scoping adds one option — no library change needed.
3. If the user answers "the endpoint MUST be scoped before it's public",
   Option C is the follow-up; budget ~1 session (mapping + tests + guide).

## Evidence

- Spike: `transport/filtered_sse_spike_test.go` — live routing
  (`SubscribeFilter`) and reconnect/backfill (`ReplayFiltered` +
  `EventsAfterFiltered` wrapper) both work with zero go-sse changes.
- Productionized: `transport.WithSSEFilter` + `filteredEventStore`
  (`transport/serve.go`) — any `sse.EventStore` works, replay is fail-closed.
- Precedent: dashboardui and setup both delegate to the same
  `transport.ServeDomainEvents` lifecycle; a filter composes identically in
  both.

## Answer template (for the user)

> A (ship as-is) / B (add scoped option to setup) / C (per-user streams) / D (drop shared /sse)
