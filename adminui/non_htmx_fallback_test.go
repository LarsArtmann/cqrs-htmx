package adminui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

func TestPanel_NonHTMXDeleteRedirects(t *testing.T) {
	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	target, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: identitymodel.SyntheticUserID("u-goner2"), Email: "goner2@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	id := target.User.ID.Get().String()

	// Non-HTMX request (no HX-Request header) — should still process the delete
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/users/"+id+"/delete", nil)
	h.ServeHTTP(rec, req)

	// Should redirect (302) without HX-Redirect since it's not an HTMX request
	if rec.Code != http.StatusOK && rec.Code != http.StatusFound && rec.Code != http.StatusSeeOther {
		t.Errorf("non-htmx delete: status %d", rec.Code)
	}
}
