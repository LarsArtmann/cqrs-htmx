package identitymodel

import (
	"fmt"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// TEMPORARY differential harness (not for commit): legacyFoldUser is the
// verbatim pre-refactor FoldUser from commit 8a0b680b.

func legacyFoldUser(
	state UserState,
	evt event.Event,
) (UserState, error) {
	next := state

	switch evt.Type() {
	case EventUserRegistered:
		p, err := UnmarshalPayload[UserRegisteredPayload](evt)
		if err != nil {
			return state, err
		}
		next = UserState{
			Email:       p.Email,
			DisplayName: p.DisplayName,
		}

	case EventRolesUpdated:
		_, err := UnmarshalPayload[RolesUpdatedPayload](evt)
		if err != nil {
			return state, err
		}

	case EventEmailChanged:
		p, err := UnmarshalPayload[EmailChangedPayload](evt)
		if err != nil {
			return state, err
		}
		next.Email = p.Email
		next.EmailVerified = false
		next.TOTPEnabled = false
		next.TOTPSecret = nil

	case EventDisplayNameChanged:
		p, err := UnmarshalPayload[DisplayNameChangedPayload](evt)
		if err != nil {
			return state, err
		}
		next.DisplayName = p.DisplayName

	case EventUserDeleted:
		p, err := UnmarshalPayload[UserDeletedPayload](evt)
		if err != nil {
			return state, err
		}
		next.Deleted = true
		next.DeleteReason = p.Reason

	case EventUserMerged:
		p, err := UnmarshalPayload[UserMergedPayload](evt)
		if err != nil {
			return state, err
		}
		for _, loser := range p.LoserIDs {
			next.MergedUserIDs = withAppendedUnique(next.MergedUserIDs, loser)
		}

	case EventUserMergedInto:
		p, err := UnmarshalPayload[UserMergedIntoPayload](evt)
		if err != nil {
			return state, err
		}
		next.Deleted = true
		next.MergedInto = p.SurvivorID
		next.DeleteReason = p.Reason

	case EventCredentialAdded:
		p, err := UnmarshalPayload[CredentialAddedPayload](evt)
		if err != nil {
			return state, err
		}
		next.Credentials = append(next.Credentials, NewCredentialFromPayload(p, evt.OccurredAt()))

	case EventCredentialRemoved:
		var err error
		next, err = applyCredentialRemoved(next, evt)
		if err != nil {
			return state, err
		}

	case EventCredentialUsed:
		var err error
		next, err = applyCredentialUsed(next, evt)
		if err != nil {
			return state, err
		}

	case EventEmailVerified:
		_, err := UnmarshalPayload[EmailVerifiedPayload](evt)
		if err != nil {
			return state, err
		}
		next.EmailVerified = true

	case EventTOTPEnabled:
		p, err := UnmarshalPayload[TOTPEnabledPayload](evt)
		if err != nil {
			return state, err
		}
		next.TOTPEnabled = true
		next.TOTPSecret = slices.Clone(p.Secret)

	case EventTOTPDisabled:
		next.TOTPEnabled = false
		next.TOTPSecret = nil

	case EventExternalAccountLinked:
		p, err := UnmarshalPayload[ExternalAccountLinkedPayload](evt)
		if err != nil {
			return state, err
		}
		next.ExternalAccounts = append(next.ExternalAccounts, ExternalAccount{
			Provider:    p.Provider,
			Subject:     p.Subject,
			Email:       p.Email,
			DisplayName: p.DisplayName,
			LinkedAt:    evt.OccurredAt(),
		})

	case EventExternalAccountUnlinked:
		p, err := UnmarshalPayload[ExternalAccountUnlinkedPayload](evt)
		if err != nil {
			return state, err
		}
		filtered := make([]ExternalAccount, 0, len(next.ExternalAccounts))
		for _, ea := range next.ExternalAccounts {
			if ea.Provider != p.Provider || ea.Subject != p.Subject {
				filtered = append(filtered, ea)
			}
		}
		next.ExternalAccounts = filtered

	default:
		return state, errorfamily.NewRejection(
			"usermgmt.user.unknown_event",
			"FoldUser received unknown event type: "+string(evt.Type()),
		)
	}

	return next, nil
}

func tmpEvent(t *testing.T, typ event.Type, payload any, raw []byte) event.Event {
	t.Helper()

	data := raw
	if data == nil {
		var err error
		data, err = MarshalPayload(payload)
		if err != nil {
			t.Fatalf("marshal %s: %v", typ, err)
		}
	}

	evt, err := event.NewEvent(typ, id.DeriveStreamID("diff", string(typ)), AggregateTypeUser, event.Version(1), data)
	if err != nil {
		t.Fatalf("event %s: %v", typ, err)
	}

	return evt
}

func tmpCorpus(t *testing.T) []event.Event {
	t.Helper()

	corrupt := []byte("{not-a-payload")
	v := CurrentSchemaVersion
	loserA, loserB := SyntheticUserID("loser-a"), SyntheticUserID("loser-b")

	type spec struct {
		typ     event.Type
		payload any
		corrupt bool
	}

	specs := []spec{
		{EventUserRegistered, UserRegisteredPayload{SchemaVersion: v, Email: "a@example.com", DisplayName: "A"}, true},
		{EventUserRegistered, UserRegisteredPayload{SchemaVersion: v, Email: "b@example.com"}, false},
		{EventRolesUpdated, RolesUpdatedPayload{SchemaVersion: v, Domain: "d"}, true},
		{EventEmailChanged, EmailChangedPayload{SchemaVersion: v, Email: "c@example.com"}, true},
		{EventDisplayNameChanged, DisplayNameChangedPayload{SchemaVersion: v, DisplayName: "C"}, true},
		{EventUserDeleted, UserDeletedPayload{SchemaVersion: v, Reason: "gdpr"}, true},
		{EventUserMerged, UserMergedPayload{SchemaVersion: v, LoserIDs: []UserID{loserA, loserB, loserA}, Reason: "dup"}, true},
		{EventUserMerged, UserMergedPayload{SchemaVersion: v, LoserIDs: nil}, false},
		{EventUserMergedInto, UserMergedIntoPayload{SchemaVersion: v, SurvivorID: loserB, Reason: "dup"}, true},
		{EventCredentialAdded, CredentialAddedPayload{SchemaVersion: v, CredentialCore: CredentialCore{ID: []byte{1}, SignCount: 7, Transports: []string{"usb"}}}, true},
		{EventCredentialAdded, CredentialAddedPayload{SchemaVersion: v, CredentialCore: CredentialCore{ID: []byte{0xFE}, Name: "k2"}}, false},
		{EventCredentialRemoved, CredentialRemovedPayload{SchemaVersion: v, ID: []byte{1}}, true},
		{EventCredentialRemoved, CredentialRemovedPayload{SchemaVersion: v, ID: []byte{0xFD}}, false},
		{EventCredentialUsed, CredentialUsedPayload{SchemaVersion: v, CredentialID: []byte{1}, SignCount: 8, UserPresent: true, BackupState: true}, true},
		{EventCredentialUsed, CredentialUsedPayload{SchemaVersion: v, CredentialID: []byte{0xFC}, SignCount: 1}, false},
		{EventEmailVerified, EmailVerifiedPayload{SchemaVersion: v, Email: "a@example.com"}, true},
		{EventTOTPEnabled, TOTPEnabledPayload{SchemaVersion: v, Secret: []byte{9, 9}}, true},
		{EventTOTPDisabled, TOTPDisabledPayload{SchemaVersion: v}, true},
		{EventExternalAccountLinked, ExternalAccountLinkedPayload{SchemaVersion: v, ExternalAccountCore: ExternalAccountCore{Provider: "google", Subject: "g1", Email: "g@example.com"}}, true},
		{EventExternalAccountLinked, ExternalAccountLinkedPayload{SchemaVersion: v, ExternalAccountCore: ExternalAccountCore{Provider: "google", Subject: "s0"}}, false},
		{EventExternalAccountUnlinked, ExternalAccountUnlinkedPayload{SchemaVersion: v, Provider: "google", Subject: "s0"}, true},
		{EventExternalAccountUnlinked, ExternalAccountUnlinkedPayload{SchemaVersion: v, Provider: "github", Subject: "nope"}, false},
	}

	var out []event.Event

	for _, s := range specs {
		out = append(out, tmpEvent(t, s.typ, s.payload, nil))
		if s.corrupt {
			out = append(out, tmpEvent(t, s.typ, nil, corrupt))
		}
	}

	for _, typ := range []event.Type{"UserActivated", "UserMergedd", EventTenantCreated, EventMemberAdded, EventBotDeleted} {
		out = append(out, tmpEvent(t, typ, map[string]any{"schema_version": v}, nil))
	}

	return out
}

func tmpSeed(rnd *rand.Rand) UserState {
	n := rnd.IntN(4)
	creds := make([]WebAuthnCredential, 0, n+2)
	eas := make([]ExternalAccount, 0, n+2)
	merged := make([]UserID, 0, n+2)

	for i := range n {
		creds = append(creds, WebAuthnCredential{ID: []byte{byte(i + 1)}, SignCount: uint32(i), Transports: []string{"usb"}})
		eas = append(eas, ExternalAccount{ExternalAccountCore: ExternalAccountCore{Provider: "google", Subject: fmt.Sprintf("s%d", i)}})
		merged = append(merged, SyntheticUserID(fmt.Sprintf("m%d", i)))
	}

	return UserState{
		Email:            "seed@example.com",
		DisplayName:      "Seed",
		EmailVerified:    rnd.IntN(2) == 0,
		TOTPEnabled:      rnd.IntN(2) == 0,
		TOTPSecret:       []byte{1, 2, 3},
		Deleted:          rnd.IntN(2) == 0,
		DeleteReason:     "old",
		Credentials:      creds,
		ExternalAccounts: eas,
		MergedUserIDs:    merged,
	}
}

func tmpDeepCopy(s UserState) UserState {
	out := s
	out.Credentials = slices.Clone(s.Credentials)
	out.ExternalAccounts = slices.Clone(s.ExternalAccounts)
	out.MergedUserIDs = slices.Clone(s.MergedUserIDs)
	out.TOTPSecret = slices.Clone(s.TOTPSecret)

	return out
}

func tmpErrSig(err error) string {
	if err == nil {
		return "<nil>"
	}

	return fmt.Sprintf("%T|%s", err, err.Error())
}

func TestTmpFoldUserDifferential(t *testing.T) {
	corpus := tmpCorpus(t)
	rnd := rand.New(rand.NewPCG(20261127, 4))
	steps := 0
	errSteps := 0

	for range 40 {
		seed := tmpSeed(rnd)
		legacy, current := tmpDeepCopy(seed), tmpDeepCopy(seed)

		for range 300 {
			evt := corpus[rnd.IntN(len(corpus))]

			legacyNext, legacyErr := legacyFoldUser(legacy, evt)
			currentNext, currentErr := FoldUser(current, evt)

			if tmpErrSig(legacyErr) != tmpErrSig(currentErr) {
				t.Fatalf("%s: error mismatch\n legacy: %s\ncurrent: %s", evt.Type(), tmpErrSig(legacyErr), tmpErrSig(currentErr))
			}

			if !reflect.DeepEqual(legacyNext, currentNext) {
				t.Fatalf("%s: state mismatch\n legacy: %#v\ncurrent: %#v", evt.Type(), legacyNext, currentNext)
			}

			if legacyErr != nil {
				errSteps++

				if !reflect.DeepEqual(legacyNext, legacy) || !reflect.DeepEqual(currentNext, current) {
					t.Fatalf("%s: error path must return the untouched input state", evt.Type())
				}
			}

			legacy, current = legacyNext, currentNext
			steps++
		}
	}

	t.Logf("steps=%d errorSteps=%d corpus=%d", steps, errSteps, len(corpus))
}
