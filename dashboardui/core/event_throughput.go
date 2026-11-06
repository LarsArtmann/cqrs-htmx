package core

import (
	"context"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

// throughputScanLimit caps the journal scan behind the throughput chart: the
// rolling tail keeps at most this many events, so a huge journal costs a
// bounded scan and the chart reflects the newest window regardless.
const throughputScanLimit = 10_000

// EventThroughput is the bucketed event-count time series behind the
// throughput chart: how many events the journal recorded per fixed-width
// time window.
type EventThroughput struct {
	// Labels are the X-axis labels, one per bucket, left (oldest) to right
	// (newest); each labels its bucket's START time.
	Labels []string

	// Counts is the per-bucket event count, parallel to Labels.
	Counts []float64

	// Window is the fixed width of one bucket.
	Window time.Duration

	// InWindow counts events inside the charted span.
	InWindow int

	// Older counts scanned events that predate the charted span (they fell
	// off the left edge — the chart is a newest-window view, not a total).
	Older int

	// Scanned is every event the journal scan saw (before window bucketing).
	Scanned int
}

// LoadEventThroughput buckets the journal's newest events into buckets
// fixed-width windows spanning span, ENDING at the newest event's
// OccurredAt (the last bucket is partial by design — "so far this window").
// Events older than the span are counted in Older, not charted.
//
// The scan reuses [LoadTailEvents]' rolling-tail paging (journals read
// oldest-first, so a zero-cursor window is the HEAD), capped at
// throughputScanLimit events.
func LoadEventThroughput(
	ctx context.Context,
	cfg Config,
	span time.Duration,
	buckets int,
) (EventThroughput, error) {
	if buckets <= 0 {
		return EventThroughput{}, errorfamily.NewRejection("dashboardui.throughput.bad_buckets",
			"buckets must be positive")
	}

	if span <= 0 {
		return EventThroughput{}, errorfamily.NewRejection("dashboardui.throughput.bad_span",
			"span must be positive")
	}

	window := span / time.Duration(buckets)

	tail, scanned, err := LoadTailEvents(ctx, cfg, EventFilter{}, throughputScanLimit)
	if err != nil {
		return EventThroughput{}, err
	}

	out := EventThroughput{Window: window, Scanned: scanned}
	if len(tail) == 0 {
		return out, nil
	}

	newest := tail[len(tail)-1].OccurredAt()

	out.Labels = make([]string, buckets)
	out.Counts = make([]float64, buckets)

	labelFormat := "Jan 02 15:04"
	if window >= 24*time.Hour {
		labelFormat = "Jan 02"
	}

	for i := range out.Counts {
		// Bucket i covers ages [(buckets-1-i)*window, (buckets-i)*window), so
		// its left edge sits (buckets-i) windows before the newest event.
		bucketStart := newest.Add(-window * time.Duration(buckets-i))
		out.Labels[i] = bucketStart.Format(labelFormat)
	}

	for _, evt := range tail {
		age := newest.Sub(evt.OccurredAt())
		age = max(age, 0) // clock skew: an event cannot be older than itself — newest bucket

		idx := buckets - 1 - int(age/window)
		if idx < 0 {
			out.Older++

			continue
		}

		out.Counts[idx]++
		out.InWindow++
	}

	return out, nil
}
