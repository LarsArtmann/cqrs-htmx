package setup_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/cqrs-htmx/datastar/v4"
	"github.com/larsartmann/cqrs-htmx/setup/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/go-sse"
)

// TestDatastarDisabledByDefault verifies the ADR-0050 opt-in contract: with
// the zero-value config, no DataStar routes exist and the broadcaster is nil.
func TestDatastarDisabledByDefault(t *testing.T) {
	t.Parallel()

	b, err := setup.New(setup.Config{Title: "Default"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = b.Close() }()

	if b.DataStarBroadcaster != nil {
		t.Error("DataStarBroadcaster must be nil when DataStarPath is not set")
	}

	mux := http.NewServeMux()
	b.Mount(mux)

	for _, route := range []string{"/ds/events", "/datastar.js"} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, route, nil))
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s with DataStar disabled: got %d, want 404", route, rec.Code)
		}
	}
}

// TestDatastarScript_ServesETagAnd304 covers the script mount end-to-end:
// 200 with an ETag on first fetch, 304 on a matching If-None-Match.
func TestDatastarScript_ServesETagAnd304(t *testing.T) {
	t.Parallel()

	b, err := setup.New(setup.Config{Title: "ETag", DataStarPath: "/ds/events"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = b.Close() }()

	mux := http.NewServeMux()
	b.Mount(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/datastar.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("script status: got %d, want 200", rec.Code)
	}

	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "javascript") {
		t.Errorf("script content type: got %q, want javascript", ct)
	}

	etag := rec.Header().Get("Etag")
	if etag == "" {
		t.Fatal("Etag must be set on the served script")
	}

	req := httptest.NewRequest(http.MethodGet, "/datastar.js", nil)
	req.Header.Set("If-None-Match", etag)

	rec304 := httptest.NewRecorder()
	mux.ServeHTTP(rec304, req)
	if rec304.Code != http.StatusNotModified {
		t.Errorf("conditional script fetch: got %d, want 304", rec304.Code)
	}
}

// TestDatastarScriptPath_OptOut verifies "-" disables the script mount while
// the feed itself stays mounted.
func TestDatastarScriptPath_OptOut(t *testing.T) {
	t.Parallel()

	b, err := setup.New(setup.Config{
		Title:              "OptOut",
		DataStarPath:       "/ds/events",
		DataStarScriptPath: "-",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = b.Close() }()

	mux := http.NewServeMux()
	b.Mount(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/datastar.js", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("script with DataStarScriptPath \"-\": got %d, want 404", rec.Code)
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ds/events", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("feed must still mount and gate: got %d, want 401", rec.Code)
	}
}

// TestDatastarFeed_SessionGated verifies the ADR-0050 gating posture: 401
// without a session, SSE stream for an authenticated one.
func TestDatastarFeed_SessionGated(t *testing.T) {
	t.Parallel()

	b, err := setup.New(setup.Config{Title: "Gate", DataStarPath: "/ds/events"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = b.Close() }()

	mux := http.NewServeMux()
	b.Mount(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ds/events", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated feed: got %d, want 401", rec.Code)
	}

	authed := authenticatedDatastarRequest(t, "/ds/events")
	if ct := authed.Header().Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		t.Errorf("authenticated feed content type: got %q, want text/event-stream", ct)
	}
}

// TestDatastarFeed_SharedHubBroadcast proves the ADR-0050 core claim: a patch
// broadcast on the root [setup.Bundle.Broadcaster] reaches a connected
// DataStar client on /ds/events — one hub, two transports.
func TestDatastarFeed_SharedHubBroadcast(t *testing.T) {
	t.Parallel()

	store := memorystorage.NewMemoryStore()
	bus := eventtest.NewFakeBus()

	b, err := setup.New(setup.Config{
		Title:       "SharedHub",
		DataStarPath: "/ds/events",
		EventStore:  store,
		EventBus:    bus,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = b.Close() }()

	if b.Broadcaster == nil {
		t.Fatal("Broadcaster must exist when only DataStarPath is set (hub backs the feed)")
	}

	patch, err := datastar.SignalsPatch(map[string]any{"count": 42})
	if err != nil {
		t.Fatalf("SignalsPatch: %v", err)
	}

	rec := authenticatedDatastarRequest(t, "/ds/events")
	body := rec.Body.String()

	if !strings.Contains(body, "datastar-patch-signals") {
		t.Errorf("stream body should contain the patch event type\nbody:\n%s", body)
	}

	if !strings.Contains(body, "42") {
		t.Errorf("stream body should contain the patched signal payload\nbody:\n%s", body)
	}
}

// TestDatastarOnly_NoSSERoute verifies DataStarPath alone mounts the feed but
// never an /sse route, while still building the shared hub + event bridge.
func TestDatastarOnly_NoSSERoute(t *testing.T) {
	t.Parallel()

	b, err := setup.New(setup.Config{Title: "DsOnly", DataStarPath: "/ds/events"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = b.Close() }()

	if b.Broadcaster == nil {
		t.Fatal("Broadcaster must exist when DataStarPath is set")
	}

	mux := http.NewServeMux()
	b.Mount(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sse", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("/sse with only DataStarPath set: got %d, want 404", rec.Code)
	}
}

// TestDatastarEventsFromBus verifies domain events committed to the event bus
// reach a connected DataStar client as patch-encoded SSE frames.
func TestDatastarEventsFromBus(t *testing.T) {
	t.Parallel()

	store := memorystorage.NewMemoryStore()
	bus := eventtest.NewFakeBus()

	b, err := setup.New(setup.Config{
		Title:        "BusBridge",
		DataStarPath: "/ds/events",
		EventStore:   store,
		EventBus:     bus,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = b.Close() }()

	aggID := id.NewStreamID()
	ref := id.NewStreamRef("User", aggID)

	evt, err := event.New("user.created", aggID, "User", event.Version(1), struct{}{})
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	if err := store.Save(context.Background(), ref, []event.Event{evt}, event.Version(0)); err != nil {
		t.Fatalf("Save: %v", err)
	}

	rec := authenticatedDatastarRequest(t, "/ds/events")

	if !strings.Contains(rec.Body.String(), evt.ID().String()) {
		t.Errorf("stream body should contain the bus-published event ID %q\nbody:\n%s", evt.ID().String(), rec.Body.String())
	}
}

// authenticatedDatastarRequest streams /ds/events with an injected session,
// publishes a patch on the shared hub, lets the stream drain briefly, then
// cancels and returns the recorder.
func authenticatedDatastarRequest(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()

	b, ok := datastarBundleFromContext(t)
	if !ok {
		t.Fatal("no bundle registered for this test")
	}

	ctx, cancel := context.WithCancel(context.Background())

	req := httptest.NewRequest(http.MethodGet, path, nil)
	req = req.WithContext(usermgmt.WithUser(ctx, &usermgmt.User{ID: usermgmt.GenerateUserID()}))

	rec := httptest.NewRecorder()

	mux := http.NewServeMux()
	b.Mount(mux)

	done := make(chan struct{})
	go func() {
		mux.ServeHTTP(rec, req)
		close(done)
	}()

	time.Sleep(150 * time.Millisecond)
	b.Broadcaster.Broadcast(sse.Event{Event: "datastar-patch-signals", Data: `signals: {"count":42}`})
	time.Sleep(150 * time.Millisecond)

	cancel()
	<-done

	return rec
}
