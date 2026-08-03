# ADR-0045: Datastar as Optional Frontend Adapter Module

## Status

ACCEPTED — 2026-08-03

## Context

cqrs-htmx uses HTMX for frontend reactivity. Consumers who need reactive state management (filters, toggles, form state, real-time dashboards) must add Alpine.js, hand-roll JavaScript, or use HTMX extensions.

[Datastar](https://data-star.dev/) offers an alternative: reactive **signals**, DOM **morphing** by default, a **structured SSE protocol**, and built-in retry. Its philosophy — backend as source of truth, SSE as default transport, no optimistic updates — is identical to cqrs-htmx's CQRS/event-sourcing architecture.

The question: how to add Datastar support without forcing every HTMX consumer to pull the Datastar dependency.

## Decision

Create `github.com/larsartmann/cqrs-htmx/datastar/v4` as a **fully isolated optional Go module** that wraps the [datastar-go](https://github.com/starfederation/datastar-go) SDK.

### Key design choices:

1. **Separate Go module** — Go modules don't support optional dependencies. A separate module with its own `go.mod` ensures HTMX consumers never download or compile the Datastar SDK.

2. **No root module dependency** — The datastar adapter depends only on `datastar-go`, `go-cqrs-lite/event/v4`, and `testify`. It does NOT depend on `cqrs-htmx/v4`. This keeps it lean (no casbin, httputil, go-sse transitive deps) and allows standalone use.

3. **Patch ring buffer replay** — Instead of coupling to root's `JournalSSEStore` (which replays domain events from an event journal), the Broadcaster maintains a bounded ring buffer of recent Datastar patches. This is simpler, more correct (replays exactly what was broadcast, not re-rendered events), and self-contained.

4. **Thin adapter pattern** — Every function wraps exactly one datastar-go SDK function. The module adds cqrs-htmx integration patterns (Broadcaster, EventBridge, Response builder) on top.

5. **Zero changes to root module** — The Verschlimmbessern safeguard ensures the stable HTMX API is never touched.

### Typed decoder limitation

Root's `handlerConfig` is unexported, so the datastar module cannot create `cqrshtmx.HandlerOption` values for typed signal decoders. Consumers use `ds.ReadSignals(r, &signals)` manually in their handlers instead. This is a root module design issue, not a datastar module issue — adding an escape hatch to root would be a separate decision.

## Consequences

### Positive

- HTMX consumers are completely unaffected
- Datastar consumers get a batteries-included adapter (script serving, signal decoding, response builder, broadcaster with replay, event bridge)
- The module follows existing patterns (separate go.mod, mirrors loginpage/dashboardui structure)
- Replay works without event store coupling (patch ring buffer)

### Negative

- Typed decoders returning `HandlerOption` are impossible without root changes
- No event-store-level replay (ring buffer is bounded; very old patches are lost)
- The module cannot use root's error mapping (`MapError`) or SSE types

### Mitigations

- `ds.ReadSignals(r, &signals)` is a one-liner — ergonomic enough for manual use
- Ring buffer default (256) covers typical reconnection scenarios
- `ds.ErrorResponse(w, r, err)` provides consistent error UX without root's error mapping

## Alternatives Considered

| Option | Verdict |
|---|---|
| Add Datastar to root go.mod | **Rejected** — forces every HTMX consumer to pull Datastar SDK |
| Abstract transport interface | **Rejected** — massive refactor of stable code for hypothetical benefit |
| Separate repo (`cqrs-datastar`) | **Rejected** — fragments the ecosystem, duplicates CQRS wiring |
| Depend on root module | **Rejected** — pulls casbin/httputil/go-sse unnecessarily; replay via JournalSSEStore is wrong abstraction (replays events, not patches) |
