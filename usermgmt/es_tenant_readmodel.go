package usermgmt

import (
	"context"
	"sync"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// Tenant is the read-model representation of a tenant. MergedInto names the
// surviving tenant when this one was merged away (zero on a live tenant) and
// MergedTenantIDs lists the tenants this one absorbed — the admin panel and
// the duplicates report need both without replaying the journal (D15).
type Tenant struct {
	ID              TenantID   `json:"id"`
	Name            string     `json:"name"`
	DisplayName     string     `json:"display_name"`
	Suspended       bool       `json:"suspended"`
	Deleted         bool       `json:"deleted"`
	MergedInto      TenantID   `json:"merged_into,omitzero"`
	MergedTenantIDs []TenantID `json:"merged_tenant_ids,omitempty"`
}

// TenantReadModel is the projection-side store for tenants.
// cqrs-lint:ignore(C035) protected by embedded readModelCore.mu (sync.RWMutex)
type TenantReadModel struct {
	readModelCore[*TenantReadModel]
	//cqrs-lint:ignore(P011) bounded by finite tenant count; in-memory dev/test default
	tenants map[id.StreamID]*Tenant
	// mergedInto holds the merge pointer of every retired tenant stream: the
	// row itself is gone, only this map keeps its trace (D16).
	mergedInto map[id.StreamID]id.StreamID
}

// NewTenantReadModel creates an empty TenantReadModel.
func NewTenantReadModel() *TenantReadModel {
	return &TenantReadModel{
		mu: sync.RWMutex{},
		handlers: map[event.Type]eventHandler[*TenantReadModel]{
			eventTenantCreated:     (*TenantReadModel).handleTenantCreated,
			eventTenantSuspended:   (*TenantReadModel).handleTenantSuspended,
			eventTenantReactivated: (*TenantReadModel).handleTenantReactivated,
			eventTenantDeleted:     (*TenantReadModel).handleTenantDeleted,
			eventTenantRenamed:     (*TenantReadModel).handleTenantRenamed,
			eventTenantMerged:      (*TenantReadModel).handleTenantMerged,
			eventTenantMergedInto:  (*TenantReadModel).handleTenantMergedInto,
		},
		tenants:    make(map[id.StreamID]*Tenant),
		mergedInto: make(map[id.StreamID]id.StreamID),
	}
}

func (m *TenantReadModel) Name() string { return "tenant-read-model" }

func (m *TenantReadModel) EventTypes() []event.Type { return allTenantEventTypes }

func (m *TenantReadModel) Handle(ctx context.Context, evt event.Event) error {
	return m.handleEvent(m, evt)
}

func (m *TenantReadModel) handleTenantCreated(aggID id.StreamID, evt event.Event) error {
	p, err := unmarshalPayload[TenantCreatedPayload](evt)
	if err != nil {
		return errorfamily.WrapCorruption(
			err, "usermgmt.tenant_readmodel.decode_failed",
			"decode TenantCreated in read model",
		).WithContextAny("agg_id", aggID)
	}
	m.tenants[aggID] = &Tenant{
		ID:              NewTenantID(aggID.Get()),
		Name:            p.Name,
		DisplayName:     p.DisplayName,
		Suspended:       false,
		Deleted:         false,
		MergedInto:      TenantID{},
		MergedTenantIDs: nil,
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

func (m *TenantReadModel) handleTenantRenamed(aggID id.StreamID, evt event.Event) error {
	p, err := unmarshalPayload[identitymodel.TenantRenamedPayload](evt)
	if err != nil {
		return errorfamily.WrapCorruption(
			err, "usermgmt.tenant_readmodel.decode_failed",
			"decode TenantRenamed in read model",
		).WithContextAny("agg_id", aggID)
	}
	if t, ok := m.tenants[aggID]; ok {
		t.DisplayName = p.DisplayName
	}
	return nil
}

// handleTenantMerged records on the survivor which tenants it absorbed.
func (m *TenantReadModel) handleTenantMerged(aggID id.StreamID, evt event.Event) error {
	p, err := unmarshalPayload[identitymodel.TenantMergedPayload](evt)
	if err != nil {
		return errorfamily.WrapCorruption(
			err, "usermgmt.tenant_readmodel.decode_failed",
			"decode TenantMerged in read model",
		).WithContextAny("agg_id", aggID)
	}
	t, ok := m.tenants[aggID]
	if !ok {
		return nil
	}
	for _, loser := range p.LoserIDs {
		if !tenantRowContains(t.MergedTenantIDs, loser) {
			t.MergedTenantIDs = append(t.MergedTenantIDs, loser)
		}
	}
	return nil
}

// handleTenantMergedInto retires the merged-away tenant: the row leaves the
// tenants map, the pointer map keeps its trace, and a re-delivered retirement
// is the no-op it must be.
func (m *TenantReadModel) handleTenantMergedInto(aggID id.StreamID, evt event.Event) error {
	p, err := unmarshalPayload[identitymodel.TenantMergedIntoPayload](evt)
	if err != nil {
		return errorfamily.WrapCorruption(
			err, "usermgmt.tenant_readmodel.decode_failed",
			"decode TenantMergedInto in read model",
		).WithContextAny("agg_id", aggID)
	}
	survivor, err := aggIDFromTenant(p.SurvivorID)
	if err != nil {
		return err
	}
	delete(m.tenants, aggID)
	m.mergedInto[aggID] = survivor
	return nil
}

func tenantRowContains(haystack []TenantID, needle TenantID) bool {
	for _, candidate := range haystack {
		if candidate.Get() == needle.Get() {
			return true
		}
	}
	return false
}

// FindByID follows the merge chain from a stream to the live tenant at its
// end; a stream that was not merged answers with itself.
func (m *TenantReadModel) FindByID(aggID id.StreamID) (*Tenant, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.live(m.resolve(aggID))
}

// FindByName returns the live tenant with the given name, following merge
// pointers: a merged-away duplicate's name is gone with its row.
func (m *TenantReadModel) FindByName(name string) (*Tenant, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for aggID, t := range m.tenants {
		if t.Name == name {
			return m.live(aggID)
		}
	}
	return nil, false
}

// FindByNormalizedName returns the live tenant whose name normalizes to the
// given name's normal form — the comparison the opt-in uniqueness check and
// the duplicates report use (issue #47: "Acme Corp" and "acme corp " are one
// tenant to a human).
func (m *TenantReadModel) FindByNormalizedName(name string) (*Tenant, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	want := identitymodel.NormalizeTenantName(name)
	for aggID, t := range m.tenants {
		if identitymodel.NormalizeTenantName(t.Name) == want {
			return m.live(aggID)
		}
	}
	return nil, false
}

// All returns every live (non-deleted, non-merged) tenant.
func (m *TenantReadModel) All() []*Tenant {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*Tenant, 0, len(m.tenants))
	for _, t := range m.tenants {
		result = append(result, t)
	}
	return result
}

// Retired reports whether the tenant stream was merged away into a survivor.
func (m *TenantReadModel) Retired(aggID id.StreamID) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, retired := m.mergedInto[aggID]
	return retired
}

// MergeTarget names the survivor a merged-away tenant stream points at,
// without following the chain. A service that must tell a merged tenant from
// a live one asks here; a lookup that wants the live tenant uses
// [TenantReadModel.FindByID].
func (m *TenantReadModel) MergeTarget(aggID id.StreamID) (id.StreamID, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	survivor, merged := m.mergedInto[aggID]
	return survivor, merged
}

// tenantMergeChainLimit bounds the pointer walk like the user model's: a
// walk longer than the pointer map found a cycle, and no tenant is live.
func (m *TenantReadModel) tenantMergeChainLimit() int {
	return len(m.mergedInto) + 1
}

// resolve follows the merge chain from a stream to the live survivor at its
// end. Callers hold at least a read lock.
func (m *TenantReadModel) resolve(aggID id.StreamID) id.StreamID {
	for range m.tenantMergeChainLimit() {
		survivor, merged := m.mergedInto[aggID]
		if !merged {
			return aggID
		}
		aggID = survivor
	}
	return id.StreamID{}
}

// live returns the stored tenant row on aggID. Callers hold at least a read
// lock.
func (m *TenantReadModel) live(aggID id.StreamID) (*Tenant, bool) {
	t, ok := m.tenants[aggID]
	return t, ok
}

// tenantRow returns a copy of what is stored on aggID itself, without
// following a merge pointer: the SQL layer needs the tenant as it was before
// a merge retires it. Callers need no lock (internal use holds it).
func (m *TenantReadModel) tenantRow(aggID id.StreamID) (*Tenant, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.tenants[aggID]
	return t, ok
}

var _ projection.Projection = (*TenantReadModel)(nil)
