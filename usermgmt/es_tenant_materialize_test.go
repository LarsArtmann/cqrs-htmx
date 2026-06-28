package usermgmt

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

// makeTombstoneTenantEvent builds a tenant event, optionally tombstone-marked (for delete).
func makeTombstoneTenantEvent(
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

func TestMaterializedTenantReadModel_CreateSuspendReactivateDelete(t *testing.T) {
	ctx := t.Context()
	rm := NewMaterializedTenantReadModel()
	tenantID := NewTenantID("01JXTENANT0000000000000000B")
	aggID, err := id.ParseAggregateID(tenantID.Get())
	if err != nil {
		t.Fatalf("ParseAggregateID: %v", err)
	}

	// --- Create ---
	createEvt := makeTombstoneTenantEvent(t, eventTenantCreated, aggID, TenantCreatedPayload{
		SchemaVersion: currentSchemaVersion,
		Name:          "acme",
		DisplayName:   "ACME Corp",
	}, false)
	if err := rm.Handle(ctx, createEvt); err != nil {
		t.Fatalf("Handle TenantCreated: %v", err)
	}

	// FindByID
	got, ok := rm.FindByID(ctx, aggID)
	if !ok {
		t.Fatal("FindByID after create: expected found")
	}
	if got.Name != "acme" || got.DisplayName != "ACME Corp" {
		t.Errorf("after create: name=%q display=%q, want acme/ACME Corp", got.Name, got.DisplayName)
	}
	if got.Suspended || got.Deleted {
		t.Errorf("after create: suspended=%v deleted=%v, want false/false", got.Suspended, got.Deleted)
	}

	// FindByName
	byName, ok := rm.FindByName(ctx, "acme")
	if !ok || byName.ID != tenantID {
		t.Errorf("FindByName: ok=%v id=%v", ok, byName)
	}

	// All
	if all := rm.All(ctx); len(all) != 1 {
		t.Errorf("All after create: len=%d, want 1", len(all))
	}

	// --- Suspend ---
	suspendPayload := TenantSuspendedPayload{SchemaVersion: currentSchemaVersion, Reason: "unpaid"}
	if err := rm.Handle(ctx, makeTombstoneTenantEvent(t, eventTenantSuspended, aggID,
		suspendPayload, false)); err != nil {
		t.Fatalf("Handle TenantSuspended: %v", err)
	}
	got, _ = rm.FindByID(ctx, aggID)
	if !got.Suspended {
		t.Error("after suspend: Suspended = false, want true")
	}

	// --- Reactivate ---
	if err := rm.Handle(ctx, makeTombstoneTenantEvent(t, eventTenantReactivated, aggID,
		TenantReactivatedPayload{SchemaVersion: currentSchemaVersion}, false)); err != nil {
		t.Fatalf("Handle TenantReactivated: %v", err)
	}
	got, _ = rm.FindByID(ctx, aggID)
	if got.Suspended {
		t.Error("after reactivate: Suspended = true, want false")
	}

	// --- Delete (tombstone-marked) ---
	if err := rm.Handle(ctx, makeTombstoneTenantEvent(t, eventTenantDeleted, aggID,
		TenantDeletedPayload{SchemaVersion: currentSchemaVersion, Reason: "gone"}, true)); err != nil {
		t.Fatalf("Handle TenantDeleted: %v", err)
	}

	// Deleted tenant excluded from all queries
	if _, ok := rm.FindByID(ctx, aggID); ok {
		t.Error("FindByID after delete: expected not found (tombstoned)")
	}
	if _, ok := rm.FindByName(ctx, "acme"); ok {
		t.Error("FindByName after delete: expected not found (tombstoned)")
	}
	if all := rm.All(ctx); len(all) != 0 {
		t.Errorf("All after delete: len=%d, want 0 (tombstoned excluded)", len(all))
	}
}

func TestMaterializedTenantReadModel_MultipleTenants(t *testing.T) {
	ctx := t.Context()
	rm := NewMaterializedTenantReadModel()

	ids := []string{
		"01JXTENANT00000000000000001",
		"01JXTENANT00000000000000002",
		"01JXTENANT00000000000000003",
	}
	names := []string{"alpha", "beta", "gamma"}

	for i, s := range ids {
		tid := NewTenantID(s)
		aggID, err := id.ParseAggregateID(tid.Get())
		if err != nil {
			t.Fatalf("ParseAggregateID %s: %v", s, err)
		}
		createPayload := TenantCreatedPayload{
			SchemaVersion: currentSchemaVersion, Name: names[i], DisplayName: names[i],
		}
		if err := rm.Handle(ctx, makeTombstoneTenantEvent(t, eventTenantCreated, aggID,
			createPayload, false)); err != nil {
			t.Fatalf("Handle create %s: %v", s, err)
		}
	}

	if all := rm.All(ctx); len(all) != 3 {
		t.Errorf("All: len=%d, want 3", len(all))
	}

	// Delete the middle one, verify 2 remain
	midID := NewTenantID(ids[1])
	midAgg, _ := id.ParseAggregateID(midID.Get())
	if err := rm.Handle(ctx, makeTombstoneTenantEvent(t, eventTenantDeleted, midAgg,
		TenantDeletedPayload{SchemaVersion: currentSchemaVersion}, true)); err != nil {
		t.Fatalf("Handle delete: %v", err)
	}
	if all := rm.All(ctx); len(all) != 2 {
		t.Errorf("All after delete: len=%d, want 2", len(all))
	}
}

func TestMaterializedTenantReadModel_UnknownEventRejected(t *testing.T) {
	ctx := t.Context()
	rm := NewMaterializedTenantReadModel()
	aggID, _ := id.ParseAggregateID(NewTenantID("01JXTENANT0000000000000000X").Get())

	// Create first so OnUpdate path is exercised for the unknown event.
	if err := rm.Handle(ctx, makeTombstoneTenantEvent(t, eventTenantCreated, aggID,
		TenantCreatedPayload{SchemaVersion: currentSchemaVersion, Name: "x"}, false)); err != nil {
		t.Fatalf("create: %v", err)
	}

	// An event that is in allTenantEventTypes but not handled in OnUpdate would
	// be rejected. We simulate this by sending a Suspended event but with an
	// unknown type label — but since EventTypes gates dispatch, we instead
	// verify the OnUpdate default case by sending an event type that passes
	// KeyFromEvent but isn't Suspended/Reactivated.
	unknownEvt := makeEventFor(t, event.Type("TenantMigrated"), 2, aggID, aggregateTypeTenant,
		map[string]any{"schema_version": currentSchemaVersion})
	err := rm.Handle(ctx, unknownEvt)
	if err == nil {
		t.Error("Handle unknown event: expected error, got nil")
	}
}

func TestMaterializedTenantReadModel_SatisfiesProjection(t *testing.T) {
	rm := NewMaterializedTenantReadModel()
	if rm.Name() != "tenant-read-model-materialize" {
		t.Errorf("Name = %q", rm.Name())
	}
	if len(rm.EventTypes()) == 0 {
		t.Error("EventTypes is empty — projection would never receive events")
	}
}
