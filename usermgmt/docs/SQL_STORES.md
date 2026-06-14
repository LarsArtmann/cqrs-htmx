# SQL Store Reference for usermgmt

**Status:** Pattern documented. No implementation shipped in this repo.
**See:** [ADR 0003](../adr/0003-numeric-ids-sql-stores.md) for the ID-type decision.

The `usermgmt` package exposes two persistence interfaces — `UserStore` and
`SessionStore` — that any backend can implement. This document describes a
recommended SQL schema and adapter-pattern approach for consumers who need
production persistence.

## Why a separate package?

The `usermgmt` module intentionally avoids pulling in a SQL driver
(`database/sql` is stdlib, but drivers like `pgx`, `mysql`, `lib/pq` are not).
The library principle states: **never enforce defaults that consumers might
disagree with** — and a database choice is the most opinionated dependency of all.

A consumer-owned `usermgmtdb` (or similarly named) package implements
`UserStore`/`SessionStore` and is constructed in `main.go`. This is the same
pattern used by Casbin (the enforcer), go-cqrs-lite (the dispatchers), and the
HTMX-CSRF pairing.

## Recommended Postgres schema

```sql
-- Users: numeric surrogate PK (see ADR 0003), string ID at the API boundary.
CREATE TABLE users (
    id              BIGSERIAL PRIMARY KEY,
    public_id       TEXT NOT NULL UNIQUE,                 -- usermgmt.UserID.Get()
    email           TEXT NOT NULL UNIQUE,
    display_name    TEXT NOT NULL,
    password_hash   TEXT NOT NULL,                        -- bcrypt
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_users_email ON users(lower(email));

-- Roles: many-to-many, simple text array works for small role sets.
CREATE TABLE user_roles (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role    TEXT NOT NULL,
    PRIMARY KEY (user_id, role)
);

-- Sessions: opaque token, FK to user, expiry index.
CREATE TABLE sessions (
    token       TEXT PRIMARY KEY,                          -- the session.Token
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
```

## Adapter skeleton

```go
package usermgmtdb

import (
    "context"
    "database/sql"
    "errors"
    "time"

    "github.com/larsartmann/cqrs-htmx/usermgmt/v2"
)

// PGUserStore implements usermgmt.UserStore backed by Postgres.
type PGUserStore struct {
    db *sql.DB
}

func NewPGUserStore(db *sql.DB) *PGUserStore { return &PGUserStore{db: db} }

func (s *PGUserStore) FindByID(ctx context.Context, id usermgmt.UserID) (*usermgmt.User, error) {
    const q = `SELECT public_id, email, display_name, password_hash, created_at, updated_at
               FROM users WHERE public_id = $1`
    // ... scan, hydrate *usermgmt.User, return Clone()
    return nil, errors.ErrUnsupported // sketch only
}

// FindByEmail, Save, Create, Delete follow the same pattern.
// See usermgmt.InMemoryUserStore for the full method set and semantics.
```

## ID type strategy

| Layer       | Type                                  | Why                                 |
| ----------- | ------------------------------------- | ----------------------------------- |
| API/HTTP    | `usermgmt.UserID` (string-backed)     | Stable, opaque, shareable in URLs   |
| Store (SQL) | `BIGSERIAL` + `public_id TEXT UNIQUE` | Efficient joins, indexed lookups    |
| Conversion  | `usermgmt.UserID.Get()` ↔ `string`    | Single conversion at the store edge |

This is exactly the approach ADR 0003 defers to. The `public_id` column
preserves the string `UserID` for cross-module use, while `id BIGSERIAL` keeps
joins and FKs cheap.

## Concurrency

- Use `SELECT ... FOR UPDATE` (or upserts) for the email-uniqueness check in
  `Create` / `Save`. Mirror `InMemoryUserStore`'s atomicity guarantees.
- Wrap mutations in a transaction when the user and role changes must be
  consistent (e.g., `Service.UpdateRoles` flow).
- bcrypt hashing dominates wall-clock time — keep it OUT of the transaction.

## Migration strategy

- Versioned migrations via `golang-migrate/migrate`, `pressly/goose`, or
  `sqlx`-based hand-rolled files. The schema above is migration `0001_init`.
- For test isolation: per-test schema or transactional rollback. Do NOT share a
  DB across tests; the in-memory store exists for this reason.

## Wiring it up

```go
func main() {
    db, _ := sql.Open("pgx", os.Getenv("DATABASE_URL"))

    svc := usermgmt.NewService(
        usermgmtdb.NewPGUserStore(db),
        usermgmtdb.NewPGSessionStore(db),
        usermgmt.ServiceConfig{ /* ... */ },
    )
    // ... wire handlers
}
```

The `Service` constructor takes `UserStore` and `SessionStore` interfaces — the
SQL backend slots in with zero changes to domain logic.
