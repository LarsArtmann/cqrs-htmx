# Roadmap — cqrs-htmx

> Long-term direction and raw ideas not yet refined into actionable tasks.
> For short-term work, see [TODO_LIST.md](TODO_LIST.md).
> For what exists today, see [FEATURES.md](FEATURES.md).

**Updated:** 2026-06-29 | **Version:** v3.3.0 (unreleased: Server-Timing, checkpoint replay)

## Current State

- **Version:** v3.3.0 (all 3 publishable modules at `/v3`)
- **Coverage:** 94.1% root, 79.5% usermgmt
- **Lint:** 0 issues (all modules)
- **ErrorFamily:** 0 violations (no stdlib error constructors)
- **Tests:** 574 usermgmt + 124 root + 35 adminui + integration, race-safe
- **Dependencies:** go-cqrs-lite v3.4.0 (most modules), go-error-family v0.5.1, go-branded-id v0.3.1, justinas/nosurf, go-webauthn v0.17.4, pquerna/otp, coreos/go-oidc, golang.org/x/oauth2
- **Architecture:** Fully event-sourced usermgmt (12 events, 20 commands, Decider pattern, WebAuthn passwordless, OAuth2/OIDC, multi-tenancy, bot accounts, membership RBAC, impersonation)
- **Modules:** 8 Go modules in go.work (root, usermgmt, adminui, integration_test, 4 examples)

---

## Shipped (v1.0.0 → v3.1.0)

Major milestones delivered. Maintained here for historical context.

| Version | Key Deliverables                                                                                                                                                                                                          |
| ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| v1.0.0  | Initial release: App builder, command/query dispatch, handler options, HTMX middleware, error handling, Casbin authorization, branded UserID, session management                                                          |
| v1.5.0  | CSRF (nosurf), rate limiting (token-bucket + heap eviction), security headers, recovery middleware, request logging (text + JSON + slog), lifecycle hooks                                                                 |
| v2.0.0  | go-cqrs-lite v2 migration, 42-file import path bump, pre-release fixes (nil-enforcer bypass, query nil panic, Login error classification, UpdateRoles ordering)                                                           |
| v2.1.0  | SSE + WebSocket real-time: Broadcaster fan-out, SSE reconnection (SSEEventStore + ReplayEvents), CQRS dispatch bridges (SSE + WS), typed WS message parser, WS OOB HTML, WS encoder, embedded HTMX v2.0.9 JS              |
| v2.2.0  | SQL Event Store (Postgres/SQLite/MySQL), SQL Session Store, pagination (DecodePagination + RenderPaginatedJSON[T]), typed dispatch adoption                                                                               |
| v2.3.0  | go-cqrs-lite v2.3.0: TypedHandler, deadline propagation, empty type validation, per-module go-cqrs-lite tags                                                                                                              |
| v2.4.0  | TOTP MFA (pquerna/otp), email verification, user import/export (JSON/CSV), audit log, per-endpoint rate limiting, account lockout                                                                                         |
| v2.5.0  | Event signing + encryption opt-in seams (ADR-0011), OAuth2/OIDC integration (ADR-0014), event schema versioning + upcasters (ADR-0013), catalog sub-package (ADR-0008)                                                    |
| v2.6.0  | Identity model redesign (ADR-0015): Tenant, Bot, Membership, Impersonation, ActorID. go-cqrs-lite v2.6.0. SQL event store delegation to upstream. Roles→memberships migration.                                            |
| v3.0.0  | go-cqrs-lite v3.0.0 migration (ADR-0016): manual projection replay, watermill EventBus, storage/memory split, Decider.Fold→Apply. Module path bump /v2→/v3. God object split (es_decide.go → 5 files). Dead code removal. |
| v3.1.0  | go-cqrs-lite v3.1.0: SQL-backed persistent read models (4 aggregates), one-call SQLite/Postgres stack presets, `OptimizeSQLiteDB`, graceful shutdown (`Service.Close`/`GracefulClose`), CI coverage gate, 697 tests.      |
| v3.2.0  | catalog/ module merged upstream into go-cqrs-lite/catalog/v3 (v3.2.0). ADR number collision fix, migration checklist (AGENTS.md #22-26), doc-honesty sweep, 51 stale status reports archived.                             |

---

## v3.2.0 — Stabilization & Documentation (Current)

_Focus: Close the gap between what shipped and what's documented. Improve test coverage for identity model._

| Area | Item                                                             | Priority | Status |
| ---- | ---------------------------------------------------------------- | -------- | ------ |
| Docs | Update FEATURES.md                                               | Critical | Done   |
| Docs | Update ROADMAP.md                                                | Critical | Done   |
| Docs | Add VERSIONING.md documenting semver policy                      | Medium   | Done   |
| Docs | Consumer migration guide (v2→v3: import paths, bus, projections) | High     | Done   |
| Docs | Add godoc examples for App, Handler, Service entry points        | Medium   | Done   |
| Test | Service-level impersonation tests through full dispatch          | High     | Done   |
| Test | Service-level membership tests through full dispatch             | High     | Done   |
| Test | Projection replay integration test (journal vs live dedup)       | High     | Done   |
| Test | Property-based tests for foldTenant, foldBot, foldMembership     | Medium   | Done   |
| Test | Fuzz tests for projection dedup + identity model deciders        | Medium   | Done   |
| Lint | Enable revive:exported linter + fix violations                   | Medium   | Done   |
| Code | Remove deprecated ClientIP() wrapper                             | Low      | Open   |
| Code | Verify and wire BrandNamer for root module marker types          | Medium   | Open   |

---

## v3.3.0 — Observability & Metrics

_Focus: Production-grade observability for CQRS dispatch pipelines._

| Area          | Item                                                                          | Priority | Status  |
| ------------- | ----------------------------------------------------------------------------- | -------- | ------- |
| Observability | Server-Timing API (W3C header, debug-gated, nil-receiver)                     | High     | Done    |
| Observability | OTel seam: document `go-cqrs-lite/otel/v3` + `middleware/v3` wiring guide     | Medium   | Planned |
| Observability | Prometheus seam: document `go-cqrs-lite/prometheus/v3` `/metrics` integration | Low      | Planned |
| CI            | Coverage gate in CI (fail on regression below threshold)                      | Medium   | Done    |

> **Note:** OpenTelemetry and Prometheus are already available via go-cqrs-lite upstream
> (`otel/v3`, `middleware/v3`, `prometheus/v3`). cqrs-htmx doesn't need to re-implement
> them — it needs a wiring guide showing consumers how to bolt them onto the App.

---

## v3.4.0 — Upstream Adoption & Scale

_Focus: Adopting go-cqrs-lite v3.4.0 capabilities to reduce hand-rolled code._

| Area | Item                                                               | Priority | Status  |
| ---- | ------------------------------------------------------------------ | -------- | ------- |
| ES   | Adopt `projectionhost/v3` — replace hand-rolled `StartProjections` | High     | Planned |
| ES   | Adopt `CatchUpSubscriber` — ordered durable projections            | Medium   | Planned |
| Test | Adopt `scenario/v3` BDD DSL for usermgmt decider tests             | Medium   | Planned |
| Perf | Adopt `snapshot/v3` for aggregates with 100+ events                | Medium   | Planned |
| Perf | Profile and optimize hot paths (dispatch, decode)                  | Low      | Planned |
| Perf | Benchmark projection replay with large stores (10K+ events)        | Low      | Planned |

---

## v4.0.0 — Advanced Event Sourcing

_Focus: Leveraging go-cqrs-lite v3's advanced capabilities._

| Area   | Item                                              | Priority | Status                                                     |
| ------ | ------------------------------------------------- | -------- | ---------------------------------------------------------- |
| ES     | stack.Materialize for persistent read models      | Low      | Done (ADR-0022: generic adapter, per-read-model decisions) |
| ES     | Adopt `kv.Cache` decorator for read-model caching | Low      | Planned                                                    |
| Schema | Adopt `schema/v3` upcasters for event evolution   | Medium   | Planned                                                    |

---

## Not Planned

These are explicitly out of scope for this library:

- **WebSocket upgrade logic** — Consumers should use dedicated libraries (gorilla/websocket, coder/websocket, etc.). The library provides protocol helpers (`WSMessage`, `WSOOBHTML`) only.
- **ORM integration** — Store interfaces are intentionally simple; consumers provide their own implementations.
- **Template engine support beyond templ** — The `TemplComponent` duck-typing pattern covers any `Render(ctx, w) error` interface.
- **Built-in HTTP router** — Framework-agnostic: works with `net/http`, Gin, Chi, etc. — no router dependency.
