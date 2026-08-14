package usermgmt

import (
	"database/sql"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
	errorfamily "github.com/larsartmann/go-error-family"
)

// NewMySQLCheckpointStore creates a MySQL-backed checkpoint store for
// projection replay positions. Checkpoints survive restarts, so projections
// resume from where they stopped instead of replaying the full journal.
//
// Pass it as EventSourcedConfig.CheckpointStore / ServiceConfig.CheckpointStore
// (or MySQLSetupConfig.CheckpointStore) when running on MySQL:
//
//	cpStore, _ := usermgmt.NewMySQLCheckpointStore(db)
//	cfg := usermgmt.EventSourcedConfig{ /* ... */ CheckpointStore: cpStore }
//
// The schema is auto-migrated by the underlying constructor.
func NewMySQLCheckpointStore(db *sql.DB) (event.CheckpointStore, error) {
	store, err := storage.NewSQLCheckpointStoreWithDialect(db, sqlpkg.MySQLDialect{})
	if err != nil {
		return nil, errorfamily.WrapTransient(
			err,
			"usermgmt.mysql.checkpoint_create",
			"create mysql checkpoint store",
		)
	}
	return store, nil
}

// NewMySQLSnapshotStore creates a MySQL-backed snapshot store for aggregate
// snapshotting. Pair it with a Codec and Strategy in SnapshotConfig:
//
//	snapStore, _ := usermgmt.NewMySQLSnapshotStore(db)
//	strategy, _ := snapshot.EveryNEvents(500)
//	cfg := usermgmt.EventSourcedConfig{ /* ... */ }
//	cfg.SnapshotConfig = usermgmt.SnapshotConfig{
//		Store:    snapStore,
//		Codec:    codec.JSONCodec{},
//		Strategy: strategy,
//	}
//
// The schema is auto-migrated by the underlying constructor.
func NewMySQLSnapshotStore(db *sql.DB) (snapshot.SnapshotStore, error) {
	store, err := storage.NewSQLSnapshotStoreWithDialect(db, sqlpkg.MySQLDialect{})
	if err != nil {
		return nil, errorfamily.WrapTransient(
			err,
			"usermgmt.mysql.snapshot_create",
			"create mysql snapshot store",
		)
	}
	return store, nil
}
