package authztest

import (
	"context"
	"fmt"
	"strings"
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// Dispatch performs one dispatch of the declared type with ctx. It is the unit
// under test: the suite supplies the identity through ctx and asserts on the
// error, so a case never needs to know how the dispatcher was wired.
type Dispatch func(ctx context.Context) error

// Identity attaches a caller to a dispatch context — WithUserID, a service
// identity, or nothing at all for the anonymous case.
type Identity func(ctx context.Context) context.Context

// AsUser returns an Identity that attaches userID to the dispatch context.
func AsUser(userID cqrshtmx.UserID) Identity {
	return func(ctx context.Context) context.Context {
		return cqrshtmx.WithUserID(ctx, userID)
	}
}

// NoIdentity is the anonymous Identity: the context is passed through untouched.
func NoIdentity() Identity {
	return func(ctx context.Context) context.Context { return ctx }
}

// DispatchCase is one declared dispatch: the gate's declaration plus the way to
// run it with an arbitrary identity.
//
// The declaration is embedded rather than restated, so a suite case and a
// production posture cannot drift into two vocabularies — and a posture copied
// from `gate.Manifest()` is a valid case body.
type DispatchCase struct {
	cqrshtmx.DispatchPosture

	// Dispatch runs one dispatch of the type with ctx.
	Dispatch Dispatch
}

// Case builds a dispatch case from a declaration and the way to run it.
func Case(declaration cqrshtmx.DispatchPosture, dispatch Dispatch) DispatchCase {
	return DispatchCase{DispatchPosture: declaration, Dispatch: dispatch}
}

// CommandSuite proves that a command/query dispatcher enforces the postures it
// declares — the dispatch-boundary counterpart of [Suite].
//
// [Suite] proves routes; this proves dispatches, and they are different
// questions. A route gate lives in a handler that the dispatcher never sees, so a
// command can be refused on its route and admitted when a job, a seed, or a
// service method dispatches it directly. This suite closes that gap: every case
// is dispatched with three identities and the error class must match the
// declaration every time.
type CommandSuite struct {
	// Cases are the declarations under test.
	Cases []DispatchCase

	// Anonymous supplies no identity. Required for every case: the anonymous
	// probe is how a non-public posture is proven.
	Anonymous Identity

	// Unprivileged is an authenticated identity WITHOUT the permission a
	// RoleGated case requires. Required when any case is RoleGated.
	Unprivileged Identity

	// Privileged is an authenticated identity WITH the permission. Required when
	// any case is Authenticated or RoleGated.
	Privileged Identity
}

// Verify checks the declarations before anything is dispatched. Like
// [Suite.Verify] it refuses a set it cannot test: the fail-closed rule that makes
// a skipped case impossible.
func (s CommandSuite) Verify() error {
	if len(s.Cases) == 0 {
		return errorfamily.NewRejection(
			"authztest.command_cases_empty",
			"CommandSuite.Cases is empty — declare the dispatches you are proving",
		)
	}

	if s.Anonymous == nil {
		return errorfamily.NewRejection(
			"authztest.command_anonymous_nil",
			"CommandSuite.Anonymous is required: every posture is proven against an anonymous caller",
		)
	}

	wantsUnprivileged, wantsPrivileged := false, false

	for _, c := range s.Cases {
		needsUnprivileged, needsPrivileged, err := c.validate()
		if err != nil {
			return err
		}

		wantsUnprivileged = wantsUnprivileged || needsUnprivileged
		wantsPrivileged = wantsPrivileged || needsPrivileged
	}

	if wantsUnprivileged && s.Unprivileged == nil {
		return errorfamily.NewRejection(
			"authztest.command_unprivileged_nil",
			"CommandSuite has a RoleGated case but no Unprivileged identity",
		)
	}

	if wantsPrivileged && s.Privileged == nil {
		return errorfamily.NewRejection(
			"authztest.command_privileged_nil",
			"CommandSuite has a gated case but no Privileged identity",
		)
	}

	return nil
}

// validate checks one declaration and reports which identities its posture needs
// to be proven.
func (c DispatchCase) validate() (bool, bool, error) {
	if c.Dispatch == nil {
		return false, false, errorfamily.NewRejection(
			"authztest.command_dispatch_nil",
			fmt.Sprintf("%s has no Dispatch function", c),
		)
	}

	if strings.TrimSpace(c.Why) == "" {
		return false, false, errorfamily.NewRejection(
			"authztest.command_why_empty",
			fmt.Sprintf("%s declares no reason; a posture without a reason is a guess", c),
		)
	}

	switch c.Posture {
	case Public:
		return false, false, nil
	case Authenticated:
		return false, true, nil
	case RoleGated:
		return true, true, nil
	case Ungated:
		return false, false, errorfamily.NewRejection(
			"authztest.command_posture_unknown",
			fmt.Sprintf("%s declares %q, which is a manifest truth, not a probe contract", c, c.Posture),
		)
	default:
		return false, false, errorfamily.NewRejection(
			"authztest.command_posture_unknown",
			fmt.Sprintf("%s declares posture %q, which the dispatch matrix cannot probe", c, c.Posture),
		)
	}
}

// Run verifies the declarations and probes every case as a subtest. A failing
// case names the declaration it contradicts, so a posture that drifted from its
// wiring reads as a one-line diagnosis.
func (s CommandSuite) Run(t *testing.T) {
	t.Helper()

	if err := s.Verify(); err != nil {
		t.Fatalf("CommandSuite is not testable: %v", err)
	}

	for _, c := range s.Cases {
		t.Run(c.String(), func(t *testing.T) {
			s.probe(t, c)
		})
	}
}

// probe runs the matrix for one declaration.
func (s CommandSuite) probe(report testingTB, c DispatchCase) {
	report.Helper()

	anonErr := c.Dispatch(s.Anonymous(context.Background()))

	switch c.Posture {
	case Public:
		requireAllowed(report, c, "anonymous", anonErr)

		if s.Privileged != nil {
			requireAllowed(report, c, "privileged", c.Dispatch(s.Privileged(context.Background())))
		}

	case Authenticated:
		requireUnauthorized(report, c, "anonymous", anonErr)
		requireAllowed(report, c, "privileged", c.Dispatch(s.Privileged(context.Background())))

	case RoleGated:
		requireUnauthorized(report, c, "anonymous", anonErr)
		requireForbidden(report, c, "unprivileged", c.Dispatch(s.Unprivileged(context.Background())))
		requireAllowed(report, c, "privileged", c.Dispatch(s.Privileged(context.Background())))

	case Ungated:
		// Unreachable through Run (Verify refuses it); listed so the switch
		// stays exhaustive if a case is probed directly.
		report.Errorf("%s declares %q, which the matrix cannot probe", c, c.Posture)

	default:
		report.Errorf("%s declares posture %q, which the matrix cannot probe", c, c.Posture)
	}
}

func requireAllowed(report testingTB, c DispatchCase, actor string, err error) {
	report.Helper()

	if err != nil {
		report.Errorf("%s declares %q but the %s caller was refused: %v", c, c.Posture, actor, err)
	}
}

func requireUnauthorized(report testingTB, c DispatchCase, actor string, err error) {
	report.Helper()

	if err == nil {
		report.Errorf("%s declares %q but the %s caller was admitted", c, c.Posture, actor)

		return
	}

	if !cqrshtmx.IsUnauthorized(err) {
		report.Errorf("%s: the %s caller must be refused as unauthenticated, got: %v", c, actor, err)
	}
}

func requireForbidden(report testingTB, c DispatchCase, actor string, err error) {
	report.Helper()

	if err == nil {
		report.Errorf("%s declares %q but the %s caller was admitted", c, c.Posture, actor)

		return
	}

	if !cqrshtmx.IsForbidden(err) {
		report.Errorf("%s: the %s caller must be refused as forbidden, got: %v", c, actor, err)
	}
}

// UndeclaredDispatches returns the registered types that carry no declaration,
// given the declaration set and the types the wiring actually serves.
//
// The dispatch boundary cannot enumerate itself (a dispatcher exposes no
// registered-type listing), so there is no equivalent of [Undeclared] reading the
// dispatcher directly. The gate refuses an undeclared type at dispatch time
// instead; this helper covers the case where the wiring can name its own types.
func UndeclaredDispatches(declared []DispatchCase, registered []string) []string {
	known := make(map[string]struct{}, len(declared))
	for _, c := range declared {
		known[c.String()] = struct{}{}
	}

	var missing []string

	for _, name := range registered {
		if _, ok := known[name]; ok {
			continue
		}

		missing = append(missing, name)
	}

	return missing
}
