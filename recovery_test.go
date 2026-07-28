package cqrshtmx_test

import (
	"net/http"
	"net/http/httptest"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
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
			// Panic detail is redacted by default (5xx → family default message)
			// so internal stack/message does not leak to the client.
			Expect(w.Body.String()).NotTo(ContainSubstring("something went terribly wrong"))
			Expect(w.Body.String()).To(ContainSubstring("currently unavailable"))
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
			// Panic detail is redacted by default (5xx → family default message).
			Expect(w.Body.String()).NotTo(ContainSubstring("app panic"))
			Expect(w.Body.String()).To(ContainSubstring("currently unavailable"))
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

		It("echoes request_id when ContextEnrichmentMiddleware ran downstream", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", noOpCommandHandler)
			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands:                 disp,
				IncludeRequestIDInErrors: true,
			})
			Expect(err).NotTo(HaveOccurred())

			panicHandler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				panic("boom")
			})
			// Simulate the documented stack order: Recovery wraps
			// ContextEnrichment wraps the handler. ContextEnrichment generates
			// the RequestID and writes it to the X-Request-ID response header,
			// but Recovery's captured request is the pre-enrichment original.
			handler := app.RecoverHandler()(
				cqrshtmx.ContextEnrichmentMiddleware(nil)(panicHandler),
			)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusInternalServerError))
			ridHeader := w.Header().Get("X-Request-ID")
			Expect(ridHeader).NotTo(BeEmpty())
			// The panic response body echoes the same RequestID.
			Expect(w.Body.String()).To(ContainSubstring("[request_id: " + ridHeader + "]"))
			// Panic detail is still redacted.
			Expect(w.Body.String()).NotTo(ContainSubstring("boom"))
		})

		It("omits request_id when no ContextEnrichmentMiddleware in stack", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", noOpCommandHandler)
			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands:                 disp,
				IncludeRequestIDInErrors: true,
			})
			Expect(err).NotTo(HaveOccurred())

			panicHandler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				panic("boom")
			})
			handler := app.RecoverHandler()(panicHandler)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusInternalServerError))
			Expect(w.Body.String()).NotTo(ContainSubstring("request_id"))
		})
	})
})
