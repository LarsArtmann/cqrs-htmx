package usermgmt

import (
	"context"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/projection/v3"
)

// Tenant is the read-model representation of a tenant.
type Tenant struct {
	ID          TenantID `json:"id"`
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Suspended   bool     `json:"suspended"`
	Deleted     bool     `json:"deleted"`
}

// TenantReadModel is the projection-side store for tenants.
type TenantReadModel struct {
	mu      sync.RWMutex
	tenants map[id.AggregateID]*Tenant
}

// NewTenantReadModel creates an empty TenantReadModel.
func NewTenantReadModel() *TenantReadModel {
	return &TenantReadModel{ //nolint:exhaustruct // mu starts zero-valued
		tenants: make(map[id.AggregateID]*Tenant),
	}
}

func (m *TenantReadModel) Name() string { return "tenant-read-model" }

func (m *TenantReadModel) EventTypes() []event.Type { return allTenantEventTypes }

func (m *TenantReadModel) Handle(_ context.Context, evt event.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	aggID := evt.AggregateID()

	switch evt.Type() {
	case eventTenantCreated:
		p, err := unmarshalPayload[TenantCreatedPayload](evt)
		if err != nil {
			return event.WrapCorruption(
				err, "usermgmt.tenant_readmodel.decode_failed",
				"decode TenantCreated in read model",
			)
		}
		m.tenants[aggID] = &Tenant{
			ID:          NewTenantID(aggID.String()),
			Name:        p.Name,
			DisplayName: p.DisplayName,
			Suspended:   false,
			Deleted:     false,
		}

	case eventTenantSuspended:
		if t, ok := m.tenants[aggID]; ok {
			t.Suspended = true
		}

	case eventTenantReactivated:
		if t, ok := m.tenants[aggID]; ok {
			t.Suspended = false
		}

	case eventTenantDeleted:
		delete(m.tenants, aggID)

	default:
		return event.NewRejection(
			"usermgmt.tenant_readmodel.unknown_event",
			"tenant read model received unknown event type: "+string(evt.Type()),
		)
	}

	return nil
}

// FindByID returns the tenant for the given aggregate ID.
func (m *TenantReadModel) FindByID(aggID id.AggregateID) (*Tenant, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.tenants[aggID]
	return t, ok
}

// FindByName returns the first tenant with the given name.
func (m *TenantReadModel) FindByName(name string) (*Tenant, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, t := range m.tenants {
		if t.Name == name {
			return t, true
		}
	}
	return nil, false
}

// All returns all active (non-deleted) tenants.
func (m *TenantReadModel) All() []*Tenant {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*Tenant, 0, len(m.tenants))
	for _, t := range m.tenants {
		result = append(result, t)
	}
	return result
}

var _ projection.Projection = (*TenantReadModel)(nil)
