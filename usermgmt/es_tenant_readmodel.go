package usermgmt

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
	errorfamily "github.com/larsartmann/go-error-family"
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
t//cqrs-lint:ignore(C035) protected by embedded readModelCore.mu (sync.RWMutex)
	//cqrs-lint:ignore(P011) bounded by finite tenant count; in-memory dev/test default
type TenantReadModel struct {
	readModelCore[*TenantReadModel]
	tenants map[id.StreamID]*Tenant
}

// NewTenantReadModel creates an empty TenantReadModel.
func NewTenantReadModel() *TenantReadModel {
	return &TenantReadModel{
		readModelCore: readModelCore[*TenantReadModel]{
			handlers: map[event.Type]eventHandler[*TenantReadModel]{
				eventTenantCreated:     (*TenantReadModel).handleTenantCreated,
				eventTenantSuspended:   (*TenantReadModel).handleTenantSuspended,
				eventTenantReactivated: (*TenantReadModel).handleTenantReactivated,
				eventTenantDeleted:     (*TenantReadModel).handleTenantDeleted,
			},
		},
		tenants: make(map[id.StreamID]*Tenant),
	}
}

func (m *TenantReadModel) Name() string { return "tenant-read-model" }

func (m *TenantReadModel) EventTypes() []event.Type { return allTenantEventTypes }

func (m *TenantReadModel) Handle(_ context.Context, evt event.Event) error {
	return m.handleEvent(m, evt)
}

func (m *TenantReadModel) handleTenantCreated(aggID id.StreamID, evt event.Event) error {
	p, err := unmarshalPayload[TenantCreatedPayload](evt)
	if err != nil {
		return errorfamily.WrapCorruption(
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
	return nil
}

func (m *TenantReadModel) handleTenantSuspended(aggID id.StreamID, _ event.Event) error {
	if t, ok := m.tenants[aggID]; ok {
		t.Suspended = true
	}
	return nil
}

func (m *TenantReadModel) handleTenantReactivated(aggID id.StreamID, _ event.Event) error {
	if t, ok := m.tenants[aggID]; ok {
		t.Suspended = false
	}
	return nil
}

func (m *TenantReadModel) handleTenantDeleted(aggID id.StreamID, _ event.Event) error {
	delete(m.tenants, aggID)
	return nil
}

// FindByID returns the tenant for the given aggregate ID.
func (m *TenantReadModel) FindByID(aggID id.StreamID) (*Tenant, bool) {
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
