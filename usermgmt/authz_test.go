package usermgmt

import (
	"errors"
	"testing"
)

func TestNewAuthz_DefaultConfig(t *testing.T) {
	a, err := NewAuthz()
	if err != nil {
		t.Fatalf("NewAuthz failed: %v", err)
	}
	if a == nil {
		t.Fatal("expected non-nil Authz")
	}
}

func TestAuthz_AdminWildcard(t *testing.T) {
	a := newTestAuthz(t)
	addTestGroupPolicy(t, a, GroupPolicy{Subject: "alice", Role: RoleAdmin, Domain: "tenant1"})

	tests := []struct{ obj, act string }{
		{"game.play_round", "execute"},
		{"game.create", "execute"},
		{"game.get", "read"},
		{"anything.at.all", "whatever"},
	}
	for _, tt := range tests {
		assertEnforce(t, a, "alice", "tenant1", tt.obj, Action(tt.act), true)
	}
}

func TestAuthz_PublicAccess(t *testing.T) {
	a := newTestAuthz(t, Policy{Subject: "*", Domain: "*", Object: "game.get", Action: ActionRead, Effect: EffectAllow})
	assertEnforce(t, a, "anyone", "any-domain", "game.get", ActionRead, true)
}

func TestAuthz_OwnerInDomain(t *testing.T) {
	a := newTestAuthz(
		t,
		Policy{Subject: RoleOwner, Domain: "*", Object: "game.play_round", Action: ActionExecute, Effect: EffectAllow},
		Policy{Subject: RoleOwner, Domain: "*", Object: "game.finish", Action: ActionExecute, Effect: EffectAllow},
	)
	addTestGroupPolicy(t, a, GroupPolicy{Subject: "player1", Role: RoleOwner, Domain: "game1"})

	assertEnforce(t, a, "player1", "game1", "game.play_round", ActionExecute, true)
	assertEnforce(t, a, "player1", "game1", "game.finish", ActionExecute, true)
	assertEnforce(t, a, "player1", "other-game", "game.play_round", ActionExecute, false)
	assertEnforce(t, a, "stranger", "game1", "game.play_round", ActionExecute, false)
}

func TestAuthz_DenyOverride(t *testing.T) {
	a := newTestAuthz(
		t,
		Policy{Subject: "*", Domain: "*", Object: "audit.replay", Action: ActionExecute, Effect: EffectDeny},
		Policy{Subject: RoleAdmin, Domain: "*", Object: "*", Action: ActionAll, Effect: EffectAllow},
	)
	addTestGroupPolicy(t, a, GroupPolicy{Subject: "admin1", Role: RoleAdmin, Domain: "*"})

	assertEnforce(t, a, "admin1", "*", "audit.replay", ActionExecute, false)
	assertEnforce(t, a, "admin1", "*", "anything.else", ActionExecute, true)
}

func TestAuthz_EnforceEx(t *testing.T) {
	a := newTestAuthz(
		t,
		Policy{Subject: RoleOwner, Domain: "*", Object: "game.play_round", Action: ActionExecute, Effect: EffectAllow},
	)
	addTestGroupPolicy(t, a, GroupPolicy{Subject: "p1", Role: RoleOwner, Domain: "g1"})

	result, err := a.EnforceEx("p1", "g1", "game.play_round", ActionExecute)
	if err != nil {
		t.Fatalf("EnforceEx error: %v", err)
	}
	if !result.Allowed {
		t.Error("expected allowed")
	}
	if len(result.MatchedRules) == 0 {
		t.Error("expected non-empty matched rule")
	}
}

func TestAuthz_Authorize_ReturnsErrForbidden(t *testing.T) {
	a := newTestAuthz(t)

	err := a.Authorize("nobody", "any-domain", "game.play_round", ActionExecute)
	if err == nil {
		t.Fatal("expected error for unauthorized access")
	}
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got: %v", err)
	}
}

func TestAuthz_Apply(t *testing.T) {
	a := newTestAuthz(
		t,
		Policy{Subject: RoleOwner, Domain: "*", Object: "game.play_round", Action: ActionExecute, Effect: EffectAllow},
	)

	if err := a.Apply(PolicyUpdate{
		AddGroups: []GroupPolicy{{Subject: "p1", Role: RoleOwner, Domain: "g1"}},
	}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	assertEnforce(t, a, "p1", "g1", "game.play_round", ActionExecute, true)

	if err := a.Apply(PolicyUpdate{
		RemoveGroups: []GroupPolicy{{Subject: "p1", Role: RoleOwner, Domain: "g1"}},
	}); err != nil {
		t.Fatalf("Apply remove: %v", err)
	}
	assertEnforce(t, a, "p1", "g1", "game.play_round", ActionExecute, false)
}

func TestAuthz_Apply_AddBeforeRemove(t *testing.T) {
	a := newTestAuthz(
		t,
		Policy{Subject: RoleAdmin, Domain: "*", Object: "*", Action: ActionAll, Effect: EffectAllow},
	)
	addTestGroupPolicy(t, a, GroupPolicy{Subject: "u1", Role: RoleUser, Domain: "d1"})

	if err := a.Apply(PolicyUpdate{
		AddGroups:    []GroupPolicy{{Subject: "u1", Role: RoleAdmin, Domain: "d1"}},
		RemoveGroups: []GroupPolicy{{Subject: "u1", Role: RoleUser, Domain: "d1"}},
	}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	assertEnforce(t, a, "u1", "d1", "anything", ActionExecute, true)
}

func TestAuthz_ApplyPolicies(t *testing.T) {
	a := newTestAuthz(t)

	if err := a.Apply(PolicyUpdate{
		AddPolicies: []Policy{
			{Subject: "*", Domain: "*", Object: "game.create", Action: ActionExecute, Effect: EffectAllow},
		},
	}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	assertEnforce(t, a, "anyone", "any-domain", "game.create", ActionExecute, true)
}

func TestAuthz_RolesForUser(t *testing.T) {
	a := newTestAuthz(t)
	addTestGroupPolicy(t, a, GroupPolicy{Subject: NewUserID("p1").Get().String(), Role: RoleOwner, Domain: "g1"})

	roles, err := a.RolesForUser(NewUserID("p1"), NewTenantID("g1"))
	if err != nil {
		t.Fatalf("RolesForUser: %v", err)
	}
	if len(roles) != 1 || roles[0] != RoleOwner {
		t.Errorf("expected [owner], got %v", roles)
	}

	roles, err = a.RolesForUser(NewUserID("p1"), NewTenantID("other"))
	if err != nil {
		t.Fatalf("RolesForUser: %v", err)
	}
	if len(roles) != 0 {
		t.Errorf("expected no roles in other domain, got %v", roles)
	}
}

func TestAuthz_DomainsForUser(t *testing.T) {
	a := newTestAuthz(t)
	addTestGroupPolicy(t, a, GroupPolicy{Subject: NewUserID("p1").Get().String(), Role: RoleOwner, Domain: "g1"})
	addTestGroupPolicy(t, a, GroupPolicy{Subject: NewUserID("p1").Get().String(), Role: RoleViewer, Domain: "g2"})

	domains, err := a.DomainsForUser(NewUserID("p1"))
	if err != nil {
		t.Fatalf("DomainsForUser: %v", err)
	}
	if len(domains) != 2 {
		t.Errorf("expected 2 domains, got %v", domains)
	}
}

func TestAuthz_UsersForRole(t *testing.T) {
	a := newTestAuthz(t)
	addTestGroupPolicy(t, a, GroupPolicy{Subject: NewUserID("p1").Get().String(), Role: RoleOwner, Domain: "g1"})

	users, err := a.UsersForRole(RoleOwner, NewTenantID("g1"))
	if err != nil {
		t.Fatalf("UsersForRole: %v", err)
	}
	if len(users) != 1 || users[0] != NewUserID("p1").Get().String() {
		t.Errorf("expected [p1], got %v", users)
	}
}

func TestAuthz_EnforceAny(t *testing.T) {
	a := newTestAuthz(t)
	addTestGroupPolicy(t, a, GroupPolicy{Subject: "alice", Role: RoleAdmin, Domain: "tenant1"})

	ok, err := a.EnforceAny("alice", "tenant1", "anything", "execute")
	if err != nil {
		t.Fatalf("EnforceAny alice: %v", err)
	}
	if !ok {
		t.Error("expected admin to be allowed via EnforceAny")
	}

	adapter := a.AsEnforcer()
	ok, err = adapter.Enforce("alice", "tenant1", "anything", "execute")
	if err != nil {
		t.Fatalf("AsEnforcer alice: %v", err)
	}
	if !ok {
		t.Error("expected admin to be allowed via AsEnforcer")
	}
}

func TestAuthz_ImplicitRolesForUser(t *testing.T) {
	a := newTestAuthz(
		t,
		Policy{Subject: RoleOwner, Domain: "*", Object: "game.play_round", Action: ActionExecute, Effect: EffectAllow},
	)
	_ = a.AddGroupPolicy(GroupPolicy{Subject: NewUserID("p1").Get().String(), Role: RoleOwner, Domain: "g1"})

	roles, err := a.ImplicitRolesForUser(NewUserID("p1"), NewTenantID("g1"))
	if err != nil {
		t.Fatalf("ImplicitRolesForUser: %v", err)
	}
	if len(roles) != 1 || roles[0] != RoleOwner {
		t.Errorf("expected [owner], got %v", roles)
	}

	roles, err = a.ImplicitRolesForUser(NewUserID("p1"), NewTenantID("other"))
	if err != nil {
		t.Fatalf("ImplicitRolesForUser: %v", err)
	}
	if len(roles) != 0 {
		t.Errorf("expected no implicit roles in other domain, got %v", roles)
	}
}

func TestAuthz_ImplicitPermissionsForUser(t *testing.T) {
	a := newTestAuthz(
		t,
		Policy{Subject: RoleOwner, Domain: "g1", Object: "game.play_round", Action: ActionExecute, Effect: EffectAllow},
	)
	_ = a.AddGroupPolicy(GroupPolicy{Subject: NewUserID("p1").Get().String(), Role: RoleOwner, Domain: "g1"})

	perms, err := a.ImplicitPermissionsForUser(NewUserID("p1"), NewTenantID("g1"))
	if err != nil {
		t.Fatalf("ImplicitPermissionsForUser: %v", err)
	}
	if len(perms) == 0 {
		t.Error("expected implicit permissions for owner in g1")
	}
}

func TestAuthz_Policies(t *testing.T) {
	a := newTestAuthz(
		t,
		Policy{Subject: "*", Domain: "*", Object: "game.get", Action: ActionRead, Effect: EffectAllow},
	)

	policies, err := a.Policies()
	if err != nil {
		t.Fatalf("Policies: %v", err)
	}
	if len(policies) < 2 {
		t.Errorf("expected at least 2 policies (admin + custom), got %d", len(policies))
	}
}

func TestAuthz_GroupPolicies(t *testing.T) {
	a := newTestAuthz(t)
	_ = a.AddGroupPolicy(GroupPolicy{Subject: "p1", Role: RoleOwner, Domain: "g1"})

	groups, err := a.GroupPolicies()
	if err != nil {
		t.Fatalf("GroupPolicies: %v", err)
	}
	if len(groups) != 1 {
		t.Errorf("expected 1 group policy, got %d", len(groups))
	}
}

func TestAuthz_AddAndRemovePolicy(t *testing.T) {
	a := newTestAuthz(t)

	if err := a.AddPolicy(
		Policy{Subject: "*", Domain: "*", Object: "game.create", Action: ActionExecute, Effect: EffectAllow},
	); err != nil {
		t.Fatalf("AddPolicy: %v", err)
	}
	assertEnforce(t, a, "anyone", "any-domain", "game.create", ActionExecute, true)

	if err := a.RemovePolicy(
		Policy{Subject: "*", Domain: "*", Object: "game.create", Action: ActionExecute, Effect: EffectAllow},
	); err != nil {
		t.Fatalf("RemovePolicy: %v", err)
	}
	assertEnforce(t, a, "anyone", "any-domain", "game.create", ActionExecute, false)
}

func TestAuthz_RemoveGroupPolicy(t *testing.T) {
	a := newTestAuthz(t)
	_ = a.AddGroupPolicy(GroupPolicy{Subject: "p1", Role: RoleOwner, Domain: "g1"})

	if err := a.RemoveGroupPolicy(
		GroupPolicy{Subject: "p1", Role: RoleOwner, Domain: "g1"},
	); err != nil {
		t.Fatalf("RemoveGroupPolicy: %v", err)
	}
	assertEnforce(t, a, "p1", "g1", "anything", ActionAll, false)
}

func newTestAuthz(t *testing.T, extra ...Policy) *Authz {
	t.Helper()
	policies := append(
		[]Policy{{Subject: RoleAdmin, Domain: "*", Object: "*", Action: ActionAll, Effect: EffectAllow}},
		extra...)
	a, err := NewAuthz(EnforcerConfig{Policies: policies})
	if err != nil {
		t.Fatalf("newTestAuthz: %v", err)
	}
	return a
}

func TestAssignableRoles(t *testing.T) {
	t.Parallel()
	roles := AssignableRoles()
	if len(roles) != 4 {
		t.Fatalf("AssignableRoles returned %d, want 4", len(roles))
	}
	for _, r := range roles {
		if r == RoleSuperAdmin {
			t.Error("AssignableRoles should not include super_admin (global-only)")
		}
	}
	// Must include the core assignable roles.
	want := map[Role]bool{RoleViewer: true, RoleUser: true, RoleAdmin: true, RoleOwner: true}
	for _, r := range roles {
		if !want[r] {
			t.Errorf("unexpected role %q", r)
		}
	}
}
