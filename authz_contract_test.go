package cqrshtmx_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// fourArgEnforcer models the library's shipped RBAC model
// (subject, domain, object, action) and rejects every other arity the way
// Casbin does.
type fourArgEnforcer struct {
	allowed bool
	last    []any
}

func (e *fourArgEnforcer) Enforce(rvals ...any) (bool, error) {
	if len(rvals) != 4 {
		return false, errorfamily.NewRejection(
			"invalid_request_size",
			fmt.Sprintf("invalid request size: expected 4, got %d", len(rvals)),
		)
	}

	e.last = rvals

	return e.allowed, nil
}

// threeArgEnforcer models a consumer's custom model (subject, object, action).
type threeArgEnforcer struct {
	allowed bool
	calls   int
	last    []any
}

func (e *threeArgEnforcer) Enforce(rvals ...any) (bool, error) {
	if len(rvals) != 3 {
		return false, errorfamily.NewRejection(
			"invalid_request_size",
			fmt.Sprintf("invalid request size: expected 3, got %d", len(rvals)),
		)
	}

	e.calls++
	e.last = rvals

	return e.allowed, nil
}

// recordingChecker is a PermissionChecker stub that records the question.
type recordingChecker struct {
	allowed bool
	matched []string
	err     error

	calls   int
	subject string
	domain  string
	object  string
	action  string
}

func (c *recordingChecker) Check(subject, domain, resource, action string) (bool, []string, error) {
	c.calls++
	c.subject = subject
	c.domain = domain
	c.object = resource
	c.action = action

	return c.allowed, c.matched, c.err
}

func TestResourceString(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		resource cqrshtmx.Resource
		want     string
	}{
		{"zero", cqrshtmx.Resource{}, ""},
		{"type only", cqrshtmx.Resource{Type: "orders"}, "orders"},
		{"id only", cqrshtmx.Resource{ID: "42"}, "42"},
		{"type and id", cqrshtmx.Resource{Type: "orders", ID: "42"}, "orders:42"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := tc.resource.String(); got != tc.want {
				t.Fatalf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDecide_Reasons(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		checker *recordingChecker
		req     cqrshtmx.Request
		want    cqrshtmx.DecisionReason
		allowed bool
	}{
		{
			name:    "allowed",
			checker: &recordingChecker{allowed: true, matched: []string{"p, admin, *, *, *"}},
			req:     cqrshtmx.Request{Subject: "u1", Resource: cqrshtmx.Resource{Type: "orders"}, Action: "read"},
			want:    cqrshtmx.ReasonAllowed,
			allowed: true,
		},
		{
			name:    "explicit deny carries the matched rule",
			checker: &recordingChecker{allowed: false, matched: []string{"p, user, *, orders, read, deny"}},
			req:     cqrshtmx.Request{Subject: "u1", Resource: cqrshtmx.Resource{Type: "orders"}, Action: "read"},
			want:    cqrshtmx.ReasonDeniedByPolicy,
			allowed: false,
		},
		{
			name:    "no matching rule is a denial of a different kind",
			checker: &recordingChecker{allowed: false, matched: nil},
			req:     cqrshtmx.Request{Subject: "u1", Resource: cqrshtmx.Resource{Type: "orders"}, Action: "read"},
			want:    cqrshtmx.ReasonNoMatchingPolicy,
			allowed: false,
		},
		{
			name:    "no subject never reaches the engine",
			checker: &recordingChecker{allowed: true},
			req:     cqrshtmx.Request{Resource: cqrshtmx.Resource{Type: "orders"}, Action: "read"},
			want:    cqrshtmx.ReasonNotAuthenticated,
			allowed: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			decision, err := cqrshtmx.Decide(tc.checker, tc.req)
			if err != nil {
				t.Fatalf("Decide: %v", err)
			}

			if decision.Reason != tc.want {
				t.Fatalf("reason = %q, want %q", decision.Reason, tc.want)
			}

			if decision.Allowed != tc.allowed {
				t.Fatalf("allowed = %v, want %v", decision.Allowed, tc.allowed)
			}

			if tc.req.Subject == "" && tc.checker.calls != 0 {
				t.Fatal("an unauthenticated question must not consult the engine")
			}
		})
	}
}

func TestDecide_EmptyDomainUsesWildcard(t *testing.T) {
	t.Parallel()

	checker := &recordingChecker{allowed: true}

	if _, err := cqrshtmx.Decide(checker, cqrshtmx.Request{
		Subject:  "u1",
		Resource: cqrshtmx.Resource{Type: "orders", ID: "42"},
		Action:   "read",
	}); err != nil {
		t.Fatalf("Decide: %v", err)
	}

	if checker.domain != cqrshtmx.WildcardDomain {
		t.Fatalf("domain = %q, want %q", checker.domain, cqrshtmx.WildcardDomain)
	}

	if checker.object != "orders:42" {
		t.Fatalf("object = %q, want orders:42", checker.object)
	}
}

func TestDecide_EngineErrorIsNotADenial(t *testing.T) {
	t.Parallel()

	engineErr := errorfamily.NewInfrastructure("boom", "engine exploded")

	_, err := cqrshtmx.Decide(&recordingChecker{err: engineErr}, cqrshtmx.Request{
		Subject:  "u1",
		Resource: cqrshtmx.Resource{Type: "orders"},
		Action:   "read",
	})
	if err == nil {
		t.Fatal("an engine failure must surface as an error, never as a silent denial")
	}

	if cqrshtmx.IsForbidden(err) {
		t.Fatal("an engine failure must not be classified as a permission denial")
	}
}

func TestDecide_NilChecker(t *testing.T) {
	t.Parallel()

	_, err := cqrshtmx.Decide(nil, cqrshtmx.Request{
		Subject:  "u1",
		Resource: cqrshtmx.Resource{Type: "orders"},
		Action:   "read",
	})
	if err == nil {
		t.Fatal("expected an error for a nil checker")
	}
}

// TestEnforce_ThreeArgModelStillWorks pins the arity adaptation: consumers with
// their own (subject, object, action) model must keep working.
func TestEnforce_ThreeArgModelStillWorks(t *testing.T) {
	t.Parallel()

	e := &threeArgEnforcer{allowed: true}

	if err := cqrshtmx.Enforce(e, "u1", "users", "read"); err != nil {
		t.Fatalf("Enforce: %v", err)
	}

	if e.calls != 1 || len(e.last) != 3 {
		t.Fatalf("expected exactly one three-argument call, got %d call(s) with %d arg(s)", e.calls, len(e.last))
	}
}

// TestEnforce_FourArgModelReturnsForbiddenNotEngineError is the 403-not-503
// regression: against the shipped four-field model the old three-argument call
// was a request-size error, which surfaced as 503 Service Unavailable.
func TestEnforce_FourArgModelReturnsForbiddenNotEngineError(t *testing.T) {
	t.Parallel()

	err := cqrshtmx.Enforce(&fourArgEnforcer{allowed: false}, "u1", "users", "read")
	if err == nil {
		t.Fatal("expected a denial")
	}

	if !cqrshtmx.IsForbidden(err) {
		t.Fatalf("denial must classify as forbidden, got %v", err)
	}

	if cqrshtmx.MapError(err) != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", cqrshtmx.MapError(err))
	}
}

func TestEnforceIn_PassesDomain(t *testing.T) {
	t.Parallel()

	if err := cqrshtmx.EnforceIn(&fourArgEnforcer{allowed: true}, "u1", "tenant-7", "orders", "read"); err != nil {
		t.Fatalf("EnforceIn: %v", err)
	}
}

func TestIsForbidden_CrossModuleCodes(t *testing.T) {
	t.Parallel()

	// A module that cannot import cqrshtmx produces its own sentinel with the
	// shared code; the helper must recognise it.
	foreign := errorfamily.NewRejection(cqrshtmx.CodeForbidden, "access denied")
	wrapped := errorfamily.Wrapf(foreign, errorfamily.Rejection, "other.module.denied", "wrapped")

	if !cqrshtmx.IsForbidden(foreign) {
		t.Fatal("foreign forbidden code was not recognised")
	}

	if !cqrshtmx.IsForbidden(wrapped) {
		t.Fatal("wrapped foreign forbidden code was not recognised")
	}

	if cqrshtmx.IsForbidden(nil) {
		t.Fatal("nil is not a forbidden error")
	}

	if cqrshtmx.IsForbidden(errorfamily.NewRejection("unrelated", "nope")) {
		t.Fatal("an unrelated rejection must not classify as forbidden")
	}
}

func TestIsUnauthorized_CrossModuleCodes(t *testing.T) {
	t.Parallel()

	foreign := errorfamily.NewRejection(cqrshtmx.CodeUnauthorized, "authentication required")

	if !cqrshtmx.IsUnauthorized(foreign) {
		t.Fatal("foreign unauthorized code was not recognised")
	}

	if cqrshtmx.IsUnauthorized(errorfamily.NewRejection(cqrshtmx.CodeForbidden, "denied")) {
		t.Fatal("forbidden must not classify as unauthorized")
	}
}

// TestAuthorize_HandlerDeniesWith403Not503 is the end-to-end regression: a
// handler authorized against the shipped model must answer 403, not 503.
func TestAuthorize_HandlerDeniesWith403Not503(t *testing.T) {
	t.Parallel()

	disp := query.NewDispatcher()

	if err := disp.Register("GetPage", func(context.Context, query.Query) (any, error) {
		return nil, nil
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	app, err := cqrshtmx.New(cqrshtmx.Config{Queries: disp, Enforcer: &fourArgEnforcer{allowed: false}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	handler := app.Query("GetPage",
		cqrshtmx.DecodeJSONQuery(func(_ struct{}) (query.Query, error) { return &getPageQuery{}, nil }),
		cqrshtmx.Authorize("users", "read"),
	)

	req := newPostJSONRequest(`{}`)
	req = req.WithContext(cqrshtmx.WithUserID(req.Context(), adminUserID))

	w := serve(handler, req)
	if w.code() != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (%s)", w.code(), w.Body.String())
	}
}

// TestAuthorizeRequest_ResolvesPerRequest covers the resolver-based option: the
// question comes from the request, and an empty Subject falls back to the
// authenticated user.
func TestAuthorizeRequest_ResolvesPerRequest(t *testing.T) {
	t.Parallel()

	disp := query.NewDispatcher()

	if err := disp.Register("GetPage", func(context.Context, query.Query) (any, error) {
		return nil, nil
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	enforcer := &fourArgEnforcer{allowed: false}

	app, err := cqrshtmx.New(cqrshtmx.Config{Queries: disp, Enforcer: enforcer})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	handler := app.Query("GetPage",
		cqrshtmx.DecodeJSONQuery(func(_ struct{}) (query.Query, error) { return &getPageQuery{}, nil }),
		cqrshtmx.AuthorizeRequest(func(r *http.Request) (cqrshtmx.Request, error) {
			return cqrshtmx.Request{
				Domain:   r.URL.Query().Get("tenant"),
				Resource: cqrshtmx.Resource{Type: "orders"},
				Action:   "read",
			}, nil
		}),
	)

	req := newPostJSONRequest(`{}`)
	req = req.WithContext(cqrshtmx.WithUserID(req.Context(), adminUserID))

	w := serve(handler, req)
	if w.code() != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (%s)", w.code(), w.Body.String())
	}

	// The resolver left Subject empty, so the authenticated user from the
	// request context must have filled it in.
	if len(enforcer.last) != 4 {
		t.Fatalf("engine saw %d argument(s), want 4", len(enforcer.last))
	}

	if got := enforcer.last[0]; got != adminUserID.String() {
		t.Fatalf("subject = %v, want the authenticated user %q", got, adminUserID.String())
	}

	if got := enforcer.last[2]; got != "orders" {
		t.Fatalf("object = %v, want orders", got)
	}
}

// TestAuthorizeRequest_EngineErrorIs503 proves an engine failure is not masked
// as a denial.
func TestAuthorizeRequest_EngineErrorIs503(t *testing.T) {
	t.Parallel()

	disp := query.NewDispatcher()

	if err := disp.Register("GetPage", func(context.Context, query.Query) (any, error) {
		return nil, nil
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	app, err := cqrshtmx.New(cqrshtmx.Config{Queries: disp, Enforcer: erroringEnforcer{}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	handler := app.Query("GetPage",
		cqrshtmx.DecodeJSONQuery(func(_ struct{}) (query.Query, error) { return &getPageQuery{}, nil }),
		cqrshtmx.Authorize("users", "read"),
	)

	req := newPostJSONRequest(`{}`)
	req = req.WithContext(cqrshtmx.WithUserID(req.Context(), adminUserID))

	w := serve(handler, req)
	if w.code() != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 for a broken engine (%s)", w.code(), w.Body.String())
	}
}

// erroringEnforcer fails every call with a non-arity error.
type erroringEnforcer struct{}

func (erroringEnforcer) Enforce(...any) (bool, error) {
	return false, errorfamily.NewInfrastructure("engine_down", "policy engine unavailable")
}
