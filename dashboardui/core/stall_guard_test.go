package core

import (
	"context"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// stallingJournal serves the first limit events regardless of the cursor —
// the shape that looped the pre-guard walkers forever (the same class as the
// fake query journal that OOM-killed CI runners on 2026-10-26).
type stallingJournal struct {
	events []event.Event
}

func (s *stallingJournal) ReadAll(_ context.Context) ([]event.Event, error) {
	return s.events, nil
}

func (s *stallingJournal) ReadFrom(
	_ context.Context,
	_ id.EventID,
	limit int,
) ([]event.Event, error) {
	return s.events[:limit], nil
}

func TestLoadTailEvents_CursorStallIsAnErrorNotAHang(t *testing.T) {
	t.Parallel()

	cfg := Config{SeekableJournal: &stallingJournal{events: makeVersionedEvents(tailScanBatch)}}

	_, _, err := LoadTailEvents(context.Background(), cfg, EventFilter{}, 5)
	if err == nil {
		t.Fatal("expected a cursor-stall error, got nil")
	}

	if !strings.Contains(err.Error(), "cursor did not advance") {
		t.Errorf("expected a cursor-stall diagnosis, got: %v", err)
	}
}

func TestLoadEventByID_CursorStallIsAnErrorNotAHang(t *testing.T) {
	t.Parallel()

	cfg := Config{SeekableJournal: &stallingJournal{events: makeVersionedEvents(tailScanBatch)}}

	_, err := LoadEventByID(context.Background(), cfg, id.NewEventID())
	if err == nil {
		t.Fatal("expected a cursor-stall error, got nil")
	}

	if !strings.Contains(err.Error(), "cursor did not advance") {
		t.Errorf("expected a cursor-stall diagnosis, got: %v", err)
	}
}

func TestFindEventNeighbors_CursorStallDegradesToNoNavigation(t *testing.T) {
	t.Parallel()

	cfg := Config{SeekableJournal: &stallingJournal{events: makeVersionedEvents(tailScanBatch)}}

	prevID, nextID := FindEventNeighbors(context.Background(), cfg, id.NewEventID())
	if prevID != "" || nextID != "" {
		t.Errorf("expected no navigation on a stalled cursor, got prev=%q next=%q", prevID, nextID)
	}
}
