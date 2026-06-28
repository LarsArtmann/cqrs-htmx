package usermgmt

import (
	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

type RegisterUserCmd struct {
	aggregateID id.AggregateID
	cmdID       id.CommandID
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
func (c *RegisterUserCmd) ID() id.CommandID            { return c.cmdID }

type ChangeEmailCmd struct {
	aggregateID id.AggregateID
	cmdID       id.CommandID
	email       string
}

func NewChangeEmailCmd(aggID id.AggregateID, email string) *ChangeEmailCmd {
	return &ChangeEmailCmd{aggregateID: aggID, cmdID: id.NewCommandID(), email: email}
}

func (c *ChangeEmailCmd) Type() command.Type          { return cmdChangeEmail }
func (c *ChangeEmailCmd) AggregateID() id.AggregateID { return c.aggregateID }
func (c *ChangeEmailCmd) ID() id.CommandID            { return c.cmdID }

type ChangeDisplayNameCmd struct {
	aggregateID id.AggregateID
	cmdID       id.CommandID
	displayName string
}

func NewChangeDisplayNameCmd(aggID id.AggregateID, displayName string) *ChangeDisplayNameCmd {
	return &ChangeDisplayNameCmd{aggregateID: aggID, cmdID: id.NewCommandID(), displayName: displayName}
}

func (c *ChangeDisplayNameCmd) Type() command.Type          { return cmdChangeDisplayName }
func (c *ChangeDisplayNameCmd) AggregateID() id.AggregateID { return c.aggregateID }
func (c *ChangeDisplayNameCmd) ID() id.CommandID            { return c.cmdID }

type DeleteUserCmd struct {
	aggregateID id.AggregateID
	cmdID       id.CommandID
	reason      string
}

func NewDeleteUserCmd(aggID id.AggregateID, reason string) *DeleteUserCmd {
	return &DeleteUserCmd{aggregateID: aggID, cmdID: id.NewCommandID(), reason: reason}
}

func (c *DeleteUserCmd) Type() command.Type          { return cmdDeleteUser }
func (c *DeleteUserCmd) AggregateID() id.AggregateID { return c.aggregateID }
func (c *DeleteUserCmd) ID() id.CommandID            { return c.cmdID }

type AddCredentialCmd struct {
	aggregateID id.AggregateID
	cmdID       id.CommandID
	credential  WebAuthnCredential
}

func NewAddCredentialCmd(aggID id.AggregateID, cred WebAuthnCredential) *AddCredentialCmd {
	return &AddCredentialCmd{aggregateID: aggID, cmdID: id.NewCommandID(), credential: cred}
}

func (c *AddCredentialCmd) Type() command.Type          { return cmdAddCredential }
func (c *AddCredentialCmd) AggregateID() id.AggregateID { return c.aggregateID }
func (c *AddCredentialCmd) ID() id.CommandID            { return c.cmdID }

type RemoveCredentialCmd struct {
	aggregateID  id.AggregateID
	cmdID        id.CommandID
	credentialID []byte
}

func NewRemoveCredentialCmd(aggID id.AggregateID, credID []byte) *RemoveCredentialCmd {
	return &RemoveCredentialCmd{aggregateID: aggID, cmdID: id.NewCommandID(), credentialID: credID}
}

func (c *RemoveCredentialCmd) Type() command.Type          { return cmdRemoveCredential }
func (c *RemoveCredentialCmd) AggregateID() id.AggregateID { return c.aggregateID }
func (c *RemoveCredentialCmd) ID() id.CommandID            { return c.cmdID }

type VerifyEmailCmd struct {
	aggregateID id.AggregateID
	cmdID       id.CommandID
}

func NewVerifyEmailCmd(aggID id.AggregateID) *VerifyEmailCmd {
	return &VerifyEmailCmd{aggregateID: aggID, cmdID: id.NewCommandID()}
}

func (c *VerifyEmailCmd) Type() command.Type          { return cmdVerifyEmail }
func (c *VerifyEmailCmd) AggregateID() id.AggregateID { return c.aggregateID }
func (c *VerifyEmailCmd) ID() id.CommandID            { return c.cmdID }

type EnableTOTPCmd struct {
	aggregateID id.AggregateID
	cmdID       id.CommandID
	secret      []byte
}

func NewEnableTOTPCmd(aggID id.AggregateID, secret []byte) *EnableTOTPCmd {
	return &EnableTOTPCmd{aggregateID: aggID, cmdID: id.NewCommandID(), secret: secret}
}

func (c *EnableTOTPCmd) Type() command.Type          { return cmdEnableTOTP }
func (c *EnableTOTPCmd) AggregateID() id.AggregateID { return c.aggregateID }
func (c *EnableTOTPCmd) ID() id.CommandID            { return c.cmdID }

type DisableTOTPCmd struct {
	aggregateID id.AggregateID
	cmdID       id.CommandID
}

func NewDisableTOTPCmd(aggID id.AggregateID) *DisableTOTPCmd {
	return &DisableTOTPCmd{aggregateID: aggID, cmdID: id.NewCommandID()}
}

func (c *DisableTOTPCmd) Type() command.Type          { return cmdDisableTOTP }
func (c *DisableTOTPCmd) AggregateID() id.AggregateID { return c.aggregateID }
func (c *DisableTOTPCmd) ID() id.CommandID            { return c.cmdID }

type LinkExternalAccountCmd struct {
	aggregateID id.AggregateID
	cmdID       id.CommandID
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
func (c *LinkExternalAccountCmd) ID() id.CommandID            { return c.cmdID }

type UnlinkExternalAccountCmd struct {
	aggregateID id.AggregateID
	cmdID       id.CommandID
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
func (c *UnlinkExternalAccountCmd) ID() id.CommandID { return c.cmdID }
