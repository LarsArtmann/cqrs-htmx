package usermgmt

import (
	"context"
	"database/sql"
	"io"
	"log/slog"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/watermill/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// SecurityHooks configures opt-in event signing and encryption for the
// event-sourced infrastructure. All fields are optional; zero values produce
// no security middleware. Embed this in config structs that create event
// stores and buses (ServiceConfig, EventSourcedConfig).
//
// See docs/adr/0011-event-signing-encryption.md for rationale and patterns.
type SecurityHooks struct {
	// StoreWrapper wraps the event store before the repository and projections
	// are wired. Use this for transparent encryption-at-rest, e.g.:
	//
	//	encryption.NewEncryptedStore(store, cipher)
	//
	// The wrapper receives the configured (or default in-memory) store and
	// must return a store implementing at least event.Store. To preserve
	// projection catch-up, the wrapper should also implement event.Journal when
	// the inner store does (encryption.NewEncryptedStore does this).
	// Return inner unchanged to opt out. A nil result is treated as no wrapping.
	StoreWrapper func(event.Store) (event.Store, error)

	// PublishMiddleware is applied to the event bus via bus.UsePublish before
	// projections subscribe. Use for signing
	// (signing.SignMiddleware) and/or encrypt-on-publish
	// (encryption.EncryptMiddleware). Applied in registration order; the first
	// element is outermost (runs first). Recommended order for sign+encrypt is
	// [SignMiddleware, EncryptMiddleware] — sign the plaintext, then encrypt.
	PublishMiddleware []event.PublishMiddleware

	// HandlerMiddleware is applied to the event bus via bus.Use before
	// projections subscribe. Use for verify-on-receive
	// (signing.VerifyMiddleware) and/or decrypt-on-handle
	// (encryption.DecryptMiddleware). Applied in registration order; the first
	// element is outermost. Recommended order for decrypt+verify is
	// [DecryptMiddleware, VerifyMiddleware] — decrypt to plaintext, then verify.
	HandlerMiddleware []event.Middleware
}

// EventSourcedConfig holds the infrastructure components for event-sourced user management.
// Zero-valued fields are replaced with in-memory defaults in NewEventSourcedSetup.
type EventSourcedConfig struct {
	EventStore event.Store
	EventBus   event.Bus

	// ReadModelDB, when set, creates SQL-backed read models (User, Membership,
	// Tenant, Bot) that persist across restarts. Use [OptimizeSQLiteDB] to tune
	// the connection before passing it here. When nil, in-memory read models
	// are used (data lost on restart).
	ReadModelDB *sql.DB

	// AuditLog, if provided, is registered as a projection to record
	// all user-related events for compliance and security monitoring.
	AuditLog *AuditLog

	// CheckpointStore, when set, enables checkpoint-based replay: on restart,
	// only events since the last checkpoint are replayed instead of the full
	// journal. The store must be backed by persistent storage (e.g. SQL) to
	// survive process restarts. When nil, full journal replay is used.
	CheckpointStore event.CheckpointStore

	// OnProjectionFailed, when set, is called when a projection worker
	// exhausts its restart budget and enters a terminal failure state.
	// Use this for alerting (e.g., emit a metric, page on-call). The
	// callback receives the projection name and the last error message.
	// Optional — when nil, terminal failures are silent (logs only).
	OnProjectionFailed func(projectionName, lastError string)

	// SecurityHooks configures opt-in event signing and encryption.
	SecurityHooks

	// SnapshotConfig optionally enables aggregate snapshotting for high-event-
	// volume aggregates. When the Store field is nil (the default), repositories
	// replay the full event journal on every Load — zero behavior change.
	SnapshotConfig
}

// EventSourcedSetup holds the wired infrastructure for the event-sourced user aggregate.
// Store and Bus are interfaces so that wrapped stores (e.g. encrypted stores)
// and middleware-applied buses are correctly exposed to callers.
type EventSourcedSetup struct {
	Store                event.Store
	Bus                  event.Bus
	Repository           *decider.Repository[UserState]
	MembershipRepository *decider.Repository[MembershipState]
	TenantRepository     *decider.Repository[TenantState]
	BotRepository        *decider.Repository[BotState]
	ReadModel            *UserReadModel
	MembershipReadModel  *MembershipReadModel
	TenantReadModel      *TenantReadModel
	BotReadModel         *BotReadModel
	casbinProjection     *CasbinProjection
	projectionHost       *projectionhost.Host
	checkpointStore      event.CheckpointStore
	auditLog             *AuditLog
	projections          []projection.Projection
}

// UserDecider returns the Decider for the User aggregate.
func UserDecider() decider.Decider[UserState] {
	return decider.Decider[UserState]{
		Initial: UserState{},
		Apply:   foldUser,
	}
}

// closeBus closes the bus if it implements io.Closer. In go-cqrs-lite v4, core
// interfaces no longer embed io.Closer, but concrete implementations
// (e.g. *watermill.EventBus) retain their Close method.
//cqrs-lint:ignore(D009) duck-typed Close() matches event.Bus interface which may not implement io.Closer
func closeBus(bus event.Bus) {
	if c, ok := bus.(io.Closer); ok {
		//cqrs-lint:ignore(C023) best-effort cleanup in error path
		//cqrs-lint:ignore(C015) best-effort cleanup in error paths; the real Close() on EventSourcedSetup handles errors properly
		_ = c.Close()
	}
}

// Close stops the event bus and closes the event store (if they implement
// io.Closer). It is safe to call multiple times. Use this for graceful
// shutdown of event-sourced infrastructure created by NewEventSourcedSetup.
func (s *EventSourcedSetup) Close() error {
	if s.projectionHost != nil {
		if err := s.projectionHost.Stop(); err != nil {
			slog.Warn(
				"usermgmt.EventSourcedSetup: failed to stop projection host during close",
				slog.String("error", err.Error()),
			)
			_ = errorfamily.WrapTransient(err, "usermgmt.es_setup.stop_projections", "stop projection host")
		}
	}
	if c, ok := s.Bus.(io.Closer); ok {
		if err := c.Close(); err != nil {
			return errorfamily.WrapTransient(err, "usermgmt.es_setup.close_bus", "close event bus")
		}
	}
	if c, ok := s.Store.(io.Closer); ok {
		if err := c.Close(); err != nil {
			return errorfamily.WrapTransient(err, "usermgmt.es_setup.close_store", "close event store")
		}
	}
	return nil
}

// DefaultEventSourcedSetup creates a complete event-sourced infrastructure with in-memory
// store, bus, decider repository, and read model. Projections are registered and started.
//
// The caller should defer closing the bus and stopping the projection runner.
//
// For security hooks (signing, encryption), use NewEventSourcedSetup with an
// EventSourcedConfig.
func DefaultEventSourcedSetup() (*EventSourcedSetup, error) {
	return NewEventSourcedSetup(EventSourcedConfig{}) //nolint:exhaustruct // all fields optional, zero = defaults
}

// NewEventSourcedSetup creates a complete event-sourced infrastructure from the
// given config. Applies security hooks (StoreWrapper, PublishMiddleware,
// HandlerMiddleware) at the same points as NewService — before journal
// detection, repository creation, and projection subscription.
//
// The caller should defer closing the bus.
func NewEventSourcedSetup(cfg EventSourcedConfig) (*EventSourcedSetup, error) {
	store := cfg.EventStore
	if store == nil {
		store = memory.NewMemoryStore()
	}

	store, err := wrapEventStore(cfg.StoreWrapper, store)
	if err != nil {
		return nil, err
	}

	bus := cfg.EventBus
	if bus == nil {
	//cqrs-lint:ignore(B024) go-cqrs-lite bus wraps handlers with recovery internally
		bus = watermill.NewEventBus()
	}

	if err := applyBusMiddleware(cfg.PublishMiddleware, cfg.HandlerMiddleware, bus); err != nil {
		return nil, err
	}

	repos, err := buildDeciderRepositories(store, bus, func() { closeBus(bus) }, cfg.SnapshotConfig)
	if err != nil {
		return nil, err
	}

	readModel := NewUserReadModel()
	membershipReadModel := NewMembershipReadModel()
	tenantReadModel := NewTenantReadModel()
	botReadModel := NewBotReadModel()

	// Projections registered with StartProjections. When ReadModelDB is set,
	// these are SQL wrappers (which embed the concrete read models above);
	// otherwise they are the concrete read models themselves.
	userProj := projection.Projection(readModel)
	membershipProj := projection.Projection(membershipReadModel)
	tenantProj := projection.Projection(tenantReadModel)
	botProj := projection.Projection(botReadModel)

	if cfg.ReadModelDB != nil {
		sqlUserRM, err := NewSQLiteUserReadModel(cfg.ReadModelDB)
		if err != nil {
			closeBus(bus)
			return nil, errorfamily.WrapTransient(err, "usermgmt.read_model.create_user_sql", "create SQL user read model")
		}
		readModel = sqlUserRM.UserReadModel
		userProj = sqlUserRM

		sqlMembershipRM, err := NewSQLiteMembershipReadModel(cfg.ReadModelDB)
		if err != nil {
			closeBus(bus)
			return nil, errorfamily.WrapTransient(err, "usermgmt.read_model.create_membership_sql", "create SQL membership read model")
		}
		membershipReadModel = sqlMembershipRM.MembershipReadModel
		membershipProj = sqlMembershipRM

		sqlTenantRM, err := NewSQLiteTenantReadModel(cfg.ReadModelDB)
		if err != nil {
			closeBus(bus)
			return nil, errorfamily.WrapTransient(err, "usermgmt.read_model.create_tenant_sql", "create SQL tenant read model")
		}
		tenantReadModel = sqlTenantRM.TenantReadModel
		tenantProj = sqlTenantRM

		sqlBotRM, err := NewSQLiteBotReadModel(cfg.ReadModelDB)
		if err != nil {
			closeBus(bus)
			return nil, errorfamily.WrapTransient(err, "usermgmt.read_model.create_bot_sql", "create SQL bot read model")
		}
		botReadModel = sqlBotRM.BotReadModel
		botProj = sqlBotRM
	}

	authz, err := NewAuthz()
	if err != nil {
		closeBus(bus)
		return nil, errorfamily.NewTransient("usermgmt.authz.create", "create authz").WithCause(err)
	}
	casbinProjection, err := NewCasbinProjection(authz)
	if err != nil {
		closeBus(bus)
		return nil, errorfamily.NewTransient("usermgmt.authz.create_casbin_projection", "create casbin projection").WithCause(err)
	}

	journal, err := journalFromStore(store)
	if err != nil {
		closeBus(bus)
		return nil, err
	}
	allProjections := collectProjections(
		userProj, membershipProj, tenantProj, botProj, casbinProjection, cfg.AuditLog,
	)

	var hostOpts []projectionhost.HostOption
	if cfg.OnProjectionFailed != nil {
		hostOpts = append(hostOpts, projectionhost.WithOnFailed(cfg.OnProjectionFailed))
	}

	host, err := startProjectionHost(
		context.Background(),
		journal,
		bus,
		cfg.CheckpointStore,
		allProjections,
		hostOpts...,
	)
	if err != nil {
		closeBus(bus)
		return nil, errorfamily.NewTransient("usermgmt.projection.start", "start projections").WithCause(err)
	}

	return &EventSourcedSetup{
		Store:                store,
		Bus:                  bus,
		Repository:           repos.User,
		MembershipRepository: repos.Membership,
		TenantRepository:     repos.Tenant,
		BotRepository:        repos.Bot,
		ReadModel:            readModel,
		MembershipReadModel:  membershipReadModel,
		TenantReadModel:      tenantReadModel,
		BotReadModel:         botReadModel,
		casbinProjection:     casbinProjection,
		projectionHost:       host,
		checkpointStore:      cfg.CheckpointStore,
		auditLog:             cfg.AuditLog,
		projections:          allProjections,
	}, nil
}
