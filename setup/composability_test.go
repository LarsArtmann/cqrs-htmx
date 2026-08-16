package setup_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/cqrs-htmx/setup/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

// TestNew_AdoptedService verifies the bring-your-own-service composition seam:
// when Config.Service is set, New adopts that exact instance and sources the
// shared infrastructure from it (Journal/EventBus accessors), instead of
// constructing a second service.
func TestNew_AdoptedService(t *testing.T) {
	t.Parallel()

	store := memorystorage.NewMemoryStore()
	bus := watermill.NewEventBus()

	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
		EventStore: store,
		EventBus:   bus,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	defer func() { _ = svc.Close() }()

	bundle, err := setup.New(setup.Config{
		Title:   "Adopted",
		Service: svc,
	})
	if err != nil {
		t.Fatalf("setup.New with adopted service: %v", err)
	}
	defer func() { _ = bundle.Close() }()

	if bundle.Service != svc {
		t.Fatal("bundle.Service must be the adopted instance, not a new one")
	}

	if bundle.Stores.EventStore != svc.Journal() {
		t.Error("Stores.EventStore must be the adopted service's journal")
	}

	if bundle.Stores.EventBus != svc.EventBus() {
		t.Error("Stores.EventBus must be the adopted service's event bus")
	}

	if bundle.Admin == nil || bundle.Dashboard == nil || bundle.Login == nil {
		t.Error("all default panels must attach to an adopted-service bundle")
	}

	mux := http.NewServeMux()
	bundle.Mount(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("health endpoint on adopted bundle: got status %d, want 200", rec.Code)
	}
}

// TestNew_AdoptedService_NotClosedByBundle verifies lifecycle ownership: an
// adopted service stays functional after Bundle.Close — the caller owns it.
func TestNew_AdoptedService_NotClosedByBundle(t *testing.T) {
	t.Parallel()

	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	defer func() { _ = svc.Close() }()

	bundle, err := setup.New(setup.Config{Service: svc})
	if err != nil {
		t.Fatalf("setup.New: %v", err)
	}

	if err := bundle.Close(); err != nil {
		t.Fatalf("bundle.Close: %v", err)
	}

	// The service must still work: register a user through it post-Close.
	_, err = svc.Register(context.Background(), usermgmt.RegisterRequest{
		ID:    usermgmt.GenerateUserID(),
		Email: "alive-after-close@example.com",
	})
	if err != nil {
		t.Fatalf("adopted service must remain functional after bundle.Close, register failed: %v", err)
	}

	// Close must also be idempotent with an adopted service.
	if err := bundle.Close(); err != nil {
		t.Fatalf("second bundle.Close: %v", err)
	}
}

// TestNew_AdoptedService_ConflictingFieldsRejected verifies the guard: fields
// that only apply when New builds the service are rejected, not silently
// ignored, when Service is set.
func TestNew_AdoptedService_ConflictingFieldsRejected(t *testing.T) {
	t.Parallel()

	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	defer func() { _ = svc.Close() }()

	_, err = setup.New(setup.Config{
		Service:    svc,
		EventStore: memorystorage.NewMemoryStore(),
	})
	if err == nil {
		t.Fatal("adopting a service while setting EventStore must be rejected")
	}

	if !strings.Contains(err.Error(), "EventStore") {
		t.Errorf("rejection must name the conflicting field, got: %v", err)
	}
}

// TestNew_SSEEndpoint verifies the shared SSE composition seam: the endpoint
// exists (401-gated without a session) and the event-bus bridge forwards
// committed domain events to the bundle broadcaster.
func TestNew_SSEEndpoint(t *testing.T) {
	t.Parallel()

	bundle, err := setup.New(setup.Config{
		Title:   "SSE",
		SSEPath: "/sse",
	})
	if err != nil {
		t.Fatalf("setup.New with SSEPath: %v", err)
	}
	defer func() { _ = bundle.Close() }()

	if bundle.Broadcaster == nil {
		t.Fatal("Broadcaster must be non-nil when SSEPath is set")
	}

	mux := http.NewServeMux()
	bundle.Mount(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sse", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("SSE endpoint without session: got status %d, want 401", rec.Code)
	}

	// The bridge must forward committed events: subscribe to the raw hub,
	// register a user through the service, expect an "event" SSE frame.
	ch := bundle.Broadcaster.Subscribe()
	defer bundle.Broadcaster.Unsubscribe(ch)

	if _, err := bundle.Service.Register(context.Background(), usermgmt.RegisterRequest{
		ID:    usermgmt.GenerateUserID(),
		Email: "sse-bridge@example.com",
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	select {
	case evt, ok := <-ch:
		if !ok {
			t.Fatal("broadcaster channel closed before any event arrived")
		}

		if evt.Event != "event" {
			t.Errorf("bridged SSE event name: got %q, want %q", evt.Event, "event")
		}

		if evt.Data == "" || !strings.Contains(evt.Data, `"type"`) {
			t.Errorf("bridged SSE payload must be the JSON envelope, got: %q", evt.Data)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no SSE event bridged within 5s of a committed domain event")
	}
}

// TestNew_SSEEndpoint_DisabledByDefault verifies the opt-in contract: no
// broadcaster and no SSE route unless SSEPath is set.
func TestNew_SSEEndpoint_DisabledByDefault(t *testing.T) {
	t.Parallel()

	bundle, err := setup.New(setup.Config{})
	if err != nil {
		t.Fatalf("setup.New: %v", err)
	}
	defer func() { _ = bundle.Close() }()

	if bundle.Broadcaster != nil {
		t.Error("Broadcaster must be nil when SSEPath is unset")
	}
}

// TestNew_SSEPath_Validation verifies SSEPath participates in the path
// validation (shape, root reservation, distinctness).
func TestNew_SSEPath_Validation(t *testing.T) {
	t.Parallel()

	t.Run("must start with slash", func(t *testing.T) {
		t.Parallel()

		_, err := setup.New(setup.Config{SSEPath: "sse"})
		if err == nil || !strings.Contains(err.Error(), "SSEPath") {
			t.Errorf("expected SSEPath shape rejection, got: %v", err)
		}
	})

	t.Run("root is reserved", func(t *testing.T) {
		t.Parallel()

		_, err := setup.New(setup.Config{SSEPath: "/"})
		if err == nil || !strings.Contains(err.Error(), "root is reserved") {
			t.Errorf("expected root-reservation rejection, got: %v", err)
		}
	})

	t.Run("distinct from health", func(t *testing.T) {
		t.Parallel()

		_, err := setup.New(setup.Config{SSEPath: "/health"})
		if err == nil || !strings.Contains(err.Error(), "conflict") {
			t.Errorf("expected path-conflict rejection, got: %v", err)
		}
	})
}
