# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [v4.2.0] - 2026-07-08

### Changed

- **go-cqrs-lite v3.5.0 → v3.7.4**: Aligned with root and usermgmt dependency upgrades.
- **go-error-family direct import**: Migrated from transitive dependency to direct import.
- Refreshed templ generated output to match latest CLI version.

### Fixed

- **go.work replace+use conflict**: Removed `replace` directive from go.work that conflicted with BuildFlow's `use` directive. Per-module go.mod files retain their own `replace` for GOWORK=off compatibility.

## [v4.1.1] - 2026-07-04

### Changed

- **httputil v0.3.0 → v0.4.0**: Transitive dependency bump. No API or behavior change.
- **templ-components v0.6.0 → v0.6.1**: Minor dependency bump.
- **HTMX v2.0.9 → v2.0.10**: Updated embedded HTMX JS. Extracted `serveJS` helper for shared caching logic between HTMX core and extensions.

## [v4.0.1] - 2026-07-02

### Fixed

- Normalized templ generated import style for BuildFlow compatibility.
- Corrected pseudo-versions in go.mod from `v0.0.0` to valid `v4.0.0` paths.

## [v4.0.0] - 2026-07-02

### Changed — v4 Module Path Bump

- **BREAKING**: Module path changed from `github.com/larsartmann/cqrs-htmx/adminui/v3` to `github.com/larsartmann/cqrs-htmx/adminui/v4`.
- Updated dependency on root `cqrs-htmx/v4` and `usermgmt/v4`.

## [v3.5.0] - 2026-07-01

### Changed

- Aligned with root v3.5.0 release (go-cqrs-lite v3.5.0).

## [v3.0.0] - 2026-06-27

### Added

- **First release of adminui module.** A ready-made, good-looking Admin Dashboard for usermgmt-backed apps.
- One-call mount behind session middleware via `New()` + `Mount()` / `Handler()`.
- Two scopes: `ModeSuperAdmin` (global dashboard/users/tenants/audit) and `ModeTenantAdmin` (tenant-scoped members/audit).
- Auth-agnostic: reads `*usermgmt.User` from context (consumer's session middleware).
- Modern CSS design system (light/dark via `prefers-color-scheme`, accent color injection).
- HTMX patterns: live search, `hx-confirm` destructive actions, `HX-Redirect`, toast notifications via `HX-Trigger`.
- Embedded assets via `go:embed` — no build step for consumers.
- Depends on root `cqrs-htmx/v3` (reuses `HTMXScriptHandler`) + `usermgmt/v3` + `a-h/templ`.
