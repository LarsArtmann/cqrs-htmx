package usermgmt

import (
	"context"
	"slices"
	"testing"
)

func TestMembershipLifecycle_AddUpdateRemove(t *testing.T) {
	svc := newTestService(t)

	ctx := context.Background()
	actorID := ActorIDFromUser(NewUserID("user-lifecycle-1"))
	tenantID := NewTenantID("tenant-lifecycle-1")

	// 1. Add member with viewer role
	cmd := NewAddMemberCmd(actorID, tenantID, []Role{RoleViewer})
	if err := svc.dispatcher.Dispatch(ctx, cmd); err != nil {
		t.Fatalf("AddMember failed: %v", err)
	}

	// 2. Verify Casbin has viewer role
	assertRolesForActor(t, svc, actorID, tenantID, []Role{RoleViewer})

	// 3. Verify read model
	memberships := svc.membershipReadModel.FindByActor(actorID.String())
	if len(memberships) != 1 || !memberships[0].HasRole(RoleViewer) {
		t.Fatalf("expected 1 membership with viewer, got %+v", memberships)
	}

	// 4. Update roles to admin
	if err := svc.dispatcher.Dispatch(
		ctx,
		NewUpdateMemberRolesCmd(actorID, tenantID, []Role{RoleAdmin}),
	); err != nil {
		t.Fatalf("UpdateMemberRoles failed: %v", err)
	}

	// 5. Verify Casbin updated: admin yes, viewer no
	assertRolesForActor(t, svc, actorID, tenantID, []Role{RoleAdmin})
	assertRolesAbsent(t, svc, actorID, tenantID, []Role{RoleViewer})

	// 6. Remove member
	if err := svc.dispatcher.Dispatch(
		ctx,
		NewRemoveMemberCmd(actorID, tenantID),
	); err != nil {
		t.Fatalf("RemoveMember failed: %v", err)
	}

	// 7. Verify Casbin cleared: no roles at all
	assertRolesForActor(t, svc, actorID, tenantID, nil)

	// 8. Verify read model cleared
	memberships = svc.membershipReadModel.FindByActor(actorID.String())
	if len(memberships) != 0 {
		t.Errorf("expected 0 memberships after removal, got %d", len(memberships))
	}
}

func TestMembershipLifecycle_MultipleTenants(t *testing.T) {
	svc := newTestService(t)

	ctx := context.Background()
	actorID := ActorIDFromUser(NewUserID("user-multi-tenant"))
	tenantA := NewTenantID("tenant-A")
	tenantB := NewTenantID("tenant-B")

	// Add same actor to two different tenants with different roles
	if err := svc.dispatcher.Dispatch(
		ctx,
		NewAddMemberCmd(actorID, tenantA, []Role{RoleAdmin}),
	); err != nil {
		t.Fatalf("AddMember to tenantA failed: %v", err)
	}
	if err := svc.dispatcher.Dispatch(
		ctx,
		NewAddMemberCmd(actorID, tenantB, []Role{RoleViewer}),
	); err != nil {
		t.Fatalf("AddMember to tenantB failed: %v", err)
	}

	// Verify Casbin: admin in tenantA, viewer in tenantB
	assertRolesForActor(t, svc, actorID, tenantA, []Role{RoleAdmin})
	assertRolesForActor(t, svc, actorID, tenantB, []Role{RoleViewer})

	// Remove from tenantA only — tenantB should be unaffected
	if err := svc.dispatcher.Dispatch(
		ctx,
		NewRemoveMemberCmd(actorID, tenantA),
	); err != nil {
		t.Fatalf("RemoveMember from tenantA failed: %v", err)
	}

	// tenantA: no roles. tenantB: still viewer.
	assertRolesForActor(t, svc, actorID, tenantA, nil)
	assertRolesForActor(t, svc, actorID, tenantB, []Role{RoleViewer})
}

func TestMembershipLifecycle_DoubleAdd_Conflict(t *testing.T) {
	svc := newTestService(t)

	ctx := context.Background()
	actorID := ActorIDFromUser(NewUserID("user-double-add"))
	tenantID := NewTenantID("tenant-double")

	if err := svc.dispatcher.Dispatch(
		ctx,
		NewAddMemberCmd(actorID, tenantID, []Role{RoleUser}),
	); err != nil {
		t.Fatalf("first AddMember failed: %v", err)
	}

	err := svc.dispatcher.Dispatch(
		ctx,
		NewAddMemberCmd(actorID, tenantID, []Role{RoleAdmin}),
	)
	if err == nil {
		t.Error("expected conflict error on double add, got nil")
	}
}

func TestMembershipLifecycle_UpdateNonExistent_Rejection(t *testing.T) {
	svc := newTestService(t)

	ctx := context.Background()
	actorID := ActorIDFromUser(NewUserID("user-nonexistent"))
	tenantID := NewTenantID("tenant-nonexistent")

	err := svc.dispatcher.Dispatch(
		ctx,
		NewUpdateMemberRolesCmd(actorID, tenantID, []Role{RoleAdmin}),
	)
	if err == nil {
		t.Error("expected rejection error on update non-existent, got nil")
	}
}

// assertRolesForActor verifies that the actor has EXACTLY the given roles
// in the given tenant (via Casbin authz).
func assertRolesForActor(
	t *testing.T, svc *Service, actorID ActorID, tenantID TenantID, expected []Role,
) {
	t.Helper()
	roles, err := svc.authz.RolesForActor(actorID, tenantID)
	if err != nil {
		t.Fatalf("RolesForActor failed: %v", err)
	}
	if len(roles) != len(expected) {
		t.Errorf("RolesForActor(%s, %s) = %v, want %v", actorID.PrefixedString(), tenantID.Get(), roles, expected)
		return
	}
	for _, exp := range expected {
		if !slices.Contains(roles, exp) {
			t.Errorf("RolesForActor(%s, %s) = %v, missing %v", actorID.PrefixedString(), tenantID.Get(), roles, exp)
		}
	}
}

// assertRolesAbsent verifies that NONE of the given roles are assigned.
func assertRolesAbsent(
	t *testing.T, svc *Service, actorID ActorID, tenantID TenantID, absent []Role,
) {
	t.Helper()
	roles, err := svc.authz.RolesForActor(actorID, tenantID)
	if err != nil {
		t.Fatalf("RolesForActor failed: %v", err)
	}
	for _, unwanted := range absent {
		if slices.Contains(roles, unwanted) {
			t.Errorf("RolesForActor(%s, %s) = %v, should NOT contain %v",
				actorID.PrefixedString(), tenantID.Get(), roles, unwanted)
		}
	}
}
