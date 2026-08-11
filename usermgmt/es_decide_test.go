package usermgmt

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

func TestDecideRegisterUser_Success(t *testing.T) {
	aggID := id.NewStreamID()
	decide := decideRegisterUser(aggID, "alice@example.com", "Alice", []Role{RoleUser})
	events, err := decide(UserState{}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].Type() != eventUserRegistered {
		t.Errorf("type = %s", events[0].Type())
	}
}

func TestDecideRegisterUser_AlreadyExists(t *testing.T) {
	decide := decideRegisterUser(id.NewStreamID(), "a@b.com", "A", []Role{RoleUser})
	events, err := decide(UserState{Email: "existing@example.com"}, 1)
	if err == nil {
		t.Fatalf("expected error, got %d events", len(events))
	}
	if errorfamily.Classify(err) != event.Conflict {
		t.Errorf("expected Conflict, got %v", err)
	}
}

func TestDecideRegisterUser_EmptyEmail(t *testing.T) {
	decide := decideRegisterUser(id.NewStreamID(), "", "A", []Role{RoleUser})
	events, err := decide(UserState{}, 0)
	if err == nil {
		t.Fatalf("expected error, got %d events", len(events))
	}
	if errorfamily.Classify(err) != event.Rejection {
		t.Errorf("expected Rejection, got %v", err)
	}
}

func TestDecideChangeEmail(t *testing.T) {
	cases := []struct {
		name        string
		newEmail    string
		stateEmail  string
		wantEvents  int
		description string
	}{
		{
			name:        "success",
			newEmail:    "new@example.com",
			stateEmail:  "old@example.com",
			wantEvents:  1,
			description: "different emails produce one event",
		},
		{
			name:        "same email is a no-op",
			newEmail:    "same@example.com",
			stateEmail:  "same@example.com",
			wantEvents:  0,
			description: "identical emails produce zero events",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			decide := decideChangeEmail(id.NewStreamID(), tc.newEmail)
			state := UserState{Email: tc.stateEmail}
			events, err := decide(state, 1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(events) != tc.wantEvents {
				t.Fatalf("%s: got %d events, want %d", tc.description, len(events), tc.wantEvents)
			}
		})
	}
}

func TestDecideDeleteUser_Success(t *testing.T) {
	decide := decideDeleteUser(id.NewStreamID(), "GDPR")
	state := UserState{Email: "u@example.com"}
	events, err := decide(state, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].Type() != eventUserDeleted {
		t.Errorf("expected event type %s, got %s", eventUserDeleted, events[0].Type())
	}
}

func TestDecideDeleteUser_AlreadyDeleted(t *testing.T) {
	decide := decideDeleteUser(id.NewStreamID(), "r")
	state := UserState{Email: "u@example.com", Deleted: true}
	events, err := decide(state, 1)
	if err == nil {
		t.Fatalf("expected error, got %d events", len(events))
	}
}

func TestDecideAddCredential_Success(t *testing.T) {
	decide := decideAddCredential(id.NewStreamID(), WebAuthnCredential{
		CredentialCore: CredentialCore{ID: []byte{1, 2, 3}, PublicKey: []byte{4, 5, 6}, AttestationType: "none"},
	})
	state := UserState{Email: "u@example.com"}
	events, err := decide(state, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].Type() != eventCredentialAdded {
		t.Errorf("type = %s", events[0].Type())
	}
}

func TestDecideAddCredential_Duplicate(t *testing.T) {
	decide := decideAddCredential(id.NewStreamID(), WebAuthnCredential{
		CredentialCore: CredentialCore{ID: []byte{1, 2, 3}},
	})
	state := UserState{
		Email:       "u@example.com",
		Credentials: []WebAuthnCredential{{CredentialCore: CredentialCore{ID: []byte{1, 2, 3}}}},
	}
	events, err := decide(state, 1)
	if err == nil {
		t.Fatalf("expected error, got %d events", len(events))
	}
	if errorfamily.Classify(err) != event.Conflict {
		t.Errorf("expected Conflict, got %v", err)
	}
}

func TestDecideRemoveCredential_Success(t *testing.T) {
	decide := decideRemoveCredential(id.NewStreamID(), []byte{1, 2, 3})
	state := UserState{
		Email:       "u@example.com",
		Credentials: []WebAuthnCredential{{CredentialCore: CredentialCore{ID: []byte{1, 2, 3}}}},
	}
	events, err := decide(state, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].Type() != eventCredentialRemoved {
		t.Errorf("type = %s", events[0].Type())
	}
}

func TestDecideRemoveCredential_NotFound(t *testing.T) {
	decide := decideRemoveCredential(id.NewStreamID(), []byte{9, 9, 9})
	state := UserState{
		Email:       "u@example.com",
		Credentials: []WebAuthnCredential{{CredentialCore: CredentialCore{ID: []byte{1, 2, 3}}}},
	}
	events, err := decide(state, 1)
	if err == nil {
		t.Fatalf("expected error, got %d events", len(events))
	}
}
