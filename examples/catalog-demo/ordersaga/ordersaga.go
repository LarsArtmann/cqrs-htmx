// Package ordersaga is the executable order-saga fixture behind the
// catalog-demo's GFL showcase: order.create emits order.created; a bus
// deriver derives payment.settle (ADR-0154), which emits payment.settled;
// order.cancel closes the order. Fulfillment is deliberately docs-only —
// the feature graph and the EventCatalog flows describe it, no executable
// scenarios — so the demo shows both faces of GFL in one artifact.
//
// The domain is the same shape a production binary would hand to
// system.New; the generated specs (order-saga_test.go) and the served
// catalog all consume it unchanged.
package ordersaga

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/deriver/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
)

// Wire names of the order-saga domain.
const (
	StreamType     id.StreamType = "Order"
	StreamName                   = "order-demo"
	CreateType     command.Type  = "order.create"
	CancelType     command.Type  = "order.cancel"
	SettleType     command.Type  = "payment.settle"
	Created        event.Type    = "order.created"
	Cancelled      event.Type    = "order.cancelled"
	Settled        event.Type    = "payment.settled"
	ProjectionName               = "ordersaga-stage-counts"
	CreatedKey                   = "created"
	CancelledKey                 = "cancelled"
	SettledKey                   = "settled"
)

// Causality rejections: dispatching a command out of chain order fails the
// derived dispatch loudly instead of silently skipping a stage.
var (
	ErrAlreadyCreated = errors.New("ordersaga: order already created")
	ErrOrderNotFound  = errors.New("ordersaga: order does not exist")
)

// OrderState folds the saga events; each stage implies the previous one.
type OrderState struct {
	Exists    bool
	Cancelled bool
	Settled   bool
}

// OrderEvent is the shared payload of every saga event: the stage name.
type OrderEvent struct {
	Stage string
}

// Domain returns the order-saga DomainConfig: one Order decider, three
// commands, and ONE bus deriver advancing the chain (payment.settle on
// order.created — the plan's single-deriver showcase).
func Domain() system.DomainConfig {
	return system.DomainConfig{
		Commands: func(sys *system.System) {
			registerDecider(sys)
			registerCommands(sys)
			subscribeSettleDeriver(sys)
		},
		Projections: []system.ProjectionDeclaration{
			system.Count(ProjectionName).
				On(string(Created), OrderEvent{}, 1, CreatedKey).
				On(string(Cancelled), OrderEvent{}, 1, CancelledKey).
				On(string(Settled), OrderEvent{}, 1, SettledKey).
				Done(),
		},
		Events: []event.Type{Created, Cancelled, Settled},
	}
}

// EventNames lists the declared event universe, in declaration order — the
// events-sync test pins the committed universe JSON to this list.
func EventNames() []string {
	return []string{string(Created), string(Cancelled), string(Settled)}
}

func registerDecider(sys *system.System) {
	if err := system.RegisterDecider(sys, string(StreamType), decider.Decider[OrderState]{
		Initial: OrderState{},
		Apply: func(state OrderState, evt event.Event) (OrderState, error) {
			switch evt.Type() {
			case Created:
				state.Exists = true
			case Cancelled:
				state.Cancelled = true
			case Settled:
				state.Settled = true
			}

			return state, nil
		},
	}); err != nil {
		panic("ordersaga: register decider: " + err.Error())
	}
}

// registerCommands wires the saga commands, each guarded by the causal
// precondition of its stage. payment.settle no-ops once the order is
// settled or cancelled: the deriver may deliver order.created twice
// (at-least-once), and a settled/cancelled order has nothing to settle.
func registerCommands(sys *system.System) {
	register := func(
		cmdType command.Type,
		guard func(state OrderState, ver event.Version) ([]event.Event, error),
	) {
		if err := system.RegisterCommand[*command.BasicCommand, OrderState](
			sys, cmdType,
			func(ctx context.Context, cmd *command.BasicCommand) system.Op[OrderState] {
				return system.Execute(ctx, cmd.StreamID(), StreamType, guard)
			}); err != nil {
			panic("ordersaga: register command " + string(cmdType) + ": " + err.Error())
		}
	}

	register(CreateType, mintStage(func(state OrderState) error {
		if state.Exists {
			return ErrAlreadyCreated
		}

		return nil
	}, Created))

	register(CancelType, mintStage(func(state OrderState) error {
		if !state.Exists {
			return ErrOrderNotFound
		}

		return nil
	}, Cancelled))

	register(SettleType, func(state OrderState, ver event.Version) ([]event.Event, error) {
		switch {
		case !state.Exists:
			return nil, ErrOrderNotFound
		case state.Settled, state.Cancelled:
			return nil, nil
		default:
			return mintEvent(Settled, ver)
		}
	})
}

// mintStage returns a command guard that checks precondition, then mints
// the stage's event at the next version.
func mintStage(
	precondition func(state OrderState) error,
	evtType event.Type,
) func(OrderState, event.Version) ([]event.Event, error) {
	return func(state OrderState, ver event.Version) ([]event.Event, error) {
		if err := precondition(state); err != nil {
			return nil, err
		}

		return mintEvent(evtType, ver)
	}
}

func mintEvent(evtType event.Type, ver event.Version) ([]event.Event, error) {
	streamID := Stream()

	evt, err := event.New(evtType, streamID, StreamType, ver+1, OrderEvent{Stage: string(evtType)})
	if err != nil {
		return nil, fmt.Errorf("ordersaga: mint %s event: %w", evtType, err)
	}

	return []event.Event{evt}, nil
}

// subscribeSettleDeriver advances the chain over the bus. The sanctioned
// saga wiring (ADR-0154): a synchronous deriver on sys.Bus() deadlocks, so
// the derived dispatch leaves the handler goroutine via WithAsyncDispatch;
// a failed dispatch logs and continues (best-effort parity with the
// tenant-deletion cascade: a transient failure must not kill the demo).
func subscribeSettleDeriver(sys *system.System) {
	dispatcher := sys.CommandDispatcher()

	settle := deriver.Deriver(func(_ context.Context, evt event.Event) ([]command.Command, error) {
		cmd, err := command.New(SettleType, evt.StreamID())
		if err != nil {
			return nil, fmt.Errorf("ordersaga: derive %s: %w", SettleType, err)
		}

		return []command.Command{cmd}, nil
	})

	handler := settle.AsHandler(dispatcher,
		deriver.WithAsyncDispatch(func(evt event.Event, cmd command.Command, err error) {
			slog.Default().Warn("ordersaga: derived payment.settle dispatch failed",
				"order_id", evt.StreamID().String(),
				"command", cmd.Type(),
				"error", err)
		}))

	if err := sys.Bus().Subscribe(Created, handler); err != nil {
		panic("ordersaga: subscribe settle deriver: " + err.Error())
	}
}

// Stream is the fixed stream every generated scenario uses.
func Stream() id.StreamID {
	streamID, err := id.ParseStreamID(StreamName)
	if err != nil {
		panic("ordersaga: parse stream id: " + err.Error())
	}

	return streamID
}

// Place builds the order.create command for the demo stream.
func Place() *command.BasicCommand {
	cmd, err := command.New(CreateType, Stream())
	if err != nil {
		panic("ordersaga: build place command: " + err.Error())
	}

	return cmd
}

// Cancel builds the order.cancel command for the demo stream.
func Cancel() *command.BasicCommand {
	cmd, err := command.New(CancelType, Stream())
	if err != nil {
		panic("ordersaga: build cancel command: " + err.Error())
	}

	return cmd
}

// Seeded mints a pre-existing order.created event on the scenario's
// stream, for the Given phase: "Given a placed order". Seeding bypasses
// the command path, so the settle deriver fires asynchronously without
// polluting the act's assertions.
func Seeded(sc *systemscenario.Scenario) []event.Event {
	ref := id.NewStreamRef(StreamType, Stream())
	scenarioEvent := sc.Event(Created, ref, OrderEvent{Stage: string(Created)})

	return []event.Event{scenarioEvent}
}

// stageCounts reads the Count projection's view: how many events each
// stage key has folded.
func stageCounts(ctx context.Context, sc *systemscenario.Scenario) (map[string]int64, error) {
	return metaengine.ExecuteTyped[system.CountInput, map[string]int64](
		ctx, sc.System().MetaEngine(), system.CountInput{})
}

// IsCancelled reports whether the read model has folded order.cancelled.
func IsCancelled(ctx context.Context, sc *systemscenario.Scenario) (bool, error) {
	counts, err := stageCounts(ctx, sc)
	if err != nil {
		return false, err
	}

	return counts[CancelledKey] > 0, nil
}

// StageCount returns the folded event count for one stage key
// (created / cancelled / settled).
func StageCount(ctx context.Context, sc *systemscenario.Scenario, stage string) (int64, error) {
	counts, err := stageCounts(ctx, sc)
	if err != nil {
		return 0, err
	}

	return counts[stage], nil
}
