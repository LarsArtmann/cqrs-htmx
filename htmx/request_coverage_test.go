package htmx_test

import (
	"context"
	"net/http"
	"net/http/httptest"

	"github.com/larsartmann/cqrs-htmx/v4/htmx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Request parsing coverage", func() {
	It("parses every HTMX header into the Request", func() {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set(htmx.HeaderRequest, htmx.HeaderTrue)
		r.Header.Set(htmx.HeaderBoosted, htmx.HeaderTrue)
		r.Header.Set(htmx.HeaderHistoryRestore, htmx.HeaderTrue)
		r.Header.Set(htmx.HeaderTarget, "#main")
		r.Header.Set(htmx.HeaderTrigger, "btn-id")
		r.Header.Set(htmx.HeaderTriggerName, "save")
		r.Header.Set(htmx.HeaderPrompt, "sure?")
		r.Header.Set(htmx.HeaderCurrentURL, "/elsewhere")

		h := htmx.ParseRequest(r)
		Expect(h.IsHTMX).To(BeTrue())
		Expect(h.IsBoosted).To(BeTrue())
		Expect(h.IsHistoryRestore).To(BeTrue())
		Expect(h.Target).To(Equal("#main"))
		Expect(h.TriggerID).To(Equal("btn-id"))
		Expect(h.TriggerName).To(Equal("save"))
		Expect(h.Prompt).To(Equal("sure?"))
		Expect(h.CurrentURL).To(Equal("/elsewhere"))

		// A history restoration renders the full page even over HTMX.
		Expect(h.RenderPartial()).To(BeFalse())
	})

	It("renders a partial for plain HTMX requests", func() {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set(htmx.HeaderRequest, htmx.HeaderTrue)

		h := htmx.ParseRequest(r)
		Expect(h.RenderPartial()).To(BeTrue())
	})

	It("round-trips the parsed Request through the context", func() {
		h := htmx.ParseRequest(httest())
		ctx := htmx.WithRequest(context.Background(), h)
		Expect(htmx.FromContext(ctx)).To(Equal(h))
	})

	It("prefers the context Request for the package-level accessors", func() {
		h := htmx.ParseRequest(httest())
		r := httptest.NewRequest(http.MethodPost, "/x", nil).WithContext(
			htmx.WithRequest(context.Background(), h),
		)
		// The raw headers are absent; only the context Request can answer.
		Expect(htmx.RenderPartial(r)).To(BeTrue())
		Expect(htmx.TriggerName(r)).To(Equal("save"))
	})

	It("falls back to raw headers when no context Request exists", func() {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set(htmx.HeaderRequest, htmx.HeaderTrue)
		Expect(htmx.RenderPartial(r)).To(BeTrue())

		r.Header.Set(htmx.HeaderHistoryRestore, htmx.HeaderTrue)
		Expect(htmx.RenderPartial(r)).To(BeFalse())

		r.Header.Set(htmx.HeaderTriggerName, "fallback")
		Expect(htmx.TriggerName(r)).To(Equal("fallback"))
	})
})

func httest() *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set(htmx.HeaderRequest, htmx.HeaderTrue)
	r.Header.Set(htmx.HeaderTriggerName, "save")
	return r
}
