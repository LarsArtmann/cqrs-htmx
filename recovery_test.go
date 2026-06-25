package cqrshtmx_test

import (
	"net/http"
	"net/http/httptest"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v3"
	"github.com/larsartmann/go-cqrs-lite/command/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Recovery Middleware", func() {
	// httpAbortHandler panics with http.ErrAbortHandler — Go's sentinel for
	// "stop processing this connection" (e.g. server is shutting down).
	// RecoveryMiddleware must re-raise this panic instead of swallowing it
	// as a generic 500. Both the standalone and App-level recovery paths
	// share this behavior, so the handler is built once.
	//nolint:ginkgolinter // shared across all It blocks in this Describe
	httpAbortHandler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic(http.ErrAbortHandler)
	})

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
			handler := cqrshtmx.RecoveryMiddleware(okHandler())

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusOK))
		})

		It("re-raises http.ErrAbortHandler", func() {
			handler := cqrshtmx.RecoveryMiddleware(httpAbortHandler)

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
			_ = disp.Register("CreateUser", noOpCommandHandler)
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
			_ = disp.Register("CreateUser", noOpCommandHandler)
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
			handler := app.RecoverHandler()(httpAbortHandler)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			Expect(func() {
				handler.ServeHTTP(w, r)
			}).To(Panic())
		})
	})
})
