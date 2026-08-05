# ADR 0046: Drop WebSocket Transport in Favor of SSE

## Status: Accepted

## Context

ADR 0004 introduced SSE and WebSocket as first-class transports; ADR 0010 then
established full transport parity (SSE ↔ WebSocket) for CQRS dispatch feedback,
ACK, and broadcaster hooks. In the years since, the WS surface grew to include
fourteen exported symbols (`WSBroadcaster`, `WSMessage`, `WSOOBHTML`,
`ParseWSMessage`, `ParseWSMessageInto`, `WriteWSMessage`, `WriteWSMessageInto`,
`DispatchWSCommand`, `DispatchWSQuery`, `DecodeWSJSON`, `DecodeWSJSONQuery`,
`BroadcastOnSuccessWS`, `BroadcastOnErrorWS`, `BroadcastOnAckWS`), one embedded
JS asset (`extensions/ws.min.js`), and a constant (`HTMXExtWS`).

SSE covers every use case the WS surface was designed for, with strictly
better operational properties:

| Property            | SSE                          | WebSocket                                                      |
| ------------------- | ---------------------------- | -------------------------------------------------------------- |
| Protocol complexity | HTTP/1.1 streaming           | Separate framing on top of HTTP upgrade                        |
| Auth/cookies/CSRF   | Native HTTP semantics        | Custom handshake + auth at upgrade                             |
| Proxy/CDN traversal | Plain HTTP                   | Often blocked, requires sticky sessions                        |
| Reconnection        | Built-in `Last-Event-ID`     | Manual reconnect + replay                                      |
| Backpressure        | Standard HTTP flow control   | Manual; buffer bloat on slow consumers                         |
| Server complexity   | `stream.Send()` per client   | Per-conn goroutine + read/write pumps                          |
| Browser API         | `EventSource` (auto-retry)   | `WebSocket` (no auto-retry)                                    |
| HTMX extension      | `hx-ext="sse"` (server push) | `hx-ext="ws"` (bi-directional, rarely used server→client only) |
| htmx 4 status       | Continued investment         | Sunset (htmx 4 ships sse as core)                              |

The bi-directional capability WS offers is not exercised by any cqrs-htmx
consumer pattern: commands and queries are dispatched through the standard HTTP
pipeline (`App.Command` / `App.Query` + `DecodeJSON`), which already handles
bi-directional traffic with full auth, CSRF, and content negotiation. The WS
dispatch bridge existed to satisfy htmx-ext-ws forms, but the equivalent
`hx-ext="sse"` form covers the same UX with strictly less ceremony.

## Decision

Remove the entire WS transport from the cqrs-htmx root module:

- Delete `ws.go`, `ws_broadcaster.go`, `ws_dispatch.go`, `ws_encoder.go`
  (and all `_test.go` siblings + `example_ws_test.go`).
- Remove `HTMXExtWS` and the embedded `extensions/ws.min.js` asset.
- Remove `BroadcastOnAckWS`, `BroadcastOnAckWSFunc`, and `ackToWSMessage` from
  `ack.go`.
- Update all `// comment` text referencing "WS" or "WebSocket" to describe the
  SSE-only transport.

The DELETE removes both the inbound `hx-ext="ws"` decoder/encoder and the
bi-directional `DispatchWSCommand`/`DispatchWSQuery` bridge. Consumers who need
bi-directional low-latency transport should use the standard HTTP dispatch
pipeline (the same `App.Command`/`App.Query` endpoints that htmx already drives
over SSE for server→client updates).

## Consequences

### Positive

- ~1,200 LOC of source + tests deleted from the root module
- One bundled JS asset removed (no more binary-asset tracking issues)
- Surface area of public API shrinks by 14 exported symbols
- Operational guidance collapses to a single transport with native HTTP
  semantics (auth, CSRF, caching, observability, proxy/CDN behavior)
- Future htmx 4 migration is simpler (htmx 4 makes SSE a first-class core
  extension; the WS extension is sunset)
- Coverage of the root module improves (93.1% vs prior baseline)

### Negative

- **Breaking change.** Consumers importing `cqrshtmx.NewWSBroadcaster`,
  `cqrshtmx.WSMessage`, `cqrshtmx.ParseWSMessage`, `cqrshtmx.WriteWSMessage`,
  `cqrshtmx.WSOOBHTML`, `cqrshtmx.DispatchWSCommand`/`DispatchWSQuery`,
  `cqrshtmx.BroadcastOnSuccessWS*`, or `cqrshtmx.BroadcastOnAckWS*` will see
  compile errors and need to migrate to the SSE equivalents. A clear migration
  path is documented in the skill and `docs/guides/`:
  - `WSBroadcaster.Broadcast(msg)` → `Broadcaster.Broadcast(sse.Event{Data: msg})`
  - `ParseWSMessage(data)` → `json.Unmarshal` into a typed struct (HEADERS
    blob is HTMX-specific; consumers can extract via
    `sse.NewStream(r).Send(...)` round-trip or via the JSON header convention
    used by the SSE htmx extension)
  - `WSOOBHTML` → `OOBHTML` (already a 1-line alias)
  - `DispatchWSCommand`/`DispatchWSQuery` → `app.Command`/`app.Query`
    endpoints (HTTP, full auth/CSRF/content-negotiation support)
- **No backward-compat aliases.** Unlike the SSE re-export and httputil
  deprecation work, this removal is unconditional: WS code is a separate
  paradigm, and keeping aliases would re-introduce the very surface we are
  deleting.

### Mitigation

The change ships in a single breaking-release window. The skill
(`.agents/skills/cqrs-htmx/SKILL.md`) is updated to describe the SSE-only
transport and the migration recipe for each removed symbol. `CHANGELOG.md`
carries a prominent "Removed" entry under the next version.
