# go-cqrs-lite Utilization Report

> Audit date: 2026-11-05 · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/go-cqrs-lite/**` (multi-module, per-module version trains) · Checkout: `/home/lars/projects/go-cqrs-lite`

## Verdict at a glance

| Pin(s) | Latest (verified 2026-11-05 via `git ls-remote`) | Prod import sites | Test import sites | Adoption | Score |
| --- | --- | --- | --- | --- | --- |
| 15 trains (see below) | **all 15 = latest, lag 0** | 159 files | 204 files | **Used — the foundation** | **73/100 audited → ~80 derived standing** |

## What the library is

The composable CQRS + event-sourcing foundation under every cqrs-htmx module: deciders, command/query dispatch (with a middleware factory family), event journals, projections (projectionhost: checkpoints, DLQ, crash-restart), snapshots, the `system.New` composition root, `metaengine` (cost-based planner over interchangeable engines), scheduling, scenario testing, id/branded-ID, schema evolution, upcasters. cqrs-htmx is, by design, the HTTP/HTMX face of this library.

## How cqrs-htmx uses it today

- **159 production files** across every module import it — the deepest dependency in the fleet.
- Pinned trains (all = latest tag): `command/v4 v4.14.1`, `event/v4 v4.15.2`, `id/v4 v4.7.4`, `query/v4 v4.10.1`, `storage/v4 v4.11.1`, `storage/memory/v4 v4.6.2`, `middleware/v4 v4.7.2`, `projectionhost/v4 v4.7.1`, `system/v4 v4.16.0`, `metaengine/v4 v4.20.0`, `decider/v4 v4.8.0`, `dispatcher/v4 v4.5.2`, `metadata/v4 v4.7.3`, `record/v4 v4.6.2`, `snapshot/v4 v4.7.0` (+ `dedup/v4 v4.3.0`, `kv/v4 v4.3.5`, `listing/v4 v4.4.4`, `scenario/v4 v4.4.3`, `commandlifecycle/v4 v4.3.1`, `schema/v4 v4.7.1`, `otel/v4 v4.5.3`, …).
- Canonical integration map: [docs/guides/leveraging-go-cqrs-lite.md](../../guides/leveraging-go-cqrs-lite.md) (58 modules catalogued); runnable proofs in `examples/middleware-demo` and `examples/observability-demo`.

## Score derivation (from the 2026-11-03 audit, updated with landed waves)

The [2026-11-03 deep dive](../../research/2026-11-03_go-cqrs-lite-deep-dive.html) graded **47 capability rows**: 10 Not-Applicable → 37 applicable; **22 fully leveraged / 10 partially used / 5 missed / 0 misused** = `(22 + 10×0.5)/37 = 73/100` (band: high).

Waves landed **after** the audit (2026-11-04, plan of record [docs/planning/2026-11-03_20-02_SUPERB-gocqrslite-leverage-maximization-pareto-plan.md](../../planning/2026-11-03_20-02_SUPERB-gocqrslite-leverage-maximization-pareto-plan.md)) resolved four graded rows:

| Row (audit grade) | Resolution |
| --- | --- |
| CBOR for binary payloads (missed) | **ADR-0059 accepted 2026-11-03** — five payloads (`CredentialAdded`, `CredentialRemoved`, `CredentialUsed`, `TOTPEnabled`, `BotRegistered`) now `codec.CBORCodec` |
| Replay-seam dedup: raw map (partial) | `dedup.Ring` at `transport/serve.go` (R3, commit `823fcd7e`) |
| `kv.Cache` FindByID write-through (partial) | Opt-in view cache over `maypok86/otter` v2 on the `kv.ViewStore` interface (R9; upstream ask go-cqrs-lite#113) |
| `WithSnapshotStateVersion` (partial) | Snapshot shape guard wired in `repositoryOptions` (R10) |

**Derived standing: (26 full + 7 partial + 0.5×?) → 26×1 + 7×0.5 = 29.5 / 37 ≈ 80/100** — derived from the audited table plus the four resolutions above, not a full re-audit.

## Open findings (still not "to the max")

1. **systemcatalog-derived event catalog** (top opportunity, impact×ease 16): `usermgmt.DefaultEventCatalog()` is hand-maintained (25 events); derive via `systemcatalog.FromDomainConfig` + a drift gate. Blocked upstream: systemcatalog has no published v4 tag yet.
2. **commandlifecycle streams**: dashboards audit event records, not command fates (Received/Rejected/Failed/Retried/DeadLettered/Completed; FailureLog/RetryCount projections).
3. **SQLite DLQ default for SQL setups** — projection DLQ defaults to memory; poison events are lost on restart (risk finding).
4. **Checkpoint batching** (`WithCheckpointEvery`/`WithCheckpointInterval`) unused.
5. **Engine breadth unexposed**: metaengine presets stop at memory/sqlite/split; pebble/bbolt/badger/postgres/duckdb/turso/bigtable/dgraph available behind `systemadapter`.
6. **v5 exposure tail**: 68 gated findings on v5-removed `SQLViewStore` APIs, cluster-1 behind ADR-0051's criterion.
7. Deprecated `meta.CausationID` reads at 3 dashboardui sites (rename to the non-deprecated accessor at the next family wave).

## Version currency

**0 versions behind on all 15 pinned trains** (re-verified 2026-11-05, `git ls-remote --tags`). The repo's own treadmill gate (`nix run .#check-upstream-treadmill`) tracks this continuously.

## Should cqrs-htmx use it (more)?

**Yes — and it does, structurally.** Verdict: high adoption, 0 anti-patterns, honest open tail tracked in the SUPERB plan. The remaining gap to "max" is the systemcatalog tag ask (upstream) plus the four wiring items above.

## How to verify

```bash
rg -oN '"github.com/larsartmann/go-cqrs-lite/[^"]+"' --glob '**/*.go' . | sort | uniq -c | sort -rn | head
for m in command event id query storage middleware projectionhost system metaengine; do
  git -C ~/projects/go-cqrs-lite ls-remote --tags origin | grep -o "$m/v4[^{}]*$" | sort -V | tail -1
done
rg -c '<tr><td>[0-9]+</td>' docs/research/2026-11-03_go-cqrs-lite-deep-dive.html   # 47 rows
nix run .#check-upstream-treadmill
```

## Prior reports

- [2026-11-03 deep dive](../../research/2026-11-03_go-cqrs-lite-deep-dive.html) — the scored audit this report updates.
- `docs/research/go-cqrs-lite-feature-audit.html` — legacy-era audit (superseded).
- [ADR-0059](../../adr/0059-cbor-binary-event-payloads.md), [ADR-0051](../../adr/), [ADR-0012 series] — decisions gating the open items.
