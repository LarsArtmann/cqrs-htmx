package cqrshtmx_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v3"
	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Root Coverage Gaps - Error Mapping", func() {
	Describe("WriteJSON error path", func() {
		It("returns error when encoder fails", func() {
			w := httptest.NewRecorder()
			err := cqrshtmx.WriteJSON(w, http.StatusOK, make(chan int))
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("MapError nil input", func() {
		It("returns 500 for nil error", func() {
			Expect(cqrshtmx.MapError(nil)).To(Equal(http.StatusInternalServerError))
		})
	})

	Describe("MapError unknown family", func() {
		It("returns 503 for Infrastructure errors", func() {
			Expect(cqrshtmx.MapError(
				event.NewInfrastructure("test.infra", "infra failure"),
			)).To(Equal(http.StatusServiceUnavailable))
		})
	})

	Describe("Enforce nil enforcer", func() {
		It("returns ErrEnforcerNil when enforcer is nil", func() {
			err := cqrshtmx.Enforce(nil, "user1", "resource", "read")
			Expect(errors.Is(err, cqrshtmx.ErrEnforcerNil)).To(BeTrue())
		})
	})

	Describe("handleCommandDispatch auth denied", func() {
		It("rejects when authorization fails", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", noOpCommandHandler)
			enforcer := newTestEnforcer()
			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands: disp,
				Enforcer: enforcer,
			})
			Expect(err).NotTo(HaveOccurred())

			r := newPostRequest("/users", `{}`)
			r = r.WithContext(context.Background())
			w := serve(app.Command(
				"CreateUser",
				decodeCreateUserJSON(),
				cqrshtmx.Authorize("users", "create"),
			), r)
			Expect(w.code()).To(Equal(http.StatusUnauthorized))
		})
	})

	Describe("applyQueryResponse with HTMX response option", func() {
		It("applies HTMX headers on query success", func() {
			app := newQueryAppWithResult(testResultQueryHandler())
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(`{}`))
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			w := serve(app.Query(
				"GetUser",
				decodeGetUserJSONQuery(),
				cqrshtmx.Trigger("dataLoaded"),
			), r)
			Expect(w.Header().Get("HX-Trigger")).To(Equal("dataLoaded"))
		})
	})

	Describe("RateLimiter eviction", func() {
		It("evicts oldest entry when MaxKeys exceeded", func() {
			handler := cqrshtmx.RateLimiterMiddleware(cqrshtmx.RateLimiterConfig{
				Limit:        100,
				Window:       time.Second,
				MaxKeys:      2,
				KeyExtractor: cqrshtmx.KeyExtractorFromRemoteAddr(),
			})

			called := 0
			next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				called++
			})

			for i := range 5 {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.RemoteAddr = fmt.Sprintf("192.168.1.%d:1234", i%3)
				w := httptest.NewRecorder()
				handler(next).ServeHTTP(w, r)
			}
			Expect(called).To(Equal(5))
		})

		It("exempts requests with empty key", func() {
			handler := cqrshtmx.RateLimiterMiddleware(cqrshtmx.RateLimiterConfig{
				Limit:        1,
				Window:       time.Second,
				KeyExtractor: func(_ *http.Request) string { return "" },
			})

			called := 0
			next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				called++
			})

			for range 5 {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				w := httptest.NewRecorder()
				handler(next).ServeHTTP(w, r)
			}
			Expect(called).To(Equal(5))
		})
	})
})
