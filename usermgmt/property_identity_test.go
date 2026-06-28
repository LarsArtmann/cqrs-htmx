package usermgmt

import (
	"testing"

	"pgregory.net/rapid"
)

// Property-based tests for foldTenant, foldBot, foldMembership.
// Uses helpers from property_extras_test.go (mustPropTenantEvent, etc.).

// TestFoldTenantProperty_CreateSetsName invariants: created tenant has the
// payload name/displayName, and is not suspended/deleted.
func TestFoldTenantProperty_CreateSetsName(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		name := rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "name")
		display := rapid.StringMatching(`[A-Z][a-z]{3,10}`).Draw(t, "displayName")

		evt := mustPropTenantEvent(eventTenantCreated, 1, TenantCreatedPayload{
			SchemaVersion: currentSchemaVersion,
			Name:          name,
			DisplayName:   display,
		})
		state, err := foldTenant(TenantState{}, evt)
		if err != nil {
			t.Fatalf("foldTenant created: %v", err)
		}
		if state.Name != name {
			t.Errorf("name = %q, want %q", state.Name, name)
		}
		if state.DisplayName != display {
			t.Errorf("displayName = %q, want %q", state.DisplayName, display)
		}
		if state.Suspended || state.Deleted {
			t.Error("new tenant should not be suspended/deleted")
		}
		if !state.IsValid() {
			t.Error("state should be valid")
		}
	})
}

// TestFoldTenantProperty_SuspendReactivate invariants: suspend sets Suspended,
// reactivate clears it; IsValid always holds.
func TestFoldTenantProperty_SuspendReactivate(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		reason := rapid.StringMatching(`[a-z ]{5,20}`).Draw(t, "reason")

		created := mustPropTenantEvent(eventTenantCreated, 1, TenantCreatedPayload{
			SchemaVersion: currentSchemaVersion,
			Name:          "prop-tenant",
			DisplayName:   "Prop",
		})
		state, _ := foldTenant(TenantState{}, created)

		suspended := mustPropTenantEvent(eventTenantSuspended, 2, TenantSuspendedPayload{
			SchemaVersion: currentSchemaVersion,
			Reason:        reason,
		})
		state, err := foldTenant(state, suspended)
		if err != nil {
			t.Fatalf("foldTenant suspend: %v", err)
		}
		if !state.Suspended {
			t.Error("should be suspended")
		}
		if state.SuspendReason != reason {
			t.Errorf("reason = %q, want %q", state.SuspendReason, reason)
		}

		reactivated := mustPropTenantEvent(eventTenantReactivated, 3, TenantReactivatedPayload{
			SchemaVersion: currentSchemaVersion,
		})
		state, err = foldTenant(state, reactivated)
		if err != nil {
			t.Fatalf("foldTenant reactivate: %v", err)
		}
		if state.Suspended {
			t.Error("should not be suspended after reactivate")
		}
		if state.SuspendReason != "" {
			t.Error("reason should be cleared")
		}
		if !state.IsValid() {
			t.Error("state should be valid throughout lifecycle")
		}
	})
}

// TestFoldBotProperty_RegisterSetsFields invariants: registered bot has name,
// owner, token hash, scopes from payload.
func TestFoldBotProperty_RegisterSetsFields(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		name := rapid.StringMatching(`[a-z-]{3,12}`).Draw(t, "name")
		scopeCount := rapid.IntRange(0, 4).Draw(t, "scopeCount")

		evt := mustPropBotEvent(eventBotRegistered, 1, BotRegisteredPayload{
			SchemaVersion: currentSchemaVersion,
			Name:          name,
			OwnerID:       NewUserID("01JXPROPOWNER000000000001"),
			TokenHash:     []byte{0xAB, 0xCD},
			Scopes:        make([]string, scopeCount),
		})
		state, err := foldBot(BotState{}, evt)
		if err != nil {
			t.Fatalf("foldBot registered: %v", err)
		}
		if state.Name != name {
			t.Errorf("name = %q, want %q", state.Name, name)
		}
		if len(state.Scopes) != scopeCount {
			t.Errorf("scopes len = %d, want %d", len(state.Scopes), scopeCount)
		}
	})
}

// TestFoldMembershipProperty_AddSetsRoles invariants: added membership has
// the roles from the payload and is not removed.
func TestFoldMembershipProperty_AddSetsRoles(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		roles := rapid.SliceOfN(rapidRole(), 1, 3).Draw(t, "roles")

		evt := mustPropMembershipEvent(eventMemberAdded, 1, MemberAddedPayload{
			SchemaVersion: currentSchemaVersion,
			ActorKind:     "user",
			ActorID:       "01JXPROPMEMBER000000000001",
			TenantID:      "01JXPROPTENANT000000000001",
			Roles:         roles,
		})
		state, err := foldMembership(MembershipState{}, evt)
		if err != nil {
			t.Fatalf("foldMembership added: %v", err)
		}
		if state.Removed {
			t.Error("membership should not be removed after add")
		}
		if len(state.Roles) != len(roles) {
			t.Errorf("roles len = %d, want %d", len(state.Roles), len(roles))
		}
	})
}

// TestFoldMembershipProperty_RemoveClearsExist invariants: removing a member
// sets Removed=true.
func TestFoldMembershipProperty_RemoveClearsExist(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		add := mustPropMembershipEvent(eventMemberAdded, 1, MemberAddedPayload{
			SchemaVersion: currentSchemaVersion,
			ActorKind:     "user",
			ActorID:       "01JXPROPMEMBER000000000002",
			TenantID:      "01JXPROPTENANT000000000002",
			Roles:         []Role{RoleAdmin},
		})
		state, _ := foldMembership(MembershipState{}, add)
		if state.Removed {
			t.Fatal("should not be removed after add")
		}

		remove := mustPropMembershipEvent(eventMemberRemoved, 2, MemberRemovedPayload{
			SchemaVersion: currentSchemaVersion,
			ActorID:       "01JXPROPMEMBER000000000002",
			TenantID:      "01JXPROPTENANT000000000002",
		})
		state, err := foldMembership(state, remove)
		if err != nil {
			t.Fatalf("foldMembership remove: %v", err)
		}
		if !state.Removed {
			t.Error("should be removed after remove event")
		}
	})
}

// TestFoldTenantProperty_UnknownEventReturnsError verifies that foldTenant
// rejects unknown event types.
func TestFoldTenantProperty_UnknownEventReturnsError(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		evt := mustPropTenantEvent("UnknownTenantEvent", 1, map[string]any{"x": 1})
		_, err := foldTenant(TenantState{}, evt)
		if err == nil {
			t.Error("expected error for unknown event type")
		}
	})
}
