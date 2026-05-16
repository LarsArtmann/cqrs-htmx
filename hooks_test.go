package cqrshtmx_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"

	cqrshtmx "github.com/larsartmann/cqrs-htmx"
	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/query"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type (
	testKey  string
	queryKey string
)

var _ = Describe("Lifecycle Hooks", func() {
	Describe("BeforeDispatch", func() {
		It("modifies context before command dispatch", func() {
			var capturedContext context.Context
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", func(ctx context.Context, _ command.Command) error {
				capturedContext = ctx
				return nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands: disp,
				BeforeDispatch: func(ctx context.Context, _ *http.Request) context.Context {
					return context.WithValue(ctx, testKey("test-key"), "test-value")
				},
			})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Command("CreateUser", decodeCreateUserJSON())
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader("{}"))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			Expect(capturedContext.Value(testKey("test-key"))).To(Equal("test-value"))
		})

		It("modifies context before query dispatch", func() {
			var capturedContext context.Context
			disp := query.NewDispatcher()
			_ = disp.Register("GetUser", func(ctx context.Context, _ query.Query) (any, error) {
				capturedContext = ctx
				return "retrieved", nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{
				Queries: disp,
				BeforeDispatch: func(ctx context.Context, _ *http.Request) context.Context {
					return context.WithValue(ctx, queryKey("query-key"), "query-value")
				},
			})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Query("GetUser", decodeGetUserJSONQuery())
			r := httptest.NewRequest(http.MethodGet, "/user", strings.NewReader("{}"))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			Expect(capturedContext.Value(queryKey("query-key"))).To(Equal("query-value"))
		})
	})

	Describe("AfterDispatch", func() {
		It("is called after successful command dispatch", func() {
			var afterCalled bool
			var afterErr error
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", func(_ context.Context, _ command.Command) error {
				return nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands: disp,
				AfterDispatch: func(_ context.Context, _ *http.Request, err error) {
					afterCalled = true
					afterErr = err
				},
			})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Command("CreateUser", decodeCreateUserJSON())
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader("{}"))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			Expect(afterCalled).To(BeTrue())
			Expect(afterErr).NotTo(HaveOccurred())
		})

		It("is called with error on failed command dispatch", func() {
			var afterCalled bool
			var afterErr error
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", func(_ context.Context, _ command.Command) error {
				return cqrshtmx.ErrDecodeFailed
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands: disp,
				AfterDispatch: func(_ context.Context, _ *http.Request, err error) {
					afterCalled = true
					afterErr = err
				},
			})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Command("CreateUser", decodeCreateUserJSON())
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader("{}"))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			Expect(afterCalled).To(BeTrue())
			Expect(afterErr).To(HaveOccurred())
		})
	})

	Describe("Correlation IDs", func() {
		It("propagates X-Correlation-ID through middleware", func() {
			var capturedCID string
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", func(ctx context.Context, _ command.Command) error {
				capturedCID = cqrshtmx.CorrelationIDFromContext(ctx)
				return nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands: disp,
			})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Middleware()(app.Command("CreateUser", decodeCreateUserJSON()))
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader("{}"))
			r.Header.Set("X-Correlation-ID", "abc-123")
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			Expect(capturedCID).To(Equal("abc-123"))
		})

		It("stores and retrieves correlation ID directly", func() {
			ctx := context.Background()
			ctx = cqrshtmx.WithCorrelationID(ctx, "test-cid")
			Expect(cqrshtmx.CorrelationIDFromContext(ctx)).To(Equal("test-cid"))
		})

		It("returns empty string when no correlation ID", func() {
			Expect(cqrshtmx.CorrelationIDFromContext(context.Background())).To(BeEmpty())
		})
	})
})
