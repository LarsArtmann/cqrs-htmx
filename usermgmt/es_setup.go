package usermgmt

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/decider/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
)

// EventSourcedConfig holds the infrastructure components for event-sourced user management.
// Zero-valued fields are replaced with in-memory defaults in NewEventSourcedSetup.
type EventSourcedConfig struct {
	EventStore event.Store
	EventBus   event.Bus

	// StoreWrapper wraps the event store before the repository and projections
	// are wired. Use for transparent encryption-at-rest:
	//
	//	encryption.NewEncryptedStore(store, cipher)
	//
	// See ServiceConfig.StoreWrapper for full documentation.
	StoreWrapper func(event.Store) (event.Store, error)

	// PublishMiddleware is applied via bus.UsePublish before projections subscribe.
	// Use for signing (signing.SignMiddleware) and/or encrypt-on-publish
	// (encryption.EncryptMiddleware). See ServiceConfig.PublishMiddleware.
	PublishMiddleware []event.PublishMiddleware

	// HandlerMiddleware is applied via bus.Use before projections subscribe.
	// Use for verify-on-receive (signing.VerifyMiddleware) and/or decrypt-on-handle
	// (encryption.DecryptMiddleware). See ServiceConfig.HandlerMiddleware.
	HandlerMiddleware []event.Middleware
}

// EventSourcedSetup holds the wired infrastructure for the event-sourced user aggregate.
type EventSourcedSetup struct {
	Store      *memory.MemoryStore
	Bus        *memory.MemoryBus
	Repository *decider.Repository[UserState]
	ReadModel  *UserReadModel
}

// UserDecider returns the Decider for the User aggregate.
func UserDecider() decider.Decider[UserState] {
	return decider.Decider[UserState]{
		Initial: UserState{},
		Fold:    foldUser,
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
		bus = memory.NewMemoryBus()
	}

	if err := applyBusMiddleware(cfg.PublishMiddleware, cfg.HandlerMiddleware, bus); err != nil {
		return nil, err
	}

	repo, err := decider.NewRepository(store, bus, UserDecider())
	if err != nil {
		_ = bus.Close()
		return nil, fmt.Errorf("create decider repository: %w", err)
	}

	readModel := NewUserReadModel()
	authz, err := NewAuthz()
	if err != nil {
		_ = bus.Close()
		return nil, err
	}
	casbinProjection, err := NewCasbinProjection(authz)
	if err != nil {
		_ = bus.Close()
		return nil, err
	}

	journal := journalFromStore(store)
	if err := StartProjections(journal, bus, readModel, casbinProjection, nil); err != nil {
		_ = bus.Close()
		return nil, err
	}

	result := &EventSourcedSetup{
		Repository: repo,
		ReadModel:  readModel,
	}
	if memStore, ok := store.(*memory.MemoryStore); ok {
		result.Store = memStore
	}
	if memBus, ok := bus.(*memory.MemoryBus); ok {
		result.Bus = memBus
	}

	return result, nil
}
