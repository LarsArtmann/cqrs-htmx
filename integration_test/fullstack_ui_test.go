package integration_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/larsartmann/cqrs-htmx/adminui/v4"
	"github.com/larsartmann/cqrs-htmx/dashboardui/v4"
	"github.com/larsartmann/cqrs-htmx/loginpage/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/watermill/v4"
	"github.com/stretchr/testify/require"
)

// TestFullstackUI_Rendering mounts adminui + dashboardui + loginpage against a
// real *usermgmt.Service and verifies each UI renders correctly via HTTP.
// This is the only integration test that mounts any UI module.
func TestFullstackUI_Rendering(t *testing.T) {
	t.Parallel()

	// Shared event infrastructure.
	store := memorystorage.NewMemoryStore()
	bus := watermill.NewEventBus()

	// Real user management service.
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
		EventStore: store,
		EventBus:   bus,
		AuditLog:   usermgmt.NewAuditLog(),
	})
	require.NoError(t, err)
	t.Cleanup(svc.Stop)

	// Admin panel.
	admin, err := adminui.New(adminui.Config{
		Service: svc,
		Title:   "Test Admin",
	})
	require.NoError(t, err)

	// CQRS observability dashboard — wired from the same stores.
	dashCfg := dashboardui.Config{
		Title:          "Test Dashboard",
		EventSource:    store,
		EventBus:       bus,
		ProjectionHost: svc.ProjectionHost(),
	}
	if journal, ok := store.(event.Journal); ok {
		dashCfg.Journal = journal
	}
	dash, err := dashboardui.New(dashCfg)
	require.NoError(t, err)

	// Login page.
	login, err := loginpage.New(loginpage.Config{
		Service: svc,
		Title:   "Test Login",
	})
	require.NoError(t, err)

	// Mount on shared mux following the documented middleware ordering:
	// Security (outer) > Session > CSRF (mutations) > handler.
	mux := http.NewServeMux()

	usermgmt.NewAuthHandler(svc).RegisterRoutes(mux)

	login.Mount(mux, "/")

	sessionMW := usermgmt.NewSessionMiddleware(svc, "session_token")
	mux.Handle("/admin/", sessionMW(http.StripPrefix("/admin", admin.Handler())))

	mux.Handle("/dashboard/", http.StripPrefix("/dashboard", dash.Handler()))

	server := httptest.NewServer(cqrshtmx.RecommendedSecurityMiddleware()(mux))
	defer server.Close()

	// Login page renders with 200.
	resp, err := http.Get(server.URL + "/")
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "login page status: %s", body)
	require.Contains(t, string(body), "Test Login", "login page should contain title")

	// Admin panel blocks unauthenticated access.
	resp, err = http.Get(server.URL + "/admin/")
	require.NoError(t, err)
	resp.Body.Close()
	require.NotEqual(t, http.StatusOK, resp.StatusCode, "admin should require session")

	// Dashboard renders with 200.
	resp, err = http.Get(server.URL + "/dashboard/")
	require.NoError(t, err)
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "dashboard status: %s", body)
	require.Contains(t, string(body), "Dashboard", "dashboard should contain title")
}
