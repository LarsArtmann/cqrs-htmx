package authztest

import (
	"context"
	"fmt"
	"strings"
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// --- fixtures: a real dispatcher pair behind a real DispatchGate ---

type suiteItemCmd struct {
	aggID id.StreamID
	cmdID id.CommandID
}

func (suiteItemCmd) Type() command.Type      { return "CreateItem" }
func (c suiteItemCmd) StreamID() id.StreamID { return c.aggID }
func (c suiteItemCmd) ID() id.CommandID      { return c.cmdID }

type suiteListQuery struct{}

func (suiteListQuery) Type() query.Type { return "ListItems" }

// suiteChecker admits exactly one subject and denies every other.
func suiteChecker(privileged string) cqrshtmx.PermissionChecker {
	return subjectChecker{allowed: privileged}
}

type subjectChecker struct{ allowed string }

func (c subjectChecker) Check(subject, _, _, _ string) (bool, []string, error) {
	return subject == c.allowed, nil, nil
}

const (
	suitePrivilegedSubject   = "01HK154ANGZHV2ZW0X3SKSNEN2"
	suiteUnprivilegedSubject = "01HK154ANGZHV2ZW0X3SKSNEN3"
)

func identityAs(userID string) Identity {
	return func(ctx context.Context) context.Context {
		return cqrshtmx.WithUserID(ctx, cqrshtmx.MustParseUserID(userID))
	}
}

func anonymousIdentity() Identity {
	return func(ctx context.Context) context.Context { return ctx }
}

// wiredSurface builds a command+query dispatcher pair with a declaring gate, and
// returns the declarations a suite should prove.
func wiredSurface(t *testing.T) []DispatchCase {
	t.Helper()

	gate, err := cqrshtmx.NewDispatchGate(suiteChecker(suitePrivilegedSubject))
	if err != nil {
		t.Fatalf("new gate: %v", err)
	}

	cmds := command.NewDispatcher()
	qrjs := query.NewDispatcher()

	if err := cqrshtmx.RegisterCommand(gate, cmds, "CreateItem", cqrshtmx.PostureRoleGated,
		"creates an item in the caller's tenant",
		func(context.Context, *suiteItemCmd) error { return nil }); err != nil {
		t.Fatalf("register command: %v", err)
	}

	if err := cqrshtmx.RegisterQuery(gate, qrjs, "ListItems", cqrshtmx.PostureAuthenticated,
		"any signed-in caller may list items",
		func(context.Context, suiteListQuery) (int, error) { return 0, nil }); err != nil {
		t.Fatalf("register query: %v", err)
	}

	cmds.Use(gate.CommandMiddleware())
	qrjs.Use(gate.QueryMiddleware())

	return []DispatchCase{
		{
			Kind: cqrshtmx.DispatchCommandKind, Name: "CreateItem", Posture: RoleGated,
			Why: "creates an item in the caller's tenant",
			Dispatch: func(ctx context.Context) error {
				return cmds.Dispatch(ctx, &suiteItemCmd{})
			},
		},
		{
			Kind: cqrshtmx.DispatchQueryKind, Name: "ListItems", Posture: Authenticated,
			Why: "any signed-in caller may list items",
			Dispatch: func(ctx context.Context) error {
				_, err := query.DispatchTyped[int](ctx, qrjs, suiteListQuery{})

				return err
			},
		},
	}
}

func correctCommandSuite(t *testing.T) CommandSuite {
	t.Helper()

	return CommandSuite{
		Cases:        wiredSurface(t),
		Anonymous:    anonymousIdentity(),
		Unprivileged: identityAs(suiteUnprivilegedSubject),
		Privileged:   identityAs(suitePrivilegedSubject),
	}
}

// TestCommandSuite_AcceptsAFaithfulDeclaration is the positive control: a suite
// whose declarations match the wiring passes.
func TestCommandSuite_AcceptsAFaithfulDeclaration(t *testing.T) {
	suite := correctCommandSuite(t)

	if err := suite.Verify(); err != nil {
		t.Fatalf("Verify: %v", err)
	}

	suite.Run(t)
}

// TestCommandSuite_AMutatedPostureReds is the mutation check the matrix exists
// for: claim a gated command is public and the suite must fail.
func TestCommandSuite_AMutatedPostureReds(t *testing.T) {
	suite := correctCommandSuite(t)
	suite.Cases[0].Posture = Public

	if err := suite.Verify(); err != nil {
		t.Fatalf("Verify rejected the mutation before probing it: %v", err)
	}

	recorder := &recordingTB{}
	suite.probe(recorder, suite.Cases[0])

	if recorder.errors == 0 {
		t.Fatal("a public declaration over a gated command passed the matrix")
	}

	if !strings.Contains(strings.Join(recorder.messages, "\n"), "admitted") {
		t.Fatalf("failure should say the anonymous caller was admitted, got: %v", recorder.messages)
	}
}

// TestCommandSuite_VerifyFailsClosed proves every unusable suite is refused
// rather than silently skipping the case it cannot prove.
func TestCommandSuite_VerifyFailsClosed(t *testing.T) {
	t.Parallel()

	t.Run("no_cases", func(t *testing.T) {
		t.Parallel()

		if err := (CommandSuite{Anonymous: anonymousIdentity()}).Verify(); err == nil {
			t.Fatal("Verify accepted an empty case set")
		}
	})

	t.Run("no_anonymous", func(t *testing.T) {
		t.Parallel()

		suite := correctCommandSuite(t)
		suite.Anonymous = nil

		if err := suite.Verify(); err == nil {
			t.Fatal("Verify accepted a suite with no anonymous identity")
		}
	})

	t.Run("no_unprivileged_for_role_gated", func(t *testing.T) {
		t.Parallel()

		suite := correctCommandSuite(t)
		suite.Unprivileged = nil

		if err := suite.Verify(); err == nil {
			t.Fatal("Verify accepted a RoleGated case with no unprivileged identity")
		}
	})

	t.Run("no_privileged_for_authenticated", func(t *testing.T) {
		t.Parallel()

		suite := correctCommandSuite(t)
		suite.Privileged = nil

		if err := suite.Verify(); err == nil {
			t.Fatal("Verify accepted a gated case with no privileged identity")
		}
	})

	t.Run("missing_reason", func(t *testing.T) {
		t.Parallel()

		suite := correctCommandSuite(t)
		suite.Cases[0].Why = ""

		if err := suite.Verify(); err == nil {
			t.Fatal("Verify accepted a declaration with no reason")
		}
	})

	t.Run("unusable_posture", func(t *testing.T) {
		t.Parallel()

		suite := correctCommandSuite(t)
		suite.Cases[0].Posture = Ungated

		if err := suite.Verify(); err == nil {
			t.Fatal("Verify accepted an Ungated declaration")
		}
	})
}

// TestCommandSuite_UndeclaredDispatchIsRefusedAtDispatch proves the runtime half
// of fail-closed: a handler registered behind the gate's back is denied, and the
// helper reports it when the wiring can name its own types.
func TestCommandSuite_UndeclaredDispatchIsRefusedAtDispatch(t *testing.T) {
	t.Parallel()

	gate, err := cqrshtmx.NewDispatchGate(suiteChecker(suitePrivilegedSubject))
	if err != nil {
		t.Fatalf("new gate: %v", err)
	}

	disp := command.NewDispatcher()
	disp.Use(gate.CommandMiddleware())

	if err := disp.Register("Rogue", func(context.Context, command.Command) error { return nil }); err != nil {
		t.Fatalf("register rogue: %v", err)
	}

	err = disp.Dispatch(identityAs(suitePrivilegedSubject)(context.Background()), suiteRogueCmd{})
	if !cqrshtmx.IsForbidden(err) {
		t.Fatalf("undeclared dispatch = %v, want a forbidden denial", err)
	}

	missing := UndeclaredDispatches(nil, []string{"command/Rogue"})
	if len(missing) != 1 || missing[0] != "command/Rogue" {
		t.Fatalf("UndeclaredDispatches = %v, want [command/Rogue]", missing)
	}
}

type suiteRogueCmd struct {
	aggID id.StreamID
	cmdID id.CommandID
}

func (suiteRogueCmd) Type() command.Type      { return "Rogue" }
func (c suiteRogueCmd) StreamID() id.StreamID { return c.aggID }
func (c suiteRogueCmd) ID() id.CommandID      { return c.cmdID }

// recordingTB captures Errorf calls so a probe can be asserted without failing
// the enclosing test.
type recordingTB struct {
	errors   int
	messages []string
}

func (r *recordingTB) Helper() {}

func (r *recordingTB) Errorf(format string, args ...any) {
	r.errors++
	r.messages = append(r.messages, fmt.Sprintf(format, args...))
}
