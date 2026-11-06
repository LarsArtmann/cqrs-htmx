package cqrshtmx_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/larsartmann/cqrs-htmx/v4/htmx"
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
					h := htmx.FromContext(req.Context())
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
					h := htmx.FromContext(req.Context())
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
			Expect(htmx.RenderPartial(r)).To(BeTrue())
		})

		It("returns false for HTMX request with history restore", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("Hx-Request", cqrshtmx.HeaderTrue)
			r.Header.Set("Hx-History-Restore-Request", cqrshtmx.HeaderTrue)
			Expect(htmx.RenderPartial(r)).To(BeFalse())
		})

		It("returns false for non-HTMX request", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			Expect(htmx.RenderPartial(r)).To(BeFalse())
		})

		It("uses context when HTMXMiddleware was applied", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("Hx-Request", cqrshtmx.HeaderTrue)

			handler := cqrshtmx.HTMXMiddleware(
				http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
					Expect(htmx.RenderPartial(req)).To(BeTrue())
					h := htmx.FromContext(req.Context())
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
			Expect(htmx.TriggerName(r)).To(Equal("action-btn"))
		})

		It("returns empty when not set", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			Expect(htmx.TriggerName(r)).To(BeEmpty())
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
					Expect(htmx.IsRequest(req)).To(BeTrue())
					Expect(htmx.IsBoosted(req)).To(BeTrue())
					Expect(htmx.Target(req)).To(Equal("main"))
					Expect(htmx.Trigger(req)).To(Equal("btn"))
					Expect(htmx.IsHistoryRestore(req)).To(BeTrue())
					Expect(htmx.TriggerName(req)).To(Equal("action-btn"))
					Expect(htmx.Prompt(req)).To(Equal("yes"))
					Expect(htmx.CurrentURL(req)).To(Equal("https://example.com/page"))
				}),
			)

			handler.ServeHTTP(httptest.NewRecorder(), r)
		})
	})

	Describe("HTMXFromContext returns nil without middleware", func() {
		It("returns nil when no HTMXMiddleware was applied", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			Expect(htmx.FromContext(r.Context())).To(BeNil())
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
			func(r *http.Request) string { return strconv.FormatBool(htmx.IsHistoryRestore(r)) },
			"true",
		),
		Entry(
			"HTMXTriggerName",
			"HX-Trigger-Name",
			"my-trigger",
			htmx.TriggerName,
			"my-trigger",
		),
		Entry("HTMXPrompt", "HX-Prompt", "confirmed", htmx.Prompt, "confirmed"),
		Entry(
			"HTMXCurrentURL",
			"HX-Current-URL",
			"https://example.com/test",
			htmx.CurrentURL,
			"https://example.com/test",
		),
	)
})
