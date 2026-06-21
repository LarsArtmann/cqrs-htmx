package usermgmt

import (
	"context"
	"log/slog"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
	"github.com/larsartmann/go-cqrs-lite/projection/v2"
)

// subscribeSignal wraps an event.Subscriber so the first Subscribe/SubscribeAll
// call closes a readiness channel. StartProjections uses it to block until the
// projection Runner has registered its live subscription with the bus —
// deterministically guaranteeing that the first event a caller publishes is
// delivered, instead of relying on a time.Sleep to paper over the race.
type subscribeSignal struct {
	event.Subscriber
	once  sync.Once
	ready chan struct{}
}

func newSubscribeSignal(s event.Subscriber) *subscribeSignal {
	return &subscribeSignal{ //nolint:exhaustruct // once must start zero-valued
		Subscriber: s,
		ready:      make(chan struct{}),
	}
}

func (s *subscribeSignal) Subscribe(t event.Type, h event.Handler) error {
	err := s.Subscriber.Subscribe(t, h)
	s.once.Do(func() { close(s.ready) })

	return err //nolint:wrapcheck // transparent delegation
}

func (s *subscribeSignal) SubscribeAll(h event.Handler) error {
	err := s.Subscriber.SubscribeAll(h)
	s.once.Do(func() { close(s.ready) })

	return err //nolint:wrapcheck // transparent delegation
}

// StartProjections creates a projection Runner, registers the read model and
// Casbin projection, replays history, and starts live tailing of the bus.
//
// Replay is synchronous: RunReplay replays every historical event from the
// journal (using checkpoints) and returns only once all registered projections
// have caught up to the current event stream.
//
// The live subscription is established before StartProjections returns: it waits
// for the Runner to register its bus handler, so the very first event a caller
// publishes is delivered. Combined with memory.MemoryBus (synchronous publish),
// this gives read-your-writes consistency with no timing-based sleeps.
func StartProjections(
	journal event.Journal,
	bus event.Subscriber,
	readModel *UserReadModel,
	membershipReadModel *MembershipReadModel,
	casbinProjection *CasbinProjection,
	auditLog *AuditLog,
) error {
	checkpointStore := memory.NewMemoryCheckpointStore()
	signal := newSubscribeSignal(bus)

	runner, err := projection.NewRunner(journal, signal, checkpointStore)
	if err != nil {
		return event.WrapInfrastructure(err, "usermgmt.projection.runner_failed", "create projection runner")
	}

	if err := runner.Register(readModel); err != nil {
		return event.WrapInfrastructure(err, "usermgmt.projection.register_failed", "register read model projection")
	}

	if membershipReadModel != nil {
		if err := runner.Register(membershipReadModel); err != nil {
			return event.WrapInfrastructure(
				err,
				"usermgmt.projection.register_failed",
				"register membership read model projection",
			)
		}
	}

	if err := runner.Register(casbinProjection); err != nil {
		return event.WrapInfrastructure(err, "usermgmt.projection.register_failed", "register casbin projection")
	}

	if auditLog != nil {
		if err := runner.Register(auditLog); err != nil {
			return event.WrapInfrastructure(err, "usermgmt.projection.register_failed", "register audit log projection")
		}
	}

	// RunReplay is synchronous: it returns only once the read model reflects
	// every committed event, catching up history with no sleeps.
	if err := runner.RunReplay(context.Background()); err != nil {
		return event.WrapInfrastructure(err, "usermgmt.projection.replay_failed", "replay projections")
	}

	// RunLive tails live events from the bus until the context is cancelled.
	go func() {
		if err := runner.RunLive(context.Background()); err != nil {
			slog.Error("usermgmt: projection runner stopped", "error", err)
		}
	}()

	// Block until the Runner has registered its live subscription, so callers
	// cannot publish before the projection is listening.
	<-signal.ready

	return nil
}
