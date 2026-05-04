package cqrshtmx_test

import (
	"net/http"
	"net/http/httptest"
	"strings"

	cqrshtmx "github.com/larsartmann/cqrs-htmx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HTMX", func() {
	Describe("IsHTMXRequest", func() {
		It("returns true when HX-Request header is set", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("HX-Request", "true")
			Expect(cqrshtmx.IsHTMXRequest(r)).To(BeTrue())
		})

		It("returns false when HX-Request header is absent", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			Expect(cqrshtmx.IsHTMXRequest(r)).To(BeFalse())
		})

		It("returns false when HX-Request header is not 'true'", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("HX-Request", "false")
			Expect(cqrshtmx.IsHTMXRequest(r)).To(BeFalse())
		})
	})

	Describe("IsBoosted", func() {
		It("returns true when HX-Boosted header is set", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("HX-Boosted", "true")
			Expect(cqrshtmx.IsBoosted(r)).To(BeTrue())
		})

		It("returns false when HX-Boosted header is absent", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			Expect(cqrshtmx.IsBoosted(r)).To(BeFalse())
		})
	})

	Describe("IsHistoryRestore", func() {
		It("returns true when HX-History-Restore-Request header is set", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("HX-History-Restore-Request", "true")
			Expect(cqrshtmx.IsHistoryRestore(r)).To(BeTrue())
		})
	})

	Describe("HTMXTarget", func() {
		It("returns the target element ID", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("HX-Target", "user-list")
			Expect(cqrshtmx.HTMXTarget(r)).To(Equal("user-list"))
		})

		It("returns empty string when not set", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			Expect(cqrshtmx.HTMXTarget(r)).To(BeEmpty())
		})
	})

	Describe("HTMXTrigger", func() {
		It("returns the trigger element ID", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("HX-Trigger", "submit-btn")
			Expect(cqrshtmx.HTMXTrigger(r)).To(Equal("submit-btn"))
		})
	})

	Describe("HTMXPrompt", func() {
		It("returns the prompt response", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("HX-Prompt", "yes")
			Expect(cqrshtmx.HTMXPrompt(r)).To(Equal("yes"))
		})
	})

	Describe("HTMXCurrentURL", func() {
		It("returns the current URL", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("HX-Current-URL", "https://example.com/users")
			Expect(cqrshtmx.HTMXCurrentURL(r)).To(Equal("https://example.com/users"))
		})
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

	Describe("NewResponse", func() {
		It("creates a response builder", func() {
			resp := cqrshtmx.NewResponse(w, r)
			Expect(resp).NotTo(BeNil())
		})
	})

	Describe("IsHTMX", func() {
		It("returns true for HTMX requests", func() {
			r.Header.Set("HX-Request", "true")
			resp := cqrshtmx.NewResponse(w, r)
			Expect(resp.IsHTMX()).To(BeTrue())
		})

		It("returns false for regular requests", func() {
			resp := cqrshtmx.NewResponse(w, r)
			Expect(resp.IsHTMX()).To(BeFalse())
		})
	})

	Describe("PushURL", func() {
		It("sets HX-Push-Url header", func() {
			cqrshtmx.NewResponse(w, r).PushURL("/users/1").Apply()
			Expect(w.Header().Get("HX-Push-Url")).To(Equal("/users/1"))
		})
	})

	Describe("ReplaceURL", func() {
		It("sets HX-Replace-Url header", func() {
			cqrshtmx.NewResponse(w, r).ReplaceURL("/users/1").Apply()
			Expect(w.Header().Get("HX-Replace-Url")).To(Equal("/users/1"))
		})
	})

	Describe("Redirect for HTMX requests", func() {
		It("sets HX-Redirect header for HTMX requests", func() {
			r.Header.Set("HX-Request", "true")
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

	Describe("Refresh", func() {
		It("sets HX-Refresh header", func() {
			cqrshtmx.NewResponse(w, r).Refresh().Apply()
			Expect(w.Header().Get("HX-Refresh")).To(Equal("true"))
		})
	})

	Describe("Reswap", func() {
		It("sets HX-Reswap header", func() {
			cqrshtmx.NewResponse(w, r).Reswap(cqrshtmx.SwapAfterEnd).Apply()
			Expect(w.Header().Get("HX-Reswap")).To(Equal("afterend"))
		})
	})

	Describe("Retarget", func() {
		It("sets HX-Retarget header", func() {
			cqrshtmx.NewResponse(w, r).Retarget("#modal").Apply()
			Expect(w.Header().Get("HX-Retarget")).To(Equal("#modal"))
		})
	})

	Describe("Reselect", func() {
		It("sets HX-Reselect header", func() {
			cqrshtmx.NewResponse(w, r).Reselect("#content").Apply()
			Expect(w.Header().Get("HX-Reselect")).To(Equal("#content"))
		})
	})

	Describe("Trigger", func() {
		It("sets HX-Trigger header", func() {
			cqrshtmx.NewResponse(w, r).Trigger("userCreated").Apply()
			Expect(w.Header().Get("HX-Trigger")).To(Equal("userCreated"))
		})

		It("appends multiple triggers", func() {
			cqrshtmx.NewResponse(w, r).Trigger("evt1").Trigger("evt2").Apply()
			Expect(w.Header().Get("HX-Trigger")).To(Equal("evt1,evt2"))
		})
	})

	Describe("TriggerAfterSwap", func() {
		It("sets HX-Trigger-After-Swap header", func() {
			cqrshtmx.NewResponse(w, r).TriggerAfterSwap("dataLoaded").Apply()
			Expect(w.Header().Get("HX-Trigger-After-Swap")).To(Equal("dataLoaded"))
		})
	})

	Describe("TriggerAfterSettle", func() {
		It("sets HX-Trigger-After-Settle header", func() {
			cqrshtmx.NewResponse(w, r).TriggerAfterSettle("animationDone").Apply()
			Expect(w.Header().Get("HX-Trigger-After-Settle")).To(Equal("animationDone"))
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
			r.Header.Set("HX-Request", "true")
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
			r.Header.Set("HX-Request", "true")
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
			r.Header.Set("HX-Request", "true")
			r.Header.Set("HX-Boosted", "true")
			r.Header.Set("HX-Target", "main")
			r.Header.Set("HX-Trigger", "btn")
			r.Header.Set("HX-Trigger-Name", "action")
			r.Header.Set("HX-Prompt", "yes")
			r.Header.Set("HX-Current-URL", "https://example.com/page")

			called := false
			handler := cqrshtmx.HTMXMiddleware(
				http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
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
			r.Header.Set("HX-Request", "true")
			r.Header.Set("HX-History-Restore-Request", "true")

			handler := cqrshtmx.HTMXMiddleware(
				http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
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
			r.Header.Set("HX-Request", "true")
			Expect(cqrshtmx.RenderPartial(r)).To(BeTrue())
		})

		It("returns false for HTMX request with history restore", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("HX-Request", "true")
			r.Header.Set("HX-History-Restore-Request", "true")
			Expect(cqrshtmx.RenderPartial(r)).To(BeFalse())
		})

		It("returns false for non-HTMX request", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			Expect(cqrshtmx.RenderPartial(r)).To(BeFalse())
		})

		It("uses context when HTMXMiddleware was applied", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("HX-Request", "true")

			handler := cqrshtmx.HTMXMiddleware(
				http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
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
			r.Header.Set("HX-Request", "true")
			r.Header.Set("HX-Target", "main")
			r.Header.Set("HX-Trigger", "btn")
			r.Header.Set("HX-Boosted", "true")

			handler := cqrshtmx.HTMXMiddleware(
				http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					Expect(cqrshtmx.IsHTMXRequest(req)).To(BeTrue())
					Expect(cqrshtmx.IsBoosted(req)).To(BeTrue())
					Expect(cqrshtmx.HTMXTarget(req)).To(Equal("main"))
					Expect(cqrshtmx.HTMXTrigger(req)).To(Equal("btn"))
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

	Describe("accessor fallback without middleware", func() {
		It("reads IsHistoryRestore from header directly", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("HX-History-Restore-Request", "true")
			Expect(cqrshtmx.IsHistoryRestore(r)).To(BeTrue())
		})

		It("reads HTMXTriggerName from header directly", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("HX-Trigger-Name", "my-trigger")
			Expect(cqrshtmx.HTMXTriggerName(r)).To(Equal("my-trigger"))
		})

		It("reads HTMXPrompt from header directly", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("HX-Prompt", "confirmed")
			Expect(cqrshtmx.HTMXPrompt(r)).To(Equal("confirmed"))
		})

		It("reads HTMXCurrentURL from header directly", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("HX-Current-URL", "https://example.com/test")
			Expect(cqrshtmx.HTMXCurrentURL(r)).To(Equal("https://example.com/test"))
		})
	})
})

var _ = Describe("Notification helpers", func() {
	Describe("Response notification methods", func() {
		It("NotifySuccess sends success notification", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			cqrshtmx.NewResponse(w, r).NotifySuccess("Done!").Apply()
			trigger := w.Header().Get("HX-Trigger")
			Expect(trigger).To(ContainSubstring("success"))
			Expect(trigger).To(ContainSubstring("Done!"))
		})

		It("NotifyError sends error notification", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			cqrshtmx.NewResponse(w, r).NotifyError("Failed").Apply()
			trigger := w.Header().Get("HX-Trigger")
			Expect(trigger).To(ContainSubstring("error"))
			Expect(trigger).To(ContainSubstring("Failed"))
		})

		It("NotifyWarning sends warning notification", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			cqrshtmx.NewResponse(w, r).NotifyWarning("Careful").Apply()
			trigger := w.Header().Get("HX-Trigger")
			Expect(trigger).To(ContainSubstring("warning"))
			Expect(trigger).To(ContainSubstring("Careful"))
		})

		It("NotifyInfo sends info notification", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			cqrshtmx.NewResponse(w, r).NotifyInfo("FYI").Apply()
			trigger := w.Header().Get("HX-Trigger")
			Expect(trigger).To(ContainSubstring("info"))
			Expect(trigger).To(ContainSubstring("FYI"))
		})
	})
})

func init() {
	_ = strings.NewReader("")
}
