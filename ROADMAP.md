# Roadmap — cqrs-htmx

> Long-term direction and raw ideas not yet refined into actionable tasks.
> For short-term work, see [TODO_LIST.md](TODO_LIST.md).
> For what exists today, see [FEATURES.md](FEATURES.md).

**Updated:** 2026-07-02 | **Version:** v4.0.0 (shipped: auth strategy extraction, modular auth sub-modules)

## Current State

- **Version:** v4.0.0 (11 modules: root + usermgmt + 3 auth sub-modules + adminui + integration_test + 4 examples)
- **Coverage:** 94.3% root, 80.1% usermgmt, 87.5% webauthn, 92.3% oauth2, 88.2% totp
- **Lint:** 0 issues (root + usermgmt + adminui)
- **ErrorFamily:** 0 violations (root + usermgmt + adminui; sub-modules intentionally exempt)
- **Tests:** 1,055+ total (root 198, usermgmt 764, totp 3, webauthn 20, oauth2 18, adminui 35, integration 17), race-safe
- **Dependencies:** go-cqrs-lite v3.5.0, go-error-family v0.5.1, go-branded-id v0.3.1. Auth deps (go-webauthn, oauth2, oidc, pquerna/otp) are now in optional sub-modules — core usermgmt has ZERO auth deps
- **Architecture:** Fully event-sourced usermgmt (12 events, 20 commands, Decider pattern, WebAuthn passwordless, OAuth2/OIDC, multi-tenancy, bot accounts, membership RBAC, impersonation, checkpoint-based projection replay). Auth strategies extracted behind interfaces (ADR-0035).
- **Modules:** 11 Go modules in go.work (root, usermgmt, usermgmt/totp, usermgmt/webauthn, usermgmt/oauth2, adminui, integration_test, 4 examples)

---

## Shipped (v1.0.0 → v3.3.0)

Major milestones delivered. Maintained here for historical context.

| Version | Key Deliverables                                                                                                                                                                                                                                           |
| ------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| v1.0.0  | Initial release: App builder, command/query dispatch, handler options, HTMX middleware, error handling, Casbin authorization, branded UserID, session management                                                                                           |
| v1.5.0  | CSRF (nosurf), rate limiting (token-bucket + heap eviction), security headers, recovery middleware, request logging (text + JSON + slog), lifecycle hooks                                                                                                  |
| v2.0.0  | go-cqrs-lite v2 migration, 42-file import path bump, pre-release fixes (nil-enforcer bypass, query nil panic, Login error classification, UpdateRoles ordering)                                                                                            |
| v2.1.0  | SSE + WebSocket real-time: Broadcaster fan-out, SSE reconnection (SSEEventStore + ReplayEvents), CQRS dispatch bridges (SSE + WS), typed WS message parser, WS OOB HTML, WS encoder, embedded HTMX v2.0.9 JS                                               |
| v2.2.0  | SQL Event Store (Postgres/SQLite/MySQL), SQL Session Store, pagination (DecodePagination + RenderPaginatedJSON[T]), typed dispatch adoption                                                                                                                |
| v2.3.0  | go-cqrs-lite v2.3.0: TypedHandler, deadline propagation, empty type validation, per-module go-cqrs-lite tags                                                                                                                                               |
| v2.4.0  | TOTP MFA (pquerna/otp), email verification, user import/export (JSON/CSV), audit log, per-endpoint rate limiting, account lockout                                                                                                                          |
| v2.5.0  | Event signing + encryption opt-in seams (ADR-0011), OAuth2/OIDC integration (ADR-0014), event schema versioning + upcasters (ADR-0013), catalog sub-package (ADR-0008)                                                                                     |
| v2.6.0  | Identity model redesign (ADR-0015): Tenant, Bot, Membership, Impersonation, ActorID. go-cqrs-lite v2.6.0. SQL event store delegation to upstream. Roles→memberships migration.                                                                             |
| v3.0.0  | go-cqrs-lite v3.0.0 migration (ADR-0016): manual projection replay, watermill EventBus, storage/memory split, Decider.Fold→Apply. Module path bump /v2→/v3. God object split (es_decide.go → 5 files). Dead code removal.                                  |
| v3.1.0  | go-cqrs-lite v3.1.0: SQL-backed persistent read models (4 aggregates), one-call SQLite/Postgres stack presets, `OptimizeSQLiteDB`, graceful shutdown (`Service.Close`/`GracefulClose`), CI coverage gate, 697 tests.                                       |
| v3.2.0  | catalog/ module merged upstream into go-cqrs-lite/catalog/v3 (v3.2.0). ADR number collision fix, migration checklist (AGENTS.md #22-26), doc-honesty sweep, 51 stale status reports archived.                                                              |
| v3.3.0  | go-cqrs-lite v3.4.0 upgrade. BasicCommand embedding (ADR-0032 — structurally eliminates zero-cmdID bug). Checkpoint-based projection replay (ADR-0031 Accepted). Server-Timing API (W3C). Offline command queue Phase 2a (ADR-0029). 8 modules in go.work. |

---

## v3.2.0 — Stabilization & Documentation (Completed)

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
| Code | Verify and wire BrandNamer for root module marker types          | Medium   | Done   |

---

## v3.3.0 — Observability & Metrics

_Focus: Production-grade observability for CQRS dispatch pipelines._

| Area          | Item                                                                          | Priority | Status                                |
| ------------- | ----------------------------------------------------------------------------- | -------- | ------------------------------------- |
| Observability | Server-Timing API (W3C header, debug-gated, nil-receiver)                     | High     | Done                                  |
| Observability | OTel seam: document `go-cqrs-lite/otel/v3` + `middleware/v3` wiring guide     | Medium   | Done (`docs/observability-wiring.md`) |
| Observability | Prometheus seam: document `go-cqrs-lite/prometheus/v3` `/metrics` integration | Low      | Done (`docs/observability-wiring.md`) |
| CI            | Coverage gate in CI (fail on regression below threshold)                      | Medium   | Done                                  |

> **Note:** OpenTelemetry and Prometheus are already available via go-cqrs-lite upstream
> (`otel/v3`, `middleware/v3`, `prometheus/v3`). cqrs-htmx doesn't need to re-implement
> them — it needs a wiring guide showing consumers how to bolt them onto the App.

---

## v3.4.0 — Upstream Adoption & Scale

_Focus: Adopting go-cqrs-lite v3.4.0 capabilities to reduce hand-rolled code._

| Area | Item                                                               | Priority | Status                                                          |
| ---- | ------------------------------------------------------------------ | -------- | --------------------------------------------------------------- |
| ES   | Adopt `projectionhost/v3` — replace hand-rolled `StartProjections` | High     | Planned (checkpoint replay shipped in v3.3.0 as interim fix)    |
| ES   | Adopt `CatchUpSubscriber` — ordered durable projections            | Medium   | Planned (ADR-0031 Accepted; deferred — needs sync-wait wrapper) |
| Test | Adopt `scenario/v3` BDD DSL for usermgmt decider tests             | Medium   | Done (all 4 aggregates: User + Tenant + Bot + Membership)       |
| Perf | Adopt `snapshot/v3` for aggregates with 100+ events                | Medium   | Planned                                                         |
| Perf | Profile and optimize hot paths (dispatch, decode)                  | Low      | Planned                                                         |
| Perf | Benchmark projection replay with large stores (10K+ events)        | Low      | Planned                                                         |

---

## v4.0.0 — Auth Strategy Extraction (Shipped)

_Focus: Module isolation for auth strategies. Consumers import only what they need._

| Area       | Item                                                                                          | Priority | Status |
| ---------- | --------------------------------------------------------------------------------------------- | -------- | ------ |
| Extraction | TOTP behind TOTPProvider interface → `usermgmt/totp/v4` module                               | Critical | Done   |
| Extraction | WebAuthn behind WebAuthnProvider interface → `usermgmt/webauthn/v4` module                    | Critical | Done   |
| Extraction | OAuth2/OIDC behind OAuth2Provider interface → `usermgmt/oauth2/v4` module                     | Critical | Done   |
| Testing    | W3C spec ceremony tests for WebAuthn provider                                                 | High     | Done   |
| Testing    | Real JWT signing tests for OAuth2/OIDC provider                                               | High     | Done   |
| Testing    | Compile-time interface assertions in integration_test                                         | High     | Done   |
| Docs       | ADR-0035 (auth strategy extraction decision)                                                  | Medium   | Done   |
| Docs       | Migration guide v3→v4                                                                         | High     | Done   |

## v4.1.0 — God-Package Split (Next Initiative)

_Focus: The 84-file usermgmt god-package. Clean seams identified but extraction deferred._

| Area          | Item                                                                | Priority | Status   |
| ------------- | ------------------------------------------------------------------- | -------- | -------- |
| Architecture  | Extract domain layer (20 pure fold/decide files, zero I/O)          | High     | Planned  |
| Architecture  | Extract SQL infrastructure (9 files)                                | Medium   | Planned  |
| Architecture  | Split Service struct into focused services                         | Medium   | Planned  |
| Testing       | Cross-module integration test through Service layer                | Medium   | Planned  |
| Feature       | Configurable WebAuthn session TTL                                   | Low      | Planned  |
| Testing       | Fuzz tests on JSON serialization boundary                           | Medium   | Planned  |

---

## Not Planned

These are explicitly out of scope for this library:

- **WebSocket upgrade logic** — Consumers should use dedicated libraries (gorilla/websocket, coder/websocket, etc.). The library provides protocol helpers (`WSMessage`, `WSOOBHTML`) only.
- **ORM integration** — Store interfaces are intentionally simple; consumers provide their own implementations.
- **Template engine support beyond templ** — The `TemplComponent` duck-typing pattern covers any `Render(ctx, w) error` interface.
- **Built-in HTTP router** — Framework-agnostic: works with `net/http`, Gin, Chi, etc. — no router dependency.
