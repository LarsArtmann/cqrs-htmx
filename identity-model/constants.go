package identitymodel

import (
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

const (
	AggregateTypeUser       event.StreamType = "User"
	AggregateTypeMembership event.StreamType = "Membership"
	AggregateTypeTenant     event.StreamType = "Tenant"
	AggregateTypeBot        event.StreamType = "Bot"

	EventUserRegistered          event.Type = "UserRegistered"
	EventRolesUpdated            event.Type = "RolesUpdated" // legacy: no longer emitted, decoded for backward compat
	EventEmailChanged            event.Type = "EmailChanged"
	EventDisplayNameChanged      event.Type = "DisplayNameChanged"
	EventUserDeleted             event.Type = "UserDeleted"
	EventCredentialAdded         event.Type = "CredentialAdded"   //nolint:gosec // event type name, not credential
	EventCredentialRemoved       event.Type = "CredentialRemoved" //nolint:gosec // event type name, not credential
	EventEmailVerified           event.Type = "EmailVerified"
	EventTOTPEnabled             event.Type = "TOTPEnabled"
	EventTOTPDisabled            event.Type = "TOTPDisabled"
	EventExternalAccountLinked   event.Type = "ExternalAccountLinked"
	EventExternalAccountUnlinked event.Type = "ExternalAccountUnlinked"

	EventMemberAdded        event.Type = "MemberAdded"
	EventMemberRolesChanged event.Type = "MemberRolesChanged"
	EventMemberRemoved      event.Type = "MemberRemoved"

	EventTenantCreated     event.Type = "TenantCreated"
	EventTenantSuspended   event.Type = "TenantSuspended"
	EventTenantReactivated event.Type = "TenantReactivated"
	EventTenantDeleted     event.Type = "TenantDeleted"

	EventBotRegistered event.Type = "BotRegistered"
	EventBotDeleted    event.Type = "BotDeleted"

	CmdRegisterUser          command.Type = "RegisterUser"
	CmdChangeEmail           command.Type = "ChangeEmail"
	CmdChangeDisplayName     command.Type = "ChangeDisplayName"
	CmdDeleteUser            command.Type = "DeleteUser"
	CmdAddCredential         command.Type = "AddCredential"    //nolint:gosec // command type name, not credential
	CmdRemoveCredential      command.Type = "RemoveCredential" //nolint:gosec // command type name, not credential
	CmdVerifyEmail           command.Type = "VerifyEmail"
	CmdEnableTOTP            command.Type = "EnableTOTP"
	CmdDisableTOTP           command.Type = "DisableTOTP"
	CmdLinkExternalAccount   command.Type = "LinkExternalAccount"
	CmdUnlinkExternalAccount command.Type = "UnlinkExternalAccount"

	CmdAddMember         command.Type = "AddMember"
	CmdUpdateMemberRoles command.Type = "UpdateMemberRoles"
	CmdRemoveMember      command.Type = "RemoveMember"

	CmdCreateTenant     command.Type = "CreateTenant"
	CmdSuspendTenant    command.Type = "SuspendTenant"
	CmdReactivateTenant command.Type = "ReactivateTenant"
	CmdDeleteTenant     command.Type = "DeleteTenant"

	CmdRegisterBot command.Type = "RegisterBot"
	CmdDeleteBot   command.Type = "DeleteBot"
)

const CurrentSchemaVersion = 2
