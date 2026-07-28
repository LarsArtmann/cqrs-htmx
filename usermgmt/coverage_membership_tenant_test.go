package usermgmt

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// TestMembershipService_AddUpdateRemove exercises the Service.AddMember,
// UpdateMemberRoles, and RemoveMember write-path methods through full
// command dispatch. Also covers TenantMembers read and the revokeMembershipsForTenant
// best-effort cleanup on tenant deletion.
func TestMembershipService_AddUpdateRemove(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	// Register a user
	userID := registerIdentityTestUser(t, svc, "member@test.com")

	// Create a tenant
	tenant, err := svc.CreateTenant(ctx, CreateTenantRequest{
		ID:          NewTenantID("01JXTENANT0000000000000001"),
		Name:        "acme",
		DisplayName: "Acme Corp",
	})
	if err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}

	actor := ActorIDFromUser(userID)

	// Add member
	err = svc.AddMember(ctx, actor, tenant.ID, []Role{RoleAdmin})
	if err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	// Verify membership is visible
	members := svc.TenantMembers(ctx, tenant.ID)
	if len(members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(members))
	}

	// Update roles
	err = svc.UpdateMemberRoles(ctx, actor, tenant.ID, []Role{RoleUser, RoleViewer})
	if err != nil {
		t.Fatalf("UpdateMemberRoles: %v", err)
	}

	members = svc.TenantMembers(ctx, tenant.ID)
	if len(members) != 1 {
		t.Fatalf("expected 1 member after update, got %d", len(members))
	}

	// Remove member
	err = svc.RemoveMember(ctx, actor, tenant.ID)
	if err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}

	members = svc.TenantMembers(ctx, tenant.ID)
	if len(members) != 0 {
		t.Fatalf("expected 0 members after remove, got %d", len(members))
	}
}

// TestTenantService_SuspendReactivateDelete exercises the full tenant lifecycle
// and covers revokeMembershipsForTenantBestEffort on delete.
func TestTenantService_SuspendReactivateDelete(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	tenant, err := svc.CreateTenant(ctx, CreateTenantRequest{
		ID:          NewTenantID("01JXTENANT0000000000000002"),
		Name:        "suspend-test",
		DisplayName: "Suspend Test",
	})
	if err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}

	// Add a member so revokeMembershipsForTenantBestEffort has work to do
	userID := registerIdentityTestUser(t, svc, "tenant-member@test.com")
	actor := ActorIDFromUser(userID)
	if err := svc.AddMember(ctx, actor, tenant.ID, []Role{RoleAdmin}); err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	// Suspend
	if err := svc.SuspendTenant(ctx, tenant.ID, "policy-violation"); err != nil {
		t.Fatalf("SuspendTenant: %v", err)
	}

	// Verify suspended
	got, err := svc.GetTenant(ctx, tenant.ID)
	if err != nil {
		t.Fatalf("GetTenant: %v", err)
	}
	if !got.Suspended {
		t.Error("expected tenant to be suspended")
	}

	// Reactivate
	if err := svc.ReactivateTenant(ctx, tenant.ID); err != nil {
		t.Fatalf("ReactivateTenant: %v", err)
	}

	got, err = svc.GetTenant(ctx, tenant.ID)
	if err != nil {
		t.Fatalf("GetTenant after reactivate: %v", err)
	}
	if got.Suspended {
		t.Error("expected tenant to be reactivated")
	}

	// Delete — exercises revokeMembershipsForTenantBestEffort
	if err := svc.DeleteTenant(ctx, tenant.ID, "cleanup"); err != nil {
		t.Fatalf("DeleteTenant: %v", err)
	}

	// Members should be cleaned up
	members := svc.TenantMembers(ctx, tenant.ID)
	if len(members) != 0 {
		t.Errorf("expected 0 members after tenant delete, got %d", len(members))
	}
}

// TestAuthzPolicies_ErrorWrapping exercises the wrapCasbinError, wrapPolicyError,
// and wrapGroupError helpers by triggering Casbin errors via nil enforcer operations.
func TestAuthzPolicies_ErrorWrapping(t *testing.T) {
	// Create an Authz with a nil inner enforcer to trigger error paths
	a := &Authz{} // zero-value: inner enforcer is nil

	// Apply should return an error (nil enforcer)
	err := a.Apply(PolicyUpdate{})
	if err == nil {
		t.Error("expected error from Apply with nil enforcer")
	}

	// AddPolicy should error
	err = a.AddPolicy(Policy{})
	if err == nil {
		t.Error("expected error from AddPolicy with nil enforcer")
	}

	// RemovePolicy should error
	err = a.RemovePolicy(Policy{})
	if err == nil {
		t.Error("expected error from RemovePolicy with nil enforcer")
	}

	// AddGroupPolicy should error
	err = a.AddGroupPolicy(GroupPolicy{})
	if err == nil {
		t.Error("expected error from AddGroupPolicy with nil enforcer")
	}

	// RemoveGroupPolicy should error
	err = a.RemoveGroupPolicy(GroupPolicy{})
	if err == nil {
		t.Error("expected error from RemoveGroupPolicy with nil enforcer")
	}
}

// TestAuthzRoles_NilEnforcer exercises the role-query methods with a nil
// enforcer, covering their error-return paths.
func TestAuthzRoles_NilEnforcer(t *testing.T) {
	a := &Authz{}
	uid := NewUserID("01JXNILTESTUSER00000000001")
	tid := NewTenantID("01JXNILTESTTENANT0000000001")

	if roles, err := a.RolesForUser(uid, tid); err == nil && len(roles) != 0 {
		t.Errorf("expected empty roles for nil enforcer, got %v", roles)
	}

	if roles, err := a.ImplicitRolesForUser(uid, tid); err == nil && len(roles) != 0 {
		t.Errorf("expected empty implicit roles, got %v", roles)
	}

	if perms, err := a.ImplicitPermissionsForUser(uid, tid); err == nil && len(perms) != 0 {
		t.Errorf("expected empty perms, got %v", perms)
	}

	if domains, err := a.DomainsForUser(uid); err == nil && len(domains) != 0 {
		t.Errorf("expected empty domains, got %v", domains)
	}

	if users, err := a.UsersForRole(RoleAdmin, tid); err == nil && len(users) != 0 {
		t.Errorf("expected empty users, got %v", users)
	}
}

// TestAPITokenMiddleware_ContextHelpers covers WithBot and BotFromContext.
func TestAPITokenMiddleware_ContextHelpers(t *testing.T) {
	ctx := context.Background()

	// No bot in context → BotFromContext returns nil + false
	bot, ok := BotFromContext(ctx)
	if ok {
		t.Error("expected ok=false for empty context")
	}
	if bot != nil {
		t.Error("expected nil bot from empty context")
	}

	// Set a bot and retrieve it
	testBot := &Bot{
		ID:      NewBotID("01JXBOT00000000000000000001"),
		Name:    "test-bot",
		OwnerID: NewUserID("01JXOWNER00000000000000001"),
	}
	ctx = WithBot(ctx, testBot)
	got, ok := BotFromContext(ctx)
	if !ok {
		t.Fatal("expected ok=true after WithBot")
	}
	if got.ID != testBot.ID {
		t.Errorf("bot ID = %v, want %v", got.ID, testBot.ID)
	}
}

// TestBotService_RegisterDelete exercises the full bot lifecycle through
// the Service layer, covering RegisterBot, GetBot, ResolveBotByToken,
// and DeleteBot.
func TestBotService_RegisterDelete(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	ownerID := registerIdentityTestUser(t, svc, "bot-owner@test.com")

	// Register a bot
	result, err := svc.RegisterBot(ctx, RegisterBotRequest{
		ID:      NewBotID("01JXBOT00000000000000000002"),
		Name:    "ci-bot",
		OwnerID: ownerID,
		Scopes:  []string{"read", "write"},
	})
	if err != nil {
		t.Fatalf("RegisterBot: %v", err)
	}
	if result.Token == "" {
		t.Error("expected non-empty token")
	}

	// GetBot
	bot, err := svc.GetBot(ctx, result.Bot.ID)
	if err != nil {
		t.Fatalf("GetBot: %v", err)
	}
	if bot.Name != "ci-bot" {
		t.Errorf("bot name = %q, want %q", bot.Name, "ci-bot")
	}

	// ResolveBotByToken
	resolved, ok := svc.ResolveBotByToken(result.Token)
	if !ok {
		t.Fatal("expected to resolve bot by token")
	}
	if resolved.ID != result.Bot.ID {
		t.Errorf("resolved bot ID mismatch: %v != %v", resolved.ID, result.Bot.ID)
	}

	// DeleteBot
	if err := svc.DeleteBot(ctx, result.Bot.ID, "decommissioned"); err != nil {
		t.Fatalf("DeleteBot: %v", err)
	}

	// Token should no longer resolve
	_, ok = svc.ResolveBotByToken(result.Token)
	if ok {
		t.Error("expected token to not resolve after bot deletion")
	}
}

// TestTenantState_IsValid verifies the struct-level invariant method.
func TestTenantState_IsValid(t *testing.T) {
	tests := []struct {
		name string
		s    TenantState
		want bool
	}{
		{"empty", TenantState{}, true},
		{"created", TenantState{Name: "acme"}, true},
		{"suspended with reason", TenantState{Name: "acme", Suspended: true, SuspendReason: "violation"}, true},
		{"suspended without reason", TenantState{Name: "acme", Suspended: true}, true},
		{"deleted", TenantState{Name: "acme", Deleted: true, DeleteReason: "cleanup"}, true},
		{"deleted+suspended (impossible)", TenantState{Name: "acme", Deleted: true, Suspended: true}, false},
		{"suspended reason without suspended", TenantState{Name: "acme", SuspendReason: "leaked"}, false},
		{"delete reason without deleted", TenantState{Name: "acme", DeleteReason: "oops"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestTenantState_FoldDeleteClearsSuspended verifies the fold invariant:
// deleting a suspended tenant clears the Suspended flag.
func TestTenantState_FoldDeleteClearsSuspended(t *testing.T) {
	aggID := id.NewStreamID()
	suspended := makeEventFor(t, eventTenantSuspended, 2, aggID, aggregateTypeTenant, TenantSuspendedPayload{
		SchemaVersion: currentSchemaVersion,
		Reason:        "test",
	})
	state, err := foldTenant(TenantState{Name: "acme"}, suspended)
	if err != nil {
		t.Fatalf("foldTenant suspend: %v", err)
	}
	if !state.Suspended {
		t.Fatal("expected suspended state after suspend event")
	}

	deleted := makeEventFor(t, eventTenantDeleted, 3, aggID, aggregateTypeTenant, TenantDeletedPayload{
		SchemaVersion: currentSchemaVersion,
		Reason:        "cleanup",
	})
	state, err = foldTenant(state, deleted)
	if err != nil {
		t.Fatalf("foldTenant delete: %v", err)
	}
	if state.Suspended {
		t.Error("expected Suspended to be cleared after delete")
	}
	if !state.IsValid() {
		t.Error("state should be valid after fold sequence")
	}
}
