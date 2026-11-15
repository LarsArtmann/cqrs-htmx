package systemadapter_test

import (
	"context"
	"testing"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	systemadapter "github.com/larsartmann/cqrs-htmx/systemadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// TestTenantDeletionDeriver_RemovesEveryMember is the parity test for the
// TenantDeleted → RemoveMember cascade (ADR-0154): deleting a tenant derives
// one RemoveMember per member — read models retire, other tenants are
// untouched, and an empty tenant derives nothing. The legacy
// usermgmt.Service loop performs the equivalent cleanup synchronously; this
// pins the promoted, journal-visible form.
func TestTenantDeletionDeriver_RemovesEveryMember(t *testing.T) {
	ctx, sys := startDeclarativeSystem(t)

	dispatch := func(cmd command.Command) {
		t.Helper()

		if err := sys.CommandDispatcher().Dispatch(ctx, cmd); err != nil {
			t.Fatalf("dispatch %s: %v", cmd.Type(), err)
		}
	}

	userA := identitymodel.ActorIDFromUser(mustUserID(t, id.NewStreamID().String()))
	userB := identitymodel.ActorIDFromUser(mustUserID(t, id.NewStreamID().String()))
	otherUser := identitymodel.ActorIDFromUser(mustUserID(t, id.NewStreamID().String()))

	tenant := identitymodel.NewTenantID(id.NewStreamID().String())
	otherTenant := identitymodel.NewTenantID(id.NewStreamID().String())

	dispatch(identitymodel.NewCreateTenantCmd(mustStreamID(t, tenant.Get()), "doomed", "Doomed"))
	dispatch(identitymodel.NewCreateTenantCmd(mustStreamID(t, otherTenant.Get()), "survivor", "Survivor"))

	dispatch(identitymodel.NewAddMemberCmd(userA, tenant, []identitymodel.Role{identitymodel.RoleAdmin}))
	dispatch(identitymodel.NewAddMemberCmd(userB, tenant, nil))
	dispatch(identitymodel.NewAddMemberCmd(otherUser, otherTenant, nil))

	if members, err := systemadapter.FindMembershipsByTenant(ctx, sys, tenant.Get()); err != nil || len(members) != 2 {
		t.Fatalf("precondition: tenant members = %d (err %v), want 2", len(members), err)
	}

	dispatch(identitymodel.NewDeleteTenantCmd(mustStreamID(t, tenant.Get()), "cascade test"))

	deadline := time.Now().Add(5 * time.Second)
	for {
		members, err := systemadapter.FindMembershipsByTenant(ctx, sys, tenant.Get())
		if err != nil {
			t.Fatalf("FindMembershipsByTenant: %v", err)
		}

		if len(members) == 0 {
			break
		}

		if time.Now().After(deadline) {
			t.Fatalf("memberships for the deleted tenant still present after 5s: %d", len(members))
		}

		time.Sleep(10 * time.Millisecond)
	}

	survivors, err := systemadapter.FindMembershipsByTenant(ctx, sys, otherTenant.Get())
	if err != nil {
		t.Fatalf("FindMembershipsByTenant(other): %v", err)
	}

	if len(survivors) != 1 || survivors[0].ActorID != otherUser.String() {
		t.Errorf("other tenant's membership must survive the cascade, got %v", survivors)
	}
}

// TestTenantDeletionDeriver_EmptyTenantDerivesNothing pins the empty case:
// a tenant with no members deletes without deriving anything (no error, no
// phantom commands).
func TestTenantDeletionDeriver_EmptyTenantDerivesNothing(t *testing.T) {
	ctx, sys := startDeclarativeSystem(t)

	tenant := identitymodel.NewTenantID(id.NewStreamID().String())

	if err := sys.CommandDispatcher().Dispatch(ctx,
		identitymodel.NewCreateTenantCmd(mustStreamID(t, tenant.Get()), "empty", "Empty")); err != nil {
		t.Fatalf("dispatch CreateTenant: %v", err)
	}

	if err := sys.CommandDispatcher().Dispatch(ctx,
		identitymodel.NewDeleteTenantCmd(mustStreamID(t, tenant.Get()), "")); err != nil {
		t.Fatalf("dispatch DeleteTenant: %v", err)
	}

	if _, err := systemadapter.FindTenantByID(ctx, sys, tenant.Get()); err == nil {
		t.Error("tenant read model still resolves after deletion")
	}
}

func mustUserID(t *testing.T, raw string) identitymodel.UserID {
	t.Helper()

	userID, err := identitymodel.ParseUserID(raw)
	if err != nil {
		t.Fatalf("ParseUserID(%q): %v", raw, err)
	}

	return userID
}

func mustStreamID(t *testing.T, raw string) id.StreamID {
	t.Helper()

	streamID, err := id.ParseStreamID(raw)
	if err != nil {
		t.Fatalf("ParseStreamID(%q): %v", raw, err)
	}

	return streamID
}

// TestTenantDeletionDeriver_ColdBootSeedsRace pins the systemscenario boot
// pattern against the cascade: raw journal appends + bus publishes (Given
// seeding) with NO projection-readiness wait, then an immediate delete —
// the deriver must still see the seeded member because its catch-up wait
// lets the projection host drain through the trigger before it queries.
//
// A/B record (2026-11-15): with awaitProjectionsCaughtUp disabled, the
// generated cascade scenario in ../gfl-usermgmt fails under
// `go test -race -count=25` (the host has not folded the seeded MemberAdded
// when the un-waited deriver queries, so the cascade derives nothing); with
// it, everything is green. This unit test's single-member window is
// microsecond-scale — the host usually folds the two-event seed before the
// act dispatch lands — so it pins the boot pattern, not the race itself;
// the race itself is pinned by the generated suite under -race.
//
// Do not "amplify" this test by seeding a large tenant through one bulk
// Publish: a single 257-event Publish permanently strands deliveries past
// the live subscriber — verified 2026-11-15, the membership view froze at
// exactly 100 of 256 members WITH the catch-up wait present. Seeding must
// mirror the real Given path: per-step appends and publishes.
func TestTenantDeletionDeriver_ColdBootSeedsRace(t *testing.T) {
	ctx := context.Background()

	sys, err := system.New(ctx, systemadapter.DomainConfig(), systemadapter.RecommendedMemoryDeployment())
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	if err := sys.Start(ctx); err != nil {
		_ = sys.Close()

		t.Fatalf("system.Start: %v", err)
	}

	t.Cleanup(func() { _ = sys.Close() })

	tenant := identitymodel.NewTenantID(id.NewStreamID().String())
	actor := identitymodel.ActorIDFromUser(mustUserID(t, id.NewStreamID().String()))

	tenantRef := id.NewStreamRef(identitymodel.AggregateTypeTenant, mustStreamID(t, tenant.Get()))
	membershipRef := id.NewStreamRef(identitymodel.AggregateTypeMembership, identitymodel.DeriveMembershipID(actor, tenant))

	tenantCreated, err := event.New(identitymodel.EventTenantCreated, tenantRef.ID, tenantRef.Type, 1,
		identitymodel.TenantCreatedPayload{
			SchemaVersion: identitymodel.CurrentSchemaVersion,
			Name:          "cold",
			DisplayName:   "Cold",
		})
	if err != nil {
		t.Fatalf("mint TenantCreated: %v", err)
	}

	memberAdded, err := event.New(identitymodel.EventMemberAdded, membershipRef.ID, membershipRef.Type, 1,
		identitymodel.MemberAddedPayload{
			SchemaVersion: identitymodel.CurrentSchemaVersion,
			ActorKind:     actor.Kind().String(),
			ActorID:       actor.String(),
			TenantID:      tenant.Get(),
			Roles:         []identitymodel.Role{identitymodel.RoleAdmin},
		})
	if err != nil {
		t.Fatalf("mint MemberAdded: %v", err)
	}

	if err := sys.EventStore().AppendBatch(ctx, tenantRef, []event.Event{tenantCreated}); err != nil {
		t.Fatalf("append TenantCreated: %v", err)
	}

	if err := sys.EventStore().AppendBatch(ctx, membershipRef, []event.Event{memberAdded}); err != nil {
		t.Fatalf("append MemberAdded: %v", err)
	}

	if err := sys.Publisher().Publish(ctx, tenantCreated, memberAdded); err != nil {
		t.Fatalf("publish seeds: %v", err)
	}

	if err := sys.CommandDispatcher().Dispatch(ctx,
		identitymodel.NewDeleteTenantCmd(mustStreamID(t, tenant.Get()), "cold boot race")); err != nil {
		t.Fatalf("dispatch DeleteTenant: %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)

	for {
		members, err := systemadapter.FindMembershipsByTenant(ctx, sys, tenant.Get())
		if err != nil {
			t.Fatalf("FindMembershipsByTenant: %v", err)
		}

		if len(members) == 0 {
			return
		}

		if time.Now().After(deadline) {
			t.Fatalf("cold-boot seeded member still present after 5s: %d", len(members))
		}

		time.Sleep(10 * time.Millisecond)
	}
}
