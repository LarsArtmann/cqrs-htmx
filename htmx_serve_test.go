package cqrshtmx_test

import (
	"net/http"
	"net/http/httptest"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HTMXScriptHandler", func() {
	var handler http.Handler
	var req *http.Request
	var w *httptest.ResponseRecorder

	BeforeEach(func() {
		handler = cqrshtmx.HTMXScriptHandler()
		w = httptest.NewRecorder()
	})

	Describe("serving the embedded JS", func() {
		BeforeEach(func() {
			req = httptest.NewRequest(http.MethodGet, "/htmx.js", nil)
		})

		It("serves the JS with correct content type", func() {
			handler.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Header().Get("Content-Type")).To(Equal("text/javascript; charset=utf-8"))
			Expect(w.Body.Len()).To(BeNumerically(">", 0))
		})

		It("sets long-lived cache headers", func() {
			handler.ServeHTTP(w, req)
			Expect(w.Header().Get("Cache-Control")).To(Equal("public, max-age=31536000, immutable"))
		})

		It("sets an ETag header", func() {
			handler.ServeHTTP(w, req)
			Expect(w.Header().Get("ETag")).To(Equal(`"htmx-2.0.9"`))
		})

		It("returns 304 for matching If-None-Match", func() {
			req.Header.Set("If-None-Match", `"htmx-2.0.9"`)
			handler.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusNotModified))
			Expect(w.Body.Len()).To(Equal(0))
		})
	})

	Describe("method handling", func() {
		It("allows GET", func() {
			req = httptest.NewRequest(http.MethodGet, "/htmx.js", nil)
			handler.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusOK))
		})

		It("allows HEAD", func() {
			req = httptest.NewRequest(http.MethodHead, "/htmx.js", nil)
			handler.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusOK))
		})

		It("rejects POST with 405", func() {
			req = httptest.NewRequest(http.MethodPost, "/htmx.js", nil)
			handler.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusMethodNotAllowed))
		})

		It("rejects PUT with 405", func() {
			req = httptest.NewRequest(http.MethodPut, "/htmx.js", nil)
			handler.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusMethodNotAllowed))
		})
	})
})

var _ = Describe("HTMXVersion", func() {
	It("returns the version string", func() {
		Expect(cqrshtmx.HTMXVersion()).To(Equal("2.0.9"))
	})
})

var _ = Describe("HTMXScriptTag", func() {
	It("returns a script tag with the given path", func() {
		Expect(cqrshtmx.HTMXScriptTag("/static/htmx.js")).To(
			Equal(`<script src="/static/htmx.js"></script>`),
		)
	})

	It("returns a script tag with a different path", func() {
		Expect(cqrshtmx.HTMXScriptTag("/assets/js/htmx.js")).To(
			Equal(`<script src="/assets/js/htmx.js"></script>`),
		)
	})
})

var _ = Describe("HTMXScriptHandlerWith", func() {
	var handler http.Handler
	var req *http.Request
	var w *httptest.ResponseRecorder

	BeforeEach(func() {
		handler = cqrshtmx.HTMXScriptHandlerWith(
			[]byte("// custom htmx build"),
			"4.0.0-beta4",
		)
		w = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodGet, "/htmx.js", nil)
	})

	It("serves the custom JS content", func() {
		handler.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Body.String()).To(Equal("// custom htmx build"))
	})

	It("sets the content type", func() {
		handler.ServeHTTP(w, req)
		Expect(w.Header().Get("Content-Type")).To(Equal("text/javascript; charset=utf-8"))
	})

	It("sets cache headers", func() {
		handler.ServeHTTP(w, req)
		Expect(w.Header().Get("Cache-Control")).To(Equal("public, max-age=31536000, immutable"))
	})

	It("sets an ETag derived from the version", func() {
		handler.ServeHTTP(w, req)
		Expect(w.Header().Get("ETag")).To(Equal(`"htmx-4.0.0-beta4"`))
	})

	It("returns 304 for matching If-None-Match", func() {
		req.Header.Set("If-None-Match", `"htmx-4.0.0-beta4"`)
		handler.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusNotModified))
		Expect(w.Body.Len()).To(Equal(0))
	})

	It("does not return 304 for non-matching If-None-Match", func() {
		req.Header.Set("If-None-Match", `"htmx-2.0.9"`)
		handler.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusOK))
	})

	It("allows HEAD", func() {
		req = httptest.NewRequest(http.MethodHead, "/htmx.js", nil)
		handler.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusOK))
	})

	It("rejects POST", func() {
		req = httptest.NewRequest(http.MethodPost, "/htmx.js", nil)
		handler.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusMethodNotAllowed))
	})
})

var _ = Describe("HTMXCDNScriptTag", func() {
	It("returns a CDN script tag with a specific version", func() {
		Expect(cqrshtmx.HTMXCDNScriptTag("4.0.0")).To(Equal(
			`<script src="https://unpkg.com/htmx.org@4.0.0"></script>`,
		))
	})

	It("uses the embedded version when version is empty", func() {
		Expect(cqrshtmx.HTMXCDNScriptTag("")).To(Equal(
			`<script src="https://unpkg.com/htmx.org@2.0.9"></script>`,
		))
	})

	It("supports pre-release versions", func() {
		Expect(cqrshtmx.HTMXCDNScriptTag("4.0.0-beta4")).To(Equal(
			`<script src="https://unpkg.com/htmx.org@4.0.0-beta4"></script>`,
		))
	})
})
