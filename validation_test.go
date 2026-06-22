package cqrshtmx_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"

	cqrshmx "github.com/larsartmann/cqrs-htmx/v2"
	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Validation HandlerOption", func() {
	Describe("ValidateCommand", func() {
		It("allows valid commands through", func() {
			var dispatched bool
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", trackingCommandHandler(&dispatched))
			app, err := cqrshmx.New(cqrshmx.Config{Commands: disp})
			Expect(err).NotTo(HaveOccurred())

			serve(app.Command(
				"CreateUser",
				decodeCreateUserJSON(),
				cqrshmx.ValidateCommand(func(_ command.Command) error { return nil }),
			), newPostRequest("/users", `{}`))
			Expect(dispatched).To(BeTrue())
		})

		It("rejects commands that fail validation", func() {
			app := newCommandApp()
			w := serve(app.Command(
				"CreateUser",
				decodeCreateUserJSON(),
				cqrshmx.ValidateCommand(func(_ command.Command) error {
					return errors.New("email is required")
				}),
			), newPostRequest("/users", `{}`))
			Expect(w.code()).To(Equal(http.StatusBadRequest))
			Expect(w.Body.String()).To(ContainSubstring("email is required"))
		})

		It("no-op when decoder is not set", func() {
			var decoded bool
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", func(_ context.Context, _ command.Command) error {
				decoded = true
				return nil
			})
			app, err := cqrshmx.New(cqrshmx.Config{Commands: disp})
			Expect(err).NotTo(HaveOccurred())

			w := serve(app.Command(
				"CreateUser",
				cqrshmx.ValidateCommand(func(_ command.Command) error {
					return errors.New("should not run")
				}),
			), newPostRequest("/users", ""))
			Expect(w.code()).NotTo(Equal(http.StatusBadRequest))
			Expect(decoded).To(BeFalse())
		})
	})

	Describe("ValidateQuery", func() {
		It("allows valid queries through", func() {
			app := newQueryAppWithResult(testResultQueryHandler())

			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(`{}`))
			w := serve(app.Query(
				"GetUser",
				decodeGetUserJSONQuery(),
				cqrshmx.ValidateQuery(func(_ query.Query) error { return nil }),
				cqrshmx.Render(encodeJSONResult),
			), r)
			Expect(w.code()).To(Equal(http.StatusOK))
		})

		It("rejects queries that fail validation", func() {
			app := newQueryAppWithResult(testResultQueryHandler())

			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(`{}`))
			w := serve(app.Query(
				"GetUser",
				decodeGetUserJSONQuery(),
				cqrshmx.ValidateQuery(func(_ query.Query) error {
					return errors.New("page must be positive")
				}),
			), r)
			Expect(w.code()).To(Equal(http.StatusBadRequest))
			Expect(w.Body.String()).To(ContainSubstring("page must be positive"))
		})

		It("wraps decoding errors before validation runs", func() {
			app := newQueryAppWithResult(testResultQueryHandler())

			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(`{}`))
			w := serve(app.Query(
				"GetUser",
				cqrshmx.DecodeJSONQuery(func(_ testGetUserQuery) (query.Query, error) {
					return nil, errors.New("decode failed")
				}),
				cqrshmx.ValidateQuery(func(_ query.Query) error {
					return errors.New("should not run")
				}),
			), r)
			Expect(w.Body.String()).To(ContainSubstring("decode"))
			Expect(w.Body.String()).NotTo(ContainSubstring("should not run"))
		})
	})
})
