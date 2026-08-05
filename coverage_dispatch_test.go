package cqrshtmx_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Root Coverage Gaps - Dispatch and CSRF", func() {
	Describe("StatusRecorder Hijack non-Hijacker fallback", func() {
		It("returns ErrNotSupported when underlying writer has no Hijacker", func() {
			rec := httptest.NewRecorder()
			sr := cqrshtmx.NewStatusRecorder(rec)
			_, _, err := sr.Hijack()
			Expect(err).To(Equal(http.ErrNotSupported))
		})
	})

	Describe("CSRF sameSite all branches", func() {
		It("maps SameSiteDefaultMode", func() {
			config := cqrshtmx.CSRFConfig{
				SameSite: http.SameSiteDefaultMode,
			}
			mw := cqrshtmx.CSRFMiddleware(config)
			handler := mw(okHandler())
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
		})
	})

	Describe("csrfTokenFromRequest context fallback", func() {
		It("falls back to context token when gorilla has none", func() {
			ctx := cqrshtmx.WithCSRFToken(context.Background(), "ctx-token")
			r := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			token := cqrshtmx.CSRFTokenFromContext(r.Context())
			Expect(token).To(Equal("ctx-token"))
		})
	})

	Describe("Enforce with enforcer error", func() {
		It("wraps error from enforcer", func() {
			enforcer := newFailingEnforcer(errors.New("internal error"))
			err := cqrshtmx.Enforce(enforcer, "user1", "resource", "read")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("casbin enforce failed"))
			Expect(err.Error()).To(ContainSubstring("internal error"))
		})
	})

	Describe("applyQueryResponse render error", func() {
		It("calls error handler when render fails", func() {
			app := newQueryAppWithResult(testResultQueryHandler())
			renderErr := errors.New("render failed")
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(`{}`))
			w := serve(app.Query(
				"GetUser",
				decodeGetUserJSONQuery(),
				cqrshtmx.Render(func(_ http.ResponseWriter, _ *http.Request, _ any) error {
					return renderErr
				}),
			), r)
			Expect(w.code()).To(Equal(http.StatusServiceUnavailable))
		})
	})

	Describe("StatusRecorder Push with actual pusher error", func() {
		It("wraps error from underlying Pusher", func() {
			w := newPusherRecorder(&mockPusher{
				ResponseWriter: httptest.NewRecorder(),
				pushFunc: func(_ string, _ *http.PushOptions) error {
					return errors.New("push failed")
				},
			})
			sr := cqrshtmx.NewStatusRecorder(w)
			err := sr.Push("/target", nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("push failed"))
		})
	})

	Describe("MapError conflict family", func() {
		It("returns 409 for conflict family errors", func() {
			err := fmt.Errorf("wrap: %w", errorfamily.NewConflict("test.conflict", "conflict occurred"))
			status := cqrshtmx.MapError(err)
			Expect(status).To(Equal(http.StatusConflict))
		})
	})

	Describe("MapError transient family", func() {
		It("returns 503 for transient family errors", func() {
			err := fmt.Errorf("wrap: %w", errorfamily.NewTransient("test.transient", "temporary failure"))
			status := cqrshtmx.MapError(err)
			Expect(status).To(Equal(http.StatusServiceUnavailable))
		})
	})

	Describe("MapError ErrRequestTooLarge", func() {
		It("returns 413 for body size exceeded", func() {
			status := cqrshtmx.MapError(cqrshtmx.ErrRequestTooLarge)
			Expect(status).To(Equal(http.StatusRequestEntityTooLarge))
		})
	})

	Describe("IsAuthenticated helper", func() {
		It("returns false when no user ID in context", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			Expect(cqrshtmx.IsAuthenticated(r)).To(BeFalse())
		})

		It("returns true when user ID is in context", func() {
			uid := cqrshtmx.NewUserID()
			ctx := cqrshtmx.WithUserID(context.Background(), uid)
			r := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			Expect(cqrshtmx.IsAuthenticated(r)).To(BeTrue())
		})
	})

	Describe("MustNew convenience function", func() {
		It("panics on invalid config", func() {
			Expect(func() { cqrshtmx.MustNew(cqrshtmx.Config{}) }).To(Panic())
		})

		It("returns app on valid config", func() {
			app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: command.NewDispatcher()})
			Expect(app).NotTo(BeNil())
		})
	})

	Describe("HasCommands and HasQueries", func() {
		It("reports correctly for command-only app", func() {
			app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: command.NewDispatcher()})
			Expect(app.HasCommands()).To(BeTrue())
			Expect(app.HasQueries()).To(BeFalse())
		})

		It("reports correctly for query-only app", func() {
			app := cqrshtmx.MustNew(cqrshtmx.Config{Queries: query.NewDispatcher()})
			Expect(app.HasCommands()).To(BeFalse())
			Expect(app.HasQueries()).To(BeTrue())
		})
	})

	Describe("Empty type validation", func() {
		It("panics on empty command type", func() {
			app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: command.NewDispatcher()})

			Expect(func() { app.Command("") }).To(PanicWith(MatchRegexp("command type must not be empty")))
		})

		It("panics on empty query type", func() {
			app := cqrshtmx.MustNew(cqrshtmx.Config{Queries: query.NewDispatcher()})

			Expect(func() { app.Query("") }).To(PanicWith(MatchRegexp("query type must not be empty")))
		})

		It("panics on empty command type even with query-only app", func() {
			app := cqrshtmx.MustNew(cqrshtmx.Config{Queries: query.NewDispatcher()})

			Expect(func() { app.Command("") }).To(PanicWith(MatchRegexp("command type must not be empty")))
		})

		It("panics on empty query type even with command-only app", func() {
			app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: command.NewDispatcher()})

			Expect(func() { app.Query("") }).To(PanicWith(MatchRegexp("query type must not be empty")))
		})
	})
})
