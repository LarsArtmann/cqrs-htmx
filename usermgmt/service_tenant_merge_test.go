package usermgmt

import (
	"context"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

func tenantTestID(n int) TenantID {
	digits := []byte("0123456789ABCDEF")
	return NewTenantID("01M47TENANT0000000000000" + string(digits[n]))
}

func createTenantTestTenant(ctx context.Context, t *testing.T, svc *Service, n int, name string) *Tenant {
	t.Helper()
	tenant, err := svc.CreateTenant(ctx, CreateTenantRequest{
		ID:   tenantTestID(n),
		Name: name,
	})
	if err != nil {
		t.Fatalf("CreateTenant %d (%q): %v", n, name, err)
	}
	return tenant
}

// TestService_RenameTenant proves issue #47's rename path end to end: the
// label moves, the ID (and with it every tenant-scoped key) stays.
func TestService_RenameTenant(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()
	tenant := createTenantTestTenant(ctx, t, svc, 0, "acme-corp")

	if err := svc.RenameTenant(ctx, tenant.ID, "Acme Corporation"); err != nil {
		t.Fatalf("RenameTenant: %v", err)
	}
	got, err := svc.GetTenant(ctx, tenant.ID)
	if err != nil {
		t.Fatalf("GetTenant: %v", err)
	}
	if got.DisplayName != "Acme Corporation" {
		t.Errorf("DisplayName = %q, want the new label", got.DisplayName)
	}
	if got.Name != "acme-corp" || got.ID.Get() != tenant.ID.Get() {
		t.Errorf("identity moved: %+v", got)
	}

	// Same value again is the decider's no-op, not an error.
	if err := svc.RenameTenant(ctx, tenant.ID, "Acme Corporation"); err != nil {
		t.Fatalf("same-value RenameTenant: %v", err)
	}
}

// TestService_MergeTenants_Issue47Scenario is the wave's second kill switch:
// duplicate tenants (same normalized name, disjoint rosters) collapse into
// one live tenant whose roster is the union, and a re-run changes nothing.
func TestService_MergeTenants_Issue47Scenario(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	survivor := createTenantTestTenant(ctx, t, svc, 0, "Acme Corp")
	loser := createTenantTestTenant(ctx, t, svc, 1, " acme corp ")

	keep := registerMergeTestUser(t, svc, 0, "keep@example.com")
	other := registerMergeTestUser(t, svc, 1, "other@example.com")
	if err := svc.AddMember(ctx, ActorIDFromUser(keep), survivor.ID, []Role{RoleUser}); err != nil {
		t.Fatalf("AddMember survivor: %v", err)
	}
	if err := svc.AddMember(ctx, ActorIDFromUser(other), loser.ID, []Role{RoleAdmin}); err != nil {
		t.Fatalf("AddMember loser: %v", err)
	}
	if err := svc.AddMember(ctx, ActorIDFromUser(keep), loser.ID, []Role{RoleViewer}); err != nil {
		t.Fatalf("AddMember loser keep: %v", err)
	}

	if err := svc.MergeTenants(ctx, survivor.ID, []TenantID{loser.ID}, "issue #47: duplicate tenants"); err != nil {
		t.Fatalf("MergeTenants: %v", err)
	}

	if got := len(svc.tenantReadModel.All()); got != 1 {
		t.Fatalf("live tenant count = %d, want 1", got)
	}
	got, err := svc.GetTenant(ctx, loser.ID)
	if err != nil {
		t.Fatalf("GetTenant(loser): %v", err)
	}
	if got.ID.Get() != survivor.ID.Get() {
		t.Errorf("GetTenant(loser) = %s, want the survivor via the pointer", got.ID.Get())
	}

	memberRoles := map[string][]Role{}
	for _, mem := range svc.TenantMembers(ctx, survivor.ID) {
		memberRoles[mem.ActorID.String()] = mem.Roles
	}
	if !roleSetHas(memberRoles[ActorIDFromUser(keep).String()], RoleUser) ||
		!roleSetHas(memberRoles[ActorIDFromUser(keep).String()], RoleViewer) {
		t.Errorf("keep's roles = %v, want the union [user viewer]", memberRoles)
	}
	if !roleSetHas(memberRoles[ActorIDFromUser(other).String()], RoleAdmin) {
		t.Errorf("other's roles = %v, want admin inherited", memberRoles)
	}
	if rows := svc.TenantMembers(ctx, loser.ID); len(rows) != 0 {
		t.Errorf("loser roster = %d rows, want 0", len(rows))
	}

	// The idempotent re-run: nothing moves, nothing errors.
	survivorAgg, err := aggIDFromTenant(survivor.ID)
	if err != nil {
		t.Fatalf("aggIDFromTenant: %v", err)
	}
	_, versionBefore, err := svc.tenantRepo.LoadRef(ctx, id.NewStreamRef(aggregateTypeTenant, survivorAgg))
	if err != nil {
		t.Fatalf("LoadRef: %v", err)
	}
	if err := svc.MergeTenants(ctx, survivor.ID, []TenantID{loser.ID}, "again"); err != nil {
		t.Fatalf("re-run MergeTenants: %v", err)
	}
	_, versionAfter, err := svc.tenantRepo.LoadRef(ctx, id.NewStreamRef(aggregateTypeTenant, survivorAgg))
	if err != nil {
		t.Fatalf("LoadRef after: %v", err)
	}
	if versionAfter != versionBefore {
		t.Errorf("survivor version moved %v -> %v on the re-run", versionBefore, versionAfter)
	}
}

// TestService_MergeTenants_Guards covers the input and terminal rejections.
func TestService_MergeTenants_Guards(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	a := createTenantTestTenant(ctx, t, svc, 0, "a")
	b := createTenantTestTenant(ctx, t, svc, 1, "b")
	c := createTenantTestTenant(ctx, t, svc, 2, "c")

	_ = requireTenantServiceError(t, svc.MergeTenants(ctx, TenantID{}, []TenantID{b.ID}, "x"),
		errorfamily.Rejection, "usermgmt.merge_tenants.survivor_required")
	_ = requireTenantServiceError(t, svc.MergeTenants(ctx, a.ID, nil, "x"),
		errorfamily.Rejection, "usermgmt.merge_tenants.no_losers")
	_ = requireTenantServiceError(t, svc.MergeTenants(ctx, a.ID, []TenantID{a.ID}, "x"),
		errorfamily.Rejection, "usermgmt.merge_tenants.self_merge")
	_ = requireTenantServiceError(t, svc.MergeTenants(ctx, a.ID, []TenantID{b.ID, b.ID}, "x"),
		errorfamily.Rejection, "usermgmt.merge_tenants.duplicate_loser")
	_ = requireTenantServiceError(t,
		svc.MergeTenants(ctx, a.ID, []TenantID{NewTenantID("01M47GHOST00000000000000Z")}, "x"),
		errorfamily.Rejection, "usermgmt.merge_tenants.loser_not_found")

	if err := svc.MergeTenants(ctx, a.ID, []TenantID{b.ID}, "chain"); err != nil {
		t.Fatalf("merge b into a: %v", err)
	}
	ef := requireTenantServiceError(t, svc.MergeTenants(ctx, b.ID, []TenantID{c.ID}, "x"),
		errorfamily.Conflict, "usermgmt.tenant_merge.survivor_merged")
	if ef.ErrorContext()["merged_into"] != a.ID.Get() {
		t.Errorf("merged_into = %q, want %q", ef.ErrorContext()["merged_into"], a.ID.Get())
	}
	ef = requireTenantServiceError(t, svc.MergeTenants(ctx, c.ID, []TenantID{b.ID}, "x"),
		errorfamily.Conflict, "usermgmt.merge_tenants.loser_merged")
	if ef.ErrorContext()["merged_into"] != a.ID.Get() {
		t.Errorf("merged_into = %q, want %q", ef.ErrorContext()["merged_into"], a.ID.Get())
	}

	if err := svc.DeleteTenant(ctx, c.ID, "gone"); err != nil {
		t.Fatalf("DeleteTenant: %v", err)
	}
	_ = requireTenantServiceError(t, svc.MergeTenants(ctx, a.ID, []TenantID{c.ID}, "x"),
		errorfamily.Conflict, "usermgmt.tenant_merge_into.deleted")
}

func requireTenantServiceError(t *testing.T, err error, family errorfamily.Family, code string) *errorfamily.Error {
	t.Helper()
	return requireDomainError(t, err, family, code)
}

// TestService_UniqueTenantNames covers the D5 knob: off by default, on it
// refuses normalized-name collisions and releases the name on delete.
func TestService_UniqueTenantNames(t *testing.T) {
	t.Run("off by default", func(t *testing.T) {
		ctx := context.Background()
		svc := setupTestService(t)
		createTenantTestTenant(ctx, t, svc, 0, "Acme Corp")
		createTenantTestTenant(ctx, t, svc, 1, " acme corp ")
		if groups := svc.DuplicateTenants(ctx); len(groups) != 1 || len(groups[0].Tenants) != 2 {
			t.Errorf("DuplicateTenants = %+v, want one group of two", groups)
		}
	})

	t.Run("on refuses collisions and names the existing tenant", func(t *testing.T) {
		svc, err := NewService(ServiceConfig{UniqueTenantNames: true})
		if err != nil {
			t.Fatalf("NewService: %v", err)
		}
		t.Cleanup(svc.Stop)
		ctx := context.Background()
		existing := createTenantTestTenant(ctx, t, svc, 0, "Acme Corp")

		ef := requireTenantServiceError(t, createTenantErr(ctx, svc, tenantTestID(1), " acme corp "),
			errorfamily.Conflict, "usermgmt.tenant.name_exists")
		if ef.ErrorContext()["existing_tenant_id"] != existing.ID.Get() {
			t.Errorf("existing_tenant_id = %q, want %q",
				ef.ErrorContext()["existing_tenant_id"], existing.ID.Get())
		}

		// A different name still passes.
		createTenantTestTenant(ctx, t, svc, 2, "Globex")
	})

	t.Run("name released after delete", func(t *testing.T) {
		svc, err := NewService(ServiceConfig{UniqueTenantNames: true})
		if err != nil {
			t.Fatalf("NewService: %v", err)
		}
		t.Cleanup(svc.Stop)
		ctx := context.Background()
		createTenantTestTenant(ctx, t, svc, 0, "Acme Corp")
		if err := svc.DeleteTenant(ctx, tenantTestID(0), "cleanup"); err != nil {
			t.Fatalf("DeleteTenant: %v", err)
		}
		createTenantTestTenant(ctx, t, svc, 1, "acme corp")
	})
}

func createTenantErr(ctx context.Context, svc *Service, id TenantID, name string) error {
	_, err := svc.CreateTenant(ctx, CreateTenantRequest{ID: id, Name: name})
	return err
}

// registerMergeTestUserDirect dispatches the RegisterUser command without the
// service-level duplicate-email guard — the shape of legacy duplicate rows.
func registerMergeTestUserDirect(ctx context.Context, t *testing.T, svc *Service, n int, email string) UserID {
	t.Helper()
	userID := mergeTestUserID(n)
	aggID, err := aggIDFromUser(userID)
	if err != nil {
		t.Fatalf("aggIDFromUser: %v", err)
	}
	if err := svc.dispatcher.Dispatch(ctx,
		identitymodel.NewRegisterUserCmd(aggID, email, "", nil)); err != nil {
		t.Fatalf("direct Register: %v", err)
	}
	return userID
}

// TestService_DuplicateUsers covers the report the admin users page renders:
// email clusters and external-account clusters, one person each.
func TestService_DuplicateUsers(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	a := registerMergeTestUser(t, svc, 0, "human@example.com")
	// The service-level duplicate-email guard canonicalizes (ParseEmail
	// trims and lowercases), so an email cluster cannot be born through
	// Register — it arrives as legacy rows. Dispatch the registration
	// directly, the way pre-guard data replays.
	b := registerMergeTestUserDirect(ctx, t, svc, 1, "Human@example.com")
	c := registerMergeTestUser(t, svc, 2, "distinct@example.com")
	d := registerMergeTestUser(t, svc, 3, "other@example.com")

	linkMergeTestAccount(t, svc, c, "github", "shared-subj")
	linkMergeTestAccount(t, svc, d, "github", "shared-subj")

	groups := svc.DuplicateUsers(ctx)
	wantPairs := map[string]int{
		"human@example.com":  2,
		"github/shared-subj": 2,
	}
	if len(groups) != len(wantPairs) {
		t.Fatalf("DuplicateUsers = %+v, want %d groups", groups, len(wantPairs))
	}
	seen := map[string]int{}
	for _, group := range groups {
		key := group.Key
		if group.Account != "" {
			key = group.Account
		}
		seen[key] = len(group.Users)
	}
	for key, want := range wantPairs {
		if seen[key] != want {
			t.Errorf("group %q has %d users, want %d", key, seen[key], want)
		}
	}

	// After the merge the clusters collapse.
	if err := svc.MergeUsers(ctx, a, []UserID{b}, "dedupe"); err != nil {
		t.Fatalf("MergeUsers: %v", err)
	}
	if err := svc.MergeUsers(ctx, c, []UserID{d}, "dedupe"); err != nil {
		t.Fatalf("MergeUsers 2: %v", err)
	}
	if groups := svc.DuplicateUsers(ctx); len(groups) != 0 {
		t.Errorf("DuplicateUsers after merges = %+v, want none", groups)
	}
}
