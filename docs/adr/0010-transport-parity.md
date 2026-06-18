# ADR 0010: Transport Parity (SSE ↔ WebSocket)

## Status: Accepted

## Context

ADR 0004 established SSE and WebSocket as first-class transports with basic
building blocks: `SSEEvent`, `SSEStream`, `Broadcaster`, `ParseWSMessage`,
and `WSOOBHTML`.

However, the two transports were **asymmetric**:

| Capability               | SSE      | WebSocket |
| ------------------------ | -------- | --------- |
| Error feedback to client | Missing  | Missing   |
| Proxy idle prevention    | Missing  | N/A       |
| Disconnect cleanup       | Missing  | N/A       |
| Fan-out                  | Yes      | Missing   |
| CQRS bridge (success)    | Yes      | Missing   |
| CQRS bridge (error)      | Missing  | Missing   |
| Outbound encoder         | N/A      | Missing   |
| Dispatch from client     | Via HTTP | Missing   |
| Transport-agnostic error | Missing  | Missing   |

The `AfterDispatchHook` already received the dispatch `error` at all four
call sites in `handler.go` — but `BroadcastOnSuccess` ignored it. SSE clients
learned when commands succeeded but never when they failed. WebSocket had no
fan-out at all and no way to dispatch commands from incoming messages.

## Decision

Close every gap in the matrix above with **new exports only** — zero breaking
changes, zero new dependencies.

### 1. StructuredError (RFC 7807)

`StructuredError` follows the RFC 7807 Problem Details shape: `type`, `title`,
`status`, `detail`, `instance`. `NewStructuredError(err, r)` maps via `MapError`
for the status and `event.Classify` for the type token. The `instance` field
carries the request ID for tracing.

**Why RFC 7807:** It is the HTTP standard for error payloads. Clients parse
it uniformly regardless of transport — the same JSON works as an SSE event
data field, a WebSocket message, or an HTTP response body.

### 2. BroadcastOnError / BroadcastOnErrorFunc

Mirror `BroadcastOnSuccess` exactly but fire when `err != nil`. Broadcasts
a `StructuredError` JSON string as the SSE event data.

`BroadcastOnError` takes only `eventName` (no `data` param). The data always
comes from `NewStructuredError(err, r)` — a consumer-supplied `data` string
would be silently dropped, which is a dishonest API. `BroadcastOnErrorFunc`
exists for custom error event generation.

### 3. SSEStream.Heartbeat / OnDisconnect

- `Heartbeat(ctx, interval)` sends SSE comment frames (`: keepalive\n\n`) on
  a ticker. Browsers ignore comments, but reverse proxies (Nginx, Cloudflare,
  AWS ALB) reset their idle timers. Prevents 30–60s silent disconnects.
- `OnDisconnect(fn)` registers cleanup callbacks fired on `Close()`. Enables
  metrics, logging, session deregistration without wrapping the stream.

### 4. WSBroadcaster

Mirrors SSE `Broadcaster` API exactly: `Subscribe`, `Unsubscribe`, `Broadcast`,
`SubscriberCount`, `BroadcastHTML`. Same O(1) unsubscribe via channel pointer
identity. Same buffered channels (64). Same non-blocking broadcast (drops to
slow consumers).

**Why broadcast `string` not `WSMessage`:** Consumers write via their own WS
library's API (`conn.WriteMessage(websocket.TextMessage, []byte(msg))`). The
broadcaster only does fan-out of the already-encoded message string.

### 5. WS Encoder (WriteWSMessage / WriteWSMessageInto[T])

Counterparts to `ParseWSMessage` / `ParseWSMessageInto[T]`. Round-trip
verified: encode then parse yields identical data.

### 6. WS Dispatch Bridge (DispatchWSCommand / DispatchWSQuery)

App methods that decode raw WS message bytes → dispatch via the App's
command/query dispatcher. Run before/after hooks, apply timeout, return
`(error)` or `(result, error)`.

**Why separate from HTTP path:** WS connections are authenticated at upgrade
time, don't need CSRF protection (origin is checked during upgrade), and
responses are written by the consumer's WS library — not via
`http.ResponseWriter`. Including auth/CSRF/response-writing would be dead
code that misleads.

**Why pass `*http.Request`:** The upgrade request carries context (user ID,
request ID, correlation ID) used by lifecycle hooks and `NewStructuredError`.
The consumer always has it (gorilla and coder both provide it at upgrade).

### 7. DecodeWSJSON[T] / DecodeWSJSONQuery[T]

Decoder factories that unmarshal JSON into a typed struct then map to
`command.Command` / `query.Query`. Mirrors the HTTP `DecodeJSON[T]` pattern
but operates on raw bytes instead of `*http.Request`.

## Rationale

1. **Existing hook already had the error** — `AfterDispatchHook(ctx, r, err)`
   received `err` at every call site. BroadcastOnError just stopped ignoring it.
2. **WSBroadcaster mirrors SSE Broadcaster** — consumers who know one pattern
   immediately know the other. Same Subscribe/Unsubscribe/Broadcast lifecycle.
3. **All additions are new exports** — zero breaking changes. Consumers who
   don't use the new features pay no cost.
4. **No new dependencies** — SSE is pure HTTP. WS protocol helpers are pure
   JSON. StructuredError is pure Go structs + `encoding/json`.
5. **Transport-agnostic error** — StructuredError works identically over HTTP,
   SSE, and WS. One shape, three transports.

## Consequences

- SSE and WebSocket now have **full feature parity** for real-time feedback.
- Consumers can compose success + error hooks manually (AfterDispatch takes a
  single hook; use BroadcastOnSuccessFunc/BroadcastOnErrorFunc or a custom
  closure that checks `err`).
- WSDispatchHandler deliberately omits auth/CSRF — consumers must authenticate
  at the WS upgrade handshake, not per-message.

## What This Is NOT

- NOT a shared Broadcaster interface — SSE and WS broadcasters have identical
  APIs but different channel types (`chan SSEEvent` vs `chan string`). A
  `FanOut[T]` generic could unify them but hasn't been needed.
- NOT a WS upgrade handler — consumers still choose their own WS library.
- NOT per-message WS auth — authentication happens at upgrade time.
