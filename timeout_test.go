package cqrshtmx_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx"
	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/query"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func newTimeoutCommandApp(
	handler func(context.Context, command.Command) error,
	timeout time.Duration,
) *cqrshtmx.App {
	disp := command.NewDispatcher()
	_ = disp.Register("CreateUser", handler)
	app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp, Timeout: timeout})
	Expect(err).NotTo(HaveOccurred())
	return app
}

func newTimeoutQueryApp(
	handler func(context.Context, query.Query) (any, error),
	timeout time.Duration,
) *cqrshtmx.App {
	disp := query.NewDispatcher()
	_ = disp.Register("GetUser", handler)
	app, err := cqrshtmx.New(cqrshtmx.Config{Queries: disp, Timeout: timeout})
	Expect(err).NotTo(HaveOccurred())
	return app
}

var _ = Describe("Timeout Support", func() {
	Describe("Command dispatch with timeout", func() {
		It("cancels command handler when timeout is exceeded", func() {
			app := newTimeoutCommandApp(func(ctx context.Context, _ command.Command) error {
				<-ctx.Done()
				return ctx.Err()
			}, 50*time.Millisecond)

			r := httptest.NewRequest(http.MethodPost, "/slow", strings.NewReader(`{}`))
			w := serve(app.Command("CreateUser", decodeCreateUserJSON()), r)
			Expect(w.code()).To(Equal(http.StatusServiceUnavailable))
		})

		It("completes command within timeout", func() {
			var dispatched bool
			app := newTimeoutCommandApp(trackingCommandHandler(&dispatched), 5*time.Second)

			r := httptest.NewRequest(http.MethodPost, "/fast", strings.NewReader(`{}`))
			w := serve(app.Command("CreateUser", decodeCreateUserJSON()), r)
			Expect(dispatched).To(BeTrue())
			Expect(w.code()).To(Equal(http.StatusNoContent))
		})
	})

	Describe("Query dispatch with timeout", func() {
		It("cancels query handler when timeout is exceeded", func() {
			app := newTimeoutQueryApp(func(ctx context.Context, _ query.Query) (any, error) {
				<-ctx.Done()
				return nil, ctx.Err()
			}, 50*time.Millisecond)

			r := httptest.NewRequest(http.MethodGet, "/slow", strings.NewReader(`{}`))
			w := serve(app.Query("GetUser", decodeGetUserJSONQuery()), r)
			Expect(w.code()).To(Equal(http.StatusServiceUnavailable))
		})

		It("completes query within timeout", func() {
			app := newTimeoutQueryApp(func(_ context.Context, _ query.Query) (any, error) {
				return testQueryResult, nil
			}, 5*time.Second)

			r := httptest.NewRequest(http.MethodGet, "/fast", strings.NewReader(`{}`))
			w := serve(app.Query("GetUser",
				decodeGetUserJSONQuery(),
				cqrshtmx.Render(encodeJSONResult),
			), r)
			Expect(w.code()).To(Equal(http.StatusOK))
		})
	})

	Describe("Timeout edge cases", func() {
		It("does not apply timeout when zero", func() {
			var dispatched bool
			app := newTimeoutCommandApp(trackingCommandHandler(&dispatched), 0)
			serve(app.Command("CreateUser", decodeCreateUserJSON()),
				newPostRequest("/users", `{}`))
			Expect(dispatched).To(BeTrue())
		})

		It("does not apply timeout when negative", func() {
			var dispatched bool
			app := newTimeoutCommandApp(trackingCommandHandler(&dispatched), -1*time.Second)
			serve(app.Command("CreateUser", decodeCreateUserJSON()),
				newPostRequest("/users", `{}`))
			Expect(dispatched).To(BeTrue())
		})

		It("timeout context respects BeforeDispatch modifications", func() {
			var hasDeadline bool
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", func(ctx context.Context, _ command.Command) error {
				_, hasDeadline = ctx.Deadline()
				return nil
			})
			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands: disp,
				Timeout:  1 * time.Second,
				BeforeDispatch: func(ctx context.Context, _ *http.Request) context.Context {
					return ctx
				},
			})
			Expect(err).NotTo(HaveOccurred())
			serve(app.Command("CreateUser", decodeCreateUserJSON()),
				newPostRequest("/users", `{}`))
			Expect(hasDeadline).To(BeTrue())
		})
	})
})
