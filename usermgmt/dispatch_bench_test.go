package usermgmt

import (
	"context"
	"fmt"
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
)

// BenchmarkDispatchAuditChain measures the marginal dispatch cost of the
// built-in audit chain (session-actor bridging + context option application +
// causation context) against a bare dispatcher, using a null handler so the
// numbers isolate middleware overhead. The full-stack sub-bench gives the
// realistic absolute dispatch cost through NewService (chain + repository
// enrichers + journal + projection fan-out) with one fresh stream per
// iteration so per-dispatch work stays O(1).
//
// Run with: GOEXPERIMENT=jsonv2 GOWORK=off go test -bench=BenchmarkDispatchAuditChain \
//
//	-benchmem -count=5 -benchtime=2s
//
// and compare variants with benchstat.
func BenchmarkDispatchAuditChain(b *testing.B) {
	b.Run("bare-null-handler", func(b *testing.B) {
		benchNullDispatch(b, false)
	})
	b.Run("audit-chain-null-handler", func(b *testing.B) {
		benchNullDispatch(b, true)
	})
	b.Run("full-stack-register", func(b *testing.B) {
		benchFullStackRegister(b)
	})
}

func benchNullDispatch(b *testing.B, withChain bool) {
	dispatcher := command.NewDispatcher()
	if withChain {
		dispatcher.Use(commandAuditMiddleware()...)
	}
	err := command.RegisterTyped(dispatcher, cmdRegisterUser,
		func(ctx context.Context, c *RegisterUserCmd) error { return nil })
	if err != nil {
		b.Fatalf("RegisterTyped: %v", err)
	}

	aggID, err := aggIDFromUser(GenerateUserID())
	if err != nil {
		b.Fatalf("aggIDFromUser: %v", err)
	}
	ctx := cqrshtmx.WithActorID(b.Context(), ActorIDFromUser(GenerateUserID()))

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if err := dispatcher.Dispatch(ctx, NewRegisterUserCmd(aggID, "bench@example.com", "Bench", nil)); err != nil {
			b.Fatalf("Dispatch: %v", err)
		}
	}
}

func benchFullStackRegister(b *testing.B) {
	svc, err := NewService(ServiceConfig{})
	if err != nil {
		b.Fatalf("NewService: %v", err)
	}
	b.Cleanup(func() {
		_ = svc.Close()
	})

	var i int
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		aggID, aggErr := aggIDFromUser(GenerateUserID())
		if aggErr != nil {
			b.Fatalf("aggIDFromUser: %v", aggErr)
		}
		cmd := NewRegisterUserCmd(aggID, fmt.Sprintf("bench-%d@example.com", i), "Bench", nil)
		if err := svc.dispatcher.Dispatch(b.Context(), cmd); err != nil {
			b.Fatalf("Dispatch: %v", err)
		}
		i++
	}
}
