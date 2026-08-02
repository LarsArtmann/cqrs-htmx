package usermgmt

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// MembershipReadModel is the projection-side store for memberships.
// It indexes memberships by aggregate ID (actor+tenant pair) and by
// actor ID for "what tenants is this actor a member of?" queries.
// cqrs-lint:ignore(C035) protected by embedded readModelCore.mu (sync.RWMutex)
type MembershipReadModel struct {
	readModelCore[*MembershipReadModel]
	//cqrs-lint:ignore(P011) bounded by finite membership count; in-memory dev/test default
	memberships map[id.StreamID]*Membership
	//cqrs-lint:ignore(P011) bounded by finite membership count; in-memory dev/test default
	byActor map[string][]id.StreamID
}

// NewMembershipReadModel creates an empty MembershipReadModel.
func NewMembershipReadModel() *MembershipReadModel {
	return &MembershipReadModel{
		readModelCore: readModelCore[*MembershipReadModel]{
			handlers: map[event.Type]eventHandler[*MembershipReadModel]{
				eventMemberAdded:        (*MembershipReadModel).applyMemberAdded,
				eventMemberRolesChanged: (*MembershipReadModel).handleMemberRolesChanged,
				eventMemberRemoved:      (*MembershipReadModel).handleMemberRemoved,
			},
		},
		memberships: make(map[id.StreamID]*Membership),
		byActor:     make(map[string][]id.StreamID),
	}
}

func (m *MembershipReadModel) Name() string { return "membership-read-model" }

func (m *MembershipReadModel) EventTypes() []event.Type { return allMembershipEventTypes }

func (m *MembershipReadModel) Handle(_ context.Context, evt event.Event) error {
	return m.handleEvent(m, evt)
}

func (m *MembershipReadModel) handleMemberRolesChanged(aggID id.StreamID, evt event.Event) error {
	p, err := unmarshalPayload[MemberRolesChangedPayload](evt)
	if err != nil {
		return errorfamily.WrapCorruption(
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
	return nil
}

func (m *MembershipReadModel) handleMemberRemoved(aggID id.StreamID, _ event.Event) error {
	m.removeMembership(aggID)
	return nil
}

func (m *MembershipReadModel) applyMemberAdded(aggID id.StreamID, evt event.Event) error {
	p, err := unmarshalPayload[MemberAddedPayload](evt)
	if err != nil {
		return errorfamily.WrapCorruption(
			err,
			"usermgmt.membership_readmodel.decode_failed",
			"decode MemberAdded in read model",
		)
	}
	roles := make([]Role, len(p.Roles))
	copy(roles, p.Roles)
	kind, err := actorKindFromString(p.ActorKind)
	if err != nil {
		return errorfamily.WrapCorruption(
			err,
			"usermgmt.membership_readmodel.unknown_actor_kind",
			"decode actor kind in MemberAdded",
		)
	}
	m.memberships[aggID] = &Membership{
		ActorID:  NewActorID(kind, p.ActorID),
		TenantID: NewTenantID(p.TenantID),
		Roles:    roles,
		AddedAt:  time.Now().UTC(),
	}
	m.byActor[p.ActorID] = append(m.byActor[p.ActorID], aggID)
	return nil
}

// FindByAggregateID returns the membership for the given aggregate ID.
func (m *MembershipReadModel) FindByAggregateID(aggID id.StreamID) (*Membership, bool) {
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

func removeAggID(slice []id.StreamID, target id.StreamID) []id.StreamID {
	for i, v := range slice {
		if v == target {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

// removeMembership removes a membership from both indexes.
func (m *MembershipReadModel) removeMembership(aggID id.StreamID) {
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

var _ projection.Projection = (*MembershipReadModel)(nil)
