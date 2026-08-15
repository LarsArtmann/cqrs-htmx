package usermgmt

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"time"

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

	// ReadModelDialect selects the constructor family used for ReadModelDB:
	// "" / "sqlite" / "sqlite3" (default, historical behavior), "postgres" /
	// "pgx", or "mysql". Ignored when ReadModelDB is nil. Pass the same
	// dialect you use for the event store and session store.
	ReadModelDialect string

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

	// DrainTimeout is the maximum time to wait for projection workers to finish
	// their initial journal drain during startup. Defaults to 30 seconds when
	// zero. Increase this for deployments with large event journals or slow
	// storage. A transient error is returned when the drain cannot complete
	// within this budget. Ignored when AsyncStartup is true.
	DrainTimeout time.Duration

	// AsyncStartup controls whether NewEventSourcedSetup blocks until projection
	// workers finish their initial journal drain. When false (the default),
	// construction blocks until all projections reach a terminal drain state
	// (or DrainTimeout elapses) — preserving the historical synchronous startup
	// behavior.
	//
	// Set to true to start the HTTP server immediately while projections catch
	// up in the background. This eliminates the startup outage window caused by
	// full-journal replay on every restart (deployments with large event
	// journals can see multi-minute downtime otherwise). When true, gate reads
	// behind a readiness check (cqrshtmx.ProjectionReadinessCheck) so the
	// reverse proxy retries on 503 until projections reach "live" state.
	//
	// With async startup, projection failures during drain are no longer
	// returned as construction errors — monitor them via ProjectionStatuses(),
	// the /health readiness endpoint, or the OnProjectionFailed callback.
	AsyncStartup bool
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
func closeBus(bus event.Bus) {
	//cqrs-lint:ignore(D009) duck-typed Close() matches event.Bus interface which may not implement io.Closer
	if c, ok := bus.(io.Closer); ok {
		if err := c.Close(); err != nil {
			slog.Debug("closeBus: best-effort close failed", "error", err)
		}
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
func NewEventSourcedSetup(config EventSourcedConfig) (*EventSourcedSetup, error) {
	store := config.EventStore
	if store == nil {
		store = memory.NewMemoryStore()
	}

	store, err := wrapEventStore(config.StoreWrapper, store)
	if err != nil {
		return nil, err
	}

	bus := config.EventBus
	if bus == nil {
		bus = watermill.NewEventBus()
	}

	if err := applyBusMiddleware(config.PublishMiddleware, config.HandlerMiddleware, bus); err != nil {
		return nil, err
	}

	repos, err := buildDeciderRepositories(store, bus, func() { closeBus(bus) }, config.SnapshotConfig)
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

	if config.ReadModelDB != nil {
		sqlRMs, err := newSQLReadModelsForDialect(config.ReadModelDB, config.ReadModelDialect)
		if err != nil {
			closeBus(bus)
			return nil, err
		}
		readModel = sqlRMs.user.UserReadModel
		userProj = sqlRMs.user
		membershipReadModel = sqlRMs.membership.MembershipReadModel
		membershipProj = sqlRMs.membership
		tenantReadModel = sqlRMs.tenant.TenantReadModel
		tenantProj = sqlRMs.tenant
		botReadModel = sqlRMs.bot.BotReadModel
		botProj = sqlRMs.bot
	}

	authz, err := NewAuthz()
	if err != nil {
		closeBus(bus)
		return nil, errorfamily.NewTransient("usermgmt.authz.create", "create authz").WithCause(err)
	}
	casbinProjection, err := NewCasbinProjection(authz)
	if err != nil {
		closeBus(bus)
		return nil, errorfamily.NewTransient("usermgmt.authz.create_casbin_projection", "create casbin projection").
			WithCause(err)
	}

	journal := journalFromStore(store)
	allProjections := collectProjections(
		userProj, membershipProj, tenantProj, botProj, casbinProjection, config.AuditLog,
	)

	var hostOpts []projectionhost.HostOption
	if config.OnProjectionFailed != nil {
		hostOpts = append(hostOpts, projectionhost.WithOnFailed(config.OnProjectionFailed))
	}

	host, err := startProjectionHost(
		context.Background(),
		journal,
		bus,
		config.CheckpointStore,
		allProjections,
		config.DrainTimeout,
		!config.AsyncStartup,
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
		checkpointStore:      config.CheckpointStore,
		auditLog:             config.AuditLog,
		projections:          allProjections,
	}, nil
}
