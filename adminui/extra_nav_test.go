package adminui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/templ-components/icons"
)

func TestNew_RejectsExtraNavWithEmptyLabel(t *testing.T) {
	svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{})

	_, err := New(Config{Service: svc, ExtraNav: []NavEntry{{Href: "/support"}}})
	if err == nil {
		t.Fatal("New() with an empty ExtraNav label should fail")
	}
	if errorfamily.Classify(err) != errorfamily.Rejection {
		t.Errorf("New error family = %v; want Rejection; got %v", errorfamily.Classify(err), err)
	}
}

func TestNew_RejectsExtraNavWithEmptyHref(t *testing.T) {
	svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{})

	_, err := New(Config{Service: svc, ExtraNav: []NavEntry{{Label: "Support"}}})
	if err == nil {
		t.Fatal("New() with an empty ExtraNav href should fail")
	}
}

func TestNew_AppendsExtraNavAfterBuiltInItems(t *testing.T) {
	svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{})

	h, err := New(Config{
		Service: svc,
		ExtraNav: []NavEntry{
			{Label: "Support", Href: "/support"},
			{Label: "Docs", Href: "https://docs.example.com", Icon: icons.Book},
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if len(h.nav) != 6 {
		t.Fatalf("nav = %+v, want 4 built-in items plus two extra entries", h.nav)
	}

	last := h.nav[len(h.nav)-2:]
	if last[0].Label != "Support" || last[0].Href != "/support" {
		t.Errorf("extra nav[0] = %+v, want Support at /support", last[0])
	}
	if last[0].Icon != string(icons.Link) {
		t.Errorf("extra nav[0].Icon = %q, want the default link icon", last[0].Icon)
	}
	if !last[0].Verbatim {
		t.Errorf("extra nav[0].Verbatim = false, want true (consumer hrefs render as-is)")
	}
	if last[1].Icon != string(icons.Book) {
		t.Errorf("extra nav[1].Icon = %q, want the explicit %q icon", last[1].Icon, icons.Book)
	}

	for i, n := range h.nav[:4] {
		if n.Verbatim {
			t.Errorf("built-in nav[%d] = %+v unexpectedly verbatim", i, n)
		}
	}
}

func TestNew_DashboardURLAppendsEventStoreEntry(t *testing.T) {
	svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{})

	h, err := New(Config{
		Service:      svc,
		ExtraNav:     []NavEntry{{Label: "Support", Href: "/support"}},
		DashboardURL: "/dashboard/",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if len(h.nav) != 6 {
		t.Fatalf("nav = %+v, want built-ins + extra + cross-link", h.nav)
	}
	link := h.nav[len(h.nav)-1]
	if link.Label != "Event store" || link.Href != "/dashboard/" {
		t.Errorf("cross-link = %+v, want Event store at /dashboard/", link)
	}
	if !link.Verbatim {
		t.Error("cross-link.Verbatim = false, want true (dashboard mounts outside the admin subtree)")
	}
	if link.Icon != string(icons.CircleStack) {
		t.Errorf(
			"cross-link.Icon = %q, want the distinct %q icon (Dashboard already uses Chart)",
			link.Icon,
			icons.CircleStack,
		)
	}
	if link.Icon == iconDashboard {
		t.Error("cross-link reuses the Dashboard icon")
	}
}

func TestNew_NoExtraNavOrCrossLinkByDefault(t *testing.T) {
	svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{})

	h, err := New(Config{Service: svc})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if len(h.nav) != 4 {
		t.Fatalf("nav = %+v, want exactly the 4 built-in super-admin items", h.nav)
	}
	for _, n := range h.nav {
		if n.Label == "Event store" {
			t.Errorf("Event store cross-link rendered without Config.DashboardURL: %+v", n)
		}
	}
}

func TestExtraNav_RendersInSidebarWithVerbatimHref(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{})
	h, _ := newTestPanel(t, user, Config{
		Service:      svc,
		Authorizer:   RequireAuthenticated(),
		ExtraNav:     []NavEntry{{Label: "Support", Href: "/support"}},
		DashboardURL: "/dashboard/",
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		`href="/support"`,
		`href="/dashboard/"`,
		"Support",
		"Event store",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("sidebar missing %q", want)
		}
	}
	if strings.Contains(body, `href="/admin/support"`) {
		t.Error("ExtraNav href was BasePath-prefixed; the contract renders it verbatim")
	}
	if strings.Contains(body, `href="/admin/dashboard/"`) {
		t.Error("DashboardURL href was BasePath-prefixed; the contract renders it verbatim")
	}
}

func TestExtraNav_TenantAdminModeAppendsAfterBuiltIns(t *testing.T) {
	svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{})

	h, err := New(Config{
		Service:  svc,
		Mode:     ModeTenantAdmin,
		TenantID: identitymodel.NewTenantID("acme"),
		ExtraNav: []NavEntry{{Label: "Billing", Href: "/billing"}},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if len(h.nav) != 4 {
		t.Fatalf("nav = %+v, want 3 tenant-mode built-ins plus the extra entry", h.nav)
	}
	last := h.nav[len(h.nav)-1]
	if last.Label != "Billing" || last.Href != "/billing" || !last.Verbatim {
		t.Errorf("tenant-mode extra entry = %+v, want verbatim Billing at /billing", last)
	}
}
