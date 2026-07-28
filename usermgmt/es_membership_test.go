package usermgmt

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

var membershipTestAggID = id.NewStreamID() //nolint:gochecknoglobals // test fixture

func makeMembershipEvent(
	t *testing.T, eventType event.Type, version event.Version, payload any,
) event.Event {
	t.Helper()
	payloadBytes, err := marshalPayload(payload)
	if err != nil {
		t.Fatalf("marshal payload for %s: %v", eventType, err)
	}
	evt, err := event.NewEvent(
		eventType, membershipTestAggID, aggregateTypeMembership, version, payloadBytes,
	)
	if err != nil {
		t.Fatalf("makeMembershipEvent %s: %v", eventType, err)
	}
	return evt
}

func TestMembershipState_FoldMemberAdded(t *testing.T) {
	evt := makeMembershipEvent(t, eventMemberAdded, 1, MemberAddedPayload{
		SchemaVersion: currentSchemaVersion,
		ActorKind:     "user",
		ActorID:       "u1",
		TenantID:      "t1",
		Roles:         []Role{RoleAdmin},
	})

	state, err := foldMembership(MembershipState{}, evt)
	if err != nil {
		t.Fatalf("foldMembership failed: %v", err)
	}
	if !state.Exists() {
		t.Error("membership should exist after MemberAdded")
	}
	if state.ActorID.String() != "u1" {
		t.Errorf("ActorID = %q, want %q", state.ActorID.String(), "u1")
	}
	if state.TenantID.Get() != "t1" {
		t.Errorf("TenantID = %q, want %q", state.TenantID.Get(), "t1")
	}
	if !state.HasRole(RoleAdmin) {
		t.Error("should have admin role")
	}
}

func TestMembershipState_FoldRolesChanged(t *testing.T) {
	added := makeMembershipEvent(t, eventMemberAdded, 1, MemberAddedPayload{
		SchemaVersion: currentSchemaVersion,
		ActorKind:     "user",
		ActorID:       "u1",
		TenantID:      "t1",
		Roles:         []Role{RoleAdmin},
	})
	changed := makeMembershipEvent(t, eventMemberRolesChanged, 2, MemberRolesChangedPayload{
		SchemaVersion: currentSchemaVersion,
		Roles:         []Role{RoleViewer},
	})

	state, err := foldMembership(MembershipState{}, added)
	if err != nil {
		t.Fatalf("fold MemberAdded: %v", err)
	}
	state, err = foldMembership(state, changed)
	if err != nil {
		t.Fatalf("fold MemberRolesChanged: %v", err)
	}
	if state.HasRole(RoleAdmin) {
		t.Error("should no longer have admin role after change")
	}
	if !state.HasRole(RoleViewer) {
		t.Error("should have viewer role after change")
	}
}

func TestMembershipState_FoldRemoved(t *testing.T) {
	added := makeMembershipEvent(t, eventMemberAdded, 1, MemberAddedPayload{
		SchemaVersion: currentSchemaVersion,
		ActorKind:     "user",
		ActorID:       "u1",
		TenantID:      "t1",
		Roles:         []Role{RoleUser},
	})
	removed := makeMembershipEvent(t, eventMemberRemoved, 2, MemberRemovedPayload{
		SchemaVersion: currentSchemaVersion,
	})

	state, err := foldMembership(MembershipState{}, added)
	if err != nil {
		t.Fatalf("fold MemberAdded: %v", err)
	}
	if !state.Exists() {
		t.Fatal("should exist before removal")
	}
	state, err = foldMembership(state, removed)
	if err != nil {
		t.Fatalf("fold MemberRemoved: %v", err)
	}
	if state.Exists() {
		t.Error("should not exist after removal")
	}
	if len(state.Roles) != 0 {
		t.Errorf("roles should be cleared, got %v", state.Roles)
	}
}

func TestMembershipState_FoldUnknownEvent(t *testing.T) {
	evt := makeMembershipEvent(t, event.Type("UnknownEvent"), 1, map[string]any{})
	_, err := foldMembership(MembershipState{}, evt)
	if err == nil {
		t.Error("expected error for unknown event type")
	}
}

func TestMembershipCommands_TypeAndAggregateID(t *testing.T) {
	aid := ActorIDFromUser(NewUserID("u1"))
	tid := NewTenantID("t1")

	addCmd := NewAddMemberCmd(aid, tid, []Role{RoleUser})
	if addCmd.Type() != cmdAddMember {
		t.Errorf("Type = %q, want %q", addCmd.Type(), cmdAddMember)
	}

	updateCmd := NewUpdateMemberRolesCmd(aid, tid, []Role{RoleAdmin})
	if updateCmd.Type() != cmdUpdateMemberRoles {
		t.Errorf("Type = %q, want %q", updateCmd.Type(), cmdUpdateMemberRoles)
	}
	if updateCmd.StreamID() != addCmd.StreamID() {
		t.Error("update and add commands should have same aggregate ID")
	}

	removeCmd := NewRemoveMemberCmd(aid, tid)
	if removeCmd.Type() != cmdRemoveMember {
		t.Errorf("Type = %q, want %q", removeCmd.Type(), cmdRemoveMember)
	}
	if removeCmd.StreamID() != addCmd.StreamID() {
		t.Error("remove and add commands should have same aggregate ID")
	}
}

func TestMembershipCommands_DeriveAggregateID_Deterministic(t *testing.T) {
	aid := ActorIDFromUser(NewUserID("u1"))
	tid := NewTenantID("t1")

	cmd1 := NewAddMemberCmd(aid, tid, nil)
	cmd2 := NewAddMemberCmd(aid, tid, nil)
	if cmd1.StreamID() != cmd2.StreamID() {
		t.Error("same actor+tenant should produce same aggregate ID")
	}

	cmd3 := NewAddMemberCmd(aid, NewTenantID("t2"), nil)
	if cmd1.StreamID() == cmd3.StreamID() {
		t.Error("different tenant should produce different aggregate ID")
	}
}
