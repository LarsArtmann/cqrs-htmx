package usermgmt

import (
	"context"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// StartProjections replays historical events from the journal into all
// registered projections, then subscribes to live events from the bus.
//
// Uses projectionhost.Host from go-cqrs-lite for projection lifecycle management:
// per-projection goroutines, per-projection checkpoints, retry with dead-letter
// queue, and crash auto-restart with backoff. This replaces the former
// hand-rolled replay+dedup+live-handler logic (~155 LOC).
//
// Read-your-writes: the function blocks until all projections have finished
// their initial journal drain (reached live state) before returning. After
// StartProjections returns, all read models reflect all historical events.
//
// Checkpoint: when cpStore is non-nil, each projection resumes from its own
// last checkpoint (keyed by projection Name()). When nil, an in-memory
// checkpoint store is used (checkpoints are lost on restart — full replay each
// time). Note: per-projection checkpoint keys replace the former single
// "usermgmt:start_projections" key; existing checkpoint data is incompatible
// and will be ignored (one-time full replay on first run after upgrade).
//
// The returned *projectionhost.Host must be stopped on shutdown (call Stop()).
func StartProjections(
	journal event.Journal,
	bus event.Subscriber,
	cpStore event.CheckpointStore,
	readModel projection.Projection,
	membershipReadModel projection.Projection,
	tenantReadModel projection.Projection,
	botReadModel projection.Projection,
	casbinProjection *CasbinProjection,
	auditLog *AuditLog,
) (*projectionhost.Host, error) {
	seekable, ok := journal.(event.SeekableJournal)
	if !ok {
		return nil, errorfamily.NewRejection(
			"usermgmt.projection.journal_not_seekable",
			"projectionhost requires a SeekableJournal (ReadFrom); "+
				"the event store does not implement event.SeekableJournal",
		)
	}

	store := cpStore
	if store == nil {
		store = memory.NewMemoryCheckpointStore()
	}

	host, err := projectionhost.New(seekable, store,
		projectionhost.WithSubscriber(bus),
		projectionhost.WithDeadLetterStore(projectionhost.NewMemoryDeadLetterStore(), 0),
	)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err,
			"usermgmt.projection.host_create_failed",
			"create projection host")
	}

	for _, p := range collectProjections(
		readModel, membershipReadModel, tenantReadModel, botReadModel, casbinProjection, auditLog,
	) {
		if err := host.Register(p); err != nil {
			return nil, errorfamily.WrapInfrastructure(err,
				"usermgmt.projection.register_failed",
				"register projection "+p.Name())
		}
	}

	if err := host.Start(context.Background()); err != nil {
		return nil, errorfamily.WrapInfrastructure(err,
			"usermgmt.projection.start_failed",
			"start projection host")
	}

	if err := waitForDrain(host); err != nil {
		_ = host.Stop()
		return nil, err
	}

	return host, nil
}

// collectProjections gathers all projection implementations into a slice.
// The read model and casbin projection are always present; the rest are optional.
func collectProjections(
	readModel projection.Projection,
	membershipReadModel projection.Projection,
	tenantReadModel projection.Projection,
	botReadModel projection.Projection,
	casbinProjection *CasbinProjection,
	auditLog *AuditLog,
) []projection.Projection {
	projections := []projection.Projection{readModel, casbinProjection}
	if membershipReadModel != nil {
		projections = append(projections, membershipReadModel)
	}
	if tenantReadModel != nil {
		projections = append(projections, tenantReadModel)
	}
	if botReadModel != nil {
		projections = append(projections, botReadModel)
	}
	if auditLog != nil {
		projections = append(projections, auditLog)
	}

	return projections
}

// waitForDrain blocks until all projection host workers have finished their
// initial journal drain, or one has failed. This preserves read-your-writes:
// after this returns, all projections have processed all historical events.
//
// The watermill EventBus implements SubscribeAll as a non-blocking registration
// (it registers the handler and returns immediately). This means projectionhost
// workers transition through WorkerLive momentarily, then exit to WorkerStopped
// once SubscribeAll returns — the live handler is registered and active, but
// the worker goroutine has exited. Both WorkerLive and WorkerStopped are valid
// drain-complete terminal states for non-blocking subscribers.
func waitForDrain(host *projectionhost.Host) error {
	const (
		pollInterval = 10 * time.Millisecond
		drainTimeout  = 30 * time.Second
	)

	timer := time.NewTimer(drainTimeout)
	defer timer.Stop()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			statuses := host.Status()
			allDone := true
			for _, s := range statuses {
				switch s.Status {
				case projectionhost.WorkerLive, projectionhost.WorkerStopped:
					// Worker has completed drain and registered live handler.
				case projectionhost.WorkerFailed:
					return errorfamily.NewInfrastructure(
						"usermgmt.projection.worker_failed",
						fmt.Sprintf("projection %q failed during initial drain: %s", s.Name, s.LastError),
					)
				default:
					// WorkerIdle, WorkerRunning, WorkerBackoff, WorkerDraining:
					// still working or hasn't started yet.
					allDone = false
				}
			}
			if allDone {
				return nil
			}
		case <-timer.C:
			return errorfamily.NewTransient(
				"usermgmt.projection.drain_timeout",
				fmt.Sprintf("projection drain timed out after %s", drainTimeout),
			)
		}
	}
}
