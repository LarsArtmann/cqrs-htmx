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

### Reconnection and Backoff

The SSE client implements automatic reconnection with exponential backoff:

- Initial reconnect delay: 1 second
- Maximum reconnect delay: 30 seconds
- Delay doubles on each failure (1s, 2s, 4s, 8s, 16s, 30s, 30s, ...)
- On reconnect, `Last-Event-ID` is sent so the server can replay missed events
- Reconnect resets to 1s on successful connection or when the tab becomes visible again

### Event Replay on Reconnect

When an SSE client reconnects with `Last-Event-ID`, the server replays all events
that occurred since the last received event ID (up to 1000 events). On first connect,
recent history is backfilled so the user immediately sees context.

## Filtering and Search

The Events page supports in-memory filtering:

- **By event type**: `?type=user.created` filters to a specific event type
- **By stream type**: `?streamType=User` filters to events from a specific aggregate type
- Filters are preserved across pagination links

When filters are active, the dashboard scans up to 500 events and filters in-memory.
For larger datasets, wire a `SeekableJournal` for paginated access.

### Sorting

Events table columns are sortable by clicking the column headers:

- **`?sort=time`** / **`?sort=type`** / **`?sort=streamType`** / **`?sort=version`**
- **`?dir=asc`** / **`?dir=desc`** (defaults to ascending)
- Arrow indicators (▲/▼) show the active sort column and direction
- Sorting is in-memory (scans up to 500 events)

### Pagination

All list pages support cursor-based pagination:

- **`?after=<id>`**: Next page cursor (event/command/query ID)
- **`?prev=<history>`**: Cursor history for Previous navigation (comma-separated)
- **`?limit=<n>`**: Items per page (default: 50, max: 200, options: 25/50/100/200)
- Count display: "Showing X-Y of Z" appears on the last page when total is known

### HTMX Integration

The dashboard supports HTMX for partial page updates:

- **`data-hx-boost`**: All internal links use HTMX for smooth transitions
- **Partial rendering**: When `HX-Request: true` header is present, the server renders only the title and main content (no full HTML document)
- **Filter form**: The events filter form uses `hx-get` for partial content swapping
- **SSE**: Live events are injected into the events table and projection health panel is refreshed automatically

### CSV and JSON Export

All list pages support data export via query parameters:

- **`?format=csv`**: Downloads a CSV file with up to 10,000 rows
- **`?format=json`**: Returns a JSON array of row objects
- Available on: `/events`, `/commands`, `/queries`

### Detail Views

Each list item links to a detail page showing full metadata and pretty-printed JSON payload:

- **Events** (`/events/{id}`): Type, stream ref, version, occurred at, metadata, payload with copy/download buttons
- **Commands** (`/commands/{id}`): Type, stream ref, received at, metadata, payload
- **Queries** (`/queries/{id}`): Type, received at, metadata, payload
- **Projections** (`/projections/{name}`): Status badge, checkpoint, processed/errors/restarts stats, lag, last error, DLQ link, reset action
- **DLQ entries** (`/dead-letters/{projection}/{eventID}`): Event details, error info, replay action
- **Aggregates** (`/aggregates/{type}/{id}`): Event timeline with pagination, total event count

### Time-Travel Slider

The time-travel detail page includes a version slider with keyboard navigation:

- **Arrow keys** (left/right) move the slider and navigate to the selected version
- **Live value display**: The version number updates as the slider moves
- **Version links**: For streams with <= 20 versions, individual version numbers are clickable

## Observability Endpoints

Three unauthenticated endpoints for load balancers and Kubernetes probes:

| Endpoint      | Purpose         | 200 Response                             | 503 Response                                |
| ------------- | --------------- | ---------------------------------------- | ------------------------------------------- |
| `/-/healthz`  | Liveness probe  | `{"status":"ok"}`                        | `{"status":"shutting_down"}`                |
| `/-/readyz`   | Readiness probe | `{"status":"ready","ready":true}`        | `{"status":"no_data_source","ready":false}` |
| `/-/versionz` | Build metadata  | Module, Go version, capabilities, config | —                                           |

All return `application/json` with `Cache-Control: no-store`.

## Mobile Responsive Design

The dashboard is fully responsive:

- **Hamburger menu**: On screens <768px, the sidebar collapses into a slide-in drawer with backdrop overlay
- **Touch targets**: All buttons have minimum 44px height on mobile (WCAG 2.5.5)
- **Table scroll**: Data tables scroll horizontally within a wrapper on narrow screens
- **Filter bar stacking**: Filter controls stack vertically on mobile
- **Stat cards**: Grid collapses to 2 columns on mobile

## Copy-to-Clipboard

Identifiers (event IDs, stream IDs, correlation IDs, etc.) are click-to-copy:

- Click any element with the copy cursor to copy its value to the clipboard
- A toast notification confirms the copy
- The `data-copyable` attribute on any HTML element enables this behavior

## Accessibility

- **Semantic HTML5 landmarks**: `<aside>`, `<nav>`, `<main>`, `<header>` for screen reader navigation
- **Skip-to-content link**: Keyboard users can bypass the sidebar
- **ARIA labels**: All interactive elements (buttons, links, forms) have descriptive aria-labels
- **Focus-visible outlines**: All focusable elements show a visible focus ring
- **Reduced motion**: Animations disabled when `prefers-reduced-motion: reduce`
- **Live regions**: SSE status updates use `aria-live="polite"`

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

See `examples/dashboard-demo/main.go` for a fully seeded demo with 8 users,
6 orders, commands, queries, snapshots, a projection host, EventBus-powered
SSE live updates, and a goroutine that publishes new events every 5 seconds.

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
