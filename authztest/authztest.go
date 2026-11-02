// Package authztest proves that an HTTP surface enforces the access posture it
// claims.
//
// It exists because the 2026-11-02 consumer audit found zero authorization tests
// across 17 services: every surface had a gate somebody wrote once, and nothing
// that would fail when the gate stopped working. A manifest plus this matrix
// turns "we check permissions" into "we check permissions, and here is the test
// that says so".
//
// The package imports net/http/httptest and testing, so it is a test-only
// dependency: consumers import it from _test.go files and it never reaches a
// production binary.
//
// # Fail-closed by construction
//
// [Suite.Verify] refuses a declaration set that cannot be tested: a RoleGated
// route with no unprivileged actor, or any gated route with no anonymous actor,
// is an error rather than a silently skipped case. A skipped case reports green,
// which is exactly the failure mode this package exists to prevent.
package authztest

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

// Posture is what a route requires of its caller.
type Posture string

const (
	// Public routes must answer an anonymous caller. A 401/403 here means the
	// route is gated but declared open — usually a declaration that went stale,
	// occasionally a gate that arrived without a declaration.
	Public Posture = "public"

	// Authenticated routes must refuse an anonymous caller and admit an
	// authenticated one.
	Authenticated Posture = "authenticated"

	// RoleGated routes must refuse an anonymous caller, refuse an authenticated
	// caller who lacks the role, and admit one who holds it.
	RoleGated Posture = "role"
)

// Valid reports whether p is a declared posture.
func (p Posture) Valid() bool {
	switch p {
	case Public, Authenticated, RoleGated:
		return true
	}

	return false
}

// Route is one declared route.
type Route struct {
	// Method is the HTTP method the route answers.
	Method string

	// Path is the request path to probe. Use a concrete path, not a pattern:
	// this is what gets requested.
	Path string

	// Posture is what the route requires.
	Posture Posture

	// Why is the reason this posture is correct. Required: a posture without a
	// reason is a guess, and a guess reviewed later is indistinguishable from a
	// decision.
	Why string
}

// String renders the route as "METHOD /path", the form used in failure messages.
func (r Route) String() string { return r.Method + " " + r.Path }

// Actor supplies credentials to a probe request. Returning nil means "send the
// request unchanged", which is also what a nil Actor does.
type Actor func(*http.Request) *http.Request

// Suite is a declared posture set plus the actors needed to probe it.
type Suite struct {
	// Handler is the fully wired HTTP surface under test — middleware included,
	// because the gates live there.
	Handler http.Handler

	// Routes are the declarations.
	Routes []Route

	// Anonymous supplies no credentials (the actor is usually just the identity
	// function). Required for every non-Public route and for every Public route
	// too: the anonymous probe is how "public" is proven.
	Anonymous Actor

	// Unprivileged is an authenticated actor WITHOUT the role a RoleGated route
	// requires. Required when any route is RoleGated.
	Unprivileged Actor

	// Privileged is an authenticated actor WITH the role. Required when any
	// route is gated.
	Privileged Actor
}

// Verify checks the declarations before any request is made. It returns the
// first problem found, so a broken suite fails loudly instead of silently
// skipping cases.
func (s Suite) Verify() error {
	if s.Handler == nil {
		return errorfamily.NewRejection("authztest.handler_nil", "Suite.Handler is required")
	}

	if len(s.Routes) == 0 {
		return errorfamily.NewRejection(
			"authztest.routes_empty",
			"Suite.Routes is empty — declare the surface you are proving",
		)
	}

	if s.Anonymous == nil {
		return errorfamily.NewRejection(
			"authztest.anonymous_nil",
			"Suite.Anonymous is required: every posture is proven against an anonymous request",
		)
	}

	for _, route := range s.Routes {
		if !route.Posture.Valid() {
			return errorfamily.NewRejection(
				"authztest.posture_unknown",
				fmt.Sprintf("%s declares posture %q, which is not public/authenticated/role", route, route.Posture),
			)
		}

		if route.Method == "" || route.Path == "" {
			return errorfamily.NewRejection(
				"authztest.route_incomplete",
				"every route needs a Method and a Path",
			)
		}

		if route.Why == "" {
			return errorfamily.NewRejection(
				"authztest.route_unjustified",
				fmt.Sprintf("%s declares a posture without a Why", route),
			)
		}

		if route.Posture != Public && s.Privileged == nil {
			return errorfamily.NewRejection(
				"authztest.privileged_nil",
				fmt.Sprintf("%s is gated but Suite.Privileged is nil", route),
			)
		}

		if route.Posture == RoleGated && s.Unprivileged == nil {
			return errorfamily.NewRejection(
				"authztest.unprivileged_nil",
				fmt.Sprintf("%s is role-gated but Suite.Unprivileged is nil — the 403 case cannot be proven", route),
			)
		}
	}

	return nil
}

// Run executes the matrix. Every failure is reported through tb, so one run
// lists every broken declaration instead of only the first.
func (s Suite) Run(t *testing.T) {
	t.Helper()

	if err := s.Verify(); err != nil {
		t.Fatalf("authztest suite is unusable: %v", err)

		return
	}

	for _, route := range s.Routes {
		t.Run(route.String(), func(t *testing.T) {
			t.Helper()
			s.probe(t, route)
		})
	}
}

// probe runs the postures' expected cases for one route.
func (s Suite) probe(tb testingTB, route Route) {
	anonStatus := s.status(route, s.Anonymous)

	switch route.Posture {
	case Public:
		if anonStatus == http.StatusUnauthorized || anonStatus == http.StatusForbidden {
			tb.Errorf(
				"declared public but refused an anonymous caller with %d — either the route is gated or the declaration is stale",
				anonStatus,
			)
		}

	case Authenticated:
		if !denied(anonStatus) {
			tb.Errorf(
				"declared authenticated but an anonymous caller got %d, want 401/403",
				anonStatus,
			)
		}

		if s.Unprivileged != nil {
			if got := s.status(route, s.Unprivileged); denied(got) {
				tb.Errorf("declared authenticated but an authenticated caller got %d", got)
			}
		}

		if got := s.status(route, s.Privileged); denied(got) {
			tb.Errorf("declared authenticated but the privileged caller got %d", got)
		}

	case RoleGated:
		if !denied(anonStatus) {
			tb.Errorf("declared role-gated but an anonymous caller got %d, want 401/403", anonStatus)
		}

		if got := s.status(route, s.Unprivileged); got != http.StatusForbidden {
			tb.Errorf(
				"role-gated route answered an authenticated caller WITHOUT the role with %d, want 403 (a 200 here is the missing-gate bug)",
				got,
			)
		}

		if got := s.status(route, s.Privileged); denied(got) {
			tb.Errorf("role-gated route refused the privileged caller with %d", got)
		}
	}
}

// status issues one probe and returns the response status.
func (s Suite) status(route Route, actor Actor) int {
	req := httptest.NewRequest(route.Method, route.Path, nil)

	if actor != nil {
		if withCreds := actor(req); withCreds != nil {
			req = withCreds
		}
	}

	rec := httptest.NewRecorder()
	s.Handler.ServeHTTP(rec, req)

	return rec.Code
}

// denied reports whether status is an authentication or authorization refusal.
func denied(status int) bool {
	return status == http.StatusUnauthorized || status == http.StatusForbidden
}

// RouteID identifies a registered route for coverage comparison.
type RouteID struct {
	Method string
	Path   string
}

// Undeclared returns the registered routes that carry no declaration.
//
// This is the direction that actually prevents the next audit: a new route that
// nobody declared is invisible to the matrix, and invisible is how a gate goes
// missing. Feed it the surface's own route table.
func Undeclared(declared []Route, registered []RouteID) []RouteID {
	known := make(map[string]struct{}, len(declared))
	for _, route := range declared {
		known[route.Method+" "+route.Path] = struct{}{}
	}

	var missing []RouteID

	for _, route := range registered {
		if _, ok := known[route.Method+" "+route.Path]; ok {
			continue
		}

		missing = append(missing, route)
	}

	return missing
}

// testingTB is the subset of *testing.T the probe helpers need. It keeps the
// per-route assertions usable from both the top-level test and a subtest handle.
type testingTB interface {
	Helper()
	Errorf(format string, args ...any)
}
