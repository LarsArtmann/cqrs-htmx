package usermgmt

import (
	"slices"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
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
	eventCredentialUsed          = identitymodel.EventCredentialUsed
	eventEmailVerified           = identitymodel.EventEmailVerified
	eventTOTPEnabled             = identitymodel.EventTOTPEnabled
	eventTOTPDisabled            = identitymodel.EventTOTPDisabled
	eventExternalAccountLinked   = identitymodel.EventExternalAccountLinked
	eventExternalAccountUnlinked = identitymodel.EventExternalAccountUnlinked
	eventUserMerged              = identitymodel.EventUserMerged
	eventUserMergedInto          = identitymodel.EventUserMergedInto

	eventLoginSucceeded     = identitymodel.EventLoginSucceeded
	eventLoginFailed        = identitymodel.EventLoginFailed
	eventRegistrationClosed = identitymodel.EventRegistrationClosed

	eventSessionStarted = identitymodel.EventSessionStarted
	eventSessionEnded   = identitymodel.EventSessionEnded

	aggregateTypeAuth = identitymodel.AggregateTypeAuth

	eventMemberAdded        = identitymodel.EventMemberAdded
	eventMemberRolesChanged = identitymodel.EventMemberRolesChanged
	eventMemberRemoved      = identitymodel.EventMemberRemoved

	eventTenantCreated     = identitymodel.EventTenantCreated
	eventTenantSuspended   = identitymodel.EventTenantSuspended
	eventTenantReactivated = identitymodel.EventTenantReactivated
	eventTenantDeleted     = identitymodel.EventTenantDeleted
	eventTenantRenamed     = identitymodel.EventTenantRenamed
	eventTenantMerged      = identitymodel.EventTenantMerged
	eventTenantMergedInto  = identitymodel.EventTenantMergedInto

	eventBotRegistered  = identitymodel.EventBotRegistered
	eventBotDeleted     = identitymodel.EventBotDeleted
	eventBotTransferred = identitymodel.EventBotTransferred

	cmdRegisterUser          = identitymodel.CmdRegisterUser
	cmdChangeEmail           = identitymodel.CmdChangeEmail
	cmdChangeDisplayName     = identitymodel.CmdChangeDisplayName
	cmdDeleteUser            = identitymodel.CmdDeleteUser
	cmdAddCredential         = identitymodel.CmdAddCredential
	cmdRemoveCredential      = identitymodel.CmdRemoveCredential
	cmdRecordCredentialUse   = identitymodel.CmdRecordCredentialUse
	cmdVerifyEmail           = identitymodel.CmdVerifyEmail
	cmdEnableTOTP            = identitymodel.CmdEnableTOTP
	cmdDisableTOTP           = identitymodel.CmdDisableTOTP
	cmdLinkExternalAccount   = identitymodel.CmdLinkExternalAccount
	cmdUnlinkExternalAccount = identitymodel.CmdUnlinkExternalAccount
	cmdMergeUsers            = identitymodel.CmdMergeUsers
	cmdMergeUserInto         = identitymodel.CmdMergeUserInto

	cmdAddMember         = identitymodel.CmdAddMember
	cmdUpdateMemberRoles = identitymodel.CmdUpdateMemberRoles
	cmdRemoveMember      = identitymodel.CmdRemoveMember

	cmdCreateTenant     = identitymodel.CmdCreateTenant
	cmdSuspendTenant    = identitymodel.CmdSuspendTenant
	cmdReactivateTenant = identitymodel.CmdReactivateTenant
	cmdDeleteTenant     = identitymodel.CmdDeleteTenant
	cmdRenameTenant     = identitymodel.CmdRenameTenant
	cmdMergeTenants     = identitymodel.CmdMergeTenants
	cmdMergeTenantInto  = identitymodel.CmdMergeTenantInto

	cmdRegisterBot = identitymodel.CmdRegisterBot
	cmdDeleteBot   = identitymodel.CmdDeleteBot
	cmdTransferBot = identitymodel.CmdTransferBot
)

var allUserEventTypes = []event.Type{
	eventUserRegistered,
	eventRolesUpdated, // legacy: decoded for backward compat, no longer emitted
	eventEmailChanged,
	eventDisplayNameChanged,
	eventUserDeleted,
	eventCredentialAdded,
	eventCredentialRemoved,
	eventCredentialUsed,
	eventEmailVerified,
	eventTOTPEnabled,
	eventTOTPDisabled,
	eventExternalAccountLinked,
	eventExternalAccountUnlinked,
	eventUserMerged,
	eventUserMergedInto,
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
	eventTenantRenamed,
	eventTenantMerged,
	eventTenantMergedInto,
}

var allBotEventTypes = []event.Type{
	eventBotRegistered,
	eventBotDeleted,
	eventBotTransferred,
}

// allAuthEventTypes lists the auth audit event types (LoginSucceeded,
// LoginFailed, RegistrationClosed, SessionStarted, SessionEnded). They live
// on dedicated Auth-typed streams, never on user streams — the user read
// model intentionally does not subscribe to them; the AuditLog does.
var allAuthEventTypes = []event.Type{
	eventLoginSucceeded,
	eventLoginFailed,
	eventRegistrationClosed,
	eventSessionStarted,
	eventSessionEnded,
}

// allAuditedEventTypes is every event type the AuditLog subscribes to: the
// user lifecycle plus the auth audit journal (cqrs-htmx#23). slices.Concat
// keeps the two source lists untouched (append would alias their backing
// array).
var allAuditedEventTypes = slices.Concat(allUserEventTypes, allAuthEventTypes)

var currentSchemaVersion = identitymodel.CurrentSchemaVersion
