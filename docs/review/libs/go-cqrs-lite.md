# go-cqrs-lite Utilization Report

> Audit date: 2026-11-15 (version re-true) · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/go-cqrs-lite/**` (multi-module, per-module version trains) · Checkout: `/home/lars/projects/go-cqrs-lite`

## Verdict at a glance

| Pin(s)                | Latest (verified 2026-11-14: tag checkout + live treadmill)                  | Prod import sites | Test import sites | Adoption                  | Score                                            |
| --------------------- | --------------------------------------------------------------------------- | ----------------- | ----------------- | ------------------------- | ------------------------------------------------ |
| 38 trains (see below) | **38/38 = latest** (2026-11-15 re-true: the 11-14 metaengine lag was swept same-day — treadmill worst lag 0) | 171 files         | 235 files         | **Used — the foundation** | **73/100 audited → ~80 (11-05) → 90.5/100 standing** |

## What the library is

The composable CQRS + event-sourcing foundation under every cqrs-htmx module: deciders, command/query dispatch (with a middleware factory family), event journals, projections (projectionhost: checkpoints, DLQ, crash-restart), snapshots, the `system.New` composition root, `metaengine` (cost-based planner over interchangeable engines), scheduling, scenario testing, id/branded-ID, schema evolution, upcasters. cqrs-htmx is, by design, the HTTP/HTMX face of this library.

## How cqrs-htmx uses it today

- **171 production files** (235 test files) across every module import it — the deepest dependency in the fleet.
- **38 pinned trains** (all verified against `go.mod` files; only stale pin in-tree is the poisoned fixture `scripts/testdata/verify-tag/real-setup-v4.8.1-poisoned/`): `event/v4 v4.15.2`, `command/v4 v4.14.1`, `query/v4 v4.10.1`, `system/v4 v4.19.0`, `metaengine/v4 v4.26.0`, `middleware/v4 v4.7.2`, `storage/v4 v4.11.1`, `projectionhost/v4 v4.7.1`, `kv/v4 v4.4.0`, `catalog/v4 v4.15.0`, `commandlifecycle/v4 v4.3.1`, `dispatcher/v4 v4.5.2`, `scheduling/v4 v4.7.0`, `snapshot/v4 v4.7.0`, `stack/v4 v4.5.1`, `systemcatalog/v4 v4.0.0`, `dedup/v4 v4.3.0`, `scenario/v4 v4.4.3`, `watermill/v4 v4.7.0`, `id/v4 v4.7.5`, `record/v4 v4.6.2`, `schema/v4 v4.7.1`, `metadata/v4 v4.7.3`, `listing/v4 v4.4.4`, `decider/v4 v4.8.0`, `otel/v4 v4.5.3`, `projection/v4 v4.4.2`, `storage/memory/v4 v4.6.2`, `metaengine/{projectionadapter v4.6.2, sqliteengine v4.9.1, bboltengine v4.3.3}`, `commandlifecycle/projections/v4 v4.2.3`, `event/v4/eventtest v0.4.0`, `claiming/v4 v4.0.2`, `encryption/v4 v4.4.6`, `prometheus/v4 v4.3.4`, `signing/v4 v4.4.2`, `systemscenario/v4 v4.3.0`.
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

1. **metaengine family lag — RESOLVED 2026-11-15 (the B10.3 replay's gotcha-39 catch, swept same-day)** — `metaengine/v4` now v4.26.0, `metaengine/sqliteengine/v4` now v4.9.1, `system/v4` v4.19.0; the treadmill gate reads worst lag 0 across 20 repos · 81 prefixes (first-hand re-run 2026-11-15 13:16). The v4.24.0+ nested `metaengine/v4/` dir restructure is consumed and hermetic-green.
2. **cqrs-lint gate binary — RESOLVED 2026-11-15 (plan B4)** — rebuilt at v4.21.0 (commit 213aa5e, ldflags version-pinned); the waiver-adoption delta exercised live when the repo-wide E014 anchor moved; the gate runs GREEN 14/14 under BOTH the new build and system 4.20.1.
3. **flightrecorder** — the last missed row; VERDICT RECORDED 2026-11-15 (plan B7 spike): the wiring recipe (`middleware.CommandFlightRecorder(rec, trigger)` over `ServiceConfig.CommandMiddleware`) is documented in `docs/guides/leveraging-go-cqrs-lite.md` §2.8 and COMPILE-VERIFIED against the pinned middleware train (throwaway module, v4.7.2 — `TriggerContext`/`WithMaxSnapshots`/`WithFile` all resolve). Deliberate N/A-with-reason for in-tree adoption: a flight recorder is PROCESS-GLOBAL (upstream doc: exactly one active recorder per process) — a library cannot safely default it on, and the incident-response story (who reads `flight.log`, when snapshots trigger) is a consumer-operational decision. The consumer seam is one `CommandMiddleware` entry; the dashboard snapshot panel idea stays on the ROADMAP, not wired. Score row stays missed ×0 honestly (documented ≠ used) — the evaluation is the receipt.
4. **testutil / idtest / querytest** still unimported (eventtest only partially covers the helper family).
5. **Deprecated `meta.CausationID` reads ×3** — `dashboardui/detail_items.go:166,187,269`; rename at the next family wave (the `record.Cause` ×3 reads stay v5-gated — upstream has no v4 API).
6. **v5 exposure tail** — the 68 `SQLViewStore`-family findings: ADR-0051's criterion re-verified **PARTIALLY MET** 2026-11-07 (checkpoints engine-backed, `ResetProjection` published, hydration met by construction on persistent engines); the 2026-11-07 migration plan (W0–W5) supersedes "blocked" — W1 probes GO, one real preset starvation bug already fixed upstream of the fleet.

## Version currency

**38 of 38 trains at latest; worst lag 0**, re-verified 2026-11-15 via `ls-remote` spot checks (metaengine v4.26.0, sqliteengine v4.9.1, system v4.19.0, catalog v4.15.0 all match their pins) and the repo's treadmill gate (`nix run .#check-upstream-treadmill` → "20 repos · 81 upstream prefixes · worst lag 0 · 0 disagreement warning(s)", first-hand run 2026-11-15). The 11-14 "metaengine lag 1" state was swept same-day (gotcha 39: the train-lag tail regrows under you — the B10 verify-block replay caught the v4.26.0 re-lag within hours; the sweep recipe landed it). The treadmill gate tracks this continuously.

## Should cqrs-htmx use it (more)?

**Yes — and it now is, near the max.** Verdict: very high adoption, 0 anti-patterns, every graded open row from the 2026-11-03 plan of record executed and receipted, and the version tail is fully current (worst lag 0). The residual is one missed row (flightrecorder — deliberately N/A-adjacent until an incident-response story exists; B7 verdict recorded), the helper-family partials, and the owner-gated usermgmt → system/metaengine migration (W2+).

## How to verify

```bash
rg -l '"github.com/larsartmann/go-cqrs-lite/' --glob '**/*.go' -g '!scripts/testdata/**' . | wc -l   # 406 files (171 prod / 235 test)
for m in command event id query storage middleware projectionhost system metaengine; do
  git -C ~/projects/go-cqrs-lite ls-remote --tags origin | grep -oE "$m/v4(\.[0-9]+)+$" | sort -V | tail -1
done   # all = pinned (worst lag 0; metaengine restructure consumed)
rg -c '<tr><td>[0-9]+</td>' docs/research/2026-11-03_go-cqrs-lite-deep-dive.html   # 47 rows
go test ./systemadapter/ -run 'TestArchitectureCatalog'   # R2 drift + parity gates
nix run .#check-upstream-treadmill   # worst lag 0
```

## Prior reports

- [2026-11-03 deep dive](../../research/2026-11-03_go-cqrs-lite-deep-dive.html) — the scored audit this report updates.
- `docs/research/go-cqrs-lite-feature-audit.html` — legacy-era audit (superseded).
- [ADR-0059](../../adr/0059-cbor-binary-event-payloads.md), [ADR-0051](../../adr/), [ADR-0165] — decisions gating the open items.
