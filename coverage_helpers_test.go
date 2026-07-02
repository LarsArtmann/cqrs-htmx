package cqrshtmx_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
	. "github.com/onsi/gomega"
)

const testQueryResult = "result"

func queryNamedResultHandler(name string) func(context.Context, query.Query) (any, error) {
	return func(_ context.Context, _ query.Query) (any, error) {
		return map[string]string{testNameKey: name}, nil
	}
}

func testNotificationTrigger(opt cqrshtmx.HandlerOption, expectedLevel string) {
	disp := command.NewDispatcher()
	_ = disp.Register("CreateUser", noOpCommandHandler)
	app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
	Expect(err).NotTo(HaveOccurred())

	r := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/users",
		strings.NewReader(`{}`),
	)
	r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
	w := serve(app.Command("CreateUser", decodeCreateUserJSON(), opt), r)
	trigger := w.Header().Get("HX-Trigger")
	Expect(trigger).To(ContainSubstring(expectedLevel))
}

func newQueryAppWithResult(handler func(context.Context, query.Query) (any, error)) *cqrshtmx.App {
	disp := query.NewDispatcher()
	_ = disp.Register("GetUser", handler)
	app, err := cqrshtmx.New(cqrshtmx.Config{Queries: disp})
	Expect(err).NotTo(HaveOccurred())
	return app
}

// testResultQueryHandler returns a query handler that yields testQueryResult.
// Used as the no-op success body in coverage/validation/error-path tests.
func testResultQueryHandler() func(context.Context, query.Query) (any, error) {
	return func(_ context.Context, _ query.Query) (any, error) {
		return testQueryResult, nil
	}
}

// constantQueryHandler returns a query handler that always yields the given value.
func constantQueryHandler(v any) func(context.Context, query.Query) (any, error) {
	return func(_ context.Context, _ query.Query) (any, error) {
		return v, nil
	}
}

func newQueryAppNamed(name string, handler func(context.Context, query.Query) (any, error)) *cqrshtmx.App {
	disp := query.NewDispatcher()
	_ = disp.Register(query.Type(name), handler)
	app, err := cqrshtmx.New(cqrshtmx.Config{Queries: disp})
	Expect(err).NotTo(HaveOccurred())
	return app
}

// newQueryAppWithConfig builds a query app with the given config overrides
// (e.g., custom AfterDispatch hook) and a fixed GetUser handler.
func newQueryAppWithConfig(handler func(context.Context, query.Query) (any, error), cfg cqrshtmx.Config) *cqrshtmx.App {
	disp := query.NewDispatcher()
	_ = disp.Register("GetUser", handler)
	cfg.Queries = disp
	app, err := cqrshtmx.New(cfg)
	Expect(err).NotTo(HaveOccurred())
	return app
}

func newCommandAppWithHandler(handler func(context.Context, command.Command) error) *cqrshtmx.App {
	disp := command.NewDispatcher()
	_ = disp.Register("CreateUser", handler)
	app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
	Expect(err).NotTo(HaveOccurred())
	return app
}

// newCommandAppWithConfig builds a command app with the given config overrides
// (e.g., custom AfterDispatch hook) and a fixed CreateUser handler.
func newCommandAppWithConfig(handler func(context.Context, command.Command) error, cfg cqrshtmx.Config) *cqrshtmx.App {
	disp := command.NewDispatcher()
	_ = disp.Register("CreateUser", handler)
	cfg.Commands = disp
	app, err := cqrshtmx.New(cfg)
	Expect(err).NotTo(HaveOccurred())
	return app
}
