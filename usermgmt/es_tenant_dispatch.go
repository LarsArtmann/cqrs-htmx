package usermgmt

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// RegisterTenantCommands wires all tenant aggregate commands to the dispatcher.
//
//nolint:funlen // mirrors RegisterCommands pattern; 4 command registrations
func RegisterTenantCommands(
	dispatcher *command.Dispatcher,
	repo *decider.Repository[TenantState],
) error {
	if err := command.RegisterTyped(
		dispatcher, cmdCreateTenant,
		func(ctx context.Context, c *CreateTenantCmd) error {
			return repo.Execute(
				ctx, c.StreamID(), aggregateTypeTenant,
				decideCreateTenant(c.StreamID(), c.name, c.displayName),
			)
		},
	); err != nil {
		return errorfamily.Wrapf(
			err, event.Infrastructure,
			"usermgmt.dispatch.register_failed",
			"register %s", cmdCreateTenant,
		)
	}

	if err := command.RegisterTyped(
		dispatcher, cmdSuspendTenant,
		func(ctx context.Context, c *SuspendTenantCmd) error {
			return repo.Execute(
				ctx, c.StreamID(), aggregateTypeTenant,
				decideSuspendTenant(c.StreamID(), c.reason),
			)
		},
	); err != nil {
		return errorfamily.Wrapf(
			err, event.Infrastructure,
			"usermgmt.dispatch.register_failed",
			"register %s", cmdSuspendTenant,
		)
	}

	if err := command.RegisterTyped(
		dispatcher, cmdReactivateTenant,
		func(ctx context.Context, c *ReactivateTenantCmd) error {
			return repo.Execute(
				ctx, c.StreamID(), aggregateTypeTenant,
				decideReactivateTenant(c.StreamID()),
			)
		},
	); err != nil {
		return errorfamily.Wrapf(
			err, event.Infrastructure,
			"usermgmt.dispatch.register_failed",
			"register %s", cmdReactivateTenant,
		)
	}

	if err := command.RegisterTyped(
		dispatcher, cmdDeleteTenant,
		func(ctx context.Context, c *DeleteTenantCmd) error {
			return repo.Execute(
				ctx, c.StreamID(), aggregateTypeTenant,
				decideDeleteTenant(c.StreamID(), c.reason),
			)
		},
	); err != nil {
		return errorfamily.Wrapf(
			err, event.Infrastructure,
			"usermgmt.dispatch.register_failed",
			"register %s", cmdDeleteTenant,
		)
	}

	return nil
}
