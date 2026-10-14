package systemadapter

import (
	"context"
	"testing"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

func TestZZScratch_DumpTenantView(t *testing.T) {
	ctx, sys := startDeclarativeSystem(t)
	_ = ctx
	tenantID := identitymodel.NewStreamID()
	must(t, sys.CommandDispatcher().Dispatch(context.Background(), identitymodel.NewCreateTenantCmd(
		tenantID, "Acme Corp", "Acme",
	)))
	deadline := time.Now().Add(5 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		if _, err := AllTenants(context.Background(), sys); err == nil {
			tenants, _ := AllTenants(context.Background(), sys)
			if len(tenants) > 0 {
				t.Logf("TENANT ROWS: %d", len(tenants))
				for _, tv := range tenants {
					t.Logf("ROW ID=%q Name=%q", tv.ID, tv.Name)
				}
				t.Logf("CMD STREAM: Get=%q String=%q", tenantID.Get(), tenantID.String())
				return
			}
			lastErr = err
		} else {
			lastErr = err
		}
		time.Sleep(10 * time.Millisecond)
	}
	_, err := system.Find[any](context.Background(), sys, "tenant_by_id")
	t.Logf("direct lookup err: %v", err)
	if lastErr != nil {
		t.Fatalf("no rows after 5s: %v", lastErr)
	}
	t.Fatal("no rows after 5s")
}
