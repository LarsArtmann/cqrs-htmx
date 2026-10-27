package core

import (
	"context"
	"strconv"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// makeVersionedEvents builds n ordered events on one stream (versions 1..n)
// so head reads and tail reads are distinguishable by version number.
func makeVersionedEvents(n int) []event.Event {
	streamID := id.NewStreamID()

	events := make([]event.Event, 0, n)

	for i := range n {
		evt, _ := event.New(
			event.Type("test.event"),
			streamID,
			"TestAgg",
			event.Version(i+1),
			map[string]string{"k": "v"},
		)

		events = append(events, evt)
	}

	return events
}

// TestLoadTailEvents_SeekableJournal pins that the tail read returns the
// NEWEST events (a zero-cursor ReadFrom returns the OLDEST n — the bug this
// replaces showed the journal head where the overview promised recency) and
// the exact journal total.
func TestLoadTailEvents_SeekableJournal(t *testing.T) {
	t.Parallel()

	cfg := Config{SeekableJournal: &fakeSeekableJournal{events: makeVersionedEvents(12)}}

	tail, total, err := LoadTailEvents(context.Background(), cfg, EventFilter{}, 5)
	if err != nil {
		t.Fatalf("LoadTailEvents: %v", err)
	}

	if total != 12 {
		t.Errorf("total = %d, want 12", total)
	}

	if len(tail) != 5 {
		t.Fatalf("tail len = %d, want 5", len(tail))
	}

	for i, evt := range tail {
		want := 8 + i
		if evt.Version().Int() != want {
			t.Errorf("tail[%d] version = %d, want %d", i, evt.Version().Int(), want)
		}
	}
}

// TestLoadTailEvents_FilterKeepsNewestMatches pins that filters combine with
// the tail read: the newest matching events survive, not the head ones.
func TestLoadTailEvents_FilterKeepsNewestMatches(t *testing.T) {
	t.Parallel()

	events := make([]event.Event, 0, 10)

	for i := range 10 {
		eventType := "odd.keeper"
		if i%2 == 0 {
			eventType = "even.skipped"
		}

		evt, _ := event.New(
			event.Type(eventType),
			id.NewStreamID(),
			"TestAgg",
			event.Version(i+1),
			map[string]string{"k": "v"},
		)

		events = append(events, evt)
	}

	cfg := Config{SeekableJournal: &fakeSeekableJournal{events: events}}

	tail, total, err := LoadTailEvents(
		context.Background(),
		cfg,
		EventFilter{Type: "odd.keeper"},
		2,
	)
	if err != nil {
		t.Fatalf("LoadTailEvents: %v", err)
	}

	if total != 10 {
		t.Errorf("total counts every event: got %d, want 10", total)
	}

	if len(tail) != 2 {
		t.Fatalf("tail len = %d, want 2", len(tail))
	}

	for _, evt := range tail {
		if evt.Version().Int() < 7 {
			t.Errorf("filtered tail kept version %d; want only the newest matches (9, 7)", evt.Version().Int())
		}
	}
}

// TestLoadTailEvents_JournalFallback mirrors the seekable contract for
// Journal-only configurations.
func TestLoadTailEvents_JournalFallback(t *testing.T) {
	t.Parallel()

	cfg := Config{Journal: &fakeSeekableJournal{events: makeVersionedEvents(9)}}

	tail, total, err := LoadTailEvents(context.Background(), cfg, EventFilter{}, 4)
	if err != nil {
		t.Fatalf("LoadTailEvents: %v", err)
	}

	if total != 9 {
		t.Errorf("total = %d, want 9", total)
	}

	if len(tail) != 4 || tail[len(tail)-1].Version().Int() != 9 || tail[0].Version().Int() != 6 {
		t.Errorf("tail window = %d..%d, want 6..9", tail[0].Version().Int(), tail[len(tail)-1].Version().Int())
	}
}

// TestLoadTailEvents_MultiBatch exercises the cursor-paging loop across
// multiple tailScanBatch windows.
func TestLoadTailEvents_MultiBatch(t *testing.T) {
	t.Parallel()

	const n = 2*tailScanBatch + 7

	cfg := Config{SeekableJournal: &fakeSeekableJournal{events: makeVersionedEvents(n)}}

	tail, total, err := LoadTailEvents(context.Background(), cfg, EventFilter{}, 3)
	if err != nil {
		t.Fatalf("LoadTailEvents: %v", err)
	}

	if total != n {
		t.Errorf("total = %d, want %d", total, n)
	}

	if len(tail) != 3 {
		t.Fatalf("tail len = %d, want 3", len(tail))
	}

	want := n - 2
	for _, evt := range tail {
		if evt.Version().Int() != want {
			t.Errorf("tail version = %d, want %d", evt.Version().Int(), want)
		}

		want++
	}
}

// TestLoadRecentEvents_JournalFallbackHonorsCursor pins the fallback seek:
// without it the Next link re-served page one forever on Journal-only
// configurations.
func TestLoadRecentEvents_JournalFallbackHonorsCursor(t *testing.T) {
	t.Parallel()

	events := makeVersionedEvents(10)
	cfg := Config{Journal: &fakeSeekableJournal{events: events}}

	got, err := LoadRecentEvents(context.Background(), cfg, events[4].ID(), 3)
	if err != nil {
		t.Fatalf("LoadRecentEvents: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}

	for i, evt := range got {
		if evt.Version().Int() != 6+i {
			t.Errorf("got[%d] version = %d, want %d", i, evt.Version().Int(), 6+i)
		}
	}
}

// TestLoadFilteredEvents_JournalFallbackHonorsCursor mirrors the cursor
// contract for the filtered fallback path.
func TestLoadFilteredEvents_JournalFallbackHonorsCursor(t *testing.T) {
	t.Parallel()

	events := makeVersionedEvents(10)
	cfg := Config{Journal: &fakeSeekableJournal{events: events}}

	got, err := LoadFilteredEvents(
		context.Background(),
		cfg,
		events[6].ID(),
		EventFilter{},
		50,
	)
	if err != nil {
		t.Fatalf("LoadFilteredEvents: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("len = %d, want 3 (events after the cursor)", len(got))
	}

	if got[0].Version().Int() != 8 {
		t.Errorf("first version = %d, want 8", got[0].Version().Int())
	}
}

// TestFindEventNeighbors_BeyondOldWindow pins neighbor resolution for events
// past the first batch — the old head-only scan silently dropped prev/next
// navigation for them.
func TestFindEventNeighbors_BeyondOldWindow(t *testing.T) {
	t.Parallel()

	events := makeVersionedEvents(600)
	cfg := Config{SeekableJournal: &fakeSeekableJournal{events: events}}

	target := events[550]

	prevID, nextID := FindEventNeighbors(context.Background(), cfg, target.ID())
	if prevID != events[549].ID().String() {
		t.Errorf("prevID = %q, want event 550's predecessor", prevID)
	}

	if nextID != events[551].ID().String() {
		t.Errorf("nextID = %q, want event 550's successor", nextID)
	}
}

// TestFindEventNeighbors_Boundaries pins the first/last edge behavior on a
// Journal-only configuration.
func TestFindEventNeighbors_Boundaries(t *testing.T) {
	t.Parallel()

	events := makeVersionedEvents(5)
	cfg := Config{Journal: &fakeSeekableJournal{events: events}}

	prevID, nextID := FindEventNeighbors(context.Background(), cfg, events[0].ID())
	if prevID != "" || nextID != events[1].ID().String() {
		t.Errorf("first event neighbors = (%q, %q), want (\"\", next)", prevID, nextID)
	}

	prevID, nextID = FindEventNeighbors(context.Background(), cfg, events[4].ID())
	if prevID != events[3].ID().String() || nextID != "" {
		t.Errorf("last event neighbors = (%q, %q), want (prev, \"\")", prevID, nextID)
	}
}

// TestFetchOverview_RecentEventsAreNewestFirst pins the overview contract:
// the card lists the newest events at the top, newest first.
func TestFetchOverview_RecentEventsAreNewestFirst(t *testing.T) {
	t.Parallel()

	cfg := Config{SeekableJournal: &fakeSeekableJournal{events: makeVersionedEvents(RecentEventsLimit + 5)}}

	overview := FetchOverview(context.Background(), cfg)

	if len(overview.RecentEvents) != RecentEventsLimit {
		t.Fatalf("len = %d, want %d", len(overview.RecentEvents), RecentEventsLimit)
	}

	for i, e := range overview.RecentEvents {
		want := RecentEventsLimit + 5 - i
		if e.Version != strconv.Itoa(want) {
			t.Errorf("RecentEvents[%d] version = %q, want %d (newest first)", i, e.Version, want)
		}
	}
}
