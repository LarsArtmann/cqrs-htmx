package usermgmt

import (
	"context"
	"database/sql"

	"github.com/larsartmann/go-cqrs-lite/decider/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	stackpostgres "github.com/larsartmann/go-cqrs-lite/stack/postgres/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/v3"
)

type PostgresSetupConfig struct {
	DSN      string
	EventDSN string
	QueryDSN string
	AuditLog *AuditLog
}

func NewPostgresEventSourcedSetup(cfg PostgresSetupConfig) (*PostgresEventSourcedSetup, error) {
	opts := []stackpostgres.Option{}
	if cfg.EventDSN != "" {
		opts = append(opts, stackpostgres.WithEventDB(cfg.EventDSN))
	}
	if cfg.QueryDSN != "" {
		opts = append(opts, stackpostgres.WithQueryDB(cfg.QueryDSN))
	}
	bundle, err := stackpostgres.New(cfg.DSN, opts...)
	if err != nil {
		return nil, event.WrapTransient(err, "usermgmt.postgres_setup.create", "create postgres stack bundle")
	}
	return newPostgresSetup(bundle, cfg.AuditLog)
}

func newPostgresSetup(bundle *stack.Bundle, auditLog *AuditLog) (*PostgresEventSourcedSetup, error) {
	repo, err := stack.Repository(bundle, UserDecider())
	if err != nil {
		_ = bundle.Close()
		return nil, event.WrapTransient(err, "internal", "create user repository")
	}
	membershipRepo, err := stack.Repository(bundle, MembershipDecider())
	if err != nil {
		_ = bundle.Close()
		return nil, event.WrapTransient(err, "internal", "create membership repository")
	}
	tenantRepo, err := stack.Repository(bundle, TenantDecider())
	if err != nil {
		_ = bundle.Close()
		return nil, event.WrapTransient(err, "internal", "create tenant repository")
	}
	botRepo, err := stack.Repository(bundle, BotDecider())
	if err != nil {
		_ = bundle.Close()
		return nil, event.WrapTransient(err, "internal", "create bot repository")
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

	if err := StartProjections(
		bundle.Journal, bundle.Subscriber,
		rm, memRm, tenRm, botRm, casbinProj, auditLog,
	); err != nil {
		_ = bundle.Close()
		return nil, event.WrapTransient(err, "internal", "start projections")
	}

	return &PostgresEventSourcedSetup{
		UserRepository:       repo,
		MembershipRepository: membershipRepo,
		TenantRepository:     tenantRepo,
		BotRepository:        botRepo,
		ReadModel:            rm,
		MembershipReadModel:  memRm,
		TenantReadModel:      tenRm,
		BotReadModel:         botRm,
		Bundle:               bundle,
		DB:                   db,
		casbinProjection:     casbinProj,
	}, nil
}

func createPostgresReadModels(db *sql.DB) (
	event.Projection, event.Projection, event.Projection, event.Projection, error,
) {
	if db == nil {
		return NewUserReadModel(), NewMembershipReadModel(), NewTenantReadModel(), NewBotReadModel(), nil
	}
	userRm, err := NewSQLUserReadModel(db)
	if err != nil {
		return nil, nil, nil, nil, event.WrapTransient(err, "internal", "create sql user read model")
	}
	memRm, err := NewSQLMembershipReadModel(db)
	if err != nil {
		return nil, nil, nil, nil, event.WrapTransient(err, "internal", "create sql membership read model")
	}
	tenRm, err := NewSQLTenantReadModel(db)
	if err != nil {
		return nil, nil, nil, nil, event.WrapTransient(err, "internal", "create sql tenant read model")
	}
	botRm, err := NewSQLBotReadModel(db)
	if err != nil {
		return nil, nil, nil, nil, event.WrapTransient(err, "internal", "create sql bot read model")
	}
	return userRm, memRm, tenRm, botRm, nil
}

type PostgresEventSourcedSetup struct {
	UserRepository       *decider.Repository[UserState]
	MembershipRepository *decider.Repository[MembershipState]
	TenantRepository     *decider.Repository[TenantState]
	BotRepository        *decider.Repository[BotState]
	ReadModel            event.Projection
	MembershipReadModel  event.Projection
	TenantReadModel      event.Projection
	BotReadModel         event.Projection
	Bundle               *stack.Bundle
	DB                   *sql.DB
	casbinProjection     *CasbinProjection
}

func (s *PostgresEventSourcedSetup) Close() error {
	if s.Bundle != nil {
		if err := s.Bundle.Close(); err != nil {
			return event.WrapTransient(err, "usermgmt.postgres_setup.close", "close postgres bundle")
		}
	}
	return nil
}

func (s *PostgresEventSourcedSetup) GracefulClose(ctx context.Context) error {
	if s.Bundle != nil {
		if err := s.Bundle.GracefulClose(ctx); err != nil {
			return event.WrapTransient(err, "usermgmt.postgres_setup.graceful_close", "graceful close postgres bundle")
		}
	}
	return nil
}

func (s *PostgresEventSourcedSetup) Authz() *Authz { return s.casbinProjection.authz }
