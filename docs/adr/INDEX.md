# ADR Index

> Architecture Decision Records, sorted by number. Each entry shows title and status.

| #                                                 | Title                                                                           | Status                                                           |
| ------------------------------------------------- | ------------------------------------------------------------------------------- | ---------------------------------------------------------------- |
| [0001](0001-htmx-go-decision.md)                  | Reject htmx-go Dependency                                                       | Accepted                                                         |
| [0002](0002-userid-type-split.md)                 | UserID Type Split Between Root Module and usermgmt                              | Accepted                                                         |
| [0003](0003-numeric-ids-sql-stores.md)            | Numeric IDs for SQL Store Backends                                              | Superseded (UserStore removed; event-sourced now — see ADR 0006) |
| [0004](0004-sse-websocket-support.md)             | SSE and WebSocket Support                                                       | Accepted                                                         |
| [0005](0005-go-cqrs-lite-v230-adoption.md)        | go-cqrs-lite v2.3.0 API Adoption                                                | Accepted                                                         |
| [0006](0006-event-sourced-user-aggregate.md)      | Event-Sourced User Aggregate                                                    | Accepted                                                         |
| [0007](0007-dependency-upgrade-v2.4.0.md)         | Dependency Upgrade to v2.4.0 and Idiomatic Error-Family Adoption                | Accepted                                                         |
| [0008](0008-catalog-sub-package.md)               | Catalog Sub-Package for API Documentation                                       | Superseded by ADR 0020                                           |
| [0009](0009-go-cqrs-lite-module-selection.md)     | go-cqrs-lite Module Selection                                                   | Accepted                                                         |
| [0010](0010-transport-parity.md)                  | Transport Parity (SSE ↔ WebSocket)                                              | Accepted                                                         |
| [0011](0011-event-signing-encryption.md)          | Opt-in Event Signing & Encryption                                               | Accepted                                                         |
| [0012](0012-sql-session-store.md)                 | SQL SessionStore                                                                | Accepted                                                         |
| [0013](0013-event-schema-versioning-upcasters.md) | Event Schema Versioning via Upcasters                                           | Accepted                                                         |
| [0014](0014-oauth2-oidc-integration.md)           | OAuth2/OIDC Integration                                                         | Accepted                                                         |
| [0015](0015-identity-model-redesign.md)           | Identity Model Redesign — Actor, Tenant, Membership                             | Accepted                                                         |
| [0016](0016-go-cqrs-lite-v3-migration.md)         | go-cqrs-lite v3.0.0 Migration — Projection Rewrite                              | Accepted                                                         |
| [0017](0017-reconcile-http-status-mapping.md)     | Reconcile HTTP Status Mapping with go-error-family Upstream                     | Accepted                                                         |
| [0018](0018-unify-userid.md)                      | Unify UserID — usermgmt Adopts id.UserID                                        | Accepted                                                         |
| [0019](0019-usermgmt-decomposition-blocked.md)    | usermgmt God-Package Decomposition                                              | Blocked (requires architectural redesign)                        |
| [0020](0020-merge-catalog-into-go-cqrs-lite.md)   | Merge catalog/ into go-cqrs-lite                                                | Accepted                                                         |
| [0021](0021-identity-module-design-spike.md)      | Identity Module Design Spike — Resolving the ActorID Split Brain                | Accepted (Design Spike)                                          |
| [0022](0022-stack-materialize-evaluation.md)      | stack.Materialize Evaluation — Prototype, Findings, and Per-Read-Model Decision | Accepted (evaluation complete — partial adoption)                |
| [0023](0023-command-sync.md)                      | Command-Sync Architecture                                                       | Accepted                                                         |
| [0024](0024-honest-ui.md)                         | Honest UI Protocol                                                              | Accepted                                                         |
| [0025](0025-phase2-research.md)                   | Phase 2 Offline Architecture — Research & Decision Framework                    | Superseded (see ADR 0027 + ADR 0029)                             |
| [0026](0026-command-idempotency-store.md)         | Command Idempotency Store                                                       | Accepted                                                         |
| [0027](0027-decide-stays-on-server.md)            | decide() stays on the server (Queue-Only client)                                | Accepted                                                         |
| [0028](0028-brand-all-id-types.md)                | Brand all ID types with go-branded-id                                           | Accepted                                                         |
| [0029](0029-sharedworker-phase2a.md)              | SharedWorker for Phase 2a Offline Command Sync                                  | Accepted                                                         |
| [0030](0030-phase2-persistence-strategy.md)       | Phase 2 Persistence Strategy — SharedWorker with IndexedDB                      | Proposed                                                         |
| [0031](0031-projection-lifecycle-decision.md)     | Projection Lifecycle — StartProjections vs projectionhost vs CatchUpSubscriber  | Proposed                                                         |
