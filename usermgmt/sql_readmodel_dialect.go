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
	switch dialect {
	case "", dialectSQLite, dialectSQLite3:
		userRM, err := NewSQLiteUserReadModel(db)
		if err != nil {
			return nil, wrapReadModelError(err, "user")
		}
		memRM, err := NewSQLiteMembershipReadModel(db)
		if err != nil {
			return nil, wrapReadModelError(err, "membership")
		}
		tenRM, err := NewSQLiteTenantReadModel(db)
		if err != nil {
			return nil, wrapReadModelError(err, "tenant")
		}
		botRM, err := NewSQLiteBotReadModel(db)
		if err != nil {
			return nil, wrapReadModelError(err, "bot")
		}
		return &sqlReadModels{user: userRM, membership: memRM, tenant: tenRM, bot: botRM}, nil
	case dialectMySQL:
		userRM, err := NewMySQLUserReadModel(db)
		if err != nil {
			return nil, wrapReadModelError(err, "user")
		}
		memRM, err := NewMySQLMembershipReadModel(db)
		if err != nil {
			return nil, wrapReadModelError(err, "membership")
		}
		tenRM, err := NewMySQLTenantReadModel(db)
		if err != nil {
			return nil, wrapReadModelError(err, "tenant")
		}
		botRM, err := NewMySQLBotReadModel(db)
		if err != nil {
			return nil, wrapReadModelError(err, "bot")
		}
		return &sqlReadModels{user: userRM, membership: memRM, tenant: tenRM, bot: botRM}, nil
	case dialectPostgres, dialectPgx:
		userRM, err := NewSQLUserReadModel(db)
		if err != nil {
			return nil, wrapReadModelError(err, "user")
		}
		memRM, err := NewSQLMembershipReadModel(db)
		if err != nil {
			return nil, wrapReadModelError(err, "membership")
		}
		tenRM, err := NewSQLTenantReadModel(db)
		if err != nil {
			return nil, wrapReadModelError(err, "tenant")
		}
		botRM, err := NewSQLBotReadModel(db)
		if err != nil {
			return nil, wrapReadModelError(err, "bot")
		}
		return &sqlReadModels{user: userRM, membership: memRM, tenant: tenRM, bot: botRM}, nil
	default:
		return nil, errorfamily.Newf(event.Rejection, "usermgmt.read_model.unsupported_dialect",
			"unsupported read-model dialect %q: use sqlite, postgres, pgx, or mysql", dialect).
			WithContext("dialect", dialect)
	}
}

func wrapReadModelError(err error, name string) error {
	return errorfamily.WrapTransient(
		err,
		"usermgmt.read_model.create_"+name+"_sql",
		"create "+name+" sql read model",
	)
}
