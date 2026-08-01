# dashboardui — CQRS/Event-Sourcing Observability Dashboard

A self-contained, capability-aware dashboard for [go-cqrs-lite](https://github.com/larsartmann/go-cqrs-lite) applications.
Mount it on any `http.ServeMux` to get instant visibility into your event store, aggregates, projections, dead letters,
commands, queries, snapshots, and time-travel state reconstruction.

## Quick Start

```go
import "github.com/larsartmann/cqrs-htmx/dashboardui/v4"

dash, err := dashboardui.New(dashboardui.Config{
    EventSource:  store,
    Journal:      store,
    StreamReader: listing.NewInMemoryStreamReader(store),
})
if err != nil {
    log.Fatal(err)
}

mux := http.NewServeMux()
dash.Mount(mux, "/dashboard/")
```

Open `/dashboard/` in your browser. The dashboard auto-detects which go-cqrs-lite
interfaces you wired and shows only relevant panels.

## How It Works

The dashboard reads from go-cqrs-lite introspection interfaces. Each optional
interface activates a panel:

| Config Field      | Interface                        | Panel Activated                           |
| ----------------- | -------------------------------- | ----------------------------------------- |
| `EventSource`     | `event.EventSource`              | Aggregates, Aggregate Detail, Time-Travel |
| `EventByIDLoader` | `LoadByEventID(ctx, id.EventID)` | Event Detail (O(1) lookup)                |
| `Journal`         | `event.Journal`                  | Events, Overview                          |
| `SeekableJournal` | `event.SeekableJournal`          | Events (paginated)                        |
| `StreamReader`    | `listing.StreamReader`           | Aggregate Browser                         |
| `ProjectionHost`  | `*projectionhost.Host`           | Projections, Projection Reset             |
| `DeadLetterStore` | `projectionhost.DeadLetterStore` | Dead-Letter Queue (delete/purge)          |
| `CommandJournal`  | `command.CommandJournal`         | Command Audit                             |
| `QueryJournal`    | `query.QueryJournal`             | Query Audit                               |
| `SnapshotStore`   | `snapshot.SnapshotStore`         | Snapshot Inspector                        |
| `EventBus`        | `event.Bus`                      | SSE Live Updates                          |

Only `EventSource` OR `Journal` OR `SeekableJournal` is required (at least one
event-reading interface). Everything else is opt-in.

## Configuration

```go
type Config struct {
    EventSource      event.EventSource      // per-aggregate loading
    EventByIDLoader  EventByIDLoader        // O(1) event-by-ID (SQL stores)
    Journal          event.Journal          // global event log
    SeekableJournal  event.SeekableJournal  // paginated event log
    StreamReader     listing.StreamReader   // aggregate listing
    ProjectionHost   *projectionhost.Host   // projection monitoring
    DeadLetterStore  projectionhost.DeadLetterStore
    CommandJournal   command.CommandJournal
    QueryJournal     query.QueryJournal
    SnapshotStore    snapshot.SnapshotStore
    EventBus         event.Bus              // enables SSE live updates

    PayloadRenderer  PayloadRenderer        // custom payload formatting
    Title            string                 // sidebar brand text
    BasePath         string                 // URL prefix (default: /dashboard)
    AccentColor      string                 // CSS highlight color
    ReadOnly         bool                   // disable write ops (default: true)
    PageSize         int                    // rows per page (default: 50, max: 200)
    Authorizer       func(*http.Request) error
}
```

### Read-Only Mode

`ReadOnly` defaults to `true` (safe). When enabled, these operations are disabled:

- Projection reset
- DLQ replay, delete, purge
- Snapshot delete

Set `ReadOnly: false` to enable write operations. The consumer MUST wrap the
dashboard with authentication middleware when not read-only.

### Custom Payload Rendering

Implement `PayloadRenderer` to format event payloads for your domain:

```go
type PayloadRenderer interface {
    Render(payload []byte, encoding codec.Encoding) ([]byte, error)
}
```

The default renderer pretty-prints JSON and decodes CBOR to JSON. No consumer
domain types are needed.

## SSE Live Updates

When `EventBus` is configured, the dashboard:

1. Subscribes to all events via `event.Bus.SubscribeAll`
2. Forwards each event to an internal `cqrshtmx.Broadcaster`
3. Serves an SSE endpoint at `/-/events/stream`

The browser auto-connects and dispatches `dashboard:event` custom events. Listen
to these events to trigger HTMX swaps or other UI updates.

## Mounting

```go
// Option 1: Mount on a mux with prefix stripping
mux := http.NewServeMux()
dash.Mount(mux, "/dashboard/")

// Option 2: Get a handler for custom routing
handler := dash.Handler()
```

### Middleware

```go
// Built-in: security headers + panic recovery
handler := dash.Middleware()(dash.Handler())

// Add your own:
handler := cqrshtmx.Chain(
    dash.Middleware(),
    authMiddleware,
    csrfMiddleware,
)(dash.Handler())
```

## Demo

See `examples/dashboard-demo/main.go` for a fully seeded demo with users,
orders, commands, queries, and snapshots.

> **Note:** The demo requires the `dashboardui/v4` module to be tagged and
> published. Once tagged, add `./examples/dashboard-demo` to `go.work` and run:
>
> ```bash
> cd examples/dashboard-demo
> GOEXPERIMENT=jsonv2 go run .
> ```
>
> Then open http://localhost:8098/dashboard/

## Build

This module requires Go 1.26+ with `GOEXPERIMENT=jsonv2`:

```bash
GOEXPERIMENT=jsonv2 go build ./dashboardui/...
GOEXPERIMENT=jsonv2 go test ./dashboardui/...
```

## Architecture

The dashboard follows the same pattern as `adminui/`:

- `config.go` — Config struct, Capabilities detection, nav building
- `dashboard.go` — Dashboard struct, New(), MustNew(), page shell
- `handler.go` — Route registration and mounting
- `handlers.go` — All panel handlers (events, aggregates, projections, DLQ, etc.)
- `handler_overview.go` — Overview page + shared rendering helpers
- `render.go` — Response writing, partial detection, toast triggers
- `layout.go` — HTML shell: sidebar, header, embedded CSS/JS
- `payload.go` — PayloadRenderer interface and default implementation
- `sse.go` — SSE event bridge (event bus to broadcaster)

Rendering is currently Go-string-builder HTML. Future iterations will migrate
to templ-components for richer UI.
