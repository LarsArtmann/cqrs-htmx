package cqrshtmx_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"

	cqrshmx "github.com/larsartmann/cqrs-htmx"
	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/query"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Validation HandlerOption", func() {
	Describe("ValidateCommand", func() {
		It("allows valid commands through", func() {
			disp := command.NewDispatcher()
			dispatched := false
			_ = disp.Register("CreateUser", func(_ context.Context, _ command.Command) error {
				dispatched = true
				return nil
			})

			app, err := cqrshmx.New(cqrshmx.Config{Commands: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Command("CreateUser",
				decodeCreateUserJSON(),
				cqrshmx.ValidateCommand(func(_ command.Command) error {
					return nil
				}),
			)

			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{}`))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			Expect(dispatched).To(BeTrue())
		})

		It("rejects commands that fail validation", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", noOpCommandHandler)

			app, err := cqrshmx.New(cqrshmx.Config{Commands: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Command("CreateUser",
				decodeCreateUserJSON(),
				cqrshmx.ValidateCommand(func(_ command.Command) error {
					return errors.New("email is required")
				}),
			)

			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{}`))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusBadRequest))
			Expect(w.Body.String()).To(ContainSubstring("email is required"))
		})

		It("no-op when decoder is not set", func() {
			disp := command.NewDispatcher()
			var decoded bool
			_ = disp.Register("CreateUser", func(_ context.Context, _ command.Command) error {
				decoded = true
				return nil
			})

			app, err := cqrshmx.New(cqrshmx.Config{Commands: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Command("CreateUser",
				// No decoder set — ValidateCommand is a no-op
				cqrshmx.ValidateCommand(func(_ command.Command) error {
					return errors.New("should not run")
				}),
			)

			r := httptest.NewRequest(http.MethodPost, "/users", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			// Should fail with decoder missing, not validation
			Expect(w.Code).NotTo(Equal(http.StatusBadRequest))
			Expect(decoded).To(BeFalse())
		})
	})

	Describe("ValidateQuery", func() {
		It("allows valid queries through", func() {
			disp := query.NewDispatcher()
			_ = disp.Register("GetUser", func(_ context.Context, _ query.Query) (any, error) {
				return testQueryResult, nil
			})

			app, err := cqrshmx.New(cqrshmx.Config{Queries: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Query("GetUser",
				decodeGetUserJSONQuery(),
				cqrshmx.ValidateQuery(func(_ query.Query) error {
					return nil
				}),
				cqrshmx.Render(encodeJSONResult),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(body))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusOK))
		})

		It("rejects queries that fail validation", func() {
			disp := query.NewDispatcher()
			_ = disp.Register("GetUser", func(_ context.Context, _ query.Query) (any, error) {
				return testQueryResult, nil
			})

			app, err := cqrshmx.New(cqrshmx.Config{Queries: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Query("GetUser",
				decodeGetUserJSONQuery(),
				cqrshmx.ValidateQuery(func(_ query.Query) error {
					return errors.New("page must be positive")
				}),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(body))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusBadRequest))
			Expect(w.Body.String()).To(ContainSubstring("page must be positive"))
		})

		It("wraps decoding errors before validation runs", func() {
			disp := query.NewDispatcher()
			_ = disp.Register("GetUser", func(_ context.Context, _ query.Query) (any, error) {
				return testQueryResult, nil
			})

			app, err := cqrshmx.New(cqrshmx.Config{Queries: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Query("GetUser",
				cqrshmx.DecodeJSONQuery(func(_ testGetUserQuery) (query.Query, error) {
					return nil, errors.New("decode failed")
				}),
				cqrshmx.ValidateQuery(func(_ query.Query) error {
					return errors.New("should not run")
				}),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(body))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			// Should show decode error, not validation error
			Expect(w.Body.String()).To(ContainSubstring("decode"))
			Expect(w.Body.String()).NotTo(ContainSubstring("should not run"))
		})
	})
})
