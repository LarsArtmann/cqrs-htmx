package adminui

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPanel_UserDetailNotFound(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users/nonexistent", nil))
	if rec.Code != http.StatusNotFound && rec.Code != http.StatusBadRequest &&
		rec.Code != http.StatusInternalServerError {
		t.Errorf("user detail not found: status %d, want 400/404/500", rec.Code)
	}
}

func TestPanel_TenantDetailNotFound(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/tenants/nonexistent", nil))
	if rec.Code != http.StatusNotFound && rec.Code != http.StatusInternalServerError {
		t.Errorf("tenant detail not found: status %d, want 404 or 500", rec.Code)
	}
}

func TestPanel_TenantReactivateNotFound(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := postForm(h, "/admin/tenants/nonexistent/reactivate", nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("reactivate nonexistent: status %d, want 400", rec.Code)
	}
}

func TestPanel_TenantDeleteNotFound(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := postForm(h, "/admin/tenants/nonexistent/delete", nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("delete nonexistent: status %d, want 400", rec.Code)
	}
}
