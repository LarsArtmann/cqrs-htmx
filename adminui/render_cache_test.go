package adminui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// newCacheTestPanel seeds a small directory (N users, two tenants, one
// platform admin, one tenant admin, one phantom-domain user) and returns the
// panel, service, and the seeded users. The shape mirrors what the render
// caches must reproduce: direct grants, hierarchy edges in the same Casbin
// section, and registration phantom domains.
func newCacheTestPanel(t *testing.T) (*Handler, *usermgmt.Service, []*identitymodel.User) {
	t.Helper()
	ctx := context.Background()
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{AuditLog: usermgmt.NewAuditLog()})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	h, err := New(Config{Service: svc, Authorizer: RequireAuthenticated()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	users := make([]*identitymodel.User, 0, 4)
	users = append(users, mustUser(t, "admin@example.com"))
	for _, email := range []string{"ada@example.com", "bob@example.com", "carol@example.com"} {
		resp, err := svc.Register(ctx, usermgmt.RegisterRequest{
			ID:    identitymodel.SyntheticUserID("u-" + email),
			Email: email,
		})
		if err != nil {
			t.Fatalf("register %s: %v", email, err)
		}
		users = append(users, resp.User)
	}
	for _, name := range []string{"acme", "globex"} {
		if _, err := svc.CreateTenant(ctx, usermgmt.CreateTenantRequest{
			ID: identitymodel.NewTenantID(name), Name: name, DisplayName: name,
		}); err != nil {
			t.Fatalf("create tenant %s: %v", name, err)
		}
	}
	// ada: platform admin (wildcard domain) + acme member.
	if err := svc.GrantRoles(ctx, identitymodel.ActorIDFromUser(users[1].ID),
		identitymodel.NewTenantID(identitymodel.WildcardDomain),
		[]identitymodel.Role{identitymodel.RoleAdmin}); err != nil {
		t.Fatalf("grant platform roles: %v", err)
	}
	if err := svc.AddMember(ctx, identitymodel.ActorIDFromUser(users[1].ID),
		identitymodel.NewTenantID("acme"), []identitymodel.Role{identitymodel.RoleUser}); err != nil {
		t.Fatalf("add ada to acme: %v", err)
	}
	// bob: member of both tenants (no platform role).
	for _, name := range []string{"acme", "globex"} {
		if err := svc.AddMember(ctx, identitymodel.ActorIDFromUser(users[2].ID),
			identitymodel.NewTenantID(name), []identitymodel.Role{identitymodel.RoleViewer}); err != nil {
			t.Fatalf("add bob to %s: %v", name, err)
		}
	}
	// carol: no grants at all — only her registration phantom domain.
	return h, svc, users
}

// TestRenderAuthz_MatchesPerRowLookups is the gotcha-45 proof: for every
// seeded user, the one-pass snapshot must agree with the published per-row
// lookups (RolesForUser / DomainsForUser + tenant resolution) on platform
// roles and tenant count.
func TestRenderAuthz_MatchesPerRowLookups(t *testing.T) {
	_, svc, users := newCacheTestPanel(t)
	ctx := context.Background()
	az := newRenderAuthz(ctx, svc)

	for _, u := range users {
		id := u.ID.Get().String()
		assertPlatformRolesMatch(t, az, svc, u, id)
		assertTenantCountMatches(t, az, svc, ctx, u, id)
	}
}

// assertPlatformRolesMatch compares the snapshot's platform roles against the
// published per-row RolesForUser lookup, multiset-equal.
func assertPlatformRolesMatch(t *testing.T, az *renderAuthz, svc *usermgmt.Service, u *identitymodel.User, id string) {
	t.Helper()

	wantRoles, err := svc.Authz().RolesForUser(u.ID, identitymodel.NewTenantID(identitymodel.WildcardDomain))
	if err != nil {
		t.Fatalf("RolesForUser %s: %v", id, err)
	}
	gotRoles := az.platformRolesFor(id)
	if len(gotRoles) != len(wantRoles) {
		t.Errorf("%s: platform roles = %v, want %v", id, gotRoles, wantRoles)
		return
	}
	got := make(map[identitymodel.Role]int, len(gotRoles))
	for _, r := range gotRoles {
		got[r]++
	}
	for _, r := range wantRoles {
		got[r]--
	}
	for r, n := range got {
		if n != 0 {
			t.Errorf("%s: platform role %s mismatched (snapshot vs per-row)", id, r)
		}
	}
}

// assertTenantCountMatches compares the snapshot's tenant count against the
// per-row DomainsForUser lookup resolved through GetTenant.
func assertTenantCountMatches(
	t *testing.T,
	az *renderAuthz,
	svc *usermgmt.Service,
	ctx context.Context,
	u *identitymodel.User,
	id string,
) {
	t.Helper()

	wantCount := 0
	domains, err := svc.Authz().DomainsForUser(u.ID)
	if err != nil {
		t.Fatalf("DomainsForUser %s: %v", id, err)
	}
	for _, dom := range domains {
		if dom.Get() == identitymodel.WildcardDomain {
			continue
		}
		if _, err := svc.GetTenant(ctx, dom); err == nil {
			wantCount++
		}
	}
	if got := az.tenantCountFor(id); got != wantCount {
		t.Errorf("%s: tenant count = %d, want %d", id, got, wantCount)
	}
}

// TestRenderAuthz_ExcludesHierarchySubjects pins gotcha 45: the hierarchy
// edges seeding writes into the same Casbin section must never surface as a
// subject in the snapshot's maps — a role name is not a user.
func TestRenderAuthz_ExcludesHierarchySubjects(t *testing.T) {
	_, svc, _ := newCacheTestPanel(t)
	az := newRenderAuthz(context.Background(), svc)

	for subject := range az.platformRoles {
		if identitymodel.Role(subject).Valid() {
			t.Errorf("platformRoles contains role-named subject %q (hierarchy row leaked)", subject)
		}
	}
	for subject := range az.tenantDomains {
		if identitymodel.Role(subject).Valid() {
			t.Errorf("tenantDomains contains role-named subject %q (hierarchy row leaked)", subject)
		}
	}
	// Sanity: seeding two tenants must have produced hierarchy edges, so the
	// raw feed really did contain rows this filter had to skip.
	rows, err := svc.Authz().GroupPolicies()
	if err != nil {
		t.Fatalf("GroupPolicies: %v", err)
	}
	hierarchyRows := 0
	for _, row := range rows {
		if len(row) == 3 && identitymodel.Role(row[0]).Valid() {
			hierarchyRows++
		}
	}
	if hierarchyRows == 0 {
		t.Fatal("expected the seed to produce hierarchy edges; test proves nothing")
	}
}

// TestRenderAuthz_PlatformAdminFilter pins the admin chip's source of truth:
// exactly the platform-granted user survives, whatever their tenant roles.
func TestRenderAuthz_PlatformAdminFilter(t *testing.T) {
	h, svc, users := newCacheTestPanel(t)
	az := newRenderAuthz(context.Background(), svc)

	got := h.filterPlatformAdmins(svc.ReadModel().AllUsers(), az)
	if len(got) != 1 || got[0].ID != users[1].ID {
		t.Errorf("filterPlatformAdmins = %d users, want exactly ada (%s)", len(got), users[1].ID)
	}
}

// TestLastSeenIndex_TTLCache proves the 30s memoization through the clock
// seam: within the TTL the cached map is returned even after new audit
// entries land; past the TTL it rebuilds.
func TestLastSeenIndex_TTLCache(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, svc := newCacheTestPanelTTL(t, user)
	ctx := context.Background()

	now := time.Unix(1_000_000, 0)
	clock := func() time.Time { return now }
	h.clock = clock

	first := h.lastSeenIndex()
	if len(first) == 0 {
		t.Fatal("expected last-seen entries from registration audit events")
	}

	// New activity lands INSIDE the TTL: the cache must still serve the old map.
	if _, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: identitymodel.SyntheticUserID("u-late@example.com"), Email: "late@example.com",
	}); err != nil {
		t.Fatalf("register late user: %v", err)
	}
	now = now.Add(10 * time.Second)
	if got := h.lastSeenIndex(); len(got) != len(first) {
		t.Errorf("within TTL: map rebuilt (%d entries), want cached %d", len(got), len(first))
	}

	// Past the TTL: the map must include the new activity (keyed by the
	// user's canonical id, which the synthetic registration id maps to).
	now = now.Add(31 * time.Second)
	late, ok := svc.ReadModel().FindByEmail("late@example.com")
	if !ok {
		t.Fatal("late user missing from read model")
	}
	got := h.lastSeenIndex()
	if _, ok := got[late.ID.Get().String()]; !ok {
		t.Error("past TTL: cache did not pick up new audit activity")
	}
}

// TestLastSeenIndex_ConcurrentReaders exercises the mutex under -race.
func TestLastSeenIndex_ConcurrentReaders(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newCacheTestPanelTTL(t, user)

	done := make(chan struct{})
	for range 4 {
		go func() {
			for range 50 {
				_ = h.lastSeenIndex()
			}
			done <- struct{}{}
		}()
	}
	for range 4 {
		<-done
	}
}

// newCacheTestPanelTTL is a minimal panel (one user, fresh clock field) for
// the TTL tests, which control h.clock directly.
func newCacheTestPanelTTL(t *testing.T, user *identitymodel.User) (*Handler, *usermgmt.Service) {
	t.Helper()
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{AuditLog: usermgmt.NewAuditLog()})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	handler, err := New(Config{Service: svc, Authorizer: RequireAuthenticated()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := svc.Register(context.Background(), usermgmt.RegisterRequest{
		ID: user.ID, Email: user.Email,
	}); err != nil {
		t.Fatalf("register: %v", err)
	}
	return handler, svc
}

// TestTenantMemberCounts_BoundedToScope pins the F045 contract: the helper
// reports counts for exactly the tenants it was given (visible page vs full
// match is the caller's choice), and agrees with a direct TenantMembers read.
func TestTenantMemberCounts_BoundedToScope(t *testing.T) {
	h, svc, _ := newCacheTestPanel(t)
	ctx := context.Background()
	tenants := svc.AllTenants()
	if len(tenants) != 2 {
		t.Fatalf("AllTenants = %d, want 2", len(tenants))
	}
	counts := h.tenantMemberCounts(ctx, tenants)
	for _, ten := range tenants {
		want := len(svc.TenantMembers(ctx, ten.ID))
		if counts[ten.ID] != want {
			t.Errorf("%s: count = %d, want %d", ten.ID, counts[ten.ID], want)
		}
	}
	// A single-tenant scope reports only that tenant.
	only := h.tenantMemberCounts(ctx, tenants[:1])
	if len(only) != 1 || only[tenants[0].ID] == 0 {
		t.Errorf("bounded scope map = %v, want exactly one populated entry", only)
	}
}

// serveAs wraps a panel the way a consumer mounts it: session middleware
// injecting the user into the request context.
func serveAs(h *Handler, user *identitymodel.User) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/admin/", http.StripPrefix("/admin", h.Handler()))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r.WithContext(usermgmt.WithUser(r.Context(), user)))
	})
}

// TestUsersIndexData_RendersWithCaches renders the users page through the
// full handler with the caches in place — the golden-path smoke the cache
// wave must not regress.
func TestUsersIndexData_RendersWithCaches(t *testing.T) {
	h, _, _ := newCacheTestPanel(t)
	httpPanel := serveAs(h, mustUser(t, "admin@example.com"))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/users?filter=admin", nil)
	httpPanel.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
}
