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
)

// TestDatastarDisabledByDefault verifies the ADR-0050 opt-in contract: with
// the zero-value config, no DataStar routes exist and the broadcaster is nil.
func TestDatastarDisabledByDefault(t *testing.T) {
	t.Parallel()

	b, err := setup.New(setup.Config{Title: "Default", DisableLogin: true})
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

	// Login disabled so the "/" catch-all cannot mask route absence: the
	// assertions below need 404 to mean "pattern not registered".
	b, err := setup.New(setup.Config{
		Title:              "OptOut",
		DataStarPath:       "/ds/events",
		DataStarScriptPath: "-",
		DisableLogin:       true,
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
		t.Errorf(`script with DataStarScriptPath "-": got %d, want 404`, rec.Code)
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ds/events", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("feed must still mount and gate: got %d, want 401", rec.Code)
	}
}

// TestDatastarFeed_SessionGated verifies the ADR-0050 gating posture: 401
// without a session, an SSE stream for an authenticated one.
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

	streamed := streamDatastarFeed(t, b, "/ds/events", nil)
	if ct := streamed.Header().Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		t.Errorf("authenticated feed content type: got %q, want text/event-stream", ct)
	}
}

// TestDatastarFeed_SharedHubBroadcast proves the ADR-0050 core claim: a patch
// broadcast on the root [setup.Bundle.Broadcaster] reaches a connected
// DataStar client on /ds/events — one hub, two transports.
func TestDatastarFeed_SharedHubBroadcast(t *testing.T) {
	t.Parallel()

	b, err := setup.New(setup.Config{Title: "SharedHub", DataStarPath: "/ds/events"})
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

	rec := streamDatastarFeed(t, b, "/ds/events", func(b *setup.Bundle) {
		b.Broadcaster.Broadcast(patch.Event())
	})
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

	b, err := setup.New(setup.Config{Title: "DsOnly", DataStarPath: "/ds/events", DisableLogin: true})
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

// TestDatastarEventsFromBus verifies domain events published to the event bus
// reach a connected DataStar client as SSE frames on the shared hub.
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

	rec := streamDatastarFeed(t, b, "/ds/events", func(b *setup.Bundle) {
		if err := bus.Publish(context.Background(), evt); err != nil {
			t.Errorf("bus publish: %v", err)
		}
	})

	if !strings.Contains(rec.Body.String(), evt.ID().String()) {
		t.Errorf("stream body should contain the bus-published event ID %q\nbody:\n%s",
			evt.ID().String(), rec.Body.String())
	}
}

// streamDatastarFeed opens an authenticated /ds/events stream on a fresh mux,
// runs the optional broadcast hook while the stream is live, then cancels and
// returns the recorder. The hook runs after the subscriber is connected, so
// hub fan-out reaches the stream deterministically.
func streamDatastarFeed(
	t *testing.T,
	b *setup.Bundle,
	path string,
	during func(*setup.Bundle),
) *httptest.ResponseRecorder {
	t.Helper()

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

	if during != nil {
		during(b)
	}

	time.Sleep(150 * time.Millisecond)

	cancel()
	<-done

	return rec
}
