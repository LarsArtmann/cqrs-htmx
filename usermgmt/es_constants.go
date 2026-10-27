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

	eventLoginSucceeded     = identitymodel.EventLoginSucceeded
	eventLoginFailed        = identitymodel.EventLoginFailed
	eventRegistrationClosed = identitymodel.EventRegistrationClosed

	aggregateTypeAuth = identitymodel.AggregateTypeAuth

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
	cmdRecordCredentialUse   = identitymodel.CmdRecordCredentialUse
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
	eventCredentialUsed,
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

// allAuthEventTypes lists the auth audit event types (LoginSucceeded,
// LoginFailed, RegistrationClosed). They live on dedicated Auth-typed
// streams, never on user streams — the user read model intentionally does
// not subscribe to them; the AuditLog does.
var allAuthEventTypes = []event.Type{
	eventLoginSucceeded,
	eventLoginFailed,
	eventRegistrationClosed,
}

// allAuditedEventTypes is every event type the AuditLog subscribes to: the
// user lifecycle plus the auth audit journal (cqrs-htmx#23). slices.Concat
// keeps the two source lists untouched (append would alias their backing
// array).
var allAuditedEventTypes = slices.Concat(allUserEventTypes, allAuthEventTypes)

var currentSchemaVersion = identitymodel.CurrentSchemaVersion
