package systemadapter

import (
	"context"
	"errors"
	"fmt"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	memory "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

const (
	maxRestarts       = 3
	dlqThreshold      = 10
	backoffMin        = 100 * time.Millisecond
	backoffMax        = 5 * time.Second
	drainPollInterval = 10 * time.Millisecond
)

// ProjectionLayer manages a dedicated projectionhost.Host for usermgmt read models,
// Casbin authz, and the audit log. It is backed by the system's event infrastructure
// (journal for catch-up, bus for live events) but runs independently of the system's
// internal metaengine projection host.
//
// This design lets consumers use system.New() for CQRS infrastructure (deciders,
// command/query dispatch, event store, bus) while still getting the full suite of
// usermgmt projections for queries, authorization, and audit.
//
// Lifecycle:
//
//	sys, _ := system.New(ctx, systemadapter.DomainConfig(), deployment)
//	pl, _ := systemadapter.NewProjectionLayer(sys)
//	pl.Start(ctx)   // start projection workers
//	sys.Start(ctx)  // start system (bus listeners, etc.)
//	// ... dispatch commands, query read models ...
//	pl.Stop()       // stop projections
//	sys.Close()     // close system
type ProjectionLayer struct {
	Host       *projectionhost.Host
	User       *usermgmt.UserReadModel
	Membership *usermgmt.MembershipReadModel
	Tenant     *usermgmt.TenantReadModel
	Bot        *usermgmt.BotReadModel
	Casbin     *usermgmt.CasbinProjection
	Authz      *identitymodel.Authz
	AuditLog   *usermgmt.AuditLog
}

// NewProjectionLayer creates a ProjectionLayer backed by the system's event store
// and bus. It creates all standard read models, the Casbin authz projection, and
// the audit log, then registers them with a new projectionhost.Host.
//
// The host is NOT started — call Start(ctx) after creation.
// Must be called AFTER system.New() but BEFORE sys.Start() (so the bus is ready).
func NewProjectionLayer(sys *system.System) (*ProjectionLayer, error) {
	store := sys.EventStore()

	if store == nil {
		return nil, errors.New("systemadapter: system has no event store")
	}

	journal, ok := store.(event.SeekableJournal)

	if !ok {
		return nil, fmt.Errorf("systemadapter: event store does not implement SeekableJournal (got %T)", store)
	}

	bus := sys.Bus()

	if bus == nil {
		return nil, errors.New("systemadapter: system has no event bus")
	}

	authz, err := usermgmt.NewAuthz()
	if err != nil {
		return nil, fmt.Errorf("systemadapter: create authz: %w", err)
	}

	casbinProj, err := usermgmt.NewCasbinProjection(authz)
	if err != nil {
		return nil, fmt.Errorf("systemadapter: create casbin projection: %w", err)
	}

	pl := &ProjectionLayer{
		Host:       nil,
		User:       usermgmt.NewUserReadModel(),
		Membership: usermgmt.NewMembershipReadModel(),
		Tenant:     usermgmt.NewTenantReadModel(),
		Bot:        usermgmt.NewBotReadModel(),
		Casbin:     casbinProj,
		Authz:      authz,
		AuditLog:   usermgmt.NewAuditLog(),
	}

	cpStore := memory.NewMemoryCheckpointStore()
	dlqStore := projectionhost.NewMemoryDeadLetterStore()

	host, err := projectionhost.New(journal, cpStore,
		projectionhost.WithSubscriber(bus),
		projectionhost.WithDeadLetterStore(dlqStore, dlqThreshold),
		projectionhost.WithMaxRestarts(maxRestarts),
		projectionhost.WithBackoff(backoffMin, backoffMax),
	)
	if err != nil {
		return nil, fmt.Errorf("systemadapter: create projection host: %w", err)
	}

	projections := []projection.Projection{
		pl.User,
		pl.Membership,
		pl.Tenant,
		pl.Bot,
		pl.Casbin,
		pl.AuditLog,
	}

	for _, p := range projections {
		if err := host.Register(p); err != nil {
			return nil, fmt.Errorf("systemadapter: register projection %q: %w", p.Name(), err)
		}
	}

	pl.Host = host

	return pl, nil
}

// Start launches the projection host workers. Must be called once after
// NewProjectionLayer and before dispatching commands.
func (pl *ProjectionLayer) Start(ctx context.Context) error {
	if err := pl.Host.Start(ctx); err != nil {
		return fmt.Errorf("systemadapter: start projection host: %w", err)
	}

	return nil
}

// Stop gracefully stops the projection host. Safe to call multiple times.
func (pl *ProjectionLayer) Stop() error {
	if err := pl.Host.Stop(); err != nil {
		return fmt.Errorf("systemadapter: stop projection host: %w", err)
	}

	return nil
}

// WaitForDrain blocks until all projection workers have processed all published
// events (reached WorkerLive or WorkerStopped), or timeout expires. This provides
// read-your-writes consistency after command dispatch.
func (pl *ProjectionLayer) WaitForDrain(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		states := pl.Host.Status()
		allReady := true

		for _, s := range states {
			if s.Status != projectionhost.WorkerLive && s.Status != projectionhost.WorkerStopped {
				allReady = false

				break
			}
		}

		if allReady && len(states) > 0 {
			return nil
		}

		time.Sleep(drainPollInterval)
	}

	return fmt.Errorf("systemadapter: projection drain timed out after %s", timeout)
}
