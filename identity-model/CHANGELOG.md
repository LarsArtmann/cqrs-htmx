# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

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
