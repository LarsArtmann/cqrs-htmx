//go:build ignore

package usermgmt

import (
	"database/sql"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
	stackpostgres "github.com/larsartmann/go-cqrs-lite/stack/postgres/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	sqlopt "github.com/larsartmann/go-cqrs-lite/stack/v4/sqlopt"
	errorfamily "github.com/larsartmann/go-error-family"
)

type PostgresSetupConfig struct {
	DSN             string
	EventDSN        string
	QueryDSN        string
	AuditLog        *AuditLog
	CheckpointStore event.CheckpointStore

	// SnapshotConfig optionally enables aggregate snapshotting. Zero value
	// (nil Store) leaves repositories in full-replay mode. See SnapshotConfig.
	SnapshotConfig
}

// DEPRECATED (ADR-0051): go-cqrs-lite v5 removes stack/v4 entirely. This
// template keeps working through v4, but new code should compose the
// declarative path instead: systemadapter.DomainConfig() + system.New (see
// docs/guides/declarative-projections.md), or NewEventSourcedSetup with a
// raw store+bus. This template is deleted in the v5 removal bundle.
func NewPostgresEventSourcedSetup(config PostgresSetupConfig) (*PostgresEventSourcedSetup, error) {
	dsnOpts := []sqlopt.DSNOption{}
	if config.EventDSN != "" {
		dsnOpts = append(dsnOpts, sqlopt.WithEventDB(config.EventDSN))
	}
	if config.QueryDSN != "" {
		dsnOpts = append(dsnOpts, sqlopt.WithQueryDB(config.QueryDSN))
	}
	bundle, err := stackpostgres.New(config.DSN, stackpostgres.WithDSN(dsnOpts...))
	if err != nil {
		return nil, errorfamily.WrapTransient(err, "usermgmt.postgres_setup.create", "create postgres stack bundle")
	}
	return newPostgresSetup(bundle, config.AuditLog, config.CheckpointStore, config.SnapshotConfig)
}

func newPostgresSetup(
	bundle *stack.Bundle,
	auditLog *AuditLog,
	checkpointStore event.CheckpointStore,
	snap SnapshotConfig,
) (*PostgresEventSourcedSetup, error) {
	core, err := buildSQLEventSourcedSetupCore(
		"postgres",
		bundle,
		auditLog,
		checkpointStore,
		snap,
		createPostgresReadModels,
	)
	if err != nil {
		return nil, err
	}
	return &PostgresEventSourcedSetup{eventSourcedSetupCore: core}, nil
}

func createPostgresReadModels(db *sql.DB) (
	projection.Projection, projection.Projection, projection.Projection, projection.Projection, error,
) {
	if db == nil {
		return NewUserReadModel(), NewMembershipReadModel(), NewTenantReadModel(), NewBotReadModel(), nil
	}
	userRm, err := NewSQLUserReadModel(db)
	if err != nil {
		return nil, nil, nil, nil, errorfamily.WrapTransient(
			err,
			"usermgmt.read_model.create_user_sql",
			"create sql user read model",
		)
	}
	memRm, err := NewSQLMembershipReadModel(db)
	if err != nil {
		return nil, nil, nil, nil, errorfamily.WrapTransient(
			err,
			"usermgmt.read_model.create_membership_sql",
			"create sql membership read model",
		)
	}
	tenRm, err := NewSQLTenantReadModel(db)
	if err != nil {
		return nil, nil, nil, nil, errorfamily.WrapTransient(
			err,
			"usermgmt.read_model.create_tenant_sql",
			"create sql tenant read model",
		)
	}
	botRm, err := NewSQLBotReadModel(db)
	if err != nil {
		return nil, nil, nil, nil, errorfamily.WrapTransient(
			err,
			"usermgmt.read_model.create_bot_sql",
			"create sql bot read model",
		)
	}
	return userRm, memRm, tenRm, botRm, nil
}

type PostgresEventSourcedSetup struct {
	eventSourcedSetupCore
}
