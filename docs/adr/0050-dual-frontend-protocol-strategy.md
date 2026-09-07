# ADR-0050: Dual Frontend Protocol Strategy (HTMX + DataStar)

## Status

ACCEPTED — 2026-09-07

## Context

The library's realtime surface is SSE-only (ADR-0046), and since the
datastar/go-sse layer split (ADR-0049) the `datastar/v4` module and the root
`cqrshtmx.Broadcaster` can share one fan-out hub (`Hub()` /
`NewBroadcasterFromHub`). A consumer can therefore serve the same domain
events to HTMX clients (partial-HTML SSE) and DataStar clients (signal and
fragment patches) simultaneously — the mechanism is shipped and tested.

What is NOT decided is the product strategy: how far the library goes in
packaging dual-protocol support. The research report
(`docs/architecture-understanding/2026-09-07_datastar-dual-frontend-support.html`)
laid out three models:

- **Model A — route-split coexistence.** Both protocols served from separate
  endpoints (`/sse` for HTMX, `/ds/events` for DataStar) backed by one shared
  hub. Works today with zero code; needs only docs and wiring.
- **Model B — dual-mode handlers.** Every handler auto-detects the protocol
  (`Datastar-Request` header) and renders either HTMX partials or DataStar
  patches from one route.
- **Model C — parallel panel variants.** Ready-made UI panels (adminui,
  dashboardui, loginpage) get DataStar twins.

Demand evidence (surveyed 2026-09-07, feeding this decision):

- GitHub: 3 issues total, 0 mention DataStar; discussions disabled. No public
  signal — but the repo is a library whose consumers are private projects.
- **6 of 15 surveyed consumer projects already integrate go-datastar AND
  cqrs-htmx together**: InboxClean (×2 incl. lint-baseline), KeyHolderAI,
  bank-sync, crush-daily, file-and-image-renamer. All hand-roll the bridge:
  crush-daily carries a dedicated 100+ LOC `datastar_events.go` re-implementing
  stream lifecycle and heartbeat intervals the library's transport layer
  already provides; InboxClean builds `datastar.NewResponse` over raw
  `sse.Stream`s by hand. This is exactly the duplicated effort the shared-hub
  architecture was built to eliminate.

A second decision is forced by the route-split model: who mounts
`/ds/events`? `setup/v4` is the one-call composition root and already mounts
the session-gated shared `/sse` feed (`Config.SSEPath`) — but adding a
DataStar mount means setup gains a dependency on the `datastar/v4` module,
and the library principle (never enforce defaults consumers might disagree
with) demands the dependency be opt-in, not ambient.

Third: sequencing against the OPEN `/sse` endpoint-shape decision
(`docs/planning/2026-08-30_sse-endpoint-shape-decision.md`) — that one-pager
asks whether the shared feed must be scoped (per-tenant filters) before its
shape is "public". It is the user's product/security call and remains open.

## Decision

**1. Model A (route-split coexistence) is the strategy. Model B is rejected.
Model C is demand-gated, panel by panel.**

- Route-split matches how consumers already work (separate channels, one
  event source), keeps the test matrix linear, and is served entirely by the
  existing shared-hub mechanism.
- Model B rejected: one route producing both wire formats doubles the
  rendering/test surface of every handler for no consumer demand, and the
  HTMX partial and DataStar patch representations of a UI state are not
  derivable from each other — they are separate render functions regardless.
- Model C (panel variants): each panel variant must justify itself with
  demand evidence before any build starts (dashboardui spike, adminui design
  doc, loginpage signals are the evaluation vehicles — see the rollout plan,
  Tier 4). No variant work without a recorded go.

**2. setup gains a direct, opt-in dependency on `datastar/v4`.**

A new `setup.Config.DataStarPath` (plus `DataStarScriptPath`) mounts, when
non-empty (default: empty = nothing mounted, zero behavior change):

- `ds.ScriptHandler()` at the script path (serves the DataStar SDK JS).
- a DataStar SSE feed at the events path via
  `ds.NewBroadcasterFromHub(bundle.Broadcaster.Hub())` — the same hub that
  backs `/sse`, so one `Broadcast` reaches both transports.

Rationale for direct dependency over a separate `setup-datastar` bridge
module:

- setup is already the composition root that opts consumers into adminui +
  dashboardui + loginpage + usermgmt; DataStar is one more panel-level
  capability behind an empty-default flag, not a new ambient dep.
- The bridge would be ~50 LOC — below the threshold where a module earns its
  keep (own go.mod, lint config, coverage gate, release-train entry, README,
  docs). The 27-module workspace does not need a 28th for 50 LOC.
- Precedent: `health/v4` depends on the go-health-dashboard UI library
  directly; ROADMAP treats that as the accepted pattern for UI-bearing
  bridges.
- Consumers who want setup without DataStar simply leave the flag empty; the
  dependency is compile-time-only until opted into.

**3. `/ds/events` is session-gated by default (401 without a session),
mirroring the dashboard gate and the `SSEPath` contract.**

Event metadata (stream IDs, types) is not public data; the DataStar feed
streams the same events as `/sse` and must not be a weaker sibling. The gate
is part of the option's contract, not a consumer afterthought.

**4. Sequencing vs the open `/sse` endpoint-shape decision: adopt, don't
block.**

The DataStar feed ships with the same posture `/sse` has today
(session-gated, authenticated = full feed). The open decision — if the user
later rules that feeds must be scoped before "public" — applies to BOTH
feeds symmetrically: `transport.WithSSEFilter` composes identically on the
DataStar path (the filter lives in the shared transport layer, below the
wire-format split). A scoping decision therefore never re-opens this ADR; it
lands once and covers both endpoints. No mount is held hostage to the open
decision.

## Consequences

### Positive

- One shared hub, two wire formats: a single `Broadcast`/domain event
  reaches HTMX and DataStar clients with no bridging code.
- The 6 known dual-integration consumers can delete their hand-rolled
  bridges; setup composes it in one config flag.
- Zero behavior change for existing consumers (both new config fields
  default to empty; no route mounted).
- Test matrix stays linear: each transport is tested against the shared hub
  independently (script 200/ETag/304, SSE handshake, cross-transport
  broadcast, default-off 404s, gate posture).

### Negative

- setup's dependency surface grows by `datastar/v4` (which pulls go-datastar
  + go-sse — both already in the workspace graph via integration_test).
- Two docs surfaces (HTMX + DataStar) must stay consistent on the shared
  feed's contract; drift risk between `sse-and-datastar.md`,
  `datastar-integration.md`, and the setup README.

### Mitigation

- The dependency is inert unless `Config.DataStarPath` is set; the feature
  is exercised by tests only when configured, and the setup README documents
  the flag next to `SSEPath` so the two are discovered together.
- Guide drift is guarded by cross-links in both directions (recipe in
  `fullstack-wiring.md` ↔ coexistence table in `sse-and-datastar.md` ↔
  "Using with setup" in `datastar-integration.md`) and by the rollout
  plan's living-docs task (M8).

## References

- Research report: `docs/architecture-understanding/2026-09-07_datastar-dual-frontend-support.html`
- Rollout plan (execution): `docs/planning/2026-09-07_17-18_datastar-dual-frontend-rollout.html`
- Open `/sse` decision this ADR sequences against:
  `docs/planning/2026-08-30_sse-endpoint-shape-decision.md`
- Shared-hub architecture: `docs/guides/sse-and-datastar.md` · ADR-0049 · ADR-0046
