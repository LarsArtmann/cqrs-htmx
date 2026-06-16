package usermgmt

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/decider/v2"
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
		return fmt.Errorf("register %s: %w", cmdRegisterUser, err)
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
		return fmt.Errorf("register %s: %w", cmdUpdateRoles, err)
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
		return fmt.Errorf("register %s: %w", cmdChangeEmail, err)
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
		return fmt.Errorf("register %s: %w", cmdChangeDisplayName, err)
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
		return fmt.Errorf("register %s: %w", cmdDeleteUser, err)
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
		return fmt.Errorf("register %s: %w", cmdAddCredential, err)
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
		return fmt.Errorf("register %s: %w", cmdRemoveCredential, err)
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
		return fmt.Errorf("register %s: %w", cmdVerifyEmail, err)
	}

	return nil
}
