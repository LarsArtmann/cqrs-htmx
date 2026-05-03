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

func init() {
	_ = strings.NewReader("")
}
