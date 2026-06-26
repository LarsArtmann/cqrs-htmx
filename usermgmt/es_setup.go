package usermgmt

import (
	"database/sql"
	"io"

	"github.com/larsartmann/go-cqrs-lite/decider/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v3"
	"github.com/larsartmann/go-cqrs-lite/watermill/v3"
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

	// SecurityHooks configures opt-in event signing and encryption.
	SecurityHooks
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
}

// UserDecider returns the Decider for the User aggregate.
func UserDecider() decider.Decider[UserState] {
	return decider.Decider[UserState]{
		Initial: UserState{},
		Apply:   foldUser,
	}
}

// closeBus closes the bus if it implements io.Closer. In go-cqrs-lite v3, core
// interfaces no longer embed io.Closer, but concrete implementations
// (e.g. *watermill.EventBus) retain their Close method.
func closeBus(bus event.Bus) {
	if c, ok := bus.(io.Closer); ok {
		_ = c.Close()
	}
}

// Close stops the event bus and closes the event store (if they implement
// io.Closer). It is safe to call multiple times. Use this for graceful
// shutdown of event-sourced infrastructure created by NewEventSourcedSetup.
func (s *EventSourcedSetup) Close() error {
	if c, ok := s.Bus.(io.Closer); ok {
		if err := c.Close(); err != nil {
			return event.WrapTransient(err, "usermgmt.es_setup.close_bus", "close event bus")
		}
	}
	if c, ok := s.Store.(io.Closer); ok {
		if err := c.Close(); err != nil {
			return event.WrapTransient(err, "usermgmt.es_setup.close_store", "close event store")
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
		bus = watermill.NewEventBus()
	}

	if err := applyBusMiddleware(cfg.PublishMiddleware, cfg.HandlerMiddleware, bus); err != nil {
		return nil, err
	}

	repos, err := buildDeciderRepositories(store, bus, func() { closeBus(bus) })
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
	userProj := event.Projection(readModel)
	membershipProj := event.Projection(membershipReadModel)
	tenantProj := event.Projection(tenantReadModel)
	botProj := event.Projection(botReadModel)

	if cfg.ReadModelDB != nil {
		sqlUserRM, err := NewSQLiteUserReadModel(cfg.ReadModelDB)
		if err != nil {
			closeBus(bus)
			return nil, event.WrapTransient(err, "internal", "create SQL user read model")
		}
		readModel = sqlUserRM.UserReadModel
		userProj = sqlUserRM

		sqlMembershipRM, err := NewSQLiteMembershipReadModel(cfg.ReadModelDB)
		if err != nil {
			closeBus(bus)
			return nil, event.WrapTransient(err, "internal", "create SQL membership read model")
		}
		membershipReadModel = sqlMembershipRM.MembershipReadModel
		membershipProj = sqlMembershipRM

		sqlTenantRM, err := NewSQLiteTenantReadModel(cfg.ReadModelDB)
		if err != nil {
			closeBus(bus)
			return nil, event.WrapTransient(err, "internal", "create SQL tenant read model")
		}
		tenantReadModel = sqlTenantRM.TenantReadModel
		tenantProj = sqlTenantRM

		sqlBotRM, err := NewSQLiteBotReadModel(cfg.ReadModelDB)
		if err != nil {
			closeBus(bus)
			return nil, event.WrapTransient(err, "internal", "create SQL bot read model")
		}
		botReadModel = sqlBotRM.BotReadModel
		botProj = sqlBotRM
	}

	authz, err := NewAuthz()
	if err != nil {
		closeBus(bus)
		return nil, event.NewTransient("internal", "create authz").WithCause(err)
	}
	casbinProjection, err := NewCasbinProjection(authz)
	if err != nil {
		closeBus(bus)
		return nil, event.NewTransient("internal", "create casbin projection").WithCause(err)
	}

	journal := journalFromStore(store)
	if err := StartProjections(
		journal,
		bus,
		userProj,
		membershipProj,
		tenantProj,
		botProj,
		casbinProjection,
		cfg.AuditLog,
	); err != nil {
		closeBus(bus)
		return nil, event.NewTransient("internal", "start projections").WithCause(err)
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
	}, nil
}
