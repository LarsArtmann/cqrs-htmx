package identitymodel

import (
	"encoding/json/v2"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

func TestMutatePolicy_UninitializedEnforcer(t *testing.T) {
	a := &Authz{}
	p := Policy{Subject: RoleUser, Domain: "tenant1", Object: "resource", Action: ActionRead, Effect: EffectAllow}
	err := a.mutatePolicy(p, "add", func(...any) (bool, error) { return true, nil })
	if !errors.Is(err, ErrEnforcerNotInitialized) {
		t.Fatalf("expected ErrEnforcerNotInitialized, got %v", err)
	}
}

func TestMutatePolicy_Error(t *testing.T) {
	a, err := NewAuthz()
	if err != nil {
		t.Fatalf("NewAuthz: %v", err)
	}
	p := Policy{Subject: RoleUser, Domain: "tenant1", Object: "resource", Action: ActionRead, Effect: EffectAllow}
	wantErr := errors.New("casbin boom")
	err = a.mutatePolicy(p, "add", func(...any) (bool, error) { return false, wantErr })
	if !errorfamily.IsClass(err, event.Transient) {
		t.Fatalf("expected Transient error, got %v", err)
	}
	if err == nil || !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped casbin error, got %v", err)
	}
}

func TestMutatePolicy_Success(t *testing.T) {
	a, err := NewAuthz()
	if err != nil {
		t.Fatalf("NewAuthz: %v", err)
	}
	p := Policy{Subject: RoleUser, Domain: "tenant1", Object: "resource", Action: ActionRead, Effect: EffectAllow}
	err = a.mutatePolicy(p, "add", func(...any) (bool, error) { return true, nil })
	if err != nil {
		t.Fatalf("mutatePolicy add: %v", err)
	}
}

func TestMutateGroupPolicy_UninitializedEnforcer(t *testing.T) {
	a := &Authz{}
	g := GroupPolicy{Subject: "user1", Role: RoleUser, Domain: "tenant1"}
	err := a.mutateGroupPolicy(g, "add", func(...any) (bool, error) { return true, nil })
	if !errors.Is(err, ErrEnforcerNotInitialized) {
		t.Fatalf("expected ErrEnforcerNotInitialized, got %v", err)
	}
}

func TestMutateGroupPolicy_Error(t *testing.T) {
	a, err := NewAuthz()
	if err != nil {
		t.Fatalf("NewAuthz: %v", err)
	}
	g := GroupPolicy{Subject: "user1", Role: RoleUser, Domain: "tenant1"}
	wantErr := errors.New("casbin boom")
	err = a.mutateGroupPolicy(g, "add", func(...any) (bool, error) { return false, wantErr })
	if !errorfamily.IsClass(err, event.Transient) {
		t.Fatalf("expected Transient error, got %v", err)
	}
	if err == nil || !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped casbin error, got %v", err)
	}
}

func TestMutateGroupPolicy_Success(t *testing.T) {
	a, err := NewAuthz()
	if err != nil {
		t.Fatalf("NewAuthz: %v", err)
	}
	g := GroupPolicy{Subject: "user1", Role: RoleUser, Domain: "tenant1"}
	err = a.mutateGroupPolicy(g, "add", func(...any) (bool, error) { return true, nil })
	if err != nil {
		t.Fatalf("mutateGroupPolicy add: %v", err)
	}
}

func TestGetPolicies_UninitializedEnforcer(t *testing.T) {
	a := &Authz{}
	_, err := a.getPolicies(func() ([][]string, error) { return nil, nil }, "get policies")
	if !errors.Is(err, ErrEnforcerNotInitialized) {
		t.Fatalf("expected ErrEnforcerNotInitialized, got %v", err)
	}
}

func TestGetPolicies_Error(t *testing.T) {
	a, err := NewAuthz()
	if err != nil {
		t.Fatalf("NewAuthz: %v", err)
	}
	wantErr := errors.New("casbin boom")
	_, err = a.getPolicies(func() ([][]string, error) { return nil, wantErr }, "get policies")
	if !errorfamily.IsClass(err, event.Transient) {
		t.Fatalf("expected Transient error, got %v", err)
	}
	if err == nil || !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped casbin error, got %v", err)
	}
}

func TestGetPolicies_Success(t *testing.T) {
	a, err := NewAuthz()
	if err != nil {
		t.Fatalf("NewAuthz: %v", err)
	}
	policies, err := a.getPolicies(func() ([][]string, error) { return [][]string{{"p1"}}, nil }, "get policies")
	if err != nil {
		t.Fatalf("getPolicies: %v", err)
	}
	if len(policies) != 1 || policies[0][0] != "p1" {
		t.Fatalf("unexpected policies: %v", policies)
	}
}

func TestRolesForUser_UninitializedEnforcer(t *testing.T) {
	a := &Authz{}
	uid := GenerateUserID()
	tid := GenerateTenantID()
	_, err := a.rolesForUser(uid, tid, func(string, ...string) ([]string, error) { return nil, nil })
	if !errors.Is(err, ErrEnforcerNotInitialized) {
		t.Fatalf("expected ErrEnforcerNotInitialized, got %v", err)
	}
}

func TestRolesForUser_ConvertsAndFilters(t *testing.T) {
	a, err := NewAuthz()
	if err != nil {
		t.Fatalf("NewAuthz: %v", err)
	}
	uid := GenerateUserID()
	tid := GenerateTenantID()
	roles, err := a.rolesForUser(uid, tid, func(string, ...string) ([]string, error) {
		return []string{"admin", "viewer", "unknown-role"}, nil
	})
	if err != nil {
		t.Fatalf("rolesForUser: %v", err)
	}
	if len(roles) != 3 {
		t.Fatalf("expected 3 roles, got %d", len(roles))
	}
	if roles[0] != RoleAdmin {
		t.Fatalf("expected first role admin, got %v", roles[0])
	}
	if roles[1] != RoleViewer {
		t.Fatalf("expected second role viewer, got %v", roles[1])
	}
	if roles[2] != "unknown-role" {
		t.Fatalf("expected third role unchanged, got %v", roles[2])
	}
}

func TestRolesForUser_Error(t *testing.T) {
	a, err := NewAuthz()
	if err != nil {
		t.Fatalf("NewAuthz: %v", err)
	}
	uid := GenerateUserID()
	tid := GenerateTenantID()
	wantErr := errors.New("casbin boom")
	_, err = a.rolesForUser(uid, tid, func(string, ...string) ([]string, error) { return nil, wantErr })
	if !errorfamily.IsClass(err, event.Transient) {
		t.Fatalf("expected Transient error, got %v", err)
	}
	if err == nil || !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped casbin error, got %v", err)
	}
}

func TestMarshalJSONOrWrap_Success(t *testing.T) {
	b, err := marshalJSONOrWrap(map[string]string{"hello": "world"}, "code", "msg")
	if err != nil {
		t.Fatalf("marshalJSONOrWrap: %v", err)
	}
	var v map[string]string
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if v["hello"] != "world" {
		t.Fatalf("unexpected value: %v", v)
	}
}

func TestMarshalJSONOrWrap_Error(t *testing.T) {
	_, err := marshalJSONOrWrap(make(chan struct{}), "code", "msg")
	if err == nil {
		t.Fatal("expected error for unmarshalable value")
	}
	if !errorfamily.IsClass(err, event.Infrastructure) {
		t.Fatalf("expected Infrastructure error, got %v", err)
	}
}
