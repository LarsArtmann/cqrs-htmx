# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [v4.0.2] - 2026-07-08

### Changed

- **go-error-family direct import**: Migrated from transitive dependency (via go-cqrs-lite event/v3) to direct import.

### Fixed

- **go-cqrs-lite version drift**: Aligned to latest available tags across all sibling modules.

## [v4.0.1] - 2026-07-02

### Added

- Fuzz tests on WebAuthn JSON serialization boundary.

### Fixed

- Resolved lint issues across auth sub-modules.

## [v4.0.0] - 2026-07-02

### Added — WebAuthn extracted as independent module

- **New module**: `github.com/larsartmann/cqrs-htmx/usermgmt/webauthn/v4`
- Implements the `WebAuthnProvider` interface from core `usermgmt` via structural typing (no import of core needed).
- `Provider` type with `BeginRegistration`/`FinishRegistration`/`BeginLogin`/`FinishLogin` methods (`[]byte` JSON boundary).
- Depends only on `go-webauthn/webauthn v0.17.4` — zero transitive auth deps.
- 18 tests: W3C spec ceremony vectors, credential conversion, error cases.
- See ADR-0035 for the extraction rationale and interface design.
