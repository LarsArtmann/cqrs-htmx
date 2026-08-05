# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added

- **Detail views for commands, queries, and DLQ entries** (`handlers_audit.go`, `handlers_dlq.go`, `handler.go`): New routes `/commands/{id}`, `/queries/{id}`, `/dead-letters/{projection}/{eventID}` render full detail pages with metadata tables, copyable IDs, and pretty-printed JSON payloads.
- **Projection detail view** (`handlers_projections.go`, `handler.go`): New route `/projections/{name}` shows checkpoint, processed/errors/restarts counters, lag, last error, DLQ link, and reset action for a single projection. Projection names in the index table are now clickable links.
- **Total count display** (`pagination.go`, all index handlers): Paginated pages now show "Showing X-Y of Z" when the total is known (last page). Uses `WithCountInfo()` builder pattern — no extra count queries for append-only logs.
- **Page-size selector** (`pagination.go`): Dropdown in the pagination bar lets users choose 25/50/100/200 items per page via `?limit=` query param.
- **Sortable column headers** (`sort.go`, `handlers_events.go`): Events table columns (Time, Type, Stream Type, Version) are clickable for ascending/descending sort via `?sort=` and `?dir=` query params. Headers show arrow indicators.
- **HTMX-powered filter form** (`handlers_events.go`, `layout.go`): Event filter form uses `hx-get`/`hx-target`/`hx-select`/`hx-swap`/`hx-push-url` for partial content swapping without full page reloads.
- **HTMX partial rendering** (`render.go`, `layout.go`): Server detects `HX-Request` header and renders title+main only (no full HTML document). Layout includes `data-hx-boost` for progressive enhancement.
- **Payload copy and download** (`handlers_events.go`, `layout.go`): Event detail page has Copy (clipboard API with toast) and Download JSON (Blob download) buttons for event payloads.
- **Keyboard navigation for time-travel slider** (`handlers_timetravel.go`, `layout.go`): Arrow keys anywhere on the time-travel page move the version slider. Live value display updates as the slider moves.
- **CSV export** (`export.go`, all index handlers): `?format=csv` on events, commands, and queries index pages exports up to 10,000 rows as a downloadable CSV file.
- **JSON API mode** (`export.go`, all index handlers): `?format=json` on events, commands, and queries index pages returns a JSON array of row objects.
- **SSE live event injection** (`layout.go`): JavaScript listener for `dashboard:event` custom events prepends new rows to the events table and refreshes projection health (capped at 50 rows).
- **Core data layer** (`core/` package): Capabilities, pagination, events, overview, payload, and format functions extracted into a pure-data `core` subpackage, bridged via type aliases in `core_bridge.go`.
- **Core package unit tests** (`core/*_test.go`): Comprehensive tests for DetectCapabilities, EventFilter, pagination cursor math, FetchOverview, LoadRecentEvents/LoadFilteredEvents/LoadEventByID, FindEventNeighbors, RelativeTime, HumanByteSize, DefaultPayloadRenderer, PrettyJSON, and DLQProjectionLinks.
- **CSP-safe rendering** (`layout.go`, `handlers_dlq.go`, `handlers_projections.go`, `handlers_snapshots.go`): All 6 inline `onsubmit="return confirm(...)"` handlers replaced with `data-confirm` attribute pattern. Inline toast `<script>` block moved to external `dashboardJS`. Single delegated `submit` listener for `data-confirm` forms. `dashboard.js` now always loaded (was conditional on EventBus).
- **Demo projection host** (`examples/dashboard-demo/main.go`): Demo now includes a projection host with a `user-read-model` projection so the projections panel and detail view show live data.
- **Templ migration evaluation** (`docs/planning/templ-migration-evaluation.md`): Documents the tradeoffs of migrating from strings.Builder to templ, with a recommendation to defer.

### Changed

- **`IMPROVEMENT_IDEAS.md` pruned** from 883 lines to ~60 lines after implementing the high-value ideas.
- **Generic journal scanning** (`handlers_audit.go`): `loadCommandByID`/`loadQueryByID` refactored to share `scanJournalByID`/`findInAll` generic helpers, eliminating code duplication.
- **Lint config updates** (`.golangci.yml`): Fixed broken canonicalheader exclusion patterns, added exhaustruct excludes for dashboardui types, added wrapcheck/revive exclusions for core_bridge.go.

### Fixed

- **Mobile responsive design** (`layout.go`, all handler files): Hamburger menu with slide-in sidebar drawer and backdrop overlay for screens <768px. All buttons have 44px minimum touch targets (WCAG 2.5.5). Data tables wrapped in horizontal scroll containers. Filter bar controls stack vertically on mobile. Stat card grid collapses to 2 columns.
- **Accessibility aria-labels** (`handlers_projections.go`, `handlers_dlq.go`, `handlers_snapshots.go`): All interactive elements (Reset, Replay, Delete, Purge buttons and forms) now have descriptive `aria-label` attributes for screen reader users.
- **Skip-to-content link** (`layout.go`): Keyboard users can bypass the sidebar navigation directly to main content.
- **Copy-to-clipboard** (`layout.go`, `handler_overview.go`, `handlers_events.go`, `handlers_aggregates.go`, `handlers_audit.go`, `handlers_timetravel.go`, `handlers_snapshots.go`): Identifiers (event IDs, stream IDs, correlation/causation IDs, user IDs, request IDs) are click-to-copy with toast confirmation.
- **Health/filter/CSS-JS/pagination tests** (`handlers_health_test.go`, `handlers_security_test.go`): 8 new tests covering healthz/readyz/versionz endpoints, event filtering by type, CSS/JS serving headers, and pagination cursor preservation with active filters.
- **EventBus live updates in demo** (`examples/dashboard-demo/main.go`): Demo now wires `eventtest.FakeBus` for SSE live updates and publishes new events every 5 seconds. Expanded seed data to 8 users and 6 orders (28 events for pagination testing).
- **ROADMAP** (`ROADMAP.md`): Documents the future templ + Tailwind v4 migration plan.

### Fixed

- **DLQ format string crash** (`handlers_dlq.go`): Replay and Purge form opening tags had 4 `%s` placeholders but only 3 arguments — would have panicked at runtime. Fixed by adding the missing 4th `esc(proj)` argument.
- **Heading hierarchy** (all handler files): Corrected heading levels throughout — page titles use `<h2>`, section headers use `<h3>`, sub-sections use `<h3>` instead of incorrectly nested `<h4>` under `<h3>`.
- **`encodingBadgeClass` semantics** (`format.go`): JSON/empty encoding changed from `badge-ok` (green) to `badge-neutral` (gray) — JSON is not a "success" state.
- **JSON export encoder** (`export.go`): Removed `encoder.SetEscapeHTML(true)` — method does not exist on `jsontext.Encoder` (stdlib-only API). Replaced with `jsontext.NewEncoder(w, jsontext.WithIndent("  "))` matching the usermgmt export pattern.

## [v4.1.0] - 2026-07-26

### Added

- **SSE reconnect replay** (`sse.go`, `dashboard.go`): the SSE handler reads `Last-Event-ID` on reconnect and replays missed events from the journal via `cqrshtmx.JournalSSEStore` + `cqrshtmx.ReplayEvents`. On first connect, recent history is backfilled (up to `DefaultMaxReplay=1000`). `Dashboard.Close()` disconnects all SSE clients.
- **SSE event IDs** (`sse.go`): emitted SSE events now carry the domain event ID, enabling reconnect dedup.
- **Heartbeat config** (`config.go`): `Config.SSEHeartbeatInterval` (15s default) drives a heartbeat to keep proxies from killing idle connections.

### Fixed

- **`Dashboard.Close()` event-bus subscription leak** (`dashboard.go`, `sse.go`): `Close()` now signals a done-channel that makes the event-bus handler a no-op before closing the broadcaster. Uses `sync.Once` for idempotent shutdown.
- **ErrorFamily compliance** (`handlers.go`, `payload.go`): migrated 7 `fmt.Errorf` calls to `errorfamily` constructors (`WrapInfrastructure`, `WrapCorruption`, `Newf`). `nix run .#errorfamily` now reports 0 violations for dashboardui.

## [v4.0.0] - 2026-07-24

### Added

- **First release** of the dashboardui module: a ready-made CQRS/Event-Sourcing observability dashboard.
- **Dashboard panel** (`dashboard.go`, `handler.go`, `handlers.go`): Plug-in HTTP dashboard showing projection health, event catalog overview, and system status. SSE-powered real-time updates.
- **SSE integration** (`sse.go`): Server-Sent Events endpoint for live projection status streaming to the dashboard.
- **Layout and rendering** (`layout.go`, `render.go`, `handler_overview.go`): Full HTML layout with sidebar navigation and responsive design.
- **Config** (`config.go`): `Config` struct with capability-detected interfaces (EventSource, Journal, SeekableJournal, StreamReader, ProjectionHost, DeadLetterStore, CommandJournal, QueryJournal, SnapshotStore, EventBus), customizable title, accent color, page size, and read-only mode.
- **Payload rendering** (`payload.go`): `PayloadRenderer` interface and `DefaultPayloadRenderer` for JSON/CBOR pretty-printing of event payloads.
