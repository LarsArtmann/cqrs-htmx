# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [v4.7.0] - 2026-08-07

### Added

- TOTP replay-window documentation tests (stateless design, RFC 6238 §5.2).

## [v4.6.1] - 2026-07-27

### Changed

- **Lockstep version bump** with root `cqrs-htmx/v4` v4.6.1 — no code changes in this sub-module; the bump keeps the lockstep release consistent. go-cqrs-lite `v4.1.0` → `v4.2.0` (command, event, id, idempotency, query); go-branded-id `v0.3.2` → `v0.5.0`.

## [v4.6.0] - 2026-07-26

### Changed

- **Lockstep version bump** with root `cqrs-htmx/v4` v4.6.0 — no code changes in this sub-module; the bump keeps the lockstep release consistent. go-cqrs-lite sub-module version refs aligned to v4.1.0; go-error-family bumped to v0.10.0.

## [v4.0.2] - 2026-07-08

### Changed

- **go-error-family direct import**: Migrated from transitive dependency (via go-cqrs-lite event/v3) to direct import. Error constructors now use `errorfamily.New*`/`Wrap*` directly.
- Enriched error contexts with domain-specific identifiers.

### Fixed

- **go-cqrs-lite version drift**: Aligned to latest available tags across all sibling modules.

## [v4.0.1] - 2026-07-02

### Fixed

- Corrected pseudo-versions in go.mod from `v0.0.0` to valid `v4.0.0` paths.
- Resolved lint issues across auth sub-modules.

## [v4.0.0] - 2026-07-02

### Added — TOTP extracted as independent module

- **New module**: `github.com/larsartmann/cqrs-htmx/usermgmt/totp/v4`
- Implements the `TOTPProvider` interface from core `usermgmt` via structural typing (no import of core needed).
- `Provider` type with `GenerateSecret` and `ValidateCode` methods.
- Depends only on `pquerna/otp v1.5.0` — zero transitive auth deps.
- 3 tests: secret generation, code validation, default config.
- See ADR-0035 for the extraction rationale and interface design.
