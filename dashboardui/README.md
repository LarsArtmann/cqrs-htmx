# dashboardui — CQRS/Event-Sourcing Observability Dashboard

A self-contained, capability-aware dashboard for [go-cqrs-lite](https://github.com/larsartmann/go-cqrs-lite) applications.
Mount it on any `http.ServeMux` to get instant visibility into your event store, aggregates, projections, dead letters,
commands, queries, snapshots, and time-travel state reconstruction.

> **dashboardui or adminui?** **dashboardui (this module) introspects your event
> store**: CQRS/event-sourcing observability over go-cqrs-lite interfaces
> (journal, aggregates, projections, dead letters, snapshots), read-only by
> default. **[adminui](../adminui/README.md) manages your users**: identity
> operations (accounts, tenants, members, audit log) over a `*usermgmt.Service`,
> with write actions, role-gated. They are complementary, not alternatives:
> most apps mount both, and [setup/v4](../setup/README.md) wires both in one call.

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

## Integration Modes

The panel meets your app at whatever depth you want — from "give me a URL" to
"render inside my chrome" to "I'll build the UI myself":

| Mode                          | You own                          | The dashboard owns            | Seam                                             |
| ----------------------------- | -------------------------------- | ----------------------------- | ------------------------------------------------ |
| **Destination** (default)     | A mount path + auth middleware   | The whole page, top to bottom | `New` + `Mount`                                  |
| **Embedded**                  | The app shell (nav, CSS, theme)  | Panel content only            | `Config.Layout`                                  |
| **From a go-cqrs-lite system**| The DeploymentConfig             | Everything the system exposes | `FromSystem(sys)`                                |
| **Headless data**             | The entire UI                    | Introspection queries         | the `core` sub-package (next section)            |

### Destination (zero-config)

`Autodetect` probes your sources for every go-cqrs-lite introspection interface,
so the hand-written type-assertion dance disappears:

```go
cfg, err := dashboardui.Autodetect(store) // wires EventSource/Journal/SeekableJournal/
if err != nil {                           // StreamReader/Host/DLQ/Snapshots/Bus — whatever
    log.Fatal(err)                        // the store implements
}
cfg.Title, cfg.BasePath, cfg.ReadOnly = "InboxClean CQRS", "/cqrs", true

dash := dashboardui.MustNew(cfg)
dash.Mount(mux, "/cqrs/")
```

Passing several sources merges their capabilities first-wins — each capability
is claimed by the earliest source that implements it, so put the primary store
first and auxiliary stores (a dedicated snapshot store, a projection host)
after it:

```go
cfg, err := dashboardui.Autodetect(store, snapshotStore, projectionHost)
```

At least one source must implement a read interface (`event.EventSource`,
`event.Journal`, or `event.SeekableJournal`) — the same rule `New` enforces.
Explicit `Config` field assignment still works (and `New` still enforces the
same "at least one read interface" rule) — `Autodetect` only removes the
plumbing, it does not change the contract.

### From a go-cqrs-lite system

If you compose with `system.New()` (the `systemadapter` bridge), `FromSystem`
maps the system's event store, bus, projection host, snapshot store, and
command/query stores onto a ready Config — no manual assertion dance, no
dashboardui dependency on the system package (the seam is a duck-typed
accessor interface any system wrapper also satisfies):

```go
sys, _ := system.New(ctx, systemadapter.DomainConfig(),
    systemadapter.RecommendedSQLiteDeployment("file:app.db"))
_ = sys.Start(ctx)

cfg := dashboardui.FromSystem(sys) // panels light up per capability
cfg.Title = "MyApp CQRS"
dash := dashboardui.MustNew(cfg)
dash.Mount(mux, "/cqrs/")
```

`Autodetect` recognizes system values in its source list too, so stores and
systems merge with the usual first-wins rule. Engines without a native stream
reader get a journal-derived one, so the aggregates panel lights up
everywhere. Guide: `docs/guides/leveraging-system-metaengine.md`.

### Embedded (your app shell, our panels)

Set `Config.Layout` and the dashboard stops owning the document. Every
full-page render calls your function with the page metadata and the
ready-rendered panel content — your nav, your fonts, your stylesheet, your
theme wrap it:

```go
cfg.Layout = func(meta dashboardui.PageMeta, content templ.Component) templ.Component {
    return appShell(appPage{
        Title: meta.FullTitle, // "Events · MyApp"
        Nav:   append(myAppNav, dashboardNavItems(meta)...), // or your own nav
    }, content)
}
```

Contract (full details on `LayoutFunc`):

- Your function owns the **entire document**. Link `meta.CSSURLs` (the panel
  content is styled by the dashboard stylesheets), and skip
  `meta.ScriptURLs`' htmx entry if your shell already loads htmx (never twice).
- `meta.Nav` is the capability-filtered navigation (`NavLink{Href, Label,
  Icon, Active}`) — render it, merge it into your sidebar, or drop it.
- `meta.Nonce` carries the per-request CSP nonce for script tags you emit.
- HTMX partial swaps and polled regions bypass the shell (no work needed);
  write-action toasts surface via the `dashboardui:toast` HX-Trigger event —
  include a `feedback.ToastContainer` in your shell to show them.
- Error/404 responses keep the dashboard's minimal built-in shell.

setup/v4 exposes the same seam as `setup.Config.DashboardLayout`.

### Using the `core` Sub-Package Directly

The `core/` sub-package contains the pure data layer — capability detection,
pagination math, event loading/filtering, overview aggregation, and payload
rendering — with zero HTML generation. Any tool (CLI, metrics exporter, custom
UI) can import it to inspect a go-cqrs-lite system without pulling in the
dashboard's rendering code:

```go
import "github.com/larsartmann/cqrs-htmx/dashboardui/v4/core"

cfg := core.Config{
    SeekableJournal: store,
    StreamReader:    reader,
}

overview := core.FetchOverview(ctx, cfg)
events, _ := core.LoadRecentEvents(ctx, cfg, id.EventID{}, 50)
```

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
    VersionzRequireAuth bool               // guard /-/versionz behind Authorizer (default: false)
}
```

### Read-Only Mode

`ReadOnly` defaults to `true` (safe). When enabled, these operations are disabled:

- Projection reset
- DLQ replay, delete, purge
- Snapshot delete

Set `ReadOnly: false` to enable write operations. The consumer MUST wrap the
dashboard with authentication middleware when not read-only.

### Route Manifest

`Routes()` returns the exact manifest of HTTP routes this instance serves —
generated from the same internal route table as the mux registration, so it
reflects the detected capabilities and `ReadOnly` (write routes disappear in
read-only mode). Each entry carries the method, the consumer-facing pattern
(wildcards in braces), the owning panel, and a `Write` flag:

```go
for _, r := range dash.Routes() {
    if r.Write {
        auditLog.Printf("mutating route exposed: %s %s (%s)", r.Method, r.Pattern, r.Panel)
    }
}
```

Typical use: building an allowlist for a reverse proxy or policy engine, or
diffing the route surface across versions in upgrade tests:

```go
readOnlyAllowlist := map[string]bool{}
for _, r := range dash.Routes() {
    if !r.Write {
        readOnlyAllowlist[r.Method+" "+r.Pattern] = true
    }
}
```

The styled-404 catch-all is the one served route the manifest omits.

### Authorization and Audit Attribution

The dashboard itself ships no authentication — access control is a config
seam with two forms:

- `Authorizer func(*http.Request) error` (legacy): deny by returning an
  error (403), allow by returning nil.
- `ActorAuthorizer func(*http.Request) (Actor, error)`: the actor-aware
  form. On success it returns an `Actor` — `{ID, Name}` — that the
  dashboard injects into the request context. When both are set,
  `ActorAuthorizer` takes precedence.

Every write operation (projection reset, DLQ replay/delete/purge, snapshot
delete) emits a `dashboardui.audit` log line. Attribution is automatic:

```json
{"msg":"dashboardui.audit","op":"dlq.replay","projection":"user-read-model",
 "result":"ok","actor_id":"user-42","actor_name":"ops@example.com",
 "request_id":"req_01HYZ..."}
```

- **actor_id / actor_name** — from `ActorAuthorizer`, or from your own
  middleware calling `dashboardui.WithActor(ctx, actor)` (works even with no
  dashboard Authorizer configured).
- **request_id** — when you run httputil's `RequestID` middleware.
- Without either, entries record `"actor":"anonymous"` — authorized but
  unidentified.

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

> **Sorted view is a window, not the whole journal.** Without filters, a
> sorted view loads the most recent 500 events (`FilterScanLimit`), sorts
> them in memory, and shows the entire window with a "sorted view: first 500
> events" badge. Pagination controls are hidden because cursor paging cannot
> follow a re-sorted order — a next page would re-sort the same window
> forever. With filters active, sorting applies within each cursor-paged
> result page instead. Server-side sorted pagination is the full fix
> (research idea 9 in
> [`docs/research/2026-10-06_dashboardui-metaengine-system-improvements.md`](../docs/research/2026-10-06_dashboardui-metaengine-system-improvements.md)).

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

## Telemetry (read-only)

Four optional Config providers light up a `/telemetry` panel and probe
composition — dashboard-owned view structs, so no system/metaengine types leak
into the rendering layer:

- `Topology` — instances, buses, and the projection-host shape as a table.
- `EngineHealths` — per-engine health rides on `/-/healthz` (`engines` key)
  and gates `/-/readyz` (an unhealthy engine is a 503: keep the pod out of
  rotation until it recovers or is quarantined).
- `Placements` — the metaengine planner's query→engine/ADT assignments with
  volume and latency estimates.
- `EngineStats` — live RTT cards (EWMA, P95, samples) with a freshness badge
  (samples older than 5 minutes render stale).

Each section renders independently — a failing provider shows its inline
error without blanking the others. `systembridge.WireTelemetry` maps a
go-cqrs-lite `*system.System` onto all four in one call (the one dashboardui
package importing the system packages).

## Observability Endpoints

Three unauthenticated endpoints for load balancers and Kubernetes probes:

| Endpoint      | Purpose         | 200 Response                             | 503 Response                                |
| ------------- | --------------- | ---------------------------------------- | ------------------------------------------- |
| `/-/healthz`  | Liveness probe  | `{"status":"ok"}`                        | `{"status":"shutting_down"}`                |
| `/-/readyz`   | Readiness probe | `{"status":"ready","ready":true}`        | `{"status":"no_data_source","ready":false}` |
| `/-/versionz` | Build metadata  | Module, version, Go version, VCS stamp, capabilities, config | —                              |

All return `application/json` with `Cache-Control: no-store`.

### versionz fields and the auth guard

`module` and `version` identify the dashboardui module itself — derived from
the binary's build info via reflection, so a **fork reports its own module
path**, not the upstream constant. `version` is the published semver when a
consumer binary links a tagged dependency, `"(devel)"` when developing
in-module, and empty for local directory replaces. `goVersion` is the toolchain.
The `vcsRevision` / `vcsTime` / `vcsModified` fields are the **binary's** VCS
stamp (the build an operator is talking to) and are omitted when the binary
was built without VCS metadata.

By default `/-/versionz` is public like the other probes, so load balancers
can read build stamps without credentials. Note that it reveals `Title`,
`BasePath`, capabilities, module version, and the VCS revision. When an
`Authorizer` is configured and that surface should not be public, opt in:

```go
dashboardui.Config{
    // ...
    Authorizer:          myAuthCheck,
    VersionzRequireAuth: true, // /-/versionz now returns 403 on denial
}
```

`healthz` and `readyz` stay public regardless — probes must not depend on
credentials.

## Mobile Responsive Design

The dashboard is fully responsive:

- **Hamburger menu**: On screens <768px, the sidebar collapses into a slide-in drawer with backdrop overlay
- **Touch targets**: All buttons have minimum 44px height on mobile (WCAG 2.5.5)
- **Table scroll**: Data tables scroll horizontally within a wrapper on narrow screens
- **Filter bar stacking**: Filter controls stack vertically on mobile
- **Stat cards**: `display.Grid` auto-fit (`minmax(190px, 1fr)`) — cards reflow to the container width (roughly 2 columns on phones, 4+ on desktop)

## Accessibility

- **Semantic HTML5 landmarks**: `<aside>`, `<nav>`, `<main>`, `<header>` for screen reader navigation
- **Skip-to-content link**: Keyboard users can bypass the sidebar
- **ARIA labels**: All interactive elements (buttons, links, forms) have descriptive aria-labels
- **Focus-visible outlines**: All focusable elements show a visible focus ring
- **Reduced motion**: Animations disabled when `prefers-reduced-motion: reduce`
- **Live regions**: SSE status updates use `aria-live="polite"`

## Theming

The dashboard ships with a user-controllable dark/light theme toggle in the header (templ-components `ThemeToggle`):

- **First visit**: follows the operating system's `prefers-color-scheme`
- **Toggle**: clicking the sun/moon button switches theme and persists the choice in `localStorage`
- **Flash-free**: an inline pre-paint script (CSP-nonce'd) applies the stored choice before first render
- **Per-instance accent**: pass `Accent` in the `Config` to brand both modes (CSS custom property)

Without JavaScript the dashboard renders in light mode.

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

> **Note:** `dashboardui/v4` is published (tagged since `dashboardui/v4.8.2`), so the
> demo resolves from the tag. `./examples/dashboard-demo` is already in `go.work`; run:
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

## Styling and templ-components Adoption

The dashboard renders through **templ** (`a-h/templ`) — nine `.templ` files
(`layout`, `components`, `overview`, `events`, `aggregates`, `projections`,
`audit`, `dlq`, `timetravel`, `snapshots`) with the generated `_templ.go`
committed, so consumers run no codegen. Library components from
[templ-components](https://github.com/larsartmann/templ-components) render
directly inside the page templates (`@display.Table(...)`, `@forms.Input`,
...); templ auto-escapes every interpolation (`templ.EscapeString` is
`html.EscapeString`, matching the former manual `esc()` byte-for-byte).
Styling ships as a compiled Tailwind bundle (`assets/dashboard-tw.css`,
served at `/-/dashboard-tw.css` with ETag/304).

Adopted capabilities: `display.StatusBadge`/`Badge` (all status/encoding
badges), `display.StatCard` with `ValueID` DOM hooks, `display.Table` with
typed sort headers + `LazyRows`, `display.EmptyState`, `display.Button`,
`display.CopyButton`, `display.DefinitionList`, `display.ListNote`
(count-only `ListNoteCount` variant under the DLQ table — the count is the
Replay All / Purge All blast radius; an empty DLQ renders `EmptyState`
instead, so the variant's always-render "Showing 0 items." never appears on
this page — deliberate), `feedback.ToastContainer`,
`htmx.GlobalErrorHandling`, `forms.Select` (page size), `errorpage.ErrorPage`
/`NotFound404`, and `icons`.

Deliberate exclusions: `navigation.Pagination` (cursor + history pagination of
append-only journals — numbered pages are meaningless), `navigation.SidebarNav`
/`layout.AppShell` (custom dark-sidebar theme + mobile drawer, same precedent
as adminui), and the hand-rolled `Showing X–Y of Z` pagination info (ListNote
speaks N-of-M truncation or N-items counts, not X–Y ranges —
templ-components v1.19.1 lifted the count-only half of the old
`display.ListNote` exclusion; the X–Y range variant is a recorded upstream
ask). `display.Grid` and `htmx.PolledRegion` were adopted 2026-09-26 (the
old "children render empty in the hybrid path" blocker died with the
full-templ migration); `display.RelativeTime` joined them on the snapshot
detail page the same day, and `display.PageHeader` is a documented
divergence (see the adoption table).
Adopted in the N16/N17 pass: `forms.Input` (event filter bar), library
Button link/submit, EmptyState, and DefinitionList render benchmarks.

#### Adoption table (grep-able inventory, 2026-09-23)

| Library capability                        | Status                                                                                                                                                                                                                | Where                                            |
| ----------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------ |
| `display.StatCard` (+`ValueID` live hook) | adopted                                                                                                                                                                                                               | `components.templ` (statCard wrapper)            |
| `display.StatusBadge`/`Badge`             | adopted                                                                                                                                                                                                               | `components.templ`                               |
| `display.Table` (sortable headers)        | adopted                                                                                                                                                                                                               | `components.templ`, `sort.go`                    |
| `display.EmptyState`                      | adopted                                                                                                                                                                                                               | `components.templ` (emptyStatePanel)             |
| `display.Button`                          | adopted                                                                                                                                                                                                               | `components.templ` (link/submit)                 |
| `display.CopyButton`                      | adopted                                                                                                                                                                                                               | `components.templ`                               |
| `display.DefinitionList`                  | adopted                                                                                                                                                                                                               | `components.templ`                               |
| `display.ListNote` (count variant)        | adopted                                                                                                                                                                                                               | `components.templ`                               |
| `forms.Input`                             | adopted                                                                                                                                                                                                               | events filter bar                                |
| `forms.Select` (page size)                | adopted                                                                                                                                                                                                               | pagination controls                              |
| `htmx.GlobalErrorHandling` (tuned)        | adopted                                                                                                                                                                                                               | `layout.templ`                                   |
| `htmx.CSRFToken`                          | adopted 2026-09-23                                                                                                                                                                                                    | dlq/projections/snapshots forms                  |
| `feedback.ToastContainer`                 | adopted                                                                                                                                                                                                               | `layout.templ`                                   |
| `errorpage.ErrorPage`/`NotFound404`       | adopted                                                                                                                                                                                                               | `render.go`, `handler.go`, `errorShell`          |
| `icons` (`Name`/`IconPathData`)           | adopted                                                                                                                                                                                                               | `layout.go` (navIcon), page icons                |
| `layout.ThemeScript`/`ThemeToggle`        | adopted 2026-09-23                                                                                                                                                                                                    | `layout.templ`, class-driven dark mode           |
| `htmx.PolledRegion`                       | adopted 2026-09-26 (Trigger=`every 10s, refresh`)                                                                                                                                                                     | `overview.templ` projection-health region        |
| `display.Grid` (auto-fit)                 | adopted 2026-09-26 (`MinColWidth: 190px`)                                                                                                                                                                             | `overview.templ`, `projections.templ` stat grids |
| `display.PageHeader`                      | divergence: 8/11 headers need rich titles (code+badges+copy inside the h2) that string Title cannot express — upstream ask recorded for Title-as-Component; the 3 plain sites stay hand-rolled for visual consistency | `.page-header` markup                            |
| `forms.Slider`                            | rejected: labeled wrapper mismatches scrubber row                                                                                                                                                                     | `timetravel.templ` version slider                |
| `display.DataTable`                       | rejected: data-driven shape fights inline templ cells                                                                                                                                                                 | `sort.go` Table+sortState composition            |
| `display.RelativeTime`                    | adopted 2026-09-26 — snapshot detail `Created` line via the nonce-carrying `snapshotRelativeTime` wrapper; live SSE rows keep `relativeTime()` (JS-injected, out of scope)                                            | `snapshots.templ`, `components.templ`            |
| `navigation.Pagination`                   | divergence: cursor paging (no lib equiv)                                                                                                                                                                              | `pagination.go`                                  |
| `navigation.SidebarNav`/`layout.AppShell` | divergence: custom shell (boost/drawer)                                                                                                                                                                               | `layout.templ`                                   |
| `layout.Base`                             | divergence: self-hosted/noindex needs                                                                                                                                                                                 | `layout.templ`                                   |

Design rationale for the divergences lives in
[ADR-0053](../docs/adr/0053-dashboardui-templ-components-divergences.md) and the
adoption-audit series
([2026-09-23 report](../docs/research/2026-09-23_templ-components-deep-dive.html);
series index: [docs/research/README.md](../docs/research/README.md)).

### Rebuilding the CSS bundle

After ANY component adoption or templ-components family bump:

```bash
nix run .#build-dashboardui-css   # from the repo root; scans .templ + *_go.go class sources
```

The build fails loudly if a utility family goes missing (canaries). Rebuild in
the SAME change as a family bump — a stale bundle ships missing utilities.

### Golden tests

`golden_test.go` pins rendered pages under `testdata/golden/`. After an
intentional markup change, regenerate:

```bash
cd dashboardui && GOEXPERIMENT=jsonv2 go test ./... -update
```

Review the diff before committing - goldens are the rendered-HTML contract.

### Benchmarks

`render_bench_test.go` compares the pre-adoption hand-rolled renderers against
the production templ component path; interpretation and recorded artifacts
live in
[`docs/benchmarks/dashboardui-render-2026-09-19.md`](../docs/benchmarks/dashboardui-render-2026-09-19.md)
(single-digit microseconds per card - noise behind network I/O; the 2026-09-19
numbers measured the strings.Builder hybrid bridge and are not directly
comparable to the templ arms).

## Architecture

The dashboard follows the same pattern as `adminui/`:

- `config.go` — Config struct, Capabilities detection, nav building
- `dashboard.go` — Dashboard struct, New(), MustNew(), page shell
- `handler.go` — Route registration and mounting
- `handlers*.go` — Panel handlers (events, aggregates, projections, DLQ,
  snapshots, time-travel, commands/queries) — data loading + `renderPage`
- `handler_overview.go` — Overview handler + stat-card helpers
- `render.go` — Response writing, error pages, partial detection, toast triggers
- `*.templ` + `*_templ.go` — Page/layout/component templates and their
  generated Go (committed; regenerate from the module dir via
  `nix run .#gen`)
- `detail_items.go` — Definition-item and small data builders shared by the
  templates
- `layout.go` — Embedded CSS/JS assets, icon-name mapping
- `payload.go` — PayloadRenderer interface and default implementation
- `sse.go` — SSE event bridge (event bus to broadcaster)

Rendering is full templ (see the Styling section above); every handler threads
the request context implicitly through `renderPage` → `Component.Render`.
