package usermgmt

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/decider/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// RegisterCommands wires all user aggregate commands to the dispatcher.
// Returns an error if any command fails to register.
func RegisterCommands(
	dispatcher *command.Dispatcher,
	repo *decider.Repository[UserState],
) error {
	if err := command.RegisterTyped(
		dispatcher, cmdRegisterUser,
		func(ctx context.Context, c *RegisterUserCmd) error {
			return repo.Execute(
				ctx, c.AggregateID(), aggregateTypeUser,
				decideRegisterUser(c.AggregateID(), c.email, c.displayName, c.roles),
			)
		},
	); err != nil {
		return event.Wrapf(
			err,
			event.Infrastructure,
			"usermgmt.dispatch.register_failed",
			"register %s",
			cmdRegisterUser,
		)
	}

	if err := command.RegisterTyped(
		dispatcher, cmdUpdateRoles,
		func(ctx context.Context, c *UpdateRolesCmd) error {
			return repo.Execute(
				ctx, c.AggregateID(), aggregateTypeUser,
				decideUpdateRoles(c.AggregateID(), c.roles, c.domain),
			)
		},
	); err != nil {
		return event.Wrapf(
			err,
			event.Infrastructure,
			"usermgmt.dispatch.register_failed",
			"register %s",
			cmdUpdateRoles,
		)
	}

	if err := command.RegisterTyped(
		dispatcher, cmdChangeEmail,
		func(ctx context.Context, c *ChangeEmailCmd) error {
			return repo.Execute(
				ctx, c.AggregateID(), aggregateTypeUser,
				decideChangeEmail(c.AggregateID(), c.email),
			)
		},
	); err != nil {
		return event.Wrapf(
			err,
			event.Infrastructure,
			"usermgmt.dispatch.register_failed",
			"register %s",
			cmdChangeEmail,
		)
	}

	if err := command.RegisterTyped(
		dispatcher, cmdChangeDisplayName,
		func(ctx context.Context, c *ChangeDisplayNameCmd) error {
			return repo.Execute(
				ctx, c.AggregateID(), aggregateTypeUser,
				decideChangeDisplayName(c.AggregateID(), c.displayName),
			)
		},
	); err != nil {
		return event.Wrapf(
			err,
			event.Infrastructure,
			"usermgmt.dispatch.register_failed",
			"register %s",
			cmdChangeDisplayName,
		)
	}

	if err := command.RegisterTyped(
		dispatcher, cmdDeleteUser,
		func(ctx context.Context, c *DeleteUserCmd) error {
			return repo.Execute(
				ctx, c.AggregateID(), aggregateTypeUser,
				decideDeleteUser(c.AggregateID(), c.reason),
			)
		},
	); err != nil {
		return event.Wrapf(err, event.Infrastructure, "usermgmt.dispatch.register_failed", "register %s", cmdDeleteUser)
	}

	if err := command.RegisterTyped(
		dispatcher, cmdAddCredential,
		func(ctx context.Context, c *AddCredentialCmd) error {
			return repo.Execute(
				ctx, c.AggregateID(), aggregateTypeUser,
				decideAddCredential(c.AggregateID(), c.credential),
			)
		},
	); err != nil {
		return event.Wrapf(
			err,
			event.Infrastructure,
			"usermgmt.dispatch.register_failed",
			"register %s",
			cmdAddCredential,
		)
	}

	if err := command.RegisterTyped(
		dispatcher, cmdRemoveCredential,
		func(ctx context.Context, c *RemoveCredentialCmd) error {
			return repo.Execute(
				ctx, c.AggregateID(), aggregateTypeUser,
				decideRemoveCredential(c.AggregateID(), c.credentialID),
			)
		},
	); err != nil {
		return event.Wrapf(
			err,
			event.Infrastructure,
			"usermgmt.dispatch.register_failed",
			"register %s",
			cmdRemoveCredential,
		)
	}

	if err := command.RegisterTyped(
		dispatcher, cmdVerifyEmail,
		func(ctx context.Context, c *VerifyEmailCmd) error {
			return repo.Execute(
				ctx, c.AggregateID(), aggregateTypeUser,
				decideVerifyEmail(c.AggregateID()),
			)
		},
	); err != nil {
		return event.Wrapf(
			err,
			event.Infrastructure,
			"usermgmt.dispatch.register_failed",
			"register %s",
			cmdVerifyEmail,
		)
	}

	if err := command.RegisterTyped(
		dispatcher, cmdEnableTOTP,
		func(ctx context.Context, c *EnableTOTPCmd) error {
			return repo.Execute(
				ctx, c.AggregateID(), aggregateTypeUser,
				decideEnableTOTP(c.AggregateID(), c.secret),
			)
		},
	); err != nil {
		return event.Wrapf(err, event.Infrastructure, "usermgmt.dispatch.register_failed", "register %s", cmdEnableTOTP)
	}

	if err := command.RegisterTyped(
		dispatcher, cmdDisableTOTP,
		func(ctx context.Context, c *DisableTOTPCmd) error {
			return repo.Execute(
				ctx, c.AggregateID(), aggregateTypeUser,
				decideDisableTOTP(c.AggregateID()),
			)
		},
	); err != nil {
		return event.Wrapf(
			err,
			event.Infrastructure,
			"usermgmt.dispatch.register_failed",
			"register %s",
			cmdDisableTOTP,
		)
	}

	if err := command.RegisterTyped(
		dispatcher, cmdLinkExternalAccount,
		func(ctx context.Context, c *LinkExternalAccountCmd) error {
			return repo.Execute(
				ctx, c.AggregateID(), aggregateTypeUser,
				decideLinkExternalAccount(
					c.AggregateID(), c.provider, c.subject, c.email, c.displayName,
				),
			)
		},
	); err != nil {
		return event.Wrapf(
			err,
			event.Infrastructure,
			"usermgmt.dispatch.register_failed",
			"register %s",
			cmdLinkExternalAccount,
		)
	}

	if err := command.RegisterTyped(
		dispatcher, cmdUnlinkExternalAccount,
		func(ctx context.Context, c *UnlinkExternalAccountCmd) error {
			return repo.Execute(
				ctx, c.AggregateID(), aggregateTypeUser,
				decideUnlinkExternalAccount(c.AggregateID(), c.provider, c.subject),
			)
		},
	); err != nil {
		return event.Wrapf(
			err,
			event.Infrastructure,
			"usermgmt.dispatch.register_failed",
			"register %s",
			cmdUnlinkExternalAccount,
		)
	}

	return nil
}
