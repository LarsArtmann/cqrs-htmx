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

	require.NoError(t, svc.ChangeDisplayName(usermgmt.WithUser(ctx, user), userID, "Attributed"))

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
	requireDashboardEventRendersActor(t, dash, displayChangedID, wantActor)
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

// TestActorAttribution_BotDispatchIsNeverAnonymous is the zero-actor guard
// for the machine path: a command dispatched under a token-authenticated bot
// context — exactly what NewAPITokenMiddleware produces for a Bearer call —
// must carry the bot's machine actor in the audit journal. Before the actor
// bridge, this dispatch landed as a successful, silently anonymous audit
// entry: the worst failure mode, because nothing alerts.
func TestActorAttribution_BotDispatchIsNeverAnonymous(t *testing.T) {
	t.Parallel()

	store := memorystorage.NewMemoryStore()
	bus := watermill.NewEventBus()
	auditLog := usermgmt.NewAuditLog()
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
		EventStore:  store,
		EventBus:    bus,
		AuditLog:    auditLog,
		TokenPepper: usermgmt.TokenPepper("test-pepper-32-bytes-long-xxxxxxx"),
	})
	require.NoError(t, err)
	t.Cleanup(svc.Stop)

	ctx := context.Background()
	reg, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID:    identitymodel.GenerateUserID(),
		Email: "bot-dispatch@example.com",
	})
	require.NoError(t, err)
	userID := reg.User.ID

	botReg, err := svc.RegisterBot(ctx, usermgmt.RegisterBotRequest{
		ID:      usermgmt.NewBotID("01JXBOT00000000000000000005"),
		Name:    "attribution-bot",
		OwnerID: userID,
		Scopes:  []string{"read"},
	})
	require.NoError(t, err)

	handler := usermgmt.NewAPITokenMiddleware(svc)(http.HandlerFunc(
		func(_ http.ResponseWriter, r *http.Request) {
			bot, ok := usermgmt.BotFromContext(r.Context())
			require.True(t, ok, "authenticated bot must be in the request context")

			actor := identitymodel.ActorIDFromBot(bot.ID)
			require.False(t, actor.IsZero(), "the bridged bot actor must never be zero")

			require.NoError(t, svc.ChangeDisplayName(r.Context(), userID, "DispatchedByBot"))
		}))
	req := httptest.NewRequest(http.MethodPost, "/dispatch-via-bot", nil)
	req.Header.Set("Authorization", "Bearer "+botReg.Token)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	wantActor := identitymodel.ActorIDFromBot(botReg.Bot.ID)
	userAggID, err := id.ParseStreamID(userID.Get().String())
	require.NoError(t, err)
	requireAuditEntryWithActor(t, auditLog, userAggID, wantActor)
}

// requireDashboardEventRendersActor asserts the dashboardui surface of the
// attribution chain: the /events/{id} detail page renders the expected actor.
func requireDashboardEventRendersActor(
	t *testing.T,
	dash *dashboardui.Dashboard,
	eventID string,
	wantActor id.ActorID,
) {
	t.Helper()

	handler := dash.Handler()
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/events/"+eventID, nil))
	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	require.Contains(
		t,
		w.Body.String(),
		wantActor.PrefixedString(),
		"dashboard event detail must render the acting user's actor ID",
	)
}
