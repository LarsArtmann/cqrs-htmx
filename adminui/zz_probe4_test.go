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

func TestProbeTitle(t *testing.T) {
	admin := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, admin)
	tenant := mustCreateTenant(t, svc, "acme4")
	member := mustRegister(t, svc, "u-attr5", "attr5@acme.com")
	ctx := usermgmt.WithUser(context.Background(), admin)
	if err := svc.AddMember(ctx, identitymodel.ActorIDFromUser(member.User.ID), tenant.ID,
		[]identitymodel.Role{identitymodel.RoleViewer}); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/tenants/acme4", nil))
	body := rec.Body.String()
	idx := strings.Index(body, "Whose grant")
	if idx < 0 {
		t.Fatal("no title at all")
	}
	t.Logf("title context: %s", body[max(0, idx-120):idx+220])
	_ = usermgmt.RoleViewer
}
