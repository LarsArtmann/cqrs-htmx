package cqrshtmx_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Root Coverage Gaps - Handler Options and Security", func() {
		Describe("DefaultMaxBodySize", func() {
			It("has a reasonable default", func() {
				Expect(cqrshtmx.DefaultMaxBodySize).To(BeNumerically(">", 0))
			})
		})

		Describe("NewRateLimiter monitoring", func() {
			It("reports active keys", func() {
				rl := cqrshtmx.NewRateLimiter(cqrshtmx.RateLimiterConfig{
					Limit:        100,
					Window:       time.Second,
					KeyExtractor: cqrshtmx.KeyExtractorFromRemoteAddr(),
				})
				Expect(rl.ActiveKeys()).To(Equal(0))

				mw := rl.Middleware()
				next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {})
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.RemoteAddr = "1.2.3.4:1234"
				mw(next).ServeHTTP(httptest.NewRecorder(), r)
				Expect(rl.ActiveKeys()).To(Equal(1))
			})
		})

		Describe("Response JSON marshal error", func() {
			It("handles non-encodable values gracefully", func() {
				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				resp := cqrshtmx.NewResponse(w, r)
				result := resp.JSON(make(chan int))
				Expect(result).To(Equal(resp))
			})
		})

		Describe("sanitizeRedirectURL traversal blocks", func() {
			DescribeTable(
				"blocks traversal",
				func(input string, expected bool) {
					w := httptest.NewRecorder()
					r := httptest.NewRequest(http.MethodGet, "/", nil)
					cqrshtmx.NewResponse(w, r).Redirect(input).Apply()
					if expected {
						Expect(w.Code).To(Equal(http.StatusSeeOther))
					} else {
						Expect(w.Code).ToNot(Equal(http.StatusSeeOther))
					}
				},
				Entry("blocks escape above root", "/../../../etc/passwd", false),
				Entry("allows legitimate normalization", "/a/../b/c", true),
				Entry("blocks deep traversal", "/../../../../../etc/passwd", false),
			)
		})

		Describe("WithSuccessStatus for query", func() {
			It("returns custom status for query success without body", func() {
				app := newQueryAppWithResult(func(_ context.Context, _ query.Query) (any, error) {
					return "ok", nil
				})
				r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(`{}`))
				w := serve(app.Query(
					"GetUser",
					decodeGetUserJSONQuery(),
					cqrshtmx.WithSuccessStatus(http.StatusOK),
				), r)
				Expect(w.code()).To(Equal(http.StatusOK))
			})
		})

		Describe("RequireMethod HandlerOption", func() {
			It("rejects wrong HTTP method with 405", func() {
				disp := command.NewDispatcher()
				_ = disp.Register("CreateUser", noOpCommandHandler)
				app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
				Expect(err).NotTo(HaveOccurred())

				handler := app.Command("CreateUser", decodeCreateUserJSON(), cqrshtmx.RequireMethod(http.MethodPost))
				r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(`{}`))
				w := serve(handler, r)
				Expect(w.code()).To(Equal(http.StatusMethodNotAllowed))
			})

			It("allows correct HTTP method", func() {
				disp := command.NewDispatcher()
				_ = disp.Register("CreateUser", noOpCommandHandler)
				app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
				Expect(err).NotTo(HaveOccurred())

				handler := app.Command("CreateUser", decodeCreateUserJSON(), cqrshtmx.RequireMethod(http.MethodPost))
				w := serve(handler, newPostRequest("/users", `{}`))
				Expect(w.code()).To(Equal(http.StatusNoContent))
			})
		})

		Describe("HX-Redirect sanitization", func() {
			It("sanitizes redirect for HTMX requests", func() {
				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
				cqrshtmx.NewResponse(w, r).Redirect("/../../etc/passwd")
				Expect(w.Header().Get("HX-Redirect")).To(BeEmpty())
			})

			It("allows safe redirect for HTMX requests", func() {
				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
				cqrshtmx.NewResponse(w, r).Redirect("/users")
				Expect(w.Header().Get("HX-Redirect")).To(Equal("/users"))
			})
		})

		Describe("Recommended security constants", func() {
			It("provides recommended HSTS value", func() {
				Expect(cqrshtmx.RecommendedHSTS).To(ContainSubstring("max-age="))
			})

			It("provides recommended CSP value", func() {
				Expect(cqrshtmx.RecommendedCSP).To(ContainSubstring("default-src"))
			})

			It("applies recommended HSTS in security middleware", func() {
				handler := cqrshtmx.SecurityHeadersMiddlewareWithConfig(cqrshtmx.SecurityHeadersConfig{
					StrictTransportSecurity: cqrshtmx.RecommendedHSTS,
				})(okHandler())

				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				handler.ServeHTTP(w, r)
				Expect(w.Header().Get("Strict-Transport-Security")).To(Equal(cqrshtmx.RecommendedHSTS))
			})
		})

		Describe("X-Request-ID response header propagation", func() {
			It("sets X-Request-ID in response when generated", func() {
				middleware := cqrshtmx.ContextEnrichmentMiddleware(nil)
				next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
				})

				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				middleware(next).ServeHTTP(w, r)
				Expect(w.Header().Get("X-Request-ID")).NotTo(BeEmpty())
			})

			It("propagates provided X-Request-ID in response", func() {
				middleware := cqrshtmx.ContextEnrichmentMiddleware(nil)
				next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
				})

				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Set("X-Request-ID", "01HK1549P84T9XF8R94E960633")
				middleware(next).ServeHTTP(w, r)
				Expect(w.Header().Get("X-Request-ID")).To(Equal("01HK1549P84T9XF8R94E960633"))
			})
		})

		Describe("ErrMethodNotAllowed mapping", func() {
			It("maps to 405", func() {
				Expect(cqrshtmx.MapError(cqrshtmx.ErrMethodNotAllowed)).To(Equal(http.StatusMethodNotAllowed))
			})
		})

		Describe("enrichUserID logs extractor errors", func() {
			It("continues when extractor returns error", func() {
				disp := command.NewDispatcher()
				_ = disp.Register("CreateUser", noOpCommandHandler)
				app, err := cqrshtmx.New(cqrshtmx.Config{
					Commands:        disp,
					UserIDExtractor: func(_ *http.Request) (id.UserID, error) { return id.UserID{}, errors.New("broken") },
				})
				Expect(err).NotTo(HaveOccurred())

				handler := app.Command("CreateUser", decodeCreateUserJSON())
				w := serve(handler, newPostRequest("/users", `{}`))
				Expect(w.code()).To(Equal(http.StatusNoContent))
			})
		})

		Describe("HealthHandler", func() {
			It("returns 200 when dispatchers are configured", func() {
				app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: command.NewDispatcher()})
				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/health", nil)
				app.HealthHandler().ServeHTTP(w, r)
				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(w.Body.String()).To(ContainSubstring("ok"))
			})
		})

})

// nonStringWriter wraps an http.ResponseWriter without implementing
// io.StringWriter, forcing WriteString to fall back to Write.
type nonStringWriter struct {
	http.ResponseWriter
	recorder *httptest.ResponseRecorder
}

func (n *nonStringWriter) Write(p []byte) (int, error) {
	return n.recorder.Write(p)
}

func (n *nonStringWriter) Header() http.Header {
	return n.recorder.Header()
}

func (n *nonStringWriter) WriteHeader(code int) {
	n.recorder.WriteHeader(code)
}
