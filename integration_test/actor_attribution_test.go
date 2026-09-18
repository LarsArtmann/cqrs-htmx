package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/larsartmann/cqrs-htmx/dashboardui/v4"
	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/watermill/v4"
	"github.com/stretchr/testify/require"
)

// TestActorAttribution_VisibleInAuditViews is the end-to-end proof of the
// audit-trail chain (usermgmt's dispatcher middleware + repository enricher):
// a mutation performed with the session user in the context produces events
// whose metadata carries the acting user — and BOTH audit surfaces that
// consumers actually look at surface it:
//
//  1. usermgmt's own AuditLog projection (ActorID field on entries)
//  2. dashboardui's event detail view (the "Actor ID" row on /events/{id})
//
// Before the audit chain landed, both views rendered the library's own
// activity as anonymous (actor zero) — this test pins the fix.
func TestActorAttribution_VisibleInAuditViews(t *testing.T) {
	t.Parallel()

	store := memorystorage.NewMemoryStore()
	bus := watermill.NewEventBus()

	auditLog := usermgmt.NewAuditLog()
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
		EventStore: store,
		EventBus:   bus,
		AuditLog:   auditLog,
	})
	require.NoError(t, err)
	t.Cleanup(svc.Stop)

	dash, err := dashboardui.New(dashboardui.Config{
		Title:          "Audit Attribution Test",
		EventSource:    store,
		Journal:        store,
		EventBus:       bus,
		ProjectionHost: svc.ProjectionHost(),
	})
	require.NoError(t, err)

	// Seed: register a user, then mutate with the session user in context —
	// exactly what the session middleware produces for authenticated calls.
	ctx := context.Background()
	reg, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID:    identitymodel.GenerateUserID(),
		Email: "attribution@example.com",
	})
	require.NoError(t, err)
	userID := reg.User.ID

	user, err := svc.GetUser(ctx, userID)
	require.NoError(t, err)

	if err := svc.ChangeDisplayName(usermgmt.WithUser(ctx, user), userID, "Attributed"); err != nil {
		t.Fatalf("ChangeDisplayName: %v", err)
	}

	wantActor := identitymodel.ActorIDFromUser(userID)
	require.False(t, wantActor.IsZero(), "actor under test must be non-zero")

	aggID, err := id.ParseStreamID(userID.Get().String())
	require.NoError(t, err)

	events, err := store.Load(ctx, id.NewStreamRef("User", aggID))
	require.NoError(t, err)
	require.NotEmpty(t, events)

	var displayChangedID string
	for _, evt := range events {
		if evt.Type().String() == "DisplayNameChanged" {
			displayChangedID = evt.ID().String()

			break
		}
	}
	require.NotEmpty(t, displayChangedID, "DisplayNameChanged event not found on stream")

	// Surface 1: usermgmt AuditLog entries carry the actor.
	requireAuditEntryWithActor(t, auditLog, aggID, wantActor)

	// Surface 2: dashboardui event detail renders the actor.
	handler := dash.Handler()

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/events/"+displayChangedID, nil))
	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	require.Contains(
		t,
		w.Body.String(),
		wantActor.PrefixedString(),
		"dashboard event detail must render the acting user's actor ID",
	)
}

// requireAuditEntryWithActor asserts the usermgmt audit surface of the
// attribution chain: at least one audit entry for the stream carries the
// expected actor.
func requireAuditEntryWithActor(t *testing.T, auditLog *usermgmt.AuditLog, aggID id.StreamID, wantActor id.ActorID) {
	t.Helper()

	for _, entry := range auditLog.EntriesFor(aggID) {
		if entry.ActorID == wantActor {
			return
		}
	}
	t.Errorf("no audit entry carries actor %s", wantActor.PrefixedString())
}
