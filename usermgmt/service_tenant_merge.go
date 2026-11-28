package usermgmt

import (
	"context"
	"errors"
	"slices"
	"strings"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// withDispatchContext attaches merge-site context to a decider's typed error
// without reclassifying it: the family IS the downstream HTTP semantics
// (errorfamily.Classify maps Rejection/Conflict/Transient to statuses), so
// this is C033's "wrap with context for debuggability" done in the
// family-preserving form. A non-event.Error passes through unchanged — there
// is no family to preserve and inventing one would lie about the failure.
func withDispatchContext(err error, kv ...string) error {
	ee, ok := errors.AsType[*event.Error](err)
	if !ok {
		return err
	}
	for i := 0; i+1 < len(kv); i += 2 {
		ee = ee.WithContextAny(kv[i], kv[i+1])
	}
	return ee
}

// RenameTenant changes a tenant's display name. The tenant ID never moves:
// it is the stream identity, the Casbin domain and the key of every
// tenant-scoped row, which is exactly why a typo is fixed by rename and not
// by delete-and-recreate (issue #47).
func (s *TenantService) RenameTenant(ctx context.Context, tenantID TenantID, displayName string) error {
	aggID, err := aggIDFromTenant(tenantID)
	if err != nil {
		return err
	}
	return s.dispatcher.Dispatch( //nolint:wrapcheck // decider returns typed domain errors
		ctx, identitymodel.NewRenameTenantCmd(aggID, displayName),
	)
}

// MergeTenants consolidates duplicate tenants into the survivor: one atomic
// TenantMerged record on the survivor's stream, then the roster moves through
// ordinary membership commands (role-set union, survivor side first), and the
// loser tombstones land LAST — until they do, a failed merge is retried, never
// left half-dead. Re-runs are idempotent: absorbed losers are recognised and
// skipped.
func (s *TenantService) MergeTenants(
	ctx context.Context, survivorID TenantID, loserIDs []TenantID, reason string,
) error {
	if err := validateMergeTenantsInput(survivorID, loserIDs); err != nil {
		return err
	}

	survivorAggID, err := aggIDFromTenant(survivorID)
	if err != nil {
		return err
	}
	survivorState, _, err := s.tenantRepo.LoadRef(ctx, id.NewStreamRef(aggregateTypeTenant, survivorAggID))
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "usermgmt.merge_tenants.load_failed", "load survivor state")
	}
	if err := requireLiveTenantSurvivor(survivorState); err != nil {
		return err
	}

	liveLosers, err := s.collectLiveTenantLosers(ctx, survivorID, loserIDs)
	if err != nil {
		return err
	}
	if len(liveLosers) == 0 {
		return nil
	}

	recordCmd := identitymodel.NewMergeTenantsCmd(survivorAggID, liveLosers, reason)
	if err := s.dispatcher.Dispatch(ctx, recordCmd); err != nil {
		return withDispatchContext(err, "operation", "merge_tenants.record", "survivor_id", survivorID.Get())
	}

	for _, loserID := range liveLosers {
		if err := s.moveTenantRoster(ctx, survivorID, loserID); err != nil {
			return err
		}
	}
	return s.retireMergeTenantLosers(ctx, survivorID, liveLosers, reason)
}

// validateMergeTenantsInput refuses malformed tenant merges before anything is
// loaded or dispatched.
func validateMergeTenantsInput(survivorID TenantID, loserIDs []TenantID) error {
	if survivorID.IsZero() {
		return errorfamily.NewRejection("usermgmt.merge_tenants.survivor_required", "survivor tenant ID is required")
	}
	if len(loserIDs) == 0 {
		return errorfamily.NewRejection("usermgmt.merge_tenants.no_losers", "a merge needs at least one loser")
	}
	for i, loser := range loserIDs {
		if loser.Get() == survivorID.Get() {
			return errorfamily.NewRejection(
				"usermgmt.merge_tenants.self_merge", "a tenant cannot be merged into itself",
			).WithContextAny("loser_id", loser.Get())
		}
		if slices.ContainsFunc(loserIDs[i+1:], func(other TenantID) bool { return other.Get() == loser.Get() }) {
			return errorfamily.NewRejection(
				"usermgmt.merge_tenants.duplicate_loser", "the loser list names the same tenant twice",
			).WithContextAny("loser_id", loser.Get())
		}
	}
	return nil
}

// collectLiveTenantLosers loads every loser's fold state, answers the typed
// rejections and conflicts up front, skips losers this survivor already
// absorbed, and returns the losers that still need merging.
func (s *TenantService) collectLiveTenantLosers(
	ctx context.Context, survivorID TenantID, loserIDs []TenantID,
) ([]TenantID, error) {
	liveLosers := make([]TenantID, 0, len(loserIDs))
	for _, loserID := range loserIDs {
		loserAggID, err := aggIDFromTenant(loserID)
		if err != nil {
			return nil, err
		}
		state, _, err := s.tenantRepo.LoadRef(ctx, id.NewStreamRef(aggregateTypeTenant, loserAggID))
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err, "usermgmt.merge_tenants.load_failed", "load loser state",
			).WithContextAny("loser_id", loserID.Get())
		}
		if state.IsMerged() {
			if s.tenantChainResolvesTo(ctx, state.MergedInto, survivorID) {
				continue
			}
			return nil, errorfamily.NewConflict(
				"usermgmt.merge_tenants.loser_merged", "loser was already merged into another tenant",
			).
				WithContext("merged_into", state.MergedInto.Get()).
				WithContextAny("loser_id", loserID.Get())
		}
		if !state.Exists() {
			if state.Deleted {
				return nil, errorfamily.NewConflict(
					"usermgmt.tenant_merge_into.deleted", "only a live tenant can be merged away",
				).WithContextAny("loser_id", loserID.Get())
			}
			return nil, errorfamily.NewRejection(
				"usermgmt.merge_tenants.loser_not_found", "loser was never created",
			).WithContextAny("loser_id", loserID.Get())
		}
		liveLosers = append(liveLosers, loserID)
	}
	return liveLosers, nil
}

// moveTenantRoster gives the survivor every membership the loser tenant
// holds: the UNION of both role sets per actor, the survivor's add or update
// first, the loser's removal after. Re-runs dispatch nothing.
func (s *TenantService) moveTenantRoster(ctx context.Context, survivorID, loserID TenantID) error {
	for _, mem := range s.membershipReadModel.FindByTenant(loserID.Get()) {
		var union []Role
		found := false
		for _, existing := range s.membershipReadModel.FindByActor(mem.ActorID.String()) {
			if existing.TenantID.Get() == survivorID.Get() {
				union, found = existing.Roles, true
				break
			}
		}
		merged := unionRoles(union, mem.Roles)
		if found {
			update := NewUpdateMemberRolesCmd(mem.ActorID, survivorID, merged)
			if err := s.dispatcher.Dispatch(ctx, update); err != nil {
				return errorfamily.Wrapf(err, event.Conflict, "usermgmt.merge_tenants.roster_move_failed",
					"move one member to the survivor").WithContextAny("loser_id", loserID.Get()).
					WithContextAny("survivor_id", survivorID.Get()).
					WithContextAny("actor_id", mem.ActorID.String())
			}
		} else {
			if err := s.dispatcher.Dispatch(ctx, NewAddMemberCmd(mem.ActorID, survivorID, merged)); err != nil {
				return errorfamily.Wrapf(err, event.Conflict, "usermgmt.merge_tenants.roster_move_failed",
					"move one member to the survivor").WithContextAny("loser_id", loserID.Get()).
					WithContextAny("survivor_id", survivorID.Get()).
					WithContextAny("actor_id", mem.ActorID.String())
			}
		}
		if err := s.dispatcher.Dispatch(ctx, NewRemoveMemberCmd(mem.ActorID, loserID)); err != nil {
			return errorfamily.Wrapf(err, event.Conflict, "usermgmt.merge_tenants.roster_move_failed",
				"retire one member of the loser").WithContextAny("loser_id", loserID.Get()).
				WithContextAny("survivor_id", survivorID.Get()).
				WithContextAny("actor_id", mem.ActorID.String())
		}
	}
	return nil
}

// retireMergeTenantLosers tombstones every live loser — the LAST step of a
// tenant merge (mirrors retireMergeLosers).
func (s *TenantService) retireMergeTenantLosers(
	ctx context.Context, survivorID TenantID, liveLosers []TenantID, reason string,
) error {
	for _, loserID := range liveLosers {
		loserAggID, err := aggIDFromTenant(loserID)
		if err != nil {
			return err
		}
		tombstone := identitymodel.NewMergeTenantIntoCmd(loserAggID, survivorID, reason)
		if err := s.dispatcher.Dispatch(ctx, tombstone); err != nil {
			return withDispatchContext(err,
				"operation", "merge_tenant_into",
				"loser_id", loserID.Get(),
				"survivor_id", survivorID.Get(),
			)
		}
	}
	return nil
}

// tenantChainResolvesTo walks the MergedInto pointers from a merged-away
// tenant and reports whether the chain ends at the target (the tenant
// spelling of mergeChainResolvesTo).
func (s *TenantService) tenantChainResolvesTo(ctx context.Context, from, target TenantID) bool {
	current := from
	for range mergeChainHopLimit {
		if current.IsZero() {
			return false
		}
		if current.Get() == target.Get() {
			return true
		}
		aggID, err := aggIDFromTenant(current)
		if err != nil {
			return false
		}
		state, _, err := s.tenantRepo.LoadRef(ctx, id.NewStreamRef(aggregateTypeTenant, aggID))
		if err != nil || !state.IsMerged() {
			return false
		}
		current = state.MergedInto
	}
	return false
}

// DuplicateTenantGroup names one cluster of duplicate tenants: every live
// tenant whose normalized name matches.
type DuplicateTenantGroup struct {
	NormalizedName string    `json:"normalized_name"`
	Tenants        []*Tenant `json:"tenants"`
}

// DuplicateTenants reports every cluster of live tenants a human would call
// the same tenant: identical names after trimming and lowercasing. Each group
// carries the normalized name and the candidate rows; the admin picks the
// survivor and calls [Service.MergeTenants].
func (s *TenantService) DuplicateTenants(_ context.Context) []DuplicateTenantGroup {
	byName := make(map[string][]*Tenant)
	for _, tenant := range s.tenantReadModel.All() {
		key := identitymodel.NormalizeTenantName(tenant.Name)
		byName[key] = append(byName[key], tenant)
	}
	groups := make([]DuplicateTenantGroup, 0)
	for key, tenants := range byName {
		if len(tenants) > 1 {
			groups = append(groups, DuplicateTenantGroup{NormalizedName: key, Tenants: tenants})
		}
	}
	slices.SortFunc(groups, func(a, b DuplicateTenantGroup) int {
		return strings.Compare(a.NormalizedName, b.NormalizedName)
	})
	return groups
}
