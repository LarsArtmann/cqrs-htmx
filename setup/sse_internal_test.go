package setup

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

func TestBundle_SSEHandlerReplay(t *testing.T) {
	t.Parallel()

	store := memorystorage.NewMemoryStore()
	bus := newFakeBus()

	aggID := id.NewStreamID()
	ref := id.NewStreamRef("User", aggID)

	evt1, err := event.New("user.created", aggID, "User", event.Version(1), struct{}{})
	if err != nil {
		t.Fatalf("create event 1: %v", err)
	}

	evt2, err := event.New("user.updated", aggID, "User", event.Version(2), struct{}{})
	if err != nil {
		t.Fatalf("create event 2: %v", err)
	}

	if err := store.Save(context.Background(), ref, []event.Event{evt1, evt2}, event.Version(0)); err != nil {
		t.Fatalf("Save: %v", err)
	}

	bundle, err := New(Config{
		Title:      "Replay",
		SSEPath:    "/sse",
		EventStore: store,
		EventBus:   bus,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = bundle.Close() }()

	if bundle.sseStore == nil {
		t.Fatal("sseStore must be built when EventStore implements event.Journal")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/sse", nil)
	req = req.WithContext(ctx)
	req.Header.Set("Last-Event-ID", evt1.ID().String())

	// Inject an authenticated user so requireSession passes.
	req = req.WithContext(usermgmt.WithUser(req.Context(), &usermgmt.User{ID: usermgmt.GenerateUserID()}))

	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		bundle.sseHandler().ServeHTTP(rec, req)
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()
	<-done

	body := rec.Body.String()

	if !strings.Contains(body, evt2.ID().String()) {
		t.Errorf("body should contain replayed event 2 ID %q\nbody:\n%s", evt2.ID().String(), body)
	}

	if strings.Contains(body, evt1.ID().String()) {
		t.Errorf("body should NOT contain the cursor event 1 ID %q\nbody:\n%s", evt1.ID().String(), body)
	}
}

func TestBundle_SSEHandlerInitialBackfill(t *testing.T) {
	t.Parallel()

	store := memorystorage.NewMemoryStore()
	bus := newFakeBus()

	aggID := id.NewStreamID()
	ref := id.NewStreamRef("User", aggID)

	evt, err := event.New("user.created", aggID, "User", event.Version(1), struct{}{})
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	if err := store.Save(context.Background(), ref, []event.Event{evt}, event.Version(0)); err != nil {
		t.Fatalf("Save: %v", err)
	}

	bundle, err := New(Config{
		Title:      "Backfill",
		SSEPath:    "/sse",
		EventStore: store,
		EventBus:   bus,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = bundle.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/sse", nil)
	req = req.WithContext(ctx)
	req = req.WithContext(usermgmt.WithUser(req.Context(), &usermgmt.User{ID: usermgmt.GenerateUserID()}))

	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		bundle.sseHandler().ServeHTTP(rec, req)
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()
	<-done

	body := rec.Body.String()

	if !strings.Contains(body, evt.ID().String()) {
		t.Errorf("body should contain backfilled event ID %q\nbody:\n%s", evt.ID().String(), body)
	}
}

// newFakeBus is a minimal event.Bus that satisfies SubscribeAll for the bridge.
// It intentionally does nothing on Publish because these tests only exercise
// replay, not the live bridge.
type fakeBus struct{}

func newFakeBus() *fakeBus { return &fakeBus{} }

func (f *fakeBus) Publish(_ context.Context, _ ...event.Event) error { return nil }

func (f *fakeBus) Subscribe(_ event.Type, _ event.Handler) error { return nil }

func (f *fakeBus) SubscribeAll(_ event.Handler) error { return nil }

func (f *fakeBus) Use(_ ...event.Middleware) error { return nil }

func (f *fakeBus) UsePublish(_ ...event.PublishMiddleware) error { return nil }
