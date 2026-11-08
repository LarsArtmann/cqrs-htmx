package systemadapter_test

import (
	"context"
	"testing"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	systemadapter "github.com/larsartmann/cqrs-htmx/systemadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	_ "github.com/larsartmann/go-cqrs-lite/metaengine/bboltengine/v4" // register bbolt driver (the one light enough to construct in this suite — see presets.go for the other engines' imports)
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// TestRecommendedBreadthPresets_Shape pins the config shape of the five
// engine-breadth presets: one engine holding every role, the DSN plumbed
// verbatim, and no SQLite pragmas leaking into non-SQLite engines.
func TestRecommendedBreadthPresets_Shape(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		preset func() system.DeploymentConfig
		driver string
		dsn    string
	}{
		{
			name:   "pebble",
			preset: func() system.DeploymentConfig { return systemadapter.RecommendedPebbleDeployment("./data/pebble") },
			driver: "pebble",
			dsn:    "./data/pebble",
		},
		{
			name:   "bbolt",
			preset: func() system.DeploymentConfig { return systemadapter.RecommendedBBoltDeployment("app.db") },
			driver: "bbolt",
			dsn:    "app.db",
		},
		{
			name: "postgres",
			preset: func() system.DeploymentConfig {
				return systemadapter.RecommendedPostgresDeployment("postgres://localhost:5432/app?sslmode=require")
			},
			driver: "postgres",
			dsn:    "postgres://localhost:5432/app?sslmode=require",
		},
		{
			name:   "duckdb",
			preset: func() system.DeploymentConfig { return systemadapter.RecommendedDuckDBDeployment("analytics.db") },
			driver: "duckdb",
			dsn:    "analytics.db",
		},
		{
			name: "turso",
			preset: func() system.DeploymentConfig {
				return systemadapter.RecommendedTursoDeployment("libsql://db.turso.io")
			},
			driver: "turso",
			dsn:    "libsql://db.turso.io",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dep := tc.preset()

			eng, ok := dep.Engines["primary"]
			if !ok {
				t.Fatalf("Engines missing the single %q engine", "primary")
			}
			if eng.Driver != tc.driver {
				t.Errorf("Driver = %q, want %q", eng.Driver, tc.driver)
			}
			if eng.DSN != tc.dsn {
				t.Errorf("DSN = %q, want %q", eng.DSN, tc.dsn)
			}
			if len(eng.Pragmas) != 0 {
				t.Errorf("Pragmas = %v, want none (SQLite-only concern)", eng.Pragmas)
			}

			if len(dep.Instances) != 2 {
				t.Fatalf(
					"Instances = %d, want 2 (one engine holds every role: source-of-truth + the explicit projections instance that keeps views off the implicit volatile fallback)",
					len(dep.Instances),
				)
			}
			if dep.Instances[0].Role != system.RoleSourceOfTruth {
				t.Errorf("Role = %q, want %q", dep.Instances[0].Role, system.RoleSourceOfTruth)
			}
			if dep.Instances[1].Role != system.RoleProjections {
				t.Errorf(
					"Role = %q, want %q (explicit projections placement)",
					dep.Instances[1].Role,
					system.RoleProjections,
				)
			}
		})
	}
}

// TestRecommendedBBoltDeployment_EndToEnd proves a breadth preset composes
// with system.New end to end: bbolt is the construction proof for the preset
// family (dependency-light and pure Go; pebble/postgres/duckdb/turso carry
// heavy trees that belong in the consumer's go.mod, not this module's).
func TestRecommendedBBoltDeployment_EndToEnd(t *testing.T) {
	ctx := context.Background()

	path := t.TempDir() + "/breadth.db"
	deployment := systemadapter.RecommendedBBoltDeployment(path)

	sys, err := system.New(ctx, systemadapter.DomainConfig(), deployment)
	if err != nil {
		t.Fatalf("system.New with bbolt preset failed: %v", err)
	}

	defer func() { _ = sys.Close() }()

	pl, err := systemadapter.NewProjectionLayer(sys)
	if err != nil {
		t.Fatalf("NewProjectionLayer failed: %v", err)
	}

	defer func() { _ = pl.Stop() }()

	if err := pl.Start(ctx); err != nil {
		t.Fatalf("ProjectionLayer.Start failed: %v", err)
	}

	streamID := id.NewStreamID()
	cmd := identitymodel.NewRegisterUserCmd(streamID, "bbolt-breadth@example.com", "BBolt Breadth", nil)
	if err := sys.CommandDispatcher().Dispatch(ctx, cmd); err != nil {
		t.Fatalf("Dispatch RegisterUser failed: %v", err)
	}

	if err := pl.WaitForDrain(10 * time.Second); err != nil {
		t.Fatalf("WaitForDrain failed: %v", err)
	}

	user, ok := pl.User.FindByEmail("bbolt-breadth@example.com")
	if !ok {
		t.Fatal("user not found in the bbolt-backed read model after drain")
	}
	if user.ID.String() != streamID.Get() {
		t.Errorf("user ID = %q, want %q", user.ID.String(), streamID.Get())
	}
}
