# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added

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
