package usermgmt

import (
	"database/sql"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// sqlReadModels bundles the four SQL-backed read-model projections created
// from a single *sql.DB. Each embeds its concrete in-memory read model (for
// typed queries) while persisting through the dialect-specific view store.
type sqlReadModels struct {
	user       *SQLUserReadModel
	membership *SQLMembershipReadModel
	tenant     *SQLTenantReadModel
	bot        *SQLBotReadModel
}

// readModelConstructors selects the four constructor functions matching a
// SQL dialect. Keeping them as one unit makes the dialect switch table-driven.
type readModelConstructors struct {
	user       func(*sql.DB) (*SQLUserReadModel, error)
	membership func(*sql.DB) (*SQLMembershipReadModel, error)
	tenant     func(*sql.DB) (*SQLTenantReadModel, error)
	bot        func(*sql.DB) (*SQLBotReadModel, error)
}

func readModelConstructorsForDialect(dialect string) (readModelConstructors, bool) {
	switch dialect {
	case "", dialectSQLite, dialectSQLite3:
		return readModelConstructors{
			user:       NewSQLiteUserReadModel,
			membership: NewSQLiteMembershipReadModel,
			tenant:     NewSQLiteTenantReadModel,
			bot:        NewSQLiteBotReadModel,
		}, true
	case dialectMySQL:
		return readModelConstructors{
			user:       NewMySQLUserReadModel,
			membership: NewMySQLMembershipReadModel,
			tenant:     NewMySQLTenantReadModel,
			bot:        NewMySQLBotReadModel,
		}, true
	case dialectPostgres, dialectPgx:
		return readModelConstructors{
			user:       NewSQLUserReadModel,
			membership: NewSQLMembershipReadModel,
			tenant:     NewSQLTenantReadModel,
			bot:        NewSQLBotReadModel,
		}, true
	}

	//nolint:exhaustruct // zero value is the "unsupported dialect" signal, paired with the bool
	return readModelConstructors{}, false
}

// newSQLReadModelsForDialect creates the four SQL-backed read models using
// the constructor family matching the dialect:
//
//   - "" / "sqlite" / "sqlite3" — SQLite view stores (historical default)
//   - "mysql"                   — MySQL view stores (backtick quoting, ? placeholders)
//   - "postgres" / "pgx"        — generic SQL view stores (Postgres dialect)
//
// The empty string defaults to SQLite, preserving the historical ReadModelDB
// behavior. The schemas are auto-migrated by the view stores.
func newSQLReadModelsForDialect(db *sql.DB, dialect string) (*sqlReadModels, error) {
	ctors, ok := readModelConstructorsForDialect(dialect)
	if !ok {
		return nil, errorfamily.Newf(event.Rejection, "usermgmt.read_model.unsupported_dialect",
			"unsupported read-model dialect %q: use sqlite, postgres, pgx, or mysql", dialect).
			WithContext("dialect", dialect)
	}

	userRM, err := ctors.user(db)
	if err != nil {
		return nil, wrapReadModelError(err, "user")
	}

	memRM, err := ctors.membership(db)
	if err != nil {
		return nil, wrapReadModelError(err, "membership")
	}

	tenRM, err := ctors.tenant(db)
	if err != nil {
		return nil, wrapReadModelError(err, "tenant")
	}

	botRM, err := ctors.bot(db)
	if err != nil {
		return nil, wrapReadModelError(err, "bot")
	}

	return &sqlReadModels{user: userRM, membership: memRM, tenant: tenRM, bot: botRM}, nil
}

func wrapReadModelError(err error, name string) error {
	return errorfamily.WrapTransient(
		err,
		"usermgmt.read_model.create_"+name+"_sql",
		"create "+name+" sql read model",
	)
}
