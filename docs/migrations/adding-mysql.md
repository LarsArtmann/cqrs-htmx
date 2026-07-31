# Adding MySQL Support

This guide walks through migrating a cqrs-htmx deployment from PostgreSQL or SQLite to MySQL/MariaDB.

## Prerequisites

- MySQL 8.0+ or MariaDB 10.6+
- `github.com/go-sql-driver/mysql` driver in your application

```bash
go get github.com/go-sql-driver/mysql
```

## Step 1: Update Your Connection String

MySQL connection strings use a different format from Postgres:

```go
// Before (Postgres):
dsn := "host=localhost user=app dbname=cqrs_htmx sslmode=disable"

// After (MySQL):
dsn := "app:password@tcp(localhost:3306)/cqrs_htmx?parseTime=true"
```

**Critical:** `parseTime=true` is required. Without it, `DATETIME` columns won't convert to Go `time.Time`.

## Step 2: Switch Event Store Dialect

```go
// Before:
eventStore, _ := usermgmt.NewSQLEventStore(ctx, db, "postgres")

// After:
eventStore, _ := usermgmt.NewSQLEventStore(ctx, db, "mysql")
```

## Step 3: Switch Read Models

```go
// Before:
userRm, _ := usermgmt.NewSQLUserReadModel(db)

// After:
userRm, _ := usermgmt.NewMySQLUserReadModel(db)
memRm, _ := usermgmt.NewMySQLMembershipReadModel(db)
tenRm, _ := usermgmt.NewMySQLTenantReadModel(db)
botRm, _ := usermgmt.NewMySQLBotReadModel(db)
```

## Step 4: Session Store

The SQL session store already detects MySQL from the dialect string and generates MySQL-compatible DDL automatically. No changes needed beyond passing `"mysql"` as the dialect.

## Step 5: Data Migration

If migrating from an existing PostgreSQL deployment:

1. Export events from Postgres:
```sql
COPY (SELECT * FROM events ORDER BY stream_id, version) TO '/tmp/events.csv' CSV HEADER;
```

2. Create the events table in MySQL (auto-migrates on first `NewSQLEventStore` call)

3. Import the data using `LOAD DATA INFILE` or your preferred tool

4. Verify event count matches

## Key Differences from PostgreSQL

| Feature | PostgreSQL | MySQL |
| ------- | ---------- | ----- |
| Placeholder | `$1, $2` | `?` |
| Upsert | `ON CONFLICT ... DO NOTHING` | `ON DUPLICATE KEY UPDATE col = col` |
| Binary type | `BYTEA` | `LONGBLOB` |
| Timestamp | `TIMESTAMPTZ` | `DATETIME(3)` |
| Identifier quoting | `"column"` | `` `column` `` |

## See Also

- [MySQL Setup Guide](../guides/mysql-setup.md) — Full setup reference
- [Event Store Storage Health](../guides/event-store-storage-health.md) — MySQL maintenance tips
- `go-cqrs-lite/storage/sql/dialect.go` — `MySQLDialect` implementation
