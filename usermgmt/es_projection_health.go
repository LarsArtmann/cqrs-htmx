package usermgmt

import (
	"context"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// --- EventSourcedSetup methods ---

// EventCatalog returns a catalog populated with all 21 usermgmt event types.
// Use this with cqrshtmx.EventCatalogHandler to serve an event schema endpoint.
func (s *EventSourcedSetup) EventCatalog() *cqrshtmx.EventCatalog {
	return DefaultEventCatalog()
}

// ProjectionStatuses adapts the internal projectionhost.Host status to the
// cqrshtmx.ProjectionStatusProvider interface. This lets consumers pass the
// setup (or Service) directly to cqrshtmx.ProjectionStatusHandler.
func (s *EventSourcedSetup) ProjectionStatuses() []cqrshtmx.ProjectionStatusEntry {
	if s.projectionHost == nil {
		return nil
	}
	return adaptWorkerStates(s.projectionHost.Status())
}

// RebuildProjection stops the projection host, resets the named projection's
// checkpoint and read-model state, then creates a fresh host that replays
// the entire event journal from scratch. Blocks until all projections reach
// live state (read-your-writes preserved).
//
// Use the projection's Name() as the name argument (e.g. "user-read-model",
// "casbin-projection", "membership-read-model", "tenant-read-model",
// "bot-read-model", "audit-log").
//
// This is a maintenance operation — all projections briefly stop while the
// named one is rebuilt. The host lifecycle is managed internally: the old
// host is stopped and replaced with a new one.
func (s *EventSourcedSetup) RebuildProjection(ctx context.Context, name string) error {
	return rebuildProjection(ctx, &s.projectionHost, s.Store, s.Bus, s.checkpointStore, s.projections, name)
}

// --- Service methods ---

// EventCatalog returns a catalog populated with all 21 usermgmt event types.
// Use this with cqrshtmx.EventCatalogHandler to serve an event schema endpoint.
func (svc *Service) EventCatalog() *cqrshtmx.EventCatalog {
	return DefaultEventCatalog()
}

// ProjectionStatuses adapts the internal projectionhost.Host status to the
// cqrshtmx.ProjectionStatusProvider interface.
func (svc *Service) ProjectionStatuses() []cqrshtmx.ProjectionStatusEntry {
	if svc.projectionHost == nil {
		return nil
	}
	return adaptWorkerStates(svc.projectionHost.Status())
}

// RebuildProjection stops the projection host, resets the named projection's
// checkpoint and read-model state, then creates a fresh host that replays
// the entire event journal from scratch. Blocks until all projections reach
// live state (read-your-writes preserved).
func (svc *Service) RebuildProjection(ctx context.Context, name string) error {
	return rebuildProjection(ctx, &svc.projectionHost, svc.store, svc.bus, svc.checkpointStore, svc.projections, name)
}

// rebuildProjection is the shared body of EventSourcedSetup.RebuildProjection
// and Service.RebuildProjection. It stops the current host, resets the named
// projection's checkpoint + read-model, creates a fresh host that replays the
// entire journal, and stores the new host back through hostPtr. The two
// receivers differ only in field-name casing, so they delegate here.
func rebuildProjection(
	ctx context.Context,
	hostPtr **projectionhost.Host,
	store event.Store,
	bus event.Subscriber,
	cpStore event.CheckpointStore,
	projections []projection.Projection,
	name string,
) error {
	if *hostPtr == nil {
		return errorfamily.NewRejection(
			"usermgmt.rebuild.no_host",
			"no projection host configured",
		)
	}

	if err := (*hostPtr).Stop(); err != nil {
		return errorfamily.WrapTransient(err,
			"usermgmt.rebuild.stop_failed",
			"stop projection host for rebuild",
		)
	}

	if err := (*hostPtr).Reset(ctx, name); err != nil {
		return err
	}

	host, err := createProjectionHost(ctx, store, bus, cpStore, projections)
	if err != nil {
		return err
	}

	*hostPtr = host

	return nil
}

// createProjectionHost converts an event.Store to a journal and delegates to
// the shared startProjectionHost factory. Used by both EventSourcedSetup and
// Service rebuild methods.
func createProjectionHost(
	ctx context.Context,
	store event.Store,
	bus event.Subscriber,
	cpStore event.CheckpointStore,
	projections []projection.Projection,
) (*projectionhost.Host, error) {
	journal, err := journalFromStore(store)
	if err != nil {
		return nil, err
	}
	return startProjectionHost(ctx, journal, bus, cpStore, projections)
}

// adaptWorkerStates converts projectionhost.WorkerState slices to the
// root-module DTO. Lag is converted from time.Duration to milliseconds.
func adaptWorkerStates(workers []projectionhost.WorkerState) []cqrshtmx.ProjectionStatusEntry {
	result := make([]cqrshtmx.ProjectionStatusEntry, len(workers))
	for i, w := range workers {
		result[i] = cqrshtmx.ProjectionStatusEntry{
			Name:       w.Name,
			Status:     string(w.Status),
			Checkpoint: w.Checkpoint,
			Processed:  w.Processed,
			Errors:     w.Errors,
			Restarts:   w.Restarts,
			LagMillis:  w.Lag.Milliseconds(),
			LastError:  w.LastError,
		}
	}
	return result
}
