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

func NewMySQLEventSourcedSetup(cfg MySQLSetupConfig) (*MySQLEventSourcedSetup, error) {
	bundle, err := stackmysql.New(cfg.DSN)
	if err != nil {
		return nil, errorfamily.WrapTransient(err, "usermgmt.mysql_setup.create", "create mysql stack bundle")
	}
	return newMySQLSetup(bundle, cfg.AuditLog, cfg.CheckpointStore, cfg.SnapshotConfig)
}

func newMySQLSetup(
	bundle *stack.Bundle,
	auditLog *AuditLog,
	checkpointStore event.CheckpointStore,
	snap SnapshotConfig,
) (*MySQLEventSourcedSetup, error) {
	repos, err := buildStackRepositories(bundle, snap)
	if err != nil {
		return nil, err
	}

	db := extractDB(bundle)
	rm, memRm, tenRm, botRm, err := createMySQLReadModels(db)
	if err != nil {
		_ = bundle.Close()
		return nil, err
	}

	casbinProj, err := createAuthzAndCasbin()
	if err != nil {
		_ = bundle.Close()
		return nil, err
	}

	host, err := StartProjections(
		bundle.Journal, bundle.Subscriber,
		checkpointStore,
		rm, memRm, tenRm, botRm, casbinProj, auditLog,
	)
	if err != nil {
		_ = bundle.Close()
		return nil, errorfamily.WrapTransient(err, "internal", "start projections")
	}

	return &MySQLEventSourcedSetup{
		eventSourcedSetupCore: eventSourcedSetupCore{
			backendName:          "mysql",
			UserRepository:       repos.User,
			MembershipRepository: repos.Membership,
			TenantRepository:     repos.Tenant,
			BotRepository:        repos.Bot,
			ReadModel:            rm,
			MembershipReadModel:  memRm,
			TenantReadModel:      tenRm,
			BotReadModel:         botRm,
			Bundle:               bundle,
			DB:                   db,
			casbinProjection:     casbinProj,
			projectionHost:       host,
		},
	}, nil
}

func createMySQLReadModels(db *sql.DB) (
	projection.Projection, projection.Projection, projection.Projection, projection.Projection, error,
) {
	if db == nil {
		return NewUserReadModel(), NewMembershipReadModel(), NewTenantReadModel(), NewBotReadModel(), nil
	}
	userRm, err := NewMySQLUserReadModel(db)
	if err != nil {
		return nil, nil, nil, nil, errorfamily.WrapTransient(err, "internal", "create mysql user read model")
	}
	memRm, err := NewMySQLMembershipReadModel(db)
	if err != nil {
		return nil, nil, nil, nil, errorfamily.WrapTransient(err, "internal", "create mysql membership read model")
	}
	tenRm, err := NewMySQLTenantReadModel(db)
	if err != nil {
		return nil, nil, nil, nil, errorfamily.WrapTransient(err, "internal", "create mysql tenant read model")
	}
	botRm, err := NewMySQLBotReadModel(db)
	if err != nil {
		return nil, nil, nil, nil, errorfamily.WrapTransient(err, "internal", "create mysql bot read model")
	}
	return userRm, memRm, tenRm, botRm, nil
}

type MySQLEventSourcedSetup struct {
	eventSourcedSetupCore
}
