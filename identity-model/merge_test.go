package identitymodel

import (
	"encoding/json/v2"
	"reflect"
	"slices"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// Pins the identity-merge vocabulary (cqrs-htmx#46, #47): the fold contract
// of the six merge events, the liveness of merged-away identities, and the
// immutability of the commands that drive them.

func mergeTestEvent(
	t *testing.T,
	eventType event.Type,
	streamType event.StreamType,
	payload any,
) event.Event {
	t.Helper()

	raw, err := MarshalPayload(payload)
	if err != nil {
		t.Fatalf("marshal %s payload: %v", eventType, err)
	}

	evt, err := event.NewEvent(
		eventType, id.DeriveStreamID("merge-test", string(eventType)), streamType, event.Version(1), raw,
	)
	if err != nil {
		t.Fatalf("build %s event: %v", eventType, err)
	}

	return evt
}

func assertCommandIdentity(t *testing.T, cmd command.Command, wantType command.Type, stream id.StreamID) {
	t.Helper()

	if cmd.Type() != wantType {
		t.Errorf("Type() = %s, want %s", cmd.Type(), wantType)
	}
	if cmd.StreamID() != stream {
		t.Errorf("StreamID() = %s, want %s", cmd.StreamID(), stream)
	}
}

func TestFoldUser_MergedRecordsEachLoserOnce(t *testing.T) {
	loserA, loserB := SyntheticUserID("loser-a"), SyntheticUserID("loser-b")
	merged := mergeTestEvent(t, EventUserMerged, AggregateTypeUser, UserMergedPayload{
		SchemaVersion: CurrentSchemaVersion,
		LoserIDs:      []UserID{loserA, loserB},
		Reason:        "duplicate",
	})

	next, err := FoldUser(UserState{Email: "survivor@example.com"}, merged)
	if err != nil {
		t.Fatalf("FoldUser: %v", err)
	}
	if want := []UserID{loserA, loserB}; !slices.Equal(next.MergedUserIDs, want) {
		t.Fatalf("MergedUserIDs = %v, want %v", next.MergedUserIDs, want)
	}

	replayed, err := FoldUser(next, merged)
	if err != nil {
		t.Fatalf("FoldUser replay: %v", err)
	}
	if want := []UserID{loserA, loserB}; !slices.Equal(replayed.MergedUserIDs, want) {
		t.Fatalf("at-least-once replay grew the list: %v", replayed.MergedUserIDs)
	}
	if next.Email != "survivor@example.com" || next.Deleted {
		t.Fatalf("recording a merge must not touch the survivor's own liveness: %+v", next)
	}
}

func TestFoldUser_MergedAppendNeverWritesThroughToTheCaller(t *testing.T) {
	loserA, loserB := SyntheticUserID("loser-a"), SyntheticUserID("loser-b")
	before := UserState{Email: "survivor@example.com", MergedUserIDs: make([]UserID, 1, 4)}
	before.MergedUserIDs[0] = loserA

	after, err := FoldUser(before, mergeTestEvent(t, EventUserMerged, AggregateTypeUser, UserMergedPayload{
		SchemaVersion: CurrentSchemaVersion,
		LoserIDs:      []UserID{loserB},
	}))
	if err != nil {
		t.Fatalf("FoldUser: %v", err)
	}

	if want := []UserID{loserA}; !slices.Equal(before.MergedUserIDs, want) {
		t.Fatalf("caller's state changed: %v, want %v", before.MergedUserIDs, want)
	}
	if want := []UserID{loserA, loserB}; !slices.Equal(after.MergedUserIDs, want) {
		t.Fatalf("MergedUserIDs = %v, want %v", after.MergedUserIDs, want)
	}
	if &after.MergedUserIDs[0] == &before.MergedUserIDs[0] {
		t.Fatal("the fold appended into the caller's backing array")
	}
}

func TestFoldUser_MergedIntoIsATerminalTombstone(t *testing.T) {
	survivor := SyntheticUserID("survivor")
	into := mergeTestEvent(t, EventUserMergedInto, AggregateTypeUser, UserMergedIntoPayload{
		SchemaVersion: CurrentSchemaVersion,
		SurvivorID:    survivor,
		Reason:        "duplicate",
	})

	next, err := FoldUser(UserState{Email: "loser@example.com", TOTPEnabled: true}, into)
	if err != nil {
		t.Fatalf("FoldUser: %v", err)
	}

	if !next.Deleted || !next.IsMerged() || next.MergedInto != survivor {
		t.Fatalf("merged-away state = %+v, want Deleted with the survivor pointer", next)
	}
	if next.DeleteReason != "duplicate" {
		t.Errorf("DeleteReason = %q, want the merge reason", next.DeleteReason)
	}
	if !next.Exists() || next.IsActive() {
		t.Errorf("a merged-away user must keep its history and lose liveness: Exists=%v IsActive=%v",
			next.Exists(), next.IsActive())
	}
}

func TestUserState_HasCredential(t *testing.T) {
	state := UserState{Credentials: []WebAuthnCredential{
		{ID: []byte{1, 2}},
	}}

	if !state.HasCredential([]byte{1, 2}) {
		t.Error("HasCredential = false for a credential the user holds")
	}
	if state.HasCredential([]byte{2, 1}) || state.HasCredential(nil) || (UserState{}).HasCredential(nil) {
		t.Error("HasCredential = true for a credential the user does not hold")
	}
}

func TestFoldTenant_RenamedMovesOnlyTheLabel(t *testing.T) {
	renamed := mergeTestEvent(t, EventTenantRenamed, AggregateTypeTenant, TenantRenamedPayload{
		SchemaVersion: CurrentSchemaVersion,
		DisplayName:   "ACME Corp",
	})

	next, err := FoldTenant(TenantState{Name: "acme", DisplayName: "Acme"}, renamed)
	if err != nil {
		t.Fatalf("FoldTenant: %v", err)
	}

	if next.Name != "acme" {
		t.Errorf("a rename must not move the tenant ID: Name = %q", next.Name)
	}
	if next.DisplayName != "ACME Corp" {
		t.Errorf("DisplayName = %q, want %q", next.DisplayName, "ACME Corp")
	}
}

func TestFoldTenant_MergedAndMergedInto(t *testing.T) {
	loserA, loserB := NewTenantID("acne"), NewTenantID("akme")
	survivor := NewTenantID("acme")

	surviving, err := FoldTenant(TenantState{Name: "acme"}, mergeTestEvent(
		t, EventTenantMerged, AggregateTypeTenant, TenantMergedPayload{
			SchemaVersion: CurrentSchemaVersion,
			LoserIDs:      []TenantID{loserA, loserB},
			Reason:        "duplicate",
		},
	))
	if err != nil {
		t.Fatalf("FoldTenant TenantMerged: %v", err)
	}
	if want := []TenantID{loserA, loserB}; !slices.Equal(surviving.MergedTenantIDs, want) {
		t.Fatalf("MergedTenantIDs = %v, want %v", surviving.MergedTenantIDs, want)
	}

	tombstone, err := FoldTenant(TenantState{Name: "acne", Suspended: true, SuspendReason: "abuse"}, mergeTestEvent(
		t, EventTenantMergedInto, AggregateTypeTenant, TenantMergedIntoPayload{
			SchemaVersion: CurrentSchemaVersion,
			SurvivorID:    survivor,
			Reason:        "duplicate",
		},
	))
	if err != nil {
		t.Fatalf("FoldTenant TenantMergedInto: %v", err)
	}

	if !tombstone.Deleted || !tombstone.IsMerged() || tombstone.MergedInto != survivor {
		t.Fatalf("merged-away tenant = %+v, want Deleted with the survivor pointer", tombstone)
	}
	if tombstone.Suspended || tombstone.SuspendReason != "" || tombstone.DeleteReason != "duplicate" {
		t.Errorf("merged-away tenant kept stale lifecycle state: %+v", tombstone)
	}
	if tombstone.Exists() || tombstone.IsActive() {
		t.Error("a merged-away tenant must not be live")
	}
	if !tombstone.IsValid() {
		t.Error("a merged-away tombstone must satisfy IsValid")
	}
}

func TestFoldTenant_CreatedStartsACleanIncarnation(t *testing.T) {
	deleted, err := FoldTenant(TenantState{Name: "acme", DisplayName: "Acme", Suspended: true, SuspendReason: "old"},
		mergeTestEvent(t, EventTenantDeleted, AggregateTypeTenant, TenantDeletedPayload{
			SchemaVersion: CurrentSchemaVersion,
			Reason:        "typo",
		}))
	if err != nil {
		t.Fatalf("FoldTenant TenantDeleted: %v", err)
	}

	reborn, err := FoldTenant(deleted, mergeTestEvent(t, EventTenantCreated, AggregateTypeTenant, TenantCreatedPayload{
		SchemaVersion: CurrentSchemaVersion,
		Name:          "acme",
		DisplayName:   "Acme 2",
	}))
	if err != nil {
		t.Fatalf("FoldTenant TenantCreated on a tombstone: %v", err)
	}

	want := TenantState{Name: "acme", DisplayName: "Acme 2"}
	if !reflect.DeepEqual(reborn, want) {
		t.Fatalf("recreated tenant = %+v, want a state that inherits nothing: %+v", reborn, want)
	}
}

func TestTenantState_IsValidRequiresDeletedWhenMerged(t *testing.T) {
	cases := []struct {
		name  string
		state TenantState
		valid bool
	}{
		{"merged and deleted", TenantState{Name: "a", Deleted: true, MergedInto: NewTenantID("s")}, true},
		{"merged but still live", TenantState{Name: "a", MergedInto: NewTenantID("s")}, false},
		{"plain tombstone", TenantState{Name: "a", Deleted: true, DeleteReason: "cleanup"}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.state.IsValid() != tc.valid {
				t.Errorf("IsValid() = %v, want %v", tc.state.IsValid(), tc.valid)
			}
		})
	}
}

func TestFoldBot_TransferredMovesOnlyTheOwner(t *testing.T) {
	before := BotState{Name: "bot", OwnerID: SyntheticUserID("old"), TokenHash: []byte{1, 2}, Scopes: []string{"read"}}
	newOwner := SyntheticUserID("new")

	next, err := FoldBot(before, mergeTestEvent(t, EventBotTransferred, AggregateTypeBot, BotTransferredPayload{
		SchemaVersion: CurrentSchemaVersion,
		OwnerID:       newOwner,
		Reason:        "owner merged",
	}))
	if err != nil {
		t.Fatalf("FoldBot: %v", err)
	}

	if next.OwnerID != newOwner {
		t.Errorf("OwnerID = %s, want the new owner", next.OwnerID.Get())
	}
	if next.Name != before.Name || !slices.Equal(next.TokenHash, before.TokenHash) ||
		!slices.Equal(next.Scopes, before.Scopes) || next.Deleted {
		t.Errorf("a transfer must keep everything but the owner: %+v", next)
	}
}

func TestMergePayloads_RoundTripJSON(t *testing.T) {
	user := SyntheticUserID("user-1")
	tenantA, tenantB := NewTenantID("acme"), NewTenantID("acne")

	cases := []struct {
		name string
		in   any
		out  any
	}{
		{"UserMerged",
			&UserMergedPayload{SchemaVersion: CurrentSchemaVersion, LoserIDs: []UserID{user}, Reason: "duplicate"},
			&UserMergedPayload{}},
		{"UserMergedInto",
			&UserMergedIntoPayload{SchemaVersion: CurrentSchemaVersion, SurvivorID: user, Reason: "duplicate"},
			&UserMergedIntoPayload{}},
		{"BotTransferred",
			&BotTransferredPayload{SchemaVersion: CurrentSchemaVersion, OwnerID: user, Reason: "owner merged"},
			&BotTransferredPayload{}},
		{"TenantRenamed",
			&TenantRenamedPayload{SchemaVersion: CurrentSchemaVersion, DisplayName: "ACME Corp"},
			&TenantRenamedPayload{}},
		{"TenantMerged",
			&TenantMergedPayload{SchemaVersion: CurrentSchemaVersion, LoserIDs: []TenantID{tenantA, tenantB}, Reason: "duplicate"},
			&TenantMergedPayload{}},
		{"TenantMergedInto",
			&TenantMergedIntoPayload{SchemaVersion: CurrentSchemaVersion, SurvivorID: tenantA, Reason: "duplicate"},
			&TenantMergedIntoPayload{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := MarshalPayload(tc.in)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if err := json.Unmarshal(raw, tc.out); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if !reflect.DeepEqual(tc.in, tc.out) {
				t.Fatalf("round trip changed the payload: in=%+v out=%+v", tc.in, tc.out)
			}
		})
	}
}

func TestMergeCommands_AreTypedAndImmutable(t *testing.T) {
	stream := id.DeriveStreamID("user", "survivor")
	loser, survivor, owner := SyntheticUserID("loser"), SyntheticUserID("survivor"), SyntheticUserID("owner")
	tenantA, tenantB := NewTenantID("acme"), NewTenantID("acne")

	merge := NewMergeUsersCmd(stream, []MergeContribution{{
		UserID:           loser,
		Email:            "loser@example.com",
		EmailVerified:    true,
		TOTPSecret:       []byte("totp-secret"),
		Credentials:      []WebAuthnCredential{{ID: []byte{1}, PublicKey: []byte{2}}},
		ExternalAccounts: []ExternalAccount{{Provider: "google", Subject: "s"}},
	}}, "duplicate")
	assertCommandIdentity(t, merge, CmdMergeUsers, stream)
	if merge.Reason() != "duplicate" {
		t.Errorf("Reason() = %q, want the merge reason", merge.Reason())
	}

	exposed := merge.Contributions()
	exposed[0].UserID = SyntheticUserID("hijacked")
	exposed[0].TOTPSecret[0] = 'x'
	exposed[0].Credentials[0].ID[0] = 9
	exposed[0].Credentials[0].PublicKey[0] = 9
	exposed[0].ExternalAccounts[0].Provider = "evil"

	inner := merge.Contributions()[0]
	switch {
	case inner.UserID != loser, inner.TOTPSecret[0] != 't', inner.Credentials[0].ID[0] != 1,
		inner.Credentials[0].PublicKey[0] != 2, inner.ExternalAccounts[0].Provider != "google":
		t.Fatalf("mutating an accessor result wrote through to the command: %+v", inner)
	}

	into := NewMergeUserIntoCmd(stream, survivor, "duplicate")
	assertCommandIdentity(t, into, CmdMergeUserInto, stream)
	if into.SurvivorID() != survivor || into.Reason() != "duplicate" {
		t.Errorf("MergeUserInto accessors = %s / %q", into.SurvivorID().Get(), into.Reason())
	}

	transfer := NewTransferBotCmd(stream, owner, "owner merged")
	assertCommandIdentity(t, transfer, CmdTransferBot, stream)
	if transfer.OwnerID() != owner || transfer.Reason() != "owner merged" {
		t.Errorf("TransferBot accessors = %s / %q", transfer.OwnerID().Get(), transfer.Reason())
	}

	rename := NewRenameTenantCmd(stream, "ACME Corp")
	assertCommandIdentity(t, rename, CmdRenameTenant, stream)
	if rename.DisplayName() != "ACME Corp" {
		t.Errorf("DisplayName() = %q", rename.DisplayName())
	}

	tenants := NewMergeTenantsCmd(stream, []TenantID{tenantA, tenantB}, "duplicate")
	assertCommandIdentity(t, tenants, CmdMergeTenants, stream)
	losers := tenants.LoserIDs()
	losers[0] = NewTenantID("hijacked")
	if got := tenants.LoserIDs(); !slices.Equal(got, []TenantID{tenantA, tenantB}) {
		t.Fatalf("mutating LoserIDs() wrote through to the command: %v", got)
	}

	tenantInto := NewMergeTenantIntoCmd(stream, tenantA, "duplicate")
	assertCommandIdentity(t, tenantInto, CmdMergeTenantInto, stream)
	if tenantInto.SurvivorID() != tenantA {
		t.Errorf("SurvivorID() = %s, want %s", tenantInto.SurvivorID().Get(), tenantA.Get())
	}
}
