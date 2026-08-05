# ADR 0004: SSE and WebSocket Support

## Status: Superseded by ADR-0046

> **Superseded** — WebSocket transport was dropped in v5; the library is
> SSE-only now (see [ADR-0046](0046-drop-websocket-sse-only.md)). The SSE
> building blocks (`SSEStream`, `Broadcaster`) described below remain valid.

## Context

HTMX provides two extensions for real-time updates:

1. **SSE Extension** (`hx-ext="sse"`): Server-Sent Events for uni-directional server→client updates.
   - Client: `sse-connect="<url>"`, `sse-swap="<event-name>"`
   - Server sends: `event: <name>\ndata: <html>\n\n`
   - Works over standard HTTP, no new dependencies

2. **WebSocket Extension** (`hx-ext="ws"`): Bi-directional communication.
   - Client: `ws-connect="<url>"`, `ws-send`
   - Server receives: JSON with form fields + HEADERS
   - Server sends: HTML fragments with OOB swap attributes
   - Requires a WebSocket library (consumer's choice)

Both FEATURES.md and ROADMAP.md previously listed these as "NOT_PLANNED". The datastar-demo shows a hand-rolled broadcaster, proving the need exists.

## Decision

Add SSE and WebSocket support to the root module:

### SSE Support (`sse.go`) — No new dependencies

- `SSEEvent`: Protocol-level SSE message (Event, Data, ID, Retry)
- `WriteSSEEvent(w, event)`: Low-level SSE protocol writer
- `SSEStream`: Manages a single SSE connection (headers, flush, context cancellation)
- `Broadcaster`: Thread-safe fan-out mechanism for distributing events to SSE clients

### WebSocket Protocol Helpers (`ws.go`) — No WebSocket library dependency

- `WSMessage`: Incoming HTMX WebSocket message (Headers + Body fields)
- `ParseWSMessage`: Parse incoming JSON into typed struct
- `WSOOBHTML`: Format HTML fragments with OOB swap attributes for HTMX
- Consumers bring their own WebSocket library (gorilla, coder, etc.)

## Rationale

1. **SSE is pure HTTP** — No external dependencies, aligns with the library's dependency-minimal philosophy
2. **WebSocket as protocol helpers** — No forced WebSocket library dependency; consumers choose their own
3. **Building blocks, not frameworks** — Provides the HTMX-specific protocol layer; consumers wire it to their event system
4. **Consistent with existing patterns** — Same flat-package, no-enforced-defaults philosophy
5. **The datastar-demo proves the pattern** — The broadcaster pattern already exists in our examples

## What This Is NOT

- NOT a full real-time framework (no reconnection management, no presence tracking)
- NOT tied to the App struct (consumers use SSE/WS independently)
- NOT a replacement for Datastar's SSE protocol (HTMX SSE ≠ Datastar SSE)
