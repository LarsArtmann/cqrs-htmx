package usermgmt

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/decider/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
)

// EventSourcedConfig holds the infrastructure components for event-sourced user management.
// Zero-valued fields are replaced with in-memory defaults in DefaultEventSourcedSetup.
type EventSourcedConfig struct {
	EventStore event.Store
	EventBus   event.Bus
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
func DefaultEventSourcedSetup() (*EventSourcedSetup, error) {
	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()

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

	if err := StartProjections(store, bus, readModel, casbinProjection, nil); err != nil {
		_ = bus.Close()
		return nil, err
	}

	return &EventSourcedSetup{
		Store:      store,
		Bus:        bus,
		Repository: repo,
		ReadModel:  readModel,
	}, nil
}
