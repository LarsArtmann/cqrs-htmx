package usermgmt

import (
	"context"
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

	// 2. Verify membership appears in read model
	memberships := svc.membershipReadModel.FindByActor(actorID.String())
	if len(memberships) != 1 {
		t.Fatalf("expected 1 membership, got %d", len(memberships))
	}
	if !memberships[0].HasRole(RoleViewer) {
		t.Error("expected viewer role")
	}

	// 3. Update roles to admin
	updateCmd := NewUpdateMemberRolesCmd(actorID, tenantID, []Role{RoleAdmin})
	if err := svc.dispatcher.Dispatch(ctx, updateCmd); err != nil {
		t.Fatalf("UpdateMemberRoles failed: %v", err)
	}

	// 4. Verify role change
	memberships = svc.membershipReadModel.FindByActor(actorID.String())
	if len(memberships) != 1 {
		t.Fatalf("expected 1 membership after update, got %d", len(memberships))
	}
	if !memberships[0].HasRole(RoleAdmin) {
		t.Error("expected admin role after update")
	}
	if memberships[0].HasRole(RoleViewer) {
		t.Error("should no longer have viewer role")
	}

	// 5. Remove member
	removeCmd := NewRemoveMemberCmd(actorID, tenantID)
	if err := svc.dispatcher.Dispatch(ctx, removeCmd); err != nil {
		t.Fatalf("RemoveMember failed: %v", err)
	}

	// 6. Verify membership removed
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

	// Verify two memberships exist
	memberships := svc.membershipReadModel.FindByActor(actorID.String())
	if len(memberships) != 2 {
		t.Fatalf("expected 2 memberships, got %d", len(memberships))
	}

	// Verify each has the correct roles
	foundAdmin, foundViewer := false, false
	for _, m := range memberships {
		if m.HasRole(RoleAdmin) {
			foundAdmin = true
		}
		if m.HasRole(RoleViewer) {
			foundViewer = true
		}
	}
	if !foundAdmin {
		t.Error("missing admin membership")
	}
	if !foundViewer {
		t.Error("missing viewer membership")
	}
}

func TestMembershipLifecycle_DoubleAdd_Conflict(t *testing.T) {
	svc := newTestService(t)

	ctx := context.Background()
	actorID := ActorIDFromUser(NewUserID("user-double-add"))
	tenantID := NewTenantID("tenant-double")

	// First add should succeed
	if err := svc.dispatcher.Dispatch(
		ctx,
		NewAddMemberCmd(actorID, tenantID, []Role{RoleUser}),
	); err != nil {
		t.Fatalf("first AddMember failed: %v", err)
	}

	// Second add should fail with conflict
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

	// Update on non-existent membership should fail
	err := svc.dispatcher.Dispatch(
		ctx,
		NewUpdateMemberRolesCmd(actorID, tenantID, []Role{RoleAdmin}),
	)
	if err == nil {
		t.Error("expected rejection error on update non-existent, got nil")
	}
}
