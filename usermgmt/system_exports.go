package usermgmt

import (
	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// DecideRegisterUser is the exported version of decideRegisterUser for system.New() integration.
func DecideRegisterUser(aggID id.StreamID, email, displayName string, roles []Role) decider.DecideFunc[UserState] {
	return decideRegisterUser(aggID, email, displayName, roles)
}

// DecideDeleteUser is the exported version of decideDeleteUser for system.New() integration.
func DecideDeleteUser(aggID id.StreamID, reason string) decider.DecideFunc[UserState] {
	return decideDeleteUser(aggID, reason)
}

// DecideChangeEmail is the exported version of decideChangeEmail for system.New() integration.
func DecideChangeEmail(aggID id.StreamID, email string) decider.DecideFunc[UserState] {
	return decideChangeEmail(aggID, email)
}

// DecideChangeDisplayName is the exported version of decideChangeDisplayName for system.New() integration.
func DecideChangeDisplayName(aggID id.StreamID, displayName string) decider.DecideFunc[UserState] {
	return decideChangeDisplayName(aggID, displayName)
}

// DecideVerifyEmail is the exported version of decideVerifyEmail for system.New() integration.
func DecideVerifyEmail(aggID id.StreamID) decider.DecideFunc[UserState] {
	return decideVerifyEmail(aggID)
}

// DecideAddCredential is the exported version of decideAddCredential for system.New() integration.
func DecideAddCredential(aggID id.StreamID, cred WebAuthnCredential) decider.DecideFunc[UserState] {
	return decideAddCredential(aggID, cred)
}

// DecideRemoveCredential is the exported version of decideRemoveCredential for system.New() integration.
func DecideRemoveCredential(aggID id.StreamID, credentialID []byte) decider.DecideFunc[UserState] {
	return decideRemoveCredential(aggID, credentialID)
}

// DecideRecordCredentialUse is the exported version of decideRecordCredentialUse
// for system.New() integration. Without it, the systemadapter composition could
// not record assertion state and the declarative UserView kept the
// registration-time signature counter forever (found by the R02 proof test,
// 2026-10-25).
func DecideRecordCredentialUse(aggID id.StreamID, use CredentialUse) decider.DecideFunc[UserState] {
	return decideRecordCredentialUse(aggID, use)
}

// DecideEnableTOTP is the exported version of decideEnableTOTP for system.New() integration.
func DecideEnableTOTP(aggID id.StreamID, secret []byte) decider.DecideFunc[UserState] {
	return decideEnableTOTP(aggID, secret)
}

// DecideDisableTOTP is the exported version of decideDisableTOTP for system.New() integration.
func DecideDisableTOTP(aggID id.StreamID) decider.DecideFunc[UserState] {
	return decideDisableTOTP(aggID)
}

// DecideLinkExternalAccount is the exported version of decideLinkExternalAccount for system.New() integration.
func DecideLinkExternalAccount(
	aggID id.StreamID,
	provider, subject, email, displayName string,
) decider.DecideFunc[UserState] {
	return decideLinkExternalAccount(aggID, provider, subject, email, displayName)
}

// DecideUnlinkExternalAccount is the exported version of decideUnlinkExternalAccount for system.New() integration.
func DecideUnlinkExternalAccount(aggID id.StreamID, provider, subject string) decider.DecideFunc[UserState] {
	return decideUnlinkExternalAccount(aggID, provider, subject)
}

// DecideMergeUsers is the exported version of decideMergeUsers for system.New() integration.
func DecideMergeUsers(
	aggID id.StreamID,
	contributions []identitymodel.MergeContribution,
	reason string,
) decider.DecideFunc[UserState] {
	return decideMergeUsers(aggID, contributions, reason)
}

// DecideMergeUserInto is the exported version of decideMergeUserInto for system.New() integration.
func DecideMergeUserInto(aggID id.StreamID, survivorID UserID, reason string) decider.DecideFunc[UserState] {
	return decideMergeUserInto(aggID, survivorID, reason)
}

// Membership decide exports

// DecideAddMember is the exported version of decideAddMember for system.New() integration.
func DecideAddMember(
	aggID id.StreamID,
	actorID ActorID,
	tenantID TenantID,
	roles []Role,
) decider.DecideFunc[MembershipState] {
	return decideAddMember(aggID, actorID, tenantID, roles)
}

// DecideUpdateMemberRoles is the exported version of decideUpdateMemberRoles for system.New() integration.
func DecideUpdateMemberRoles(aggID id.StreamID, roles []Role) decider.DecideFunc[MembershipState] {
	return decideUpdateMemberRoles(aggID, roles)
}

// DecideRemoveMember is the exported version of decideRemoveMember for system.New() integration.
func DecideRemoveMember(aggID id.StreamID) decider.DecideFunc[MembershipState] {
	return decideRemoveMember(aggID)
}

// Tenant decide exports

// DecideCreateTenant is the exported version of decideCreateTenant for system.New() integration.
func DecideCreateTenant(aggID id.StreamID, name, displayName string) decider.DecideFunc[TenantState] {
	return decideCreateTenant(aggID, name, displayName)
}

// DecideSuspendTenant is the exported version of decideSuspendTenant for system.New() integration.
func DecideSuspendTenant(aggID id.StreamID, reason string) decider.DecideFunc[TenantState] {
	return decideSuspendTenant(aggID, reason)
}

// DecideReactivateTenant is the exported version of decideReactivateTenant for system.New() integration.
func DecideReactivateTenant(aggID id.StreamID) decider.DecideFunc[TenantState] {
	return decideReactivateTenant(aggID)
}

// DecideDeleteTenant is the exported version of decideDeleteTenant for system.New() integration.
func DecideDeleteTenant(aggID id.StreamID, reason string) decider.DecideFunc[TenantState] {
	return decideDeleteTenant(aggID, reason)
}

// DecideRenameTenant is the exported version of decideRenameTenant for system.New() integration.
func DecideRenameTenant(aggID id.StreamID, displayName string) decider.DecideFunc[TenantState] {
	return decideRenameTenant(aggID, displayName)
}

// DecideMergeTenants is the exported version of decideMergeTenants for system.New() integration.
func DecideMergeTenants(
	aggID id.StreamID, loserIDs []identitymodel.TenantID, reason string,
) decider.DecideFunc[TenantState] {
	return decideMergeTenants(aggID, loserIDs, reason)
}

// DecideMergeTenantInto is the exported version of decideMergeTenantInto for system.New() integration.
func DecideMergeTenantInto(
	aggID id.StreamID, survivorID identitymodel.TenantID, reason string,
) decider.DecideFunc[TenantState] {
	return decideMergeTenantInto(aggID, survivorID, reason)
}

// Bot decide exports

// DecideRegisterBot is the exported version of decideRegisterBot for system.New() integration.
func DecideRegisterBot(
	aggID id.StreamID,
	name string,
	ownerID UserID,
	tokenHash []byte,
	scopes []string,
) decider.DecideFunc[BotState] {
	return decideRegisterBot(aggID, name, ownerID, tokenHash, scopes)
}

// DecideDeleteBot is the exported version of decideDeleteBot for system.New() integration.
func DecideDeleteBot(aggID id.StreamID, reason string) decider.DecideFunc[BotState] {
	return decideDeleteBot(aggID, reason)
}

// DecideTransferBot is the exported version of decideTransferBot for system.New() integration.
func DecideTransferBot(aggID id.StreamID, ownerID UserID, reason string) decider.DecideFunc[BotState] {
	return decideTransferBot(aggID, ownerID, reason)
}
