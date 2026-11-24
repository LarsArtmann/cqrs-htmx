package adminui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPanel_AuditIndexRenders(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/audit", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("audit index: status %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestPanel_AuditPageWithEntries(t *testing.T) {
	t.Parallel()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	// Generate audit entries by registering users
	for i := range 3 {
		mustRegister(t, svc, "u-audit"+string(rune('0'+i)), "audit"+string(rune('0'+i))+"@test.com")
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/audit", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("audit page: status %d, body %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "audit") {
		t.Error("audit page should contain audit-related content")
	}
}
