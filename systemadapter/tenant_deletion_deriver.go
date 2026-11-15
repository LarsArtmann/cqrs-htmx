package systemadapter

import (
	"context"
	"fmt"
	"log/slog"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/deriver/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
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
		tenantID := identitymodel.NewTenantID(evt.StreamID().String())

		members, err := FindMembershipsByTenant(ctx, sys, tenantID.Get())
		if err != nil {
			return nil, fmt.Errorf("systemadapter: tenant-deletion cascade: list memberships for tenant %s: %w", tenantID.Get(), err)
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
