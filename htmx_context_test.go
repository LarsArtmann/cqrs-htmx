package cqrshtmx_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HTMX Request Context", func() {
	Describe("HTMXMiddleware", func() {
		It("parses all HTMX headers and stores in context", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("Hx-Request", cqrshtmx.HeaderTrue)
			r.Header.Set("Hx-Boosted", cqrshtmx.HeaderTrue)
			r.Header.Set("Hx-Target", "main")
			r.Header.Set("Hx-Trigger", "btn")
			r.Header.Set("Hx-Trigger-Name", "action")
			r.Header.Set("Hx-Prompt", "yes")
			r.Header.Set("Hx-Current-Url", "https://example.com/page")

			called := false
			handler := cqrshtmx.HTMXMiddleware(
				http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
					called = true
					h := cqrshtmx.HTMXFromContext(req.Context())
					Expect(h).NotTo(BeNil())
					Expect(h.IsHTMX).To(BeTrue())
					Expect(h.IsBoosted).To(BeTrue())
					Expect(h.IsHistoryRestore).To(BeFalse())
					Expect(h.Target).To(Equal("main"))
					Expect(h.TriggerID).To(Equal("btn"))
					Expect(h.TriggerName).To(Equal("action"))
					Expect(h.Prompt).To(Equal("yes"))
					Expect(h.CurrentURL).To(Equal("https://example.com/page"))
				}),
			)

			handler.ServeHTTP(httptest.NewRecorder(), r)
			Expect(called).To(BeTrue())
		})

		It("parses history restore request", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("Hx-Request", cqrshtmx.HeaderTrue)
			r.Header.Set("Hx-History-Restore-Request", cqrshtmx.HeaderTrue)

			handler := cqrshtmx.HTMXMiddleware(
				http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
					h := cqrshtmx.HTMXFromContext(req.Context())
					Expect(h.IsHistoryRestore).To(BeTrue())
				}),
			)

			handler.ServeHTTP(httptest.NewRecorder(), r)
		})
	})

	Describe("RenderPartial", func() {
		It("returns true for HTMX request without history restore", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("Hx-Request", cqrshtmx.HeaderTrue)
			Expect(cqrshtmx.RenderPartial(r)).To(BeTrue())
		})

		It("returns false for HTMX request with history restore", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("Hx-Request", cqrshtmx.HeaderTrue)
			r.Header.Set("Hx-History-Restore-Request", cqrshtmx.HeaderTrue)
			Expect(cqrshtmx.RenderPartial(r)).To(BeFalse())
		})

		It("returns false for non-HTMX request", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			Expect(cqrshtmx.RenderPartial(r)).To(BeFalse())
		})

		It("uses context when HTMXMiddleware was applied", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("Hx-Request", cqrshtmx.HeaderTrue)

			handler := cqrshtmx.HTMXMiddleware(
				http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
					Expect(cqrshtmx.RenderPartial(req)).To(BeTrue())
					h := cqrshtmx.HTMXFromContext(req.Context())
					Expect(h.RenderPartial()).To(BeTrue())
				}),
			)

			handler.ServeHTTP(httptest.NewRecorder(), r)
		})
	})

	Describe("HTMXTriggerName", func() {
		It("returns the trigger name", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("Hx-Trigger-Name", "action-btn")
			Expect(cqrshtmx.HTMXTriggerName(r)).To(Equal("action-btn"))
		})

		It("returns empty when not set", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			Expect(cqrshtmx.HTMXTriggerName(r)).To(BeEmpty())
		})
	})

	Describe("accessors use context when available", func() {
		It("reads from context when middleware was applied", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("Hx-Request", cqrshtmx.HeaderTrue)
			r.Header.Set("Hx-Target", "main")
			r.Header.Set("Hx-Trigger", "btn")
			r.Header.Set("Hx-Boosted", cqrshtmx.HeaderTrue)
			r.Header.Set("Hx-History-Restore-Request", cqrshtmx.HeaderTrue)
			r.Header.Set("Hx-Trigger-Name", "action-btn")
			r.Header.Set("Hx-Prompt", "yes")
			r.Header.Set("Hx-Current-Url", "https://example.com/page")

			handler := cqrshtmx.HTMXMiddleware(
				http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
					Expect(cqrshtmx.IsHTMXRequest(req)).To(BeTrue())
					Expect(cqrshtmx.IsBoosted(req)).To(BeTrue())
					Expect(cqrshtmx.HTMXTarget(req)).To(Equal("main"))
					Expect(cqrshtmx.HTMXTrigger(req)).To(Equal("btn"))
					Expect(cqrshtmx.IsHistoryRestore(req)).To(BeTrue())
					Expect(cqrshtmx.HTMXTriggerName(req)).To(Equal("action-btn"))
					Expect(cqrshtmx.HTMXPrompt(req)).To(Equal("yes"))
					Expect(cqrshtmx.HTMXCurrentURL(req)).To(Equal("https://example.com/page"))
				}),
			)

			handler.ServeHTTP(httptest.NewRecorder(), r)
		})
	})

	Describe("HTMXFromContext returns nil without middleware", func() {
		It("returns nil when no HTMXMiddleware was applied", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			Expect(cqrshtmx.HTMXFromContext(r.Context())).To(BeNil())
		})
	})

	DescribeTable(
		"accessor fallback without middleware",
		func(header, value string, accessor func(*http.Request) string, expected string) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set(header, value)
			Expect(accessor(r)).To(Equal(expected))
		},
		Entry(
			"IsHistoryRestore",
			"HX-History-Restore-Request",
			cqrshtmx.HeaderTrue,
			func(r *http.Request) string { return strconv.FormatBool(cqrshtmx.IsHistoryRestore(r)) },
			"true",
		),
		Entry(
			"HTMXTriggerName",
			"HX-Trigger-Name",
			"my-trigger",
			cqrshtmx.HTMXTriggerName,
			"my-trigger",
		),
		Entry("HTMXPrompt", "HX-Prompt", "confirmed", cqrshtmx.HTMXPrompt, "confirmed"),
		Entry(
			"HTMXCurrentURL",
			"HX-Current-URL",
			"https://example.com/test",
			cqrshtmx.HTMXCurrentURL,
			"https://example.com/test",
		),
	)
})
