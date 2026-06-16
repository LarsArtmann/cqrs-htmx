package usermgmt

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
	"github.com/larsartmann/go-cqrs-lite/projection/v2"
)

// StartProjections creates a projection Runner, registers the read model and
// Casbin projection, and starts the runner in a background goroutine.
//
// The runner replays all events from the journal first (using checkpoints),
// then subscribes to live events from the bus.
//
// With memory.MemoryBus, event publishing is synchronous — projections update
// before Execute() returns, providing read-your-writes consistency.
func StartProjections(
	journal event.Journal,
	bus event.Subscriber,
	readModel *UserReadModel,
	casbinProjection *CasbinProjection,
	auditLog *AuditLog,
) error {
	checkpointStore := memory.NewMemoryCheckpointStore()

	runner, err := projection.NewRunner(journal, bus, checkpointStore)
	if err != nil {
		return fmt.Errorf("create projection runner: %w", err)
	}

	if err := runner.Register(readModel); err != nil {
		return fmt.Errorf("register read model projection: %w", err)
	}

	if err := runner.Register(casbinProjection); err != nil {
		return fmt.Errorf("register casbin projection: %w", err)
	}

	if auditLog != nil {
		if err := runner.Register(auditLog); err != nil {
			return fmt.Errorf("register audit log projection: %w", err)
		}
	}

	go func() {
		if err := runner.Run(context.Background()); err != nil {
			slog.Error("usermgmt: projection runner stopped", "error", err)
		}
	}()

	time.Sleep(50 * time.Millisecond)

	return nil
}
