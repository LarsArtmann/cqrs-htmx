package usermgmt

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestNewService_CustomAuthz_ReceivesEventPolicies pins the wiring fix for the
// Authz split brain: when ServiceConfig.Authz is set, Service.Authz() must
// expose that same engine instance AND event-derived policies must reach it
// (register grants the base roles in the user's self-domain through the
// casbin projection).
//
// Before the fix, NewService built a replacement CasbinProjection around the
// injected engine but never registered it on the projection host — the host
// kept feeding the internally-created engine, so the injected one stayed
// empty forever while Service.Authz() exposed it as if it were live.
func TestNewService_CustomAuthz_ReceivesEventPolicies(t *testing.T) {
	injected := newTestAuthz(t)

	svc, err := NewService(ServiceConfig{Authz: injected})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	defer svc.Close() //nolint:errcheck // test cleanup

	if svc.Authz() != injected {
		t.Fatal("Service.Authz() must expose the injected engine instance")
	}

	uid := NewUserID(strings.Repeat("a", 26))
	if _, err := svc.Register(context.Background(), RegisterRequest{
		ID:    uid,
		Email: "authz-wiring@test.com",
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	selfDomain := NewTenantID(uid.Get().String())

	roles, err := injected.RolesForUser(uid, selfDomain)
	if err != nil {
		t.Fatalf("RolesForUser: %v", err)
	}

	if len(roles) == 0 {
		t.Fatal("injected engine received no event-derived roles — the Authz split brain is back")
	}
}

// TestNewService_CustomAuthz_MembershipPoliciesReachEngine covers the
// tenant-membership path from the consumer bug report: AddMember must land in
// the injected engine, not only in an internal one.
func TestNewService_CustomAuthz_MembershipPoliciesReachEngine(t *testing.T) {
	injected := newTestAuthz(t)

	svc, err := NewService(ServiceConfig{Authz: injected})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	defer svc.Close() //nolint:errcheck // test cleanup

	ownerID := NewUserID(strings.Repeat("b", 26))
	if _, err := svc.Register(context.Background(), RegisterRequest{
		ID:    ownerID,
		Email: "owner@test.com",
	}); err != nil {
		t.Fatalf("Register owner: %v", err)
	}

	memberID := NewUserID(strings.Repeat("c", 26))
	if _, err := svc.Register(context.Background(), RegisterRequest{
		ID:    memberID,
		Email: "member@test.com",
	}); err != nil {
		t.Fatalf("Register member: %v", err)
	}

	tenant, err := svc.CreateTenant(context.Background(), CreateTenantRequest{
		ID:   NewTenantID("01JXTENANT0000000000000002"),
		Name: "acme",
	})
	if err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}

	if err := svc.AddMember(
		context.Background(),
		ActorIDFromUser(memberID),
		tenant.ID,
		[]Role{RoleUser},
	); err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		roles, rolesErr := injected.RolesForUser(memberID, tenant.ID)
		if rolesErr != nil {
			t.Fatalf("RolesForUser: %v", rolesErr)
		}

		if len(roles) > 0 {
			return
		}

		if time.Now().After(deadline) {
			t.Fatal("injected engine never received the MemberAdded policy — the Authz split brain is back")
		}

		time.Sleep(10 * time.Millisecond)
	}
}
