package usermgmt

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/decider/v2"
)

func RegisterCommands(
	dispatcher *command.Dispatcher,
	repo *decider.Repository[UserState],
) {
	_ = command.RegisterTyped(
		dispatcher, cmdRegisterUser,
		func(ctx context.Context, c *RegisterUserCmd) error {
			return repo.Execute(
				ctx, c.AggregateID(), aggregateTypeUser,
				decideRegisterUser(c.AggregateID(), c.email, c.displayName, c.roles),
			)
		},
	)

	_ = command.RegisterTyped(
		dispatcher, cmdUpdateRoles,
		func(ctx context.Context, c *UpdateRolesCmd) error {
			return repo.Execute(
				ctx, c.AggregateID(), aggregateTypeUser,
				decideUpdateRoles(c.AggregateID(), c.roles, c.domain),
			)
		},
	)

	_ = command.RegisterTyped(
		dispatcher, cmdChangeEmail,
		func(ctx context.Context, c *ChangeEmailCmd) error {
			return repo.Execute(
				ctx, c.AggregateID(), aggregateTypeUser,
				decideChangeEmail(c.AggregateID(), c.email),
			)
		},
	)

	_ = command.RegisterTyped(
		dispatcher, cmdChangeDisplayName,
		func(ctx context.Context, c *ChangeDisplayNameCmd) error {
			return repo.Execute(
				ctx, c.AggregateID(), aggregateTypeUser,
				decideChangeDisplayName(c.AggregateID(), c.displayName),
			)
		},
	)

	_ = command.RegisterTyped(
		dispatcher, cmdDeleteUser,
		func(ctx context.Context, c *DeleteUserCmd) error {
			return repo.Execute(
				ctx, c.AggregateID(), aggregateTypeUser,
				decideDeleteUser(c.AggregateID(), c.reason),
			)
		},
	)

	_ = command.RegisterTyped(
		dispatcher, cmdAddCredential,
		func(ctx context.Context, c *AddCredentialCmd) error {
			return repo.Execute(
				ctx, c.AggregateID(), aggregateTypeUser,
				decideAddCredential(c.AggregateID(), c.credential),
			)
		},
	)

	_ = command.RegisterTyped(
		dispatcher, cmdRemoveCredential,
		func(ctx context.Context, c *RemoveCredentialCmd) error {
			return repo.Execute(
				ctx, c.AggregateID(), aggregateTypeUser,
				decideRemoveCredential(c.AggregateID(), c.credentialID),
			)
		},
	)
}
