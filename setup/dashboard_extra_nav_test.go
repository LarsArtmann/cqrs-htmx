package setup_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/dashboardui/v4"
	"github.com/larsartmann/cqrs-htmx/setup/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// TestBundleHandler_DashboardExtraNavRendersVerbatim is the end-to-end proof of
// cqrs-htmx#38's motivating case: a consumer mounting the bundle adds a link to
// a surface served NEXT DOOR to the dashboard (outside the mounted subtree) and
// the rendered sidebar points at it as written, not at a BasePath-prefixed path
// that 404s.
func TestBundleHandler_DashboardExtraNavRendersVerbatim(t *testing.T) {
	t.Parallel()

	bundle, err := setup.New(setup.Config{
		Title:             "Dashboard Extra Nav E2E",
		DashboardPath:     "/dash",
		DashboardExtraNav: []dashboardui.NavEntry{{Label: "Event Catalog", Href: "/event-catalog"}},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = bundle.Close() }()

	reg, err := bundle.Service.Register(context.Background(), usermgmt.RegisterRequest{
		ID: usermgmt.SyntheticUserID("dashnav1"), Email: "dashnav1@test.com",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	handler := bundle.Handler(http.NewServeMux())

	req := httptest.NewRequest(http.MethodGet, "/dash/", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: reg.Session.Token})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("dashboard status = %d, want 200, body: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()

	if !strings.Contains(body, `href="/event-catalog"`) {
		t.Error("extra nav href missing from the rendered sidebar, or not verbatim")
	}

	if strings.Contains(body, `href="/dash/event-catalog"`) {
		t.Error("extra nav href was BasePath-prefixed; a sibling surface must render as written")
	}

	if !strings.Contains(body, "Event Catalog") {
		t.Error("extra nav label missing from the rendered sidebar")
	}
}
