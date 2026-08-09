package setup_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/larsartmann/cqrs-htmx/setup/v4"
)

func TestNew_DefaultConfig_AllPanelsEnabled(t *testing.T) {
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
	defer bundle.Close()

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

	bundle := setup.MustNew(setup.Config{ //nolint:exhaustruct // testing defaults
		Title: "Mount Test",
	})
	defer bundle.Close()

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

	bundle := setup.MustNew(setup.Config{ //nolint:exhaustruct // testing defaults
		Title: "Admin Test",
	})
	defer bundle.Close()

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

	bundle := setup.MustNew(setup.Config{ //nolint:exhaustruct // testing defaults
		Title: "ProjectionHost Test",
	})
	defer bundle.Close()

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

	bundle := setup.MustNew(setup.Config{ //nolint:exhaustruct // testing defaults
		Title: "MustNew Test",
	})
	defer bundle.Close()
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
	if err := bundle.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}
