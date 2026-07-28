package dashboardui

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

// --- Stubs ---

// fakeSnapshotStore implements snapshot.SnapshotStore for testing.
type fakeSnapshotStore struct {
	deleteErr error
	deleted   bool
}

func (f *fakeSnapshotStore) Save(_ context.Context, _ snapshot.Snapshot) error { return nil }

func (f *fakeSnapshotStore) Delete(_ context.Context, _ id.StreamRef) error {
	f.deleted = true

	return f.deleteErr
}

func (f *fakeSnapshotStore) Load(_ context.Context, _ id.StreamRef) (*snapshot.Snapshot, error) {
	return nil, nil
}

func (f *fakeSnapshotStore) LoadAtVersion(
	_ context.Context, _ id.StreamRef, _ event.Version,
) (*snapshot.Snapshot, error) {
	return nil, nil
}

// errorDeadLetterStore returns errors on Delete and Purge.
type errorDeadLetterStore struct{}

func (errorDeadLetterStore) Store(_ context.Context, _ projectionhost.DeadLetterEntry) error {
	return nil
}

func (errorDeadLetterStore) List(_ context.Context, _ string) ([]projectionhost.DeadLetterEntry, error) {
	return nil, nil
}

func (errorDeadLetterStore) Delete(_ context.Context, _, _ string) error {
	return errors.New("delete failed")
}

func (errorDeadLetterStore) Purge(_ context.Context, _ string) error {
	return errors.New("purge failed")
}

// fakeEventSource implements event.EventSource for testing.
type fakeEventSource struct {
	allEvents    []event.Event
	toVersion    []event.Event
	loadErr      error
	loadToVerErr error
}

func (f *fakeEventSource) Load(_ context.Context, _ id.StreamRef) ([]event.Event, error) {
	return f.allEvents, f.loadErr
}

func (f *fakeEventSource) LoadFromVersion(
	_ context.Context, _ id.StreamRef, _ event.Version,
) ([]event.Event, error) {
	return f.allEvents, nil
}

func (f *fakeEventSource) LoadToVersion(
	_ context.Context, _ id.StreamRef, _ event.Version,
) ([]event.Event, error) {
	return f.toVersion, f.loadToVerErr
}

func (f *fakeEventSource) LoadToTimestamp(
	_ context.Context, _ id.StreamRef, _ time.Time,
) ([]event.Event, error) {
	return f.allEvents, nil
}

// --- DLQ Delete Handler ---

func TestDLQDeleteHandler_NoStore(t *testing.T) {
	d := mustTestDashboard(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/dead-letters/proj1/evt1/delete", nil)
	r.SetPathValue("projection", "proj1")
	r.SetPathValue("eventID", "evt1")
	d.dlqDeleteHandler(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestDLQDeleteHandler_Success(t *testing.T) {
	store := &fakeDeadLetterStore{}
	d := mustTestDashboardWithConfig(t, Config{
		Journal:         &stubJournal{},
		DeadLetterStore: store,
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/dead-letters/proj1/evt1/delete", nil)
	r.SetPathValue("projection", "proj1")
	r.SetPathValue("eventID", "evt1")
	d.dlqDeleteHandler(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect, got %d", w.Code)
	}

	loc := w.Header().Get("Location")
	if !strings.Contains(loc, "/dead-letters/proj1") {
		t.Fatalf("unexpected redirect location: %s", loc)
	}

	trigger := w.Header().Get("Hx-Trigger")
	if !strings.Contains(trigger, "Dead letter deleted") {
		t.Fatalf("expected success toast, got Hx-Trigger: %s", trigger)
	}
}

func TestDLQDeleteHandler_StoreError(t *testing.T) {
	d := mustTestDashboardWithConfig(t, Config{
		Journal:         &stubJournal{},
		DeadLetterStore: errorDeadLetterStore{},
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/dead-letters/proj1/evt1/delete", nil)
	r.SetPathValue("projection", "proj1")
	r.SetPathValue("eventID", "evt1")
	d.dlqDeleteHandler(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}

	trigger := w.Header().Get("Hx-Trigger")
	if !strings.Contains(trigger, "Delete failed") {
		t.Fatalf("expected error toast, got Hx-Trigger: %s", trigger)
	}
}

// --- DLQ Purge Handler ---

func TestDLQPurgeHandler_NoStore(t *testing.T) {
	d := mustTestDashboard(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/dead-letters/proj1/purge", nil)
	r.SetPathValue("projection", "proj1")
	d.dlqPurgeHandler(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestDLQPurgeHandler_Success(t *testing.T) {
	d := mustTestDashboardWithConfig(t, Config{
		Journal:         &stubJournal{},
		DeadLetterStore: fakeDeadLetterStore{},
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/dead-letters/proj1/purge", nil)
	r.SetPathValue("projection", "proj1")
	d.dlqPurgeHandler(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect, got %d", w.Code)
	}

	loc := w.Header().Get("Location")
	if !strings.Contains(loc, "/dead-letters/proj1") {
		t.Fatalf("unexpected redirect location: %s", loc)
	}

	trigger := w.Header().Get("Hx-Trigger")
	if !strings.Contains(trigger, "Dead letters purged") {
		t.Fatalf("expected success toast, got Hx-Trigger: %s", trigger)
	}
}

func TestDLQPurgeHandler_StoreError(t *testing.T) {
	d := mustTestDashboardWithConfig(t, Config{
		Journal:         &stubJournal{},
		DeadLetterStore: errorDeadLetterStore{},
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/dead-letters/proj1/purge", nil)
	r.SetPathValue("projection", "proj1")
	d.dlqPurgeHandler(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}

	trigger := w.Header().Get("Hx-Trigger")
	if !strings.Contains(trigger, "Purge failed") {
		t.Fatalf("expected error toast, got Hx-Trigger: %s", trigger)
	}
}

// --- DLQ Replay Handler ---

func TestDLQReplayHandler_NoHost(t *testing.T) {
	d := mustTestDashboard(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/dead-letters/proj1/replay", nil)
	r.SetPathValue("projection", "proj1")
	d.dlqReplayHandler(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestDLQReplayHandler_ZeroHostError(t *testing.T) {
	d := mustTestDashboardWithConfig(t, Config{
		Journal:        &stubJournal{},
		ProjectionHost: &projectionhost.Host{},
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/dead-letters/proj1/replay", nil)
	r.SetPathValue("projection", "proj1")
	d.dlqReplayHandler(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 (no DLQ store on zero-value host), got %d", w.Code)
	}

	trigger := w.Header().Get("Hx-Trigger")
	if !strings.Contains(trigger, "Replay failed") {
		t.Fatalf("expected error toast, got Hx-Trigger: %s", trigger)
	}
}

// --- Projection Reset Handler ---

func TestProjectionResetHandler_NoHost(t *testing.T) {
	d := mustTestDashboard(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/projections/proj1/reset", nil)
	r.SetPathValue("name", "proj1")
	d.projectionResetHandler(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestProjectionResetHandler_ZeroHostError(t *testing.T) {
	d := mustTestDashboardWithConfig(t, Config{
		Journal:        &stubJournal{},
		ProjectionHost: &projectionhost.Host{},
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/projections/unknown/reset", nil)
	r.SetPathValue("name", "unknown")
	d.projectionResetHandler(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 (unknown projection on zero-value host), got %d", w.Code)
	}

	trigger := w.Header().Get("Hx-Trigger")
	if !strings.Contains(trigger, "Reset failed") {
		t.Fatalf("expected error toast, got Hx-Trigger: %s", trigger)
	}
}

// --- Snapshot Delete Handler ---

func TestSnapshotDeleteHandler_NoStore(t *testing.T) {
	d := mustTestDashboard(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/snapshots/user/abc/delete", nil)
	r.SetPathValue("type", "user")
	r.SetPathValue("id", "abc")
	d.snapshotDeleteHandler(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSnapshotDeleteHandler_InvalidRef(t *testing.T) {
	store := &fakeSnapshotStore{} //nolint:exhaustruct // test stub: zero values are correct defaults
	d := mustTestDashboardWithConfig(t, Config{
		Journal:       &stubJournal{},
		SnapshotStore: store,
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/snapshots///delete", nil)
	r.SetPathValue("type", "")
	r.SetPathValue("id", "abc")
	d.snapshotDeleteHandler(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty stream type, got %d", w.Code)
	}

	if store.deleted {
		t.Fatal("Delete should not have been called on invalid ref")
	}
}

func TestSnapshotDeleteHandler_Success(t *testing.T) {
	store := &fakeSnapshotStore{} //nolint:exhaustruct // test stub: zero values are correct defaults
	d := mustTestDashboardWithConfig(t, Config{
		Journal:       &stubJournal{},
		SnapshotStore: store,
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/snapshots/user/abc/delete", nil)
	r.SetPathValue("type", "user")
	r.SetPathValue("id", "abc")
	d.snapshotDeleteHandler(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect, got %d", w.Code)
	}

	if !store.deleted {
		t.Fatal("expected SnapshotStore.Delete to be called")
	}

	loc := w.Header().Get("Location")
	if !strings.Contains(loc, "/snapshots") {
		t.Fatalf("unexpected redirect location: %s", loc)
	}

	trigger := w.Header().Get("Hx-Trigger")
	if !strings.Contains(trigger, "Snapshot deleted") {
		t.Fatalf("expected success toast, got Hx-Trigger: %s", trigger)
	}
}

// --- Time-Travel Detail Handler ---

func TestTimeTravelDetailHandler_WithEvents(t *testing.T) {
	streamID := id.NewStreamID()
	streamType := id.StreamType("user")

	evts := make([]event.Event, 3)
	for i := range evts {
		ver := event.Version(i + 1)

		e, err := event.New(
			event.Type("UserRegistered"),
			streamID,
			streamType,
			ver,
			[]byte(`{"event":"test"}`),
		)
		if err != nil {
			t.Fatalf("event.New v%d: %v", ver, err)
		}

		evts[i] = e
	}

	src := &fakeEventSource{ //nolint:exhaustruct // test stub: only fields needed for this test
		allEvents: evts,
		toVersion: evts[:2], // viewing version 2
	}

	d := mustTestDashboardWithConfig(t, Config{
		EventSource: src,
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/time-travel/user/"+streamID.String()+"?v=2", nil)
	r.SetPathValue("type", "user")
	r.SetPathValue("id", streamID.String())
	d.timeTravelDetailHandler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()

	if !strings.Contains(body, "Time Travel:") {
		t.Fatal("expected time travel heading")
	}

	if !strings.Contains(body, "Viewing version 2 of 3") {
		t.Fatalf("expected version indicator, got body containing: %s", body[:min(200, len(body))])
	}

	for _, v := range []string{">1<", ">2<", ">3<"} {
		if !strings.Contains(body, v) {
			t.Fatalf("expected version link %q in body", v)
		}
	}

	for _, e := range evts[:2] {
		if !strings.Contains(body, e.ID().String()) {
			t.Fatal("expected event ID in timeline")
		}
	}
}

func TestTimeTravelDetailHandler_NoEvents(t *testing.T) {
	src := &fakeEventSource{ //nolint:exhaustruct // test stub: only fields needed for this test
		allEvents: nil,
	}

	d := mustTestDashboardWithConfig(t, Config{
		EventSource: src,
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/time-travel/user/abc", nil)
	r.SetPathValue("type", "user")
	r.SetPathValue("id", "abc")
	d.timeTravelDetailHandler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "No events") {
		t.Fatal("expected 'No events' message")
	}
}

func TestTimeTravelDetailHandler_LoadToVersionError(t *testing.T) {
	streamID := id.NewStreamID()

	e, err := event.New(
		event.Type("TestEvent"),
		streamID,
		id.StreamType("user"),
		event.Version(1),
		[]byte(`{}`),
	)
	if err != nil {
		t.Fatalf("event.New: %v", err)
	}

	src := &fakeEventSource{ //nolint:exhaustruct // test stub: only fields needed for this test
		allEvents:    []event.Event{e},
		toVersion:    nil,
		loadToVerErr: errors.New("version load failed"),
	}

	d := mustTestDashboardWithConfig(t, Config{
		EventSource: src,
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/time-travel/user/"+streamID.String(), nil)
	r.SetPathValue("type", "user")
	r.SetPathValue("id", streamID.String())
	d.timeTravelDetailHandler(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// --- Helpers ---

func mustTestDashboardWithConfig(t *testing.T, cfg Config) *Dashboard {
	t.Helper()

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("New Dashboard: %v", err)
	}

	return d
}

var (
	_ snapshot.SnapshotStore         = (*fakeSnapshotStore)(nil)
	_ projectionhost.DeadLetterStore = errorDeadLetterStore{}
	_ event.EventSource              = (*fakeEventSource)(nil)
)
