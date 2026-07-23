package usermgmt

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// RegisterMembershipCommands wires all membership aggregate commands to the dispatcher.
// Returns an error if any command fails to register.
func RegisterMembershipCommands(
	dispatcher *command.Dispatcher,
	repo *decider.Repository[MembershipState],
) error {
	if err := command.RegisterTyped(
		dispatcher, cmdAddMember,
		func(ctx context.Context, c *AddMemberCmd) error {
			return repo.Execute(
				ctx, c.StreamID(), aggregateTypeMembership,
				decideAddMember(c.StreamID(), c.actorID, c.tenantID, c.roles),
			)
		},
	); err != nil {
		return errorfamily.Wrapf(
			err, event.Infrastructure,
			"usermgmt.dispatch.register_failed",
			"register %s", cmdAddMember,
		)
	}

	if err := command.RegisterTyped(
		dispatcher, cmdUpdateMemberRoles,
		func(ctx context.Context, c *UpdateMemberRolesCmd) error {
			return repo.Execute(
				ctx, c.StreamID(), aggregateTypeMembership,
				decideUpdateMemberRoles(c.StreamID(), c.roles),
			)
		},
	); err != nil {
		return errorfamily.Wrapf(
			err, event.Infrastructure,
			"usermgmt.dispatch.register_failed",
			"register %s", cmdUpdateMemberRoles,
		)
	}

	if err := command.RegisterTyped(
		dispatcher, cmdRemoveMember,
		func(ctx context.Context, c *RemoveMemberCmd) error {
			return repo.Execute(
				ctx, c.StreamID(), aggregateTypeMembership,
				decideRemoveMember(c.StreamID()),
			)
		},
	); err != nil {
		return errorfamily.Wrapf(
			err, event.Infrastructure,
			"usermgmt.dispatch.register_failed",
			"register %s", cmdRemoveMember,
		)
	}

	return nil
}
