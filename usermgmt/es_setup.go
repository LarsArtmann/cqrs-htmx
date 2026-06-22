package usermgmt

import (
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
}

// UserDecider returns the Decider for the User aggregate.
func UserDecider() decider.Decider[UserState] {
	return decider.Decider[UserState]{
		Initial: UserState{},
		Apply:    foldUser,
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

	repo, err := decider.NewRepository(store, bus, UserDecider())
	if err != nil {
		closeBus(bus)
		return nil, event.NewTransient("internal", "create decider repository").WithCause(err)
	}

	membershipRepo, err := decider.NewRepository(store, bus, MembershipDecider())
	if err != nil {
		closeBus(bus)
		return nil, event.NewTransient("internal", "create membership decider repository").WithCause(err)
	}

	tenantRepo, err := decider.NewRepository(store, bus, TenantDecider())
	if err != nil {
		closeBus(bus)
		return nil, event.NewTransient("internal", "create tenant decider repository").WithCause(err)
	}

	botRepo, err := decider.NewRepository(store, bus, BotDecider())
	if err != nil {
		closeBus(bus)
		return nil, event.NewTransient("internal", "create bot decider repository").WithCause(err)
	}

	readModel := NewUserReadModel()
	membershipReadModel := NewMembershipReadModel()
	tenantReadModel := NewTenantReadModel()
	botReadModel := NewBotReadModel()
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
		readModel,
		membershipReadModel,
		tenantReadModel,
		botReadModel,
		casbinProjection,
		nil,
	); err != nil {
		closeBus(bus)
		return nil, event.NewTransient("internal", "start projections").WithCause(err)
	}

	return &EventSourcedSetup{
		Store:                store,
		Bus:                  bus,
		Repository:           repo,
		MembershipRepository: membershipRepo,
		TenantRepository:     tenantRepo,
		BotRepository:        botRepo,
		ReadModel:            readModel,
		MembershipReadModel:  membershipReadModel,
		TenantReadModel:      tenantReadModel,
		BotReadModel:         botReadModel,
	}, nil
}
