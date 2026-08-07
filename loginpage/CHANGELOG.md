# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [v4.7.0] - 2026-08-07

### Added

- HTMX partial rendering support.
- Updated dependencies (root v4.7.0, usermgmt v4.7.0).

## [v4.6.1] - 2026-07-27

### Changed

- **Lockstep version bump** with root `cqrs-htmx/v4` v4.6.1 — no code changes in this sub-module; the bump keeps the lockstep release consistent. go-cqrs-lite `v4.1.0` → `v4.2.0` (command, event, id, idempotency, query); go-branded-id `v0.3.2` → `v0.5.0`.

## [v4.6.0] - 2026-07-27

### Changed

- **Lockstep version bump** with root `cqrs-htmx/v4` v4.6.0 — no code changes in this sub-module; the bump keeps the lockstep release consistent. go-cqrs-lite sub-module version refs aligned to v4.1.0; go-error-family bumped to v0.10.0.

## [v4.5.0] - 2026-07-24

### Changed

- **Dependency tidy**: aligned go-cqrs-lite and sibling module version refs across the workspace.

## [v4.4.0] - 2026-07-23

### Changed

- **httputil upgrade** to v0.6.0 — adapted to go-cqrs-lite stack/sqlopt split.
- **identity-model extraction**: identity-model extracted as a standalone module; loginpage now depends on it transitively via usermgmt.
- **go-cqrs-lite schema bump** to v4.0.3.

## [v4.3.0] - 2026-07-12

### Changed

- **go-cqrs-lite v3 → v4 migration**: all module paths migrated from `/v3` to `/v4`; vendored eventtest removed.

## [v4.0.0] - 2026-07-12

### Added — loginpage extracted as independent module

- **New module**: `github.com/larsartmann/cqrs-htmx/loginpage/v4`
- Self-contained passwordless WebAuthn login page (templ + HTMX).
- OAuth2 button support with auto-discovery — providers are rendered automatically when configured on the usermgmt service.
- Server-side ID generation for WebAuthn registration ceremonies.
- Graceful no-auth fallback (renders a message when no auth providers are configured).
- Depends on `cqrs-htmx/v4` and `cqrs-htmx/usermgmt/v4` — renders a ready-made login UI that mounts alongside any usermgmt-backed application.
