package usermgmt

import (
	"bytes"
	"errors"
	"reflect"
	"slices"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-codec"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

const (
	mergeSurvivorEmail = "lars@larsartmann.cloud"
	mergeReason        = "nine rows, one human"
)

// mergeStream resolves the user stream a merge decision targets.
func mergeStream(t *testing.T, userID UserID) id.StreamID {
	t.Helper()
	streamID, err := aggIDFromUser(userID)
	if err != nil {
		t.Fatalf("aggIDFromUser(%s): %v", userID, err)
	}
	return streamID
}

// requireDomainError asserts the typed error a decider must answer with.
func requireDomainError(t *testing.T, err error, family errorfamily.Family, code string) *errorfamily.Error {
	t.Helper()
	if err == nil {
		t.Fatalf("expected a %v error with code %q, got nil", family, code)
	}
	ef, ok := errors.AsType[*errorfamily.Error](err)
	if !ok {
		t.Fatalf("expected an errorfamily error with code %q, got %T: %v", code, err, err)
	}
	if ef.Family() != family {
		t.Errorf("family = %v, want %v (code %q)", ef.Family(), family, code)
	}
	if ef.Code() != code {
		t.Errorf("code = %q, want %q", ef.Code(), code)
	}
	return ef
}

func mergeEventTypes(events []event.Event) []event.Type {
	types := make([]event.Type, len(events))
	for i, evt := range events {
		types[i] = evt.Type()
	}
	return types
}

func mergeEventEncodings(events []event.Event) []codec.Encoding {
	encodings := make([]codec.Encoding, len(events))
	for i, evt := range events {
		encodings[i] = evt.Encoding()
	}
	return encodings
}

func mergeEventVersions(events []event.Event) []event.Version {
	versions := make([]event.Version, len(events))
	for i, evt := range events {
		versions[i] = evt.Version()
	}
	return versions
}

// replayUser folds events onto a start state the way the repository does.
func replayUser(t *testing.T, start UserState, events []event.Event) UserState {
	t.Helper()
	state := start
	for _, evt := range events {
		next, err := foldUser(state, evt)
		if err != nil {
			t.Fatalf("foldUser(%s): %v", evt.Type(), err)
		}
		state = next
	}
	return state
}

// replayBot folds events onto a start state the way the repository does.
func replayBot(t *testing.T, start BotState, events []event.Event) BotState {
	t.Helper()
	state := start
	for _, evt := range events {
		next, err := foldBot(state, evt)
		if err != nil {
			t.Fatalf("foldBot(%s): %v", evt.Type(), err)
		}
		state = next
	}
	return state
}

// mergeSurvivor is a live user holding one credential and one external
// account, without TOTP and with an unverified email.
func mergeSurvivor() UserState {
	return UserState{
		Email:       mergeSurvivorEmail,
		DisplayName: "Lars",
		Credentials: []identitymodel.WebAuthnCredential{{
			CredentialCore: identitymodel.CredentialCore{
				ID:        []byte("survivor-cred"),
				PublicKey: []byte("survivor-key"),
			},
		}},
		ExternalAccounts: []identitymodel.ExternalAccount{{
			ExternalAccountCore: identitymodel.ExternalAccountCore{
				Provider: "github",
				Subject:  "survivor-subject",
			},
		}},
	}
}

func mergeCredential(id, publicKey string, verified bool) identitymodel.WebAuthnCredential {
	return identitymodel.WebAuthnCredential{
		CredentialCore: identitymodel.CredentialCore{
			ID:              []byte(id),
			PublicKey:       []byte(publicKey),
			AttestationType: "none",
			Transports:      []string{"internal"},
			AAGUID:          []byte{1, 2},
			SignCount:       7,
			UserPresent:     true,
			UserVerified:    verified,
			BackupEligible:  true,
			BackupState:     true,
			Name:            "passkey " + id,
		},
	}
}

func mergeAccount(provider, subject, email, displayName string) identitymodel.ExternalAccount {
	return identitymodel.ExternalAccount{
		ExternalAccountCore: identitymodel.ExternalAccountCore{
			Provider:    provider,
			Subject:     subject,
			Email:       email,
			DisplayName: displayName,
		},
	}
}

func contributionFor(userID UserID) identitymodel.MergeContribution {
	return identitymodel.MergeContribution{UserID: userID}
}

// richContribution carries every asset a loser can hand over, including the
// ones the survivor already holds and duplicates inside the same batch.
func richContribution(loserID UserID) identitymodel.MergeContribution {
	return identitymodel.MergeContribution{
		UserID:        loserID,
		Email:         mergeSurvivorEmail,
		EmailVerified: true,
		TOTPSecret:    []byte("loser-totp-secret"),
		Credentials: []identitymodel.WebAuthnCredential{
			mergeCredential("survivor-cred", "duplicate-of-the-survivor", false),
			{CredentialCore: identitymodel.CredentialCore{PublicKey: []byte("no-id")}},
			mergeCredential("loser-cred-a", "key-a", true),
			mergeCredential("loser-cred-b", "key-b", false),
			mergeCredential("loser-cred-a", "key-a-again", true),
		},
		ExternalAccounts: []identitymodel.ExternalAccount{
			mergeAccount("github", "survivor-subject", "", ""),
			mergeAccount("github", "loser-subject-1", mergeSurvivorEmail, "Lars"),
			mergeAccount("google", "loser-subject-2", "", ""),
			mergeAccount("", "providerless", "", ""),
		},
	}
}

func TestDecideMergeUsers_Guards(t *testing.T) {
	survivorID := GenerateUserID()
	aggID := mergeStream(t, survivorID)
	mergedAway := mergeSurvivor()
	mergedAway.MergedInto = GenerateUserID()
	mergedAway.Deleted = true

	tests := []struct {
		name          string
		state         UserState
		contributions []identitymodel.MergeContribution
		wantFamily    errorfamily.Family
		wantCode      string
	}{
		{
			name:          "unregistered survivor",
			state:         UserState{},
			contributions: []identitymodel.MergeContribution{contributionFor(GenerateUserID())},
			wantFamily:    errorfamily.Rejection,
			wantCode:      "usermgmt.merge_users.not_found",
		},
		{
			name:          "survivor already merged away",
			state:         mergedAway,
			contributions: []identitymodel.MergeContribution{contributionFor(GenerateUserID())},
			wantFamily:    errorfamily.Conflict,
			wantCode:      "usermgmt.merge_users.survivor_merged",
		},
		{
			name:          "deleted survivor",
			state:         deletedState(),
			contributions: []identitymodel.MergeContribution{contributionFor(GenerateUserID())},
			wantFamily:    errorfamily.Rejection,
			wantCode:      "usermgmt.merge_users.deleted",
		},
		{
			name:          "no contributions",
			state:         existingState(),
			contributions: nil,
			wantFamily:    errorfamily.Rejection,
			wantCode:      "usermgmt.merge_users.no_contributions",
		},
		{
			name:  "contribution without a user ID",
			state: existingState(),
			contributions: []identitymodel.MergeContribution{
				contributionFor(GenerateUserID()),
				contributionFor(UserID{}),
			},
			wantFamily: errorfamily.Rejection,
			wantCode:   "usermgmt.merge_users.invalid_contribution",
		},
		{
			name:          "survivor merged into itself",
			state:         existingState(),
			contributions: []identitymodel.MergeContribution{contributionFor(survivorID)},
			wantFamily:    errorfamily.Rejection,
			wantCode:      "usermgmt.merge_users.self_merge",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events, err := decideMergeUsers(aggID, tt.contributions, mergeReason)(tt.state, event.Version(2))
			ef := requireDomainError(t, err, tt.wantFamily, tt.wantCode)
			if len(events) != 0 {
				t.Fatalf("a refused decision must emit nothing, got %d events", len(events))
			}
			if tt.wantCode == "usermgmt.merge_users.survivor_merged" &&
				ef.ContextValue("merged_into") != mergedAway.MergedInto.Get().String() {
				t.Errorf("merged_into = %q, want the real survivor %q",
					ef.ContextValue("merged_into"), mergedAway.MergedInto.Get().String())
			}
		})
	}
}

func TestDecideMergeUsers_AdoptsInOrderOnOneAppend(t *testing.T) {
	survivorID := GenerateUserID()
	loserID := GenerateUserID()
	aggID := mergeStream(t, survivorID)

	events, err := decideMergeUsers(aggID, []identitymodel.MergeContribution{richContribution(loserID)}, mergeReason)(
		mergeSurvivor(), event.Version(4))
	if err != nil {
		t.Fatalf("decideMergeUsers: %v", err)
	}

	wantTypes := []event.Type{
		eventCredentialAdded, eventCredentialAdded,
		eventExternalAccountLinked, eventExternalAccountLinked,
		eventTOTPEnabled,
		eventEmailVerified,
		eventUserMerged,
	}
	if got := mergeEventTypes(events); !slices.Equal(got, wantTypes) {
		t.Fatalf("event order = %v, want %v", got, wantTypes)
	}
	wantVersions := []event.Version{5, 6, 7, 8, 9, 10, 11}
	if got := mergeEventVersions(events); !slices.Equal(got, wantVersions) {
		t.Errorf("versions = %v, want %v", got, wantVersions)
	}
	wantEncodings := []codec.Encoding{
		codec.EncodingCBOR, codec.EncodingCBOR,
		codec.EncodingJSON, codec.EncodingJSON,
		codec.EncodingCBOR,
		codec.EncodingJSON,
		codec.EncodingJSON,
	}
	if got := mergeEventEncodings(events); !slices.Equal(got, wantEncodings) {
		t.Errorf("encodings = %v, want %v", got, wantEncodings)
	}
	requireOnSurvivorStream(t, events, aggID)

	requireAdoptedCredential(t, events[0], "loser-cred-a", "key-a", true)
	requireCredentialKeepsItsFields(t, events[0])
	requireAdoptedCredential(t, events[1], "loser-cred-b", "key-b", false)
	requireAdoptedAccount(t, events[2], "github", "loser-subject-1")
	requireAccountKeepsItsProfile(t, events[2])
	requireAdoptedAccount(t, events[3], "google", "loser-subject-2")
	requireAdoptedTOTP(t, events[4])
	requireAdoptedVerification(t, events[5])
	requireMergeRecord(t, events[6], loserID)
}

// requireOnSurvivorStream checks that every event lands on the survivor's
// stream at schema version 1.
func requireOnSurvivorStream(t *testing.T, events []event.Event, aggID id.StreamID) {
	t.Helper()
	for i, evt := range events {
		if evt.StreamID().Get() != aggID.Get() {
			t.Errorf("event %d lands on stream %q, want the survivor stream %q", i, evt.StreamID().Get(), aggID.Get())
		}
		if evt.SchemaVersion() != event.SchemaVersion(1) {
			t.Errorf("event %d schema version = %v, want 1", i, evt.SchemaVersion())
		}
	}
}

// requireAdoptedCredential decodes a CredentialAdded event and checks which
// credential it carries.
func requireAdoptedCredential(t *testing.T, evt event.Event, wantID, wantKey string, wantVerified bool) {
	t.Helper()
	got, err := unmarshalPayload[CredentialAddedPayload](evt)
	if err != nil {
		t.Fatalf("decode %s: %v", evt.Type(), err)
	}
	if string(got.ID) != wantID || string(got.PublicKey) != wantKey || got.UserVerified != wantVerified {
		t.Errorf("%s carries credential %s (verified %t), want %s (verified %t)",
			evt.Type(), got.ID, got.UserVerified, wantID, wantVerified)
	}
}

// requireCredentialKeepsItsFields checks that an adopted credential arrives
// with every field the loser's credential held.
func requireCredentialKeepsItsFields(t *testing.T, evt event.Event) {
	t.Helper()
	got, err := unmarshalPayload[CredentialAddedPayload](evt)
	if err != nil {
		t.Fatalf("decode %s: %v", evt.Type(), err)
	}
	if !slices.Equal(got.Transports, []string{"internal"}) || !slices.Equal(got.AAGUID, []byte{1, 2}) ||
		got.SignCount != 7 || got.AttestationType != "none" || got.Name != "passkey loser-cred-a" {
		t.Errorf("adopted credential lost fields: %+v", got)
	}
}

// requireAdoptedAccount decodes an ExternalAccountLinked event and checks
// which provider subject it carries.
func requireAdoptedAccount(t *testing.T, evt event.Event, wantProvider, wantSubject string) {
	t.Helper()
	got, err := unmarshalPayload[ExternalAccountLinkedPayload](evt)
	if err != nil {
		t.Fatalf("decode %s: %v", evt.Type(), err)
	}
	if got.Provider != wantProvider || got.Subject != wantSubject {
		t.Errorf(
			"%s carries account %s/%s, want %s/%s",
			evt.Type(),
			got.Provider,
			got.Subject,
			wantProvider,
			wantSubject,
		)
	}
}

// requireAccountKeepsItsProfile checks that an adopted external account
// arrives with the profile the loser's account held.
func requireAccountKeepsItsProfile(t *testing.T, evt event.Event) {
	t.Helper()
	got, err := unmarshalPayload[ExternalAccountLinkedPayload](evt)
	if err != nil {
		t.Fatalf("decode %s: %v", evt.Type(), err)
	}
	if got.Email != mergeSurvivorEmail || got.DisplayName != "Lars" {
		t.Errorf("adopted account profile = %q / %q, want %q / %q",
			got.Email, got.DisplayName, mergeSurvivorEmail, "Lars")
	}
}

func requireAdoptedTOTP(t *testing.T, evt event.Event) {
	t.Helper()
	got, err := unmarshalPayload[TOTPEnabledPayload](evt)
	if err != nil {
		t.Fatalf("decode %s: %v", evt.Type(), err)
	}
	if !bytes.Equal(got.Secret, []byte("loser-totp-secret")) {
		t.Errorf("TOTP secret = %q, want the loser's", got.Secret)
	}
}

func requireAdoptedVerification(t *testing.T, evt event.Event) {
	t.Helper()
	got, err := unmarshalPayload[EmailVerifiedPayload](evt)
	if err != nil {
		t.Fatalf("decode %s: %v", evt.Type(), err)
	}
	if got.Email != mergeSurvivorEmail {
		t.Errorf("EmailVerified email = %q, want the survivor's %q", got.Email, mergeSurvivorEmail)
	}
}

// requireMergeRecord decodes the UserMerged event and checks which losers it
// names.
func requireMergeRecord(t *testing.T, evt event.Event, wantLosers ...UserID) {
	t.Helper()
	got, err := unmarshalPayload[identitymodel.UserMergedPayload](evt)
	if err != nil {
		t.Fatalf("decode %s: %v", evt.Type(), err)
	}
	if !slices.Equal(got.LoserIDs, wantLosers) || got.Reason != mergeReason {
		t.Errorf(
			"UserMerged names %v with reason %q, want %v with reason %q",
			got.LoserIDs,
			got.Reason,
			wantLosers,
			mergeReason,
		)
	}
}

func TestDecideMergeUsers_AdoptionReplaysThroughFold(t *testing.T) {
	survivorID := GenerateUserID()
	loserID := GenerateUserID()
	aggID := mergeStream(t, survivorID)

	events, err := decideMergeUsers(aggID, []identitymodel.MergeContribution{richContribution(loserID)}, mergeReason)(
		mergeSurvivor(), event.Version(4))
	if err != nil {
		t.Fatalf("decideMergeUsers: %v", err)
	}
	state := replayUser(t, mergeSurvivor(), events)

	for _, credentialID := range []string{"survivor-cred", "loser-cred-a", "loser-cred-b"} {
		if !state.HasCredential([]byte(credentialID)) {
			t.Errorf("survivor lacks credential %q after the merge", credentialID)
		}
	}
	if len(state.Credentials) != 3 {
		t.Errorf("credentials = %d, want 3", len(state.Credentials))
	}
	subjects := make([]string, len(state.ExternalAccounts))
	for i, account := range state.ExternalAccounts {
		subjects[i] = account.Provider + "/" + account.Subject
	}
	wantSubjects := []string{"github/survivor-subject", "github/loser-subject-1", "google/loser-subject-2"}
	if !slices.Equal(subjects, wantSubjects) {
		t.Errorf("external accounts = %v, want %v", subjects, wantSubjects)
	}
	if len(state.ExternalAccounts) != 3 {
		t.Errorf("external accounts = %d, want 3", len(state.ExternalAccounts))
	}
	if !state.TOTPEnabled || !bytes.Equal(state.TOTPSecret, []byte("loser-totp-secret")) {
		t.Errorf("TOTP not adopted: enabled=%v", state.TOTPEnabled)
	}
	if !state.EmailVerified {
		t.Error("email verification not adopted")
	}
	if !slices.Equal(state.MergedUserIDs, []UserID{loserID}) {
		t.Errorf("merged user IDs = %v, want [%s]", state.MergedUserIDs, loserID)
	}
	if state.Deleted || state.IsMerged() {
		t.Errorf("the survivor must stay live and unmerged: deleted=%v merged_into=%s",
			state.Deleted, state.MergedInto)
	}
}

func TestDecideMergeUsers_RetryWithSameContributionsIsNoop(t *testing.T) {
	survivorID := GenerateUserID()
	loserID := GenerateUserID()
	aggID := mergeStream(t, survivorID)
	contributions := []identitymodel.MergeContribution{richContribution(loserID)}
	decide := decideMergeUsers(aggID, contributions, mergeReason)

	first, err := decide(mergeSurvivor(), event.Version(4))
	if err != nil {
		t.Fatalf("first decision: %v", err)
	}
	state := replayUser(t, mergeSurvivor(), first)

	again, err := decide(state, 4+event.Version(len(first)))
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	if len(again) != 0 {
		t.Fatalf(
			"retry emitted %d events (%v); UserMerged must appear exactly once per merge",
			len(again),
			mergeEventTypes(again),
		)
	}
}

func TestDecideMergeUsers_LateCredentialAdoptedWithoutSecondUserMerged(t *testing.T) {
	survivorID := GenerateUserID()
	loserID := GenerateUserID()
	aggID := mergeStream(t, survivorID)
	decide := decideMergeUsers(aggID, []identitymodel.MergeContribution{richContribution(loserID)}, mergeReason)

	first, err := decide(mergeSurvivor(), event.Version(0))
	if err != nil {
		t.Fatalf("first decision: %v", err)
	}
	state := replayUser(t, mergeSurvivor(), first)
	before := len(state.Credentials)

	late := richContribution(loserID)
	late.Credentials = append(late.Credentials, mergeCredential("late-cred", "late-key", true))
	second, err := decideMergeUsers(aggID, []identitymodel.MergeContribution{late}, mergeReason)(
		state, event.Version(len(first)))
	if err != nil {
		t.Fatalf("second decision: %v", err)
	}
	if got := mergeEventTypes(second); !slices.Equal(got, []event.Type{eventCredentialAdded}) {
		t.Fatalf("second decision emitted %v, want exactly one CredentialAdded", got)
	}
	if second[0].Version() != event.Version(len(first)+1) {
		t.Errorf("version = %v, want %d", second[0].Version(), len(first)+1)
	}
	state = replayUser(t, state, second)
	if !state.HasCredential([]byte("late-cred")) || len(state.Credentials) != before+1 {
		t.Errorf("the late credential was not adopted: %d credentials, want %d", len(state.Credentials), before+1)
	}
	if !slices.Equal(state.MergedUserIDs, []UserID{loserID}) {
		t.Errorf("merged user IDs = %v, want the loser listed once", state.MergedUserIDs)
	}
}

func TestDecideMergeUsers_TwoLosersShareOneUserMerged(t *testing.T) {
	survivorID := GenerateUserID()
	firstLoser := GenerateUserID()
	secondLoser := GenerateUserID()
	aggID := mergeStream(t, survivorID)

	contributions := []identitymodel.MergeContribution{
		{
			UserID:      firstLoser,
			Credentials: []identitymodel.WebAuthnCredential{mergeCredential("cred-1", "key-1", false)},
		},
		{
			UserID:      secondLoser,
			Credentials: []identitymodel.WebAuthnCredential{mergeCredential("cred-2", "key-2", false)},
		},
		{
			UserID:      firstLoser,
			Credentials: []identitymodel.WebAuthnCredential{mergeCredential("cred-1", "key-1", false)},
		},
	}
	events, err := decideMergeUsers(aggID, contributions, mergeReason)(mergeSurvivor(), event.Version(0))
	if err != nil {
		t.Fatalf("decideMergeUsers: %v", err)
	}
	want := []event.Type{eventCredentialAdded, eventCredentialAdded, eventUserMerged}
	if got := mergeEventTypes(events); !slices.Equal(got, want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
	merged, err := unmarshalPayload[identitymodel.UserMergedPayload](events[2])
	if err != nil {
		t.Fatalf("decode UserMerged: %v", err)
	}
	wantLosers := []UserID{firstLoser, secondLoser}
	if !slices.Equal(merged.LoserIDs, wantLosers) {
		t.Errorf("loser IDs = %v, want %v in contribution order, each once", merged.LoserIDs, wantLosers)
	}
}

func TestDecideMergeUsers_KeepsSurvivorAssetsItAlreadyHolds(t *testing.T) {
	survivorID := GenerateUserID()
	loserID := GenerateUserID()
	aggID := mergeStream(t, survivorID)
	survivor := mergeSurvivor()
	survivor.TOTPEnabled = true
	survivor.TOTPSecret = []byte("survivor-totp-secret")
	survivor.EmailVerified = true

	events, err := decideMergeUsers(aggID, []identitymodel.MergeContribution{richContribution(loserID)}, mergeReason)(
		survivor, event.Version(0))
	if err != nil {
		t.Fatalf("decideMergeUsers: %v", err)
	}
	if slices.Contains(mergeEventTypes(events), eventTOTPEnabled) {
		t.Error("a survivor with TOTP must not have a second secret adopted")
	}
	if slices.Contains(mergeEventTypes(events), eventEmailVerified) {
		t.Error("a survivor with a verified email must not have it verified again")
	}
	state := replayUser(t, survivor, events)
	if !bytes.Equal(state.TOTPSecret, []byte("survivor-totp-secret")) {
		t.Error("the survivor's own TOTP secret was replaced")
	}
}

func TestDecideMergeUsers_FirstAvailableTOTPSecretWins(t *testing.T) {
	survivorID := GenerateUserID()
	aggID := mergeStream(t, survivorID)
	first := contributionFor(GenerateUserID())
	first.TOTPSecret = []byte("first-secret")
	second := contributionFor(GenerateUserID())
	second.TOTPSecret = []byte("second-secret")

	events, err := decideMergeUsers(aggID, []identitymodel.MergeContribution{first, second}, mergeReason)(
		mergeSurvivor(), event.Version(0))
	if err != nil {
		t.Fatalf("decideMergeUsers: %v", err)
	}
	if got := mergeEventTypes(events); !slices.Equal(got, []event.Type{eventTOTPEnabled, eventUserMerged}) {
		t.Fatalf("events = %v, want one TOTPEnabled then one UserMerged", got)
	}
	totp, err := unmarshalPayload[TOTPEnabledPayload](events[0])
	if err != nil {
		t.Fatalf("decode TOTPEnabled: %v", err)
	}
	if !bytes.Equal(totp.Secret, []byte("first-secret")) {
		t.Errorf("secret = %q, want the first contribution's", totp.Secret)
	}
}

func TestDecideMergeUsers_EmailVerificationNeedsTheSameEmail(t *testing.T) {
	survivorID := GenerateUserID()
	aggID := mergeStream(t, survivorID)

	tests := []struct {
		name  string
		email string
	}{
		{name: "another address", email: "other@larsartmann.cloud"},
		{name: "same address in another case", email: "LARS@LarsArtmann.Cloud"},
		{name: "same address with surrounding space", email: " lars@larsartmann.cloud "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contribution := contributionFor(GenerateUserID())
			contribution.Email = tt.email
			contribution.EmailVerified = true
			events, err := decideMergeUsers(aggID, []identitymodel.MergeContribution{contribution}, mergeReason)(
				mergeSurvivor(), event.Version(0))
			if err != nil {
				t.Fatalf("decideMergeUsers: %v", err)
			}
			if got, want := mergeEventTypes(events), []event.Type{eventUserMerged}; !slices.Equal(got, want) {
				t.Fatalf(
					"events = %v, want %v: verification of %q proves nothing about the survivor's email",
					got,
					want,
					tt.email,
				)
			}
		})
	}
}

func TestDecideMergeUsers_LeavesTheCallersContributionsUntouched(t *testing.T) {
	survivorID := GenerateUserID()
	loserID := GenerateUserID()
	aggID := mergeStream(t, survivorID)
	contributions := []identitymodel.MergeContribution{richContribution(loserID)}
	before := []identitymodel.MergeContribution{contributions[0].Clone()}

	if _, err := decideMergeUsers(aggID, contributions, mergeReason)(mergeSurvivor(), event.Version(0)); err != nil {
		t.Fatalf("decideMergeUsers: %v", err)
	}
	if !reflect.DeepEqual(contributions, before) {
		t.Error("the decider mutated its input")
	}
}

func TestUnmergedLosers(t *testing.T) {
	known := GenerateUserID()
	first := GenerateUserID()
	second := GenerateUserID()
	state := UserState{MergedUserIDs: []UserID{known}}

	got := unmergedLosers(state, []identitymodel.MergeContribution{
		contributionFor(first), contributionFor(known), contributionFor(second), contributionFor(first),
	})
	if want := []UserID{first, second}; !slices.Equal(got, want) {
		t.Errorf("unmergedLosers = %v, want %v", got, want)
	}
	if got := unmergedLosers(state, []identitymodel.MergeContribution{contributionFor(known)}); len(got) != 0 {
		t.Errorf("unmergedLosers = %v, want none", got)
	}
}

func TestDecideMergeUserInto_Guards(t *testing.T) {
	survivorID := GenerateUserID()
	loserID := GenerateUserID()
	aggID := mergeStream(t, loserID)
	otherSurvivor := GenerateUserID()
	alreadyMerged := existingState()
	alreadyMerged.MergedInto = otherSurvivor
	alreadyMerged.Deleted = true

	tests := []struct {
		name       string
		state      UserState
		survivorID UserID
		wantFamily errorfamily.Family
		wantCode   string
	}{
		{
			name:       "unregistered user",
			state:      UserState{},
			survivorID: survivorID,
			wantFamily: errorfamily.Rejection,
			wantCode:   "usermgmt.merge_user_into.not_found",
		},
		{
			name:       "no survivor",
			state:      existingState(),
			survivorID: UserID{},
			wantFamily: errorfamily.Rejection,
			wantCode:   "usermgmt.merge_user_into.survivor_required",
		},
		{
			name:       "merged into itself",
			state:      existingState(),
			survivorID: loserID,
			wantFamily: errorfamily.Rejection,
			wantCode:   "usermgmt.merge_user_into.self_merge",
		},
		{
			name:       "already merged into another survivor",
			state:      alreadyMerged,
			survivorID: survivorID,
			wantFamily: errorfamily.Conflict,
			wantCode:   "usermgmt.merge_user_into.already_merged",
		},
		{
			name:       "plain deleted user",
			state:      deletedState(),
			survivorID: survivorID,
			wantFamily: errorfamily.Conflict,
			wantCode:   "usermgmt.merge_user_into.deleted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events, err := decideMergeUserInto(aggID, tt.survivorID, mergeReason)(tt.state, event.Version(3))
			ef := requireDomainError(t, err, tt.wantFamily, tt.wantCode)
			if len(events) != 0 {
				t.Fatalf("a refused decision must emit nothing, got %d events", len(events))
			}
			if tt.wantCode == "usermgmt.merge_user_into.already_merged" &&
				ef.ContextValue("merged_into") != otherSurvivor.Get().String() {
				t.Errorf("merged_into = %q, want the first survivor %q",
					ef.ContextValue("merged_into"), otherSurvivor.Get().String())
			}
		})
	}
}

func TestDecideMergeUserInto_TombstonesTheLoserOnce(t *testing.T) {
	survivorID := GenerateUserID()
	loserID := GenerateUserID()
	aggID := mergeStream(t, loserID)

	events, err := decideMergeUserInto(aggID, survivorID, mergeReason)(existingState(), event.Version(3))
	if err != nil {
		t.Fatalf("decideMergeUserInto: %v", err)
	}
	if got, want := mergeEventTypes(events), []event.Type{eventUserMergedInto}; !slices.Equal(got, want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
	if events[0].Version() != event.Version(4) {
		t.Errorf("version = %v, want 4", events[0].Version())
	}
	if events[0].Encoding() != codec.EncodingJSON {
		t.Errorf("encoding = %v, want JSON", events[0].Encoding())
	}
	if events[0].StreamID().Get() != aggID.Get() {
		t.Errorf("event lands on stream %q, want the loser stream %q", events[0].StreamID().Get(), aggID.Get())
	}
	payload, err := unmarshalPayload[identitymodel.UserMergedIntoPayload](events[0])
	if err != nil {
		t.Fatalf("decode UserMergedInto: %v", err)
	}
	if payload.SurvivorID != survivorID || payload.Reason != mergeReason {
		t.Errorf("payload = %+v, want survivor %s and reason %q", payload, survivorID, mergeReason)
	}

	state, err := foldUser(existingState(), events[0])
	if err != nil {
		t.Fatalf("foldUser: %v", err)
	}
	if !state.Deleted || !state.IsMerged() || state.MergedInto != survivorID {
		t.Errorf("folded loser state = %+v, want deleted and pointing at the survivor", state)
	}
	if state.DeleteReason != mergeReason {
		t.Errorf("delete reason = %q, want %q", state.DeleteReason, mergeReason)
	}

	again, err := decideMergeUserInto(aggID, survivorID, mergeReason)(state, event.Version(4))
	if err != nil {
		t.Fatalf("repeated MergeUserInto for the same survivor: %v", err)
	}
	if len(again) != 0 {
		t.Fatalf("the repeat emitted %d events; a merged user is terminal", len(again))
	}
}

func TestDecideTransferBot_Guards(t *testing.T) {
	aggID := id.NewStreamID()
	ownerID := GenerateUserID()
	deletedBot := BotState{Name: "ci-bot", OwnerID: ownerID, Deleted: true}

	tests := []struct {
		name       string
		state      BotState
		ownerID    UserID
		wantFamily errorfamily.Family
		wantCode   string
	}{
		{
			name:       "unregistered bot",
			state:      BotState{},
			ownerID:    ownerID,
			wantFamily: errorfamily.Rejection,
			wantCode:   "usermgmt.bot_transfer.not_found",
		},
		{
			name:       "deleted bot",
			state:      deletedBot,
			ownerID:    GenerateUserID(),
			wantFamily: errorfamily.Rejection,
			wantCode:   "usermgmt.bot_transfer.not_found",
		},
		{
			name:       "no new owner",
			state:      BotState{Name: "ci-bot", OwnerID: ownerID},
			ownerID:    UserID{},
			wantFamily: errorfamily.Rejection,
			wantCode:   "usermgmt.bot_transfer.owner_required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events, err := decideTransferBot(aggID, tt.ownerID, mergeReason)(tt.state, event.Version(2))
			_ = requireDomainError(t, err, tt.wantFamily, tt.wantCode)
			if len(events) != 0 {
				t.Fatalf("a refused decision must emit nothing, got %d events", len(events))
			}
		})
	}
}

func TestDecideTransferBot_SameOwnerIsNoop(t *testing.T) {
	ownerID := GenerateUserID()
	state := BotState{Name: "ci-bot", OwnerID: ownerID}

	events, err := decideTransferBot(id.NewStreamID(), ownerID, mergeReason)(state, event.Version(2))
	if err != nil {
		t.Fatalf("decideTransferBot: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("transfer to the current owner emitted %d events", len(events))
	}
}

func TestDecideTransferBot_MovesOnlyTheOwner(t *testing.T) {
	aggID := id.NewStreamID()
	oldOwner := GenerateUserID()
	newOwner := GenerateUserID()
	state := BotState{
		Name:      "ci-bot",
		OwnerID:   oldOwner,
		TokenHash: []byte("token-hash"),
		Scopes:    []string{"read"},
	}

	events, err := decideTransferBot(aggID, newOwner, mergeReason)(state, event.Version(5))
	if err != nil {
		t.Fatalf("decideTransferBot: %v", err)
	}
	if got, want := mergeEventTypes(events), []event.Type{eventBotTransferred}; !slices.Equal(got, want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
	if events[0].Version() != event.Version(6) || events[0].Encoding() != codec.EncodingJSON {
		t.Errorf("version/encoding = %v/%v, want 6/JSON", events[0].Version(), events[0].Encoding())
	}
	payload, err := unmarshalPayload[identitymodel.BotTransferredPayload](events[0])
	if err != nil {
		t.Fatalf("decode BotTransferred: %v", err)
	}
	if payload.OwnerID != newOwner || payload.Reason != mergeReason {
		t.Errorf("payload = %+v, want owner %s and reason %q", payload, newOwner, mergeReason)
	}

	moved := replayBot(t, state, events)
	if moved.OwnerID != newOwner {
		t.Errorf("owner = %s, want %s", moved.OwnerID, newOwner)
	}
	if moved.Name != "ci-bot" || !bytes.Equal(moved.TokenHash, []byte("token-hash")) ||
		!slices.Equal(moved.Scopes, []string{"read"}) || moved.Deleted {
		t.Errorf("the transfer changed more than the owner: %+v", moved)
	}
	if state.OwnerID != oldOwner {
		t.Error("the decider mutated its input state")
	}
}
