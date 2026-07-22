package cqrshtmx_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// typedEchoCommand is a self-contained command type used to exercise the
// type-safe CommandTyped path. It implements command.Command directly so it
// can be decoded from JSON without relying on an embedded BasicCommand.
type typedEchoCommand struct {
	Message string `json:"message"`
}

func (c *typedEchoCommand) Type() command.Type          { return "Echo" }
func (c *typedEchoCommand) AggregateID() id.AggregateID { return id.NewAggregateID() }
func (c *typedEchoCommand) ID() id.CommandID            { return id.NewCommandID() }

// typedSumQuery is a self-contained query type used to exercise the
// type-safe QueryTyped path.
type typedSumQuery struct {
	A int `json:"a"`
	B int `json:"b"`
}

func (q *typedSumQuery) Type() query.Type { return "Sum" }

var _ = Describe("Integration: Typed CQRS handlers", func() {
	Describe("CommandTyped", func() {
		It("dispatches a typed command from JSON", func() {
			var received string

			disp := command.NewDispatcher()
			err := command.RegisterTyped(
				disp, "Echo",
				func(_ context.Context, cmd *typedEchoCommand) error {
					received = cmd.Message

					return nil
				},
			)
			Expect(err).NotTo(HaveOccurred())

			app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := cqrshtmx.CommandTyped[*typedEchoCommand](
				app, "Echo",
				cqrshtmx.DecodeJSONTyped[*typedEchoCommand](),
				cqrshtmx.WithSuccessStatus(http.StatusAccepted),
			)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(
				http.MethodPost, "/echo",
				strings.NewReader(`{"message":"hello"}`),
			)
			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusAccepted))
			Expect(received).To(Equal("hello"))
		})

		It("returns 400 when the decoder returns a different command type", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("Echo", noOpCommandHandler)

			app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
			Expect(err).NotTo(HaveOccurred())

			// The decoder returns testCreateUserCmd, but the handler expects
			// *typedEchoCommand. The type assertion should fail gracefully.
			handler := cqrshtmx.CommandTyped[*typedEchoCommand](
				app, "Echo",
				cqrshtmx.DecodeJSON(func(_ struct{}) (command.Command, error) {
					return &testCreateUserCmd{}, nil
				}),
			)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(
				http.MethodPost, "/echo",
				strings.NewReader(`{}`),
			)
			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})
	})

	Describe("QueryTyped", func() {
		It("dispatches a typed query and renders a typed result", func() {
			disp := query.NewDispatcher()
			err := query.RegisterTyped(
				disp, "Sum",
				func(_ context.Context, q *typedSumQuery) (int, error) {
					return q.A + q.B, nil
				},
			)
			Expect(err).NotTo(HaveOccurred())

			app, err := cqrshtmx.New(cqrshtmx.Config{Queries: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := cqrshtmx.QueryTyped[*typedSumQuery, int](
				app, "Sum",
				cqrshtmx.DecodeJSONQueryTyped[*typedSumQuery](),
				cqrshtmx.RenderJSON[int](),
			)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(
				http.MethodPost, "/sum",
				strings.NewReader(`{"a":3,"b":4}`),
			)
			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(ContainSubstring("7"))
		})

		It("returns 400 when the decoder returns a different query type", func() {
			disp := query.NewDispatcher()
			_ = disp.Register("Sum", func(_ context.Context, _ query.Query) (any, error) {
				return 0, nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{Queries: disp})
			Expect(err).NotTo(HaveOccurred())

			// The decoder returns testGetUserQuery, but the handler expects
			// *typedSumQuery. The type assertion should fail gracefully.
			handler := cqrshtmx.QueryTyped[*typedSumQuery, int](
				app, "Sum",
				cqrshtmx.DecodeJSONQuery(func(_ struct{}) (query.Query, error) {
					return &testGetUserQuery{}, nil
				}),
				cqrshtmx.RenderJSON[int](),
			)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(
				http.MethodPost, "/sum",
				strings.NewReader(`{}`),
			)
			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})
	})

	Describe("CommandTyped with form decoder", func() {
		It("dispatches a typed command from form data", func() {
			var received string

			disp := command.NewDispatcher()
			err := command.RegisterTyped(
				disp, "Echo",
				func(_ context.Context, cmd *typedEchoCommand) error {
					received = cmd.Message

					return nil
				},
			)
			Expect(err).NotTo(HaveOccurred())

			app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := cqrshtmx.CommandTyped[*typedEchoCommand](
				app, "Echo",
				cqrshtmx.DecodeFormTyped[*typedEchoCommand](),
				cqrshtmx.WithSuccessStatus(http.StatusAccepted),
			)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(
				http.MethodPost, "/echo",
				strings.NewReader("message=hello"),
			)
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusAccepted))
			Expect(received).To(Equal("hello"))
		})
	})

	Describe("QueryTyped with form decoder", func() {
		It("dispatches a typed query from form data and renders a typed result", func() {
			disp := query.NewDispatcher()
			err := query.RegisterTyped(
				disp, "Sum",
				func(_ context.Context, q *typedSumQuery) (int, error) {
					return q.A + q.B, nil
				},
			)
			Expect(err).NotTo(HaveOccurred())

			app, err := cqrshtmx.New(cqrshtmx.Config{Queries: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := cqrshtmx.QueryTyped[*typedSumQuery, int](
				app, "Sum",
				cqrshtmx.DecodeFormQueryTyped[*typedSumQuery](),
				cqrshtmx.RenderJSON[int](),
			)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(
				http.MethodPost, "/sum",
				strings.NewReader("a=3&b=4"),
			)
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(ContainSubstring("7"))
		})
	})

	Describe("CommandTyped error paths", func() {
		It("returns 400 when JSON body is malformed", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("Echo", noOpCommandHandler)

			app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := cqrshtmx.CommandTyped[*typedEchoCommand](
				app, "Echo",
				cqrshtmx.DecodeJSONTyped[*typedEchoCommand](),
			)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(
				http.MethodPost, "/echo",
				strings.NewReader(`{not json`),
			)
			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusBadRequest))
			Expect(w.Body.String()).To(ContainSubstring("decode"))
		})

		It("dispatches with nil pointer from empty JSON body", func() {
			var dispatched bool

			disp := command.NewDispatcher()
			err := command.RegisterTyped(
				disp, "Echo",
				func(_ context.Context, cmd *typedEchoCommand) error {
					dispatched = true

					return nil
				},
			)
			Expect(err).NotTo(HaveOccurred())

			app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := cqrshtmx.CommandTyped[*typedEchoCommand](
				app, "Echo",
				cqrshtmx.DecodeJSONTyped[*typedEchoCommand](),
			)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(
				http.MethodPost, "/echo",
				strings.NewReader(""),
			)
			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(dispatched).To(BeTrue())
		})

		It("dispatches with nil pointer from empty form body", func() {
			var dispatched bool

			disp := command.NewDispatcher()
			err := command.RegisterTyped(
				disp, "Echo",
				func(_ context.Context, cmd *typedEchoCommand) error {
					dispatched = true

					return nil
				},
			)
			Expect(err).NotTo(HaveOccurred())

			app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := cqrshtmx.CommandTyped[*typedEchoCommand](
				app, "Echo",
				cqrshtmx.DecodeFormTyped[*typedEchoCommand](),
			)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(
				http.MethodPost, "/echo",
				strings.NewReader(""),
			)
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(dispatched).To(BeTrue())
		})
	})

	Describe("QueryTyped error paths", func() {
		It("returns 400 when JSON body is malformed", func() {
			disp := query.NewDispatcher()
			_ = disp.Register("Sum", func(_ context.Context, _ query.Query) (any, error) {
				return 0, nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{Queries: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := cqrshtmx.QueryTyped[*typedSumQuery, int](
				app, "Sum",
				cqrshtmx.DecodeJSONQueryTyped[*typedSumQuery](),
				cqrshtmx.RenderJSON[int](),
			)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(
				http.MethodPost, "/sum",
				strings.NewReader(`{not json`),
			)
			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusBadRequest))
			Expect(w.Body.String()).To(ContainSubstring("decode"))
		})

		It("dispatches with zero values from explicit JSON body", func() {
			disp := query.NewDispatcher()
			err := query.RegisterTyped(
				disp, "Sum",
				func(_ context.Context, q *typedSumQuery) (int, error) {
					return q.A + q.B, nil
				},
			)
			Expect(err).NotTo(HaveOccurred())

			app, err := cqrshtmx.New(cqrshtmx.Config{Queries: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := cqrshtmx.QueryTyped[*typedSumQuery, int](
				app, "Sum",
				cqrshtmx.DecodeJSONQueryTyped[*typedSumQuery](),
				cqrshtmx.RenderJSON[int](),
			)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(
				http.MethodPost, "/sum",
				strings.NewReader(`{"a":0,"b":0}`),
			)
			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(ContainSubstring("0"))
		})

		It("returns 400 when form body has wrong type for int field", func() {
			disp := query.NewDispatcher()
			_ = disp.Register("Sum", func(_ context.Context, _ query.Query) (any, error) {
				return 0, nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{Queries: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := cqrshtmx.QueryTyped[*typedSumQuery, int](
				app, "Sum",
				cqrshtmx.DecodeFormQueryTyped[*typedSumQuery](),
				cqrshtmx.RenderJSON[int](),
			)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(
				http.MethodPost, "/sum",
				strings.NewReader("a=not-a-number&b=4"),
			)
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusBadRequest))
			Expect(w.Body.String()).To(ContainSubstring("decode"))
		})
	})
})
