package dashboardui

import (
	"context"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

// routeProbeTimeout bounds every probe request in route-walking tests: a
// never-ending route (SSE stream, long poll) degrades to one bounded slow
// probe instead of hanging the suite at the package wall-clock timeout.
const routeProbeTimeout = 5 * time.Second

// fullRoutesConfig wires every capability so the manifest covers all
// conditional panels, read-write included.
func fullRoutesConfig() Config {
	return Config{
		EventSource:     &fakeEventSource{},
		Journal:         &stubJournal{},
		SeekableJournal: &fakeSeekableJournal{},
		StreamReader:    &fakeStreamReader{},
		EventByIDLoader: &fakeEventByIDLoader{},
		ProjectionHost:  &projectionhost.Host{},
		DeadLetterStore: &populatedDeadLetterStore{},
		CommandJournal:  &fakeCommandJournal{},
		QueryJournal:    &fakeQueryJournal{},
		SnapshotStore:   &fakeSnapshotStore{},
		EventBus:        eventtest.NewFakeBus(),
	}
}

// wildcardRE matches manifest wildcard segments like {id} or {projection}.
var wildcardRE = regexp.MustCompile(`\{[^}]+\}`)

// samplePath turns a manifest pattern into a requestable path by substituting
// a placeholder value for each wildcard.
func samplePath(pattern string) string {
	return wildcardRE.ReplaceAllString(pattern, "sample")
}

func TestRoutes_MatchRegisteredMuxPatterns(t *testing.T) {
	d := mustTestDashboardWithConfig(t, fullRoutesConfig())

	handler := d.Handler()

	for _, route := range d.Routes() {
		// Streams never complete: a probe at the SSE endpoint would block
		// the recorder forever (the 600s package-timeout hang). Streams are
		// registered in the same mux pattern table, so skipping the live
		// probe loses no registration coverage.
		if route.Panel == panelLiveUpdates {
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), routeProbeTimeout)

		req := httptest.NewRequest(route.Method, samplePath(route.Pattern), nil).WithContext(ctx)
		req.Header.Set("HX-Request", "true")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		cancel()

		// A registration miss lands on the catch-all (styled 404 for GETs)
		// or ServeMux's bare form (method-specific misses) — either way the
		// exact "404 page not found" line is the bare form's signature, and no
		// matched route can produce it (error pages render real markup).
		if strings.HasSuffix(rec.Body.String(), "404 page not found\n") {
			t.Errorf("manifest lists %s %s, but the mux does not serve it (drift between Routes() and routes())",
				route.Method, route.Pattern)
		}
	}
}

func TestRoutes_FullConfigExposesEveryPanel(t *testing.T) {
	d := mustTestDashboardWithConfig(t, fullRoutesConfig())

	wantPanels := map[string]int{
		"Assets":        5,
		"Observability": 3,
		"Overview":      1,
		"Live updates":  1,
		"Events":        4,
		"Aggregates":    2,
		"Projections":   4,
		"Dead letters":  6,
		"Commands":      2,
		"Command fates": 1,
		"Throughput":    1,
		"Queries":       2,
		"Time travel":   2,
		"Snapshots":     3,
	}

	got := map[string]int{}

	for _, route := range d.Routes() {
		got[route.Panel]++
	}

	for panel, count := range wantPanels {
		if got[panel] != count {
			t.Errorf("panel %q: %d routes, want %d", panel, got[panel], count)
		}
	}

	if len(got) != len(wantPanels) {
		t.Errorf("manifest covers %d panels, want %d (%v)", len(got), len(wantPanels), got)
	}
}

func TestRoutes_WriteRoutesMarkedAndDroppedByReadOnly(t *testing.T) {
	writable := mustTestDashboardWithConfig(t, fullRoutesConfig())

	var writeRoutes int

	for _, route := range writable.Routes() {
		if route.Write {
			writeRoutes++
		}
	}

	const wantWriteRoutes = 5 // projection reset, DLQ replay/delete/purge, snapshot delete

	if writeRoutes != wantWriteRoutes {
		t.Errorf("full config marks %d write routes, want %d", writeRoutes, wantWriteRoutes)
	}

	cfg := fullRoutesConfig()
	cfg.ReadOnly = true

	readOnly := mustTestDashboardWithConfig(t, cfg)

	routes := readOnly.Routes()
	if len(routes) == 0 {
		t.Fatal("ReadOnly dashboard must still list read routes")
	}

	for _, route := range routes {
		if route.Write {
			t.Errorf("ReadOnly dashboard must not list write route %s %s", route.Method, route.Pattern)
		}
	}

	if got := len(routes); got != len(writable.Routes())-wantWriteRoutes {
		t.Errorf("ReadOnly manifest has %d routes, want %d", got, len(writable.Routes())-wantWriteRoutes)
	}
}

func TestRoutes_MinimalConfigListsOnlyUnconditionalRoutes(t *testing.T) {
	d := mustTestDashboardWithConfig(t, Config{Journal: &stubJournal{}})

	routes := d.Routes()

	// 5 assets + 3 observability probes + the overview + 4 event routes
	// (the index, the QUERY search rung, its POST twin, the detail page)
	// + the command-fates window (journal-derived, same gate as events)
	// + throughput.
	const want = 15

	if len(routes) != want {
		t.Fatalf("Journal-only manifest has %d routes, want %d: %+v", len(routes), want, routes)
	}

	for _, route := range routes {
		switch route.Panel {
		case "Assets", "Observability", "Overview", "Events", "Command fates", "Throughput":
		default:
			t.Errorf("Journal-only manifest must not list panel %q (route %s %s)",
				route.Panel, route.Method, route.Pattern)
		}
	}
}
