package identitymodel

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// The S32 credential-use proof tests (Lane-D plan R02): the fold contract,
// the counter-regression signal, and the apply semantics — each pinned so the
// sign-counter persistence cannot silently regress.

func credentialUsedEvent(t *testing.T, use CredentialUse) event.Event {
	t.Helper()

	payload, err := MarshalPayload(CredentialUsedPayload{
		SchemaVersion: CurrentSchemaVersion,
		CredentialID:  use.CredentialID,
		SignCount:     use.SignCount,
		UserPresent:   use.UserPresent,
		UserVerified:  use.UserVerified,
		BackupState:   use.BackupState,
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	evt, evtErr := event.NewEvent(EventCredentialUsed, id.NewStreamID(), AggregateTypeUser, 3, payload)
	if evtErr != nil {
		t.Fatalf("event.NewEvent: %v", evtErr)
	}

	return evt
}

func twoCredentialState() UserState {
	return UserState{
		Credentials: []WebAuthnCredential{
			{CredentialCore: CredentialCore{ID: []byte{0x01}, SignCount: 5, Transports: []string{"usb"}}},
			{CredentialCore: CredentialCore{ID: []byte{0x02}, SignCount: 9}},
		},
	}
}

func TestFoldUser_CredentialUsed_AppliesToNamedCredential(t *testing.T) {
	state := twoCredentialState()

	next, err := FoldUser(state, credentialUsedEvent(t, CredentialUse{
		CredentialID: []byte{0x02},
		SignCount:    10,
		UserPresent:  true,
		UserVerified: false,
		BackupState:  true,
	}))
	if err != nil {
		t.Fatalf("FoldUser: %v", err)
	}

	applied := next.Credentials[1]

	if applied.SignCount != 10 || !applied.UserPresent || applied.UserVerified || !applied.BackupState {
		t.Fatalf("assertion state not applied: %+v", applied)
	}

	if next.Credentials[0].SignCount != 5 {
		t.Fatalf("the unnamed credential must be untouched, got sign count %d", next.Credentials[0].SignCount)
	}
}

func TestFoldUser_CredentialUsed_UnknownCredentialIsCorruption(t *testing.T) {
	state := twoCredentialState()

	_, err := FoldUser(state, credentialUsedEvent(t, CredentialUse{
		CredentialID: []byte{0xAB},
		SignCount:    10,
	}))
	if err == nil {
		t.Fatal("a CredentialUsed event for an unknown credential must fail the fold")
	}

	if !strings.Contains(err.Error(), "unknown_credential") {
		t.Fatalf("error should carry the corruption code, got: %v", err)
	}
}

// TestFoldUser_CredentialUsed_Purity pins the fold-purity contract: folding
// never writes through to the caller's state — neither on a successful apply
// (the credential slice is copied) nor on a rejected one (state returned as-is).
func TestFoldUser_CredentialUsed_Purity(t *testing.T) {
	state := twoCredentialState()
	original := state.Credentials

	next, err := FoldUser(state, credentialUsedEvent(t, CredentialUse{
		CredentialID: []byte{0x01},
		SignCount:    42,
	}))
	if err != nil {
		t.Fatalf("FoldUser: %v", err)
	}

	if original[0].SignCount != 5 {
		t.Fatalf("successful fold wrote through to the input state: sign count %d", original[0].SignCount)
	}

	if next.Credentials[0].SignCount != 42 {
		t.Fatalf("fold result missing the applied count: %d", next.Credentials[0].SignCount)
	}

	_, err = FoldUser(state, credentialUsedEvent(t, CredentialUse{CredentialID: []byte{0xAB}}))
	if err == nil {
		t.Fatal("expected the unknown-credential rejection")
	}

	if original[0].SignCount != 5 || original[1].SignCount != 9 {
		t.Fatalf("rejected fold mutated the input state: %+v", original)
	}
}

func TestCounterRegressed_Table(t *testing.T) {
	cases := []struct {
		current uint32
		next    uint32
		want    bool
	}{
		{current: 0, next: 0, want: false},
		{current: 0, next: 5, want: false},
		{current: 5, next: 5, want: true},
		{current: 5, next: 1, want: true},
		{current: 5, next: 0, want: true},
		{current: 5, next: 6, want: false},
	}

	for _, tc := range cases {
		cred := WebAuthnCredential{CredentialCore: CredentialCore{SignCount: tc.current}}

		if got := cred.CounterRegressed(tc.next); got != tc.want {
			t.Errorf("CounterRegressed(%d -> %d) = %v, want %v", tc.current, tc.next, got, tc.want)
		}
	}
}

func TestApplyUse_Table(t *testing.T) {
	cred := WebAuthnCredential{CredentialCore: CredentialCore{
		ID:             []byte{0x01},
		Transports:     []string{"usb", "nfc"},
		SignCount:      5,
		UserPresent:    false,
		UserVerified:   false,
		BackupEligible: true,
		BackupState:    false,
	}}

	next := cred.ApplyUse(CredentialUse{
		CredentialID: []byte{0x01},
		SignCount:    42,
		UserPresent:  true,
		UserVerified: true,
		BackupState:  true,
	})

	if next.SignCount != 42 || !next.UserPresent || !next.UserVerified || !next.BackupState {
		t.Fatalf("assertion state not applied: %+v", next.CredentialCore)
	}

	if !next.BackupEligible {
		t.Fatal("BackupEligible is registration-time state and must survive ApplyUse")
	}

	if cred.SignCount != 5 || cred.UserPresent || cred.BackupState {
		t.Fatalf("ApplyUse must not mutate the receiver: %+v", cred.CredentialCore)
	}

	next.Transports[0] = "mutated"

	if cred.Transports[0] != "usb" {
		t.Fatal("ApplyUse result must not alias the original's slices")
	}
}
