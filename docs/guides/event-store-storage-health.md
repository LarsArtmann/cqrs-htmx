# Event Store Storage Health

> Guidance for managing storage growth in an append-only event sourcing system. Events are immutable and permanent — this document explains how to keep the database healthy without ever deleting data.

---

## The Append-Only Principle

**Events are the source of truth. They are never deleted, never truncated, and never archived.** This is the fundamental principle of event sourcing:

- Events are the auditable history of every state change.
- Aggregates rebuild their state by replaying events.
- Projections derive read models from events.
- Deleting events would destroy the ability to rebuild or audit.

This means the events table grows monotonically. Storage growth is a feature, not a bug — it represents accumulated domain history.

---

## Snapshotting: Read Performance Without Data Loss

Snapshots (ADR-0041) solve the read-performance problem of long event streams without deleting events:

- When `SnapshotConfig.Store` + `Codec` + `Strategy` are set, the repository periodically saves aggregate state snapshots.
- On load, the repository reads the snapshot and replays only the events after the snapshot version (`LoadFromVersion`).
- **The full event journal is still intact.** Snapshots are a read optimization, not a data reduction.

```go
svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
    // ... other config ...
    SnapshotConfig: usermgmt.SnapshotConfig{
        Store:    snapshotStore,     // e.g. MemorySnapshotStore (dev) or SQL
        Codec:    snapshotCodec,      // JSON or CBOR
        Strategy: snapshot.EveryN(100), // snapshot every 100 events per aggregate
    },
})
```

**Snapshots do not reduce storage. They reduce read latency for high-volume aggregates.**

---

## Postgres Storage Health

For production deployments using Postgres as the event store backend:

### VACUUM and ANALYZE

Postgres uses MVCC (Multi-Version Concurrency Control). Even though events are append-only, the database creates dead tuples for index updates and transaction visibility tracking. Regular VACUUM reclaims this space:

```sql
-- Regular autovacuum handles this, but for high-write systems:
VACUUM (ANALYZE, VERBOSE) events;

-- For aggressive bloat reclaim without locking:
VACUUM (ANALYZE, VERBOSE, INDEX_CLEANUP = AUTO) events;
```

**Recommended:** Enable autovacuum with aggressive settings for the events table:

```sql
ALTER TABLE events SET (
    autovacuum_vacuum_scale_factor = 0.05,  -- vacuum when 5% tuples are dead
    autovacuum_analyze_scale_factor = 0.02  -- analyze when 2% rows change
);
```

### Partitioning by Time or Aggregate Type

For very high-volume systems, partition the events table:

```sql
CREATE TABLE events (
    -- columns as defined by go-cqrs-lite storage
) PARTITION BY RANGE (created_at);

-- Monthly partitions
CREATE TABLE events_2026_07 PARTITION OF events
    FOR VALUES FROM ('2026-07-01') TO ('2026-08-01');
```

**Partitioning improves query performance** (the journal can scan fewer rows for recent events) and **simplifies storage management** (old partitions can be moved to slower tablespaces without deleting data). The event store still sees one logical table.

### Index Maintenance

```sql
REINDEX INDEX CONCURRENTLY events_pkey;
REINDEX INDEX CONCURRENTLY idx_events_stream_version;
```

---

## SQLite Storage Health

For development, testing, or single-node production deployments:

### WAL Checkpointing

SQLite's WAL (Write-Ahead Log) mode is recommended for concurrent access. The WAL file grows with writes and must be checkpointed:

```sql
PRAGMA wal_checkpoint(PASSIVE);  -- non-blocking, incremental
PRAGMA wal_checkpoint(TRUNCATE); -- blocking, shrinks WAL file
```

**Recommended:** Enable automatic checkpointing with aggressive threshold:

```sql
PRAGMA wal_autocheckpoint = 1000;  -- checkpoint every 1000 pages (default)
```

Use `OptimizeSQLiteDB` from usermgmt to configure these pragmas automatically.

### VACUUM

SQLite's `VACUUM` rebuilds the database file, reclaiming space from deleted rows in OTHER tables (read models that get rebuilt, sessions that expire, etc.). The events table itself is append-only, but other tables in the same database file fragment over time:

```sql
VACUUM;
```

**Note:** `VACUUM` requires free disk space equal to the database size and locks the database. Use `PRAGMA incremental_vacuum` for online maintenance with `PRAGMA auto_vacuum = INCREMENTAL`.

---

## MySQL Storage Health

For production deployments using MySQL/MariaDB as the event store backend:

### OPTIMIZE TABLE

MySQL uses InnoDB by default (ACID-compliant). While the events table is append-only, InnoDB maintains undo logs and indexes that fragment over time. Use `OPTIMIZE TABLE` to rebuild the table and reclaim space:

```sql
OPTIMIZE TABLE events;
```

**Note:** `OPTIMIZE TABLE` on InnoDB creates a temporary table and copies data. It locks the table during the operation. Run during low-traffic periods or use `pt-online-schema-change` for zero-downtime maintenance.

### Connection Pool Tuning

MySQL connections are expensive to establish. Configure the connection pool:

```go
db.SetMaxOpenConns(50)
db.SetMaxIdleConns(10)
db.SetConnMaxLifetime(5 * time.Minute)
```

### Important Connection String Parameters

- `parseTime=true` — **required** (converts `DATETIME` columns to `time.Time`)
- `interpolateParams=true` — small performance gain (client-side parameter interpolation)
- `multiStatements=true` — needed if auto-migration DDL contains multiple statements

### Error Handling

MySQL errors are automatically classified by `classifyMySQLError` in go-cqrs-lite:
- Error 1062 (duplicate entry) → **Conflict** (409)
- Error 1205 (lock wait timeout), 1213 (deadlock) → **Transient** (503, retry)
- Error 2003/2006/2013 (connection errors) → **Transient** (503)

See [MySQL Setup Guide](./mysql-setup.md) for full setup instructions.

---

## Monitoring Storage Growth

Track these metrics:

| Metric                   | Query (Postgres)                                                              | Alert Threshold                                |
| ------------------------ | ----------------------------------------------------------------------------- | ---------------------------------------------- |
| Events table size        | `SELECT pg_size_pretty(pg_total_relation_size('events'));`                    | Plan capacity at 10GB+                         |
| Event count              | `SELECT count(*) FROM events;`                                                | Context-dependent                              |
| Events per day           | `SELECT date_trunc('day', created_at), count(*) FROM events GROUP BY 1;`      | Growth rate trend                              |
| Largest aggregate stream | `SELECT stream_id, count(*) FROM events GROUP BY 1 ORDER BY 2 DESC LIMIT 10;` | >10k events per aggregate → consider snapshots |
| Database bloat ratio     | Compare `pg_stat_user_tables.n_dead_tup` to `n_live_tup`                      | >20% dead tuples → VACUUM                      |

---

## Capacity Planning

For a user management system:

- Average events per user lifetime: ~5-20 (registration, email verification, credential add/remove, role changes)
- Average event size: ~200-500 bytes (JSON payload with metadata)
- Estimate: 10,000 users = ~100,000 events = ~30-50MB

**Events table growth is linear with user activity, not exponential.** Most user management systems will not hit storage problems for years. Monitor the growth rate and provision accordingly.

---

## What NOT to Do

- **Do NOT delete old events.** They are the source of truth.
- **Do NOT truncate the events table.** You lose the ability to rebuild aggregates and projections.
- **Do NOT archive events to cold storage.** Rebuilding requires the full event history.
- **Do NOT add TTL-based event expiration.** Events are permanent by design.
- **Do NOT implement in-library compaction.** The database (Postgres/SQLite) handles its own storage management. cqrs-htmx delegates to go-cqrs-lite's storage layer.

---

## See Also

- [Consistency Model](./consistency-model.md) — Why events must be permanent
- [Event Replay and Rebuild](./event-replay-and-rebuild.md) — How rebuilding uses the full event log
- ADR-0041 — Snapshot configuration for read performance
