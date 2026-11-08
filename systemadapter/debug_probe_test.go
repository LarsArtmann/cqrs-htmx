package systemadapter_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	systemadapter "github.com/larsartmann/cqrs-htmx/systemadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

func TestDebugSingleFileRestart(t *testing.T) {
	ctx := context.Background()
	dsn := "file:" + filepath.Join(t.TempDir(), "single.db")

	boot := func() (*system.System, error) {
		return system.New(ctx, systemadapter.DomainConfig(),
			systemadapter.RecommendedSQLiteDeployment(dsn))
	}

	sys1, err := boot()
	if err != nil { t.Fatal(err) }
	if err := sys1.Start(ctx); err != nil { _ = sys1.Close(); t.Fatal(err) }
	waitForProjectionsReady(t, sys1)
	must(t, sys1.CommandDispatcher().Dispatch(ctx, identitymodel.NewCreateTenantCmd(id.NewStreamID(), "Dbg Corp", "dbg")))
	eventually(t, 10 * time.Second, func() error { _, err := systemadapter.FindTenantByName(ctx, sys1, "Dbg Corp"); return err })
	t.Log("boot1 explain:", sys1.ProjectionExplain())
	_ = sys1.Close()

	sys2, err := boot()
	if err != nil { t.Fatal(err) }
	t.Cleanup(func() { _ = sys2.Close() })
	if err := sys2.Start(ctx); err != nil { t.Fatal(err) }
	waitForProjectionsReady(t, sys2)
	t.Log("boot2 explain:", sys2.ProjectionExplain())
	tenants, err := systemadapter.AllTenants(ctx, sys2)
	t.Logf("boot2 AllTenants: %d err=%v", len(tenants), err)
}

