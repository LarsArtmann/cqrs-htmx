# go-cqrs-lite Utilization Report

> Audit date: 2026-11-22 · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/go-cqrs-lite/**` (multi-module, per-module version trains) · Checkout: `/home/lars/projects/go-cqrs-lite` (read-only; HEAD feec9bc74, committed 2026-11-22)

## Verdict at a glance

| Pin(s)                | Latest (verified 2026-11-22: checkout tags + live treadmill)                                        | Prod import sites | Test import sites | Adoption                  | Score                                                     |
| --------------------- | ---------------------------------------------------------------------------------------------------- | ----------------- | ----------------- | ------------------------- | --------------------------------------------------------- |
| 39 trains (see below) | **39/39 = latest, worst lag 0** (treadmill run 2026-11-22; re-derived first-hand against the checkout's tags this session) | 181 files         | 243 files         | **Used — the foundation** | **73/100 audited → 90.5 (11-15) → 90.8/100 standing**     |

## What the library is

The composable CQRS + event-sourcing foundation under every cqrs-htmx module: deciders, command/query dispatch (with a middleware factory family), event journals, projections (projectionhost: checkpoints, DLQ, crash-restart, `WithClock`/`WithFlightRecorder` options), snapshots, the `system.New` composition root, `metaengine` (cost-based planner over interchangeable engines — the whole engine family promoted EXPERIMENTAL→**Stable**, T27 owner ruling 2026-11-03), scheduling, scenario testing, id/branded-ID, schema evolution, upcasters. cqrs-htmx is, by design, the HTTP/HTMX face of this library. The v5 shape is now staged: [Unreleased] core/v5 type finalization (ADR-0166 — `record.StreamRef`, fused `Record.Payload`, branded `InstanceID`/`Version`/`CorrelationID`, typed `event.Cause()`) plus a `cmd/cqrs-upgrade --to-v5` pass that rewrites it.

## How cqrs-htmx uses it today

- **424 import files** (181 production / 243 test — `-g '!**/*_test.go'` split) across every module — the deepest dependency in the fleet.
- **39 pinned trains** (38 last audit + **deriver/v4 v4.5.0, new 2026-11-16/17**), all verified against `go.mod` files; the only stale pins in-tree are the poisoned `scripts/testdata/verify-tag/` fixtures — which are also the ONLY place the deleted-module pins `codec/v4` and `flightrecorder/v4` still appear (go-cqrs-lite deleted them per ADR-0128; cqrs-lint F030 now carries the seven-entry deleted-module registry): `event/v4 v4.15.2`, `command/v4 v4.14.1`, `query/v4 v4.10.1`, `system/v4 v4.19.0`, `metaengine/v4 v4.26.0`, `middleware/v4 v4.7.2`, `storage/v4 v4.11.1`, `projectionhost/v4 v4.7.1`, `kv/v4 v4.4.0`, `catalog/v4 v4.15.0`, `commandlifecycle/v4 v4.3.1`, `dispatcher/v4 v4.5.2`, `scheduling/v4 v4.7.0`, `snapshot/v4 v4.7.0`, `stack/v4 v4.5.1`, `systemcatalog/v4 v4.0.0`, `dedup/v4 v4.3.0`, `deriver/v4 v4.5.0`, `scenario/v4 v4.4.3`, `watermill/v4 v4.7.0`, `id/v4 v4.7.5`, `record/v4 v4.6.2`, `schema/v4 v4.7.1`, `metadata/v4 v4.7.3`, `listing/v4 v4.4.4`, `decider/v4 v4.8.0`, `otel/v4 v4.5.3`, `projection/v4 v4.4.2`, `storage/memory/v4 v4.6.2`, `metaengine/{projectionadapter v4.6.2, sqliteengine v4.9.1, bboltengine v4.3.3}`, `commandlifecycle/projections/v4 v4.2.3`, `event/v4/eventtest v0.4.0`, `claiming/v4 v4.0.2`, `encryption/v4 v4.4.6`, `prometheus/v4 v4.3.4`, `signing/v4 v4.4.2`, `systemscenario/v4 v4.3.0`.
- Canonical integration map: [docs/guides/leveraging-go-cqrs-lite.md](../../guides/leveraging-go-cqrs-lite.md); runnable proofs in `examples/middleware-demo` and `examples/observability-demo`.
- **deriver — NEW since last audit (T04/T07, 2026-11-16/17; published in systemadapter/v4.14.0 2026-11-21):** deep-dive row 29 ("N/A — purity contract") flips. `subscribeTenantDeletionDeriver` (`systemadapter/tenant_deletion_deriver.go:40`) derives one `RemoveMember` per member on `TenantDeleted` as a bus deriver per go-cqrs-lite [ADR-0154](https://github.com/LarsArtmann/go-cqrs-lite) — `deriver.WithAsyncDispatch` (`:73`), a bounded 5s `awaitProjectionsCaughtUp` wait before querying the membership view (the eventual-consistency race the old objection feared, A/B-proven), mounted at `systemadapter/domain_config.go:109`; the `usermgmt.Service.revokeMembershipsForTenantBestEffort` loop it supersedes stays until consumers ride the train. Second site: the order-saga showcase (`examples/catalog-demo/ordersaga/ordersaga.go:65` — payment.settle derived from order.created). Pinned by a generated `ThenCommands("DeleteTenant","RemoveMember")` scenario in `gfl-usermgmt/usermgmt.gfl` plus `TestTenantDeletionDeriver_ColdBootSeedsRace`.
- **R2 (systemcatalog derivation)** — `systemadapter/architecture_catalog.go:39` derives the full command+event inventory from `DomainConfig()` via `systemcatalog.FromDomainConfig`, with drift gates at `systemadapter/architecture_catalog_test.go:46-58` (21-command + 27-event inventories) and `:64` (parity with the usermgmt published language). It does **not replace** `usermgmt.DefaultEventCatalog()` (`usermgmt/es_event_catalog.go:30`, still the served endpoint via `setup/machine.go:24`): one hand-curated table feeds both wire formats, and the parity test makes silent drift impossible. **T03 (2026-11-17):** `GET /events/catalog` now serves the committed generated artifact `usermgmt/event-catalog.json` byte-identically with ETag/304 (`usermgmt/es_event_catalog_artifact.go`); root grew the generic loader `eventcatalog.NewFromJSON` (`eventcatalog/catalog.go:51`) for generated languages — usermgmt keeps its local parse loop until its published pin rides past it.
- **CBOR (ADR-0059)** — five payloads CBOR-encoded via the shared `cborEventCodec` instance (`usermgmt/es_decide_credentials.go:15`): `CredentialAdded` :50, `CredentialRemoved` :93, `CredentialUsed` :155, `TOTPEnabled` (`es_decide_security.go:32`), `BotRegistered` (`es_bot_decide.go:63`). `dashboardui/core/payload.go:105-118` renders JSON/CBOR/Raw exhaustively (gotcha 55: `event.New` defaults CBOR since event/v4 v4.15.2).
- **QUERY wave (root v4.18.x)** — typed query surface at the cqrs-htmx root: `Config.Queries *query.Dispatcher` (`app.go:54`), `App.Query` (`app.go:270`), `QueryTyped[Q,R]` (`app.go:325`), plus the `cqrs-query` htmx extension (native QUERY with 405→POST downgrade).
- **M2M W0** — `usermgmt.NewAPITokenMiddleware` (`api_token_middleware.go:45`) bridges `ActorIDFromBot` (`:68`) into `cqrshtmx.WithActorID`, composed with `middleware.CommandActorContext()` (`audit_context.go:40`). **NEW: idempotency by default (usermgmt v4.21.0, K2 owner tick 2026-11-19)** — `NewService` mounts a replay guard inside the audit chain that mirrors upstream `middleware.CommandIdempotency`'s `CheckAndRecord` discipline (a wrapper over the upstream factory was tried and REJECTED for sentinel-attribution semantics — see usermgmt CHANGELOG [v4.21.0]); `ServiceConfig.IdempotencyStore`/`DisableCommandIdempotency` are the seams.

## Score derivation

The [2026-11-03 deep dive](../../research/2026-11-03_go-cqrs-lite-deep-dive.html) graded **47 capability rows**: 10 Not-Applicable → 37 applicable; **22 fully leveraged / 10 partially used / 5 missed** = `(22 + 10×0.5)/37 = 73/100` (band: high).

Ten graded rows were resolved by 2026-11-15 (each re-verified against source THIS session — line numbers re-stamped):

| Row (audit grade)                           | Resolution (wave; current receipt)                                                                                                                                                                       |
| ------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| CBOR for binary payloads (missed)           | ADR-0059 — five payloads `event.WithCodec(cborEventCodec)` (2026-11-04)                                                                                                                                    |
| Replay-seam dedup: raw map (partial)        | `dedup.Ring` at `transport/serve.go:193` (seam) / `:539` (`isReplayDuplicate`) (R3, `823fcd7e`)                                                                                                             |
| `kv` FindByID write-through cache (partial) | upstream `kv.NewCacheForViewStore` at `usermgmt/view_cache.go:97` (kv v4.4.0; ask #113 resolved; the 182-line otter adapter is gone) (R9)                                                                   |
| `WithSnapshotStateVersion` (partial)        | `decider.WithSnapshotStateVersion[State]("v1")` in `repositoryOptions` (`usermgmt/snapshot.go:112`) (R10)                                                                                                   |
| systemcatalog derivation (missed)           | R2/T03, see above — derived catalogs + inventory/parity gates + served artifact                                                                                                                             |
| commandlifecycle streams (missed)           | `ServiceConfig.CommandLifecycle` (`usermgmt/service_core.go:234`, factory `service_factory.go:268-274`) + dashboardui Command Fates panel (`handlers_command_fates.go`, `core/command_fates.go`) (R7/F1)   |
| projectionhost DLQ memory-default (partial) | `durableDeadLetterStore` (`usermgmt/es_projection_setup.go:80`, wired `:154`) — durable DLQ by default in the SQL setups (R4)                                                                               |
| Checkpoint batching unused (partial)        | `ProjectionHostOptions []projectionhost.HostOption` threaded setup→usermgmt (`setup/config.go:490`, `usermgmt/service_core.go:189`), `WithCheckpointEvery` documented (`setup/config.go:487`) (R8)          |
| metaengine engine breadth (partial)         | 8 `Recommended*Deployment` presets — memory/sqlite/split/pebble/bbolt/postgres/duckdb/turso (`systemadapter/presets.go:36-165`) via `system.Standard` (ADR-0165)                                             |
| test helpers imported nowhere (missed)      | → **partial**: `eventtest.NewFakeBus` adopted in 4 modules / 10 files (root `setup`, `dashboardui`, `e2e/server`, `examples/dashboard-demo`); `testutil`/`idtest`/`querytest` still unimported               |

**Row flip this refresh: deriver (deep-dive row 29, N/A → 🟢 fully leveraged)** — the 2026-07-30 "purity contract" objection (`ROADMAP.md:76`) is retired by the shipped ADR-0154 wiring above (async dispatch + bounded catch-up wait); per FORMAT's non-adopter rule a citable objection that no longer holds is re-asked, and the answer is now yes. The row leaves the N/A set and enters numerator AND denominator as full.

**Standing: 32 full ×1 + 5 partial ×0.5 + 1 missed ×0 = 34.5 / 38 ≈ 90.8/100** (band: very high). Remaining partial: record structural metadata (the v4 claim stands — no `metadata.Cause` in any v4 release; **the v5 API now exists pre-tag**: [Unreleased] core/v5 `event.Cause()` typed at birth via `WithCause`, ADR-0166), metaengine planner/folds direct use, `system.New` for usermgmt's own runtime (migration plan W1 probes GO), otel/otlp one-call exporter, test helpers. Remaining missed: flightrecorder.

## Open findings (the residual tail)

1. **Version tail — current.** 39/39 trains at latest, worst lag 0 (treadmill 2026-11-22; first-hand tag re-derivation this session). The 11-14 metaengine re-lag class stays dead; the treadmill gate tracks it continuously.
2. **cqrs-lint gate binary — v4.22.2 (was v4.21.0), and the gate is now ESCALATED** (2026-11-17/21, plan B4 lineage): the system binary comes from the `cmd/cqrs-lint` tag train (v4.22.0 per-module truth wave — field-proven on THIS repo: the BuildFlow provider went 143 accumulated findings → the correct per-module 180 → 1 after triage; v4.22.1 ancestor-inheritance fix; v4.22.2 `--scope=module`). The consumer gate now fails on ANY finding, runs every candidate `--scope=module`, and refuses binaries < v4.22.2; green across every configured module, BuildFlow `cqrs-lint` = 0 findings (receipts: root CHANGELOG [v4.19.0] Fixed; `scripts/checks/check-cqrs-lint.sh:83`).
3. **flightrecorder** — still the one missed row; the 2026-11-15 B7 verdict stands (process-global recorder, library cannot default it on; recipe documented in the guide §2.8 at `docs/guides/leveraging-go-cqrs-lite.md:385`). NEW this refresh: the upstream surface GREW — projectionhost gained `WithFlightRecorder` (`go-cqrs-lite/projectionhost/v4/options.go:165`) and `WithClock` (`:210`), and the recorder core now lives in the external extraction `github.com/larsartmann/go-flightrecorder` (v0.2.1, transitively pinned via `middleware/v4` — visible in `legalparty-model/go.mod:33`). Still not wired in-tree; row stays missed ×0 honestly (documented ≠ used).
4. **testutil / idtest / querytest** still unimported (eventtest covers 4 of the modules).
5. **Deprecated `meta.CausationID` reads ×3** — `dashboardui/detail_items.go:166,187,269`; rename at the next family wave. The `record.Cause` ×3 reads were v5-gated on "no v4 API" — that erratum's v5 half is now answered: [Unreleased] core/v5 `event.Cause()`/`WithCause` + `cmd/cqrs-upgrade --to-v5` exist pre-tag (ADR-0166), so the rename target is staged upstream.
6. **v5 exposure tail** — the 68 `SQLViewStore`-family findings: ADR-0051's criterion PARTIALLY MET, migration plan W1 probes GO (W1.2 starvation probe fixed a real preset bug upstream). New since: the whole metaengine engine family is **Stable** (T27 — breaking changes wait for v5; 42 direct + 39 indirect fleet consumers in the 2026-11-15 census), ADR-0167 (Proposed) retires `idempotency/sqlstore` at v5 behind a system-exposed dedup facade, and cqrs-lint F030 is the deleted-module registry (7 entries). Upstream M45 fixes RawQuery-mixed domains' partial decoder — cqrs-htmx's `systemadapter/declarations.go:26-29` declares three `system.RawQuery` projections, squarely in the healed class.
7. **NEW (Minor): `ROADMAP.md:76` deriver row is stale** — still reads "Evaluated 2026-07-30 — Not a fit … violates deriver's purity contract" while `systemadapter/tenant_deletion_deriver.go:40` ships the ADR-0154 deriver that answers exactly that objection. Trigger: this refresh's row flip. Owner: next ROADMAP truth pass (the T04 lane's docs tail).

## Version currency

**39 of 39 trains at latest; worst lag 0.** Verified 2026-11-22: the orchestrator's treadmill run AND a first-hand per-prefix loop over the workspace go.mods against the checkout's tags (`ALIGNED=39 LAGGED=0`, this session). New train since the 11-15 re-true: `deriver/v4 v4.5.0`. Tag waves after 2026-11-14: metaengine v4.25.0/v4.26.0 + sqliteengine v4.9.1 + system v4.19.0 (all 2026-11-15, consumed) and `cmd/cqrs-lint` v4.22.0/.1/.2 (2026-11-17, the gate binary). The 11-14 "metaengine lag 1" → 11-15 sweep → closed; the train-lag tail regrows under you (gotcha 39) — the treadmill gate watches it.

## Should cqrs-htmx use it (more)?

**Yes — and it now is, near the max.** Verdict: very high adoption, 0 anti-patterns, every graded open row from the 2026-11-03 plan of record executed and receipted, version tail fully current, and the last N/A-objection (deriver) flipped by real adoption. Residual: one missed row (flightrecorder — deliberately unwired until an incident-response story exists), the helper-family partials, the owner-gated usermgmt → system/metaengine migration (W2+), and the fresh v5 staged surface (core/v5 types + upgrade tool) that the W-waves will consume.

## How to verify

```bash
# Every command below was run green during the 2026-11-22 audit.
rg -l '"github.com/larsartmann/go-cqrs-lite/' --glob '**/*.go' -g '!scripts/testdata/**' . | wc -l     # 424 files (181 prod / 243 test)
rg -oN 'github\.com/larsartmann/go-cqrs-lite/[a-z0-9/_-]+' --glob 'go.mod' -g '!scripts/testdata/**' . \
  | sed 's|.*/go-cqrs-lite/||' | sort -u | wc -l                                                        # 39 pinned prefixes (deriver/v4 joined 2026-11-16)
ok=0; lag=0
for p in $(rg -oN 'github\.com/larsartmann/go-cqrs-lite/[a-z0-9/_-]+' --glob 'go.mod' -g '!scripts/testdata/**' . | sed 's|.*/go-cqrs-lite/||' | sort -u); do
  pin=$(rg -oN --no-filename "github\.com/larsartmann/go-cqrs-lite/$p v\S+" --glob 'go.mod' -g '!scripts/testdata/**' . | head -1 | awk '{print $NF}')
  if [[ "$p" == */v4 ]]; then tp="${p%/v4}/v"; else tp="$p/v"; fi
  latest=$(git -C /home/lars/projects/go-cqrs-lite tag -l "$tp*" | grep -E "^${tp}[0-9]+\.[0-9]+\.[0-9]+$" | sort -V | tail -1 | sed "s|^$tp|v|")
  if [ "$pin" != "$latest" ]; then echo "LAG: $p pin=$pin latest=$latest"; lag=$((lag+1)); else ok=$((ok+1)); fi
done; echo "ALIGNED=$ok LAGGED=$lag"                                                                    # ALIGNED=39 LAGGED=0
rg -l 'go-cqrs-lite/deriver' --glob '*.go' -g '!scripts/testdata/**' .                                  # systemadapter/tenant_deletion_deriver.go + examples/catalog-demo/ordersaga/ordersaga.go (row 29 flip)
rg -n 'func TestArchitectureCatalog' systemadapter/architecture_catalog_test.go                         # :46 CommandInventory · :50 EventInventory (27) · :64 parity (R2 gates)
rg -c '<tr><td>[0-9]+</td>' docs/research/2026-11-03_go-cqrs-lite-deep-dive.html                        # 47 rows
rg -n 'deriver \(reactive sagas\)' docs/research/2026-11-03_go-cqrs-lite-deep-dive.html                 # :1610 — row 29, the N/A this refresh flips
rg -n 'core/v5 \(pre-tag type finalization' /home/lars/projects/go-cqrs-lite/CHANGELOG.md | head -1     # :32 — [Unreleased] v5 surface (StreamRef, typed Cause)
git -C /home/lars/projects/go-cqrs-lite tag -l 'cmd/cqrs-lint/*' | sort -V | tail -1                    # cmd/cqrs-lint/v4/v4.22.2 — the gate-binary train
```

## Prior reports

- 2026-11-15 (prior state of this file — the version re-true of the 2026-11-14 refresh): 90.5/100 · 38 trains at latest, 0 lag; no row flips, the same-day metaengine re-lag swept.
- [2026-11-03 deep dive](../../research/2026-11-03_go-cqrs-lite-deep-dive.html) — the scored audit this report updates.
- `docs/research/go-cqrs-lite-feature-audit.html` — legacy-era audit (superseded).
- [ADR-0059](../../adr/0059-cbor-binary-event-payloads.md), [ADR-0051](../../adr/), ADR-0165 (cqrs-htmx) · go-cqrs-lite ADR-0154 (deriver), ADR-0166 (core/v5 types), ADR-0167 (idempotency retirement) — decisions gating the open items.
