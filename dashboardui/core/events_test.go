package core

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func makeTestEvent(eventType string, version int) event.Event {
	streamID := id.NewStreamID()
	evt, _ := event.New(event.Type(eventType), streamID, "TestAgg", event.Version(version), map[string]string{"k": "v"})
	return evt
}

func TestEventFilter_Active(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		filter EventFilter
		want   bool
	}{
		{"empty", EventFilter{}, false},
		{"type only", EventFilter{Type: "user.created"}, true},
		{"streamType only", EventFilter{StreamType: "User"}, true},
		{"streamID only", EventFilter{StreamID: "01HX..."}, true},
		{"all set", EventFilter{Type: "x", StreamType: "Y", StreamID: "Z"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.filter.Active(); got != tt.want {
				t.Errorf("Active() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEventFilter_Matches(t *testing.T) {
	t.Parallel()

	evt := makeTestEvent("user.created", 1)

	tests := []struct {
		name   string
		filter EventFilter
		want   bool
	}{
		{"empty filter matches all", EventFilter{}, true},
		{"matching type", EventFilter{Type: "user.created"}, true},
		{"non-matching type", EventFilter{Type: "user.deleted"}, false},
		{"matching stream type", EventFilter{StreamType: "TestAgg"}, true},
		{"non-matching stream type", EventFilter{StreamType: "Other"}, false},
		{"non-matching stream ID", EventFilter{StreamID: "nonexistent"}, false},
		{"matching stream ID", EventFilter{StreamID: evt.StreamID().String()}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.filter.Matches(evt); got != tt.want {
				t.Errorf("Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEventFilter_ExtraParams(t *testing.T) {
	t.Parallel()

	t.Run("inactive filter returns empty", func(t *testing.T) {
		t.Parallel()

		f := EventFilter{}
		if got := f.ExtraParams(); got != "" {
			t.Errorf("ExtraParams() = %q, want empty", got)
		}
	})

	t.Run("type only", func(t *testing.T) {
		t.Parallel()

		f := EventFilter{Type: "user.created"}
		if got := f.ExtraParams(); got != "type=user.created" {
			t.Errorf("ExtraParams() = %q, want %q", got, "type=user.created")
		}
	})

	t.Run("all fields", func(t *testing.T) {
		t.Parallel()

		f := EventFilter{Type: "x", StreamType: "Y", StreamID: "Z"}
		got := f.ExtraParams()
		// Should contain all three params
		for _, s := range []string{"type=x", "streamType=Y", "streamID=Z"} {
			if !contains(got, s) {
				t.Errorf("ExtraParams() = %q, should contain %q", got, s)
			}
		}
	})
}

func TestParseEventFilter(t *testing.T) {
	t.Parallel()

	r := httptest.NewRequest(http.MethodGet, "/?type=user.created&streamType=User&streamID=01HX", nil)
	f := ParseEventFilter(r)

	if f.Type != "user.created" {
		t.Errorf("Type = %q, want %q", f.Type, "user.created")
	}

	if f.StreamType != "User" {
		t.Errorf("StreamType = %q, want %q", f.StreamType, "User")
	}

	if f.StreamID != "01HX" {
		t.Errorf("StreamID = %q, want %q", f.StreamID, "01HX")
	}
}

func TestLoadRecentEvents_SeekableJournal(t *testing.T) {
	t.Parallel()

	evt1 := makeTestEvent("test.created", 1)
	evt2 := makeTestEvent("test.updated", 2)
	cfg := Config{
		SeekableJournal: &fakeSeekableJournal{events: []event.Event{evt1, evt2}},
	}

	events, err := LoadRecentEvents(context.Background(), cfg, id.EventID{}, 10)
	if err != nil {
		t.Fatalf("LoadRecentEvents returned error: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}

func TestLoadRecentEvents_JournalFallback(t *testing.T) {
	t.Parallel()

	evt1 := makeTestEvent("test.created", 1)
	cfg := Config{
		Journal: &fakeSeekableJournal{events: []event.Event{evt1}},
	}

	events, err := LoadRecentEvents(context.Background(), cfg, id.EventID{}, 10)
	if err != nil {
		t.Fatalf("LoadRecentEvents returned error: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}

func TestLoadRecentEvents_NoSource(t *testing.T) {
	t.Parallel()

	events, err := LoadRecentEvents(context.Background(), Config{}, id.EventID{}, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if events != nil {
		t.Fatalf("expected nil events, got %d", len(events))
	}
}

func TestLoadFilteredEvents(t *testing.T) {
	t.Parallel()

	evt1 := makeTestEvent("user.created", 1)
	evt2 := makeTestEvent("user.deleted", 2)
	evt3 := makeTestEvent("user.created", 3)

	cfg := Config{
		SeekableJournal: &fakeSeekableJournal{events: []event.Event{evt1, evt2, evt3}},
	}

	filter := EventFilter{Type: "user.created"}
	events, err := LoadFilteredEvents(context.Background(), cfg, id.EventID{}, filter, 10)
	if err != nil {
		t.Fatalf("LoadFilteredEvents returned error: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 matching events, got %d", len(events))
	}

	for _, e := range events {
		if string(e.Type()) != "user.created" {
			t.Errorf("expected user.created, got %s", e.Type())
		}
	}
}

func TestLoadEventByID_EventByIDLoader(t *testing.T) {
	t.Parallel()

	target := makeTestEvent("test.created", 1)
	cfg := Config{
		EventByIDLoader: &fakeEventByIDLoader{evt: target},
	}

	evt, err := LoadEventByID(context.Background(), cfg, target.ID())
	if err != nil {
		t.Fatalf("LoadEventByID returned error: %v", err)
	}

	if evt.ID() != target.ID() {
		t.Error("returned event ID mismatch")
	}
}

func TestLoadEventByID_SeekableJournalScan(t *testing.T) {
	t.Parallel()

	target := makeTestEvent("test.created", 1)
	other := makeTestEvent("test.updated", 2)
	cfg := Config{
		SeekableJournal: &fakeSeekableJournal{events: []event.Event{target, other}},
	}

	evt, err := LoadEventByID(context.Background(), cfg, target.ID())
	if err != nil {
		t.Fatalf("LoadEventByID returned error: %v", err)
	}

	if evt.ID() != target.ID() {
		t.Error("returned event ID mismatch")
	}
}

func TestLoadEventByID_NotFound(t *testing.T) {
	t.Parallel()

	evt := makeTestEvent("test.created", 1)
	cfg := Config{
		SeekableJournal: &fakeSeekableJournal{events: []event.Event{evt}},
	}

	_, err := LoadEventByID(context.Background(), cfg, id.NewEventID())
	if err == nil {
		t.Error("expected error for not-found event")
	}
}

func TestLoadEventByID_NoSource(t *testing.T) {
	t.Parallel()

	_, err := LoadEventByID(context.Background(), Config{}, id.NewEventID())
	if err == nil {
		t.Error("expected error when no event source configured")
	}
}

func TestFindEventNeighbors(t *testing.T) {
	t.Parallel()

	evt1 := makeTestEvent("a", 1)
	evt2 := makeTestEvent("b", 2)
	evt3 := makeTestEvent("c", 3)

	cfg := Config{
		SeekableJournal: &fakeSeekableJournal{events: []event.Event{evt1, evt2, evt3}},
	}

	t.Run("middle event has both neighbors", func(t *testing.T) {
		t.Parallel()

		prev, next := FindEventNeighbors(context.Background(), cfg, evt2.ID())
		if prev != evt1.ID().String() {
			t.Errorf("prev = %q, want %q", prev, evt1.ID().String())
		}

		if next != evt3.ID().String() {
			t.Errorf("next = %q, want %q", next, evt3.ID().String())
		}
	})

	t.Run("first event has no prev", func(t *testing.T) {
		t.Parallel()

		prev, next := FindEventNeighbors(context.Background(), cfg, evt1.ID())
		if prev != "" {
			t.Errorf("prev should be empty for first event, got %q", prev)
		}

		if next != evt2.ID().String() {
			t.Errorf("next = %q, want %q", next, evt2.ID().String())
		}
	})

	t.Run("last event has no next", func(t *testing.T) {
		t.Parallel()

		prev, next := FindEventNeighbors(context.Background(), cfg, evt3.ID())
		if prev != evt2.ID().String() {
			t.Errorf("prev = %q, want %q", prev, evt2.ID().String())
		}

		if next != "" {
			t.Errorf("next should be empty for last event, got %q", next)
		}
	})

	t.Run("not found returns empty", func(t *testing.T) {
		t.Parallel()

		prev, next := FindEventNeighbors(context.Background(), cfg, id.NewEventID())
		if prev != "" || next != "" {
			t.Errorf("expected empty neighbors for unknown event, got prev=%q next=%q", prev, next)
		}
	})
}
