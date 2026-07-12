package usermgmt

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"pgregory.net/rapid"
)

// Property-based tests for foldUser() using pgregory.net/rapid.
// Verifies key invariants with random inputs.

var propAggID = id.NewAggregateID() //nolint:gochecknoglobals // test fixture

func rapidEmail() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-z]{3,8}@test\.com`)
}

func rapidRole() *rapid.Generator[Role] {
	return rapid.SampledFrom([]Role{RoleAdmin, RoleUser, RoleViewer, RoleOwner})
}

func mustPropEvent(eventType event.Type, version event.Version, payload any) event.Event {
	b, err := marshalPayload(payload)
	if err != nil {
		panic(err)
	}
	evt, err := event.NewEvent(eventType, propAggID, aggregateTypeUser, version, b)
	if err != nil {
		panic(err)
	}
	return evt
}

func TestFoldUserProperty_RegistrationSetsEmail(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		email := rapidEmail().Draw(t, "email")
		roles := rapid.SliceOfN(rapidRole(), 1, 4).Draw(t, "roles")

		evt := mustPropEvent(eventUserRegistered, 1, UserRegisteredPayload{
			Email: email,
			Roles: roles,
		})
		state, err := foldUser(UserState{}, evt)
		if err != nil {
			t.Fatalf("foldUser: %v", err)
		}
		if state.Email != email {
			t.Errorf("email = %q, want %q", state.Email, email)
		}
		if state.Deleted {
			t.Error("newly registered user should not be deleted")
		}
	})
}

func TestFoldUserProperty_EmailChangedPreservesRoles(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		initial := UserState{
			Email: "before@test.com",
		}
		newEmail := rapidEmail().Draw(t, "newEmail")

		evt := mustPropEvent(eventEmailChanged, 2, EmailChangedPayload{
			Email: newEmail,
		})
		state, err := foldUser(initial, evt)
		if err != nil {
			t.Fatalf("foldUser: %v", err)
		}
		if state.Email != newEmail {
			t.Errorf("email = %q, want %q", state.Email, newEmail)
		}
	})
}

func TestFoldUserProperty_DisplayNameChangedPreservesEmail(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		email := rapidEmail().Draw(t, "email")
		newName := rapid.StringN(1, 50, 50).Draw(t, "displayName")

		initial := UserState{Email: email}
		evt := mustPropEvent(eventDisplayNameChanged, 2, DisplayNameChangedPayload{
			DisplayName: newName,
		})
		state, err := foldUser(initial, evt)
		if err != nil {
			t.Fatalf("foldUser: %v", err)
		}
		if state.Email != email {
			t.Errorf("email changed: %q → %q", email, state.Email)
		}
		if state.DisplayName != newName {
			t.Errorf("displayName = %q, want %q", state.DisplayName, newName)
		}
	})
}

func TestFoldUserProperty_DeletedSetsTombstone(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		initial := UserState{
			Email: "doomed@test.com",
		}
		reason := rapid.StringN(0, 100, 100).Draw(t, "reason")

		evt := mustPropEvent(eventUserDeleted, 2, UserDeletedPayload{
			Reason: reason,
		})
		state, err := foldUser(initial, evt)
		if err != nil {
			t.Fatalf("foldUser: %v", err)
		}
		if !state.Deleted {
			t.Error("expected Deleted=true after UserDeleted event")
		}
		if state.DeleteReason != reason {
			t.Errorf("DeleteReason = %q, want %q", state.DeleteReason, reason)
		}
	})
}

func TestFoldUserProperty_CredentialAddRemoveRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		initial := UserState{
			Email:       "cred@test.com",
			Credentials: []WebAuthnCredential{},
		}
		credID := rapid.SliceOfN(rapid.Byte(), 16, 32).Draw(t, "credID")

		addEvt := mustPropEvent(eventCredentialAdded, 2, CredentialAddedPayload{
			credentialCore: credentialCore{
				ID:             credID,
				PublicKey:      []byte{0x01, 0x02},
				SignCount:      42,
				BackupEligible: true,
			},
		})
		added, err := foldUser(initial, addEvt)
		if err != nil {
			t.Fatalf("foldUser add: %v", err)
		}
		if len(added.Credentials) != 1 {
			t.Fatalf("expected 1 credential after add, got %d", len(added.Credentials))
		}
		if added.Credentials[0].SignCount != 42 {
			t.Errorf("SignCount = %d, want 42", added.Credentials[0].SignCount)
		}

		removeEvt := mustPropEvent(eventCredentialRemoved, 3, CredentialRemovedPayload{
			ID: credID,
		})
		removed, err := foldUser(added, removeEvt)
		if err != nil {
			t.Fatalf("foldUser remove: %v", err)
		}
		if len(removed.Credentials) != 0 {
			t.Errorf("expected 0 credentials after remove, got %d", len(removed.Credentials))
		}
	})
}

func TestFoldUserProperty_RolesUpdatedPreservesEmail(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		email := rapidEmail().Draw(t, "email")
		newRoles := rapid.SliceOfN(rapidRole(), 1, 3).Draw(t, "newRoles")

		initial := UserState{Email: email}
		evt := mustPropEvent(eventRolesUpdated, 2, RolesUpdatedPayload{
			Roles:  newRoles,
			Domain: "test",
		})
		state, err := foldUser(initial, evt)
		if err != nil {
			t.Fatalf("foldUser: %v", err)
		}
		if state.Email != email {
			t.Errorf("email changed: %q → %q", email, state.Email)
		}
	})
}

func TestFoldUserProperty_UnknownEventReturnsError(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		email := rapidEmail().Draw(t, "email")
		initial := UserState{
			Email: email,
		}
		unknownEvt, err := event.NewEvent(
			event.Type("FutureUnknownEvent"), propAggID, aggregateTypeUser, 1,
			[]byte(`{}`),
		)
		if err != nil {
			t.Fatalf("create unknown event: %v", err)
		}
		state, err := foldUser(initial, unknownEvt)
		if err == nil {
			t.Fatalf("foldUser should return error for unknown event")
		}
		if state.Email != initial.Email {
			t.Errorf("email changed on unknown event")
		}
	})
}

func TestFoldUserProperty_Idempotency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		email := rapidEmail().Draw(t, "email")
		displayName := rapid.StringN(1, 50, 50).Draw(t, "displayName")

		evt := mustPropEvent(eventUserRegistered, 1, UserRegisteredPayload{
			Email:       email,
			DisplayName: displayName,
		})
		s1, err := foldUser(UserState{}, evt)
		if err != nil {
			t.Fatalf("first fold: %v", err)
		}
		s2, err := foldUser(UserState{}, evt)
		if err != nil {
			t.Fatalf("second fold: %v", err)
		}
		if s1.Email != s2.Email || s1.DisplayName != s2.DisplayName {
			t.Error("idempotency violated: same event produces different states")
		}
	})
}
