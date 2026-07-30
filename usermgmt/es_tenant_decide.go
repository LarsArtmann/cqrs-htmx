package usermgmt

import (
	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// TenantDecider returns the Decider for the Tenant aggregate.
func TenantDecider() decider.Decider[TenantState] {
	return decider.Decider[TenantState]{
		Initial: TenantState{}, //nolint:exhaustruct // zero-value is correct for aggregate initial state
		Apply:   foldTenant,
	}
}

func decideCreateTenant(
	aggID id.StreamID,
	name, displayName string,
) func(TenantState, event.Version) ([]event.Event, error) {
	return func(state TenantState, version event.Version) ([]event.Event, error) {
		if state.Name != "" {
			return nil, errorfamily.NewConflict(
				"usermgmt.tenant.already_exists",
				"tenant with this ID already exists",
			)
		}
		if name == "" {
			return nil, errorfamily.NewRejection(
				"usermgmt.tenant.name_required",
				"tenant name is required",
			)
		}
		evt, err := event.New(
			eventTenantCreated, aggID, aggregateTypeTenant, version.Increment(),
			TenantCreatedPayload{
				SchemaVersion: currentSchemaVersion,
				Name:          name,
				DisplayName:   displayName,
			},
			event.WithCodec(codec.JSONCodec{}),
		)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.tenant.event_failed",
				"create TenantCreated event",
			)
		}
		return []event.Event{evt}, nil
	}
}

func decideSuspendTenant(
	aggID id.StreamID,
	reason string,
) func(TenantState, event.Version) ([]event.Event, error) {
	return func(state TenantState, version event.Version) ([]event.Event, error) {
		if !state.Exists() {
			return nil, errorfamily.NewRejection(
				"usermgmt.tenant_suspend.not_found",
				"tenant does not exist",
			)
		}
		if state.Suspended {
			return nil, nil
		}
		evt, err := event.New(
			eventTenantSuspended, aggID, aggregateTypeTenant, version.Increment(),
			TenantSuspendedPayload{
				SchemaVersion: currentSchemaVersion,
				Reason:        reason,
			},
			event.WithCodec(codec.JSONCodec{}),
		)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.tenant_suspend.event_failed",
				"create TenantSuspended event",
			)
		}
		return []event.Event{evt}, nil
	}
}

func decideReactivateTenant(
	aggID id.StreamID,
) func(TenantState, event.Version) ([]event.Event, error) {
	return func(state TenantState, version event.Version) ([]event.Event, error) {
		if !state.Exists() {
			return nil, errorfamily.NewRejection(
				"usermgmt.tenant_reactivate.not_found",
				"tenant does not exist",
			)
		}
		if !state.Suspended {
			return nil, nil
		}
		evt, err := event.New(
			eventTenantReactivated, aggID, aggregateTypeTenant, version.Increment(),
			TenantReactivatedPayload{
				SchemaVersion: currentSchemaVersion,
			},
			event.WithCodec(codec.JSONCodec{}),
		)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.tenant_reactivate.event_failed",
				"create TenantReactivated event",
			)
		}
		return []event.Event{evt}, nil
	}
}

//nolint:dupl // mirrors decideDeleteBot; tombstone deletion is structurally identical
func decideDeleteTenant(
	aggID id.StreamID,
	reason string,
) func(TenantState, event.Version) ([]event.Event, error) {
	return func(state TenantState, version event.Version) ([]event.Event, error) {
		if !state.Exists() {
			return nil, errorfamily.NewRejection(
				"usermgmt.tenant_delete.not_found",
				"tenant does not exist",
			)
		}
		if state.Deleted {
			return nil, errorfamily.NewRejection(
				"usermgmt.tenant_delete.already_deleted",
				"tenant is already deleted",
			)
		}
		evt, err := event.New(
			eventTenantDeleted, aggID, aggregateTypeTenant, version.Increment(),
			TenantDeletedPayload{
				SchemaVersion: currentSchemaVersion,
				Reason:        reason,
			},
			event.WithCodec(codec.JSONCodec{}),
		)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.tenant_delete.event_failed",
				"create TenantDeleted event",
			)
		}
		marked, markErr := event.MarkTombstone(evt)
		if markErr != nil {
			return nil, errorfamily.WrapInfrastructure(
				markErr,
				"usermgmt.tenant_delete.tombstone_failed",
				"mark tenant tombstone",
			)
		}
		return []event.Event{marked}, nil
	}
}
