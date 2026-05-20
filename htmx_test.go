package cqrshtmx_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"

	cqrshtmx "github.com/larsartmann/cqrs-htmx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HTMX", func() {
	DescribeTable(
		"bool accessors",
		func(header string, setHeader bool, accessor func(*http.Request) bool, expected bool) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if setHeader {
				r.Header.Set(header, cqrshtmx.HeaderTrue)
			}
			Expect(accessor(r)).To(Equal(expected))
		},
		Entry("IsHTMXRequest true", "HX-Request", true, cqrshtmx.IsHTMXRequest, true),
		Entry("IsHTMXRequest absent", "HX-Request", false, cqrshtmx.IsHTMXRequest, false),
		Entry("IsBoosted true", "HX-Boosted", true, cqrshtmx.IsBoosted, true),
		Entry("IsBoosted absent", "HX-Boosted", false, cqrshtmx.IsBoosted, false),
		Entry(
			"IsHistoryRestore true",
			"HX-History-Restore-Request",
			true,
			cqrshtmx.IsHistoryRestore,
			true,
		),
	)

	DescribeTable(
		"string accessors",
		func(header, value string, accessor func(*http.Request) string, expected string) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if value != "" {
				r.Header.Set(header, value)
			}
			Expect(accessor(r)).To(Equal(expected))
		},
		Entry("HTMXTarget set", "HX-Target", "user-list", cqrshtmx.HTMXTarget, "user-list"),
		Entry("HTMXTarget empty", "HX-Target", "", cqrshtmx.HTMXTarget, ""),
		Entry("HTMXTrigger set", "HX-Trigger", "submit-btn", cqrshtmx.HTMXTrigger, "submit-btn"),
		Entry("HTMXPrompt set", "HX-Prompt", "yes", cqrshtmx.HTMXPrompt, "yes"),
		Entry(
			"HTMXCurrentURL set",
			"HX-Current-URL",
			"https://example.com/users",
			cqrshtmx.HTMXCurrentURL,
			"https://example.com/users",
		),
	)

	It("returns false when HX-Request header is not 'true'", func() {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("HX-Request", "false")
		Expect(cqrshtmx.IsHTMXRequest(r)).To(BeFalse())
	})
})

var _ = Describe("HTMX Response Builder", func() {
	var (
		w *httptest.ResponseRecorder
		r *http.Request
	)

	BeforeEach(func() {
		w = httptest.NewRecorder()
		r = httptest.NewRequest(http.MethodGet, "/", nil)
	})

	assertHeader := func(resp *cqrshtmx.Response, header, expected string) {
		resp.Apply()
		Expect(w.Header().Get(header)).To(Equal(expected))
	}

	Describe("NewResponse", func() {
		It("creates a response builder", func() {
			Expect(cqrshtmx.NewResponse(w, r)).NotTo(BeNil())
		})
	})

	DescribeTable(
		"header setters",
		func(build func(*cqrshtmx.Response) *cqrshtmx.Response, header, expected string) {
			assertHeader(build(cqrshtmx.NewResponse(w, r)), header, expected)
		},
		Entry(
			"PushURL",
			func(r *cqrshtmx.Response) *cqrshtmx.Response { return r.PushURL("/users/1") },
			"HX-Push-Url",
			"/users/1",
		),
		Entry(
			"ReplaceURL",
			func(r *cqrshtmx.Response) *cqrshtmx.Response { return r.ReplaceURL("/users/1") },
			"HX-Replace-Url",
			"/users/1",
		),
		Entry(
			"Refresh",
			func(r *cqrshtmx.Response) *cqrshtmx.Response { return r.Refresh() },
			"HX-Refresh",
			cqrshtmx.HeaderTrue,
		),
		Entry(
			"Reswap",
			func(r *cqrshtmx.Response) *cqrshtmx.Response { return r.Reswap(cqrshtmx.SwapAfterEnd) },
			"HX-Reswap",
			"afterend",
		),
		Entry(
			"Retarget",
			func(r *cqrshtmx.Response) *cqrshtmx.Response { return r.Retarget("#modal") },
			"HX-Retarget",
			"#modal",
		),
		Entry(
			"Reselect",
			func(r *cqrshtmx.Response) *cqrshtmx.Response { return r.Reselect("#content") },
			"HX-Reselect",
			"#content",
		),
		Entry(
			"Trigger",
			func(r *cqrshtmx.Response) *cqrshtmx.Response { return r.Trigger("userCreated") },
			"HX-Trigger",
			"userCreated",
		),
		Entry(
			"TriggerAfterSwap",
			func(r *cqrshtmx.Response) *cqrshtmx.Response { return r.TriggerAfterSwap("dataLoaded") },
			"HX-Trigger-After-Swap",
			"dataLoaded",
		),
		Entry(
			"TriggerAfterSettle",
			func(r *cqrshtmx.Response) *cqrshtmx.Response { return r.TriggerAfterSettle("animationDone") },
			"HX-Trigger-After-Settle",
			"animationDone",
		),
	)

	Describe("IsHTMX", func() {
		It("returns true for HTMX requests", func() {
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			Expect(cqrshtmx.NewResponse(w, r).IsHTMX()).To(BeTrue())
		})

		It("returns false for regular requests", func() {
			Expect(cqrshtmx.NewResponse(w, r).IsHTMX()).To(BeFalse())
		})
	})

	Describe("Redirect for HTMX requests", func() {
		It("sets HX-Redirect header for HTMX requests", func() {
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			cqrshtmx.NewResponse(w, r).Redirect("/login").Apply()
			Expect(w.Header().Get("HX-Redirect")).To(Equal("/login"))
		})
	})

	Describe("Redirect for regular requests", func() {
		It("uses HTTP redirect for non-HTMX requests", func() {
			cqrshtmx.NewResponse(w, r).Redirect("/login").Apply()
			Expect(w.Code).To(Equal(http.StatusSeeOther))
			Expect(w.Header().Get("Location")).To(Equal("/login"))
		})
	})

	Describe("Trigger", func() {
		It("appends multiple triggers", func() {
			cqrshtmx.NewResponse(w, r).Trigger("evt1").Trigger("evt2").Apply()
			Expect(w.Header().Get("HX-Trigger")).To(Equal("evt1,evt2"))
		})
	})

	Describe("TriggerWithDetail", func() {
		It("sets HX-Trigger header with JSON detail", func() {
			cqrshtmx.NewResponse(w, r).
				TriggerWithDetail("userCreated", map[string]string{"id": "123"}).
				Apply()
			triggerHeader := w.Header().Get("HX-Trigger")
			Expect(triggerHeader).To(ContainSubstring("userCreated"))
			Expect(triggerHeader).To(ContainSubstring("123"))
		})
	})

	Describe("Apply", func() {
		It("sets Content-Type to text/html for HTMX requests", func() {
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			cqrshtmx.NewResponse(w, r).Apply()
			Expect(w.Header().Get("Content-Type")).To(Equal("text/html; charset=utf-8"))
		})

		It("does not set Content-Type for regular requests", func() {
			cqrshtmx.NewResponse(w, r).Apply()
			Expect(w.Header().Get("Content-Type")).To(BeEmpty())
		})
	})

	Describe("Chaining", func() {
		It("allows fluent method chaining", func() {
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			cqrshtmx.NewResponse(w, r).
				Trigger("userCreated").
				PushURL("/users/1").
				Retarget("#user-list").
				Reswap(cqrshtmx.SwapInnerHTML).
				Apply()

			Expect(w.Header().Get("HX-Trigger")).To(Equal("userCreated"))
			Expect(w.Header().Get("HX-Push-Url")).To(Equal("/users/1"))
			Expect(w.Header().Get("HX-Retarget")).To(Equal("#user-list"))
			Expect(w.Header().Get("HX-Reswap")).To(Equal("innerHTML"))
		})
	})
})

var _ = Describe("SwapStrategy constants", func() {
	It("has correct swap strategy values", func() {
		Expect(string(cqrshtmx.SwapInnerHTML)).To(Equal("innerHTML"))
		Expect(string(cqrshtmx.SwapOuterHTML)).To(Equal("outerHTML"))
		Expect(string(cqrshtmx.SwapBeforeBegin)).To(Equal("beforebegin"))
		Expect(string(cqrshtmx.SwapAfterBegin)).To(Equal("afterbegin"))
		Expect(string(cqrshtmx.SwapBeforeEnd)).To(Equal("beforeend"))
		Expect(string(cqrshtmx.SwapAfterEnd)).To(Equal("afterend"))
		Expect(string(cqrshtmx.SwapDelete)).To(Equal("delete"))
		Expect(string(cqrshtmx.SwapNone)).To(Equal("none"))
	})
})

var _ = Describe("HTMXRequest context", func() {
	Describe("HTMXMiddleware", func() {
		It("parses all HTMX headers and stores in context", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			r.Header.Set("HX-Boosted", cqrshtmx.HeaderTrue)
			r.Header.Set("HX-Target", "main")
			r.Header.Set("HX-Trigger", "btn")
			r.Header.Set("HX-Trigger-Name", "action")
			r.Header.Set("HX-Prompt", "yes")
			r.Header.Set("HX-Current-URL", "https://example.com/page")

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
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			r.Header.Set("HX-History-Restore-Request", cqrshtmx.HeaderTrue)

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
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			Expect(cqrshtmx.RenderPartial(r)).To(BeTrue())
		})

		It("returns false for HTMX request with history restore", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			r.Header.Set("HX-History-Restore-Request", cqrshtmx.HeaderTrue)
			Expect(cqrshtmx.RenderPartial(r)).To(BeFalse())
		})

		It("returns false for non-HTMX request", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			Expect(cqrshtmx.RenderPartial(r)).To(BeFalse())
		})

		It("uses context when HTMXMiddleware was applied", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)

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
			r.Header.Set("HX-Trigger-Name", "action-btn")
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
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			r.Header.Set("HX-Target", "main")
			r.Header.Set("HX-Trigger", "btn")
			r.Header.Set("HX-Boosted", cqrshtmx.HeaderTrue)
			r.Header.Set("HX-History-Restore-Request", cqrshtmx.HeaderTrue)
			r.Header.Set("HX-Trigger-Name", "action-btn")
			r.Header.Set("HX-Prompt", "yes")
			r.Header.Set("HX-Current-URL", "https://example.com/page")

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

var _ = DescribeTable(
	"notification methods",
	func(notify func(*cqrshtmx.Response) *cqrshtmx.Response, level, message string) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		notify(cqrshtmx.NewResponse(w, r)).Apply()
		trigger := w.Header().Get("HX-Trigger")
		Expect(trigger).To(ContainSubstring(level))
		Expect(trigger).To(ContainSubstring(message))
	},
	Entry(
		"NotifySuccess",
		func(r *cqrshtmx.Response) *cqrshtmx.Response { return r.NotifySuccess("Done!") },
		"success",
		"Done!",
	),
	Entry(
		"NotifyError",
		func(r *cqrshtmx.Response) *cqrshtmx.Response { return r.NotifyError("Failed") },
		"error",
		"Failed",
	),
	Entry(
		"NotifyWarning",
		func(r *cqrshtmx.Response) *cqrshtmx.Response { return r.NotifyWarning("Careful") },
		"warning",
		"Careful",
	),
	Entry(
		"NotifyInfo",
		func(r *cqrshtmx.Response) *cqrshtmx.Response { return r.NotifyInfo("FYI") },
		"info",
		"FYI",
	),
)
