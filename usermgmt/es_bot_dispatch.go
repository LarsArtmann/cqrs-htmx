package usermgmt

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/decider/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// RegisterBotCommands wires all bot aggregate commands to the dispatcher.
func RegisterBotCommands(
	dispatcher *command.Dispatcher,
	repo *decider.Repository[BotState],
) error {
	if err := command.RegisterTyped(
		dispatcher, cmdRegisterBot,
		func(ctx context.Context, c *RegisterBotCmd) error {
			return repo.Execute(
				ctx, c.AggregateID(), aggregateTypeBot,
				decideRegisterBot(
					c.AggregateID(), c.name, c.ownerID, c.tokenHash, c.scopes,
				),
			)
		},
	); err != nil {
		return event.Wrapf(
			err, event.Infrastructure,
			"usermgmt.dispatch.register_failed",
			"register %s", cmdRegisterBot,
		)
	}

	if err := command.RegisterTyped(
		dispatcher, cmdDeleteBot,
		func(ctx context.Context, c *DeleteBotCmd) error {
			return repo.Execute(
				ctx, c.AggregateID(), aggregateTypeBot,
				decideDeleteBot(c.AggregateID(), c.reason),
			)
		},
	); err != nil {
		return event.Wrapf(
			err, event.Infrastructure,
			"usermgmt.dispatch.register_failed",
			"register %s", cmdDeleteBot,
		)
	}

	return nil
}
