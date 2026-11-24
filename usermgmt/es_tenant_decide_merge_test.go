package usermgmt

import (
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// replayTenant folds events onto a start state the way the repository does.
func replayTenant(t *testing.T, start TenantState, events []event.Event) TenantState {
	t.Helper()
	state := start
	for _, evt := range events {
		next, err := foldTenant(state, evt)
		if err != nil {
			t.Fatalf("foldTenant(%s): %v", evt.Type(), err)
		}
		state = next
	}
	return state
}

// decideTenantError runs one tenant decider decision and returns only its
// error, for guard assertions.
func decideTenantError(
	decide func(TenantState, event.Version) ([]event.Event, error),
	state TenantState,
) error {
	_, err := decide(state, 1)
	return err
}

func liveTenantState() TenantState {
	return TenantState{Name: "acme", DisplayName: "Acme"}
}

func deletedTenantState() TenantState {
	return TenantState{Name: "acme", DisplayName: "Acme", Deleted: true, DeleteReason: "cleanup"}
}

func mergedTenantState(survivor identitymodel.TenantID) TenantState {
	return TenantState{
		Name: "acme", DisplayName: "Acme", Deleted: true, MergedInto: survivor,
	}
}

func tenantMergeLoserIDs(ids ...string) []identitymodel.TenantID {
	losers := make([]identitymodel.TenantID, len(ids))
	for i, raw := range ids {
		losers[i] = identitymodel.NewTenantID(raw)
	}
	return losers
}

// TestDecideRenameTenant covers rename: the happy path, the terminal-state
// refusals, and the same-value no-op.
func TestDecideRenameTenant(t *testing.T) {
	aggID := tenantTestAggID

	events, err := decideRenameTenant(aggID, "Acme Incorporated")(liveTenantState(), 1)
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if len(events) != 1 || events[0].Type() != eventTenantRenamed {
		t.Fatalf("events = %v, want one TenantRenamed", mergeEventTypes(events))
	}
	state := replayTenant(t, liveTenantState(), events)
	if state.DisplayName != "Acme Incorporated" {
		t.Errorf("DisplayName = %q, want the new label", state.DisplayName)
	}
	if state.Name != "acme" {
		t.Errorf("Name = %q, want the immutable stream name", state.Name)
	}

	requireDomainError(t, decideTenantError(decideRenameTenant(aggID, "X"), TenantState{}),
		errorfamily.Rejection, "usermgmt.tenant_rename.not_found")

	survivor := identitymodel.NewTenantID("01M47SURVIVOR0000000000T1")
	ef := requireDomainError(t,
		decideTenantError(decideRenameTenant(aggID, "X"), mergedTenantState(survivor)),
		errorfamily.Conflict, "usermgmt.tenant_rename.merged")
	if ef.ErrorContext()["merged_into"] != survivor.Get() {
		t.Errorf("merged_into = %q, want %q", ef.ErrorContext()["merged_into"], survivor.Get())
	}

	requireDomainError(t, decideTenantError(decideRenameTenant(aggID, ""), liveTenantState()),
		errorfamily.Rejection, "usermgmt.tenant_rename.display_name_required")

	events, err = decideRenameTenant(aggID, "Acme")(liveTenantState(), 1)
	if err != nil {
		t.Fatalf("same-value rename: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("same-value rename emitted %v, want none", mergeEventTypes(events))
	}
}

// TestDecideMergeTenants covers the survivor-side record: happy path with
// fold-through, guards, and the already-recorded no-op.
func TestDecideMergeTenants(t *testing.T) {
	aggID := tenantTestAggID
	losers := tenantMergeLoserIDs("01M47LOSER0000000000001A", "01M47LOSER0000000000002B")

	events, err := decideMergeTenants(aggID, losers, "duplicate tenants")(liveTenantState(), 3)
	if err != nil {
		t.Fatalf("merge tenants: %v", err)
	}
	if len(events) != 1 || events[0].Type() != eventTenantMerged {
		t.Fatalf("events = %v, want one TenantMerged", mergeEventTypes(events))
	}
	state := replayTenant(t, liveTenantState(), events)
	if len(state.MergedTenantIDs) != len(losers) {
		t.Fatalf("MergedTenantIDs = %v, want both losers", state.MergedTenantIDs)
	}

	requireDomainError(t, decideTenantError(decideMergeTenants(aggID, nil, "x"), liveTenantState()),
		errorfamily.Rejection, "usermgmt.tenant_merge.no_losers")
	requireDomainError(t,
		decideTenantError(decideMergeTenants(aggID, tenantMergeLoserIDs(""), "x"), liveTenantState()),
		errorfamily.Rejection, "usermgmt.tenant_merge.invalid_loser")
	requireDomainError(t,
		decideTenantError(decideMergeTenants(aggID, tenantMergeLoserIDs(aggID.Get()), "x"), liveTenantState()),
		errorfamily.Rejection, "usermgmt.tenant_merge.self_merge")

	otherSurvivor := identitymodel.NewTenantID("01M47OTHERSURVIVOR00000C1")
	ef := requireDomainError(t,
		decideTenantError(decideMergeTenants(aggID, losers, "x"), mergedTenantState(otherSurvivor)),
		errorfamily.Conflict, "usermgmt.tenant_merge.survivor_merged")
	if ef.ErrorContext()["merged_into"] != otherSurvivor.Get() {
		t.Errorf("merged_into = %q, want %q", ef.ErrorContext()["merged_into"], otherSurvivor.Get())
	}

	requireDomainError(t,
		decideTenantError(decideMergeTenants(aggID, losers, "x"), deletedTenantState()),
		errorfamily.Rejection, "usermgmt.tenant_merge.deleted")

	// Re-recording absorbed losers is a no-op.
	events, err = decideMergeTenants(aggID, losers, "again")(state, 4)
	if err != nil {
		t.Fatalf("re-record: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("re-record emitted %v, want none", mergeEventTypes(events))
	}
}

// TestDecideMergeTenantInto covers the loser-side tombstone: happy path,
// idempotent re-delivery, terminal conflicts, and malformed input.
func TestDecideMergeTenantInto(t *testing.T) {
	aggID := tenantTestAggID
	survivor := identitymodel.NewTenantID("01M47SURVIVOR0000000000T1")

	events, err := decideMergeTenantInto(aggID, survivor, "duplicate")(liveTenantState(), 1)
	if err != nil {
		t.Fatalf("merge tenant into: %v", err)
	}
	if len(events) != 1 || events[0].Type() != eventTenantMergedInto {
		t.Fatalf("events = %v, want one TenantMergedInto", mergeEventTypes(events))
	}
	state := replayTenant(t, liveTenantState(), events)
	if !state.IsMerged() || state.MergedInto.Get() != survivor.Get() {
		t.Errorf("state = %+v, want merged into the survivor", state)
	}

	events, err = decideMergeTenantInto(aggID, survivor, "again")(state, 2)
	if err != nil {
		t.Fatalf("idempotent re-delivery: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("re-delivery emitted %v, want none", mergeEventTypes(events))
	}

	other := identitymodel.NewTenantID("01M47OTHERSURVIVOR00000C1")
	ef := requireDomainError(t,
		decideTenantError(decideMergeTenantInto(aggID, other, "x"), state),
		errorfamily.Conflict, "usermgmt.tenant_merge_into.already_merged")
	if ef.ErrorContext()["merged_into"] != survivor.Get() {
		t.Errorf("merged_into = %q, want %q", ef.ErrorContext()["merged_into"], survivor.Get())
	}

	requireDomainError(t,
		decideTenantError(decideMergeTenantInto(aggID, survivor, "x"), deletedTenantState()),
		errorfamily.Conflict, "usermgmt.tenant_merge_into.deleted")
	requireDomainError(t,
		decideTenantError(decideMergeTenantInto(aggID, survivor, "x"), TenantState{}),
		errorfamily.Rejection, "usermgmt.tenant_merge_into.not_found")
	requireDomainError(t,
		decideTenantError(decideMergeTenantInto(aggID, identitymodel.TenantID{}, "x"), liveTenantState()),
		errorfamily.Rejection, "usermgmt.tenant_merge_into.survivor_required")
	requireDomainError(t,
		decideTenantError(
			decideMergeTenantInto(aggID, identitymodel.NewTenantID(aggID.Get()), "x"), liveTenantState()),
		errorfamily.Rejection, "usermgmt.tenant_merge_into.self_merge")
}

// TestDecideCreateTenant_RecreateRules pins D2: a plain tombstone may be
// recreated under its ID, a merged-away stream never.
func TestDecideCreateTenant_RecreateRules(t *testing.T) {
	aggID := tenantTestAggID

	events, err := decideCreateTenant(aggID, "acme", "Acme Redux")(deletedTenantState(), 2)
	if err != nil {
		t.Fatalf("recreate after delete: %v", err)
	}
	if len(events) != 1 || events[0].Type() != eventTenantCreated {
		t.Fatalf("events = %v, want one TenantCreated", mergeEventTypes(events))
	}
	state := replayTenant(t, deletedTenantState(), events)
	if state.Deleted || state.DeleteReason != "" {
		t.Errorf("recreated state = %+v, want the tombstone flags reset", state)
	}
	if state.Name != "acme" || state.DisplayName != "Acme Redux" {
		t.Errorf("recreated state = %+v, want the new incarnation's fields", state)
	}

	survivor := identitymodel.NewTenantID("01M47SURVIVOR0000000000T1")
	ef := requireDomainError(t,
		decideTenantError(decideCreateTenant(aggID, "acme", "X"), mergedTenantState(survivor)),
		errorfamily.Conflict, "usermgmt.tenant_create.merged")
	if ef.ErrorContext()["merged_into"] != survivor.Get() {
		t.Errorf("merged_into = %q, want %q", ef.ErrorContext()["merged_into"], survivor.Get())
	}

	requireDomainError(t,
		decideTenantError(decideCreateTenant(aggID, "other", "X"), liveTenantState()),
		errorfamily.Conflict, "usermgmt.tenant.already_exists")
}
