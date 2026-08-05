package adminui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestLayout_ContainsToastContainer verifies that the rendered layout HTML
// includes the templ-components feedback.ToastContainer div. Without this, the
// tcShowToast JS function is never defined and all toast notifications silently
// disappear.
func TestLayout_ContainsToastContainer(t *testing.T) {
	t.Parallel()
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/", nil))

	body := rec.Body.String()
	if !strings.Contains(body, `id="tc-toast-container"`) {
		t.Error("layout HTML must contain #tc-toast-container — ToastContainer was not rendered")
	}
}

// TestLayout_ContainsToastScript verifies that the tcShowToast function
// definition is present in the rendered HTML. This function is injected inline
// by ToastContainer and is called by the adminui:toast bridge in admin.js.
func TestLayout_ContainsToastScript(t *testing.T) {
	t.Parallel()
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/", nil))

	body := rec.Body.String()
	if !strings.Contains(body, "tcShowToast") {
		t.Error("layout HTML must define tcShowToast — toast bridge will fail silently without it")
	}
}

// TestLayout_ContainsGlobalErrorHandling verifies that the GlobalErrorHandling
// component rendered its inline script with the error announcer div.
func TestLayout_ContainsGlobalErrorHandling(t *testing.T) {
	t.Parallel()
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/", nil))

	body := rec.Body.String()
	if !strings.Contains(body, `id="tc-error-announcer"`) {
		t.Error("layout HTML must contain #tc-error-announcer — GlobalErrorHandling was not rendered")
	}
}

// TestLayout_ContainsErrorHandlingScript verifies that the GlobalErrorHandling
// inline script includes the HTMX error handling logic.
func TestLayout_ContainsErrorHandlingScript(t *testing.T) {
	t.Parallel()
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/", nil))

	body := rec.Body.String()
	if !strings.Contains(body, "htmx:responseError") {
		t.Error("layout HTML must contain htmx:responseError listener — GlobalErrorHandling script is missing")
	}
}
