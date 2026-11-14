package adminui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

func TestProbePage(t *testing.T) {
	admin := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, admin)
	tenant := mustCreateTenant(t, svc, "acme3")
	member := mustRegister(t, svc, "u-attr4", "attr4@acme.com")
	ctx := usermgmt.WithUser(context.Background(), admin)
	if err := svc.AddMember(ctx, identitymodel.ActorIDFromUser(member.User.ID), tenant.ID,
		[]identitymodel.Role{identitymodel.RoleViewer}); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/tenants/acme3", nil))
	body := rec.Body.String()
	t.Logf("status=%d has-by=%v has-title=%v len=%d", rec.Code, strings.Contains(body, "by admin@example.com"), strings.Contains(body, "Whose grant"), len(body))
	if !strings.Contains(body, "by admin@example.com") {
		idx := strings.Index(body, "attr4@acme.com")
		t.Logf("context around member: %s", body[max(0, idx-200):idx+300])
	}
}
