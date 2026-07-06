package usermgmt

import (
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	errorfamily "github.com/larsartmann/go-error-family"
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

// IsValid enforces struct-level invariants that cannot be violated by any
// sequence of events: a deleted tenant is never suspended; SuspendReason is
// only set when Suspended; DeleteReason is only set when Deleted.
func (s TenantState) IsValid() bool {
	if s.Deleted && s.Suspended {
		return false
	}
	if !s.Suspended && s.SuspendReason != "" {
		return false
	}
	if !s.Deleted && s.DeleteReason != "" {
		return false
	}
	return true
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
