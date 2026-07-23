package usermgmt

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
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
				ctx, c.StreamID(), aggregateTypeBot,
				decideRegisterBot(
					c.StreamID(), c.name, c.ownerID, c.tokenHash, c.scopes,
				),
			)
		},
	); err != nil {
		return errorfamily.Wrapf(
			err, event.Infrastructure,
			"usermgmt.dispatch.register_failed",
			"register %s", cmdRegisterBot,
		)
	}

	if err := command.RegisterTyped(
		dispatcher, cmdDeleteBot,
		func(ctx context.Context, c *DeleteBotCmd) error {
			return repo.Execute(
				ctx, c.StreamID(), aggregateTypeBot,
				decideDeleteBot(c.StreamID(), c.reason),
			)
		},
	); err != nil {
		return errorfamily.Wrapf(
			err, event.Infrastructure,
			"usermgmt.dispatch.register_failed",
			"register %s", cmdDeleteBot,
		)
	}

	return nil
}
