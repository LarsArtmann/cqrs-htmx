package adminui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// seedDirectory tenants + members across two tenants: alice (admin+viewer in
// acme), bob (viewer in acme AND owner in globex) — the multi-tenant row the
// directory exists to surface.
func seedDirectory(t *testing.T, svc *usermgmt.Service) {
	t.Helper()

	mustCreateTenant(t, svc, "acme")
	mustCreateTenant(t, svc, "globex")
	alice := mustRegister(t, svc, "u-alice", "alice@acme.com")
	bob := mustRegister(t, svc, "u-bob", "bob@globex.com")
	mustAddMember(t, svc, alice.User.ID, identitymodel.NewTenantID("acme"),
		identitymodel.RoleAdmin, identitymodel.RoleViewer)
	mustAddMember(t, svc, bob.User.ID, identitymodel.NewTenantID("acme"), identitymodel.RoleViewer)
	mustAddMember(t, svc, bob.User.ID, identitymodel.NewTenantID("globex"), identitymodel.RoleOwner)
}

func TestPanel_MembersDirectoryRenders(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "root@example.com")
	h, svc := newTestPanel(t, user)
	seedDirectory(t, svc)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/members", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("directory: status %d, body %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{"Membership directory", "alice@acme.com", "bob@globex.com", "globex"} {
		if !strings.Contains(body, want) {
			t.Errorf("directory page missing %q", want)
		}
	}
	// Roles render as chips, joined column present.
	if !strings.Contains(body, "owner") {
		t.Error("directory page missing role chip")
	}
}

func TestPanel_MembersDirectoryEmpty(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "root@example.com")
	h, _ := newTestPanel(t, user)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/members", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("empty directory: status %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "No memberships yet") {
		t.Error("empty directory must show the explicit empty note, not a blank table")
	}
}

func TestPanel_MembersDirectoryFilteredEmpty(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "root@example.com")
	h, svc := newTestPanel(t, user)
	seedDirectory(t, svc)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/members?q=zebra", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("filtered empty: status %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "No memberships match your filters") {
		t.Error("filtered-empty directory must say so explicitly")
	}
}

func TestPanel_MembersDirectorySearch(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "root@example.com")
	h, svc := newTestPanel(t, user)
	seedDirectory(t, svc)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/members?q=alice", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("search: status %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "alice@acme.com") {
		t.Error("search should keep alice")
	}
	if strings.Contains(body, "bob@globex.com") {
		t.Error("search should drop bob")
	}
}

func TestPanel_MembersDirectoryTenantFilter(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "root@example.com")
	h, svc := newTestPanel(t, user)
	seedDirectory(t, svc)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/members?tenant=globex", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("tenant filter: status %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "bob@globex.com") {
		t.Error("globex filter should keep bob")
	}
	if strings.Contains(body, "alice@acme.com") {
		t.Error("globex filter should drop alice")
	}
}

func TestPanel_MembersDirectoryRoleFilter(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "root@example.com")
	h, svc := newTestPanel(t, user)
	seedDirectory(t, svc)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/members?role=owner", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("role filter: status %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "bob@globex.com") {
		t.Error("owner filter should keep bob")
	}
	if strings.Contains(body, "alice@acme.com") {
		t.Error("owner filter should drop alice")
	}
}

func TestPanel_MembersDirectoryUnknownRoleDegrades(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "root@example.com")
	h, svc := newTestPanel(t, user)
	seedDirectory(t, svc)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/members?role=wild", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("unknown role: status %d", rec.Code)
	}
	// An unknown chip value degrades to the unfiltered directory, not an
	// empty table that would read as "no memberships exist".
	if !strings.Contains(rec.Body.String(), "alice@acme.com") {
		t.Error("unknown role filter must degrade to unfiltered")
	}
}

func TestPanel_MembersDirectoryPager(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "root@example.com")
	h, svc := newTestPanel(t, user, Config{PageSize: MinPageSize})
	for i := 0; i < 12; i++ {
		mustCreateTenant(t, svc, "t"+string(rune('a'+i)))
		u := mustRegister(t, svc, "u-pager"+string(rune('a'+i)), "pager"+string(rune('a'+i))+"@x.com")
		mustAddMember(t, svc, u.User.ID, identitymodel.NewTenantID("t"+string(rune('a'+i))), identitymodel.RoleUser)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/members", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("pager page 1: status %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Showing page") || !strings.Contains(body, "pagera@x.com") {
		t.Error("page 1 should show the pager and the first rows")
	}
	if strings.Contains(body, "pagerl@x.com") {
		t.Error("page 1 must not show rows beyond the page size")
	}

	// Page 2 keeps the state and shows the tail.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/members?page=2", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("pager page 2: status %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "pagerl@x.com") {
		t.Error("page 2 should show the tail rows")
	}
}

func TestPanel_MembersDirectoryModeAbsence(t *testing.T) {
	t.Parallel()

	// Tenant-admin mode mounts the tenant ROSTER at /members, not the
	// directory: the directory route row is super-admin-scoped, so in tenant
	// mode it is served by NO mode and the roster answers instead.
	user := mustUser(t, "ops@acme.com")
	h, svc := newTenantAdminPanel(t, user, "acme")
	mustCreateTenant(t, svc, "acme")
	member := mustRegister(t, svc, "u-roster", "roster@acme.com")
	mustAddMember(t, svc, member.User.ID, identitymodel.NewTenantID("acme"), identitymodel.RoleUser)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/members", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("tenant roster: status %d", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, "Membership directory") {
		t.Error("tenant mode must NOT serve the global directory at /members")
	}
	if !strings.Contains(body, "roster@acme.com") {
		t.Error("tenant mode /members should render the tenant roster")
	}
}

func TestPanel_MembersDirectoryURLBuilder(t *testing.T) {
	t.Parallel()

	d := membersDirectoryData{
		Search: "alice", TenantFilter: "acme", RoleFilter: "admin", Sort: "joined",
		BasePath: "",
	}
	got := membersDirectoryURL("", d)
	want := "/members?" + url.Values{
		"q": []string{"alice"}, "tenant": []string{"acme"}, "role": []string{"admin"}, "sort": []string{"joined"},
	}.Encode()
	if got != want {
		t.Errorf("membersDirectoryURL = %q, want %q", got, want)
	}
	if plain := membersDirectoryURL("/admin", membersDirectoryData{BasePath: "/admin"}); plain != "/admin/members" {
		t.Errorf("plain URL = %q, want /admin/members", plain)
	}
}
