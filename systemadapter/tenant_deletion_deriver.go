package systemadapter

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/deriver/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// Catch-up bounds for the tenant-deletion cascade's read-model wait. The
// projection host folds bus events on its own goroutine, so the membership
// view lags delivery by microseconds; 5s is a pathological ceiling, not an
// expected wait.
const (
	cascadeCatchUpTimeout  = 5 * time.Second
	cascadeCatchUpInterval = 2 * time.Millisecond
)

// subscribeTenantDeletionDeriver wires the TenantDeleted → RemoveMember
// cascade as a bus deriver (ADR-0154): when a tenant is deleted, one
// RemoveMember command is derived per member so the membership read models
// retire alongside the tenant — the same cleanup the legacy
// usermgmt.Service performs synchronously in
// revokeMembershipsForTenantBestEffort, promoted into the event flow so
// journal-asserting specs can pin it (ThenCommands + Await).
//
// The sanctioned saga wiring: a synchronous deriver on sys.Bus() deadlocks
// (the derived dispatch publishes MemberRemoved on the same bus mid-delivery),
// so derived dispatches leave the handler goroutine via WithAsyncDispatch.
// Per-member failures log and continue — best-effort parity with the legacy
// loop: the tenant is already deleted, a partial cleanup must not vanish
// silently but also must not fail the cascade.
func subscribeTenantDeletionDeriver(sys *system.System) {
	dispatcher := sys.CommandDispatcher()

	cascade := deriver.Deriver(func(ctx context.Context, evt event.Event) ([]command.Command, error) {
		awaitProjectionsCaughtUp(ctx, sys, evt)

		tenantID := identitymodel.NewTenantID(evt.StreamID().Get())

		members, err := FindMembershipsByTenant(ctx, sys, tenantID.Get())
		if err != nil {
			return nil, fmt.Errorf(
				"systemadapter: tenant-deletion cascade: list memberships for tenant %s: %w",
				tenantID.Get(),
				err,
			)
		}

		commands := make([]command.Command, 0, len(members))

		for _, member := range members {
			kind, err := identitymodel.ActorKindFromString(member.ActorKind)
			if err != nil {
				return nil, fmt.Errorf("systemadapter: tenant-deletion cascade: member %q: %w", member.ActorID, err)
			}

			commands = append(commands, identitymodel.NewRemoveMemberCmd(
				identitymodel.NewActorID(kind, member.ActorID), tenantID))
		}

		return commands, nil
	})

	handler := cascade.AsHandler(dispatcher,
		deriver.WithAsyncDispatch(func(evt event.Event, cmd command.Command, err error) {
			slog.Default().Warn("systemadapter: derived RemoveMember dispatch failed on tenant deletion",
				"tenant_id", evt.StreamID().String(),
				"command", cmd.Type(),
				"error", err)
		}))

	if err := sys.Bus().Subscribe(identitymodel.EventTenantDeleted, handler); err != nil {
		panic("systemadapter: subscribe TenantDeleted cascade: " + err.Error())
	}
}

// awaitProjectionsCaughtUp blocks until every projection host worker has
// processed at least through trigger before the cascade reads its member
// view. The projection host folds bus events asynchronously on its own
// goroutine, so a deriver querying at delivery time races those folds — a
// tenant whose MemberAdded is still in flight would list zero members and
// the cascade would silently skip them. Each worker's checkpoint is the ID
// of the last event it processed, and event IDs are ULIDs minted monotonic
// within the process, so the lexical comparison is order-safe.
//
// Bounded by cascadeCatchUpTimeout: on expiry (or context cancellation) it
// logs and proceeds with whatever the view holds — the same best-effort
// posture as the legacy synchronous loop, never a hard failure.
func awaitProjectionsCaughtUp(ctx context.Context, sys *system.System, trigger event.Event) {
	host := sys.ProjectionHost()
	if host == nil {
		return
	}

	if len(host.Status()) == 0 {
		return
	}

	triggerID := trigger.ID().String()
	deadline := time.Now().Add(cascadeCatchUpTimeout)

	for time.Now().Before(deadline) {
		if allWorkersCaughtUp(host.Status(), triggerID) {
			return
		}

		select {
		case <-ctx.Done():
			logCascadeCatchUpAbandoned("context cancelled", triggerID)
			return
		case <-time.After(cascadeCatchUpInterval):
		}
	}

	logCascadeCatchUpAbandoned("timeout after "+cascadeCatchUpTimeout.String(), triggerID)
}

// allWorkersCaughtUp reports whether every worker's checkpoint has reached
// triggerID. An empty checkpoint (nothing processed yet) sorts before any
// ULID, so a fresh worker correctly reads as behind.
func allWorkersCaughtUp(states []projectionhost.WorkerState, triggerID string) bool {
	for _, workerState := range states {
		if workerState.Checkpoint < triggerID {
			return false
		}
	}

	return true
}

func logCascadeCatchUpAbandoned(reason, triggerID string) {
	slog.Default().
		Warn("systemadapter: tenant-deletion cascade: membership projection lagged the trigger; deriving from a possibly stale view",
			"reason", reason,
			"trigger_event", triggerID)
}
