package cqrshtmx_test

import (
	"context"
	"net/http"
	"net/http/httptest"

	cqrshtmx "github.com/larsartmann/cqrs-htmx"
	"github.com/larsartmann/go-cqrs-lite/command/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Recovery Middleware", func() {
	Describe("Standalone RecoveryMiddleware", func() {
		It("recovers from panics and writes 500", func() {
			panicHandler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				panic("something went terribly wrong")
			})
			handler := cqrshtmx.RecoveryMiddleware(panicHandler)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusInternalServerError))
			Expect(w.Body.String()).To(ContainSubstring("something went terribly wrong"))
		})

		It("allows normal requests through", func() {
			normalHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			handler := cqrshtmx.RecoveryMiddleware(normalHandler)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusOK))
		})

		It("re-raises http.ErrAbortHandler", func() {
			abortHandler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				panic(http.ErrAbortHandler)
			})
			handler := cqrshtmx.RecoveryMiddleware(abortHandler)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			Expect(func() {
				handler.ServeHTTP(w, r)
			}).To(Panic())
		})
	})

	Describe("App.RecoverHandler", func() {
		It("recovers from panics using the App error handler", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", func(_ context.Context, _ command.Command) error {
				return nil
			})
			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands:      disp,
				LoginRedirect: "/auth/signin",
			})
			Expect(err).NotTo(HaveOccurred())

			panicHandler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				panic("app panic")
			})
			handler := app.RecoverHandler()(panicHandler)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusInternalServerError))
			Expect(w.Body.String()).To(ContainSubstring("app panic"))
		})

		It("uses custom error handler when configured", func() {
			called := false
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", func(_ context.Context, _ command.Command) error {
				return nil
			})
			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands: disp,
				ErrorHandler: func(_ http.ResponseWriter, _ *http.Request, _ error) {
					called = true
				},
			})
			Expect(err).NotTo(HaveOccurred())

			panicHandler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				panic("custom panic")
			})
			handler := app.RecoverHandler()(panicHandler)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)

			Expect(called).To(BeTrue())
		})

		It("re-raises http.ErrAbortHandler", func() {
			app := newCommandApp()
			abortHandler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				panic(http.ErrAbortHandler)
			})
			handler := app.RecoverHandler()(abortHandler)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			Expect(func() {
				handler.ServeHTTP(w, r)
			}).To(Panic())
		})
	})
})
