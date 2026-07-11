package usermgmt

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

// makeEventFor constructs an event with the given aggregate ID and type.
// Used for membership/tenant/bot read model tests that need non-user aggregates.
func makeEventFor(
	t *testing.T,
	eventType event.Type,
	version event.Version,
	aggID id.AggregateID,
	aggType id.AggregateType,
	payload any,
) event.Event {
	t.Helper()
	payloadBytes, err := marshalPayload(payload)
	if err != nil {
		t.Fatalf("marshal payload for %s: %v", eventType, err)
	}
	evt, err := event.NewEvent(eventType, aggID, aggType, version, payloadBytes)
	if err != nil {
		t.Fatalf("makeEventFor %s: %v", eventType, err)
	}
	return evt
}

// TestSQLMembershipReadModel_HandleAndQuery exercises NewSQLiteMembershipReadModel,
// Handle, FindByActor, FindByTenant, and SQL query paths.
func TestSQLMembershipReadModel_HandleAndQuery(t *testing.T) {
	ctx := t.Context()
	db := newSQLTestDB(t)
	rm, err := NewSQLiteMembershipReadModel(db)
	if err != nil {
		t.Fatalf("NewSQLiteMembershipReadModel: %v", err)
	}

	tenantID := NewTenantID("01JXTENANT0000000000000000A")
	actor := ActorIDFromUser(NewUserID("01JXMEMBER0000000000000001"))
	aggID := deriveMembershipID(actor, tenantID)

	evt := makeEventFor(t, eventMemberAdded, 1, aggID, aggregateTypeMembership, MemberAddedPayload{
		SchemaVersion: currentSchemaVersion,
		ActorKind:     "user",
		ActorID:       actor.String(),
		TenantID:      tenantID.Get(),
		Roles:         []Role{RoleAdmin},
	})
	if err := rm.Handle(ctx, evt); err != nil {
		t.Fatalf("Handle MemberAdded: %v", err)
	}

	byActor := rm.FindByActor(actor.String())
	if len(byActor) != 1 {
		t.Fatalf("FindByActor: expected 1, got %d", len(byActor))
	}

	byTenant := rm.FindByTenant(tenantID.Get())
	if len(byTenant) != 1 {
		t.Fatalf("FindByTenant: expected 1, got %d", len(byTenant))
	}

	sqlByActor, err := rm.FindByActorSQL(ctx, actor.String())
	if err != nil {
		t.Fatalf("FindByActorSQL: %v", err)
	}
	if len(sqlByActor) != 1 {
		t.Fatalf("FindByActorSQL: expected 1, got %d", len(sqlByActor))
	}
}

// TestSQLTenantReadModel_HandleAndQuery exercises NewSQLiteTenantReadModel,
// Handle, FindByID, FindByName, and SQL query paths.
func TestSQLTenantReadModel_HandleAndQuery(t *testing.T) {
	ctx := t.Context()
	db := newSQLTestDB(t)
	rm, err := NewSQLiteTenantReadModel(db)
	if err != nil {
		t.Fatalf("NewSQLiteTenantReadModel: %v", err)
	}

	tenantID := NewTenantID("01JXTENANT0000000000000000B")
	aggID, err := id.ParseAggregateID(tenantID.Get())
	if err != nil {
		t.Fatalf("ParseAggregateID: %v", err)
	}

	evt := makeEventFor(t, eventTenantCreated, 1, aggID, aggregateTypeTenant, TenantCreatedPayload{
		SchemaVersion: currentSchemaVersion,
		Name:          "test-tenant",
		DisplayName:   "Test Tenant",
	})
	if err := rm.Handle(ctx, evt); err != nil {
		t.Fatalf("Handle TenantCreated: %v", err)
	}

	_, ok := rm.FindByID(aggID)
	if !ok {
		t.Fatal("FindByID: expected to find tenant")
	}

	_, ok = rm.FindByName("test-tenant")
	if !ok {
		t.Fatal("FindByName: expected to find tenant")
	}

	sqlTenants, err := rm.FindByNameSQL(ctx, "test-tenant")
	if err != nil {
		t.Fatalf("FindByNameSQL: %v", err)
	}
	if len(sqlTenants) != 1 {
		t.Fatalf("FindByNameSQL: expected 1, got %d", len(sqlTenants))
	}
	if sqlTenants[0].Name != "test-tenant" {
		t.Errorf("SQL tenant name = %q, want %q", sqlTenants[0].Name, "test-tenant")
	}
}

// TestSQLBotReadModel_HandleAndQuery exercises NewSQLiteBotReadModel,
// Handle, FindByTokenHash, and SQL query paths.
func TestSQLBotReadModel_HandleAndQuery(t *testing.T) {
	ctx := t.Context()
	db := newSQLTestDB(t)
	rm, err := NewSQLiteBotReadModel(db)
	if err != nil {
		t.Fatalf("NewSQLiteBotReadModel: %v", err)
	}

	botID := NewBotID("01JXBOT0000000000000000AB")
	ownerID := NewUserID("01JXBOTOWNER00000000000001")
	tokenHash := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	aggID, err := id.ParseAggregateID(botID.Get())
	if err != nil {
		t.Fatalf("ParseAggregateID: %v", err)
	}

	evt := makeEventFor(t, eventBotRegistered, 1, aggID, aggregateTypeBot, BotRegisteredPayload{
		SchemaVersion: currentSchemaVersion,
		Name:          "ci-bot",
		OwnerID:       ownerID,
		TokenHash:     tokenHash,
		Scopes:        []string{"read"},
	})
	if err := rm.Handle(ctx, evt); err != nil {
		t.Fatalf("Handle BotRegistered: %v", err)
	}

	_, ok := rm.FindByTokenHash(tokenHash)
	if !ok {
		t.Fatal("FindByTokenHash: expected to find bot")
	}

	sqlBots, err := rm.FindByNameSQL(ctx, "ci-bot")
	if err != nil {
		t.Fatalf("FindByNameSQL: %v", err)
	}
	if len(sqlBots) != 1 {
		t.Fatalf("FindByNameSQL: expected 1, got %d", len(sqlBots))
	}
	if sqlBots[0].Name != "ci-bot" {
		t.Errorf("SQL bot name = %q, want %q", sqlBots[0].Name, "ci-bot")
	}
}
