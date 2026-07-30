# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

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
