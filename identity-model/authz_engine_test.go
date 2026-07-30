package identitymodel

import (
	"errors"
	"slices"
	"testing"
)

func newTestAuthz(t *testing.T) *Authz {
	t.Helper()
	a, err := NewAuthz()
	if err != nil {
		t.Fatalf("NewAuthz: %v", err)
	}
	return a
}

// userActorPair returns a deterministic UserID and a matching ActorID for the
// same raw value. The UserID string is what Casbin stores as the group-policy
// subject; the ActorID uses the raw value so RolesForActor hashes it to the
// same UserID string.
func userActorPair(raw string) (UserID, ActorID) {
	uid := NewUserID(raw)
	return uid, NewActorID(ActorUser, raw)
}

func TestAuthz_Enforce(t *testing.T) {
	a := newTestAuthz(t)

	uid, _ := userActorPair("u1")
	uid2, _ := userActorPair("u2")
	uid3, _ := userActorPair("u3")

	// Seed: u1 is super_admin in tenant-a, u3 is admin in tenant-a.
	if err := a.AddGroupPolicy(
		GroupPolicy{Subject: uid.String(), Role: RoleSuperAdmin, Domain: "tenant-a"},
	); err != nil {
		t.Fatalf("add super admin: %v", err)
	}
	if err := a.AddGroupPolicy(GroupPolicy{Subject: uid3.String(), Role: RoleAdmin, Domain: "tenant-a"}); err != nil {
		t.Fatalf("add admin: %v", err)
	}

	cases := []struct {
		name   string
		userID UserID
		domain string
		obj    string
		act    Action
		want   bool
	}{
		{"super admin in domain", uid, "tenant-a", "resource", ActionRead, true},
		{"plain user no role", uid2, "tenant-a", "resource", ActionRead, false},
		{"admin in own domain", uid3, "tenant-a", "resource", ActionRead, true},
		{"admin in other domain", uid3, "tenant-b", "resource", ActionRead, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ok, err := a.Enforce(tc.userID.String(), tc.domain, tc.obj, tc.act)
			if err != nil {
				t.Fatalf("Enforce: %v", err)
			}
			if ok != tc.want {
				t.Fatalf("Enforce() = %v, want %v", ok, tc.want)
			}
		})
	}
}

func TestAuthz_Enforce_Uninitialized(t *testing.T) {
	a := &Authz{}
	if _, err := a.Enforce("u", "d", "o", ActionRead); !errors.Is(err, ErrEnforcerNotInitialized) {
		t.Fatalf("expected ErrEnforcerNotInitialized, got %v", err)
	}
}

func TestAuthz_EnforceAny(t *testing.T) {
	a := newTestAuthz(t)
	ok, err := a.EnforceAny("u1", "tenant-a", "resource", string(ActionRead))
	if err != nil {
		t.Fatalf("EnforceAny: %v", err)
	}
	if ok {
		t.Fatal("expected false for unprivileged user")
	}
}

func TestAuthz_EnforceEx(t *testing.T) {
	a := newTestAuthz(t)
	uid, _ := userActorPair("u1")
	if err := a.AddGroupPolicy(
		GroupPolicy{Subject: uid.String(), Role: RoleSuperAdmin, Domain: "tenant-a"},
	); err != nil {
		t.Fatalf("add group: %v", err)
	}
	res, err := a.EnforceEx(uid.String(), "tenant-a", "resource", ActionRead)
	if err != nil {
		t.Fatalf("EnforceEx: %v", err)
	}
	if !res.Allowed {
		t.Fatal("expected allowed")
	}
	if res.Subject != uid.String() || res.Domain != "tenant-a" || res.Object != "resource" || res.Action != ActionRead {
		t.Fatalf("unexpected result fields: %+v", res)
	}
}

func TestAuthz_Authorize(t *testing.T) {
	a := newTestAuthz(t)
	uid, _ := userActorPair("u1")
	uid2, _ := userActorPair("u2")
	if err := a.AddGroupPolicy(
		GroupPolicy{Subject: uid.String(), Role: RoleSuperAdmin, Domain: "tenant-a"},
	); err != nil {
		t.Fatalf("add group: %v", err)
	}
	if err := a.Authorize(uid.String(), "tenant-a", "resource", ActionRead); err != nil {
		t.Fatalf("Authorize: %v", err)
	}
	if err := a.Authorize(uid2.String(), "tenant-a", "resource", ActionRead); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestAuthz_AsEnforcer(t *testing.T) {
	a := newTestAuthz(t)
	e := a.AsEnforcer()
	ok, err := e.Enforce("u1", "tenant-a", "resource", string(ActionRead))
	if err != nil {
		t.Fatalf("Enforce: %v", err)
	}
	if ok {
		t.Fatal("expected false")
	}
}

func TestAuthz_Apply(t *testing.T) {
	a := newTestAuthz(t)
	uid, _ := userActorPair("u1")
	update := PolicyUpdate{
		AddGroups: []GroupPolicy{{Subject: uid.String(), Role: RoleAdmin, Domain: "tenant-a"}},
		AddPolicies: []Policy{
			{Subject: RoleUser, Domain: "tenant-a", Object: "resource", Action: ActionRead, Effect: EffectAllow},
		},
	}
	if err := a.Apply(update); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	ok, err := a.Enforce(uid.String(), "tenant-a", "resource", ActionRead)
	if err != nil {
		t.Fatalf("Enforce: %v", err)
	}
	if !ok {
		t.Fatal("expected allowed after Apply")
	}

	if err := a.Apply(
		PolicyUpdate{RemoveGroups: []GroupPolicy{{Subject: uid.String(), Role: RoleAdmin, Domain: "tenant-a"}}},
	); err != nil {
		t.Fatalf("Apply remove: %v", err)
	}
	ok, err = a.Enforce(uid.String(), "tenant-a", "resource", ActionRead)
	if err != nil {
		t.Fatalf("Enforce: %v", err)
	}
	if ok {
		t.Fatal("expected false after removing group")
	}
}

func TestAuthz_AddRemovePolicy(t *testing.T) {
	a := newTestAuthz(t)
	p := Policy{Subject: RoleUser, Domain: "tenant-a", Object: "resource", Action: ActionRead, Effect: EffectAllow}
	if err := a.AddPolicy(p); err != nil {
		t.Fatalf("AddPolicy: %v", err)
	}
	policies, err := a.Policies()
	if err != nil {
		t.Fatalf("Policies: %v", err)
	}
	if len(policies) == 0 {
		t.Fatal("expected policies")
	}
	if err := a.RemovePolicy(p); err != nil {
		t.Fatalf("RemovePolicy: %v", err)
	}
	policies, err = a.Policies()
	if err != nil {
		t.Fatalf("Policies: %v", err)
	}
	if slices.ContainsFunc(policies, func(row []string) bool {
		return len(row) > 0 && row[0] == string(RoleUser)
	}) {
		t.Fatal("expected policy removed")
	}
}

func TestAuthz_AddRemoveGroupPolicy(t *testing.T) {
	a := newTestAuthz(t)
	uid, _ := userActorPair("u1")
	g := GroupPolicy{Subject: uid.String(), Role: RoleUser, Domain: "tenant-a"}
	if err := a.AddGroupPolicy(g); err != nil {
		t.Fatalf("AddGroupPolicy: %v", err)
	}
	groups, err := a.GroupPolicies()
	if err != nil {
		t.Fatalf("GroupPolicies: %v", err)
	}
	found := false
	for _, row := range groups {
		if len(row) > 0 && row[0] == uid.String() {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected group policy, got %v", groups)
	}
	if err := a.RemoveGroupPolicy(g); err != nil {
		t.Fatalf("RemoveGroupPolicy: %v", err)
	}
	groups, err = a.GroupPolicies()
	if err != nil {
		t.Fatalf("GroupPolicies: %v", err)
	}
	for _, row := range groups {
		if len(row) > 0 && row[0] == uid.String() {
			t.Fatal("expected group policy removed")
		}
	}
}

func TestAuthz_RemoveAllRolesForUser(t *testing.T) {
	a := newTestAuthz(t)
	uid, _ := userActorPair("u1")
	for _, domain := range []string{"tenant-a", "tenant-b"} {
		if err := a.AddGroupPolicy(GroupPolicy{Subject: uid.String(), Role: RoleAdmin, Domain: domain}); err != nil {
			t.Fatalf("add group: %v", err)
		}
	}
	if err := a.RemoveAllRolesForUser(uid.String()); err != nil {
		t.Fatalf("RemoveAllRolesForUser: %v", err)
	}
	groups, err := a.GroupPolicies()
	if err != nil {
		t.Fatalf("GroupPolicies: %v", err)
	}
	for _, row := range groups {
		if len(row) > 0 && row[0] == uid.String() {
			t.Fatalf("expected all roles removed, got %v", groups)
		}
	}
}

func TestAuthz_RemoveAllRolesInDomain(t *testing.T) {
	a := newTestAuthz(t)
	uid, _ := userActorPair("u1")
	tidA := NewTenantID("tenant-a")
	tidB := NewTenantID("tenant-b")
	if err := a.AddGroupPolicy(GroupPolicy{Subject: uid.String(), Role: RoleAdmin, Domain: tidA.Get()}); err != nil {
		t.Fatalf("add group: %v", err)
	}
	if err := a.AddGroupPolicy(GroupPolicy{Subject: uid.String(), Role: RoleViewer, Domain: tidB.Get()}); err != nil {
		t.Fatalf("add group: %v", err)
	}
	if err := a.RemoveAllRolesInDomain(uid.String(), tidA); err != nil {
		t.Fatalf("RemoveAllRolesInDomain: %v", err)
	}
	roles, err := a.RolesForUser(uid, tidA)
	if err != nil {
		t.Fatalf("RolesForUser: %v", err)
	}
	if len(roles) != 0 {
		t.Fatalf("expected no roles in tenant-a, got %v", roles)
	}
	roles, err = a.RolesForUser(uid, tidB)
	if err != nil {
		t.Fatalf("RolesForUser: %v", err)
	}
	if len(roles) != 1 || roles[0] != RoleViewer {
		t.Fatalf("expected viewer in tenant-b, got %v", roles)
	}
}

func TestAuthz_PoliciesAndGroupPolicies_Uninitialized(t *testing.T) {
	a := &Authz{}
	if _, err := a.Policies(); !errors.Is(err, ErrEnforcerNotInitialized) {
		t.Fatalf("expected ErrEnforcerNotInitialized, got %v", err)
	}
	if _, err := a.GroupPolicies(); !errors.Is(err, ErrEnforcerNotInitialized) {
		t.Fatalf("expected ErrEnforcerNotInitialized, got %v", err)
	}
}

func TestAuthz_RolesForUser(t *testing.T) {
	a := newTestAuthz(t)
	uid, _ := userActorPair("u1")
	tid := NewTenantID("tenant-a")
	if err := a.AddGroupPolicy(GroupPolicy{Subject: uid.String(), Role: RoleAdmin, Domain: tid.Get()}); err != nil {
		t.Fatalf("add group: %v", err)
	}
	roles, err := a.RolesForUser(uid, tid)
	if err != nil {
		t.Fatalf("RolesForUser: %v", err)
	}
	if len(roles) != 1 || roles[0] != RoleAdmin {
		t.Fatalf("expected [admin], got %v", roles)
	}
}

func TestAuthz_ImplicitRolesForUser(t *testing.T) {
	a := newTestAuthz(t)
	uid, _ := userActorPair("u1")
	tid := NewTenantID("tenant-a")
	if err := a.AddGroupPolicy(GroupPolicy{Subject: uid.String(), Role: RoleAdmin, Domain: tid.Get()}); err != nil {
		t.Fatalf("add group: %v", err)
	}
	roles, err := a.ImplicitRolesForUser(uid, tid)
	if err != nil {
		t.Fatalf("ImplicitRolesForUser: %v", err)
	}
	if len(roles) == 0 {
		t.Fatal("expected inherited roles")
	}
}

func TestAuthz_ImplicitPermissionsForUser(t *testing.T) {
	a := newTestAuthz(t)
	uid, _ := userActorPair("u1")
	tid := NewTenantID("tenant-a")
	// Add a domain-scoped policy for the admin role so ImplicitPermissionsForUser
	// finds it (Casbin matches domain exactly, not via wildcard).
	if err := a.AddPolicy(
		Policy{Subject: RoleAdmin, Domain: tid.Get(), Object: "resource", Action: ActionRead, Effect: EffectAllow},
	); err != nil {
		t.Fatalf("add policy: %v", err)
	}
	if err := a.AddGroupPolicy(GroupPolicy{Subject: uid.String(), Role: RoleAdmin, Domain: tid.Get()}); err != nil {
		t.Fatalf("add group: %v", err)
	}
	perms, err := a.ImplicitPermissionsForUser(uid, tid)
	if err != nil {
		t.Fatalf("ImplicitPermissionsForUser: %v", err)
	}
	if len(perms) == 0 {
		t.Fatal("expected implicit permissions")
	}
}

func TestAuthz_DomainsForUser(t *testing.T) {
	a := newTestAuthz(t)
	uid, _ := userActorPair("u1")
	if err := a.AddGroupPolicy(GroupPolicy{Subject: uid.String(), Role: RoleAdmin, Domain: "tenant-a"}); err != nil {
		t.Fatalf("add group: %v", err)
	}
	if err := a.AddGroupPolicy(GroupPolicy{Subject: uid.String(), Role: RoleViewer, Domain: "tenant-b"}); err != nil {
		t.Fatalf("add group: %v", err)
	}
	domains, err := a.DomainsForUser(uid)
	if err != nil {
		t.Fatalf("DomainsForUser: %v", err)
	}
	if len(domains) != 2 {
		t.Fatalf("expected 2 domains, got %v", domains)
	}
}

func TestAuthz_UsersForRole(t *testing.T) {
	a := newTestAuthz(t)
	uid, _ := userActorPair("u1")
	tid := NewTenantID("tenant-a")
	if err := a.AddGroupPolicy(GroupPolicy{Subject: uid.String(), Role: RoleAdmin, Domain: tid.Get()}); err != nil {
		t.Fatalf("add group: %v", err)
	}
	users, err := a.UsersForRole(RoleAdmin, tid)
	if err != nil {
		t.Fatalf("UsersForRole: %v", err)
	}
	found := false
	for _, u := range users {
		if u == uid.String() {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected user in users, got %v", users)
	}
}

func TestAuthz_RolesForActor(t *testing.T) {
	a := newTestAuthz(t)
	uid, aid := userActorPair("u1")
	tid := NewTenantID("tenant-a")
	if err := a.AddGroupPolicy(GroupPolicy{Subject: uid.String(), Role: RoleAdmin, Domain: tid.Get()}); err != nil {
		t.Fatalf("add group: %v", err)
	}
	roles, err := a.RolesForActor(aid, tid)
	if err != nil {
		t.Fatalf("RolesForActor: %v", err)
	}
	if len(roles) != 1 || roles[0] != RoleAdmin {
		t.Fatalf("expected [admin], got %v", roles)
	}
}

func TestAuthz_ImplicitRolesForActor(t *testing.T) {
	a := newTestAuthz(t)
	uid, aid := userActorPair("u1")
	tid := NewTenantID("tenant-a")
	if err := a.AddGroupPolicy(GroupPolicy{Subject: uid.String(), Role: RoleAdmin, Domain: tid.Get()}); err != nil {
		t.Fatalf("add group: %v", err)
	}
	roles, err := a.ImplicitRolesForActor(aid, tid)
	if err != nil {
		t.Fatalf("ImplicitRolesForActor: %v", err)
	}
	if len(roles) == 0 {
		t.Fatal("expected inherited roles")
	}
}

func TestAuthz_NewAuthz_WithConfig(t *testing.T) {
	uid, _ := userActorPair("u1")
	a, err := NewAuthz(EnforcerConfig{
		ModelString: DefaultRBACModel,
		Policies:    DefaultPolicies(),
		Groups:      []GroupPolicy{{Subject: uid.String(), Role: RoleSuperAdmin, Domain: "tenant-a"}},
	})
	if err != nil {
		t.Fatalf("NewAuthz with config: %v", err)
	}
	ok, err := a.Enforce(uid.String(), "tenant-a", "resource", ActionRead)
	if err != nil {
		t.Fatalf("Enforce: %v", err)
	}
	if !ok {
		t.Fatal("expected allowed via configured group")
	}
}
