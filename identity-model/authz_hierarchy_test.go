package identitymodel

import (
	"errors"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

// The role hierarchy is documented as super_admin > admin > user > viewer.
// These tests pin that it actually fires. Before the domain mirror existed the
// g2 seed was inert: Casbin keeps one role manager per role definition, so a
// grant in `g` and an inheritance edge in `g2` never composed.

func TestAuthz_RoleHierarchy_UserInheritsViewer(t *testing.T) {
	a := newTestAuthz(t)
	uid, _ := userActorPair("u1")
	tid := NewTenantID("tenant-a")

	if err := a.AddPolicy(Policy{
		Subject: RoleViewer, Domain: tid.Get(), Object: "report", Action: ActionRead, Effect: EffectAllow,
	}); err != nil {
		t.Fatalf("add policy: %v", err)
	}
	if err := a.AddGroupPolicy(GroupPolicy{Subject: uid.String(), Role: RoleUser, Domain: tid.Get()}); err != nil {
		t.Fatalf("add group: %v", err)
	}

	ok, err := a.Enforce(uid.String(), tid.Get(), "report", ActionRead)
	if err != nil {
		t.Fatalf("enforce: %v", err)
	}
	if !ok {
		t.Fatal("a user grant must inherit viewer permissions")
	}

	roles, err := a.ImplicitRolesForUser(uid, tid)
	if err != nil {
		t.Fatalf("implicit roles: %v", err)
	}
	if !containsRole(roles, RoleViewer) {
		t.Fatalf("implicit roles must include viewer, got %v", roles)
	}
}

func TestAuthz_RoleHierarchy_IsNotDownward(t *testing.T) {
	a := newTestAuthz(t)
	uid, _ := userActorPair("u1")
	tid := NewTenantID("tenant-a")

	if err := a.AddPolicy(Policy{
		Subject: RoleUser, Domain: tid.Get(), Object: "report", Action: ActionRead, Effect: EffectAllow,
	}); err != nil {
		t.Fatalf("add policy: %v", err)
	}
	if err := a.AddGroupPolicy(GroupPolicy{Subject: uid.String(), Role: RoleViewer, Domain: tid.Get()}); err != nil {
		t.Fatalf("add group: %v", err)
	}

	ok, err := a.Enforce(uid.String(), tid.Get(), "report", ActionRead)
	if err != nil {
		t.Fatalf("enforce: %v", err)
	}
	if ok {
		t.Fatal("viewer must NOT inherit user permissions")
	}
}

func TestAuthz_RoleHierarchy_OwnerIsOutsideTheChain(t *testing.T) {
	a := newTestAuthz(t)
	uid, _ := userActorPair("u1")
	tid := NewTenantID("tenant-a")

	if err := a.AddPolicy(Policy{
		Subject: RoleUser, Domain: tid.Get(), Object: "report", Action: ActionRead, Effect: EffectAllow,
	}); err != nil {
		t.Fatalf("add policy: %v", err)
	}
	if err := a.AddGroupPolicy(GroupPolicy{Subject: uid.String(), Role: RoleOwner, Domain: tid.Get()}); err != nil {
		t.Fatalf("add group: %v", err)
	}

	ok, err := a.Enforce(uid.String(), tid.Get(), "report", ActionRead)
	if err != nil {
		t.Fatalf("enforce: %v", err)
	}
	if ok {
		t.Fatal("owner is documented outside the hierarchy chain and must not inherit user")
	}
}

func TestAuthz_RoleHierarchy_IsDomainScoped(t *testing.T) {
	a := newTestAuthz(t)
	uid, _ := userActorPair("u1")
	tidA := NewTenantID("tenant-a")
	tidB := NewTenantID("tenant-b")

	if err := a.AddPolicy(Policy{
		Subject: RoleViewer, Domain: tidB.Get(), Object: "report", Action: ActionRead, Effect: EffectAllow,
	}); err != nil {
		t.Fatalf("add policy: %v", err)
	}
	if err := a.AddGroupPolicy(GroupPolicy{Subject: uid.String(), Role: RoleUser, Domain: tidA.Get()}); err != nil {
		t.Fatalf("add group: %v", err)
	}

	ok, err := a.Enforce(uid.String(), tidB.Get(), "report", ActionRead)
	if err != nil {
		t.Fatalf("enforce: %v", err)
	}
	if ok {
		t.Fatal("a tenant-a grant must not satisfy a tenant-b policy through the hierarchy")
	}
}

func TestAuthz_EnsureRoleHierarchy_IsIdempotent(t *testing.T) {
	a := newTestAuthz(t)
	tid := NewTenantID("tenant-a")

	for range 3 {
		if err := a.EnsureRoleHierarchy(tid); err != nil {
			t.Fatalf("EnsureRoleHierarchy: %v", err)
		}
	}

	groups, err := a.GroupPolicies()
	if err != nil {
		t.Fatalf("group policies: %v", err)
	}
	edges := 0
	for _, row := range groups {
		if len(row) == 3 && Role(row[0]).Valid() && Role(row[1]).Valid() {
			edges++
		}
	}
	if edges != len(DefaultRoleHierarchy()) {
		t.Fatalf("expected %d hierarchy edges, got %d in %v", len(DefaultRoleHierarchy()), edges, groups)
	}
}

func TestAuthz_EnsureRoleHierarchy_Uninitialized(t *testing.T) {
	a := &Authz{}
	if err := a.EnsureRoleHierarchy(NewTenantID("tenant-a")); !errors.Is(err, ErrEnforcerNotInitialized) {
		t.Fatalf("expected ErrEnforcerNotInitialized, got %v", err)
	}
}

func TestAuthz_UsersForRole_ExcludesRoleSubjects(t *testing.T) {
	a := newTestAuthz(t)
	uid, _ := userActorPair("u1")
	tid := NewTenantID("tenant-a")

	if err := a.AddGroupPolicy(GroupPolicy{Subject: uid.String(), Role: RoleUser, Domain: tid.Get()}); err != nil {
		t.Fatalf("add group: %v", err)
	}

	users, err := a.UsersForRole(RoleViewer, tid)
	if err != nil {
		t.Fatalf("UsersForRole: %v", err)
	}
	for _, subject := range users {
		if Role(subject).Valid() {
			t.Fatalf("role %q leaked into the user list: %v", subject, users)
		}
	}

	users, err = a.UsersForRole(RoleUser, tid)
	if err != nil {
		t.Fatalf("UsersForRole: %v", err)
	}
	if len(users) != 1 || users[0] != uid.String() {
		t.Fatalf("expected exactly the granted user, got %v", users)
	}
}

// stubMutator fails the nth mutation so Apply's rollback path is reachable
// without a broken policy store. `depth` tracks net applied mutations: adds
// increase it, removes (including compensations) decrease it.
type stubMutator struct {
	calls  int
	failAt int
	depth  int
	undos  int
}

func (s *stubMutator) AddGroupingPolicy(params ...any) (bool, error) {
	return s.record(true)
}

func (s *stubMutator) RemoveGroupingPolicy(params ...any) (bool, error) {
	return s.record(false)
}

func (s *stubMutator) AddPolicy(params ...any) (bool, error) {
	return s.record(true)
}

func (s *stubMutator) RemovePolicy(params ...any) (bool, error) {
	return s.record(false)
}

func (s *stubMutator) record(add bool) (bool, error) {
	s.calls++
	if s.calls == s.failAt {
		return false, errorfamily.NewTransient("casbin_error", "stub failure")
	}
	if add {
		s.depth++
	} else {
		s.depth--
		s.undos++
	}

	return true, nil
}

func TestApplyPolicyUpdate_RollsBackOnFailure(t *testing.T) {
	m := &stubMutator{failAt: 3}
	update := PolicyUpdate{
		AddGroups: []GroupPolicy{
			{Subject: "u1", Role: RoleAdmin, Domain: "d"},
			{Subject: "u2", Role: RoleUser, Domain: "d"},
		},
		AddPolicies: []Policy{
			{Subject: RoleUser, Domain: "d", Object: "o", Action: ActionRead, Effect: EffectAllow},
		},
	}

	err := applyPolicyUpdate(m, update)
	if err == nil {
		t.Fatal("expected the stub failure to surface")
	}
	if m.depth != 0 {
		t.Fatalf("rollback must return the enforcer to its pre-batch state, net depth is %d", m.depth)
	}
	if m.undos != 2 {
		t.Fatalf("expected 2 compensating mutations, got %d", m.undos)
	}
}

func TestApplyPolicyUpdate_NoRollbackWhenNothingApplied(t *testing.T) {
	m := &stubMutator{failAt: 1}
	err := applyPolicyUpdate(m, PolicyUpdate{
		AddGroups: []GroupPolicy{{Subject: "u1", Role: RoleAdmin, Domain: "d"}},
	})
	if err == nil {
		t.Fatal("expected failure")
	}
	if m.undos != 0 {
		t.Fatalf("nothing was applied, so nothing may be rolled back (got %d compensations)", m.undos)
	}
}

func TestApplyPolicyUpdate_SuccessKeepsEverything(t *testing.T) {
	m := &stubMutator{}
	update := PolicyUpdate{
		AddGroups:   []GroupPolicy{{Subject: "u1", Role: RoleAdmin, Domain: "d"}},
		AddPolicies: []Policy{{Subject: RoleUser, Domain: "d", Object: "o", Action: ActionRead, Effect: EffectAllow}},
	}
	if err := applyPolicyUpdate(m, update); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if m.depth != 2 || m.undos != 0 {
		t.Fatalf("expected 2 net adds and no compensations, got depth=%d undos=%d", m.depth, m.undos)
	}
}

func containsRole(roles []Role, want Role) bool {
	for _, r := range roles {
		if r == want {
			return true
		}
	}

	return false
}
