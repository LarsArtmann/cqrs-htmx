package usermgmt

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

// --- decideLinkExternalAccount guards (Task 14) ---

func TestDecideLinkExternalAccount_Success(t *testing.T) {
	decide := decideLinkExternalAccount(
		id.NewAggregateID(), "google", "sub-123", "user@gmail.com", "User Name",
	)
	events, err := decide(existingState(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 || events[0].Type() != eventExternalAccountLinked {
		t.Fatalf("expected 1 ExternalAccountLinked event, got %d (%v)", len(events), events)
	}
}

func TestDecideLinkExternalAccount_NotFound(t *testing.T) {
	decide := decideLinkExternalAccount(id.NewAggregateID(), "google", "sub-1", "", "")
	if _, err := decide(UserState{}, 0); err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestDecideLinkExternalAccount_Deleted(t *testing.T) {
	decide := decideLinkExternalAccount(id.NewAggregateID(), "google", "sub-1", "", "")
	if _, err := decide(deletedState(), 1); err == nil {
		t.Fatal("expected deleted error")
	}
}

func TestDecideLinkExternalAccount_Duplicate(t *testing.T) {
	state := existingState()
	state.ExternalAccounts = []ExternalAccount{
		{Provider: "google", Subject: "sub-123"},
	}
	decide := decideLinkExternalAccount(id.NewAggregateID(), "google", "sub-123", "", "")
	events, err := decide(state, 1)
	if err == nil {
		t.Fatalf("expected conflict error for duplicate, got %d events", len(events))
	}
}

func TestDecideLinkExternalAccount_DifferentProviderAllowed(t *testing.T) {
	state := existingState()
	state.ExternalAccounts = []ExternalAccount{
		{Provider: "google", Subject: "sub-123"},
	}
	decide := decideLinkExternalAccount(id.NewAggregateID(), "github", "sub-123", "", "")
	events, err := decide(state, 1)
	if err != nil {
		t.Fatalf("unexpected error for different provider: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}

func TestDecideLinkExternalAccount_Invalid(t *testing.T) {
	decide := decideLinkExternalAccount(id.NewAggregateID(), "", "sub-1", "", "")
	if _, err := decide(existingState(), 1); err == nil {
		t.Fatal("expected invalid error for empty provider")
	}
	decide2 := decideLinkExternalAccount(id.NewAggregateID(), "google", "", "", "")
	if _, err := decide2(existingState(), 1); err == nil {
		t.Fatal("expected invalid error for empty subject")
	}
}

// --- decideUnlinkExternalAccount guards (Task 14) ---

func TestDecideUnlinkExternalAccount_Success(t *testing.T) {
	state := existingState()
	state.ExternalAccounts = []ExternalAccount{
		{Provider: "google", Subject: "sub-123"},
		{Provider: "github", Subject: "sub-456"},
	}
	decide := decideUnlinkExternalAccount(id.NewAggregateID(), "google", "sub-123")
	events, err := decide(state, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 || events[0].Type() != eventExternalAccountUnlinked {
		t.Fatalf("expected 1 ExternalAccountUnlinked event, got %d (%v)", len(events), events)
	}
}

func TestDecideUnlinkExternalAccount_NotFound(t *testing.T) {
	decide := decideUnlinkExternalAccount(id.NewAggregateID(), "google", "sub-1")
	if _, err := decide(UserState{}, 0); err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestDecideUnlinkExternalAccount_Deleted(t *testing.T) {
	state := deletedState()
	state.ExternalAccounts = []ExternalAccount{
		{Provider: "google", Subject: "sub-123"},
	}
	decide := decideUnlinkExternalAccount(id.NewAggregateID(), "google", "sub-123")
	if _, err := decide(state, 1); err == nil {
		t.Fatal("expected deleted error")
	}
}

func TestDecideUnlinkExternalAccount_NotLinked(t *testing.T) {
	decide := decideUnlinkExternalAccount(id.NewAggregateID(), "google", "sub-999")
	if _, err := decide(existingState(), 1); err == nil {
		t.Fatal("expected not-linked error")
	}
}

func TestDecideUnlinkExternalAccount_LastAuthMethod(t *testing.T) {
	state := existingState()
	state.ExternalAccounts = []ExternalAccount{
		{Provider: "google", Subject: "sub-123"},
	}
	// 0 credentials + 1 external account = last auth method
	decide := decideUnlinkExternalAccount(id.NewAggregateID(), "google", "sub-123")
	events, err := decide(state, 1)
	if err == nil {
		t.Fatalf("expected last-auth-method rejection, got %d events", len(events))
	}
}

func TestDecideUnlinkExternalAccount_AllowedWithCredential(t *testing.T) {
	state := existingState()
	state.ExternalAccounts = []ExternalAccount{
		{Provider: "google", Subject: "sub-123"},
	}
	state.Credentials = []WebAuthnCredential{{ID: []byte{1}}}
	// Has a WebAuthn credential, so removing the external account is OK
	decide := decideUnlinkExternalAccount(id.NewAggregateID(), "google", "sub-123")
	events, err := decide(state, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}

func TestDecideUnlinkExternalAccount_AllowedWithMultipleExternal(t *testing.T) {
	state := existingState()
	state.ExternalAccounts = []ExternalAccount{
		{Provider: "google", Subject: "sub-123"},
		{Provider: "github", Subject: "sub-456"},
	}
	// Has another external account, so removing one is OK
	decide := decideUnlinkExternalAccount(id.NewAggregateID(), "google", "sub-123")
	events, err := decide(state, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}
