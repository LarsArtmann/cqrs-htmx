package integration_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/larsartmann/cqrs-htmx/adminui/v4"
	"github.com/larsartmann/cqrs-htmx/dashboardui/v4"
	"github.com/larsartmann/cqrs-htmx/loginpage/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/watermill/v4"
	"github.com/stretchr/testify/require"
)

// setupFullstackUI wires adminui + dashboardui + loginpage against a real
// *usermgmt.Service and returns the composed http.Handler. Each UI module is
// constructed individually (not via setup.Bundle) to verify direct cross-module
// composition.
func setupFullstackUI(t *testing.T) http.Handler {
	t.Helper()

	store := memorystorage.NewMemoryStore()
	bus := watermill.NewEventBus()

	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
		EventStore: store,
		EventBus:   bus,
		AuditLog:   usermgmt.NewAuditLog(),
	})
	require.NoError(t, err)
	t.Cleanup(svc.Stop)

	admin, err := adminui.New(adminui.Config{Service: svc, Title: "Test Admin"})
	require.NoError(t, err)

	dash, err := dashboardui.New(dashboardui.Config{
		Title:          "Test Dashboard",
		EventSource:    store,
		Journal:        store,
		EventBus:       bus,
		ProjectionHost: svc.ProjectionHost(),
	})
	require.NoError(t, err)

	login, err := loginpage.New(loginpage.Config{Service: svc, Title: "Test Login"})
	require.NoError(t, err)

	mux := http.NewServeMux()
	usermgmt.NewAuthHandler(svc).RegisterRoutes(mux)
	login.Mount(mux, "/")

	sessionMW := usermgmt.NewSessionMiddleware(svc, "session_token")
	mux.Handle("/admin/", sessionMW(http.StripPrefix("/admin", admin.Handler())))
	mux.Handle("/dashboard/", http.StripPrefix("/dashboard", dash.Handler()))

	return cqrshtmx.RecommendedSecurityMiddleware()(mux)
}

func TestFullstackUI_LoginPageRenders(t *testing.T) {
	t.Parallel()
	handler := setupFullstackUI(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	require.Contains(t, w.Body.String(), "Test Login")
}

func TestFullstackUI_AdminRequiresSession(t *testing.T) {
	t.Parallel()
	handler := setupFullstackUI(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	handler.ServeHTTP(w, r)

	require.NotEqual(t, http.StatusOK, w.Code, "admin should require session")
}

func TestFullstackUI_DashboardRenders(t *testing.T) {
	t.Parallel()
	handler := setupFullstackUI(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/dashboard/", nil)
	handler.ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	require.Contains(t, w.Body.String(), "Dashboard")
}
