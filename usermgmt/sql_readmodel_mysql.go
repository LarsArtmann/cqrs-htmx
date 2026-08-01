package usermgmt

import (
	"database/sql"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

// mysqlViewStoreCreator adapts storage.NewViewStoreWithDialect to the
// viewStoreCreator signature by capturing MySQLDialect.
func mysqlViewStoreCreator[V any, K fmt.Stringer](
	db *sql.DB, mapper storage.ViewMapper[V], opts ...storage.ViewStoreOption,
) (*storage.SQLViewStore[V, K], error) {
	return storage.NewViewStoreWithDialect[V, K](db, sqlpkg.MySQLDialect{}, mapper, opts...)
}

// NewMySQLUserReadModel creates a SQL-backed user read model using MySQL syntax.
func NewMySQLUserReadModel(db *sql.DB) (*SQLUserReadModel, error) {
	return buildSQLUserReadModel(db, mysqlViewStoreCreator[UserView, UserID], "create mysql user view store")
}

// NewMySQLMembershipReadModel creates a SQL-backed membership read model using MySQL syntax.
func NewMySQLMembershipReadModel(db *sql.DB) (*SQLMembershipReadModel, error) {
	return buildSQLMembershipReadModel(
		db,
		mysqlViewStoreCreator[MembershipView, id.StreamID],
		"create mysql membership view store",
	)
}

// NewMySQLTenantReadModel creates a SQL-backed tenant read model using MySQL syntax.
func NewMySQLTenantReadModel(db *sql.DB) (*SQLTenantReadModel, error) {
	return buildSQLTenantReadModel(db, mysqlViewStoreCreator[TenantView, TenantID], "create mysql tenant view store")
}

// NewMySQLBotReadModel creates a SQL-backed bot read model using MySQL syntax.
func NewMySQLBotReadModel(db *sql.DB) (*SQLBotReadModel, error) {
	return buildSQLBotReadModel(db, mysqlViewStoreCreator[BotView, BotID], "create mysql bot view store")
}
