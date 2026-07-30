# Roadmap — cqrs-htmx

> Long-term direction and raw ideas not yet refined into actionable tasks.
> For short-term work, see [TODO_LIST.md](TODO_LIST.md).
> For what exists today, see [FEATURES.md](FEATURES.md).
> For completed work, see [CHANGELOG.md](CHANGELOG.md).

**Updated:** 2026-07-29 | **Version:** v4.6.1 (go-cqrs-lite v4.2.0; see AGENTS.md for per-sub-module versions) | **Lint:** 0 issues across all 15 modules | **Coverage gates:** root 93.7% (gate 90%), usermgmt 80.9% (gate 74%), identity-model 74.9% (gate 70%), dashboardui 72.5% (gate 60%)

## Current State

- **Version:** v4.6.1 (15 modules: root + identity-model + usermgmt + 3 auth sub-modules + adminui + loginpage + dashboardui + integration_test + 5 examples). v4.6.0 released 2026-07-26; v4.6.1 released 2026-07-27 (dependency bumps, identity-model metadata, slices.Contains refactor). All inter-module version refs resolved to clean tags (`e274540` + subsequent releases).
- **Coverage:** 93.7% root, 80.9% usermgmt, 74.9% identity-model (gate 70%), 88.2% totp, 89.2% webauthn, 88.3% oauth2, 69.0% adminui, 80.1% loginpage, 72.5% dashboardui (gate 60%). Race-safe. CI gates: root 90%, usermgmt 74%, identity-model 70%, auth 80%, adminui 66%, loginpage 79%, dashboardui 60% (see `nix run .#coverage-gate`). All 9 modules have coverage gates.
- **Lint:** All 15 modules lint-clean (0 issues each). Achieved via the SA1019 deprecation migration (`id.AggregateID` → `id.StreamID` across all modules), dead-code removal (`renderStatCardsTempl`, `notImplemented`, `eventRow`), and targeted `.golangci.yml` exclusions for intentional patterns (builder-pattern partial init, re-export wrappers, generated `_templ.go`). Some exclusions are flagged as lazy shortcuts for future audit (see TODO_LIST "Audit `.golangci.yml` exclusions"). Recompute uncapped: `GOEXPERIMENT=jsonv2 golangci-lint run --max-issues-per-linter 0 --max-same-issues 0 ./...` per module.
- **ErrorFamily:** 0 violations across all modules.
- **Dependencies:** go-cqrs-lite v4.2.0 (storage/memory and snapshot at v4.1.0), go-error-family v0.10.0, go-branded-id v0.5.0, go-sse v0.3.0, httputil v0.6.1, templ-components v1.2.0. Casbin v3 is a first-class dependency of identity-model (ADR-0044). Auth deps (go-webauthn, oauth2, oidc, pquerna/otp) are in optional sub-modules — core usermgmt has ZERO auth deps.
- **Architecture:** identity-model is the domain source of truth (pure types, fold functions, Authz engine, constants). usermgmt re-exports via type aliases. Fully event-sourced (22 events, 19 commands, Decider pattern, WebAuthn passwordless, OAuth2/OIDC, multi-tenancy, bot accounts, membership RBAC, impersonation, checkpoint-based projection replay via projectionhost). dashboardui provides CQRS/ES observability (event browser, projection health, time-travel inspector, SSE live updates). Auth strategies extracted behind interfaces (ADR-0035). Harmful code duplication driven to zero across two dedup sweeps (2026-07-26).
- **Modules:** 15 Go modules in `go.work` (root, identity-model, usermgmt, usermgmt/totp, usermgmt/webauthn, usermgmt/oauth2, adminui, loginpage, dashboardui, integration_test, examples/basic, examples/catalog-demo, examples/datastar-demo, examples/admin-demo, examples/dashboard-demo).

---

## Upstream Adoption & Scale

_Focus: Adopting go-cqrs-lite capabilities to reduce hand-rolled code._

| Area | Item                                                               | Priority | Status                                                                                                                                                                                                                                                                                                                                                            |
| ---- | ------------------------------------------------------------------ | -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ES   | Adopt `projectionhost/v4` — replace hand-rolled `StartProjections` | High     | **Done** (projectionhost adopted; DLQ + per-projection checkpoints + crash-restart)                                                                                                                                                                                                                                                                               |
| ES   | Adopt `CatchUpSubscriber` — ordered durable projections            | Medium   | **Not Needed** (projectionhost `WithSubscriber` provides same replay→live handoff; CatchUpSubscriber would add message-model adapter overhead — see ADR-0031 Superseded)                                                                                                                                                                                          |
| Bus  | `event.Bus.UnsubscribeAll` / context-cancellable `SubscribeAll`    | Low      | **Open** — `dashboardui.Dashboard.Close()` cannot fully unsubscribe its bus handler (no removal API). **Mitigated in v4.6.0** (`15c27c3`): `Close()` signals a done-channel that makes the handler a no-op before closing the broadcaster. The handler remains registered on the bus but is inert. Full removal still requires the upstream `UnsubscribeAll` API. |
| Perf | Profile and optimize hot paths (dispatch, decode)                  | Low      | **Done** (benchmarks: dispatch ~1µs, decode ~1-2µs, error mapping ~1-5µs — within bounds)                                                                                                                                                                                                                                                                         |
| Perf | Benchmark projection replay with large stores (10K+ events)        | Low      | **Done** (benchmark: 10K events = 30ms, ~3µs/event, linear scaling — see `es_projection_replay_bench_test.go`)                                                                                                                                                                                                                                                    |
| Perf | `decider.WithStateCache` for usermgmt aggregates                    | Medium   | **Evaluated 2026-07-30.** High-value, zero-risk: eliminates full event replay on every `Execute` (O(total events) → O(new events)). Auto-invalidating (decider manages cache after writes). NOT currently wired (`buildDeciderRepositories` runs full-replay mode by default). Recommendation: add `WithStateCache[UserState](NewStateCache(256))` to repository construction. No consumer-visible API change. |
| Perf | `kv.Cache[UserView, UserID]` for SQL-backed read model             | Low      | **Evaluated 2026-07-30.** Wrap `SQLUserReadModel`'s view store in `kv.NewCache` to avoid SQL round-trips on every `FindByIDSql`. Write-through invalidation already handled by projection's `syncToSQL`. Do NOT cache `FindByEmail` (mutable secondary index — invalidation requires dual-key invalidation on `EmailChanged`). |
| ES   | `deriver` for event→command cascades                               | Low      | **Evaluated 2026-07-30 — Not a fit.** usermgmt's cascades are (a) projections (Casbin policy cleanup, read-model updates) or (b) in-process best-effort calls (session revocation, tenant membership cleanup). The tenant cleanup violates deriver's purity contract (must read mutable read model). Session revocation is a direct store call, not a command. Re-evaluate only if cross-service async derivations with idempotent redelivery are needed. |

---

## Data Mesh Interchange (Researched — Not Yet Adopted)

Research and a proposal (`docs/research/2026-07-25_*`, `docs/proposals/2026-07-25_data-mesh-interchange.md`) concluded that cqrs-htmx should **not** build a data-mesh interchange from scratch — go-cqrs-lite `catalog/v4` already provides the documentation layer (DataProduct / Channel / Message / exporters / docserver). The proposal's recommendation is **Approach C + D**: evaluate consolidating the hand-rolled `EventCatalog`/`openapi/` with `catalog/v4`, plus build the three genuinely missing runtime pieces:

| Gap                             | What                                                                         | Effort   |
| ------------------------------- | ---------------------------------------------------------------------------- | -------- |
| 1. Channel-to-runtime binding   | Connect `catalog.Channel` (docs) to `event.Bus`/`StreamingJournal` (runtime) | ~50 LOC  |
| 2. CloudEvents envelope         | Standardized event envelope for cross-system interchange                     | ~30 LOC  |
| 3. Pull-based machine transport | `GET /events?after=<id>` → JSON/NDJSON stream                                | ~100 LOC |

**Status:** under consideration. No code written. The strategic angle (per the landscape research): event sourcing _structurally prevents_ the data-discovery problems DataHub/OpenMetadata/ODDS exist to solve, so time-travel + the catalog are a stronger positioning than a bespoke mesh product. Not yet committed to a release.

---

## v5 Vision: usermgmt Decomposition (Deferred)

The usermgmt module is a god-package: 4 aggregates (User, Membership, Tenant, Bot) plus shared infrastructure in one Go module. ADR-0019 (Blocked) and ADR-0038 (Proposed, deferred to v5) acknowledge this. The current v4 module works correctly and the split has zero consumer benefit while everything shares one `go.mod`.

### Decomposition Trigger (When to Split)

The split becomes worthwhile when:

1. **A consumer needs only User/Membership without Tenant/Bot** — currently they get the full dep tree.
2. **Dep-tree analysis shows >30% of usermgmt dependencies are pulled in for aggregates the consumer doesn't use.** Current dep-tree: go-cqrs-lite (event/command/decider/projection/projectionhost), casbin, go-error-family, go-branded-id. These are all shared infrastructure — the split would only help if aggregate-specific deps diverge.
3. **Compile times become a bottleneck** — the current module compiles in ~3s. No urgency.

### Proposed Module Boundaries (v5)

```
usermgmt/v5                  ← core: Service, shared infra, session/authz
usermgmt/user/v5             ← User aggregate + UserReadModel + UserDecider
usermgmt/membership/v5       ← Membership aggregate + MembershipReadModel
usermgmt/tenant/v5           ← Tenant aggregate + TenantReadModel
usermgmt/bot/v5              ← Bot aggregate + BotReadModel
usermgmt/webauthn/v5         ← (unchanged: auth strategy sub-module)
usermgmt/oauth2/v5           ← (unchanged: auth strategy sub-module)
usermgmt/totp/v5             ← (unchanged: auth strategy sub-module)
```

### What the Split Enables

- Consumers who only need User auth get a smaller dep tree (no casbin policy for tenants/bots).
- Independent versioning per aggregate (Tenant schema can evolve without a User release).
- Clearer bounded context boundaries (DDD alignment).

### What the Split Costs

- Cross-module event references (Tenant references UserID from user module).
- More `go.mod` files to maintain.
- The `Service` struct must compose sub-services or use a facade pattern.
- Breaking change for all consumers (new import paths).

### Current Assessment

**Not justified for v4.** No consumer has requested a reduced dep tree. The god-package is well-organized internally (clean seams between aggregates, separate files per concern). Re-open when a real consumer need emerges.

---

## Operational Tooling Ideas (From Dashboard Design Research)

_These emerged from the CQRS dashboard design brainstorm (`docs/brainstorming/2026-07-23_cqrs-dashboard-design.html`) and have not been refined into actionable tasks. They are candidates for future development if consumer demand emerges._

| Idea                          | What                                                                                                                                                                                         | Effort  |
| ----------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------- |
| Composite readiness checker   | `cqrshtmx.ReadinessHandler()` — combines `HealthHandler` + projection lag + DLQ depth into a single `/readyz` endpoint for load-balancer probes that should fail when projections are behind | ~50 LOC |
| CQRS admin CLI (`cqrs-admin`) | `cqrs-admin events list`, `projections reset`, `dlq replay`, `aggregates list` — a command-line tool for operational CQRS/ES tasks without a running dashboard                               | Medium  |
| JSON debug endpoint           | `GET /debug/cqrs` returning structured debug info (registered commands/queries, projection states, event counts) from `Bundle.DebugStructured()`                                             | ~30 LOC |

---

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
- **Re-export go-cqrs-lite middleware factories** — Evaluated 2026-07-30. Re-exporting `middleware/v4` factories (Retry, CircuitBreaker, Recovery, Logging, etc.) from the cqrs-htmx root would pull ~29 new dependencies into every consumer's build: the full OTel SDK (`go.opentelemetry.io/otel/sdk`, `/sdk/metric`, `/trace`), `failsafe-go` (circuit breaker), and `modernc.org/sqlite` (dead-letter SQL store, which alone drags 10+ transitive deps). The root currently has zero OTel dependencies by design (library principle). Since the dispatcher type is shared by identity (`*command.Dispatcher`), consumers already wire middleware with a single import (`middleware.CommandRetry(...)`) — no re-export needed. **Decision: do NOT re-export.** Documentation (`docs/guides/leveraging-go-cqrs-lite.md` §1) + runnable examples (`examples/middleware-demo/`, `examples/observability-demo/`) are the correct discoverability mechanism.
- **usermgmt god-package split** (domain layer extraction, SQL infrastructure extraction, Service struct split, cross-module Service-layer integration test) — Sub-package extraction within the same Go module provides zero consumer benefit: same `go.mod` = same dep tree. Clean seams are identified (20 pure fold/decide files with zero I/O, 9 SQL infrastructure files) but only separate Go modules would reduce transitive deps, and that is not justified by current consumer demand. Re-open when a consumer specifically requests a reduced dep tree.
- **`TypedRepository` / `TypedDecider` adoption across usermgmt** — Premise invalid: (1) zero command type assertions exist — `command.RegisterTyped[Cmd]` already gives fully-typed handlers (see `es_dispatch.go`); (2) `TypedDecider` binds ONE command type per repository, incompatible with usermgmt's multi-command aggregates (User has Register/ChangeEmail/AddRole/Suspend/...); (3) the current `repo.Execute(ctx, aggID, aggType, decideFn)` + per-command closure pattern is the correct, already-type-safe design for multi-command aggregates.
- **Integration test importing the published version (not local replace)** — Blocked, not rejected: the `go.work` local replaces exist precisely because published go-cqrs-lite tags carry broken zero pseudo-versions. An integration test against the published version would fail until upstream cuts a clean consolidated release (v4.0.3+ or v4.1.0). Re-open once the publishing bug is resolved.
- **Standardize import grouping** — Cosmetic defer. gofmt + goimports already enforce a consistent style; further normalization has no functional impact.
- **Automate GitHub Release creation via CI on tag push** — Manual `gh release create` is sufficient for the current release cadence; automating adds CI complexity without near-term payoff.
- **`SyncWorkerURL(path)` Go helper** — Rejected across three sync sessions (each time re-discovered and re-rejected): consumers already control the worker path via the `data-sync-worker-url` HTML attribute and `SyncWorkerHandlerWith(js, version)`. A Go-side URL builder would add API surface for a concern that belongs in markup/template, not server code.
