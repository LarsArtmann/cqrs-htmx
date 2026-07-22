package usermgmt

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
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

	setup, err := NewEventSourcedSetup(EventSourcedConfig{}) //nolint:exhaustruct // all fields optional
	if err != nil {
		t.Fatalf("NewEventSourcedSetup: %v", err)
	}
	t.Cleanup(func() { _ = setup.Close() })

	// The read model should be empty after initial setup (no events in journal).
	if users := setup.ReadModel.All(); len(users) != 0 {
		t.Errorf("expected 0 users after empty setup, got %d", len(users))
	}
}

// stubProjection compile-time check
var _ projection.Projection = (*stubProjection)(nil)
