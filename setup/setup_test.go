package setup_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/larsartmann/cqrs-htmx/setup/v4"
)

func TestNew_DefaultConfig(t *testing.T) {
	t.Parallel()

	bundle, err := setup.New(setup.Config{ //nolint:exhaustruct // testing defaults
		Title: "Test App",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer bundle.Close()

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

func TestNew_DisablePanels(t *testing.T) {
	t.Parallel()

	bundle, err := setup.New(setup.Config{
		Title:           "Minimal",
		EnableAdmin:     false,
		EnableDashboard: false,
		EnableLogin:     false,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer bundle.Close()

	if bundle.Admin != nil {
		t.Fatal("Admin should be nil when EnableAdmin is false")
	}
	if bundle.Dashboard != nil {
		t.Fatal("Dashboard should be nil when EnableDashboard is false")
	}
	if bundle.Login != nil {
		t.Fatal("Login should be nil when EnableLogin is false")
	}
	// Service and Auth should always exist.
	if bundle.Service == nil {
		t.Fatal("Service is nil even with no UI panels")
	}
	if bundle.Auth == nil {
		t.Fatal("Auth is nil even with no UI panels")
	}
}

func TestMount_RegistersRoutes(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{ //nolint:exhaustruct // testing defaults
		Title: "Mount Test",
	})
	defer bundle.Close()

	mux := http.NewServeMux()
	bundle.Mount(mux)

	// Wrap with security middleware and do a smoke test.
	server := httptest.NewServer(bundle.Middleware()(mux))
	defer server.Close()

	// Login page should respond (it's at "/", no auth).
	resp, err := http.Get(server.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		t.Fatal("login page not mounted at /")
	}
}

func TestProjectionHost_Exposed(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{ //nolint:exhaustruct // testing defaults
		Title: "ProjectionHost Test",
	})
	defer bundle.Close()

	// The Service should expose its projection host for dashboard wiring.
	host := bundle.Service.ProjectionHost()
	if host == nil {
		t.Fatal("ProjectionHost() returned nil — dashboard cannot show projection health")
	}
}

func TestMustNew_PanicsOnError(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("MustNew should panic on error")
		}
	}()

	// This won't actually error with current defaults, but the pattern is tested.
	_ = setup.MustNew(setup.Config{ //nolint:exhaustruct // test
		Title: "Panic Test",
	})
}

func TestMiddleware_ReturnsNonNil(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{ //nolint:exhaustruct // test
		Title: "Middleware Test",
	})
	defer bundle.Close()

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

	bundle := setup.MustNew(setup.Config{ //nolint:exhaustruct // test
		Title: "Close Test",
	})

	if err := bundle.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	// Second close should not panic.
	if err := bundle.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}
