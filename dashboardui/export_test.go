package dashboardui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

func TestEventsExport_CSV(t *testing.T) {
	t.Parallel()

	streamID := id.NewStreamID()
	evt1, _ := event.New(
		event.Type("test.created"),
		streamID,
		"TestAgg",
		1,
		map[string]string{"k": "v"},
	)
	evt2, _ := event.New(
		event.Type("test.updated"),
		streamID,
		"TestAgg",
		2,
		map[string]string{"k": "v2"},
	)

	d := MustNew(Config{
		Journal: &fakeSeekableJournal{events: []event.Event{evt1, evt2}},
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events?format=csv", nil)
	d.eventsIndexHandler(w, r)

	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "text/csv") {
		t.Fatalf("expected text/csv content-type, got %q", ct)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Event ID") {
		t.Error("expected CSV header row with 'Event ID'")
	}

	if !strings.Contains(body, "test.created") {
		t.Error("expected CSV row with event type 'test.created'")
	}
}

func TestEventsExport_JSON(t *testing.T) {
	t.Parallel()

	streamID := id.NewStreamID()
	evt, _ := event.New(
		event.Type("test.created"),
		streamID,
		"TestAgg",
		1,
		map[string]string{"k": "v"},
	)

	d := MustNew(Config{
		Journal: &fakeSeekableJournal{events: []event.Event{evt}},
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events?format=json", nil)
	d.eventsIndexHandler(w, r)

	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json content-type, got %q", ct)
	}

	if !strings.Contains(w.Body.String(), "test.created") {
		t.Error("expected JSON body to contain event type")
	}
}

func TestCommandsExport_CSV(t *testing.T) {
	t.Parallel()

	cmd := makeTestCommand(t)
	d := MustNew(Config{
		Journal:        stubJournal{},
		CommandJournal: &fakeCommandJournal{cmds: []*command.PersistedCommand{cmd}},
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/commands?format=csv", nil)
	d.commandsIndexHandler(w, r)

	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "text/csv") {
		t.Fatalf("expected text/csv content-type, got %q", ct)
	}

	if !strings.Contains(w.Body.String(), "Command ID") {
		t.Error("expected CSV header row")
	}
}

// The export must carry the COMPLETE audit trail: the old zero-cursor
// ReadFrom(zero, exportLimit) returned only the OLDEST exportLimit commands,
// silently dropping every newer entry, and never paged past one batch.
func TestCommandsExport_PagesSeekableJournal_CompleteTrail(t *testing.T) {
	t.Parallel()

	total := exportLimit + 5 // one entry past the old hard cap: forces paging AND proves no truncation
	cmds := make([]*command.PersistedCommand, 0, total)
	for range total {
		cmds = append(cmds, makeTestCommand(t))
	}

	d := MustNew(Config{
		Journal:        stubJournal{},
		CommandJournal: &fakeCommandJournal{cmds: cmds},
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/commands?format=csv", nil)
	d.commandsIndexHandler(w, r)

	if got := strings.Count(w.Body.String(), "test.create"); got != total {
		t.Errorf("export must contain every command: got %d rows, want %d", got, total)
	}
}

// Queries twin: JSON export pages a seekable journal and stays complete.
func TestQueriesExport_PagesSeekableJournal_CompleteTrail(t *testing.T) {
	t.Parallel()

	total := exportBatch + 3 // two cursor pages
	queries := make([]*query.PersistedQuery, 0, total)
	for range total {
		queries = append(queries, makeTestQuery(t))
	}

	d := MustNew(Config{
		Journal:      stubJournal{},
		QueryJournal: &fakeQueryJournal{queries: queries},
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/queries?format=json", nil)
	d.queriesIndexHandler(w, r)

	if got := strings.Count(w.Body.String(), `"requestId"`); got != total {
		t.Errorf("export must contain every query: got %d rows, want %d", got, total)
	}
}
