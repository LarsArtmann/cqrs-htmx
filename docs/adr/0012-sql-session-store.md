# ADR 0012: SQL SessionStore

**Date:** 2026-06-18
**Status:** Accepted

## Context

`usermgmt` ships `InMemorySessionStore` for development and testing. Production
deployments need persistent session storage that survives process restarts and
works across multiple instances. The existing `SQLEventStore` (ADR 0006) already
established the pattern for SQL-backed persistence using `database/sql` (stdlib)
with dialect-aware DDL.

The `SessionStore` interface has four methods:

```go
type SessionStore interface {
    Create(ctx, userID, ttl) (*Session, error)
    Find(ctx, token) (*Session, error)
    Delete(ctx, token) error
    DeleteByUserID(ctx, userID) error
}
```

## Decision

Ship `SQLSessionStore` in the `usermgmt` package, mirroring `SQLEventStore`:

1. **`database/sql` only** — no driver import. Consumers register their own
   driver (`pgx`, `lib/pq`, `modernc.org/sqlite`, etc.) in `main.go`.
2. **Multi-dialect** — Postgres, SQLite, MySQL with correct placeholder syntax
   (`$N` vs `?`) and type-specific DDL.
3. **Auto-migration** — `NewSQLSessionStore` creates the `user_sessions` table
   on first use.
4. **Background sweeper** — `StartCleanupSweeper(ctx, interval)` periodically
   evicts expired sessions.
5. **Contract tests** — `runSessionStoreContract` verifies both
   `InMemorySessionStore` and `SQLSessionStore` have identical semantics.

## Why `database/sql` (not a driver)?

Same rationale as `SQLEventStore`: `database/sql` is stdlib and driver-agnostic.
Adding a driver dependency would violate the library principle: "never enforce
defaults that consumers might disagree with." Consumers already choose their
own Casbin enforcer, CQRS dispatcher, and HTTP router — the database driver is
no different.

## Schema

```sql
CREATE TABLE user_sessions (
    token       TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL
);
CREATE INDEX idx_user_sessions_user   ON user_sessions (user_id);
CREATE INDEX idx_user_sessions_expires ON user_sessions (expires_at);
```

- `token` — the opaque session token (base64, 32 random bytes).
- `user_id` — `UserID.Get()` (string-backed branded ID).
- No FK to users table — users are event-sourced aggregates, not rows.

## Consequences

**Positive:**

- Production-ready session persistence with zero driver dependencies.
- Same multi-dialect pattern as `SQLEventStore` — familiar to consumers.
- Contract tests ensure behavioral parity between in-memory and SQL stores.
- Background sweeper prevents unbounded table growth.

**Negative:**

- `Find` returns expired sessions — callers must check `Session.IsExpired()`.
  This matches `InMemorySessionStore` semantics (the sweeper is lazy, not eager).

## Wiring

```go
db, _ := sql.Open("pgx", os.Getenv("DATABASE_URL"))
sessionStore, _ := usermgmt.NewSQLSessionStore(ctx, db, "postgres")
stop := sessionStore.StartCleanupSweeper(ctx, 5*time.Minute)
defer stop()

svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{
    SessionStore: sessionStore,
    EventStore:   eventStore, // SQLEventStore or memory.NewMemoryStore
    // ...
})
```
