package adminui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

func newManifestPanel(t *testing.T, config Config) *Handler {
	t.Helper()

	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{AuditLog: usermgmt.NewAuditLog()})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	config.Service = svc

	panel, err := New(config)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	return panel
}

// TestRoutes_ManifestGolden pins the super-admin manifest: every row, its
// posture, and its justification. A new route changes this golden — update it
// in the same commit and say why in the route table.
func TestRoutes_ManifestGolden(t *testing.T) {
	t.Parallel()

	panel := newManifestPanel(t, Config{})
	routes := panel.Routes()
	lines := make([]string, 0, len(routes))
	for _, route := range routes {
		if route.Why == "" {
			t.Errorf("%s: manifest row without a Why", route)
		}
		lines = append(lines, route.Method+" "+route.Pattern+" "+string(route.Posture)+" "+route.Panel)
	}

	got := strings.Join(lines, "\n")
	want := strings.Join([]string{
		"GET /-/admin-tw.css public Assets",
		"GET /-/admin.js public Assets",
		"GET /-/htmx.js public Assets",
		"GET /-/sync-worker.js public Assets",
		"GET /-/sync-client.js public Assets",
		"GET / role Dashboard",
		"GET /partials/stats role Dashboard",
		"GET /audit role Audit log",
		"GET /users role Users",
		"GET /users/new role Users",
		"POST /users role Users",
		"POST /users/{id}/display-name role Users",
		"POST /users/{id}/email role Users",
		"GET /users/{id} role Users",
		"POST /users/{id}/delete role Users",
		"POST /users/{id}/external/{provider}/unlink role Users",
		"POST /users/{id}/roles role Users",
		"POST /users/{id}/roles/revoke role Users",
		"POST /users/{id}/credentials/{credential}/delete role Users",
		"POST /users/{id}/resend-verification role Users",
		"GET /tenants role Tenants",
		"GET /tenants/new role Tenants",
		"POST /tenants role Tenants",
		"GET /tenants/{id} role Tenants",
		"POST /tenants/{id}/suspend role Tenants",
		"POST /tenants/{id}/reactivate role Tenants",
		"POST /tenants/{id}/delete role Tenants",
		"POST /tenants/{id}/members role Tenants",
		"POST /tenants/{id}/members/{actor}/delete role Tenants",
		"POST /tenants/{id}/members/{actor} role Tenants",
		"GET /* role Fallback",
		"* /* public Fallback",
	}, "\n")

	if got != want {
		t.Fatalf("super-admin manifest drifted:\n--- want ---\n%s\n--- got ---\n%s", want, got)
	}
}

// TestRoutes_TenantModeManifest pins the tenant-admin manifest: the
// super-admin-only surfaces are absent, the member surface is present.
func TestRoutes_TenantModeManifest(t *testing.T) {
	t.Parallel()

	panel := newManifestPanel(t, Config{
		Mode:     ModeTenantAdmin,
		TenantID: identitymodel.NewTenantID("t-1"),
	})

	for _, route := range panel.Routes() {
		if route.Panel == panelUsers || route.Panel == panelTenants {
			t.Errorf("%s: %s surface must be absent in tenant mode", route, route.Panel)
		}
	}

	joined := manifestText(panel)
	for _, want := range []string{"GET /members", "POST /members", "GET /audit", "GET /"} {
		if !strings.Contains(joined, want+"\n") && !strings.HasSuffix(joined, want) {
			t.Errorf("tenant manifest missing %q:\n%s", want, joined)
		}
	}
}

// TestRoutes_AuthorizerPosture proves the manifest honors
// Config.AuthorizerPosture and rejects lying values.
func TestRoutes_AuthorizerPosture(t *testing.T) {
	t.Parallel()

	t.Run("authenticated override", func(t *testing.T) {
		t.Parallel()

		panel := newManifestPanel(t, Config{
			Authorizer:        RequireAuthenticated(),
			AuthorizerPosture: cqrshtmx.PostureAuthenticated,
		})
		for _, route := range panel.Routes() {
			if route.Posture == cqrshtmx.PostureRoleGated {
				t.Fatalf("%s: declared authenticated but manifest still says role-gated", route)
			}
		}
	})

	t.Run("lying posture rejected", func(t *testing.T) {
		t.Parallel()

		for _, posture := range []cqrshtmx.Posture{cqrshtmx.PosturePublic, cqrshtmx.PostureUngated, "wild"} {
			_, err := New(Config{
				Service:           mustService(t),
				Authorizer:        RequireAuthenticated(),
				AuthorizerPosture: posture,
			})
			if err == nil {
				t.Fatalf("posture %q must be rejected at New", posture)
			}
		}
	})
}

// TestRoutes_FallbackSemantics pins the catch-all conversions the table
// refactor risked: the guarded 404 catch-all, the methodless 405, the
// exact-match dashboard root, and the public assets.
func TestRoutes_FallbackSemantics(t *testing.T) {
	t.Parallel()

	panel := newManifestPanel(t, Config{})
	handler := panel.Handler()

	probes := []struct {
		name string
		req  *http.Request
		want int
	}{
		{"dashboard root is exact", httptest.NewRequest(http.MethodGet, "/", nil), http.StatusUnauthorized},
		{"unknown page is guarded 404", httptest.NewRequest(http.MethodGet, "/nope", nil), http.StatusUnauthorized},
		{"unknown method is 405", httptest.NewRequest(http.MethodPut, "/users", nil), http.StatusMethodNotAllowed},
		{"asset is public", httptest.NewRequest(http.MethodGet, "/-/admin.js", nil), http.StatusOK},
	}

	for _, probe := range probes {
		t.Run(probe.name, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, probe.req)
			if rec.Code != probe.want {
				t.Fatalf("got %d, want %d", rec.Code, probe.want)
			}
		})
	}
}

func manifestText(panel *Handler) string {
	routes := panel.Routes()
	lines := make([]string, 0, len(routes))
	for _, route := range routes {
		lines = append(lines, route.Method+" "+route.Pattern)
	}

	return strings.Join(lines, "\n") + "\n"
}

func mustService(t *testing.T) *usermgmt.Service {
	t.Helper()

	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{AuditLog: usermgmt.NewAuditLog()})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	return svc
}
