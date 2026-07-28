package usermgmt

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// TestAllCommandsHaveNonZeroID is a regression test for a bug where 7 of 21
// command constructors returned a zero-value cmdID, silently breaking
// idempotency dedup and Watermill message UUIDs (which derive from cmd.ID()).
//
// Every command constructor must produce a non-zero ID(). This table-driven
// test iterates all 20 command constructors and asserts each returns a
// non-zero command ID.
func TestAllCommandsHaveNonZeroID(t *testing.T) {
	t.Parallel()

	aggID := id.NewStreamID()
	userID := NewUserID("01JXTESTUSERID00000000")
	actorID := ActorIDFromUser(userID)
	tenantID := NewTenantID("01JXTESTTENANTID00000000")

	tests := []struct {
		name string
		cmd  command.Command
	}{
		// User commands (11)
		{"RegisterUser", NewRegisterUserCmd(aggID, "user@test.com", "Test", []Role{"admin"})},
		{"ChangeEmail", NewChangeEmailCmd(aggID, "new@test.com")},
		{"ChangeDisplayName", NewChangeDisplayNameCmd(aggID, "New Name")},
		{"DeleteUser", NewDeleteUserCmd(aggID, "test")},
		{"AddCredential", NewAddCredentialCmd(aggID, WebAuthnCredential{})},
		{"RemoveCredential", NewRemoveCredentialCmd(aggID, []byte("cred1"))},
		{"VerifyEmail", NewVerifyEmailCmd(aggID)},
		{"EnableTOTP", NewEnableTOTPCmd(aggID, []byte("secret"))},
		{"DisableTOTP", NewDisableTOTPCmd(aggID)},
		{"LinkExternalAccount", NewLinkExternalAccountCmd(aggID, "google", "sub123", "user@test.com", "Test")},
		{"UnlinkExternalAccount", NewUnlinkExternalAccountCmd(aggID, "google", "sub123")},

		// Membership commands (3)
		{"AddMember", NewAddMemberCmd(actorID, tenantID, []Role{"member"})},
		{"UpdateMemberRoles", NewUpdateMemberRolesCmd(actorID, tenantID, []Role{"admin"})},
		{"RemoveMember", NewRemoveMemberCmd(actorID, tenantID)},

		// Tenant commands (4)
		{"CreateTenant", NewCreateTenantCmd(aggID, "acme", "Acme Corp")},
		{"SuspendTenant", NewSuspendTenantCmd(aggID, "policy violation")},
		{"ReactivateTenant", NewReactivateTenantCmd(aggID)},
		{"DeleteTenant", NewDeleteTenantCmd(aggID, "closing")},

		// Bot commands (2)
		{"RegisterBot", NewRegisterBotCmd(aggID, "CI Bot", userID, []byte("hash"), []string{"read"})},
		{"DeleteBot", NewDeleteBotCmd(aggID, "decommissioned")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.cmd.ID().IsZero() {
				t.Errorf(
					"%s constructor returned zero command ID — idempotency and Watermill dedup will silently break",
					tt.name,
				)
			}
		})
	}
}

// TestCommandIDsAreUnique verifies that two instances of the same command type
// produce different command IDs (auto-minted ULIDs must be unique).
func TestCommandIDsAreUnique(t *testing.T) {
	t.Parallel()

	aggID := id.NewStreamID()
	cmd1 := NewChangeEmailCmd(aggID, "a@test.com")
	cmd2 := NewChangeEmailCmd(aggID, "b@test.com")

	if cmd1.ID() == cmd2.ID() {
		t.Error("two command instances should have different IDs for dedup")
	}
}

// TestAllCommandsProduceDifferentIDs constructs every command constructor twice
// (two batches) and verifies all 40 IDs are mutually unique. This catches a
// regression where ULID minting could collide or where a constructor
// accidentally hardcodes a fixed ID.
func TestAllCommandsProduceDifferentIDs(t *testing.T) {
	t.Parallel()

	aggID := id.NewStreamID()
	userID := NewUserID("01JXTESTUSERID00000000")
	actorID := ActorIDFromUser(userID)
	tenantID := NewTenantID("01JXTESTTENANTID00000000")

	// buildAll returns one instance of every command constructor.
	buildAll := func() []command.Command {
		return []command.Command{
			NewRegisterUserCmd(aggID, "u@t.com", "T", []Role{"admin"}),
			NewChangeEmailCmd(aggID, "n@t.com"),
			NewChangeDisplayNameCmd(aggID, "N"),
			NewDeleteUserCmd(aggID, "r"),
			NewAddCredentialCmd(aggID, WebAuthnCredential{}),
			NewRemoveCredentialCmd(aggID, []byte("c")),
			NewVerifyEmailCmd(aggID),
			NewEnableTOTPCmd(aggID, []byte("s")),
			NewDisableTOTPCmd(aggID),
			NewLinkExternalAccountCmd(aggID, "g", "s", "u@t.com", "T"),
			NewUnlinkExternalAccountCmd(aggID, "g", "s"),
			NewAddMemberCmd(actorID, tenantID, []Role{"m"}),
			NewUpdateMemberRolesCmd(actorID, tenantID, []Role{"a"}),
			NewRemoveMemberCmd(actorID, tenantID),
			NewCreateTenantCmd(aggID, "acme", "Acme"),
			NewSuspendTenantCmd(aggID, "violation"),
			NewReactivateTenantCmd(aggID),
			NewDeleteTenantCmd(aggID, "closing"),
			NewRegisterBotCmd(aggID, "Bot", userID, []byte("h"), []string{"r"}),
			NewDeleteBotCmd(aggID, "decommissioned"),
		}
	}

	batch1 := buildAll()
	batch2 := buildAll()

	seen := make(map[id.CommandID]struct{}, len(batch1)+len(batch2))
	for _, cmd := range append(batch1, batch2...) {
		cid := cmd.ID()
		if cid.IsZero() {
			t.Fatal("found zero command ID — constructor bug")
		}
		if _, dup := seen[cid]; dup {
			t.Fatalf("duplicate command ID %s — ULIDs must be unique", cid)
		}
		seen[cid] = struct{}{}
	}

	if len(seen) != len(batch1)+len(batch2) {
		t.Fatalf("expected %d unique IDs, got %d", len(batch1)+len(batch2), len(seen))
	}
}

// TestMustCommand_PanicsOnZeroAggregateID verifies that mustCommand panics
// when given a zero-value aggregate ID. This is a programming bug — the
// constructor should never be called without a valid aggregate ID — so a
// panic is the correct behavior (fail-fast, not silent zero cmdID).
func TestMustCommand_PanicsOnZeroAggregateID(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("mustCommand should panic when aggregate ID is zero")
		}
	}()

	_ = mustCommand(cmdRegisterUser, id.StreamID{})
}

// TestMustCommand_PanicsOnEmptyCommandType verifies that mustCommand panics
// when given an empty command type. Like the zero aggregate ID, this is a
// programming bug that should fail immediately, not produce a zero cmdID.
func TestMustCommand_PanicsOnEmptyCommandType(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("mustCommand should panic when command type is empty")
		}
	}()

	_ = mustCommand("", id.NewStreamID())
}
