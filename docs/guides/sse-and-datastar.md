# SSE and Datastar Broadcaster Guide

> How cqrs-htmx's `Broadcaster` and datastar's `Broadcaster` relate, and how to share one fan-out hub across both transports.

## The Two Broadcasters

cqrs-htmx has two SSE broadcaster types, one per frontend transport:

| Type | Module | Transport | Purpose |
| --- | --- | --- | --- |
| `cqrshtmx.Broadcaster` | Root (`cqrs-htmx/v4`) | HTMX SSE | Fan-out `sse.Event` to HTMX clients, with CQRS dispatch-hook constructors (`BroadcastOnSuccess`, `BroadcastOnError`) |
| `datastar.Broadcaster` | `datastar/v4` | Datastar SSE | Fan-out Datastar patches (elements, signals, redirects) to Datastar clients, with optional replay ring buffer |

Both wrap the same underlying type: `*sse.Broadcaster[sse.Event]` from [go-sse](https://github.com/larsartmann/go-sse). This is intentional — go-sse provides the core fan-out mechanics (subscribe, broadcast, health, graceful shutdown, buffer sizing, replay), and each Broadcaster adds transport-specific ergonomics on top.

## Architecture

```
                    ┌─────────────────────────────────┐
                    │     sse.Broadcaster[sse.Event]   │  ← go-sse (core fan-out)
                    │  Subscribe / Broadcast / Health   │
                    │  Shutdown / OnSubscribe / etc.   │
                    └──────────────┬──────────────────┘
                                   │
               ┌───────────────────┴───────────────────┐
               │                                       │
    ┌──────────┴──────────┐              ┌─────────────┴──────────────┐
    │ cqrshtmx.Broadcaster │              │  datastar.Broadcaster      │
    │  (embeds *sse.BC)    │              │  (wraps unexported inner)  │
    │                       │              │                            │
    │  + BroadcastOnSuccess │              │  + Broadcast(patch)        │
    │  + BroadcastOnError   │              │  + BroadcastMany           │
    │  + ServeSSE           │              │  + ServeHTTP               │
    │  + Raw()              │              │  + Raw()                   │
    └───────────────────────┘              └────────────────────────────┘
```

### Key difference

- **Root** `Broadcaster` **embeds** `*sse.Broadcaster[sse.Event]`, so all go-sse methods (`Subscribe`, `Unsubscribe`, `SubscribeFilter`, `Health`, `Shutdown`, etc.) are automatically promoted and accessible directly.
- **datastar** `Broadcaster` uses a **named unexported field** (`inner`), so only the explicitly delegated methods are public. This prevents accidental coupling to go-sse internals, but also hides `Subscribe` and `SubscribeFilter`.

The `Raw()` accessor bridges this gap for both types.

## The Raw() Accessor

Both Broadcasters expose their underlying `*sse.Broadcaster[sse.Event]` via `Raw()`:

```go
// Root
b := cqrshtmx.NewBroadcaster()
raw := b.Raw() // *sse.Broadcaster[sse.Event]

// Datastar
dsb := ds.NewBroadcaster()
raw := dsb.Raw() // *sse.Broadcaster[sse.Event]
```

Use `Raw()` when you need:

- **`SubscribeFilter`** — subscribe with a custom filter predicate (only available on the raw go-sse broadcaster).
- **Direct `Subscribe`** from the datastar Broadcaster (not exposed by the wrapper).
- **Cross-transport hub sharing** (see below).

## Cross-Transport Hub Sharing

When your application serves both HTMX and Datastar clients, you can share a single `sse.Broadcaster` as the fan-out hub so events published from one transport reach subscribers of the other:

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
    htmxBroadcaster := cqrshtmx.NewBroadcasterFromRaw(hub)
    dsBroadcaster := ds.NewBroadcasterFromRaw(hub)

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

| Approach | Subscribers | Event copies |
| --- | --- | --- |
| Separate broadcasters | HTMX clients see HTMX events only; Datastar clients see Datastar events only | 2 broadcast calls per update |
| Shared hub | All clients see all events | 1 broadcast call per update |

Sharing is the right choice when HTMX and Datastar pages display the same real-time data (e.g., a live dashboard with both an HTMX table view and a Datastar chart view).

### When NOT to share

If HTMX and Datastar clients consume **different event shapes** (HTML fragments vs. JSON patches), keep separate broadcasters. Mixing event types on one hub means each client must filter irrelevant events, adding complexity.

## The RawBroadcaster Interface

The root module exports a structural interface for code that needs to accept either Broadcaster type:

```go
type RawBroadcaster interface {
    Raw() *sse.Broadcaster[sse.Event]
}
```

Both `*cqrshtmx.Broadcaster` and `*datastar.Broadcaster` satisfy this interface structurally (duck typing — no import from datastar to root required). Use it in shared libraries or middleware:

```go
func RegisterRealtime(mux *http.ServeMux, bc cqrshtmx.RawBroadcaster) {
    raw := bc.Raw()
    raw.OnSubscribe(func() { log.Printf("client connected, total: %d", raw.Health().SubscriberCount) })
    raw.OnUnsubscribe(func() { log.Printf("client disconnected") })
}
```

## Choosing a Transport

| Criterion | HTMX | Datastar |
| --- | --- | --- |
| **Philosophy** | Server renders HTML fragments | Server sends signals, client morphs DOM |
| **Bundle size** | 0 JS (uses HTMX, loaded separately) | 11.76 KiB (`datastar.js`) |
| **Reactive state** | None (server is source of truth) | Signals (client-side reactive state) |
| **Use when** | CRUD forms, tables, navigation | Live dashboards, interactive widgets, optimistic-feeling UI |
| **Coexistence** | Both can run in the same app — different routes, different endpoints | Same |

For full-stack wiring with both transports, see the [Full-Stack Wiring Guide](fullstack-wiring.md).

## See Also

- [Datastar Integration Guide](datastar-integration.md) — Datastar setup, patches, replay, SDK re-exports
- [Full-Stack Wiring Guide](fullstack-wiring.md) — composing all cqrs-htmx sub-modules
- [Production Readiness](production-readiness.md) — SSE in production (heartbeats, graceful shutdown)
- [go-sse](https://github.com/larsartmann/go-sse) — the underlying SSE library
