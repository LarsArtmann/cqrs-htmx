package usermgmt

import (
	"fmt"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

type (
	RegisterUserCmd          = identitymodel.RegisterUserCmd
	ChangeEmailCmd           = identitymodel.ChangeEmailCmd
	ChangeDisplayNameCmd     = identitymodel.ChangeDisplayNameCmd
	DeleteUserCmd            = identitymodel.DeleteUserCmd
	AddCredentialCmd         = identitymodel.AddCredentialCmd
	RemoveCredentialCmd      = identitymodel.RemoveCredentialCmd
	VerifyEmailCmd           = identitymodel.VerifyEmailCmd
	EnableTOTPCmd            = identitymodel.EnableTOTPCmd
	DisableTOTPCmd           = identitymodel.DisableTOTPCmd
	LinkExternalAccountCmd   = identitymodel.LinkExternalAccountCmd
	UnlinkExternalAccountCmd = identitymodel.UnlinkExternalAccountCmd
)

func NewRegisterUserCmd(aggID id.AggregateID, email, displayName string, roles []Role) *RegisterUserCmd {
	return identitymodel.NewRegisterUserCmd(aggID, email, displayName, roles)
}

func NewChangeEmailCmd(aggID id.AggregateID, email string) *ChangeEmailCmd {
	return identitymodel.NewChangeEmailCmd(aggID, email)
}

func NewChangeDisplayNameCmd(aggID id.AggregateID, displayName string) *ChangeDisplayNameCmd {
	return identitymodel.NewChangeDisplayNameCmd(aggID, displayName)
}

func NewDeleteUserCmd(aggID id.AggregateID, reason string) *DeleteUserCmd {
	return identitymodel.NewDeleteUserCmd(aggID, reason)
}

func NewAddCredentialCmd(aggID id.AggregateID, cred WebAuthnCredential) *AddCredentialCmd {
	return identitymodel.NewAddCredentialCmd(aggID, cred)
}

func NewRemoveCredentialCmd(aggID id.AggregateID, credID []byte) *RemoveCredentialCmd {
	return identitymodel.NewRemoveCredentialCmd(aggID, credID)
}

func NewVerifyEmailCmd(aggID id.AggregateID) *VerifyEmailCmd {
	return identitymodel.NewVerifyEmailCmd(aggID)
}

func NewEnableTOTPCmd(aggID id.AggregateID, secret []byte) *EnableTOTPCmd {
	return identitymodel.NewEnableTOTPCmd(aggID, secret)
}

func NewDisableTOTPCmd(aggID id.AggregateID) *DisableTOTPCmd {
	return identitymodel.NewDisableTOTPCmd(aggID)
}

func NewLinkExternalAccountCmd(
	aggID id.AggregateID, provider, subject, email, displayName string,
) *LinkExternalAccountCmd {
	return identitymodel.NewLinkExternalAccountCmd(aggID, provider, subject, email, displayName)
}

func NewUnlinkExternalAccountCmd(aggID id.AggregateID, provider, subject string) *UnlinkExternalAccountCmd {
	return identitymodel.NewUnlinkExternalAccountCmd(aggID, provider, subject)
}

func mustCommand(cmdType command.Type, aggID id.AggregateID) *command.BasicCommand {
	base, err := command.New(cmdType, aggID)
	if err != nil {
		panic(fmt.Sprintf("mustCommand(%q, %s): %v", cmdType, aggID, err))
	}
	return base
}
