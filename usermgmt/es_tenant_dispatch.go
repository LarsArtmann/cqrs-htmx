package usermgmt

import (
	"context"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
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
			return repo.ExecuteRef(
				ctx, id.NewStreamRef(aggregateTypeTenant, c.StreamID()),
				decideCreateTenant(c.StreamID(), c.Name(), c.DisplayName()),
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
			return repo.ExecuteRef(
				ctx, id.NewStreamRef(aggregateTypeTenant, c.StreamID()),
				decideSuspendTenant(c.StreamID(), c.Reason()),
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
			return repo.ExecuteRef(
				ctx, id.NewStreamRef(aggregateTypeTenant, c.StreamID()),
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
			return repo.ExecuteRef(
				ctx, id.NewStreamRef(aggregateTypeTenant, c.StreamID()),
				decideDeleteTenant(c.StreamID(), c.Reason()),
			)
		},
	); err != nil {
		return errorfamily.Wrapf(
			err, event.Infrastructure,
			"usermgmt.dispatch.register_failed",
			"register %s", cmdDeleteTenant,
		)
	}

	if err := command.RegisterTyped(
		dispatcher, cmdRenameTenant,
		func(ctx context.Context, c *identitymodel.RenameTenantCmd) error {
			return repo.ExecuteRef(
				ctx, id.NewStreamRef(aggregateTypeTenant, c.StreamID()),
				decideRenameTenant(c.StreamID(), c.DisplayName()),
			)
		},
	); err != nil {
		return errorfamily.Wrapf(
			err, event.Infrastructure,
			"usermgmt.dispatch.register_failed",
			"register %s", cmdRenameTenant,
		)
	}

	if err := command.RegisterTyped(
		dispatcher, cmdMergeTenants,
		func(ctx context.Context, c *identitymodel.MergeTenantsCmd) error {
			return repo.ExecuteRef(
				ctx, id.NewStreamRef(aggregateTypeTenant, c.StreamID()),
				decideMergeTenants(c.StreamID(), c.LoserIDs(), c.Reason()),
			)
		},
	); err != nil {
		return errorfamily.Wrapf(
			err, event.Infrastructure,
			"usermgmt.dispatch.register_failed",
			"register %s", cmdMergeTenants,
		)
	}

	if err := command.RegisterTyped(
		dispatcher, cmdMergeTenantInto,
		func(ctx context.Context, c *identitymodel.MergeTenantIntoCmd) error {
			return repo.ExecuteRef(
				ctx, id.NewStreamRef(aggregateTypeTenant, c.StreamID()),
				decideMergeTenantInto(c.StreamID(), c.SurvivorID(), c.Reason()),
			)
		},
	); err != nil {
		return errorfamily.Wrapf(
			err, event.Infrastructure,
			"usermgmt.dispatch.register_failed",
			"register %s", cmdMergeTenantInto,
		)
	}

	return nil
}
