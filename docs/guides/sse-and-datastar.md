# SSE and Datastar Broadcaster Guide

> How cqrs-htmx's `Broadcaster` and datastar's `Broadcaster` relate, and how to share one fan-out hub across both transports.

## The Hub Comes First

The canonical shareable object is go-sse's [`*sse.Broadcaster[sse.Event]`](https://github.com/larsartmann/go-sse) — the fan-out hub. It provides subscribe, broadcast, health, graceful shutdown, buffer sizing, predicate filtering, and replay. Everything else in this guide is a thin transport adapter **over** that hub:

| Type                          | Module                | Transport    | Purpose                                                                                                              |
| ----------------------------- | --------------------- | ------------ | -------------------------------------------------------------------------------------------------------------------- |
| `*sse.Broadcaster[sse.Event]` | go-sse                | (none — hub) | Core fan-out: Subscribe, Broadcast, SubscribeFilter, Health, Shutdown, replay plumbing                               |
| `cqrshtmx.Broadcaster`        | Root (`cqrs-htmx/v4`) | HTMX SSE     | CQRS dispatch-hook constructors (`BroadcastOnSuccess`, `BroadcastOnError`) + `ServeSSE` lifecycle helper             |
| `datastar.Broadcaster`        | `datastar/v4`         | Datastar SSE | Patch ergonomics (`Broadcast(patch)`, typed patch constructors) + `http.Handler` mount + optional replay ring buffer |

Both adapters **embed** `*sse.Broadcaster[sse.Event]`, so the full go-sse method set (including `SubscribeFilter`, `Health`, `Shutdown`, `OnSubscribe`, `OnUnsubscribe`) is promoted and callable directly on either adapter.

## Architecture

```
                ┌─────────────────────────────────┐
                │     sse.Broadcaster[sse.Event]   │  ← the hub (go-sse)
                │  Subscribe / Broadcast / Health   │
                │  Shutdown / OnSubscribe / etc.   │
                └──────────────┬──────────────────┘
                               │ (one shared pointer)
           ┌───────────────────┴───────────────────┐
           │                                       │
┌──────────┴──────────┐              ┌─────────────┴──────────────┐
│ cqrshtmx.Broadcaster │              │  datastar.Broadcaster      │
│  (embeds the hub)    │              │  (embeds the hub)          │
│                       │              │                            │
│  + BroadcastOnSuccess │              │  + Broadcast(patch)        │
│  + BroadcastOnError   │              │  + BroadcastMany           │
│  + ServeSSE           │              │  + ServeHTTP               │
│  + Hub()              │              │  + Hub()                   │
└───────────────────────┘              └────────────────────────────┘
```

### Key properties

- Both adapters **embed** `*sse.Broadcaster[sse.Event]`, so all go-sse methods are promoted and accessible directly (`b.Subscribe()`, `b.SubscribeFilter(pred)`, `b.Health()`, ...).
- `Broadcast(patch Patch)` on the datastar adapter shadows the hub's `Broadcast(sse.Event)` — that is intentional (patch ergonomics). To send a raw event through the datastar adapter, use `BroadcastEvent` or the hub directly.
- `Hub()` on either adapter returns the embedded hub — the canonical handle for sharing.

## The Hub() Accessor

Both adapters expose their embedded hub via `Hub()`:

```go
// Root
b := cqrshtmx.NewBroadcaster()
hub := b.Hub() // *sse.Broadcaster[sse.Event]

// Datastar
dsb := ds.NewBroadcaster()
hub := dsb.Hub() // *sse.Broadcaster[sse.Event]
```

Use `Hub()` when you need:

- **`SubscribeFilter`** — predicate-based subscription (promoted on both adapters now, but `Hub()` documents that you are using go-sse directly).
- **Cross-transport hub sharing** (see below).
- **Passing the hub to lean helpers** that take `*sse.Broadcaster[sse.Event]` (e.g., `transport.ServeDomainEvents`).

## Cross-Transport Hub Sharing

When your application serves both HTMX and Datastar clients, share a single hub so events published from one transport reach subscribers of the other:

```go
package main

import (
    "net/http"

    cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
    ds "github.com/larsartmann/cqrs-htmx/datastar/v4"
    "github.com/larsartmann/go-sse"
)

func main() {
    // 1. Create one shared fan-out hub
    hub := sse.NewBroadcaster[sse.Event]()

    // 2. Wrap it for each transport
    htmxBroadcaster := cqrshtmx.NewBroadcasterFromHub(hub)
    dsBroadcaster := ds.NewBroadcasterFromHub(hub)

    mux := http.NewServeMux()

    // HTMX SSE endpoint
    mux.HandleFunc("GET /htmx/events", htmxBroadcaster.ServeSSE)

    // Datastar SSE endpoint
    mux.Handle("GET /ds/events", dsBroadcaster)

    // Broadcasting from either wrapper reaches ALL subscribers
    // (both HTMX and Datastar clients)
    htmxBroadcaster.Broadcast(sse.Event{Event: "update", Data: "<div>Hi</div>"})
    // ↑ also received by Datastar SSE clients on /ds/events
}
```

### Why share?

| Approach              | Subscribers                                                                  | Event copies                 |
| --------------------- | ---------------------------------------------------------------------------- | ---------------------------- |
| Separate broadcasters | HTMX clients see HTMX events only; Datastar clients see Datastar events only | 2 broadcast calls per update |
| Shared hub            | All clients see all events                                                   | 1 broadcast call per update  |

Sharing is the right choice when HTMX and Datastar pages display the same real-time data (e.g., a live dashboard with both an HTMX table view and a Datastar chart view).

### When NOT to share

If HTMX and Datastar clients consume **different event shapes** (HTML fragments vs. JSON patches), keep separate broadcasters. Mixing event types on one hub means each client must filter irrelevant events, adding complexity.

## Deprecated Raw API (removal bundled with v5)

The pre-hub vocabulary framed the hub as a "raw" escape hatch. It is deprecated in favor of the hub-first API:

| Deprecated (until v5)                 | Use instead                                         |
| ------------------------------------- | --------------------------------------------------- |
| `b.Raw()`                             | `b.Hub()`                                           |
| `cqrshtmx.NewBroadcasterFromRaw(hub)` | `cqrshtmx.NewBroadcasterFromHub(hub)`               |
| `ds.NewBroadcasterFromRaw(hub)`       | `ds.NewBroadcasterFromHub(hub)`                     |
| `cqrshtmx.RawBroadcaster` interface   | Pass `*sse.Broadcaster[sse.Event]` (the hub itself) |

The deprecated symbols remain functional through v4; staticcheck flags call sites. The datastar adapter's hub was previously a hidden unexported field (`inner`) with hand-written pass-throughs — it is now embedded, so `Subscribe`/`SubscribeFilter` and friends promote for free.

## Choosing a Transport

| Criterion          | HTMX                                                                 | Datastar                                                    |
| ------------------ | -------------------------------------------------------------------- | ----------------------------------------------------------- |
| **Philosophy**     | Server renders HTML fragments                                        | Server sends signals, client morphs DOM                     |
| **Bundle size**    | 0 JS (uses HTMX, loaded separately)                                  | 11.76 KiB (`datastar.js`)                                   |
| **Reactive state** | None (server is source of truth)                                     | Signals (client-side reactive state)                        |
| **Use when**       | CRUD forms, tables, navigation                                       | Live dashboards, interactive widgets, optimistic-feeling UI |
| **Coexistence**    | Both can run in the same app — different routes, different endpoints | Same                                                        |

For full-stack wiring with both transports, see the [Full-Stack Wiring Guide](fullstack-wiring.md).

## CORS and Cross-Origin SSE

SSE endpoints (`/sse` from setup, `/-/events/stream` from dashboardui) are
**same-origin by default**. No CORS headers are sent — the library principle
of "never enforce defaults consumers might disagree with" applies.

If your SSE consumers are on a different origin (e.g., a dashboard served from
`dashboard.example.com` connecting to `api.example.com/sse`), wrap the SSE
handler with [`httputil.CORS`](https://github.com/larsartmann/httputil) before
mounting:

```go
import "github.com/larsartmann/httputil"

mux.Handle("/sse", httputil.CORS(httputil.DefaultCORSConfig())(bundle.sseHandler()))
```

For the dashboard's built-in SSE endpoint, wrap the entire dashboard handler
or mount a CORS middleware in front of the dashboard mux.

**Note:** SSE requires the `text/event-stream` content type and `Connection:
keep-alive`. Ensure your CORS config allows these headers. The default
`httputil.DefaultCORSConfig()` permits all methods and headers; tighten it
with `CORSConfig{AllowedOrigins: []string{"https://dashboard.example.com"}}`
in production.

## Scoped Feeds: WithSSEFilter

`transport.ServeDomainEvents` accepts `transport.WithSSEFilter(pred)` to
restrict BOTH stream paths — live delivery and journal replay — to events
matching a predicate. This is the mechanism behind stream-type-scoped SSE
endpoints (the open `/sse` authz-posture decision, see
`docs/planning/2026-08-30_sse-endpoint-shape-decision.md`): the domain
envelope's stream type lives in the payload, so a predicate on it is all a
scoped endpoint needs.

```go
userOnly := func(evt sse.Event) bool {
    return strings.Contains(evt.Data, `"streamType":"user"`)
}
h := transport.ServeDomainEvents(broadcaster, store, heartbeat,
    transport.WithSSEFilter(userOnly))
```

Fail-closed by construction: the live path subscribes via go-sse's
`SubscribeFilter`, and replay goes through a store wrapper so a reconnecting
client's backfill only ever receives matching events — even when the store
only implements plain `sse.EventStore`. A filter that leaked excluded events
during backfill would be a security hole, never a degradation.

## See Also

- [Datastar Integration Guide](datastar-integration.md) — Datastar setup, patches, replay, SDK re-exports
- [Full-Stack Wiring Guide](fullstack-wiring.md) — composing all cqrs-htmx sub-modules
- [Production Readiness](production-readiness.md) — SSE in production (heartbeats, graceful shutdown)
- [go-sse](https://github.com/larsartmann/go-sse) — the underlying SSE library and the hub itself
