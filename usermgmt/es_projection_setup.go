package usermgmt

import (
	"context"
	"log/slog"
	"slices"

	"github.com/larsartmann/go-cqrs-lite/dedup/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/projection/v3"
	errorfamily "github.com/larsartmann/go-error-family"
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
// Checkpoint: when cpStore is non-nil AND the journal implements
// event.SeekableJournal, replay resumes from the last checkpoint instead of
// reading the full journal. This avoids re-processing the entire event
// history on every restart. When cpStore is nil, full journal replay is used.
//
// Dedup: event IDs processed during replay are tracked in a bounded [dedup.Ring]
// (1024 entries, ~90KB) and skipped in the live handler to prevent
// doubleprocessing at the replay-to-live boundary. Overlapping events are
// always at the tail of the replay sequence, so a ring covering the live
// channel buffer size (4-10x margin) is sufficient regardless of journal size.
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
) error {
	projections := collectProjections(
		readModel, membershipReadModel, tenantReadModel, botReadModel, casbinProjection, auditLog,
	)

	replayIDs, err := replayProjections(journal, cpStore, projections)
	if err != nil {
		return err
	}

	liveHandler := buildLiveHandler(projections, replayIDs)
	if err := bus.SubscribeAll(liveHandler); err != nil {
		return errorfamily.WrapInfrastructure(err,
			"usermgmt.projection.subscribe_failed",
			"subscribe to live events")
	}

	return nil
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

// replayProjections reads events from the journal and dispatches each to
// every projection that handles its event type. Returns a [dedup.Ring] of
// replayed event IDs for live-handler dedup.
//
// When cpStore is non-nil AND the journal is seekable, replay resumes from
// the stored checkpoint (ReadFrom). Otherwise, full journal replay (ReadAll).
func replayProjections(
	journal event.Journal,
	cpStore event.CheckpointStore,
	projections []projection.Projection,
) (*dedup.Ring, error) {
	replayCtx := event.WithProcessingMode(context.Background(), event.ModeReplay)

	events, cpName, err := loadReplayEvents(context.Background(), journal, cpStore)
	if err != nil {
		return nil, err
	}

	replayIDs := dedup.NewRing(dedup.DefaultCapacity)
	for _, evt := range events {
		replayIDs.Add(evt.ID().String())

		for _, proj := range projections {
			if !slices.Contains(proj.EventTypes(), evt.Type()) {
				continue
			}

			if err := proj.Handle(replayCtx, evt); err != nil {
				return nil, errorfamily.WrapInfrastructure(err,
					"usermgmt.projection.replay_failed",
					"replay event in projection "+proj.Name())
			}
		}

		// Save checkpoint after each event so restarts resume from here.
		if cpStore != nil && cpName != "" {
			if saveErr := cpStore.Save(context.Background(), cpName, event.Checkpoint{
				EventID:     evt.ID(),
				ProcessedAt: evt.OccurredAt(),
			}); saveErr != nil {
				slog.Warn("usermgmt: save checkpoint during replay",
					"event_id", evt.ID().String(), "error", saveErr)
			}
		}
	}

	return replayIDs, nil
}

// loadReplayEvents loads events from the journal for replay. When a
// checkpoint store is provided AND the journal is seekable, it resumes
// from the stored checkpoint. Otherwise it reads the full journal.
// Returns the events and the checkpoint name (empty if not using checkpoints).
func loadReplayEvents(
	ctx context.Context,
	journal event.Journal,
	cpStore event.CheckpointStore,
) ([]event.Event, string, error) {
	if cpStore == nil {
		events, err := journal.ReadAll(ctx)
		if err != nil {
			return nil, "", errorfamily.WrapInfrastructure(err,
				"usermgmt.projection.replay_failed",
				"read events from journal")
		}
		return events, "", nil
	}

	seekable, ok := journal.(event.SeekableJournal)
	if !ok {
		events, err := journal.ReadAll(ctx)
		if err != nil {
			return nil, "", errorfamily.WrapInfrastructure(err,
				"usermgmt.projection.replay_failed",
				"read events from journal")
		}
		return events, "", nil
	}

	const cpName = "usermgmt:start_projections"
	cp, err := cpStore.Load(ctx, cpName)
	if err != nil {
		return nil, "", errorfamily.WrapInfrastructure(err,
			"usermgmt.projection.checkpoint_load_failed",
			"load checkpoint for replay")
	}

	events, err := seekable.ReadFrom(ctx, cp.EventID, 0)
	if err != nil {
		return nil, "", errorfamily.WrapInfrastructure(err,
			"usermgmt.projection.replay_failed",
			"read events from journal via ReadFrom")
	}

	return events, cpName, nil
}

// buildLiveHandler creates an event.Handler that routes live events to all
// projections. Events already seen during replay are skipped (dedup via
// [dedup.Ring]).
//
// Error handling: projection errors during live processing are logged at
// error level but do not stop event delivery. This is intentional — a single
// failing projection should not block other projections or cause the bus to
// retry the event indefinitely. The previous projection.Runner had retry
// and dead-letter-queue support; that complexity was intentionally dropped
// in favor of simplicity (ADR-0016). Consumers needing retry semantics can
// wrap their projection's Handle method.
//
// Memory: the replayIDs ring is a fixed-capacity dedup set (1024 entries).
// Only the most recent replay event IDs are retained, which is sufficient
// because overlapping events are always at the tail of the replay sequence.
// A nil ring (no replay occurred) always returns false from Has — safe no-op.
func buildLiveHandler(
	projections []projection.Projection,
	replayIDs *dedup.Ring,
) event.Handler {
	return event.Handler(func(ctx context.Context, evt event.Event) error {
		if replayIDs.Has(evt.ID().String()) {
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
