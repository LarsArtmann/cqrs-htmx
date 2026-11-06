package adminui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/templ-components/display"
)

func TestAuditLabel_KnownAndUnknown(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"register":             "User registered",
		"session_started":      "Session started",
		"session_ended":        "Session ended",
		"display_name_changed": "Display name changed",
		"login_failed":         "Login failed",
		"weird_future_action":  "weird_future_action", // unknown passes through
	}
	for action, want := range cases {
		if got := auditLabel(action); got != want {
			t.Errorf("auditLabel(%q) = %q, want %q", action, got, want)
		}
	}
}

func TestAuditActionBadge_Tones(t *testing.T) {
	t.Parallel()

	cases := map[string]display.BadgeType{
		"user_deleted":    display.BadgeError,
		"login_failed":    display.BadgeWarning,
		"login_succeeded": display.BadgeSuccess,
		"session_started": display.BadgeNeutral,
		"register":        display.BadgePrimary,
	}
	for action, want := range cases {
		if got := auditActionBadge(action); got != want {
			t.Errorf("auditActionBadge(%q) = %v, want %v", action, got, want)
		}
	}
}

func TestFilterAuditEntries_Omnibox(t *testing.T) {
	ctx := context.Background()

	// Seed a real audit log so the aggregate ids are genuine StreamIDs.
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{AuditLog: usermgmt.NewAuditLog()})
	if err != nil {
		t.Fatal(err)
	}
	ada, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: identitymodel.SyntheticUserID("u-omni1"), Email: "omni1@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	bob, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: identitymodel.SyntheticUserID("u-omni2"), Email: "omni2@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.ChangeDisplayName(ctx, bob.User.ID, "Bob"); err != nil {
		t.Fatal(err)
	}

	al := svc.AuditLog()
	all := al.Entries()
	resolveAuditEmails(svc, all)
	if len(all) < 3 {
		t.Fatalf("seed produced %d entries, want >= 3", len(all))
	}

	if got := filterAuditEntries(all, ""); len(got) != len(all) {
		t.Errorf("empty needle should keep all, got %d of %d", len(got), len(all))
	}
	if got := filterAuditEntries(all, "omni1@"); len(got) != 1 {
		t.Errorf("email filter: got %d entries, want 1", len(got))
	}
	if got := filterAuditEntries(all, "Display name"); len(got) != 1 || got[0].Action != "display_name_changed" {
		t.Errorf("label-text filter: got %+v", got)
	}
	if got := filterAuditEntries(all, ada.User.ID.Get().String()); len(got) < 1 {
		t.Errorf("aggregate filter matched nothing for ada's id")
	}
	if got := filterAuditEntries(all, "nothing-matches-this"); len(got) != 0 {
		t.Errorf("no-match needle returned %d entries", len(got))
	}
}

func TestFlow_AuditFilterAndPartial(t *testing.T) {
	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	ada, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: identitymodel.SyntheticUserID("u-audit1"), Email: "audit1@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.ChangeDisplayName(ctx, ada.User.ID, "Renamed"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: identitymodel.SyntheticUserID("u-audit2"), Email: "audit2@example.com",
	}); err != nil {
		t.Fatal(err)
	}

	// Filtered page: only the display-name change matches.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/audit?q=Display+name", nil)
	h.ServeHTTP(rec, req)
	body := rec.Body.String()
	if !strings.Contains(body, "Display name changed") {
		t.Error("filtered audit page missing the matched label")
	}
	if strings.Contains(body, ">audit2@example.com<") {
		t.Error("unmatched register row leaked into filtered view")
	}

	// HTMX partial: fragment only, same filter.
	req2 := httptest.NewRequest(http.MethodGet, "/admin/audit?q=Display+name", nil)
	req2.Header.Set("HX-Request", "true")
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	part := rec2.Body.String()
	if strings.Contains(part, "<html") {
		t.Error("audit search must swap a fragment, not a full page")
	}
	if !strings.Contains(part, "Display name changed") {
		t.Error("audit partial missing filtered rows")
	}
}

func TestRender_AuditPageLabels(t *testing.T) {
	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	if _, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: identitymodel.SyntheticUserID("u-labels"), Email: "labels@example.com",
	}); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/audit", nil))
	body := rec.Body.String()
	if !strings.Contains(body, "User registered") {
		t.Error("audit page must render the human label, not the raw action")
	}
	if !strings.Contains(body, `placeholder="Search actions, people, aggregates`) {
		t.Error("audit page missing the omnibox")
	}
}

func TestRender_DashboardAdoptionStatsAndBanner(t *testing.T) {
	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	// No admin grant anywhere: the bootstrap warning must be visible.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/", nil))
	body := rec.Body.String()
	if !strings.Contains(body, "No admin grant exists") {
		t.Error("dashboard missing the no-admin-grant warning")
	}
	for _, want := range []string{"Verified", "MFA adoption", "Suspended tenants"} {
		if !strings.Contains(body, want) {
			t.Errorf("dashboard missing the %q stat card", want)
		}
	}

	// Grant an admin: the warning disappears.
	if err := svc.GrantRoles(ctx,
		usermgmt.ActorIDFromUser(user.ID),
		identitymodel.NewTenantID(identitymodel.WildcardDomain),
		[]identitymodel.Role{identitymodel.RoleSuperAdmin},
	); err != nil {
		t.Fatal(err)
	}
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/admin/", nil))
	if strings.Contains(rec2.Body.String(), "No admin grant exists") {
		t.Error("no-admin-grant warning should disappear once a grant exists")
	}
}

func TestRender_DashboardEntityLinks(t *testing.T) {
	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	target, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: identitymodel.SyntheticUserID("u-link"), Email: "link@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/", nil))
	want := `href="/admin/users/` + target.User.ID.Get().String() + `"`
	if !strings.Contains(rec.Body.String(), want) {
		t.Errorf("recent activity must link the entity, missing %q", want)
	}
}

func TestRender_UserTimelineCard(t *testing.T) {
	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	target, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: identitymodel.SyntheticUserID("u-timeline"), Email: "timeline@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.ChangeDisplayName(ctx, target.User.ID, "Timeline User"); err != nil {
		t.Fatal(err)
	}

	body := renderUserDetail(t, h, target.User.ID.Get().String())
	if !strings.Contains(body, "Activity") {
		t.Error("user detail missing the activity card")
	}
	if !strings.Contains(body, "User registered") || !strings.Contains(body, "Display name changed") {
		t.Error("user timeline missing the seeded events")
	}
}
