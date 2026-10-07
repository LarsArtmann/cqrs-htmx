package dashboardui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/cqrs-htmx/dashboardui/v4/core"
)

func telemetryTestConfig() Config {
	return Config{
		Journal: &stubJournal{},
		Topology: func(context.Context) (core.TopologyView, error) {
			return core.TopologyView{
				Instances: []core.InstanceView{
					{
						Name:         "primary",
						Role:         "source-of-truth",
						EngineName:   "engine-1",
						DriverName:   "sqlite",
						Durability:   "strict",
						HealthStatus: "healthy",
						Collections:  []string{"events", "commands"},
					},
				},
				Buses: []core.BusView{{Name: "bus", Driver: "gochannel", Mode: "fan-out"}},
				ProjectionHost: &core.ProjectionHostView{
					Started: true,
					Workers: 6,
				},
			}, nil
		},
		EngineHealths: func(context.Context) []core.EngineHealthView {
			return []core.EngineHealthView{
				{Name: "engine-1", Healthy: true},
				{Name: "engine-2", Healthy: false, Error: "connection refused"},
			}
		},
		Placements: func(context.Context) ([]core.PlacementView, error) {
			return []core.PlacementView{
				{Query: "user_by_email", Engine: "engine-1", ADT: "Map", Volume: 100, EstLatencyMs: 0.25},
			}, nil
		},
		EngineStats: func(context.Context) []core.EngineStatsView {
			return []core.EngineStatsView{
				{
					Name:       "engine-1",
					HasRTT:     true,
					Samples:    42,
					EWMA:       1500 * time.Microsecond,
					P95:        3 * time.Millisecond,
					LastSample: time.Now(),
				},
				{Name: "engine-2", HasRTT: false},
			}
		},
	}
}

func TestTelemetryPage_RendersAllSections(t *testing.T) {
	d := mustTestDashboardWithConfig(t, telemetryTestConfig())

	rec := httptest.NewRecorder()
	d.telemetryHandler(rec, httptest.NewRequest(http.MethodGet, "/telemetry", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	body := rec.Body.String()

	for _, want := range []string{
		"Topology",
		"source-of-truth",
		"engine-1",
		"bus", // Buses section header row label comes from the bus name
		"Projection host:",
		"Query Placements",
		"user_by_email",
		"Engine Latency",
		"no tracker",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("telemetry page missing %q", want)
		}
	}
}

func TestTelemetry_RouteAndNavGatedByCapability(t *testing.T) {
	with := mustTestDashboardWithConfig(t, telemetryTestConfig())
	found := false

	for _, route := range with.Routes() {
		if route.Panel == panelTelemetry {
			found = true
		}
	}

	if !found {
		t.Error("Telemetry panel must be in the route manifest when providers are set")
	}

	without := mustTestDashboardWithConfig(t, Config{Journal: &stubJournal{}})

	for _, route := range without.Routes() {
		if route.Panel == panelTelemetry {
			t.Error("Telemetry panel must be absent without providers")
		}
	}

	navHasTelemetry := false

	for _, item := range buildNav(without.caps) {
		if item.Href == "/telemetry" {
			navHasTelemetry = true
		}
	}

	if navHasTelemetry {
		t.Error("nav must not list Telemetry without providers")
	}

	rec := httptest.NewRecorder()
	without.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/telemetry", nil))

	if rec.Code != http.StatusNotFound {
		t.Errorf("GET /telemetry without providers: status %d, want 404 via catch-all", rec.Code)
	}
}

func TestTelemetry_ProviderErrorRenderedInline(t *testing.T) {
	cfg := Config{
		Journal: &stubJournal{},
		Topology: func(context.Context) (core.TopologyView, error) {
			return core.TopologyView{}, context.DeadlineExceeded
		},
		Placements: func(context.Context) ([]core.PlacementView, error) {
			return []core.PlacementView{{Query: "q", Engine: "e", ADT: "Map"}}, nil
		},
	}
	d := mustTestDashboardWithConfig(t, cfg)

	rec := httptest.NewRecorder()
	d.telemetryHandler(rec, httptest.NewRequest(http.MethodGet, "/telemetry", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("one failing provider must not 500 the page; status = %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "topology unavailable") {
		t.Error("topology section must show its inline error")
	}

	if !strings.Contains(body, "q") {
		t.Error("placements section must still render")
	}
}

func TestTelemetry_HealthzListsEnginesAndReadyzGatesOnHealth(t *testing.T) {
	d := mustTestDashboardWithConfig(t, telemetryTestConfig())

	mux, ok := d.Handler().(*http.ServeMux)
	if !ok {
		t.Fatalf("Handler() should expose the internal mux, got %T", d.Handler())
	}

	healthRec := httptest.NewRecorder()
	mux.ServeHTTP(healthRec, httptest.NewRequest(http.MethodGet, "/-/healthz", nil))

	if healthRec.Code != http.StatusOK {
		t.Fatalf("healthz status = %d, liveness must stay 200 with a sick engine", healthRec.Code)
	}

	body := healthRec.Body.String()
	for _, want := range []string{`"engines"`, "engine-2", "connection refused"} {
		if !strings.Contains(body, want) {
			t.Errorf("healthz payload missing %q: %s", want, body)
		}
	}

	readyRec := httptest.NewRecorder()
	mux.ServeHTTP(readyRec, httptest.NewRequest(http.MethodGet, "/-/readyz", nil))

	if readyRec.Code != http.StatusServiceUnavailable {
		t.Fatalf("readyz status = %d, want 503 while an engine is unhealthy", readyRec.Code)
	}

	if !strings.Contains(readyRec.Body.String(), "engine_unhealthy") {
		t.Errorf("readyz payload must name the failure mode: %s", readyRec.Body.String())
	}
}

func TestTelemetry_ProbesUnchangedWithoutProvider(t *testing.T) {
	d := mustTestDashboardWithConfig(t, Config{Journal: &stubJournal{}})

	mux, ok := d.Handler().(*http.ServeMux)
	if !ok {
		t.Fatalf("Handler() should expose the internal mux, got %T", d.Handler())
	}

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/-/healthz", nil))

	if strings.Contains(rec.Body.String(), "engines") {
		t.Errorf("healthz must not carry an engines key without a provider: %s", rec.Body.String())
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/-/readyz", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("readyz status = %d, want 200 (provider-absent behavior unchanged)", rec.Code)
	}
}
