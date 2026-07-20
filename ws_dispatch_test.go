package cqrshtmx_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type wsTestKey string

func wsCreateUserDecoder() cqrshtmx.WSCommandDecoder {
	return cqrshtmx.DecodeWSJSON(testCreateUserCommand)
}

func wsNoOpCreateUserDecoder() cqrshtmx.WSCommandDecoder {
	return cqrshtmx.DecodeWSJSON(func(_ testCreateUserRequest) (command.Command, error) {
		return &testCreateUserCmd{aggID: id.NewAggregateID(), cmdID: id.NewCommandID(), email: "", name: ""}, nil
	})
}

func wsGetUserDecoder() cqrshtmx.WSQueryDecoder {
	return cqrshtmx.DecodeWSJSONQuery(func(_ testGetUserQuery) (query.Query, error) {
		return &testGetUserQuery{}, nil
	})
}

var _ = Describe("WS Dispatch", func() {
	Describe("DispatchWSCommand", func() {
		It("dispatches a command from raw WS bytes", func() {
			var handlerCalled bool

			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", func(_ context.Context, _ command.Command) error {
				handlerCalled = true

				return nil
			})

			app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: disp})
			err := app.DispatchWSCommand(nil, "CreateUser", wsCreateUserDecoder(),
				[]byte(`{"email":"test@example.com","name":"Alice"}`))

			Expect(err).NotTo(HaveOccurred())
			Expect(handlerCalled).To(BeTrue())
		})

		It("returns error when commands dispatcher is nil", func() {
			app := cqrshtmx.MustNew(cqrshtmx.Config{Queries: query.NewDispatcher()})
			err := app.DispatchWSCommand(nil, "CreateUser", wsCreateUserDecoder(), []byte(`{}`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("command dispatcher is required"))
		})

		It("returns error when decoder fails on invalid JSON", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", noOpCommandHandler)
			app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: disp})

			err := app.DispatchWSCommand(nil, "CreateUser", wsCreateUserDecoder(),
				[]byte(`{bad json`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("decode"))
		})

		It("returns error when decoder returns nil command", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", noOpCommandHandler)
			app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: disp})

			nilDecoder := func([]byte) (command.Command, error) { return nil, nil }
			err := app.DispatchWSCommand(nil, "CreateUser", nilDecoder, []byte(`{}`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("decoder is required"))
		})

		It("returns error when dispatch fails", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", erroringCommandHandler("database connection failed"))
			app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: disp})

			err := app.DispatchWSCommand(nil, "CreateUser", wsCreateUserDecoder(), []byte(`{}`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("database connection failed"))
		})

		It("calls afterDispatch hook with nil error on success", func() {
			var (
				hookCalled bool
				hookErr    error
			)

			app := newCommandAppWithConfig(
				noOpCommandHandler,
				cqrshtmx.Config{
					AfterDispatch: trackingAfterDispatchHook(&hookCalled, &hookErr),
				},
			)

			_ = app.DispatchWSCommand(nil, "CreateUser", wsCreateUserDecoder(), []byte(`{}`))

			Expect(hookCalled).To(BeTrue())
			Expect(hookErr).NotTo(HaveOccurred())
		})

		It("calls afterDispatch hook with error on failure", func() {
			var (
				hookCalled bool
				hookErr    error
			)

			app := newCommandAppWithConfig(
				erroringCommandHandler("fail"),
				cqrshtmx.Config{
					AfterDispatch: trackingAfterDispatchHook(&hookCalled, &hookErr),
				},
			)

			_ = app.DispatchWSCommand(nil, "CreateUser", wsCreateUserDecoder(), []byte(`{}`))

			Expect(hookCalled).To(BeTrue())
			Expect(hookErr).To(HaveOccurred())
		})

		It("applies beforeDispatch hook when request is provided", func() {
			var capturedCtx context.Context

			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", func(ctx context.Context, _ command.Command) error {
				capturedCtx = ctx //nolint:fatcontext // intentional capture into outer test var for assertion

				return nil
			})

			app, _ := cqrshtmx.New(cqrshtmx.Config{
				Commands: disp,
				BeforeDispatch: func(ctx context.Context, _ *http.Request) context.Context {
					return context.WithValue(ctx, wsTestKey("hook"), "applied")
				},
			})

			req := httptest.NewRequest(http.MethodPost, "/ws", nil)
			_ = app.DispatchWSCommand(req, "CreateUser", wsCreateUserDecoder(), []byte(`{}`))

			Expect(capturedCtx.Value(wsTestKey("hook"))).To(Equal("applied"))
		})

		It("panics on empty command type", func() {
			app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: command.NewDispatcher()})

			Expect(func() {
				_ = app.DispatchWSCommand(nil, "", wsCreateUserDecoder(), nil)
			}).To(Panic())
		})
	})

	Describe("DispatchWSQuery", func() {
		It("dispatches a query and returns the result", func() {
			disp := query.NewDispatcher()
			_ = disp.Register("GetUser", func(_ context.Context, _ query.Query) (any, error) {
				return map[string]string{"name": "Alice"}, nil
			})

			app := cqrshtmx.MustNew(cqrshtmx.Config{Queries: disp})
			result, err := app.DispatchWSQuery(nil, "GetUser", wsGetUserDecoder(), []byte(`{}`))

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(map[string]string{"name": "Alice"}))
		})

		It("returns error when queries dispatcher is nil", func() {
			app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: command.NewDispatcher()})
			_, err := app.DispatchWSQuery(nil, "GetUser", wsGetUserDecoder(), []byte(`{}`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("query dispatcher is required"))
		})

		It("returns error when decoder fails", func() {
			disp := query.NewDispatcher()
			_ = disp.Register("GetUser", func(_ context.Context, _ query.Query) (any, error) {
				return nil, nil
			})
			app := cqrshtmx.MustNew(cqrshtmx.Config{Queries: disp})

			_, err := app.DispatchWSQuery(nil, "GetUser", wsGetUserDecoder(), []byte(`{bad`))
			Expect(err).To(HaveOccurred())
		})

		It("returns error when dispatch fails", func() {
			disp := query.NewDispatcher()
			_ = disp.Register("GetUser", func(_ context.Context, _ query.Query) (any, error) {
				return nil, errors.New("query timeout")
			})
			app := cqrshtmx.MustNew(cqrshtmx.Config{Queries: disp})

			_, err := app.DispatchWSQuery(nil, "GetUser", wsGetUserDecoder(), []byte(`{}`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("query timeout"))
		})

		It("calls afterDispatch hook on success", func() {
			var (
				hookCalled bool
				hookErr    error
			)

			app := newQueryAppWithConfig(
				func(_ context.Context, _ query.Query) (any, error) {
					return "result", nil
				},
				cqrshtmx.Config{
					AfterDispatch: trackingAfterDispatchHook(&hookCalled, &hookErr),
				},
			)

			_, _ = app.DispatchWSQuery(nil, "GetUser", wsGetUserDecoder(), []byte(`{}`))

			Expect(hookCalled).To(BeTrue())
		})

		It("panics on empty query type", func() {
			app := cqrshtmx.MustNew(cqrshtmx.Config{Queries: query.NewDispatcher()})

			Expect(func() {
				_, _ = app.DispatchWSQuery(nil, "", wsGetUserDecoder(), nil)
			}).To(Panic())
		})
	})

	Describe("DecodeWSJSON", func() {
		It("decodes valid JSON into a command", func() {
			decoder := cqrshtmx.DecodeWSJSON(testCreateUserCommand)

			cmd, err := decoder([]byte(`{"email":"a@b.com","name":"Bob"}`))
			Expect(err).NotTo(HaveOccurred())

			typed, ok := cmd.(*testCreateUserCmd)
			Expect(ok).To(BeTrue())
			Expect(typed.email).To(Equal("a@b.com"))
			Expect(typed.name).To(Equal("Bob"))
		})

		It("returns error on invalid JSON", func() {
			_, err := wsNoOpCreateUserDecoder()([]byte(`{not json`))
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("DecodeWSJSONQuery", func() {
		It("decodes valid JSON into a query", func() {
			decoder := cqrshtmx.DecodeWSJSONQuery(func(_ testGetUserQuery) (query.Query, error) {
				return &testGetUserQuery{}, nil
			})

			qry, err := decoder([]byte(`{}`))
			Expect(err).NotTo(HaveOccurred())
			Expect(qry.Type()).To(Equal(query.Type("GetUser")))
		})

		It("returns error on invalid JSON", func() {
			decoder := cqrshtmx.DecodeWSJSONQuery(func(_ testGetUserQuery) (query.Query, error) {
				return &testGetUserQuery{}, nil
			})

			_, err := decoder([]byte(`{bad`))
			Expect(err).To(HaveOccurred())
		})
	})
})
