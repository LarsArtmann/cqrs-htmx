package dashboardui

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

// --- Fakes ---

type fakeSeekableJournal struct {
	events  []event.Event
	readErr error
	allErr  error
}

func (f *fakeSeekableJournal) ReadAll(_ context.Context) ([]event.Event, error) {
	if f.allErr != nil {
		return nil, f.allErr
	}
	return f.events, nil
}

func (f *fakeSeekableJournal) ReadFrom(_ context.Context, _ id.EventID, _ int) ([]event.Event, error) {
	if f.readErr != nil {
		return nil, f.readErr
	}
	return f.events, nil
}

type fakeEventByIDLoader struct {
	evt event.Event
	err error
}

func (f *fakeEventByIDLoader) LoadByEventID(_ context.Context, _ id.EventID) (event.Event, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.evt, nil
}

type populatedDeadLetterStore struct {
	entries []projectionhost.DeadLetterEntry
}

func (s *populatedDeadLetterStore) Store(_ context.Context, _ projectionhost.DeadLetterEntry) error {
	return nil
}

func (s *populatedDeadLetterStore) List(_ context.Context, _ string) ([]projectionhost.DeadLetterEntry, error) {
	return s.entries, nil
}

func (s *populatedDeadLetterStore) Delete(_ context.Context, _, _ string) error { return nil }
func (s *populatedDeadLetterStore) Purge(_ context.Context, _ string) error     { return nil }

type errorSnapshotStore struct {
	loadErr error
}

func (e *errorSnapshotStore) Save(_ context.Context, _ snapshot.Snapshot) error { return nil }
func (e *errorSnapshotStore) Delete(_ context.Context, _ id.StreamRef) error    { return nil }
func (e *errorSnapshotStore) Load(_ context.Context, _ id.StreamRef) (*snapshot.Snapshot, error) {
	return nil, e.loadErr
}
func (e *errorSnapshotStore) LoadAtVersion(_ context.Context, _ id.StreamRef, _ event.Version) (*snapshot.Snapshot, error) {
	return nil, e.loadErr
}

func makeTestEvent(t *testing.T, eventType string, version event.Version) event.Event {
	t.Helper()
	aggID := id.NewStreamID()
	evt, err := event.New(event.Type(eventType), aggID, "TestAggregate", version, map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("event.New: %v", err)
	}
	return evt
}

// --- T15: overviewStats ---

func TestOverviewStats_SeekableJournal(t *testing.T) {
	evt := makeTestEvent(t, "test.created", 1)
	d, err := New(Config{
		SeekableJournal: &fakeSeekableJournal{events: []event.Event{evt}}, //nolint:exhaustruct // test fake: only events needed
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	stats := d.overviewStats(context.Background())
	if stats.TotalEvents != "1" {
		t.Errorf("expected TotalEvents=1, got %s", stats.TotalEvents)
	}
	if len(stats.RecentEvents) != 1 {
		t.Errorf("expected 1 recent event, got %d", len(stats.RecentEvents))
	}
}

func TestOverviewStats_SeekableJournalError(t *testing.T) {
	d, err := New(Config{
		SeekableJournal: &fakeSeekableJournal{readErr: errors.New("boom")}, //nolint:exhaustruct // test fake: only readErr needed
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	stats := d.overviewStats(context.Background())
	if stats.TotalEvents != "0" {
		t.Errorf("expected TotalEvents=0 on error, got %s", stats.TotalEvents)
	}
}

func TestOverviewStats_JournalError(t *testing.T) {
	d, err := New(Config{
		Journal: &fakeSeekableJournal{allErr: errors.New("readall failed")}, //nolint:exhaustruct // test fake: only allErr needed
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	stats := d.overviewStats(context.Background())
	if stats.TotalEvents != "0" {
		t.Errorf("expected TotalEvents=0 on error, got %s", stats.TotalEvents)
	}
}

func TestOverviewStats_SeekableJournalManyEvents(t *testing.T) {
	events := make([]event.Event, overviewCountLimit+1)
	for i := range events {
		events[i] = makeTestEvent(t, "test.bulk", event.Version(i+1))
	}
	d, err := New(Config{
		SeekableJournal: &fakeSeekableJournal{events: events}, //nolint:exhaustruct // test fake: only events needed
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	stats := d.overviewStats(context.Background())
	if stats.TotalEvents != strconv.Itoa(overviewCountLimit+1)+"+" {
		t.Errorf("expected TotalEvents with + suffix, got %s", stats.TotalEvents)
	}
}

// --- T15: eventDetailHandler ---

func TestEventDetailHandler_InvalidID(t *testing.T) {
	d := mustTestDashboard(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events/not-a-valid-ulid", nil)
	r.SetPathValue("id", "not-a-valid-ulid")
	d.eventDetailHandler(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestEventDetailHandler_NotFound(t *testing.T) {
	d := mustTestDashboard(t)
	w := httptest.NewRecorder()
	eid := id.NewEventID()
	r := httptest.NewRequest(http.MethodGet, "/events/"+eid.String(), nil)
	r.SetPathValue("id", eid.String())
	d.eventDetailHandler(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestEventDetailHandler_EventByIDLoader(t *testing.T) {
	evt := makeTestEvent(t, "test.loaded", 1)
	d, err := New(Config{
		EventByIDLoader: &fakeEventByIDLoader{evt: evt}, //nolint:exhaustruct // test fake: only evt needed
		Journal:         &stubJournal{},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events/"+evt.ID().String(), nil)
	r.SetPathValue("id", evt.ID().String())
	d.eventDetailHandler(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}
}

func TestLoadEventByID_NoSource(t *testing.T) {
	d := &Dashboard{cfg: Config{}}
	_, err := d.loadEventByID(context.Background(), id.NewEventID())
	if err == nil {
		t.Fatal("expected error with no event source")
	}
}

// --- T16: DLQ handlers ---

func TestDLQDetailHandler_WithEntries(t *testing.T) {
	entries := []projectionhost.DeadLetterEntry{
		{
			ProjectionName: "test-proj",
			EventID:        "evt-1",
			EventType:      "test.event",
			Error:          "processing failed",
			StreamID:       id.NewStreamID().String(),
			FailedAt:       time.Now(),
		},
	}
	d, err := New(Config{
		Journal:         &stubJournal{},
		DeadLetterStore: &populatedDeadLetterStore{entries: entries},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/dead-letters/test-proj/evt-1", nil)
	r.SetPathValue("projection", "test-proj")
	r.SetPathValue("id", "evt-1")
	d.dlqDetailHandler(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestDLQIndexHandler_WithDeadLetterStore(t *testing.T) {
	d, err := New(Config{
		Journal:         &stubJournal{},
		DeadLetterStore: &populatedDeadLetterStore{entries: nil},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/dead-letters", nil)
	d.dlqIndexHandler(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRenderDLQ_WithEntries(t *testing.T) {
	entries := []projectionhost.DeadLetterEntry{
		{
			ProjectionName: "proj-a",
			EventID:        "evt-1",
			EventType:      "test.event",
			Error:          "fail",
			StreamID:       id.NewStreamID().String(),
			FailedAt:       time.Now(),
			ErrorFamily:    "Transient",
		},
	}
	d, err := New(Config{
		Journal:         &stubJournal{},
		DeadLetterStore: &populatedDeadLetterStore{entries: entries},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/dead-letters/proj-a", nil)
	r.SetPathValue("projection", "proj-a")
	d.dlqDetailHandler(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "proj-a") {
		t.Error("expected body to contain projection name")
	}
	if !strings.Contains(w.Body.String(), "evt-1") {
		t.Error("expected body to contain event ID")
	}
}

// --- T17: snapshotDetailHandler ---

func TestSnapshotDetailHandler_InvalidRef(t *testing.T) {
	d := mustTestDashboard(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/snapshots/", nil)
	r.SetPathValue("streamType", "")
	r.SetPathValue("streamID", "")
	d.snapshotDetailHandler(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSnapshotDetailHandler_StoreError(t *testing.T) {
	d, err := New(Config{
		Journal:       &stubJournal{},
		SnapshotStore: &errorSnapshotStore{loadErr: errors.New("db connection lost")},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/snapshots/User/01HXTEST", nil)
	r.SetPathValue("type", "User")
	r.SetPathValue("id", "01HXTEST")
	d.snapshotDetailHandler(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 (renders 'no snapshot' page), got %d", w.Code)
	}
}

// --- T17: loadRecentEvents ---

func TestLoadRecentEvents_SeekableJournal(t *testing.T) {
	evt := makeTestEvent(t, "test.recent", 1)
	d, err := New(Config{
		SeekableJournal: &fakeSeekableJournal{events: []event.Event{evt}}, //nolint:exhaustruct // test fake: only events needed
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	events, err := d.loadRecentEvents(context.Background(), id.EventID{}, 10)
	if err != nil {
		t.Fatalf("loadRecentEvents: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
}

func TestLoadRecentEvents_SeekableJournalError(t *testing.T) {
	d, err := New(Config{
		SeekableJournal: &fakeSeekableJournal{readErr: errors.New("seek failed")}, //nolint:exhaustruct // test fake: only readErr needed
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	events, err := d.loadRecentEvents(context.Background(), id.EventID{}, 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if events != nil {
		t.Errorf("expected nil events on error, got %v", events)
	}
}

func TestLoadRecentEvents_JournalError(t *testing.T) {
	d, err := New(Config{
		Journal: &fakeSeekableJournal{allErr: errors.New("readall failed")}, //nolint:exhaustruct // test fake: only allErr needed
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	events, err := d.loadRecentEvents(context.Background(), id.EventID{}, 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if events != nil {
		t.Errorf("expected nil events on error, got %v", events)
	}
}

func TestLoadRecentEvents_NoJournal(t *testing.T) {
	d := &Dashboard{cfg: Config{}}
	events, err := d.loadRecentEvents(context.Background(), id.EventID{}, 10)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if events != nil {
		t.Errorf("expected nil events, got %v", events)
	}
}
