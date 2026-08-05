# Roadmap — cqrs-htmx

> Long-term direction and raw ideas not yet refined into actionable tasks.
> For short-term work, see [TODO_LIST.md](TODO_LIST.md).
> For what exists today, see [FEATURES.md](FEATURES.md).
> For completed work, see [CHANGELOG.md](CHANGELOG.md).

**Updated:** 2026-08-05 | **Version:** v4.6.1 + `[Unreleased]` (WebSocket dropped — SSE-only; ADR 0046) | **Lint:** 0 issues across all lint-checked modules | **Coverage gates:** root ~93% (gate 90%), usermgmt 81.6% (gate 74%), identity-model 74.9% (gate 70%), dashboardui 84.0% (gate 60%), datastar 96.7% (gate 90%) | **`*Service` methods:** 72 (leading v5 indicator)

## Current State

- **Version:** v4.6.1 (released 2026-07-27) + a large `[Unreleased]` (19 modules: root + identity-model + usermgmt + 3 auth sub-modules + adminui + loginpage + dashboardui + datastar + integration_test + 7 examples + e2e/server). The `[Unreleased]` work includes the WebSocket transport drop (14 exported symbols deleted — breaking; ADR 0046), the httputil adoption to 100% (39 re-export symbols deprecated), the identity-model re-export deprecation (23 files), the OAuth2 sub-service extraction prototype, and the dashboardui `core/` pure-data layer. All inter-module version refs resolved to clean tags.
- **Coverage:** 93.3% root, 81.6% usermgmt, 74.9% identity-model (gate 70%), 88.2% totp, 89.2% webauthn, 88.3% oauth2, 68.7% adminui, 79.9% loginpage, 84.0% dashboardui (gate 60%), 96.7% datastar (gate 90%). Race-safe. CI gates: root 90%, usermgmt 74%, identity-model 70%, auth 80%, adminui 66%, loginpage 79%, dashboardui 60%, datastar 90% (see `nix run .#coverage-gate`). All 10 modules have coverage gates.
- **Lint:** All 19 modules lint-clean (0 issues each). Achieved via the SA1019 deprecation migration (`id.AggregateID` → `id.StreamID` across all modules), dead-code removal (`renderStatCardsTempl`, `notImplemented`, `eventRow`), and targeted `.golangci.yml` exclusions for intentional patterns (builder-pattern partial init, re-export wrappers, generated `_templ.go`). The exclusion audit confirmed zero masked bugs. Recompute uncapped: `GOEXPERIMENT=jsonv2 golangci-lint run --max-issues-per-linter 0 --max-same-issues 0 ./...` per module.
- **ErrorFamily:** 0 violations across all modules.
- **Dependencies:** go-cqrs-lite v4.2.0 (storage/memory and snapshot at v4.1.0), go-error-family v0.10.0, go-branded-id v0.5.0, go-sse v0.3.0, httputil v0.8.0 (v0.9.0 pending publish — see TODO P0), templ-components v1.7.0. Casbin v3 is a first-class dependency of identity-model (ADR-0044). Auth deps (go-webauthn, oauth2, oidc, pquerna/otp) are in optional sub-modules — core usermgmt has ZERO auth deps.
- **Architecture:** identity-model is the domain source of truth (pure types, fold functions, Authz engine, constants). usermgmt re-exports via type aliases (now deprecated; see v5 re-export retirement). Fully event-sourced (21 events, 20 commands, Decider pattern, WebAuthn passwordless, OAuth2/OIDC, multi-tenancy, bot accounts, membership RBAC, impersonation, checkpoint-based projection replay via projectionhost). **Transport is SSE-only** since the WebSocket drop (ADR 0046). dashboardui provides CQRS/ES observability (event browser, projection health, time-travel inspector, SSE live updates) and now has a pure-data `core/` sub-package (Phase 1 of a templ migration). Auth strategies extracted behind interfaces (ADR-0035). Harmful code duplication driven to zero across two dedup sweeps (2026-07-26) plus a 2026-08-05 `logAuth` pass.
- **Modules:** 19 Go modules in `go.work` (root, identity-model, usermgmt, usermgmt/totp, usermgmt/webauthn, usermgmt/oauth2, adminui, loginpage, dashboardui, datastar, integration_test, examples/basic, examples/catalog-demo, examples/datastar-demo, examples/admin-demo, examples/dashboard-demo, examples/middleware-demo, examples/observability-demo, e2e/server).

---

## Open Questions

_Unresolved decisions that block a release or a direction. These route here (not TODO_LIST) until a decision is made._

1. **Version bump for the WebSocket removal.** 14 exported symbols were deleted (ADR 0046), which is breaking under SemVer → v5.0.0. But the CHANGELOG `[Unreleased]` also accumulates many additive changes. Decision: cut v5.0.0 (SSE-only) now, or v4.7.0 with the WS removal framed as a major-bump preview? The code is already removed; only the tag/version is undecided.
2. **SSE re-export alias deletion timing.** `sse_event.go`/`sse_store.go` re-export go-sse symbols with `// Deprecated:` markers. Delete them in v5, or keep as zero-cost transparent aliases indefinitely? They are type aliases (no runtime cost), but they give consumers two import paths.
3. **Publish the `datastar/v4` tag now?** The module is tested, documented, and integration-tested, but has no published tag. Publish immediately, or wait for additional features (see Datastar Future Scope)?
4. **httputil `ContentTypeNosniff` vs `ContentTypeOptions`.** httputil grew `ContentTypeOptions string` alongside the older `ContentTypeNosniff bool`. Decide whether to deprecate/remove `ContentTypeNosniff` (would be a breaking change for httputil consumers) before tagging v0.9.0.

## Upstream Adoption & Scale

_Focus: Adopting go-cqrs-lite capabilities to reduce hand-rolled code._

| Area | Item                                                               | Priority | Status                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| ---- | ------------------------------------------------------------------ | -------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ES   | Adopt `projectionhost/v4` — replace hand-rolled `StartProjections` | High     | **Done** (projectionhost adopted; DLQ + per-projection checkpoints + crash-restart)                                                                                                                                                                                                                                                                                                                                                                       |
| ES   | Adopt `CatchUpSubscriber` — ordered durable projections            | Medium   | **Not Needed** (projectionhost `WithSubscriber` provides same replay→live handoff; CatchUpSubscriber would add message-model adapter overhead — see ADR-0031 Superseded)                                                                                                                                                                                                                                                                                  |
| Bus  | `event.Bus.UnsubscribeAll` / context-cancellable `SubscribeAll`    | Low      | **Open** — `dashboardui.Dashboard.Close()` cannot fully unsubscribe its bus handler (no removal API). **Mitigated in v4.6.0** (`15c27c3`): `Close()` signals a done-channel that makes the handler a no-op before closing the broadcaster. The handler remains registered on the bus but is inert. Full removal still requires the upstream `UnsubscribeAll` API.                                                                                         |
| Perf | Profile and optimize hot paths (dispatch, decode)                  | Low      | **Done** (benchmarks: dispatch ~1µs, decode ~1-2µs, error mapping ~1-5µs — within bounds)                                                                                                                                                                                                                                                                                                                                                                 |
| Perf | Benchmark projection replay with large stores (10K+ events)        | Low      | **Done** (benchmark: 10K events = 30ms, ~3µs/event, linear scaling — see `es_projection_replay_bench_test.go`)                                                                                                                                                                                                                                                                                                                                            |
| Perf | `decider.WithStateCache` for usermgmt aggregates                   | Medium   | **Done (2026-07-31).** Wired via `repositoryOptions[State]` in all 4 aggregate repositories (User/Membership/Tenant/Bot). Eliminates full event replay on every Execute (O(total events) → O(new events)). Auto-invalidating (decider manages cache after writes). Zero consumer-visible API change.                                                                                                                                                      |
| Perf | `kv.Cache[UserView, UserID]` for SQL-backed read model             | Low      | **Evaluated 2026-07-30.** Wrap `SQLUserReadModel`'s view store in `kv.NewCache` to avoid SQL round-trips on every `FindByIDSql`. Write-through invalidation already handled by projection's `syncToSQL`. Do NOT cache `FindByEmail` (mutable secondary index — invalidation requires dual-key invalidation on `EmailChanged`).                                                                                                                            |
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

**Related research:** [Iroh (n0-computer) P2P/QUIC networking fit analysis](docs/research/2026-08-02_iroh-p2p-networking-fit-analysis.md) — evaluated for broker-less multi-instance fanout (`iroh-gossip`), distributed snapshot store (`iroh-blobs`), and local-first read projections (`iroh-docs`). Conclusion: not a core-dependency fit (Rust-first, no official Go bindings for the protocol layer; write-path conflict with server-authoritative ES). One low-risk opt-in idea (gossip-backed `event.Bus`) kept as a raw idea pending official Go protocol bindings or a consumer request.

---

## Datastar Future Scope

The `datastar/v4` module shipped in [Unreleased] with 71 tests, 96.7% coverage (gate 90%), 0 lint issues. It is feature-complete for its initial scope (script serving, signal decoding, response builder, broadcaster with replay + heartbeat, EventBridge with OnError). These items are future scope, each requiring a separate decision:

| Item                         | What                                                                                                          | Effort | Notes                                                                                                                  |
| ---------------------------- | ------------------------------------------------------------------------------------------------------------- | ------ | ---------------------------------------------------------------------------------------------------------------------- |
| Publish `datastar/v4` tag    | Tag the module so consumers can `go get` without `replace` directives                                         | 5min   | Module is ready. Requires stripping local `replace` directives from demo/integration_test go.mod first.                |
| dashboardui Datastar variant | Replace HTMX polling with Datastar signal patches for real-time projection health                             | ~8hr   | Templ components stay the same; only transport changes from polling to SSE signal patches.                             |
| adminui Datastar variant     | Optional morph-based rendering mode alongside HTMX                                                            | ~6hr   | Same pattern as dashboardui variant.                                                                                   |
| loginpage Datastar forms     | Signal-based form state/validation (client-side)                                                              | ~3hr   | Replace server-side form validation roundtrips with client-side signal validation.                                     |
| Offline sync evaluation      | Compare Datastar built-in retry primitives vs `sync-worker.js`                                                | ~4hr   | Datastar's CQRS model (long-lived SSE + short writes with auto-retry) may simplify the sync layer.                     |
| Broadcaster SSE compression  | Re-export SDK compression options (`WithGzip`, `WithBrotli`, etc.)                                            | ~4hr   | Blocked: `writeEventID` bypasses SDK write path (documented limitation). Requires SDK `SetEventID` method or refactor. |
| Broadcaster options pattern  | Refactor constructors to `NewBroadcaster(opts ...BroadcasterOption)` with `WithReplay(n)`, `WithHeartbeat(d)` | ~2hr   | Currently 3 separate constructors; no way to combine replay + heartbeat.                                               |

**Open question:** Should the `datastar/v4` tag be published now (module is tested, documented, integration-tested) or wait for additional features? The module has no published tag — consumers outside the workspace cannot `go get` it.

---

## v5 Vision: usermgmt Decomposition (Deferred)

The usermgmt module is a god-package: 4 aggregates (User, Membership, Tenant, Bot) plus shared infrastructure in one Go module. ADR-0019 (Blocked) and ADR-0038 (Proposed, deferred to v5) acknowledge this. The current v4 module works correctly and the split has zero consumer benefit while everything shares one `go.mod`.

### Decomposition Trigger (When to Split)

The split becomes worthwhile when:

1. **A consumer needs only User/Membership without Tenant/Bot** — currently they get the full dep tree.
2. **Dep-tree analysis shows >30% of usermgmt dependencies are pulled in for aggregates the consumer doesn't use.** Current dep-tree: go-cqrs-lite (event/command/decider/projection/projectionhost), casbin, go-error-family, go-branded-id. These are all shared infrastructure — the split would only help if aggregate-specific deps diverge.
3. **Compile times become a bottleneck** — the current module compiles in ~3s. No urgency.

**Trigger status (2026-08-05 architecture review):** 0 of 3 met. Independently confirmed: zero cross-aggregate co-change, 0% dep divergence, ~3s compile. The v5 deferral is correct. The **OAuth2 sub-service extraction prototype is DONE** (`service_oauth2_extracted.go`): 8 OAuth2 methods moved into a focused `oauth2Service` that `*Service` holds, validating the ADR-0038 composition pattern within v4. `*Service` now has **72 methods** (+20 since ADR-0038 on 2026-07-19) — track this as the leading v5 indicator (trigger at 80). The prototype establishes the shared dispatcher/error-classifier plumbing for the remaining 5 domain extractions (User, Membership, Tenant, Bot, Auth), all deferred to v5.

### Re-export Layer Retirement (v5)

26 usermgmt files re-export identity-model types via type aliases and constructor wrappers. All 160 exported re-export symbols now carry `// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.` markers (added 2026-08-05). Removal is a breaking change bundled with the v5 major bump. Decision confirmed by maintainer 2026-08-05: remove in v5. **SA1019 suppression removal is a v5 blocker:** adminui and integration_test `.golangci.yml` contain a scoped text-based staticcheck exclusion (`'Import github\.com/larsartmann/cqrs-htmx/identity-model/v4 directly'`) that suppresses 155 deprecation warnings. This MUST be removed once both modules migrate to direct identity-model imports — until then it hides all identity-model deprecation warnings in those modules.

### httputil Re-export Retirement (v5)

3 root-module files (`csrf_reexport.go`, `ratelimit_reexport.go`, `server_timing_reexport.go`) re-export 39 symbols from `github.com/larsartmann/httputil`. All now carry `// Deprecated:` markers (added 2026-08-05). Internal callers, examples, and docs migrated to direct `httputil.*` imports. Removal is bundled with the v5 major bump. The SecurityHeaders split-brain is **resolved**: httputil gained the richer config fields (`PermissionsPolicy`, `Custom`, `ContentTypeOptions`, `SecurityHeaderSkip`, `RecommendedHSTS`/`RecommendedCSP`) in a pending v0.9.0, and `security.go` is now a deprecated alias + delegating wrapper over `httputil.SecurityHeadersConfig`. **Publish step required:** tag httputil v0.9.0, bump cqrs-htmx `go.mod`, remove the `go.work` replace. See `docs/guides/leveraging-httputil.md` for the migration table.

### WebSocket Transport Removal (DONE — pending version tag)

The entire WebSocket surface (`ws.go`, `ws_broadcaster.go`, `ws_dispatch.go`, `ws_encoder.go` + 5 test files + `extensions/ws.min.js`) was deleted in `[Unreleased]` without a `Deprecated:` phase — the WS code path was small enough, structurally isolated, and zero consumers were using it in production (ADR 0046). The library is now SSE-only. ADR-0004 and ADR-0010 are superseded. Migration recipe in `docs/adr/0046-drop-websocket-sse-only.md`; a dedicated `docs/migrations/v4-to-v5.md` is tracked in TODO_LIST. The version tag (v5.0.0 vs v4.7.0) is an open question (see above).

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

## Operational Tooling Ideas

_Candidates for future development if consumer demand emerges. Items that became actionable have graduated to TODO_LIST._

| Idea                                     | What                                                                                                                                                                                                                                                   | Effort   |
| ---------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------- |
| CQRS admin CLI (`cqrs-admin`)            | `cqrs-admin events list`, `projections reset`, `dlq replay`, `aggregates list` — a command-line tool for operational CQRS/ES tasks without a running dashboard                                                                                         | Medium   |
| MetricsRecorder through projectionhost   | Wire `projectionhost.WithMetrics` so projection lag, DLQ depth, and restart counts flow through a metrics pipeline. `OnProjectionFailed` callback is wired; metrics are not. Requires deciding whether metrics are library-wired or consumer-provided. | ~100 LOC |
| SQL-backed defaults for checkpoint + DLQ | When a `*sql.DB` is available, default to SQL-backed `CheckpointStore` and `DeadLetterStore` instead of in-memory (which loses state on restart). Requires an architectural decision on library-vs-consumer boundary for projectionhost defaults.      | Medium   |

---

## Not Planned

These are explicitly out of scope for this library:

- **WebSocket upgrade logic** — Dropped in v5 (ADR 0046). SSE covers the same use cases. Consumers needing bi-directional transport should integrate a dedicated WebSocket library directly.
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
- **Revert dedup-round-4 closure-wrapper chain** (`withTimeout` → `withTimeoutCtx` → `authContext` → `withAuthContext`) — Evaluated 2026-07-29. The brutal self-review flagged the 4-layer depth as adding indirection. However, the chain is correct, tested with `-race`, and eliminates harmful clone groups (the dedup was the primary goal). Each layer adds a distinct concern (timeout, auth preflight, rate-limit check). Collapsing them would re-introduce the clone groups. Decision: keep the chain as-is; document the pattern.
- **Durable expiry via `go-cqrs-lite/scheduling`** — Evaluated 2026-07-30 via design doc (`docs/design/durable-scheduling.md`). Session TTL, email-verification-token TTL, and account-lockout duration are currently handled by in-process sweepers (`EvictStale()`, `EvictExpired()`) that are not durable across restarts. **Conclusion: NOT needed.** Every expiry mechanism already has a lazy check (correctness preserved regardless of restart). The SQL store provides multi-instance safety for the longest-TTL item (sessions, 24h) because `EvictExpired` is shared + idempotent. Short-lived tokens (5-10 min) are not worth the complexity of durable timers. In-memory deployments lose all data on restart anyway, so durable timers add nothing. Re-evaluate only if cross-instance lockout coordination or immediate (non-lazy) session revocation is needed.
- **Configurable lockout eviction interval** — Evaluated 2026-07-31. The 5-minute hard-coded interval balances CPU usage vs. memory growth. Lockout entries are tiny (email + timestamp). Making it configurable adds API surface for negligible benefit. **Decision: keep hard-coded.**
- **UserDelete cascade error aggregation** — Evaluated 2026-07-31. Cascade errors (session revocation, membership removal, bot deletion) are logged but not returned. The user IS already deleted when cascades run; returning errors would be misleading. **Decision: keep best-effort (log, don't return).**
- **Configurable state cache capacity** — Evaluated 2026-07-31. Currently unbounded (`NewStateCache(0)`). For <100k users, memory is negligible. A bounded LRU adds complexity for a premature optimization. **Decision: keep unbounded. Re-open if memory pressure is reported.**
- **MySQLDialect real UPSERT** — Evaluated 2026-07-31. Current `ON DUPLICATE KEY UPDATE col = col` (no-op) suffices for checkpoint stores. Event store uses append-only inserts with version constraints, not UPSERT. **Decision: keep no-op. Re-open if an idempotency store needs MySQL UPSERT.**
- **Cascade cleanup shared helper (DeleteTenant + DeleteUser)** — Evaluated 2026-07-31. Cascades are structurally similar but semantically different (different read models, different cleanup commands). Extracting a generic helper would lose type safety. Duplication is minimal (3-4 lines per cascade). **Decision: don't extract. Re-open if 3+ cascades share the exact same pattern.**
