package core

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

func TestProjectionStatusKind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"running", StatusGood},
		{"RUNNING", StatusGood},
		{"live", StatusGood},
		{"idle", StatusWarn},
		{"backoff", StatusWarn},
		{"draining", StatusWarn},
		{"stopped", StatusBad},
		{"failed", StatusBad},
		{"unknown", StatusNeutral},
		{"", StatusNeutral},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got := ProjectionStatusKind(tt.input)
			if got != tt.want {
				t.Errorf("ProjectionStatusKind(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestProjectionStats_NilHost(t *testing.T) {
	t.Parallel()

	stats := ProjectionStats(nil)
	if stats != nil {
		t.Errorf("ProjectionStats(nil) should return nil, got %d entries", len(stats))
	}
}

func TestFetchOverview_EmptyConfig(t *testing.T) {
	t.Parallel()

	overview := FetchOverview(context.Background(), Config{})

	if overview.TotalEvents != "0" {
		t.Errorf("TotalEvents = %q, want %q", overview.TotalEvents, "0")
	}

	if overview.TotalAggregates != "0" {
		t.Errorf("TotalAggregates = %q, want %q", overview.TotalAggregates, "0")
	}

	if len(overview.RecentEvents) != 0 {
		t.Errorf("RecentEvents should be empty, got %d", len(overview.RecentEvents))
	}

	if len(overview.Projections) != 0 {
		t.Errorf("Projections should be empty, got %d", len(overview.Projections))
	}
}

func TestFetchOverview_WithSeekableJournal(t *testing.T) {
	t.Parallel()

	evt1 := makeTestEvent("user.created", 1)
	evt2 := makeTestEvent("user.updated", 2)
	cfg := Config{
		SeekableJournal: &fakeSeekableJournal{events: []event.Event{evt1, evt2}},
	}

	overview := FetchOverview(context.Background(), cfg)

	if overview.TotalEvents != "2" {
		t.Errorf("TotalEvents = %q, want %q", overview.TotalEvents, "2")
	}

	if len(overview.RecentEvents) != 2 {
		t.Fatalf("RecentEvents should have 2 entries, got %d", len(overview.RecentEvents))
	}

	if overview.RecentEvents[0].Type != "user.created" {
		t.Errorf("first event type = %q, want %q", overview.RecentEvents[0].Type, "user.created")
	}
}

func TestFetchOverview_WithJournalFallback(t *testing.T) {
	t.Parallel()

	evt1 := makeTestEvent("a", 1)
	cfg := Config{
		Journal: &fakeSeekableJournal{events: []event.Event{evt1}},
	}

	overview := FetchOverview(context.Background(), cfg)

	if overview.TotalEvents != "1" {
		t.Errorf("TotalEvents = %q, want %q", overview.TotalEvents, "1")
	}

	if len(overview.RecentEvents) != 1 {
		t.Fatalf("RecentEvents should have 1 entry, got %d", len(overview.RecentEvents))
	}
}

func TestFetchOverview_RecentEventsLimit(t *testing.T) {
	t.Parallel()

	events := make([]event.Event, RecentEventsLimit+5)
	for i := range events {
		events[i] = makeTestEvent("test", i+1)
	}

	cfg := Config{
		SeekableJournal: &fakeSeekableJournal{events: events},
	}

	overview := FetchOverview(context.Background(), cfg)

	if len(overview.RecentEvents) != RecentEventsLimit {
		t.Errorf("RecentEvents should be capped at %d, got %d", RecentEventsLimit, len(overview.RecentEvents))
	}
}

func TestFetchOverview_SeekableJournalTakesPrecedence(t *testing.T) {
	t.Parallel()

	evtSeekable := makeTestEvent("seekable", 1)
	evtJournal := makeTestEvent("journal", 1)

	journal := &fakeSeekableJournal{events: []event.Event{evtJournal}}
	seekable := &fakeSeekableJournal{events: []event.Event{evtSeekable}}

	cfg := Config{
		Journal:         journal,
		SeekableJournal: seekable,
	}

	overview := FetchOverview(context.Background(), cfg)

	if len(overview.RecentEvents) != 1 {
		t.Fatalf("expected 1 event, got %d", len(overview.RecentEvents))
	}

	if overview.RecentEvents[0].Type != "seekable" {
		t.Errorf("should use SeekableJournal data, got type %q", overview.RecentEvents[0].Type)
	}
}

func TestFetchOverview_WithStreamReader(t *testing.T) {
	t.Parallel()

	cfg := Config{
		StreamReader: &fakeStreamReader{},
	}

	overview := FetchOverview(context.Background(), cfg)

	// With nil page from fake, TotalAggregates stays "0"
	if overview.TotalAggregates != "0" {
		t.Errorf("TotalAggregates = %q, want %q for nil page", overview.TotalAggregates, "0")
	}
}
