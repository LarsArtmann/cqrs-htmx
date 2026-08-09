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

	"github.com/larsartmann/cqrs-htmx/identity-model/v4"
	systemadapter "github.com/larsartmann/cqrs-htmx/systemadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

func main() {
	ctx := context.Background()

	// 1. Define deployment: which engines and buses to use.
	//    This is the operator-facing config — pure data, loadable from YAML/env.
	deployment := system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "memory"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engines: []string{"primary"}},
		},
	}

	// 2. Get the pre-wired domain config: all 4 deciders, all 20 commands,
	//    and a TypeDecoder mapping all 21 event types to their payload structs.
	domain := systemadapter.DomainConfig()

	// 3. Create the system: auto-wires event store, command/query dispatchers,
	//    event bus, snapshot store, and runs safety checks.
	sys, err := system.New(ctx, domain, deployment)
	if err != nil {
		log.Fatalf("Failed to create system: %v", err)
	}
	defer sys.Close()

	// 4. Create projections: user/tenant/membership/bot read models, Casbin authz,
	//    and audit log. Backed by the system's event store + bus.
	projLayer, err := systemadapter.NewProjectionLayer(sys)
	if err != nil {
		log.Fatalf("Failed to create projection layer: %v", err)
	}
	defer projLayer.Stop()

	// 5. Start projections first (so they're ready for events), then start the system.
	if err := projLayer.Start(ctx); err != nil {
		log.Fatalf("Failed to start projections: %v", err)
	}

	// 6. Run some commands.
	runDemo(ctx, sys, projLayer)

	// 7. Show system introspection.
	fmt.Println("\n=== System Topology ===")
	fmt.Println(sys.Explain(ctx))

	fmt.Println("\n=== System Health ===")
	fmt.Printf("Health: %s\n", sys.Health(ctx))

	// 8. Wait for shutdown signal.
	fmt.Println("\nPress Ctrl+C to stop...")
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nShutting down...")
}

func runDemo(ctx context.Context, sys *system.System, pl *systemadapter.ProjectionLayer) {
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

	pl.WaitForDrain(5 * time.Second)

	user, ok := pl.User.FindByID(userID)
	if ok {
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

	pl.WaitForDrain(5 * time.Second)

	tenant, ok := pl.Tenant.FindByID(tenantID)
	if ok {
		fmt.Printf("  Tenant: %s (%s) suspended=%v\n", tenant.DisplayName, tenant.Name, tenant.Suspended)
	}

	// Show audit log
	fmt.Println("\n=== Audit Log ===")
	for _, entry := range pl.AuditLog.Entries() {
		fmt.Printf("  [%s] %s <%s>\n", entry.EventType, entry.Action, entry.Email)
	}
}
