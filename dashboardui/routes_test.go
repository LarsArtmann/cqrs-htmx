package dashboardui

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

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

// muxPattern maps a consumer-facing pattern to the ServeMux registration form:
// only the overview root differs ("/" registers as "/{$}").
func muxPattern(pattern string) string {
	if pattern == "/" {
		return "/{$}"
	}

	return pattern
}

func TestRoutes_MatchRegisteredMuxPatterns(t *testing.T) {
	d := mustTestDashboardWithConfig(t, fullRoutesConfig())

	mux, ok := d.Handler().(*http.ServeMux)
	if !ok {
		t.Fatalf("Handler() should expose the internal mux, got %T", d.Handler())
	}

	for _, route := range d.Routes() {
		req := httptest.NewRequest(route.Method, samplePath(route.Pattern), nil)
		_, matched := mux.Handler(req)

		if want := route.Method + " " + muxPattern(route.Pattern); matched != want {
			t.Errorf("manifest lists %s %s, but the mux serves %q (drift between Routes() and routes())",
				route.Method, route.Pattern, matched)
		}
	}
}

func TestRoutes_FullConfigExposesEveryPanel(t *testing.T) {
	d := mustTestDashboardWithConfig(t, fullRoutesConfig())

	wantPanels := map[string]int{
		"Assets":        4,
		"Observability": 3,
		"Overview":      1,
		"Live updates":  1,
		"Events":        2,
		"Aggregates":    2,
		"Projections":   4,
		"Dead letters":  6,
		"Commands":      2,
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

	// 4 assets + 3 observability probes + the overview + 2 event routes.
	const want = 10

	if len(routes) != want {
		t.Fatalf("Journal-only manifest has %d routes, want %d: %+v", len(routes), want, routes)
	}

	for _, route := range routes {
		switch route.Panel {
		case "Assets", "Observability", "Overview", "Events":
		default:
			t.Errorf("Journal-only manifest must not list panel %q (route %s %s)",
				route.Panel, route.Method, route.Pattern)
		}
	}
}
