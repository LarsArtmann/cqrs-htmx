package adminui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPanel_StyledBadRequest(t *testing.T) {
	srv, _ := newTestPanel(t, mustUser(t, "admin@demo.dev"))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users/not-a-ulid", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want html", got)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Invalid user id") {
		t.Error("full-page error should contain the error title")
	}
	if !strings.Contains(body, "Back to dashboard") {
		t.Error("full-page error should offer a way out")
	}
}

func TestPanel_ErrorPageHTMXBareCard(t *testing.T) {
	srv, _ := newTestPanel(t, mustUser(t, "admin@demo.dev"))
	req := httptest.NewRequest(http.MethodGet, "/admin/users/not-a-ulid", nil)
	req.Header.Set("Hx-Request", "true")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "<html") {
		t.Error("HTMX error should be a bare card, not a full document")
	}
	if !strings.Contains(rec.Body.String(), "Invalid user id") {
		t.Error("HTMX error should still contain the error title")
	}
}

func TestPanel_Styled404CatchAll(t *testing.T) {
	srv, _ := newTestPanel(t, mustUser(t, "admin@demo.dev"))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/definitely-not-a-page", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "404") {
		t.Error("styled 404 should show the 404 numeral")
	}
	if !strings.Contains(body, "Back to dashboard") {
		t.Error("styled 404 should offer a way back")
	}
}

func TestPanel_UnauthenticatedStyled(t *testing.T) {
	srv, _ := newTestPanel(t, nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Sign in required") {
		t.Error("styled 401 should contain the title")
	}
}
