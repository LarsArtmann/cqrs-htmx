package usermgmt

import (
	"context"
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// loadUserEvents loads all events on the given user's stream, failing the
// test on error.
func loadUserEvents(t *testing.T, svc *Service, userID UserID) []event.Event {
	t.Helper()

	aggID, err := aggIDFromUser(userID)
	if err != nil {
		t.Fatalf("aggIDFromUser: %v", err)
	}

	events, err := svc.Journal().Load(
		t.Context(),
		id.NewStreamRef(aggregateTypeUser, aggID),
	)
	if err != nil {
		t.Fatalf("Journal.Load: %v", err)
	}

	return events
}

// findEvent returns the first event of the given type, or nil.
func findEvent(events []event.Event, eventType event.Type) event.Event {
	for _, evt := range events {
		if evt.Type() == eventType {
			return evt
		}
	}
	return nil
}

// TestDispatch_EventsCarryActorAndCausation proves the full audit chain:
// a service call made with the session user in the context produces events
// whose metadata records both WHO issued the command (actor, bridged from
// usermgmt's session context onto the command and back into the handler
// context) and WHICH command caused the event (causation, stamped from the
// command's type and ID). This also pins the middleware ordering: enrichment
// must run before the actor lift, otherwise the command metadata would not
// carry the actor when middleware.CommandActorContext reads it.
func TestDispatch_EventsCarryActorAndCausation(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)
	defer svc.Close() //nolint:errcheck // test cleanup

	reg := registerTestUser(t, svc, "01HXQACT0RAUD1T0000000000", "audit@example.com")
	userID := reg.User.ID

	user, err := svc.GetUser(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}

	ctx := WithUser(t.Context(), user)
	if err := svc.ChangeDisplayName(ctx, userID, "Audit Trail"); err != nil {
		t.Fatalf("ChangeDisplayName: %v", err)
	}

	events := loadUserEvents(t, svc, userID)
	changed := findEvent(events, eventDisplayNameChanged)
	if changed == nil {
		t.Fatalf("no %s event on stream; got %d events", eventDisplayNameChanged, len(events))
	}

	meta := changed.Metadata()

	wantActor := ActorIDFromUser(userID)
	if meta.ActorID != wantActor {
		t.Errorf("event actor = %q, want %q (bridged from session user)", meta.ActorID, wantActor)
	}

	if meta.Causation == nil {
		t.Fatal("event causation = nil, want command type + ID")
	}
	if meta.Causation.CommandType != string(cmdChangeDisplayName) {
		t.Errorf(
			"causation command type = %q, want %q",
			meta.Causation.CommandType,
			cmdChangeDisplayName,
		)
	}
	if meta.Causation.CommandID.IsZero() {
		t.Error("causation command ID = zero, want the causing command's ID")
	}
}

// TestDispatch_CorrelationIDPropagatesToEventMetadata proves that a
// correlation ID set in the cqrshtmx context chain (as
// cqrshtmx.ContextEnrichmentMiddleware does for HTTP requests) reaches the
// command metadata and from there the emitted event.
func TestDispatch_CorrelationIDPropagatesToEventMetadata(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)
	defer svc.Close() //nolint:errcheck // test cleanup

	reg := registerTestUser(t, svc, "01HXQACT0RAUD1T0000000001", "corr@example.com")
	userID := reg.User.ID

	cid := cqrshtmx.NewCorrelationID()
	ctx := cqrshtmx.WithCorrelationID(t.Context(), cid)

	if err := svc.ChangeEmail(ctx, userID, "correlated@example.com"); err != nil {
		t.Fatalf("ChangeEmail: %v", err)
	}

	events := loadUserEvents(t, svc, userID)
	changed := findEvent(events, eventEmailChanged)
	if changed == nil {
		t.Fatalf("no %s event on stream; got %d events", eventEmailChanged, len(events))
	}

	if got := changed.Metadata().CorrelationID; got != cid {
		t.Errorf("event correlation ID = %q, want %q", got, cid)
	}
}

// TestDispatch_ConsumerActorWins proves the bridge never overwrites
// consumer-set identity: an explicit actor in the cqrshtmx context chain
// (e.g. set by impersonation or service middleware) survives dispatch even
// when a session user is also present.
func TestDispatch_ConsumerActorWins(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)
	defer svc.Close() //nolint:errcheck // test cleanup

	reg := registerTestUser(t, svc, "01HXQACT0RAUD1T0000000002", "explicit@example.com")
	userID := reg.User.ID

	user, err := svc.GetUser(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}

	systemActor := id.NewSystemActor("test-runner")
	ctx := WithUser(cqrshtmx.WithActorID(t.Context(), systemActor), user)

	if err := svc.ChangeDisplayName(ctx, userID, "Explicit Actor"); err != nil {
		t.Fatalf("ChangeDisplayName: %v", err)
	}

	events := loadUserEvents(t, svc, userID)
	changed := findEvent(events, eventDisplayNameChanged)
	if changed == nil {
		t.Fatalf("no %s event on stream", eventDisplayNameChanged)
	}

	if got := changed.Metadata().ActorID; got != systemActor {
		t.Errorf("event actor = %q, want consumer-set %q", got, systemActor)
	}
}

// TestDispatch_UnauthenticatedContextStillRecordsCausation proves the
// causation stamp is independent of authentication: even with no identity in
// the context, events record which command caused them (actor stays zero).
func TestDispatch_UnauthenticatedContextStillRecordsCausation(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)
	defer svc.Close() //nolint:errcheck // test cleanup

	reg := registerTestUser(t, svc, "01HXQACT0RAUD1T0000000003", "anon@example.com")
	userID := reg.User.ID

	if err := svc.ChangeDisplayName(t.Context(), userID, "Anonymous Cause"); err != nil {
		t.Fatalf("ChangeDisplayName: %v", err)
	}

	events := loadUserEvents(t, svc, userID)
	changed := findEvent(events, eventDisplayNameChanged)
	if changed == nil {
		t.Fatalf("no %s event on stream", eventDisplayNameChanged)
	}

	meta := changed.Metadata()
	if !meta.ActorID.IsZero() {
		t.Errorf("event actor = %q, want zero for unauthenticated context", meta.ActorID)
	}
	if meta.Causation == nil || meta.Causation.CommandType != string(cmdChangeDisplayName) {
		t.Errorf("event causation missing or wrong: %+v", meta.Causation)
	}
}

// TestCommandOptionApplier_SatisfiedByDomainCommands proves the structural
// interface covers every identity-model command: all 20 embed
// *command.BasicCommand, whose ApplyOptions promotes to the wrapper.
func TestCommandOptionApplier_SatisfiedByDomainCommands(t *testing.T) {
	t.Parallel()

	commands := []command.Command{
		NewRegisterUserCmd(GenerateStreamID(), "applier@example.com", "Applier", nil),
		NewChangeEmailCmd(GenerateStreamID(), "applier@example.com"),
		NewChangeDisplayNameCmd(GenerateStreamID(), "Applier"),
		NewDeleteUserCmd(GenerateStreamID(), "test"),
		NewVerifyEmailCmd(GenerateStreamID()),
	}

	for i, cmd := range commands {
		if _, ok := cmd.(commandOptionApplier); !ok {
			t.Errorf("command %d (%T) does not satisfy commandOptionApplier", i, cmd)
		}
	}
}
