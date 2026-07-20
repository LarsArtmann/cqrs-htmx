# Roadmap — cqrs-htmx

> Long-term direction and raw ideas not yet refined into actionable tasks.
> For short-term work, see [TODO_LIST.md](TODO_LIST.md).
> For what exists today, see [FEATURES.md](FEATURES.md).

**Updated:** 2026-07-20 | **Version:** v4.3.0+unreleased (go-cqrs-lite v4.0.x; see AGENTS.md for per-sub-module versions)

## Current State

- **Version:** v4.3.0+unreleased (12 modules: root + usermgmt + 3 auth sub-modules + adminui + loginpage + integration_test + 4 examples)
- **Coverage:** 93.8% root, 80.2% usermgmt, 88.2% totp, 89.2% webauthn, 88.3% oauth2, 69.0% adminui, 80.1% loginpage (~920 tests), race-safe. CI gates: root 90%, usermgmt 74%, auth 80%, adminui 66%, loginpage 80% (see `nix run .#coverage-gate`)
- **Lint:** 0 issues across all linted modules
- **ErrorFamily:** 0 violations across all modules (sub-modules adopted go-error-family directly in v4.2.0)
- **Dependencies:** go-cqrs-lite v4.0.x (sub-modules v4.0.0–v4.0.2), go-error-family v0.7.0, go-branded-id v0.3.2, httputil v0.5.0, templ-components v0.16.0. Auth deps (go-webauthn, oauth2, oidc, pquerna/otp) are in optional sub-modules — core usermgmt has ZERO auth deps
- **Architecture:** Fully event-sourced usermgmt (12 events, 20 commands, Decider pattern, WebAuthn passwordless, OAuth2/OIDC, multi-tenancy, bot accounts, membership RBAC, impersonation, checkpoint-based projection replay). Auth strategies extracted behind interfaces (ADR-0035). loginpage module (passwordless login UI). adminui module (templ+HTMX dashboard).
- **Modules:** 12 Go modules in go.work (root, usermgmt, usermgmt/totp, usermgmt/webauthn, usermgmt/oauth2, adminui, loginpage, integration_test, 4 examples)

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
| Perf | Opt-in aggregate snapshotting                                      | Medium   | Done (shipped as `SnapshotConfig` — see CHANGELOG [Unreleased]) |
| Perf | Profile and optimize hot paths (dispatch, decode)                  | Low      | Planned                                                         |
| Perf | Benchmark projection replay with large stores (10K+ events)        | Low      | Planned                                                         |

---

## v4.1.0 — Embedded HTMX Extensions (Shipped)

_Focus: Zero-CDN-dependency HTMX setup with embedded extensions._

| Area | Item                                                               | Priority | Status |
| ---- | ------------------------------------------------------------------ | -------- | ------ |
| UX   | Embedded HTMX extensions (SSE, WS, idiomorph) via `go:embed`       | High     | Done   |
| UX   | `HTMXExtensionHandler(name)` + `HTMXExtensionsHandler(bundle)` API | High     | Done   |
| UX   | HTMX core bumped 2.0.9 → 2.0.10                                    | Medium   | Done   |

## v4.2.0 — Consumer Feedback + Error Quality (Shipped)

_Focus: APIs requested by 3 real consumers (Overview, DiscordSync, SwettySwipper). Error family adoption across all modules._

| Area  | Item                                                                             | Priority | Status |
| ----- | -------------------------------------------------------------------------------- | -------- | ------ |
| API   | `RequestGuard` custom auth guard                                                 | High     | Done   |
| API   | Request-aware decoders (`*WithRequest` variants)                                 | High     | Done   |
| API   | `DefaultRateLimiterConfig()` constructor                                         | Medium   | Done   |
| API   | `SecurityHeaderSkip` sentinel                                                    | Medium   | Done   |
| API   | `RenderHTML(html)` HandlerOption                                                 | Medium   | Done   |
| API   | `SSEEventConnected`/`SSEEventHeartbeat` constants                                | Low      | Done   |
| API   | `Broadcaster.Close()` + `fanOut.Close()` graceful shutdown                       | Medium   | Done   |
| API   | JSON error `"code"` field + `StructuredError.Code`                               | High     | Done   |
| API   | `CSRFTestToken(mw)` test helper (returns token + cookie)                         | Medium   | Done   |
| Error | go-error-family direct dep in ALL modules (32 violations → 0)                    | Critical | Done   |
| Error | Error context enrichment (`.WithContext()` chaining)                             | High     | Done   |
| Perf  | `dedup.Ring` (O(1) memory projection dedup) + `codec.ForEncoding` (CBOR support) | Medium   | Done   |

## v4.2.1 — Release Hygiene (Shipped)

_Focus: Version drift alignment, go.work fix, CHANGELOGs for all modules._

| Area  | Item                                                | Priority | Status |
| ----- | --------------------------------------------------- | -------- | ------ |
| Infra | go.work replace+use conflict fix                    | Critical | Done   |
| Infra | go-cqrs-lite version drift alignment (all → v3.7.4) | High     | Done   |
| Docs  | CHANGELOGs created for all 6 modules                | High     | Done   |

## Unreleased — go-cqrs-lite v4 Migration

_Focus: Migrate from go-cqrs-lite v3 to v4. Eliminate vendored eventtest._

| Area  | Item                                                                    | Priority | Status |
| ----- | ----------------------------------------------------------------------- | -------- | ------ |
| Deps  | go-cqrs-lite v3.7.4 → v4.0.0 (import paths `/v3` → `/v4`)               | Critical | Done   |
| Deps  | go-error-family v0.6.1 → v0.7.0                                         | High     | Done   |
| Deps  | templ-components v0.15.0 → v0.16.0                                      | Medium   | Done   |
| Infra | `.vendor-local/eventtest` eliminated                                    | High     | Done   |
| API   | `ErrorCode` exported, `ErrorRecorder` extracted from `StatusRecorder`   | Medium   | Done   |
| API   | `writeDispatchError` consolidates 15 error-writing sites in usermgmt    | Medium   | Done   |
| Refac | `UserReadModel.Handle` dispatch table (eliminates last lint issue)      | Medium   | Done   |
| Test  | CBOR round-trip tests, request-aware decoder tests, CSRFTestToken tests | High     | Done   |

## v4.0.0 — Auth Strategy Extraction (Shipped)

_Focus: Module isolation for auth strategies. Consumers import only what they need._

| Area       | Item                                                                       | Priority | Status |
| ---------- | -------------------------------------------------------------------------- | -------- | ------ |
| Extraction | TOTP behind TOTPProvider interface → `usermgmt/totp/v4` module             | Critical | Done   |
| Extraction | WebAuthn behind WebAuthnProvider interface → `usermgmt/webauthn/v4` module | Critical | Done   |
| Extraction | OAuth2/OIDC behind OAuth2Provider interface → `usermgmt/oauth2/v4` module  | Critical | Done   |
| Testing    | W3C spec ceremony tests for WebAuthn provider                              | High     | Done   |
| Testing    | Real JWT signing tests for OAuth2/OIDC provider                            | High     | Done   |
| Testing    | Compile-time interface assertions in integration_test                      | High     | Done   |
| Docs       | ADR-0035 (auth strategy extraction decision)                               | Medium   | Done   |
| Docs       | Migration guide v3→v4                                                      | High     | Done   |

## Not Planned

These are explicitly out of scope for this library:

- **WebSocket upgrade logic** — Consumers should use dedicated libraries (gorilla/websocket, coder/websocket, etc.). The library provides protocol helpers (`WSMessage`, `WSOOBHTML`) only.
- **ORM integration** — Store interfaces are intentionally simple; consumers provide their own implementations.
- **Template engine support beyond templ** — The `TemplComponent` duck-typing pattern covers any `Render(ctx, w) error` interface.
- **Built-in HTTP router** — Framework-agnostic: works with `net/http`, Gin, Chi, etc. — no router dependency.
- **TOTP management views in adminui** — This library is passwordless-first: WebAuthn passkeys + OAuth2 only. TOTP remains available as an optional sub-module (`usermgmt/totp/v4`) for consumers who genuinely want it, but the admin UI will not ship TOTP enable/disable/QR-code views. We are not building for the old-school TOTP world.
- **Redis adapters (SessionStore / OAuth2StateStore / IdempotencyStore)** — Multi-instance ephemeral-store adapters belong in go-cqrs-lite (or consumer code), not cqrs-htmx. Low consumer demand, Redis is overrated, and the existing in-memory + SQL stores cover the documented use cases. Re-open upstream if a real consumer needs it.
- **Consumer-facing v3→v4 codemod** — Automated migration tool. All known consumers are already on v4; the one-time migration is documented in `docs/migrations/v3-to-v4.md`. Building a codemod now would be cost without an audience.
- **Root module: extract SSE/WS/ratelimit into optional sub-packages** — 16 of 46 root files have zero logic coupling to the core, but they share the same go.mod = same dep tree = zero consumer benefit. Only a separate Go module would reduce transitive deps, and that is not justified by current demand.
- **Shared types module (`usermgmt/types/`)** — A cross-module types boundary would add a JSON serialization round-trip (~400ns–1.2µs per ceremony). The cost is negligible, the conceptual smell is real, but the extraction has no consumer benefit until dep-tree reduction is needed.
- **`broadcaster.ServeSSE()` high-level helper** — Crosses the "building blocks, not a server" design line. Consumers compose `Broadcaster` + `SSEStream` themselves; a one-call server helper would impose opinionated routing/response semantics this library deliberately avoids.
- **usermgmt god-package split** (domain layer extraction, SQL infrastructure extraction, Service struct split, cross-module Service-layer integration test) — Sub-package extraction within the same Go module provides zero consumer benefit: same `go.mod` = same dep tree. Clean seams are identified (20 pure fold/decide files with zero I/O, 9 SQL infrastructure files) but only separate Go modules would reduce transitive deps, and that is not justified by current consumer demand. Re-open when a consumer specifically requests a reduced dep tree.
- **`TypedRepository` / `TypedDecider` adoption across usermgmt** — Premise invalid: (1) zero command type assertions exist — `command.RegisterTyped[Cmd]` already gives fully-typed handlers (see `es_dispatch.go`); (2) `TypedDecider` binds ONE command type per repository, incompatible with usermgmt's multi-command aggregates (User has Register/ChangeEmail/AddRole/Suspend/...); (3) the current `repo.Execute(ctx, aggID, aggType, decideFn)` + per-command closure pattern is the correct, already-type-safe design for multi-command aggregates.
- **Integration test importing the published version (not local replace)** — Blocked, not rejected: the `go.work` local replaces exist precisely because published go-cqrs-lite tags carry broken zero pseudo-versions. An integration test against the published version would fail until upstream cuts a clean consolidated release (v4.0.3+ or v4.1.0). Re-open once the publishing bug is resolved.
- **Standardize import grouping** — Cosmetic defer. gofmt + goimports already enforce a consistent style; further normalization has no functional impact.
- **Automate GitHub Release creation via CI on tag push** — Manual `gh release create` is sufficient for the current release cadence; automating adds CI complexity without near-term payoff.
