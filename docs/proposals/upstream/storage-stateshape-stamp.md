# Upstream draft: go-cqrs-lite/storage — SQL snapshot store drops the StateShape stamp

Target: github.com/LarsArtmann/go-cqrs-lite/storage v4.11.1 (newest published
tag — re-verified 2026-10-31 via ls-remote after the 2026-10-25 audit).

## Finding (verified at source level, twice)

`SQLiteSnapshotSchema()` persists only `(stream_type, stream_id, version,
state, created_at)`. A snapshot's `StateShape` stamp — the wire-visible
version marker the memory store round-trips — is silently dropped on
Save/Load, degrading stamped snapshots to unstamped across every restart for
SQL consumers (schema-evolution guards can no longer detect a stale shape).

Pinned consumer-side by
`integration_test/s46_s49_seams_test.go:TestS49_SQLSnapshotStore_DropsStateShape_UpstreamGap`
(the assertion flips to `Equal("user@v2")` in the family-bump change that
adopts the fix).

## Ask

Add a `state_shape` column (nullable for pre-stamp rows) to the snapshot
schemas (SQLite/Postgres/MySQL) and carry it through Save/Load, with an
upcaster-friendly migration note: old rows read as shapeless, which is
exactly what they are.

## Why not consumer-side

The column belongs to the store's schema; a consumer-side side-table would
split the snapshot's atomic write across two stores.
