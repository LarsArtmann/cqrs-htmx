package identitymodel

import (
	"reflect"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// Pins the per-event fold contract of the User stream where it is defined.
// FoldUser is replayed against persisted journals, so each row states the one
// change an event is allowed to make: every field the row leaves alone has to
// come out exactly as it went in, and the caller's state must stay untouched.

// foldUserRow is one event-to-state contract of the User stream.
type foldUserRow struct {
	name    string
	event   event.Type
	payload any
	// raw replaces the marshaled payload verbatim.
	raw []byte
	// given adjusts the seeded input before folding.
	given func(state *UserState)
	// want derives the expected state from an untouched copy of the input.
	want func(state UserState, evt event.Event) UserState
}

// seededUserState is a live user that already carries every kind of
// attachment, so a row can tell "left alone" from "reset to the zero value".
func seededUserState() UserState {
	return UserState{
		Email:         "old@example.com",
		DisplayName:   "Old Name",
		EmailVerified: true,
		TOTPEnabled:   true,
		TOTPSecret:    []byte{0x01, 0x02},
		Credentials: []WebAuthnCredential{
			{ID: []byte{0x11}, SignCount: 3},
			{ID: []byte{0x22}, SignCount: 4},
		},
		ExternalAccounts: []ExternalAccount{
			{ExternalAccountCore: ExternalAccountCore{Provider: "google", Subject: "google-1"}},
			{ExternalAccountCore: ExternalAccountCore{Provider: "github", Subject: "github-1"}},
		},
	}
}

func userStreamEvent(t *testing.T, eventType event.Type, payload any) event.Event {
	t.Helper()

	raw, err := MarshalPayload(payload)
	if err != nil {
		t.Fatalf("marshal %s payload: %v", eventType, err)
	}

	return userStreamRawEvent(t, eventType, raw)
}

func userStreamRawEvent(t *testing.T, eventType event.Type, raw []byte) event.Event {
	t.Helper()

	evt, err := event.NewEvent(
		eventType,
		id.DeriveStreamID("fold-user-test", string(eventType)),
		AggregateTypeUser,
		event.Version(1),
		raw,
	)
	if err != nil {
		t.Fatalf("build %s event: %v", eventType, err)
	}

	return evt
}

func runFoldUserRows(t *testing.T, rows []foldUserRow) {
	t.Helper()

	for _, tt := range rows {
		t.Run(tt.name, func(t *testing.T) {
			input := func() UserState {
				state := seededUserState()
				if tt.given != nil {
					tt.given(&state)
				}

				return state
			}

			var evt event.Event
			if tt.raw != nil {
				evt = userStreamRawEvent(t, tt.event, tt.raw)
			} else {
				evt = userStreamEvent(t, tt.event, tt.payload)
			}

			folded := input()

			got, err := FoldUser(folded, evt)
			if err != nil {
				t.Fatalf("FoldUser(%s): %v", tt.event, err)
			}

			if want := tt.want(input(), evt); !reflect.DeepEqual(got, want) {
				t.Fatalf("FoldUser(%s)\n got: %#v\nwant: %#v", tt.event, got, want)
			}

			if !reflect.DeepEqual(folded, input()) {
				t.Fatalf("FoldUser(%s) wrote through to the input state: %#v", tt.event, folded)
			}
		})
	}
}

func TestFoldUser_IdentityEvents(t *testing.T) {
	v := CurrentSchemaVersion
	loser, survivor := SyntheticUserID("loser"), SyntheticUserID("survivor")

	runFoldUserRows(t, []foldUserRow{
		{
			name:    "UserRegistered replaces the whole state",
			event:   EventUserRegistered,
			payload: UserRegisteredPayload{SchemaVersion: v, Email: "new@example.com", DisplayName: "New Name"},
			want: func(_ UserState, _ event.Event) UserState {
				return UserState{Email: "new@example.com", DisplayName: "New Name"}
			},
		},
		{
			name:    "EmailChanged swaps the address and drops what was attached to it",
			event:   EventEmailChanged,
			payload: EmailChangedPayload{SchemaVersion: v, Email: "new@example.com"},
			want: func(state UserState, _ event.Event) UserState {
				state.Email = "new@example.com"
				state.EmailVerified = false
				state.TOTPEnabled = false
				state.TOTPSecret = nil

				return state
			},
		},
		{
			name:    "EmailVerified marks the address verified",
			event:   EventEmailVerified,
			payload: EmailVerifiedPayload{SchemaVersion: v, Email: "old@example.com"},
			given:   func(state *UserState) { state.EmailVerified = false },
			want: func(state UserState, _ event.Event) UserState {
				state.EmailVerified = true

				return state
			},
		},
		{
			name:    "DisplayNameChanged renames the user",
			event:   EventDisplayNameChanged,
			payload: DisplayNameChangedPayload{SchemaVersion: v, DisplayName: "New Name"},
			want: func(state UserState, _ event.Event) UserState {
				state.DisplayName = "New Name"

				return state
			},
		},
		{
			name:    "UserDeleted raises the tombstone",
			event:   EventUserDeleted,
			payload: UserDeletedPayload{SchemaVersion: v, Reason: "gdpr"},
			want: func(state UserState, _ event.Event) UserState {
				state.Deleted = true
				state.DeleteReason = "gdpr"

				return state
			},
		},
		{
			name:  "UserMerged records the losers on the survivor",
			event: EventUserMerged,
			payload: UserMergedPayload{
				SchemaVersion: v,
				LoserIDs:      []UserID{loser},
				Reason:        "duplicate",
			},
			want: func(state UserState, _ event.Event) UserState {
				state.MergedUserIDs = []UserID{loser}

				return state
			},
		},
		{
			name:  "UserMergedInto tombstones the loser and names the survivor",
			event: EventUserMergedInto,
			payload: UserMergedIntoPayload{
				SchemaVersion: v,
				SurvivorID:    survivor,
				Reason:        "duplicate",
			},
			want: func(state UserState, _ event.Event) UserState {
				state.Deleted = true
				state.MergedInto = survivor
				state.DeleteReason = "duplicate"

				return state
			},
		},
		{
			name:    "RolesUpdated is legacy and changes nothing",
			event:   EventRolesUpdated,
			payload: RolesUpdatedPayload{SchemaVersion: v, Domain: "example.com"},
			want: func(state UserState, _ event.Event) UserState {
				return state
			},
		},
	})
}

func TestFoldUser_SignInEvents(t *testing.T) {
	v := CurrentSchemaVersion

	runFoldUserRows(t, []foldUserRow{
		{
			name:  "CredentialAdded appends the authenticator stamped with the event time",
			event: EventCredentialAdded,
			payload: CredentialAddedPayload{
				SchemaVersion: v,
				CredentialCore: CredentialCore{
					ID:        []byte{0x33},
					PublicKey: []byte{0xAA, 0xBB},
					SignCount: 9,
					Name:      "third",
				},
			},
			want: func(state UserState, evt event.Event) UserState {
				state.Credentials = append(state.Credentials, WebAuthnCredential{
					ID:        []byte{0x33},
					PublicKey: []byte{0xAA, 0xBB},
					SignCount: 9,
					Name:      "third",
					CreatedAt: evt.OccurredAt(),
				})

				return state
			},
		},
		{
			name:    "CredentialRemoved drops the named authenticator",
			event:   EventCredentialRemoved,
			payload: CredentialRemovedPayload{SchemaVersion: v, ID: []byte{0x11}},
			want: func(state UserState, _ event.Event) UserState {
				state.Credentials = []WebAuthnCredential{{ID: []byte{0x22}, SignCount: 4}}

				return state
			},
		},
		{
			name:  "CredentialUsed advances the named authenticator",
			event: EventCredentialUsed,
			payload: CredentialUsedPayload{
				SchemaVersion: v,
				CredentialID:  []byte{0x22},
				SignCount:     40,
				UserPresent:   true,
			},
			want: func(state UserState, _ event.Event) UserState {
				state.Credentials = []WebAuthnCredential{
					{ID: []byte{0x11}, SignCount: 3},
					{ID: []byte{0x22}, SignCount: 40, UserPresent: true},
				}

				return state
			},
		},
		{
			name:    "TOTPEnabled switches the second factor on",
			event:   EventTOTPEnabled,
			payload: TOTPEnabledPayload{SchemaVersion: v, Secret: []byte{0x0A, 0x0B}},
			given:   func(state *UserState) { state.TOTPEnabled = false; state.TOTPSecret = nil },
			want: func(state UserState, _ event.Event) UserState {
				state.TOTPEnabled = true
				state.TOTPSecret = []byte{0x0A, 0x0B}

				return state
			},
		},
		{
			name:  "TOTPDisabled switches the second factor off without reading a payload",
			event: EventTOTPDisabled,
			raw:   []byte("not a payload"),
			want: func(state UserState, _ event.Event) UserState {
				state.TOTPEnabled = false
				state.TOTPSecret = nil

				return state
			},
		},
		{
			name:  "ExternalAccountLinked appends the identity stamped with the event time",
			event: EventExternalAccountLinked,
			payload: ExternalAccountLinkedPayload{
				SchemaVersion: v,
				ExternalAccountCore: ExternalAccountCore{
					Provider: "gitlab",
					Subject:  "gitlab-1",
				},
			},
			want: func(state UserState, evt event.Event) UserState {
				state.ExternalAccounts = append(state.ExternalAccounts, ExternalAccount{
					ExternalAccountCore: ExternalAccountCore{Provider: "gitlab", Subject: "gitlab-1"},
					LinkedAt:            evt.OccurredAt(),
				})

				return state
			},
		},
		{
			name:  "ExternalAccountUnlinked drops the matching identity only",
			event: EventExternalAccountUnlinked,
			payload: ExternalAccountUnlinkedPayload{
				SchemaVersion: v,
				Provider:      "google",
				Subject:       "google-1",
			},
			want: func(state UserState, _ event.Event) UserState {
				state.ExternalAccounts = []ExternalAccount{
					{ExternalAccountCore: ExternalAccountCore{Provider: "github", Subject: "github-1"}},
				}

				return state
			},
		},
	})
}

// TestFoldUser_UndecodablePayload: an event whose payload does not decode must
// fail the fold as corruption and hand the input back untouched.
func TestFoldUser_UndecodablePayload(t *testing.T) {
	decoded := []event.Type{
		EventUserRegistered,
		EventRolesUpdated,
		EventEmailChanged,
		EventDisplayNameChanged,
		EventUserDeleted,
		EventUserMerged,
		EventUserMergedInto,
		EventCredentialAdded,
		EventCredentialRemoved,
		EventCredentialUsed,
		EventEmailVerified,
		EventTOTPEnabled,
		EventExternalAccountLinked,
		EventExternalAccountUnlinked,
	}

	for _, eventType := range decoded {
		t.Run(string(eventType), func(t *testing.T) {
			before := seededUserState()

			got, err := FoldUser(before, userStreamRawEvent(t, eventType, []byte("not a payload")))
			if err == nil {
				t.Fatalf("FoldUser(%s) accepted an undecodable payload", eventType)
			}

			if fam := errorfamily.Classify(err); fam != errorfamily.Corruption {
				t.Fatalf("FoldUser(%s) family = %v, want Corruption: %v", eventType, fam, err)
			}

			if !reflect.DeepEqual(got, before) {
				t.Fatalf("FoldUser(%s) returned changed state with the error: %#v", eventType, got)
			}
		})
	}
}

// TestFoldUser_UnknownEventIsRejected: an event type outside the User stream's
// vocabulary is a rejection that hands the input back untouched.
func TestFoldUser_UnknownEventIsRejected(t *testing.T) {
	before := seededUserState()

	got, err := FoldUser(before, userStreamRawEvent(t, "SomeFutureUserEvent", []byte("{}")))
	if err == nil {
		t.Fatal("FoldUser accepted an event type outside the User stream's vocabulary")
	}

	if fam := errorfamily.Classify(err); fam != errorfamily.Rejection {
		t.Fatalf("family = %v, want Rejection: %v", fam, err)
	}

	if !strings.Contains(err.Error(), "usermgmt.user.unknown_event") {
		t.Fatalf("error does not carry the rejection code: %v", err)
	}

	if !reflect.DeepEqual(got, before) {
		t.Fatalf("FoldUser returned changed state with the error: %#v", got)
	}
}
