# Migrations

SQL migration scripts for `usermgmt` schema changes that require operator action.

## When to use these

These are **not** run automatically. `NewSQLEventStore` and `NewSQLSessionStore`
auto-migrate their tables via `CREATE TABLE IF NOT EXISTS`. These scripts are
only needed when:

- An existing deployment has data in an older schema shape.
- A `ALTER TABLE` migration is required (rename, add column with backfill, etc.).
- The auto-migration cannot reconcile the difference in-place.

## Files

| Script                                                               | Applies to           | When                                                                                                                                                                                               |
| -------------------------------------------------------------------- | -------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [`0001_user_events_to_events.sql`](./0001_user_events_to_events.sql) | Postgres event store | Upgrading usermgmt from `< v2.5.0` to `>= v2.5.0` on a deployment that used the old hand-rolled `SQLEventStore` (which created a `user_events` table). The delegated upstream store uses `events`. |

## Running

Apply with your standard Postgres migration tool (`psql`, `migrate`, `goose`,
`sql-migrate`, schema-as-code pipeline, etc.). All scripts are idempotent and
safe to re-run.

```sh
psql "$DATABASE_URL" -f migrations/0001_user_events_to_events.sql
```

## SQLite

SQLite is treated as ephemeral for the event store (typically `:memory:` or
development-only file databases). No migration script is provided for SQLite;
re-create the database if you need to migrate. For production event sourcing,
use Postgres.

## MySQL

The event store no longer supports MySQL (upstream `go-cqrs-lite/storage` has
no MySQL dialect). `SQLSessionStore` retains MySQL support (it manages its
own schema). MySQL event-store consumers must export events to JSON and
re-import on Postgres.
