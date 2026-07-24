package usermgmt

import (
	"context"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
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
// checkpoint and state, then restarts the host and blocks until all
// projections have replayed the full event journal. This is the canonical
// way to rebuild a projection's read model from scratch.
//
// Use the projection's Name() as the name argument (e.g. "user-read-model",
// "casbin-projection", "membership-read-model", "tenant-read-model",
// "bot-read-model", "audit-log").
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

	if err := s.projectionHost.Start(ctx); err != nil {
		return errorfamily.WrapInfrastructure(err,
			"usermgmt.rebuild.start_failed",
			"restart projection host after rebuild",
		)
	}

	if err := waitForDrain(s.projectionHost); err != nil {
		return err
	}

	return nil
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
// checkpoint and state, then restarts the host and blocks until all
// projections have replayed the full event journal.
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

	if err := svc.projectionHost.Start(ctx); err != nil {
		return errorfamily.WrapInfrastructure(err,
			"usermgmt.rebuild.start_failed",
			"restart projection host after rebuild",
		)
	}

	if err := waitForDrain(svc.projectionHost); err != nil {
		return err
	}

	return nil
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
