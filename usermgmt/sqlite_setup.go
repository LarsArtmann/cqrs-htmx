//go:build ignore

package usermgmt

import (
	"database/sql"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
	stacksqlite "github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	sqlopt "github.com/larsartmann/go-cqrs-lite/stack/v4/sqlopt"
	errorfamily "github.com/larsartmann/go-error-family"
)

// SQLiteSetupConfig configures NewSQLiteEventSourcedSetup.
type SQLiteSetupConfig struct {
	DSN             string
	AuditLog        *AuditLog
	CheckpointStore event.CheckpointStore

	// SnapshotConfig optionally enables aggregate snapshotting. Zero value
	// (nil Store) leaves repositories in full-replay mode. See SnapshotConfig.
	SnapshotConfig
}

// NewSQLiteEventSourcedSetup creates a complete event-sourced infrastructure
// backed by SQLite with persistent SQL read models. Uses go-cqrs-lite's
// stack/sqlite preset for event store, bus, and database management.
//
// For event signing/encryption (SecurityHooks), use NewEventSourcedSetup with
// a manually-configured EventStore and EventBus — the stack preset does not
// expose the injection points needed for wrapping.
func NewSQLiteEventSourcedSetup(config SQLiteSetupConfig) (*SQLiteEventSourcedSetup, error) {
	bundle, err := stacksqlite.New(
		config.DSN,
		stacksqlite.WithPragmas(sqlopt.WithOptimizations(), sqlopt.WithForeignKeys()),
	)
	if err != nil {
		return nil, errorfamily.WrapTransient(err, "usermgmt.sqlite_setup.create", "create sqlite stack bundle")
	}

	return newSQLiteSetup(bundle, config.AuditLog, config.CheckpointStore, config.SnapshotConfig)
}

func newSQLiteSetup(
	bundle *stack.Bundle,
	auditLog *AuditLog,
	checkpointStore event.CheckpointStore,
	snap SnapshotConfig,
) (*SQLiteEventSourcedSetup, error) {
	core, err := buildSQLEventSourcedSetupCore(
		"sqlite",
		bundle,
		auditLog,
		checkpointStore,
		snap,
		createSQLiteReadModels,
	)
	if err != nil {
		return nil, err
	}
	return &SQLiteEventSourcedSetup{eventSourcedSetupCore: core}, nil
}

func createSQLiteReadModels(db *sql.DB) (
	projection.Projection, projection.Projection, projection.Projection, projection.Projection, error,
) {
	if db == nil {
		userRm := projection.Projection(NewUserReadModel())
		memRm := projection.Projection(NewMembershipReadModel())
		tenRm := projection.Projection(NewTenantReadModel())
		botRm := projection.Projection(NewBotReadModel())
		return userRm, memRm, tenRm, botRm, nil
	}
	userRm, err := NewSQLiteUserReadModel(db)
	if err != nil {
		return nil, nil, nil, nil, errorfamily.WrapTransient(
			err,
			"usermgmt.read_model.create_user_sql",
			"create sql user read model",
		)
	}
	memRm, err := NewSQLiteMembershipReadModel(db)
	if err != nil {
		return nil, nil, nil, nil, errorfamily.WrapTransient(
			err,
			"usermgmt.read_model.create_membership_sql",
			"create sql membership read model",
		)
	}
	tenRm, err := NewSQLiteTenantReadModel(db)
	if err != nil {
		return nil, nil, nil, nil, errorfamily.WrapTransient(
			err,
			"usermgmt.read_model.create_tenant_sql",
			"create sql tenant read model",
		)
	}
	botRm, err := NewSQLiteBotReadModel(db)
	if err != nil {
		return nil, nil, nil, nil, errorfamily.WrapTransient(
			err,
			"usermgmt.read_model.create_bot_sql",
			"create sql bot read model",
		)
	}
	return userRm, memRm, tenRm, botRm, nil
}

// SQLiteEventSourcedSetup provides SQLite-backed event-sourced infrastructure.
type SQLiteEventSourcedSetup struct {
	eventSourcedSetupCore
}
