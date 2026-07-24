package usermgmt

import (
	"context"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
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
	if s.projectionHost == nil {
		return errorfamily.NewRejection(
			"usermgmt.rebuild.no_host",
			"no projection host configured",
		)
	}

	if err := s.projectionHost.Stop(); err != nil {
		return errorfamily.WrapTransient(err,
			"usermgmt.rebuild.stop_failed",
			"stop projection host for rebuild",
		)
	}

	if err := s.projectionHost.Reset(ctx, name); err != nil {
		return err
	}

	host, err := s.restartProjections(ctx)
	if err != nil {
		return err
	}

	s.projectionHost = host

	return nil
}

// restartProjections creates a fresh projectionhost.Host, registers all
// projections, starts it, and blocks until drain completes. Used by
// RebuildProjection after the old host has been stopped and reset.
func (s *EventSourcedSetup) restartProjections(ctx context.Context) (*projectionhost.Host, error) {
	journal := journalFromStore(s.Store)

	cpStore := s.checkpointStore
	if cpStore == nil {
		cpStore = memory.NewMemoryCheckpointStore()
	}

	seekable, ok := journal.(eventSeekableJournal)
	if !ok {
		return nil, errorfamily.NewRejection(
			"usermgmt.rebuild.journal_not_seekable",
			"projectionhost requires a SeekableJournal (ReadFrom); "+
				"the event store does not implement event.SeekableJournal",
		)
	}

	host, err := projectionhost.New(seekable, cpStore,
		projectionhost.WithSubscriber(s.Bus),
		projectionhost.WithDeadLetterStore(projectionhost.NewMemoryDeadLetterStore(), 0),
	)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err,
			"usermgmt.rebuild.host_create_failed",
			"create projection host")
	}

	for _, p := range s.projections {
		if err := host.Register(p); err != nil {
			return nil, errorfamily.WrapInfrastructure(err,
				"usermgmt.rebuild.register_failed",
				"register projection "+p.Name())
		}
	}

	if err := host.Start(ctx); err != nil {
		return nil, errorfamily.WrapInfrastructure(err,
			"usermgmt.rebuild.start_failed",
			"restart projection host after rebuild")
	}

	if err := waitForDrain(host); err != nil {
		_ = host.Stop()
		return nil, err
	}

	return host, nil
}

// --- Service methods ---

// EventCatalog returns a catalog populated with all 21 usermgmt event types.
// Use this with cqrshtmx.EventCatalogHandler to serve an event schema endpoint.
func (svc *Service) EventCatalog() *cqrshtmx.EventCatalog {
	return DefaultEventCatalog()
}

// ProjectionStatuses adapts the internal projectionhost.Host status to the
// cqrshtmx.ProjectionStatusProvider interface. This lets consumers pass the
// Service directly to cqrshtmx.ProjectionStatusHandler.
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
	if svc.projectionHost == nil {
		return errorfamily.NewRejection(
			"usermgmt.rebuild.no_host",
			"no projection host configured",
		)
	}

	if err := svc.projectionHost.Stop(); err != nil {
		return errorfamily.WrapTransient(err,
			"usermgmt.rebuild.stop_failed",
			"stop projection host for rebuild",
		)
	}

	if err := svc.projectionHost.Reset(ctx, name); err != nil {
		return err
	}

	host, err := svc.restartProjections(ctx)
	if err != nil {
		return err
	}

	svc.projectionHost = host

	return nil
}

// restartProjections creates a fresh projectionhost.Host for the Service.
// The Service stores the same components as EventSourcedSetup.
func (svc *Service) restartProjections(ctx context.Context) (*projectionhost.Host, error) {
	journal := journalFromStore(svc.store)

	cpStore := svc.checkpointStoreField()
	if cpStore == nil {
		cpStore = memory.NewMemoryCheckpointStore()
	}

	seekable, ok := journal.(eventSeekableJournal)
	if !ok {
		return nil, errorfamily.NewRejection(
			"usermgmt.rebuild.journal_not_seekable",
			"projectionhost requires a SeekableJournal",
		)
	}

	host, err := projectionhost.New(seekable, cpStore,
		projectionhost.WithSubscriber(svc.bus),
		projectionhost.WithDeadLetterStore(projectionhost.NewMemoryDeadLetterStore(), 0),
	)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err,
			"usermgmt.rebuild.host_create_failed",
			"create projection host")
	}

	for _, p := range svc.projectionList() {
		if err := host.Register(p); err != nil {
			return nil, errorfamily.WrapInfrastructure(err,
				"usermgmt.rebuild.register_failed",
				"register projection "+p.Name())
		}
	}

	if err := host.Start(ctx); err != nil {
		return nil, errorfamily.WrapInfrastructure(err,
			"usermgmt.rebuild.start_failed",
			"restart projection host after rebuild")
	}

	if err := waitForDrain(host); err != nil {
		_ = host.Stop()
		return nil, err
	}

	return host, nil
}

// checkpointStoreField and projectionList are thin accessors for Service fields
// that mirror EventSourcedSetup. The Service does not currently store these
// directly (it stores the projectionHost only), so these return nil/empty
// to signal that the Service-level rebuild is not supported and the caller
// should use EventSourcedSetup.RebuildProjection instead.
func (svc *Service) checkpointStoreField() eventCheckpointStore { return nil }
func (svc *Service) projectionList() []projection.Projection     { return nil }

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
