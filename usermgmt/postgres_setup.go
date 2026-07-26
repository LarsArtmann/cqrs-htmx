//go:build ignore

package usermgmt

import (
	"context"
	"database/sql"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
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

func NewPostgresEventSourcedSetup(cfg PostgresSetupConfig) (*PostgresEventSourcedSetup, error) {
	dsnOpts := []sqlopt.DSNOption{}
	if cfg.EventDSN != "" {
		dsnOpts = append(dsnOpts, sqlopt.WithEventDB(cfg.EventDSN))
	}
	if cfg.QueryDSN != "" {
		dsnOpts = append(dsnOpts, sqlopt.WithQueryDB(cfg.QueryDSN))
	}
	bundle, err := stackpostgres.New(cfg.DSN, stackpostgres.WithDSN(dsnOpts...))
	if err != nil {
		return nil, errorfamily.WrapTransient(err, "usermgmt.postgres_setup.create", "create postgres stack bundle")
	}
	return newPostgresSetup(bundle, cfg.AuditLog, cfg.CheckpointStore, cfg.SnapshotConfig)
}

func newPostgresSetup(
	bundle *stack.Bundle,
	auditLog *AuditLog,
	checkpointStore event.CheckpointStore,
	snap SnapshotConfig,
) (*PostgresEventSourcedSetup, error) {
	repos, err := buildStackRepositories(bundle, snap)
	if err != nil {
		return nil, err
	}

	db := extractDB(bundle)
	rm, memRm, tenRm, botRm, err := createPostgresReadModels(db)
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

	return &PostgresEventSourcedSetup{
		eventSourcedSetupCore: eventSourcedSetupCore{
			backendName: "postgres",
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

func createPostgresReadModels(db *sql.DB) (
	projection.Projection, projection.Projection, projection.Projection, projection.Projection, error,
) {
	if db == nil {
		return NewUserReadModel(), NewMembershipReadModel(), NewTenantReadModel(), NewBotReadModel(), nil
	}
	userRm, err := NewSQLUserReadModel(db)
	if err != nil {
		return nil, nil, nil, nil, errorfamily.WrapTransient(err, "internal", "create sql user read model")
	}
	memRm, err := NewSQLMembershipReadModel(db)
	if err != nil {
		return nil, nil, nil, nil, errorfamily.WrapTransient(err, "internal", "create sql membership read model")
	}
	tenRm, err := NewSQLTenantReadModel(db)
	if err != nil {
		return nil, nil, nil, nil, errorfamily.WrapTransient(err, "internal", "create sql tenant read model")
	}
	botRm, err := NewSQLBotReadModel(db)
	if err != nil {
		return nil, nil, nil, nil, errorfamily.WrapTransient(err, "internal", "create sql bot read model")
	}
	return userRm, memRm, tenRm, botRm, nil
}

type PostgresEventSourcedSetup struct {
	eventSourcedSetupCore
}
