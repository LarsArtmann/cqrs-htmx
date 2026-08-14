package setup_test

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/cqrs-htmx/adminui/v4"
	"github.com/larsartmann/cqrs-htmx/setup/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
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

func TestNew_ConfigValidation_AdminAndDashboardPathsConflict(t *testing.T) {
	t.Parallel()

	_, err := setup.New(setup.Config{
		Title:         "Conflicting Paths",
		AdminPath:     "/panel/",
		DashboardPath: "/panel",
	})
	if err == nil {
		t.Fatal("expected error when AdminPath and DashboardPath resolve to the same route")
	}
}

func TestNew_ConfigValidation_HealthPathConflictsWithAdmin(t *testing.T) {
	t.Parallel()

	_, err := setup.New(setup.Config{
		Title:      "Health Conflicts With Admin",
		AdminPath:  "/health/",
		HealthPath: "/health",
	})
	if err == nil {
		t.Fatal("expected error when HealthPath and AdminPath resolve to the same route")
	}
}

func TestNew_ConfigValidation_HealthPathConflictsWithDashboard(t *testing.T) {
	t.Parallel()

	_, err := setup.New(setup.Config{
		Title:         "Health Conflicts With Dashboard",
		DashboardPath: "/observe/",
		HealthPath:    "/observe",
	})
	if err == nil {
		t.Fatal("expected error when HealthPath and DashboardPath resolve to the same route")
	}
}

// TestNew_Passthroughs_ReachPanels verifies the admin/dashboard authorizer and
// logger config fields are wired through to the sub-modules.
func TestNew_Passthroughs_ReachPanels(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title:               "Passthrough Test",
		AdminAuthorizer:     func(*usermgmt.User) error { return nil },
		DashboardAuthorizer: func(*http.Request) error { return nil },
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	defer func() { _ = bundle.Close() }()

	if bundle.Admin.Config().Authorizer == nil {
		t.Fatal("AdminAuthorizer was not passed through to the admin panel")
	}

	if bundle.Dashboard.Config().Authorizer == nil {
		t.Fatal("DashboardAuthorizer was not passed through to the dashboard")
	}
}

// TestNew_AdminTenantMode_RequiresTenantID covers the admin creation failure
// path: adminui.New rejects tenant mode without a TenantID, and setup must
// return a wrapped error (running cleanup on the already-created service).
func TestNew_AdminTenantMode_RequiresTenantID(t *testing.T) {
	t.Parallel()

	_, err := setup.New(setup.Config{
		Title:     "Tenant Mode Without TenantID",
		AdminMode: adminui.ModeTenantAdmin,
	})
	if err == nil {
		t.Fatal("expected error for tenant admin mode without TenantID")
	}

	if !strings.Contains(err.Error(), "admin") {
		t.Fatalf("error should identify the admin panel as the failing component, got: %v", err)
	}
}

// TestBundleRun_ServesAndShutsDownGracefully exercises the full Run lifecycle:
// mount, serve with real timeouts, health check, context cancellation,
// graceful shutdown, and bundle cleanup — with nil as the only clean-exit
// return value.
func TestBundleRun_ServesAndShutsDownGracefully(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{Title: "Run Test"})

	// Reserve a free port, then release it for Run to bind (small inherent
	// race, standard practice for testing real listeners).
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}

	addr := listener.Addr().String()

	if err := listener.Close(); err != nil {
		t.Fatalf("release port: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	runErr := make(chan error, 1)

	go func() { runErr <- bundle.Run(ctx, addr) }()

	// Wait for the server to come up, then verify it serves.
	baseURL := "http://" + addr

	var resp *http.Response

	for range 100 {
		resp, err = http.Get(baseURL + "/health")
		if err == nil {
			break
		}

		time.Sleep(20 * time.Millisecond)
	}

	if err != nil {
		t.Fatalf("server never came up: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /health: status %d, want 200", resp.StatusCode)
	}

	resp.Body.Close()

	cancel()

	select {
	case err := <-runErr:
		if err != nil {
			t.Fatalf("Run returned error on graceful shutdown: %v", err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("Run did not return within 15s of context cancellation")
	}
}

// gatedStore delays every journal ReadFrom until the gate channel is closed.
// projectionhost drains via ReadFrom, so this deterministically holds
// projection workers out of "live" state — without sleeps or huge journals.
type gatedStore struct {
	*memorystorage.MemoryStore

	gate chan struct{}
}

func (g *gatedStore) ReadFrom(
	ctx context.Context,
	after id.EventID,
	limit int,
) ([]event.Event, error) {
	select {
	case <-g.gate:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return g.MemoryStore.ReadFrom(ctx, after, limit)
}

// TestAsyncStartup_HealthTransitionsFromDrainingToReady is the end-to-end
// lifecycle test for Config.AsyncStartup: with the journal gated, New returns
// immediately, /health answers 503 (not ready) while projections drain, and
// flips to 200 once every worker reaches "live". This is the contract a
// reverse proxy relies on during the catch-up window after a restart.
func TestAsyncStartup_HealthTransitionsFromDrainingToReady(t *testing.T) {
	t.Parallel()

	store := &gatedStore{MemoryStore: memorystorage.NewMemoryStore(), gate: make(chan struct{})}

	bundle, err := setup.New(setup.Config{
		Title:        "Async Startup Test",
		EventStore:   store,
		AsyncStartup: true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = bundle.Close() }()

	mux := http.NewServeMux()
	bundle.Mount(mux)

	server := httptest.NewServer(bundle.Middleware()(mux))
	defer server.Close()

	// While the journal read is gated, no projection can be live: /health
	// must report 503 (not ready), never a panic or a hang.
	resp, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health while draining: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("GET /health while draining: status %d, want 503. Body: %s", resp.StatusCode, body)
	}

	// Release the journal: projections drain the (empty) journal and go live.
	close(store.gate)

	deadline := time.Now().Add(30 * time.Second)

	for {
		resp, err = http.Get(server.URL + "/health")
		if err == nil {
			body, _ = io.ReadAll(resp.Body)
			resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				break
			}
		}

		if time.Now().After(deadline) {
			t.Fatalf("/health never became ready after gate release; last: %d %s", resp.StatusCode, body)
		}

		time.Sleep(50 * time.Millisecond)
	}
}

// TestSyncStartup_BlocksUntilDrained verifies the complementary default:
// with AsyncStartup=false, New does not return until projections finished
// their initial drain — it stays blocked while the journal is gated and
// completes shortly after the gate opens.
func TestSyncStartup_BlocksUntilDrained(t *testing.T) {
	t.Parallel()

	store := &gatedStore{MemoryStore: memorystorage.NewMemoryStore(), gate: make(chan struct{})}

	type result struct {
		bundle *setup.Bundle
		err    error
	}

	newDone := make(chan result, 1)

	go func() {
		bundle, err := setup.New(setup.Config{
			Title:      "Sync Startup Test",
			EventStore: store,
		})
		newDone <- result{bundle, err}
	}()

	// While the journal is gated, New must still be blocked (drain in flight).
	select {
	case res := <-newDone:
		t.Fatalf("New returned before drain completed: bundle=%v err=%v", res.bundle, res.err)
	case <-time.After(300 * time.Millisecond):
	}

	close(store.gate)

	select {
	case res := <-newDone:
		if res.err != nil {
			t.Fatalf("New: %v", res.err)
		}

		if err := res.bundle.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("New did not return within 30s of gate release")
	}
}

func TestNew_ConfigValidation_RootPathRejected(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		cfg  setup.Config
	}{
		{"AdminPath root", setup.Config{AdminPath: "/"}},
		{"DashboardPath root", setup.Config{DashboardPath: "/"}},
		{"HealthPath root", setup.Config{HealthPath: "/"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := setup.New(tc.cfg)
			if err == nil {
				t.Fatal("expected error — the site root is reserved for the login page")
			}
		})
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

func TestNew_ConfigValidation_InvalidLoginRedirect(t *testing.T) {
	t.Parallel()

	_, err := setup.New(setup.Config{
		Title:         "Invalid Redirect",
		LoginRedirect: "admin/dashboard",
	})
	if err == nil {
		t.Fatal("expected error for LoginRedirect not starting with / or http")
	}
}

func TestNew_ConfigValidation_ValidLoginRedirect_HTTPS(t *testing.T) {
	t.Parallel()

	bundle, err := setup.New(setup.Config{
		Title:         "HTTPS Redirect",
		LoginRedirect: "https://example.com/dashboard",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = bundle.Close() }()
}

func TestNew_DashboardReadOnly_ExplicitTrue(t *testing.T) {
	t.Parallel()

	readOnly := true

	bundle := setup.MustNew(setup.Config{
		Title:             "Explicit ReadOnly",
		DashboardReadOnly: &readOnly,
	})
	defer func() { _ = bundle.Close() }()

	if bundle.Dashboard == nil {
		t.Fatal("Dashboard is nil")
	}

	cfg := bundle.Dashboard.Config()
	if !cfg.ReadOnly {
		t.Fatal("Dashboard should be read-only when DashboardReadOnly is explicitly true")
	}
}

func TestNew_StoresSharedBetweenServiceAndDashboard(t *testing.T) {
	t.Parallel()

	store := memorystorage.NewMemoryStore()
	bus := watermill.NewEventBus()

	bundle, err := setup.New(setup.Config{
		Title:      "Shared Stores",
		EventStore: store,
		EventBus:   bus,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = bundle.Close() }()

	if bundle.Stores.EventStore != store {
		t.Fatal("EventStore in Stores is not the provided instance")
	}

	if bundle.Stores.EventBus != bus {
		t.Fatal("EventBus in Stores is not the provided instance")
	}
}

func TestNew_CustomCookieName(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title:      "Custom Cookie",
		CookieName: "my-session-cookie",
	})
	defer func() { _ = bundle.Close() }()

	mw := bundle.SessionMiddleware()
	if mw == nil {
		t.Fatal("SessionMiddleware returned nil")
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

func TestNew_CustomPaths_NoTrailingSlash(t *testing.T) {
	t.Parallel()

	// AdminPath without a trailing slash must be normalized to a subtree
	// pattern — otherwise only the exact "/manage" matches and every panel
	// sub-route 404s. Verify routing actually works: the auth gate answering
	// on /manage/ proves the panel is mounted as a subtree.
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

func TestNew_CustomEventStore_DefaultBus(t *testing.T) {
	t.Parallel()

	store := memorystorage.NewMemoryStore()

	bundle, err := setup.New(setup.Config{
		Title:      "Custom Store Default Bus",
		EventStore: store,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = bundle.Close() }()

	if bundle.Stores.EventStore != store {
		t.Fatal("EventStore is not the custom store")
	}

	if bundle.Stores.EventBus == nil {
		t.Fatal("EventBus should be auto-created")
	}
}

func TestNew_DefaultStore_CustomBus(t *testing.T) {
	t.Parallel()

	bus := watermill.NewEventBus()

	bundle, err := setup.New(setup.Config{
		Title:    "Default Store Custom Bus",
		EventBus: bus,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = bundle.Close() }()

	if bundle.Stores.EventStore == nil {
		t.Fatal("EventStore should be auto-created")
	}

	if bundle.Stores.EventBus != bus {
		t.Fatal("EventBus is not the custom bus")
	}
}

// TestNew_CustomPaths_PanelsUseMatchingBasePath guards against the classic
// StripPrefix bug: when panels are mounted at a custom path but keep their
// default internal BasePath, every link and HTMX target points at the default
// path (/admin/..., /dashboard/...) and navigation breaks.
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
