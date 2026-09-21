# ADR 0051: V007 stack-surface disposition — deprecate now, delete in v5, gate cluster 1 on metaengine maturity

**Status:** Accepted (deprecations shipped in v4.12.0, 2026-09-22) · **Supersedes:** none · **Related:** ADR-0123/ADR-0126 (go-cqrs-lite), ADR-0050, `docs/planning/archived/2026-09-10_v007-spike-plan.md` (annotated), `docs/guides/v5-removal-inventory.md`

## Context

`cqrs-upgrade -dry-run --workspace` (2026-09-20) reports 78 V007 findings in
usermgmt against go-cqrs-lite's v5-removed APIs, in three clusters:

1. **68× `storage.SQLViewStore`-family** — the four SQL read models
   (`SQLUserReadModel` etc.) and their view-store plumbing.
2. **6× `stack.Materialize`** — `es_materialize_adapter.go`
   (`MaterializeProjection` + constructor + accessor + test).
3. **4× `stack.Bundle` → `system.New`** — `es_setup_core.go` (the `Bundle`
   field), `stack_repositories.go` (`buildStackRepositories`), and the three
   `//go:build ignore` SQL setup templates (sqlite/postgres/mysql).

The prepared spike plan (2026-09-10) sketched a code migration for clusters
2+3. Executing it against today's tree surfaces facts that change the
conclusion.

## Findings from the spike

1. **The default composition path is already stack-free.**
   `NewEventSourcedSetup` composes a raw `event.Store` + `event.Publisher`
   via `buildDeciderRepositories` — it never touches stack. The stack surface
   exists only in (a) the optional SQL setup **templates** and (b) the
   Materialize **adapter**.

2. **The SQL setup templates' value IS the preset wiring.**
   `stacksqlite.New(dsn)` composes ~200 lines of backend logic (pragmas,
   schema, durability tiers, view DSNs, secondary backends, capabilities).
   Re-deriving that from raw storage modules inside usermgmt would duplicate
   the presets — the exact composition stack exists to own. The v5-clean
   equivalent is not "hand-wire storage"; it is **systemadapter**:
   `systemadapter.DomainConfig()` + `system.New`, whose projection layer is
   already equivalence-tested against the legacy path
   (`TestDeclarative_EquivalenceWithProjectionLayer`,
   `TestDeclarative_SQLite_UserLifecycle`, `TestDeclarative_SQLite_AuthzPolicies`).

3. **The Materialize adapter has zero in-repo consumers.** Only its own test
   references it. It exists to bridge `stack.Materialize` (a v5-removed API)
   into `projection.Projection`; the declarative path replaces it.

4. **Owner decision (2026-09-21, recorded in go-cqrs-lite's ROADMAP):**
   `stack/v4` is REMOVED entirely in go-cqrs-lite v5 — not decoupled from
   metaengine. metaengine + `system/` is the intended direction; the coupling
   is direction-aligned.

## Decision

1. **Clusters 2+3: deprecate now, delete in the v5 bundle.** Exported symbols
   (`MaterializeProjection`, `NewMaterializeProjection`,
   `MaterializeProjection.Materialize`, `eventSourcedSetupCore.Bundle`) carry
   `// Deprecated:` markers pointing at the declarative path
   (`docs/guides/declarative-projections.md`) and ADR-0051. The SQL setup
   templates carry equivalent deprecation notes in their doc comments (they
   are `//go:build ignore` copy-paste templates — not compiled in the
   published module). No behavioral change; the existing equivalence tests
   in systemadapter are the proof the replacement path is sound.

2. **v5 cut:** delete `es_materialize_adapter.go` (+ test),
   `stack_repositories.go`'s `buildStackRepositories`, the three SQL setup
   templates + `sql_setup_shared.go`, and the `Bundle` field; consumers
   migrate to `NewEventSourcedSetup` (raw store+bus) or the systemadapter
   declarative path. Recorded in `docs/guides/v5-removal-inventory.md`.

3. **Cluster 1 stays gated — with a criterion, not a date.** The 68
   `SQLViewStore` findings are the SQL read models, whose v5 replacement is
   metaengine engines with layout planning (ADR-0126). **Go/no-go criterion:**
   migrate when go-cqrs-lite's metaengine layout-planning API covers
   secondary-index semantics (the read models' `FindByEmail`-style lookups
   need keyed access the current planner API does not express for SQL-backed
   view stores) AND the hydration contract (`Hydrator`) has a declarative
   equivalent. Until then the SQL read models compile and test green on the
   v4 trains — they are v5-prep, not a v4 blocker.

## Consequences

- V007's actionable v4 surface drops from 78 findings to 0 blocking: 10
  findings are dispositioned by deprecation (this ADR), 68 stay gated behind
  the cluster-1 criterion.
- usermgmt keeps a compile-time dependency on `stack/v4` until v5 (the
  templates + adapter + the deprecated field reference it). This matches the
  owner's decision that stack dies with v5, not before.
- The `#lint` SA1019 suppressions for the deprecated symbols ride under the
  existing "V007 v5-migration debt" rule in `usermgmt/.golangci.yml` until
  the v5 sweep deletes the files.
