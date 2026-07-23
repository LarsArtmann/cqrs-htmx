package usermgmt

import (
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
)

var (
	aggregateTypeUser       = identitymodel.AggregateTypeUser
	aggregateTypeMembership = identitymodel.AggregateTypeMembership
	aggregateTypeTenant     = identitymodel.AggregateTypeTenant
	aggregateTypeBot        = identitymodel.AggregateTypeBot

	eventUserRegistered          = identitymodel.EventUserRegistered
	eventRolesUpdated            = identitymodel.EventRolesUpdated
	eventEmailChanged            = identitymodel.EventEmailChanged
	eventDisplayNameChanged      = identitymodel.EventDisplayNameChanged
	eventUserDeleted             = identitymodel.EventUserDeleted
	eventCredentialAdded         = identitymodel.EventCredentialAdded
	eventCredentialRemoved       = identitymodel.EventCredentialRemoved
	eventEmailVerified           = identitymodel.EventEmailVerified
	eventTOTPEnabled             = identitymodel.EventTOTPEnabled
	eventTOTPDisabled            = identitymodel.EventTOTPDisabled
	eventExternalAccountLinked   = identitymodel.EventExternalAccountLinked
	eventExternalAccountUnlinked = identitymodel.EventExternalAccountUnlinked

	eventMemberAdded        = identitymodel.EventMemberAdded
	eventMemberRolesChanged = identitymodel.EventMemberRolesChanged
	eventMemberRemoved      = identitymodel.EventMemberRemoved

	eventTenantCreated     = identitymodel.EventTenantCreated
	eventTenantSuspended   = identitymodel.EventTenantSuspended
	eventTenantReactivated = identitymodel.EventTenantReactivated
	eventTenantDeleted     = identitymodel.EventTenantDeleted

	eventBotRegistered = identitymodel.EventBotRegistered
	eventBotDeleted    = identitymodel.EventBotDeleted

	cmdRegisterUser          = identitymodel.CmdRegisterUser
	cmdChangeEmail           = identitymodel.CmdChangeEmail
	cmdChangeDisplayName     = identitymodel.CmdChangeDisplayName
	cmdDeleteUser            = identitymodel.CmdDeleteUser
	cmdAddCredential         = identitymodel.CmdAddCredential
	cmdRemoveCredential      = identitymodel.CmdRemoveCredential
	cmdVerifyEmail           = identitymodel.CmdVerifyEmail
	cmdEnableTOTP            = identitymodel.CmdEnableTOTP
	cmdDisableTOTP           = identitymodel.CmdDisableTOTP
	cmdLinkExternalAccount   = identitymodel.CmdLinkExternalAccount
	cmdUnlinkExternalAccount = identitymodel.CmdUnlinkExternalAccount

	cmdAddMember         = identitymodel.CmdAddMember
	cmdUpdateMemberRoles = identitymodel.CmdUpdateMemberRoles
	cmdRemoveMember      = identitymodel.CmdRemoveMember

	cmdCreateTenant     = identitymodel.CmdCreateTenant
	cmdSuspendTenant    = identitymodel.CmdSuspendTenant
	cmdReactivateTenant = identitymodel.CmdReactivateTenant
	cmdDeleteTenant     = identitymodel.CmdDeleteTenant

	cmdRegisterBot = identitymodel.CmdRegisterBot
	cmdDeleteBot   = identitymodel.CmdDeleteBot
)

var allUserEventTypes = []event.Type{
	eventUserRegistered,
	eventRolesUpdated, // legacy: decoded for backward compat, no longer emitted
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

var currentSchemaVersion = identitymodel.CurrentSchemaVersion
