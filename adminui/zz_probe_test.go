package adminui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestT16ProbeUsersBody(t *testing.T) {
	svc := mustService(t)
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user, Config{Service: svc, Locale: "ar", Dir: "rtl"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin", nil))
	t.Logf("STATUS: %d", rec.Code)
	i := strings.Index(rec.Body.String(), "<!DOCTYPE")
	if i < 0 {
		i = 0
	}
	t.Logf("HEAD: %.300s", rec.Body.String()[i:])
	body := rec.Body.String()
	for _, line := range strings.Split(body, "\n") {
		if i := strings.Index(line, ">Users<"); i >= 0 {
			lo := max(0, i-120)
			t.Logf("CTX: ...%s...", line[lo:min(i+40, len(line))])
		}
	}
	t.Logf("HAS HTML: %v", strings.Contains(body, "<html"))
}
