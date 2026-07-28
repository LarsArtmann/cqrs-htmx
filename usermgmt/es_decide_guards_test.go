package usermgmt

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// existingState is a valid, non-deleted user state used as the base for guard tests.
func existingState() UserState {
	return UserState{Email: "user@example.com", DisplayName: "User"}
}

// deletedState is an existing-but-deleted user state.
func deletedState() UserState {
	s := existingState()
	s.Deleted = true
	return s
}

func TestDecideChangeEmail_NotFound(t *testing.T) {
	decide := decideChangeEmail(id.NewStreamID(), "new@example.com")
	if _, err := decide(UserState{}, 0); err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestDecideChangeEmail_Deleted(t *testing.T) {
	decide := decideChangeEmail(id.NewStreamID(), "new@example.com")
	if _, err := decide(deletedState(), 1); err == nil {
		t.Fatal("expected deleted error")
	}
}

func TestDecideChangeDisplayName_Success(t *testing.T) {
	decide := decideChangeDisplayName(id.NewStreamID(), "New Name")
	events, err := decide(existingState(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 || events[0].Type() != eventDisplayNameChanged {
		t.Fatalf("expected 1 DisplayNameChanged event, got %d (%v)", len(events), events)
	}
}

func TestDecideChangeDisplayName_NotFound(t *testing.T) {
	decide := decideChangeDisplayName(id.NewStreamID(), "New Name")
	if _, err := decide(UserState{}, 0); err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestDecideChangeDisplayName_Deleted(t *testing.T) {
	decide := decideChangeDisplayName(id.NewStreamID(), "New Name")
	if _, err := decide(deletedState(), 1); err == nil {
		t.Fatal("expected deleted error")
	}
}

func TestDecideDeleteUser_NotFound(t *testing.T) {
	decide := decideDeleteUser(id.NewStreamID(), "gdpr")
	if _, err := decide(UserState{}, 0); err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestDecideAddCredential_NotFound(t *testing.T) {
	decide := decideAddCredential(
		id.NewStreamID(),
		WebAuthnCredential{CredentialCore: CredentialCore{ID: []byte{1}}},
	)
	if _, err := decide(UserState{}, 0); err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestDecideAddCredential_Deleted(t *testing.T) {
	decide := decideAddCredential(
		id.NewStreamID(),
		WebAuthnCredential{CredentialCore: CredentialCore{ID: []byte{1}}},
	)
	if _, err := decide(deletedState(), 1); err == nil {
		t.Fatal("expected deleted error")
	}
}

func TestDecideRemoveCredential_UserNotFound(t *testing.T) {
	decide := decideRemoveCredential(id.NewStreamID(), []byte{1})
	if _, err := decide(UserState{}, 0); err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestDecideRemoveCredential_Deleted(t *testing.T) {
	decide := decideRemoveCredential(id.NewStreamID(), []byte{1})
	if _, err := decide(deletedState(), 1); err == nil {
		t.Fatal("expected deleted error")
	}
}

func TestDecideVerifyEmail_Success(t *testing.T) {
	decide := decideVerifyEmail(id.NewStreamID())
	events, err := decide(existingState(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 || events[0].Type() != eventEmailVerified {
		t.Fatalf("expected 1 EmailVerified event, got %d (%v)", len(events), events)
	}
}

func TestDecideVerifyEmail_NotFound(t *testing.T) {
	decide := decideVerifyEmail(id.NewStreamID())
	if _, err := decide(UserState{}, 0); err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestDecideVerifyEmail_Deleted(t *testing.T) {
	decide := decideVerifyEmail(id.NewStreamID())
	if _, err := decide(deletedState(), 1); err == nil {
		t.Fatal("expected deleted error")
	}
}

func TestDecideVerifyEmail_AlreadyVerified(t *testing.T) {
	decide := decideVerifyEmail(id.NewStreamID())
	state := existingState()
	state.EmailVerified = true
	events, err := decide(state, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected idempotent no-op, got %d events", len(events))
	}
}

func TestDecideEnableTOTP_Success(t *testing.T) {
	decide := decideEnableTOTP(id.NewStreamID(), []byte("secret"))
	events, err := decide(existingState(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 || events[0].Type() != eventTOTPEnabled {
		t.Fatalf("expected 1 TOTPEnabled event, got %d (%v)", len(events), events)
	}
}

func TestDecideEnableTOTP_NotFound(t *testing.T) {
	decide := decideEnableTOTP(id.NewStreamID(), []byte("secret"))
	if _, err := decide(UserState{}, 0); err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestDecideEnableTOTP_Deleted(t *testing.T) {
	decide := decideEnableTOTP(id.NewStreamID(), []byte("secret"))
	if _, err := decide(deletedState(), 1); err == nil {
		t.Fatal("expected deleted error")
	}
}

func TestDecideEnableTOTP_AlreadyEnabled(t *testing.T) {
	decide := decideEnableTOTP(id.NewStreamID(), []byte("secret"))
	state := existingState()
	state.TOTPEnabled = true
	events, err := decide(state, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected idempotent no-op, got %d events", len(events))
	}
}

func TestDecideDisableTOTP_Success(t *testing.T) {
	decide := decideDisableTOTP(id.NewStreamID())
	state := existingState()
	state.TOTPEnabled = true
	events, err := decide(state, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 || events[0].Type() != eventTOTPDisabled {
		t.Fatalf("expected 1 TOTPDisabled event, got %d (%v)", len(events), events)
	}
}

func TestDecideDisableTOTP_NotFound(t *testing.T) {
	decide := decideDisableTOTP(id.NewStreamID())
	if _, err := decide(UserState{}, 0); err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestDecideDisableTOTP_Deleted(t *testing.T) {
	decide := decideDisableTOTP(id.NewStreamID())
	if _, err := decide(deletedState(), 1); err == nil {
		t.Fatal("expected deleted error")
	}
}

func TestDecideDisableTOTP_AlreadyDisabled(t *testing.T) {
	decide := decideDisableTOTP(id.NewStreamID())
	events, err := decide(existingState(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected idempotent no-op, got %d events", len(events))
	}
}

// TestDecideGuardErrorsAreRejections confirms the not-found/deleted guards all
// classify as Rejections (a consistent contract across every decide function).
func TestDecideGuardErrorsAreRejections(t *testing.T) {
	aggID := id.NewStreamID()
	cases := []struct {
		name   string
		decide func(UserState, event.Version) ([]event.Event, error)
	}{
		{"change_email", decideChangeEmail(aggID, "new@example.com")},
		{"change_display_name", decideChangeDisplayName(aggID, "N")},
		{"delete_user", decideDeleteUser(aggID, "r")},
		{
			"add_credential",
			decideAddCredential(aggID, WebAuthnCredential{CredentialCore: CredentialCore{ID: []byte{1}}}),
		},
		{"remove_credential", decideRemoveCredential(aggID, []byte{1})},
		{"verify_email", decideVerifyEmail(aggID)},
		{"enable_totp", decideEnableTOTP(aggID, []byte("s"))},
		{"disable_totp", decideDisableTOTP(aggID)},
	}
	for _, tc := range cases {
		t.Run(tc.name+"/not_found", func(t *testing.T) {
			_, err := tc.decide(UserState{}, 0)
			if err == nil {
				t.Fatal("expected error")
			}
			if errorfamily.Classify(err) != event.Rejection {
				t.Errorf("not-found: expected Rejection, got %v", err)
			}
		})
		t.Run(tc.name+"/deleted", func(t *testing.T) {
			_, err := tc.decide(deletedState(), 1)
			if err == nil {
				t.Fatal("expected error")
			}
			if errorfamily.Classify(err) != event.Rejection {
				t.Errorf("deleted: expected Rejection, got %v", err)
			}
		})
	}
}
