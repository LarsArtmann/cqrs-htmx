package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	dashboardui "github.com/larsartmann/cqrs-htmx/dashboardui/v4"
	"github.com/larsartmann/cqrs-htmx/dashboardui/v4/systembridge"
	systemadapter "github.com/larsartmann/cqrs-htmx/systemadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// TestDashboard_FromSystem_PanelsLightUp proves the FromSystem bridge end to
// end: a REAL system.New instance (declarative DomainConfig on the memory
// preset) feeds a dashboard Config whose panels render live data — events
// from the store, projections from the system's host — through the mounted
// routes.
func TestDashboard_FromSystem_PanelsLightUp(t *testing.T) {
	ctx := context.Background()

	sys, err := system.New(
		ctx,
		systemadapter.DomainConfig(),
		systemadapter.RecommendedMemoryDeployment(),
	)
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	if err := sys.Start(ctx); err != nil {
		_ = sys.Close()

		t.Fatalf("sys.Start: %v", err)
	}

	t.Cleanup(func() { _ = sys.Close() })

	cfg := dashboardui.FromSystem(sys)
	cfg.Title = "FromSystem integration"
	cfg.BasePath = "/cqrs"

	dash, err := dashboardui.New(cfg)
	if err != nil {
		t.Fatalf("dashboardui.New(FromSystem(sys)): %v", err)
	}

	assertSystemCapabilities(t, dash)
	assertSystemPanelsRender(t, dash)
}

// TestDashboard_TelemetryFromSystem proves the systemadapter telemetry
// wiring: topology, engine health, placements, and stats providers mapped
// from a real system render through the dashboard's telemetry page and
// compose into healthz.
func TestDashboard_TelemetryFromSystem(t *testing.T) {
	ctx := context.Background()

	sys, err := system.New(
		ctx,
		systemadapter.DomainConfig(),
		systemadapter.RecommendedMemoryDeployment(),
	)
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	if err := sys.Start(ctx); err != nil {
		_ = sys.Close()

		t.Fatalf("sys.Start: %v", err)
	}

	t.Cleanup(func() { _ = sys.Close() })

	cfg := dashboardui.FromSystem(sys)
	systembridge.WireTelemetry(&cfg, sys)
	cfg.Title = "Telemetry integration"

	dash, err := dashboardui.New(cfg)
	if err != nil {
		t.Fatalf("dashboardui.New: %v", err)
	}

	mux := http.NewServeMux()
	dash.Mount(mux, "/cqrs/")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/cqrs/telemetry", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /cqrs/telemetry: status %d, want 200", rec.Code)
	}

	body := rec.Body.String()
	for _, want := range []string{"Topology", "source-of-truth", "memory"} {
		if !strings.Contains(body, want) {
			t.Errorf("telemetry page missing %q", want)
		}
	}

	healthRec := httptest.NewRecorder()
	mux.ServeHTTP(healthRec, httptest.NewRequest(http.MethodGet, "/cqrs/-/healthz", nil))

	if healthRec.Code != http.StatusOK {
		t.Fatalf("healthz: status %d, want 200", healthRec.Code)
	}

	if !strings.Contains(healthRec.Body.String(), `"engines"`) {
		t.Errorf("healthz must list engine health after wiring: %s", healthRec.Body.String())
	}
}

func assertSystemCapabilities(t *testing.T, dash *dashboardui.Dashboard) {
	t.Helper()

	caps := dash.Capabilities()
	if !caps.HasEventRead() {
		t.Fatal("event read capability missing — the system's event store did not map")
	}

	if !caps.ProjectionHost {
		t.Fatal("projection host capability missing — the system's host did not map")
	}

	if !caps.StreamReader {
		t.Fatal("stream reader capability missing — neither native nor derived reader wired")
	}
}

func assertSystemPanelsRender(t *testing.T, dash *dashboardui.Dashboard) {
	t.Helper()

	panels := map[string]bool{}
	for _, route := range dash.Routes() {
		panels[route.Panel] = true
	}

	for _, panel := range []string{"Overview", "Events", "Aggregates", "Projections"} {
		if !panels[panel] {
			t.Errorf("panel %q absent from the route manifest; panels: %v", panel, panels)
		}
	}

	mux := http.NewServeMux()
	dash.Mount(mux, "/cqrs/")

	for _, target := range []string{"/cqrs/", "/cqrs/events", "/cqrs/projections"} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))

		if rec.Code != http.StatusOK {
			t.Errorf("GET %s: status %d, want 200", target, rec.Code)
		}
	}
}
