package usermgmt

import (
	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

// RegisterBotCmd registers a new bot with an API token.
type RegisterBotCmd struct {
	aggregateID id.AggregateID
	name        string
	ownerID     string
	tokenHash   []byte
	scopes      []string
}

func NewRegisterBotCmd(
	aggID id.AggregateID, name, ownerID string, tokenHash []byte, scopes []string,
) *RegisterBotCmd {
	return &RegisterBotCmd{
		aggregateID: aggID,
		name:        name,
		ownerID:     ownerID,
		tokenHash:   tokenHash,
		scopes:      scopes,
	}
}

func (c *RegisterBotCmd) Type() command.Type          { return cmdRegisterBot }
func (c *RegisterBotCmd) AggregateID() id.AggregateID { return c.aggregateID }

// DeleteBotCmd permanently deletes a bot.
type DeleteBotCmd struct {
	aggregateID id.AggregateID
	reason      string
}

func NewDeleteBotCmd(aggID id.AggregateID, reason string) *DeleteBotCmd {
	return &DeleteBotCmd{aggregateID: aggID, reason: reason}
}

func (c *DeleteBotCmd) Type() command.Type          { return cmdDeleteBot }
func (c *DeleteBotCmd) AggregateID() id.AggregateID { return c.aggregateID }
