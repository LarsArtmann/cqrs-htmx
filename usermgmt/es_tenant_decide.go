package usermgmt

import (
	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-codec"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// TenantDecider returns the Decider for the Tenant aggregate.
func TenantDecider() decider.Decider[TenantState] {
	return decider.Decider[TenantState]{
		Initial: TenantState{}, //nolint:exhaustruct // zero-value is correct for aggregate initial state
		Apply:   foldTenant,
	}
}

func decideCreateTenant(
	aggID id.StreamID,
	name, displayName string,
) func(TenantState, event.Version) ([]event.Event, error) {
	return func(state TenantState, version event.Version) ([]event.Event, error) {
		// A merged-away tenant is terminal (D2): recreating its ID would
		// orphan every pointer that names it as merged. The conflict names
		// the survivor so the caller sees the real tenant.
		if state.IsMerged() {
			return nil, errorfamily.NewConflict(
				"usermgmt.tenant_create.merged",
				"a merged-away tenant ID cannot be recreated",
			).WithContext("merged_into", state.MergedInto.Get())
		}
		// A plain tombstone may be recreated: the TenantCreated fold resets
		// the state, so the new incarnation starts clean.
		if state.Name != "" && !state.Deleted {
			return nil, errorfamily.NewConflict(
				"usermgmt.tenant.already_exists",
				"tenant with this ID already exists",
			)
		}
		if name == "" {
			return nil, errorfamily.NewRejection(
				"usermgmt.tenant.name_required",
				"tenant name is required",
			)
		}
		evt, err := event.New(
			eventTenantCreated, aggID, aggregateTypeTenant, version.Increment(),
			TenantCreatedPayload{
				SchemaVersion: currentSchemaVersion,
				Name:          name,
				DisplayName:   displayName,
			},
			//cqrs-lint:ignore(A027) one codec per repository is intentional; global DefaultCodec would couple unrelated aggregates
			event.WithCodec(codec.JSONCodec{}), event.WithSchemaVersion(1),
		)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.tenant.event_failed",
				"create TenantCreated event",
			)
		}
		return []event.Event{evt}, nil
	}
}

func decideSuspendTenant(
	aggID id.StreamID,
	reason string,
) func(TenantState, event.Version) ([]event.Event, error) {
	return func(state TenantState, version event.Version) ([]event.Event, error) {
		if !state.Exists() {
			return nil, errorfamily.NewRejection(
				"usermgmt.tenant_suspend.not_found",
				"tenant does not exist",
			)
		}
		if state.Suspended {
			return nil, nil
		}
		evt, err := event.New(
			eventTenantSuspended, aggID, aggregateTypeTenant, version.Increment(),
			TenantSuspendedPayload{
				SchemaVersion: currentSchemaVersion,
				Reason:        reason,
			},
			event.WithCodec(codec.JSONCodec{}), event.WithSchemaVersion(1),
		)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.tenant_suspend.event_failed",
				"create TenantSuspended event",
			)
		}
		return []event.Event{evt}, nil
	}
}

func decideReactivateTenant(
	aggID id.StreamID,
) func(TenantState, event.Version) ([]event.Event, error) {
	return func(state TenantState, version event.Version) ([]event.Event, error) {
		if !state.Exists() {
			return nil, errorfamily.NewRejection(
				"usermgmt.tenant_reactivate.not_found",
				"tenant does not exist",
			)
		}
		if !state.Suspended {
			return nil, nil
		}
		evt, err := event.New(
			eventTenantReactivated, aggID, aggregateTypeTenant, version.Increment(),
			TenantReactivatedPayload{
				SchemaVersion: currentSchemaVersion,
			},
			event.WithCodec(codec.JSONCodec{}), event.WithSchemaVersion(1),
		)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.tenant_reactivate.event_failed",
				"create TenantReactivated event",
			)
		}
		return []event.Event{evt}, nil
	}
}

func decideDeleteTenant(
	aggID id.StreamID,
	reason string,
) func(TenantState, event.Version) ([]event.Event, error) {
	return func(state TenantState, version event.Version) ([]event.Event, error) {
		if !state.Exists() {
			return nil, errorfamily.NewRejection(
				"usermgmt.tenant_delete.not_found",
				"tenant does not exist",
			)
		}
		if state.Deleted {
			return nil, errorfamily.NewRejection(
				"usermgmt.tenant_delete.already_deleted",
				"tenant is already deleted",
			)
		}
		evt, err := event.New(
			eventTenantDeleted, aggID, aggregateTypeTenant, version.Increment(),
			TenantDeletedPayload{
				SchemaVersion: currentSchemaVersion,
				Reason:        reason,
			},
			event.WithCodec(codec.JSONCodec{}), event.WithSchemaVersion(1),
		)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.tenant_delete.event_failed",
				"create TenantDeleted event",
			)
		}
		return []event.Event{evt}, nil
	}
}

// decideRenameTenant changes a tenant's display name. The tenant ID never
// moves: it is the stream identity, the Casbin domain and the key of every
// tenant-scoped row, which is exactly why a typo is fixed by rename and not
// by delete-and-recreate. Renaming the same value again is a no-op.
func decideRenameTenant(
	aggID id.StreamID,
	displayName string,
) func(TenantState, event.Version) ([]event.Event, error) {
	return func(state TenantState, version event.Version) ([]event.Event, error) {
		if state.IsMerged() {
			return nil, errorfamily.NewConflict(
				"usermgmt.tenant_rename.merged",
				"a merged-away tenant is terminal",
			).WithContext("merged_into", state.MergedInto.Get())
		}
		if !state.Exists() {
			return nil, errorfamily.NewRejection(
				"usermgmt.tenant_rename.not_found",
				"tenant does not exist",
			)
		}
		if displayName == "" {
			return nil, errorfamily.NewRejection(
				"usermgmt.tenant_rename.display_name_required",
				"display name is required",
			)
		}
		if state.DisplayName == displayName {
			return nil, nil
		}
		evt, err := event.New(
			eventTenantRenamed, aggID, aggregateTypeTenant, version.Increment(),
			identitymodel.TenantRenamedPayload{
				SchemaVersion: currentSchemaVersion,
				DisplayName:   displayName,
			},
			event.WithCodec(codec.JSONCodec{}), event.WithSchemaVersion(1),
		)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.tenant_rename.event_failed",
				"create TenantRenamed event",
			)
		}
		return []event.Event{evt}, nil
	}
}

// requireLiveTenantSurvivor refuses a tenant stream that cannot absorb
// another one: never registered, merged away itself (the conflict names the
// survivor it points at), or deleted.
func requireLiveTenantSurvivor(state TenantState) error {
	if !state.Exists() {
		if state.IsMerged() {
			return errorfamily.NewConflict(
				"usermgmt.tenant_merge.survivor_merged",
				"a merged-away tenant cannot be the survivor of another merge",
			).WithContext("merged_into", state.MergedInto.Get())
		}
		if state.Deleted {
			return errorfamily.NewRejection(
				"usermgmt.tenant_merge.deleted",
				"cannot merge into a deleted tenant",
			)
		}
		return errorfamily.NewRejection(
			"usermgmt.tenant_merge.not_found",
			"tenant does not exist",
		)
	}
	return nil
}

// unmergedTenantLosers returns the losing tenants the survivor has not
// recorded yet, in order and without duplicates.
func unmergedTenantLosers(state TenantState, loserIDs []identitymodel.TenantID) []identitymodel.TenantID {
	var losers []identitymodel.TenantID
	for _, loser := range loserIDs {
		if !containsTenant(state.MergedTenantIDs, loser) && !containsTenant(losers, loser) {
			losers = append(losers, loser)
		}
	}
	return losers
}

func containsTenant(haystack []identitymodel.TenantID, needle identitymodel.TenantID) bool {
	for _, candidate := range haystack {
		if candidate.Get() == needle.Get() {
			return true
		}
	}
	return false
}

// decideMergeTenants records on the surviving tenant's stream that the listed
// tenants were consolidated into it. The roster itself moves through ordinary
// membership commands (the service drives those), so this decision only
// records what the survivor absorbed — one atomic append.
func decideMergeTenants(
	aggID id.StreamID,
	loserIDs []identitymodel.TenantID,
	reason string,
) func(TenantState, event.Version) ([]event.Event, error) {
	return func(state TenantState, version event.Version) ([]event.Event, error) {
		if err := requireLiveTenantSurvivor(state); err != nil {
			return nil, err
		}
		if len(loserIDs) == 0 {
			return nil, errorfamily.NewRejection(
				"usermgmt.tenant_merge.no_losers",
				"a merge needs at least one loser",
			)
		}
		for _, loser := range loserIDs {
			switch {
			case loser.IsZero():
				return nil, errorfamily.NewRejection(
					"usermgmt.tenant_merge.invalid_loser",
					"loser tenant ID is required",
				)
			case loser.Get() == aggID.Get():
				return nil, errorfamily.NewRejection(
					"usermgmt.tenant_merge.self_merge",
					"a tenant cannot be merged into itself",
				)
			}
		}
		losers := unmergedTenantLosers(state, loserIDs)
		if len(losers) == 0 {
			return nil, nil
		}
		evt, err := event.New(
			eventTenantMerged, aggID, aggregateTypeTenant, version.Increment(),
			identitymodel.TenantMergedPayload{
				SchemaVersion: currentSchemaVersion,
				LoserIDs:      losers,
				Reason:        reason,
			},
			event.WithCodec(codec.JSONCodec{}), event.WithSchemaVersion(1),
		)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.tenant_merge.event_failed",
				"create TenantMerged event",
			)
		}
		return []event.Event{evt}, nil
	}
}

// decideMergeTenantInto tombstones the tenant stream it targets as merged
// into SurvivorID. It is the LAST step of a tenant merge: until it lands the
// tenant is still live, so a failed merge is retried rather than left
// half-dead.
func decideMergeTenantInto(
	aggID id.StreamID,
	survivorID identitymodel.TenantID,
	reason string,
) func(TenantState, event.Version) ([]event.Event, error) {
	return func(state TenantState, version event.Version) ([]event.Event, error) {
		if state.IsMerged() && state.MergedInto.Get() == survivorID.Get() {
			return nil, nil
		}
		if state.IsMerged() {
			return nil, errorfamily.NewConflict(
				"usermgmt.tenant_merge_into.already_merged",
				"tenant was already merged into another tenant",
			).WithContext("merged_into", state.MergedInto.Get())
		}
		if state.Deleted {
			return nil, errorfamily.NewConflict(
				"usermgmt.tenant_merge_into.deleted",
				"only a live tenant can be merged away",
			)
		}
		if !state.Exists() {
			return nil, errorfamily.NewRejection(
				"usermgmt.tenant_merge_into.not_found",
				"tenant does not exist",
			)
		}
		if survivorID.IsZero() {
			return nil, errorfamily.NewRejection(
				"usermgmt.tenant_merge_into.survivor_required",
				"survivor tenant ID is required",
			)
		}
		if survivorID.Get() == aggID.Get() {
			return nil, errorfamily.NewRejection(
				"usermgmt.tenant_merge_into.self_merge",
				"a tenant cannot be merged into itself",
			)
		}
		evt, err := event.New(
			eventTenantMergedInto, aggID, aggregateTypeTenant, version.Increment(),
			identitymodel.TenantMergedIntoPayload{
				SchemaVersion: currentSchemaVersion,
				SurvivorID:    survivorID,
				Reason:        reason,
			},
			event.WithCodec(codec.JSONCodec{}), event.WithSchemaVersion(1),
		)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.tenant_merge_into.event_failed",
				"create TenantMergedInto event",
			)
		}
		return []event.Event{evt}, nil
	}
}
