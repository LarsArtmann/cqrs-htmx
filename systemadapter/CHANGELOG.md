# Changelog

All notable changes to this module are documented here. Format based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [v4.11.0] - 2026-09-20

First published tag, cut in lockstep with the cqrs-htmx v4.11.0 family train (all sibling modules v4.11.0, go-cqrs-lite aligned to the 2026-09-19/20 release wave).

### Added

- **Bridge from cqrs-htmx's identity domain into go-cqrs-lite's `system.New()` composition root and `metaengine` storage planner:** `DomainConfig()` pre-wires all 4 deciders + 20 commands + the 21-event TypeDecoder; `EventTypeDecoder()` maps event types to payload structs for `metaengine/projectionadapter`; `NewProjectionLayer(sys)` builds usermgmt read models, Casbin authz, and the audit log on the system's event store + bus, with `WithCheckpointStore`/`WithDeadLetterStore` options. Typed query helpers (`FindTenantByID`, `FindUserByEmail`, bot/membership lookups, ...) run over `metaengine` views. Declarative-projection declarations (`DeclarativeProjections()`) cover the `system.New` projection route. Guide: `docs/guides/leveraging-system-metaengine.md`; runnable proof: `examples/system-demo/`.

### Changed

- **Dependency alignment:** requires bumped to the published go-cqrs-lite maxima (notably `metaengine/projectionadapter/v4 v4.5.0`, which carries `OccurredAt` on `EventWithID`, and `metaengine/v4 v4.14.0` with the Reset API) — the long-standing local sibling `replace` directives are gone and the module now builds hermetically (`GOWORK=off`) from published tags alone.
