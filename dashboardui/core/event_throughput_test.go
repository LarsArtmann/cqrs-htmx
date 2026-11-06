package core

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// throughputEvent builds a plain domain event stamped at a fixed time.
func throughputEvent(t *testing.T, at time.Time) event.Event {
	t.Helper()

	evt, err := event.New(
		"user.created", id.NewStreamID(), "User", event.Version(1),
		struct{}{},
		event.WithOccurredAt(at),
	)
	if err != nil {
		t.Fatalf("event.New: %v", err)
	}

	return evt
}

func TestLoadEventThroughput_BucketsNewestWindow(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	window := time.Hour
	buckets := 4
	span := window * time.Duration(buckets)

	// Ages are measured from the NEWEST event (base-30m): -3h30m lands in
	// bucket 0, -2h30m and -2h45m share bucket 1 (ages 2h and 2h15m), no
	// event in bucket 2, the newest is bucket 3, and one far-older event
	// falls off the left edge (Older).
	fake := &fakeSeekableJournal{events: []event.Event{
		throughputEvent(t, base.Add(-100*time.Hour)), // off-chart
		throughputEvent(t, base.Add(-3*time.Hour-30*time.Minute)),
		throughputEvent(t, base.Add(-2*time.Hour-45*time.Minute)),
		throughputEvent(t, base.Add(-2*time.Hour-30*time.Minute)),
		throughputEvent(t, base.Add(-30*time.Minute)),
	}}

	got, err := LoadEventThroughput(context.Background(), Config{SeekableJournal: fake}, span, buckets)
	if err != nil {
		t.Fatalf("LoadEventThroughput: %v", err)
	}

	if got.Window != window {
		t.Errorf("Window = %v, want %v", got.Window, window)
	}

	wantCounts := []float64{1, 2, 0, 1}
	for i, want := range wantCounts {
		if got.Counts[i] != want {
			t.Errorf("Counts[%d] = %v, want %v (all: %v)", i, got.Counts[i], want, got.Counts)
		}
	}

	if got.InWindow != 4 {
		t.Errorf("InWindow = %d, want 4", got.InWindow)
	}

	if got.Older != 1 {
		t.Errorf("Older = %d, want 1", got.Older)
	}

	if got.Scanned != 5 {
		t.Errorf("Scanned = %d, want 5", got.Scanned)
	}

	if len(got.Labels) != buckets {
		t.Fatalf("Labels len = %d, want %d", len(got.Labels), buckets)
	}

	// Hourly buckets label their bucket-start time; the newest bucket starts
	// one window before the newest event.
	if want := base.Add(-30 * time.Minute).Add(-window).Format("Jan 02 15:04"); got.Labels[buckets-1] != want {
		t.Errorf("Labels[newest] = %q, want %q", got.Labels[buckets-1], want)
	}
}

func TestLoadEventThroughput_EmptyJournalIsEmptyState(t *testing.T) {
	t.Parallel()

	got, err := LoadEventThroughput(context.Background(),
		Config{SeekableJournal: &fakeSeekableJournal{}}, 24*time.Hour, 12)
	if err != nil {
		t.Fatalf("LoadEventThroughput: %v", err)
	}

	if got.InWindow != 0 || len(got.Counts) != 0 || len(got.Labels) != 0 {
		t.Errorf("empty journal must produce the empty state, got %+v", got)
	}
}

func TestLoadEventThroughput_RejectsBadArguments(t *testing.T) {
	t.Parallel()

	cfg := Config{SeekableJournal: &fakeSeekableJournal{}}

	if _, err := LoadEventThroughput(context.Background(), cfg, time.Hour, 0); err == nil {
		t.Error("buckets=0 must be rejected")
	}

	if _, err := LoadEventThroughput(context.Background(), cfg, 0, 4); err == nil {
		t.Error("span=0 must be rejected")
	}
}
