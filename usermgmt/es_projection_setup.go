package usermgmt

import (
	"context"
	"log/slog"
	"slices"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

// StartProjections replays historical events from the journal into all
// registered projections, then subscribes to live events from the bus.
//
// Replay is synchronous: every historical event is dispatched to projections
// before StartProjections returns. Combined with the synchronous event bus
// (watermill.EventBus with BlockPublishUntilSubscriberAck), this provides
// read-your-writes consistency: after a command completes, the read model
// already reflects the change — no timing-based sleeps.
//
// Dedup: event IDs processed during replay are tracked and skipped in the
// live handler to prevent double-processing at the replay-to-live boundary.
func StartProjections(
	journal event.Journal,
	bus event.Subscriber,
	readModel event.Projection,
	membershipReadModel event.Projection,
	tenantReadModel event.Projection,
	botReadModel event.Projection,
	casbinProjection *CasbinProjection,
	auditLog *AuditLog,
) error {
	projections := collectProjections(
		readModel, membershipReadModel, tenantReadModel, botReadModel, casbinProjection, auditLog,
	)

	seenIDs, err := replayProjections(journal, projections)
	if err != nil {
		return err
	}

	liveHandler := buildLiveHandler(projections, seenIDs)
	if err := bus.SubscribeAll(liveHandler); err != nil {
		return event.WrapInfrastructure(err,
			"usermgmt.projection.subscribe_failed",
			"subscribe to live events")
	}

	return nil
}

// collectProjections gathers all projection implementations into a slice.
// The read model and casbin projection are always present; the rest are optional.
func collectProjections(
	readModel event.Projection,
	membershipReadModel event.Projection,
	tenantReadModel event.Projection,
	botReadModel event.Projection,
	casbinProjection *CasbinProjection,
	auditLog *AuditLog,
) []event.Projection {
	projections := []event.Projection{readModel, casbinProjection}
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

// replayProjections reads all events from the journal and dispatches each to
// every projection that handles its event type. Returns a set of seen event
// IDs for live-handler dedup.
func replayProjections(
	journal event.Journal,
	projections []event.Projection,
) (map[id.EventID]struct{}, error) {
	replayCtx := event.WithProcessingMode(context.Background(), event.ModeReplay)

	events, err := journal.ReadAll(context.Background())
	if err != nil {
		return nil, event.WrapInfrastructure(err,
			"usermgmt.projection.replay_failed",
			"read events from journal")
	}

	seenIDs := make(map[id.EventID]struct{}, len(events))
	for _, evt := range events {
		seenIDs[evt.ID()] = struct{}{}

		for _, proj := range projections {
			if !slices.Contains(proj.EventTypes(), evt.Type()) {
				continue
			}

			if err := proj.Handle(replayCtx, evt); err != nil {
				return nil, event.WrapInfrastructure(err,
					"usermgmt.projection.replay_failed",
					"replay event in projection "+proj.Name())
			}
		}
	}

	return seenIDs, nil
}

// buildLiveHandler creates an event.Handler that routes live events to all
// projections. Events already seen during replay are skipped (dedup).
//
// Error handling: projection errors during live processing are logged at
// error level but do not stop event delivery. This is intentional — a single
// failing projection should not block other projections or cause the bus to
// retry the event indefinitely. The previous projection.Runner had retry
// and dead-letter-queue support; that complexity was intentionally dropped
// in favor of simplicity (ADR-0016). Consumers needing retry semantics can
// wrap their projection's Handle method.
//
// Memory: the seenIDs map is seeded once during replay and never grows
// during live processing. Its size is bounded by the number of events in
// the journal at startup time.
func buildLiveHandler(
	projections []event.Projection,
	seenIDs map[id.EventID]struct{},
) event.Handler {
	return event.Handler(func(ctx context.Context, evt event.Event) error {
		if _, seen := seenIDs[evt.ID()]; seen {
			return nil
		}

		for _, proj := range projections {
			if !slices.Contains(proj.EventTypes(), evt.Type()) {
				continue
			}

			if err := proj.Handle(ctx, evt); err != nil {
				slog.Error("usermgmt: projection handler failed",
					"projection", proj.Name(),
					"event_type", evt.Type().String(),
					"error", err)
			}
		}

		return nil
	})
}
