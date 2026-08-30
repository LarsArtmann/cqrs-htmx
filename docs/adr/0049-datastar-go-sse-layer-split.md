# ADR-0049: datastar/go-sse Layer Split — SDK Owns the Wire, go-sse Owns the Fan-out

## Status

ACCEPTED — 2026-08-31

## Context

The 2026-08-07 analysis (`docs/status/archived/2026-08-07_06-25_datastar-go-sse-analysis-self-review.md`)
concluded go-sse could not produce the Datastar wire format and therefore the
`datastar/v4` module should couple to the datastar-go SDK for everything. That
conclusion contained a technical error: go-sse HAS Datastar-format primitives
(`KeyedLines`/`SendKeyed`/`SendLines`, designed for the Datastar SSE protocol).
The self-review flagged it; the TODO item has been open since. This ADR
resolves it.

The real question was never "can go-sse encode Datastar frames" (it can) but
"which dependency should own which layer" of the datastar module's realtime
stack.

## Decision

Keep the two-layer split as it exists today (formalized since the v4.8.x
alignment train):

1. **The datastar-go SDK owns the wire format.** `Patch` values and the
   response builder's `ServerSentEvents` renderer are the protocol contract.
   Patch classes (MergeFragments, RemoveFragments, PatchSignals, ...) evolve
   with the Datastar specification; encoding them through the SDK keeps the
   module correct by construction and automatically correct when the SDK
   updates.

2. **go-sse owns transport and fan-out.** The Broadcaster EMBEDS
   `*sse.Broadcaster[sse.Event]` (the canonical "hub"): subscriber
   management, buffered channels, replay via `godatastar.MemoryStore`
   (`NewBroadcasterWithReplay`), heartbeats, and the `ServeSSE` lifecycle
   come from go-sse. The hub is shareable across transports: root
   `cqrshtmx.Broadcaster` and `datastar.Broadcaster` both expose `Hub()` /
   `NewBroadcasterFromHub(hub)`, so one fan-out tree can serve an HTMX
   endpoint and a Datastar endpoint simultaneously — each subscriber's
   frames are encoded by whichever layer that endpoint uses.

3. **Patch generation is NOT migrated to go-sse's `SendKeyed`/`KeyedLines`.**
   Doing so would fork the wire format from the SDK's typed `Patch` model:
   every Datastar spec change would need a manual re-encode here, and the
   response builder's contract (`Patch` in, correct frames out) would be
   lost. `SendKeyed` remains available for consumers hand-rolling custom
   frames through a hub they share with the datastar module.

### Rejected alternative: full migration to go-sse encoding

- Forks protocol correctness away from the SDK (spec drift becomes our bug).
- The module's value proposition IS "batteries-included, spec-correct
  Datastar" — the SDK coupling is the feature, not incidental.
- Nothing blocks cross-transport fan-out (solved by the shared hub), which
  was the actual motivation behind the original migration idea.

## Consequences

- The prior analysis's technical claim is formally corrected here; the
  archived document keeps its original text with the self-review annotation.
- go-sse's "CQRS dispatch hooks + ServeSSE handlers as non-goals" stance
  (its ROADMAP, v0.2.0) is respected: adapter glue (Broadcaster,
  EventBridge, response builder) lives in this repo.
- If the datastar-go SDK ever stalls, `SendKeyed`-based encoding is the
  documented escape hatch — a deliberate, scoped migration, not an
  architectural accident.
- `Raw()`/`NewBroadcasterFromRaw` remain deprecated (v5 removal bundle,
  ADR-0047 sibling); `Hub()` is the canonical accessor.
