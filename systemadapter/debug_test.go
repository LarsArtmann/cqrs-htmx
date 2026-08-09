package systemadapter_test

import (
	"context"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	systemadapter "github.com/larsartmann/cqrs-htmx/systemadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

func TestDebug_AllPolicies(t *testing.T) {
	ctx := context.Background()
	sys := setupDeclarativeSystem(t)
	defer func() { _ = sys.Close() }()

	userStreamID := id.NewStreamID()
	tenantID := identitymodel.NewTenantID("tenant-debug")
	actorID := identitymodel.NewActorID(identitymodel.ActorUser, userStreamID.String())

	if err := sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewRegisterUserCmd(
		userStreamID, "debug@example.com", "Debug",
		[]identitymodel.Role{identitymodel.RoleUser},
	)); err != nil {
		t.Fatal(err)
	}

	if err := sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewAddMemberCmd(
		actorID, tenantID,
		[]identitymodel.Role{identitymodel.RoleAdmin},
	)); err != nil {
		t.Fatal(err)
	}

	waitForProjections(t, sys, 5e9)

	// Dump ALL policies
	all, err := system.Find[systemadapter.PolicyEntry](ctx, sys, "authz_policies")
	if err != nil {
		t.Fatalf("Find all: %v", err)
	}
	t.Logf("total policies: %d", len(all))
	for i, p := range all {
		t.Logf("  [%d] Key=%s Subject=%s Domain=%s Roles=%v", i, p.Key, p.Subject, p.Domain, p.Roles)
	}

	// Try with filters
	filtered, err := system.Find[systemadapter.PolicyEntry](ctx, sys, "authz_policies",
		system.Where("Subject", userStreamID.String()))
	if err != nil {
		t.Fatalf("Find filtered: %v", err)
	}
	t.Logf("filtered by Subject=%s: %d results", userStreamID.String(), len(filtered))

	// Try the query helper
	entries, err := systemadapter.FindPolicies(ctx, sys, userStreamID.String(), tenantID.Get())
	if err != nil {
		t.Fatalf("FindPolicies: %v", err)
	}
	t.Logf("FindPolicies: %d results", len(entries))
}
