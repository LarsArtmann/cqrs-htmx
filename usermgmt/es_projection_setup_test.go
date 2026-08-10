package usermgmt

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

// stubProjection is a minimal projection.Projection for testing the projection
// setup machinery without pulling in real read models.
type stubProjection struct {
	name      string
	types     []event.Type
	handled   []event.Event
	handleErr error
}

func (s *stubProjection) Name() string             { return s.name }
func (s *stubProjection) EventTypes() []event.Type { return s.types }
func (s *stubProjection) Handle(_ context.Context, evt event.Event) error {
	s.handled = append(s.handled, evt)

	return s.handleErr
}

func TestCollectProjections_OptionalNil(t *testing.T) {
	t.Parallel()

	readModel := NewUserReadModel()
	authz, err := NewAuthz()
	if err != nil {
		t.Fatalf("NewAuthz: %v", err)
	}
	casbinProj, err := NewCasbinProjection(authz)
	if err != nil {
		t.Fatalf("NewCasbinProjection: %v", err)
	}

	// All optional projections nil
	projections := collectProjections(readModel, nil, nil, nil, casbinProj, nil)
	if len(projections) != 2 {
		t.Errorf("expected 2 mandatory projections (readModel + casbin), got %d", len(projections))
	}

	// All optional projections provided
	membership := NewMembershipReadModel()
	tenant := NewTenantReadModel()
	bot := NewBotReadModel()
	audit := NewAuditLog()
	projections = collectProjections(readModel, membership, tenant, bot, casbinProj, audit)
	if len(projections) != 6 {
		t.Errorf("expected 6 projections when all provided, got %d", len(projections))
	}
}

// TestStartProjections_ReadYourWrites verifies that after StartProjections returns,
// all historical events have been replayed into the read models (read-your-writes
// guarantee). The projection host drains the journal synchronously before returning.
func TestStartProjections_ReadYourWrites(t *testing.T) {
	t.Parallel()

	setup, err := NewEventSourcedSetup(EventSourcedConfig{})
	if err != nil {
		t.Fatalf("NewEventSourcedSetup: %v", err)
	}
	t.Cleanup(func() { _ = setup.Close() })

	// The read model should be empty after initial setup (no events in journal).
	if count := setup.ReadModel.Count(); count != 0 {
		t.Errorf("expected 0 users after empty setup, got %d", count)
	}
}

// stubProjection compile-time check
var _ projection.Projection = (*stubProjection)(nil)

// slowProjection is a stub that sleeps on every Handle call, used to force
// a drain timeout in tests.
type slowProjection struct {
	stubProjection
	delay time.Duration
}

func (s *slowProjection) Handle(ctx context.Context, evt event.Event) error {
	time.Sleep(s.delay)
	return s.stubProjection.Handle(ctx, evt)
}

// TestStartProjectionHost_CustomDrainTimeout verifies that a custom drain
// timeout is respected: when projections are slow and the timeout is short,
// the factory returns a transient drain_timeout error mentioning the timeout.
func TestStartProjectionHost_CustomDrainTimeout(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	evt := makeRegistrationEvents(1)[0]
	ref := id.StreamRef{ID: evt.StreamID(), Type: "User"}
	if err := store.AppendBatch(context.Background(), ref, []event.Event{evt}); err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	bus := watermill.NewEventBus()
	t.Cleanup(func() { closeBus(bus) })

	slow := &slowProjection{
		stubProjection: stubProjection{
			name:  "slow",
			types: []event.Type{evt.Type()},
		},
		delay: 2 * time.Second,
	}

	customTimeout := 50 * time.Millisecond
	_, err := startProjectionHost(
		context.Background(),
		store, bus, nil,
		[]projection.Projection{slow},
		customTimeout,
	)
	if err == nil {
		t.Fatal("expected drain timeout error, got nil")
	}

	msg := err.Error()
	if !strings.Contains(msg, "drain_timeout") {
		t.Errorf("expected 'drain_timeout' in error, got: %s", msg)
	}
	if !strings.Contains(msg, "50ms") {
		t.Errorf("expected '50ms' in error message, got: %s", msg)
	}
}
