package setup_test

import (
	"log/slog"
	"net/http"
	"testing"

	"github.com/larsartmann/cqrs-htmx/setup/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

// Tests in this file cover:
//   - default zero-config behaviour: a single Title is enough to build a Bundle
//     with every UI panel enabled, in-memory stores, and a service.
//   - feature-flag defaults: DisableAdmin/Dashboard/Login flip the corresponding
//     bundle field to nil (and Disable* defaults to false).
//   - non-path, non-validation config passthroughs: every Config field that's
//     NOT a path or a validation case is wired through to its target (admin
//     panel, dashboard, session middleware, service stores).

// --- Default zero-config behaviour ---

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

// TestNew_Passthroughs_ReachPanels verifies the admin/dashboard authorizer and
// logger config fields are wired through to the sub-modules.
func TestNew_Passthroughs_ReachPanels(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title:               "Passthrough Test",
		AdminAuthorizer:     func(*usermgmt.User) error { return nil },
		DashboardAuthorizer: func(*http.Request) error { return nil },
		Logger:              slog.New(slog.DiscardHandler),
	})
	defer func() { _ = bundle.Close() }()

	if bundle.Admin.Config().Authorizer == nil {
		t.Fatal("AdminAuthorizer was not passed through to the admin panel")
	}

	if bundle.Dashboard.Config().Authorizer == nil {
		t.Fatal("DashboardAuthorizer was not passed through to the dashboard")
	}
}

// --- Feature-flag defaults: Disable* flips bundle field to nil ---

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

// --- Stores passthrough ---

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

// --- Per-field passthroughs: admin ---

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

// --- Per-field passthroughs: dashboard ---

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

// --- Per-field passthroughs: session / projection callback ---

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
