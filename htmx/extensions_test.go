package htmx_test

import (
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/larsartmann/cqrs-htmx/v4/htmx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ExtensionHandler", func() {
	DescribeTable("serves each embedded extension",
		func(name, wantVersion string) {
			handler := htmx.ExtensionHandler(name)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/ext/"+name+".js", nil)
			handler.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Header().Get("Content-Type")).To(Equal("text/javascript; charset=utf-8"))
			Expect(w.Header().Get("Cache-Control")).To(Equal("public, max-age=31536000, immutable"))
			Expect(w.Header().Get("ETag")).To(Equal(`"htmx-ext-` + name + `-` + wantVersion + `"`))
			Expect(w.Body.Len()).To(BeNumerically(">", 0))
		},
		Entry("sse", htmx.ExtSSE, "2.2.4"),
		Entry("idiomorph", htmx.ExtIdiomorph, "0.8.0"),
		Entry("cqrs-query", htmx.ExtCQRSQuery, "1.0.0"),
	)

	It("returns 304 for matching If-None-Match", func() {
		handler := htmx.ExtensionHandler(htmx.ExtSSE)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/ext/sse.js", nil)
		req.Header.Set("If-None-Match", `"htmx-ext-sse-2.2.4"`)
		handler.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusNotModified))
		Expect(w.Body.Len()).To(Equal(0))
	})

	It("returns 200 for non-matching If-None-Match", func() {
		handler := htmx.ExtensionHandler(htmx.ExtSSE)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/ext/sse.js", nil)
		req.Header.Set("If-None-Match", `"htmx-ext-sse-2.2.3"`)
		handler.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusOK))
	})

	It("returns 405 for POST", func() {
		handler := htmx.ExtensionHandler(htmx.ExtIdiomorph)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/ext/idiomorph.js", nil)
		handler.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusMethodNotAllowed))
	})

	It("panics for unknown extension name", func() {
		Expect(func() { htmx.ExtensionHandler("unknown") }).To(Panic())
	})
})

var _ = Describe("ExtensionsHandler", func() {
	It("serves a concatenated bundle of multiple extensions", func() {
		handler := htmx.ExtensionsHandler(htmx.ExtSSE, htmx.ExtIdiomorph)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/ext/bundle.js", nil)
		handler.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Header().Get("Content-Type")).To(Equal("text/javascript; charset=utf-8"))
		Expect(w.Header().Get("Cache-Control")).To(Equal("public, max-age=31536000, immutable"))
		Expect(w.Header().Get("ETag")).To(Equal(`"htmx-ext-bundle-sse-2.2.4,idiomorph-0.8.0"`))
		Expect(w.Body.Len()).To(BeNumerically(">", 0))

		body := w.Body.String()
		Expect(body).To(ContainSubstring("htmx-ext-sse 2.2.4"))
		Expect(body).To(ContainSubstring("htmx-ext-idiomorph 0.8.0"))
	})

	It("returns 304 for matching If-None-Match", func() {
		handler := htmx.ExtensionsHandler(htmx.ExtSSE, htmx.ExtIdiomorph)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/ext/bundle.js", nil)
		req.Header.Set("If-None-Match", `"htmx-ext-bundle-sse-2.2.4,idiomorph-0.8.0"`)
		handler.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusNotModified))
	})

	It("returns 405 for POST", func() {
		handler := htmx.ExtensionsHandler(htmx.ExtSSE)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/ext/bundle.js", nil)
		handler.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusMethodNotAllowed))
	})

	It("panics for unknown extension name in bundle", func() {
		Expect(func() {
			htmx.ExtensionsHandler(htmx.ExtSSE, "bogus")
		}).To(Panic())
	})

	It("panics when no names are given", func() {
		Expect(func() { htmx.ExtensionsHandler() }).To(Panic())
	})
})

var _ = Describe("ExtensionVersion", func() {
	DescribeTable("returns the version for known extensions",
		func(name, want string) {
			Expect(htmx.ExtensionVersion(name)).To(Equal(want))
		},
		Entry("sse", htmx.ExtSSE, "2.2.4"),
		Entry("idiomorph", htmx.ExtIdiomorph, "0.8.0"),
		Entry("cqrs-query", htmx.ExtCQRSQuery, "1.0.0"),
	)

	It("returns empty string for unknown extension", func() {
		Expect(htmx.ExtensionVersion("bogus")).To(Equal(""))
	})
})

var _ = Describe("ExtensionNames", func() {
	It("returns all extension names sorted alphabetically", func() {
		names := htmx.ExtensionNames()
		Expect(names).To(Equal([]string{"cqrs-query", "idiomorph", "sse"}))
	})

	It("returns names that join cleanly for error messages", func() {
		names := htmx.ExtensionNames()
		Expect(strings.Join(names, ", ")).To(Equal("cqrs-query, idiomorph, sse"))
	})
})

var _ = Describe("ExtensionCDNScriptTag", func() {
	DescribeTable("returns a CDN script tag with the embedded version",
		func(name, wantURL string) {
			Expect(htmx.ExtensionCDNScriptTag(name)).To(Equal(
				`<script src="` + wantURL + `"></script>`,
			))
		},
		Entry("sse", htmx.ExtSSE, "https://unpkg.com/htmx-ext-sse@2.2.4/dist/sse.min.js"),
		Entry("idiomorph", htmx.ExtIdiomorph, "https://unpkg.com/idiomorph@0.8.0/dist/idiomorph-ext.min.js"),
	)

	It("returns empty string for unknown extension", func() {
		Expect(htmx.ExtensionCDNScriptTag("bogus")).To(Equal(""))
	})
})
