package usermgmt

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

// TestService_InfrastructureAccessors verifies the composition seam accessors:
// Journal and EventBus must return the exact instances configured on the
// service, so downstream consumers (setup bundle, dashboards, custom SSE
// bridges) wire against the service itself instead of re-tracking config.
func TestService_InfrastructureAccessors(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	bus := watermill.NewEventBus()

	svc, err := NewService(ServiceConfig{
		EventStore: store,
		EventBus:   bus,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	defer func() { _ = svc.Close() }()

	if svc.Journal() == nil {
		t.Fatal("Journal() returned nil for a service built with an explicit event store")
	}

	if svc.EventBus() == nil {
		t.Fatal("EventBus() returned nil for a service built with an explicit event bus")
	}

	var gotStore event.Store = store
	if svc.Journal() != gotStore {
		t.Error("Journal() must return the configured event.Store instance")
	}

	var gotBus event.Bus = bus
	if svc.EventBus() != gotBus {
		t.Error("EventBus() must return the configured event.Bus instance")
	}
}

// TestService_InfrastructureAccessors_Defaults verifies the accessors surface
// the service-created defaults too, not just explicitly configured instances.
func TestService_InfrastructureAccessors_Defaults(t *testing.T) {
	t.Parallel()

	svc, err := NewService(ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	defer func() { _ = svc.Close() }()

	if svc.Journal() == nil {
		t.Error("Journal() returned nil; the default memory store must be exposed")
	}

	if svc.EventBus() == nil {
		t.Error("EventBus() returned nil; the default event bus must be exposed")
	}
}
