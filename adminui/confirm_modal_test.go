package adminui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// get renders a page through the test panel and returns status + body.
func get(t *testing.T, h http.Handler, path string) (int, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	h.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

// TestConfirmModalRendered verifies the shared confirm dialog is present on
// every laid-out page: native <dialog>, dynamic-fill hook IDs, a
// data-tc-close Cancel button, and the library's modal script (which carries
// the page nonce — CSP-safe). The dialog is rendered once via pageFoot;
// admin.js fills title/body from the triggering button's data-confirm /
// data-confirm-title attributes.
func TestConfirmModalRendered(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	code, html := get(t, h, "/admin/users")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
	for _, want := range []string{
		`<dialog id="admin-confirm-modal"`,
		`id="admin-confirm-modal-title"`,
		`id="admin-confirm-body"`,
		`id="admin-confirm-ok"`,
		`data-tc-close`,
		`tcOpenModal`,
		`Confirm</button>`,
		`Cancel</button>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("confirm modal missing %q", want)
		}
	}
}

// TestConfirmModalNoInlineHandlers keeps the CSP posture: the modal (and its
// library script) must not introduce inline event handler attributes.
func TestConfirmModalNoInlineHandlers(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	_, html := get(t, h, "/admin/users")
	for _, banned := range []string{" onclick=", " onsubmit=", " onchange=", " oninput="} {
		if strings.Contains(html, banned) {
			t.Errorf("inline handler %q leaked into modal page", banned)
		}
	}
}

// TestConfirmAttributesOnDestructiveButtons asserts every destructive action
// carries the data-confirm pair (body + title) the shared modal is filled
// from — and that no hx-confirm (htmx's native window.confirm path) remains:
// the modal is the ONLY confirm UX. The server-side handlers are unchanged;
// the modal is progressive enhancement over the same htmx requests.
func TestConfirmAttributesOnDestructiveButtons(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	_, usersHTML := get(t, h, "/admin/users")
	if !strings.Contains(usersHTML, `data-confirm-title="Delete user"`) {
		t.Error("user delete button missing data-confirm-title")
	}
	if !strings.Contains(usersHTML, "This cannot be undone") {
		t.Error("user delete button missing data-confirm body")
	}
	if strings.Contains(usersHTML, "hx-confirm=") {
		t.Error("hx-confirm must not remain: the shared modal replaces the native confirm")
	}

	_, tenantsHTML := get(t, h, "/admin/tenants")
	if !strings.Contains(tenantsHTML, `data-confirm-title="Delete tenant"`) {
		t.Error("tenant delete button missing data-confirm-title")
	}
}
