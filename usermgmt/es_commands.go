package usermgmt

// All usermgmt command structs embed *command.BasicCommand, which promotes
// Type(), AggregateID(), and ID() methods automatically. This structurally
// eliminates the class of bugs where a constructor forgets to mint a command
// ID (which would silently break idempotency dedup and Watermill message
// UUIDs). The mustCommand helper panics on construction failure — the only
// error cases (empty command type, zero aggregate ID) are programming bugs.
// See ADR-0032 for the full rationale.

import (
	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	errorfamily "github.com/larsartmann/go-error-family"
)

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

// mustCommand creates a BasicCommand, panicking on error.
// The only error cases are empty command type (programming bug) or zero
// aggregate ID (programming bug) — neither should happen at runtime.
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
