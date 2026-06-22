package cqrshtmx_test

import (
	"context"
	"net/http"
	"net/http/httptest"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CSRF Defaults", func() {
	It("uses default field name when empty", func() {
		cfg := cqrshtmx.CSRFConfig{}
		mw := cqrshtmx.CSRFMiddleware(cfg)
		handler := mw(okHandler())
		w := httptest.NewRecorder()
		r := httptest.NewRequestWithContext(
			context.Background(), http.MethodGet, "/", nil,
		)
		handler.ServeHTTP(w, r)
		Expect(w.Code).To(Equal(http.StatusOK))
	})

	It("uses default SameSite when zero", func() {
		cfg := cqrshtmx.CSRFConfig{
			SameSite: 0,
		}
		Expect(cfg.Validate()).To(Succeed())
	})

	It("reads token from context when nosurf context has none", func() {
		token := cqrshtmx.CSRFTokenFromContext(
			cqrshtmx.WithCSRFToken(context.Background(), "fallback-token"),
		)
		Expect(token).To(Equal("fallback-token"))
	})

	It("returns empty token from empty context", func() {
		token := cqrshtmx.CSRFTokenFromContext(context.Background())
		Expect(token).To(BeEmpty())
	})
})
