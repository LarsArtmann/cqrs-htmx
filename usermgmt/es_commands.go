package usermgmt

import (
	"fmt"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

type (
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	RegisterUserCmd = identitymodel.RegisterUserCmd
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ChangeEmailCmd = identitymodel.ChangeEmailCmd
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ChangeDisplayNameCmd = identitymodel.ChangeDisplayNameCmd
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	DeleteUserCmd = identitymodel.DeleteUserCmd
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	AddCredentialCmd = identitymodel.AddCredentialCmd
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	RemoveCredentialCmd = identitymodel.RemoveCredentialCmd
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	VerifyEmailCmd = identitymodel.VerifyEmailCmd
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	EnableTOTPCmd = identitymodel.EnableTOTPCmd
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	DisableTOTPCmd = identitymodel.DisableTOTPCmd
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	LinkExternalAccountCmd = identitymodel.LinkExternalAccountCmd
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	UnlinkExternalAccountCmd = identitymodel.UnlinkExternalAccountCmd
)

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func NewRegisterUserCmd(aggID id.StreamID, email, displayName string, roles []Role) *RegisterUserCmd {
	return identitymodel.NewRegisterUserCmd(aggID, email, displayName, roles)
}

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func NewChangeEmailCmd(aggID id.StreamID, email string) *ChangeEmailCmd {
	return identitymodel.NewChangeEmailCmd(aggID, email)
}

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func NewChangeDisplayNameCmd(aggID id.StreamID, displayName string) *ChangeDisplayNameCmd {
	return identitymodel.NewChangeDisplayNameCmd(aggID, displayName)
}

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func NewDeleteUserCmd(aggID id.StreamID, reason string) *DeleteUserCmd {
	return identitymodel.NewDeleteUserCmd(aggID, reason)
}

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func NewAddCredentialCmd(aggID id.StreamID, cred WebAuthnCredential) *AddCredentialCmd {
	return identitymodel.NewAddCredentialCmd(aggID, cred)
}

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func NewRemoveCredentialCmd(aggID id.StreamID, credID []byte) *RemoveCredentialCmd {
	return identitymodel.NewRemoveCredentialCmd(aggID, credID)
}

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func NewVerifyEmailCmd(aggID id.StreamID) *VerifyEmailCmd {
	return identitymodel.NewVerifyEmailCmd(aggID)
}

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func NewEnableTOTPCmd(aggID id.StreamID, secret []byte) *EnableTOTPCmd {
	return identitymodel.NewEnableTOTPCmd(aggID, secret)
}

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func NewDisableTOTPCmd(aggID id.StreamID) *DisableTOTPCmd {
	return identitymodel.NewDisableTOTPCmd(aggID)
}

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func NewLinkExternalAccountCmd(
	aggID id.StreamID, provider, subject, email, displayName string,
) *LinkExternalAccountCmd {
	return identitymodel.NewLinkExternalAccountCmd(aggID, provider, subject, email, displayName)
}

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func NewUnlinkExternalAccountCmd(aggID id.StreamID, provider, subject string) *UnlinkExternalAccountCmd {
	return identitymodel.NewUnlinkExternalAccountCmd(aggID, provider, subject)
}

func mustCommand(cmdType command.Type, aggID id.StreamID) *command.BasicCommand {
	base, err := command.New(cmdType, aggID)
	if err != nil {
		panic(fmt.Sprintf("mustCommand(%q, %s): %v", cmdType, aggID, err))
	}
	return base
}
