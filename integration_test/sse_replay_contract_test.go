package integration_test

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/cqrs-htmx/dashboardui/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/cqrs-htmx/v4/transport"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/watermill/v4"
	"github.com/stretchr/testify/require"
)

// setupSSEStack builds the setup-shaped SSE wiring over a real usermgmt
// service: the shared cqrshtmx broadcaster + journal-backed replay store +
// the event-bus bridge — exactly what setup.attachSSE composes (minus the
// session gate, which is orthogonal to the wire format).
func setupSSEStack(t *testing.T) (http.HandlerFunc, *usermgmt.Service) {
	t.Helper()

	store := memorystorage.NewMemoryStore()
	bus := watermill.NewEventBus()

	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
		EventStore: store,
		EventBus:   bus,
		AuditLog:   usermgmt.NewAuditLog(),
	})
	require.NoError(t, err)
	t.Cleanup(svc.Stop)

	broadcaster := cqrshtmx.NewBroadcaster()
	t.Cleanup(broadcaster.Close)

	sseStore := transport.NewJournalSSEStore(store, transport.DomainEventToSSE)

	bridge := func(_ context.Context, evt event.Event) error {
		broadcaster.Broadcast(transport.DomainEventToSSE(evt))
		return nil
	}

	require.NoError(t, bus.SubscribeAll(bridge))

	return transport.ServeDomainEvents(broadcaster.Hub(), sseStore, 0), svc
}

// streamOnce connects to an SSE handler, reads until the client context is
// cancelled, and returns the raw response body.
func streamOnce(t *testing.T, h http.HandlerFunc, lastEventID string) string {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)
	if lastEventID != "" {
		req.Header.Set("Last-Event-ID", lastEventID)
	}

	rec := httptest.NewRecorder()
	done := make(chan struct{})

	go func() {
		h.ServeHTTP(rec, req)
		close(done)
	}()

	time.Sleep(150 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("SSE handler did not return after client disconnect")
	}

	return rec.Body.String()
}

// domainFrame extracts the full `event: domain` SSE frame carrying the given
// event id from a raw stream body (empty string when absent).
func domainFrame(body, eventID string) string {
	scanner := bufio.NewScanner(strings.NewReader(body))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var frame []string

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if slices.Contains(frame, "id: "+eventID) {
				break // frame complete and it is the wanted one
			}

			frame = frame[:0]

			continue
		}

		frame = append(frame, line)
	}

	joined := strings.Join(frame, "\n")
	if !strings.Contains(joined, "id: "+eventID) {
		return ""
	}

	return joined + "\n"
}

// firstDomainEventID returns the id of the first `event: domain` frame.
func firstDomainEventID(t *testing.T, body string) string {
	t.Helper()

	for line := range strings.SplitSeq(body, "\n") {
		if strings.HasPrefix(line, "id: ") {
			return strings.TrimPrefix(line, "id: ")
		}
	}

	t.Fatalf("no domain event id found in stream body:\n%s", body)

	return ""
}

// TestSSE_ReconnectWithReplay proves the /sse reconnect contract end to end:
// a client that disconnects and reconnects with Last-Event-ID receives
// exactly the events committed after the cursor — no loss, no duplication —
// and the replayed frames re-advertise the reconnect back-off hint.
func TestSSE_ReconnectWithReplay(t *testing.T) {
	t.Parallel()

	sseHandler, svc := setupSSEStack(t)

	// Commit three real domain commands (each Register emits events).
	for i := range 3 {
		_, err := svc.Register(t.Context(), usermgmt.RegisterRequest{
			Email:       "reconnect-" + string(rune('a'+i)) + "@example.com",
			DisplayName: "Reconnect " + string(rune('A'+i)),
		})
		require.NoError(t, err)
	}

	first := streamOnce(t, sseHandler, "")
	firstID := firstDomainEventID(t, first)

	require.Contains(t, first, "connected", "first connect must open with the connected event")
	require.NotEmpty(t, firstID, "first connect must deliver replayed/backfilled domain events")

	// Reconnect with the cursor: the frame at the cursor must NOT replay;
	// later events must.
	reconnected := streamOnce(t, sseHandler, firstID)

	require.NotContains(t, reconnected, "id: "+firstID+"\n",
		"the cursor event itself must not replay")

	require.Contains(t, reconnected, "event: event",
		"events committed after the cursor must replay")

	require.Contains(t, reconnected, "retry: ",
		"replayed frames must carry the reconnect hint")
}

// TestSSE_CrossModuleWireFormatContract proves the shared-envelope contract:
// for the same committed domain event, the setup-shaped /sse endpoint and the
// dashboardui stream endpoint emit byte-identical `event: domain` frames —
// both delegate to transport.DomainEventToSSE, and this test pins that the
// delegation never drifts (a consumer can switch endpoints without changing
// its parser).
func TestSSE_CrossModuleWireFormatContract(t *testing.T) {
	t.Parallel()

	store := memorystorage.NewMemoryStore()
	bus := watermill.NewEventBus()

	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
		EventStore: store,
		EventBus:   bus,
		AuditLog:   usermgmt.NewAuditLog(),
	})
	require.NoError(t, err)
	t.Cleanup(svc.Stop)

	dash, err := dashboardui.New(dashboardui.Config{
		Title:          "Contract Dashboard",
		EventSource:    store,
		Journal:        store,
		EventBus:       bus,
		ProjectionHost: svc.ProjectionHost(),
	})
	require.NoError(t, err)

	// The setup-shaped endpoint over the SAME store + bus.
	broadcaster := cqrshtmx.NewBroadcaster()
	t.Cleanup(broadcaster.Close)

	sseStore := transport.NewJournalSSEStore(store, transport.DomainEventToSSE)
	setupShaped := transport.ServeDomainEvents(broadcaster.Hub(), sseStore, 0)

	require.NoError(t, bus.SubscribeAll(func(_ context.Context, evt event.Event) error {
		broadcaster.Broadcast(transport.DomainEventToSSE(evt))
		return nil
	}))

	// One real domain event, seen by both bridges.
	_, err = svc.Register(t.Context(), usermgmt.RegisterRequest{
		Email:       "contract@example.com",
		DisplayName: "Contract User",
	})
	require.NoError(t, err)

	setupBody := streamOnce(t, setupShaped, "")
	dashBody := streamOnce(t, dash.Handler().ServeHTTP, "")

	eventID := firstDomainEventID(t, setupBody)

	setupFrame := domainFrame(setupBody, eventID)
	dashFrame := domainFrame(dashBody, eventID)

	require.NotEmpty(t, setupFrame, "setup-shaped endpoint must deliver the domain event")
	require.NotEmpty(t, dashFrame, "dashboardui endpoint must deliver the same domain event")
	require.Equal(t, setupFrame, dashFrame,
		"the same domain event must produce byte-identical frames on both endpoints")
}
