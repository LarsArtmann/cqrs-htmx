package dashboardui

import (
	"context"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-sse"
	"github.com/larsartmann/templ-components/display"
)

func newHubTestDashboard(t *testing.T, withBus bool) *Dashboard {
	t.Helper()

	cfg := Config{
		Title:           "t",
		SeekableJournal: &fakeSeekableJournal{},
	}
	if withBus {
		cfg.EventBus = eventtest.NewFakeBus()
	}

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	t.Cleanup(func() { d.Close() })

	return d
}

func TestOverview_SSEHubCardReflectsSubscribers(t *testing.T) {
	t.Parallel()

	d := newHubTestDashboard(t, true)

	if d.broadcaster == nil {
		t.Fatal("expected broadcaster for EventBus config")
	}

	ch := d.broadcaster.Subscribe()
	defer d.broadcaster.Unsubscribe(ch)

	stats := d.overviewStats(context.Background())
	if stats.SSEHub == nil {
		t.Fatal("expected SSEHub populated when broadcaster exists")
	}

	if stats.SSEHub.Subscribers != 1 {
		t.Errorf("Subscribers: got %d, want 1", stats.SSEHub.Subscribers)
	}

	grid := goldenRender(t, overviewStatGrid(stats))

	if !strings.Contains(grid, `id="stat-sse-hub"`) {
		t.Errorf("overview grid missing the SSE hub card\nrendered grid:\n%s", grid)
	}

	if !strings.Contains(grid, ">1 <") {
		t.Errorf("SSE hub card should show subscriber count 1\ncard grid:\n%s", grid)
	}
}

func TestOverview_SSEHubCardClosedAfterShutdown(t *testing.T) {
	t.Parallel()

	d := newHubTestDashboard(t, true)
	d.Close()

	stats := d.overviewStats(context.Background())
	if stats.SSEHub == nil {
		t.Fatal("expected SSEHub populated even after close")
	}

	if !stats.SSEHub.Closed {
		t.Error("expected SSEHub.Closed=true after Close")
	}

	grid := goldenRender(t, overviewStatGrid(stats))

	if !strings.Contains(grid, ">closed <") {
		t.Errorf("SSE hub card should show closed state\ncard grid:\n%s", grid)
	}
}

func TestOverview_NoSSEHubCardWithoutEventBus(t *testing.T) {
	t.Parallel()

	d := newHubTestDashboard(t, false)

	stats := d.overviewStats(context.Background())
	if stats.SSEHub != nil {
		t.Fatal("expected SSEHub nil without EventBus")
	}

	grid := goldenRender(t, overviewStatGrid(stats))

	if strings.Contains(grid, `id="stat-sse-hub"`) {
		t.Errorf("SSE hub card must not render without a broadcaster\ncard grid:\n%s", grid)
	}
}

// TestOverview_SSEHubCardCountsDrops proves the drop-observability wiring
// (A2): a subscriber whose buffer overflows surfaces in the hub card value
// and downgrades the tone to amber — drops stop being silent.
func TestOverview_SSEHubCardCountsDrops(t *testing.T) {
	t.Parallel()

	d := newHubTestDashboard(t, true)

	if d.dropStats == nil {
		t.Fatal("expected dropStats wired alongside the broadcaster")
	}

	ch := d.broadcaster.Subscribe()
	defer d.broadcaster.Unsubscribe(ch)

	// Default buffer is 64: fill it, then overflow once.
	for range 65 {
		d.broadcaster.Broadcast(sse.Event{Event: "overflow"})
	}

	stats := d.overviewStats(context.Background())
	if stats.SSEHub.Drops != 1 {
		t.Errorf("Drops: got %d, want 1 (65 broadcasts into a 64-buffer, unread)", stats.SSEHub.Drops)
	}

	if got := sseHubTone(stats); got != display.StatToneYellow {
		t.Errorf("sseHubTone with drops = %v, want yellow", got)
	}
	if got := sseHubValue(stats); got != "1 · 1 dropped" {
		t.Errorf("sseHubValue with drops = %q, want %q", got, "1 · 1 dropped")
	}

	grid := goldenRender(t, overviewStatGrid(stats))
	if !strings.Contains(grid, "dropped") {
		t.Errorf("SSE hub card should surface the drop count\ncard grid:\n%s", grid)
	}
}
