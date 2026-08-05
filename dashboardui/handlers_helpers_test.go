package dashboardui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

type fakeDeadLetterStore struct{}

func (fakeDeadLetterStore) Store(ctx context.Context, entry projectionhost.DeadLetterEntry) error {
	return nil
}

func (fakeDeadLetterStore) List(ctx context.Context, projectionName string) ([]projectionhost.DeadLetterEntry, error) {
	return nil, nil
}

func (fakeDeadLetterStore) Delete(ctx context.Context, projectionName, eventID string) error {
	return nil
}

func (fakeDeadLetterStore) Purge(ctx context.Context, projectionName string) error { return nil }

func TestWithProjectionHost_Missing(t *testing.T) {
	d := &Dashboard{config: Config{}}

	w := httptest.NewRecorder()
	called := false

	d.withProjectionHost(w, func(host *projectionhost.Host) {
		called = true
	})

	if called {
		t.Fatal("expected fn to not be called")
	}

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}

	if w.Body.String() != "projection host not configured\n" {
		t.Fatalf("unexpected body: %q", w.Body.String())
	}
}

func TestWithProjectionHost_Present(t *testing.T) {
	host := &projectionhost.Host{}
	d := &Dashboard{config: Config{ProjectionHost: host}}

	w := httptest.NewRecorder()

	var received *projectionhost.Host

	d.withProjectionHost(w, func(h *projectionhost.Host) {
		received = h
	})

	if received != host {
		t.Fatal("expected fn to receive the configured projection host")
	}

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

func TestWithDeadLetterStore_Missing(t *testing.T) {
	d := &Dashboard{config: Config{}}

	w := httptest.NewRecorder()
	called := false

	d.withDeadLetterStore(w, func(store projectionhost.DeadLetterStore) {
		called = true
	})

	if called {
		t.Fatal("expected fn to not be called")
	}

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}

	if w.Body.String() != "dead letter store not configured\n" {
		t.Fatalf("unexpected body: %q", w.Body.String())
	}
}

func TestWithDeadLetterStore_Present(t *testing.T) {
	store := fakeDeadLetterStore{}
	d := &Dashboard{config: Config{DeadLetterStore: store}}

	w := httptest.NewRecorder()

	var received projectionhost.DeadLetterStore

	d.withDeadLetterStore(w, func(s projectionhost.DeadLetterStore) {
		received = s
	})

	if received != store {
		t.Fatal("expected fn to receive the configured dead letter store")
	}

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}
