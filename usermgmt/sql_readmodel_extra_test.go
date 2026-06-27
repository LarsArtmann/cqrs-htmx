package usermgmt

import (
	"testing"
)

// --- SQLMembershipReadModel ---

func TestSQLMembershipReadModel_LifecycleAndQuery(t *testing.T) {
	ctx := t.Context()
	db := newSQLTestDB(t)
	rm, err := NewSQLiteMembershipReadModel(db)
	if err != nil {
		t.Fatalf("NewSQLiteMembershipReadModel: %v", err)
	}

	// Add member.
	added := makeMembershipEvent(t, eventMemberAdded, 1, MemberAddedPayload{
		SchemaVersion: currentSchemaVersion,
		ActorKind:     "user",
		ActorID:       "user-123",
		TenantID:      "tenant-abc",
		Roles:         []Role{RoleAdmin},
	})
	if err := rm.Handle(ctx, added); err != nil {
		t.Fatalf("Handle MemberAdded: %v", err)
	}

	// Verify SQL store via FindByActorSQL.
	views, err := rm.FindByActorSQL(ctx, "user-123")
	if err != nil {
		t.Fatalf("FindByActorSQL: %v", err)
	}
	if len(views) != 1 {
		t.Fatalf("FindByActorSQL: got %d, want 1", len(views))
	}
	if views[0].ActorID != "user-123" {
		t.Errorf("actor_id = %q, want user-123", views[0].ActorID)
	}
	if views[0].TenantID != "tenant-abc" {
		t.Errorf("tenant_id = %q, want tenant-abc", views[0].TenantID)
	}

	// Change roles.
	changed := makeMembershipEvent(t, eventMemberRolesChanged, 2, MemberRolesChangedPayload{
		SchemaVersion: currentSchemaVersion,
		ActorKind:     "user",
		ActorID:       "user-123",
		TenantID:      "tenant-abc",
		Roles:         []Role{RoleUser},
	})
	if err := rm.Handle(ctx, changed); err != nil {
		t.Fatalf("Handle MemberRolesChanged: %v", err)
	}

	// Remove member.
	removed := makeMembershipEvent(t, eventMemberRemoved, 3, MemberRemovedPayload{
		SchemaVersion: currentSchemaVersion,
		ActorID:       "user-123",
		TenantID:      "tenant-abc",
	})
	if err := rm.Handle(ctx, removed); err != nil {
		t.Fatalf("Handle MemberRemoved: %v", err)
	}

	// SQL store should no longer have the row.
	views, err = rm.FindByActorSQL(ctx, "user-123")
	if err != nil {
		t.Fatalf("FindByActorSQL after remove: %v", err)
	}
	if len(views) != 0 {
		t.Errorf("FindByActorSQL after remove: got %d, want 0", len(views))
	}
}

// --- SQLTenantReadModel ---

func TestSQLTenantReadModel_LifecycleAndQuery(t *testing.T) {
	ctx := t.Context()
	db := newSQLTestDB(t)
	rm, err := NewSQLiteTenantReadModel(db)
	if err != nil {
		t.Fatalf("NewSQLiteTenantReadModel: %v", err)
	}

	// Create tenant.
	created := makeTenantEvent(t, eventTenantCreated, 1, TenantCreatedPayload{
		SchemaVersion: currentSchemaVersion,
		Name:          "acme",
		DisplayName:   "Acme Corp",
	})
	if err := rm.Handle(ctx, created); err != nil {
		t.Fatalf("Handle TenantCreated: %v", err)
	}

	// Verify SQL store via FindByNameSQL.
	views, err := rm.FindByNameSQL(ctx, "acme")
	if err != nil {
		t.Fatalf("FindByNameSQL: %v", err)
	}
	if len(views) != 1 {
		t.Fatalf("FindByNameSQL: got %d, want 1", len(views))
	}
	if views[0].Name != "acme" {
		t.Errorf("name = %q, want acme", views[0].Name)
	}
	if views[0].DisplayName != "Acme Corp" {
		t.Errorf("display_name = %q, want Acme Corp", views[0].DisplayName)
	}

	// Suspend.
	suspended := makeTenantEvent(t, eventTenantSuspended, 2, TenantSuspendedPayload{
		SchemaVersion: currentSchemaVersion,
		Reason:        "billing",
	})
	if err := rm.Handle(ctx, suspended); err != nil {
		t.Fatalf("Handle TenantSuspended: %v", err)
	}
	views, err = rm.FindByNameSQL(ctx, "acme")
	if err != nil {
		t.Fatalf("FindByNameSQL after suspend: %v", err)
	}
	if len(views) != 1 && !views[0].Suspended {
		t.Error("tenant not suspended in SQL after TenantSuspended")
	}

	// Reactivate.
	reactivated := makeTenantEvent(t, eventTenantReactivated, 3, TenantReactivatedPayload{
		SchemaVersion: currentSchemaVersion,
	})
	if err := rm.Handle(ctx, reactivated); err != nil {
		t.Fatalf("Handle TenantReactivated: %v", err)
	}

	// Delete.
	deleted := makeTenantEvent(t, eventTenantDeleted, 4, TenantDeletedPayload{
		SchemaVersion: currentSchemaVersion,
		Reason:        "gone",
	})
	if err := rm.Handle(ctx, deleted); err != nil {
		t.Fatalf("Handle TenantDeleted: %v", err)
	}

	// SQL store should no longer have the row.
	views, err = rm.FindByNameSQL(ctx, "acme")
	if err != nil {
		t.Fatalf("FindByNameSQL after delete: %v", err)
	}
	if len(views) != 0 {
		t.Errorf("FindByNameSQL after delete: got %d, want 0", len(views))
	}
}

// --- SQLBotReadModel ---

func TestSQLBotReadModel_LifecycleAndQuery(t *testing.T) {
	ctx := t.Context()
	db := newSQLTestDB(t)
	rm, err := NewSQLiteBotReadModel(db)
	if err != nil {
		t.Fatalf("NewSQLiteBotReadModel: %v", err)
	}

	// Register bot.
	registered := makeBotEvent(t, eventBotRegistered, 1, BotRegisteredPayload{
		SchemaVersion: currentSchemaVersion,
		Name:          "ci-bot",
		OwnerID:       NewUserID("user-1"),
		TokenHash:     []byte{1, 2, 3},
		Scopes:        []string{"read", "write"},
	})
	if err := rm.Handle(ctx, registered); err != nil {
		t.Fatalf("Handle BotRegistered: %v", err)
	}

	// Verify SQL store via FindByNameSQL.
	views, err := rm.FindByNameSQL(ctx, "ci-bot")
	if err != nil {
		t.Fatalf("FindByNameSQL: %v", err)
	}
	if len(views) != 1 {
		t.Fatalf("FindByNameSQL: got %d, want 1", len(views))
	}
	if views[0].Name != "ci-bot" {
		t.Errorf("name = %q, want ci-bot", views[0].Name)
	}
	if views[0].OwnerID != NewUserID("user-1").Get().String() {
		t.Errorf("owner_id = %q, want %s", views[0].OwnerID, NewUserID("user-1").Get().String())
	}
	if views[0].TokenHash != string([]byte{1, 2, 3}) {
		t.Errorf("token_hash mismatch")
	}

	// Delete bot.
	deleted := makeBotEvent(t, eventBotDeleted, 2, BotDeletedPayload{
		SchemaVersion: currentSchemaVersion,
		Reason:        "rotated",
	})
	if err := rm.Handle(ctx, deleted); err != nil {
		t.Fatalf("Handle BotDeleted: %v", err)
	}

	// SQL store should no longer have the row.
	views, err = rm.FindByNameSQL(ctx, "ci-bot")
	if err != nil {
		t.Fatalf("FindByNameSQL after delete: %v", err)
	}
	if len(views) != 0 {
		t.Errorf("FindByNameSQL after delete: got %d, want 0", len(views))
	}
}
