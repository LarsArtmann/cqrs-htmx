# dashboardui — Architecture Feedback & Improvement Plan

**Consumer:** [DiscordSync](https://github.com/LarsArtmann/DiscordSync) — real-time Discord archiving tool (CQRS event store, 24 projections, SQLite/Turso, templ + templ-components + Tailwind v4)
**Module reviewed:** `cqrs-htmx/dashboardui/v4` (dashboardui@latest, ~7,570 LOC across 21 source files)
**Dashboard version used by DiscordSync:** **None** — DiscordSync reimplements 3 of dashboardui's 9 panels from scratch and is missing 3 more entirely.
**Date:** 2026-08-05
**Depth:** Full source audit of dashboardui + side-by-side comparison with DiscordSync's custom implementations

---

## TL;DR

dashboardui solves the right problem (CQRS observability UI) with the right data layer (capability-detection over go-cqrs-lite interfaces) but with the **wrong rendering technology for its own ecosystem**. It is the **only module in the cqrs-htmx family** that renders HTML via `fmt.Fprintf` + `strings.Builder` with a self-contained CSS design system. Its sibling module `adminui` already uses templ + Tailwind v4 + templ-components. DiscordSync does too. Every other consumer Lars builds uses templ. dashboardui is the outlier, and its rendering choice makes it **impossible to embed in any existing templ-components app** — the exact apps most likely to need a CQRS dashboard.

The fix is not cosmetic. It requires decomposing the monolith into three independently consumable layers: a pure-data `core`, a templ-components `panels` package, and an optional standalone shell. This unlocks dashboardui for DiscordSync and every future templ-components consumer, while preserving the "mount and go" experience for simple apps.

---

## Table of Contents

1. [The Five Walls of Incompatibility](#1-the-five-walls-of-incompatibility)
2. [What dashboardui Gets Right](#2-what-dashboardui-gets-right)
3. [What DiscordSync Reimplements (and Why)](#3-what-discordsync-reimplements-and-why)
4. [What DiscordSync Is Missing Entirely](#4-what-discordsync-is-missing-entirely)
5. [The Proposed Architecture: 3-Layer Decomposition](#5-the-proposed-architecture-3-layer-decomposition)
6. [Panel-by-Panel Improvement Opportunities](#6-panel-by-panel-improvement-opportunities)
7. [Migration Path](#7-migration-path)
8. [Concrete Impact Numbers](#8-concrete-impact-numbers)

---

## 1. The Five Walls of Incompatibility

### Wall 1: `fmt.Fprintf` vs. templ (the root cause)

dashboardui generates HTML via string concatenation. Every handler builds HTML through `strings.Builder` + `fmt.Fprintf`, with a manual `esc()` wrapper for HTML escaping.

**The numbers:**

| File | `fmt.Fprintf`/`WriteString`/`Builder` call sites |
|---|---|
| `handlers_events.go` | 28 |
| `handler_overview.go` | 24 |
| `handlers_timetravel.go` | 28 |
| `handlers_dlq.go` | 23 |
| `handlers_aggregates.go` | 15 |
| `layout.go` | 31 |
| `handlers_snapshots.go` | 14 |
| `handlers_audit.go` | 10 |
| `handlers_projections.go` | 5 |
| **Total** | **178** |

The manual `esc()` function (`handler_overview.go:335`) is a reimplementation of what templ does automatically at compile time:

```go
// dashboardui — manual escaping (handler_overview.go:335)
func esc(s string) string {
    return html.EscapeString(s) // import "html"
}

// Then sprinkled everywhere:
fmt.Fprintf(&b, `<td class="cell-emph">%s</td>`, esc(proj.Name))
```

```go
// templ — automatic, compile-time-safe, impossible to forget
<td class="cell-emph">{ proj.Name }</td>
```

**Why this matters beyond aesthetics:** forgetting an `esc()` call is an XSS vector. templ makes HTML injection a compile-time error. Every new contributor to dashboardui must remember to wrap every interpolated string in `esc()` — the kind of discipline that works for a solo project and fails silently in a team.

**For comparison:** the sibling module `adminui` uses templ (see `adminui/layout.templ`). `adminui/icons.go` is the only `.go` (non-templ) file with a `strings.Builder` call, and it's for icon path storage, not HTML generation. dashboardui has **zero `.templ` files**.

### Wall 2: Self-contained CSS vs. shared design system

dashboardui ships a **335-line embedded CSS string** (`layout.go:200-534`, the `dashboardCSS` constant) defining its own complete design system:

```css
/* layout.go:200 — a parallel design universe */
const dashboardCSS = `
:root {
    --accent: #4f46e5;
    --bg: #f6f7f9;
    --surface: #ffffff;
    --surface-hover: #f0f1f4;
    --text: #0f172a;
    --muted: #64748b;
    --border: #e6e8ec;
    --ok: #16a34a;
    --warn: #d97706;
    /* ... 300+ more lines ... */
}
```

This defines its own `.btn`, `.data-table`, `.nav-link`, `.empty-state`, `.sidebar`, `.app-layout`, `.toast-container`, `.skip-link`, `.hamburger`, and dozens more classes. It is a complete, independent CSS framework.

DiscordSync uses **templ-components** (Lars's own library: `display.StatCard`, `display.Badge`, `display.EmptyState`, `display.Grid`, etc.) + **Tailwind v4** with shared design tokens (`--ds-brand`, `--ds-brand-dark`, `--ds-status-live`). The dashboard, every list page, every detail page, and every filter form share one consistent design language.

**The problem:** two design systems in one deployment produce visual inconsistency. The CQRS dashboard looks nothing like the rest of the app. Users would see two different sidebar styles, two different card styles, two different table styles, two different color palettes, two different dark-mode implementations. It looks like two apps bolted together — because it is.

**For comparison:** `adminui` uses Tailwind v4 via `assets/admin-tw.css` + Tailwind utility classes in its templ files. It is visually consistent with any Tailwind-based app.

### Wall 3: Monolithic layout vs. composable components

Every dashboardui handler calls `renderLayout()`, which emits a **complete HTML document** — `<!DOCTYPE html>`, `<head>`, `<body>`, sidebar, header, `<main>`, toast container, footer (`layout.go:16-51`). There is no way to render just the DLQ panel without also rendering dashboardui's sidebar, header, CSS link, and HTMX script tag.

```go
// layout.go:16 — the layout wraps everything, always
func (d *Dashboard) renderLayout(p pageData, content func() string) string {
    var b strings.Builder
    b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
    // ... CSS, JS, meta tags ...
    b.WriteString(d.renderSidebar(p))        // dashboardui's own sidebar
    b.WriteString(d.renderHeader(p))         // dashboardui's own header
    fmt.Fprintf(&b, `<main id="main-content" class="content-area">%s</main>`, content())
    // ...
}
```

DiscordSync's pages render as **composable templ components** that slot into a shared `@Layout(...)` call:

```templ
// DiscordSync — a panel IS a component, not a page
templ DLQPage(data DLQViewModel) {
    @Layout(LayoutProps{Title: "Dead-Letter Queue", CurrentPath: routeDLQ}) {
        <div class="space-y-6">
            @display.PageHeader(display.PageHeaderProps{Title: "Dead-Letter Queue"})
            // ... panel content ...
        </div>
    }
}
```

To embed dashboardui's DLQ panel inside DiscordSync, you would need to either:
- Accept a second complete HTML document inside DiscordSync's layout (broken DOM)
- Rewrite dashboardui's DLQ handler to not call `renderLayout()` (the panel and layout are coupled)

Neither is viable. The layout ownership is binary: either dashboardui owns the page or it doesn't.

### Wall 4: Duplicate navigation and app shell

dashboardui builds its own sidebar, mobile hamburger menu, header, SSE status indicator, and toast container (`layout.go:53-140`):

```go
// dashboardui — its own complete app chrome
func (d *Dashboard) renderSidebar(p pageData) string {
    b.WriteString(`<aside class="sidebar">`)
    // brand, nav links, logout link, footer
}

func (d *Dashboard) renderHeader(p pageData) string {
    // hamburger button, title, SSE indicator
}
```

DiscordSync has its own hand-rolled desktop sidebar (`layout.templ`, with documented flexbox blowout guards requiring `shrink-0` + `min-w-0` — see DiscordSync's `AGENTS.md` "Sidebar flexbox blowout" section), mobile nav via `navigation.Nav`, desktop sidebar via `navigation.SidebarNav`, a command palette with fuzzy search, bot status badge polling, skip-to-content link, relative-time auto-ticking, view-transition error suppression, and a CSP nonce system.

If dashboardui were mounted inside DiscordSync, users would see **two sidebars, two headers, two toast systems, and two SSE indicators** in the same deployment. The visual and UX inconsistency would be unacceptable.

### Wall 5: Inline event handlers violate strict CSP

dashboardui uses inline `onsubmit="return confirm(...)"` for all destructive actions — DLQ replay, DLQ purge, DLQ delete, projection reset, snapshot delete:

```go
// handlers_dlq.go:237 — CSP violation
`<form method="POST" action="%s/dead-letters/%s/replay" class="inline-form"
    onsubmit="return confirm('Replay all dead letters for %s?')" ...>`

// handlers_projections.go:125 — CSP violation
`<form method="POST" action="%s/projections/%s/reset" class="inline-form"
    onsubmit="return confirm('Reset projection %s? ...')" ...>`
```

**5 inline `onsubmit` handlers** across `handlers_dlq.go` (3), `handlers_projections.go` (1), `handlers_snapshots.go` (1), plus 1 inline `<script>` in `layout.go:121-139` (toast container).

DiscordSync enforces a strict Content Security Policy with `script-src 'self' 'nonce-{nonce}'` — **zero inline event handlers are permitted**. All confirmation dialogs use a CSP-safe `data-confirm` attribute pattern:

```templ
// DiscordSync — CSP-safe confirmation pattern
@confirmForm("/dlq/replay", "Replay ALL dead-lettered events across all projections?", "mt-3") {
    <button type="submit" class={ "px-3 py-1 text-xs font-medium " + primaryButtonClass }>Replay All</button>
}
```

dashboardui **cannot be deployed inside any app with a strict CSP** without either:
- Disabling the CSP for dashboardui routes (security hole)
- Rewriting every `onsubmit` to use `data-confirm` + a delegated listener (defeats the purpose of using the library)

---

## 2. What dashboardui Gets Right

Despite the rendering incompatibility, dashboardui's **architecture is fundamentally sound**. The non-rendering layers are well-designed and reusable.

### Capability detection is excellent

The `Config` → `Capabilities` → conditional panel activation pattern (`config.go:38-181`) is the right abstraction:

```go
// dashboardui — clean capability detection
type Capabilities struct {
    EventSource     bool
    Journal         bool
    SeekableJournal bool
    ProjectionHost  bool
    DeadLetterStore bool
    CommandJournal  bool
    // ...
}

func (cfg Config) capabilities() Capabilities {
    return Capabilities{
        EventSource:     cfg.EventSource != nil,
        ProjectionHost:  cfg.ProjectionHost != nil,
        // ...
    }
}
```

Panels auto-activate based on what the consumer wires. This is exactly right — a consumer that only wires a `Journal` gets just the events panel; a consumer that wires `ProjectionHost` gets projections + DLQ. No configuration needed beyond passing interfaces.

### Data extraction logic is clean and isolated

The data-fetching functions (`buildProjectionStats`, `overviewStats`, `listStreamsPaged`, `loadRecentEvents`) are well-separated from rendering. They already return typed structs (`projectionStat`, `overviewStats`, `recentEvent`). The logic that converts `projectionhost.Host.Status()` into display data is reusable regardless of rendering technology.

### SSE with reconnection and replay is production-grade

The SSE layer (`sse.go`) with exponential backoff (1s → 30s), `Last-Event-ID` replay, and first-connect backfill is well-engineered. The `cqrshtmx.Broadcaster` integration is clean.

### Observability probes are useful

The unauthenticated `/-/healthz`, `/-/readyz`, `/-/versionz` endpoints (`handlers_health.go`) are correct for Kubernetes/load-balancer integration.

### Pagination is well-factored

The `paginationState` type and `renderPagination()` function (`pagination.go`) are clean and reusable. Cursor-based pagination with `after` + `hasPrev`/`hasNext` is the right approach.

### The payload renderer abstraction is smart

The `PayloadRenderer` interface (`config.go:91-95`) with `DefaultPayloadRenderer` (JSON/CBOR pretty-print) lets consumers plug in domain-specific payload formatting without forking the library.

---

## 3. What DiscordSync Reimplements (and Why)

DiscordSync builds **3 panels from scratch** that dashboardui already provides. Total: **~520 lines** of custom templ + handlers.

### DLQ: `dlq.templ` (84 lines) + `handlers_extras.go:205-263` (59 lines) = 143 lines

DiscordSync's DLQ page (`dlq.templ`) renders dead-lettered events with:
- Per-projection replay forms (dashboardui only has per-projection replay at the detail page level)
- Error code + error family display (dashboardui shows raw error string)
- CSP-safe `data-confirm` forms (dashboardui uses inline `onsubmit`)
- Integration with the shared `Layout`, nav, sidebar, dark mode

DiscordSync's handler (`handlers_extras.go:205-263`) has **dual-path dispatch** (command dispatcher vs direct call) — a production pattern dashboardui doesn't need because it calls `host.ReplayDeadLetters()` directly.

### Projection Health: `projection_health.templ` (203 lines) + `handlers_projection_health_page.go` (50 lines) = 253 lines

DiscordSync's projection health page is **richer** than dashboardui's:
- **Sparkline lag trends** — each projection shows a `display.Sparkline` of recent lag samples (from `observability.SparklineStats`). dashboardui shows plain text lag.
- **HTMX polled regions** — the entire health table auto-refreshes via `polledRegion()` without SSE. dashboardui requires SSE for live updates.
- **Health summary banner** — green/red summary computed from worker states (`projectionHealthBanner`). dashboardui has this in the overview but not on the projection page itself.
- **Error highlighting** — rows with errors get red background. dashboardui uses a simpler badge.
- **Last event relative time** — auto-ticking `<time data-tc-relative>` element. dashboardui doesn't have this.

### Command Audit: `command_audit.templ` (107 lines) + `handlers_command_audit.go` (74 lines) = 181 lines

DiscordSync's command audit page has:
- **Type filter** — dropdown to filter by command type (`filterSelect` with `change from:find select` HTMX auto-submit). dashboardui has no filtering.
- **Payload display** — JSON payload column with expandable formatting. dashboardui only shows type, stream type, stream ID, and command ID (no payload).
- **Offset pagination** — simple `Load more` link. dashboardui uses cursor-based pagination (slightly better, but both work).
- **Integration with `db.Database.ListCommands()`** — reads from its own SQL layer. dashboardui reads from `command.CommandJournal`.

### Why DiscordSync reimplements instead of consuming

The answer is **always the same 5 walls**: dashboardui's panels can't be embedded without pulling in dashboardui's layout, CSS, sidebar, inline scripts, and complete HTML document. The rendering technology makes consumption impossible, so consumers who already have a UI must rebuild the panels in their own stack.

---

## 4. What DiscordSync Is Missing Entirely

Three dashboardui panels have **no DiscordSync equivalent at all**:

### Event Journal Browser (CQRS raw event log)

dashboardui's events browser (`handlers_events.go`, 480 lines) paginates through the raw event journal with:
- Cursor-based pagination over `SeekableJournal.ReadFrom()`
- Filter by event type, stream type, stream ID
- Event detail view with payload rendering (CBOR/JSON pretty-print)
- Version history per event

DiscordSync has a **live SSE event viewer** (`events.templ`) — but that's a completely different tool. It shows live Discord events streaming in real time. It does not browse the CQRS event store. There is no way to look at a specific event by ID, see its payload, or trace its version history.

**This is a real operational gap.** When debugging projection failures, DiscordSync operators have no UI to inspect the raw event that caused a DLQ entry — they must query the SQLite `events` table directly.

### Time-Travel (state reconstruction)

dashboardui's time-travel feature (`handlers_timetravel.go`, 234 lines) reconstructs aggregate state at any point in history by replaying events up to a version. This is one of the most powerful features of event sourcing.

DiscordSync has **nothing equivalent**. You cannot ask "what did message #1234 look like after its 3rd edit but before its 5th?" The event data is all there in the store — DiscordSync just lacks the UI to reconstruct and display it.

### Aggregate Browser

dashboardui's aggregate browser (`handlers_aggregates.go`, 229 lines) lists all streams (aggregates) with their type, ID, and current version. Click through to see all events for that aggregate.

DiscordSync has no aggregate browser. The closest thing is the message detail page's collapsible "Events (N)" timeline, but that's per-message and doesn't list all aggregates.

---

## 5. The Proposed Architecture: 3-Layer Decomposition

The goal: make dashboardui consumable by **any** app — whether it has its own UI or not.

```
cqrs-htmx/dashboardui/
├── core/           Layer 1: Pure data — zero HTML, zero rendering opinions
├── panels/         Layer 2: templ-component panels — embeddable in any templ-components app
└── *.go (existing) Layer 3: The standalone dashboard (rebuilt on 1+2, preserves "mount and go")
```

### Layer 1: `core` — Pure Data Layer

Extract all data-fetching logic. Returns typed structs. Zero HTML imports, zero rendering dependencies.

**What moves here:**

| Current location | What it does | New location |
|---|---|---|
| `buildProjectionStats()` (`handlers_projections.go:18-43`) | Converts `host.Status()` + `host.LagPerProjection()` into display structs | `core.ProjectionStats()` |
| `overviewStats()` (`handler_overview.go:79-172`) | Aggregates events count, aggregates count, projection health, DLQ count, recent events | `core.FetchOverview()` |
| `eventFilter` + `loadFilteredEvents()` (`handlers_events.go:16-100`) | Event journal filtering and loading | `core.ListEvents()` |
| `listStreamsPaged()` (in `handlers_aggregates.go`) | Cursor-paginated stream listing | `core.ListStreams()` |
| DLQ entry loading | Dead letter listing | `core.ListDeadLetters()` |
| Command loading from `CommandJournal` | Command audit data | `core.ListCommands()` |
| `Capabilities` + `capabilities()` (`config.go:150-186`) | Capability detection | `core.DetectCapabilities()` |
| `payload.go` | Payload formatting | `core.DefaultPayloadRenderer` |
| `pagination.go` | Pagination math | `core.PageResult`, `core.ParseCursor()` |
| `sse.go` | SSE broadcasting | `core.SSEBroadcaster` |
| `format.go` | `relativeTime`, `humanByteSize` | `core.RelativeTime()`, `core.HumanByteSize()` |

```go
// Example: core package API
package core

type Config struct {
    EventSource      event.EventSource
    SeekableJournal  event.SeekableJournal
    StreamReader     listing.StreamReader
    ProjectionHost   *projectionhost.Host
    DeadLetterStore  projectionhost.DeadLetterStore
    CommandJournal   command.CommandJournal
    EventBus         event.Bus
    PageSize         int
}

type Capabilities struct { /* same as today, public */ }
func DetectCapabilities(cfg Config) Capabilities

type ProjectionStat struct {
    Name, Status, Lag, Checkpoint, LastError string
    Processed, Errors                        int64
    Restarts                                 int
    StatusKind                               string
}
func ProjectionStats(host *projectionhost.Host) []ProjectionStat

type Overview struct {
    TotalEvents, TotalAggregates string
    Projections                  []ProjectionStat
    DLQCount                     string
    RecentEvents                 []RecentEvent
    HealthStatus, HealthKind     string
}
func FetchOverview(ctx context.Context, cfg Config) Overview

type DeadLetterEntry struct {
    ProjectionName, EventType, EventID string
    Error, ErrorCode, ErrorFamily      string
    FailedAt                           time.Time
}
func ListDeadLetters(ctx context.Context, store projectionhost.DeadLetterStore, projection string) ([]DeadLetterEntry, error)

// Any consumer can use this: templ app, React backend, CLI tool, metrics exporter
```

### Layer 2: `panels` — templ-component Renderers

Each panel becomes a `templ.Component` built with **templ-components** (Lars's library: `display.StatCard`, `display.Badge`, `display.EmptyState`, `display.Grid`, etc.). These slot into any templ-components app's existing layout.

```go
package panels

type PanelOpts struct {
    ReadOnly bool
    BasePath string // for form actions
    Nonce    string // for CSP-safe inline scripts (if any)
}

// Each panel is a standalone templ component — no Layout, no sidebar, no <html>
templ OverviewPanel(stats core.Overview, opts PanelOpts) {
    @display.Grid(display.GridProps{Cols: display.GridCols4, Gap: display.GridGapSM}) {
        @display.StatCard(display.StatCardProps{Label: "Events", Value: stats.TotalEvents})
        @display.StatCard(display.StatCardProps{Label: "Aggregates", Value: stats.TotalAggregates})
        @display.StatCard(display.StatCardProps{Label: "DLQ", Value: stats.DLQCount})
        // ... projection health summary
    }
    // Recent events table, etc.
}

templ ProjectionsPanel(stats []core.ProjectionStat, opts PanelOpts) {
    // Stat cards for counts
    // Table with display.Badge for status, display.Sparkline for lag trend
    // CSP-safe data-confirm forms for reset action
}

templ DLQPanel(entries []core.DeadLetterEntry, opts PanelOpts) {
    if len(entries) == 0 {
        @display.EmptyState(display.EmptyStateProps{
            Title: "No dead-lettered events",
        })
    } else {
        // Per-projection replay forms with data-confirm
        // Table with error code + family display
    }
}

templ EventsBrowser(events []event.Event, page core.PageResult, filter core.EventFilter, opts PanelOpts) { ... }
templ CommandAudit(rows []core.CommandRow, types []string, filter string, opts PanelOpts) { ... }
templ TimeTravelIndex(streams []listing.StreamListing, page core.PageResult, opts PanelOpts) { ... }
templ TimeTravelDetail(streamType string, streamID id.StreamID, events []event.Event, opts PanelOpts) { ... }
templ AggregateBrowser(streams []listing.StreamListing, page core.PageResult, opts PanelOpts) { ... }
templ SnapshotInspector(snapshots []snapshot.Snapshot, page core.PageResult, opts PanelOpts) { ... }
```

**How DiscordSync would consume:**

```go
// DiscordSync — 5 lines instead of 253
func (v *Views) handleProjectionHealth(w http.ResponseWriter, r *http.Request) {
    stats := core.ProjectionStats(v.projectionHost)
    render(w, r, panels.ProjectionsPanel(stats, panels.PanelOpts{
        BasePath:  "",
        ReadOnly:  false,
    }))
}
```

The panel renders inside DiscordSync's existing `@Layout(...)`. It uses the same `display.StatCard`, `display.Badge`, and `display.Sparkline` components as the rest of the app. It matches the design system. It respects the CSP. No second sidebar, no second CSS file, no `fmt.Fprintf`.

### Layer 3: Standalone Dashboard (existing API, rebuilt on 1+2)

The current "mount and go" experience is preserved. The standalone dashboard is rebuilt on top of `core` + `panels`:

```go
// standalone — for apps that don't have their own UI
dash, _ := dashboardui.New(dashboardui.Config{
    EventSource:     store,
    SeekableJournal: store,
    ProjectionHost:  host,
})
dash.Mount(mux, "/dashboard/")
```

Internally, this wires `core.Config` → fetches data → wraps `panels.*` components in a standalone layout. The layout can be a templ version of the current sidebar + header design (or a `display`-based layout using templ-components).

**The key difference:** the standalone layout is just another consumer of the panels, not a prerequisite for using them.

---

## 6. Panel-by-Panel Improvement Opportunities

When porting panels from `fmt.Fprintf` to templ-components, absorb these production-validated improvements from DiscordSync:

### DLQ Panel

| Current (dashboardui) | Improved (from DiscordSync) |
|---|---|
| Per-projection detail page with replay | Per-projection replay forms on the main page (fewer clicks) |
| Raw error string | Error code + error family badges (structured error display) |
| `onsubmit="return confirm(...)"` | `data-confirm` attribute (CSP-safe) |
| `fmt.Fprintf` table rows | `listTableWithHeader` shared component |

### Projections Panel

| Current (dashboardui) | Improved (from DiscordSync) |
|---|---|
| Plain text lag (`proj.Lag`) | `display.Sparkline` showing lag trend over time |
| SSE-only live updates | HTMX `polledRegion` for auto-refresh without SSE |
| Status badge only | Health summary banner (green when all healthy, red when any failed) |
| No row highlighting | Error rows highlighted with red background |
| No last-event display | Relative-time auto-ticking "Last Event" stat card |

### Command Audit Panel

| Current (dashboardui) | Improved (from DiscordSync) |
|---|---|
| No filtering | Type filter dropdown with HTMX auto-submit |
| No payload display | Expandable JSON payload column |
| Keyset pagination only | Offset pagination with "Load more" (simpler UX for audit logs) |

### Events Browser

| Current (dashboardui) | Improvement opportunity |
|---|---|
| Filter by type/streamType/streamID | Add date-range filter, payload search |
| In-memory filter scan (500 events) | Delegate to `SeekableJournal` when available for DB-level filtering |
| No event correlation view | Show all events for the same stream on the detail page |

### Cross-cutting: CSP Safety

Replace all 5 inline `onsubmit` handlers with a `data-confirm` attribute pattern. This is a single delegated listener:

```js
// CSP-safe confirmation — one listener, many forms
document.addEventListener('submit', function(e) {
    var form = e.target.closest('[data-confirm]');
    if (form && !confirm(form.dataset.confirm)) {
        e.preventDefault();
    }
});
```

DiscordSync has this pattern production-validated across all destructive-action forms.

---

## 7. Migration Path

Each step is independently shippable. Steps 1-3 don't touch any consumer.

### Step 1: Extract `core` package (pure refactor)

Move all data-fetching functions from `handlers_*.go` into a new `core/` sub-package. The existing standalone dashboard consumes its own extracted core. No behavior change, no consumer impact.

**Validation:** all existing dashboardui tests pass unchanged (they test the same data through a new package path).

### Step 2: Build `panels` package (templ port)

Port each panel from `fmt.Fprintf` to templ-components, one at a time:
1. Start with DLQ (smallest, most duplicated)
2. Then projections (highest value)
3. Then command audit
4. Then events browser
5. Then time-travel + aggregates
6. Then overview (composes the others)

Each panel absorbs DiscordSync's UX improvements during the port.

**Validation:** rendering tests (snapshot/golden) for each panel.

### Step 3: Rebuild standalone on `core` + `panels`

The standalone dashboard's handlers become thin: fetch data via `core`, render via `panels`, wrap in standalone layout. The standalone layout itself becomes a templ component.

**Validation:** the existing standalone test suite (`dashboard_test.go`, `handler_test.go`, etc.) passes.

### Step 4: Wire DiscordSync (pure deletion + import)

For each of the 3 overlapping panels:
1. Delete the custom templ file (`dlq.templ`, `projection_health.templ`, `command_audit.templ`)
2. Delete the custom handler logic
3. Import `core` + `panels`
4. Write a 5-line handler that fetches data and renders the panel
5. Run the test suite

Then wire the missing go-cqrs-lite interfaces (`EventSource`, `SeekableJournal`) to activate time-travel + event journal browser — **net-new capabilities for free**.

**Validation:** DiscordSync's existing web test suite (19 a11y checks, HTMX contract tests, DOM helper assertions, golden snapshots).

---

## 8. Concrete Impact Numbers

### For dashboardui

| Metric | Before | After |
|---|---|---|
| `fmt.Fprintf` / `strings.Builder` call sites | 178 | ~0 (all in panels, replaced by templ) |
| Embedded CSS lines | 335 | 0 (uses templ-components + Tailwind) |
| Inline `onsubmit` handlers | 5 | 0 (CSP-safe `data-confirm`) |
| Inline `<script>` blocks | 1 (toast container) | 0 (use templ-components `feedback.ToastContainer`) |
| Consumable by templ-components apps | No | Yes |
| Panels available to non-templ apps (via `core`) | No | Yes |

### For DiscordSync

| Metric | Before | After |
|---|---|---|
| Custom DLQ templ + handler | 143 lines | ~5 lines (handler only) |
| Custom projection health templ + handler | 253 lines | ~5 lines |
| Custom command audit templ + handler | 181 lines | ~5 lines |
| **Total lines deleted** | — | **~570 lines** |
| Event journal browser | Missing | Available (wire `SeekableJournal`) |
| Time-travel state reconstruction | Missing | Available (wire `EventSource`) |
| Aggregate browser | Missing | Available (wire `StreamReader`) |

### For the cqrs-htmx ecosystem

| Metric | Before | After |
|---|---|---|
| Rendering consistency across modules | dashboardui is the outlier (`fmt.Fprintf`); adminui uses templ | Both use templ + templ-components |
| Design system consistency | dashboardui has its own CSS; adminui uses Tailwind | Both use templ-components |
| Cross-module panel sharing | Impossible | Any panel embeddable in any templ-components app |

---

## Appendix: The Sibling Module Precedent

`adminui` already proves this works. It is a complete admin dashboard built on templ + Tailwind v4 + templ-components:

| Aspect | dashboardui | adminui |
|---|---|---|
| HTML generation | `fmt.Fprintf` + `strings.Builder` (178 sites) | templ (zero string builders in non-templ .go) |
| Design system | 335-line embedded CSS (`dashboardCSS`) | Tailwind v4 + templ-components (`assets/admin-tw.css`) |
| Component library | None (hand-rolled `.btn`, `.data-table`, etc.) | `templ-components/display` (uses `display.StatCard`, etc.) |
| Layout | `strings.Builder` in `layout.go:16-51` | templ component in `layout.templ` |
| Inline event handlers | 5 `onsubmit` + 1 `<script>` | 0 |
| Generated files | No `.templ` files | `*_templ.go` committed |
| CSP compatibility | Requires `unsafe-inline` | Compatible with strict CSP |

dashboardui should follow adminui's precedent. The rendering technology decision was made when dashboardui was a standalone experiment — it's time to align it with the ecosystem.
