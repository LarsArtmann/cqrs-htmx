package usermgmt

import (
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

type TenantState = identitymodel.TenantState

// foldTenant applies an event to the current TenantState, returning the new state.
func foldTenant(state TenantState, evt event.Event) (TenantState, error) {
	next := state

	switch evt.Type() {
	case eventTenantCreated:
		p, err := unmarshalPayload[TenantCreatedPayload](evt)
		if err != nil {
			return state, err
		}
		next.Name = p.Name
		next.DisplayName = p.DisplayName
		next.Suspended = false
		next.Deleted = false

	case eventTenantSuspended:
		p, err := unmarshalPayload[TenantSuspendedPayload](evt)
		if err != nil {
			return state, err
		}
		next.Suspended = true
		next.SuspendReason = p.Reason

	case eventTenantReactivated:
		next.Suspended = false
		next.SuspendReason = ""

	case eventTenantDeleted:
		p, err := unmarshalPayload[TenantDeletedPayload](evt)
		if err != nil {
			return state, err
		}
		next.Deleted = true
		next.Suspended = false
		next.SuspendReason = ""
		next.DeleteReason = p.Reason

	default:
		return state, errorfamily.NewRejection(
			"usermgmt.tenant.unknown_event",
			"foldTenant received unknown event type: "+string(evt.Type()),
		)
	}

	return next, nil
}
