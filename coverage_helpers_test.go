package cqrshtmx_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
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
	_ = disp.Register("CreateUser", func(_ context.Context, _ command.Command) error { return nil })
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

func newCommandAppWithHandler(handler func(context.Context, command.Command) error) *cqrshtmx.App {
	disp := command.NewDispatcher()
	_ = disp.Register("CreateUser", handler)
	app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
	Expect(err).NotTo(HaveOccurred())
	return app
}
