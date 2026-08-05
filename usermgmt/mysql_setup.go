//go:build ignore

// This file is a reference template showing how to wire a full MySQL-backed
// usermgmt event-sourced setup. It is excluded from the build because it
// imports stack/mysql, which consumers may not need. Copy this file into
// your application and remove the //go:build ignore directive to use it.
//
// Requires: go-sql-driver/mysql in your application's go.mod.
//
// See docs/guides/mysql-setup.md for connection string tips and prerequisites.

package usermgmt

import (
	"database/sql"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
	stackmysql "github.com/larsartmann/go-cqrs-lite/stack/mysql/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

type MySQLSetupConfig struct {
	DSN             string
	AuditLog        *AuditLog
	CheckpointStore event.CheckpointStore

	// SnapshotConfig optionally enables aggregate snapshotting. Zero value
	// (nil Store) leaves repositories in full-replay mode. See SnapshotConfig.
	SnapshotConfig
}

func NewMySQLEventSourcedSetup(config MySQLSetupConfig) (*MySQLEventSourcedSetup, error) {
	bundle, err := stackmysql.New(config.DSN)
	if err != nil {
		return nil, errorfamily.WrapTransient(err, "usermgmt.mysql_setup.create", "create mysql stack bundle")
	}
	return newMySQLSetup(bundle, config.AuditLog, config.CheckpointStore, config.SnapshotConfig)
}

func newMySQLSetup(
	bundle *stack.Bundle,
	auditLog *AuditLog,
	checkpointStore event.CheckpointStore,
	snap SnapshotConfig,
) (*MySQLEventSourcedSetup, error) {
	core, err := buildSQLEventSourcedSetupCore("mysql", bundle, auditLog, checkpointStore, snap, createMySQLReadModels)
	if err != nil {
		return nil, err
	}
	return &MySQLEventSourcedSetup{eventSourcedSetupCore: core}, nil
}

func createMySQLReadModels(db *sql.DB) (
	projection.Projection, projection.Projection, projection.Projection, projection.Projection, error,
) {
	if db == nil {
		return NewUserReadModel(), NewMembershipReadModel(), NewTenantReadModel(), NewBotReadModel(), nil
	}
	userRm, err := NewMySQLUserReadModel(db)
	if err != nil {
		return nil, nil, nil, nil, errorfamily.WrapTransient(
			err,
			"usermgmt.read_model.create_user_mysql",
			"create mysql user read model",
		)
	}
	memRm, err := NewMySQLMembershipReadModel(db)
	if err != nil {
		return nil, nil, nil, nil, errorfamily.WrapTransient(
			err,
			"usermgmt.read_model.create_membership_mysql",
			"create mysql membership read model",
		)
	}
	tenRm, err := NewMySQLTenantReadModel(db)
	if err != nil {
		return nil, nil, nil, nil, errorfamily.WrapTransient(
			err,
			"usermgmt.read_model.create_tenant_mysql",
			"create mysql tenant read model",
		)
	}
	botRm, err := NewMySQLBotReadModel(db)
	if err != nil {
		return nil, nil, nil, nil, errorfamily.WrapTransient(
			err,
			"usermgmt.read_model.create_bot_mysql",
			"create mysql bot read model",
		)
	}
	return userRm, memRm, tenRm, botRm, nil
}

type MySQLEventSourcedSetup struct {
	eventSourcedSetupCore
}
