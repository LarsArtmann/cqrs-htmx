package usermgmt

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

func TestAggIDFromUser_InvalidID(t *testing.T) {
	svc := newTestService(t)
	err := svc.ChangeEmail(context.Background(), NewUserID("not-a-valid-ulid"), "new@test.com")
	if err == nil {
		t.Fatal("expected error for invalid UserID")
	}
}

func TestAggIDFromUser_EmptyID(t *testing.T) {
	svc := newTestService(t)
	assertChangeEmailError(t, svc, NewUserID(""), "new@test.com")
}

func TestClassifyDispatchError_Transient(t *testing.T) {
	svc := newTestService(t)
	transient := event.NewTransient("test", "some transient error")
	result := svc.classifyDispatchError(transient, NewUserID("u1"))
	if result == nil {
		t.Fatal("expected non-nil error")
	}
	if !errors.Is(result, transient) && !isTransient(result) {
		t.Errorf("expected transient classification, got: %v", result)
	}
}

func isTransient(err error) bool {
	return event.IsRetryable(err)
}

func TestDecideChangeDisplayName_NoOp(t *testing.T) {
	aggID := id.NewAggregateID()
	state := UserState{
		Email:       "u@test.com",
		DisplayName: "Alice",
	}
	decider := decideChangeDisplayName(aggID, "Alice")
	events, err := decider(state, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events for no-op change, got %d", len(events))
	}
}

func TestDecideChangeEmail_NoOp(t *testing.T) {
	aggID := id.NewAggregateID()
	state := UserState{
		Email:       "same@test.com",
		DisplayName: "Alice",
	}
	decider := decideChangeEmail(aggID, "same@test.com")
	events, err := decider(state, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events for no-op email change, got %d", len(events))
	}
}

func TestRegisterCommands_Success(t *testing.T) {
	setup, err := DefaultEventSourcedSetup()
	if err != nil {
		t.Fatalf("DefaultEventSourcedSetup: %v", err)
	}
	dispatcher := newTestDispatcher(t, setup)
	if dispatcher == nil {
		t.Fatal("expected non-nil dispatcher")
	}
}

func TestMarshalPayload_Error(t *testing.T) {
	_, err := marshalPayload(make(chan int))
	if err == nil {
		t.Fatal("expected error for unmarshallable type")
	}
}

func TestFoldUser_CredentialAdded_SignCountPreserved(t *testing.T) {
	initial := UserState{
		Email: "u@test.com",
	}
	payload, err := marshalPayload(CredentialAddedPayload{
		ID:        []byte{1, 2, 3},
		PublicKey: []byte{4, 5},
		SignCount: 42,
	})
	if err != nil {
		t.Fatalf("marshalPayload: %v", err)
	}
	evt, err := event.NewEvent(
		eventCredentialAdded, testAggID, aggregateTypeUser, 2,
		payload,
	)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}
	state, err := foldUser(initial, evt)
	if err != nil {
		t.Fatalf("foldUser: %v", err)
	}
	if len(state.Credentials) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(state.Credentials))
	}
	if state.Credentials[0].SignCount != 42 {
		t.Errorf("expected SignCount=42, got %d", state.Credentials[0].SignCount)
	}
}
