package usermgmt

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/decider/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
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
				ctx, c.AggregateID(), aggregateTypeTenant,
				decideCreateTenant(c.AggregateID(), c.name, c.displayName),
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
				ctx, c.AggregateID(), aggregateTypeTenant,
				decideSuspendTenant(c.AggregateID(), c.reason),
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
				ctx, c.AggregateID(), aggregateTypeTenant,
				decideReactivateTenant(c.AggregateID()),
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
				ctx, c.AggregateID(), aggregateTypeTenant,
				decideDeleteTenant(c.AggregateID(), c.reason),
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
