package dashboardui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// countingProjection is a minimal projection.Projection implementation for
// testing the dashboard's projection-host integration. It processes events
// without error, so the worker stays in a healthy state.
type countingProjection struct {
	name  string
	count int64
}

func (p *countingProjection) Name() string { return p.name }
func (p *countingProjection) Handle(_ context.Context, _ event.Event) error {
	p.count++

	return nil
}
func (p *countingProjection) EventTypes() []event.Type { return nil }

// newTestProjectionHost creates a real projectionhost.Host with one
// registered projection and a pre-seeded event in the journal.
func newTestProjectionHost(t *testing.T) *projectionhost.Host {
	t.Helper()

	store := memory.NewMemoryStore()

	aggID := id.NewStreamID()

	evt, err := event.New(event.Type("test.event"), aggID, "TestAggregate", 1, map[string]string{"k": "v"})
	if err != nil {
		t.Fatalf("event.New: %v", err)
	}

	ref := id.StreamRef{Type: "TestAggregate", ID: aggID}
	if err := store.AppendBatch(context.Background(), ref, []event.Event{evt}); err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	host, err := projectionhost.New(store, memory.NewMemoryCheckpointStore())
	if err != nil {
		t.Fatalf("projectionhost.New: %v", err)
	}

	if err := host.Register(&countingProjection{name: "test-projection", count: 0}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	return host
}

// TestOverviewStats_WithProjectionHost exercises the ProjectionHost branch
// of overviewStats, which classifies projection health from WorkerState data.
func TestOverviewStats_WithProjectionHost(t *testing.T) {
	t.Parallel()

	host := newTestProjectionHost(t)

	ctx := t.Context()

	if err := host.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	defer func() { _ = host.Stop() }()

	time.Sleep(200 * time.Millisecond)

	d := MustNew(Config{Journal: stubJournal{}, ProjectionHost: host})

	stats := d.overviewStats(ctx)

	if len(stats.Projections) == 0 {
		t.Fatal("expected at least 1 projection in stats")
	}

	if stats.HealthStatus == "" {
		t.Fatal("expected non-empty HealthStatus when ProjectionHost is set")
	}

	pr := stats.Projections[0]
	if pr.Name != "test-projection" {
		t.Errorf("expected projection name 'test-projection', got %q", pr.Name)
	}
}

// TestOverviewStats_ProjectionHostHealthClassification verifies that
// overviewStats correctly classifies health based on projection worker status.
// After stopping the host, workers transition to "stopped" (statusBad → Unhealthy).
func TestOverviewStats_ProjectionHostHealthClassification(t *testing.T) {
	t.Parallel()

	host := newTestProjectionHost(t)

	ctx := t.Context()

	if err := host.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	_ = host.Stop()

	d := MustNew(Config{Journal: stubJournal{}, ProjectionHost: host})
	stats := d.overviewStats(ctx)

	if len(stats.Projections) == 0 {
		t.Fatal("expected at least 1 projection")
	}

	// After Stop, worker status is "stopped" → statusBad → Unhealthy.
	if stats.HealthStatus != "Unhealthy" {
		t.Errorf("expected HealthStatus 'Unhealthy' after stop, got %q", stats.HealthStatus)
	}

	if stats.HealthKind != statusBad {
		t.Errorf("expected HealthKind %q, got %q", statusBad, stats.HealthKind)
	}
}

// TestDLQIndexHandler_WithProjectionHost verifies that dlqIndexHandler
// renders projection links when ProjectionHost is populated.
func TestDLQIndexHandler_WithProjectionHost(t *testing.T) {
	t.Parallel()

	host := newTestProjectionHost(t)

	ctx := t.Context()

	if err := host.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	defer func() { _ = host.Stop() }()

	time.Sleep(200 * time.Millisecond)

	d := MustNew(Config{Journal: stubJournal{}, ProjectionHost: host})

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/dead-letters", nil)
	rr := httptest.NewRecorder()
	d.dlqIndexHandler(rr, req)

	body := rr.Body.String()

	if !strings.Contains(body, "test-projection") {
		t.Errorf("expected body to contain projection link 'test-projection', got:\n%s", body)
	}

	if !strings.Contains(body, "/dead-letters/test-projection") {
		t.Errorf("expected body to contain link href, got:\n%s", body)
	}
}

// TestDLQIndexHandler_ShowsDeadLetterCounts verifies that the DLQ index
// displays the dead-letter count per projection.
func TestDLQIndexHandler_ShowsDeadLetterCounts(t *testing.T) {
	t.Parallel()

	host := newTestProjectionHost(t)

	ctx := t.Context()

	if err := host.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	defer func() { _ = host.Stop() }()

	time.Sleep(200 * time.Millisecond)

	entries := []projectionhost.DeadLetterEntry{
		{ProjectionName: "test-projection", EventID: "evt-1", EventType: "test.event"},
		{ProjectionName: "test-projection", EventID: "evt-2", EventType: "test.event"},
	}

	d := MustNew(Config{
		Journal:         stubJournal{},
		ProjectionHost:  host,
		DeadLetterStore: &populatedDeadLetterStore{entries: entries},
	})

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/dead-letters", nil)
	rr := httptest.NewRecorder()
	d.dlqIndexHandler(rr, req)

	body := rr.Body.String()

	if !strings.Contains(body, "Dead Letters") {
		t.Fatalf("expected count column header, got:\n%s", body)
	}

	// The count (2) should appear as a badge value.
	if !strings.Contains(body, ">2<") {
		t.Errorf("expected dead-letter count '2' in body, got:\n%s", body)
	}
}

// TestOverview_HealthStatCard verifies that the overview shows a System Health
// stat card when projections are configured.
func TestOverview_HealthStatCard(t *testing.T) {
	t.Parallel()

	host := newTestProjectionHost(t)

	ctx := t.Context()

	if err := host.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	defer func() { _ = host.Stop() }()

	time.Sleep(200 * time.Millisecond)

	d := MustNew(Config{Journal: stubJournal{}, ProjectionHost: host})

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	d.overviewHandler(rr, req)

	body := rr.Body.String()

	if !strings.Contains(body, "System Health") {
		t.Errorf("expected System Health stat card, got:\n%s", body)
	}

	if !strings.Contains(body, "stat-card ok") {
		t.Errorf("expected healthy stat-card with ok variant, got:\n%s", body)
	}
}
