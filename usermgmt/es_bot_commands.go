package usermgmt

import (
	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

// RegisterBotCmd registers a new bot with an API token.
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

// DeleteBotCmd permanently deletes a bot.
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
