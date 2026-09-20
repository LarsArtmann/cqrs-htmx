package adminui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// TestUserMenuDropdownRendered verifies the header identity cluster on desktop:
// a library Dropdown (native Popover API) whose trigger carries the email and
// whose single item is the sign-out link, plus the singleton positioner script.
func TestUserMenuDropdownRendered(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{AuditLog: usermgmt.NewAuditLog()})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	h, _ := newTestPanel(t, user, Config{Service: svc, LogoutURL: "/logout-test"})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/", nil))
	html := rec.Body.String()

	for _, want := range []string{
		`popovertarget="dropdown`,        // trigger toggles the popover panel
		`popover="auto"`,                 // native light-dismiss + top layer
		`role="menu"`,                    // menu semantics
		`href="/logout-test"`,            // the Sign out item keeps its link
		`Sign out`,                       // item label
		`aria-label="admin@example.com"`, // trigger is labelled by the email
		`tcPopoverPosition`,              // library positioner singleton shipped
		`data-tc-anchor`,                 // panel anchored to the trigger
	} {
		if !strings.Contains(html, want) {
			t.Errorf("user menu dropdown missing %q", want)
		}
	}
	if strings.Contains(html, " onclick=") || strings.Contains(html, " onsubmit=") {
		t.Error("user menu must not introduce inline handlers (CSP posture)")
	}
}

// TestUserMenuMobileFallback verifies the plain sign-out link stays available
// on small screens where the dropdown is hidden (lg breakpoint).
func TestUserMenuMobileFallback(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{AuditLog: usermgmt.NewAuditLog()})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	h, _ := newTestPanel(t, user, Config{Service: svc, LogoutURL: "/logout-test"})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/", nil))
	html := rec.Body.String()

	if !strings.Contains(html, `class="lg:hidden inline-flex`) {
		t.Error("mobile fallback sign-out link (lg:hidden) missing")
	}
	if strings.Count(html, `href="/logout-test"`) != 2 {
		t.Errorf("expected exactly 2 sign-out hrefs (dropdown item + mobile fallback), got %d", strings.Count(html, `href="/logout-test"`))
	}
}
