package usermgmt

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

func newTestSetup(t *testing.T) *EventSourcedSetup {
	t.Helper()
	setup, err := DefaultEventSourcedSetup()
	if err != nil {
		t.Fatalf("DefaultEventSourcedSetup: %v", err)
	}
	return setup
}

func newTestDispatcher(setup *EventSourcedSetup) *command.Dispatcher {
	disp := command.NewDispatcher()
	RegisterCommands(disp, setup.Repository)
	return disp
}

func TestWiring_RegisterUser(t *testing.T) {
	setup := newTestSetup(t)
	disp := newTestDispatcher(setup)
	ctx := context.Background()

	aggID := id.NewAggregateID()
	err := disp.Dispatch(ctx, NewRegisterUserCmd(
		aggID, "alice@example.com", "Alice", []Role{RoleUser},
	))
	if err != nil {
		t.Fatalf("dispatch RegisterUser: %v", err)
	}

	user, ok := setup.ReadModel.FindByID(aggID)
	if !ok {
		t.Fatal("user not found in read model after register")
	}
	if user.Email != "alice@example.com" {
		t.Errorf("email = %q", user.Email)
	}
}

func TestWiring_RegisterDuplicate(t *testing.T) {
	setup := newTestSetup(t)
	disp := newTestDispatcher(setup)
	ctx := context.Background()

	aggID := id.NewAggregateID()
	if err := disp.Dispatch(ctx, NewRegisterUserCmd(
		aggID, "dup@example.com", "Dup", []Role{RoleUser},
	)); err != nil {
		t.Fatalf("first register: %v", err)
	}

	err := disp.Dispatch(ctx, NewRegisterUserCmd(
		aggID, "dup@example.com", "Dup", []Role{RoleUser},
	))
	if err == nil {
		t.Fatal("expected error on duplicate register")
	}
}

func TestWiring_UpdateRoles(t *testing.T) {
	setup := newTestSetup(t)
	disp := newTestDispatcher(setup)
	ctx := context.Background()

	aggID := id.NewAggregateID()
	if err := disp.Dispatch(ctx, NewRegisterUserCmd(
		aggID, "carol@example.com", "Carol", []Role{RoleUser},
	)); err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := disp.Dispatch(ctx, NewUpdateRolesCmd(aggID, []Role{RoleAdmin}, "domain1")); err != nil {
		t.Fatalf("update roles: %v", err)
	}

	user, _ := setup.ReadModel.FindByID(aggID)
	if len(user.Roles) != 1 || user.Roles[0] != RoleAdmin {
		t.Errorf("roles = %v", user.Roles)
	}
}

func TestWiring_DeleteUser(t *testing.T) {
	setup := newTestSetup(t)
	disp := newTestDispatcher(setup)
	ctx := context.Background()

	aggID := id.NewAggregateID()
	if err := disp.Dispatch(ctx, NewRegisterUserCmd(
		aggID, "delete@example.com", "Del", []Role{RoleUser},
	)); err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := disp.Dispatch(ctx, NewDeleteUserCmd(aggID, "test")); err != nil {
		t.Fatalf("delete: %v", err)
	}

	_, ok := setup.ReadModel.FindByID(aggID)
	if ok {
		t.Error("user should not exist after delete")
	}
}

func TestWiring_AddCredential(t *testing.T) {
	setup := newTestSetup(t)
	disp := newTestDispatcher(setup)
	ctx := context.Background()

	aggID := id.NewAggregateID()
	if err := disp.Dispatch(ctx, NewRegisterUserCmd(
		aggID, "cred@example.com", "Cred", []Role{RoleUser},
	)); err != nil {
		t.Fatalf("register: %v", err)
	}

	err := disp.Dispatch(ctx, NewAddCredentialCmd(aggID, WebAuthnCredential{
		ID: []byte{1, 2, 3}, PublicKey: []byte{4, 5, 6}, AttestationType: "none",
	}))
	if err != nil {
		t.Fatalf("add credential: %v", err)
	}

	user, _ := setup.ReadModel.FindByID(aggID)
	if len(user.Credentials) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(user.Credentials))
	}
}

func TestWiring_RemoveCredential(t *testing.T) {
	setup := newTestSetup(t)
	disp := newTestDispatcher(setup)
	ctx := context.Background()

	aggID := id.NewAggregateID()
	if err := disp.Dispatch(ctx, NewRegisterUserCmd(
		aggID, "remcred@example.com", "RemCred", []Role{RoleUser},
	)); err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := disp.Dispatch(ctx, NewAddCredentialCmd(aggID, WebAuthnCredential{
		ID: []byte{1, 2, 3}, PublicKey: []byte{4, 5, 6}, AttestationType: "none",
	})); err != nil {
		t.Fatalf("add credential: %v", err)
	}

	if err := disp.Dispatch(ctx, NewRemoveCredentialCmd(aggID, []byte{1, 2, 3})); err != nil {
		t.Fatalf("remove credential: %v", err)
	}

	user, _ := setup.ReadModel.FindByID(aggID)
	if len(user.Credentials) != 0 {
		t.Fatalf("expected 0 credentials, got %d", len(user.Credentials))
	}
}
