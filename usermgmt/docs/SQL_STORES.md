# SQL Store Reference for usermgmt

**Status:** `SQLSessionStore` shipped in-package. `SQLEventStore` shipped in-package.
**See:** [ADR 0003](../../docs/adr/0003-numeric-ids-sql-stores.md) for ID-type strategy.
**See:** [ADR 0006](../../docs/adr/0006-event-sourced-user-aggregate.md) for event-sourced architecture.
**See:** [ADR 0012](../../docs/adr/0012-sql-session-store.md) for SQLSessionStore design.

The `usermgmt` package ships two SQL-backed stores using `database/sql` (stdlib):

| Store             | Implements                      | Dialects                | File                   |
| ----------------- | ------------------------------- | ----------------------- | ---------------------- |
| `SQLEventStore`   | `event.Store` + `event.Journal` | Postgres, SQLite, MySQL | `sql_event_store.go`   |
| `SQLSessionStore` | `SessionStore`                  | Postgres, SQLite, MySQL | `sql_session_store.go` |

Both auto-migrate their tables on construction and work with any
`database/sql`-compatible driver. The library does **not** import any driver —
consumers register their own (`pgx`, `modernc.org/sqlite`, etc.) in `main.go`.

## Why `database/sql` (not a driver)?

`database/sql` is stdlib. Adding a driver dependency would violate the library
principle: "never enforce defaults that consumers might disagree with." Consumers
already choose their own Casbin enforcer, CQRS dispatcher, and HTTP router — the
database driver is no different.

## SQLEventStore

Stores event-sourced User aggregate events. Used as the write model for the
Decider pattern.

```sql
CREATE TABLE user_events (
    event_id TEXT PRIMARY KEY,
    event_type TEXT NOT NULL,
    aggregate_id TEXT NOT NULL,
    aggregate_type TEXT NOT NULL,
    version INTEGER NOT NULL,
    payload BYTEA NOT NULL,
    metadata JSONB,
    occurred_at TIMESTAMPTZ NOT NULL
);
```

### Usage

```go
db, _ := sql.Open("pgx", os.Getenv("DATABASE_URL"))
eventStore, _ := usermgmt.NewSQLEventStore(ctx, db, "postgres")
svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{
    EventStore: eventStore,
})
```

## SQLSessionStore

Stores session tokens with expiry. Includes a background cleanup sweeper.

```sql
CREATE TABLE user_sessions (
    token       TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL
);
CREATE INDEX idx_user_sessions_user    ON user_sessions (user_id);
CREATE INDEX idx_user_sessions_expires ON user_sessions (expires_at);
```

### Usage

```go
db, _ := sql.Open("pgx", os.Getenv("DATABASE_URL"))
sessionStore, _ := usermgmt.NewSQLSessionStore(ctx, db, "postgres")
stop := sessionStore.StartCleanupSweeper(ctx, 5*time.Minute)
defer stop()

svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{
    SessionStore: sessionStore,
    EventStore:   eventStore,
})
```

### Cleanup

`EvictExpired(ctx)` removes expired sessions and returns the count. Call it
directly for one-off cleanup, or use `StartCleanupSweeper` for periodic
background eviction.

## Full Wiring Example

```go
func main() {
    db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    eventStore, err := usermgmt.NewSQLEventStore(ctx, db, "postgres")
    if err != nil {
        log.Fatal(err)
    }
    defer eventStore.Close()

    sessionStore, err := usermgmt.NewSQLSessionStore(ctx, db, "postgres")
    if err != nil {
        log.Fatal(err)
    }
    defer sessionStore.Close()

    stopSweeper := sessionStore.StartCleanupSweeper(ctx, 5*time.Minute)
    defer stopSweeper()

    svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
        EventStore:   eventStore,
        SessionStore: sessionStore,
        // ... config, hooks, etc.
    })
    if err != nil {
        log.Fatal(err)
    }
    // ... wire HTTP handlers
}
```

## Contract Tests

Both `InMemorySessionStore` and `SQLSessionStore` pass the same
`runSessionStoreContract` test suite, ensuring identical behavioral semantics.
Any future `SessionStore` implementation can reuse the same contract tests.

## Migration Strategy

- Versioned migrations via `golang-migrate/migrate`, `pressly/goose`, or
  hand-rolled migration files.
- The auto-migration in `NewSQLEventStore`/`NewSQLSessionStore` uses
  `CREATE TABLE IF NOT EXISTS` — safe for initial setup but not for schema
  evolution. Use a migration tool for production schema changes.
- For test isolation: use SQLite in-memory (`sql.Open("sqlite", ":memory:")`).
  Tests in this repo already follow this pattern.
