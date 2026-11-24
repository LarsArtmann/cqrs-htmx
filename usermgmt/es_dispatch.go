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

// RegisterCommands wires all user aggregate commands to the dispatcher.
// Returns an error if any command fails to register.
func RegisterCommands(
	dispatcher *command.Dispatcher,
	repo *decider.Repository[UserState],
) error {
	if err := command.RegisterTyped(
		dispatcher, cmdRegisterUser,
		func(ctx context.Context, c *RegisterUserCmd) error {
			return repo.ExecuteRef(
				ctx, id.NewStreamRef(aggregateTypeUser, c.StreamID()),
				decideRegisterUser(c.StreamID(), c.Email(), c.DisplayName(), c.Roles()),
			)
		},
	); err != nil {
		return errorfamily.Wrapf(
			err,
			event.Infrastructure,
			"usermgmt.dispatch.register_failed",
			"register %s",
			cmdRegisterUser,
		)
	}

	if err := command.RegisterTyped(
		dispatcher, cmdChangeEmail,
		func(ctx context.Context, c *ChangeEmailCmd) error {
			return repo.ExecuteRef(
				ctx, id.NewStreamRef(aggregateTypeUser, c.StreamID()),
				decideChangeEmail(c.StreamID(), c.Email()),
			)
		},
	); err != nil {
		return errorfamily.Wrapf(
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
			return repo.ExecuteRef(
				ctx, id.NewStreamRef(aggregateTypeUser, c.StreamID()),
				decideChangeDisplayName(c.StreamID(), c.DisplayName()),
			)
		},
	); err != nil {
		return errorfamily.Wrapf(
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
			return repo.ExecuteRef(
				ctx, id.NewStreamRef(aggregateTypeUser, c.StreamID()),
				decideDeleteUser(c.StreamID(), c.Reason()),
			)
		},
	); err != nil {
		return errorfamily.Wrapf(
			err,
			event.Infrastructure,
			"usermgmt.dispatch.register_failed",
			"register %s",
			cmdDeleteUser,
		)
	}

	if err := command.RegisterTyped(
		dispatcher, cmdAddCredential,
		func(ctx context.Context, c *AddCredentialCmd) error {
			return repo.ExecuteRef(
				ctx, id.NewStreamRef(aggregateTypeUser, c.StreamID()),
				decideAddCredential(c.StreamID(), c.Credential()),
			)
		},
	); err != nil {
		return errorfamily.Wrapf(
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
			return repo.ExecuteRef(
				ctx, id.NewStreamRef(aggregateTypeUser, c.StreamID()),
				decideRemoveCredential(c.StreamID(), c.CredentialID()),
			)
		},
	); err != nil {
		return errorfamily.Wrapf(
			err,
			event.Infrastructure,
			"usermgmt.dispatch.register_failed",
			"register %s",
			cmdRemoveCredential,
		)
	}

	if err := command.RegisterTyped(
		dispatcher, cmdRecordCredentialUse,
		func(ctx context.Context, c *RecordCredentialUseCmd) error {
			return repo.ExecuteRef(
				ctx, id.NewStreamRef(aggregateTypeUser, c.StreamID()),
				decideRecordCredentialUse(c.StreamID(), c.Use()),
			)
		},
	); err != nil {
		return errorfamily.Wrapf(
			err,
			event.Infrastructure,
			"usermgmt.dispatch.register_failed",
			"register %s",
			cmdRecordCredentialUse,
		)
	}

	if err := command.RegisterTyped(
		dispatcher, cmdVerifyEmail,
		func(ctx context.Context, c *VerifyEmailCmd) error {
			return repo.ExecuteRef(
				ctx, id.NewStreamRef(aggregateTypeUser, c.StreamID()),
				decideVerifyEmail(c.StreamID()),
			)
		},
	); err != nil {
		return errorfamily.Wrapf(
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
			return repo.ExecuteRef(
				ctx, id.NewStreamRef(aggregateTypeUser, c.StreamID()),
				decideEnableTOTP(c.StreamID(), c.Secret()),
			)
		},
	); err != nil {
		return errorfamily.Wrapf(
			err,
			event.Infrastructure,
			"usermgmt.dispatch.register_failed",
			"register %s",
			cmdEnableTOTP,
		)
	}

	if err := command.RegisterTyped(
		dispatcher, cmdDisableTOTP,
		func(ctx context.Context, c *DisableTOTPCmd) error {
			return repo.ExecuteRef(
				ctx, id.NewStreamRef(aggregateTypeUser, c.StreamID()),
				decideDisableTOTP(c.StreamID()),
			)
		},
	); err != nil {
		return errorfamily.Wrapf(
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
			return repo.ExecuteRef(
				ctx, id.NewStreamRef(aggregateTypeUser, c.StreamID()),
				decideLinkExternalAccount(
					c.StreamID(), c.Provider(), c.Subject(), c.Email(), c.DisplayName(),
				),
			)
		},
	); err != nil {
		return errorfamily.Wrapf(
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
			return repo.ExecuteRef(
				ctx, id.NewStreamRef(aggregateTypeUser, c.StreamID()),
				decideUnlinkExternalAccount(c.StreamID(), c.Provider(), c.Subject()),
			)
		},
	); err != nil {
		return errorfamily.Wrapf(
			err,
			event.Infrastructure,
			"usermgmt.dispatch.register_failed",
			"register %s",
			cmdUnlinkExternalAccount,
		)
	}

	return registerMergeCommands(dispatcher, repo)
}

// registerMergeCommands wires the two identity-merge commands. It is split
// from RegisterCommands so that function stays inside the maintainability
// budget; a failure is wrapped like every other registration failure.
func registerMergeCommands(
	dispatcher *command.Dispatcher,
	repo *decider.Repository[UserState],
) error {
	if err := command.RegisterTyped(
		dispatcher, cmdMergeUsers,
		func(ctx context.Context, c *identitymodel.MergeUsersCmd) error {
			return repo.ExecuteRef(
				ctx, id.NewStreamRef(aggregateTypeUser, c.StreamID()),
				decideMergeUsers(c.StreamID(), c.Contributions(), c.Reason()),
			)
		},
	); err != nil {
		return errorfamily.Wrapf(
			err,
			event.Infrastructure,
			"usermgmt.dispatch.register_failed",
			"register %s",
			cmdMergeUsers,
		)
	}

	if err := command.RegisterTyped(
		dispatcher, cmdMergeUserInto,
		func(ctx context.Context, c *identitymodel.MergeUserIntoCmd) error {
			return repo.ExecuteRef(
				ctx, id.NewStreamRef(aggregateTypeUser, c.StreamID()),
				decideMergeUserInto(c.StreamID(), c.SurvivorID(), c.Reason()),
			)
		},
	); err != nil {
		return errorfamily.Wrapf(
			err,
			event.Infrastructure,
			"usermgmt.dispatch.register_failed",
			"register %s",
			cmdMergeUserInto,
		)
	}

	return nil
}
