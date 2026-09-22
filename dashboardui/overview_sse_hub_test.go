package dashboardui

import (
	"context"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
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
