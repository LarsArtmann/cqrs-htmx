> [!NOTE]
> This filing was drafted by GLM-5.3 via Crush from an AI-run investigation, not at my request. When the failure traces to an external report, that source is linked in the body.
>
> - [ ] MANUALLY REVIEWED by `@Lars Artmann` at `[<date-time>]`

## TL;DR

`NewSQLSnapshotStoreWithDialect` (storage/v4 v4.11.1, `eventstore/snapshot.go`) drops `Snapshot.StateShape` on the floor — the schema has no `state_shape` column, the INSERT does not bind it, and the scan does not read it. A snapshot written with a shape stamp therefore loads back UNSTAMPED, and the decider's "absent = accept" rule (`decider/v4` snapshot_load) silently admits it — exactly the stale-shaped snapshot `WithSnapshotStateVersion` exists to discard. Ask: persist the stamp (column + INSERT + scan) behind the usual migration, the way `Encoding` already round-trips.

## Problem

`WithSnapshotStateVersion[State]("v2")` guards against loading a snapshot whose `State` was shaped by an older code version: on save the snapshot is stamped (`snapshot_save.go` sets `snap.StateShape = r.snapshotStateShape`), on load a mismatched non-empty stamp discards the snapshot and rebuilds from the journal. The contract only works if the stamp SURVIVES the store.

The in-memory store round-trips it. The SQL store does not (storage/v4@v4.11.1, `eventstore/snapshot.go`):

- `SnapshotSchema()` (both dialects): columns are stream identity + version + state + encoding + created_at — no `state_shape`.
- `Save`: the INSERT binds no stamp.
- `Load`/`LoadAtVersion`: the scan reads no stamp, so every loaded snapshot has `StateShape == ""`.

## Repro

Pinned as an executable test in cqrs-htmx (`usermgmt/snapshot_shape_sql_test.go`, `TestSnapshotShape_SQLStoreDropsStampToday`): SQLite in-memory + `SQLiteDialect{}`, save `Snapshot{StateShape: "v1", ...}`, load it back, assert the stamp. It asserts `StateShape == ""` today (documenting the drop, with the in-memory store as the passing control) and is written to FAIL the moment upstream persists the stamp, so the cqrs-htmx side flips to asserting preservation in the same change as the dependency bump.

## Impact

Silent, restart-timed: a consumer stamps snapshots, runs on SQL, later bumps its state shape version — and every pre-bump snapshot that should have been discarded and rebuilt from the journal is instead accepted as "legacy unstamped". First observable symptom is a decode error or, worse, subtly wrong state. cqrs-htmx's own `snapshotStateShapeVersion` stamping (`usermgmt/snapshot.go`, R10 of the 2026-11-03 leverage plan) is affected on every SQL deployment.

## Suggested fix

Mirror `Encoding`: add `state_shape TEXT NOT NULL DEFAULT ''`, bind it in the INSERT, scan it on load, behind the same auto-migration pattern the snapshots table already uses. Pre-change rows keep `''` (absent = accept), which is the documented semantics for unstamped snapshots — so the migration is backward-safe and the guard starts working from the first stamped write.
