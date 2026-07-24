# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [v4.0.0] - 2026-07-24

### Added

- **First release** of the dashboardui module: a ready-made CQRS/Event-Sourcing observability dashboard.
- **Dashboard panel** (`dashboard.go`, `handler.go`, `handlers.go`): Plug-in HTTP dashboard showing projection health, event catalog overview, and system status. SSE-powered real-time updates.
- **SSE integration** (`sse.go`): Server-Sent Events endpoint for live projection status streaming to the dashboard.
- **Layout and rendering** (`layout.go`, `render.go`, `handler_overview.go`): Full HTML layout with sidebar navigation, responsive design, and partial/full render modes for HTMX.
- **Config** (`config.go`): `Config` struct with `AuthService` (for auth checks), projection status provider, event catalog, and customizable refresh interval.
- **Payload types** (`payload.go`): DTOs for dashboard data (projection statuses, event catalog entries, system metrics).
- **Templ rendering** (`templ_render.go`): Integration with templ for type-safe HTML generation.
