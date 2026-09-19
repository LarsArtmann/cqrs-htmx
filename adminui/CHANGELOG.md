# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

_(nothing yet)_

## [v4.11.0] - 2026-09-19

### Added

- **templ-components core-shell migration (2026-09-17→09-19):** the hand-rolled layout shell is retired — `Layout` is now `layout.Base` + `layout.AppShell` + `navigation.SidebarNav` (the drawer keeps the legacy `.admin-sidebar`/`.admin-toggle` class names so `admin.js` is unchanged), and every `@Layout(p)` wrapper call site became a `content templ.Component` parameter (templ 0.3.1020 has no children-as-value). Verified across desktop/dark/mobile-390/tablet-768 with a browser run against a throwaway demo built from HEAD.
- **Structured error pages:** new published require `github.com/larsartmann/templ-components/errorpage v1.18.0`; `Handler.writeErrorPage` renders the `ErrorPage` card (bare for HTMX swaps, inside `Layout` for navigations, HTTP statuses preserved) and a guarded 404 catch-all serves `NotFound404` — every bare `http.Error`/`http.NotFound` in the panel handlers is replaced (render.go's post-header fallback stays, deliberate). 4 new errorpage tests.
- **Audit "Who" columns resolve actor emails** via the shared `resolveAuditEmails` read-model helper (dashboard + audit page).

### Fixed

- **Sidebar transparency + dark-mode gray inversion (browser-evidence verification):** (1) SidebarNav's `bg-[var(--tc-sidebar-bg)]` needs the variable defined — adminui's compiled bundle has no library `:root` defaults, so `.admin-shell` sets it alongside the CSP-safe `--tc-sidebar-w` (AppShell's inline style attribute is dropped under the CSP nonce posture); (2) adminui's `@theme` maps gray-800/900 to `--text`, which flips light in dark mode, so the library's `dark:` dark surfaces (e.g. `thead dark:bg-gray-800`) painted as light bands — the dark-mode media block pins those two tokens to the literal dark palette (safe: all 115 library `text-gray-800/900` occurrences carry a `dark:` override). `build-adminui-css` now also scans the errorpage module dir including its `styles.go`.

## [v4.8.0] – [v4.10.0] — 2026-08-14 → 2026-09-07

- Coordinated lockstep bumps riding the family release trains (root v4.8.0–v4.10.0, usermgmt, templ-components v1.18.0 CSS-bundle era). No adminui-specific feature entries were recorded in this file for those trains; the historical detail lives in the root `CHANGELOG.md` sections for those versions.

## [v4.7.0] - 2026-08-07

### Added

- CSP nonce support for inline scripts.
- HTMX partial rendering support via `RenderPartialOrFull`.
- `feedback.ToastContainer` adoption (bridges `adminui:toast` HX-Trigger events).
- `htmx.GlobalErrorHandling` adoption (5xx retry, network error toast, session-expiry redirect).
- Offline sync served from root module (`sync-worker.js`, `sync-client.js`).

## [v4.6.0] - 2026-07-26

### Changed

- **Dependency bump**: templ-components `v1.1.0` → `v1.2.0`.
- **Dedup sweep**: `ToastDetail` extracted to root module (`cqrshtmx.ToastDetail`); adminui now re-exports the shared type. See root CHANGELOG `[v4.6.0]` for the full dedup sweep summary.

## [v4.5.0] - 2026-07-24

### Changed

- **Explicit dependencies on root + usermgmt**: go.mod now directly requires `cqrs-htmx/v4` and `cqrs-htmx/usermgmt/v4` (previously transitive). All transitive dependencies materialized in go.sum.
- **go-cqrs-lite v4.0.x dependency alignment**: Updated all go-cqrs-lite module references.

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
