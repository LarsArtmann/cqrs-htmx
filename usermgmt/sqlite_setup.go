package usermgmt

import (
	"context"
	"database/sql"

	"github.com/larsartmann/go-cqrs-lite/decider/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/projection/v3"
	stacksqlite "github.com/larsartmann/go-cqrs-lite/stack/sqlite/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/v3"
	errorfamily "github.com/larsartmann/go-error-family"
)

// SQLiteSetupConfig configures NewSQLiteEventSourcedSetup.
type SQLiteSetupConfig struct {
	DSN             string
	AuditLog        *AuditLog
	CheckpointStore event.CheckpointStore
}

// NewSQLiteEventSourcedSetup creates a complete event-sourced infrastructure
// backed by SQLite with persistent SQL read models. Uses go-cqrs-lite's
// stack/sqlite preset for event store, bus, and database management.
//
// For event signing/encryption (SecurityHooks), use NewEventSourcedSetup with
// a manually-configured EventStore and EventBus — the stack preset does not
// expose the injection points needed for wrapping.
func NewSQLiteEventSourcedSetup(cfg SQLiteSetupConfig) (*SQLiteEventSourcedSetup, error) {
	bundle, err := stacksqlite.New(
		cfg.DSN,
		stacksqlite.WithOptimizations(),
		stacksqlite.WithForeignKeys(),
	)
	if err != nil {
		return nil, errorfamily.WrapTransient(err, "usermgmt.sqlite_setup.create", "create sqlite stack bundle")
	}

	return newSQLiteSetup(bundle, cfg.AuditLog, cfg.CheckpointStore)
}

func newSQLiteSetup(
	bundle *stack.Bundle,
	auditLog *AuditLog,
	checkpointStore event.CheckpointStore,
) (*SQLiteEventSourcedSetup, error) {
	repos, err := buildStackRepositories(bundle)
	if err != nil {
		return nil, err
	}

	rm, memRm, tenRm, botRm, err := createSQLReadModels(bundle)
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
		checkpointStore,
		rm, memRm, tenRm, botRm, casbinProj, auditLog,
	); err != nil {
		_ = bundle.Close()
		return nil, errorfamily.WrapTransient(err, "internal", "start projections")
	}

	return &SQLiteEventSourcedSetup{
		UserRepository:       repos.User,
		MembershipRepository: repos.Membership,
		TenantRepository:     repos.Tenant,
		BotRepository:        repos.Bot,
		ReadModel:            rm,
		MembershipReadModel:  memRm,
		TenantReadModel:      tenRm,
		BotReadModel:         botRm,
		Bundle:               bundle,
		DB:                   extractDB(bundle),
		casbinProjection:     casbinProj,
	}, nil
}

func createSQLReadModels(bundle *stack.Bundle) (
	projection.Projection, projection.Projection, projection.Projection, projection.Projection, error,
) {
	db := extractDB(bundle)
	if db == nil {
		userRm := projection.Projection(NewUserReadModel())
		memRm := projection.Projection(NewMembershipReadModel())
		tenRm := projection.Projection(NewTenantReadModel())
		botRm := projection.Projection(NewBotReadModel())
		return userRm, memRm, tenRm, botRm, nil
	}
	userRm, err := NewSQLiteUserReadModel(db)
	if err != nil {
		return nil, nil, nil, nil, errorfamily.WrapTransient(err, "internal", "create sql user read model")
	}
	memRm, err := NewSQLiteMembershipReadModel(db)
	if err != nil {
		return nil, nil, nil, nil, errorfamily.WrapTransient(err, "internal", "create sql membership read model")
	}
	tenRm, err := NewSQLiteTenantReadModel(db)
	if err != nil {
		return nil, nil, nil, nil, errorfamily.WrapTransient(err, "internal", "create sql tenant read model")
	}
	botRm, err := NewSQLiteBotReadModel(db)
	if err != nil {
		return nil, nil, nil, nil, errorfamily.WrapTransient(err, "internal", "create sql bot read model")
	}
	return userRm, memRm, tenRm, botRm, nil
}

func createAuthzAndCasbin() (*CasbinProjection, error) {
	authz, err := NewAuthz()
	if err != nil {
		return nil, errorfamily.WrapTransient(err, "internal", "create authz")
	}
	casbinProj, err := NewCasbinProjection(authz)
	if err != nil {
		return nil, errorfamily.WrapTransient(err, "internal", "create casbin projection")
	}
	return casbinProj, nil
}

func extractDB(bundle *stack.Bundle) *sql.DB {
	db, _ := bundle.Database().(*sql.DB)
	return db
}

// SQLiteEventSourcedSetup provides SQLite-backed event-sourced infrastructure.
type SQLiteEventSourcedSetup struct {
	UserRepository       *decider.Repository[UserState]
	MembershipRepository *decider.Repository[MembershipState]
	TenantRepository     *decider.Repository[TenantState]
	BotRepository        *decider.Repository[BotState]
	ReadModel            projection.Projection
	MembershipReadModel  projection.Projection
	TenantReadModel      projection.Projection
	BotReadModel         projection.Projection
	Bundle               *stack.Bundle
	DB                   *sql.DB
	casbinProjection     *CasbinProjection
}

func (s *SQLiteEventSourcedSetup) Close() error {
	if s.Bundle != nil {
		if err := s.Bundle.Close(); err != nil {
			return errorfamily.WrapTransient(err, "usermgmt.sqlite_setup.close", "close sqlite bundle")
		}
	}
	return nil
}

func (s *SQLiteEventSourcedSetup) GracefulClose(ctx context.Context) error {
	if s.Bundle != nil {
		if err := s.Bundle.GracefulClose(ctx); err != nil {
			return errorfamily.WrapTransient(
				err,
				"usermgmt.sqlite_setup.graceful_close",
				"graceful close sqlite bundle",
			)
		}
	}
	return nil
}

func (s *SQLiteEventSourcedSetup) Authz() *Authz { return s.casbinProjection.authz }
