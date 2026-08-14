# MySQL Event Store Setup

cqrs-htmx supports MySQL/MariaDB as a backing store for the event-sourced user management module. This guide covers setup, configuration, and current limitations.

## Prerequisites

- MySQL 8.0+ or MariaDB 10.6+
- `github.com/go-sql-driver/mysql` driver in your application

```go
import _ "github.com/go-sql-driver/mysql"
```

## Event Store Setup

Use `usermgmt.NewSQLEventStore` with the `"mysql"` dialect string. The function auto-migrates the events table on first call:

```go
db, err := sql.Open("mysql", "user:password@tcp(localhost:3306)/cqrs_htmx?parseTime=true")
if err != nil { /* ... */ }

eventStore, err := usermgmt.NewSQLEventStore(ctx, db, "mysql")
if err != nil { /* ... */ }
```

### What happens under the hood

The `"mysql"` dialect maps to `go-cqrs-lite`'s `MySQLDialect`, which provides:

| Feature                 | MySQL-specific behavior                                                      |
| ----------------------- | ---------------------------------------------------------------------------- |
| Placeholders            | `?` (not `$1`, `$2`)                                                         |
| Schema                  | `LONGBLOB` for payload, `JSON` for metadata, `DATETIME(3)` for timestamps    |
| Upsert                  | `ON DUPLICATE KEY UPDATE col = col` (self-assign no-op)                      |
| Identifier quoting      | Backticks (`` `stream_id` ``)                                                |
| Duplicate-key detection | MySQL error 1062 + `"Duplicate entry"` string fallback                       |
| Error classification    | 1062→Conflict, 1205/1213/2003/2006/2013→Transient (via `classifyMySQLError`) |

The events table DDL is auto-created by `EventSchema()`. Indexes use inline `KEY`/`UNIQUE KEY` syntax.

## Full Service Setup

Like the SQLite and Postgres setups, the MySQL convenience constructor ships as a copy-in template: `usermgmt/mysql_setup.go` and `usermgmt/sql_setup_shared.go` (both `//go:build ignore`) — copy **both** files into your application and remove the build tags. The templates are excluded from the build only because they import `go-cqrs-lite/stack/mysql`, which would force the MySQL driver into every consumer's dependency graph. They are compile-verified by `nix run .#check-templates`.

`NewMySQLEventSourcedSetup` creates the full stack (event store, bus, repositories, read models, Casbin, projection host) and additionally:

- **defaults `CheckpointStore` to `NewMySQLCheckpointStore(db)`** — projection positions survive restarts (pass your own `CheckpointStore` to override),
- **creates a MySQL-backed session store** when `CreateSessionStore: true`, exposed as `setup.SessionStore` (pass it to `ServiceConfig.SessionStore`).

```go
import (
    _ "github.com/go-sql-driver/mysql"
    usermgmt "github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

setup, err := usermgmt.NewMySQLEventSourcedSetup(usermgmt.MySQLSetupConfig{
    DSN:                "user:password@tcp(localhost:3306)/cqrs_htmx?parseTime=true",
    CreateSessionStore: true,
})
if err != nil { /* ... */ }
defer setup.Close()

// setup now exposes everything a custom app needs:
//   setup.UserRepository / MembershipRepository / TenantRepository / BotRepository
//   setup.ReadModel / MembershipReadModel / TenantReadModel / BotReadModel
//   setup.Authz(), setup.DB, setup.SessionStore
```

To build the standard `*usermgmt.Service` instead, pass the stores directly
(`ServiceConfig` builds its own repositories and projection host):

```go
db, _ := sql.Open("mysql", "user:password@tcp(localhost:3306)/cqrs_htmx?parseTime=true")
eventStore, _ := usermgmt.NewSQLEventStore(ctx, db, "mysql")
sessionStore, _ := usermgmt.NewSQLSessionStore(ctx, db, "mysql")
cpStore, _ := usermgmt.NewMySQLCheckpointStore(db)

svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
    EventStore:      eventStore,
    ReadModelDB:     db,
    ReadModelDialect: "mysql", // MySQL read models (default "" = SQLite)
    SessionStore:    sessionStore,
    CheckpointStore: cpStore,
})
```

### Manual wiring

If you prefer to wire it yourself (or need event signing/encryption, which the stack preset does not expose):

```go
eventStore, _ := usermgmt.NewSQLEventStore(ctx, db, "mysql")
cpStore, _ := usermgmt.NewMySQLCheckpointStore(db)

cfg := usermgmt.EventSourcedConfig{
    EventStore:      eventStore,
    CheckpointStore: cpStore,
    // Add session store, read models, etc.
}

setup, err := usermgmt.NewEventSourcedSetup(ctx, cfg)
```

### MySQL Read Models

MySQL-specific read model constructors are available:

```go
userRm, _ := usermgmt.NewMySQLUserReadModel(db)
memRm, _ := usermgmt.NewMySQLMembershipReadModel(db)
tenRm, _ := usermgmt.NewMySQLTenantReadModel(db)
botRm, _ := usermgmt.NewMySQLBotReadModel(db)
```

These use `MySQLDialect{}` internally for correct `?` placeholders, `ON DUPLICATE KEY UPDATE` upsert, and backtick identifier quoting.

### Sessions, Snapshots, and Checkpoints

All three auxiliary stores have compiled MySQL-aware constructors:

```go
// Sessions — dedicated MySQL DDL (VARCHAR(255) keys, DATETIME(3), inline indexes)
sessionStore, _ := usermgmt.NewSQLSessionStore(ctx, db, "mysql")

// Snapshots — pair with SnapshotConfig.Store
cpStore, _ := usermgmt.NewMySQLCheckpointStore(db)
snapStore, _ := usermgmt.NewMySQLSnapshotStore(db)
strategy, _ := snapshot.EveryNEvents(500)
cfg.SnapshotConfig = usermgmt.SnapshotConfig{
    Store:    snapStore,
    Codec:    codec.JSONCodec{},
    Strategy: strategy,
}
```

## What's Supported

| Component               | MySQL support | Notes                                                                 |
| ----------------------- | ------------- | --------------------------------------------------------------------- |
| Event store             | ✅ Full       | `NewSQLEventStore(ctx, db, "mysql")`                                  |
| Error classification    | ✅ Full       | `classifyMySQLError` in go-cqrs-lite                                  |
| Duplicate-key detection | ✅ Full       | Error 1062 detection                                                  |
| Session store           | ✅ Full       | `NewSQLSessionStore(ctx, db, "mysql")` — dedicated MySQL DDL          |
| Snapshot store          | ✅ Full       | `NewMySQLSnapshotStore(db)` — MySQL-dialect schema                    |
| Checkpoint store        | ✅ Full       | `NewMySQLCheckpointStore(db)` — MySQL-dialect schema                  |
| Convenience constructor | ✅ Template   | `NewMySQLEventSourcedSetup` — copy-in template (like SQLite/Postgres) |
| MySQL read models       | ✅ Full       | `NewMySQL{User,Membership,Tenant,Bot}ReadModel(db)`                   |

## Connection String Tips

```
user:password@tcp(host:3306)/dbname?parseTime=true&multiStatements=true
```

- `parseTime=true` is **required** — converts MySQL `DATETIME` columns to Go `time.Time`.
- `multiStatements=true` allows the auto-migration DDL to run (if it contains multiple statements).
- Use `interpolateParams=true` for a small performance gain (at the cost of larger packet sizes).

## See Also

- [Event Store Storage Health](./event-store-storage-health.md) — Health checks for SQL backends
- [Event Replay and Rebuild](./event-replay-and-rebuild.md) — Projection recovery
- [Consistency Model](./consistency-model.md) — Read-after-write guarantees
- `go-cqrs-lite/storage/sql/dialect.go` — `MySQLDialect` implementation
