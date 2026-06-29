package usermgmt

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
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

	aggID := id.NewAggregateID()
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

	aggID := id.NewAggregateID()
	cmd1 := NewChangeEmailCmd(aggID, "a@test.com")
	cmd2 := NewChangeEmailCmd(aggID, "b@test.com")

	if cmd1.ID() == cmd2.ID() {
		t.Error("two command instances should have different IDs for dedup")
	}
}
