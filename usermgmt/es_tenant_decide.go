package usermgmt

import (
	"github.com/larsartmann/go-cqrs-lite/decider/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
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
	aggID id.AggregateID,
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
		payload, err := marshalPayload(TenantCreatedPayload{
			SchemaVersion: currentSchemaVersion,
			Name:          name,
			DisplayName:   displayName,
		})
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.tenant.marshal_failed",
				"marshal TenantCreated payload",
			)
		}
		evt, err := event.NewEvent(
			eventTenantCreated, aggID, aggregateTypeTenant, version.Increment(),
			payload,
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
	aggID id.AggregateID,
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
		payload, err := marshalPayload(TenantSuspendedPayload{
			SchemaVersion: currentSchemaVersion,
			Reason:        reason,
		})
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.tenant_suspend.marshal_failed",
				"marshal TenantSuspended payload",
			)
		}
		evt, err := event.NewEvent(
			eventTenantSuspended, aggID, aggregateTypeTenant, version.Increment(),
			payload,
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
	aggID id.AggregateID,
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
		payload, err := marshalPayload(TenantReactivatedPayload{
			SchemaVersion: currentSchemaVersion,
		})
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.tenant_reactivate.marshal_failed",
				"marshal TenantReactivated payload",
			)
		}
		evt, err := event.NewEvent(
			eventTenantReactivated, aggID, aggregateTypeTenant, version.Increment(),
			payload,
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
	aggID id.AggregateID,
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
		payload, err := marshalPayload(TenantDeletedPayload{
			SchemaVersion: currentSchemaVersion,
			Reason:        reason,
		})
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.tenant_delete.marshal_failed",
				"marshal TenantDeleted payload",
			)
		}
		evt, err := event.NewEvent(
			eventTenantDeleted, aggID, aggregateTypeTenant, version.Increment(),
			payload,
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
