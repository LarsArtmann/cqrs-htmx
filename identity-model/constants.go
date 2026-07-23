package identitymodel

import (
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

const (
	aggregateTypeUser       event.StreamType = "User"
	aggregateTypeMembership event.StreamType = "Membership"
	aggregateTypeTenant     event.StreamType = "Tenant"
	aggregateTypeBot        event.StreamType = "Bot"

	eventUserRegistered          event.Type = "UserRegistered"
	eventRolesUpdated            event.Type = "RolesUpdated" // legacy: no longer emitted, decoded for backward compat
	eventEmailChanged            event.Type = "EmailChanged"
	eventDisplayNameChanged      event.Type = "DisplayNameChanged"
	eventUserDeleted             event.Type = "UserDeleted"
	eventCredentialAdded         event.Type = "CredentialAdded"   //nolint:gosec // event type name, not credential
	eventCredentialRemoved       event.Type = "CredentialRemoved" //nolint:gosec // event type name, not credential
	eventEmailVerified           event.Type = "EmailVerified"
	eventTOTPEnabled             event.Type = "TOTPEnabled"
	eventTOTPDisabled            event.Type = "TOTPDisabled"
	eventExternalAccountLinked   event.Type = "ExternalAccountLinked"
	eventExternalAccountUnlinked event.Type = "ExternalAccountUnlinked"

	eventMemberAdded        event.Type = "MemberAdded"
	eventMemberRolesChanged event.Type = "MemberRolesChanged"
	eventMemberRemoved      event.Type = "MemberRemoved"

	eventTenantCreated     event.Type = "TenantCreated"
	eventTenantSuspended   event.Type = "TenantSuspended"
	eventTenantReactivated event.Type = "TenantReactivated"
	eventTenantDeleted     event.Type = "TenantDeleted"

	eventBotRegistered event.Type = "BotRegistered"
	eventBotDeleted    event.Type = "BotDeleted"

	cmdRegisterUser          command.Type = "RegisterUser"
	cmdChangeEmail           command.Type = "ChangeEmail"
	cmdChangeDisplayName     command.Type = "ChangeDisplayName"
	cmdDeleteUser            command.Type = "DeleteUser"
	cmdAddCredential         command.Type = "AddCredential"    //nolint:gosec // command type name, not credential
	cmdRemoveCredential      command.Type = "RemoveCredential" //nolint:gosec // command type name, not credential
	cmdVerifyEmail           command.Type = "VerifyEmail"
	cmdEnableTOTP            command.Type = "EnableTOTP"
	cmdDisableTOTP           command.Type = "DisableTOTP"
	cmdLinkExternalAccount   command.Type = "LinkExternalAccount"
	cmdUnlinkExternalAccount command.Type = "UnlinkExternalAccount"

	cmdAddMember         command.Type = "AddMember"
	cmdUpdateMemberRoles command.Type = "UpdateMemberRoles"
	cmdRemoveMember      command.Type = "RemoveMember"

	cmdCreateTenant     command.Type = "CreateTenant"
	cmdSuspendTenant    command.Type = "SuspendTenant"
	cmdReactivateTenant command.Type = "ReactivateTenant"
	cmdDeleteTenant     command.Type = "DeleteTenant"

	cmdRegisterBot command.Type = "RegisterBot"
	cmdDeleteBot   command.Type = "DeleteBot"
)

var allUserEventTypes = []event.Type{
	eventUserRegistered,
	eventRolesUpdated,
	eventEmailChanged,
	eventDisplayNameChanged,
	eventUserDeleted,
	eventCredentialAdded,
	eventCredentialRemoved,
	eventEmailVerified,
	eventTOTPEnabled,
	eventTOTPDisabled,
	eventExternalAccountLinked,
	eventExternalAccountUnlinked,
}

var allMembershipEventTypes = []event.Type{
	eventMemberAdded,
	eventMemberRolesChanged,
	eventMemberRemoved,
}

var allTenantEventTypes = []event.Type{
	eventTenantCreated,
	eventTenantSuspended,
	eventTenantReactivated,
	eventTenantDeleted,
}

var allBotEventTypes = []event.Type{
	eventBotRegistered,
	eventBotDeleted,
}

const currentSchemaVersion = 2
