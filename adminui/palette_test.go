package adminui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// T18 F131–F136 pins: the command palette ships as a native <dialog> in the
// layout (rendered on EVERY page), and admin.js carries the open-time index
// builder, fuzzy filter, keyboard navigation, and Ctrl/Cmd+K binding.

func TestPalette_DialogRenderedInLayout(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "palette@example.com")
	h, _ := newTestPanel(t, user)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /admin/users: status %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`<dialog id="admin-palette"`, // native dialog (F131), CSP-safe
		`data-palette-input`,
		`data-palette-list`,
		`data-palette-help`, // F134: "?" help lives in the dialog
		`role="listbox"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("layout must render palette markup %q on plain pages", want)
		}
	}
}

func TestPalette_IndexContractPinnedInJS(t *testing.T) {
	t.Parallel()

	js := panelAdminJS(t)
	for _, want := range []string{
		// F132: index built at OPEN time from DOM data attrs — no endpoint.
		"[data-palette-label][data-palette-href]",
		"admin-sidebar-scope",
		"#users-table",
		"#tenants-table",
		"#members-directory-table",
		"#audit-panel",
		// F133: fuzzy subsequence filter + keyboard nav + Enter jump.
		"fuzzyScore",
		"ArrowDown",
		"Enter",
		// F134: Ctrl/Cmd+K binding.
		"ctrlKey",
		"showModal",
		// CSP-safe: no inline handler assembly.
		"location.href",
	} {
		if !strings.Contains(js, want) {
			t.Errorf("admin.js missing palette hook %q", want)
		}
	}
}

func TestPalette_NoServerEndpointAdded(t *testing.T) {
	t.Parallel()

	// The palette indexes existing pages; the manifest must not grow a
	// palette/search route for it (F132: DOM-built index, NOT a server
	// endpoint).
	h := newManifestPanel(t, Config{})
	for _, r := range h.Routes() {
		if strings.Contains(r.Pattern, "palette") {
			t.Errorf("palette must not add routes; found %q", r.String())
		}
	}
}
