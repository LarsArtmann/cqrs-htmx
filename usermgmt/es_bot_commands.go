package usermgmt

import (
	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

// RegisterBotCmd registers a new bot with an API token.
type RegisterBotCmd struct {
	aggregateID id.AggregateID
	cmdID       id.CommandID
	name        string
	ownerID     UserID
	tokenHash   []byte
	scopes      []string
}

func NewRegisterBotCmd(
	aggID id.AggregateID, name string, ownerID UserID, tokenHash []byte, scopes []string,
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
func (c *RegisterBotCmd) ID() id.CommandID            { return c.cmdID }

// DeleteBotCmd permanently deletes a bot.
type DeleteBotCmd struct {
	aggregateID id.AggregateID
	cmdID       id.CommandID
	reason      string
}

func NewDeleteBotCmd(aggID id.AggregateID, reason string) *DeleteBotCmd {
	return &DeleteBotCmd{aggregateID: aggID, cmdID: id.NewCommandID(), reason: reason}
}

func (c *DeleteBotCmd) Type() command.Type          { return cmdDeleteBot }
func (c *DeleteBotCmd) AggregateID() id.AggregateID { return c.aggregateID }
func (c *DeleteBotCmd) ID() id.CommandID            { return c.cmdID }
