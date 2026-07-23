package usermgmt

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func newTestSetup(t *testing.T) *EventSourcedSetup {
	t.Helper()
	setup, err := DefaultEventSourcedSetup()
	if err != nil {
		t.Fatalf("DefaultEventSourcedSetup: %v", err)
	}
	return setup
}

func newTestDispatcher(t *testing.T, setup *EventSourcedSetup) *command.Dispatcher {
	t.Helper()
	disp := command.NewDispatcher()
	if err := RegisterCommands(disp, setup.Repository); err != nil {
		t.Fatalf("RegisterCommands: %v", err)
	}
	return disp
}

func sampleWebAuthnCredential() WebAuthnCredential {
	return WebAuthnCredential{
		CredentialCore: CredentialCore{ID: []byte{1, 2, 3}, PublicKey: []byte{4, 5, 6}, AttestationType: "none"},
	}
}

func dispatchAddCredential(t *testing.T, disp *command.Dispatcher, ctx context.Context, aggID id.AggregateID) {
	t.Helper()
	if err := disp.Dispatch(ctx, NewAddCredentialCmd(aggID, sampleWebAuthnCredential())); err != nil {
		t.Fatalf("add credential: %v", err)
	}
}

func dispatchRegisterUser(
	t *testing.T,
	disp *command.Dispatcher,
	ctx context.Context,
	aggID id.AggregateID,
	email, name string,
) {
	t.Helper()
	if err := disp.Dispatch(ctx, NewRegisterUserCmd(aggID, email, name, []Role{RoleUser})); err != nil {
		t.Fatalf("register %s: %v", email, err)
	}
}

func TestWiring_RegisterUser(t *testing.T) {
	setup := newTestSetup(t)
	disp := newTestDispatcher(t, setup)
	ctx := context.Background()

	aggID := id.NewAggregateID()
	dispatchRegisterUser(t, disp, ctx, aggID, "alice@example.com", "Alice")

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
	disp := newTestDispatcher(t, setup)
	ctx := context.Background()

	aggID := id.NewAggregateID()
	dispatchRegisterUser(t, disp, ctx, aggID, "dup@example.com", "Dup")

	err := disp.Dispatch(ctx, NewRegisterUserCmd(
		aggID, "dup@example.com", "Dup", []Role{RoleUser},
	))
	if err == nil {
		t.Fatal("expected error on duplicate register")
	}
}

func TestWiring_DeleteUser(t *testing.T) {
	setup := newTestSetup(t)
	disp := newTestDispatcher(t, setup)
	ctx := context.Background()

	aggID := id.NewAggregateID()
	dispatchRegisterUser(t, disp, ctx, aggID, "delete@example.com", "Del")

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
	disp := newTestDispatcher(t, setup)
	ctx := context.Background()

	aggID := id.NewAggregateID()
	dispatchRegisterUser(t, disp, ctx, aggID, "cred@example.com", "Cred")

	dispatchAddCredential(t, disp, ctx, aggID)

	user, _ := setup.ReadModel.FindByID(aggID)
	if len(user.Credentials) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(user.Credentials))
	}
}

func TestWiring_RemoveCredential(t *testing.T) {
	setup := newTestSetup(t)
	disp := newTestDispatcher(t, setup)
	ctx := context.Background()

	aggID := id.NewAggregateID()
	dispatchRegisterUser(t, disp, ctx, aggID, "remcred@example.com", "RemCred")

	dispatchAddCredential(t, disp, ctx, aggID)

	if err := disp.Dispatch(ctx, NewRemoveCredentialCmd(aggID, []byte{1, 2, 3})); err != nil {
		t.Fatalf("remove credential: %v", err)
	}

	user, _ := setup.ReadModel.FindByID(aggID)
	if len(user.Credentials) != 0 {
		t.Fatalf("expected 0 credentials, got %d", len(user.Credentials))
	}
}
