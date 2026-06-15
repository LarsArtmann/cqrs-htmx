package usermgmt

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

var testAggID = id.NewAggregateID() //nolint:gochecknoglobals // test fixture

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
		Email:       "alice@example.com",
		DisplayName: "Alice",
		Roles:       []Role{RoleUser},
	}))
	if err != nil {
		t.Fatalf("foldUser: %v", err)
	}
	if state.Email != "alice@example.com" {
		t.Errorf("Email = %q", state.Email)
	}
	if state.DisplayName != "Alice" {
		t.Errorf("DisplayName = %q", state.DisplayName)
	}
	if len(state.Roles) != 1 || state.Roles[0] != RoleUser {
		t.Errorf("Roles = %v", state.Roles)
	}
}

func TestFoldUser_RolesUpdated(t *testing.T) {
	initial := UserState{Email: "carol@example.com", Roles: []Role{RoleUser}}
	state, err := foldUser(initial, makeEvent(t, eventRolesUpdated, 2, RolesUpdatedPayload{
		Roles: []Role{RoleAdmin, RoleOwner}, Domain: "domain1",
	}))
	if err != nil {
		t.Fatalf("foldUser: %v", err)
	}
	if len(state.Roles) != 2 || state.Roles[0] != RoleAdmin || state.Roles[1] != RoleOwner {
		t.Errorf("Roles = %v", state.Roles)
	}
}

func TestFoldUser_EmailChanged(t *testing.T) {
	initial := UserState{Email: "old@example.com", Roles: []Role{RoleUser}}
	state, err := foldUser(initial, makeEvent(t, eventEmailChanged, 2, EmailChangedPayload{Email: "new@example.com"}))
	if err != nil {
		t.Fatalf("foldUser: %v", err)
	}
	if state.Email != "new@example.com" {
		t.Errorf("Email = %q", state.Email)
	}
}

func TestFoldUser_DisplayNameChanged(t *testing.T) {
	initial := UserState{Email: "d@example.com", DisplayName: "Old", Roles: []Role{RoleUser}}
	state, err := foldUser(initial, makeEvent(t, eventDisplayNameChanged, 2, DisplayNameChangedPayload{DisplayName: "New Name"}))
	if err != nil {
		t.Fatalf("foldUser: %v", err)
	}
	if state.DisplayName != "New Name" {
		t.Errorf("DisplayName = %q", state.DisplayName)
	}
}

func TestFoldUser_UserDeleted(t *testing.T) {
	initial := UserState{Email: "e@example.com", Roles: []Role{RoleUser}}
	state, err := foldUser(initial, makeEvent(t, eventUserDeleted, 2, UserDeletedPayload{Reason: "GDPR"}))
	if err != nil {
		t.Fatalf("foldUser: %v", err)
	}
	if !state.Deleted {
		t.Errorf("Deleted = false, want true")
	}
	if state.DeleteReason != "GDPR" {
		t.Errorf("DeleteReason = %q", state.DeleteReason)
	}
}

func TestFoldUser_CredentialAdded(t *testing.T) {
	initial := UserState{Email: "f@example.com", Roles: []Role{RoleUser}}
	state, err := foldUser(initial, makeEvent(t, eventCredentialAdded, 2, CredentialAddedPayload{
		ID:              []byte{1, 2, 3},
		PublicKey:       []byte{4, 5, 6},
		AttestationType: "none",
	}))
	if err != nil {
		t.Fatalf("foldUser: %v", err)
	}
	if len(state.Credentials) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(state.Credentials))
	}
	if state.Credentials[0].AttestationType != "none" {
		t.Errorf("AttestationType = %q", state.Credentials[0].AttestationType)
	}
}

func TestFoldUser_CredentialRemoved(t *testing.T) {
	initial := UserState{
		Email: "g@example.com",
		Roles: []Role{RoleUser},
		Credentials: []WebAuthnCredential{
			{ID: []byte{1, 2, 3}},
			{ID: []byte{4, 5, 6}},
		},
	}
	state, err := foldUser(initial, makeEvent(t, eventCredentialRemoved, 2, CredentialRemovedPayload{ID: []byte{1, 2, 3}}))
	if err != nil {
		t.Fatalf("foldUser: %v", err)
	}
	if len(state.Credentials) != 1 {
		t.Fatalf("expected 1 credential after removal, got %d", len(state.Credentials))
	}
	if state.Credentials[0].ID[0] != 4 {
		t.Errorf("wrong credential remained")
	}
}

func TestFoldUser_MultipleEvents(t *testing.T) {
	events := []event.Event{
		makeEvent(t, eventUserRegistered, 1, UserRegisteredPayload{
			Email: "multi@example.com", DisplayName: "Multi",
			Roles: []Role{RoleViewer},
		}),
		makeEvent(t, eventRolesUpdated, 2, RolesUpdatedPayload{Roles: []Role{RoleAdmin}, Domain: "dom"}),
		makeEvent(t, eventEmailChanged, 3, EmailChangedPayload{Email: "updated@example.com"}),
		makeEvent(t, eventDisplayNameChanged, 4, DisplayNameChangedPayload{DisplayName: "Updated"}),
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
		t.Errorf("Email = %q", state.Email)
	}
	if state.DisplayName != "Updated" {
		t.Errorf("DisplayName = %q", state.DisplayName)
	}
	if len(state.Roles) != 1 || state.Roles[0] != RoleAdmin {
		t.Errorf("Roles = %v", state.Roles)
	}
}

func TestFoldUser_UnknownEvent(t *testing.T) {
	initial := UserState{Email: "u@example.com", Roles: []Role{RoleUser}}
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
		t.Errorf("state changed on unknown event")
	}
}
