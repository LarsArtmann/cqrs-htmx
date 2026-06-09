package cqrshtmx_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
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

			serve(app.Command("CreateUser", decodeCreateUserJSON()),
				newPostRequest("/users", "{}"))
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

			serve(app.Query("GetUser", decodeGetUserJSONQuery()),
				httptest.NewRequest(http.MethodGet, "/user", strings.NewReader("{}")))
			Expect(capturedContext.Value(queryKey("query-key"))).To(Equal("query-value"))
		})
	})

	Describe("AfterDispatch", func() {
		trackAfterDispatch := func() (called *bool, capturedErr *error, hook func(context.Context, *http.Request, error)) {
			var c bool
			var e error
			return &c, &e, func(_ context.Context, _ *http.Request, err error) {
				c = true
				e = err
			}
		}

		It("is called after successful command dispatch", func() {
			called, capturedErr, hook := trackAfterDispatch()
			disp := command.NewDispatcher()
			_ = disp.Register(
				"CreateUser",
				func(_ context.Context, _ command.Command) error { return nil },
			)

			app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp, AfterDispatch: hook})
			Expect(err).NotTo(HaveOccurred())

			serve(app.Command("CreateUser", decodeCreateUserJSON()),
				newPostRequest("/users", "{}"))
			Expect(*called).To(BeTrue())
			Expect(*capturedErr).NotTo(HaveOccurred())
		})

		It("is called with error on failed command dispatch", func() {
			called, capturedErr, hook := trackAfterDispatch()
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", func(_ context.Context, _ command.Command) error {
				return cqrshtmx.ErrDecodeFailed
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp, AfterDispatch: hook})
			Expect(err).NotTo(HaveOccurred())

			serve(app.Command("CreateUser", decodeCreateUserJSON()),
				newPostRequest("/users", "{}"))
			Expect(*called).To(BeTrue())
			Expect(*capturedErr).To(HaveOccurred())
		})
	})

	Describe("Correlation IDs", func() {
		It("propagates X-Correlation-ID through middleware", func() {
			capture := registerCIDCapture()
			app, err := cqrshtmx.New(cqrshtmx.Config{Commands: capture.disp})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Middleware()(app.Command("CreateUser", decodeCreateUserJSON()))
			r := newPostRequest(
				"/users",
				"{}",
				withHeader("X-Correlation-ID", "01HK1549P84T9XF8R94E960633"),
			)
			serve(handler, r)
			Expect(capture.value.String()).To(Equal("01HK1549P84T9XF8R94E960633"))
		})

		It("silently drops non-ULID X-Correlation-ID in middleware", func() {
			capture := registerCIDCapture()
			app, err := cqrshtmx.New(cqrshtmx.Config{Commands: capture.disp})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Middleware()(app.Command("CreateUser", decodeCreateUserJSON()))
			r := newPostRequest("/users", "{}", withHeader("X-Correlation-ID", "not-a-ulid"))
			serve(handler, r)
			Expect(capture.value.IsZero()).To(BeTrue())
		})
	})
})

// cidCapture bundles a dispatcher that records the CorrelationID from the
// command context and the value it captured.
type cidCapture struct {
	disp  *command.Dispatcher
	value cqrshtmx.CorrelationID
}

// registerCIDCapture creates a dispatcher that records the CorrelationID seen
// by the "CreateUser" command handler.
func registerCIDCapture() *cidCapture {
	c := &cidCapture{disp: command.NewDispatcher()}
	_ = c.disp.Register("CreateUser", func(ctx context.Context, _ command.Command) error {
		c.value = cqrshtmx.CorrelationIDFromContext(ctx)
		return nil
	})
	return c
}
