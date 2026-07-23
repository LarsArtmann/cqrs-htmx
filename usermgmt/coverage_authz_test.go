package usermgmt

import (
	"testing"
)

func TestAuthz_EnforceEx_Denied(t *testing.T) {
	a := newTestAuthz(t)
	result, err := a.EnforceEx("nobody", "dom", "secret", ActionRead)
	if err != nil {
		t.Fatalf("EnforceEx: %v", err)
	}
	if result.Allowed {
		t.Error("expected denied")
	}
}

func TestAuthz_Authorize_Allowed(t *testing.T) {
	a := newTestAuthz(t)
	_ = a.AddGroupPolicy(GroupPolicy{Subject: "admin1", Role: RoleAdmin, Domain: "d1"})

	if err := a.Authorize("admin1", "d1", "anything", ActionAll); err != nil {
		t.Errorf("expected allowed, got: %v", err)
	}
}

func TestAuthz_Apply_RemoveAndAddPolicies(t *testing.T) {
	a := newTestAuthz(t)
	_ = a.AddPolicy(Policy{"*", "*", "res.get", ActionRead, EffectAllow})

	err := a.Apply(PolicyUpdate{
		RemovePolicies: []Policy{{"*", "*", "res.get", ActionRead, EffectAllow}},
		AddPolicies:    []Policy{{"*", "*", "res.put", ActionExecute, EffectAllow}},
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	ok, _ := a.Enforce("anyone", "dom", "res.get", ActionRead)
	if ok {
		t.Error("expected removed policy to deny access")
	}

	ok, _ = a.Enforce("anyone", "dom", "res.put", ActionExecute)
	if !ok {
		t.Error("expected added policy to allow access")
	}
}

func TestAuthz_EnforceEx_Error(t *testing.T) {
	a, _ := NewAuthz(EnforcerConfig{
		ModelString: "",
		Policies:    []Policy{},
	})

	result, err := a.EnforceEx("nobody", "dom", "secret", ActionRead)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Error("expected denied for nobody")
	}
	if result.Subject != "nobody" || result.Domain != "dom" {
		t.Errorf("unexpected result fields: %+v", result)
	}
}

func TestAuthz_Apply_RemovePolicies(t *testing.T) {
	a := newTestAuthz(
		t,
		Policy{"*", "*", "res.action", ActionExecute, EffectAllow},
	)

	if err := a.Apply(PolicyUpdate{
		RemovePolicies: []Policy{{"*", "*", "res.action", ActionExecute, EffectAllow}},
	}); err != nil {
		t.Fatalf("Apply remove policies: %v", err)
	}

	ok, _ := a.Enforce("anyone", "dom", "res.action", ActionExecute)
	if ok {
		t.Error("expected denied after removing policy")
	}
}

func TestNewAuthz_WithGroups(t *testing.T) {
	_, err := NewAuthz(EnforcerConfig{
		Groups: []GroupPolicy{
			{Subject: "p1", Role: RoleOwner, Domain: "g1"},
		},
	})
	if err != nil {
		t.Fatalf("NewAuthz with groups: %v", err)
	}
}

func TestNewAuthz_InvalidModel(t *testing.T) {
	_, err := NewAuthz(EnforcerConfig{ModelString: "not a valid model"})
	if err == nil {
		t.Error("expected error for invalid model")
	}
}

func TestNewAuthz_EmptyModelString(t *testing.T) {
	a, err := NewAuthz(EnforcerConfig{ModelString: ""})
	if err != nil {
		t.Fatalf("NewAuthz with empty model string: %v", err)
	}
	if a == nil {
		t.Error("expected non-nil Authz with default model fallback")
	}
}
