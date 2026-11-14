package adminui

import (
	"context"

	"net/url"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
)

func TestProbeEveryTenant(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)
	_ = h
	mustCreateTenant(t, svc, "acme")
	target := mustRegister(t, svc, "u-all", "all@acme.com")
	userBase := "/admin/users/" + target.User.ID.Get().String()
	rec := postForm(h, userBase+"/tenants", url.Values{"tenant": {"acme"}, "roles": {"viewer"}})
	t.Logf("redirect=%q trigger=%q", rec.Header().Get("HX-Redirect"), rec.Header().Get("HX-Trigger"))
	members := svc.TenantMembers(context.Background(), identitymodel.NewTenantID("acme"))
	t.Logf("members=%d", len(members))
	tenants := svc.AllTenants()
	t.Logf("allTenants=%d", len(tenants))
	for _, tn := range tenants {
		t.Logf("tenant id=%q name=%q", tn.ID.Get(), tn.Name)
	}
}
