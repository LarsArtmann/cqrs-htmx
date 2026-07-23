package identitymodel

import (
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// --- User commands ---

type RegisterUserCmd struct {
	*command.BasicCommand
	email       string
	displayName string
	roles       []Role
}

func NewRegisterUserCmd(
	aggID id.AggregateID, email, displayName string, roles []Role,
) *RegisterUserCmd {
	return &RegisterUserCmd{
		BasicCommand: mustCommand(cmdRegisterUser, aggID),
		email:        email,
		displayName:  displayName,
		roles:        roles,
	}
}

func (c *RegisterUserCmd) Email() string       { return c.email }
func (c *RegisterUserCmd) DisplayName() string { return c.displayName }
func (c *RegisterUserCmd) Roles() []Role       { return c.roles }

type ChangeEmailCmd struct {
	*command.BasicCommand
	email string
}

func NewChangeEmailCmd(aggID id.AggregateID, email string) *ChangeEmailCmd {
	return &ChangeEmailCmd{
		BasicCommand: mustCommand(cmdChangeEmail, aggID),
		email:        email,
	}
}

func (c *ChangeEmailCmd) Email() string { return c.email }

type ChangeDisplayNameCmd struct {
	*command.BasicCommand
	displayName string
}

func NewChangeDisplayNameCmd(aggID id.AggregateID, displayName string) *ChangeDisplayNameCmd {
	return &ChangeDisplayNameCmd{
		BasicCommand: mustCommand(cmdChangeDisplayName, aggID),
		displayName:  displayName,
	}
}

func (c *ChangeDisplayNameCmd) DisplayName() string { return c.displayName }

type DeleteUserCmd struct {
	*command.BasicCommand
	reason string
}

func NewDeleteUserCmd(aggID id.AggregateID, reason string) *DeleteUserCmd {
	return &DeleteUserCmd{
		BasicCommand: mustCommand(cmdDeleteUser, aggID),
		reason:       reason,
	}
}

func (c *DeleteUserCmd) Reason() string { return c.reason }

type AddCredentialCmd struct {
	*command.BasicCommand
	credential WebAuthnCredential
}

func NewAddCredentialCmd(aggID id.AggregateID, cred WebAuthnCredential) *AddCredentialCmd {
	return &AddCredentialCmd{
		BasicCommand: mustCommand(cmdAddCredential, aggID),
		credential:   cred,
	}
}

func (c *AddCredentialCmd) Credential() WebAuthnCredential { return c.credential }

type RemoveCredentialCmd struct {
	*command.BasicCommand
	credentialID []byte
}

func NewRemoveCredentialCmd(aggID id.AggregateID, credID []byte) *RemoveCredentialCmd {
	return &RemoveCredentialCmd{
		BasicCommand: mustCommand(cmdRemoveCredential, aggID),
		credentialID: credID,
	}
}

func (c *RemoveCredentialCmd) CredentialID() []byte { return c.credentialID }

type VerifyEmailCmd struct {
	*command.BasicCommand
}

func NewVerifyEmailCmd(aggID id.AggregateID) *VerifyEmailCmd {
	return &VerifyEmailCmd{
		BasicCommand: mustCommand(cmdVerifyEmail, aggID),
	}
}

type EnableTOTPCmd struct {
	*command.BasicCommand
	secret []byte
}

func NewEnableTOTPCmd(aggID id.AggregateID, secret []byte) *EnableTOTPCmd {
	return &EnableTOTPCmd{
		BasicCommand: mustCommand(cmdEnableTOTP, aggID),
		secret:       secret,
	}
}

func (c *EnableTOTPCmd) Secret() []byte { return c.secret }

type DisableTOTPCmd struct {
	*command.BasicCommand
}

func NewDisableTOTPCmd(aggID id.AggregateID) *DisableTOTPCmd {
	return &DisableTOTPCmd{
		BasicCommand: mustCommand(cmdDisableTOTP, aggID),
	}
}

type LinkExternalAccountCmd struct {
	*command.BasicCommand
	provider    string
	subject     string
	email       string
	displayName string
}

func NewLinkExternalAccountCmd(
	aggID id.AggregateID, provider, subject, email, displayName string,
) *LinkExternalAccountCmd {
	return &LinkExternalAccountCmd{
		BasicCommand: mustCommand(cmdLinkExternalAccount, aggID),
		provider:     provider,
		subject:      subject,
		email:        email,
		displayName:  displayName,
	}
}

func (c *LinkExternalAccountCmd) Provider() string    { return c.provider }
func (c *LinkExternalAccountCmd) Subject() string     { return c.subject }
func (c *LinkExternalAccountCmd) Email() string       { return c.email }
func (c *LinkExternalAccountCmd) DisplayName() string { return c.displayName }

type UnlinkExternalAccountCmd struct {
	*command.BasicCommand
	provider string
	subject  string
}

func NewUnlinkExternalAccountCmd(
	aggID id.AggregateID, provider, subject string,
) *UnlinkExternalAccountCmd {
	return &UnlinkExternalAccountCmd{
		BasicCommand: mustCommand(cmdUnlinkExternalAccount, aggID),
		provider:     provider,
		subject:      subject,
	}
}

func (c *UnlinkExternalAccountCmd) Provider() string { return c.provider }
func (c *UnlinkExternalAccountCmd) Subject() string  { return c.subject }

// --- Membership commands ---

// DeriveMembershipID constructs a deterministic AggregateID for a membership.
func DeriveMembershipID(actorID ActorID, tenantID TenantID) id.AggregateID {
	return id.DeriveAggregateID("membership", actorID.PrefixedString(), tenantID.Get())
}

type AddMemberCmd struct {
	*command.BasicCommand
	actorID  ActorID
	tenantID TenantID
	roles    []Role
}

func NewAddMemberCmd(actorID ActorID, tenantID TenantID, roles []Role) *AddMemberCmd {
	return &AddMemberCmd{
		BasicCommand: mustCommand(cmdAddMember, DeriveMembershipID(actorID, tenantID)),
		actorID:      actorID,
		tenantID:     tenantID,
		roles:        roles,
	}
}

func (c *AddMemberCmd) ActorID() ActorID   { return c.actorID }
func (c *AddMemberCmd) TenantID() TenantID { return c.tenantID }
func (c *AddMemberCmd) Roles() []Role      { return c.roles }

type UpdateMemberRolesCmd struct {
	*command.BasicCommand
	roles []Role
}

func NewUpdateMemberRolesCmd(
	actorID ActorID, tenantID TenantID, roles []Role,
) *UpdateMemberRolesCmd {
	return &UpdateMemberRolesCmd{
		BasicCommand: mustCommand(cmdUpdateMemberRoles, DeriveMembershipID(actorID, tenantID)),
		roles:        roles,
	}
}

func (c *UpdateMemberRolesCmd) Roles() []Role { return c.roles }

type RemoveMemberCmd struct {
	*command.BasicCommand
}

func NewRemoveMemberCmd(actorID ActorID, tenantID TenantID) *RemoveMemberCmd {
	return &RemoveMemberCmd{
		BasicCommand: mustCommand(cmdRemoveMember, DeriveMembershipID(actorID, tenantID)),
	}
}

// --- Tenant commands ---

type CreateTenantCmd struct {
	*command.BasicCommand
	name        string
	displayName string
}

func NewCreateTenantCmd(aggID id.AggregateID, name, displayName string) *CreateTenantCmd {
	return &CreateTenantCmd{
		BasicCommand: mustCommand(cmdCreateTenant, aggID),
		name:         name,
		displayName:  displayName,
	}
}

func (c *CreateTenantCmd) Name() string        { return c.name }
func (c *CreateTenantCmd) DisplayName() string { return c.displayName }

type SuspendTenantCmd struct {
	*command.BasicCommand
	reason string
}

func NewSuspendTenantCmd(aggID id.AggregateID, reason string) *SuspendTenantCmd {
	return &SuspendTenantCmd{
		BasicCommand: mustCommand(cmdSuspendTenant, aggID),
		reason:       reason,
	}
}

func (c *SuspendTenantCmd) Reason() string { return c.reason }

type ReactivateTenantCmd struct {
	*command.BasicCommand
}

func NewReactivateTenantCmd(aggID id.AggregateID) *ReactivateTenantCmd {
	return &ReactivateTenantCmd{
		BasicCommand: mustCommand(cmdReactivateTenant, aggID),
	}
}

type DeleteTenantCmd struct {
	*command.BasicCommand
	reason string
}

func NewDeleteTenantCmd(aggID id.AggregateID, reason string) *DeleteTenantCmd {
	return &DeleteTenantCmd{
		BasicCommand: mustCommand(cmdDeleteTenant, aggID),
		reason:       reason,
	}
}

func (c *DeleteTenantCmd) Reason() string { return c.reason }

// --- Bot commands ---

type RegisterBotCmd struct {
	*command.BasicCommand
	name      string
	ownerID   UserID
	tokenHash []byte
	scopes    []string
}

func NewRegisterBotCmd(
	aggID id.AggregateID, name string, ownerID UserID, tokenHash []byte, scopes []string,
) *RegisterBotCmd {
	return &RegisterBotCmd{
		BasicCommand: mustCommand(cmdRegisterBot, aggID),
		name:         name,
		ownerID:      ownerID,
		tokenHash:    tokenHash,
		scopes:       scopes,
	}
}

func (c *RegisterBotCmd) Name() string      { return c.name }
func (c *RegisterBotCmd) OwnerID() UserID   { return c.ownerID }
func (c *RegisterBotCmd) TokenHash() []byte { return c.tokenHash }
func (c *RegisterBotCmd) Scopes() []string  { return c.scopes }

type DeleteBotCmd struct {
	*command.BasicCommand
	reason string
}

func NewDeleteBotCmd(aggID id.AggregateID, reason string) *DeleteBotCmd {
	return &DeleteBotCmd{
		BasicCommand: mustCommand(cmdDeleteBot, aggID),
		reason:       reason,
	}
}

func (c *DeleteBotCmd) Reason() string { return c.reason }

// mustCommand creates a BasicCommand, panicking on error.
func mustCommand(cmdType command.Type, aggID id.AggregateID) *command.BasicCommand {
	base, err := command.New(cmdType, aggID)
	if err != nil {
		panic(errorfamily.Wrapf(
			err, event.Infrastructure,
			"usermgmt.command.create_failed",
			"create %s command", cmdType,
		))
	}
	return base
}
