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
		aggID, "alice@example.com", "Alice", "hashed-pw", []Role{RoleUser},
	))
	if err != nil {
		t.Fatalf("dispatch RegisterUser: %v", err)
	}

	user, ok := setup.ReadModel.FindByID(aggID)
	if !ok {
		t.Fatal("user not found in read model after register")
	}
	if user.Email != "alice@example.com" {
		t.Errorf("email = %q, want alice@example.com", user.Email)
	}
	if user.PasswordHash != "hashed-pw" {
		t.Errorf("password hash = %q, want hashed-pw", user.PasswordHash)
	}
	if len(user.Roles) != 1 || user.Roles[0] != RoleUser {
		t.Errorf("roles = %v, want [user]", user.Roles)
	}
}

func TestWiring_RegisterAndChangePassword(t *testing.T) {
	setup := newTestSetup(t)
	disp := newTestDispatcher(setup)
	ctx := context.Background()

	aggID := id.NewAggregateID()
	if err := disp.Dispatch(ctx, NewRegisterUserCmd(
		aggID, "bob@example.com", "Bob", "old-hash", []Role{RoleUser},
	)); err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := disp.Dispatch(ctx, NewChangePasswordCmd(aggID, "new-hash")); err != nil {
		t.Fatalf("change password: %v", err)
	}

	user, _ := setup.ReadModel.FindByID(aggID)
	if user.PasswordHash != "new-hash" {
		t.Errorf("password hash = %q, want new-hash", user.PasswordHash)
	}
}

func TestWiring_RegisterDuplicate(t *testing.T) {
	setup := newTestSetup(t)
	disp := newTestDispatcher(setup)
	ctx := context.Background()

	aggID := id.NewAggregateID()
	if err := disp.Dispatch(ctx, NewRegisterUserCmd(
		aggID, "dup@example.com", "Dup", "hash", []Role{RoleUser},
	)); err != nil {
		t.Fatalf("first register: %v", err)
	}

	err := disp.Dispatch(ctx, NewRegisterUserCmd(
		aggID, "dup@example.com", "Dup", "hash", []Role{RoleUser},
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
		aggID, "carol@example.com", "Carol", "hash", []Role{RoleUser},
	)); err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := disp.Dispatch(ctx, NewUpdateRolesCmd(aggID, []Role{RoleAdmin}, "domain1")); err != nil {
		t.Fatalf("update roles: %v", err)
	}

	user, _ := setup.ReadModel.FindByID(aggID)
	if len(user.Roles) != 1 || user.Roles[0] != RoleAdmin {
		t.Errorf("roles = %v, want [admin]", user.Roles)
	}
}

func TestWiring_DeleteUser(t *testing.T) {
	setup := newTestSetup(t)
	disp := newTestDispatcher(setup)
	ctx := context.Background()

	aggID := id.NewAggregateID()
	if err := disp.Dispatch(ctx, NewRegisterUserCmd(
		aggID, "delete@example.com", "Del", "hash", []Role{RoleUser},
	)); err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := disp.Dispatch(ctx, NewDeleteUserCmd(aggID, "test")); err != nil {
		t.Fatalf("delete: %v", err)
	}

	_, ok := setup.ReadModel.FindByID(aggID)
	if ok {
		t.Error("user should not exist in read model after delete")
	}
}

func TestWiring_ChangeEmail(t *testing.T) {
	setup := newTestSetup(t)
	disp := newTestDispatcher(setup)
	ctx := context.Background()

	aggID := id.NewAggregateID()
	if err := disp.Dispatch(ctx, NewRegisterUserCmd(
		aggID, "old@example.com", "User", "hash", []Role{RoleUser},
	)); err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := disp.Dispatch(ctx, NewChangeEmailCmd(aggID, "new@example.com")); err != nil {
		t.Fatalf("change email: %v", err)
	}

	user, _ := setup.ReadModel.FindByID(aggID)
	if user.Email != "new@example.com" {
		t.Errorf("email = %q, want new@example.com", user.Email)
	}

	_, ok := setup.ReadModel.FindByEmail("old@example.com")
	if ok {
		t.Error("old email should not be found")
	}

	_, ok = setup.ReadModel.FindByEmail("new@example.com")
	if !ok {
		t.Error("new email should be found")
	}
}
