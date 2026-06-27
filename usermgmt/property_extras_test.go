package usermgmt

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"pgregory.net/rapid"
)

// Property-based tests for foldTenant, foldBot, foldMembership.
// Verifies key invariants with random inputs.

var (
	propTenantAggID = id.NewAggregateID() //nolint:gochecknoglobals // test fixture
	propBotAggID    = id.NewAggregateID() //nolint:gochecknoglobals // test fixture
)

// --- foldTenant property tests ---

func mustPropTenantEvent(eventType event.Type, version event.Version, payload any) event.Event {
	b, err := marshalPayload(payload)
	if err != nil {
		panic(err)
	}
	evt, err := event.NewEvent(eventType, propTenantAggID, aggregateTypeTenant, version, b)
	if err != nil {
		panic(err)
	}
	return evt
}

func TestFoldTenantProperty_CreatedSetsName(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		name := rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "name")
		display := rapid.StringMatching(`[A-Z][a-z]{3,10}`).Draw(t, "display")
		evt := mustPropTenantEvent(eventTenantCreated, 1, TenantCreatedPayload{
			SchemaVersion: currentSchemaVersion,
			Name:          name, DisplayName: display,
		})
		state, err := foldTenant(TenantState{}, evt)
		if err != nil {
			t.Fatalf("foldTenant: %v", err)
		}
		if state.Name != name {
			t.Errorf("Name = %q, want %q", state.Name, name)
		}
		if state.DisplayName != display {
			t.Errorf("DisplayName = %q, want %q", state.DisplayName, display)
		}
	})
}

func TestFoldTenantProperty_DeletedClearsSuspended(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		base := TenantState{
			Name:          "acme",
			DisplayName:   "Acme",
			Suspended:     true,
			SuspendReason: "billing",
		}
		evt := mustPropTenantEvent(eventTenantDeleted, 2, TenantDeletedPayload{
			SchemaVersion: currentSchemaVersion,
		})
		marked, _ := event.MarkTombstone(evt)
		state, err := foldTenant(base, marked)
		if err != nil {
			t.Fatalf("foldTenant: %v", err)
		}
		if !state.Deleted {
			t.Error("Deleted should be true")
		}
		if state.Suspended {
			t.Error("Suspended should be cleared on delete")
		}
		if state.SuspendReason != "" {
			t.Error("SuspendReason should be cleared on delete")
		}
	})
}

// --- foldBot property tests ---

func mustPropBotEvent(eventType event.Type, version event.Version, payload any) event.Event {
	b, err := marshalPayload(payload)
	if err != nil {
		panic(err)
	}
	evt, err := event.NewEvent(eventType, propBotAggID, aggregateTypeBot, version, b)
	if err != nil {
		panic(err)
	}
	return evt
}

func TestFoldBotProperty_RegisteredSetsFields(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		name := rapid.StringMatching(`[a-z]{3,10}-bot`).Draw(t, "name")
		scopes := rapid.SliceOf(rapid.StringMatching(`[a-z]{3,8}`)).Draw(t, "scopes")
		ownerID := NewUserID(rapid.StringMatching(`user-[0-9]+`).Draw(t, "owner"))
		hash := rapid.SliceOfN(rapid.Byte(), 16, 64).Draw(t, "hash")
		evt := mustPropBotEvent(eventBotRegistered, 1, BotRegisteredPayload{
			SchemaVersion: currentSchemaVersion,
			Name:          name, OwnerID: ownerID,
			TokenHash: hash, Scopes: scopes,
		})
		state, err := foldBot(BotState{}, evt)
		if err != nil {
			t.Fatalf("foldBot: %v", err)
		}
		if state.Name != name {
			t.Errorf("Name = %q, want %q", state.Name, name)
		}
		if state.OwnerID.Get().String() != ownerID.Get().String() {
			t.Errorf("OwnerID = %s, want %s", state.OwnerID.Get().String(), ownerID.Get().String())
		}
		if len(state.Scopes) != len(scopes) {
			t.Errorf("Scopes len = %d, want %d", len(state.Scopes), len(scopes))
		}
		if !state.Exists() {
			t.Error("bot should exist after registration")
		}
	})
}

func TestFoldBotProperty_DeletedSetsDeletedFlag(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		state := BotState{
			Name:      "ci-bot",
			OwnerID:   NewUserID("u1"),
			TokenHash: []byte{1, 2, 3},
		}
		evt := mustPropBotEvent(eventBotDeleted, 2, BotDeletedPayload{
			SchemaVersion: currentSchemaVersion,
			Reason:        rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "reason"),
		})
		marked, _ := event.MarkTombstone(evt)
		state, err := foldBot(state, marked)
		if err != nil {
			t.Fatalf("foldBot: %v", err)
		}
		if !state.Deleted {
			t.Error("Deleted should be true")
		}
		if state.Exists() {
			t.Error("deleted bot should not exist")
		}
	})
}

// --- foldMembership property tests ---

func mustPropMembershipEvent(eventType event.Type, version event.Version, payload any) event.Event {
	b, err := marshalPayload(payload)
	if err != nil {
		panic(err)
	}
	aggID := id.NewAggregateID()
	evt, err := event.NewEvent(eventType, aggID, aggregateTypeMembership, version, b)
	if err != nil {
		panic(err)
	}
	return evt
}

func TestFoldMembershipProperty_AddedSetsFields(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		actorRaw := rapid.StringMatching(`[a-z0-9]+`).Draw(t, "actor")
		tenantRaw := rapid.StringMatching(`t-[a-z0-9]+`).Draw(t, "tenant")
		roles := rapid.SliceOf(rapidRole()).Draw(t, "roles")
		evt := mustPropMembershipEvent(eventMemberAdded, 1, MemberAddedPayload{
			SchemaVersion: currentSchemaVersion,
			ActorKind:     actorKindUserStr,
			ActorID:       actorRaw, TenantID: tenantRaw, Roles: roles,
		})
		state, err := foldMembership(MembershipState{}, evt)
		if err != nil {
			t.Fatalf("foldMembership: %v", err)
		}
		if state.ActorID.String() != actorRaw {
			t.Errorf("ActorID = %q, want %q", state.ActorID.String(), actorRaw)
		}
		if state.TenantID.Get() != tenantRaw {
			t.Errorf("TenantID = %q, want %q", state.TenantID.Get(), tenantRaw)
		}
		if len(state.Roles) != len(roles) {
			t.Errorf("Roles len = %d, want %d", len(state.Roles), len(roles))
		}
	})
}

func TestFoldMembershipProperty_RolesChangedUpdatesRoles(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		base := MembershipState{
			ActorID:  NewActorID(ActorUser, "abc"),
			TenantID: NewTenantID("t1"),
			Roles:    []Role{RoleViewer},
		}
		newRoles := rapid.SliceOf(rapidRole()).Draw(t, "newRoles")
		evt := mustPropMembershipEvent(eventMemberRolesChanged, 2, MemberRolesChangedPayload{
			SchemaVersion: currentSchemaVersion,
			ActorID:       "abc", TenantID: "t1", Roles: newRoles,
		})
		state, err := foldMembership(base, evt)
		if err != nil {
			t.Fatalf("foldMembership: %v", err)
		}
		if len(state.Roles) != len(newRoles) {
			t.Errorf("Roles len = %d, want %d", len(state.Roles), len(newRoles))
		}
	})
}
