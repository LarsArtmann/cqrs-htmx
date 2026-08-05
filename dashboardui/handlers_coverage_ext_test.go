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

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
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

func (e *errorSnapshotStore) LoadAtVersion(
	_ context.Context, _ id.StreamRef, _ event.Version,
) (*snapshot.Snapshot, error) {
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
		SeekableJournal: &fakeSeekableJournal{events: []event.Event{evt}},
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
		SeekableJournal: &fakeSeekableJournal{readErr: errors.New("boom")},
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
		Journal: &fakeSeekableJournal{allErr: errors.New("readall failed")},
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
		SeekableJournal: &fakeSeekableJournal{events: events},
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
		EventByIDLoader: &fakeEventByIDLoader{evt: evt},
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
	d := &Dashboard{config: Config{}}

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
		SeekableJournal: &fakeSeekableJournal{events: []event.Event{evt}},
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
		SeekableJournal: &fakeSeekableJournal{readErr: errors.New("seek failed")},
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
		Journal: &fakeSeekableJournal{allErr: errors.New("readall failed")},
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
	d := &Dashboard{config: Config{}}

	events, err := d.loadRecentEvents(context.Background(), id.EventID{}, 10)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if events != nil {
		t.Errorf("expected nil events, got %v", events)
	}
}

// --- HTMX Partial Rendering ---

func TestIsHTMXRequest_NormalRequest(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/events", nil)
	if isHTMXRequest(r) {
		t.Fatal("expected false for normal request")
	}
}

func TestIsHTMXRequest_BoostedRequest(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/events", nil)
	r.Header.Set("HX-Request", "true")

	if !isHTMXRequest(r) {
		t.Fatal("expected true for HTMX request")
	}
}

func TestRenderLayout_HTMXPartial(t *testing.T) {
	d := mustTestDashboard(t)
	r := httptest.NewRequest(http.MethodGet, "/events", nil)
	r.Header.Set("HX-Request", "true")

	p := d.page("Events", "/events", r)
	html := d.renderLayout(p, func() string { return "<p>test content</p>" })

	if !strings.Contains(html, "<main id=\"main-content\"") {
		t.Errorf("expected <main> in HTMX partial, got: %s", html)
	}

	if !strings.Contains(html, "<title>") {
		t.Errorf("expected <title> in HTMX partial for tab update")
	}

	if strings.Contains(html, "<!DOCTYPE html>") {
		t.Errorf("expected no DOCTYPE in HTMX partial")
	}

	if strings.Contains(html, "<aside") {
		t.Errorf("expected no sidebar in HTMX partial")
	}
}

func TestRenderLayout_FullPage(t *testing.T) {
	d := mustTestDashboard(t)
	r := httptest.NewRequest(http.MethodGet, "/events", nil)

	p := d.page("Events", "/events", r)
	html := d.renderLayout(p, func() string { return "<p>test content</p>" })

	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Errorf("expected DOCTYPE in full page")
	}

	if !strings.Contains(html, "<aside") {
		t.Errorf("expected sidebar in full page")
	}

	if !strings.Contains(html, "data-hx-boost") {
		t.Errorf("expected hx-boost attribute on layout div")
	}

	if !strings.Contains(html, "data-hx-select=\"#main-content\"") {
		t.Errorf("expected hx-select attribute for partial extraction")
	}
}

// --- Command Detail Handler ---

type fakeCommandJournal struct {
	cmds []*command.PersistedCommand
	err  error
}

func (f *fakeCommandJournal) ReadAll(_ context.Context) ([]*command.PersistedCommand, error) {
	if f.err != nil {
		return nil, f.err
	}

	return f.cmds, nil
}

func (f *fakeCommandJournal) ReadFrom(_ context.Context, _ id.CommandID, _ int) ([]*command.PersistedCommand, error) {
	if f.err != nil {
		return nil, f.err
	}

	return f.cmds, nil
}

func makeTestCommand(t *testing.T) *command.PersistedCommand {
	t.Helper()

	cmdID := id.NewCommandID()
	ref := id.NewStreamRef("TestAggregate", id.NewStreamID())

	cmd, err := command.NewPersistedCommand(
		command.Type("test.create"),
		ref,
		[]byte(`{"name":"test","value":42}`),
		command.WithPersistedCommandID(cmdID),
	)
	if err != nil {
		t.Fatalf("NewPersistedCommand: %v", err)
	}

	return cmd
}

func TestCommandDetailHandler_Renders(t *testing.T) {
	cmd := makeTestCommand(t)

	d, err := New(Config{
		Journal:        &stubJournal{},
		CommandJournal: &fakeCommandJournal{cmds: []*command.PersistedCommand{cmd}},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/commands/"+cmd.ID().String(), nil)
	r.SetPathValue("id", cmd.ID().String())
	d.commandDetailHandler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}

	if !strings.Contains(w.Body.String(), "test.create") {
		t.Error("expected body to contain command type")
	}
}

func TestCommandDetailHandler_NotFound(t *testing.T) {
	d, err := New(Config{
		Journal:        &stubJournal{},
		CommandJournal: &fakeCommandJournal{cmds: nil},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	missingID := id.NewCommandID()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/commands/"+missingID.String(), nil)
	r.SetPathValue("id", missingID.String())
	d.commandDetailHandler(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestCommandDetailHandler_InvalidID(t *testing.T) {
	d, err := New(Config{
		Journal:        &stubJournal{},
		CommandJournal: &fakeCommandJournal{},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/commands/not-a-valid-ulid", nil)
	r.SetPathValue("id", "not-a-valid-ulid")
	d.commandDetailHandler(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- Query Detail Handler ---

type fakeQueryJournal struct {
	queries []*query.PersistedQuery
	err     error
}

func (f *fakeQueryJournal) ReadAllQueries(_ context.Context) ([]*query.PersistedQuery, error) {
	if f.err != nil {
		return nil, f.err
	}

	return f.queries, nil
}

func (f *fakeQueryJournal) ReadQueriesFrom(_ context.Context, _ id.RequestID, _ int) ([]*query.PersistedQuery, error) {
	if f.err != nil {
		return nil, f.err
	}

	return f.queries, nil
}

func makeTestQuery(t *testing.T) *query.PersistedQuery {
	t.Helper()

	queryID := id.NewRequestID()

	q, err := query.NewPersistedQuery(
		query.Type("test.search"),
		[]byte(`{"filter":"active"}`),
		query.WithQueryID(queryID),
	)
	if err != nil {
		t.Fatalf("NewPersistedQuery: %v", err)
	}

	return q
}

func TestQueryDetailHandler_Renders(t *testing.T) {
	q := makeTestQuery(t)

	d, err := New(Config{
		Journal:      &stubJournal{},
		QueryJournal: &fakeQueryJournal{queries: []*query.PersistedQuery{q}},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/queries/"+q.ID().String(), nil)
	r.SetPathValue("id", q.ID().String())
	d.queryDetailHandler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}

	if !strings.Contains(w.Body.String(), "test.search") {
		t.Error("expected body to contain query type")
	}
}

func TestQueryDetailHandler_NotFound(t *testing.T) {
	d, err := New(Config{
		Journal:      &stubJournal{},
		QueryJournal: &fakeQueryJournal{queries: nil},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	missingID := id.NewRequestID()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/queries/"+missingID.String(), nil)
	r.SetPathValue("id", missingID.String())
	d.queryDetailHandler(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestQueryDetailHandler_InvalidID(t *testing.T) {
	d, err := New(Config{
		Journal:      &stubJournal{},
		QueryJournal: &fakeQueryJournal{},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/queries/not-valid", nil)
	r.SetPathValue("id", "not-valid")
	d.queryDetailHandler(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- DLQ Entry Detail Handler ---

func TestDLQEntryDetailHandler_Renders(t *testing.T) {
	entry := projectionhost.DeadLetterEntry{
		ProjectionName: "test-proj",
		EventID:        "evt-001",
		EventType:      "test.failed",
		Error:          "processing failed at step 3",
		StreamID:       id.NewStreamID().String(),
		FailedAt:       time.Now(),
		ErrorFamily:    "Transient",
		ErrorCode:      "TIMEOUT",
	}

	d, err := New(Config{
		Journal:         &stubJournal{},
		DeadLetterStore: &populatedDeadLetterStore{entries: []projectionhost.DeadLetterEntry{entry}},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/dead-letters/test-proj/evt-001", nil)
	r.SetPathValue("projection", "test-proj")
	r.SetPathValue("eventID", "evt-001")
	d.dlqEntryDetailHandler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !strings.Contains(body, "evt-001") {
		t.Error("expected body to contain event ID")
	}

	if !strings.Contains(body, "test.failed") {
		t.Error("expected body to contain event type")
	}

	if !strings.Contains(body, "processing failed") {
		t.Error("expected body to contain error message")
	}
}

func TestDLQEntryDetailHandler_NotFound(t *testing.T) {
	d, err := New(Config{
		Journal:         &stubJournal{},
		DeadLetterStore: &populatedDeadLetterStore{entries: nil},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/dead-letters/test-proj/nonexistent", nil)
	r.SetPathValue("projection", "test-proj")
	r.SetPathValue("eventID", "nonexistent")
	d.dlqEntryDetailHandler(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// --- Aggregate Detail Pagination ---

func TestAggregateDetail_Pagination(t *testing.T) { //nolint:cyclop // multi-page test
	// Generate 60 events for a single aggregate — exceeds default page size of 50.
	events := make([]event.Event, 60)
	streamID := id.NewStreamID()

	for i := range events {
		evt, err := event.New(
			event.Type("test.event"),
			streamID,
			"TestAggregate",
			event.Version(i+1),
			map[string]string{"seq": strconv.Itoa(i)},
		)
		if err != nil {
			t.Fatalf("event.New(%d): %v", i, err)
		}

		events[i] = evt
	}

	d, err := New(Config{
		EventSource: &fakeEventSource{allEvents: events},
		Journal:     &stubJournal{},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Page 1: default page size, should show 50 events and a Next link.
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/aggregates/TestAggregate/"+streamID.String(), nil)
	r.SetPathValue("type", "TestAggregate")
	r.SetPathValue("id", streamID.String())
	d.aggregateDetailHandler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("page 1: expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !strings.Contains(body, "60 events") {
		t.Error("expected total event count in subtitle")
	}

	if !strings.Contains(body, "Next") && !strings.Contains(body, "next") {
		t.Error("expected Next link when there are more events than page size")
	}

	// Count event rows (each row has a version cell).
	row1Count := strings.Count(body, `<td class="cell-emph">`)
	if row1Count > 51 {
		t.Errorf("page 1: expected <=50 event rows, got %d", row1Count)
	}

	// Page 2: after version 50, should show remaining 10 events and a Prev link.
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "/aggregates/TestAggregate/"+streamID.String()+"?after=50", nil)
	r2.SetPathValue("type", "TestAggregate")
	r2.SetPathValue("id", streamID.String())
	d.aggregateDetailHandler(w2, r2)

	if w2.Code != http.StatusOK {
		t.Fatalf("page 2: expected 200, got %d (body: %s)", w2.Code, w2.Body.String())
	}

	body2 := w2.Body.String()
	if !strings.Contains(body2, "Prev") && !strings.Contains(body2, "prev") {
		t.Error("expected Prev link on page 2")
	}

	row2Count := strings.Count(body2, `<td class="cell-emph">`)
	if row2Count != 10 {
		t.Errorf("page 2: expected 10 event rows, got %d", row2Count)
	}
}

func TestOverviewHandler_HTMXReturnsPartial(t *testing.T) {
	d := mustTestDashboard(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("HX-Request", "true")
	d.overviewHandler(w, r)

	body := w.Body.String()
	if strings.Contains(body, "<!DOCTYPE html>") {
		t.Errorf("expected no DOCTYPE in HTMX response")
	}

	if !strings.Contains(body, "<main id=\"main-content\"") {
		t.Errorf("expected <main> in HTMX response")
	}
}
