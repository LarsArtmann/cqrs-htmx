package usermgmt

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

var tenantTestAggID = id.NewAggregateID() //nolint:gochecknoglobals // test fixture

func makeTenantEvent(t *testing.T, eventType event.Type, version event.Version, payload any) event.Event {
	t.Helper()
	payloadBytes, err := marshalPayload(payload)
	if err != nil {
		t.Fatalf("marshal payload for %s: %v", eventType, err)
	}
	evt, err := event.NewEvent(eventType, tenantTestAggID, aggregateTypeTenant, version, payloadBytes)
	if err != nil {
		t.Fatalf("makeTenantEvent %s: %v", eventType, err)
	}
	return evt
}

func TestFoldTenant_Created(t *testing.T) {
	state, err := foldTenant(TenantState{}, makeTenantEvent(t, eventTenantCreated, 1, TenantCreatedPayload{
		Name:        "acme",
		DisplayName: "Acme Corp",
	}))
	if err != nil {
		t.Fatalf("foldTenant: %v", err)
	}
	if state.Name != "acme" {
		t.Errorf("Name = %q", state.Name)
	}
	if state.DisplayName != "Acme Corp" {
		t.Errorf("DisplayName = %q", state.DisplayName)
	}
	if !state.Exists() {
		t.Error("tenant should exist after creation")
	}
	if !state.IsActive() {
		t.Error("newly created tenant should be active")
	}
}

func TestFoldTenant_Suspended(t *testing.T) {
	state := TenantState{Name: "acme", DisplayName: "Acme"}
	state, err := foldTenant(state, makeTenantEvent(t, eventTenantSuspended, 2, TenantSuspendedPayload{
		Reason: "non-payment",
	}))
	if err != nil {
		t.Fatalf("foldTenant: %v", err)
	}
	if !state.Suspended {
		t.Error("tenant should be suspended")
	}
	if state.SuspendReason != "non-payment" {
		t.Errorf("SuspendReason = %q", state.SuspendReason)
	}
	if state.IsActive() {
		t.Error("suspended tenant should not be active")
	}
}

func TestFoldTenant_Reactivated(t *testing.T) {
	state := TenantState{Name: "acme", Suspended: true, SuspendReason: "test"}
	state, err := foldTenant(state, makeTenantEvent(t, eventTenantReactivated, 3, TenantReactivatedPayload{}))
	if err != nil {
		t.Fatalf("foldTenant: %v", err)
	}
	if state.Suspended {
		t.Error("tenant should not be suspended after reactivation")
	}
	if state.SuspendReason != "" {
		t.Errorf("SuspendReason should be cleared, got %q", state.SuspendReason)
	}
}

func TestFoldTenant_Deleted(t *testing.T) {
	state := TenantState{Name: "acme"}
	markEvt, err := event.NewEvent(
		eventTenantDeleted, tenantTestAggID, aggregateTypeTenant, 2,
		mustMarshalPayload(t, TenantDeletedPayload{Reason: "shutdown"}),
	)
	if err != nil {
		t.Fatalf("create deleted event: %v", err)
	}
	markEvt, err = event.MarkTombstone(markEvt)
	if err != nil {
		t.Fatalf("mark tombstone: %v", err)
	}
	state, err = foldTenant(state, markEvt)
	if err != nil {
		t.Fatalf("foldTenant: %v", err)
	}
	if !state.Deleted {
		t.Error("tenant should be deleted")
	}
	if state.DeleteReason != "shutdown" {
		t.Errorf("DeleteReason = %q", state.DeleteReason)
	}
	if state.Exists() {
		t.Error("deleted tenant should not exist")
	}
}

func TestFoldTenant_UnknownEvent(t *testing.T) {
	unknownEvt, err := event.NewEvent(
		event.Type("FutureTenantEvent"), tenantTestAggID, aggregateTypeTenant, 1,
		[]byte(`{}`),
	)
	if err != nil {
		t.Fatalf("create unknown event: %v", err)
	}
	_, err = foldTenant(TenantState{}, unknownEvt)
	if err == nil {
		t.Error("expected error for unknown event")
	}
}

func TestDecideCreateTenant_AlreadyExists(t *testing.T) {
	state := TenantState{Name: "existing"}
	decide := decideCreateTenant(tenantTestAggID, "new", "")
	_, err := decide(state, 1)
	if err == nil {
		t.Error("expected conflict for already-existing tenant")
	}
}

func TestDecideCreateTenant_EmptyName(t *testing.T) {
	decide := decideCreateTenant(tenantTestAggID, "", "")
	_, err := decide(TenantState{}, 1)
	if err == nil {
		t.Error("expected rejection for empty name")
	}
}

func TestDecideSuspendTenant_NotFound(t *testing.T) {
	decide := decideSuspendTenant(tenantTestAggID, "test")
	_, err := decide(TenantState{}, 1)
	if err == nil {
		t.Error("expected rejection for non-existent tenant")
	}
}

func TestDecideSuspendTenant_AlreadySuspended(t *testing.T) {
	state := TenantState{Name: "acme", Suspended: true}
	decide := decideSuspendTenant(tenantTestAggID, "test")
	events, err := decide(state, 1)
	if err != nil {
		t.Fatalf("expected nil error for already-suspended (idempotent), got: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events for idempotent suspend, got %d", len(events))
	}
}

func TestDecideDeleteTenant_AlreadyDeleted(t *testing.T) {
	state := TenantState{Name: "acme", Deleted: true}
	decide := decideDeleteTenant(tenantTestAggID, "test")
	_, err := decide(state, 1)
	if err == nil {
		t.Error("expected rejection for already-deleted tenant")
	}
}

func mustMarshalPayload(t *testing.T, payload any) []byte {
	t.Helper()
	b, err := marshalPayload(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return b
}
