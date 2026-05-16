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

var _ = Describe("Timeout Support", func() {
	Describe("Command dispatch with timeout", func() {
		It("cancels command handler when timeout is exceeded", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", func(ctx context.Context, _ command.Command) error {
				<-ctx.Done()
				return ctx.Err()
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands: disp,
				Timeout:  50 * time.Millisecond,
			})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Command("CreateUser", decodeCreateUserJSON())

			r := httptest.NewRequest(http.MethodPost, "/slow", strings.NewReader(`{}`))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusServiceUnavailable))
		})

		It("completes command within timeout", func() {
			disp := command.NewDispatcher()
			dispatched := false
			_ = disp.Register("CreateUser", func(_ context.Context, _ command.Command) error {
				dispatched = true
				return nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands: disp,
				Timeout:  5 * time.Second,
			})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Command("CreateUser", decodeCreateUserJSON())

			r := httptest.NewRequest(http.MethodPost, "/fast", strings.NewReader(`{}`))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			Expect(dispatched).To(BeTrue())
			Expect(w.Code).To(Equal(http.StatusNoContent))
		})
	})

	Describe("Query dispatch with timeout", func() {
		It("cancels query handler when timeout is exceeded", func() {
			disp := query.NewDispatcher()
			_ = disp.Register("GetUser", func(ctx context.Context, _ query.Query) (any, error) {
				<-ctx.Done()
				return nil, ctx.Err()
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{
				Queries: disp,
				Timeout: 50 * time.Millisecond,
			})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Query("GetUser", decodeGetUserJSONQuery())

			r := httptest.NewRequest(http.MethodGet, "/slow", strings.NewReader(`{}`))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusServiceUnavailable))
		})

		It("completes query within timeout", func() {
			disp := query.NewDispatcher()
			_ = disp.Register("GetUser", func(_ context.Context, _ query.Query) (any, error) {
				return testQueryResult, nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{
				Queries: disp,
				Timeout: 5 * time.Second,
			})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Query("GetUser",
				decodeGetUserJSONQuery(),
				cqrshtmx.Render(encodeJSONResult),
			)

			r := httptest.NewRequest(http.MethodGet, "/fast", strings.NewReader(`{}`))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusOK))
		})
	})

	Describe("Timeout edge cases", func() {
		It("does not apply timeout when zero", func() {
			disp := command.NewDispatcher()
			dispatched := false
			_ = disp.Register("CreateUser", func(_ context.Context, _ command.Command) error {
				dispatched = true
				return nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands: disp,
				Timeout:  0,
			})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Command("CreateUser", decodeCreateUserJSON())

			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{}`))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			Expect(dispatched).To(BeTrue())
		})

		It("does not apply timeout when negative", func() {
			disp := command.NewDispatcher()
			dispatched := false
			_ = disp.Register("CreateUser", func(_ context.Context, _ command.Command) error {
				dispatched = true
				return nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands: disp,
				Timeout:  -1 * time.Second,
			})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Command("CreateUser", decodeCreateUserJSON())

			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{}`))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			Expect(dispatched).To(BeTrue())
		})

		It("timeout context respects BeforeDispatch modifications", func() {
			disp := command.NewDispatcher()
			var hasDeadline bool
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

			handler := app.Command("CreateUser", decodeCreateUserJSON())

			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{}`))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			Expect(hasDeadline).To(BeTrue())
		})
	})
})
