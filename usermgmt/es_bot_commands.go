package usermgmt

import (
	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

type (
	RegisterBotCmd = identitymodel.RegisterBotCmd
	DeleteBotCmd   = identitymodel.DeleteBotCmd
)

func NewRegisterBotCmd(
	aggID id.StreamID,
	name string,
	ownerID UserID,
	tokenHash []byte,
	scopes []string,
) *RegisterBotCmd {
	return identitymodel.NewRegisterBotCmd(aggID, name, ownerID, tokenHash, scopes)
}

func NewDeleteBotCmd(aggID id.StreamID, reason string) *DeleteBotCmd {
	return identitymodel.NewDeleteBotCmd(aggID, reason)
}
