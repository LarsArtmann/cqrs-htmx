package usermgmt

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

func TestDecideRegisterUser_Success(t *testing.T) {
	aggID := id.NewAggregateID()
	decide := decideRegisterUser(aggID, "alice@example.com", "Alice", "hash", []Role{RoleUser})
	events, err := decide(UserState{}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].Type() != eventUserRegistered {
		t.Errorf("type = %s, want UserRegistered", events[0].Type())
	}
	if events[0].Version().Int() != 1 {
		t.Errorf("version = %d, want 1", events[0].Version().Int())
	}
	if events[0].AggregateID() != aggID {
		t.Errorf("aggregate ID mismatch")
	}
}

func TestDecideRegisterUser_AlreadyExists(t *testing.T) {
	decide := decideRegisterUser(id.NewAggregateID(), "a@b.com", "A", "h", []Role{RoleUser})
	existing := UserState{Email: "existing@example.com"}
	events, err := decide(existing, 1)
	if err == nil {
		t.Fatalf("expected error, got %d events", len(events))
	}
	if event.Classify(err) != event.Conflict {
		t.Errorf("expected Conflict error, got %v", err)
	}
}

func TestDecideRegisterUser_EmptyEmail(t *testing.T) {
	decide := decideRegisterUser(id.NewAggregateID(), "", "A", "h", []Role{RoleUser})
	events, err := decide(UserState{}, 0)
	if err == nil {
		t.Fatalf("expected error, got %d events", len(events))
	}
	if event.Classify(err) != event.Rejection {
		t.Errorf("expected Rejection error, got %v", err)
	}
}

func TestDecideChangePassword_Success(t *testing.T) {
	decide := decideChangePassword(id.NewAggregateID(), "new-hash")
	state := UserState{Email: "user@example.com", PasswordHash: "old"}
	events, err := decide(state, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].Type() != eventPasswordChanged {
		t.Errorf("type = %s, want PasswordChanged", events[0].Type())
	}
}

func TestDecideChangePassword_NotFound(t *testing.T) {
	decide := decideChangePassword(id.NewAggregateID(), "h")
	events, err := decide(UserState{}, 0)
	if err == nil {
		t.Fatalf("expected error, got %d events", len(events))
	}
	if event.Classify(err) != event.Rejection {
		t.Errorf("expected Rejection error, got %v", err)
	}
}

func TestDecideChangePassword_Deleted(t *testing.T) {
	decide := decideChangePassword(id.NewAggregateID(), "h")
	state := UserState{Email: "d@example.com", Deleted: true}
	events, err := decide(state, 1)
	if err == nil {
		t.Fatalf("expected error, got %d events", len(events))
	}
	if event.Classify(err) != event.Rejection {
		t.Errorf("expected Rejection error, got %v", err)
	}
}

func TestDecideUpdateRoles_Success(t *testing.T) {
	decide := decideUpdateRoles(id.NewAggregateID(), []Role{RoleAdmin}, "domain1")
	state := UserState{Email: "u@example.com"}
	events, err := decide(state, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].Type() != eventRolesUpdated {
		t.Errorf("type = %s, want RolesUpdated", events[0].Type())
	}
}

func TestDecideUpdateRoles_Deleted(t *testing.T) {
	decide := decideUpdateRoles(id.NewAggregateID(), []Role{RoleAdmin}, "d")
	state := UserState{Email: "u@example.com", Deleted: true}
	events, err := decide(state, 1)
	if err == nil {
		t.Fatalf("expected error, got %d events", len(events))
	}
}

func TestDecideUpdateRoles_NotFound(t *testing.T) {
	decide := decideUpdateRoles(id.NewAggregateID(), []Role{RoleAdmin}, "d")
	events, err := decide(UserState{}, 0)
	if err == nil {
		t.Fatalf("expected error, got %d events", len(events))
	}
}

func TestDecideChangeEmail_Success(t *testing.T) {
	decide := decideChangeEmail(id.NewAggregateID(), "new@example.com")
	state := UserState{Email: "old@example.com"}
	events, err := decide(state, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].Type() != eventEmailChanged {
		t.Errorf("type = %s, want EmailChanged", events[0].Type())
	}
}

func TestDecideChangeEmail_SameEmail(t *testing.T) {
	decide := decideChangeEmail(id.NewAggregateID(), "same@example.com")
	state := UserState{Email: "same@example.com"}
	events, err := decide(state, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("got %d events, want 0 (no change)", len(events))
	}
}

func TestDecideChangeDisplayName_Success(t *testing.T) {
	decide := decideChangeDisplayName(id.NewAggregateID(), "New Name")
	state := UserState{Email: "u@example.com", DisplayName: "Old"}
	events, err := decide(state, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
}

func TestDecideChangeDisplayName_SameName(t *testing.T) {
	decide := decideChangeDisplayName(id.NewAggregateID(), "Same")
	state := UserState{Email: "u@example.com", DisplayName: "Same"}
	events, err := decide(state, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("got %d events, want 0", len(events))
	}
}

func TestDecideDeleteUser_Success(t *testing.T) {
	decide := decideDeleteUser(id.NewAggregateID(), "GDPR")
	state := UserState{Email: "u@example.com"}
	events, err := decide(state, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].Type() != eventUserDeleted {
		t.Errorf("type = %s, want UserDeleted", events[0].Type())
	}
	if events[0].Metadata().Custom[event.MetadataKeyTombstone] == "" {
		t.Error("expected tombstone metadata on UserDeleted event")
	}
}

func TestDecideDeleteUser_AlreadyDeleted(t *testing.T) {
	decide := decideDeleteUser(id.NewAggregateID(), "r")
	state := UserState{Email: "u@example.com", Deleted: true}
	events, err := decide(state, 1)
	if err == nil {
		t.Fatalf("expected error, got %d events", len(events))
	}
}

func TestDecideDeleteUser_NotFound(t *testing.T) {
	decide := decideDeleteUser(id.NewAggregateID(), "r")
	events, err := decide(UserState{}, 0)
	if err == nil {
		t.Fatalf("expected error, got %d events", len(events))
	}
}
