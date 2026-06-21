package usermgmt

import (
	"context"
	"database/sql"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/storage/v2"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v2/sql"
)

// SQLEventStore persists user domain events in a SQL database with optimistic
// concurrency. It delegates entirely to go-cqrs-lite/storage/v2's
// SQLEventStore, which provides schema versioning, payload encoding tracking,
// OpenTelemetry tracing, and SeekableJournal/BackwardsSource support.
//
// Call [NewSQLEventStore] to create and auto-migrate the schema.
//
// The store borrows the *sql.DB — calling Close does NOT close the database
// connection; the caller owns its lifecycle and must close it separately.
//
// Usage:
//
//	db, _ := sql.Open("pgx", "postgres://localhost/users")
//	store, _ := usermgmt.NewSQLEventStore(ctx, db, "postgres")
//	defer store.Close() // marks store closed; db stays open
//	defer db.Close()    // caller closes the DB
//	svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{
//	    EventStore: store,
//	})
type SQLEventStore = storage.SQLEventStore

// SQL dialect string identifiers shared by the event store and session store.
// Event store supports: postgres, pgx, sqlite, sqlite3 (no MySQL — upstream
// go-cqrs-lite/storage has no MySQL dialect).
// Session store supports all five including MySQL (manages its own schema).
const (
	dialectPostgres = "postgres"
	dialectPgx      = "pgx"
	dialectSQLite   = "sqlite"
	dialectSQLite3  = "sqlite3"
	dialectMySQL    = "mysql"
)

// NewSQLEventStore creates a SQLEventStore and auto-migrates the events table.
// The dialect must be "postgres", "pgx", "sqlite", or "sqlite3".
//
// MySQL is not supported for the event store — go-cqrs-lite/storage does not
// ship a MySQL dialect. The session store ([NewSQLSessionStore]) does support
// MySQL because it manages its own simpler schema.
func NewSQLEventStore(ctx context.Context, db *sql.DB, dialect string) (*SQLEventStore, error) {
	d, err := dialectToUpstream(dialect)
	if err != nil {
		return nil, err
	}
	store, err := storage.NewSQLEventStoreWithDialect(db, d)
	if err != nil {
		return nil, event.WrapTransient(err, "usermgmt.sql_event_store.create_failed", "create sql event store")
	}
	// Upstream does not auto-migrate — apply the event schema ourselves.
	if _, err := db.ExecContext(ctx, d.EventSchema()); err != nil {
		return nil, event.WrapTransient(err, "usermgmt.sql_event_store.migrate_failed", "migrate sql event store")
	}
	return store, nil
}

// dialectToUpstream maps usermgmt dialect strings to storage/v2 Dialect
// implementations. Returns an error for unsupported dialects (e.g. MySQL).
func dialectToUpstream(dialect string) (sqlpkg.Dialect, error) {
	switch dialect {
	case dialectPostgres, dialectPgx:
		return sqlpkg.PostgresDialect{}, nil
	case dialectSQLite, dialectSQLite3:
		return sqlpkg.SQLiteDialect{}, nil
	default:
		return nil, event.Newf(event.Rejection, "usermgmt.sql_event_store.unsupported_dialect",
			"unsupported event store dialect %q: use postgres, pgx, sqlite, or sqlite3", dialect)
	}
}

var (
	_ event.Store   = (*SQLEventStore)(nil)
	_ event.Journal = (*SQLEventStore)(nil)
)
