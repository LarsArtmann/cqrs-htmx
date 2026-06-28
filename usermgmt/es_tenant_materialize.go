package usermgmt

import (
	"context"
	"errors"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/kv/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/v3"
)

// IsTombstoned implements the tombstoner interface checked by
// [stack.FilterTombstoned]. A tenant is tombstoned when it has been deleted.
// This enables stack.Materialize to filter deleted tenants from List results
// without a separate Deleted flag check.
func (t *Tenant) IsTombstoned() bool { return t.Deleted }

// MaterializedTenantReadModel is a [TenantReadModel] alternative backed by
// [stack.Materialize] instead of a hand-written event switch.
//
// This type is the proof-of-concept deliverable for the stack.Materialize
// evaluation (ROADMAP v4.0, ADR-0017). It demonstrates that go-cqrs-lite's
// declarative projection can replace the manual Handle switch for the simplest
// read model (Tenant: 4 events, no secondary indexes, tombstone already marked
// by the decider).
//
// The embedded [*stack.Materialize] provides [View] and [List] queries plus the
// declarative callback fields (OnCreate, OnUpdate, OnTombstone). The wrapper
// implements the [event.Projection] interface (Name/Handle/EventTypes) because
// [stack.Materialize] in v3.1.0 only exposes [HandlerFunc] for Watermill router
// integration — it does not yet implement event.Projection directly. The Handle
// method below faithfully replicates the upstream handleEvent dispatch.
//
// In-memory storage uses [kv.NewTypedStore] over [kv.NewMemStore]. For SQL
// persistence, pass a [storage.SQLViewStore] to [NewMaterializedTenantReadModelWithStore].
type MaterializedTenantReadModel struct {
	mat *stack.Materialize[Tenant, TenantID]
}

// NewMaterializedTenantReadModel creates a MaterializedTenantReadModel backed
// by an in-memory [kv.TypedStore]. This is the zero-dependency default,
// equivalent to [NewTenantReadModel].
func NewMaterializedTenantReadModel() *MaterializedTenantReadModel {
	store := kv.NewTypedStore[Tenant, TenantID](kv.NewMemStore())
	return NewMaterializedTenantReadModelWithStore(store)
}

// NewMaterializedTenantReadModelWithStore creates a MaterializedTenantReadModel
// over a caller-provided [kv.ViewStore]. Use this to back the read model with a
// SQL view store ([storage.NewSQLiteViewStore] / [storage.NewSQLViewStore]) for
// persistence, or a Pebble store for embedded deployments.
func NewMaterializedTenantReadModelWithStore(store kv.ViewStore[Tenant, TenantID]) *MaterializedTenantReadModel {
	return &MaterializedTenantReadModel{
		mat: &stack.Materialize[Tenant, TenantID]{ //nolint:exhaustruct // OnRebirth intentionally nil
			Store: store,
			KeyFromEvent: func(evt event.Event) (TenantID, error) {
				return NewTenantID(evt.AggregateID().String()), nil
			},
			OnCreate: func(_ context.Context, evt event.Event) (*Tenant, error) {
				p, err := unmarshalPayload[TenantCreatedPayload](evt)
				if err != nil {
					return nil, event.WrapCorruption(
						err,
						"usermgmt.tenant_materialize.decode_create",
						"decode TenantCreated in materialized read model",
					)
				}
				return &Tenant{
					ID:          NewTenantID(evt.AggregateID().String()),
					Name:        p.Name,
					DisplayName: p.DisplayName,
					Suspended:   false,
					Deleted:     false,
				}, nil
			},
			OnUpdate: func(_ context.Context, evt event.Event, existing *Tenant) (*Tenant, error) {
				// OnUpdate receives ALL non-create, non-tombstone events. Branch
				// on event type exactly as the hand-written switch does.
				switch evt.Type() {
				case eventTenantSuspended:
					existing.Suspended = true
				case eventTenantReactivated:
					existing.Suspended = false
				default:
					return nil, event.NewRejection(
						"usermgmt.tenant_materialize.unknown_update_event",
						"materialized tenant read model received unexpected event in OnUpdate: "+string(evt.Type()),
					)
				}
				return existing, nil
			},
			OnTombstone: func(_ context.Context, _ event.Event, existing *Tenant) (*Tenant, error) {
				if existing == nil {
					return &Tenant{Deleted: true}, nil //nolint:exhaustruct // tombstone sentinel
				}
				existing.Deleted = true
				return existing, nil
			},
		},
	}
}

// Name implements [event.Projection].
func (m *MaterializedTenantReadModel) Name() string { return "tenant-read-model-materialize" }

// EventTypes implements [event.Projection]. Returns [allTenantEventTypes] so
// the projection dispatch router ([slices.Contains]) routes tenant events here.
// [stack.Materialize] itself returns nil (handles all types) — incompatible
// with our dispatch — so the wrapper must declare its interests explicitly.
func (m *MaterializedTenantReadModel) EventTypes() []event.Type { return allTenantEventTypes }

// Handle implements [event.Projection]. It replicates the upstream
// [stack.Materialize].handleEvent dispatch (unexported in v3.1.0):
//
//  1. Extract the key via KeyFromEvent.
//  2. If the event carries tombstone metadata, route to OnTombstone/OnRebirth.
//  3. Otherwise, load the existing record: if found → OnUpdate, if not → OnCreate.
//  4. Persist the result via Store.Set.
//
// When go-cqrs-lite ships Materialize with a public Handle method, this can
// delegate directly to m.mat.Handle(ctx, evt).
func (m *MaterializedTenantReadModel) Handle(ctx context.Context, evt event.Event) error {
	key, err := m.mat.KeyFromEvent(evt)
	if err != nil {
		return err //nolint:wrapcheck // KeyFromEvent returns classified errors
	}

	if evt.Metadata().Tombstone != nil {
		return m.handleTombstone(ctx, evt, key)
	}

	return m.handleRegular(ctx, evt, key)
}

// handleTombstone routes tombstone/rebirth-marked events to the corresponding
// callback and persists the result.
func (m *MaterializedTenantReadModel) handleTombstone(
	ctx context.Context,
	evt event.Event,
	key TenantID,
) error {
	existing, getErr := m.mat.Store.Get(ctx, key)
	if getErr != nil && !errors.Is(getErr, kv.ErrNotFound) {
		return event.WrapTransient(
			getErr,
			"usermgmt.tenant_materialize.tombstone_load",
			"load existing tenant for tombstone",
		)
	}
	if errors.Is(getErr, kv.ErrNotFound) {
		existing = nil
	}

	switch md := evt.Metadata().Tombstone.Status; md {
	case event.TombstoneTombstoned:
		if m.mat.OnTombstone == nil {
			return nil
		}
		updated, tErr := m.mat.OnTombstone(ctx, evt, existing)
		if tErr != nil {
			return tErr //nolint:wrapcheck // callback classifies
		}
		return m.persist(ctx, key, updated)
	case event.TombstoneActive:
		if m.mat.OnRebirth == nil {
			return nil
		}
		updated, rErr := m.mat.OnRebirth(ctx, evt, existing)
		if rErr != nil {
			return rErr //nolint:wrapcheck // callback classifies
		}
		return m.persist(ctx, key, updated)
	case event.TombstoneUndetermined:
		return nil
	default:
		return nil
	}
}

// handleRegular routes non-tombstone events: OnUpdate if the record exists,
// OnCreate if it does not.
func (m *MaterializedTenantReadModel) handleRegular(
	ctx context.Context,
	evt event.Event,
	key TenantID,
) error {
	existing, err := m.mat.Store.Get(ctx, key)
	if err != nil {
		if !errors.Is(err, kv.ErrNotFound) {
			return event.WrapTransient(
				err,
				"usermgmt.tenant_materialize.load_existing",
				"load existing tenant",
			)
		}
		if m.mat.OnCreate == nil {
			return nil
		}
		val, cErr := m.mat.OnCreate(ctx, evt)
		if cErr != nil {
			return cErr //nolint:wrapcheck // callback classifies
		}
		return m.persist(ctx, key, val)
	}

	if m.mat.OnUpdate == nil {
		return nil
	}
	updated, uErr := m.mat.OnUpdate(ctx, evt, existing)
	if uErr != nil {
		return uErr //nolint:wrapcheck // callback classifies
	}
	return m.persist(ctx, key, updated)
}

// persist writes a value to the store, wrapping the error as Transient.
func (m *MaterializedTenantReadModel) persist(ctx context.Context, key TenantID, val *Tenant) error {
	if err := m.mat.Store.Set(ctx, key, val); err != nil {
		return event.WrapTransient(
			err,
			"usermgmt.tenant_materialize.persist",
			"persist tenant to view store",
		)
	}
	return nil
}

// FindByID returns the active tenant for the given aggregate ID.
// Tombstoned (deleted) tenants are excluded.
func (m *MaterializedTenantReadModel) FindByID(ctx context.Context, aggID id.AggregateID) (*Tenant, bool) {
	t, err := m.mat.View(ctx, NewTenantID(aggID.String()))
	if err != nil || t == nil || t.IsTombstoned() {
		return nil, false
	}
	return t, true
}

// FindByName returns the first active tenant with the given name.
// This is a full scan over [stack.Materialize.List], matching the O(n)
// behavior of [TenantReadModel.FindByName].
func (m *MaterializedTenantReadModel) FindByName(ctx context.Context, name string) (*Tenant, bool) {
	all, err := m.mat.List(ctx, stack.ExcludeTombstoned)
	if err != nil {
		return nil, false
	}
	for _, t := range all {
		if t.Name == name {
			return t, true
		}
	}
	return nil, false
}

// All returns all active (non-deleted) tenants.
func (m *MaterializedTenantReadModel) All(ctx context.Context) []*Tenant {
	all, err := m.mat.List(ctx, stack.ExcludeTombstoned)
	if err != nil {
		return nil
	}
	return all
}

// Compile-time assertion: MaterializedTenantReadModel satisfies event.Projection.
var _ event.Projection = (*MaterializedTenantReadModel)(nil)
