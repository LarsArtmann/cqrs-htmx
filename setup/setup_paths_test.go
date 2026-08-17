package setup_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/setup/v4"
)

// Tests in this file cover path resolution, mounting, and routing:
//
//   - default paths (/, /admin/, /dashboard/, /health) reach the expected
//     panels/gates.
//   - custom paths are mounted at the configured URL and BasePath is wired
//     through to the panels so their internal links and HTMX targets resolve.
//   - auth gates: admin and dashboard are 401 without a session, login is
//     public, health is public.
//   - normalisation: paths without a trailing slash become subtree patterns
//     (/manage and /manage/ both work) so panel sub-routes don't 404.

// --- Default paths ---

func TestMount_LoginPageReachable(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title: "Mount Test",
	})
	defer func() { _ = bundle.Close() }()

	mux := http.NewServeMux()
	bundle.Mount(mux)

	server := httptest.NewServer(bundle.Middleware()(mux))
	defer server.Close()

	resp, err := http.Get(server.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /: status %d, want 200. Body: %s", resp.StatusCode, body)
	}
}

func TestMount_AdminRedirectsWithoutSession(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title: "Admin Test",
	})
	defer func() { _ = bundle.Close() }()

	mux := http.NewServeMux()
	bundle.Mount(mux)

	server := httptest.NewServer(bundle.Middleware()(mux))
	defer server.Close()

	resp, err := http.Get(server.URL + "/admin/")
	if err != nil {
		t.Fatalf("GET /admin/: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Fatal("admin panel accessible without session — auth gate not working")
	}
}

func TestNew_DashboardRoute_RequiresSession(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title: "Dashboard Route Test",
	})
	defer func() { _ = bundle.Close() }()

	mux := http.NewServeMux()
	bundle.Mount(mux)

	server := httptest.NewServer(bundle.Middleware()(mux))
	defer server.Close()

	// The dashboard renders event payloads and stream IDs — it must never be
	// reachable without an authenticated session. The session middleware only
	// enriches the context, so Mount applies an explicit 401 gate.
	resp, err := http.Get(server.URL + "/dashboard/")
	if err != nil {
		t.Fatalf("GET /dashboard/: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("GET /dashboard/: status %d, want 401 (dashboard must be session-gated)", resp.StatusCode)
	}
}

func TestNew_HealthEndpoint_DefaultPath(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title: "Default Health Path",
	})
	defer func() { _ = bundle.Close() }()

	mux := http.NewServeMux()
	bundle.Mount(mux)

	server := httptest.NewServer(bundle.Middleware()(mux))
	defer server.Close()

	// Default health path should be /health
	resp, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /health: status %d, want 200", resp.StatusCode)
	}
}

func TestNew_HealthEndpoint_Reachable(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title: "Health Test",
	})
	defer func() { _ = bundle.Close() }()

	mux := http.NewServeMux()
	bundle.Mount(mux)

	server := httptest.NewServer(bundle.Middleware()(mux))
	defer server.Close()

	resp, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /health: status %d, want 200", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"status":"ok"`) {
		t.Fatalf("GET /health: body does not contain status ok: %s", body)
	}
}

func TestMount_HealthEndpoint_ContentJSON(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{Title: "Content-Type Test"})
	defer func() { _ = bundle.Close() }()

	mux := http.NewServeMux()
	bundle.Mount(mux)

	server := httptest.NewServer(bundle.Middleware()(mux))
	defer server.Close()

	resp, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("GET /health: Content-Type = %q, want application/json", ct)
	}
}

// --- Custom paths ---

func TestNew_CustomPaths(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title:         "Custom Paths",
		AdminPath:     "/manage/",
		DashboardPath: "/observe/",
		HealthPath:    "/healthz",
	})
	defer func() { _ = bundle.Close() }()

	mux := http.NewServeMux()
	bundle.Mount(mux)

	server := httptest.NewServer(bundle.Middleware()(mux))
	defer server.Close()

	// Admin should be at /manage/ not /admin/ — verify it's auth-gated (non-200).
	resp, err := http.Get(server.URL + "/manage/")
	if err != nil {
		t.Fatalf("GET /manage/: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Fatal("custom admin path accessible without session — auth gate not working")
	}

	// Health should be at /healthz, not /health.
	respHealth, err := http.Get(server.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer respHealth.Body.Close()

	if respHealth.StatusCode != http.StatusOK {
		t.Fatalf("GET /healthz: status %d, want 200", respHealth.StatusCode)
	}
}

func TestNew_HealthEndpoint_CustomPath(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title:      "Health Custom",
		HealthPath: "/healthz",
	})
	defer func() { _ = bundle.Close() }()

	mux := http.NewServeMux()
	bundle.Mount(mux)

	server := httptest.NewServer(bundle.Middleware()(mux))
	defer server.Close()

	resp, err := http.Get(server.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /healthz: status %d, want 200", resp.StatusCode)
	}
}

// --- Path normalisation: trailing slash ---

// TestNew_CustomPaths_NoTrailingSlash guards against the StripPrefix bug: when
// panels are mounted at a custom path but keep their default internal BasePath,
// every link and HTMX target points at the default path (/admin/...,
// /dashboard/...) and navigation breaks.
func TestNew_CustomPaths_PanelsUseMatchingBasePath(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title:         "BasePath Test",
		AdminPath:     "/manage/",
		DashboardPath: "/observe/",
	})
	defer func() { _ = bundle.Close() }()

	if bundle.Admin == nil || bundle.Dashboard == nil {
		t.Fatal("Admin or Dashboard is nil")
	}

	if got := bundle.Admin.Config().BasePath; got != "/manage" {
		t.Fatalf("admin BasePath = %q, want %q (panel links must match the mount path)", got, "/manage")
	}

	if got := bundle.Dashboard.Config().BasePath; got != "/observe" {
		t.Fatalf("dashboard BasePath = %q, want %q (panel links must match the mount path)", got, "/observe")
	}
}

// TestNew_DefaultPaths_PanelsUseDefaultBasePath verifies the default paths
// still resolve to the panels' default BasePaths.
func TestNew_DefaultPaths_PanelsUseDefaultBasePath(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{Title: "Default BasePath"})
	defer func() { _ = bundle.Close() }()

	if got := bundle.Admin.Config().BasePath; got != "/admin" {
		t.Fatalf("admin BasePath = %q, want %q", got, "/admin")
	}

	if got := bundle.Dashboard.Config().BasePath; got != "/dashboard" {
		t.Fatalf("dashboard BasePath = %q, want %q", got, "/dashboard")
	}
}

// TestNew_CustomPaths_NoTrailingSlash verifies the consumer can pass
// AdminPath without a trailing slash and the bundle still mounts the panel
// as a subtree (so panel sub-routes don't 404). Without normalisation,
// "/manage" would only match "/manage" exactly and every panel link 404s.
func TestNew_CustomPaths_NoTrailingSlash(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title:     "No Trailing Slash",
		AdminPath: "/manage", // no trailing slash
	})
	defer func() { _ = bundle.Close() }()

	mux := http.NewServeMux()
	bundle.Mount(mux)

	server := httptest.NewServer(bundle.Middleware()(mux))
	defer server.Close()

	resp, err := http.Get(server.URL + "/manage/")
	if err != nil {
		t.Fatalf("GET /manage/: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("GET /manage/: status %d, want 401 (panel mounted and auth-gated)", resp.StatusCode)
	}

	if got := bundle.Admin.Config().BasePath; got != "/manage" {
		t.Fatalf("admin BasePath = %q, want %q", got, "/manage")
	}
}

// --- Mount + Handler wrapper ---

func TestMount_NoPanic_WhenAllDisabled(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title:            "All Disabled Mount",
		DisableAdmin:     true,
		DisableDashboard: true,
		DisableLogin:     true,
	})
	defer func() { _ = bundle.Close() }()

	mux := http.NewServeMux()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Mount panicked with all panels disabled: %v", r)
		}
	}()

	bundle.Mount(mux)
}

func TestNew_Handler_ReturnsNonNil(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title: "Handler Test",
	})
	defer func() { _ = bundle.Close() }()

	mux := http.NewServeMux()

	handler := bundle.Handler(mux)
	if handler == nil {
		t.Fatal("Handler() returned nil")
	}

	server := httptest.NewServer(handler)
	defer server.Close()

	// The server should be functional
	resp, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /health: status %d, want 200", resp.StatusCode)
	}
}
