# Changelog

All notable changes to this module are documented here. Format based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added


- **Deployment presets — the three common topologies as one-liners (M23):** `RecommendedMemoryDeployment()` (dev/test), `RecommendedSQLiteDeployment(dsn)` (single WAL file, every role), and `RecommendedSplitSQLiteDeployment(eventsDSN, projectionsDSN)` (journal and projections on separate files — the production shape where replays never contend with the write path). Each returns a plain `system.DeploymentConfig` to inspect or tweak; SQLite presets set `journal_mode=wal`. The module gets a README (declarative-first quick start, preset table, projection knobs, legacy note), the package doc points at `examples/system-demo/` and the presets, a godoc `ExampleDomainConfig` pins the one-call wiring, and the leveraging-system-metaengine guide's three hand-rolled DeploymentConfig snippets dedupe onto the presets. The memory preset is live-proven — `setupDeclarativeSystem` in the test suite now boots through it.

- **Durable checkpoints + host tuning on the declarative path — `DomainConfig` grows options (M22):** `DomainConfig(opts ...DomainOption)` now accepts `WithCheckpointStore` (becomes `system.DomainConfig.CheckpointStore` — left nil, system.New's engine-backed ADR-0142 default applies), `WithDeadLetterStore` (replaces the in-memory DLQ default), and `WithHostOptions(...)` (projectionhost options appended after systemadapter's curated defaults — memory DLQ threshold 10, 3 restarts, capped backoff, batch 256, the same tuning the legacy ProjectionLayer ships; consumer per-field options win). `DomainOption` and `ProjectionLayerOption` are now the same underlying option type, so the checkpoint/dead-letter vocabulary works at both call sites unchanged. Pinned by `TestDomainConfig_DefaultsAndOverrides` and `TestDeclarative_DurableCheckpointSurvivesRestart` — the restart proof runs a file-backed SQLite journal through two `system.New` boots with one shared checkpoint store and asserts the second boot resumes at exactly the saved positions. Guide § Custom checkpoint / dead-letter stores rewritten accordingly, including the durable-checkpoint + fresh-in-memory-read-model semantics (no replay = stays empty; pair with durable read models or rebuild). Backward compatible: the no-option `DomainConfig()` call is unchanged.

## [v4.11.0] - 2026-09-20

First published tag, cut in lockstep with the cqrs-htmx v4.11.0 family train (all sibling modules v4.11.0, go-cqrs-lite aligned to the 2026-09-19/20 release wave).

### Added

- **Bridge from cqrs-htmx's identity domain into go-cqrs-lite's `system.New()` composition root and `metaengine` storage planner:** `DomainConfig()` pre-wires all 4 deciders + 20 commands + the 21-event TypeDecoder; `EventTypeDecoder()` maps event types to payload structs for `metaengine/projectionadapter`; `NewProjectionLayer(sys)` builds usermgmt read models, Casbin authz, and the audit log on the system's event store + bus, with `WithCheckpointStore`/`WithDeadLetterStore` options. Typed query helpers (`FindTenantByID`, `FindUserByEmail`, bot/membership lookups, ...) run over `metaengine` views. Declarative-projection declarations (`DeclarativeProjections()`) cover the `system.New` projection route. Guide: `docs/guides/leveraging-system-metaengine.md`; runnable proof: `examples/system-demo/`.

### Changed

- **Dependency alignment:** requires bumped to the published go-cqrs-lite maxima (notably `metaengine/projectionadapter/v4 v4.5.0`, which carries `OccurredAt` on `EventWithID`, and `metaengine/v4 v4.14.0` with the Reset API) — the long-standing local sibling `replace` directives are gone and the module now builds hermetically (`GOWORK=off`) from published tags alone.
