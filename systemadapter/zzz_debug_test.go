package systemadapter_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	systemadapter "github.com/larsartmann/cqrs-htmx/systemadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

func TestZ_Debug_AuthzAfterAll(t *testing.T) {
	ctx := context.Background()
	sys := setupDeclarativeSystem(t)
	defer func() { _ = sys.Close() }()

	userStreamID := id.NewStreamID()
	tenantID := identitymodel.NewTenantID("tenant-authz")
	actorID := identitymodel.NewActorID(identitymodel.ActorUser, userStreamID.String())

	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewRegisterUserCmd(
		userStreamID, "admin@example.com", "Admin User",
		[]identitymodel.Role{identitymodel.RoleUser},
	)))
	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewAddMemberCmd(
		actorID, tenantID, []identitymodel.Role{identitymodel.RoleAdmin},
	)))

	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		all, _ := system.Find[systemadapter.PolicyEntry](ctx, sys, "authz_policies")
		allowed, err := systemadapter.Enforce(ctx, sys, userStreamID.String(), tenantID.Get(), "manage")
		states := sys.ProjectionHost().Status()
		wInfo := ""
		for _, s := range states {
			wInfo = fmt.Sprintf("status=%s proc=%d err=%d", s.Status, s.Processed, s.Errors)
		}
		t.Logf("[%d] policies=%d enforce=%v enforceErr=%v worker=[%s]", i, len(all), allowed, err, wInfo)
		if allowed {
			return
		}
	}
	t.Fatal("never succeeded")
}
