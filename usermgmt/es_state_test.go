package usermgmt

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

var testAggID = id.NewAggregateID()

func makeEvent(t *testing.T, eventType event.Type, version event.Version, payload any) event.Event {
	t.Helper()
	evt, err := event.NewEvent(eventType, testAggID, aggregateTypeUser, version, marshalPayload(payload))
	if err != nil {
		t.Fatalf("makeEvent %s: %v", eventType, err)
	}
	return evt
}

func TestFoldUser_EmptyState(t *testing.T) {
	state, err := foldUser(UserState{}, makeEvent(t, eventUserRegistered, 1, UserRegisteredPayload{
		Email:        "alice@example.com",
		DisplayName:  "Alice",
		PasswordHash: "$2a$12$hash",
		Roles:        []Role{RoleUser},
	}))
	if err != nil {
		t.Fatalf("foldUser: %v", err)
	}
	if state.Email != "alice@example.com" {
		t.Errorf("Email = %q, want alice@example.com", state.Email)
	}
	if state.DisplayName != "Alice" {
		t.Errorf("DisplayName = %q, want Alice", state.DisplayName)
	}
	if state.PasswordHash != "$2a$12$hash" {
		t.Errorf("PasswordHash = %q, want hash", state.PasswordHash)
	}
	if len(state.Roles) != 1 || state.Roles[0] != RoleUser {
		t.Errorf("Roles = %v, want [user]", state.Roles)
	}
	if state.Deleted {
		t.Errorf("Deleted = true, want false")
	}
}

func TestFoldUser_PasswordChanged(t *testing.T) {
	initial := UserState{
		Email: "bob@example.com", PasswordHash: "old", Roles: []Role{RoleUser},
	}
	state, err := foldUser(initial, makeEvent(t, eventPasswordChanged, 2, PasswordChangedPayload{
		PasswordHash: "new-hash",
	}))
	if err != nil {
		t.Fatalf("foldUser: %v", err)
	}
	if state.PasswordHash != "new-hash" {
		t.Errorf("PasswordHash = %q, want new-hash", state.PasswordHash)
	}
	if state.Email != "bob@example.com" {
		t.Errorf("Email changed: %q", state.Email)
	}
}

func TestFoldUser_RolesUpdated(t *testing.T) {
	initial := UserState{
		Email: "carol@example.com", PasswordHash: "h", Roles: []Role{RoleUser},
	}
	state, err := foldUser(initial, makeEvent(t, eventRolesUpdated, 2, RolesUpdatedPayload{
		Roles:  []Role{RoleAdmin, RoleOwner},
		Domain: "domain1",
	}))
	if err != nil {
		t.Fatalf("foldUser: %v", err)
	}
	if len(state.Roles) != 2 || state.Roles[0] != RoleAdmin || state.Roles[1] != RoleOwner {
		t.Errorf("Roles = %v, want [admin owner]", state.Roles)
	}
}

func TestFoldUser_EmailChanged(t *testing.T) {
	initial := UserState{
		Email: "old@example.com", PasswordHash: "h", Roles: []Role{RoleUser},
	}
	state, err := foldUser(initial, makeEvent(t, eventEmailChanged, 2, EmailChangedPayload{
		Email: "new@example.com",
	}))
	if err != nil {
		t.Fatalf("foldUser: %v", err)
	}
	if state.Email != "new@example.com" {
		t.Errorf("Email = %q, want new@example.com", state.Email)
	}
}

func TestFoldUser_DisplayNameChanged(t *testing.T) {
	initial := UserState{
		Email: "d@example.com", DisplayName: "Old", PasswordHash: "h", Roles: []Role{RoleUser},
	}
	state, err := foldUser(initial, makeEvent(t, eventDisplayNameChanged, 2, DisplayNameChangedPayload{
		DisplayName: "New Name",
	}))
	if err != nil {
		t.Fatalf("foldUser: %v", err)
	}
	if state.DisplayName != "New Name" {
		t.Errorf("DisplayName = %q, want New Name", state.DisplayName)
	}
}

func TestFoldUser_UserDeleted(t *testing.T) {
	initial := UserState{
		Email: "e@example.com", PasswordHash: "h", Roles: []Role{RoleUser},
	}
	state, err := foldUser(initial, makeEvent(t, eventUserDeleted, 2, UserDeletedPayload{
		Reason: "GDPR",
	}))
	if err != nil {
		t.Fatalf("foldUser: %v", err)
	}
	if !state.Deleted {
		t.Errorf("Deleted = false, want true")
	}
	if state.DeleteReason != "GDPR" {
		t.Errorf("DeleteReason = %q, want GDPR", state.DeleteReason)
	}
}

func TestFoldUser_MultipleEvents(t *testing.T) {
	events := []event.Event{
		makeEvent(t, eventUserRegistered, 1, UserRegisteredPayload{
			Email: "multi@example.com", DisplayName: "Multi", PasswordHash: "h1",
			Roles: []Role{RoleViewer},
		}),
		makeEvent(t, eventPasswordChanged, 2, PasswordChangedPayload{PasswordHash: "h2"}),
		makeEvent(t, eventRolesUpdated, 3, RolesUpdatedPayload{
			Roles: []Role{RoleAdmin}, Domain: "dom",
		}),
		makeEvent(t, eventEmailChanged, 4, EmailChangedPayload{Email: "updated@example.com"}),
		makeEvent(t, eventDisplayNameChanged, 5, DisplayNameChangedPayload{DisplayName: "Updated"}),
	}
	state := UserState{}
	var err error
	for _, evt := range events {
		state, err = foldUser(state, evt)
		if err != nil {
			t.Fatalf("foldUser at event %s: %v", evt.Type(), err)
		}
	}
	if state.Email != "updated@example.com" {
		t.Errorf("Email = %q, want updated@example.com", state.Email)
	}
	if state.DisplayName != "Updated" {
		t.Errorf("DisplayName = %q, want Updated", state.DisplayName)
	}
	if state.PasswordHash != "h2" {
		t.Errorf("PasswordHash = %q, want h2", state.PasswordHash)
	}
	if len(state.Roles) != 1 || state.Roles[0] != RoleAdmin {
		t.Errorf("Roles = %v, want [admin]", state.Roles)
	}
}

func TestFoldUser_UnknownEvent(t *testing.T) {
	initial := UserState{
		Email: "u@example.com", PasswordHash: "h", Roles: []Role{RoleUser},
	}
	unknownEvt, err := event.NewEvent(
		event.Type("SomeFutureEvent"), testAggID, aggregateTypeUser, 2,
		marshalPayload(map[string]string{"data": "whatever"}),
	)
	if err != nil {
		t.Fatalf("create unknown event: %v", err)
	}
	state, err := foldUser(initial, unknownEvt)
	if err != nil {
		t.Fatalf("foldUser unknown event: %v", err)
	}
	if state.Email != "u@example.com" {
		t.Errorf("state changed on unknown event: %+v", state)
	}
}

func TestFoldUser_RolesSliceIsCopied(t *testing.T) {
	state, err := foldUser(UserState{}, makeEvent(t, eventUserRegistered, 1, UserRegisteredPayload{
		Email: "copy@example.com", PasswordHash: "h", Roles: []Role{RoleUser},
	}))
	if err != nil {
		t.Fatalf("foldUser: %v", err)
	}
	original := state.Roles[0]
	state.Roles[0] = RoleAdmin
	if original != RoleUser {
		t.Error("modifying state.Roles affected the event payload")
	}
}
