package usermgmt

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewUser_Defaults(t *testing.T) {
	u := NewUser(NewUserID("u1"), "test@example.com", "Test")
	if u.Email != "test@example.com" {
		t.Errorf("email = %q", u.Email)
	}
	if len(u.Roles) != 1 || u.Roles[0] != RoleViewer {
		t.Errorf("expected default [viewer] role, got %v", u.Roles)
	}
	if u.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
}

func TestUser_Clone_DeepCopy(t *testing.T) {
	u := &User{
		ID:    NewUserID("u1"),
		Email: "a@b.com",
		Roles: []Role{RoleAdmin},
		Credentials: []WebAuthnCredential{
			{ID: []byte{1}},
		},
	}
	cp := u.Clone()
	cp.Roles[0] = RoleViewer
	cp.Credentials[0].ID = []byte{2}

	if u.Roles[0] != RoleAdmin {
		t.Error("Clone did not deep copy Roles")
	}
	if u.Credentials[0].ID[0] != 1 {
		t.Error("Clone did not deep copy Credentials")
	}
}

func TestUser_HasCredential(t *testing.T) {
	u := &User{
		Credentials: []WebAuthnCredential{
			{ID: []byte{1, 2, 3}},
		},
	}
	if !u.HasCredential([]byte{1, 2, 3}) {
		t.Error("expected to have credential")
	}
	if u.HasCredential([]byte{9, 9}) {
		t.Error("expected to not have credential")
	}
}

func TestUser_HasCredential_Empty(t *testing.T) {
	u := &User{}
	if u.HasCredential([]byte{1}) {
		t.Error("expected false for user with no credentials")
	}
}

func TestUser_MarshalJSON(t *testing.T) {
	u := &User{
		ID:          NewUserID("u1"),
		Email:       "test@example.com",
		DisplayName: "Test",
		Roles:       []Role{RoleUser},
		Credentials: []WebAuthnCredential{{ID: []byte{1}}},
	}

	data, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	count, ok := m["credential_count"].(float64)
	if !ok || count != 1 {
		t.Errorf("credential_count = %v, want 1", m["credential_count"])
	}
}

func TestNewSession_GeneratesToken(t *testing.T) {
	s, err := NewSession(NewUserID("u1"), time.Hour)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if s.Token == "" {
		t.Error("expected non-empty token")
	}
	if s.UserID != NewUserID("u1") {
		t.Errorf("UserID = %q", s.UserID)
	}
	if !s.ExpiresAt.After(s.CreatedAt) {
		t.Error("ExpiresAt should be after CreatedAt")
	}
}

func TestSession_IsExpired(t *testing.T) {
	s := &Session{
		Token:     "x",
		ExpiresAt: time.Now().Add(-time.Hour),
	}
	if !s.IsExpired() {
		t.Error("expected expired session")
	}
}

func TestSession_Valid(t *testing.T) {
	s := &Session{
		Token:     "valid-token",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if !s.Valid("valid-token") {
		t.Error("expected valid session")
	}
	if s.Valid("wrong-token") {
		t.Error("expected invalid for wrong token")
	}
}

func TestUserReadModel_Count(t *testing.T) {
	rm := NewUserReadModel()
	if rm.Count() != 0 {
		t.Errorf("expected 0, got %d", rm.Count())
	}
}

func TestService_ReadModel(t *testing.T) {
	svc := newTestService(t)
	if svc.ReadModel() == nil {
		t.Error("expected non-nil ReadModel")
	}
}

func TestAuthz_NilEnforcer(t *testing.T) {
	a := &Authz{enforcer: nil}
	uid := NewUserID("u1")

	tests := []struct {
		name string
		fn   func() error
	}{
		{"Enforce", func() error { _, err := a.Enforce("s", "d", "o", ActionRead); return err }},
		{"EnforceAny", func() error { _, err := a.EnforceAny("s"); return err }},
		{"EnforceEx", func() error { _, err := a.EnforceEx("s", "d", "o", ActionRead); return err }},
		{"Apply", func() error { return a.Apply(PolicyUpdate{}) }},
		{"AddPolicy", func() error { return a.AddPolicy(Policy{}) }},
		{"RemovePolicy", func() error { return a.RemovePolicy(Policy{}) }},
		{"AddGroupPolicy", func() error { return a.AddGroupPolicy(GroupPolicy{}) }},
		{"RemoveGroupPolicy", func() error { return a.RemoveGroupPolicy(GroupPolicy{}) }},
		{"RemoveAllRolesForUser", func() error { return a.RemoveAllRolesForUser("user1") }},
		{"Policies", func() error { _, err := a.Policies(); return err }},
		{"GroupPolicies", func() error { _, err := a.GroupPolicies(); return err }},
		{"RolesForUser", func() error { _, err := a.RolesForUser(uid, "d"); return err }},
		{"ImplicitRolesForUser", func() error { _, err := a.ImplicitRolesForUser(uid, "d"); return err }},
		{"ImplicitPermissionsForUser", func() error { _, err := a.ImplicitPermissionsForUser(uid, "d"); return err }},
		{"DomainsForUser", func() error { _, err := a.DomainsForUser(uid); return err }},
		{"UsersForRole", func() error { _, err := a.UsersForRole(RoleAdmin, "d"); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fn(); err == nil {
				t.Error("expected error for nil enforcer")
			}
		})
	}
}
