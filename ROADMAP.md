# Roadmap — cqrs-htmx

> Long-term direction and raw ideas not yet refined into actionable tasks.
> For short-term work, see [TODO_LIST.md](TODO_LIST.md).
> For what exists today, see [FEATURES.md](FEATURES.md).
> For completed work, see [CHANGELOG.md](CHANGELOG.md).

**Updated:** 2026-09-09 | **Version:** v4.9.0 family train (13 tags, 2026-09-01) + setup/v4.10.0 (2026-09-07) + go-cqrs-lite 2026-09-08-train bump (2026-09-09); 122 advisory train-lag entries toward upstream's newest (templ-components v1.16.0 leading) — next-train alignment tracked in TODO_LIST | **Lint:** 0 issues across 15 modules (2026-09-07 M9; v2.12.2 re-verified 2026-09-01) | **Coverage gates (all green; 2026-09-07 spot-check):** root 93.2% (gate 90%), setup 86.3% (gate 80%), datastar 97.4% (gate 90%) — full table `nix run .#coverage-gate` | **`*Service` methods:** 73 (leading v5 indicator)

## Current State

- **Version:** v4.9.0 family train (13 tags, 2026-09-01) + `setup/v4.10.0` (2026-09-07) + go-cqrs-lite 2026-09-08-train bump (2026-09-09). 27 modules: root, identity-model, usermgmt + 3 auth sub-modules, adminui, loginpage, dashboardui, datastar, setup, systemadapter, health, auditlog, integration_test, 11 examples, e2e/server. Since v4.7.0: security middleware consolidation (`RecommendedSecurityMiddleware`), httputil v0.11.0→v0.12.0, the Broadcaster hub-first refactor (`Hub()` canonical; `Raw()` deprecated), the `setup/v4` SDK (one-call bundle + shared `/sse` with replay/backfill/heartbeat + DataStar dual-frontend ADR-0050 + stable `RunWithAppkit`), the `systemadapter/v4` bridge, and the `health/v4` + `auditlog/v4` optional bridge modules. Async projection startup (ADR-0048) + ActorID consolidation (ADR-0111) + tombstone migration (ADR-0114) shipped. Internal dep drift: ZERO (`check-version-drift --strict` green 2026-09-09); ~122 advisory lag entries toward upstream's newest are the next-train sweep (TODO_LIST P1).
- **Coverage:** all 15 gates green (last full table 2026-08-29; spot-verified 2026-09-07: root 93.2%/90, setup 86.3%/80, datastar 97.4%/90). Full per-module table: `nix run .#coverage-gate` (thresholds: root 90, usermgmt 74, identity-model 70, auth 80, adminui 66, loginpage 79, dashboardui 60, datastar 90, setup 80, systemadapter 70, health 90, auditlog 90). Race-safe.
- **Lint:** All 15 lint-checked modules at 0 issues (2026-09-07 M9; golangci v2.12.2 re-verified 2026-09-01). Achieved via the Aggregate→Stream API migration, the identity-model/usermgmt deprecated-alias migration in adminui + integration_test, dead-code removal, and targeted `.golangci.yml` exclusions for intentional patterns (builder-pattern partial init, re-export wrappers, generated `_templ.go`). Recompute uncapped: `GOEXPERIMENT=jsonv2 golangci-lint run --max-issues-per-linter 0 --max-same-issues 0 ./...` per module. Examples are covered by the pre-commit hook's golangci steps (2 example SA1019 batches fixed 2026-09-07/09).
- **ErrorFamily:** 0 violations across all modules.
- **Dependencies (2026-09-09):** go-cqrs-lite per-module trains (root: command/v4 v4.10.0, event/v4 v4.11.0, id/v4 v4.6.0, query/v4 v4.8.0, storage/memory v4.5.1), go-error-family, go-branded-id, go-sse v0.6.0, httputil v0.12.0, templ-components family v1.14.0, go-datastar v0.5.0, go-appkit v0.4.0 (setup, published). Casbin v3 is a first-class dependency of identity-model (ADR-0044). Auth deps (go-webauthn, oauth2, oidc, pquerna/otp) live in optional sub-modules — core usermgmt has ZERO auth deps.
- **Architecture:** identity-model is the domain source of truth (pure types, fold functions, Authz engine, constants). usermgmt re-exports via type aliases (deprecated; see v5 re-export retirement). Fully event-sourced (21 events, 20 commands, Decider pattern, WebAuthn passwordless, OAuth2/OIDC, TOTP optional, multi-tenancy, bot accounts, membership RBAC, impersonation, checkpoint-based projection replay via projectionhost, optional SQL hydration + async startup). **Transport is SSE-only** since the WebSocket drop (ADR 0046). dashboardui provides CQRS/ES observability with a pure-data `core/` sub-package (Phase 1 of a templ migration). Auth strategies extracted behind interfaces (ADR-0035). Harmful duplication at zero (sweeps 2026-07-26 + 2026-08-05).

---

## Open Questions

_Unresolved decisions that block a release or a direction. These route here (not TODO_LIST) until a decision is made._

1. ~~**Version bump for the WebSocket removal.**~~ **Resolved:** v4.7.0 was released on 2026-08-07 with the WS removal as a minor bump (semver violation flagged — the deletion of 14 exported symbols is technically breaking under SemVer). The code is shipped; if strict SemVer compliance is needed, a v5.0.0 tag can be cut retroactively.
2. **SSE re-export alias deletion timing.** `sse_event.go`/`sse_store.go` re-export go-sse symbols with `// Deprecated:` markers. Delete them in v5, or keep as zero-cost transparent aliases indefinitely? They are type aliases (no runtime cost), but they give consumers two import paths.
3. ~~**Publish the `datastar/v4` tag now?**~~ **Resolved:** `datastar/v4` is published (current: v4.9.0 in the 2026-09-01 family train). ~~Local `replace` directives in `examples/datastar-demo/go.mod` and `integration_test/go.mod` reference higher versions — remain until matching tags are published.~~ Those replaces were stripped 2026-08-15/2026-08-30; every family require now resolves from the proxy.
4. ~~**httputil `ContentTypeNosniff` vs `ContentTypeOptions`.**~~ **Resolved:** httputil v0.11.0 is published with both fields.
5. **Event payload ActorID format decision.** Membership event payloads (`MemberAddedPayload`, `MemberRolesChangedPayload`, `MemberRemovedPayload`) still store `ActorKind string` + `ActorID string` as separate fields. The domain model now has a single consolidated `id.ActorID` type. Should payloads change to a single `ActorID string` (PrefixedString, self-describing) with an upcaster for old events, or is the current 2-field format acceptable? Deferred across 3 sessions. This is a domain modeling + migration decision.
6. **`AuditEntry` and `Session` UserID field redundancy.** Now that `ActorID` carries the full kind-discriminated identity, `UserID` is a convenience field that duplicates information (for ActorUser) or is always zero (for non-user actors). Should these fields be dropped in v5, or kept for query convenience? Dropping them means query patterns that filter by UserID need to extract it from ActorID.
7. **Non-user actor roles strategy.** Currently bots, systems, and services are deny-by-default (nil roles) in the authz engine. If bots need specific permissions (e.g., a CI bot that can deploy), the engine needs a strategy — either bot-specific Casbin roles or kind-based defaults. What is the right security posture?
8. **`AsyncStartup` default in v5.** The feedback recommends async startup as the production default, but changing the zero-value from sync→async would be a behavioral breaking change. Should v5 flip the default, or keep sync as the default with async as documented best practice?
9. **`ProjectionReadinessCheck` backoff semantics.** Currently returns 503 for projections in `backoff` state. Should "ready" mean "fully caught up" (503 during backoff) or "healthy and retrying" (200 during backoff)? This is a judgment call about operator expectations.
10. **Release-train cadence (opened 2026-09-09).** The workspace is internally uniform at the 2026-09-08 go-cqrs-lite trains (0 unpublished) with ~122 advisory lag entries toward upstream's newest (templ-components v1.16.0 leading). Execute the next alignment sweep immediately, cut a family train from the current verified state first, or bundle both into one later train? Owner call — see docs/status/2026-09-09 §g1.
11. **V007 migration start (opened 2026-09-09).** The 39 v5-removed-API findings are v4-functional; cluster (2) (`stack.Bundle` → `system.New`) rewrites usermgmt's composition core and overlaps the P1 systemadapter-tag work. Start a dedicated migration branch now, or gate until go-cqrs-lite announces the v5 timeline / metaengine layout-planning APIs stabilize? See docs/status/2026-09-09 §g2.
12. **systemadapter first-tag version number.** When the upstream `metaengine/projectionadapter/v4 v4.5.0` blocker clears: tag systemadapter at the then-current family version (v4.10.0+) or at an independent v4.8.0 (its recommended first-version on record, docs/status/archived/2026-08-30_21-00 g2)? Related decision: build or reject the `setup.NewFromSystem()` bridge (asked 2026-08-09 ×2).
13. **Integration recipes for foreign frameworks (deferred consumer asks, 2026-07-05 feedback).** "Using cqrs-htmx with Huma" recipe and a Huma adapter were deferred pending consumer validation — no consumer has asked since. Parked here; revisit on the first real request.

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

The `datastar/v4` module shipped in [Unreleased] with 54 tests, 97.4% coverage (gate 90%), 0 lint issues. It is feature-complete for its initial scope (script serving, signal decoding, response builder, broadcaster with replay + heartbeat, EventBridge with OnError). These items are future scope, each requiring a separate decision.

**Routing note (2026-09-07):** the dual-frontend strategy is decided — ADR-0050 (`docs/adr/0050-dual-frontend-protocol-strategy.md`) accepted route-split coexistence, and `setup.Config.DataStarPath` now mounts the DataStar feed on the shared hub (Tiers 1-3 of the rollout plan shipped). The variant rows below are Tier 4 of that plan (`docs/planning/2026-09-07_17-18_datastar-dual-frontend-rollout.html`) and are GATED on consumer demand evidence per ADR-0050 — demand evidence so far: 6 of 15 surveyed consumer projects hand-roll go-datastar + cqrs-htmx bridges (integration friction is real; panel-variant demand is unproven).

| Item                         | What                                                                                                          | Effort | Notes                                                                                                                                                                                                           |
| ---------------------------- | ------------------------------------------------------------------------------------------------------------- | ------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Publish `datastar/v4` tag    | Tag the module so consumers can `go get` without `replace` directives                                         | 5min   | **Done.** `datastar/v4.0.0` and `datastar/v4.1.0` published. Demo/integration_test local replaces reference higher versions (`v4.6.1`/`v4.7.0`) — strip when matching tags are published.                       |
| dashboardui Datastar variant | Replace HTMX polling with Datastar signal patches for real-time projection health                             | ~8hr   | Templ components stay the same; only transport changes from polling to SSE signal patches. **Routed:** plan Tier 4 (M11 spike → M12 module), demand-gated per ADR-0050.                                         |
| adminui Datastar variant     | Optional morph-based rendering mode alongside HTMX                                                            | ~6hr   | Same pattern as dashboardui variant. **Routed:** plan Tier 4 (M15 design doc first), demand-gated per ADR-0050.                                                                                                 |
| loginpage Datastar forms     | Signal-based form state/validation (client-side)                                                              | ~3hr   | Replace server-side form validation roundtrips with client-side signal validation. **Routed:** plan Tier 4 (M13), demand-gated per ADR-0050.                                                                    |
| Offline sync evaluation      | Compare Datastar built-in retry primitives vs `sync-worker.js`                                                | ~4hr   | Datastar's CQRS model (long-lived SSE + short writes with auto-retry) may simplify the sync layer. **Routed:** plan Tier 4 (M14 evaluation report), demand-gated per ADR-0050.                                  |
| setup DataStar option        | One-flag mount of the DataStar feed + SDK script on the shared hub                                            | ~1.75h | **Done 2026-09-07 (ADR-0050).** `setup.Config.DataStarPath`/`DataStarScriptPath` + `Bundle.DataStarBroadcaster`; session-gated 401 like `/sse`; see `docs/guides/datastar-integration.md` ("Using with setup"). |
| Broadcaster SSE compression  | Re-export SDK compression options (`WithGzip`, `WithBrotli`, etc.)                                            | ~4hr   | Blocked: `writeEventID` bypasses SDK write path (documented limitation). Requires SDK `SetEventID` method or refactor.                                                                                          |
| Cross-transport hub sharing  | `Raw()` accessor + `NewBroadcasterFromRaw` for sharing one fan-out hub across HTMX and Datastar               | ~3hr   | **Done.** Both `cqrshtmx.Broadcaster` and `datastar.Broadcaster` expose `Raw() *sse.Broadcaster[sse.Event]`. `RawBroadcaster` interface for duck-typed sharing. See `docs/guides/sse-and-datastar.md`.          |
| Broadcaster options pattern  | Refactor constructors to `NewBroadcaster(opts ...BroadcasterOption)` with `WithReplay(n)`, `WithHeartbeat(d)` | ~2hr   | Currently 3 separate constructors; no way to combine replay + heartbeat.                                                                                                                                        |

**Open question:** ~~Should the `datastar/v4` tag be published now~~ — **Resolved.** `datastar/v4.0.0` and `v4.1.0` are published. Consumers outside the workspace can `go get` the module.

---

## v5 Vision: usermgmt Decomposition (Deferred)

The usermgmt module is a god-package: 4 aggregates (User, Membership, Tenant, Bot) plus shared infrastructure in one Go module. ADR-0019 (Blocked) and ADR-0038 (Proposed, deferred to v5) acknowledge this. The current v4 module works correctly and the split has zero consumer benefit while everything shares one `go.mod`.

### Decomposition Trigger (When to Split)

The split becomes worthwhile when:

1. **A consumer needs only User/Membership without Tenant/Bot** — currently they get the full dep tree.
2. **Dep-tree analysis shows >30% of usermgmt dependencies are pulled in for aggregates the consumer doesn't use.** Current dep-tree: go-cqrs-lite (event/command/decider/projection/projectionhost), casbin, go-error-family, go-branded-id. These are all shared infrastructure — the split would only help if aggregate-specific deps diverge.
3. **Compile times become a bottleneck** — the current module compiles in ~3s. No urgency.

**Trigger status (2026-08-05 architecture review):** 0 of 3 met. Independently confirmed: zero cross-aggregate co-change, 0% dep divergence, ~3s compile. The v5 deferral is correct. The **OAuth2 sub-service extraction prototype is DONE** (`service_oauth2_extracted.go`): 8 OAuth2 methods moved into a focused `oauth2Service` that `*Service` holds, validating the ADR-0038 composition pattern within v4. `*Service` now has **73 methods** (+20 since ADR-0038 on 2026-07-19) — track this as the leading v5 indicator (trigger at 80). The prototype establishes the shared dispatcher/error-classifier plumbing for the remaining 5 domain extractions (User, Membership, Tenant, Bot, Auth), all deferred to v5.

### Re-export Layer Retirement (v5)

26 usermgmt files re-export identity-model types via type aliases and constructor wrappers. All 160 exported re-export symbols now carry `// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.` markers (added 2026-08-05). Removal is a breaking change bundled with the v5 major bump. Decision confirmed by maintainer 2026-08-05: remove in v5. **SA1019 suppression removal is a v5 blocker:** adminui and integration_test `.golangci.yml` contain a scoped text-based staticcheck exclusion (`'Import github\.com/larsartmann/cqrs-htmx/identity-model/v4 directly'`) that suppresses 155 deprecation warnings. This MUST be removed once both modules migrate to direct identity-model imports — until then it hides all identity-model deprecation warnings in those modules.

### httputil Re-export Retirement (v5)

3 root-module files (`csrf_reexport.go`, `ratelimit_reexport.go`, `server_timing_reexport.go`) re-export 39 symbols from `github.com/larsartmann/httputil`. All now carry `// Deprecated:` markers (added 2026-08-05). Internal callers, examples, and docs migrated to direct `httputil.*` imports. Removal is bundled with the v5 major bump. The SecurityHeaders split-brain is **resolved**: httputil v0.11.0 is published with `PermissionsPolicy`, `Custom`, `ContentTypeOptions`, `SecurityHeaderSkip`, `RecommendedHSTS`/`RecommendedCSP`, and cqrs-htmx's `security.go` is now a deprecated alias + delegating wrapper over `httputil.SecurityHeadersConfig`. httputil v0.11.0 is published; all module `go.mod` files reference it and the `go.work` replace has been removed. Hermetic build (`nix run .#build`) and test (`nix run .#test`) pass. See `docs/guides/leveraging-httputil.md` for the migration table.

### WebSocket Transport Removal (DONE — pending version tag)

The entire WebSocket surface (`ws.go`, `ws_broadcaster.go`, `ws_dispatch.go`, `ws_encoder.go` + 5 test files + `extensions/ws.min.js`) was deleted and released in v4.7.0 (2026-08-07) without a `Deprecated:` phase — the WS code path was small enough, structurally isolated, and zero consumers were using it in production (ADR 0046). The library is now SSE-only. ADR-0004 and ADR-0010 are superseded. Migration recipe in `docs/migrations/v4-to-v5.md` (created) and `docs/adr/0046-drop-websocket-sse-only.md`. Note: the WS removal is technically a breaking change under SemVer (14 exported symbols deleted) but was released as a minor bump (v4.7.0, not v5.0.0) — see open question #1 above (resolved).

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

## Composition & Integration Layer (Researched — Proposed)

Architecture review (`docs/architecture-understanding/2026-08-09_05-36_module-integration-composability.html`) found the library's weakest dimension is **Composability (3/5)**: individual modules are superbly decoupled, but there is no wiring layer to help consumers compose them. The full-stack prebundle question (should we offer a model that includes samber-do-auditlog + go-health-dashboard?) was answered **yes, but as optional integration modules** — never as hard dependencies.

### Proposed New Modules

| Module                  | Purpose                                                                                                     | External Deps                                    | Effort |
| ----------------------- | ----------------------------------------------------------------------------------------------------------- | ------------------------------------------------ | ------ |
| `cqrs-htmx/setup/v4`    | Internal wiring kit: `DashboardConfig(svc)` adapter, `MountAll(mux, opts)` helper, `FullstackConfig` struct | Zero (internal siblings only)                    | ~6hr   |
| `cqrs-htmx/health/v4`   | go-health + go-health-dashboard integration: projection health → health checks, pre-configured dashboard    | go-health, go-health-dashboard, templ-components | ~6hr   |
| `cqrs-htmx/auditlog/v4` | samber-do-auditlog integration: `WithAuditLog(opts)` hook providers, HTML report viewer mount               | samber-do-auditlog, samber/do                    | ~3hr   |

### Why Separate Modules (Not Hard Deps)

The library principle ("never enforce defaults consumers might disagree with") prohibits bundling external deps into existing modules. go-health-dashboard pulls templ, templ-components, go-datastar, go-sse, go-health — if bundled into root, every Path A consumer (root only, no auth) would get these transitively. Separate optional modules = three independent opt-in points, mirroring how `usermgmt/totp`, `usermgmt/webauthn`, and `usermgmt/oauth2` already work.

### Phased Delivery

1. **Phase 1 (v4.x):** `examples/fullstack-demo/` + `setup/v4` module. Zero external deps. Eliminates the composition gap for the majority of consumers. ~80% of the value.
2. **Phase 2 (v4.x/v5):** `health/v4` + `auditlog/v4`. Independent modules, ship when ready. High-value for production users.
3. **Phase 3:** `docs/guides/fullstack-wiring.md` integration guide + README/SKILL.md cross-links. Discoverability layer.

**Status:** Phase 1 (`setup/v4`) shipped — module built, tested (8 tests), integrated into all workspace gates. The `systemadapter/v4` module (bridging go-cqrs-lite `system/` + `metaengine/`) also shipped as a work-in-progress module (3 tests, builds, but needs lint remediation — see TODO_LIST). Phase 2 (`health/v4` + `auditlog/v4`) not yet started. Phase 3 (`docs/guides/fullstack-wiring.md`) shipped. Actionable items in TODO_LIST (P2).

---

## Operational Tooling Ideas

_Candidates for future development if consumer demand emerges. Items that became actionable have graduated to TODO_LIST._

| Idea                                     | What                                                                                                                                                                                                                                                                                                                                   | Effort   |
| ---------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| CQRS admin CLI (`cqrs-admin`)            | `cqrs-admin events list`, `projections reset`, `dlq replay`, `aggregates list` — a command-line tool for operational CQRS/ES tasks without a running dashboard                                                                                                                                                                         | Medium   |
| MetricsRecorder through projectionhost   | Wire `projectionhost.WithMetrics` so projection lag, DLQ depth, and restart counts flow through a metrics pipeline. `OnProjectionFailed` callback is wired; metrics are not. Requires deciding whether metrics are library-wired or consumer-provided.                                                                                 | ~100 LOC |
| SQL-backed defaults for checkpoint + DLQ | When a `*sql.DB` is available, default to SQL-backed `CheckpointStore` and `DeadLetterStore` instead of in-memory (which loses state on restart). Requires an architectural decision on library-vs-consumer boundary for projectionhost defaults.                                                                                      | Medium   |
| Read-model hydrator (Option B)           | `ReadModelHydrator` interface: hydrate in-memory read models from SQL on restart BEFORE projections resume from checkpoint. Eliminates full journal replay without the empty-maps problem. Pairs checkpoint + persistent state. Source: consumer feedback (`docs/feedback/processed/2026-08-12_projection-drain-startup-downtime.md`). | ~150 LOC |
| Projection snapshots (Option C)          | Periodically snapshot materialized read-model state (the in-memory maps). On restart, load snapshot, replay only events since snapshot. Standard event-sourcing pattern — between async startup (no persistence) and full SQL hydration. Source: same feedback.                                                                        | ~200 LOC |
| SQLite `CheckpointStore` (Option D)      | Provide a ready-to-use `CheckpointStore` implementation backed by SQLite. Lowers the barrier to incremental replay. Alone doesn't solve empty-maps problem; pairs with Option B or C. Source: same feedback.                                                                                                                                        | ~80 LOC  |
| setup surface ideas                      | Raw ideas from the 2026-08-14 setup overhaul review, none decided: `RunListener(net.Listener)`/`RunConfig`/TLS passthrough, `/livez` vs `/readyz` split, booleans to mount `ProjectionStatusHandler`/`EventCatalogHandler`, `setup-demo` `seed()` returning an error instead of `log.Fatalf`, godoc `Example` functions for `Run`/`RunHandler`, richer seed data + AsyncStartup showcase. Each needs a demand signal before graduating to TODO_LIST. | Small each |
| Websites for health/auditlog modules     | Both bridge modules are tagged (v4.9.0) and the website-launch pattern exists; neither has a project website. Candidate when the family next gets attention.                                                                                                                                                                                       | Medium   |

### Residual micro-ideas (harvested 2026-09-09 docs-health sweep)

_Small hardening/doc ideas found while annotating the 2026-08-09 → 2026-09-07 session reports. None is worth a TODO_LIST slot alone; pick them up when touching the nearby code._

- Script/gate hardening: tag-message guard rejecting pasted SSH signatures in `scripts/verify-tag.sh`; `check-modules` making `check-require-tags` containment explicit; `nix flake check` WITH builds (only `--no-build` ever recorded); release-train honoring retract directives when resolving "latest published"; buildflow golangci steps inheriting `/tmp` GOCACHE + repo `.golangci.yml`; codespell ignore list ("deriver", ...); buildflow `go-mod-normalize` run (direct/indirect mixing); `meta.mainProgram` in flake packages; markdown-lint MD024 exclusion (CHANGELOG duplicate headings); LICENSE-presence check in check-modules; per-module full-lint pre-commit step; coverage `flock`; delete `backup/pre-blob-purge` + `refs/original/*` once confident.
- Docs: `SessionMiddleware` loud-failure on missing store + `Config.SelfCheck` cookie round-trip (would have caught the cookie split-brain); cross-link `async-projection-startup.md` from `fullstack-wiring.md`; `WithCheckpointStore`/`WithDeadLetterStore` section missing from `leveraging-system-metaengine.md`; `WithOpenAPI` merge-into-Spec consumer guide; go-structure-linter 55 root-package findings triage (intentional vs v5); "bump playbook" runbook (`docs/runbooks/dependency-train-bump.md`) + reusable `bump-dep.sh` helper + a documented `go work sync` policy (none run since 2026-08-30); committed `cqrs-upgrade -dry-run` artifact per train; eventtest v0.x churn note in the bump playbook.
- Examples/tests: smoke tests for `examples/basic` + `examples/datastar-demo` (no `*_test.go`); admin-panel SSE e2e spec; catalog-demo visual smoke; samber-do SSE e2e.
- systemadapter: `Volume` hint provenance/tuning + a `Volume > 0` regression test; system-demo Volume-informed engine-plan display; `WithLatencyBudget`/`WithLayoutPriority` evaluation; declarative fold cleanup for `ExternalAccountLink` is tracked in TODO_LIST (bug).

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
- ~~**`broadcaster.ServeSSE()` high-level helper** — Crosses the "building blocks, not a server" design line. Consumers compose `Broadcaster` + `SSEStream` themselves; a one-call server helper would impose opinionated routing/response semantics this library deliberately avoids.~~ **SUPERSEDED 2026-08-17:** `Broadcaster.ServeSSE` DID ship (`sse_broadcaster.go:153`) as a scoped convenience (connected event + heartbeat + unsubscribe on one hub), and the shared domain-event lifecycle lives in `transport.ServeDomainEvents`. The rejection's reasoning stands only for a full-server helper.
- **Re-export go-cqrs-lite middleware factories** — Evaluated 2026-07-30. Re-exporting `middleware/v4` factories (Retry, CircuitBreaker, Recovery, Logging, etc.) from the cqrs-htmx root would pull ~29 new dependencies into every consumer's build: the full OTel SDK (`go.opentelemetry.io/otel/sdk`, `/sdk/metric`, `/trace`), `failsafe-go` (circuit breaker), and `modernc.org/sqlite` (dead-letter SQL store, which alone drags 10+ transitive deps). The root currently has zero OTel dependencies by design (library principle). Since the dispatcher type is shared by identity (`*command.Dispatcher`), consumers already wire middleware with a single import (`middleware.CommandRetry(...)`) — no re-export needed. **Decision: do NOT re-export.** Documentation (`docs/guides/leveraging-go-cqrs-lite.md` §1) + runnable examples (`examples/middleware-demo/`, `examples/observability-demo/`) are the correct discoverability mechanism.
- **usermgmt god-package split** (domain layer extraction, SQL infrastructure extraction, Service struct split, cross-module Service-layer integration test) — Sub-package extraction within the same Go module provides zero consumer benefit: same `go.mod` = same dep tree. Clean seams are identified (20 pure fold/decide files with zero I/O, 9 SQL infrastructure files) but only separate Go modules would reduce transitive deps, and that is not justified by current consumer demand. Re-open when a consumer specifically requests a reduced dep tree.
- **`TypedRepository` / `TypedDecider` adoption across usermgmt** — Premise invalid: (1) zero command type assertions exist — `command.RegisterTyped[Cmd]` already gives fully-typed handlers (see `es_dispatch.go`); (2) `TypedDecider` binds ONE command type per repository, incompatible with usermgmt's multi-command aggregates (User has Register/ChangeEmail/AddRole/Suspend/...); (3) the current `repo.Execute(ctx, aggID, aggType, decideFn)` + per-command closure pattern is the correct, already-type-safe design for multi-command aggregates.
- **Integration test importing the published version (not local replace)** — Blocked, not rejected: the `go.work` local replaces exist precisely because published go-cqrs-lite tags carry broken zero pseudo-versions. An integration test against the published version would fail until upstream cuts a clean consolidated release (v4.0.3+ or v4.1.0). Re-open once the publishing bug is resolved.
- **Standardize import grouping** — Cosmetic defer. gofmt + goimports already enforce a consistent style; further normalization has no functional impact.
- **Automate GitHub Release creation via CI on tag push** — Manual `gh release create` is sufficient for the current release cadence; automating adds CI complexity without near-term payoff.
- **cspell/vitest/jest in the devShell** — Stale ask (2026-08-05): spell-checking is covered by `codespell` (already in the devShell and BuildFlow), and BuildFlow explicitly skips `jest-test`/`vitest-test` because the repo has no JS tests (Go library; e2e uses Playwright). Adding unused Node tooling would bloat the shell for zero coverage. Revisit only if real JS tests appear.
- **`SyncWorkerURL(path)` Go helper** — Rejected across three sync sessions (each time re-discovered and re-rejected): consumers already control the worker path via the `data-sync-worker-url` HTML attribute and `SyncWorkerHandlerWith(js, version)`. A Go-side URL builder would add API surface for a concern that belongs in markup/template, not server code.
- **Revert dedup-round-4 closure-wrapper chain** (`withTimeout` → `withTimeoutCtx` → `authContext` → `withAuthContext`) — Evaluated 2026-07-29. The brutal self-review flagged the 4-layer depth as adding indirection. However, the chain is correct, tested with `-race`, and eliminates harmful clone groups (the dedup was the primary goal). Each layer adds a distinct concern (timeout, auth preflight, rate-limit check). Collapsing them would re-introduce the clone groups. Decision: keep the chain as-is; document the pattern.
- **Durable expiry via `go-cqrs-lite/scheduling`** — Evaluated 2026-07-30 via design doc (`docs/design/durable-scheduling.md`). Session TTL, email-verification-token TTL, and account-lockout duration are currently handled by in-process sweepers (`EvictStale()`, `EvictExpired()`) that are not durable across restarts. **Conclusion: NOT needed.** Every expiry mechanism already has a lazy check (correctness preserved regardless of restart). The SQL store provides multi-instance safety for the longest-TTL item (sessions, 24h) because `EvictExpired` is shared + idempotent. Short-lived tokens (5-10 min) are not worth the complexity of durable timers. In-memory deployments lose all data on restart anyway, so durable timers add nothing. Re-evaluate only if cross-instance lockout coordination or immediate (non-lazy) session revocation is needed.
- **Configurable lockout eviction interval** — Evaluated 2026-07-31. The 5-minute hard-coded interval balances CPU usage vs. memory growth. Lockout entries are tiny (email + timestamp). Making it configurable adds API surface for negligible benefit. **Decision: keep hard-coded.**
- **UserDelete cascade error aggregation** — Evaluated 2026-07-31. Cascade errors (session revocation, membership removal, bot deletion) are logged but not returned. The user IS already deleted when cascades run; returning errors would be misleading. **Decision: keep best-effort (log, don't return).**
- **Configurable state cache capacity** — Evaluated 2026-07-31. Currently unbounded (`NewStateCache(0)`). For <100k users, memory is negligible. A bounded LRU adds complexity for a premature optimization. **Decision: keep unbounded. Re-open if memory pressure is reported.**
- **MySQLDialect real UPSERT** — Evaluated 2026-07-31. Current `ON DUPLICATE KEY UPDATE col = col` (no-op) suffices for checkpoint stores. Event store uses append-only inserts with version constraints, not UPSERT. **Decision: keep no-op. Re-open if an idempotency store needs MySQL UPSERT.**
- **Cascade cleanup shared helper (DeleteTenant + DeleteUser)** — Evaluated 2026-07-31. Cascades are structurally similar but semantically different (different read models, different cleanup commands). Extracting a generic helper would lose type safety. Duplication is minimal (3-4 lines per cascade). **Decision: don't extract. Re-open if 3+ cascades share the exact same pattern.**
