package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	gohealth "github.com/larsartmann/go-health"
	"github.com/samber/do/v2"
)

// newTestContainer creates a production container then overrides the TOTP
// provider with a stub. This demonstrates the canonical test-container pattern:
// production wiring + targeted overrides. The override targets the service's
// INTERFACE name (do.NameOf[identitymodel.TOTPProvider]) — the name do.As
// registered — so every InvokeAs consumer resolves the stub.
//
// do.Override* is safe in _test.go files (DO-3 rule explicitly allows it).
func newTestContainer(t *testing.T) (*Container, func()) {
	t.Helper()

	container, cleanup, err := NewContainer(AppConfig{
		TOTPIssuer: "test",
	})
	if err != nil {
		t.Fatalf("build DI container: %v", err)
	}

	// Override the TOTP provider with a no-op stub for tests.
	// This avoids real TOTP secret generation during unit tests.
	do.OverrideNamed(container.injector, do.NameOf[identitymodel.TOTPProvider](),
		func(_ do.Injector) (identitymodel.TOTPProvider, error) {
			return stubTOTP{}, nil
		})

	return container, cleanup
}

// stubTOTP satisfies identitymodel.TOTPProvider without doing any real TOTP work.
type stubTOTP struct{}

func (stubTOTP) GenerateSecret(_ string) ([]byte, string, string, error) {
	return []byte("test"), "test-base32", "otpauth://test", nil
}
func (stubTOTP) ValidateCode(_ []byte, _ string) bool { return true }

// TestContainerResolvesService verifies that the container can resolve the
// usermgmt.Service with the overridden TOTP provider.
func TestContainerResolvesService(t *testing.T) {
	container, cleanup := newTestContainer(t)
	defer cleanup()

	svc, err := container.Service()
	if err != nil {
		t.Fatalf("resolve Service: %v", err)
	}
	if svc == nil {
		t.Fatal("Service is nil")
	}
}

// TestSmoke_AuditViewerServes verifies the live audit-viewer http.Handler
// (the auditlog bridge's SSE-backed dashboard) serves its page — the demo's
// real-time surface.
func TestSmoke_AuditViewerServes(t *testing.T) {
	container, cleanup := newTestContainer(t)
	defer cleanup()

	if container.AuditViewer == nil {
		t.Fatal("AuditViewer is nil — the auditlog bridge did not wire the live viewer")
	}

	rec := httptest.NewRecorder()
	container.AuditViewer.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/audit/", nil))
	if rec.Code == http.StatusTemporaryRedirect || rec.Code == http.StatusMovedPermanently {
		t.Fatalf(
			"audit viewer GET /audit/: unexpected redirect %d (Location: %s)",
			rec.Code,
			rec.Header().Get("Location"),
		)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("audit viewer GET /audit/: status %d, want 200", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "<") {
		t.Fatalf("audit viewer GET /audit: body does not look like HTML (%q)", body[:min(80, len(body))])
	}
}

// TestContainerResolvesApp verifies that the cqrshtmx.App resolves correctly.
func TestContainerResolvesApp(t *testing.T) {
	container, cleanup := newTestContainer(t)
	defer cleanup()

	app, err := container.App()
	if err != nil {
		t.Fatalf("resolve App: %v", err)
	}
	if app == nil {
		t.Fatal("App is nil")
	}
}

// TestContainerServiceIsSingleton verifies that the container returns the
// same *usermgmt.Service instance on every invocation (lazy singleton).
func TestContainerServiceIsSingleton(t *testing.T) {
	container, cleanup := newTestContainer(t)
	defer cleanup()

	svc1, err := container.Service()
	if err != nil {
		t.Fatalf("resolve Service: %v", err)
	}
	svc2, err := container.Service()
	if err != nil {
		t.Fatalf("resolve Service (2nd): %v", err)
	}
	if svc1 != svc2 {
		t.Fatal("Service is not a singleton — got different instances")
	}
}

// TestContainerResolveTOTPViaInvokeAs verifies the interface-consumption
// wiring: the do.As alias resolves, and the override actually replaced the
// production provider underneath it.
func TestContainerResolveTOTPViaInvokeAs(t *testing.T) {
	container, cleanup := newTestContainer(t)
	defer cleanup()

	totp, err := do.InvokeAs[identitymodel.TOTPProvider](container.injector)
	if err != nil {
		t.Fatalf("resolve TOTP provider via InvokeAs: %v", err)
	}

	if _, ok := totp.(stubTOTP); !ok {
		t.Fatalf("expected stubTOTP, got %T", totp)
	}
}

// TestHealthProbe_StartedByNewContainer proves the HW-4 posture end to end:
// NewContainer starts the probe (refresh loop + first evaluation), the
// critical projection names are real checks (Start validates them — a typo
// fails NewContainer), and the background cache holds a pass verdict with
// every projection check present — never the pre-fix zero-value empty pass.
// (The startup LATCH itself is set by the /startupz endpoint on its first
// all-criticals-pass evaluation — asserted in TestHealthRoutes_ServeRealVerdicts.)
func TestHealthProbe_StartedByNewContainer(t *testing.T) {
	container, cleanup := newTestContainer(t)
	defer cleanup()

	probe, err := container.Probe()
	if err != nil {
		t.Fatalf("resolve health probe: %v", err)
	}

	resp := probe.CachedResponse()
	if resp.Status != gohealth.StatusPass {
		t.Fatalf("cached response status = %q, want pass (checks: %v)", resp.Status, resp.Checks)
	}
	if len(resp.Checks) == 0 {
		t.Fatal("cached response has ZERO checks — the pre-fix false-green: probe constructed but never started")
	}
	for _, name := range []string{"user-read-model", "casbin-projection"} {
		if _, ok := resp.Checks[name]; !ok {
			t.Fatalf("critical projection %q missing from probe checks: %v", name, resp.Checks)
		}
	}
}

// TestHealthRoutes_ServeRealVerdicts wires the mux exactly like main.go and
// proves the three Kubernetes probes answer from the STARTED probe (the
// static {"status":"ok"} handler is gone — a dead feed must never answer 200).
func TestHealthRoutes_ServeRealVerdicts(t *testing.T) {
	container, cleanup := newTestContainer(t)
	defer cleanup()

	probe, err := container.Probe()
	if err != nil {
		t.Fatalf("resolve health probe: %v", err)
	}

	mux := http.NewServeMux()
	probe.RegisterRoutes(mux, gohealth.DefaultRoutes())

	srv := httptest.NewServer(mux)
	defer srv.Close()

	for _, tc := range []struct{ path, wantStatus string }{
		{"/healthz", "pass"},
		{"/readyz", "pass"},
		{"/startupz", "pass"},
	} {
		resp, err := http.Get(srv.URL + tc.path)
		if err != nil {
			t.Fatalf("GET %s: %v", tc.path, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET %s: status = %d, want 200 (body: %s)", tc.path, resp.StatusCode, body)
		}
		if !strings.Contains(string(body), `"status":"`+tc.wantStatus+`"`) {
			t.Fatalf("GET %s: body does not report status %q: %s", tc.path, tc.wantStatus, body)
		}
	}

	// The /startupz request above evaluated every critical check and latched
	// the startup probe — StartupComplete flips exactly once, from the
	// endpoint's own evaluation (not from Start).
	if !probe.StartupComplete() {
		t.Fatal("startup latch not set after a passing /startupz request")
	}
}

// TestHealthDashboard_ShowsProjectionChecks pins the dashboard's data source
// to the STARTED probe's cache: the rendered page must name real projection
// checks, never an empty-but-green page.
func TestHealthDashboard_ShowsProjectionChecks(t *testing.T) {
	container, cleanup := newTestContainer(t)
	defer cleanup()

	dashboard, err := container.HealthDashboard()
	if err != nil {
		t.Fatalf("resolve health dashboard: %v", err)
	}

	rec := httptest.NewRecorder()
	dashboard.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health-ui", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /health-ui: status = %d", rec.Code)
	}

	body := rec.Body.String()
	for _, name := range []string{"user-read-model", "casbin-projection"} {
		if !strings.Contains(body, name) {
			t.Fatalf("GET /health-ui: projection check %q missing from rendered page (the empty-green regression)", name)
		}
	}
}

// TestAuditViewerMount_NoServeMuxConflict regression-pins the 2026-10-09
// boot panic: a method-less "/audit/" pattern conflicts with "GET /" under
// Go 1.22+ ServeMux rules, so the demo MUST mount the viewer GET-scoped.
// No StripPrefix: the live server serves its own configured prefix
// (Prefix: "/audit" routes /audit/, /audit/api/..., internally).
func TestAuditViewerMount_NoServeMuxConflict(t *testing.T) {
	container, cleanup := newTestContainer(t)
	defer cleanup()

	mux := http.NewServeMux()
	mux.HandleFunc("/", indexHandler)
	mux.Handle("GET /audit/", container.AuditViewer)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/audit/")
	if err != nil {
		t.Fatalf("GET /audit/: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /audit/: status = %d, want 200", resp.StatusCode)
	}
}

// TestContainerCleanupCallsShutdown verifies that the cleanup function
// properly shuts down the container without panicking.
func TestContainerCleanupCallsShutdown(t *testing.T) {
	container, cleanup, err := NewContainer(AppConfig{
		TOTPIssuer: "shutdown-test",
	})
	if err != nil {
		t.Fatalf("build DI container: %v", err)
	}

	// Force Service creation so the lifecycle wrapper is tracked.
	if _, err := container.Service(); err != nil {
		t.Fatalf("resolve Service: %v", err)
	}

	// cleanup should call injector.Shutdown() which calls
	// serviceLifecycle.Shutdown() which calls svc.Close().
	// If Close() panics or errors, the test fails.
	cleanup()
}

// TestHelloCommandEndToEnd dispatches the Hello command over a real mux
// route wired exactly like main.go, proving the DI-resolved dispatcher
// serves HTTP end-to-end (not just container resolution).
func TestHelloCommandEndToEnd(t *testing.T) {
	container, cleanup := newTestContainer(t)
	defer cleanup()

	app, err := container.App()
	if err != nil {
		t.Fatalf("resolve App: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("POST /command/hello", app.Command(
		"Hello",
		cqrshtmx.DecodeJSON(func(req helloRequest) (command.Command, error) {
			core, err := command.New("Hello", id.NewStreamID())
			if err != nil {
				return nil, err
			}
			return &helloCmd{BasicCommand: core, Name: req.Name}, nil
		}),
	))

	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/command/hello", "application/json", strings.NewReader(`{"name":"world"}`))
	if err != nil {
		t.Fatalf("POST /command/hello: %v", err)
	}
	defer resp.Body.Close()
	// Hello is a void command: the dispatch pipeline answers 204 No Content.
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("valid Hello dispatch: status = %d, want 204", resp.StatusCode)
	}

	rejected, err := http.Post(srv.URL+"/command/hello", "application/json", strings.NewReader(`{"name":""}`))
	if err != nil {
		t.Fatalf("POST empty name: %v", err)
	}
	defer rejected.Body.Close()
	if rejected.StatusCode == http.StatusOK {
		t.Fatal("empty-name Hello dispatch: status = 200, want an error status (handler rejects empty names)")
	}
}
