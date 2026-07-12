package usermgmt

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// MigrateRolesToMemberships converts legacy UserRegistered and RolesUpdated
// events into Membership aggregate commands. This is an opt-in migration tool
// for consumers upgrading from pre-membership event stores.
//
// For each UserRegistered event with roles, it dispatches an AddMember command
// using the user's own ID as the tenant domain (matching the legacy behavior
// where Casbin used subject=domain for self-scoped roles).
//
// For each RolesUpdated event, it dispatches an UpdateMemberRoles command
// to sync the membership with the new role set.
//
// Idempotency: AddMember returns a Conflict error if the membership already
// exists — these are silently skipped. UpdateMemberRoles is idempotent by
// nature (it replaces the full role set).
//
// Usage:
//
//	events := readAllEvents(myStore)
//	err := MigrateRolesToMemberships(ctx, events, dispatcher)
func MigrateRolesToMemberships(
	ctx context.Context,
	events []event.Event,
	dispatcher *command.Dispatcher,
) error {
	for _, evt := range events {
		if err := migrateEvent(ctx, evt, dispatcher); err != nil {
			return errorfamily.Wrapf(
				err, event.Infrastructure,
				"usermgmt.migration.event_failed",
				"migrate event %s for aggregate %s",
				evt.Type(), evt.AggregateID(),
			)
		}
	}
	return nil
}

func migrateEvent(ctx context.Context, evt event.Event, dispatcher *command.Dispatcher) error {
	switch evt.Type() {
	case eventUserRegistered:
		p, err := unmarshalPayload[UserRegisteredPayload](evt)
		if err != nil {
			return err
		}
		if len(p.Roles) == 0 {
			return nil
		}
		userID := evt.AggregateID().String()
		actor := ActorIDFromUser(NewUserID(userID))
		tenant := NewTenantID(userID) // legacy: self-scoped domain
		cmd := NewAddMemberCmd(actor, tenant, p.Roles)
		err = dispatcher.Dispatch(ctx, cmd)
		if err != nil && !isConflict(err) {
			return err //nolint:wrapcheck // conflict = already migrated, other errors are typed
		}
		return nil

	case eventRolesUpdated:
		p, err := unmarshalPayload[RolesUpdatedPayload](evt)
		if err != nil {
			return err
		}
		userID := evt.AggregateID().String()
		actor := ActorIDFromUser(NewUserID(userID))
		domain := p.Domain
		if domain == "" {
			domain = userID
		}
		tenant := NewTenantID(domain)
		cmd := NewUpdateMemberRolesCmd(actor, tenant, p.Roles)
		return dispatcher.Dispatch(ctx, cmd) //nolint:wrapcheck // idempotent replace

	default:
		return nil
	}
}

func isConflict(err error) bool {
	return errorfamily.Classify(err) == event.Conflict
}
