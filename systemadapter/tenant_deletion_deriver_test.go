package systemadapter_test

import (
	"context"
	"testing"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	systemadapter "github.com/larsartmann/cqrs-htmx/systemadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// TestTenantDeletionDeriver_RemovesEveryMember is the parity test for the
// TenantDeleted → RemoveMember cascade (ADR-0154): deleting a tenant derives
// one RemoveMember per member — read models retire, other tenants are
// untouched, and an empty tenant derives nothing. The legacy
// usermgmt.Service loop performs the equivalent cleanup synchronously; this
// pins the promoted, journal-visible form.
func TestTenantDeletionDeriver_RemovesEveryMember(t *testing.T) {
	ctx, sys := startDeclarativeSystem(t)

	dispatch := func(cmd any) {
		t.Helper()

		if err := sys.CommandDispatcher().Dispatch(ctx, cmd); err != nil {
			t.Fatalf("dispatch %T: %v", cmd, err)
		}
	}

	userA := identitymodel.ActorIDFromUser(mustUserID(t, id.NewStreamID().String()))
	userB := identitymodel.ActorIDFromUser(mustUserID(t, id.NewStreamID().String()))
	otherUser := identitymodel.ActorIDFromUser(mustUserID(t, id.NewStreamID().String()))

	tenant := identitymodel.NewTenantID(id.NewStreamID().String())
	otherTenant := identitymodel.NewTenantID(id.NewStreamID().String())

	dispatch(identitymodel.NewCreateTenantCmd(id.ParseStreamIDOrPanic(tenant.Get()), "doomed", "Doomed"))
	dispatch(identitymodel.NewCreateTenantCmd(id.ParseStreamIDOrPanic(otherTenant.Get()), "survivor", "Survivor"))

	dispatch(identitymodel.NewAddMemberCmd(userA, tenant, []identitymodel.Role{identitymodel.RoleAdmin}))
	dispatch(identitymodel.NewAddMemberCmd(userB, tenant, nil))
	dispatch(identitymodel.NewAddMemberCmd(otherUser, otherTenant, nil))

	if members, err := systemadapter.FindMembershipsByTenant(ctx, sys, tenant.Get()); err != nil || len(members) != 2 {
		t.Fatalf("precondition: tenant members = %d (err %v), want 2", len(members), err)
	}

	dispatch(identitymodel.NewDeleteTenantCmd(id.ParseStreamIDOrPanic(tenant.Get()), "cascade test"))

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
		identitymodel.NewCreateTenantCmd(id.ParseStreamIDOrPanic(tenant.Get()), "empty", "Empty")); err != nil {
		t.Fatalf("dispatch CreateTenant: %v", err)
	}

	if err := sys.CommandDispatcher().Dispatch(ctx,
		identitymodel.NewDeleteTenantCmd(id.ParseStreamIDOrPanic(tenant.Get()), "")); err != nil {
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
