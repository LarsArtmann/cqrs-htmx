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

## Full Service Setup (Manual)

MySQL does not yet have a compiled convenience constructor (`NewMySQLEventSourcedSetup`). A reference template exists at `usermgmt/mysql_setup.go` (`//go:build ignore`) — copy it into your application and remove the build tag to use it. Until then, wire it manually:

```go
eventStore, _ := usermgmt.NewSQLEventStore(ctx, db, "mysql")

cfg := usermgmt.EventSourcedConfig{
    EventStore: eventStore,
    // Add session store, read models, checkpoint store, etc.
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

## What's Supported vs. What's Not

| Component               | MySQL support   | Notes                                                              |
| ----------------------- | --------------- | ------------------------------------------------------------------ |
| Event store             | ✅ Full         | `NewSQLEventStore(ctx, db, "mysql")`                               |
| Error classification    | ✅ Full         | `classifyMySQLError` in go-cqrs-lite                               |
| Duplicate-key detection | ✅ Full         | Error 1062 detection                                               |
| Session store           | ⚠️ Placeholders | Uses `?` placeholders (MySQL-compatible), but no dedicated dialect |
| Snapshot store          | ⚠️ Manual       | Pass the same `*sql.DB` with `MySQLDialect`                        |
| Checkpoint store        | ⚠️ Manual       | Same as snapshot store                                             |
| Convenience constructor | ❌ Not yet      | `NewMySQLEventSourcedSetup` is planned                             |

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
