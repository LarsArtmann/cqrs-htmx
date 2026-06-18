package usermgmt

import (
	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

type RegisterUserCmd struct {
	aggregateID id.AggregateID
	email       string
	displayName string
	roles       []Role
}

func NewRegisterUserCmd(
	aggID id.AggregateID, email, displayName string, roles []Role,
) *RegisterUserCmd {
	return &RegisterUserCmd{
		aggregateID: aggID,
		email:       email,
		displayName: displayName,
		roles:       roles,
	}
}

func (c *RegisterUserCmd) Type() command.Type          { return cmdRegisterUser }
func (c *RegisterUserCmd) AggregateID() id.AggregateID { return c.aggregateID }

type UpdateRolesCmd struct {
	aggregateID id.AggregateID
	roles       []Role
	domain      string
}

func NewUpdateRolesCmd(aggID id.AggregateID, roles []Role, domain string) *UpdateRolesCmd {
	return &UpdateRolesCmd{aggregateID: aggID, roles: roles, domain: domain}
}

func (c *UpdateRolesCmd) Type() command.Type          { return cmdUpdateRoles }
func (c *UpdateRolesCmd) AggregateID() id.AggregateID { return c.aggregateID }

type ChangeEmailCmd struct {
	aggregateID id.AggregateID
	email       string
}

func NewChangeEmailCmd(aggID id.AggregateID, email string) *ChangeEmailCmd {
	return &ChangeEmailCmd{aggregateID: aggID, email: email}
}

func (c *ChangeEmailCmd) Type() command.Type          { return cmdChangeEmail }
func (c *ChangeEmailCmd) AggregateID() id.AggregateID { return c.aggregateID }

type ChangeDisplayNameCmd struct {
	aggregateID id.AggregateID
	displayName string
}

func NewChangeDisplayNameCmd(aggID id.AggregateID, displayName string) *ChangeDisplayNameCmd {
	return &ChangeDisplayNameCmd{aggregateID: aggID, displayName: displayName}
}

func (c *ChangeDisplayNameCmd) Type() command.Type          { return cmdChangeDisplayName }
func (c *ChangeDisplayNameCmd) AggregateID() id.AggregateID { return c.aggregateID }

type DeleteUserCmd struct {
	aggregateID id.AggregateID
	reason      string
}

func NewDeleteUserCmd(aggID id.AggregateID, reason string) *DeleteUserCmd {
	return &DeleteUserCmd{aggregateID: aggID, reason: reason}
}

func (c *DeleteUserCmd) Type() command.Type          { return cmdDeleteUser }
func (c *DeleteUserCmd) AggregateID() id.AggregateID { return c.aggregateID }

type AddCredentialCmd struct {
	aggregateID id.AggregateID
	credential  WebAuthnCredential
}

func NewAddCredentialCmd(aggID id.AggregateID, cred WebAuthnCredential) *AddCredentialCmd {
	return &AddCredentialCmd{aggregateID: aggID, credential: cred}
}

func (c *AddCredentialCmd) Type() command.Type          { return cmdAddCredential }
func (c *AddCredentialCmd) AggregateID() id.AggregateID { return c.aggregateID }

type RemoveCredentialCmd struct {
	aggregateID  id.AggregateID
	credentialID []byte
}

func NewRemoveCredentialCmd(aggID id.AggregateID, credID []byte) *RemoveCredentialCmd {
	return &RemoveCredentialCmd{aggregateID: aggID, credentialID: credID}
}

func (c *RemoveCredentialCmd) Type() command.Type          { return cmdRemoveCredential }
func (c *RemoveCredentialCmd) AggregateID() id.AggregateID { return c.aggregateID }

type VerifyEmailCmd struct {
	aggregateID id.AggregateID
}

func NewVerifyEmailCmd(aggID id.AggregateID) *VerifyEmailCmd {
	return &VerifyEmailCmd{aggregateID: aggID}
}

func (c *VerifyEmailCmd) Type() command.Type          { return cmdVerifyEmail }
func (c *VerifyEmailCmd) AggregateID() id.AggregateID { return c.aggregateID }

type EnableTOTPCmd struct {
	aggregateID id.AggregateID
	secret      []byte
}

func NewEnableTOTPCmd(aggID id.AggregateID, secret []byte) *EnableTOTPCmd {
	return &EnableTOTPCmd{aggregateID: aggID, secret: secret}
}

func (c *EnableTOTPCmd) Type() command.Type          { return cmdEnableTOTP }
func (c *EnableTOTPCmd) AggregateID() id.AggregateID { return c.aggregateID }

type DisableTOTPCmd struct {
	aggregateID id.AggregateID
}

func NewDisableTOTPCmd(aggID id.AggregateID) *DisableTOTPCmd {
	return &DisableTOTPCmd{aggregateID: aggID}
}

func (c *DisableTOTPCmd) Type() command.Type          { return cmdDisableTOTP }
func (c *DisableTOTPCmd) AggregateID() id.AggregateID { return c.aggregateID }

type LinkExternalAccountCmd struct {
	aggregateID id.AggregateID
	provider    string
	subject     string
	email       string
	displayName string
}

func NewLinkExternalAccountCmd(
	aggID id.AggregateID, provider, subject, email, displayName string,
) *LinkExternalAccountCmd {
	return &LinkExternalAccountCmd{
		aggregateID: aggID,
		provider:    provider,
		subject:     subject,
		email:       email,
		displayName: displayName,
	}
}

func (c *LinkExternalAccountCmd) Type() command.Type          { return cmdLinkExternalAccount }
func (c *LinkExternalAccountCmd) AggregateID() id.AggregateID { return c.aggregateID }

type UnlinkExternalAccountCmd struct {
	aggregateID id.AggregateID
	provider    string
	subject     string
}

func NewUnlinkExternalAccountCmd(
	aggID id.AggregateID, provider, subject string,
) *UnlinkExternalAccountCmd {
	return &UnlinkExternalAccountCmd{
		aggregateID: aggID,
		provider:    provider,
		subject:     subject,
	}
}

func (c *UnlinkExternalAccountCmd) Type() command.Type { return cmdUnlinkExternalAccount } //nolint:lll // struct method
func (c *UnlinkExternalAccountCmd) AggregateID() id.AggregateID {
	return c.aggregateID
}
