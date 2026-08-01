package usermgmt

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// TestService_DeleteUser_CascadeMemberships verifies that deleting a user
// removes all their memberships across all tenants.
func TestService_DeleteUser_CascadeMemberships(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	userID := registerIdentityTestUser(t, svc, "cascade-member@test.com")
	actor := ActorIDFromUser(userID)

	tenantA, err := svc.CreateTenant(ctx, CreateTenantRequest{
		ID:   NewTenantID("01JXTENANT00000000000000A1"),
		Name: "tenant-a",
	})
	if err != nil {
		t.Fatalf("CreateTenant A: %v", err)
	}
	tenantB, err := svc.CreateTenant(ctx, CreateTenantRequest{
		ID:   NewTenantID("01JXTENANT00000000000000B2"),
		Name: "tenant-b",
	})
	if err != nil {
		t.Fatalf("CreateTenant B: %v", err)
	}

	if err := svc.AddMember(ctx, actor, tenantA.ID, []Role{RoleUser}); err != nil {
		t.Fatalf("AddMember A: %v", err)
	}
	if err := svc.AddMember(ctx, actor, tenantB.ID, []Role{RoleAdmin}); err != nil {
		t.Fatalf("AddMember B: %v", err)
	}

	if got := len(svc.membershipReadModel.FindByActor(userID.Get().String())); got != 2 {
		t.Fatalf("expected 2 memberships before delete, got %d", got)
	}

	if err := svc.DeleteUser(ctx, userID, "test cascade"); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}

	if got := len(svc.membershipReadModel.FindByActor(userID.Get().String())); got != 0 {
		t.Fatalf("expected 0 memberships after delete, got %d", got)
	}
}

// TestService_DeleteUser_CascadeBots verifies that deleting a user
// removes all bots owned by that user, preventing orphaned API tokens.
func TestService_DeleteUser_CascadeBots(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	userID := registerIdentityTestUser(t, svc, "cascade-bots@test.com")

	botID1 := NewBotID(id.NewStreamID().String())
	result1, err := svc.RegisterBot(ctx, RegisterBotRequest{
		ID:      botID1,
		Name:    "bot-1",
		OwnerID: userID,
		Scopes:  []string{"read"},
	})
	if err != nil {
		t.Fatalf("RegisterBot 1: %v", err)
	}

	botID2 := NewBotID(id.NewStreamID().String())
	_, err = svc.RegisterBot(ctx, RegisterBotRequest{
		ID:      botID2,
		Name:    "bot-2",
		OwnerID: userID,
		Scopes:  []string{"deploy"},
	})
	if err != nil {
		t.Fatalf("RegisterBot 2: %v", err)
	}

	if got := len(svc.botReadModel.FindByOwner(userID)); got != 2 {
		t.Fatalf("expected 2 bots before delete, got %d", got)
	}

	if err := svc.DeleteUser(ctx, userID, "test cascade"); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}

	if got := len(svc.botReadModel.FindByOwner(userID)); got != 0 {
		t.Fatalf("expected 0 bots after delete, got %d", got)
	}

	if _, ok := svc.botReadModel.FindByTokenHash(result1.Bot.TokenHash); ok {
		t.Fatal("deleted user's bot token should no longer authenticate")
	}
}
