-- Migration 0001: user_events → events
--
-- Migrates the pre-v2.5.0 user_events table (hand-rolled in usermgmt) to the
-- events table schema used by go-cqrs-lite/storage/v2 (the delegated store).
--
-- Required when upgrading from cqrs-htmx/usermgmt < v2.5.0 to >= v2.5.0
-- on a Postgres deployment with existing event data.
--
-- NOT required for:
--   * Fresh deployments (NewSQLEventStore creates the events table directly)
--   * SQLite deployments using :memory: or ephemeral databases
--   * Consumers who never used the old hand-rolled SQLEventStore
--
-- Postgres only. MySQL was supported by the old store but is NOT supported
-- by the new delegated store (upstream go-cqrs-lite/storage has no MySQL
-- dialect). MySQL consumers must export events and re-import on Postgres.
--
-- Idempotent: safe to run multiple times. The existence check on
-- information_schema.tables guards the rename; ADD COLUMN IF NOT EXISTS
-- guards the new columns.

-- 1. Rename user_events → events (only if the old table exists).
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_schema = current_schema() AND table_name = 'user_events'
    ) THEN
        ALTER TABLE user_events RENAME TO events;
    END IF;
END $$;

-- 2. Rename event_id → id (only if the old column name exists).
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'events'
          AND column_name = 'event_id'
    ) THEN
        ALTER TABLE events RENAME COLUMN event_id TO id;
    END IF;
END $$;

-- 3. Add new columns required by the upstream schema.
--    schema_version: tracks payload schema evolution (upcaster support).
--    payload_encoding: 'json' (default) or 'msgpack' — how payload BYTEA is decoded.
--    created_at: insertion timestamp for operational queries (defaults to occurred_at for legacy rows).
ALTER TABLE events ADD COLUMN IF NOT EXISTS schema_version INTEGER NOT NULL DEFAULT 1;
ALTER TABLE events ADD COLUMN IF NOT EXISTS payload_encoding TEXT NOT NULL DEFAULT 'json';
ALTER TABLE events ADD COLUMN IF NOT EXISTS created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW();

-- 4. Backfill created_at from occurred_at for legacy rows that have NULL or default created_at.
--    Runs only once: subsequent runs find no rows matching the WHERE clause.
UPDATE events
SET created_at = occurred_at
WHERE created_at = NOW() AND occurred_at <> NOW();

-- 5. Add the upstream indexes (idempotent — IF NOT EXISTS).
CREATE INDEX IF NOT EXISTS idx_events_aggregate ON events(aggregate_type, aggregate_id);
CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type);
CREATE INDEX IF NOT EXISTS idx_events_occurred_at ON events(occurred_at);
CREATE INDEX IF NOT EXISTS idx_events_agg_time ON events(aggregate_type, aggregate_id, occurred_at);

-- 6. Drop the old user_events indexes (renamed by ALTER TABLE but may have stale names).
--    Postgres renames indexes with the table, but explicit cleanup handles any orphans.
DROP INDEX IF EXISTS idx_user_events_agg;
DROP INDEX IF EXISTS idx_user_events_time;

-- Verification (run manually after migration):
--   SELECT column_name, data_type FROM information_schema.columns
--   WHERE table_name = 'events' ORDER BY ordinal_position;
--
-- Expected columns: id, event_type, aggregate_type, aggregate_id, version,
--   payload, metadata, occurred_at, schema_version, payload_encoding, created_at
