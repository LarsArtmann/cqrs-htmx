package cqrshtmx

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
)

// benchChecker is the cheapest possible engine: it answers from a field. The
// benchmark measures the gate's own cost, not a policy engine's.
type benchChecker struct{ allow bool }

func (c benchChecker) Check(string, string, string, string) (bool, []string, error) {
	return c.allow, nil, nil
}

// BenchmarkDispatchGate measures the cost the gate adds to the hot dispatch path,
// split by the work each posture does: nothing, one map lookup, one map lookup
// plus a decision, and the denial path.
func BenchmarkDispatchGate(b *testing.B) {
	ctx := WithUserID(context.Background(), MustParseUserID("01HK154ANGZHV2ZW0X3SKSNEN2"))

	dispatchers := map[string]*command.Dispatcher{
		"no_gate":       newBenchDispatch(b, nil),
		"public":        newBenchDispatch(b, gatedBenchMiddleware(b, PosturePublic, true)),
		"authenticated": newBenchDispatch(b, gatedBenchMiddleware(b, PostureAuthenticated, true)),
		"gated_allow":   newBenchDispatch(b, gatedBenchMiddleware(b, PostureRoleGated, true)),
		"gated_deny":    newBenchDispatch(b, gatedBenchMiddleware(b, PostureRoleGated, false)),
	}

	for _, name := range []string{"no_gate", "public", "authenticated", "gated_allow", "gated_deny"} {
		b.Run(name, func(b *testing.B) {
			disp := dispatchers[name]
			cmd := &gatedCmd{}

			b.ReportAllocs()

			for b.Loop() {
				if err := disp.Dispatch(ctx, cmd); err != nil && name != "gated_deny" {
					b.Fatalf("dispatch: %v", err)
				}
			}
		})
	}
}

// newBenchDispatch registers the bench command, optionally behind a gate.
func newBenchDispatch(b *testing.B, middleware command.Middleware) *command.Dispatcher {
	b.Helper()

	disp := command.NewDispatcher()

	if err := command.RegisterTyped(disp, "CreateItem", func(context.Context, *gatedCmd) error {
		return nil
	}); err != nil {
		b.Fatalf("register: %v", err)
	}

	if middleware != nil {
		disp.Use(middleware)
	}

	return disp
}

// gatedBenchMiddleware builds a gate whose single declaration is posture.
func gatedBenchMiddleware(b *testing.B, posture Posture, allow bool) command.Middleware {
	b.Helper()

	gate, err := NewDispatchGate(benchChecker{allow: allow})
	if err != nil {
		b.Fatalf("new gate: %v", err)
	}

	if err := gate.Declare(DispatchPosture{
		Kind:    DispatchCommandKind,
		Name:    "CreateItem",
		Posture: posture,
		Why:     "benchmark fixture",
	}); err != nil {
		b.Fatalf("declare: %v", err)
	}

	return gate.CommandMiddleware()
}
