package cqrshtmx_test

import (
	"net/http"
	"net/http/httptest"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

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
