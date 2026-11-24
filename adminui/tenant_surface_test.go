package adminui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

// newPanelConfig is newTestPanel with a caller-supplied Config (mode, page
// size, tenant scope): the service is created here and injected, so callers
// never pass a half-built Config.
func newPanelConfig(t *testing.T, user *identitymodel.User, cfg Config) (http.Handler, *usermgmt.Service) {
	t.Helper()
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{AuditLog: usermgmt.NewAuditLog()})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	cfg.Service = svc
	if cfg.Authorizer == nil {
		cfg.Authorizer = RequireAuthenticated()
	}
	panel, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	mux := http.NewServeMux()
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if user != nil {
				r = r.WithContext(usermgmt.WithUser(r.Context(), user))
			}
			next.ServeHTTP(w, r)
		})
	}
	mux.Handle("/admin/", mw(http.StripPrefix("/admin", panel.Handler())))
	return mux, svc
}

// panelAdminJS fetches the served admin.js asset body.
func panelAdminJS(t *testing.T) string {
	t.Helper()
	user := mustUser(t, "js@example.com")
	h, _ := newTestPanel(t, user)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/-/admin.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET admin.js: status %d", rec.Code)
	}
	return rec.Body.String()
}

// --- F152: slug auto-suggest (JS pin + template marker) ---

func TestTenantNew_SlugSuggestWired(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/tenants/new", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /tenants/new: status %d", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "data-slug-suggest") {
		t.Error("tenant new form must mark the slug field with data-slug-suggest")
	}

	js := panelAdminJS(t)
	for _, want := range []string{"slugify", "data-slug-touched"} {
		if !strings.Contains(js, want) {
			t.Errorf("admin.js missing slug-suggest hook %q", want)
		}
	}
}

// --- F153: role descriptions in the checkbox group ---

func TestRoleCheckboxGroup_CarriesDescriptions(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)
	mustCreateTenant(t, svc, "acme")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/tenants/acme", nil))
	body := rec.Body.String()
	for _, want := range []string{
		"Full control of this tenant",
		"Manage members and roles inside this tenant",
		"Read-only access to this tenant",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("role descriptions missing %q", want)
		}
	}
}

// --- F154 + F155: tombstone view, resolved from the audit window ---

func TestTenantDetail_DeletedRendersTombstoneNotLiveView(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)
	tenant := mustCreateTenant(t, svc, "ghost")
	u := mustRegister(t, svc, "m1", "m1@acme.dev")
	mustAddMember(t, svc, u.User.ID, tenant.ID, identitymodel.RoleOwner)

	ctx := cqrshtmx.WithActorID(context.Background(), identitymodel.ActorIDFromUser(user.ID))
	if err := svc.DeleteTenant(ctx, tenant.ID, "test deletion"); err != nil {
		t.Fatalf("DeleteTenant: %v", err)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/tenants/ghost", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET deleted tenant: status %d", rec.Code)
	}
	body := rec.Body.String()
	// The tombstone renders, with the audit-resolved actor; the slug is the
	// only surviving identity (the display name lived in the read model).
	for _, want := range []string{"This tenant is deleted", "Deleted by", "admin@example.com", "ghost"} {
		if !strings.Contains(body, want) {
			t.Errorf("tombstone missing %q", want)
		}
	}
	// The audit window resolved BOTH facts: nothing says "not recorded".
	if strings.Contains(body, labelTombstoneNotRecorded) {
		t.Error("recorded deletion must not render the not-recorded fallback")
	}
	// F155: the LIVE view must not render for a deleted tenant.
	for _, forbidden := range []string{"Add member", "Delete tenant", "data-transfer-confirm"} {
		if strings.Contains(body, forbidden) {
			t.Errorf("live-view element %q must not render on the tombstone", forbidden)
		}
	}
}

// TestTenantTombstone_NotRecordedHonesty pins the honest fallback: an audit
// window without the TenantDeleted entry yields Recorded=false — the view
// says "not recorded" instead of guessing a time or an actor.
func TestTenantTombstone_NotRecordedHonesty(t *testing.T) {
	t.Parallel()

	panel := newManifestPanel(t, Config{Authorizer: RequireAuthenticated()})
	svc := panel.config.Service
	tenant := mustCreateTenant(t, svc, "quiet")

	// Simulate a rotated audit window: a fresh service whose audit log never
	// saw the deletion.
	panel.config.Service = rebuildServiceWithAuditLog(t, svc)

	tb := panel.tenantTombstone(context.Background(), tenant.ID)
	if tb.Recorded {
		t.Fatal("empty audit window must yield Recorded=false")
	}
	if !tb.DeletedAt.IsZero() || tb.DeletedBy != "" {
		t.Errorf("unrecorded tombstone must carry no guessed facts, got %+v", tb)
	}
}

// --- F156: roster CSV export ---

func TestRosterCSV_SuperAdminExport(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)
	tenant := mustCreateTenant(t, svc, "acme")
	u := mustRegister(t, svc, "m1", "m1@acme.dev")
	mustAddMember(t, svc, u.User.ID, tenant.ID, identitymodel.RoleAdmin, identitymodel.RoleViewer)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/tenants/acme/roster.csv", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("roster export: status %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/csv") {
		t.Errorf("content type = %q, want text/csv", ct)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"actor,email,display_name,roles,added_at",
		"m1@acme.dev",
		"admin viewer",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("roster CSV missing %q:\n%s", want, body)
		}
	}
}

func TestRosterCSV_TenantAdminExport(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	tenant := identitymodel.NewTenantID("scoped")
	h, svc := newPanelConfig(t, user, Config{Mode: ModeTenantAdmin, TenantID: tenant})
	u := mustRegister(t, svc, "m1", "m1@scoped.dev")
	mustAddMember(t, svc, u.User.ID, tenant, identitymodel.RoleViewer)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/members/roster.csv", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("tenant-admin roster export: status %d", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "m1@scoped.dev") {
		t.Errorf("tenant-admin roster CSV missing the member:\n%s", body)
	}
}

// --- F157: roster pagination beyond MaxListRows ---

func TestRosterPagination_PagesBeyondMaxListRows(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	// Config.PageSize is clamped to MinPageSize (10) at New — so 11 members
	// make two pages without violating the clamp.
	h, svc := newPanelConfig(t, user, Config{PageSize: MinPageSize})
	tenant := mustCreateTenant(t, svc, "crowd")
	for i := range 11 {
		u := mustRegister(t, svc, "m"+string(rune('a'+i)), "m"+string(rune('a'+i))+"@crowd.dev")
		mustAddMember(t, svc, u.User.ID, tenant.ID, identitymodel.RoleViewer)
	}

	getBody := func(page string) string {
		t.Helper()
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/tenants/crowd?page="+page, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("GET page=%s: status %d", page, rec.Code)
		}
		return rec.Body.String()
	}

	page1, page2 := getBody("1"), getBody("2")
	// Row presence is counted on the roster row links (…</a>): the transfer
	// select and the add-member datalist repeat emails outside the table.
	if got := strings.Count(page1, "@crowd.dev</a>"); got != 10 {
		t.Errorf("page 1 must carry exactly ten member rows, got %d", got)
	}
	if got := strings.Count(page2, "@crowd.dev</a>"); got != 1 || !strings.Contains(page2, ">mk@crowd.dev") {
		t.Errorf("page 2 must carry only the eleventh member, got %d rows", got)
	}
	if !strings.Contains(page1, "Showing page") || !strings.Contains(page1, ">2</span>") {
		t.Error("pager must announce 2 pages")
	}
	// The identity card quotes the FULL roster, not the page.
	if !strings.Contains(page1, "<dt>Members</dt><dd>11</dd>") &&
		!strings.Contains(page1, ">11</dd>") {
		t.Error("identity card must quote the full roster size (11)")
	}
}

// --- F158: switcher search (JS pin + data attribute) ---

func TestTenantSwitcher_SearchWired(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)
	mustCreateTenant(t, svc, "acme")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/tenants", nil))
	if body := rec.Body.String(); !strings.Contains(body, "data-tenant-search") {
		t.Error("switcher items must carry data-tenant-search for the filter")
	}

	js := panelAdminJS(t)
	for _, want := range []string{"admin-tenant-switcher-menu", "Filter tenants", "data-tenant-search"} {
		if !strings.Contains(js, want) {
			t.Errorf("admin.js missing switcher-search hook %q", want)
		}
	}
}

// --- T22 F163: the transfer form names both sides ---

func TestTenantDetail_TransferFormNamesBothSides(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)
	tenant := mustCreateTenant(t, svc, "acme")
	owner := mustRegister(t, svc, "owner", "owner@acme.dev")
	target := mustRegister(t, svc, "target", "target@acme.dev")
	mustAddMember(t, svc, owner.User.ID, tenant.ID, identitymodel.RoleOwner)
	mustAddMember(t, svc, target.User.ID, tenant.ID, identitymodel.RoleViewer)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/tenants/acme", nil))
	body := rec.Body.String()
	for _, want := range []string{
		"data-transfer-confirm",
		"Transfer ownership from owner@acme.dev to target@acme.dev?",
		"target@acme.dev gains the owner role",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("transfer UI missing %q", want)
		}
	}
	// The current owner is never offered as a target (that transfer is refused).
	if strings.Contains(body, `value="owner@acme.dev"`) {
		t.Error("the current owner must not be a transfer candidate")
	}
}

// --- T22 F165: transfer success ---

func TestTransferOwnership_Success(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)
	tenant := mustCreateTenant(t, svc, "acme")
	owner := mustRegister(t, svc, "owner", "owner@acme.dev")
	target := mustRegister(t, svc, "target", "target@acme.dev")
	mustAddMember(t, svc, owner.User.ID, tenant.ID, identitymodel.RoleOwner, identitymodel.RoleAdmin)
	mustAddMember(t, svc, target.User.ID, tenant.ID, identitymodel.RoleViewer)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/tenants/acme/transfer", strings.NewReader("email=target@acme.dev"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(rec, req)

	members := svc.TenantMembers(context.Background(), tenant.ID)
	owners := 0
	for _, m := range members {
		isTarget := m.ActorID.PrefixedString() == identitymodel.ActorIDFromUser(target.User.ID).PrefixedString()
		isOwner := m.ActorID.PrefixedString() == identitymodel.ActorIDFromUser(owner.User.ID).PrefixedString()
		if hasRole(m.Roles, identitymodel.RoleOwner) {
			owners++
			if !isTarget {
				t.Error("the TARGET must hold owner after the transfer")
			}
		}
		if isTarget && !hasRole(m.Roles, identitymodel.RoleViewer) {
			t.Error("the target's pre-existing viewer role must survive the grant")
		}
		if isOwner && hasRole(m.Roles, identitymodel.RoleOwner) {
			t.Error("the prior owner must be demoted")
		}
		if isOwner && !hasRole(m.Roles, identitymodel.RoleAdmin) {
			t.Error("the prior owner's non-owner role (admin) must be kept")
		}
	}
	if owners != 1 {
		t.Fatalf("owners after transfer = %d, want 1", owners)
	}
}

// --- T22 F165: self-transfer refusal (target already owner) ---

func TestTransferOwnership_SelfTransferRefused(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)
	tenant := mustCreateTenant(t, svc, "acme")
	owner := mustRegister(t, svc, "owner", "owner@acme.dev")
	mustAddMember(t, svc, owner.User.ID, tenant.ID, identitymodel.RoleOwner)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/tenants/acme/transfer", strings.NewReader("email=owner@acme.dev"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(rec, req)

	members := svc.TenantMembers(context.Background(), tenant.ID)
	if len(members) != 1 || !hasRole(members[0].Roles, identitymodel.RoleOwner) {
		t.Fatalf("refused transfer must leave the roster unchanged, got %+v", members)
	}
	for _, m := range members {
		if m.ActorID.PrefixedString() != identitymodel.ActorIDFromUser(owner.User.ID).PrefixedString() {
			t.Error("refused transfer must not create or change memberships")
		}
	}
}

// TestTransferOwnership_NotAMemberRefused pins the second refusal case:
// nothing lands when the target is not a member.
func TestTransferOwnership_NotAMemberRefused(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)
	tenant := mustCreateTenant(t, svc, "acme")
	owner := mustRegister(t, svc, "owner", "owner@acme.dev")
	_ = mustRegister(t, svc, "out", "out@acme.dev")
	mustAddMember(t, svc, owner.User.ID, tenant.ID, identitymodel.RoleOwner)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/tenants/acme/transfer", strings.NewReader("email=out@acme.dev"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(rec, req)

	if members := svc.TenantMembers(context.Background(), tenant.ID); len(members) != 1 {
		t.Fatalf("refused transfer must not create a membership, got %d", len(members))
	}
}

// --- T22 F164+F165: mid-sequence failure honesty ---

// TestTransferOwnership_MidSequenceFailureHonest drives the sequencer with a
// prior owner whose membership vanishes after the snapshot: the GRANT lands
// (durable), the DEMOTION fails, and the outcome must say exactly that —
// both sides named, the reason quoted, and no pretense of a clean transfer.
func TestTransferOwnership_MidSequenceFailureHonest(t *testing.T) {
	t.Parallel()

	svc := mustService(t)
	tenant := mustCreateTenant(t, svc, "acme")
	owner := mustRegister(t, svc, "owner", "owner@acme.dev")
	target := mustRegister(t, svc, "target", "target@acme.dev")
	mustAddMember(t, svc, owner.User.ID, tenant.ID, identitymodel.RoleOwner)
	mustAddMember(t, svc, target.User.ID, tenant.ID, identitymodel.RoleViewer)

	// Roster snapshot taken BEFORE the prior owner's membership is removed
	// out-of-band — the decider will refuse the demotion.
	members := toMemberRows(svc.TenantMembers(context.Background(), tenant.ID), func(identitymodel.UserID) (*identitymodel.User, bool) {
		return nil, false
	})
	var prior *memberRow
	var targetActor identitymodel.ActorID
	var targetRoles []identitymodel.Role
	targetActor = identitymodel.ActorIDFromUser(target.User.ID)
	for i := range members {
		if hasRole(members[i].Roles, identitymodel.RoleOwner) {
			m := members[i]
			prior = &m
		}
		if members[i].Actor.PrefixedString() == targetActor.PrefixedString() {
			targetRoles = members[i].Roles
		}
	}
	// The membership disappears between snapshot and dispatch.
	if err := svc.RemoveMember(context.Background(), prior.Actor, tenant.ID); err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}

	out := executeOwnershipTransfer(context.Background(), svc, tenant.ID, targetActor, targetRoles, prior)
	if !out.GrantedOK {
		t.Fatalf("grant must land before the demotion fails: %+v", out)
	}
	if out.DemotedOK || out.DemoteErr == nil {
		t.Fatalf("demotion must fail: %+v", out)
	}

	msg := ownershipOutcomeMessage(out)
	for _, want := range []string{
		"Owner granted to",      // what landed
		"demotion of",           // what failed
		"STILL HOLDS the owner", // the honest residual state
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("mid-sequence message missing %q: %s", want, msg)
		}
	}
	if toastKindFor(out) != "err" {
		t.Error("a partial outcome must not toast as plain success")
	}

	// And the durable state matches the message: target owns, prior is gone.
	for _, m := range svc.TenantMembers(context.Background(), tenant.ID) {
		if m.ActorID.PrefixedString() == targetActor.PrefixedString() && !hasRole(m.Roles, identitymodel.RoleOwner) {
			t.Error("the target must hold owner after the landed grant")
		}
	}
}

// TestTransferOwnership_GrantOnlyWhenNoOwner pins the honest grant-only
// outcome: a tenant with no owner gains one and the message says nobody was
// demoted.
func TestTransferOwnership_GrantOnlyWhenNoOwner(t *testing.T) {
	t.Parallel()

	svc := mustService(t)
	tenant := mustCreateTenant(t, svc, "orphan")
	target := mustRegister(t, svc, "t1", "t1@acme.dev")
	mustAddMember(t, svc, target.User.ID, tenant.ID, identitymodel.RoleViewer)

	members := toMemberRows(svc.TenantMembers(context.Background(), tenant.ID), func(identitymodel.UserID) (*identitymodel.User, bool) {
		return nil, false
	})
	out := executeOwnershipTransfer(
		context.Background(), svc, tenant.ID,
		members[0].Actor, members[0].Roles, nil,
	)
	if !out.GrantedOK || out.DemotedOK || out.PriorExisted {
		t.Fatalf("grant-only outcome wrong: %+v", out)
	}
	if msg := ownershipOutcomeMessage(out); !strings.Contains(msg, "no prior owner to demote") {
		t.Errorf("grant-only message must say so: %s", msg)
	}
}

// rebuildServiceWithAuditLog returns a NEW service sharing nothing with svc
// but backed by an empty audit log — simulating a rotated audit window.
func rebuildServiceWithAuditLog(t *testing.T, _ *usermgmt.Service) *usermgmt.Service {
	t.Helper()
	svc2, err := usermgmt.NewService(usermgmt.ServiceConfig{AuditLog: usermgmt.NewAuditLog()})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return svc2
}
