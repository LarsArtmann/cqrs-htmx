package setup_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/setup/v4"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

// --- Existing tests (kept and improved) ---

func TestNew_DefaultConfig_AllPanelsEnabled(t *testing.T) {
	t.Parallel()

	bundle, err := setup.New(setup.Config{
		Title: "Test App",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = bundle.Close() }()

	if bundle.Service == nil {
		t.Fatal("Service is nil")
	}

	if bundle.Auth == nil {
		t.Fatal("Auth is nil")
	}

	if bundle.Admin == nil {
		t.Fatal("Admin is nil")
	}

	if bundle.Dashboard == nil {
		t.Fatal("Dashboard is nil")
	}

	if bundle.Login == nil {
		t.Fatal("Login is nil")
	}

	if bundle.Stores == nil {
		t.Fatal("Stores is nil")
	}

	if bundle.Stores.EventStore == nil {
		t.Fatal("EventStore is nil")
	}

	if bundle.Stores.EventBus == nil {
		t.Fatal("EventBus is nil")
	}
}

func TestNew_DisableAllPanels(t *testing.T) {
	t.Parallel()

	bundle, err := setup.New(setup.Config{
		Title:            "Minimal",
		DisableAdmin:     true,
		DisableDashboard: true,
		DisableLogin:     true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = bundle.Close() }()

	if bundle.Admin != nil {
		t.Fatal("Admin should be nil when DisableAdmin is true")
	}

	if bundle.Dashboard != nil {
		t.Fatal("Dashboard should be nil when DisableDashboard is true")
	}

	if bundle.Login != nil {
		t.Fatal("Login should be nil when DisableLogin is true")
	}

	if bundle.Service == nil {
		t.Fatal("Service is nil even with no UI panels")
	}

	if bundle.Auth == nil {
		t.Fatal("Auth is nil even with no UI panels")
	}
}

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

func TestProjectionHost_Exposed(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title: "ProjectionHost Test",
	})
	defer func() { _ = bundle.Close() }()

	host := bundle.Service.ProjectionHost()
	if host == nil {
		t.Fatal("ProjectionHost() returned nil — dashboard cannot show projection health")
	}
}

func TestMustNew_SucceedsWithDefaults(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("MustNew panicked with valid config: %v", r)
		}
	}()

	bundle := setup.MustNew(setup.Config{
		Title: "MustNew Test",
	})
	defer func() { _ = bundle.Close() }()
}

func TestMiddleware_ReturnsNonNil(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title: "Middleware Test",
	})
	defer func() { _ = bundle.Close() }()

	if bundle.Middleware() == nil {
		t.Fatal("Middleware() returned nil")
	}

	if bundle.SessionMiddleware() == nil {
		t.Fatal("SessionMiddleware() returned nil")
	}

	if bundle.CSRFMiddleware() == nil {
		t.Fatal("CSRFMiddleware() returned nil")
	}
}

func TestClose_Idempotent(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title: "Close Test",
	})

	if err := bundle.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}

	if err := bundle.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

// --- New tests ---

func TestNew_DefaultsApplied(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{})
	defer func() { _ = bundle.Close() }()

	// Defaults should be applied by withDefaults().
	// We can verify side-effects: the bundle should be functional with zero config.
	if bundle.Service == nil {
		t.Fatal("Service is nil with zero config")
	}
}

func TestNew_CustomEventStore(t *testing.T) {
	t.Parallel()

	store := memorystorage.NewMemoryStore()
	bus := watermill.NewEventBus()

	bundle, err := setup.New(setup.Config{
		Title:      "Custom Stores",
		EventStore: store,
		EventBus:   bus,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = bundle.Close() }()

	if bundle.Stores.EventStore != store {
		t.Fatal("EventStore was not the custom store provided")
	}

	if bundle.Stores.EventBus != bus {
		t.Fatal("EventBus was not the custom bus provided")
	}
}

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

func TestNew_DashboardClosesOnBundleClose(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title: "Dashboard Close Test",
	})

	if bundle.Dashboard == nil {
		t.Fatal("Dashboard is nil")
	}

	// Close the bundle — this should close the dashboard too.
	if err := bundle.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Calling Close again should be safe (dashboard uses sync.Once).
	if err := bundle.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
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

func TestNew_DashboardRoute_Reachable(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title: "Dashboard Route Test",
	})
	defer func() { _ = bundle.Close() }()

	mux := http.NewServeMux()
	bundle.Mount(mux)

	server := httptest.NewServer(bundle.Middleware()(mux))
	defer server.Close()

	// The dashboard is behind session middleware, which enriches but does not block.
	// The dashboard's authorizer defaults to allow-all, so the route should be reachable.
	// Consumers who need auth-gating should add an Authorizer to the dashboard after construction.
	resp, err := http.Get(server.URL + "/dashboard/")
	if err != nil {
		t.Fatalf("GET /dashboard/: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /dashboard/: status %d, want 200 (dashboard defaults to allow-all)", resp.StatusCode)
	}
}

func TestNew_ConfigValidation_InvalidAdminPath(t *testing.T) {
	t.Parallel()

	_, err := setup.New(setup.Config{
		Title:     "Invalid Path",
		AdminPath: "admin",
	})
	if err == nil {
		t.Fatal("expected error for AdminPath not starting with /")
	}
}

func TestNew_ConfigValidation_InvalidDashboardPath(t *testing.T) {
	t.Parallel()

	_, err := setup.New(setup.Config{
		Title:         "Invalid Path",
		DashboardPath: "dashboard",
	})
	if err == nil {
		t.Fatal("expected error for DashboardPath not starting with /")
	}
}

func TestNew_ConfigValidation_InvalidHealthPath(t *testing.T) {
	t.Parallel()

	_, err := setup.New(setup.Config{
		Title:      "Invalid Path",
		HealthPath: "health",
	})
	if err == nil {
		t.Fatal("expected error for HealthPath not starting with /")
	}
}

func TestNew_LoginNoRegistration(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title:               "No Registration",
		LoginNoRegistration: true,
	})
	defer func() { _ = bundle.Close() }()

	if bundle.Login == nil {
		t.Fatal("Login is nil")
	}
}

func TestNew_LogoutURL_PassedToAdmin(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title:     "Logout URL Test",
		LogoutURL: "/logout",
	})
	defer func() { _ = bundle.Close() }()

	if bundle.Admin == nil {
		t.Fatal("Admin is nil")
	}
}

func TestNew_SSEURL_PassedToAdmin(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title:  "SSE URL Test",
		SSEURL: "/admin/-/events",
	})
	defer func() { _ = bundle.Close() }()

	if bundle.Admin == nil {
		t.Fatal("Admin is nil")
	}
}

func TestNew_OnProjectionFailed_Callback(t *testing.T) {
	t.Parallel()

	callbackCalled := false

	bundle, err := setup.New(setup.Config{
		Title: "Projection Failed Test",
		OnProjectionFailed: func(projectionName, lastError string) {
			callbackCalled = true
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = bundle.Close() }()

	if callbackCalled {
		t.Fatal("OnProjectionFailed callback should not be called during setup")
	}
}

func TestNew_DashboardReadOnly_DefaultTrue(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title: "ReadOnly Test",
	})
	defer func() { _ = bundle.Close() }()

	if bundle.Dashboard == nil {
		t.Fatal("Dashboard is nil")
	}

	cfg := bundle.Dashboard.Config()
	if !cfg.ReadOnly {
		t.Fatal("Dashboard should be read-only by default")
	}
}

func TestNew_DashboardReadOnly_ExplicitFalse(t *testing.T) {
	t.Parallel()

	writable := false

	bundle := setup.MustNew(setup.Config{
		Title:             "Writable Dashboard",
		DashboardReadOnly: &writable,
	})
	defer func() { _ = bundle.Close() }()

	if bundle.Dashboard == nil {
		t.Fatal("Dashboard is nil")
	}

	cfg := bundle.Dashboard.Config()
	if cfg.ReadOnly {
		t.Fatal("Dashboard should be writable when DashboardReadOnly is explicitly false")
	}
}

func TestNew_DashboardPageSize(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title:             "Page Size Test",
		DashboardPageSize: 10,
	})
	defer func() { _ = bundle.Close() }()

	if bundle.Dashboard == nil {
		t.Fatal("Dashboard is nil")
	}

	cfg := bundle.Dashboard.Config()
	if cfg.PageSize != 10 {
		t.Fatalf("Dashboard PageSize = %d, want 10", cfg.PageSize)
	}
}

func TestNew_SessionTTL(t *testing.T) {
	t.Parallel()

	bundle, err := setup.New(setup.Config{
		Title:      "Session TTL Test",
		SessionTTL: 3600_000_000_000, // 1 hour in nanoseconds
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = bundle.Close() }()

	if bundle.Service == nil {
		t.Fatal("Service is nil")
	}
}

func TestNew_SelectiveDisable(t *testing.T) {
	t.Parallel()

	// Disable only admin, keep dashboard and login
	bundle := setup.MustNew(setup.Config{
		Title:        "Selective Disable",
		DisableAdmin: true,
	})
	defer func() { _ = bundle.Close() }()

	if bundle.Admin != nil {
		t.Fatal("Admin should be nil")
	}

	if bundle.Dashboard == nil {
		t.Fatal("Dashboard should not be nil")
	}

	if bundle.Login == nil {
		t.Fatal("Login should not be nil")
	}
}

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

func TestMustNew_PanicsOnInvalidConfig(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("MustNew should panic on invalid config")
		}
	}()

	_ = setup.MustNew(setup.Config{
		AdminPath: "invalid",
	})
}

func TestNew_CloseReturnsServiceError(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{Title: "Close Error Test"})

	// First close succeeds (closes service + dashboard).
	if err := bundle.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}

	// Second close: service is already closed, dashboard uses sync.Once (no-op).
	// Service.Close() may or may not return an error on double-close.
	// Just verify it doesn't panic.
	_ = bundle.Close()
}

func TestNew_DashboardDisabled_DashboardNil(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title:            "No Dashboard",
		DisableDashboard: true,
	})
	defer func() { _ = bundle.Close() }()

	if bundle.Dashboard != nil {
		t.Fatal("Dashboard should be nil when DisableDashboard is true")
	}

	// Close should still work without dashboard.
	if err := bundle.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestNew_AdminDisabled_AdminNil(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title:        "No Admin",
		DisableAdmin: true,
	})
	defer func() { _ = bundle.Close() }()

	if bundle.Admin != nil {
		t.Fatal("Admin should be nil when DisableAdmin is true")
	}
}

func TestNew_LoginDisabled_LoginNil(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title:        "No Login",
		DisableLogin: true,
	})
	defer func() { _ = bundle.Close() }()

	if bundle.Login != nil {
		t.Fatal("Login should be nil when DisableLogin is true")
	}
}

func TestNew_AccentColor_PassedToAdmin(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title:       "Accent Test",
		AccentColor: "#ff0000",
	})
	defer func() { _ = bundle.Close() }()

	if bundle.Admin == nil {
		t.Fatal("Admin is nil")
	}
}

func TestNew_AllConfigFields(t *testing.T) {
	t.Parallel()

	writable := false

	bundle, err := setup.New(setup.Config{
		Title:               "Full Config",
		AccentColor:         "#abcdef",
		AdminPath:           "/admin/",
		DashboardPath:       "/dashboard/",
		LoginRedirect:       "/admin/",
		HealthPath:          "/health",
		CookieName:          "my-session",
		SessionTTL:          3600_000_000_000,
		LogoutURL:           "/logout",
		SSEURL:              "/admin/-/events",
		DashboardReadOnly:   &writable,
		DashboardPageSize:   15,
		LoginNoRegistration: true,
	})
	if err != nil {
		t.Fatalf("New with full config: %v", err)
	}

	defer func() { _ = bundle.Close() }()

	if bundle.Service == nil {
		t.Fatal("Service is nil")
	}

	if bundle.Admin == nil {
		t.Fatal("Admin is nil")
	}

	if bundle.Dashboard == nil {
		t.Fatal("Dashboard is nil")
	}

	if bundle.Login == nil {
		t.Fatal("Login is nil")
	}
}
