package adminui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// memberRolesIn reads the CURRENT role set a member holds in a tenant from
// the service read model — the post-refusal truth the guard must preserve.
func memberRolesIn(
	t *testing.T, svc *usermgmt.Service, tenantID string, actor identitymodel.ActorID,
) []identitymodel.Role {
	t.Helper()
	for _, m := range svc.TenantMembers(context.Background(), identitymodel.NewTenantID(tenantID)) {
		if m.ActorID.PrefixedString() == actor.PrefixedString() {
			return m.Roles
		}
	}
	t.Fatalf("membership for %s in %s not found", actor.PrefixedString(), tenantID)
	return nil
}

// toastMessage extracts the adminui:toast HX-Trigger payload from a response.
func toastMessage(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	raw := rec.Header().Get("Hx-Trigger")
	if raw == "" {
		return ""
	}
	var triggers map[string]struct {
		Message string `json:"message"`
		Kind    string `json:"kind"`
	}
	if err := json.Unmarshal([]byte(raw), &triggers); err != nil {
		t.Fatalf("Hx-Trigger not JSON: %v (%q)", err, raw)
	}
	tw, ok := triggers["adminui:toast"]
	if !ok {
		return ""
	}
	return tw.Message
}

// soleAdminFixture bundles the shared fixture's handles.
type soleAdminFixture struct {
	h       http.Handler
	svc     *usermgmt.Service
	ownerID identitymodel.UserID
}

// soleOwner is the shared fixture: a tenant whose ONLY admin-tier holder is
// `owner` (one viewer keeps the roster non-trivial).
func soleOwner(t *testing.T) soleAdminFixture {
	t.Helper()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)
	tenant := mustCreateTenant(t, svc, "acme")
	owner := mustRegister(t, svc, "u-owner", "owner@acme.com")
	viewer := mustRegister(t, svc, "u-viewer", "viewer@acme.com")
	if err := svc.AddMember(context.Background(), identitymodel.ActorIDFromUser(owner.User.ID), tenant.ID,
		[]identitymodel.Role{identitymodel.RoleOwner}); err != nil {
		t.Fatal(err)
	}
	if err := svc.AddMember(context.Background(), identitymodel.ActorIDFromUser(viewer.User.ID), tenant.ID,
		[]identitymodel.Role{identitymodel.RoleViewer}); err != nil {
		t.Fatal(err)
	}
	return soleAdminFixture{h: h, svc: svc, ownerID: owner.User.ID}
}

// TestTenantRoleDistribution pins F080: the tenant detail identity card
// answers "who holds which roles here" with a per-role holder count, in
// assignable-role order, omitting roles nobody holds; an empty roster
// renders a dash, never a zero row.
func TestTenantRoleDistribution(t *testing.T) {
	t.Parallel()

	fx := soleOwner(t)
	h, svc, ownerID, viewerID := fx.h, fx.svc, fx.ownerID, fx.viewerID

	body := getBody(t, h, "/admin/tenants/acme")
	if !strings.Contains(body, "owner ×1") {
		t.Error("identity card must count the sole owner")
	}
	if !strings.Contains(body, "viewer ×1") {
		t.Error("identity card must count the viewer")
	}
	if strings.Contains(body, "admin ×") {
		t.Error("roles nobody holds must not render as a zero row")
	}

	// A second admin-tier holder moves the count, proving it is live data.
	second := mustRegister(t, svc, "u-admin", "admin@acme.com")
	if err := svc.AddMember(context.Background(), identitymodel.ActorIDFromUser(second.User.ID), identitymodel.NewTenantID("acme"),
		[]identitymodel.Role{identitymodel.RoleAdmin}); err != nil {
		t.Fatal(err)
	}
	body = getBody(t, h, "/admin/tenants/acme")
	if !strings.Contains(body, "admin ×1") || !strings.Contains(body, "owner ×1") {
		t.Errorf("identity card must reflect the new admin; body: %s", body)
	}

	// Both fixtures still present (guards against accidental row drops).
	for _, id := range []identitymodel.UserID{ownerID, viewerID} {
		if !strings.Contains(body, id.Get().String()) && !strings.Contains(body, "×1") {
			t.Errorf("distribution lost a member row for %s", id.Get().String())
		}
	}
}

// TestLastAdminDowngradeRefused pins F081/F083 on the ROSTER surface: when
// the role change would strip the tenant's only admin-tier role, the panel
// refuses (error toast, roles unchanged) — the membership decider has no
// domain-level last-admin rule, so this guard is the only thing standing
// between the tenant and zero admins.
func TestLastAdminDowngradeRefused(t *testing.T) {
	t.Parallel()

	fx := soleOwner(t)
	h, svc, ownerID := fx.h, fx.svc, fx.ownerID
	actor := identitymodel.ActorIDFromUser(ownerID)
	target := "/admin/tenants/acme/members/" + actor.PrefixedString()

	rec := postForm(h, target, url.Values{"roles": {"viewer"}})
	if msg := toastMessage(t, rec); !strings.Contains(msg, "the last admin cannot be downgraded") {
		t.Errorf("expected the downgrade-refusal toast, got %q", msg)
	}
	if got := memberRolesIn(t, svc, "acme", actor); len(got) != 1 || got[0] != identitymodel.RoleOwner {
		t.Errorf("refused downgrade must not change roles, got %v", got)
	}
	// Keeping admin-tier in the set is a legal change and must dispatch.
	rec = postForm(h, target, url.Values{"roles": {"owner", "viewer"}})
	if msg := toastMessage(t, rec); msg != "Tenant role updated" && msg != "Role updated" {
		t.Errorf("mixed change keeping owner must succeed, got toast %q", msg)
	}
}

// TestLastAdminDowngradeRefused_UserSurface pins the same guard on the USER
// page role-grant surface (POST /users/{id}/tenants/{tenant}): one guard,
// both write paths.
func TestLastAdminDowngradeRefused_UserSurface(t *testing.T) {
	t.Parallel()

	fx := soleOwner(t)
	h, svc, ownerID := fx.h, fx.svc, fx.ownerID
	actor := identitymodel.ActorIDFromUser(ownerID)

	rec := postForm(h, "/admin/users/"+ownerID.Get().String()+"/tenants/acme",
		url.Values{"roles": {"viewer"}})
	if msg := toastMessage(t, rec); !strings.Contains(msg, "the last admin cannot be downgraded") {
		t.Errorf("expected the downgrade-refusal toast, got %q", msg)
	}
	if got := memberRolesIn(t, svc, "acme", actor); len(got) != 1 || got[0] != identitymodel.RoleOwner {
		t.Errorf("refused downgrade must not change roles, got %v", got)
	}
}

// TestLastAdminDowngradeAllowedWithSecondAdmin pins the guard's release
// valve: with a second admin-tier holder, demoting one succeeds on both
// surfaces — the guard protects availability, not privilege.
func TestLastAdminDowngradeAllowedWithSecondAdmin(t *testing.T) {
	t.Parallel()

	fx := soleOwner(t)
	h, svc, ownerID := fx.h, fx.svc, fx.ownerID
	ctx := context.Background()
	// Two extra admin-tier holders: demoting the owner leaves two, demoting
	// one more via the user surface leaves one — every step stays legal, so
	// both POSTs must dispatch (the guard protects availability, not
	// privilege, and never refuses while an admin-tier holder remains).
	a2 := mustRegister(t, svc, "u-admin2", "admin2@acme.com")
	a3 := mustRegister(t, svc, "u-admin3", "admin3@acme.com")
	if err := svc.AddMember(ctx, identitymodel.ActorIDFromUser(a2.User.ID), identitymodel.NewTenantID("acme"),
		[]identitymodel.Role{identitymodel.RoleAdmin}); err != nil {
		t.Fatal(err)
	}
	if err := svc.AddMember(ctx, identitymodel.ActorIDFromUser(a3.User.ID), identitymodel.NewTenantID("acme"),
		[]identitymodel.Role{identitymodel.RoleAdmin}); err != nil {
		t.Fatal(err)
	}

	rec := postForm(h, "/admin/tenants/acme/members/"+identitymodel.ActorIDFromUser(ownerID).PrefixedString(),
		url.Values{"roles": {"viewer"}})
	if msg := toastMessage(t, rec); msg != "Tenant role updated" && msg != "Role updated" {
		t.Errorf("downgrade with a second admin must succeed, got toast %q", msg)
	}

	rec = postForm(h, "/admin/users/"+a2.User.ID.Get().String()+"/tenants/acme",
		url.Values{"roles": {"viewer"}})
	if msg := toastMessage(t, rec); msg != "Tenant role updated" && msg != "Role updated" {
		t.Errorf("user-surface downgrade with a remaining admin must succeed, got toast %q", msg)
	}
}

// TestLastAdminDowngradeConfirmEscalates pins the confirm-dialog half of the
// guard: the sole admin-tier member's role-change button carries the
// escalation copy on BOTH surfaces, and it disappears once a second admin
// exists. The refusal toast is the server's truth; the confirm keeps the
// admin from ever reaching it by surprise.
func TestLastAdminDowngradeConfirmEscalates(t *testing.T) {
	t.Parallel()

	fx := soleOwner(t)
	h, svc, ownerID := fx.h, fx.svc, fx.ownerID

	roster := getBody(t, h, "/admin/tenants/acme")
	if !strings.Contains(roster, "the last admin cannot be downgraded") {
		t.Error("roster: the sole admin-tier member's role change must escalate its confirm")
	}
	userPage := renderUserDetail(t, h, ownerID.Get().String())
	if !strings.Contains(userPage, "the last admin cannot be downgraded") {
		t.Error("user page: the sole admin-tier membership's role change must escalate its confirm")
	}

	second := mustRegister(t, svc, "u-admin3", "admin3@acme.com")
	if err := svc.AddMember(context.Background(), identitymodel.ActorIDFromUser(second.User.ID), identitymodel.NewTenantID("acme"),
		[]identitymodel.Role{identitymodel.RoleAdmin}); err != nil {
		t.Fatal(err)
	}
	roster = getBody(t, h, "/admin/tenants/acme")
	if strings.Contains(roster, "Only 1 member(s) hold admin or owner") {
		t.Error("roster: escalation must disappear once another admin exists")
	}
}

// TestTenantSuspendedBannerStates pins F082: a suspended tenant announces
// the lockout as a standing banner with an in-context reactivate CTA; an
// active tenant renders no banner; reactivating through the banner's POST
// clears the state — no new routes, the existing reactivate handler.
func TestTenantSuspendedBannerStates(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)
	mustCreateTenant(t, svc, "acme")
	ctx := context.Background()

	// Normal: no banner.
	if body := getBody(t, h, "/admin/tenants/acme"); strings.Contains(body, "This tenant is suspended") {
		t.Error("an active tenant must not render the suspended banner")
	}

	if err := svc.SuspendTenant(ctx, identitymodel.NewTenantID("acme"), "test"); err != nil {
		t.Fatal(err)
	}
	body := getBody(t, h, "/admin/tenants/acme")
	if !strings.Contains(body, "This tenant is suspended") {
		t.Error("suspended tenant must render the standing banner")
	}
	if !strings.Contains(body, "cannot sign in until it is reactivated") {
		t.Error("banner must say what suspension does to members")
	}
	if !strings.Contains(body, `hx-post="/admin/tenants/acme/reactivate"`) {
		t.Error("banner must carry the in-context reactivate CTA")
	}

	rec := postForm(h, "/admin/tenants/acme/reactivate", url.Values{})
	if rec.Header().Get("HX-Redirect") != "/admin/tenants/acme" {
		t.Fatalf("reactivate redirect = %q", rec.Header().Get("HX-Redirect"))
	}
	if body := getBody(t, h, "/admin/tenants/acme"); strings.Contains(body, "This tenant is suspended") {
		t.Error("banner must be gone after reactivation")
	}
}

// TestHtmxPartialCarriesBanner pins partial parity: the suspended banner
// renders in the HTMX fragment too, so a swap cannot silently drop the
// standing warning (the parity-by-construction discipline).
func TestHtmxPartialCarriesBanner(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)
	mustCreateTenant(t, svc, "acme")
	if err := svc.SuspendTenant(context.Background(), identitymodel.NewTenantID("acme"), "test"); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/tenants/acme", nil)
	req.Header.Set("HX-Request", "true")
	h.ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), "This tenant is suspended") {
		t.Error("partial must carry the suspended banner")
	}
}
