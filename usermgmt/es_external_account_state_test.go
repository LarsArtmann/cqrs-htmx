package usermgmt

import (
	"testing"
)

// --- foldUser ExternalAccountLinked/Unlinked invariants (Task 15) ---

func TestFoldUser_ExternalAccountLinked(t *testing.T) {
	initial := UserState{Email: "h@example.com"}
	state, err := foldUser(initial, makeEvent(t, eventExternalAccountLinked, 2, ExternalAccountLinkedPayload{
		Provider:    "google",
		Subject:     "sub-123",
		Email:       "h@gmail.com",
		DisplayName: "H Person",
	}))
	if err != nil {
		t.Fatalf("foldUser: %v", err)
	}
	if len(state.ExternalAccounts) != 1 {
		t.Fatalf("expected 1 external account, got %d", len(state.ExternalAccounts))
	}
	ea := state.ExternalAccounts[0]
	if ea.Provider != "google" || ea.Subject != "sub-123" {
		t.Errorf("Provider/Subject = %q/%q", ea.Provider, ea.Subject)
	}
	if ea.Email != "h@gmail.com" || ea.DisplayName != "H Person" {
		t.Errorf("Email/DisplayName = %q/%q", ea.Email, ea.DisplayName)
	}
}

func TestFoldUser_ExternalAccountUnlinked(t *testing.T) {
	initial := UserState{
		Email: "i@example.com",
		ExternalAccounts: []ExternalAccount{
			{Provider: "google", Subject: "sub-123"},
			{Provider: "github", Subject: "sub-456"},
		},
	}
	state, err := foldUser(initial, makeEvent(t, eventExternalAccountUnlinked, 3, ExternalAccountUnlinkedPayload{
		Provider: "google",
		Subject:  "sub-123",
	}))
	if err != nil {
		t.Fatalf("foldUser: %v", err)
	}
	if len(state.ExternalAccounts) != 1 {
		t.Fatalf("expected 1 external account after unlink, got %d", len(state.ExternalAccounts))
	}
	if state.ExternalAccounts[0].Provider != "github" || state.ExternalAccounts[0].Subject != "sub-456" {
		t.Errorf("remaining account = %q/%q", state.ExternalAccounts[0].Provider, state.ExternalAccounts[0].Subject)
	}
}

func TestFoldUser_ExternalAccountLinkedPreservesExistingState(t *testing.T) {
	initial := UserState{
		Email:         "j@example.com",
		DisplayName:   "J",
		EmailVerified: true,
		TOTPEnabled:   true,
		TOTPSecret:    []byte{9, 8, 7},
		Credentials:   []WebAuthnCredential{{ID: []byte{1}}},
		ExternalAccounts: []ExternalAccount{
			{Provider: "github", Subject: "old"},
		},
	}
	state, err := foldUser(initial, makeEvent(t, eventExternalAccountLinked, 3, ExternalAccountLinkedPayload{
		Provider: "google",
		Subject:  "new",
	}))
	if err != nil {
		t.Fatalf("foldUser: %v", err)
	}
	// All existing fields must be preserved
	if state.Email != "j@example.com" || state.DisplayName != "J" {
		t.Errorf("Email/DisplayName changed: %q/%q", state.Email, state.DisplayName)
	}
	if !state.EmailVerified || !state.TOTPEnabled {
		t.Errorf("EmailVerified/TOTPEnabled reset: %v/%v", state.EmailVerified, state.TOTPEnabled)
	}
	if len(state.TOTPSecret) != 3 {
		t.Errorf("TOTPSecret len changed: %d", len(state.TOTPSecret))
	}
	if len(state.Credentials) != 1 {
		t.Errorf("Credentials changed: %d", len(state.Credentials))
	}
	// Both external accounts should be present
	if len(state.ExternalAccounts) != 2 {
		t.Fatalf("expected 2 external accounts, got %d", len(state.ExternalAccounts))
	}
}

func TestFoldUser_ExternalAccountUnlinkedNonExistent_NoOp(t *testing.T) {
	initial := UserState{
		Email: "k@example.com",
		ExternalAccounts: []ExternalAccount{
			{Provider: "google", Subject: "sub-123"},
		},
	}
	state, err := foldUser(initial, makeEvent(t, eventExternalAccountUnlinked, 3, ExternalAccountUnlinkedPayload{
		Provider: "github", // not linked
		Subject:  "sub-999",
	}))
	if err != nil {
		t.Fatalf("foldUser: %v", err)
	}
	if len(state.ExternalAccounts) != 1 {
		t.Fatalf("expected 1 external account (no-op), got %d", len(state.ExternalAccounts))
	}
}

func TestFoldUser_ExternalAccountUnlinkedRemovesAllMatching(t *testing.T) {
	// If somehow there are duplicate entries (shouldn't happen via decider guards,
	// but fold should handle it), unlinking removes ALL matching provider+subject.
	initial := UserState{
		Email: "l@example.com",
		ExternalAccounts: []ExternalAccount{
			{Provider: "google", Subject: "dup"},
			{Provider: "google", Subject: "dup"}, // duplicate
			{Provider: "github", Subject: "keep"},
		},
	}
	state, err := foldUser(initial, makeEvent(t, eventExternalAccountUnlinked, 3, ExternalAccountUnlinkedPayload{
		Provider: "google",
		Subject:  "dup",
	}))
	if err != nil {
		t.Fatalf("foldUser: %v", err)
	}
	if len(state.ExternalAccounts) != 1 {
		t.Fatalf("expected 1 external account after removing duplicates, got %d", len(state.ExternalAccounts))
	}
	if state.ExternalAccounts[0].Provider != "github" {
		t.Errorf("expected github to remain, got %q", state.ExternalAccounts[0].Provider)
	}
}
