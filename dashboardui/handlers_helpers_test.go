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

func TestRequireProjectionHost_Missing(t *testing.T) {
	d := &Dashboard{cfg: Config{}}
	w := httptest.NewRecorder()
	if d.requireProjectionHost(w) {
		t.Fatal("expected requireProjectionHost to return false")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	if w.Body.String() != "projection host not configured\n" {
		t.Fatalf("unexpected body: %q", w.Body.String())
	}
}

func TestRequireProjectionHost_Present(t *testing.T) {
	d := &Dashboard{cfg: Config{ProjectionHost: &projectionhost.Host{}}}
	w := httptest.NewRecorder()
	if !d.requireProjectionHost(w) {
		t.Fatal("expected requireProjectionHost to return true")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

func TestRequireDeadLetterStore_Missing(t *testing.T) {
	d := &Dashboard{cfg: Config{}}
	w := httptest.NewRecorder()
	if d.requireDeadLetterStore(w) {
		t.Fatal("expected requireDeadLetterStore to return false")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	if w.Body.String() != "dead letter store not configured\n" {
		t.Fatalf("unexpected body: %q", w.Body.String())
	}
}

func TestRequireDeadLetterStore_Present(t *testing.T) {
	d := &Dashboard{cfg: Config{DeadLetterStore: fakeDeadLetterStore{}}}
	w := httptest.NewRecorder()
	if !d.requireDeadLetterStore(w) {
		t.Fatal("expected requireDeadLetterStore to return true")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}
