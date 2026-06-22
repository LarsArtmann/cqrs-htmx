package usermgmt

import (
	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// TenantState is the aggregate state for a Tenant, reconstructed by folding events.
type TenantState struct {
	Name          string
	DisplayName   string
	Suspended     bool
	SuspendReason string
	Deleted       bool
	DeleteReason  string
}

// Exists reports whether the tenant has been created (has at least one event).
func (s TenantState) Exists() bool {
	return s.Name != "" && !s.Deleted
}

// IsActive reports whether the tenant exists and is not suspended or deleted.
func (s TenantState) IsActive() bool {
	return s.Exists() && !s.Suspended
}

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
		next.DeleteReason = p.Reason

	default:
		return state, event.NewRejection(
			"usermgmt.tenant.unknown_event",
			"foldTenant received unknown event type: "+string(evt.Type()),
		)
	}

	return next, nil
}
