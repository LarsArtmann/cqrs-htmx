package usermgmt

import (
	"context"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// tenantMergeSteps is the canonical tenant-merge scenario as event recipes:
// a survivor ("acme") renamed, a duplicate loser ("acme-corp") merged into
// it — adoption record on the survivor, retirement on the loser last.
func tenantMergeSteps(survivor, loser id.StreamID) []struct {
	aggID   id.StreamID
	kind    event.Type
	version event.Version
	payload any
} {
	return []struct {
		aggID   id.StreamID
		kind    event.Type
		version event.Version
		payload any
	}{
		{survivor, eventTenantCreated, 1, TenantCreatedPayload{Name: "acme", DisplayName: "Acme"}},
		{loser, eventTenantCreated, 1, TenantCreatedPayload{Name: "acme-corp", DisplayName: "Acme Corp"}},
		{survivor, eventTenantRenamed, 2, identitymodel.TenantRenamedPayload{DisplayName: "Acme Incorporated"}},
		{survivor, eventTenantMerged, 3, identitymodel.TenantMergedPayload{
			LoserIDs: []identitymodel.TenantID{identitymodel.NewTenantID(loser.Get())},
			Reason:   "duplicate tenants",
		}},
		{loser, eventTenantMergedInto, 2, identitymodel.TenantMergedIntoPayload{
			SurvivorID: identitymodel.NewTenantID(survivor.Get()),
			Reason:     "duplicate tenants",
		}},
	}
}

func tenantScenarioEvents(t *testing.T, survivor, loser id.StreamID) []event.Event {
	t.Helper()
	events := make([]event.Event, 0, 5)
	for _, s := range tenantMergeSteps(survivor, loser) {
		evt, err := event.NewEvent(s.kind, s.aggID, aggregateTypeTenant, s.version,
			mustMarshalTenantPayload(t, s.payload))
		if err != nil {
			t.Fatalf("make %s: %v", s.kind, err)
		}
		events = append(events, evt)
	}
	return events
}

func applyTenantEvents(t *testing.T, apply func(context.Context, event.Event) error, events []event.Event) {
	t.Helper()
	ctx := context.Background()
	for _, evt := range events {
		if err := apply(ctx, evt); err != nil {
			t.Fatalf("handle %s: %v", evt.Type(), err)
		}
	}
}

func mustMarshalTenantPayload(t *testing.T, payload any) []byte {
	t.Helper()
	raw, err := marshalPayload(payload)
	if err != nil {
		t.Fatalf("marshal tenant payload: %v", err)
	}
	return raw
}

// TestTenantReadModel_MergeLivePath covers the in-memory live path: rename
// lands on the row, the survivor records what it absorbed, the loser is
// retired with a pointer, and lookups follow the pointer.
func TestTenantReadModel_MergeLivePath(t *testing.T) {
	m := NewTenantReadModel()
	survivor, loser := id.NewStreamID(), id.NewStreamID()
	applyTenantEvents(t, m.Handle, tenantScenarioEvents(t, survivor, loser))

	got, ok := m.FindByID(survivor)
	if !ok {
		t.Fatal("survivor row missing")
	}
	if got.DisplayName != "Acme Incorporated" {
		t.Errorf("DisplayName = %q, want the renamed label", got.DisplayName)
	}
	if len(got.MergedTenantIDs) != 1 || got.MergedTenantIDs[0].Get() != loser.Get() {
		t.Errorf("MergedTenantIDs = %v, want the loser", got.MergedTenantIDs)
	}

	// The pointer-following lookup: the loser's ID answers the survivor.
	viaPointer, ok := m.FindByID(loser)
	if !ok || viaPointer.ID.Get() != survivor.Get() {
		t.Errorf("FindByID(loser) = %v,%v; want the survivor row", viaPointer, ok)
	}
	if !m.Retired(loser) {
		t.Error("loser is not Retired")
	}
	target, merged := m.MergeTarget(loser)
	if !merged || target.Get() != survivor.Get() {
		t.Errorf("MergeTarget = %v,%v; want survivor", target, merged)
	}
	if _, ok := m.FindByName("acme-corp"); ok {
		t.Error("loser still answers FindByName after the merge")
	}
	if rows := m.All(); len(rows) != 1 {
		t.Errorf("All = %d rows, want the survivor only", len(rows))
	}
	if _, ok := m.tenantRow(loser); ok {
		t.Error("the loser kept a live row")
	}

	// A re-delivered retirement is the no-op the fold promises.
	evt, err := event.NewEvent(eventTenantMergedInto, loser, aggregateTypeTenant, 2,
		mustMarshalTenantPayload(t, identitymodel.TenantMergedIntoPayload{
			SurvivorID: identitymodel.NewTenantID(survivor.Get()),
		}))
	if err != nil {
		t.Fatalf("make re-delivered retirement: %v", err)
	}
	if err := m.Handle(context.Background(), evt); err != nil {
		t.Fatalf("re-delivered retirement: %v", err)
	}
	if !m.Retired(loser) {
		t.Error("re-delivered retirement lost the pointer")
	}
}

// TestTenantReadModel_NormalizedNameLookup covers the issue #47 comparison:
// case and whitespace do not make a new tenant.
func TestTenantReadModel_NormalizedNameLookup(t *testing.T) {
	if got := identitymodel.NormalizeTenantName("  Acme Corp "); got != "acme corp" {
		t.Fatalf("NormalizeTenantName = %q", got)
	}

	m := NewTenantReadModel()
	survivor := id.NewStreamID()
	evt, err := event.NewEvent(eventTenantCreated, survivor, aggregateTypeTenant, 1,
		mustMarshalTenantPayload(t, TenantCreatedPayload{Name: "Acme Corp"}))
	if err != nil {
		t.Fatalf("make TenantCreated: %v", err)
	}
	if err := m.Handle(context.Background(), evt); err != nil {
		t.Fatalf("handle TenantCreated: %v", err)
	}

	got, ok := m.FindByNormalizedName("  acme corp ")
	if !ok || got.Name != "Acme Corp" {
		t.Errorf("FindByNormalizedName = %v,%v; want the row", got, ok)
	}
	if _, ok := m.FindByNormalizedName("other"); ok {
		t.Error("FindByNormalizedName matched an unrelated name")
	}
}

// TestSQLTenantReadModel_MergeSurvivesRestart covers the SQL live path: the
// tombstone carries the pointer, the hydrate rebuilds both maps, and a
// re-delivered pre-merge event stays retired (the G9 class).
func TestSQLTenantReadModel_MergeSurvivesRestart(t *testing.T) {
	db := newSQLTestDB(t)
	sqlModel, err := NewSQLiteTenantReadModel(db)
	if err != nil {
		t.Fatalf("NewSQLiteTenantReadModel: %v", err)
	}
	survivor, loser := id.NewStreamID(), id.NewStreamID()
	applyTenantEvents(t, sqlModel.Handle, tenantScenarioEvents(t, survivor, loser))

	if got, ok := sqlModel.FindByID(loser); !ok || got.ID.Get() != survivor.Get() {
		t.Errorf("FindByID(loser) = %v,%v; want the survivor via the pointer", got, ok)
	}

	fresh, err := NewSQLiteTenantReadModel(db)
	if err != nil {
		t.Fatalf("fresh model: %v", err)
	}
	if err := fresh.Hydrate(context.Background()); err != nil {
		t.Fatalf("Hydrate: %v", err)
	}
	if !fresh.Retired(loser) {
		t.Error("hydrate lost the merge pointer")
	}
	if target, merged := fresh.MergeTarget(loser); !merged || target.Get() != survivor.Get() {
		t.Errorf("MergeTarget after hydrate = %v,%v", target, merged)
	}
	got, ok := fresh.FindByID(survivor)
	if !ok || got.DisplayName != "Acme Incorporated" {
		t.Errorf("survivor after hydrate = %v,%v", got, ok)
	}
	if len(fresh.All()) != 1 {
		t.Errorf("All after hydrate = %d rows, want 1", len(fresh.All()))
	}

	// The G9 class: a re-delivered pre-merge event on the retired stream must
	// not write the survivor's row under the loser's key.
	replayed, err := event.NewEvent(eventTenantRenamed, loser, aggregateTypeTenant, 9,
		mustMarshalTenantPayload(t, identitymodel.TenantRenamedPayload{DisplayName: "zombie"}))
	if err != nil {
		t.Fatalf("make replayed rename: %v", err)
	}
	if err := fresh.Handle(context.Background(), replayed); err != nil {
		t.Fatalf("replayed rename: %v", err)
	}
	if _, ok := fresh.tenantRow(loser); ok {
		t.Error("replayed rename resurrected the loser row")
	}
	if got, ok := fresh.tenantRow(survivor); !ok || got.DisplayName != "Acme Incorporated" {
		t.Errorf("survivor row after replay = %v,%v; want the renamed label untouched", got, ok)
	}
}
