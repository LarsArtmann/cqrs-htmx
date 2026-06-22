package usermgmt

import (
	"context"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

// MembershipReadModel is the projection-side store for memberships.
// It indexes memberships by aggregate ID (actor+tenant pair) and by
// actor ID for "what tenants is this actor a member of?" queries.
type MembershipReadModel struct {
	mu          sync.RWMutex
	memberships map[id.AggregateID]*Membership
	byActor     map[string][]id.AggregateID
}

// NewMembershipReadModel creates an empty MembershipReadModel.
func NewMembershipReadModel() *MembershipReadModel {
	return &MembershipReadModel{ //nolint:exhaustruct // mu starts zero-valued
		memberships: make(map[id.AggregateID]*Membership),
		byActor:     make(map[string][]id.AggregateID),
	}
}

func (m *MembershipReadModel) Name() string { return "membership-read-model" }

func (m *MembershipReadModel) EventTypes() []event.Type { return allMembershipEventTypes }

func (m *MembershipReadModel) Handle(_ context.Context, evt event.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	aggID := evt.AggregateID()

	switch evt.Type() {
	case eventMemberAdded:
		p, err := unmarshalPayload[MemberAddedPayload](evt)
		if err != nil {
			return event.WrapCorruption(
				err,
				"usermgmt.membership_readmodel.decode_failed",
				"decode MemberAdded in read model",
			)
		}
		roles := make([]Role, len(p.Roles))
		copy(roles, p.Roles)
		m.memberships[aggID] = &Membership{
			ActorID:  NewActorID(actorKindFromString(p.ActorKind), p.ActorID),
			TenantID: NewTenantID(p.TenantID),
			Roles:    roles,
			AddedAt:  time.Now().UTC(),
		}
		m.byActor[p.ActorID] = append(m.byActor[p.ActorID], aggID)

	case eventMemberRolesChanged:
		p, err := unmarshalPayload[MemberRolesChangedPayload](evt)
		if err != nil {
			return event.WrapCorruption(
				err,
				"usermgmt.membership_readmodel.decode_failed",
				"decode MemberRolesChanged in read model",
			)
		}
		mem, ok := m.memberships[aggID]
		if !ok {
			return nil
		}
		roles := make([]Role, len(p.Roles))
		copy(roles, p.Roles)
		mem.Roles = roles

	case eventMemberRemoved:
		m.removeMembership(aggID)

	default:
		return event.NewRejection(
			"usermgmt.membership_readmodel.unknown_event",
			"membership read model received unknown event type: "+string(evt.Type()),
		)
	}

	return nil
}

// FindByAggregateID returns the membership for the given aggregate ID.
func (m *MembershipReadModel) FindByAggregateID(aggID id.AggregateID) (*Membership, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	mem, ok := m.memberships[aggID]
	return mem, ok
}

// FindByActor returns all memberships for the given actor ID.
func (m *MembershipReadModel) FindByActor(actorID string) []*Membership {
	m.mu.RLock()
	defer m.mu.RUnlock()
	aggIDs := m.byActor[actorID]
	result := make([]*Membership, 0, len(aggIDs))
	for _, id := range aggIDs {
		if mem, ok := m.memberships[id]; ok {
			result = append(result, mem)
		}
	}
	return result
}

// FindByTenant returns all memberships for the given tenant ID.
func (m *MembershipReadModel) FindByTenant(tenantID string) []*Membership {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*Membership, 0)
	for _, mem := range m.memberships {
		if mem.TenantID.Get() == tenantID {
			result = append(result, mem)
		}
	}
	return result
}

func removeAggID(slice []id.AggregateID, target id.AggregateID) []id.AggregateID {
	for i, v := range slice {
		if v == target {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

// removeMembership removes a membership from both indexes.
func (m *MembershipReadModel) removeMembership(aggID id.AggregateID) {
	mem, ok := m.memberships[aggID]
	if !ok {
		return
	}
	actorKey := mem.ActorID.String()
	m.byActor[actorKey] = removeAggID(m.byActor[actorKey], aggID)
	if len(m.byActor[actorKey]) == 0 {
		delete(m.byActor, actorKey)
	}
	delete(m.memberships, aggID)
}

var _ event.Projection = (*MembershipReadModel)(nil)
