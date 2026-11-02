package usermgmt

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// globalDomain is the wildcard authorization domain — the one the admin panel
// checks for super_admin/admin.
func globalDomain() TenantID {
	return NewTenantID(identitymodel.WildcardDomain)
}

func testActor(seed string) ActorID {
	return ActorIDFromUser(NewUserID(strings.Repeat(seed, 26)))
}

// waitForAdminGrant polls HasAdminGrant until it reports true or the deadline
// passes. The projection is a separate goroutine, so a bounded poll keeps this
// honest without racing it.
func waitForAdminGrant(t *testing.T, svc *Service, domain TenantID) bool {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)

	for {
		has, err := svc.HasAdminGrant(domain)
		if err != nil {
			t.Fatalf("HasAdminGrant: %v", err)
		}

		if has {
			return true
		}

		if time.Now().After(deadline) {
			return false
		}

		time.Sleep(10 * time.Millisecond)
	}
}

// openGrantsJournal opens one process lifetime over the file-backed journal at
// dbPath. initSchema creates the event tables on the first lifetime; later
// lifetimes find them. The caller owns closing both the database and the
// service built on the returned store.
func openGrantsJournal(t *testing.T, dbPath string, initSchema bool) (*sql.DB, event.Store) {
	t.Helper()

	db, store, _ := openRestartPhase(t, dbPath)

	if initSchema {
		if err := storage.SQLiteInitSchema(context.Background(), db); err != nil {
			t.Fatalf("SQLiteInitSchema: %v", err)
		}
	}

	return db, store
}

// TestGrantRoles_GlobalAdminSurvivesRestart is the durability proof for the
// defect that motivated Service.GrantRoles: a grant written through the
// in-memory enforcer (AddGroupPolicy) disappears when the process restarts,
// because every restart rebuilds the policy set from the journal. GrantRoles
// dispatches a Membership command, so the grant IS in the journal and the
// second lifetime reconstructs it with zero consumer code.
func TestGrantRoles_GlobalAdminSurvivesRestart(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "grants.db")
	actor := testActor("a")

	db1, store1 := openGrantsJournal(t, dbPath, true)

	svc1, err := NewService(ServiceConfig{EventStore: store1})
	if err != nil {
		t.Fatalf("NewService lifetime 1: %v", err)
	}

	if err := svc1.GrantRoles(ctx, actor, globalDomain(), []Role{RoleSuperAdmin}); err != nil {
		t.Fatalf("GrantRoles: %v", err)
	}

	if !waitForAdminGrant(t, svc1, globalDomain()) {
		t.Fatal("grant is not visible in lifetime 1 — the projection never applied MemberAdded")
	}

	if err := svc1.Close(); err != nil {
		t.Fatalf("Close lifetime 1: %v", err)
	}

	if err := db1.Close(); err != nil {
		t.Fatalf("Close db lifetime 1: %v", err)
	}

	db2, store2 := openGrantsJournal(t, dbPath, false)
	defer db2.Close() //nolint:errcheck // test cleanup

	svc2, err := NewService(ServiceConfig{EventStore: store2})
	if err != nil {
		t.Fatalf("NewService lifetime 2: %v", err)
	}
	defer svc2.Close() //nolint:errcheck // test cleanup

	if !waitForAdminGrant(t, svc2, globalDomain()) {
		t.Fatal("global admin grant did not survive the restart — GrantRoles is not durable")
	}

	roles, err := svc2.Authz().RolesForUser(NewUserID(strings.Repeat("a", 26)), globalDomain())
	if err != nil {
		t.Fatalf("RolesForUser: %v", err)
	}

	if len(roles) != 1 || roles[0] != RoleSuperAdmin {
		t.Fatalf("restored roles = %v, want [super_admin]", roles)
	}
}

// TestAddGroupPolicy_DoesNotSurviveRestart pins the contract behind the
// deprecation: the in-memory write path is ephemeral by construction. If this
// test ever starts failing, the deprecation is no longer justified.
func TestAddGroupPolicy_DoesNotSurviveRestart(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ephemeral.db")
	actorID := NewUserID(strings.Repeat("b", 26))

	db1, store1 := openGrantsJournal(t, dbPath, true)

	svc1, err := NewService(ServiceConfig{EventStore: store1})
	if err != nil {
		t.Fatalf("NewService lifetime 1: %v", err)
	}

	//nolint:staticcheck // the deprecation is exactly what this test pins
	if err := svc1.Authz().AddGroupPolicy(GroupPolicy{
		Subject: actorID.Get().String(),
		Role:    RoleSuperAdmin,
		Domain:  identitymodel.WildcardDomain,
	}); err != nil {
		t.Fatalf("AddGroupPolicy: %v", err)
	}

	if !waitForAdminGrant(t, svc1, globalDomain()) {
		t.Fatal("AddGroupPolicy did not take effect in lifetime 1")
	}

	if err := svc1.Close(); err != nil {
		t.Fatalf("Close lifetime 1: %v", err)
	}

	if err := db1.Close(); err != nil {
		t.Fatalf("Close db lifetime 1: %v", err)
	}

	db2, store2 := openGrantsJournal(t, dbPath, false)
	defer db2.Close() //nolint:errcheck // test cleanup

	svc2, err := NewService(ServiceConfig{EventStore: store2})
	if err != nil {
		t.Fatalf("NewService lifetime 2: %v", err)
	}
	defer svc2.Close() //nolint:errcheck // test cleanup

	has, err := svc2.HasAdminGrant(globalDomain())
	if err != nil {
		t.Fatalf("HasAdminGrant: %v", err)
	}

	if has {
		t.Fatal("AddGroupPolicy grant survived a restart — it is supposed to be ephemeral")
	}
}

// TestGrantRoles_IsConvergent covers the second call: a membership already
// exists, so GrantRoles must replace the role set instead of surfacing the
// AddMember conflict.
func TestGrantRoles_IsConvergent(t *testing.T) {
	ctx := context.Background()

	svc, err := NewService(ServiceConfig{EventStore: memory.NewMemoryStore()})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	defer svc.Close() //nolint:errcheck // test cleanup

	actor := testActor("c")

	if err := svc.GrantRoles(ctx, actor, globalDomain(), []Role{RoleAdmin}); err != nil {
		t.Fatalf("GrantRoles (create): %v", err)
	}

	if err := svc.GrantRoles(ctx, actor, globalDomain(), []Role{RoleSuperAdmin}); err != nil {
		t.Fatalf("GrantRoles (replace): %v", err)
	}

	roles, err := svc.Authz().RolesForUser(NewUserID(strings.Repeat("c", 26)), globalDomain())
	if err != nil {
		t.Fatalf("RolesForUser: %v", err)
	}

	if len(roles) != 1 || roles[0] != RoleSuperAdmin {
		t.Fatalf("roles after replace = %v, want [super_admin] only", roles)
	}
}

// TestGrantRoles_RejectsUnknownRole keeps junk out of the journal: the
// projection would otherwise write a policy row for a role nothing ever checks.
func TestGrantRoles_RejectsUnknownRole(t *testing.T) {
	svc, err := NewService(ServiceConfig{EventStore: memory.NewMemoryStore()})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	defer svc.Close() //nolint:errcheck // test cleanup

	err = svc.GrantRoles(context.Background(), testActor("d"), globalDomain(), []Role{"administrator"})
	if err == nil {
		t.Fatal("expected a rejection for an unknown role")
	}

	if fam := errorfamily.Classify(err); fam != errorfamily.Rejection {
		t.Fatalf("error family = %v, want Rejection", fam)
	}

	if code := errorfamily.Code(err); code != "usermgmt.grant.unknown_role" {
		t.Fatalf("error code = %q, want usermgmt.grant.unknown_role", code)
	}
}

// TestRevokeRoles_RemovesGrant proves the removal path is as durable as the
// grant: the projection drops the policy rows when it replays MemberRemoved.
func TestRevokeRoles_RemovesGrant(t *testing.T) {
	ctx := context.Background()

	svc, err := NewService(ServiceConfig{EventStore: memory.NewMemoryStore()})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	defer svc.Close() //nolint:errcheck // test cleanup

	actor := testActor("e")

	if err := svc.GrantRoles(ctx, actor, globalDomain(), []Role{RoleAdmin}); err != nil {
		t.Fatalf("GrantRoles: %v", err)
	}

	if !waitForAdminGrant(t, svc, globalDomain()) {
		t.Fatal("grant is not visible before revoke")
	}

	if err := svc.RevokeRoles(ctx, actor, globalDomain()); err != nil {
		t.Fatalf("RevokeRoles: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)

	for {
		has, err := svc.HasAdminGrant(globalDomain())
		if err != nil {
			t.Fatalf("HasAdminGrant: %v", err)
		}

		if !has {
			return
		}

		if time.Now().After(deadline) {
			t.Fatal("revoked grant is still visible")
		}

		time.Sleep(10 * time.Millisecond)
	}
}

// TestHasAdminGrant_CountsBothAdminTiers pins the vocabulary: "admin" is an
// admin-tier role, not just "super_admin" — the admin panel's default
// authorizer accepts either.
func TestHasAdminGrant_CountsBothAdminTiers(t *testing.T) {
	for _, role := range identitymodel.AdminRoles() {
		t.Run(string(role), func(t *testing.T) {
			svc, err := NewService(ServiceConfig{EventStore: memory.NewMemoryStore()})
			if err != nil {
				t.Fatalf("NewService: %v", err)
			}
			defer svc.Close() //nolint:errcheck // test cleanup

			if err := svc.GrantRoles(
				context.Background(), testActor("f"), globalDomain(), []Role{role},
			); err != nil {
				t.Fatalf("GrantRoles(%s): %v", role, err)
			}

			if !waitForAdminGrant(t, svc, globalDomain()) {
				t.Fatalf("%s in the wildcard domain was not recognised as an admin grant", role)
			}
		})
	}
}

// TestHasAdminGrant_IgnoresRoleHierarchyRows is the regression for the trap the
// first draft of this check fell into: the Casbin `g` section also holds the
// mirrored role-hierarchy edges (super_admin->admin, admin->user, user->viewer)
// written by ensureRoleHierarchy, so a naive scan sees "somebody holds admin"
// the moment any grant seeds a domain's hierarchy.
func TestHasAdminGrant_IgnoresRoleHierarchyRows(t *testing.T) {
	svc, err := NewService(ServiceConfig{EventStore: memory.NewMemoryStore()})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	defer svc.Close() //nolint:errcheck // test cleanup

	// Seed the hierarchy in the wildcard domain without granting anybody a role.
	if err := svc.Authz().EnsureRoleHierarchy(globalDomain()); err != nil {
		t.Fatalf("EnsureRoleHierarchy: %v", err)
	}

	has, err := svc.HasAdminGrant(globalDomain())
	if err != nil {
		t.Fatalf("HasAdminGrant: %v", err)
	}

	if has {
		t.Fatal("role-hierarchy rows were mistaken for an admin grant")
	}
}

// TestHasAdminGrant_IgnoresOtherDomains proves the check is domain-scoped: an
// admin of one tenant must not count as a global admin.
func TestHasAdminGrant_IgnoresOtherDomains(t *testing.T) {
	svc, err := NewService(ServiceConfig{EventStore: memory.NewMemoryStore()})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	defer svc.Close() //nolint:errcheck // test cleanup

	tenant := NewTenantID("01JXTENANTGRANTS0000000001")

	if err := svc.GrantRoles(context.Background(), testActor("g"), tenant, []Role{RoleAdmin}); err != nil {
		t.Fatalf("GrantRoles: %v", err)
	}

	has, err := svc.HasAdminGrant(globalDomain())
	if err != nil {
		t.Fatalf("HasAdminGrant: %v", err)
	}

	if has {
		t.Fatal("a tenant-scoped admin must not satisfy the global admin check")
	}
}
