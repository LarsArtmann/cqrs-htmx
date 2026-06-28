package usermgmt

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/kv/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/v3"
)

// makeMaterializeTenantEvent builds a tenant event for adapter tests.
// If markTombstone is true, the event is tombstone-marked (for delete).
func makeMaterializeTenantEvent(
	t *testing.T,
	eventType event.Type,
	aggID id.AggregateID,
	payload any,
	markTombstone bool,
) event.Event {
	t.Helper()
	evt := makeEventFor(t, eventType, 1, aggID, aggregateTypeTenant, payload)
	if !markTombstone {
		return evt
	}
	marked, err := event.MarkTombstone(evt)
	if err != nil {
		t.Fatalf("MarkTombstone %s: %v", eventType, err)
	}
	return marked
}

func TestMaterializeProjection_TenantLifecycle(t *testing.T) {
	ctx := t.Context()

	store := kv.NewTypedStore[Tenant, TenantID](kv.NewMemStore())
	mat := &stack.Materialize[Tenant, TenantID]{
		Store: store,
		KeyFromEvent: func(evt event.Event) (TenantID, error) {
			return NewTenantID(evt.AggregateID().String()), nil
		},
		OnCreate: func(_ context.Context, evt event.Event) (*Tenant, error) {
			p, err := unmarshalPayload[TenantCreatedPayload](evt)
			if err != nil {
				return nil, event.WrapCorruption(err,
					"test.decode_create", "decode TenantCreated")
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
			switch evt.Type() {
			case eventTenantSuspended:
				existing.Suspended = true
			case eventTenantReactivated:
				existing.Suspended = false
			}
			return existing, nil
		},
		OnTombstone: func(_ context.Context, _ event.Event, existing *Tenant) (*Tenant, error) {
			if existing != nil {
				existing.Deleted = true
			}
			return existing, nil
		},
	}

	proj := NewMaterializeProjection(mat, "tenant-test", allTenantEventTypes)

	if proj.Name() != "tenant-test" {
		t.Errorf("Name = %q, want %q", proj.Name(), "tenant-test")
	}
	if len(proj.EventTypes()) != len(allTenantEventTypes) {
		t.Errorf("EventTypes len = %d, want %d", len(proj.EventTypes()), len(allTenantEventTypes))
	}

	tenantID := NewTenantID("01JXTENANT0000000000000000B")
	aggID, err := id.ParseAggregateID(tenantID.Get())
	if err != nil {
		t.Fatalf("ParseAggregateID: %v", err)
	}

	// --- Create ---
	createPayload := TenantCreatedPayload{
		SchemaVersion: currentSchemaVersion,
		Name:          "acme",
		DisplayName:   "ACME Corp",
	}
	createEvt := makeMaterializeTenantEvent(t, eventTenantCreated, aggID, createPayload, false)
	if err := proj.Handle(ctx, createEvt); err != nil {
		t.Fatalf("Handle TenantCreated: %v", err)
	}

	got, err := mat.View(ctx, tenantID)
	if err != nil {
		t.Fatalf("View after create: %v", err)
	}
	if got.Name != "acme" || got.DisplayName != "ACME Corp" {
		t.Errorf("after create: name=%q display=%q", got.Name, got.DisplayName)
	}
	if got.Suspended || got.Deleted {
		t.Errorf("after create: suspended=%v deleted=%v", got.Suspended, got.Deleted)
	}

	// --- Suspend ---
	suspendPayload := TenantSuspendedPayload{SchemaVersion: currentSchemaVersion, Reason: "unpaid"}
	suspendEvt := makeMaterializeTenantEvent(t, eventTenantSuspended, aggID, suspendPayload, false)
	if err := proj.Handle(ctx, suspendEvt); err != nil {
		t.Fatalf("Handle TenantSuspended: %v", err)
	}
	got, _ = mat.View(ctx, tenantID)
	if !got.Suspended {
		t.Error("after suspend: Suspended = false, want true")
	}

	// --- Reactivate ---
	reactivatePayload := TenantReactivatedPayload{SchemaVersion: currentSchemaVersion}
	reactivateEvt := makeMaterializeTenantEvent(t, eventTenantReactivated, aggID, reactivatePayload, false)
	if err := proj.Handle(ctx, reactivateEvt); err != nil {
		t.Fatalf("Handle TenantReactivated: %v", err)
	}
	got, _ = mat.View(ctx, tenantID)
	if got.Suspended {
		t.Error("after reactivate: Suspended = true, want false")
	}

	// --- Delete (tombstone-marked) ---
	deletePayload := TenantDeletedPayload{SchemaVersion: currentSchemaVersion, Reason: "gone"}
	deleteEvt := makeMaterializeTenantEvent(t, eventTenantDeleted, aggID, deletePayload, true)
	if err := proj.Handle(ctx, deleteEvt); err != nil {
		t.Fatalf("Handle TenantDeleted: %v", err)
	}
	got, _ = mat.View(ctx, tenantID)
	if !got.Deleted {
		t.Error("after delete: Deleted = false, want true (tombstoned)")
	}

	// Verify the record is still in the store (Materialize soft-deletes, never hard-deletes)
	all, err := mat.List(ctx, stack.IncludeTombstoned)
	if err != nil {
		t.Fatalf("List Include: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("List IncludeTombstoned: len=%d, want 1", len(all))
	}
}

func TestMaterializeProjection_SatisfiesEventProjection(t *testing.T) {
	store := kv.NewTypedStore[Tenant, TenantID](kv.NewMemStore())
	mat := &stack.Materialize[Tenant, TenantID]{
		Store: store,
		KeyFromEvent: func(evt event.Event) (TenantID, error) {
			return NewTenantID(evt.AggregateID().String()), nil
		},
	}
	proj := NewMaterializeProjection(mat, "test-proj", allTenantEventTypes)

	// Compile-time check is via var _, but let's verify at runtime too
	var _ event.Projection = proj
}
