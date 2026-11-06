package cqrshtmx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// countingDenyEnforcer answers "no" to every authorization question and counts how
// often it was asked. It is the sharpest way to prove a gate is not on a code
// path: a deny-everything engine that is never consulted denies nothing.
type countingDenyEnforcer struct {
	calls int
}

func (e *countingDenyEnforcer) Enforce(_ ...any) (bool, error) {
	e.calls++

	return false, nil
}

// gatedCmd is a command type used to prove where the gate does and does not
// reach. It carries no payload on purpose: the question under test is
// "is this dispatch authorized", not "what does the handler do".
type gatedCmd struct {
	aggID id.StreamID
	cmdID id.CommandID
}

func (c *gatedCmd) Type() command.Type    { return "CreateItem" }
func (c *gatedCmd) StreamID() id.StreamID { return c.aggID }
func (c *gatedCmd) ID() id.CommandID      { return c.cmdID }

// TestDispatchWithoutRouteIsUngated documents, and thereby pins, the hole this
// workstream exists to close: authorization is a property of the HTTP route, so
// a dispatch that never travels through a route is authorized by nobody.
//
// Both subtests are expected to pass today. They are the "before" half of the
// proof; authz_dispatch_gate_test.go is the "after" half, where the same
// dispatcher carries a gate and the same call is denied.
//
// OWNS: the gate that makes these denials land in M04.
func TestDispatchWithoutRouteIsUngated(t *testing.T) {
	t.Run("service_level_dispatch_has_no_gate", func(t *testing.T) {
		disp := command.NewDispatcher()

		var reached bool

		err := command.RegisterTyped(disp, "CreateItem", func(_ context.Context, _ *gatedCmd) error {
			reached = true

			return nil
		})
		if err != nil {
			t.Fatalf("register typed command: %v", err)
		}

		// This is the call shape of Service.GrantRoles, a seed, a scheduled
		// job and an admin CLI: background context, no session, no route.
		if err := disp.Dispatch(context.Background(), &gatedCmd{}); err != nil {
			t.Fatalf("dispatch returned %v, want nil", err)
		}

		if !reached {
			t.Fatal("handler did not run")
		}
	})

	t.Run("route_gate_does_not_cover_the_dispatcher", func(t *testing.T) {
		enforcer := &countingDenyEnforcer{}
		disp := command.NewDispatcher()

		var reached bool

		err := command.RegisterTyped(disp, "CreateItem", func(_ context.Context, _ *gatedCmd) error {
			reached = true

			return nil
		})
		if err != nil {
			t.Fatalf("register typed command: %v", err)
		}

		app, err := New(Config{Commands: disp, Enforcer: enforcer})
		if err != nil {
			t.Fatalf("new app: %v", err)
		}

		// The route IS gated: an unauthenticated POST is refused.
		route := app.Command("CreateItem",
			Authorize("items", "create"),
			DecodeJSON(func(_ struct{}) (command.Command, error) { return &gatedCmd{}, nil }),
		)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/items", strings.NewReader(`{}`))
		route(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("gated route answered %d, want %d", rec.Code, http.StatusUnauthorized)
		}

		// The same command, dispatched without a route, is admitted — the
		// route's gate lives in the handler config the dispatcher never sees.
		if err := disp.Dispatch(context.Background(), &gatedCmd{}); err != nil {
			t.Fatalf("dispatch returned %v, want nil", err)
		}

		if !reached {
			t.Fatal("handler did not run")
		}

		if enforcer.calls != 0 {
			t.Fatalf("enforcer consulted %d times by the unauthenticated route, want 0", enforcer.calls)
		}
	})
}
