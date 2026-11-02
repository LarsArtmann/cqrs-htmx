package authztest

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

// fixtureMux is a miniature surface with all three postures.
//
//	GET /public  — always 200
//	GET /private — 200 with any credential, 401 without
//	GET /admin   — 200 as "admin", 403 as "user", 401 without
func fixtureMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /public", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /private", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Actor") == "" {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /admin", func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("X-Actor") {
		case "":
			w.WriteHeader(http.StatusUnauthorized)
		case "admin":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusForbidden)
		}
	})

	return mux
}

func asActor(name string) Actor {
	return func(r *http.Request) *http.Request {
		r.Header.Set("X-Actor", name)

		return r
	}
}

func anonymous(r *http.Request) *http.Request { return r }

// reporter collects the failures probe() reports, so a case can assert on the
// message instead of depending on a nested test's failed state.
type reporter struct {
	messages []string
}

func (r *reporter) Helper() {}

func (r *reporter) Errorf(format string, args ...any) {
	r.messages = append(r.messages, strings.TrimSpace(fmt.Sprintf(format, args...)))
}

func (r *reporter) failed() bool { return len(r.messages) > 0 }

func (r *reporter) joined() string { return strings.Join(r.messages, " | ") }

func correctSuite() Suite {
	return Suite{
		Handler: fixtureMux(),
		Routes: []Route{
			{Method: http.MethodGet, Path: "/public", Posture: Public, Why: "landing page"},
			{Method: http.MethodGet, Path: "/private", Posture: Authenticated, Why: "any signed-in user"},
			{Method: http.MethodGet, Path: "/admin", Posture: RoleGated, Why: "admin only"},
		},
		Anonymous:    anonymous,
		Unprivileged: asActor("user"),
		Privileged:   asActor("admin"),
	}
}

func TestProbe_CorrectDeclarationsPass(t *testing.T) {
	t.Parallel()

	for _, route := range correctSuite().Routes {
		rec := &reporter{}
		correctSuite().probe(rec, route)

		if rec.failed() {
			t.Errorf("%s: unexpected failure(s): %s", route, rec.joined())
		}
	}
}

func TestProbe_GatedRouteDeclaredPublicFails(t *testing.T) {
	t.Parallel()

	suite := correctSuite()
	route := Route{Method: http.MethodGet, Path: "/admin", Posture: Public, Why: "wrong on purpose"}

	rec := &reporter{}
	suite.probe(rec, route)

	if !rec.failed() {
		t.Fatal("a gated route declared public must fail the probe")
	}

	if !strings.Contains(rec.joined(), "declared public") {
		t.Fatalf("unexpected message: %s", rec.joined())
	}
}

func TestProbe_OpenRouteDeclaredRoleGatedFails(t *testing.T) {
	t.Parallel()

	suite := correctSuite()
	route := Route{Method: http.MethodGet, Path: "/public", Posture: RoleGated, Why: "wrong on purpose"}

	rec := &reporter{}
	suite.probe(rec, route)

	if !rec.failed() {
		t.Fatal("an ungated route declared role-gated must fail the probe")
	}
}

// TestProbe_MissingGateIsReportedWithTheStatusCode is the failure this package
// exists for: a role-gated route that answers 200 to a caller without the role.
func TestProbe_MissingGateIsReportedWithTheStatusCode(t *testing.T) {
	t.Parallel()

	// /private admits ANY credential, so probing it as role-gated finds the
	// missing role check.
	suite := correctSuite()
	route := Route{Method: http.MethodGet, Path: "/private", Posture: RoleGated, Why: "should be admin only"}

	rec := &reporter{}
	suite.probe(rec, route)

	if !rec.failed() {
		t.Fatal("a route without a role check must fail a role-gated declaration")
	}

	if !strings.Contains(rec.joined(), "want 403") {
		t.Fatalf("message must name the expected status: %s", rec.joined())
	}
}

func TestProbe_AnonymousAllowedOnGatedRouteFails(t *testing.T) {
	t.Parallel()

	suite := correctSuite()
	suite.Handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	route := Route{Method: http.MethodGet, Path: "/private", Posture: Authenticated, Why: "should be gated"}

	rec := &reporter{}
	suite.probe(rec, route)

	if !rec.failed() {
		t.Fatal("an ungated route declared authenticated must fail the probe")
	}
}

func TestVerify_RejectsUntestableSuites(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		mutate func(*Suite)
		code   string
	}{
		{"no handler", func(s *Suite) { s.Handler = nil }, "authztest.handler_nil"},
		{"no routes", func(s *Suite) { s.Routes = nil }, "authztest.routes_empty"},
		{"no anonymous actor", func(s *Suite) { s.Anonymous = nil }, "authztest.anonymous_nil"},
		{
			"no privileged actor",
			func(s *Suite) { s.Privileged = nil },
			"authztest.privileged_nil",
		},
		{
			"no unprivileged actor for a role-gated route",
			func(s *Suite) { s.Unprivileged = nil },
			"authztest.unprivileged_nil",
		},
		{
			"undeclared posture",
			func(s *Suite) { s.Routes[0].Posture = "open" },
			"authztest.posture_unknown",
		},
		{
			"route without a reason",
			func(s *Suite) { s.Routes[0].Why = "" },
			"authztest.route_unjustified",
		},
		{
			"route without a path",
			func(s *Suite) { s.Routes[0].Path = "" },
			"authztest.route_incomplete",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			suite := correctSuite()
			tc.mutate(&suite)

			err := suite.Verify()
			if err == nil {
				t.Fatal("expected a verification error")
			}

			if errorfamily.Code(err) != tc.code && !strings.Contains(err.Error(), tc.code) {
				t.Fatalf("error %q does not carry code %q", err, tc.code)
			}
		})
	}
}

func TestVerify_AcceptsCorrectSuite(t *testing.T) {
	t.Parallel()

	if err := correctSuite().Verify(); err != nil {
		t.Fatalf("correct suite rejected: %v", err)
	}
}

func TestRun_ExecutesEveryRoute(t *testing.T) {
	t.Parallel()

	correctSuite().Run(t)
}

func TestUndeclared_FindsMissingDeclarations(t *testing.T) {
	t.Parallel()

	declared := correctSuite().Routes
	registered := []RouteID{
		{Method: http.MethodGet, Path: "/public"},
		{Method: http.MethodGet, Path: "/private"},
		{Method: http.MethodGet, Path: "/admin"},
		{Method: http.MethodPost, Path: "/admin/users"},
	}

	missing := Undeclared(declared, registered)
	if len(missing) != 1 || missing[0].Path != "/admin/users" {
		t.Fatalf("missing = %v, want exactly POST /admin/users", missing)
	}
}

func TestPostureValid(t *testing.T) {
	t.Parallel()

	for _, p := range []Posture{Public, Authenticated, RoleGated} {
		if !p.Valid() {
			t.Errorf("%q should be valid", p)
		}
	}

	if Posture("admin-only").Valid() {
		t.Error("an invented posture must not validate")
	}
}

// TestStatus_AppliesActorCredentials proves the probe actually sends the actor,
// so a matrix that passes is not passing on an unauthenticated request.
func TestStatus_AppliesActorCredentials(t *testing.T) {
	t.Parallel()

	suite := correctSuite()
	route := Route{Method: http.MethodGet, Path: "/admin", Posture: RoleGated, Why: "admin only"}

	if got := suite.status(route, asActor("admin")); got != http.StatusOK {
		t.Fatalf("privileged status = %d, want 200", got)
	}

	if got := suite.status(route, asActor("user")); got != http.StatusForbidden {
		t.Fatalf("unprivileged status = %d, want 403", got)
	}

	if got := suite.status(route, nil); got != http.StatusUnauthorized {
		t.Fatalf("anonymous status = %d, want 401", got)
	}
}

// TestStatus_NilFromActorLeavesRequestUntouched covers the documented contract:
// an actor returning nil keeps the original request.
func TestStatus_NilFromActorLeavesRequestUntouched(t *testing.T) {
	t.Parallel()

	suite := correctSuite()
	route := Route{Method: http.MethodGet, Path: "/admin", Posture: RoleGated, Why: "admin only"}

	nilActor := func(*http.Request) *http.Request { return nil }

	if got := suite.status(route, nilActor); got != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (untouched request)", got)
	}
}

func TestRouteString(t *testing.T) {
	t.Parallel()

	route := Route{Method: http.MethodPost, Path: "/admin/users", Posture: RoleGated, Why: "x"}
	if got := route.String(); got != "POST /admin/users" {
		t.Fatalf("String() = %q", got)
	}
}
