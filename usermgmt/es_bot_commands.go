package usermgmt

import (
	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

type (
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	RegisterBotCmd = identitymodel.RegisterBotCmd
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	DeleteBotCmd = identitymodel.DeleteBotCmd
)

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func NewRegisterBotCmd(
	aggID id.StreamID,
	name string,
	ownerID UserID,
	tokenHash []byte,
	scopes []string,
) *RegisterBotCmd {
	return identitymodel.NewRegisterBotCmd(aggID, name, ownerID, tokenHash, scopes)
}

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func NewDeleteBotCmd(aggID id.StreamID, reason string) *DeleteBotCmd {
	return identitymodel.NewDeleteBotCmd(aggID, reason)
}
