package usermgmt

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/projection/v3"
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

func makeTestEvent(eventType event.Type) event.Event {
	evt, _ := event.NewEvent(
		eventType,
		id.NewAggregateID(),
		"TestAggregate",
		1,
		[]byte("{}"),
	)

	return evt
}

func TestShouldDispatch(t *testing.T) {
	t.Parallel()

	types := []event.Type{"UserRegistered", "UserDeleted"}

	tests := []struct {
		name      string
		eventType event.Type
		want      bool
	}{
		{"matching first type", "UserRegistered", true},
		{"matching second type", "UserDeleted", true},
		{"non-matching type", "EmailChanged", false},
		{"empty type", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := slices.Contains(types, tt.eventType)
			if got != tt.want {
				t.Errorf("slices.Contains(%q) = %v, want %v", tt.eventType, got, tt.want)
			}
		})
	}
}

func TestShouldDispatch_EmptyEventTypes(t *testing.T) {
	t.Parallel()

	var types []event.Type
	if slices.Contains(types, "UserRegistered") {
		t.Error("slices.Contains should return false for nil slice")
	}
}

func TestBuildLiveHandler_DedupSkipsReplayedEvents(t *testing.T) {
	t.Parallel()

	proj := &stubProjection{
		name:  "test",
		types: []event.Type{"UserRegistered"},
	}

	replayedEvt := makeTestEvent("UserRegistered")
	seenIDs := map[id.EventID]struct{}{
		replayedEvt.ID(): {},
	}

	handler := buildLiveHandler([]projection.Projection{proj}, seenIDs)

	// The replayed event should be skipped (dedup)
	if err := handler(context.Background(), replayedEvt); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(proj.handled) != 0 {
		t.Errorf("dedup should have skipped replayed event, but handled %d events", len(proj.handled))
	}

	// A new event should be processed
	newEvt := makeTestEvent("UserRegistered")
	if err := handler(context.Background(), newEvt); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(proj.handled) != 1 {
		t.Errorf("handler should have processed 1 new event, got %d", len(proj.handled))
	}
}

func TestBuildLiveHandler_DispatchesToMatchingProjectionsOnly(t *testing.T) {
	t.Parallel()

	userProj := &stubProjection{name: "user", types: []event.Type{"UserRegistered"}}
	tenantProj := &stubProjection{name: "tenant", types: []event.Type{"TenantCreated"}}

	handler := buildLiveHandler(
		[]projection.Projection{userProj, tenantProj},
		map[id.EventID]struct{}{},
	)

	evt := makeTestEvent("UserRegistered")
	if err := handler(context.Background(), evt); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if len(userProj.handled) != 1 {
		t.Errorf("user projection should have handled 1 event, got %d", len(userProj.handled))
	}
	if len(tenantProj.handled) != 0 {
		t.Errorf("tenant projection should not have handled UserRegistered, got %d", len(tenantProj.handled))
	}
}

func TestBuildLiveHandler_ContinuesOnProjectionError(t *testing.T) {
	t.Parallel()

	failingProj := &stubProjection{
		name:      "failing",
		types:     []event.Type{"UserRegistered"},
		handleErr: errors.New("boom"),
	}
	healthyProj := &stubProjection{
		name:  "healthy",
		types: []event.Type{"UserRegistered"},
	}

	handler := buildLiveHandler(
		[]projection.Projection{failingProj, healthyProj},
		map[id.EventID]struct{}{},
	)

	evt := makeTestEvent("UserRegistered")
	if err := handler(context.Background(), evt); err != nil {
		t.Fatalf("handler should not propagate projection errors: %v", err)
	}

	// Both projections should have been attempted despite the first failing
	if len(failingProj.handled) != 1 {
		t.Errorf("failing projection should have been attempted, got %d", len(failingProj.handled))
	}
	if len(healthyProj.handled) != 1 {
		t.Errorf("healthy projection should still have been called after failure, got %d", len(healthyProj.handled))
	}
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
