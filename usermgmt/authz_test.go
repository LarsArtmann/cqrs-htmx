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
	if err := a.AddGroupPolicy(
		GroupPolicy{User: "alice", Role: RoleAdmin, Domain: "tenant1"},
	); err != nil {
		t.Fatalf("AddGroupPolicy: %v", err)
	}

	tests := []struct{ obj, act string }{
		{"game.play_round", "execute"},
		{"game.create", "execute"},
		{"game.get", "read"},
		{"anything.at.all", "whatever"},
	}
	for _, tt := range tests {
		ok, err := a.Enforce("alice", "tenant1", tt.obj, Action(tt.act))
		if err != nil {
			t.Errorf("Enforce(alice, tenant1, %s, %s) error: %v", tt.obj, tt.act, err)
		} else if !ok {
			t.Errorf("Enforce(alice, tenant1, %s, %s) = false, want true", tt.obj, tt.act)
		}
	}
}

func TestAuthz_PublicAccess(t *testing.T) {
	a := newTestAuthz(t, Policy{"*", "*", "game.get", ActionRead, EffectAllow})

	ok, err := a.Enforce("anyone", "any-domain", "game.get", ActionRead)
	if err != nil {
		t.Fatalf("Enforce error: %v", err)
	}
	if !ok {
		t.Error("expected public read access to game.get")
	}
}

func TestAuthz_OwnerInDomain(t *testing.T) {
	a := newTestAuthz(t,
		Policy{RoleOwner, "*", "game.play_round", ActionExecute, EffectAllow},
		Policy{RoleOwner, "*", "game.finish", ActionExecute, EffectAllow},
	)
	if err := a.AddGroupPolicy(
		GroupPolicy{User: "player1", Role: RoleOwner, Domain: "game1"},
	); err != nil {
		t.Fatalf("AddGroupPolicy: %v", err)
	}

	ok, _ := a.Enforce("player1", "game1", "game.play_round", ActionExecute)
	if !ok {
		t.Error("owner should execute game.play_round in their domain")
	}

	ok, _ = a.Enforce("player1", "game1", "game.finish", ActionExecute)
	if !ok {
		t.Error("owner should execute game.finish in their domain")
	}

	ok, _ = a.Enforce("player1", "other-game", "game.play_round", ActionExecute)
	if ok {
		t.Error("owner should NOT have access in other domain")
	}

	ok, _ = a.Enforce("stranger", "game1", "game.play_round", ActionExecute)
	if ok {
		t.Error("non-owner should NOT have access")
	}
}

func TestAuthz_DenyOverride(t *testing.T) {
	a := newTestAuthz(t,
		Policy{"*", "*", "audit.replay", ActionExecute, EffectDeny},
		Policy{RoleAdmin, "*", "*", ActionAll, EffectAllow},
	)
	if err := a.AddGroupPolicy(
		GroupPolicy{User: "admin1", Role: RoleAdmin, Domain: "*"},
	); err != nil {
		t.Fatalf("AddGroupPolicy: %v", err)
	}

	ok, _ := a.Enforce("admin1", "*", "audit.replay", ActionExecute)
	if ok {
		t.Error("deny should override admin allow for audit.replay")
	}

	ok, _ = a.Enforce("admin1", "*", "anything.else", ActionExecute)
	if !ok {
		t.Error("admin should be allowed for non-denied resources")
	}
}

func TestAuthz_EnforceEx(t *testing.T) {
	a := newTestAuthz(t, Policy{RoleOwner, "*", "game.play_round", ActionExecute, EffectAllow})
	if err := a.AddGroupPolicy(GroupPolicy{User: "p1", Role: RoleOwner, Domain: "g1"}); err != nil {
		t.Fatalf("AddGroupPolicy: %v", err)
	}

	result, err := a.EnforceEx("p1", "g1", "game.play_round", ActionExecute)
	if err != nil {
		t.Fatalf("EnforceEx error: %v", err)
	}
	if !result.Allowed {
		t.Error("expected allowed")
	}
	if len(result.MatchedRule) == 0 {
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
	a := newTestAuthz(t,
		Policy{RoleOwner, "*", "game.play_round", ActionExecute, EffectAllow},
	)

	err := a.Apply(PolicyUpdate{
		AddGroups: []GroupPolicy{
			{User: "p1", Role: RoleOwner, Domain: "g1"},
		},
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	ok, _ := a.Enforce("p1", "g1", "game.play_round", ActionExecute)
	if !ok {
		t.Error("expected access after Apply")
	}

	err = a.Apply(PolicyUpdate{
		RemoveGroups: []GroupPolicy{
			{User: "p1", Role: RoleOwner, Domain: "g1"},
		},
	})
	if err != nil {
		t.Fatalf("Apply remove: %v", err)
	}

	ok, _ = a.Enforce("p1", "g1", "game.play_round", ActionExecute)
	if ok {
		t.Error("expected denied after removing group")
	}
}

func TestAuthz_ApplyPolicies(t *testing.T) {
	a := newTestAuthz(t)

	err := a.Apply(PolicyUpdate{
		AddPolicies: []Policy{
			{"*", "*", "game.create", ActionExecute, EffectAllow},
		},
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	ok, _ := a.Enforce("anyone", "any-domain", "game.create", ActionExecute)
	if !ok {
		t.Error("expected public access after adding policy")
	}
}

func TestAuthz_RolesForUser(t *testing.T) {
	a := newTestAuthz(t)
	if err := a.AddGroupPolicy(GroupPolicy{User: "p1", Role: RoleOwner, Domain: "g1"}); err != nil {
		t.Fatalf("AddGroupPolicy: %v", err)
	}

	roles, err := a.RolesForUser("p1", "g1")
	if err != nil {
		t.Fatalf("RolesForUser: %v", err)
	}
	if len(roles) != 1 || roles[0] != RoleOwner {
		t.Errorf("expected [owner], got %v", roles)
	}

	roles, err = a.RolesForUser("p1", "other")
	if err != nil {
		t.Fatalf("RolesForUser: %v", err)
	}
	if len(roles) != 0 {
		t.Errorf("expected no roles in other domain, got %v", roles)
	}
}

func TestAuthz_DomainsForUser(t *testing.T) {
	a := newTestAuthz(t)
	if err := a.AddGroupPolicy(GroupPolicy{User: "p1", Role: RoleOwner, Domain: "g1"}); err != nil {
		t.Fatalf("AddGroupPolicy: %v", err)
	}
	if err := a.AddGroupPolicy(
		GroupPolicy{User: "p1", Role: RoleViewer, Domain: "g2"},
	); err != nil {
		t.Fatalf("AddGroupPolicy: %v", err)
	}

	domains, err := a.DomainsForUser("p1")
	if err != nil {
		t.Fatalf("DomainsForUser: %v", err)
	}
	if len(domains) != 2 {
		t.Errorf("expected 2 domains, got %v", domains)
	}
}

func TestAuthz_UsersForRole(t *testing.T) {
	a := newTestAuthz(t)
	if err := a.AddGroupPolicy(GroupPolicy{User: "p1", Role: RoleOwner, Domain: "g1"}); err != nil {
		t.Fatalf("AddGroupPolicy: %v", err)
	}

	users, err := a.UsersForRole(RoleOwner, "g1")
	if err != nil {
		t.Fatalf("UsersForRole: %v", err)
	}
	if len(users) != 1 || users[0] != "p1" {
		t.Errorf("expected [p1], got %v", users)
	}
}

func TestAuthz_CustomModel(t *testing.T) {
	basicModel := `[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = r.sub == p.sub && r.obj == p.obj && r.act == p.act
`
	a, err := NewAuthz(EnforcerConfig{
		ModelString: basicModel,
	})
	if err != nil {
		t.Fatalf("NewAuthz with custom model: %v", err)
	}

	e := a.RawEnforcer()
	_, _ = e.AddPolicy("alice", "data1", "read")

	ok, err := e.Enforce("alice", "data1", "read")
	if err != nil {
		t.Fatalf("Enforce alice: %v", err)
	}
	if !ok {
		t.Error("expected alice to read data1")
	}

	ok, err = e.Enforce("bob", "data1", "read")
	if err != nil {
		t.Fatalf("Enforce bob: %v", err)
	}
	if ok {
		t.Error("expected bob to be denied")
	}
}

func newTestAuthz(t *testing.T, extra ...Policy) *Authz {
	t.Helper()
	policies := append([]Policy{{RoleAdmin, "*", "*", ActionAll, EffectAllow}}, extra...)
	a, err := NewAuthz(EnforcerConfig{Policies: policies})
	if err != nil {
		t.Fatalf("newTestAuthz: %v", err)
	}
	return a
}
