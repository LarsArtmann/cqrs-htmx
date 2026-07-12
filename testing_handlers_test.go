package cqrshtmx_test

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/casbin/casbin/v3"
	"github.com/casbin/casbin/v3/model"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	. "github.com/onsi/gomega"
)

func encodeJSONResult(w http.ResponseWriter, _ *http.Request, result any) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(result)
}

// registerBDDListUsers registers a "ListUsers" query handler that returns a
// single user (alice). Shared by BDD and integration tests.
func registerBDDListUsers(disp *query.Dispatcher) {
	_ = disp.Register("ListUsers", func(_ context.Context, _ query.Query) (any, error) {
		return []bddUser{{Email: aliceEmail, Name: aliceName}}, nil
	})
}

// registerGetUserEmail registers a "GetUser" query handler that returns
// {testEmailKey: testEmailValue}. Shared by app and benchmark tests.
func registerGetUserEmail(disp *query.Dispatcher) {
	_ = disp.Register("GetUser", func(_ context.Context, _ query.Query) (any, error) {
		return map[string]string{testEmailKey: testEmailValue}, nil
	})
}

func rejectionHandler(code, message string) func(context.Context, command.Command) error {
	return func(_ context.Context, _ command.Command) error {
		return errorfamily.NewRejection(code, message)
	}
}

func middlewareCaptureHandler(called *bool) http.Handler {
	return http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		*called = true
	})
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func createdHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
}

// --- Extractor helpers ---

func staticExtractor(uid cqrshtmx.UserID) cqrshtmx.UserIDExtractor {
	return func(_ *http.Request) (cqrshtmx.UserID, error) { return uid, nil }
}

func headerExtractor(header string) cqrshtmx.UserIDExtractor {
	return func(r *http.Request) (cqrshtmx.UserID, error) {
		return cqrshtmx.ParseUserID(r.Header.Get(header))
	}
}

// --- Tracking dispatcher helpers ---

func trackingCommandHandler(dispatched *bool) func(context.Context, command.Command) error {
	return func(_ context.Context, _ command.Command) error {
		*dispatched = true
		return nil
	}
}

// ctxCancelCommandHandler returns a command handler that blocks until the
// context is cancelled, then returns the cancellation error. Used to verify
// that timeouts propagate correctly through the dispatch path.
func ctxCancelCommandHandler(ctx context.Context, _ command.Command) error {
	<-ctx.Done()
	return ctx.Err()
}

// ctxCancelQueryHandler is the query-side counterpart of ctxCancelCommandHandler.
func ctxCancelQueryHandler(ctx context.Context, _ query.Query) (any, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func trackingAfterDispatchHook(called *bool, errOut *error) func(context.Context, *http.Request, error) {
	return func(_ context.Context, _ *http.Request, err error) {
		*called = true
		*errOut = err
	}
}

// --- Test enforcer ---

func newTestEnforcer() *casbin.Enforcer {
	m := model.NewModel()
	m.AddDef("r", "r", "sub, obj, act")
	m.AddDef("p", "p", "sub, obj, act")
	m.AddDef("e", "e", "some(where (p.eft == allow))")
	m.AddDef("m", "m", "r.sub == p.sub && r.obj == p.obj && r.act == p.act")

	e, _ := casbin.NewEnforcer(m)
	_, _ = e.AddPolicy(adminUserID.String(), "users", "create")
	_, _ = e.AddPolicy(adminUserID.String(), "users", "read")
	_, _ = e.AddPolicy(viewerUserID.String(), "users", "read")

	return e
}

// unauthenticatedReadMiddleware returns a middleware that requires read on users
// and extracts an empty (unauthenticated) UserID. Use in tests that exercise
// unauthenticated paths without rebuilding the middleware boilerplate.
func unauthenticatedReadMiddleware() func(http.Handler) http.Handler {
	return cqrshtmx.AuthorizeMiddleware(newTestEnforcer(), "users", "read",
		func(_ *http.Request) (cqrshtmx.UserID, error) { return cqrshtmx.UserID{}, nil })
}

// --- App creation helpers ---

func newCommandApp() *cqrshtmx.App {
	disp := command.NewDispatcher()
	_ = disp.Register("CreateUser", noOpCommandHandler)
	cfg := cqrshtmx.Config{Commands: disp}
	app, err := cqrshtmx.New(cfg)
	Expect(err).NotTo(HaveOccurred())
	return app
}
