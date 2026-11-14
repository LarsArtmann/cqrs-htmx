package adminui

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
)

func TestDebugM10(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)
	mustCreateTenant(t, svc, "lonely")
	mustCreateTenant(t, svc, "busy")
	u1 := mustRegister(t, svc, "m1", "m1@acme.dev")
	u2 := mustRegister(t, svc, "m2", "m2@acme.dev")
	mustAddMember(t, svc, u1.User.ID, identitymodel.NewTenantID("busy"), identitymodel.RoleViewer)
	mustAddMember(t, svc, u2.User.ID, identitymodel.NewTenantID("busy"), identitymodel.RoleViewer)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/tenants", nil))
	body := rec.Body.String()
	fmt.Printf("IDX lonely=%d busy=%d busyCount=%d\n", strings.Index(body, "lonely"), strings.Index(body, "busy"), strings.Count(body, "busy"))
	i := strings.Index(body, "lonely")
	if i >= 0 && i < strings.Index(body, "busy") {
		fmt.Printf("LONELY_FIRST_CTX: %.200s\n", body[i-100:i+60])
	}
}
