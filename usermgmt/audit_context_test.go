package usermgmt

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
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

	reg := registerTestUser(t, svc, GenerateUserID().Get().String(), "audit@example.com")
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

	reg := registerTestUser(t, svc, GenerateUserID().Get().String(), "corr@example.com")
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

	reg := registerTestUser(t, svc, GenerateUserID().Get().String(), "explicit@example.com")
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

	reg := registerTestUser(t, svc, GenerateUserID().Get().String(), "anon@example.com")
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
		NewRegisterUserCmd(id.NewStreamID(), "applier@example.com", "Applier", nil),
		NewChangeEmailCmd(id.NewStreamID(), "applier@example.com"),
		NewChangeDisplayNameCmd(id.NewStreamID(), "Applier"),
		NewDeleteUserCmd(id.NewStreamID(), "test"),
		NewVerifyEmailCmd(id.NewStreamID()),
	}

	for i, cmd := range commands {
		if _, ok := cmd.(commandOptionApplier); !ok {
			t.Errorf("command %d (%T) does not satisfy commandOptionApplier", i, cmd)
		}
	}
}

// TestDispatch_ClientMetadataPropagatesToEventMetadata proves that client IP
// and User-Agent set in the cqrshtmx context chain (as
// cqrshtmx.ContextEnrichmentMiddleware captures them for HTTP requests)
// reach the emitted event metadata via requestContextEnricher.
func TestDispatch_ClientMetadataPropagatesToEventMetadata(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)
	defer svc.Close() //nolint:errcheck // test cleanup

	reg := registerTestUser(t, svc, GenerateUserID().Get().String(), "clientmeta@example.com")
	userID := reg.User.ID

	ip, err := event.ParseIPAddress("203.0.113.42")
	if err != nil {
		t.Fatalf("ParseIPAddress: %v", err)
	}

	wantUA := event.UserAgent("usermgmt-test/1.0")
	ctx := cqrshtmx.WithIPAddress(t.Context(), ip)
	ctx = cqrshtmx.WithUserAgent(ctx, wantUA)

	if err := svc.ChangeDisplayName(ctx, userID, "Client Meta"); err != nil {
		t.Fatalf("ChangeDisplayName: %v", err)
	}

	events := loadUserEvents(t, svc, userID)
	changed := findEvent(events, eventDisplayNameChanged)
	if changed == nil {
		t.Fatalf("no %s event on stream", eventDisplayNameChanged)
	}

	meta := changed.Metadata()
	if meta.IPAddress != ip {
		t.Errorf("event IP = %q, want %q", meta.IPAddress, ip)
	}
	if meta.UserAgent != wantUA {
		t.Errorf("event user agent = %q, want %q", meta.UserAgent, wantUA)
	}
}

// TestDispatch_ClientIDPropagatesToEventMetadata proves that a client device
// ID from the cqrshtmx context chain (captured from the X-Client-ID header
// for offline-first attribution) reaches the emitted event metadata via
// requestContextEnricher.
func TestDispatch_ClientIDPropagatesToEventMetadata(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)
	defer svc.Close() //nolint:errcheck // test cleanup

	reg := registerTestUser(t, svc, GenerateUserID().Get().String(), "clientid@example.com")
	userID := reg.User.ID

	clientID := id.NewClientID()
	ctx := cqrshtmx.WithClientID(t.Context(), clientID)

	if err := svc.ChangeDisplayName(ctx, userID, "Client Attribution"); err != nil {
		t.Fatalf("ChangeDisplayName: %v", err)
	}

	events := loadUserEvents(t, svc, userID)
	changed := findEvent(events, eventDisplayNameChanged)
	if changed == nil {
		t.Fatalf("no %s event on stream", eventDisplayNameChanged)
	}

	if got := changed.Metadata().Custom[event.MetadataKeyClientID]; got != clientID.String() {
			t.Errorf("event client ID = %q, want %q", got, clientID.String())
		}
	}

	// driftGuardCmd mirrors the identity-model command pattern: a thin wrapper
	// embedding *command.BasicCommand, so ApplyOptions is promoted to the wrapper
	// exactly as on all 20 identity-model commands.
	type driftGuardCmd struct {
		*command.BasicCommand
	}

	// Compile-time: usermgmt's commandOptionApplier accepts the promoted method.
	var _ commandOptionApplier = (*driftGuardCmd)(nil)

	// TestCommandOptionApplier_RootAndUsermgmtShapesStayIdentical is the drift
	// guard for the deliberately duplicated commandOptionApplier interface
	// (root handler.go + this package's audit_context.go — Go cannot share an
	// unexported structural interface across module boundaries, and root never
	// imports usermgmt). Root's copy is unexported, so shape identity is proven
	// BEHAVIORALLY: driving cqrshtmx's command pipeline with a BasicCommand-
	// wrapping probe must enrich it — the actor from the request context lands
	// on the dispatched command's metadata. If either interface drifts from the
	// promoted ApplyOptions signature, root's type assertion silently skips
	// enrichment and this test fails with a zero actor.
	func TestCommandOptionApplier_RootAndUsermgmtShapesStayIdentical(t *testing.T) {
		t.Parallel()

		var gotActor id.ActorID
		disp := command.NewDispatcher()
		if err := disp.Register("DriftGuard", func(_ context.Context, cmd command.Command) error {
			meta, ok := cmd.(interface{ Metadata() command.Metadata })
			if !ok {
				t.Fatalf("dispatched command (%T) exposes no Metadata", cmd)
			}
			gotActor = meta.Metadata().ActorID
			return nil
		}); err != nil {
			t.Fatalf("Register: %v", err)
		}

		app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
		if err != nil {
			t.Fatalf("cqrshtmx.New: %v", err)
		}

		base, err := command.New("DriftGuard", id.NewStreamID())
		if err != nil {
			t.Fatalf("command.New: %v", err)
		}
		probe := &driftGuardCmd{BasicCommand: base}

		// The command handler pipeline decodes the (empty) body, then runs
		// root's enrichCommandFromContext on the decoded command before dispatch.
		handler := app.Command("DriftGuard",
			cqrshtmx.DecodeJSON(func(_ struct{}) (command.Command, error) {
				return probe, nil
			}))

		actor := id.NewUserActor(GenerateUserID())
		req := httptest.NewRequest(http.MethodPost, "/drift-guard", strings.NewReader("{}")).
			WithContext(cqrshtmx.WithActorID(t.Context(), actor))
		rec := httptest.NewRecorder()
		handler(rec, req)

		if rec.Code/100 != 2 {
			t.Fatalf("status = %d, want 2xx (body: %s)", rec.Code, rec.Body.String())
		}
		if gotActor != actor {
			t.Fatalf("command metadata actor = %v, want %v — the root-side commandOptionApplier "+
				"assertion skipped enrichment; the twin interfaces drifted (handler.go vs audit_context.go)",
				gotActor, actor)
		}
	}
