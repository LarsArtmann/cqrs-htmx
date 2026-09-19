# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

_(nothing yet)_

## [v4.11.0] - 2026-09-19

### Changed

- **Go 1.27.1 floor + dependency carrier:** `go.mod` requires Go 1.27.1 (matching the 2026-09-19 fleet cut; `go-etag v0.4.0` and go-cqrs-lite's re-pin force the floor). No domain-type or behavior changes since v4.9.0 (verified by tree diff): this release is the coordinated-train cut carrying the dependency alignment — go-branded-id v0.5.1→v0.6.0, go-codec v0.2.0→v0.3.0, go-cqrs-lite command v4.8.1→v4.10.0, event v4.9.0→v4.11.0, id v4.5.0→v4.6.0, go-error-family v0.10.0→v0.10.1.

## [v4.3.0] – [v4.9.0] — 2026-08-14 → 2026-09-09

- Coordinated lockstep bumps riding the family release trains (identity-model shipped v4.8.0 and v4.9.0 in the 2026-08-14/2026-09-01 trains). No identity-model-specific feature entries were recorded in this file for those trains; the historical detail lives in the root `CHANGELOG.md` sections for those versions.

## [v4.2.0] - 2026-08-07

## [v4.1.1] - 2026-07-27

### Changed

- **Simplified `HasRole`/`HasAnyRole`** (`fold.go`, `membership.go`): replaced manual loops with `slices.Contains`/`slices.ContainsFunc` for clarity and correctness.
- **go-cqrs-lite v4.1.0**: all go-cqrs-lite module references updated to v4.1.0 (codec at v4.1.1).

### Added

- **Module metadata files**: `.editorconfig`, `.gitattributes`, `.gitignore`, `CONTRIBUTING.md`, `CHANGELOG.md` — standard project hygiene for an independently versioned Go module.

## [v4.1.0] - 2026-07-24

### Added

- **Pure domain types for event-sourced identity management** (ADR-0043): IDs (UserID, TenantID, BotID, ActorID), event payloads (22 structs), commands (19 structs with accessor methods), state structs + fold functions (FoldUser/FoldMembership/FoldTenant/FoldBot), Authz engine (Casbin-backed — ADR-0044), RBAC model + default policies + role hierarchy, domain errors (errorfamily-only, no HTTP dependency), crypto helpers, upcaster registry, exported constants (41 event/command/aggregate-type constants).
- **Casbin as a first-class dependency** (ADR-0044): the Authz engine wraps a Casbin enforcer, enabling declarative role/tenant authorization policies.
