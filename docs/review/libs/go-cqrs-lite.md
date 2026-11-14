# go-cqrs-lite Utilization Report

> Audit date: 2026-11-14 · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/go-cqrs-lite/**` (multi-module, per-module version trains) · Checkout: `/home/lars/projects/go-cqrs-lite`

## Verdict at a glance

| Pin(s)                | Latest (verified 2026-11-14: tag checkout + live treadmill)                  | Prod import sites | Test import sites | Adoption                  | Score                                            |
| --------------------- | --------------------------------------------------------------------------- | ----------------- | ----------------- | ------------------------- | ------------------------------------------------ |
| 38 trains (see below) | **37/38 = latest; metaengine lag 1** (v4.23.0 → v4.24.0, treadmill worst-lag) | 170 files         | 233 files         | **Used — the foundation** | **73/100 audited → ~80 (11-05) → 90.5/100 standing** |

## What the library is

The composable CQRS + event-sourcing foundation under every cqrs-htmx module: deciders, command/query dispatch (with a middleware factory family), event journals, projections (projectionhost: checkpoints, DLQ, crash-restart), snapshots, the `system.New` composition root, `metaengine` (cost-based planner over interchangeable engines), scheduling, scenario testing, id/branded-ID, schema evolution, upcasters. cqrs-htmx is, by design, the HTTP/HTMX face of this library.

## How cqrs-htmx uses it today

- **170 production files** (233 test files) across every module import it — the deepest dependency in the fleet.
- **38 pinned trains** (all verified against `go.mod` files; only stale pin in-tree is the poisoned fixture `scripts/testdata/verify-tag/real-setup-v4.8.1-poisoned/`): `event/v4 v4.15.2`, `command/v4 v4.14.1`, `query/v4 v4.10.1`, `system/v4 v4.18.0`, `metaengine/v4 v4.23.0`, `middleware/v4 v4.7.2`, `storage/v4 v4.11.1`, `projectionhost/v4 v4.7.1`, `kv/v4 v4.4.0`, `catalog/v4 v4.15.0`, `commandlifecycle/v4 v4.3.1`, `dispatcher/v4 v4.5.2`, `scheduling/v4 v4.7.0`, `snapshot/v4 v4.7.0`, `stack/v4 v4.5.1`, `systemcatalog/v4 v4.0.0`, `dedup/v4 v4.3.0`, `scenario/v4 v4.4.3`, `watermill/v4 v4.7.0`, `id/v4 v4.7.5`, `record/v4 v4.6.2`, `schema/v4 v4.7.1`, `metadata/v4 v4.7.3`, `listing/v4 v4.4.4`, `decider/v4 v4.8.0`, `otel/v4 v4.5.3`, `projection/v4 v4.4.2`, `storage/memory/v4 v4.6.2`, `metaengine/{projectionadapter v4.6.2, sqliteengine v4.8.0, bboltengine v4.3.3}`, `commandlifecycle/projections/v4 v4.2.3`, `event/v4/eventtest v0.4.0`, `claiming/v4 v4.0.2`, `encryption/v4 v4.4.6`, `prometheus/v4 v4.3.4`, `signing/v4 v4.4.2`, `systemscenario/v4 v4.3.0`.
- Canonical integration map: [docs/guides/leveraging-go-cqrs-lite.md](../../guides/leveraging-go-cqrs-lite.md); runnable proofs in `examples/middleware-demo` and `examples/observability-demo`.
- **R2 (landed 2026-11-06)** — `systemcatalog` is now pinned and used: `systemadapter/architecture_catalog.go:24-40` derives the full command+event inventory from `DomainConfig()` via `systemcatalog.FromDomainConfig` (ephemeral in-memory boot), with drift gates at `systemadapter/architecture_catalog_test.go:46-57` (21-command + 27-event inventories) and `:64-92` (parity with the usermgmt published language). It does **not replace** `usermgmt.DefaultEventCatalog()` (`usermgmt/es_event_catalog.go:27`, still the served endpoint via `setup/machine.go:24`): one hand-curated table (`eventCatalogSpecs`) now feeds both wire formats, and `usermgmt/es_architecture_catalog.go:29` + its parity tests make silent drift impossible. The F032 `no-catalog-import` lint disables were removed.
- **CBOR (ADR-0059)** — five payloads CBOR-encoded via the shared `cborEventCodec` instance (`usermgmt/es_decide_credentials.go:15`): `CredentialAdded` :50, `CredentialRemoved` :93, `CredentialUsed` :155, `TOTPEnabled` (`es_decide_security.go:32`), `BotRegistered` (`es_bot_decide.go:63`). `dashboardui/core/payload.go` renders JSON/CBOR/Raw exhaustively (gotcha 55: `event.New` defaults CBOR since event/v4 v4.15.2; corrupt bytes fall back to CBOR EDN).
- **QUERY wave (root v4.18.x)** — typed query surface at the cqrs-htmx root: `Config.Queries *query.Dispatcher` (`app.go:54`), `App.Query` (`app.go:270`), `QueryTyped[Q,R]` (`app.go:325`), plus the `cqrs-query` htmx extension (native QUERY method with 405→POST downgrade).
- **M2M W0 (2026-11-06)** — `usermgmt.NewAPITokenMiddleware` (`api_token_middleware.go:45`) bridges `ActorIDFromBot` (:68) into `cqrshtmx.WithActorID`, composed with `middleware.CommandActorContext()` (`audit_context.go:40`) so bot calls land on command metadata (actor/audit chain).

## Score derivation

The [2026-11-03 deep dive](../../research/2026-11-03_go-cqrs-lite-deep-dive.html) graded **47 capability rows**: 10 Not-Applicable → 37 applicable; **22 fully leveraged / 10 partially used / 5 missed** = `(22 + 10×0.5)/37 = 73/100` (band: high).

Ten graded rows have since been resolved (each re-verified against source 2026-11-14):

| Row (audit grade)                            | Resolution (wave)                                                                                                                                  |
| -------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| CBOR for binary payloads (missed)            | ADR-0059 — five payloads `event.WithCodec(cborEventCodec)` (2026-11-04)                                                                            |
| Replay-seam dedup: raw map (partial)         | `dedup.Ring` at `transport/serve.go:475` (R3, `823fcd7e`)                                                                                          |
| `kv` FindByID write-through cache (partial)  | Opt-in cache at `usermgmt/view_cache.go:97` — now **upstream** `kv.NewCacheForViewStore` (kv v4.4.0; ask #113 answered; the 182-line otter adapter is gone, otter now only indirect) (R9, `72a0ae84`) |
| `WithSnapshotStateVersion` (partial)         | `decider.WithSnapshotStateVersion[State]("v1")` in `repositoryOptions` (`usermgmt/snapshot.go:112`) (R10, `13d0f0e9`)                              |
| systemcatalog derivation (missed)            | R2, see above — derived catalogs + `Assert{Event,Command}Inventory` gates + F032 removals (2026-11-06)                                              |
| commandlifecycle streams (missed)            | `ServiceConfig.CommandLifecycle` (`usermgmt/service_core.go:215`, factory `service_factory.go:128-134`) + dashboardui Command Fates panel (`handlers_command_fates.go`, `core/command_fates.go`) (R7/F1) |
| projectionhost DLQ memory-default (partial)  | `durableDeadLetterStore` — durable DLQ by default in the SQL setups (`usermgmt/es_projection_setup.go:72-80`, wired :154) (R4)                      |
| Checkpoint batching unused (partial)         | `ProjectionHostOptions []projectionhost.HostOption` threaded setup→usermgmt (`setup/config.go:490`, `usermgmt/service_core.go:187`), `WithCheckpointEvery` documented (`setup/config.go:487`) (R8) |
| metaengine engine breadth (partial)          | 8 `Recommended*Deployment` presets — memory/sqlite/split/pebble/bbolt/postgres/duckdb/turso (`systemadapter/presets.go:36-165`) via `system.Standard` (ADR-0165) |
| test helpers imported nowhere (missed)       | → **partial**: `eventtest.NewFakeBus` adopted in 4 modules (root `setup`, `dashboardui`, `e2e/server`, examples); `testutil`/`idtest`/`querytest` still unimported |

**Standing: 31 full ×1 + 5 partial ×0.5 + 1 missed ×0 = 33.5 / 37 ≈ 90.5/100** (band: very high). Remaining partial: record structural metadata (Cause reads are v4-impossible — no `metadata.Cause` exists in any v4 release, plan erratum), metaengine planner/folds direct use, `system.New` for usermgmt's own runtime (migration plan drafted, W1 probes GO 2026-11-08), otel/otlp one-call exporter, test helpers. Remaining missed: flightrecorder.

## Open findings (the residual tail)

1. **metaengine pin v4.23.0 vs upstream v4.24.0 (lag 1)** — the only stale train; v4.24.0 also restructures the module into a nested `metaengine/v4/` dir (dual tags `metaengine/v4.24.0` + `metaengine/v4/v4.24.0`), so the bump needs a path check, not just a version edit.
2. **cqrs-lint gate binary at v4.20.0 vs upstream v4.21.0** — delta is waiver adoption (orphaned waiver reasons reattach to moved findings; `cmd/cqrs-lint/waiver_adoption.go`, +92/-0 with tests). Rebuild via the documented git-archive + ldflags recipe.
3. **flightrecorder** — the last missed row; no import anywhere.
4. **testutil / idtest / querytest** still unimported (eventtest only partially covers the helper family).
5. **Deprecated `meta.CausationID` reads ×3** — `dashboardui/detail_items.go:166,187,269`; rename at the next family wave (the `record.Cause` ×3 reads stay v5-gated — upstream has no v4 API).
6. **v5 exposure tail** — the 68 `SQLViewStore`-family findings: ADR-0051's criterion re-verified **PARTIALLY MET** 2026-11-07 (checkpoints engine-backed, `ResetProjection` published, hydration met by construction on persistent engines); the 2026-11-07 migration plan (W0–W5) supersedes "blocked" — W1 probes GO, one real preset starvation bug already fixed upstream of the fleet.

## Version currency

**37 of 38 trains at latest; worst lag 1** (`metaengine/v4` v4.23.0 → v4.24.0), re-verified 2026-11-14 via the tag checkout and the repo's treadmill gate (`nix run .#check-upstream-treadmill` → "20 repos · 79 prefixes · worst lag 1"). Spot-checked 9 head trains with `ls-remote`: all match their pins except metaengine. The treadmill gate tracks this continuously.

## Should cqrs-htmx use it (more)?

**Yes — and it now is, near the max.** Verdict: very high adoption, 0 anti-patterns, every graded open row from the 2026-11-03 plan of record executed and receipted. The residual is the metaengine/cqrs-lint bumps, one missed row (flightrecorder — genuinely N/A-adjacent until an incident-response story exists), and the owner-gated usermgmt → system/metaengine migration (W2+).

## How to verify

```bash
rg -l '"github.com/larsartmann/go-cqrs-lite/' --glob '**/*.go' -g '!scripts/testdata/**' . | wc -l   # 403 files (170 prod / 233 test)
for m in command event id query storage middleware projectionhost system metaengine; do
  git -C ~/projects/go-cqrs-lite ls-remote --tags origin | grep -oE "$m/v4(\.[0-9]+)+$" | sort -V | tail -1
done   # all = pinned except metaengine v4.24.0 (pin v4.23.0)
rg -c '<tr><td>[0-9]+</td>' docs/research/2026-11-03_go-cqrs-lite-deep-dive.html   # 47 rows
go test ./systemadapter/ -run 'TestArchitectureCatalog'   # R2 drift + parity gates
nix run .#check-upstream-treadmill   # worst lag 1 (metaengine)
```

## Prior reports

- [2026-11-03 deep dive](../../research/2026-11-03_go-cqrs-lite-deep-dive.html) — the scored audit this report updates.
- `docs/research/go-cqrs-lite-feature-audit.html` — legacy-era audit (superseded).
- [ADR-0059](../../adr/0059-cbor-binary-event-payloads.md), [ADR-0051](../../adr/), [ADR-0165] — decisions gating the open items.
