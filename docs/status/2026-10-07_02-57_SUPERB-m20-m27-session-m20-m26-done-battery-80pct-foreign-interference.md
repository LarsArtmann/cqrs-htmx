# SUPERB Session Status — M20–M26 Executed, M27 Battery 80% (foreign-session interference), Plan 26/27 Milestones Done

> Written 2026-10-07 02:57 CEST. Scope: this session only (M20 through M26
> fully executed + M27 partially; M17–M19 were the previous session's work,
> verified green at session start). Tree state at write time: foreign session
> actively editing (components.templ churn) — my modules last verified rc=0.

## a) FULLY DONE (verified green + daemon-landed)

**M20 — Autodetect variadic + probe table (F68–F71)** ✅
- `Autodetect(sources ...any)` — backward-compatible variadic; first-wins
  merge across sources (primary store first, auxiliaries after); conflicts
  never error. `fieldProbe[T]` generic probe over `readProbes`/`optionalProbes`
  tables replaces the eleven-branch assertion chain and its cyclop nolint.
  Aggregate read-interface gate; zero-sources/nil rejections name source
  types. Tests: first-wins conflict, complementary merge, read-from-later
  source, no-sources. README § Integration Modes documents merge semantics.
  Fix worth remembering: comparing a type-parameter value to nil is illegal —
  `any(*p.field(config)) != nil` boxing is the pattern.
  dashboardui tests rc=0; lint clean of new findings; CHANGELOG receipt.

**M21 — Routes() manifest + single-source route table (F72–F74)** ✅
- `Routes() []Route{Method,Pattern,Panel,Write}` manifest generated from ONE
  `routeSpec` table that also drives mux registration — `routes()`'s
  eleven-block if-chain and its cyclop nolint are DELETED. Guard modes
  (always/never/versionz) default fail-closed; write routes vanish under
  ReadOnly; panel names are constants; capabilityRoutes split into
  unconditional/capability/inspection groups (funlen). The F73 test pins
  manifest ≡ registration via `mux.Handler` pattern matching (every listed
  route resolves to its exact ServeMux pattern); write-count (5) and
  Journal-only (10 routes) pinned. README § Route Manifest (allowlist
  example) + CHANGELOG receipt. Full suite green; lint clean.

**M22 — systemadapter DomainConfig options (F75–F78)** ✅
- `DomainConfig(opts ...DomainOption)` with `WithCheckpointStore` (→
  system.DomainConfig.CheckpointStore; nil = system's ADR-0142 engine-backed
  default), `WithDeadLetterStore` (replaces in-memory DLQ default), and
  `WithHostOptions(...)` (appended after curated defaults: memory DLQ
  threshold 10, 3 restarts, capped backoff, batch 256 — ProjectionLayer
  parity). Name collision with the legacy option solved by UNIFYING:
  `DomainOption = ProjectionLayerOption = func(*adapterOptions)` — one
  vocabulary at both call sites. Restart proof
  (`TestDeclarative_DurableCheckpointSurvivesRestart`): file-backed SQLite
  journal, two `system.New` boots, one shared recording checkpoint store —
  second boot loads exactly the saved non-zero positions. Passed FIRST RUN.
  Guide § Custom checkpoint / dead-letter stores rewritten with the
  durable-checkpoint + fresh-in-memory-read-model semantics. Module tests
  rc=0 AND lint rc=0 (zero findings).

**M23 — Deployment presets (F79–F83)** ✅
- `RecommendedMemoryDeployment()` / `RecommendedSQLiteDeployment(dsn)` /
  `RecommendedSplitSQLiteDeployment(eventsDSN, projectionsDSN)` (WAL pragmas;
  engine/driver name constants after goconst; per-call pragma slices after
  gochecknoglobals). `ExampleDomainConfig` godoc example;
  `setupDeclarativeSystem` test helper now boots through the memory preset
  (live coverage). systemadapter/README.md CREATED (declarative-first quick
  start, preset table, knobs, legacy note — caught myself referencing the
  not-yet-existing FromSystem and fixed at write time). Package doc polished
  (examples/system-demo link). Guide's three hand-rolled DeploymentConfigs
  deduped onto presets. Tests rc=0, lint rc=0.

**M24 — docs truth pass (F84–F89)** ✅
- metaengine guide: ADR-0051 deprecation banner; Quick Start rewritten
  declarative-first (steps 4-5 ProjectionLayer steps deleted; preset + typed
  query helpers); export table updated; Advanced section rewritten to the
  REAL pattern (`OnRecordTyped` + `projectionadapter.EventWithID[P]` — the
  old `metaengine.On` snippet was wrong); Lifecycle section marked legacy.
- dashboardui README: dead Copy-to-Clipboard section REMOVED (verified:
  `data-copyable` exists only in docs — removed v4.10.0 for library
  CopyButtons). Demo section KEPT — research idea 75 was already obsolete:
  `examples/dashboard-demo` exists (verified: 8 users Alice..Henry, orders,
  projection host, 5s live-event publisher).
- core/capabilities.go package doc: "fmt.Fprintf-based handlers" claim
  dropped (templ since 2026-09-22).
- IMPROVEMENT_IDEAS.md pruned with per-item verification: T12/T13/T17/T21/
  T22/T24 + the data-loading split struck as SHIPPED with evidence; T18
  (command/query status badge) verified genuinely open and kept; stale-item
  check note added to the header.

**M25 — FromSystem duck-typed bridge (F90–F94)** ✅
- `dashboardui.System` — a STRUCTURAL accessor interface (EventStore/Bus/
  ProjectionHost/SnapshotStore/CommandStore/QueryStore) satisfied by
  `*system.System` and any forwarding wrapper, WITHOUT importing the system
  package (idea 293 + 308 solved better than the plan's concrete mapping:
  decorators stay probeable, dashboardui gains no system/metaengine dep).
  `FromSystem(sys)` maps every capability first-wins; engines without a
  native StreamReader get a journal-derived one
  (`listing.NewInMemoryStreamReader(config.Journal)`), so the aggregates
  panel lights up everywhere. `Autodetect` special-cases System sources
  (stores and systems mix under the usual first-wins). Unit tests
  (full/partial/merge) + F93 integration proof
  (`TestDashboard_FromSystem_PanelsLightUp`: real system.New on the memory
  preset; Overview/Events/Aggregates/Projections render 200 through the
  mounted routes) — passed FIRST RUN. README § Integration Modes row +
  section; CHANGELOG receipt.

**M26 — Telemetry phase 1 (F95–F99)** ✅
- `core/telemetry.go`: dashboard-owned view structs (TopologyView,
  EngineHealthView, PlacementView, EngineStatsView) + provider types — no
  system/metaengine types in the rendering layer.
- Config: four optional providers (Topology, EngineHealths, Placements,
  EngineStats) in core.Config + root Config + coreConfig mapping;
  `Capabilities.Telemetry`; nav "Telemetry" (icons.Signal); `/telemetry`
  route row (inspectionRoutes split).
- Page (`telemetry.templ` + handlers + helpers): topology table (instances/
  buses/projection-host line), query-placement table (query → engine + ADT +
  volume + est. latency), engine latency cards (RTT EWMA/P95/samples via
  display.Grid — the bundle-covered utility) + stats table with freshness
  badges (stale at 5 min; explicit no-tracker/no-samples states). Sections
  render independently; provider errors show inline without blanking others.
- Probe composition (F96): `/-/healthz` gains an `engines` array (liveness
  STAYS 200 — a sick engine is not a dead process); `/-/readyz` returns 503
  `engine_unhealthy` while any engine is down (quarantine posture). Without
  a provider, probes are byte-for-byte unchanged (pinned by test).
- Adapter: `dashboardui/systembridge` package — `WireTelemetry(&cfg, sys)`
  maps sys.Snapshot/HealthCheckDetailed/MetaEngine().Plan()/GetEngineStats
  onto the four providers. Integration proof
  (`TestDashboard_TelemetryFromSystem`) passed FIRST RUN. 5 unit tests +
  2 integration tests green; lint clean after fixes (varnamelen, wsl,
  wrapcheck→errorfamily.WrapInfrastructure, funlen, gci).
- README § Telemetry + CHANGELOG receipts (dashboardui).

**Session housekeeping** ✅
- Post-daemon compile checks after every noticed daemon commit (the M19
  incident's lesson) — every one green.
- Defaults applied to the previous report's §g questions: (1) push/train
  HOLD (F103/F104 untouched), (2) M18 dual-field kept, (3) manual compile
  discipline (no guard mechanization without owner decision).
- preflight-tree-check + wait-tree-quiet used before every tree-wide battery.

## b) PARTIALLY DONE

**M27 — verification battery (F100–F102)** 🟡
- F102 (CHANGELOG receipts): DONE continuously — dashboardui (M20, M21, M25,
  M26) + systemadapter (M22, M23; the M26 telemetry receipt was REMOVED when
  the adapter moved to systembridge).
- F100: preflight ✅ → wait-tree-quiet ✅ → `.#check-modules` ❌ → diagnosed
  + fixed MY half → scoped re-verification ✅ → **full battery re-run NOT
  done** (interrupted by foreign-session breakage, then by this report).
  - check-modules failed: systemadapter hermetic (GOWORK=off) build —
    systemadapter required dashboardui/v4 v4.13.1 whose PUBLISHED tag lacks
    the new APIs. FIXED PROPERLY: the adapter moved from systemadapter into
    a NEW `dashboardui/systembridge` sub-package (system + metaengine are
    CROSS-REPO PUBLISHED deps — hermetic-clean; only the intra-family
    require was the problem). systemadapter tidy'd back (dashboardui require
    dropped); systemadapter hermetic rc=0. integration_test (never tagged)
    got a documented `replace dashboardui/v4 => ../dashboardui` (hermetic
    rc=0; remove after the family train publishes the tags).
  - Remaining battery redness is NOT mine: dashboardui hermetic fails on
    `cqrshtmx.ServeAsset`/`AssetETag` — the FOREIGN session's new
    unpublished root APIs (their assets refactor across root + adminui +
    dashboardui); `.#test` failed in adminui on the same class. After their
    tree went quiet, scoped re-verification: dashboardui, systemadapter,
    integration_test ALL rc=0.
- F101 (GCL battery): GCL untouched this session — reduces to the #lint
  verdict re-run, STILL NOT DONE (carried from previous session).
- F103/F104 (trains): owner-gated, deliberately NOT started.

## c) NOT STARTED

- The full `.#check-modules` / `.#lint` / `.#test` / `.#coverage-gate`
  re-run after the systembridge move (my half is verified scoped-green; the
  meta-gates want one clean pass).
- GCL `#lint` verdict (fixed log path — the lesson from the lost shell 008).
- Release trains (owner-gated): wave-ordered tags (dashboardui FIRST —
  FromSystem/systembridge/telemetry APIs — then systemadapter, then the
  integration_test replace removal).
- Post-train: bump systemadapter/integration_test requires, annotate the
  research report's consumed ideas (42, 281, 301, 38, 293, 294, 308, 274,
  275, 263, 265, 75–78), TODO_LIST harvest, AGENTS.md memory notes.

## d) TOTALLY FUCKED UP (incidents, recovered)

1. **Silent write loss on routes.go (M21):** the write tool reported SUCCESS
   but disk held the old content (mtime PREDATED the write) — a concurrent
   session raced my full-file rewrite; I only noticed via stale lint output.
   Recovery: re-applied, then verified ON DISK with grep immediately after
   every large write from then on. Lesson (now standing practice): after any
   full-file write in this contended tree, grep for a new symbol + check
   mtime before building.
2. **Foreign-session transient breakages ×3** (layout.go `contentETag`
   undefined; assets_test.go half-commented function = syntax error; adminui
   `AssetETag` undefined in the full battery): all handled by NOT touching
   their files + waiting (sleep-then-revet / wait-tree-quiet). The battery
   redness they caused cost ~20 min of re-verification.
3. **My python-edit anchor typos ×2:** (a) omitted the `...` variadic spread
   in a routeTable anchor — count 0 with bytes looking identical until I
   dumped `repr()`; (b) misread go.work replaces as integration_test/go.mod
   replaces (grep output of a two-command call). Both caught by asserts;
   fixed with byte-precise anchors (cat -A / python repr).
4. **One wasted draft:** first system_bridge.go rewrite drifted into a
   pointless indirection (projectionhostAccessor with undefined types).
   Recognized immediately, replaced with the simple concrete version.
5. **Near docs-lie caught twice:** wrote `systemadapter.WireDashboardTelemetry`
   into README/CHANGELOG before the API existed (fixed at write time), then
   had to chase all references again when the hermetic constraint moved the
   function to systembridge. Docs now name only real APIs.

## e) WHAT WE SHOULD IMPROVE

1. **Write-then-verify discipline in contended trees:** every full-file write
   needs an immediate on-disk grep. Consider having the daemon pause on
   files modified <60s ago.
2. **Hermetic-first thinking:** every new cross-module API call inside the
   family should be checked with `GOWORK=off go build` in the CONSUMING
   module BEFORE writing the code around it — the systembridge move would
   have been designed-in from the start (5 min then, 25 min later).
3. **Battery orchestration under concurrency:** the full battery is
   all-or-nothing over a tree a foreign session keeps churning. Scoped
   per-module verification (what I fell back to) should be a first-class
   flake app so "my modules are green" is cheap to prove.
4. **GCL lint verdict:** two sessions in a row now. Run it as a background
   job with a FIXED absolute log path, FIRST thing next session, before any
   other work.
5. **CHANGELOG receipts reference call sites that move** (M26's
   systemadapter→systembridge). Receipt should name the API, not the module
   path, when the owning module is still settling.

## f) NEXT (priority order)

1. Re-run `nix run .#check-modules` (expect: systemadapter green; dashboardui
   red ONLY on foreign ServeAsset pin — recheck after their session lands).
2. Re-run `nix run .#test` full battery once the foreign session is quiet.
3. `nix run .#lint` + `nix run .#coverage-gate` (new files: telemetry,
   systembridge, routes table — thresholds may need bumps).
4. GCL `#lint` verdict re-run (fixed path `/tmp/gcl-lint-verdict.log`).
5. `nix run .#check-css-bundle-classes` — telemetry.templ adopted
   display.Grid/statCard/badge; verify the bundle class set still matches.
6. `nix run .#check-templates` + `.#check-codegen` (telemetry_templ.go fresh).
7. `cqrs-lint` on dashboardui (new package systembridge + new exports).
8. Owner decision: open the push/train window. Wave order: dashboardui
   v4.14.0 FIRST (FromSystem, Routes, Autodetect variadic, telemetry,
   systembridge, presets-independent changes), then systemadapter v4.12.0
   (DomainConfig options + presets), then examples/integration_test re-pins,
   then drop integration_test's dashboardui replace.
9. After train: bump integration_test require to the published dashboardui
   tag and DELETE the local replace.
10. Annotate `docs/research/2026-10-06_dashboardui-metaengine-system-
    improvements.md` ideas consumed this session (42, 38, 281, 301, 293, 294,
    308, 274, 275, 263, 265, 75, 76, 77, 78) with outcome notes.
11. Annotate the previous status report (§g questions) with the defaults
    applied + pointers to this report.
12. TODO_LIST.md harvest: T18 (status badge — verified open), SECURITY.md
    (idea 79), telemetry phase 2 items.
13. AGENTS.md memory: systembridge package exists (the one system-importing
    package), integration_test dashboardui replace + why, FromSystem duck
    pattern, routes table single-source invariant + the mux.Handler drift
    test.
14. examples/system-demo: adopt presets + FromSystem + WireTelemetry (it
    still hand-wires; dedupe + it becomes the living demo of M22/M23/M25/M26).
15. examples/dashboard-demo: consider FromSystem mix-in demonstration.
16. godoc Example for FromSystem + WireTelemetry (parity with
    ExampleDomainConfig).
17. Telemetry page: htmx.PolledRegion or SSE refresh (phase 2; idea 274/275
    remainder).
18. Surface planner RuleTrace/Diagnostics in the placements table (phase 2).
19. CacheTierInfo (Topology.CacheTiers) — currently unmapped in
    topologyViewFrom; map or document why not.
20. DispatcherInfo rows in the topology section (currently unmapped).
21. Golden-file tests for telemetry panel markup (F99 used content
    assertions; goldens pin regressions harder).
22. E2E (e2e/server) route for /telemetry once setup wires providers.
23. setup module: expose telemetry wiring seam (setup.Config.Dashboard*
    surface — likely just docs once consumers ask).
24. adminui parity: Routes() manifest + single-source route table (the M21
    pattern transfers 1:1).
25. Fix the 5 pre-existing dashboardui lint findings (handlers_events cyclop,
    handlers_audit dupl ×2, accent_color gochecknoglobals+mnd) — foreign
    debt, needs owner OK since another session owns those files.
26. Bench: routes() now builds a 31-row table per Handler() call — benchmark
    vs the old if-chain; if measurable, cache routeTable per Dashboard.
27. Bench-spike re-pin ONLY if (26) changes hot paths (same-change rule).
28. branching-flow baseline re-pin DOWN after new code lands ratchet-clean.
29. SECURITY.md for write mode (idea 79; ReadOnly=false checklist).
30. versionz: consider exposing telemetry-provider presence in capabilities
    JSON (already automatic via caps — verify rendering).
31. Consider `Autodetect` doc-comment update: it now says nothing about
    System sources (the special case is documented on FromSystem only).
32. Verify `.#check-session-route-wrappers` still green (new /telemetry is
    guard-wrapped, not session-gated — expected pass).
33. Consider exporting systembridge.FromSystem-style combined helper
    (`systembridge.Configure(&cfg, sys)` = FromSystem + WireTelemetry) —
    one-call ergonomics for the system path.
34. Coverage-gate thresholds for systemadapter (new tests added) + dashboardui
    (new files) — check the flake's numbers before the train.
35. Post-train: pkg.go.dev check for the new dashboardui exports rendering.

## g) QUESTIONS (cannot figure out myself)

1. **Push/train window:** CH is now ~65+ commits ahead of origin (M17–M26 all
   unpushed), GCL ~40+ ahead, and a foreign session is mid-flight on the
   root/adminui asset refactor. Open the window now (I would: full battery
   green → wave-order tags dashboardui-first → push), or keep HOLD until the
   asset refactor lands?
2. **Foreign-session coordination:** dashboardui's hermetic build is red on
   THEIR unpublished `cqrshtmx.ServeAsset`/`AssetETag` APIs, which blocks a
   fully-green `.#check-modules`. Is that session done (should I finish/fix
   its pin), in progress (I keep waiting), or should the check-modules
   battery exclude their churn explicitly?
3. **Post-daemon-commit compile guard:** mechanize it (a lightweight daemon
   hook or watcher that compiles touched modules after every daemon commit
   and logs/flags red states — would have caught the M19 incident within
   seconds), or keep manual discipline? If mechanize: hook into the daemon,
   or a separate watcher script + flake app?
