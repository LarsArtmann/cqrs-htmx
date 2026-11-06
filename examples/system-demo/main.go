// Command system-demo demonstrates using go-cqrs-lite's system.New() composition root
// with cqrs-htmx's identity-model domain. This is the recommended way to get the full
// power of go-cqrs-lite's metaengine storage planner, introspection, safety checks,
// and driver registry alongside cqrs-htmx's event-sourced user management.
//
// Run: GOEXPERIMENT=jsonv2 go run .
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	systemadapter "github.com/larsartmann/cqrs-htmx/systemadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	_ "github.com/larsartmann/go-cqrs-lite/metaengine/bboltengine/v4" // register bbolt driver for -engine bbolt
	_ "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4" // register sqlite driver for -engine sqlite
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

func main() {
	ctx := context.Background()

	// 1. Pick a deployment preset — the operator-facing config as one call.
	//    The breadth presets (pebble/postgres/duckdb/turso) exist in
	//    systemadapter too; this demo links only the light drivers, so add
	//    the engine's blank import to use more.
	engine := flag.String("engine", "memory", "storage engine preset: memory, sqlite, bbolt")
	dsn := flag.String("dsn", "", "engine DSN: sqlite file (file:app.db) or bbolt path (app.db)")
	flag.Parse()

	var deployment system.DeploymentConfig

	switch *engine {
	case "memory":
		deployment = systemadapter.RecommendedMemoryDeployment()
	case "sqlite":
		if *dsn == "" {
			log.Fatalf("-engine sqlite requires -dsn (e.g. -dsn file:app.db)")
		}
		deployment = systemadapter.RecommendedSQLiteDeployment(*dsn)
	case "bbolt":
		if *dsn == "" {
			log.Fatalf("-engine bbolt requires -dsn (e.g. -dsn app.db)")
		}
		deployment = systemadapter.RecommendedBBoltDeployment(*dsn)
	default:
		log.Fatalf("unknown -engine %q (memory, sqlite, bbolt)", *engine)
	}

	fmt.Printf("Engine preset: %s\n", *engine)

	// 2. Get the pre-wired domain config: all 4 deciders, all 20 commands,
	//    and a TypeDecoder mapping all 21 event types to their payload structs.
	domain := systemadapter.DomainConfig()

	// 3. Create the system: auto-wires event store, command/query dispatchers,
	//    event bus, snapshot store, the declarative projection host (user/
	//    tenant/membership/bot read models, authz policies, audit log — all
	//    registered via DomainConfig.Projections), and runs safety checks.
	sys, err := system.New(ctx, domain, deployment)
	if err != nil {
		log.Fatalf("Failed to create system: %v", err)
	}
	defer sys.Close()

	// 4. Start the system — this starts the internal declarative projection
	//    host. There is no separate projection host to wire or stop.
	if err := sys.Start(ctx); err != nil {
		log.Fatalf("Failed to start system: %v", err)
	}

	// 5. Run some commands.
	runDemo(ctx, sys)

	// 6. Show system introspection.
	fmt.Println("\n=== System Topology ===")
	fmt.Println(sys.Explain(ctx))

	fmt.Println("\n=== System Health ===")
	fmt.Printf("Health: %s\n", sys.Health(ctx))

	// 7. Wait for shutdown signal.
	fmt.Println("\nPress Ctrl+C to stop...")
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nShutting down...")
}

func runDemo(ctx context.Context, sys *system.System) {
	disp := sys.CommandDispatcher()

	// Register a user
	userID := id.NewStreamID()
	fmt.Println("=== Registering User ===")
	if err := disp.Dispatch(ctx, identitymodel.NewRegisterUserCmd(
		userID, "alice@example.com", "Alice",
		[]identitymodel.Role{identitymodel.RoleUser},
	)); err != nil {
		log.Printf("RegisterUser failed: %v", err)
	}

	if user, ok := waitForView(func() (systemadapter.UserView, error) {
		return systemadapter.FindUserByID(ctx, sys, userID.String())
	}); ok {
		fmt.Printf("  User: %s <%s> (verified=%v)\n", user.DisplayName, user.Email, user.EmailVerified)
	}

	// Create a tenant
	tenantID := id.NewStreamID()
	fmt.Println("\n=== Creating Tenant ===")
	if err := disp.Dispatch(ctx, identitymodel.NewCreateTenantCmd(
		tenantID, "acme", "Acme Corporation",
	)); err != nil {
		log.Printf("CreateTenant failed: %v", err)
	}

	if tenant, ok := waitForView(func() (systemadapter.TenantView, error) {
		return systemadapter.FindTenantByID(ctx, sys, tenantID.String())
	}); ok {
		fmt.Printf("  Tenant: %s (%s) suspended=%v\n", tenant.DisplayName, tenant.Name, tenant.Suspended)
	}

	// Show audit log (declarative projection, queried like any other view)
	fmt.Println("\n=== Audit Log ===")
	entries, err := systemadapter.AuditEntries(ctx, sys)
	if err != nil {
		log.Printf("AuditEntries failed: %v", err)
	}
	for _, entry := range entries {
		fmt.Printf("  [%s] %s (aggregate=%s)\n", entry.EventType, entry.Action, entry.AggregateID)
	}
}

// waitForView polls a declarative read-model query until it succeeds or the
// deadline passes — the queryable equivalent of the deprecated
// ProjectionLayer.WaitForDrain.
func waitForView[V any](read func() (V, error)) (V, bool) {
	var zero V
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		view, err := read()
		if err == nil {
			return view, true
		}
		time.Sleep(10 * time.Millisecond) //cqrs-lint:ignore(P006) demo poll helper, bounded 5s deadline
	}
	return zero, false
}
