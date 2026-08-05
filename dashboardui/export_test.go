package dashboardui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestEventsExport_CSV(t *testing.T) {
	t.Parallel()

	streamID := id.NewStreamID()
	evt1, _ := event.New(event.Type("test.created"), streamID, "TestAgg", 1, map[string]string{"k": "v"})
	evt2, _ := event.New(event.Type("test.updated"), streamID, "TestAgg", 2, map[string]string{"k": "v2"})

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
	evt, _ := event.New(event.Type("test.created"), streamID, "TestAgg", 1, map[string]string{"k": "v"})

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
