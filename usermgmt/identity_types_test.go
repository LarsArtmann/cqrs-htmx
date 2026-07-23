package usermgmt

import (
	"testing"
	"time"
)

func TestActorID_FromUser(t *testing.T) {
	uid := NewUserID("01HX...")
	aid := ActorIDFromUser(uid)
	if aid.Kind() != ActorUser {
		t.Errorf("Kind() = %v, want ActorUser", aid.Kind())
	}
	if aid.String() != uid.Get().String() {
		t.Errorf("String() = %q, want %q", aid.String(), uid.Get().String())
	}
	if aid.IsZero() {
		t.Error("IsZero() = true, want false")
	}
	if aid.PrefixedString() != "user:"+uid.Get().String() {
		t.Errorf("PrefixedString() = %q, want %q", aid.PrefixedString(), "user:"+uid.Get().String())
	}
}

func TestActorID_FromBot(t *testing.T) {
	bid := NewBotID("bot-01HX...")
	aid := ActorIDFromBot(bid)
	if aid.Kind() != ActorBot {
		t.Errorf("Kind() = %v, want ActorBot", aid.Kind())
	}
	if aid.String() != "bot-01HX..." {
		t.Errorf("String() = %q, want %q", aid.String(), "bot-01HX...")
	}
	if aid.PrefixedString() != "bot:bot-01HX..." {
		t.Errorf("PrefixedString() = %q, want %q", aid.PrefixedString(), "bot:bot-01HX...")
	}
}

func TestActorID_ZeroValue(t *testing.T) {
	var zero ActorID
	if !zero.IsZero() {
		t.Error("zero-value ActorID should report IsZero=true")
	}
}

func TestActorKind_String(t *testing.T) {
	tests := []struct {
		kind ActorKind
		want string
	}{
		{ActorUser, "user"},
		{ActorBot, "bot"},
		{ActorKind(99), "unknown"},
	}
	for _, tc := range tests {
		if got := tc.kind.String(); got != tc.want {
			t.Errorf("ActorKind(%d).String() = %q, want %q", tc.kind, got, tc.want)
		}
	}
}

func TestTenantID_Branded(t *testing.T) {
	tid := NewTenantID("tenant-01AAA")
	if tid.Get() != "tenant-01AAA" {
		t.Errorf("Get() = %q, want %q", tid.Get(), "tenant-01AAA")
	}
	if tid.IsZero() {
		t.Error("IsZero() = true, want false")
	}
}

func TestBotID_Branded(t *testing.T) {
	bid := NewBotID("bot-01BBB")
	if bid.Get() != "bot-01BBB" {
		t.Errorf("Get() = %q, want %q", bid.Get(), "bot-01BBB")
	}
}

func TestSessionOrigin_DirectLogin(t *testing.T) {
	uid := NewUserID("01JX...")
	origin := DirectLogin{AuthenticatedAs: ActorIDFromUser(uid)}
	if origin.AuthenticatedAs.Kind() != ActorUser {
		t.Error("DirectLogin should carry user actor kind")
	}
	var _ SessionOrigin = origin // interface satisfaction
}

func TestSessionOrigin_Impersonation(t *testing.T) {
	admin := NewUserID("01ADM...")
	imp := Impersonation{
		By:     ActorIDFromUser(admin),
		Reason: "investigating user report",
		At:     time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC),
	}
	if imp.By.String() != admin.Get().String() {
		t.Errorf("By.String() = %q, want %q", imp.By.String(), admin.Get().String())
	}
	if imp.Reason != "investigating user report" {
		t.Errorf("Reason = %q", imp.Reason)
	}
	var _ SessionOrigin = imp // interface satisfaction
}

func TestMembership_HasRole(t *testing.T) {
	m := Membership{
		ActorID:  ActorIDFromUser(NewUserID("u1")),
		TenantID: NewTenantID("t1"),
		Roles:    []Role{RoleAdmin},
	}
	if !m.HasRole(RoleAdmin) {
		t.Error("HasRole(RoleAdmin) = false, want true")
	}
	if m.HasRole(RoleViewer) {
		t.Error("HasRole(RoleViewer) = true, want false")
	}
}

func TestMembership_HasAnyRole(t *testing.T) {
	m := Membership{
		ActorID:  ActorIDFromUser(NewUserID("u1")),
		TenantID: NewTenantID("t1"),
		Roles:    []Role{RoleUser},
	}
	if !m.HasAnyRole(RoleAdmin, RoleUser) {
		t.Error("HasAnyRole(Admin, User) = false, want true")
	}
	if m.HasAnyRole(RoleAdmin, RoleViewer) {
		t.Error("HasAnyRole(Admin, Viewer) = true, want false")
	}
}

func TestRoleHierarchy_SuperAdminInheritsAdmin(t *testing.T) {
	az, err := NewAuthz()
	if err != nil {
		t.Fatalf("NewAuthz failed: %v", err)
	}

	if err := az.AddGroupPolicy(
		GroupPolicy{Subject: "user-001", Role: RoleSuperAdmin, Domain: "tenant-X"},
	); err != nil {
		t.Fatalf("AddGroupPolicy failed: %v", err)
	}

	// super_admin inherits admin via g2, and admin has wildcard allow
	allowed, err := az.Enforce("user-001", "tenant-X", "anything", ActionExecute)
	if err != nil {
		t.Fatalf("Enforce failed: %v", err)
	}
	if !allowed {
		t.Error("super_admin should inherit admin's wildcard access via g2 hierarchy")
	}
}

func TestRoleHierarchy_AdminInheritsViewer(t *testing.T) {
	az, err := NewAuthz()
	if err != nil {
		t.Fatalf("NewAuthz failed: %v", err)
	}

	// Add a viewer-only policy
	if err := az.AddPolicy(
		Policy{Subject: RoleViewer, Domain: "tenant-Y", Object: "documents", Action: ActionRead, Effect: EffectAllow},
	); err != nil {
		t.Fatalf("AddPolicy failed: %v", err)
	}

	// Assign user admin role
	if err := az.AddGroupPolicy(GroupPolicy{Subject: "user-002", Role: RoleAdmin, Domain: "tenant-Y"}); err != nil {
		t.Fatalf("AddGroupPolicy failed: %v", err)
	}

	// admin inherits user > viewer, so should be able to read documents
	allowed, err := az.Enforce("user-002", "tenant-Y", "documents", ActionRead)
	if err != nil {
		t.Fatalf("Enforce failed: %v", err)
	}
	if !allowed {
		t.Error("admin should inherit viewer's read access via g2 hierarchy")
	}
}
